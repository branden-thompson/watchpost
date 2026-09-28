package tty

// map_badges_test.go — 0.18.0 D-131 to D-133 (UAT-2 U2-36): with every layer
// on, the notes under the map were six lines. One row of badges credits the
// layers drawn; the full credits are the Status window's.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// badgeMap is the map in Radar mode with alerts, fire, temperature and
// waves on, buoys off; the temperature's answer names its sources.
func badgeMap(t *testing.T, cost MapCost) Dashboard {
	t.Helper()
	var asked []string
	var asks []MapAsk
	cfg := Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, "MRMS", &asked), MapTemperature: tempAnswer(&asks),
		MapCost: func(MapAsk, func(string) bool) MapCost { return cost },
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true, Chips: []string{"NWS"}}, {Key: RadarLayer, Label: "Radar", On: true},
			{Key: FireLayer, Label: "Fire", On: true, Chips: []string{"NIFC", "HMS"}}, {Key: TemperatureLayer, Label: "Temperature", On: true},
			{Key: WaveLayer, Label: "Waves", On: true}, {Key: BuoyLayer, Label: "Buoys", Chips: []string{"NDBC"}}}}
	d := mapDash(t, cfg)
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	return settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
}

// underTheMap is the rows under the map, without colour.
func underTheMap(d Dashboard) []string {
	var out []string
	for _, l := range d.scrubRows(d.mapTextW()) {
		out = append(out, stripANSITest(l))
	}
	return out
}

// TestTheBadgeRowCreditsEachLayerDrawn is D-131 and D-133: one row, a badge
// a layer on and drawn in the Overlays menu's order, each naming its sources
// as drawn - never a layer off, never the radar (its chip is where it was) -
// and no credit sentence under the map.
func TestTheBadgeRowCreditsEachLayerDrawn(t *testing.T) {
	d := badgeMap(t, MapCost{})
	rows := underTheMap(d)
	want := "ALERTS [NWS]    FIRE [NIFC]/[HMS]    TEMP [O-METEO]    WAVES [NDFD]/[O-METEO]"
	found := false
	for _, r := range rows {
		if strings.TrimSpace(r) == want {
			found = true
		}
	}
	all := strings.Join(rows, "\n")
	if !found {
		t.Errorf("no badge row %q under the map:\n%s", want, all)
	}
	for _, gone := range []string{"BUOYS", "CC BY", "Each radar frame", "RADAR [", "approximate"} {
		if strings.Contains(all, gone) {
			t.Errorf("%q is under the map:\n%s", gone, all)
		}
	}
}

// TestTheBadgesWrapOnlyWhenNarrow is D-133: a second row only when the
// window cannot hold them on one, and never a badge cut in two.
func TestTheBadgesWrapOnlyWhenNarrow(t *testing.T) {
	d := badgeMap(t, MapCost{})
	one := d.badgeRows(200)
	two := d.badgeRows(40)
	if len(one) != 1 || len(two) < 2 {
		t.Fatalf("200 columns: %d rows; 40 columns: %d rows; want one, then more", len(one), len(two))
	}
	for _, r := range two {
		r = stripANSITest(r)
		if strings.Count(r, "[")-strings.Count(r, "]") != 0 || len(r) > 40 {
			t.Errorf("a wrapped row %q cuts a badge or passes the width", r)
		}
	}
}

// TestTheCostWarningIsOneLineAndTheEstimateSitsUnderTheScrubber is D-133:
// the warning one line in the HUM LEAD's words, the estimate under the
// timeline's right end, as drawn - and neither under the map at or under the
// thresholds.
func TestTheCostWarningIsOneLineAndTheEstimateSitsUnderTheScrubber(t *testing.T) {
	d := badgeMap(t, MapCost{Bytes: 2_800_000, Requests: 174})
	rows := underTheMap(d)
	warn, est := -1, -1
	for i, r := range rows {
		if strings.Contains(r, "Map may experience performance issues at this zoom level. Adjust layers/zoom to improve experience.") {
			warn = i
		}
		if strings.Contains(r, "Est. 2.8MB / 174 Requests") {
			est = i
		}
	}
	scrub := d.scrubW()
	if warn < 0 || est < 0 || est <= warn+radarRows {
		t.Fatalf("warning at row %d, estimate at %d:\n%s", warn, est, strings.Join(rows, "\n"))
	}
	if end := strings.Index(rows[est], "Requests") + len("Requests"); end != scrub+1 {
		t.Errorf("the estimate ends at column %d; want under the timeline's right end, %d", end, scrub+1)
	}
	if strings.Contains(strings.Join(rows, "\n"), "Switch off layers") {
		t.Error("the old advice is still said")
	}
	quiet := underTheMap(badgeMap(t, MapCost{Bytes: 1, Requests: 1}))
	if s := strings.Join(quiet, "\n"); strings.Contains(s, "performance") || strings.Contains(s, "Est.") {
		t.Errorf("under the thresholds the cost is said:\n%s", s)
	}
}

// TestTheMRMSChipIsMarkedApproximate is D-132: MRMS's colours are read from
// its legend, so its chip reads MRMS≈ where it is shown.
func TestTheMRMSChipIsMarkedApproximate(t *testing.T) {
	d := badgeMap(t, MapCost{})
	if w := stripANSITest(d.mapBadgeWords()); !strings.Contains(w, "MRMS≈") {
		t.Errorf("the badge reads %q; want MRMS≈", w)
	}
	if r := stripANSITest(d.loopRow(d.scrubW())); !strings.Contains(r, "MRMS≈") {
		t.Errorf("the loop's row reads %q; want MRMS≈", r)
	}
}

// TestTheStatusWindowCarriesTheFullCredits is D-131 and D-132: a source's
// notes - its full credit, what never changes about it - are said under it
// in the MAP block.
func TestTheStatusWindowCarriesTheFullCredits(t *testing.T) {
	d := mapDash(t, Config{MapSources: []MapSource{{Name: "Open-Meteo", Host: "api.open-meteo.com", Use: "temperature",
		Notes: []string{"Temperature: Open-Meteo.com (CC BY 4.0), interpolated."}}}})
	if s := stripANSITest(strings.Join(d.mapSourceLines(), "\n")); !strings.Contains(s, "Temperature: Open-Meteo.com (CC BY 4.0), interpolated.") {
		t.Errorf("the MAP block:\n%s", s)
	}
}
