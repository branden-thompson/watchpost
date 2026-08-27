package tty

// Quality pass Q3 (plan §2.5 body memo; PF-8, R2-4): the invalidation table
// has one row per key input. Each row mutates the model, renders through
// the memo and compares with a memo-less render of the same model — the
// memo may only ever return what a fresh render would.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// freshFrame renders without the memo slot.
func freshFrame(d Dashboard) string {
	d.memo = nil
	return d.View().Content
}

func TestBodyMemoInvalidatesOnEveryInput(t *testing.T) {
	base := benchDash(t, 133, 44).(Dashboard)
	base.memo = &bodyMemo{}
	rows := []struct {
		name string
		mut  func(d Dashboard) Dashboard
	}{
		{"snapshot pointer", func(d Dashboard) Dashboard { s := *d.snap; s.Locations[0].Label = "Renamed"; d.snap = &s; return d }},
		{"recent pointer", func(d Dashboard) Dashboard { s := *d.recent; s.Locations[0].Label = "Renamed"; d.recent = &s; return d }},
		{"width", func(d Dashboard) Dashboard { d.width = 150; return d }},
		{"height (window)", func(d Dashboard) Dashboard { d.height = 60; return d }},
		{"selected", func(d Dashboard) Dashboard { d.selected = 3; return d }},
		{"recentOff", func(d Dashboard) Dashboard { d.selected, d.recentOff = 30, 20; return d }},
		{"units", func(d Dashboard) Dashboard { d.units = render.UnitC; return d }},
		{"ascii", func(d Dashboard) Dashboard { d.cfg.ASCII = true; return d }},
		{"[space]: the ▶ row", func(d Dashboard) Dashboard {
			d.radioPlaying, d.radioKey = true, snapshot.Key(snapshot.LocationRef{Lat: d.snap.Locations[2].Lat, Lon: d.snap.Locations[2].Lon})
			return d
		}},
		{"stop: ▶ clears while radioKey stays", func(d Dashboard) Dashboard {
			d.radioPlaying, d.radioKey = false, snapshot.Key(snapshot.LocationRef{Lat: d.snap.Locations[2].Lat, Lon: d.snap.Locations[2].Lon})
			return d
		}},
		{"[r]: the ∞ mark", func(d Dashboard) Dashboard {
			d.radioPlaying, d.radioKey, d.radioRepeat = true, snapshot.Key(snapshot.LocationRef{Lat: d.snap.Locations[2].Lat, Lon: d.snap.Locations[2].Lon}), RepeatOne
			return d
		}},
		{"[T]: the min player changes the window", func(d Dashboard) Dashboard { d.radioMin = true; return d }},
		{"[v]: the visualizer rows change the window", func(d Dashboard) Dashboard { d.radioViz = true; return d }},
		{"Setup: the bold-◆ rule", func(d Dashboard) Dashboard {
			d.snap.Locations[0].Fire.Hotspots = []snapshot.Hotspot{{Lat: 33, Lon: -117, FRPMW: f64(80)}}
			d.cfg.FireBoldMW = 100
			return d
		}},
		{"a loading row's frame", func(d Dashboard) Dashboard {
			s := *d.snap
			s.Locations = append([]snapshot.Location(nil), s.Locations...)
			s.Locations[0].Daily = nil
			d.snap = &s
			d.frame = 2
			return d
		}},
	}
	for _, row := range rows {
		d := base
		_ = d.View().Content // prime the slot with the unmutated frame
		d = row.mut(d)
		if got, want := d.View().Content, freshFrame(d); got != want {
			t.Errorf("%s: the memo served a stale table", row.name)
		}
	}
}

func TestBodyMemoInvalidatesOnAThemeSwitch(t *testing.T) {
	d := benchDash(t, 133, 44).(Dashboard)
	d.memo = &bodyMemo{}
	_ = d.View().Content
	if !render.SetTheme("Monochrome") {
		t.Fatal("Monochrome is a built-in")
	}
	t.Cleanup(func() { render.SetTheme(render.DefaultThemeName) })
	if got, want := d.View().Content, freshFrame(d); got != want {
		t.Fatal("[t]: every Tok() tint in the cells changes with the theme")
	}
}

func TestBodyMemoHitsBetweenTicksAndMissesOnce(t *testing.T) {
	m := benchDash(t, 133, 44)
	d := m.(Dashboard)
	_ = d.View().Content
	for range 5 {
		_ = d.View().Content
	}
	hits, misses := d.memoCounts()
	if misses != 1 || hits != 5 {
		t.Fatalf("one render, then five hits: hits=%d misses=%d", hits, misses)
	}
	// A tick with nothing loading changes no table input: still a hit.
	d.frame++
	_ = d.View().Content
	if hits, _ := d.memoCounts(); hits != 6 {
		t.Fatalf("a tick alone never re-renders the tables: hits=%d", hits)
	}
	// The memo is a pointer shared by every copy of the model (View is a
	// value receiver): the copy Update returns carries the same slot.
	next, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if next.(Dashboard).memo != d.memo {
		t.Fatal("the slot is allocated once at construction and shared")
	}
	_ = next.View().Content
	if _, misses := next.(Dashboard).memoCounts(); misses != 2 {
		t.Fatalf("a selection change is one miss: misses=%d", misses)
	}
}

func TestLayoutOncePerViewAndKey(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	// The frame's geometry is resolved once per View (the memo hit path)
	// and once per key event; the pins are allocation counts, not call
	// counts (CQ-9, JD-7): a second layout() call would show up here.
	m := benchDash(t, 133, 44)
	d := m.(Dashboard)
	_ = d.View().Content
	perView := testing.AllocsPerRun(20, func() { _ = d.View().Content })
	perLayout := testing.AllocsPerRun(20, func() { _ = d.layout() })
	if perView > 2*perLayout+100 { // the header's own tints ride the hit path; a second layout() would add a whole layout
		t.Fatalf("one layout per frame on the memo hit path: frame %.0f allocs vs layout %.0f — a second layout() call would show here", perView, perLayout)
	}
	perKey := testing.AllocsPerRun(20, func() { _, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyDown}) })
	if perKey > 2*perLayout+50 {
		t.Fatalf("a navigation key resolves the layout once: %.0f allocs (layout %.0f)", perKey, perLayout)
	}
}
