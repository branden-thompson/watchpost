package tty

// map_describe_test.go — 0.18.0 W1.4 with W9.2 folded in (D-60): the text
// description, built from the library's Report joined to watchpost's own
// alerts (FR-7.4, D-26, D-29, D-55), in the words M1 asks about.

import (
	"context"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

var windExpires = time.Date(2026, 8, 24, 7, 0, 0, 0, time.UTC)

// boxFeed is one Severe alert over a box of longitudes around Oceanside's
// latitude; the snapshot carries the same alert, as watchpost holds it.
func boxFeed(west, east float64, inMissing bool) func(context.Context, *snapshot.Snapshot, *snapshot.Location) MapFeed {
	return func(context.Context, *snapshot.Snapshot, *snapshot.Location) MapFeed {
		ring := []tuimaps.LonLat{{Lon: west, Lat: 33.0}, {Lon: east, Lat: 33.0}, {Lon: east, Lat: 33.4}, {Lon: west, Lat: 33.4}, {Lon: west, Lat: 33.0}}
		f := MapFeed{Overlays: []tuimaps.Overlay{{ID: "alert/w1", Valid: time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC), Keeps: 6 * time.Hour,
			Features: []tuimaps.Feature{{Kind: tuimaps.Polygon, Rings: [][]tuimaps.LonLat{ring}, Role: tuimaps.AlertSevere,
				Label: "Wind Warning", Severity: tuimaps.SeveritySevere, Expires: windExpires, ID: "w1"}}}}}
		if inMissing {
			f.InMissing = map[string]bool{"w1": true}
		}
		return f
	}
}

// describedDash opens the map under --ascii on Oceanside with one alert, and
// returns what the window says.
func describedDash(t *testing.T, feed func(context.Context, *snapshot.Snapshot, *snapshot.Location) MapFeed) string {
	t.Helper()
	return describedIn(t, feed, render.UnitC)
}

// describedIn is describedDash in a unit system.
func describedIn(t *testing.T, feed func(context.Context, *snapshot.Snapshot, *snapshot.Location) MapFeed, units render.Units) string {
	t.Helper()
	d := mapDash(t, Config{ASCII: true, MapFeed: feed})
	d.units = units
	s := placedSnap()
	s.Locations[0].TZ = "America/Los_Angeles"
	s.Locations[0].Alerts = []snapshot.Alert{{ID: "w1", Event: "Wind Warning", Severity: "Severe", Expires: windExpires}}
	m, _ := d.Update(SnapshotMsg{Snap: s})
	d = m.(Dashboard)
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	return stripANSITest(d.View().Content)
}

// TestTheDescriptionSaysWhatCoversThePlace is W1.4 (FR-7.4, FR-1.7): under
// --ascii the window says, in words, which alert covers the place, how
// severe, how far its edge is and which way, and until when - and never a
// coordinate, a glyph that does not speak, or "you" (FR-7.3, D-29).
func TestTheDescriptionSaysWhatCoversThePlace(t *testing.T) {
	out := describedDash(t, boxFeed(-117.6, -117.1, false))
	for _, want := range []string{"Wind Warning, severe, covers Oceanside, CA", "kilometres to the", "in effect until"} {
		if !strings.Contains(out, want) {
			t.Errorf("the description does not say %q:\n%s", want, out)
		}
	}
	lower := strings.ToLower(out)
	for _, never := range []string{" you ", " your ", "33.2", "117.38", "·", "⠀"} {
		if strings.Contains(lower, never) {
			t.Errorf("the description carries %q:\n%s", never, out)
		}
	}
}

// TestTheDescriptionSpeaksM1sWords is W1.4 against M1's rule: an alert whose
// edge is within 15 km stops short; one farther lies to one side; one whose
// missing zone holds the place covers it by its zone.
func TestTheDescriptionSpeaksM1sWords(t *testing.T) {
	for _, c := range []struct {
		name       string
		west, east float64
		missing    bool
		want       string
	}{
		{"an edge 7 km off", -117.30, -117.0, false, "stops short of Oceanside, CA"},
		{"an edge 60 km off", -116.7, -116.4, false, "lies to one side of Oceanside, CA"},
		{"the place in a missing zone", -116.7, -116.4, true, "covers Oceanside, CA by a zone that could not be drawn"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if out := describedDash(t, boxFeed(c.west, c.east, c.missing)); !strings.Contains(out, c.want) {
				t.Errorf("want %q:\n%s", c.want, out)
			}
		})
	}
}

// TestNoAlertIsAStatedState is W1.4: with no alert on the map the description
// says so, rather than saying nothing.
func TestNoAlertIsAStatedState(t *testing.T) {
	out := describedDash(t, func(context.Context, *snapshot.Snapshot, *snapshot.Location) MapFeed { return MapFeed{} })
	if !strings.Contains(out, "No alert on the map covers or comes near Oceanside, CA.") {
		t.Errorf("an empty map is not stated:\n%s", out)
	}
}

// TestTheDescriptionFollowsTheUnits is W1.4 with the station's units: a
// listener in Fahrenheit hears miles and Fahrenheit, one in Celsius
// kilometres and Celsius.
func TestTheDescriptionFollowsTheUnits(t *testing.T) {
	f := describedIn(t, boxFeed(-117.30, -117.0, false), render.UnitF)
	if !strings.Contains(f, "miles to the") || !strings.Contains(f, "degrees Fahrenheit") {
		t.Errorf("in Fahrenheit the description says:\n%s", f)
	}
	c := describedIn(t, boxFeed(-117.30, -117.0, false), render.UnitC)
	if !strings.Contains(c, "kilometres to the") || !strings.Contains(c, "degrees Celsius") {
		t.Errorf("in Celsius the description says:\n%s", c)
	}
}

// TestAUnitChangeRedrawsTheDescription is D-45's units row (W2.3): switching
// units with the map open redraws it, so the description's distances follow.
func TestAUnitChangeRedrawsTheDescription(t *testing.T) {
	d := mapDash(t, Config{ASCII: true, MapFeed: boxFeed(-117.30, -117.0, false)})
	d.units = render.UnitF
	s := placedSnap()
	s.Locations[0].Alerts = []snapshot.Alert{{ID: "w1", Event: "Wind Warning", Severity: "Severe", Expires: windExpires}}
	m, _ := d.Update(SnapshotMsg{Snap: s})
	d = m.(Dashboard)
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	d = pressCode(d, 'c', "c")
	if out := stripANSITest(d.View().Content); !strings.Contains(out, "kilometres to the") {
		t.Errorf("after c the description still says:\n%s", out)
	}
}
