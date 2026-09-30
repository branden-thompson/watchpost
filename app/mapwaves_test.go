package app

// mapwaves_test.go — 0.18.0 D-125, D-126: wave height, NDFD's where it
// reaches and Open-Meteo Marine's beyond, drawn over the sea alone.

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
)

// fakeWaves answers every lattice with one height every hour from three
// back to one on, and every day's highest; or nothing, failing.
type fakeWaves struct {
	metres, max float64
	failed      bool
	gapDays     bool // no day at all, as NDFD past its reach
	ahead       int  // hours answered after the current one; one when zero
	ashore      bool // the lattice's first point has none, as a point on land
}

func (f fakeWaves) Waves(_ context.Context, l temperature.Lattice, now time.Time) (temperature.Waves, error) {
	if f.failed {
		return temperature.Waves{}, errors.New("no answer")
	}
	n := l.Cols * l.Rows
	fill := func(v float64) []float64 {
		out := make([]float64, n)
		for i := range out {
			out[i] = v
		}
		return out
	}
	w := temperature.Waves{Lattice: l}
	for h := -3; h <= max(f.ahead, 1); h++ {
		w.Hours = append(w.Hours, now.Truncate(time.Hour).Add(time.Duration(h)*time.Hour))
		row := fill(f.metres)
		if f.ashore {
			row[0] = math.NaN()
		}
		w.Hourly = append(w.Hourly, row)
	}
	for k := range temperature.Days {
		w.Max[k] = fill(f.max)
		if f.gapDays {
			w.Max[k] = fill(math.NaN())
		}
	}
	return w, nil
}

// TestWavesFollowTheModes is D-126: Radar mode's every hour up to now,
// Forecast mode's Now and each day's highest, each during its step, in the
// listener's units, the preset's own - drawn over the sea alone - with its
// labelled contours; NDFD's where it has them, Open-Meteo's where not.
func TestWavesFollowTheModes(t *testing.T) {
	ndfd, om := fakeWaves{metres: 2 * 0.3048, max: 3 * 0.3048, gapDays: true}, fakeWaves{metres: 1, max: 2}
	radarMode := withWaves(context.Background(), tty.MapTemperature{}, ndfd, om, tempAsk(false), tempNow)
	if len(radarMode.Waves) != 4 {
		t.Fatalf("Radar mode's waves are %d hours; want the four up to now", len(radarMode.Waves))
	}
	for _, o := range radarMode.Waves {
		g := o.Grid
		if !strings.HasPrefix(o.ID, tty.WaveLayer+"/") || g.Type.Preset != "waves" || g.Type.Unit != "ft" || !g.Lines || math.Abs(g.Values[0]-2) > 1e-6 {
			t.Errorf("%s: %v %+v lines %v; want NDFD's 2 ft in the wave preset, lined", o.ID, g.Values[0], g.Type, g.Lines)
		}
	}
	ask := tempAsk(true)
	ask.Fahrenheit = false
	fc := withWaves(context.Background(), tty.MapTemperature{}, ndfd, om, ask, tempNow)
	steps := tty.ForecastSteps(ask.Anchor)
	if len(fc.Waves) != 1 || fc.Waves[0].During != steps[0].Span || len(fc.WaveDays) != temperature.Days {
		t.Fatalf("Forecast mode's waves: %d Now, %d days", len(fc.Waves), len(fc.WaveDays))
	}
	if d := fc.WaveDays[2]; d.During != steps[3].Span || d.Grid.Type.Unit != "m" || math.Abs(d.Grid.Values[0]-2) > 1e-6 {
		t.Errorf("day 3's waves are %v %s during %v; want Open-Meteo's 2 m where NDFD has no day", d.Grid.Values[0], d.Grid.Type.Unit, d.During)
	}
	if got := strings.Join(fc.Chips[tty.WaveLayer], "/"); got != "NDFD/O-METEO" {
		t.Errorf("the waves' badge names %q; want NDFD and Open-Meteo (D-133)", got)
	}
}

// TestWavesThatDidNotAnswerGoToTheDiagnostics is D-124 for the waves: no
// Setting offers another source, so a failure is the diagnostics' alone.
func TestWavesThatDidNotAnswerGoToTheDiagnostics(t *testing.T) {
	got := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{failed: true}, fakeWaves{failed: true}, tempAsk(false), tempNow)
	if len(got.Waves) != 0 || len(got.Chips[tty.WaveLayer]) != 0 || len(got.Problems) == 0 {
		t.Errorf("failed waves: %d grids, chips %v, problems %v; want the diagnostics told alone", len(got.Waves), got.Chips, got.Problems)
	}
}

// TestWavesAreOffByDefaultAndCosted is D-126 with D-23.
func TestWavesAreOffByDefaultAndCosted(t *testing.T) {
	for _, l := range mapLayers {
		if l.key == tty.WaveLayer {
			b, r := l.cost(mapInputs{region: geo.RegionContiguous, view: fireView})
			if l.on || r != 2*len(fieldBoxes(geo.RegionContiguous, fireView)) || b <= 0 {
				t.Errorf("waves are on %v, cost %d bytes in %d requests; want off, two requests a field box", l.on, b, r)
			}
			return
		}
	}
	t.Fatal("waves are not registered")
}

// TestTheFieldsRunOnIntoTheHoursAhead is UAT-2 U2-32: Radar mode's hourly
// fields stopped at the current hour, and the loop plays on into the hours
// ahead (D-113) - the waves came and went as it played. Each hourly field is
// drawn up to the loop's horizon.
func TestTheFieldsRunOnIntoTheHoursAhead(t *testing.T) {
	ask := tempAsk(false)
	ask.RadarAhead = 1
	waves := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{metres: 1, max: 2, gapDays: true}, fakeWaves{metres: 1, max: 2}, ask, tempNow)
	temp := buildTemperature(context.Background(), &noGap{&fakeTemp{name: "Open-Meteo", now: tempNow}}, nil, ask, tempNow, nil)
	next := ask.Anchor.Add(time.Hour)
	for name, grids := range map[string][]tuimaps.Overlay{"waves": waves.Waves, "temperature": temp.Overlays, "wind": temp.Wind, "feels": temp.Feels} {
		found := false
		for _, o := range grids {
			found = found || o.During.From.Equal(next)
		}
		if !found {
			t.Errorf("Radar mode's %s has no grid for the hour ahead the loop plays into", name)
		}
	}
}

// TestEveryHourAheadIsAcceptedByTheLibrary is U2-32 with U2-29's lesson: a
// grid of an hour twelve ahead, stamped with its own hour, would have had no
// currency left, and the library refuses it. Every Radar-mode wave grid to
// the longest horizon, a point ashore among them, is handed to a real map and
// accepted, and none is blank for the point ashore (U2-31).
func TestEveryHourAheadIsAcceptedByTheLibrary(t *testing.T) {
	m, err := tuimaps.New(tuimaps.WithSize(69, 12))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()
	ask := tempAsk(false)
	ask.RadarAhead = 12
	got := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{failed: true}, fakeWaves{metres: 1, max: 2, ahead: 12, ashore: true}, ask, tempNow)
	if len(got.Waves) != 16*len(fieldBoxes(ask.Region, ask.View)) {
		t.Fatalf("%d wave grids; want the three past, the current and twelve ahead, a box", len(got.Waves))
	}
	for _, o := range got.Waves {
		if _, err := m.Set(o); err != nil {
			t.Errorf("%s was refused: %v", o.ID, err)
		}
		for _, v := range o.Grid.Values {
			if math.IsNaN(v) {
				t.Fatalf("%s has a blank cell: the point ashore left it without the sea's value", o.ID)
			}
		}
	}
}

// A LAYER'S CHIPS NAME WHAT IT IS DRAWN FROM NOW (D-183): the waves NDFD's
// alone while Open-Meteo refuses, Open-Meteo's alone where NDFD does; UV's
// chip only while UV draws; temperature's, feels-like's and wind's only for
// what drew - an overlay drawing nothing names no source.
func TestAnOverlaysChipsNameWhatItIsDrawnFrom(t *testing.T) {
	ndfd, om := fakeWaves{metres: 2 * 0.3048, max: 3 * 0.3048}, fakeWaves{metres: 1, max: 2}
	refused := fakeWaves{failed: true}
	for _, tc := range []struct {
		name      string
		near, far waveSource
		want      string
	}{
		{"both answer", ndfd, om, "NDFD/O-METEO"},
		{"Open-Meteo refuses", ndfd, refused, "NDFD"},
		{"NDFD refuses", refused, om, "O-METEO"},
		{"neither", refused, refused, ""},
	} {
		got := withWaves(context.Background(), tty.MapTemperature{}, tc.near, tc.far, tempAsk(true), tempNow)
		if chips := strings.Join(got.Chips[tty.WaveLayer], "/"); chips != tc.want {
			t.Errorf("%s: the waves' chips are %q; want %q", tc.name, chips, tc.want)
		}
	}
	nothing := buildTemperature(context.Background(), &fakeTemp{name: "Open-Meteo", now: tempNow, failed: true}, nil, tempAsk(false), tempNow, nil)
	for _, key := range []string{tty.TemperatureLayer, tty.FeelsLayer, tty.WindLayer} {
		if chips := nothing.Chips[key]; len(chips) != 0 {
			t.Errorf("%s drew nothing and names %v", key, chips)
		}
	}
}
