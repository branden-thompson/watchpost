package temperature

// waves_test.go — 0.18.0 D-125: wave height, NDFD's where it reaches and
// Open-Meteo Marine's beyond, over the recorded answers.

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// wavesGet answers with the wave fixtures, recording the addresses.
type wavesGet struct {
	t    *testing.T
	asks []string
}

func (w *wavesGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	w.asks = append(w.asks, rawURL)
	if strings.Contains(rawURL, "/v1/marine") {
		return fixture(w.t, "openmeteo-waves.json"), nil
	}
	return fixture(w.t, "ndfd-waves.xml"), nil
}

// wavesCaptured is when the wave fixtures were recorded: 22:35 in San Diego.
var wavesCaptured = time.Date(2026, 9, 28, 5, 35, 0, 0, time.UTC)

const feetToM = 0.3048

// TestNDFDReadsTheWaves is D-125: NDFD's significant wave height, in feet,
// hourly from the next hour; each day's highest worked out on the local
// date; nothing ashore.
func TestNDFDReadsTheWaves(t *testing.T) {
	g := &wavesGet{t: t}
	w, err := NewNDFD(g, "").Waves(context.Background(), fixtureLattice, wavesCaptured)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.asks) != 1 || !strings.Contains(g.asks[0], "waveh=waveh") {
		t.Errorf("asked %v; want NDFD's wave height", g.asks)
	}
	if v, ok := w.At(time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC)); !ok || !near(v[0], 7*feetToM) {
		t.Errorf("121W at 06Z is %v m (%v); want 7 ft", v, ok)
	}
	if !near(w.Max[1][0], 10*feetToM) || !math.IsNaN(w.Max[1][escondido]) {
		t.Errorf("tomorrow's highest is %v m at 121W and %v ashore; want 10 ft, and nothing", w.Max[1][0], w.Max[1][escondido])
	}
}

// TestOpenMeteoReadsTheWaves is D-125: Open-Meteo Marine's hourly wave
// height and each day's highest, in metres, on its own host; nothing ashore.
func TestOpenMeteoReadsTheWaves(t *testing.T) {
	g := &wavesGet{t: t}
	w, err := NewOpenMeteo(g, "").Waves(context.Background(), fixtureLattice, wavesCaptured)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.asks) != 1 || !strings.HasPrefix(g.asks[0], "https://marine-api.open-meteo.com/v1/marine?") || !strings.Contains(g.asks[0], "wave_height_max") {
		t.Errorf("asked %v; want the marine API's wave height", g.asks)
	}
	now, ok := w.At(wavesCaptured) // 05:35Z: 21:35 in the sea's own zone, UTC-8 - not California's daylight time
	if !ok || !near(now[0], 2.2) || !math.IsNaN(now[escondido]) {
		t.Errorf("now the waves are %v (%v); want 2.2 m at 121W, 21:00 in the sea's zone, nothing ashore", now, ok)
	}
	if !near(w.Max[0][0], 2.38) {
		t.Errorf("today's highest at 121W is %v; want 2.38 m", w.Max[0][0])
	}
}

// TestNDFDsWavesAreFilledFromOpenMeteo is D-125's merge: each point and hour
// NDFD's where it has one, Open-Meteo's where it has none.
func TestNDFDsWavesAreFilledFromOpenMeteo(t *testing.T) {
	g := &wavesGet{t: t}
	ndfd, _ := NewNDFD(g, "").Waves(context.Background(), fixtureLattice, wavesCaptured)
	om, _ := NewOpenMeteo(g, "").Waves(context.Background(), fixtureLattice, wavesCaptured)
	w := ndfd.FilledFrom(om)
	if now, ok := w.At(wavesCaptured); !ok || !near(now[0], 2.2) {
		t.Errorf("now, before NDFD's first hour, is %v (%v); want Open-Meteo's 2.2 m", now, ok)
	}
	if !near(w.Max[1][0], 10*feetToM) {
		t.Errorf("tomorrow's highest is %v; want NDFD's 10 ft", w.Max[1][0])
	}
	if both, ok := w.At(time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC)); !ok || !near(both[0], 7*feetToM) { // an hour both answer: Open-Meteo's is 2.22 m
		t.Errorf("at 06Z, an hour both sources answer, 121W is %v; want NDFD's 7 ft", both)
	}
}

// TestWavesReachTheCoast is UAT-2 U2-31: a sea cell whose nearest lattice
// point was ashore was left blank - the rule that keeps a source to its reach
// (D-101) - so the bands stopped short of the coast, in blocks. Waves are
// carried out to every cell from the sea's own points; the map draws them
// over the sea alone.
func TestWavesReachTheCoast(t *testing.T) {
	l := Lattice{Cols: 3, Rows: 2}
	l.Box.W, l.Box.S, l.Box.E, l.Box.N = -121, 31, -117, 33
	nan := math.NaN()
	f := l.InterpolateOut([]float64{2, 1.5, nan, 2.2, 2, 1.8}) // Escondido, ashore, has none
	for i, v := range f.Values {
		if math.IsNaN(v) {
			t.Fatalf("cell %d is blank; a cell by the coast takes the sea's values", i)
		}
	}
	if all := l.InterpolateOut([]float64{nan, nan, nan, nan, nan, nan}); !math.IsNaN(all.Values[0]) {
		t.Error("with no value anywhere a cell was given one")
	}
}

// TestOpenMeteoIsAskedForTheLoopsHoursAhead is U2-32: the fields are drawn to
// the radar loop's longest horizon, twelve hours past the current one, so
// both of Open-Meteo's requests ask for that many.
func TestOpenMeteoIsAskedForTheLoopsHoursAhead(t *testing.T) {
	g := &wavesGet{t: t}
	_, _ = NewOpenMeteo(g, "").Waves(context.Background(), fixtureLattice, wavesCaptured)
	f := &feelsGet{t: t}
	_, _ = NewOpenMeteo(f, "").Fetch(context.Background(), fixtureLattice, feelsCaptured)
	for _, a := range append(g.asks, f.asks...) {
		if !strings.Contains(a, "forecast_hours=13") {
			t.Errorf("%s does not ask for the loop's thirteen hours", a)
		}
	}
}
