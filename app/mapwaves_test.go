package app

// mapwaves_test.go — 0.18.0 D-125, D-126: wave height, NDFD's where it
// reaches and Open-Meteo Marine's beyond, drawn over the sea alone.

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
)

// fakeWaves answers every lattice with one height every hour from three
// back to one on, and every day's highest; or nothing, failing.
type fakeWaves struct {
	metres, max float64
	failed      bool
	gapDays     bool // no day at all, as NDFD past its reach
	ahead       int  // hours answered after the current one; one when zero
	ashore      bool // the lattice's first point has none, as a point on land
	fromNext    bool // hours from the next one, as NDFD's (D-194)
	beyond      int  // the points from this index on have nothing: past NDFD's reach
	land        int  // Open-Meteo answers nothing for the points from this index on
	asked       *[][]int
}

// WavesAt is Waves for the points named, the rest missing: Open-Meteo
// Marine's (D-194); the points asked are kept.
func (f fakeWaves) WavesAt(ctx context.Context, l temperature.Lattice, only []int, now time.Time) (temperature.Waves, error) {
	if f.asked != nil {
		*f.asked = append(*f.asked, append([]int(nil), only...))
	}
	w, err := f.Waves(ctx, l, now)
	if err != nil {
		return w, err
	}
	named := map[int]bool{}
	for _, i := range only {
		named[i] = f.land == 0 || i < f.land
	}
	blank := func(row []float64) {
		for i := range row {
			if !named[i] {
				row[i] = math.NaN()
			}
		}
	}
	for _, row := range w.Hourly {
		blank(row)
	}
	for k := range w.Max {
		blank(w.Max[k])
	}
	return w, nil
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
	first := -3
	if f.fromNext {
		first = 1
	}
	for h := first; h <= max(f.ahead, 1); h++ {
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
	if f.beyond > 0 {
		for _, row := range append(append([][]float64(nil), w.Hourly...), w.Max[:]...) {
			for i := f.beyond; i < len(row); i++ {
				row[i] = math.NaN()
			}
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
	radarMode := withWaves(context.Background(), tty.MapTemperature{}, ndfd, om, tempAsk(false), tempNow, waveKeep{})
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
	fc := withWaves(context.Background(), tty.MapTemperature{}, ndfd, om, ask, tempNow, waveKeep{})
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
	got := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{failed: true}, fakeWaves{failed: true}, tempAsk(false), tempNow, waveKeep{})
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

// TestTheFieldsRunOnIntoTheHoursAhead is UAT-2 U2-32: the loop plays on into
// the hours ahead (D-113), so an hourly field that stops at the current hour
// comes and goes as it plays. Each hourly field is drawn up to the loop's
// horizon.
func TestTheFieldsRunOnIntoTheHoursAhead(t *testing.T) {
	ask := tempAsk(false)
	ask.RadarAhead = 1
	waves := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{metres: 1, max: 2, gapDays: true}, fakeWaves{metres: 1, max: 2}, ask, tempNow, waveKeep{})
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
	got := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{failed: true}, fakeWaves{metres: 1, max: 2, ahead: 12, ashore: true}, ask, tempNow, waveKeep{})
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
		near, far fakeWaves
		want      string
	}{
		{"both answer, Open-Meteo past NDFD's reach", fakeWaves{metres: 2 * 0.3048, max: 3 * 0.3048, beyond: 1}, om, "NDFD/O-METEO"},
		{"NDFD reaches everywhere: Open-Meteo not asked (D-194)", ndfd, om, "NDFD"},
		{"Open-Meteo refuses", ndfd, refused, "NDFD"},
		{"NDFD refuses", refused, om, "O-METEO"},
		{"neither", refused, refused, ""},
	} {
		got := withWaves(context.Background(), tty.MapTemperature{}, tc.near, tc.far, tempAsk(true), tempNow, waveKeep{})
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

// THE WAVES ARE NDFD'S FIRST, OPEN-METEO ONLY PAST ITS REACH (W19.5, D-194):
// NDFD's waves start at the next hour, so the current hour and the loop's
// past ones are the history's - NDFD's next hour as each hour recorded it -
// and Open-Meteo Marine is asked only for the points NDFD does not reach; a
// point it answers nothing for (land) is never asked again. The badge names
// NDFD, Open-Meteo and RECORDED. Nothing recorded, NDFD's next hour is
// stretched under the loop (the cold start).
func TestTheWavesAreNDFDsFirstAndOpenMeteoOnlyPastItsReach(t *testing.T) {
	ask := tempAsk(false)
	box := fieldBoxes(ask.Region, ask.View)[0]
	lat := temperature.LatticeFor(box.Name, box.Box)
	n := lat.Cols * lat.Rows
	anchor := ask.Anchor
	store := history.Open(t.TempDir(), func() time.Time { return tempNow }, ndfdWaves)
	all := make([]float64, n)
	for i := range all {
		all[i] = 1.5
	}
	all[n-2], all[n-1] = math.NaN(), math.NaN() // the history is NDFD's: nothing past its reach
	for back := pastHours; back >= 0; back-- {
		h := anchor.Add(-time.Duration(back) * time.Hour)
		if !store.Put(ndfdWaves.Name, history.Record{Key: history.Key{Source: "ndfd", Place: box.Name}, At: h, IssuedAt: h.Add(-time.Hour), Shape: shapeOf(lat), Values: map[string][]float64{"waves": all}}) {
			t.Fatal("could not seed the history")
		}
	}
	var asks [][]int
	ndfd := fakeWaves{metres: 1, max: 2, fromNext: true, beyond: n - 2}
	om := fakeWaves{metres: 3, max: 4, land: n - 1, asked: &asks}
	keep := waveKeep{store: store, land: &landPoints{}}
	got := withWaves(context.Background(), tty.MapTemperature{}, ndfd, om, ask, tempNow, keep)
	if len(asks) != 1 || len(asks[0]) != 2 || asks[0][0] != n-2 || asks[0][1] != n-1 {
		t.Fatalf("Open-Meteo was asked for %v; want the two points past NDFD's reach alone", asks)
	}
	if chips := strings.Join(got.Chips[tty.WaveLayer], "/"); chips != "NDFD/O-METEO/RECORDED" {
		t.Errorf("the badge names %q; want NDFD, Open-Meteo and RECORDED", chips)
	}
	from := map[time.Time]bool{}
	for _, o := range got.Waves {
		from[o.During.From] = true
	}
	for back := pastHours; back >= 0; back-- {
		if !from[anchor.Add(-time.Duration(back)*time.Hour)] {
			t.Errorf("the hour %d before was not drawn from the history: %v", back, from)
		}
	}
	withWaves(context.Background(), tty.MapTemperature{}, ndfd, om, ask, tempNow, keep)
	if len(asks) != 2 || len(asks[1]) != 1 || asks[1][0] != n-2 {
		t.Errorf("the second refresh asked %v; want the sea point alone - the land learned", asks[1:])
	}
	cold := withWaves(context.Background(), tty.MapTemperature{}, ndfd, om, ask, tempNow, waveKeep{land: &landPoints{}})
	covered(t, "the waves, nothing recorded", cold.Waves, anchor, tempNow) // every observed frame drawn (U2-55)
	if slices.Contains(cold.Chips[tty.WaveLayer], "RECORDED") {
		t.Error("nothing recorded, yet RECORDED was said")
	}
}

// A WAVE DAY PAST NDFD'S REACH IS OPEN-METEO'S (D-195): in Forecast mode a
// day NDFD gives no waves anywhere is filled from Open-Meteo at the sea's
// points - every point but the land it has learned - and where Open-Meteo
// gives nothing either, a note says so.
func TestAWaveDayPastNDFDsReachIsOpenMeteos(t *testing.T) {
	ask := tempAsk(true)
	var asks [][]int
	keep := waveKeep{land: &landPoints{}}
	keep.land.learn(fieldBoxes(ask.Region, ask.View)[0].Name, []int{0})
	got := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{metres: 1, max: 2, gapDays: true}, fakeWaves{metres: 3, max: 4, asked: &asks}, ask, tempNow, keep)
	if len(asks) == 0 || slices.Contains(asks[0], 0) || len(asks[0]) < 2 {
		t.Fatalf("Open-Meteo was asked for %v; want every sea point, the land's left out", asks)
	}
	if len(got.WaveDays) != temperature.Days {
		t.Errorf("%d wave days drawn; want each day, Open-Meteo's where NDFD has none", len(got.WaveDays))
	}
	none := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{metres: 1, max: 2, gapDays: true}, fakeWaves{failed: true}, ask, tempNow, waveKeep{land: &landPoints{}})
	if !slices.ContainsFunc(none.LayerNotes[tty.WaveLayer], func(n string) bool { return strings.Contains(n, "past NDFD's reach") }) {
		t.Errorf("Open-Meteo refusing too, no wave note says why the days are empty: %v", none.LayerNotes)
	}
}
