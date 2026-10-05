package temperature

// guards_test.go — the guards over what a source answers and what it is
// asked with (D-248).

import (
	"context"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"
)

// bodyGet answers every request with one body, and counts the asks.
type bodyGet struct {
	body []byte
	asks int
}

func (g *bodyGet) GetText(context.Context, string, ...httpx.Option) ([]byte, error) {
	g.asks++
	return g.body, nil
}

// TestALatticeOfNoPointsIsNeverAskedFor: a lattice under two points a side
// has no spacing between its points - Points would divide by zero and send
// NaN coordinates, which Open-Meteo bills - so it has no points and no source
// asks for it.
func TestALatticeOfNoPointsIsNeverAskedFor(t *testing.T) {
	ctx := context.Background()
	for _, l := range []Lattice{{Name: "thin", Box: fixtureLattice.Box, Cols: 1, Rows: 5}, {Name: "flat", Box: fixtureLattice.Box, Cols: 5, Rows: 1}, {}} {
		if pts := l.Points(); len(pts) != 0 {
			t.Errorf("a %dx%d lattice has points %v", l.Cols, l.Rows, pts)
		}
		get := &bodyGet{body: fixture(t, "openmeteo.json")}
		om, nd := NewOpenMeteo(get, ""), NewNDFD(get, "")
		if _, err := om.Fetch(ctx, l, captured); err == nil {
			t.Errorf("Open-Meteo answered a %dx%d lattice", l.Cols, l.Rows)
		}
		if _, err := om.Rain(ctx, l, captured, 0); err == nil {
			t.Errorf("Open-Meteo's rain answered a %dx%d lattice", l.Cols, l.Rows)
		}
		if _, err := nd.Fetch(ctx, l, captured); err == nil {
			t.Errorf("NDFD answered a %dx%d lattice", l.Cols, l.Rows)
		}
		if _, err := nd.Hour(ctx, l, captured); err == nil {
			t.Errorf("NDFD's hour answered a %dx%d lattice", l.Cols, l.Rows)
		}
		if _, err := nd.Waves(ctx, l, captured); err == nil {
			t.Errorf("NDFD's waves answered a %dx%d lattice", l.Cols, l.Rows)
		}
		if get.asks != 0 {
			t.Errorf("a %dx%d lattice was asked for %d times", l.Cols, l.Rows, get.asks)
		}
	}
}

// far is an hour no answer for captured holds: past every day a series keeps.
const farOM, farDWML = "2031-01-01T00:00", "2031-01-01T00:00:00-07:00"

// noHourPast fails when a series holds an hour beyond the days it keeps.
func noHourPast(t *testing.T, what string, hours []time.Time) {
	t.Helper()
	for _, h := range hours {
		if h.After(captured.Add((Days + 1) * 24 * time.Hour)) {
			t.Errorf("%s holds the hour %v, past every day it keeps", what, h)
		}
	}
}

// TestAnHourFarFromNowIsNotKept: every distinct hour an answer names adds a
// row a measure to the series, so an answer of hours far from now - broken or
// hostile, within the body cap - would grow the series without bound. An hour
// past the days a series keeps, or a day before now, is left out.
func TestAnHourFarFromNowIsNotKept(t *testing.T) {
	ctx := context.Background()
	om := regexp.MustCompile(`"time":\["2026-09-26T14:00"`)
	body := fixture(t, "openmeteo.json")
	if !om.Match(body) {
		t.Fatal("the fixture's first hour moved: this measures nothing")
	}
	s, err := NewOpenMeteo(&bodyGet{body: om.ReplaceAll(body, []byte(`"time":["`+farOM+`"`))}, "").Fetch(ctx, fixtureLattice, captured)
	if err != nil {
		t.Fatal(err)
	}
	noHourPast(t, "Open-Meteo's series", s.Hours)

	nd := fixture(t, "ndfd-hour.xml")
	first := "<start-valid-time>2026-09-26T18:00:00-07:00</start-valid-time>"
	if !strings.Contains(string(nd), first) {
		t.Fatal("the NDFD fixture's first hour moved: this measures nothing")
	}
	s = newSeries(fixtureLattice)
	if err := parseDWML([]byte(strings.Replace(string(nd), first, "<start-valid-time>"+farDWML+"</start-valid-time>", 1)), captured, &s); err != nil {
		t.Fatal(err)
	}
	noHourPast(t, "NDFD's series", s.Hours)
}

// TestAValueThatIsNoNumberStaysMissing: NDFD's values are read with
// ParseFloat, which takes "Inf" and "NaN"; a value that is no finite number
// is missing, never an infinite temperature on the map.
func TestAValueThatIsNoNumberStaysMissing(t *testing.T) {
	num := regexp.MustCompile(`<value>-?\d+(\.\d+)?</value>`)
	for _, name := range []string{"ndfd-hour.xml", "ndfd-days.xml"} {
		body := fixture(t, name)
		if !num.Match(body) {
			t.Fatalf("%s holds no number: this measures nothing", name)
		}
		s := newSeries(fixtureLattice)
		if err := parseDWML(num.ReplaceAll(body, []byte("<value>Inf</value>")), captured, &s); err != nil {
			t.Fatal(err)
		}
		rows := append(append([][]float64{}, s.Hourly...), s.High[:]...)
		rows = append(rows, s.Low[:]...)
		for _, row := range rows {
			for _, v := range row {
				if math.IsInf(v, 0) {
					t.Fatalf("%s: a value of Inf was kept", name)
				}
			}
		}
	}
}
