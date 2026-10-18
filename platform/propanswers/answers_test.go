package propanswers

// answers_test.go — 0.19.0 D-212: the Propagation mode's answers, a package
// of their own; the chart's targets an input (D-204), bands read by name.

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// snapAt is a snapshot whose hour holds a flat field of each layer over the
// world: MUF(3000) mhz and foF2 a third of it.
func snapAt(at time.Time, mhz float32) ionomaps.Snapshot {
	field := func(v float32) ionomaps.Field {
		f := ionomaps.Field{West: -180, South: -90, East: 180, North: 90, Cols: 180, Rows: 90, Values: make([]float32, 180*90)}
		for i := range f.Values {
			f.Values[i] = v
		}
		return f
	}
	return ionomaps.Snapshot{Computed: at, Now: ionomaps.Hour{At: at, MUF3000: field(mhz), FoF2: field(mhz / 3)},
		Background: ionomaps.Backgrounds{FoF2: ionomaps.GloTEC, M3000: ionomaps.Climatology}, Inputs: ionomaps.Inputs{GloTECValid: at.Add(-30 * time.Minute), F107Mean: 150}}
}

// TestTheTargetsAreAnInput is D-212 with D-204: a chart answers the targets
// it is asked for, in their order - the continents, and a place found after
// them, answered from the origin like the rest; a target within the
// near-vertical radius is the area's answer.
func TestTheTargetsAreAnInput(t *testing.T) {
	snap := snapAt(time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC), 18)
	origin := ionomaps.LatLon{Lat: 33.2, Lon: -117.4}
	found := Target{Name: "Reykjavik, IS", Short: "FOUND", At: ionomaps.LatLon{Lat: 64.1, Lon: -21.9}}
	chart, err := WorkChart(snap, Ask{Origin: origin, RadiusKm: 400, Targets: append(append([]Target(nil), Continents...), found)})
	if err != nil || len(chart.Targets) != len(Continents)+1 {
		t.Fatalf("asked for the continents and a found place: %d answers (%v)", len(chart.Targets), err)
	}
	if bands := chart.Targets[len(Continents)].Bands; len(bands) == 0 || bands[0].Band != ionomaps.Band160m {
		t.Errorf("the place found is not answered: %+v", bands)
	}
	home := Target{Name: "Here", At: origin}
	chart, err = WorkChart(snap, Ask{Origin: origin, RadiusKm: 400, Targets: []Target{home}})
	if err != nil || len(chart.Targets) != 1 || !reflect.DeepEqual(chart.Targets[0], chart.Local) {
		t.Errorf("a target within the near-vertical radius is not the area's answer (%v)", err)
	}
	if !home.Near(origin, 400) || found.Near(origin, 400) {
		t.Error("Near does not say which targets are within the radius")
	}
}

// TestBandsAreReadByName is the BUILD red team's L-F8: an answer's bands
// are named by the band each carries, never by its place in the list - a
// list that starts at 80 metres, or out of order, is said as it is.
func TestBandsAreReadByName(t *testing.T) {
	bands := []ionomaps.BandStatus{
		{Band: ionomaps.Band20m, Status: ionomaps.Open},
		{Band: ionomaps.Band80m, Status: ionomaps.AboveUpper},
		{Band: ionomaps.Band40m, Status: ionomaps.Disturbed},
	}
	if got := BandsSaid(bands); got != "open on 20m; disturbed on 40m; above the upper limit on 80m" {
		t.Errorf("bands out of order are said %q", got)
	}
	if got := Usable(bands); strings.Join(got, " ") != "20m 40m" {
		t.Errorf("the usable bands are %v", got)
	}
	if got := HighestOpen(bands); got != "20m" {
		t.Errorf("the highest open band is %q", got)
	}
	if mhz, band, ok := Best(Limits{Upper: 17.6, LUF: 6.2}); !ok || band != "20m" || mhz < 14.9 || mhz > 15.0 {
		t.Errorf("the best frequency under a 17.6 MHz MUF is %.2f in %q (%v)", mhz, band, ok)
	}
}

// TestAnAskOffTheGlobeIsRefused is the package's boundary (D-212, the BUILD
// red team's L-F12): a radius negative or not a number, or a place - the
// origin, a target, the centre - off the globe or not a number, is refused as
// ErrAsk, never worked into an answer; a frequency that is none is said so,
// and a figure that is not a number is never said "NaN".
func TestAnAskOffTheGlobeIsRefused(t *testing.T) {
	snap := snapAt(time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC), 18)
	nan := math.NaN()
	here := ionomaps.LatLon{Lat: 33.2, Lon: -117.4}
	for name, ask := range map[string]Ask{
		"a negative radius":         {Origin: here, RadiusKm: -1},
		"a radius not a number":     {Origin: here, RadiusKm: nan},
		"the origin's latitude 91":  {Origin: ionomaps.LatLon{Lat: 91, Lon: 0}, RadiusKm: 400},
		"the origin's latitude NaN": {Origin: ionomaps.LatLon{Lat: nan, Lon: 0}, RadiusKm: 400},
		"the origin's longitude":    {Origin: ionomaps.LatLon{Lat: 0, Lon: 181}, RadiusKm: 400},
		"a target's latitude":       {Origin: here, RadiusKm: 400, Targets: []Target{{Name: "x", At: ionomaps.LatLon{Lat: nan}}}},
		"a target's longitude":      {Origin: here, RadiusKm: 400, Targets: []Target{{Name: "x", At: ionomaps.LatLon{Lon: -200}}}},
	} {
		if _, err := WorkChart(snap, ask); !errors.Is(err, ErrAsk) {
			t.Errorf("%s: %v; refused as ErrAsk", name, err)
		}
	}
	for name, c := range map[string][3]ionomaps.LatLon{
		"the origin's latitude":  {{Lat: 95}, here},
		"the origin's longitude": {{Lon: nan}, here},
		"the centre's latitude":  {here, {Lat: -95}},
		"the centre's longitude": {here, {Lon: 190}},
	} {
		if _, err := WorkCentre(snap, c[0], c[1], 400); !errors.Is(err, ErrAsk) {
			t.Errorf("the centre, %s off the globe: %v", name, err)
		}
	}
	if _, err := WorkCentre(snap, here, here, -5); !errors.Is(err, ErrAsk) {
		t.Errorf("the centre with a negative radius: %v", err)
	}
	if (Target{At: here}).Near(here, nan) || (Target{At: ionomaps.LatLon{Lat: nan}}).Near(here, 400) {
		t.Error("a radius or a target not a number is near")
	}
	if MHz(nan) != "?" || FreqVerdict(0, "held", Limits{Upper: 14}) != "held: not a frequency" {
		t.Errorf("a figure not a number is said %q; no frequency %q", MHz(nan), FreqVerdict(0, "held", Limits{Upper: 14}))
	}
	if _, _, ok := Best(Limits{Upper: 14, LUF: nan}); ok {
		t.Error("a LUF not a number decides a best band")
	}
}
