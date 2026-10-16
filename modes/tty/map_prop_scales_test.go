package tty

// map_prop_scales_test.go — 0.19.0 W5.5: NOAA's space-weather scales named
// (FR-1.18, D-88, D-109).

import (
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// scaled is chartDash with NOAA's scales as given, read at the snapshot's
// time.
func scaled(t *testing.T, sc ionomaps.Scales) Dashboard {
	t.Helper()
	d := chartDash(t)
	snap := *d.mapPane.propSnap
	snap.Scales = sc
	d.mapPane.propSnap = &snap
	return d
}

// chartText is the chart window's text, plain.
func chartText(d Dashboard) string {
	return stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n"))
}

// TestARaisedScaleIsNamed is FR-1.18 (D-88): a scale above 0 is named, by
// its level and NOAA's word, in the chart and in the words, with NOAA's
// outlook of a storm to come; scales all at 0 name nothing; a level NOAA
// left unset is never named as 0.
func TestARaisedScaleIsNamed(t *testing.T) {
	d := scaled(t, ionomaps.Scales{R: 2, S: 0, G: ionomaps.ScaleUnset, Valid: drawnAt,
		Outlook: []ionomaps.DayScale{{Day: drawnAt, G: 0}, {Day: drawnAt.AddDate(0, 0, 1), G: 2}, {Day: drawnAt.AddDate(0, 0, 2), G: ionomaps.ScaleUnset}}})
	chart := chartText(d)
	if !strings.Contains(chart, "R2 moderate radio blackouts") || !strings.Contains(chart, "outlook G2 Mon 12 Oct") {
		t.Errorf("R2 and a G2 outlook are not named in the chart:\n%s", chart)
	}
	for _, never := range []string{"S0", "G0", "G-1", "G1 "} {
		if strings.Contains(chart, never) {
			t.Errorf("the chart names %q:\n%s", never, chart)
		}
	}
	words := propWordsFrom(t, d)
	status := words[lineWith(words, "Status: ")]
	if !strings.Contains(status, "space weather: R2, moderate radio blackouts; NOAA expects G2, a moderate geomagnetic storm, Monday 12 October") {
		t.Errorf("the status block does not name the scales: %q", status)
	}
	calm := scaled(t, ionomaps.Scales{Valid: drawnAt, Outlook: []ionomaps.DayScale{{Day: drawnAt}}})
	if strings.Contains(chartText(calm), "SPACE WEATHER") || strings.Contains(strings.Join(propWordsFrom(t, calm), " "), "space weather") {
		t.Errorf("scales all at 0 are named:\n%s", chartText(calm))
	}
}

// TestAStormSaysTypicalMayNotHold is FR-1.18 and D-109: while G is 1 or
// more, every hour ahead says a storm is in progress and typical
// conditions may not hold; with G at 0 or unset, it does not.
func TestAStormSaysTypicalMayNotHold(t *testing.T) {
	for _, c := range []struct {
		g     int
		storm bool
	}{{1, true}, {4, true}, {0, false}, {ionomaps.ScaleUnset, false}} {
		d := scaled(t, ionomaps.Scales{G: c.g, Valid: drawnAt})
		d.mapPane.propAhead = 3
		lat, lon := d.mapPane.propOrigin.at.Lat, d.mapPane.propOrigin.at.Lon
		label, words := d.propAheadLabel(lat, lon), d.propAheadWords(lat, lon)
		if got := strings.Contains(label, "STORM: TYPICAL MAY NOT HOLD"); got != c.storm {
			t.Errorf("G%d: the hour ahead's label %q", c.g, label)
		}
		if got := strings.Contains(words, "a geomagnetic storm is in progress: typical conditions may not hold"); got != c.storm {
			t.Errorf("G%d: the hour ahead's words %q", c.g, words)
		}
	}
}

// TestMissingScalesAreNotShownAsZero is FR-1.18 (D-88): with no scales
// read, the chart and the words say so; nothing reads as level 0.
func TestMissingScalesAreNotShownAsZero(t *testing.T) {
	d := scaled(t, ionomaps.Scales{R: ionomaps.ScaleUnset, S: ionomaps.ScaleUnset, G: ionomaps.ScaleUnset})
	if chart := chartText(d); !strings.Contains(chart, "SPACE WEATHER: NOAA's scales not read") {
		t.Errorf("the chart does not say the scales are missing:\n%s", chart)
	}
	words := propWordsFrom(t, d)
	if status := words[lineWith(words, "Status: ")]; !strings.Contains(status, "space weather: NOAA's scales not read") {
		t.Errorf("the status block does not say the scales are missing: %q", status)
	}
	stale := scaled(t, ionomaps.Scales{G: 3, Valid: drawnAt.Add(-25 * time.Hour)})
	if chart := chartText(stale); !strings.Contains(chart, "G3 strong geomagnetic storm") {
		t.Errorf("a raised level read is named, whatever its age the library allows:\n%s", chart)
	}
}
