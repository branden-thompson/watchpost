package app

import (
	"context"
	ionomaps "github.com/branden-thompson/go-ionomaps"
	"github.com/branden-thompson/go-ionomaps/ionomapstest"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/propagation"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// TestThePropagationClientsAsksReachTheStatusWindow is D-169: the
// propagation client's asks are in the counters the Status window reads for
// the map's hosts.
func TestThePropagationClientsAsksReachTheStatusWindow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("grid")) }))
	defer srv.Close()
	client, err := httpx.NewPlain(httpx.PlainConfig{UserAgent: "watchpost-test"})
	if err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{}
	if n := len(lp.propagation.requestStats().Hosts); n != 0 {
		t.Fatalf("%d hosts counted before any update", n)
	}
	lp.propagation.client.Store(client)
	if _, err := client.Get(context.Background(), srv.URL, "", ""); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(srv.URL)
	for _, h := range lp.ttyStats().MapRequests.Hosts {
		if h.Host == u.Hostname() && h.Attempts == 1 && h.Net == 1 {
			return
		}
	}
	t.Errorf("the propagation client's ask is not in the map's counters: %+v", lp.ttyStats().MapRequests.Hosts)
}

// TestThePropagationClientsWireBytesReachTheCounters is 0.19.0 G1's
// downloads (D-141, DN-12): the counters record carries what the propagation
// client read off the wire, nothing before the first update makes it.
func TestThePropagationClientsWireBytesReachTheCounters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("grid")) }))
	defer srv.Close()
	client, err := httpx.NewPlain(httpx.PlainConfig{UserAgent: "watchpost-test"})
	if err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{}
	d := newDumper(t.TempDir(), time.Now(), lp.sources, lp.ttyStats)
	if got := d.record(time.Now()).PropagationWireBytes; got != 0 {
		t.Fatalf("%d bytes counted before any update", got)
	}
	lp.propagation.client.Store(client)
	if _, err := client.Get(context.Background(), srv.URL, "", ""); err != nil {
		t.Fatal(err)
	}
	if got := d.record(time.Now()).PropagationWireBytes; got <= int64(len("grid")) || got != client.WireBytes() {
		t.Errorf("the counters carry %d wire bytes; the client read %d", got, client.WireBytes())
	}
	var none *propagationHolder
	if none.wireBytes() != 0 {
		t.Error("no holder counts bytes")
	}
}

// TestOneLibraryObjectPerProcess is 0.19.0 FR-4.8: however many updates are
// asked, and however many times the window's configuration is built, the
// process makes one go-ionomaps object, and only when an update is asked.
func TestOneLibraryObjectPerProcess(t *testing.T) {
	made := 0
	was := newPropagationService
	newPropagationService = func(note func(string), client *httpx.Plain) (*propagation.Service, error) {
		made++
		return was(note, client)
	}
	t.Cleanup(func() { newPropagationService = was })
	lp := &livePipelines{}
	var c tty.Config
	lp.mapConfig(&c, config.Default())
	lp.mapConfig(&c, config.Default())
	if made != 0 {
		t.Fatalf("building the window's configuration made %d objects; none until an update is asked", made)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // nothing is fetched: the updates end at once
	for range 3 {
		<-c.PropagationUpdate(ctx, ionomaps.UpdateOptions{})
	}
	if made != 1 {
		t.Errorf("three updates made %d objects; want one", made)
	}
}

// TestMapStatusListsEveryPropagationHost is 0.19.0 FR-4.3 (D-113): every
// host go-ionomaps exports is a row of MAP STATUS, read from the library's
// own list, with what it learns.
func TestMapStatusListsEveryPropagationHost(t *testing.T) {
	rows := map[string]tty.MapSource{}
	for _, s := range mapSourceList() {
		rows[s.Host] = s
	}
	var hosts []string
	for _, s := range ionomaps.Sources() { // the library's own list, read with no library made (C-5)
		hosts = append(hosts, s.Hosts...)
	}
	if len(hosts) < 2 {
		t.Fatalf("go-ionomaps exports %d hosts; GIRO and NOAA are asked at run time", len(hosts))
	}
	for _, h := range hosts {
		r, ok := rows[h]
		if !ok {
			t.Errorf("MAP STATUS does not list %s", h)
			continue
		}
		if !strings.Contains(strings.Join(r.Notes, " "), "this computer's IP") {
			t.Errorf("%s's row does not say it learns the IP address: %+v", h, r)
		}
	}
}

// solarDays answers NOAA's daily solar file and nothing else: enough for
// go-ionomaps to build a field from its climatology, asking nothing more.
type solarDays struct{}

func (solarDays) Get(_ context.Context, rawURL, _, _ string) (httpx.Answer, error) {
	if !strings.HasSuffix(rawURL, "daily-solar-indices.txt") {
		return httpx.Answer{Status: 404}, nil
	}
	var b strings.Builder
	for d := 30; d >= 1; d-- {
		b.WriteString(solarNoon().AddDate(0, 0, -d).Format("2006 01 02") + "  150  140  1  2  0  0  0  0  0  0  0\n")
	}
	return httpx.Answer{Status: 200, Body: []byte(b.String())}, nil
}

func solarNoon() time.Time { return time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC) }

// TestTheWindowsOptionsReachTheLibrary is FR-4.8's seam with D-187 and
// D-105: the update the window asks carries its options to go-ionomaps
// whole, so the snapshot answers for the noise asked.
func TestTheWindowsOptionsReachTheLibrary(t *testing.T) {
	was := newPropagationService
	newPropagationService = func(func(string), *httpx.Plain) (*propagation.Service, error) {
		return propagation.NewService(solarDays{}, propagation.Options{Clock: solarNoon})
	}
	t.Cleanup(func() { newPropagationService = was })
	lp := &livePipelines{}
	var c tty.Config
	lp.mapConfig(&c, config.Default())
	r := <-c.PropagationUpdate(context.Background(), ionomaps.UpdateOptions{Noise: ionomaps.NoiseCity, CorrectNoReadings: true})
	if r.Err != nil || len(r.Snapshot.Now.FoF2.Values) == 0 || r.Snapshot.Noise != ionomaps.NoiseCity {
		t.Errorf("asked with city noise: %v, %d cells, noise %v", r.Err, len(r.Snapshot.Now.FoF2.Values), r.Snapshot.Noise)
	}
}

// TestEachPropagationSourceHasItsOwnCap is 0.19.0 W0.2 (FR-4.4, D-201): the
// propagation client takes no more from a host than go-ionomaps parses from
// it - GIRO's bound, not NOAA's grid's.
func TestEachPropagationSourceHasItsOwnCap(t *testing.T) {
	caps := propagationClientConfig().MaxBodyByHost
	for _, s := range ionomaps.Sources() {
		for _, h := range s.Hosts {
			if caps[h] != int64(s.MaxBytes) || caps[h] <= 0 {
				t.Errorf("%s's cap is %d; the library's bound is %d", h, caps[h], s.MaxBytes)
			}
		}
	}
	if caps["lgdc.uml.edu"] >= caps["services.swpc.noaa.gov"] {
		t.Errorf("GIRO's cap %d is not under NOAA's %d", caps["lgdc.uml.edu"], caps["services.swpc.noaa.gov"])
	}
}

// seededStations are the library's 39 seeded live stations at their
// published places (go-ionomaps' seed), so a burst reads a full hour.
var seededStations = []ionomapstest.Station{
	{Code: "AL945", Lat: 45.1, Lon: -83.6, FoF2: 6, M3000: 3},
	{Code: "AS00Q", Lat: -7.9, Lon: -14.4, FoF2: 6, M3000: 3},
	{Code: "AT138", Lat: 38.0, Lon: 23.5, FoF2: 6, M3000: 3},
	{Code: "BVJ03", Lat: 2.8, Lon: -60.7, FoF2: 6, M3000: 3},
	{Code: "CAJ2M", Lat: -22.7, Lon: -45.0, FoF2: 6, M3000: 3},
	{Code: "CB53N", Lat: -35.3, Lon: 149.0, FoF2: 6, M3000: 3},
	{Code: "CGK21", Lat: -20.5, Lon: -55.0, FoF2: 6, M3000: 3},
	{Code: "DB049", Lat: 50.1, Lon: 4.6, FoF2: 6, M3000: 3},
	{Code: "DW41K", Lat: -12.4, Lon: 130.9, FoF2: 6, M3000: 3},
	{Code: "EA036", Lat: 37.1, Lon: -6.7, FoF2: 6, M3000: 3},
	{Code: "EA653", Lat: 52.7, Lon: -174.1, FoF2: 6, M3000: 3},
	{Code: "EG931", Lat: 30.5, Lon: -86.5, FoF2: 6, M3000: 3},
	{Code: "FF051", Lat: 51.7, Lon: -1.5, FoF2: 6, M3000: 3},
	{Code: "GA762", Lat: 62.4, Lon: -145.0, FoF2: 6, M3000: 3},
	{Code: "GM037", Lat: 37.9, Lon: 14.0, FoF2: 6, M3000: 3},
	{Code: "GR13L", Lat: -33.3, Lon: 26.5, FoF2: 6, M3000: 3},
	{Code: "GU513", Lat: 13.6, Lon: 144.9, FoF2: 6, M3000: 3},
	{Code: "HE13N", Lat: -34.4, Lon: 19.2, FoF2: 6, M3000: 3},
	{Code: "HO54K", Lat: -42.9, Lon: 147.3, FoF2: 6, M3000: 3},
	{Code: "IF843", Lat: 43.8, Lon: -112.7, FoF2: 6, M3000: 3},
	{Code: "JI91J", Lat: -12.0, Lon: -76.8, FoF2: 6, M3000: 3},
	{Code: "JR055", Lat: 54.6, Lon: 13.4, FoF2: 6, M3000: 3},
	{Code: "LL721", Lat: 21.4, Lon: -158.1, FoF2: 6, M3000: 3},
	{Code: "LM42B", Lat: -21.8, Lon: 114.1, FoF2: 6, M3000: 3},
	{Code: "MHJ45", Lat: 42.6, Lon: -71.5, FoF2: 6, M3000: 3},
	{Code: "MZ152", Lat: 52.2, Lon: 21.1, FoF2: 6, M3000: 3},
	{Code: "ND61R", Lat: -19.1, Lon: -169.9, FoF2: 6, M3000: 3},
	{Code: "NI135", Lat: 35.0, Lon: 33.2, FoF2: 6, M3000: 3},
	{Code: "PE43K", Lat: -32.0, Lon: 116.1, FoF2: 6, M3000: 3},
	{Code: "PF765", Lat: 65.1, Lon: -147.4, FoF2: 6, M3000: 3},
	{Code: "PQ052", Lat: 50.0, Lon: 14.6, FoF2: 6, M3000: 3},
	{Code: "RM041", Lat: 41.8, Lon: 12.5, FoF2: 6, M3000: 3},
	{Code: "RO041", Lat: 41.9, Lon: 12.5, FoF2: 6, M3000: 3},
	{Code: "SO148", Lat: 47.6, Lon: 16.7, FoF2: 6, M3000: 3},
	{Code: "THJ76", Lat: 76.5, Lon: -68.4, FoF2: 6, M3000: 3},
	{Code: "TR169", Lat: 69.6, Lon: 19.2, FoF2: 6, M3000: 3},
	{Code: "TV51R", Lat: -19.6, Lon: 146.9, FoF2: 6, M3000: 3},
	{Code: "VT139", Lat: 40.6, Lon: 17.8, FoF2: 6, M3000: 3},
	{Code: "WA619", Lat: 19.3, Lon: 166.6, FoF2: 6, M3000: 3},
}

// replayGetter answers the service's asks from a replay of synthetic days,
// GIRO's held until the ask's context ends - a burst in flight - while hold
// is set; asking closes at the first GIRO ask held, and giroAsked counts
// GIRO's asks.
type replayGetter struct {
	replay    *ionomapstest.Replay
	hold      atomic.Bool
	giroAsked atomic.Int32
	once      sync.Once
	asking    chan struct{}
}

func (g *replayGetter) Get(ctx context.Context, rawURL, etag, lastModified string) (httpx.Answer, error) {
	if strings.Contains(rawURL, "lgdc.uml.edu") {
		g.giroAsked.Add(1)
		if g.hold.Load() {
			g.once.Do(func() { close(g.asking) })
			<-ctx.Done()
			return httpx.Answer{}, ctx.Err()
		}
	}
	r, err := g.replay.Fetch(ctx, rawURL, ionomaps.Validators{ETag: etag, LastModified: lastModified})
	if err != nil {
		return httpx.Answer{}, err
	}
	return httpx.Answer{Status: r.Status, Header: http.Header{}, Body: r.Body}, nil
}

// twoUpdates is NOAA's round and then, when GIRO is due, its burst.
func twoUpdates(lp *livePipelines) tty.PropagationResult {
	r := <-lp.propagationUpdate(context.Background(), ionomaps.UpdateOptions{})
	if r.Err == nil && r.Snapshot.Reason == ionomaps.StationsNext {
		r = <-lp.propagationUpdate(context.Background(), ionomaps.UpdateOptions{})
	}
	return r
}

// TestTheAppsEndStopsAnUpdateAndKeepsItsState is D-213 with D-39 at a quit
// (R6T-1, R7T-3): the app's end cancels its context before it waits, and
// GIRO's burst in flight - the hour before it read all 39 seeded stations,
// so the field it builds as it stops is a full one - stops with it and
// keeps GIRO's throttle state, the burst's hour spent, inside settleBound -
// under the race detector, inside settleBound times its slowdown; a restart
// within the hour asks GIRO nothing new.
func TestTheAppsEndStopsAnUpdateAndKeepsItsState(t *testing.T) {
	dir, file := t.TempDir(), filepath.Join(t.TempDir(), "propagation-throttle.json")
	day := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	var readings []time.Time
	for m := 0; m <= 120; m += 15 {
		readings = append(readings, day.Add(time.Duration(m)*time.Minute))
	}
	solar, err := ionomapstest.Write(dir, ionomapstest.Synthetic{
		Grids: []time.Time{day.Add(5 * time.Minute)}, GridFoF2: 6, F107: 100,
		Stations: seededStations, Readings: readings})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := ionomapstest.Open(dir, solar)
	if err != nil {
		t.Fatal(err)
	}
	replay := rec.Replay(ionomapstest.Options{})
	replay.SetNow(day.Add(30 * time.Minute))
	g := &replayGetter{replay: replay, asking: make(chan struct{})}
	svc, err := propagation.NewService(g, propagation.Options{Clock: replay.Now, StateFile: file})
	if err != nil {
		t.Fatal(err)
	}
	appCtx, appEnd := context.WithCancel(context.Background())
	lp := &livePipelines{ctx: appCtx}
	lp.propagation.once.Do(func() { lp.propagation.svc = svc })
	if r := twoUpdates(lp); r.Err != nil || r.Snapshot.Inputs.Stations < len(seededStations) {
		t.Fatalf("control: the first hour: %v, %d stations read; want all %d", r.Err, r.Snapshot.Inputs.Stations, len(seededStations))
	}
	replay.SetNow(day.Add(90 * time.Minute))
	g.hold.Store(true)
	if r := <-lp.propagationUpdate(context.Background(), ionomaps.UpdateOptions{}); r.Err != nil || r.Snapshot.Reason != ionomaps.StationsNext {
		t.Fatalf("control: the next hour's NOAA round: %v, %v; want GIRO due", r.Snapshot.Reason, r.Err)
	}
	out := lp.propagationUpdate(context.Background(), ionomaps.UpdateOptions{}) // the burst, on a session a quit never cancels
	select {
	case <-g.asking: // GIRO's burst is in flight
	case <-time.After(5 * time.Second):
		t.Fatal("the burst never asked GIRO")
	}
	if svc.Settle(50 * time.Millisecond) {
		t.Fatal("control: the burst ended before the app's end")
	}
	appEnd()
	if !svc.Settle(settleBound * raceSlowdown) {
		t.Fatalf("the burst outlived the app's end by more than %v", settleBound*raceSlowdown)
	}
	<-out
	again := &replayGetter{replay: replay, asking: make(chan struct{})}
	restarted, err := propagation.NewService(again, propagation.Options{Clock: replay.Now, StateFile: file})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		<-restarted.Ask(context.Background(), ionomaps.UpdateOptions{})
	}
	if n := again.giroAsked.Load(); n != 0 {
		t.Errorf("a restart within the hour asked GIRO %d times; want the burst's hour kept at the app's end", n)
	}
}
