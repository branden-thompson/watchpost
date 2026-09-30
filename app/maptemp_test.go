package app

// maptemp_test.go — 0.18.0 W10 in the app (D-93 to D-98): the mode's source,
// the grids built for each hour or step with its span, the listener's unit,
// what a source lacks said, the credit, and every feed overlay timed.

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"context"
	"errors"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// fakeTemp answers every lattice with fixed values: 10 C every hour from
// three hours back to one on, each day's high 20 C and low 5 C - but no high
// today, as NDFD after its daytime; feels-like 12 C each hour, each day 22 C
// to 3 C, or no feels-like hour at all, as NDFD's hours begin at the next.
type fakeTemp struct {
	name        string
	now         time.Time
	failed      bool
	noFeelsHour bool
	noWindDay   bool // day 1 without its wind or gust, for Open-Meteo to fill (D-100, D-136)
	asked       int
}

func (f *fakeTemp) Name() string       { return f.name }
func (f *fakeTemp) Covers(string) bool { return f.name != "NDFD" }
func (f *fakeTemp) Fetch(_ context.Context, l temperature.Lattice, _ time.Time) (temperature.Series, error) {
	f.asked++
	if f.failed {
		return temperature.Series{}, errors.New("no answer")
	}
	n := l.Cols * l.Rows
	fill := func(v float64) []float64 {
		out := make([]float64, n)
		for i := range out {
			out[i] = v
		}
		return out
	}
	s := temperature.Series{Lattice: l}
	for h := -3; h <= 1; h++ {
		s.Hours = append(s.Hours, f.now.Truncate(time.Hour).Add(time.Duration(h)*time.Hour))
		s.Hourly = append(s.Hourly, fill(10))
		s.WindSpeed, s.WindFrom = append(s.WindSpeed, fill(16.09344)), append(s.WindFrom, fill(270)) // 10 mph from the west
		s.WindGust = append(s.WindGust, fill(48.28032))                                              // gusting 30: 20 over, said (D-136)
		feels := fill(12)
		if f.noFeelsHour {
			feels = fill(math.NaN())
		}
		s.Feels = append(s.Feels, feels)
	}
	for k := range temperature.Days {
		s.High[k], s.Low[k] = fill(20), fill(5)
		s.FeelsHigh[k], s.FeelsLow[k] = fill(22), fill(3)
		s.PeakSpeed[k], s.PeakFrom[k] = fill(32.18688), fill(225) // 20 mph from the south-west
		s.PeakGust[k] = fill(40.2336)                             // gusting 25: 5 over, not said (D-136)
	}
	s.High[0] = fill(math.NaN())
	s.PeakGust[1] = fill(64.37376) // day 1 gusting 40: 20 over its peak, said (D-136)
	if f.noWindDay {
		s.PeakSpeed[1], s.PeakFrom[1], s.PeakGust[1] = fill(math.NaN()), fill(math.NaN()), fill(math.NaN())
	}
	return s, nil
}

var tempNow = time.Date(2026, 8, 24, 20, 30, 0, 0, time.UTC)

func tempAsk(forecast bool) tty.MapAsk {
	return tty.MapAsk{Region: geo.RegionContiguous, View: geo.Box{W: -120, S: 32, E: -115, N: 36}, Forecast: forecast,
		Anchor: tempNow.Truncate(time.Hour), Fahrenheit: true}
}

func TestRadarModeDrawsEachHourDuringItself(t *testing.T) {
	src := &fakeTemp{name: "Open-Meteo", now: tempNow}
	got := buildTemperature(context.Background(), src, nil, tempAsk(false), tempNow, nil)
	if len(got.High) != 0 || len(got.Low) != 0 {
		t.Error("Radar mode was given days")
	}
	hours := map[time.Time]bool{}
	for _, o := range got.Overlays {
		sp := o.During
		if sp.From.After(tempNow) {
			t.Errorf("%s is a forecast hour; no radar frame is in it", o.ID)
		}
		if sp.Until.Sub(sp.From) != time.Hour-time.Nanosecond || !strings.HasPrefix(o.ID, tty.TemperatureLayer+"/") {
			t.Errorf("%s spans %v; want its own hour, under the layer's key", o.ID, sp)
		}
		hours[sp.From] = true
		if v := o.Grid.Values[0]; v != 50 {
			t.Errorf("%s's value is %v; want 10 C as 50 F, the listener's unit", o.ID, v)
		}
		if o.Grid.Type.Unit != "F" {
			t.Errorf("%s says its unit is %q", o.ID, o.Grid.Type.Unit)
		}
	}
	if len(hours) != 4 {
		t.Errorf("%d hours drawn; want the three past and the current one", len(hours))
	}
	if !slices.Contains(got.Chips[tty.TemperatureLayer], "O-METEO") {
		t.Errorf("Open-Meteo drawn without its chip: %v (D-133)", got.Chips)
	}
}

func TestForecastModeDrawsEachDayDuringItsStep(t *testing.T) {
	src := &fakeTemp{name: "NDFD", now: tempNow}
	ask := tempAsk(true)
	got := buildTemperature(context.Background(), src, nil, ask, tempNow, nil)
	steps := tty.ForecastSteps(ask.Anchor)
	nows := 0
	for _, o := range got.Overlays {
		if o.During != steps[0].Span {
			t.Errorf("%s spans %v; want Now's", o.ID, o.During)
		}
		nows++
	}
	if nows == 0 {
		t.Fatal("no grid for Now")
	}
	if len(got.High) != temperature.Days-1 || len(got.Low) != temperature.Days {
		t.Fatalf("%d highs and %d lows; want every day's, less today's high, which the source lacks", len(got.High), len(got.Low))
	}
	for _, o := range got.Low {
		k := int(o.ID[strings.Index(o.ID, "/d")+2] - '0')
		if o.During != steps[k+1].Span {
			t.Errorf("%s spans %v; want its day's step, %s", o.ID, o.During, steps[k+1].Label)
		}
	}
	if !strings.Contains(strings.Join(got.Notes, " "), "No high for Today") {
		t.Errorf("today's missing high was not said: %v", got.Notes)
	}
	if strings.Contains(strings.Join(got.Notes, " "), "CC BY") {
		t.Error("NDFD drawn with Open-Meteo's credit")
	}
}

func TestTheModeChoosesTheSource(t *testing.T) {
	ts := &tempSources{ndfd: &fakeTemp{name: "NDFD"}, om: &fakeTemp{name: "Open-Meteo"}}
	ask := tempAsk(false)
	if got := ts.sourceFor(ask).Name(); got != "Open-Meteo" {
		t.Errorf("Radar mode asks %s; want Open-Meteo, the one with past hours (D-96)", got)
	}
	ask.Forecast, ask.TempNDFD = true, false
	ts.ndfd = temperature.NewNDFD(nil, "")
	if got := ts.sourceFor(ask).Name(); got != "Open-Meteo" {
		t.Errorf("Forecast mode asks %s; want Open-Meteo by default (D-101)", got)
	}
	ask.TempNDFD = true
	if got := ts.sourceFor(ask).Name(); got != "NDFD" {
		t.Errorf("Forecast mode with NDFD chosen asks %s", got)
	}
	ask.Region = geo.RegionSamoa
	if got := ts.sourceFor(ask).Name(); got != "Open-Meteo" {
		t.Errorf("NDFD has no American Samoa, and %s was asked", got)
	}
}

func TestATemperatureThatDidNotAnswerIsSaid(t *testing.T) {
	src := &fakeTemp{name: "NDFD", now: tempNow, failed: true}
	got := buildTemperature(context.Background(), src, nil, tempAsk(true), tempNow, nil)
	if len(got.Overlays)+len(got.High)+len(got.Low) != 0 || !strings.Contains(strings.Join(got.Notes, " "), "NDFD did not answer") { // with its path (D-124)
		t.Errorf("a failed source gave %d grids and the notes %v", len(got.Overlays)+len(got.High), got.Notes)
	}
}

func TestTemperatureIsAskedForTheRadarsBoxesNeverTheView(t *testing.T) {
	src := &fakeTemp{name: "Open-Meteo", now: tempNow}
	ask := tempAsk(false)
	got := buildTemperature(context.Background(), src, nil, ask, tempNow, nil)
	for _, o := range got.Overlays {
		g := o.Grid
		if g.West == ask.View.W || g.East == ask.View.E {
			t.Errorf("%s is the view's box; a source is asked for a fixed box (D-47)", o.ID)
		}
	}
	if src.asked == 0 {
		t.Fatal("nothing was asked")
	}
}

func TestEveryFeedOverlayIsTimed(t *testing.T) {
	_, feed := feedFor(t, "03-two-alerts-harlan", nil)
	if len(feed.Overlays) == 0 {
		t.Fatal("the scenario drew nothing")
	}
	for _, o := range feed.Overlays {
		tm, ok := feed.Times[o.ID]
		if !ok || tm.From.IsZero() {
			t.Errorf("%s has no time: the mode cannot say when it is drawn (D-98)", o.ID)
		}
	}
	_ = tuimaps.Span{}
}

func TestAnAlertIsTimedByItsHazardNotItsProduct(t *testing.T) {
	effective, onset := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC), time.Date(2026, 8, 25, 6, 0, 0, 0, time.UTC)
	expires, ends := time.Date(2026, 8, 24, 20, 0, 0, 0, time.UTC), time.Date(2026, 8, 25, 18, 0, 0, 0, time.UTC)
	a := snapshot.Alert{Sent: effective.Add(-time.Minute), Effective: effective, Expires: expires}
	if got := alertTimes(a); !got.From.Equal(effective) || !got.Until.Equal(expires) {
		t.Errorf("with no onset or end: %v to %v; want effective to expires", got.From, got.Until)
	}
	a.Onset, a.Ends = &onset, &ends
	if got := alertTimes(a); !got.From.Equal(onset) || !got.Until.Equal(ends) {
		t.Errorf("a watch for tomorrow: %v to %v; want its onset to its end, the hazard's own (D-98)", got.From, got.Until)
	}
	a.Effective = time.Time{}
	a.Onset = nil
	if got := alertTimes(a); !got.From.Equal(a.Sent) {
		t.Errorf("with no effective time: from %v; want when it was sent", got.From)
	}
}

// TestAnAnswerRepeatsWithinTheHour is UAT-2 U2-13: two answers inside one
// hour are the same overlays, so the window hands nothing in again - an
// overlay's currency worked out from the clock made every answer new, and
// every grid blinked.
func TestAnAnswerRepeatsWithinTheHour(t *testing.T) {
	for _, forecast := range []bool{false, true} {
		src := &fakeTemp{name: "Open-Meteo", now: tempNow}
		a := buildTemperature(context.Background(), src, nil, tempAsk(forecast), tempNow, nil)
		b := buildTemperature(context.Background(), src, nil, tempAsk(forecast), tempNow.Add(20*time.Minute), nil)
		if !sameAnswer(a, b) {
			t.Errorf("forecast %v: two answers twenty minutes apart differ", forecast)
		}
	}
}

// TestADayTheSourceLacksIsFilledFromOpenMeteo is D-100: NDFD has no high for
// today after its daytime; Open-Meteo's fills it, said so and credited.
func TestADayTheSourceLacksIsFilledFromOpenMeteo(t *testing.T) {
	ndfd, om := &fakeTemp{name: "NDFD", now: tempNow}, &fakeTemp{name: "Open-Meteo", now: tempNow}
	ask := tempAsk(true)
	got := buildTemperature(context.Background(), ndfd, &noGap{om}, ask, tempNow, nil)
	if len(got.High) != temperature.Days {
		t.Fatalf("%d highs; want every day's, today's from Open-Meteo", len(got.High))
	}
	if !got.Filled["0/high"] || got.Filled["1/high"] || got.Filled["0/low"] {
		t.Errorf("filled %v; want today's high alone", got.Filled)
	}
	notes := strings.Join(got.Notes, " ")
	if !slices.Contains(got.Chips[tty.TemperatureLayer], "O-METEO") || strings.Contains(notes, "No high for Today") {
		t.Errorf("the notes are %q, the chips %v; want Open-Meteo's chip and no gap said", notes, got.Chips)
	}
	if om.asked == 0 {
		t.Error("Open-Meteo was never asked")
	}
	// Neither has it: nothing is marked filled, and the gap is said.
	both := buildTemperature(context.Background(), &fakeTemp{name: "NDFD", now: tempNow}, &fakeTemp{name: "Open-Meteo", now: tempNow}, ask, tempNow, nil)
	if both.Filled["0/high"] || !strings.Contains(strings.Join(both.Notes, " "), "No high for Today") {
		t.Errorf("with neither source holding today's high: filled %v, notes %v", both.Filled, both.Notes)
	}
	full := &fakeTemp{name: "NDFD", now: tempNow}
	om2 := &fakeTemp{name: "Open-Meteo", now: tempNow}
	_ = buildTemperature(context.Background(), &noGap{full}, om2, ask, tempNow, nil)
	if om2.asked != 0 {
		t.Error("Open-Meteo was asked although the chosen source had every day")
	}
}

// noGap is a source with every day's values.
type noGap struct{ *fakeTemp }

func (n *noGap) Fetch(ctx context.Context, l temperature.Lattice, now time.Time) (temperature.Series, error) {
	s, err := n.fakeTemp.Fetch(ctx, l, now)
	s.High[0] = s.High[1]
	return s, err
}

func TestTemperatureIsOffByDefault(t *testing.T) {
	for _, l := range mapLayers {
		if l.key == tty.TemperatureLayer && l.on {
			t.Error("temperature is on by default; D-99 has it off, loaded in the background")
		}
	}
}

// TestABoxNDFDRefusesFallsBackToOpenMeteo is D-101 (UAT-2 U2-17): NDFD
// refuses Hawaii's whole lattice, which straddles its grid's edge. That box
// is Open-Meteo's, said and credited; every box so, the chip names it.
func TestABoxNDFDRefusesFallsBackToOpenMeteo(t *testing.T) {
	ask := tempAsk(true)
	ask.Region, ask.View = geo.RegionHawaii, geo.Box{W: -160, S: 19, E: -155, N: 22}
	got := buildTemperature(context.Background(), &fakeTemp{name: "NDFD", failed: true}, &noGap{&fakeTemp{name: "Open-Meteo", now: tempNow}}, ask, tempNow, nil)
	if len(got.High) == 0 || got.Source != "Open-Meteo" {
		t.Fatalf("%d highs from %q; want Hawaii from Open-Meteo, named", len(got.High), got.Source)
	}
	notes := strings.Join(got.Notes, " ")
	if !strings.Contains(notes, "NDFD did not answer") || !strings.Contains(notes, "Open-Meteo") || !slices.Contains(got.Chips[tty.TemperatureLayer], "O-METEO") {
		t.Errorf("the fallback was not said and credited: %q, chips %v", notes, got.Chips)
	}
}

// TestEveryTemperatureGridIsLined is D-102: one look in both modes -
// labelled isotherms over faint bands - asked of the library on every grid.
func TestEveryTemperatureGridIsLined(t *testing.T) {
	for _, forecast := range []bool{false, true} {
		got := buildTemperature(context.Background(), &noGap{&fakeTemp{name: "Open-Meteo", now: tempNow}}, nil, tempAsk(forecast), tempNow, nil)
		for _, o := range append(append(got.Overlays, got.High...), got.Low...) {
			if !o.Grid.Lines {
				t.Errorf("forecast %v: %s is not lined", forecast, o.ID)
			}
		}
	}
}

// TestWindGridsFollowTheModes is W11.2 (D-108): Radar mode's every hour and
// Forecast mode's Now and each day's peak, each with its span, in the
// listener's unit, each a vector grid.
func TestWindGridsFollowTheModes(t *testing.T) {
	src := &noGap{&fakeTemp{name: "Open-Meteo", now: tempNow}}
	radarMode := buildTemperature(context.Background(), src, nil, tempAsk(false), tempNow, nil)
	if len(radarMode.Wind) != 4 || len(radarMode.WindDays) != 0 {
		t.Fatalf("Radar mode's wind is %d hours and %d days; want the four hours up to now", len(radarMode.Wind), len(radarMode.WindDays))
	}
	for _, o := range radarMode.Wind {
		if o.Grid.From == nil || o.Grid.Type.Unit != "mph" || math.Abs(o.Grid.Values[0]-10) > 1e-6 || o.Grid.From[0] != 270 {
			t.Errorf("%s: %v mph from %v (%q); want a vector grid of 10 mph from 270", o.ID, o.Grid.Values[0], o.Grid.From, o.Grid.Type.Unit)
		}
		if o.During.Until.Sub(o.During.From) != time.Hour-time.Nanosecond || !strings.HasPrefix(o.ID, tty.WindLayer+"/") {
			t.Errorf("%s spans %v", o.ID, o.During)
		}
	}
	ask := tempAsk(true)
	fc := buildTemperature(context.Background(), src, nil, ask, tempNow, nil)
	steps := tty.ForecastSteps(ask.Anchor)
	if len(fc.Wind) == 0 || fc.Wind[0].During != steps[0].Span {
		t.Fatalf("Forecast mode's Now wind is %d grids", len(fc.Wind))
	}
	if len(fc.WindDays) != temperature.Days || fc.WindDays[1].During != steps[2].Span || math.Abs(fc.WindDays[1].Grid.Values[0]-20) > 1e-6 {
		t.Errorf("the days' wind is %d grids; want every day's peak, 20 mph, during its step", len(fc.WindDays))
	}
	ask.Fahrenheit = false
	if kmh := buildTemperature(context.Background(), src, nil, ask, tempNow, nil); kmh.Wind[0].Grid.Type.Unit != "km/h" {
		t.Errorf("the metric listener's wind is in %q", kmh.Wind[0].Grid.Type.Unit)
	}
}

func TestWindIsOffByDefault(t *testing.T) {
	found := false
	for _, l := range mapLayers {
		if l.key == tty.WindLayer {
			found = true
			if l.on {
				t.Error("wind is on by default; D-110 has it off")
			}
		}
	}
	if !found {
		t.Error("no wind layer is registered")
	}
}

// TestEveryRegionHasItsFieldsWhole is D-111 (UAT-2 U2-26): temperature and
// wind are asked for boxes of their own; outside the lower 48 they cover the
// whole region - Hawaii's wind stopped at MRMS's box inside the map - split at
// the antimeridian, which a grid cannot cross. Every region draws, American
// Samoa among them.
func TestEveryRegionHasItsFieldsWhole(t *testing.T) {
	for _, r := range geo.Regions() {
		view := geo.Box{W: r.W, S: r.S, E: r.E, N: r.N}
		boxes := fieldBoxes(r.Name, view)
		if len(boxes) == 0 {
			t.Errorf("%s has no field box: it draws no temperature or wind", r.Name)
			continue
		}
		if r.Name == geo.RegionContiguous {
			// THE RADAR'S BOXES, GROWN TO THE REGION'S EDGES (UAT-2 U2-30): they
			// stop at 126W and 65W, and the region reaches 130W and 64W - the
			// waves stopped at a line in the Pacific. At any view, the boxes
			// that meet an edge of the radar's reach reach the region's.
			for _, v := range []geo.Box{view, {W: -129, S: 40, E: -121, N: 46}, {W: -70, S: 38, E: -64, N: 44}} {
				bs := fieldBoxes(r.Name, v)
				w, e, s, n := 180.0, -180.0, 90.0, -90.0
				for _, b := range bs {
					w, e, s, n = math.Min(w, b.W), math.Max(e, b.E), math.Min(s, b.S), math.Max(n, b.N)
				}
				if v.W <= r.W+1 && w > r.W || v.E >= r.E-1 && e < r.E || v.S <= r.S+1 && s > r.S || v.N >= r.N-1 && n < r.N {
					t.Errorf("the lower 48 at %+v: its boxes span %v to %v, %v to %v; want the region's edges, %v to %v", v, w, e, s, n, r.W, r.E)
				}
			}
			continue
		}
		width := 0.0
		for _, b := range boxes {
			if !(b.W >= -180 && b.E <= 180 && b.W < b.E && b.S == r.S && b.N == r.N) {
				t.Errorf("%s: a box %+v is not a grid inside the region's latitudes", r.Name, b)
			}
			width += b.E - b.W
		}
		want := r.E - r.W
		if want < 0 {
			want += 360
		}
		if math.Abs(width-want) > 1e-9 {
			t.Errorf("%s: the boxes span %v degrees of the region's %v", r.Name, width, want)
		}
	}
	ask := tempAsk(true)
	ask.Region = geo.RegionSamoa
	if got := buildTemperature(context.Background(), &noGap{&fakeTemp{name: "Open-Meteo", now: tempNow}}, nil, ask, tempNow, nil); len(got.High) == 0 || len(got.WindDays) == 0 {
		t.Errorf("American Samoa drew %d highs and %d wind days; want both", len(got.High), len(got.WindDays))
	}
}

// TestTheWindSaysItsGustsAsAviationDoes is D-136: a gust is handed in beside
// the speed, in the listener's unit, where it beats the sustained wind by 10
// mph - 30 on a 10 mph wind - and NaN where it does not - 25 on a 20 mph
// day's peak; so the arrows read "10G30" and "20".
func TestTheWindSaysItsGustsAsAviationDoes(t *testing.T) {
	src := &noGap{&fakeTemp{name: "Open-Meteo", now: tempNow}}
	radarMode := buildTemperature(context.Background(), src, nil, tempAsk(false), tempNow, nil)
	for _, o := range radarMode.Wind {
		if len(o.Grid.Gusts) != len(o.Grid.Values) || math.Abs(o.Grid.Gusts[0]-30) > 1e-6 {
			t.Errorf("%s gusts %v; want 30 mph beside 10", o.ID, o.Grid.Gusts)
		}
	}
	fc := buildTemperature(context.Background(), src, nil, tempAsk(true), tempNow, nil)
	if len(fc.Wind) == 0 || len(fc.Wind[0].Grid.Gusts) == 0 || math.Abs(fc.Wind[0].Grid.Gusts[0]-30) > 1e-6 {
		t.Errorf("Forecast mode's Now does not say its gust: %v", fc.Wind)
	}
	for k, o := range fc.WindDays {
		if k == 1 {
			if len(o.Grid.Gusts) == 0 || math.Abs(o.Grid.Gusts[0]-40) > 1e-6 {
				t.Errorf("day 1 gusts %v; want 40 mph over its 20 mph peak", o.Grid.Gusts)
			}
			continue
		}
		for _, g := range o.Grid.Gusts {
			if !math.IsNaN(g) {
				t.Fatalf("%s says a gust of %v, 5 mph over its peak; want none said", o.ID, g)
			}
		}
	}
	ask := tempAsk(false)
	ask.Fahrenheit = false
	if kmh := buildTemperature(context.Background(), src, nil, ask, tempNow, nil); math.Abs(kmh.Wind[0].Grid.Gusts[0]-48.28032) > 1e-6 {
		t.Errorf("the metric listener's gust is %v; want 48.3 km/h", kmh.Wind[0].Grid.Gusts[0])
	}
}

// TestAFilledDayKeepsItsGust is D-100 with D-136: a day whose wind the source
// lacks takes Open-Meteo's peak, and its gust comes with it.
func TestAFilledDayKeepsItsGust(t *testing.T) {
	ndfd, om := &fakeTemp{name: "NDFD", now: tempNow, noWindDay: true}, &fakeTemp{name: "Open-Meteo", now: tempNow}
	got := buildTemperature(context.Background(), ndfd, &noGap{om}, tempAsk(true), tempNow, nil)
	if !got.Filled["1/wind"] || len(got.WindDays) < 2 || len(got.WindDays[1].Grid.Gusts) == 0 || math.Abs(got.WindDays[1].Grid.Gusts[0]-40) > 1e-6 {
		t.Errorf("filled %v; day 1's gusts %v; want Open-Meteo's wind and its 40 mph gust", got.Filled, got.WindDays)
	}
}

// sameAnswer is whether two answers are the same as the window reads them:
// every overlay the one it already holds (tty.SameOverlay - a missing value
// matching a missing one, D-136), and all else equal.
func sameAnswer(a, b tty.MapTemperature) bool {
	lists := func(t *tty.MapTemperature) []*[]tuimaps.Overlay {
		return []*[]tuimaps.Overlay{&t.Overlays, &t.High, &t.Low, &t.Wind, &t.WindDays, &t.Rain, &t.Feels, &t.FeelsHigh, &t.FeelsLow, &t.Waves, &t.WaveDays}
	}
	la, lb := lists(&a), lists(&b)
	for i := range la {
		if len(*la[i]) != len(*lb[i]) {
			return false
		}
		for j := range *la[i] {
			if !tty.SameOverlay((*la[i])[j], (*lb[i])[j]) {
				return false
			}
		}
		*la[i], *lb[i] = nil, nil
	}
	return reflect.DeepEqual(a, b)
}

// A GRID IS CURRENT THROUGH ITS HOUR, AND AN ASK WITH NO ANCHOR IS THE
// CURRENT HOUR'S (W14: held once the grids share linedGrid and askAnchor). An
// hour past keeps three hours from the anchor, so a loop's earlier hours are
// never drawn stale; an ask the window made before it knew its hour builds
// what one anchored at the start of the hour builds.
func TestAGridIsCurrentThroughItsHourAndTheAnchorIsTheHour(t *testing.T) {
	src := &fakeTemp{name: "Open-Meteo", now: tempNow}
	anchor := tempNow.Truncate(time.Hour)
	got := buildTemperature(context.Background(), src, nil, tempAsk(false), tempNow, nil)
	if len(got.Overlays) == 0 {
		t.Fatal("no hourly grid was built")
	}
	for _, o := range append(got.Overlays, got.Wind...) {
		if want := anchor.Sub(o.Valid) + 3*time.Hour; o.Keeps != want {
			t.Errorf("%s keeps %v; want %v, three hours past the anchor", o.ID, o.Keeps, want)
		}
	}
	anchored, unanchored := tempAsk(true), tempAsk(true)
	unanchored.Anchor = time.Time{}
	a := buildTemperature(context.Background(), src, nil, anchored, tempNow, nil)
	b := buildTemperature(context.Background(), src, nil, unanchored, tempNow, nil)
	spans := func(os ...[]tuimaps.Overlay) []string {
		var out []string
		for _, list := range os {
			for _, o := range list {
				out = append(out, o.ID+" "+o.During.From.String()+" "+o.During.Until.String())
			}
		}
		return out
	}
	if x, y := spans(a.Overlays, a.High, a.Low), spans(b.Overlays, b.High, b.Low); !slices.Equal(x, y) || len(x) == 0 {
		t.Errorf("an ask with no anchor built\n%v\nwhere the hour's anchor built\n%v", y, x)
	}
}

// OPEN-METEO'S SPENT QUOTA REACHES THE MAP, AND IS NOT ASKED AGAIN (W18.1,
// D-165): the answer carries the quota the gate holds, so the window can say
// it; and the next ask does not go out to be refused again.
func TestOpenMeteosSpentQuotaReachesTheMap(t *testing.T) {
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":true,"reason":"Daily API request limit exceeded. Please try again tomorrow."}`))
	}))
	defer srv.Close()
	c, err := httpx.New(httpx.Config{UserAgent: "t", RatePerSec: 100, RetryBase: time.Millisecond, MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}
	ndfd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer ndfd.Close()
	lp := &livePipelines{temp: tempSourcesAt(c, srv.URL, ndfd.URL)} // NDFD apart: only Open-Meteo's asks are counted
	got := lp.mapTemperature(context.Background(), tempAsk(false))
	if got.Quota == nil || got.Quota.Period != "Daily" || got.Quota.Source != "Open-Meteo" {
		t.Fatalf("the answer does not say Open-Meteo's daily quota is spent: %+v", got.Quota)
	}
	before := asked.Load()
	lp.mapTemperature(context.Background(), tempAsk(false))
	if n := asked.Load() - before; n != 0 {
		t.Errorf("a spent quota was asked again %d times", n)
	}
}

// OPEN-METEO REFUSED, NDFD DRAWS (W18.2, D-165, D-166): every box Open-Meteo
// does not answer is asked of NDFD; the chips say NDFD; and in Radar mode
// NDFD's current hour lies under the loop's earlier frames too - it has no
// hour before it (D-166's cold start). With Open-Meteo answering, nothing is
// stretched and NDFD is not asked.
func TestNDFDDrawsWhereOpenMeteoRefused(t *testing.T) {
	anchor := tempNow.Truncate(time.Hour)
	for _, forecast := range []bool{false, true} {
		ndfd := &fakeTemp{name: "NDFD", now: tempNow}
		got := buildTemperature(context.Background(), &fakeTemp{name: "Open-Meteo", now: tempNow, failed: true}, nil, tempAsk(forecast), tempNow, &fallback{src: ndfd})
		if len(got.Overlays) == 0 || len(got.Feels) == 0 || len(got.Wind) == 0 {
			t.Fatalf("forecast %v: NDFD drew %d temperature, %d feels-like, %d wind grids", forecast, len(got.Overlays), len(got.Feels), len(got.Wind))
		}
		if got.Source != "NDFD" || !slices.Equal(got.Chips[tty.TemperatureLayer], []string{"NDFD"}) {
			t.Errorf("forecast %v: drawn as %q, chips %v; want NDFD's", forecast, got.Source, got.Chips[tty.TemperatureLayer])
		}
		if forecast {
			continue
		}
		stretched := 0
		for _, list := range [][]tuimaps.Overlay{got.Overlays, got.Feels, got.Wind} {
			for _, o := range list {
				if o.During.From.Equal(anchor.Add(-time.Hour)) && o.During.Until.Equal(anchor.Add(time.Hour-time.Nanosecond)) {
					stretched++
				}
			}
		}
		if stretched != 3 {
			t.Errorf("%d current-hour grids reach under the loop's earlier frames; want temperature's, feels-like's and wind's", stretched)
		}
	}
	answered := buildTemperature(context.Background(), &fakeTemp{name: "Open-Meteo", now: tempNow}, nil, tempAsk(false), tempNow, &fallback{src: &fakeTemp{name: "NDFD", now: tempNow, failed: true}})
	for _, o := range answered.Overlays {
		if o.During.From.Before(o.During.Until.Add(-time.Hour)) {
			t.Errorf("%s is stretched though Open-Meteo answered", o.ID)
		}
	}
	if answered.Source != "Open-Meteo" {
		t.Errorf("with Open-Meteo answering, drawn as %q", answered.Source)
	}
}
