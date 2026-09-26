package tty

// map_bound_test.go — 0.18.0 W4 with W9.1 folded in (D-60): the map is never
// wider than the region its selected place is in (FR-2.1, D-28), and a place
// in no region is a stated state (FR-2.5). M2's instrument: every frame the
// suite draws here is checked against the region.

import (
	"math"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// frameBox is the ground a drawn frame covers: its centre, zoom and size in
// the library's published scale (256-dot tiles, braille's 2x4 dots a cell).
func frameBox(v mapView) (w, s, e, n float64) {
	world := 256 * math.Exp2(v.zoom)
	halfLon := float64(v.size.Cols*2) / world * 180
	y := mercY(v.centre.Lat)
	halfY := float64(v.size.Rows*4) / world / 2
	return v.centre.Lon - halfLon, latOfY(y + halfY), v.centre.Lon + halfLon, latOfY(y - halfY)
}

func mercY(lat float64) float64 {
	r := lat * math.Pi / 180
	return (1 - math.Log(math.Tan(r)+1/math.Cos(r))/math.Pi) / 2
}

func latOfY(y float64) float64 {
	return math.Atan(math.Sinh(math.Pi*(1-2*y))) * 180 / math.Pi
}

// inside reports whether a frame is within a region, to a hundredth of a
// degree, crossing the antimeridian where the region does.
func inside(v mapView, r geo.Region) bool {
	const slack = 0.01
	w, s, e, n := frameBox(v)
	if s < r.S-slack || n > r.N+slack {
		return false
	}
	width := r.E - r.W
	if width < 0 {
		width += 360
	}
	west := math.Mod(w-r.W+720, 360) // how far east of the region's west edge the frame starts
	return west <= width+slack && west+(e-w) <= width+slack
}

// driveEverywhere pans and zooms as far as the keys go, and resizes, drawing
// every step.
func driveEverywhere(d Dashboard) Dashboard {
	for range 12 {
		d = pressCode(d, '-', "-")
	}
	for _, k := range []rune{tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown} {
		for range 40 {
			d = pressCode(d, k, "")
		}
	}
	for _, size := range [][2]int{{200, 60}, {80, 24}, {133, 44}} {
		m, _ := d.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		d = m.(Dashboard)
		for range 12 {
			d = pressCode(d, '-', "-")
		}
	}
	return d
}

// TestNoFrameIsWiderThanTheRegion is W4.2, W4.4 and W9.1 (FR-2.1, FR-2.2,
// FR-2.4 as HR-3): from the first frame on, through every pan, zoom and
// resize, no frame is wider than the region it is bound to - the first the
// selected place's, the contiguous US for Oceanside and Alaska across the
// antimeridian for Adak, and after a pan across an edge the neighbour's
// (D-77): one region at a time (D-28).
func TestNoFrameIsWiderThanTheRegion(t *testing.T) {
	for _, c := range []struct {
		name     string
		lat, lon float64
		region   string
	}{{"Oceanside, CA", 33.2, -117.38, geo.RegionContiguous}, {"Adak, AK", 51.88, -176.66, geo.RegionAlaska}, {"Hilo, HI", 19.72, -155.08, geo.RegionHawaii}} {
		t.Run(c.name, func(t *testing.T) {
			s := placedSnap()
			s.Locations[0].Label, s.Locations[0].Lat, s.Locations[0].Lon = c.name, c.lat, c.lon
			d := mapDash(t, Config{})
			m, _ := d.Update(SnapshotMsg{Snap: s})
			d = m.(Dashboard)
			views := &[]mapView{}
			d.mapPane.views = views
			d, _ = pressKey(d, "g")
			d = driveEverywhere(d)
			r, _ := geo.RegionOf(c.lat, c.lon)
			if r.Name != c.region {
				t.Fatalf("%s is in %q", c.name, r.Name)
			}
			if len(*views) < 100 {
				t.Fatalf("%d frames drawn, so this proves little", len(*views))
			}
			if (*views)[0].region.Name != r.Name {
				t.Fatalf("the first frame is bound to %q, not the place's %s", (*views)[0].region.Name, r.Name)
			}
			for i, v := range *views {
				if v.region.Name == "" || !inside(v, v.region) {
					w, s, e, n := frameBox(v)
					t.Fatalf("frame %d (%+v) covers %.2f,%.2f to %.2f,%.2f: wider than %s", i, v, w, s, e, n, v.region.Name)
				}
			}
		})
	}
}

// TestAPlaceInNoRegionIsAStatedState is W4.1 (FR-2.5): the window says the
// place is outside what the map covers, and draws no wider view instead.
func TestAPlaceInNoRegionIsAStatedState(t *testing.T) {
	s := placedSnap()
	s.Locations[0].Label, s.Locations[0].Lat, s.Locations[0].Lon = "London", 51.5, -0.12
	d := mapDash(t, Config{})
	m, _ := d.Update(SnapshotMsg{Snap: s})
	d = m.(Dashboard)
	views := &[]mapView{}
	d.mapPane.views = views
	d, _ = pressKey(d, "g")
	out := stripANSITest(d.View().Content)
	if !strings.Contains(out, "London is outside") {
		t.Errorf("a place in no region is not stated:\n%s", out)
	}
	if len(*views) != 0 {
		t.Errorf("%d frames drawn for a place in no region", len(*views))
	}
	_ = tuimaps.Size{}
}
