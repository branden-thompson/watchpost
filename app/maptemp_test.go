package app

// maptemp_test.go — 0.18.0 W10 in the app (D-93 to D-98): the mode's source,
// the grids built for each hour or step with its span, the listener's unit,
// what a source lacks said, the credit, and every feed overlay timed.

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
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// fakeTemp answers every lattice with fixed values: 10 C every hour from
// three hours back to one on, each day's high 20 C and low 5 C - but no high
// today, as NDFD after its daytime.
type fakeTemp struct {
	name   string
	now    time.Time
	failed bool
	asked  int
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
	}
	for k := range temperature.Days {
		s.High[k], s.Low[k] = fill(20), fill(5)
	}
	s.High[0] = fill(math.NaN())
	return s, nil
}

var tempNow = time.Date(2026, 8, 24, 20, 30, 0, 0, time.UTC)

func tempAsk(forecast bool) tty.MapAsk {
	return tty.MapAsk{Region: geo.RegionContiguous, View: geo.Box{W: -120, S: 32, E: -115, N: 36}, Forecast: forecast,
		Anchor: tempNow.Truncate(time.Hour), Fahrenheit: true}
}

func TestRadarModeDrawsEachHourDuringItself(t *testing.T) {
	src := &fakeTemp{name: "Open-Meteo", now: tempNow}
	got := buildTemperature(context.Background(), src, tempAsk(false), tempNow)
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
	if !strings.Contains(strings.Join(got.Notes, " "), "CC BY 4.0") {
		t.Errorf("Open-Meteo drawn without its credit: %v", got.Notes)
	}
}

func TestForecastModeDrawsEachDayDuringItsStep(t *testing.T) {
	src := &fakeTemp{name: "NDFD", now: tempNow}
	ask := tempAsk(true)
	got := buildTemperature(context.Background(), src, ask, tempNow)
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
	ask.Forecast, ask.TempOpenMeteo = true, false
	ts.ndfd = temperature.NewNDFD(nil, "")
	if got := ts.sourceFor(ask).Name(); got != "NDFD" {
		t.Errorf("Forecast mode asks %s; want NDFD by default (D-93)", got)
	}
	ask.TempOpenMeteo = true
	if got := ts.sourceFor(ask).Name(); got != "Open-Meteo" {
		t.Errorf("Forecast mode with Open-Meteo chosen asks %s", got)
	}
	ask.TempOpenMeteo, ask.Region = false, geo.RegionSamoa
	if got := ts.sourceFor(ask).Name(); got != "Open-Meteo" {
		t.Errorf("NDFD has no American Samoa, and %s was asked", got)
	}
}

func TestATemperatureThatDidNotAnswerIsSaid(t *testing.T) {
	src := &fakeTemp{name: "NDFD", now: tempNow, failed: true}
	got := buildTemperature(context.Background(), src, tempAsk(true), tempNow)
	if len(got.Overlays)+len(got.High)+len(got.Low) != 0 || !strings.Contains(strings.Join(got.Notes, " "), "NDFD did not answer") {
		t.Errorf("a failed source gave %d grids and the notes %v", len(got.Overlays)+len(got.High), got.Notes)
	}
}

func TestTemperatureIsAskedForTheRadarsBoxesNeverTheView(t *testing.T) {
	src := &fakeTemp{name: "Open-Meteo", now: tempNow}
	ask := tempAsk(false)
	got := buildTemperature(context.Background(), src, ask, tempNow)
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
