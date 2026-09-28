package app

// mapproblems_test.go — 0.18.0 D-124 (UAT-2 U2-29): "We should never show
// error messages to the end user unless we give them a path to resolve it."
// What went wrong is said with its path, or goes to the diagnostics alone.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// TestASourceThatDidNotAnswerGivesItsPath: a source that did not answer is
// said only where a Setting offers another, with the Setting; where none
// does, it goes to the diagnostics.
func TestASourceThatDidNotAnswerGivesItsPath(t *testing.T) {
	ndfd := buildTemperature(context.Background(), &fakeTemp{name: "NDFD", now: tempNow, failed: true}, nil, tempAsk(true), tempNow)
	if notes := strings.Join(ndfd.Notes, " "); !strings.Contains(notes, "Settings → Maps → Temperature") {
		t.Errorf("NDFD not answering is said %q; want the Setting that draws Open-Meteo instead", notes)
	}
	om := buildTemperature(context.Background(), &fakeTemp{name: "Open-Meteo", now: tempNow, failed: true}, nil, tempAsk(false), tempNow)
	if strings.Contains(strings.Join(om.Notes, " "), "did not answer") || len(om.Problems) == 0 {
		t.Errorf("Open-Meteo not answering is said %v, told the diagnostics %v; no Setting offers another", om.Notes, om.Problems)
	}

	mrms := &fakeRadar{name: "MRMS", regions: []string{geo.RegionContiguous, geo.RegionHawaii}}
	iem := &fakeRadar{name: "IEM", regions: []string{geo.RegionContiguous}}
	lp := &livePipelines{radar: &radarSources{iem: iem, mrms: mrms}}
	lower48 := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionContiguous, View: tty.MapView{W: -120, S: 32, E: -115, N: 36}})
	if !strings.Contains(lower48.Note, "Settings → Maps → Radar") {
		t.Errorf("the lower 48's radar not answering is said %q; want the Setting for the other source", lower48.Note)
	}
	hawaii := lp.mapRadar(context.Background(), tty.MapAsk{Region: geo.RegionHawaii})
	if hawaii.Note != "" || len(hawaii.Problems) == 0 {
		t.Errorf("Hawaii's radar not answering is said %q, told the diagnostics %v; no other source covers it", hawaii.Note, hawaii.Problems)
	}
}

// failGet answers nothing.
type failGet struct{}

func (failGet) GetText(context.Context, string, ...httpx.Option) ([]byte, error) {
	return nil, errors.New("no answer")
}

// TestWhatNoSettingFixesGoesToTheDiagnostics: the hours ahead and the rain
// and snow not answering are nothing the listener can act on.
func TestWhatNoSettingFixesGoesToTheDiagnostics(t *testing.T) {
	newest := time.Date(2026, 9, 27, 20, 40, 0, 0, time.UTC)
	hrrr := withForecast(context.Background(), tty.MapRadar{Note: ""}, radar.NewHRRR(failGet{}, ""), nil, newest, newest.Add(3*time.Hour))
	model := withModelRain(context.Background(), tty.MapRadar{}, temperature.NewOpenMeteo(failGet{}, ""), geo.RegionHawaii, geo.Box{}, newest, newest.Add(3*time.Hour))
	rain := withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(failGet{}, ""), tempAsk(true), tempNow)
	for name, c := range map[string]struct {
		said     string
		problems []string
	}{"HRRR": {hrrr.Note, hrrr.Problems}, "Open-Meteo ahead": {model.Note, model.Problems}, "rain": {strings.Join(rain.RainNotes, " "), rain.Problems}} {
		if c.said != "" || len(c.problems) == 0 {
			t.Errorf("%s not answering is said %q, told the diagnostics %v", name, c.said, c.problems)
		}
	}
}

// TestTheDiagnosticsHoldTheMapsProblems is D-124's wiring: the window's
// problems reach the diagnostic dump, the last fifty.
func TestTheDiagnosticsHoldTheMapsProblems(t *testing.T) {
	lp := &livePipelines{}
	cfg := lp.ttyConfig("t", Options{}, false, config.Config{}, nil, nil, nil, nil, nil, nil)
	if cfg.MapProblem == nil {
		t.Fatal("the window is handed no way to tell the diagnostics")
	}
	for i := range 60 {
		cfg.MapProblem("Earthquakes: " + string(rune('A'+i%26)) + " not drawn")
	}
	st := lp.ttyStats()
	if len(st.MapProblems) != 50 || !strings.Contains(st.MapProblems[49], "Earthquakes") {
		t.Errorf("the diagnostics hold %d problems; want the last fifty", len(st.MapProblems))
	}
}
