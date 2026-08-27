package tty

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestDetailMaritimeSectionForCoastalOnly(t *testing.T) {
	// B3 UAT 29: MARITIME rows render only when marine data exists — swell
	// heading + height in display units, sea state, buoy water temp.
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Marine = &snapshot.Marine{
		SwellHeight: f64(0.6096), SwellDirDeg: f64(280), WaveHeight: f64(0.9), WavePeriod: f64(14),
		WaterTemp: f64(23.3), Buoy: "46224", BuoyDistanceKM: f64(9),
	}
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.showDetails = true
	joined := strings.Join(d.detailLines(), "\n")
	// Grid: labels at 0, values at col 14, secondary values at col 37 (the
	// FORECAST HIGH/LOW column) — UAT 32.2.
	for _, want := range []string{
		"MARITIME │ Conditions    Slight Chop",
		"Water Temp    74ºF             (buoy 46224, 6 mi)", // 9 km in display units (UAT 60.2)
		"Swell         W        2.0 ft  (period 14 s)",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("maritime section missing %q:\n%s", want, joined)
		}
	}
	inland := m.(Dashboard)
	inland.showDetails = true
	if strings.Contains(strings.Join(inland.detailLines(), "\n"), "MARITIME") {
		t.Fatal("inland location must not render a MARITIME section")
	}
}

func TestDetailTideAndCurrentRows(t *testing.T) {
	// B3 UAT 61-64: tides and currents on the MARITIME grid (value col 14 /
	// sub-column 8 / fixed 4-cell number / note at col 37 with HIGH) — trend from the next
	// predicted event, one row per next high and low, local hh:mm; a
	// negative low never shifts "ft"; currents in knots.
	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2026, 8, 24, 22, 0, 0, 0, time.UTC) // 15:00 PDT
	at := func(h, m int) time.Time { return time.Date(2026, 8, 24, h, m, 0, 0, time.UTC) }
	mar := &snapshot.Marine{
		TideLevel: f64(1.131), TideStation: "La Jolla (Scripps Institution Wharf)", TideStationKM: f64(38.8),
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
	rows := strings.Join(maritimeRows(o, mar, tz, now), "\n")
	for _, want := range []string{
		"Tide          Rising   3.7 ft  (La Jolla, 24 mi)",
		"Next High     19:40    5.7 ft",
		"Next Low      02:49   -0.1 ft",
		"Currents      Flood    1.4 kt  (Slack 16:05)",
	} {
		if !strings.Contains(rows, want) {
			t.Fatalf("maritime rows missing %q:\n%s", want, rows)
		}
	}
	// Falling when the next event is a low; slack phase names the next flow.
	falling := &snapshot.Marine{Tides: []snapshot.TideEvent{{Time: at(23, 30), Height: 0.5, Type: "L"}},
		Currents: []snapshot.CurrentEvent{{Time: at(21, 0), Speed: 0, Type: "slack"}, {Time: at(23, 30), Speed: 0.6, Type: "ebb"}}}
	rows = strings.Join(maritimeRows(o, falling, tz, now), "\n")
	if !strings.Contains(rows, "Tide          Falling") || !strings.Contains(rows, "Next Low      16:30    1.6 ft") || strings.Contains(rows, "Next High") || !strings.Contains(rows, "Currents      Slack            (Ebb 16:30)") {
		t.Fatalf("falling / slack rendering:\n%s", rows)
	}
	if got := strings.Join(maritimeRows(render.Opts{Units: render.UnitC}, mar, tz, now), "\n"); !strings.Contains(got, "Rising  1.13 m") || !strings.Contains(got, "1.4 kt") || !strings.Contains(got, "39 km") {
		t.Fatalf("metric tide rows:\n%s", got)
	}
	// No tide data: nothing added (coastal buoy-only sections unchanged).
	if got := maritimeRows(o, &snapshot.Marine{WaterTemp: f64(20)}, tz, now); len(got) != 1 {
		t.Fatalf("buoy-only section must not grow tide rows: %v", got)
	}
}

func TestTideHeightsShareOneColumn(t *testing.T) {
	// UAT 62/63: the height starts at the same cell on the Tide and Next
	// rows, for Rising and Falling, positive and negative values; and no
	// MARITIME row can exceed the details modal's 78-cell wrap budget.
	tz := time.UTC
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, tz)
	for _, tc := range []struct {
		level, next float64
		typ         string
	}{{1.13, 1.73, "H"}, {0.3, -0.03, "L"}} {
		m := &snapshot.Marine{TideLevel: f64(tc.level), Tides: []snapshot.TideEvent{{Time: now.Add(time.Hour), Height: tc.next, Type: tc.typ}}}
		rows := maritimeRows(render.Opts{Units: render.UnitF}, m, tz, now)
		tide, next := stripANSITest(rows[0]), stripANSITest(rows[1])
		ti, ni := strings.Index(tide, " ft"), strings.Index(next, " ft")
		if ti < 0 || ti != ni {
			t.Fatalf("heights must align (%d vs %d):\n%s\n%s", ti, ni, tide, next)
		}
	}
	wide := &snapshot.Marine{WaterTemp: f64(20), Buoy: "46224", BuoyDistanceKM: f64(9), ObservedAt: now.Add(-40 * time.Minute),
		WaveHeight: f64(0.9), WavePeriod: f64(14), SwellDirDeg: f64(200), TideLevel: f64(1), TideStation: "La Jolla (Scripps Institution Wharf) with a very long name", TideStationKM: f64(38.8),
		Tides:    []snapshot.TideEvent{{Time: now.Add(time.Hour), Height: 1.7, Type: "H"}, {Time: now.Add(7 * time.Hour), Height: -0.1, Type: "L"}},
		Currents: []snapshot.CurrentEvent{{Time: now.Add(-time.Hour), Speed: 0.7, Type: "flood"}, {Time: now.Add(time.Hour), Speed: 0, Type: "slack"}}}
	for _, r := range maritimeRows(render.Opts{Units: render.UnitF}, wide, tz, now) {
		if w := len([]rune(stripANSITest(r))); w > 78 {
			t.Fatalf("row exceeds the modal wrap budget (%d > 78): %q", w, r)
		}
	}
	// UAT 67: notes share one column, marNoteGap cells past the widest value.
	laid := maritimeRows(render.Opts{Units: render.UnitF}, wide, tz, now)
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
	if got := stripANSITest(strings.Join(maritimeRows(render.Opts{Units: render.UnitF}, wide, tz, now), "\n")); !strings.Contains(strings.Split(got, "\n")[0], "MARITIME │ Observed      40m") {
		t.Fatalf("mock order starts with Observed:\n%s", got)
	}
}
