package app

import (
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestAnUnchangedOverlayIsTheSameOverlay is W14's P-6: fire, buoys, tides,
// quakes and AirNow were stamped Valid with the moment of each ask, so the
// same data a minute later was a new overlay - handed to the library again,
// and prepared again, on every answer. Stamped to the step they fall in,
// unchanged data compares the same across asks; a changed reading does not.
func TestAnUnchangedOverlayIsTheSameOverlay(t *testing.T) {
	t1 := time.Date(2026, 9, 29, 14, 0, 30, 0, time.UTC)
	t2 := t1.Add(time.Minute)
	view := tty.MapView{W: -121, S: 32, E: -114, N: 36}
	same := func(name string, a, b []tuimaps.Overlay) {
		t.Helper()
		if len(a) == 0 || len(a) != len(b) {
			t.Fatalf("%s: control: %d and %d overlays", name, len(a), len(b))
		}
		for i := range a {
			if !tty.SameOverlay(a[i], b[i]) {
				t.Errorf("%s: the same data a minute later is a new overlay (Valid %v, then %v)", name, a[i].Valid, b[i].Valid)
			}
		}
	}
	obs := []ndbc.Obs{{ID: "46025", Lat: 33.765, Lon: -119.077, At: t1.Add(-10 * time.Minute), WaveM: f64(1.2), WaterC: f64(22.6)}}
	b1, _ := buoyOverlay(obs, view, t1, true)
	b2, _ := buoyOverlay(obs, view, t2, true)
	same("buoys", []tuimaps.Overlay{b1}, []tuimaps.Overlay{b2})
	changed := []ndbc.Obs{obs[0]}
	changed[0].WaveM = f64(2.4)
	if b3, _ := buoyOverlay(changed, view, t2, true); tty.SameOverlay(b1, b3) {
		t.Error("control: a changed reading compared the same - the check would pass anything")
	}
	next := func(string) (snapshot.TideEvent, bool) {
		return snapshot.TideEvent{Time: t1.Add(3 * time.Hour), Type: "H", Height: 1.2}, true
	}
	st := []coops.Station{{ID: "9410230", Name: "La Jolla", Lat: 32.87, Lon: -117.26}}
	td1, _ := tideOverlay(st, view, t1, true, render.Clock12, time.UTC, next)
	td2, _ := tideOverlay(st, view, t2, true, render.Clock12, time.UTC, next)
	same("tides", []tuimaps.Overlay{td1}, []tuimaps.Overlay{td2})
	mag := 3.1
	q := []globalfeed.Event{{ID: "q", Class: globalfeed.ClassQuake, Lat: 33.5, Lon: -117, HasPoint: true, At: t1.Add(-5 * time.Hour), Quake: &globalfeed.QuakeDetail{Mag: &mag}}}
	q1, _ := quakeOverlay(q, t1, render.Clock12)
	q2, _ := quakeOverlay(q, t2, render.Clock12)
	same("quakes", []tuimaps.Overlay{q1}, []tuimaps.Overlay{q2})
	same("fire", fireOverlays(someFire(), fireView, fire.DefaultRules(), t1), fireOverlays(someFire(), fireView, fire.DefaultRules(), t2))
	areas := []airquality.Area{{Name: "San Diego", State: "CA", Lat: 32.7, Lon: -117.1, Now: &airquality.Reading{AQI: 42}}}
	a1, _ := airnowOverlays(areas, view, time.Time{}, t1)
	a2, _ := airnowOverlays(areas, view, time.Time{}, t2)
	same("AirNow", a1, a2)
}
