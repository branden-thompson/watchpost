package temperature

// reach_test.go — D-201 (UAT-2 U2-50): NDFD's temperature was blank over the
// Florida panhandle, northeast New England and southwest Texas - land whose
// nearest lattice point lay over the sea, Mexico or Canada, where NDFD answers
// nothing. A cell is blank now only where all four of its corners are, and
// NDFD's lattice is about twice as dense, asked a hundred points at a time.

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// A CELL BESIDE AN EMPTY POINT IS WEIGHED FROM THE OTHERS (D-201): only
// where all four corners are empty is a cell blank.
func TestACellIsBlankOnlyWhereAllFourCornersAre(t *testing.T) {
	l := Lattice{Box: geo.Box{W: 0, S: 0, E: 1, N: 1}, Cols: 2, Rows: 2}
	nan := math.NaN()
	f := l.InterpolateWide([]float64{10, nan, 10, 10}) // the north-east point over the sea
	for i, v := range f.Values {
		if !near(v, 10) {
			t.Fatalf("cell %d is %v; the three points that answered all say 10, and the land beside the sea is drawn", i, v)
		}
	}
	for i, v := range l.InterpolateWide([]float64{nan, nan, nan, nan}).Values {
		if !math.IsNaN(v) {
			t.Fatalf("cell %d is %v with every corner empty; want blank", i, v)
		}
	}
	// THE NEAREST RULE STAYS FOR EVERY OTHER FIELD (rain, UV, waves): only
	// temperature, feels like and wind take D-201's.
	if v := l.Interpolate([]float64{10, nan, 10, 10}).Values[Fine-1]; !math.IsNaN(v) {
		t.Errorf("Interpolate changed: the cell nearest the empty point is %v; want blank", v)
	}
}

// THE WIND TOO (D-201): a cell beside an empty point has a speed and a
// direction from the points that answered.
func TestTheWindIsWeighedFromTheCornersThatAnswered(t *testing.T) {
	l := Lattice{Box: geo.Box{W: 0, S: 0, E: 1, N: 1}, Cols: 2, Rows: 2}
	nan := math.NaN()
	sp, dirs := l.InterpolateWindWide([]float64{20, nan, 20, 20}, []float64{90, nan, 90, 90})
	for i := range sp.Values {
		if !near(sp.Values[i], 20) || math.Abs(dirs[i]-90) > 1e-6 {
			t.Fatalf("cell %d is %v from %v; want 20 from 90", i, sp.Values[i], dirs[i])
		}
	}
}

// NDFD'S LATTICE IS ABOUT TWICE AS DENSE (D-201): about twice the columns
// and rows of Open-Meteo's, which keeps its 80 points - it bills each.
func TestNDFDsLatticeIsTwiceAsDense(t *testing.T) {
	lower48 := geo.Box{W: -130, S: 23, E: -64, N: 51}
	om, ndfd := LatticeFor("us", lower48), NDFDLatticeFor("us", lower48)
	if om.Cols*om.Rows > MaxPoints {
		t.Errorf("Open-Meteo's lattice is %d points; want at most %d", om.Cols*om.Rows, MaxPoints)
	}
	if n := ndfd.Cols * ndfd.Rows; n > NDFDMaxPoints || ndfd.Cols < 2*om.Cols-1 || ndfd.Rows < 2*om.Rows-1 {
		t.Errorf("NDFD's lattice is %dx%d (%d points) against Open-Meteo's %dx%d; want about twice each way, at most %d", ndfd.Cols, ndfd.Rows, n, om.Cols, om.Rows, NDFDMaxPoints)
	}
	// THE DRAWN FIELD IS AS FINE AS BEFORE, NOT FOUR TIMES THE CELLS: a lattice
	// twice as dense is split half as fine.
	vals := make([]float64, ndfd.Cols*ndfd.Rows)
	f, g := ndfd.InterpolateWide(vals), om.Interpolate(make([]float64, om.Cols*om.Rows))
	if f.Cols > 2*g.Cols+Fine || f.Rows > 2*g.Rows+Fine {
		t.Errorf("NDFD's field is %dx%d cells against Open-Meteo's %dx%d", f.Cols, f.Rows, g.Cols, g.Rows)
	}
}

// pointsGet is NDFD answering for whatever points it is asked: each point's
// hour at its latitude in Fahrenheit, so an answer's every value says which
// point it is.
type pointsGet struct{ asks []string }

func (g *pointsGet) GetText(_ context.Context, raw string, _ ...httpx.Option) ([]byte, error) {
	g.asks = append(g.asks, raw)
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><dwml><data>`)
	pts := strings.Fields(u.Query().Get("listLatLon"))
	for i, p := range pts {
		ll := strings.Split(p, ",")
		fmt.Fprintf(&b, `<location><location-key>point%d</location-key><point latitude="%s" longitude="%s"/></location>`, i+1, ll[0], ll[1])
	}
	b.WriteString(`<time-layout><layout-key>k-p1h-n1-1</layout-key><start-valid-time>2026-09-27T01:00:00Z</start-valid-time></time-layout>`)
	for i, p := range pts {
		lat := strings.Split(p, ",")[0]
		fmt.Fprintf(&b, `<parameters applicable-location="point%d"><temperature type="hourly" units="Fahrenheit" time-layout="k-p1h-n1-1"><value>%s</value></temperature></parameters>`, i+1, lat)
	}
	b.WriteString(`</data></dwml>`)
	return []byte(b.String()), nil
}

// NDFD IS ASKED A HUNDRED POINTS AT A TIME (D-201): it answers only the first
// hundred of a request, silently, so a denser lattice is several asks - each
// point asked once a kind, every answer read into the one series - and the
// recorder's hour asks the very addresses Fetch's hour did, sharing the cache.
func TestNDFDIsAskedAHundredPointsAtATime(t *testing.T) {
	l := NDFDLatticeFor("us", geo.Box{W: -130, S: 23, E: -64, N: 51})
	get := &pointsGet{}
	s, err := NewNDFD(get, "").Fetch(context.Background(), l, captured)
	if err != nil {
		t.Fatal(err)
	}
	n := l.Cols * l.Rows
	chunks := (n + 99) / 100 // NDFD answers a hundred points an ask, and no more
	if len(get.asks) != 2*chunks {
		t.Fatalf("NDFD was asked %d times for %d points; want %d - the days and the hour, %d asks each", len(get.asks), n, 2*chunks, chunks)
	}
	seen := map[string]int{}
	var hourAsks []string
	for _, a := range get.asks {
		u, _ := url.Parse(a)
		pts := strings.Fields(u.Query().Get("listLatLon"))
		if len(pts) > 100 {
			t.Errorf("an ask carried %d points; NDFD answers only the first hundred", len(pts))
		}
		for _, p := range pts {
			seen[p]++
		}
		if u.Query().Get("temp") != "" {
			hourAsks = append(hourAsks, a)
		}
	}
	for _, p := range l.Points() {
		if k := ftoa(p.Lat) + "," + ftoa(p.Lon); seen[k] != 2 {
			t.Fatalf("point %s was asked %d times; want once a kind", k, seen[k])
		}
	}
	hour, _, ok := s.HourAt(captured)
	if !ok {
		t.Fatal("no current hour")
	}
	for i, p := range l.Points() {
		if !near(hour[i], fahrenheitToC(p.Lat)) {
			t.Fatalf("point %d (%v) reads %v; want its own answer %v - the asks' answers mixed up", i, p, hour[i], fahrenheitToC(p.Lat))
		}
	}
	rec := &pointsGet{}
	if _, err := NewNDFD(rec, "").Hour(context.Background(), l, captured); err != nil {
		t.Fatal(err)
	}
	if strings.Join(rec.asks, "\n") != strings.Join(hourAsks, "\n") {
		t.Errorf("the recorder's hour asked other addresses than Fetch's: the cache is not shared\n%v\n%v", rec.asks, hourAsks)
	}
}

// ANOTHER LATTICE'S VALUES ARE RESAMPLED ONTO THIS ONE'S POINTS (D-201): a
// day NDFD left empty is filled from Open-Meteo's coarser lattice (D-189),
// whose values are put on NDFD's points - copied whole, the lengths
// disagreed and the day drew blank.
func TestValuesAreResampledOntoAnotherLattice(t *testing.T) {
	box := geo.Box{W: -120, S: 30, E: -100, N: 45}
	coarse, dense := LatticeFor("b", box), NDFDLatticeFor("b", box)
	plane := func(p Point) float64 { return 2*p.Lon - 3*p.Lat }
	var vals []float64
	for _, p := range coarse.Points() {
		vals = append(vals, plane(p))
	}
	got := coarse.Resample(vals, dense)
	if len(got) != dense.Cols*dense.Rows {
		t.Fatalf("%d values for %d points", len(got), dense.Cols*dense.Rows)
	}
	for i, p := range dense.Points() {
		if math.Abs(got[i]-plane(p)) > 0.1 { // the points are rounded to hundredths
			t.Fatalf("point %d (%v) is %v; a plane resamples to itself, %v", i, p, got[i], plane(p))
		}
	}
	nan := math.NaN()
	flat := make([]float64, len(vals)) // ten everywhere but one point, which is empty: the others weigh up to one
	for i := range flat {
		flat[i] = 10
	}
	flat[coarse.Cols+1] = nan
	for i, v := range coarse.Resample(flat, dense) {
		if !math.IsNaN(v) && !near(v, 10) {
			t.Fatalf("point %d is %v beside an empty point; want 10, the others weighed up to one", i, v)
		}
	}
	empty := make([]float64, len(vals))
	for i := range empty {
		empty[i] = nan
	}
	for i, v := range coarse.Resample(empty, dense) {
		if !math.IsNaN(v) {
			t.Fatalf("point %d is %v from nothing; want missing", i, v)
		}
	}
	if same := coarse.Resample(vals, coarse); &same[0] == &vals[0] || len(same) != len(vals) || same[3] != vals[3] {
		t.Error("the same lattice is not a copy of its values")
	}
}

// A DIRECTION IS RESAMPLED BY THE NEAREST POINT, NEVER BLENDED (D-201): 350
// and 10 degrees blended would say 180, the wind turned round.
func TestADirectionIsResampledByItsNearestPoint(t *testing.T) {
	box := geo.Box{W: 0, S: 0, E: 1, N: 1}
	coarse := Lattice{Box: box, Cols: 2, Rows: 2}
	dense := Lattice{Box: box, Cols: 3, Rows: 3}
	got := coarse.ResampleNearest([]float64{350, 10, 350, 10}, dense)
	for i, v := range got {
		if v != 350 && v != 10 {
			t.Fatalf("point %d's direction is %v; want 350 or 10, the nearest point's", i, v)
		}
	}
	if got[0] != 350 || got[2] != 10 {
		t.Errorf("the corners are %v and %v; want their own points', 350 and 10", got[0], got[2])
	}
}
