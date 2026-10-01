package temperature

// waves_test.go — 0.18.0 D-125: wave height, NDFD's where it reaches and
// Open-Meteo Marine's beyond, over the recorded answers.

import (
	"context"
	"math"
	"net/url"
	"strconv"
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

// marinePoints answers Open-Meteo Marine for every point asked: each its own
// height, its index among the asked plus one, every hour from start.
type marinePoints struct{ asked []string }

func (m *marinePoints) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	m.asked = append(m.asked, rawURL)
	q, _ := url.ParseQuery(rawURL[strings.Index(rawURL, "?")+1:])
	var pts []string
	for i := range strings.Split(q.Get("latitude"), ",") {
		v := strconv.Itoa(i + 1)
		pts = append(pts, `{"utc_offset_seconds":0,"hourly":{"time":["2026-09-28T05:00"],"wave_height":[`+v+`]},"daily":{"time":["2026-09-28"],"wave_height_max":[`+v+`]}}`)
	}
	return []byte("[" + strings.Join(pts, ",") + "]"), nil
}

// OPEN-METEO MARINE IS ASKED FOR THE POINTS NAMED ALONE (W19.5, D-194):
// every one it bills, so a box's points NDFD reaches - and the land - are
// not asked; the answers land at the points asked for, every other point
// missing.
func TestOpenMeteoMarineIsAskedForThePointsNamedAlone(t *testing.T) {
	get := &marinePoints{}
	w, err := NewOpenMeteo(get, "").WavesAt(context.Background(), fixtureLattice, []int{1, 4}, wavesCaptured)
	if err != nil {
		t.Fatal(err)
	}
	q, _ := url.ParseQuery(get.asked[0][strings.Index(get.asked[0], "?")+1:])
	pts := fixtureLattice.Points()
	if lats := strings.Split(q.Get("latitude"), ","); len(get.asked) != 1 || len(lats) != 2 || lats[0] != ftoa(pts[1].Lat) || lats[1] != ftoa(pts[4].Lat) {
		t.Fatalf("asked %v; want points 1 and 4 alone", get.asked)
	}
	row := w.Hourly[0]
	for i, v := range row {
		switch i {
		case 1:
			if v != 1 {
				t.Errorf("point 1 is %v; want the first answer", v)
			}
		case 4:
			if v != 2 {
				t.Errorf("point 4 is %v; want the second answer", v)
			}
		default:
			if !math.IsNaN(v) {
				t.Errorf("point %d, not asked, is %v; want missing", i, v)
			}
		}
	}
	if _, err := NewOpenMeteo(get, "").WavesAt(context.Background(), fixtureLattice, []int{}, wavesCaptured); err == nil || len(get.asked) != 1 {
		t.Error("no point named, Open-Meteo was asked, or nothing was said")
	}
}

// A RECORDED HOUR FILLS ONLY AN HOUR THE SOURCE LEFT EMPTY (D-194): SetHour
// puts the history's values where the series has none, never over the
// source's own, and never values that do not cover the lattice.
func TestARecordedHourFillsOnlyAnEmptyHour(t *testing.T) {
	w := newWaves(fixtureLattice)
	h := wavesCaptured.Truncate(time.Hour)
	vals := []float64{1, 1, 1, 1, 1, 1}
	if !w.SetHour(h, vals) {
		t.Fatal("an empty hour was not filled")
	}
	if w.SetHour(h, []float64{2, 2, 2, 2, 2, 2}) {
		t.Error("an hour with values was written over")
	}
	if got, _ := w.At(h); got[0] != 1 {
		t.Errorf("the hour is %v; want the first values kept", got)
	}
	if w.SetHour(h.Add(time.Hour), []float64{1}) {
		t.Error("values not covering the lattice were put in")
	}
}
