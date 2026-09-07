package synth

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func fx(v float64) *float64 { return &v }

// marineFixture is a coastal location with everything: a buoy, a swell, tides
// and currents.
func marineFixture(now time.Time) MarineReport {
	return MarineReport{
		Known: true,
		TZ:    time.UTC,
		State: snapshot.Marine{
			Buoy: "46086", BuoyDistanceKM: fx(9.7), ObservedAt: now.Add(-39 * time.Minute),
			WaveHeight: fx(0.9), SwellHeight: fx(0.6), SwellDirDeg: fx(270), WavePeriod: fx(14),
			WaterTemp: fx(23.3), WindSpeed: fx(5.7), WindGust: fx(8.2),
			TideLevel: fx(1.13), TideStation: "La Jolla (9410230)", TideStationKM: fx(38.6),
			Tides: []snapshot.TideEvent{
				{Type: "H", Time: now.Add(2 * time.Hour), Height: 1.74},
				{Type: "L", Time: now.Add(9 * time.Hour), Height: -0.03},
			},
			Currents: []snapshot.CurrentEvent{
				{Type: "flood", Time: now.Add(-30 * time.Minute), Speed: 0.72},
				{Type: "slack", Time: now.Add(50 * time.Minute)},
			},
		},
	}
}

func joinSegs(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
		b.WriteString("\n")
	}
	return b.String()
}

func TestMarineSegmentsReadTheWholeReport(t *testing.T) {
	now := time.Date(2026, 8, 30, 15, 0, 0, 0, time.UTC)
	segs := std.MarineSegments("Oceanside, CA", marineFixture(now), true, now)
	if len(segs) == 0 {
		t.Fatal("a coastal location composes a maritime report")
	}
	got := joinSegs(segs)
	for _, want := range []string{
		"This is the Watchpost Marine report for Oceanside, California",
		"National Data Buoy Center",
		"observed 39 minutes ago",
		"Seas are slight chop",
		"a primary swell from the west",
		"dominant period of 14 seconds",
		"Water temperature",
		"Wind at the buoy",
		"gusting to",
		"The tide is rising",
		"above the low-water mark", // MVS-D-21: not "mean lower low water"
		"La Jolla",                 // the station id is cut
		"The next high tide is at 5:00 PM",
		"the next low at 12:00 AM",
		"Tidal currents are flooding at",
		"slack water at 3:50 PM",
		"https://www.ndbc.noaa.gov",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// The buoy id itself is deliberately NOT spoken — it reads as noise.
	if strings.Contains(got, "46086") {
		t.Error("the buoy id must not be spoken")
	}
	// Every segment is the Maritime correspondent's.
	for _, s := range segs {
		if s.Role != cast.Maritime {
			t.Errorf("segment %q has role %v, want maritime", s.Key, s.Role)
		}
	}
}

// Imperial and metric read differently, and currents are ALWAYS knots
// (MVS-D-21) — the mariner's convention, whatever the listener's unit.
func TestMarineReadsTheListenersUnitsButCurrentsInKnots(t *testing.T) {
	now := time.Date(2026, 8, 30, 15, 0, 0, 0, time.UTC)
	imperial := joinSegs(std.MarineSegments("Oceanside, CA", marineFixture(now), true, now))
	metric := joinSegs(std.MarineSegments("Oceanside, CA", marineFixture(now), false, now))
	if imperial == metric {
		t.Fatal("the two unit systems must read differently")
	}
	if !strings.Contains(imperial, "feet") {
		t.Errorf("imperial reads feet:\n%s", imperial)
	}
	if !strings.Contains(metric, " m") {
		t.Errorf("metric reads metres:\n%s", metric)
	}
	for _, s := range []string{imperial, metric} {
		if !strings.Contains(s, "knots") {
			t.Errorf("currents are always knots:\n%s", s)
		}
	}
}

// No tides and no currents is a STATED absence, not silence: the listener needs
// to know the station is out of range rather than wonder if the app broke.
func TestMarineStatesTheAbsenceOfTidesAndCurrents(t *testing.T) {
	now := time.Now()
	mr := MarineReport{Known: true, State: snapshot.Marine{Buoy: "1", ObservedAt: now, WaveHeight: fx(0.4)}}
	got := joinSegs(std.MarineSegments("Somewhere, CA", mr, true, now))
	if !strings.Contains(got, "no tide or current predictions") {
		t.Errorf("want the absence line:\n%s", got)
	}
}

// An inland location composes NOTHING — not an absence line. The sea is not
// missing for Denver, it is not applicable.
func TestAnInlandLocationComposesNoMaritimeReport(t *testing.T) {
	if segs := std.MarineSegments("Denver, CO", MarineReport{}, true, time.Now()); segs != nil {
		t.Fatalf("an inland location composes nothing, got %d segments", len(segs))
	}
}

// The forecast is prose from the network: it must be BOUNDED, or one unusual
// product holds the air and the fire report is never reached.
func TestTheMaritimeSectionIsBounded(t *testing.T) {
	now := time.Now()
	mr := marineFixture(now)
	mr.Forecast = strings.Repeat("The seas will be lively today. ", 400)
	segs := std.MarineSegments("Oceanside, CA", mr, true, now)
	if len(segs) > maxMaritimePieces {
		t.Fatalf("the section composed %d segments, the bound is %d", len(segs), maxMaritimePieces)
	}
}

// A hostile station name reaches neither a synthesiser's stdin nor a frame.
func TestAHostileStationNameIsMadePlainAndBounded(t *testing.T) {
	now := time.Now()
	mr := marineFixture(now)
	mr.State.TideStation = "Evil\x1b[31m\x07Station " + strings.Repeat("X", 300) + " (9410230)"
	got := joinSegs(std.MarineSegments("Oceanside, CA", mr, true, now))
	if strings.ContainsAny(got, "\x1b\x07") {
		t.Error("a control character reached the spoken text")
	}
	if strings.Contains(got, strings.Repeat("X", 60)) {
		t.Error("the station name is not bounded")
	}
}

// --- Task 3.5: the coastal-waters forecast, (B) only ---

const cwfFixture = `
FZUS56 KSGX 301030
CWFSGX

Coastal Waters Forecast for Southern California
National Weather Service San Diego CA
330 AM PDT Sun Aug 30 2026

.SYNOPSIS...High pressure builds over the waters through midweek.

$$

PZZ750-301800-
Coastal waters from San Mateo Point to the Mexican border-
330 AM PDT Sun Aug 30 2026

.TODAY...W winds 5 to 10 kt. Wind waves 2 ft. SW swell 2 ft.
.TONIGHT...W winds 5 kt. Wind waves 1 ft. SW swell 2 ft.
.MON...SW winds 5 to 10 kt. Wind waves 1 ft.
.MON NIGHT...Variable winds less than 5 kt.
.TUE...W winds 5 to 10 kt.

$$

PZZ775-301800-
Waters from San Mateo Point to the Mexican border out 60 nm-
330 AM PDT Sun Aug 30 2026

.TODAY...NW winds 10 to 15 kt. Wind waves 3 ft.

$$
`

func TestCoastalForecastTakesTheFirstNearshoreBlockAndThreePeriods(t *testing.T) {
	got := CoastalForecast([]Product{{Type: "CWF", Text: cwfFixture}}, "")
	if got == "" {
		t.Fatal("the CWF must be found")
	}
	if !strings.Contains(got, "SYNOPSIS") {
		t.Errorf("the office synopsis is read:\n%s", got)
	}
	// The FIRST nearshore block, not the offshore one 60 nm out.
	if !strings.Contains(got, "San Mateo Point to the Mexican border-") || strings.Contains(got, "out 60 nm") {
		t.Errorf("the first nearshore block only:\n%s", got)
	}
	// Three periods (RAT-6, MVS-D-37), so the fourth is not read.
	for _, want := range []string{".TODAY...", ".TONIGHT...", ".MON..."} {
		if !strings.Contains(got, want) {
			t.Errorf("missing period %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, ".MON NIGHT...") || strings.Contains(got, ".TUE...") {
		t.Errorf("only %d periods are read:\n%s", SpokenPeriodsCap, got)
	}
}

func TestCoastalForecastIsEmptyWithoutACWF(t *testing.T) {
	if got := CoastalForecast([]Product{{Type: "ZFP", Text: "not a marine product"}}, ""); got != "" {
		t.Errorf("no CWF means no forecast, got %q", got)
	}
	if got := CoastalForecast(nil, ""); got != "" {
		t.Errorf("no products means no forecast, got %q", got)
	}
	// The report still composes: the buoy and the tides are read without it.
	now := time.Now()
	if segs := std.MarineSegments("Oceanside, CA", marineFixture(now), true, now); len(segs) == 0 {
		t.Error("a missing forecast must not silence the whole report")
	}
}
