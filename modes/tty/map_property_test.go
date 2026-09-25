package tty

// map_property_test.go — 0.18.0 W2.4, guard 3 (D-45): after any sequence of
// events, the lines the window prints are the lines a second map, built from
// the same inputs, renders. A fresh render on the same map would disturb what
// it checks, so the second map is new.

import (
	"context"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// propertyRuns is how many random sequences are tried: two hundred in every
// `go test`, the race run and each mutant's included, and W2.4's 10,000 under
// the `property` tag, which `make property` runs in a step of its own
// (map_property_full_test.go).
var propertyRuns = 200

// secondMapLines renders what the window should show from a new map given
// the window's inputs: its size, bound, view, units, nearby and overlays.
func secondMapLines(t *testing.T, d Dashboard, feed func(context.Context, MapAsk) MapFeed) []string {
	t.Helper()
	size := d.mapBodySize()
	m2, err := embeddedMap(size)
	if err != nil {
		t.Fatal(err)
	}
	defer m2.Close()
	d2 := d
	d2.mapPane.m, d2.mapPane.calls, d2.mapPane.where = m2, nil, nil
	d2.boundMap()
	centre, zoom := d.mapPane.m.Centre()
	miles := d.units != 0
	for _, step := range []func() error{
		func() error { return m2.Zoom(zoom) },
		func() error { return m2.Recentre(centre) },
		func() error { return m2.SetNearby(float64(d.mapNearbyKm)) },
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	m2.Units(miles, miles)
	for _, o := range d.feedForLayers(feed(context.Background(), d.mapAsk())).Overlays {
		if _, err := m2.Set(o); err != nil {
			t.Fatal(err)
		}
	}
	// A render asks for the tiles it needs, the work fetches them, and a work
	// that did something is drawn - the window's own order (a failed tile is
	// the library's to retry, and the window does not stop for it).
	f, err := m2.Render(size, d.now())
	for i := 0; err == nil && m2.Pending() > 0 && i < 50; i++ {
		if did, _ := m2.Work(context.Background()); did {
			f, err = m2.Render(size, d.now())
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return insetLines(f.Lines)
}

// TestThePrintedMapIsAFreshRender is guard 3 over random sequences of the
// events that move the map: pan, zoom, resize, units, new data and the
// library's tick, each sequence settled before the comparison.
func TestThePrintedMapIsAFreshRender(t *testing.T) {
	now := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	feed := boxFeed(-117.6, -117.1, false)
	fresh := func() Dashboard { // a window of its own per run: runs share no map
		d := mapDash(t, Config{MapFeed: feed})
		d.now = func() time.Time { return now }
		d, _ = pressKey(d, "g")
		return feedAndSettle(t, d)
	}
	sizes := []tea.WindowSizeMsg{{Width: 133, Height: 44}, {Width: 80, Height: 24}, {Width: 150, Height: 50}, {Width: 100, Height: 30}}
	keys := []tea.KeyPressMsg{{Code: tea.KeyLeft}, {Code: tea.KeyRight}, {Code: tea.KeyUp}, {Code: tea.KeyDown},
		{Code: '+', Text: "+"}, {Code: '-', Text: "-"}, {Code: 'c', Text: "c"}}
	runs := propertyRuns
	rng := rand.New(rand.NewPCG(18, 0))
	for run := range runs {
		d := fresh()
		var trail []string
		for range 1 + rng.IntN(6) {
			var msg tea.Msg
			switch k := rng.IntN(10); {
			case k < 6:
				msg = keys[rng.IntN(len(keys))]
			case k < 8:
				msg = sizes[rng.IntN(len(sizes))]
			case k < 9:
				msg = SnapshotMsg{Snap: placedSnap()}
			default:
				msg = mapTickMsg{at: d.mapPane.tickAt}
			}
			trail = append(trail, typeName(msg))
			m, _ := d.Update(msg)
			d = m.(Dashboard)
		}
		d = feedAndSettle(t, d)
		if !d.mapFits() {
			continue // under the floor nothing is drawn (FR-1.4); the floor's own tests hold it
		}
		if want := secondMapLines(t, d, feed); !slices.Equal(d.mapPane.lines, want) {
			t.Fatalf("run %d, after %v: the printed map is not a fresh render of the same inputs:\n%s", run, trail, lineDiff(d.mapPane.lines, want))
		}
		d.closeMap()
	}
}

// typeName names a message for a failing run's trail.
func typeName(msg tea.Msg) string {
	switch v := msg.(type) {
	case tea.KeyPressMsg:
		return "key " + v.String()
	case tea.WindowSizeMsg:
		return "size"
	case SnapshotMsg:
		return "data"
	case mapTickMsg:
		return "tick"
	}
	return "?"
}

// lineDiff shows the lines that differ, got over want, with the sizes.
func lineDiff(got, want []string) string {
	out := "got " + strconv.Itoa(len(got)) + " lines, want " + strconv.Itoa(len(want)) + "\n"
	for i := range max(len(got), len(want)) {
		var g, w string
		if i < len(got) {
			g = got[i]
		}
		if i < len(want) {
			w = want[i]
		}
		if g != w {
			out += "line " + strconv.Itoa(i) + ":\n  got  " + g + "\n  want " + w + "\n"
		}
	}
	return out
}
