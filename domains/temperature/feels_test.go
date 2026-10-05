package temperature

// feels_test.go — 0.18.0 D-119 over the recorded answers: the apparent
// temperature in the requests temperature already makes.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// feelsGet answers with the feels-like fixtures, recording the addresses.
type feelsGet struct {
	t    *testing.T
	asks []string
}

func (f *feelsGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	f.asks = append(f.asks, rawURL)
	switch {
	case strings.Contains(rawURL, "/v1/forecast"):
		return fixture(f.t, "openmeteo-feels.json"), nil
	case strings.Contains(rawURL, "temp=temp"):
		return fixture(f.t, "ndfd-hour-feels.xml"), nil
	}
	return fixture(f.t, "ndfd-days-feels.xml"), nil
}

// feelsCaptured is when the feels-like fixtures were recorded: 16:51 in San
// Diego.
var feelsCaptured = time.Date(2026, 9, 27, 23, 51, 0, 0, time.UTC)

// TestOpenMeteoReadsFeelsLike is D-119: each hour's apparent temperature,
// and each day's high and low, in the request temperature makes.
func TestOpenMeteoReadsFeelsLike(t *testing.T) {
	g := &feelsGet{t: t}
	s, err := NewOpenMeteo(g, "").Fetch(context.Background(), fixtureLattice, feelsCaptured)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.asks) != 1 || !strings.Contains(g.asks[0], "apparent_temperature_max") {
		t.Errorf("asked %v; want feels-like in temperature's one request", g.asks)
	}
	now, ok := s.FeelsAt(feelsCaptured)
	if !ok || !near(now[escondido], 27.9) {
		t.Errorf("Escondido feels %v now (%v); want 27.9, 16:00's", now, ok)
	}
	if !near(s.FeelsHigh[0][escondido], 30.0) || !near(s.FeelsLow[1][escondido], 13.3) {
		t.Errorf("today feels as high as %v, tomorrow as low as %v; want 30.0 and 13.3", s.FeelsHigh[0][escondido], s.FeelsLow[1][escondido])
	}
}

// TestNDFDWorksOutEachDaysFeelsLike is D-119: NDFD has no daily apparent
// temperature; each day's high and low are its hours', on the local date.
func TestNDFDWorksOutEachDaysFeelsLike(t *testing.T) {
	g := &feelsGet{t: t}
	s, err := NewNDFD(g, "").Fetch(context.Background(), fixtureLattice, feelsCaptured)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range g.asks {
		if !strings.Contains(a, "appt=appt") {
			t.Errorf("%s does not ask for feels-like", a)
		}
	}
	if !near(s.FeelsHigh[1][escondido], cOf(78)) || !near(s.FeelsLow[1][escondido], cOf(57)) {
		t.Errorf("tomorrow feels %v to %v; want 57 to 78 F", s.FeelsLow[1][escondido], s.FeelsHigh[1][escondido])
	}
	if !near(s.FeelsHigh[0][escondido], cOf(80)) {
		t.Errorf("today's feels-like high is %v; want 80 F, its hours' highest", s.FeelsHigh[0][escondido])
	}
}
