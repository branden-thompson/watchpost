package tty

// map_same_test.go — 0.18.0 D-136 (and UAT-2 U2-13): an overlay the window
// already handed in is not handed in again. A missing value is NaN, and NaN
// is never equal to itself, so a wind grid whose gusts say "none here" would
// be new at every answer, handed in again, and blink.

import (
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// TestAGridWithMissingValuesIsTheSameGridAgain is D-136 with U2-13: two
// answers with the same grid - its missing values, its gusts said nowhere -
// are the same overlay; one that differs anywhere is not.
func TestAGridWithMissingValuesIsTheSameGridAgain(t *testing.T) {
	grid := func(gust float64) tuimaps.Overlay {
		g := tuimaps.Grid{West: -126, South: 23, East: -65, North: 51, Cols: 2, Rows: 1, Values: []float64{15, math.NaN()}, Gusts: []float64{gust, math.NaN()}}
		return tuimaps.WindGrid("wind/us/now", g, []float64{270, math.NaN()}, tuimaps.MilesPerHour, time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC))
	}
	if !SameOverlay(grid(math.NaN()), grid(math.NaN())) {
		t.Error("the same grid, its missing values missing in both, is new")
	}
	if !SameOverlay(grid(30), grid(30)) {
		t.Error("the same grid with a gust is new")
	}
	if SameOverlay(grid(30), grid(math.NaN())) || SameOverlay(grid(30), grid(31)) {
		t.Error("grids whose gusts differ are the same")
	}
	a, b := grid(30), grid(30)
	b.ID = "wind/us/d1"
	if SameOverlay(a, b) {
		t.Error("overlays that differ outside the grid are the same")
	}
}

// bigAlertFeed is PF-10's measure: 200 alerts of 8 zones, each a ring of 300
// points, as a feed answer carries them, every slice its own.
func bigAlertFeed() []tuimaps.Overlay {
	out := make([]tuimaps.Overlay, 0, 200)
	for a := range 200 {
		o := tuimaps.Overlay{ID: "alert/" + strconv.Itoa(a), Valid: time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC), Keeps: time.Hour}
		for z := range 8 {
			ring := make([]tuimaps.LonLat, 300)
			for i := range ring {
				ring[i] = tuimaps.LonLat{Lon: -100 + float64(a)/10 + float64(i)/1000, Lat: 35 + float64(z)/10}
			}
			o.Features = append(o.Features, tuimaps.Feature{Kind: tuimaps.Polygon, Rings: [][]tuimaps.LonLat{ring}, Label: "Wind Warning", ID: strconv.Itoa(z)})
		}
		out = append(out, o)
	}
	return out
}

// TestSameOverlayIsTheDeepAnswerCheaply is PF-10: every feed answer asks
// SameOverlay of every overlay, on the UI goroutine, so it must not walk
// every ring point by reflection - and its answer is exactly the deep
// comparison's, changed geometry included.
func TestSameOverlayIsTheDeepAnswerCheaply(t *testing.T) {
	a, b := bigAlertFeed(), bigAlertFeed()
	took := func(same func(x, y tuimaps.Overlay) bool) time.Duration {
		best := time.Duration(math.MaxInt64)
		for range 3 {
			start := time.Now()
			for i := range a {
				if !same(a[i], b[i]) {
					t.Fatal("two copies of one feed differ")
				}
			}
			best = min(best, time.Since(start))
		}
		return best
	}
	deep := took(func(x, y tuimaps.Overlay) bool { return reflect.DeepEqual(x, y) })
	ours := took(SameOverlay)
	t.Logf("over the feed: SameOverlay %v, the deep comparison %v", ours, deep)
	if ours*3 > deep {
		t.Errorf("SameOverlay took %v over the feed, the deep comparison %v: want a third of it at most", ours, deep)
	}
	base := bigAlertFeed()[0]
	changed := map[string]func(o *tuimaps.Overlay){
		"a point moved":     func(o *tuimaps.Overlay) { o.Features[7].Rings[0][299].Lat += 1e-9 },
		"a point gone":      func(o *tuimaps.Overlay) { o.Features[3].Rings[0] = o.Features[3].Rings[0][:299] },
		"a ring added":      func(o *tuimaps.Overlay) { o.Features[0].Rings = append(o.Features[0].Rings, o.Features[1].Rings[0]) },
		"a zone gone":       func(o *tuimaps.Overlay) { o.Features = o.Features[:7] },
		"a label":           func(o *tuimaps.Overlay) { o.Features[5].Label = "Gale Warning" },
		"no rings, not nil": func(o *tuimaps.Overlay) { o.Features[2].Rings = [][]tuimaps.LonLat{} },
		"rings nil":         func(o *tuimaps.Overlay) { o.Features[2].Rings = nil },
		"no features":       func(o *tuimaps.Overlay) { o.Features = []tuimaps.Feature{} },
		"features nil":      func(o *tuimaps.Overlay) { o.Features = nil },
		"the id":            func(o *tuimaps.Overlay) { o.ID = "alert/x" },
		"nothing at all":    func(*tuimaps.Overlay) {},
	}
	for name, change := range changed {
		o := bigAlertFeed()[0]
		change(&o)
		if got, want := SameOverlay(base, o), reflect.DeepEqual(base, o); got != want {
			t.Errorf("%s: SameOverlay says %v, the deep comparison %v", name, got, want)
		}
	}
	empty := tuimaps.Overlay{ID: "x", Features: []tuimaps.Feature{{Rings: [][]tuimaps.LonLat{{}}}}}
	none := tuimaps.Overlay{ID: "x", Features: []tuimaps.Feature{{Rings: [][]tuimaps.LonLat{nil}}}}
	if got, want := SameOverlay(empty, none), reflect.DeepEqual(empty, none); got != want {
		t.Errorf("an empty ring and a nil one: SameOverlay says %v, the deep comparison %v", got, want)
	}
}
