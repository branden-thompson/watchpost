package app

// mapmarine_test.go — 0.18.0 D-127, D-128: the sea's stations - buoys with
// their latest readings, tide stations with their next high or low.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// marineView is off Southern California.
var marineView = tty.MapView{W: -121, S: 32, E: -117, N: 34.5}

func f64(v float64) *float64 { return &v }

// TestBuoysAreDrawnWithTheirReadings is D-127: each station that read in the
// last two hours and is in view, a marker in the buoy's role labelled with its
// waves and water - "4ft 73°" - or its wind where it has no waves - "12kt" -
// in the listener's units.
func TestBuoysAreDrawnWithTheirReadings(t *testing.T) {
	now := time.Date(2026, 9, 28, 14, 0, 0, 0, time.UTC)
	obs := []ndbc.Obs{
		{ID: "46025", Lat: 33.765, Lon: -119.077, At: now.Add(-10 * time.Minute), WaveM: f64(1.2), WaterC: f64(22.6), WindMS: f64(4)},
		{ID: "46086", Lat: 32.504, Lon: -118.029, At: now.Add(-10 * time.Minute), WindMS: f64(6)},
		{ID: "stale", Lat: 33, Lon: -118, At: now.Add(-3 * time.Hour), WaveM: f64(1)},
		{ID: "far", Lat: 45, Lon: -125, At: now, WaveM: f64(1)},
		{ID: "silent", Lat: 33.1, Lon: -118.1, At: now},
	}
	labels := func(imperial bool) map[string]string {
		out := map[string]string{}
		o, _ := buoyOverlay(obs, marineView, now, imperial)
		if !strings.HasPrefix(o.ID, tty.BuoyLayer+"/") {
			t.Errorf("the buoys are in overlay %q; want the Buoys row's", o.ID)
		}
		for _, f := range o.Features {
			if f.Role != tuimaps.Buoy {
				t.Errorf("%s is drawn in %v; want the buoy's role", f.ID, f.Role)
			}
			out[f.ID] = f.Label
		}
		return out
	}
	got := labels(true)
	if len(got) != 2 || got["46025"] != "4ft 73°" || got["46086"] != "12kt" {
		t.Errorf("the buoys drawn are %v; want 46025 at 4ft 73° and 46086 at 12kt, the stale, the far and the silent left out", got)
	}
	if m := labels(false); m["46025"] != "1.2m 23°" || m["46086"] != "12kt" {
		t.Errorf("in metric the buoys read %v", m)
	}
}

// TestTideStationsAreLabelledWhenFewAreInView is D-128: every tide station in
// view a marker in the tide's role; labelled with its next high or low while
// twenty or fewer are in view, the markers alone past that - and asked for
// no prediction then.
func TestTideStationsAreLabelledWhenFewAreInView(t *testing.T) {
	now := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	asked := 0
	next := func(id string) (snapshot.TideEvent, bool) {
		asked++
		return snapshot.TideEvent{Type: "H", Height: 1.193, Time: time.Date(2026, 8, 24, 16, 1, 0, 0, time.UTC)}, true
	}
	few := []coops.Station{{ID: "a", Name: "La Jolla", Lat: 32.87, Lon: -117.26}, {ID: "far", Lat: 45, Lon: -124}}
	o, _ := tideOverlay(few, marineView, now, true, render.ClockByKey("12h"), time.UTC, next)
	marks := o.Features
	if len(marks) != 1 || marks[0].Label != "H 3.9ft 4:01 PM" || marks[0].Role != tuimaps.Tide || asked != 1 {
		t.Fatalf("the few in view are %+v (%d asked); want La Jolla, H 3.9ft 4:01 PM", marks, asked)
	}
	var many []coops.Station
	for i := range tideLabelMost + 1 {
		many = append(many, coops.Station{ID: string(rune('a' + i)), Lat: 33, Lon: -118 + float64(i)*0.01})
	}
	asked = 0
	o, _ = tideOverlay(many, marineView, now, true, render.ClockByKey("12h"), time.UTC, next)
	marks = o.Features
	if len(marks) != tideLabelMost+1 || marks[0].Label != "" || asked != 0 {
		t.Errorf("%d in view: %d markers, the first labelled %q, %d asked; want markers alone, none asked", tideLabelMost+1, len(marks), marks[0].Label, asked)
	}
}

// TestTheSeasStationsAreAskedOnlyWhileOn is D-127 and D-128's wiring, from
// the composition root's providers: each asked while its row is on, and not
// otherwise - the tides are a request a station.
func TestTheSeasStationsAreAskedOnlyWhileOn(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?"+r.URL.Query().Get("type")+r.URL.Query().Get("product"))
		switch {
		case strings.HasSuffix(r.URL.Path, "latest_obs.txt"):
			b, _ := os.ReadFile("../domains/marine/ndbc/testdata/latest_obs.txt")
			_, _ = w.Write(freshObs(b, time.Now()))
		case strings.HasSuffix(r.URL.Path, "stations.json"):
			b, _ := os.ReadFile("../domains/marine/coops/testdata/stations_" + r.URL.Query().Get("type") + ".json")
			_, _ = w.Write(b)
		default:
			b, _ := os.ReadFile("../domains/marine/coops/testdata/predictions.json")
			_, _ = w.Write(b)
		}
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	lp := &livePipelines{marine: []snapshot.Provider{ndbc.New(c, srv.URL), coops.New(c, srv.URL)}}
	ask := tty.MapAsk{Snap: &snapshot.Snapshot{}, Region: geo.RegionContiguous, View: marineView}
	if in := lp.mapInputsFetching(context.Background(), ask); len(paths) != 0 || len(in.buoys) != 0 || len(in.tides) != 0 {
		t.Fatalf("with both rows off the map asked %v", paths)
	}
	ask.Buoys, ask.Tides = true, true
	in := lp.mapInputsFetching(context.Background(), ask)
	if len(in.buoys) == 0 || len(in.tides) == 0 {
		t.Fatalf("with both rows on: %d buoys, %d tide stations; asked %v", len(in.buoys), len(in.tides), paths)
	}
	feed := lp.mapFeedWith(context.Background(), in, func(snapshot.Location) []string { return nil })
	kinds := map[string]bool{}
	for _, o := range feed.Overlays {
		key, _, _ := strings.Cut(o.ID, "/")
		kinds[key] = true
		if tm := feed.Times[o.ID]; !tm.Happened {
			t.Errorf("%s is not timed as a thing so now", o.ID)
		}
	}
	if !kinds[tty.BuoyLayer] || !kinds[tty.TideLayer] {
		t.Errorf("the feed carries %v; want buoys and tides", kinds)
	}
}

// TestTheSeasStationsAreCostedAndNamed is D-23 and FR-9.4 for D-127, D-128.
func TestTheSeasStationsAreCostedAndNamed(t *testing.T) {
	for _, key := range []string{tty.BuoyLayer, tty.TideLayer} {
		found := false
		for _, l := range mapLayers {
			if l.key == key {
				found = true
				if l.on {
					t.Errorf("%s is on by default; D-127 and D-128 have it off", key)
				}
			}
		}
		if !found {
			t.Errorf("%s is not registered", key)
		}
	}
	if b, r := buoyLayerCost(mapInputs{region: geo.RegionContiguous}); b <= 0 || r != 1 {
		t.Errorf("the buoys cost %d bytes in %d requests; want NDBC's one file", b, r)
	}
	hosts := map[string]bool{}
	for _, s := range mapSourceList() {
		hosts[s.Host] = true
	}
	if !hosts["www.ndbc.noaa.gov"] || !hosts["api.tidesandcurrents.noaa.gov"] {
		t.Errorf("the Status window's map list lacks NDBC or CO-OPS: %v", hosts)
	}
}

// freshObs is NDBC's file read ten minutes ago: the recorded file's readings
// were hours old two hours after it was recorded, and a buoy past two hours
// is not drawn (D-127) - the wiring test failed on the clock alone.
func freshObs(raw []byte, now time.Time) []byte {
	at := strings.Fields(now.UTC().Add(-10 * time.Minute).Format("2006 01 02 15 04"))
	lines := strings.Split(string(raw), "\n")
	for i, line := range lines {
		f := strings.Fields(line)
		if strings.HasPrefix(line, "#") || len(f) < 8 {
			continue
		}
		copy(f[3:8], at)
		lines[i] = strings.Join(f, " ")
	}
	return []byte(strings.Join(lines, "\n"))
}
