package app

import (
	"context"
	ionomaps "github.com/branden-thompson/go-ionomaps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// waitingGetter answers no ask until the ask's context ends: a host that
// accepts and never answers. asked closes at its first ask.
type waitingGetter struct {
	once  sync.Once
	asked chan struct{}
}

func (g *waitingGetter) Get(ctx context.Context, _, _, _ string) (httpx.Answer, error) {
	g.once.Do(func() { close(g.asked) })
	<-ctx.Done()
	return httpx.Answer{}, ctx.Err()
}

// TestTheAppsEndStopsAnUpdateAndKeepsItsState is D-213 with D-39 at a quit
// (R6T-1): the app's end cancels its context before it waits, and an update
// in flight - asked on a session the quit never cancelled - stops with it,
// keeping GIRO's throttle state inside settleBound, so a restart within the
// hour asks GIRO nothing new.
func TestTheAppsEndStopsAnUpdateAndKeepsItsState(t *testing.T) {
	file := filepath.Join(t.TempDir(), "propagation-throttle.json")
	g := &waitingGetter{asked: make(chan struct{})}
	svc, err := propagation.NewService(g, propagation.Options{StateFile: file})
	if err != nil {
		t.Fatal(err)
	}
	appCtx, appEnd := context.WithCancel(context.Background())
	lp := &livePipelines{ctx: appCtx}
	lp.propagation.once.Do(func() { lp.propagation.svc = svc })
	out := lp.propagationUpdate(context.Background(), ionomaps.UpdateOptions{}) // the session's context: a quit never cancels it
	select {
	case <-g.asked: // the update is in flight, asking
	case <-time.After(5 * time.Second):
		t.Fatal("the update never asked")
	}
	if svc.Settle(50 * time.Millisecond) {
		t.Fatal("control: the update ended before the app's end")
	}
	appEnd()
	if !svc.Settle(settleBound) {
		t.Fatalf("the update outlived the app's end by more than %v", settleBound)
	}
	if _, err := os.Stat(file); err != nil {
		t.Errorf("the state was not kept at the app's end: %v", err)
	}
	<-out
}
