package tty

import (
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// stepAhead presses a key that steps the hours and works out the hour it
// armed, as the app's command would.
func stepAhead(t *testing.T, d Dashboard, key string) Dashboard {
	t.Helper()
	m, _ := d.Update(mapKeyMsg(t, key))
	d = m.(Dashboard)
	d.mapPane.propAheadAsked = 0 // the harness dropped the command the Update armed: ask again
	d, cmd := d.armPropAhead()
	if cmd != nil {
		m, _ = d.Update(cmd())
		d = m.(Dashboard)
	}
	return d
}

// TestTheHoursStepOnAKey is FR-1.13 (D-103): shift+right steps an hour ahead,
// to 24 at most, shift+left back, n to now; the field drawn is the hour's,
// the climatology's, not the snapshot's flat field.
func TestTheHoursStepOnAKey(t *testing.T) {
	d := chartDash(t)
	now := d.mapPane.propSnap.Now
	d = stepAhead(t, d, "shift+right")
	if d.mapPane.propAhead != 1 {
		t.Fatalf("shift+right steps to hour %d; want 1", d.mapPane.propAhead)
	}
	shown, ok := d.propShown()
	if !ok || !shown.Now.At.Equal(now.At.Add(time.Hour)) || !shown.Now.Typical {
		t.Fatalf("the hour shown is %v (typical %v, ready %v); want the hour after %v, typical", shown.Now.At, shown.Now.Typical, ok, now.At)
	}
	g := propGrid(t, d)
	if g.Values[0] == float64(now.MUF3000.Values[0]) && g.Values[len(g.Values)/2] == float64(now.MUF3000.Values[len(g.Values)/2]) {
		t.Error("an hour ahead draws the snapshot's own field, not the climatology's")
	}
	for range 30 {
		d = stepAhead(t, d, "shift+right")
	}
	if d.mapPane.propAhead != 24 {
		t.Errorf("stepping on stops at %d; want 24", d.mapPane.propAhead)
	}
	if d = stepAhead(t, d, "shift+left"); d.mapPane.propAhead != 23 {
		t.Errorf("shift+left steps to %d; want 23", d.mapPane.propAhead)
	}
	if d = stepAhead(t, d, "n"); d.mapPane.propAhead != 0 {
		t.Errorf("n steps to %d; want now", d.mapPane.propAhead)
	}
	if g := propGrid(t, d); g.Values[0] != float64(now.MUF3000.Values[0]) {
		t.Error("back at now, the field drawn is not the snapshot's")
	}
}

// TestAnHourAheadSaysTypical is FR-1.13 (D-109): an hour ahead is labelled
// typical for that hour, not a forecast, under the map and in the words;
// now is not.
func TestAnHourAheadSaysTypical(t *testing.T) {
	d := chartDash(t)
	if strings.Contains(stripANSITest(strings.Join(d.propBlock(d.scrubW()), "\n")), "TYPICAL") {
		t.Fatal("the current hour is labelled typical")
	}
	d = stepAhead(t, stepAhead(t, d, "shift+right"), "shift+right")
	block := stripANSITest(strings.Join(d.propBlock(d.scrubW()), "\n"))
	for _, want := range []string{"+2 h", "TYPICAL, NOT A FORECAST", "+2 h ·"} {
		if !strings.Contains(block, want) {
			t.Errorf("two hours ahead the block does not say %q:\n%s", want, block)
		}
	}
	if words := strings.Join(d.describeLinesAll(), " "); !strings.Contains(words, "typical for this hour, not a forecast") {
		t.Errorf("without the picture an hour ahead is not said to be typical: %q", words)
	}
}

// TestTheTypicalErrorStatesItsBasis is FR-1.13 (D-134): an hour ahead says
// its typical error for the place's distance band, in the layer drawn, and
// the words state what it was measured on.
func TestTheTypicalErrorStatesItsBasis(t *testing.T) {
	d := chartDash(t)
	d.mapPane.propSnap.TypicalError.Basis = "the week the plan measured"
	d.mapPane.propSnap.TypicalError.Bands = append(d.mapPane.propSnap.TypicalError.Bands[:0], typicalBand(0, 1.36, 4.58))
	d = stepAhead(t, d, "shift+right")
	if block := stripANSITest(strings.Join(d.propBlock(d.scrubW()), "\n")); !strings.Contains(block, "±4.6 MHz") {
		t.Errorf("the block does not give MUF(3000)'s typical error, 4.6 MHz:\n%s", block)
	}
	if words := strings.Join(d.describeLinesAll(), " "); !strings.Contains(words, "the week the plan measured") {
		t.Errorf("the words do not state the basis: %q", words)
	}
}

// TestAnHourStepNeverWaitsForAnUpdate is FR-1.17: stepping the hours asks
// the network nothing; the hour is worked out from the snapshot held.
func TestAnHourStepNeverWaitsForAnUpdate(t *testing.T) {
	d, s := drawnDash(t)
	asked := s.count()
	m, cmd := d.Update(mapKeyMsg(t, "shift+right"))
	if runProp(t, m.(Dashboard), cmd); s.count() != asked {
		t.Errorf("an hour step asked for an update: %d asks, %d before", s.count(), asked)
	}
}

// TestANewSnapshotMissesTheAnswerMemo is FR-1.13 and D-131: a new snapshot
// drops the hours ahead worked out from the old one, and the chart is asked
// again for it.
func TestANewSnapshotMissesTheAnswerMemo(t *testing.T) {
	d := settled(t, stepAhead(t, chartDash(t), "shift+right")) // the hour stands: its answers are asked (D-227)
	if len(d.mapPane.propAheadSnaps) == 0 {
		t.Fatal("no hour ahead was kept")
	}
	next := *d.mapPane.propSnap
	next.Computed = next.Computed.Add(10 * time.Minute)
	m, _ := d.Update(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: next}})
	d = m.(Dashboard)
	if len(d.mapPane.propAheadSnaps) != 0 {
		t.Error("the hours ahead of the old snapshot were kept for the new one")
	}
	if d.mapPane.propAheadAsked != 1 {
		t.Fatalf("the hour ahead shown was not asked again for the new snapshot (asked %d)", d.mapPane.propAheadAsked)
	}
	d.mapPane.propAheadAsked = 0
	d, cmd := d.armPropAhead()
	m, _ = d.Update(cmd()) // the hour ahead lands; the Update asks for the chart of the new snapshot
	if d = m.(Dashboard); d.mapPane.propChartAsked.computed != next.Computed {
		t.Error("the chart was not asked again for the new snapshot")
	}
}

// typicalBand is a typical error's distance band.
func typicalBand(upToKm, fo, muf float32) ionomaps.BandError {
	return ionomaps.BandError{UpToKm: upToKm, FoF2MHz: fo, MUF3000MHz: muf}
}

// TestAnHourAheadCarriesNoDisturbance is the BUILD red team's L-F2: an hour
// ahead is typical, so today's D-RAP disturbance is not on it - the chart
// says a band disturbed now and none at six, twelve or eighteen hours ahead.
func TestAnHourAheadCarriesNoDisturbance(t *testing.T) {
	d, _ := drawnDash(t)
	f := d.mapPane.propSnap.Now.FoF2
	f.Values = make([]float32, len(f.Values))
	for i := range f.Values {
		f.Values[i] = 25 // D-RAP's 1 dB frequency at 25 MHz everywhere: every open band disturbed
	}
	f.NoData = ionomaps.Bitset{}
	held := *d.mapPane.propSnap
	held.Disturbance = f
	d.mapPane.propSnap = &held
	disturbed := func(c propChart) bool {
		for _, a := range append([]ionomaps.Answer{c.local}, c.targets...) {
			for _, b := range a.Bands {
				if b.Status == ionomaps.Disturbed {
					return true
				}
			}
		}
		return false
	}
	if d = workChart(t, d); !disturbed(d.mapPane.propChart) {
		t.Fatal("control: now's chart says no band disturbed under D-RAP at 25 MHz")
	}
	for _, k := range []int{6, 12, 18} {
		for d.mapPane.propAhead < k {
			d = stepAhead(t, d, "shift+right")
		}
		if d = workChart(t, d); disturbed(d.mapPane.propChart) {
			t.Errorf("+%d h: the chart says a band disturbed; an hour ahead is typical", k)
		}
	}
}

// TestAnHourAheadIsLabelledByTheHourShown is go-ionomaps' A-23 as watchpost
// says it: an hour ahead is the climatology at its UTC hour's start, so a
// snapshot made at 08:40 labels its first hour ahead 09:00, the hour drawn.
func TestAnHourAheadIsLabelledByTheHourShown(t *testing.T) {
	d, _ := drawnDash(t)
	next := drawnAt.Add(40 * time.Minute)
	m, _ := d.Update(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: propSnapAt(next, 18)}})
	d = stepAhead(t, m.(Dashboard), "shift+right")
	if label := d.propAheadLabel(33.2, -117.38); !strings.HasPrefix(label, "+1 h 09:00 UTC") {
		t.Errorf("a snapshot made at 08:40 labels its first hour ahead %q; the hour shown, 09:00", label)
	}
}

// TestAnHourAheadOfARebuiltSnapshotIsDropped is the REVIEW red team's RC-4,
// L-F3's class for the hours ahead: an hour ahead asked of a snapshot that
// is rebuilt at the same moment before it is answered - another noise, the
// correction - is the old snapshot's, and is dropped, never kept for the
// new one.
func TestAnHourAheadOfARebuiltSnapshotIsDropped(t *testing.T) {
	d := chartDash(t)
	m, _ := d.Update(mapKeyMsg(t, "shift+right"))
	d = m.(Dashboard)
	d.mapPane.propAheadAsked = 0
	d, cmd := d.armPropAhead()
	if cmd == nil {
		t.Fatal("control: no hour ahead was asked")
	}
	rebuilt := *d.mapPane.propSnap
	rebuilt.Noise = ionomaps.NoiseRural // the same moment, another noise: another snapshot (D-187)
	m, _ = d.Update(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: rebuilt}, correct: d.mapPane.propSnapCorrect})
	d = m.(Dashboard)
	m, _ = d.Update(cmd())
	d = m.(Dashboard)
	if _, kept := d.mapPane.propAheadSnaps[1]; kept {
		t.Error("the hour ahead of the snapshot before the rebuild was kept for the rebuilt one")
	}
}
