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
	for h := -3; h <= 1; h++ {
		w.Hours = append(w.Hours, now.Truncate(time.Hour).Add(time.Duration(h)*time.Hour))
		w.Hourly = append(w.Hourly, fill(f.metres))
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
	if !strings.Contains(strings.Join(fc.WaveNotes, " "), "Open-Meteo") {
		t.Errorf("the waves' notes are %v; want the credit", fc.WaveNotes)
	}
}

// TestWavesThatDidNotAnswerGoToTheDiagnostics is D-124 for the waves: no
// Setting offers another source, so a failure is the diagnostics' alone.
func TestWavesThatDidNotAnswerGoToTheDiagnostics(t *testing.T) {
	got := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{failed: true}, fakeWaves{failed: true}, tempAsk(false), tempNow)
	if len(got.Waves) != 0 || len(got.WaveNotes) != 0 || len(got.Problems) == 0 {
		t.Errorf("failed waves: %d grids, notes %v, problems %v; want the diagnostics told alone", len(got.Waves), got.WaveNotes, got.Problems)
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
