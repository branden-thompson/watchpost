package airquality

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// kmlGet answers with the recorded contours.
type kmlGet struct {
	t     *testing.T
	asked []string
}

func (k *kmlGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	k.asked = append(k.asked, rawURL)
	return os.ReadFile(filepath.Join("testdata", "cur_aqi_combined.kml"))
}

// AIRNOW'S CONTOURS ARE READ BY CATEGORY, HOLES AND ALL (W19.4, D-193): the
// file's hour in UTC; a point takes the highest category whose polygon holds
// it - an Unhealthy contour nests in an Unhealthy-for-Sensitive-Groups one,
// a hole in it - and a point no contour holds has none.
func TestAirNowsContoursAreReadByCategory(t *testing.T) {
	get := &kmlGet{t: t}
	c, err := New(get, "").Contours(context.Background(), time.Date(2026, 10, 1, 4, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(get.asked) != 1 || get.asked[0] != "https://files.airnowtech.org/airnow/today/cur_aqi_combined.kml" {
		t.Errorf("asked %v; want AirNow's combined contours", get.asked)
	}
	if !c.Hour.Equal(time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)) || len(c.Areas) != 12 {
		t.Errorf("read the hour %v and %d contours; want 03:00 UTC and twelve", c.Hour, len(c.Areas))
	}
	for _, p := range []struct {
		name     string
		lat, lon float64
		want     int
		ok       bool
	}{
		{"Fresno's south-east, Unhealthy", 36.35, -119.25, Unhealthy, true},
		{"Los Angeles, Moderate", 34.05, -118.25, Moderate, true},
		{"Nevada, outside every contour kept", 39, -117, 0, false},
	} {
		got, ok := c.CategoryAt(p.lat, p.lon)
		if ok != p.ok || got != p.want {
			t.Errorf("%s: %d (%v); want %d (%v)", p.name, got, ok, p.want, p.ok)
		}
	}
}

// A HOLE IS NOT ITS POLYGON'S (D-193): a point in a contour's hole takes
// the category of the contour filling it, not the one around it.
func TestAContoursHoleIsNotItsOwn(t *testing.T) {
	outer := [][]Point{{{-10, -10}, {10, -10}, {10, 10}, {-10, 10}, {-10, -10}}, {{-2, -2}, {2, -2}, {2, 2}, {-2, 2}, {-2, -2}}}
	inner := [][]Point{{{-2, -2}, {2, -2}, {2, 2}, {-2, 2}, {-2, -2}}}
	c := Contours{Areas: []Contour{newContour(Good, outer), newContour(Moderate, inner)}}
	if got, ok := c.CategoryAt(5, 5); !ok || got != Good {
		t.Errorf("outside the hole: %d (%v); want Good", got, ok)
	}
	if got, ok := c.CategoryAt(0, 0); !ok || got != Moderate {
		t.Errorf("in the hole: %d (%v); want Moderate, the contour filling it", got, ok)
	}
	if _, ok := (Contours{Areas: []Contour{newContour(Good, outer)}}).CategoryAt(0, 0); ok {
		t.Error("a point in a hole no contour fills was given its polygon's category")
	}
}

// A RASTER IS THE POINTS' CATEGORIES (W19.4): a box's cells, each the
// highest category holding its middle - the same answer CategoryAt gives the
// point - and -1 where none does; the scanline is how the map affords it.
func TestARasterIsItsCellsCategories(t *testing.T) {
	c, err := New(&kmlGet{t: t}, "").Contours(context.Background(), time.Date(2026, 10, 1, 4, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	w, s, e, n, cols, rows := -124.5, 32.5, -114.0, 42.0, 120, 90
	got := c.Raster(w, s, e, n, cols, rows)
	seen := map[int]int{}
	for r := range rows {
		lat := n - (float64(r)+0.5)*(n-s)/float64(rows)
		for col := range cols {
			lon := w + (float64(col)+0.5)*(e-w)/float64(cols)
			want, ok := c.CategoryAt(lat, lon)
			if !ok {
				want = -1
			}
			if got[r*cols+col] != want {
				t.Fatalf("cell %d,%d (%.3f, %.3f) is %d; the point is %d", r, col, lat, lon, got[r*cols+col], want)
			}
			seen[want]++
		}
	}
	if seen[Unhealthy] == 0 || seen[Moderate] == 0 || seen[-1] == 0 {
		t.Errorf("the raster's categories %v; want Unhealthy, Moderate and none among them", seen)
	}
}

// THE HIGHEST CATEGORY HOLDING A POINT IS ITS OWN (D-193): where two
// contours overlap - a file whose contours do not tile - the worse air is
// said, in a point's answer and a raster's cell alike, whichever comes first.
func TestTheHighestCategoryWins(t *testing.T) {
	square := func(r float64) [][]Point {
		return [][]Point{{{-r, -r}, {r, -r}, {r, r}, {-r, r}, {-r, -r}}}
	}
	c := Contours{Areas: []Contour{newContour(Moderate, square(2)), newContour(Good, square(10))}}
	if got, ok := c.CategoryAt(0, 0); !ok || got != Moderate {
		t.Errorf("inside both: %d (%v); want Moderate, the worse", got, ok)
	}
	if got := c.Raster(-10, -10, 10, 10, 20, 20)[10*20+10]; got != Moderate {
		t.Errorf("the raster's middle cell is %d; want Moderate, the worse", got)
	}
}

// THE FILE'S HOUR IS NEVER LATER THAN NOW (D-193): a folder naming an hour
// ahead - a clock's skew, a zone misread - is read as now's hour; one naming
// none, likewise.
func TestTheContoursHourIsNeverAhead(t *testing.T) {
	now := time.Date(2026, 10, 1, 4, 30, 0, 0, time.UTC)
	if got := contourHour("1hr_Combined_AQI_USA_2026100103", now); !got.Equal(time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("the named hour read as %v", got)
	}
	for _, name := range []string{"1hr_Combined_AQI_USA_2026100109", "contours"} {
		if got := contourHour(name, now); !got.Equal(now.Truncate(time.Hour)) {
			t.Errorf("%q read as %v; want now's hour", name, got)
		}
	}
}

// THE CONTOURS ARE PARSED ONCE AN HOUR A FILE (W14 P-19, D-212): the file is
// served from the cache on every map ask while Air is on, and its ~2 MB of
// KML is parsed once for the hour it was read in, whoever asks.
func TestTheContoursAreParsedOnceAnHourAFile(t *testing.T) {
	p := New(&kmlGet{t: t}, "")
	at := time.Date(2026, 10, 1, 4, 30, 0, 0, time.UTC)
	for range 3 {
		if _, err := p.Contours(context.Background(), at); err != nil {
			t.Fatal(err)
		}
	}
	if n := p.ContourParses(); n != 1 {
		t.Errorf("three asks in the hour parsed the file %d times; want once", n)
	}
	if _, err := p.Contours(context.Background(), at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if n := p.ContourParses(); n != 2 {
		t.Errorf("the next hour parsed %d times in all; want the file parsed again for its hour", n)
	}
}

// TestARingsVertexThatIsNoPlaceIsLeftOut holds ring's guard (D-248): a vertex
// whose coordinates are not finite or not on Earth would put NaN or a far
// edge into every crossing of its ring.
func TestARingsVertexThatIsNoPlaceIsLeftOut(t *testing.T) {
	got := ring("-120,36,0 NaN,36,0 -119,Inf,0 -119,91,0 -181,37,0 -119,37,0")
	want := []Point{{Lon: -120, Lat: 36}, {Lon: -119, Lat: 37}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ring read %v, want %v", got, want)
	}
}
