package tty

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestDetailMaritimeSectionForCoastalOnly(t *testing.T) {
	// B3 UAT 29: MARINE rows render only when marine data exists — swell
	// heading + height in display units, sea state, buoy water temp.
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Marine = &snapshot.Marine{
		SwellHeight: new(0.6096), SwellDirDeg: new(float64(280)), WaveHeight: new(0.9), WavePeriod: new(float64(14)),
		WaterTemp: new(23.3), Buoy: "46224", BuoyDistanceKM: new(float64(9)),
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.modal = modalDetails
	joined := strings.Join(d.detailLines(), "\n")
	// Grid: labels at 0, values at col 14, secondary values at col 37 (the
	// FORECAST HIGH/LOW column) — UAT 32.2.
	for _, want := range []string{
		"MARINE │ Conditions    Slight Chop",
		"Water Temp    74°F             (buoy 46224, 6 mi)", // 9 km in display units (UAT 60.2)
		"Swell         W        2.0 ft  (period 14 s)",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("maritime section missing %q:\n%s", want, joined)
		}
	}
	inland := m.(Dashboard)
	inland.modal = modalDetails
	if strings.Contains(strings.Join(inland.detailLines(), "\n"), "MARINE") {
		t.Fatal("inland location must not render a MARINE section")
	}
}

func TestDetailTideAndCurrentRows(t *testing.T) {
	// B3 UAT 61-64: tides and currents on the MARINE grid (value col 14 /
	// sub-column 8 / fixed 4-cell number / note at col 37 with HIGH) — trend from the next
	// predicted event, one row per next high and low, local hh:mm; a
	// negative low never shifts "ft"; currents in knots.
	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2026, 8, 24, 22, 0, 0, 0, time.UTC) // 15:00 PDT
	at := func(h, m int) time.Time { return time.Date(2026, 8, 24, h, m, 0, 0, time.UTC) }
	mar := &snapshot.Marine{
		TideLevel: new(1.131), TideStation: "La Jolla (Scripps Institution Wharf)", TideStationKM: new(38.8),
		Tides: []snapshot.TideEvent{
			{Time: at(16, 1), Height: 1.193, Type: "H"}, {Time: at(20, 35), Height: 0.799, Type: "L"},
			{Time: at(26, 40), Height: 1.731, Type: "H"}, {Time: at(33, 49), Height: -0.028, Type: "L"},
		},
		Currents: []snapshot.CurrentEvent{
			{Time: at(20, 9), Speed: 0.712, Type: "flood"}, {Time: at(23, 5), Speed: 0, Type: "slack"}, {Time: at(26, 30), Speed: 0.9, Type: "ebb"},
		},
		CurrentStation: "San Diego Bay Entrance",
	}
	o := render.Opts{Width: 120, Units: render.UnitF}
	rows := strings.Join(maritimeRows(o, mar, tz, now, 65), "\n")
	for _, want := range []string{
		"Tide          Rising   3.7 ft  (La Jolla, 24 mi)",
		"Next High     7:40 PM  5.7 ft",
		"Next Low      2:49 AM -0.1 ft",
		"Currents      Flood    1.4 kt  (Slack 4:05 PM)",
	} {
		if !strings.Contains(rows, want) {
			t.Fatalf("maritime rows missing %q:\n%s", want, rows)
		}
	}
	// Falling when the next event is a low; slack phase names the next flow.
	falling := &snapshot.Marine{Tides: []snapshot.TideEvent{{Time: at(23, 30), Height: 0.5, Type: "L"}},
		Currents: []snapshot.CurrentEvent{{Time: at(21, 0), Speed: 0, Type: "slack"}, {Time: at(23, 30), Speed: 0.6, Type: "ebb"}}}
	rows = strings.Join(maritimeRows(o, falling, tz, now, 65), "\n")
	if !strings.Contains(rows, "Tide          Falling") || !strings.Contains(rows, "Next Low      4:30 PM  1.6 ft") || strings.Contains(rows, "Next High") || !strings.Contains(rows, "Currents      Slack            (Ebb 4:30 PM)") {
		t.Fatalf("falling / slack rendering:\n%s", rows)
	}
	if got := strings.Join(maritimeRows(render.Opts{Units: render.UnitC}, mar, tz, now, 65), "\n"); !strings.Contains(got, "Rising  1.13 m") || !strings.Contains(got, "1.4 kt") || !strings.Contains(got, "39 km") {
		t.Fatalf("metric tide rows:\n%s", got)
	}
	// No tide data: nothing added (coastal buoy-only sections unchanged).
	if got := maritimeRows(o, &snapshot.Marine{WaterTemp: new(float64(20))}, tz, now, 65); len(got) != 1 {
		t.Fatalf("buoy-only section must not grow tide rows: %v", got)
	}
}

func TestTideHeightsShareOneColumn(t *testing.T) {
	// UAT 62/63: the height starts at the same cell on the Tide and Next
	// rows, for Rising and Falling, positive and negative values; and no
	// MARINE row can exceed the details modal's 78-cell wrap budget.
	tz := time.UTC
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, tz)
	for _, tc := range []struct {
		level, next float64
		typ         string
	}{{1.13, 1.73, "H"}, {0.3, -0.03, "L"}} {
		m := &snapshot.Marine{TideLevel: new(tc.level), Tides: []snapshot.TideEvent{{Time: now.Add(time.Hour), Height: tc.next, Type: tc.typ}}}
		rows := maritimeRows(render.Opts{Units: render.UnitF}, m, tz, now, 65)
		tide, next := stripANSITest(rows[0]), stripANSITest(rows[1])
		ti, ni := strings.Index(tide, " ft"), strings.Index(next, " ft")
		if ti < 0 || ti != ni {
			t.Fatalf("heights must align (%d vs %d):\n%s\n%s", ti, ni, tide, next)
		}
	}
	wide := &snapshot.Marine{WaterTemp: new(float64(20)), Buoy: "46224", BuoyDistanceKM: new(float64(9)), ObservedAt: now.Add(-40 * time.Minute),
		WaveHeight: new(0.9), WavePeriod: new(float64(14)), SwellDirDeg: new(float64(200)), TideLevel: new(float64(1)), TideStation: "La Jolla (Scripps Institution Wharf) with a very long name", TideStationKM: new(38.8),
		Tides:    []snapshot.TideEvent{{Time: now.Add(time.Hour), Height: 1.7, Type: "H"}, {Time: now.Add(7 * time.Hour), Height: -0.1, Type: "L"}},
		Currents: []snapshot.CurrentEvent{{Time: now.Add(-time.Hour), Speed: 0.7, Type: "flood"}, {Time: now.Add(time.Hour), Speed: 0, Type: "slack"}}}
	for _, r := range maritimeRows(render.Opts{Units: render.UnitF}, wide, tz, now, 65) {
		if w := len([]rune(stripANSITest(r))); w > 78 {
			t.Fatalf("row exceeds the modal wrap budget (%d > 78): %q", w, r)
		}
	}
	// UAT 67: notes share one column, marNoteGap cells past the widest value.
	laid := maritimeRows(render.Opts{Units: render.UnitF}, wide, tz, now, 65)
	widest, noteCol := 0, -1
	for _, r := range laid {
		p := stripANSITest(r)[detailPrefixW:]
		if v := strings.TrimRight(strings.SplitN(p, "(", 2)[0], " "); len([]rune(v)) > widest {
			widest = len([]rune(v))
		}
	}
	for _, r := range laid {
		p := stripANSITest(r)[detailPrefixW:]
		if i := strings.Index(p, "("); i >= 0 {
			col := len([]rune(p[:i]))
			if noteCol == -1 {
				noteCol = col
			}
			if col != noteCol || col != widest+marNoteGap {
				t.Fatalf("notes must share one column %d past the widest value (%d): got %d/%d in %q", marNoteGap, widest, col, noteCol, p)
			}
		}
	}
	if got := stripANSITest(strings.Join(maritimeRows(render.Opts{Units: render.UnitF}, wide, tz, now, 65), "\n")); !strings.Contains(strings.Split(got, "\n")[0], "MARINE │ Observed      40m") {
		t.Fatalf("mock order starts with Observed:\n%s", got)
	}
}

// TestTheMarineSectionDrawsEverySwellAndTheBuoysWind is 0.19.5 Q2.8 (RN-13):
// the second swell, the wind waves and the buoy's wind with its gusts each
// have a row, in the listener's units.
func TestTheMarineSectionDrawsEverySwellAndTheBuoysWind(t *testing.T) {
	m := &snapshot.Marine{
		SwellHeight: new(1.2), SwellDirDeg: new(270.0),
		SecondarySwellHeight: new(0.6), SecondarySwellDirDeg: new(200.0),
		WindWaveHeight: new(0.3), WindSpeed: new(5.0), WindGust: new(9.0),
	}
	for _, c := range []struct {
		units render.Units
		wind  string
		gust  string
	}{
		{render.UnitF, "11 mph", "(gusts 20 mph)"},
		{render.UnitC, "18 km/h", "(gusts 32 km/h)"},
	} {
		rows := map[string]marineCell{}
		for _, r := range swellRows(render.Opts{Units: c.units}, m) {
			rows[r.label] = r
		}
		for _, label := range []string{"Swell", "Swell 2", "Wind Waves", "Buoy Wind"} {
			if _, ok := rows[label]; !ok {
				t.Errorf("units %v: the marine section has no %q row", c.units, label)
			}
		}
		if w := rows["Buoy Wind"]; w.value != c.wind || w.note != c.gust {
			t.Errorf("units %v: the buoy wind reads %q %q; want %q %q", c.units, w.value, w.note, c.wind, c.gust)
		}
		if !strings.Contains(rows["Swell 2"].value, "SW") {
			t.Errorf("units %v: the second swell's direction is missing: %q", c.units, rows["Swell 2"].value)
		}
	}
}
