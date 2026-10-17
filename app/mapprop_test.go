package app

import (
	"context"
	ionomaps "github.com/branden-thompson/go-ionomaps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
	hosts := propagationHostList()
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
		return propagation.New(solarDays{}, propagation.Options{Clock: solarNoon})
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
	for _, s := range propagationSources() {
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
