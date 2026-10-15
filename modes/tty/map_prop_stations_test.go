package tty

import (
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// stationsSnap is a snapshot at noon on 9 October, GloTEC the background,
// fed by n stations, the oldest reading at 11:55, said early for the reason.
func stationsSnap(n int, early ionomaps.Reason) ionomaps.Snapshot {
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	s := ionomaps.Snapshot{Computed: at, Hours: []ionomaps.Hour{{At: at}}, Early: early,
		Background: ionomaps.Backgrounds{FoF2: ionomaps.GloTEC, M3000: ionomaps.Climatology},
		Inputs:     ionomaps.Inputs{GloTECValid: at.Add(-30 * time.Minute), Stations: n}}
	if n > 0 {
		s.Inputs.OldestReading = at.Add(-5 * time.Minute)
	}
	return s
}

// TestStationsNextAsksAgainAtOnce is go-ionomaps D-63 (watchpost D-183):
// NOAA's snapshot said StationsNext is shown at once, said as asking GIRO's
// stations, and the next update is asked straight away, not at the refresh
// tick; any other snapshot waits for the tick.
func TestStationsNextAsksAgainAtOnce(t *testing.T) {
	s := &seam{answer: PropagationResult{Snapshot: stationsSnap(0, ionomaps.StationsNext)}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	msgs := propMsgs(cmd)
	if len(msgs) != 1 {
		t.Fatalf("P carried %d results; want one", len(msgs))
	}
	m, next := d.Update(msgs[0])
	d = m.(Dashboard)
	if words := stripANSITest(d.mapStatusLine()); !strings.Contains(words, "asking GIRO's stations") {
		t.Errorf("NOAA's snapshot, stations next, says %q", words)
	}
	if again := propMsgs(next); len(again) != 1 || s.count() != 2 {
		t.Fatalf("after StationsNext the update was asked %d times in all, its next result %d; want it asked again at once", s.count(), len(again))
	}
	s.answer = PropagationResult{Snapshot: stationsSnap(40, ionomaps.Fresh)}
	m, next = d.Update(propUpdatedMsg{gen: d.mapPane.propGen, res: s.answer})
	if propMsgs(next) != nil {
		t.Error("a fresh snapshot asked again at once; it waits for the refresh tick")
	}
	d = m.(Dashboard)
	words := stripANSITest(d.mapStatusLine())
	if !strings.Contains(words, "40 stations, oldest 11:55 UTC") {
		t.Errorf("a snapshot fed by 40 stations says %q", words)
	}
}

// TestTheStationsAreCreditedWhenTheyFedIt is FR-4.1 (D-168): GIRO's badge is
// lit for a snapshot its stations fed, and absent from the words when none
// did; with none, the words say the background alone.
func TestTheStationsAreCreditedWhenTheyFedIt(t *testing.T) {
	d := propDash(t, &seam{}, true)
	snap := stationsSnap(12, ionomaps.Fresh)
	d.mapPane.propSnap = &snap
	if got := strings.Join(d.propBadges(false), " "); !strings.Contains(got, "GIRO") {
		t.Errorf("12 stations fed the snapshot and the badges are %q", got)
	}
	none := stationsSnap(0, ionomaps.Fresh)
	d.mapPane.propSnap = &none
	if got := strings.Join(d.propBadges(false), " "); strings.Contains(got, "GIRO") {
		t.Errorf("no stations fed the snapshot and the badges are %q", got)
	}
	if w := snapshotWords(none, 0); !strings.Contains(w, "no stations: the background alone") {
		t.Errorf("a snapshot no station fed says %q", w)
	}
}

// TestADisturbedBandIsDrawnAndNamed is R-2.5 (watchpost D-73): a band under
// D-RAP's 1 dB frequency has its own glyph in the chart, the legend names
// it, and the row's words keep its upper limit and add D-RAP's.
func TestADisturbedBandIsDrawnAndNamed(t *testing.T) {
	d := pressMap(t, chartDash(t), "B")
	c := &d.mapPane.propChart
	bands := append([]ionomaps.BandStatus(nil), c.local.Bands...)
	bands[0] = ionomaps.BandStatus{Band: bands[0].Band, Status: ionomaps.Disturbed, Limit: ionomaps.LimitDRAP, LimitMHz: 2.5}
	c.local.Bands = bands
	text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n"))
	var local string
	for _, l := range strings.Split(text, "\n") {
		if strings.Contains(l, "Local, ~") && local == "" { // the row, not the typical day's heading
			local = l
		}
	}
	if !strings.Contains(local, string(propDisturbedGlyph)) || !strings.HasSuffix(local, "D-RAP 2.5 MHz") || !strings.Contains(local, "foF2 6.0 MHz · ") {
		t.Errorf("the local row, 160 m disturbed, reads %q", local)
	}
	if !strings.Contains(text, string(propDisturbedGlyph)+" disturbed (D-RAP)") {
		t.Errorf("the legend does not name the disturbed glyph:\n%s", text)
	}
	if w := propRowLimitWords(bands[:1]); w != "D-RAP 2.5 MHz" {
		t.Errorf("a row whose only band is disturbed says %q", w)
	}
}

// TestAnUnscoredNearestStationIsSaid is D-181 (go-ionomaps R-2.11): the
// chart names the rows whose nearest station is unscored, to verify on air;
// the centre's answer says its nearest station's distance, or that none fed
// it.
func TestAnUnscoredNearestStationIsSaid(t *testing.T) {
	d := pressMap(t, chartDash(t), "B")
	c := &d.mapPane.propChart
	if text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n")); strings.Contains(text, "unscored") {
		t.Errorf("no answer's station is unscored, yet the chart says so:\n%s", text)
	}
	c.targets[2].Nearest = ionomaps.Nearest{Km: 300, Unscored: true, Known: true}
	text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n"))
	if !strings.Contains(text, "Nearest station unscored, verify on air: "+propContinents[2].name) {
		t.Errorf("%s's nearest station is unscored and the chart does not say so:\n%s", propContinents[2].name, text)
	}
	if w := d.propNearestWords(ionomaps.Nearest{}); w != "No station fed this answer" {
		t.Errorf("no station: %q", w)
	}
	if w := d.propNearestWords(ionomaps.Nearest{Km: 300, Unscored: true, Known: true}); !strings.HasPrefix(w, "Nearest station ") || !strings.HasSuffix(w, ", unscored: verify on air") {
		t.Errorf("an unscored station 300 km away: %q", w)
	}
	box := workCentre(t, stationDash(t))
	if lines := stripANSITest(strings.Join(box.propCentreBox(), "\n")); !strings.Contains(lines, "No station fed this answer") {
		t.Errorf("the centre box, no station fed, does not say so:\n%s", lines)
	}
}
