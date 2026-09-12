package app

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

func bedPipelines(t *testing.T) *livePipelines {
	t.Helper()
	tbl, err := stream.LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{idx: indexForTest(t), relayTable: tbl}
	lp.setStation(stationFrom(config.Config{Locations: []config.Location{bonsallCfg}}))
	lp.bedRadiusMi = config.DefaultBedRadiusMi
	return lp
}

// THE SELECTOR WALKS THE STATION'S FENCE, NOT THE LISTENER'S WATCHLIST (D-78).
//
//	"Lone Pine's 'Fresno' relay is completely inappropriate for being the relay"
//	 — HUM LEAD, 2026-09-10
//
// WHAT IT WALKS IS MEASURED: a 100-mile fence around Bonsall reaches eight
// transmitters, which is why the fence is 100 and not the 25-mile service radius
// that would leave this control with one thing to select.
func TestTheBedSelectorWalksTheStationsFence(t *testing.T) {
	lp := bedPipelines(t)
	relays := lp.bedRelays()
	if len(relays) < 6 {
		t.Fatalf("the station's fence reaches a real choice; got %d", len(relays))
	}
	// NEAREST FIRST, and all of them inside the fence.
	for _, n := range relays {
		if mi := n.KM * 0.621371; mi > config.DefaultBedRadiusMi+0.001 {
			t.Errorf("%s is %.1f mi out and the fence is %v", n.Callsign, mi, config.DefaultBedRadiusMi)
		}
	}
	// AND A STATION WITH NO EPICENTRE HAS NOTHING TO CHOOSE BETWEEN, rather than
	// the whole country.
	bare := &livePipelines{relayTable: lp.relayTable}
	if got := bare.bedRelays(); len(got) != 0 {
		t.Errorf("no transmitter is no region; got %d relays", len(got))
	}
}

// THE ARROWS MOVE THE SELECTION, AND THEY WRAP.
//
// A SELECTOR THAT STOPPED AT THE ENDS would leave the operator pressing a key
// that does nothing and wondering which of the two reasons it was.
func TestSteppingTheBedWrapsBothWays(t *testing.T) {
	lp := bedPipelines(t)
	n := len(lp.bedRelays())
	if n < 2 {
		t.Fatalf("the fixture needs somewhere to step; got %d relays", n)
	}
	if cmd := lp.stepBedRelay(-1); cmd != nil {
		cmd()
	}
	if lp.bedPick != n-1 {
		t.Errorf("stepping back from the first wraps to the last; got %d of %d", lp.bedPick, n)
	}
	if cmd := lp.stepBedRelay(1); cmd != nil {
		cmd()
	}
	if lp.bedPick != 0 {
		t.Errorf("and forward from the last wraps to the first; got %d", lp.bedPick)
	}
	// A STATION WITH NO RELAYS IS A NO-OP, never a panic or a negative index.
	bare := &livePipelines{relayTable: lp.relayTable}
	_ = bare.stepBedRelay(1)
	if bare.bedPick != 0 {
		t.Errorf("a fence with nothing in it selects nothing; got %d", bare.bedPick)
	}
}

// THE ROW READS THE WAY THE REFERENCE DRAWS IT.
//
//	[ ← ] KIG78 Coachella CA 162.400 MHz · 41mi from TOWER GPS  [ → ]
func TestTheRelayRowNamesTheTransmitter(t *testing.T) {
	lp := bedPipelines(t)
	got := relayLine(lp.bedRelays()[0])
	for _, want := range []string{"MHz", "mi from TOWER GPS"} {
		if !strings.Contains(got, want) {
			t.Errorf("the relay row is missing %q: %q", want, got)
		}
	}
	if strings.HasPrefix(got, " ") || strings.Contains(got, "  ") {
		t.Errorf("the row carries a stray gap: %q", got)
	}
}

// AND THE TWO PUBLISHERS AGREE ABOUT WHETHER THE BED IS CARRYING.
//
// The Director publishes it and the selector publishes it, and a selector that
// GUESSED would flicker the row between them. It is recorded from the Director,
// never decided here.
func TestTheSelectorReportsWhatTheDirectorSaidAboutCarrying(t *testing.T) {
	lp := bedPipelines(t)
	if lp.bedCarrying() {
		t.Error("a station that has been told nothing is not carrying")
	}
	lp.noteBedCarrying(true)
	if !lp.bedCarrying() {
		t.Error("what the Director said is what the selector reports")
	}
}

// THE OPERATOR'S CHOICE STICKS (F-98, D-90).
//
// HUM LEAD, UAT 2026-09-11: "Relay control ← (no relay tuned) → doesn't 'stick' to
// my choice — no matter what I choose it will constantly go back to (no relay
// tuned) so there's no way to ensure which bed relay I've actually selected."
//
// THE MECHANISM: two publishers of BedMsg, and the frequent one did not know about
// the operator. `stepBedRelay` sends the relay it tuned; the SETTLE sends
// `describeBed(v.Bed)` — the DIRECTOR's bed ref, a watchlist LOCATION key the
// station's relay selector never touches — and a settle happens on every tick. So
// the selection was overwritten with "" about a second after every keypress.
//
// It is the same two-publisher problem `noteBedCarrying` already solved for
// `Carrying`, left unsolved for the relay itself.
func TestTheOperatorsRelayChoiceSurvivesTheNextSettle(t *testing.T) {
	lp := bedPipelines(t)
	if cmd := lp.stepBedRelay(1); cmd != nil {
		cmd()
	}
	chose := lp.selectedRelay()
	if chose == "" {
		t.Fatalf("the fixture selected nothing, so this test cannot fail")
	}

	// THE SETTLE, WITH THE DIRECTOR'S BED UNTUNED — which is the ordinary state of
	// a station whose operator is choosing a relay before going on the air, and
	// the exact state the defect was reported in.
	var got []tty.BedMsg
	b := newBench(t, &scriptVoice{})
	b.x.publish = func(m tea.Msg) {
		if bm, ok := m.(tty.BedMsg); ok {
			got = append(got, bm)
		}
	}
	b.x.bedLabel = func(string) string { return "" }
	b.x.selected = lp.selectedRelay
	b.x.run(context.Background(), lineup.Publish{})

	if len(got) != 1 {
		t.Fatalf("a settle tells the console about the bed exactly once; got %d", len(got))
	}
	if got[0].Relay != chose {
		t.Errorf("the settle overwrote the operator's choice:\n  chose:     %q\n  published: %q\n"+
			"a row that reverts a second after every keypress gives the operator no way to know "+
			"which relay they have", chose, got[0].Relay)
	}
}

// A STATION NOBODY HAS TOUCHED HAS CHOSEN NOTHING, and says so (mY4).
//
// `bedPick` CANNOT STAND IN FOR THE CHOICE, which is the whole reason `bedRelay`
// exists: an index's zero value is a REAL relay, so a selector that reported
// `relays[bedPick]` would name the nearest transmitter as though the operator had
// picked it — claiming a tune that never happened, on the row whose entire job is
// to say which relay the station is on.
//
// THE STUB IN THE TEST BELOW CANNOT CATCH THIS. It hands `describeBed` a
// `func() string { return "" }`, which asserts the FALLBACK and says nothing about
// whether the real implementation would have returned "". A mutant found that gap.
func TestAStationNobodyHasTouchedHasChosenNoRelay(t *testing.T) {
	lp := bedPipelines(t)
	if len(lp.bedRelays()) == 0 {
		t.Fatalf("the fixture reaches no relays, so there is nothing to wrongly report")
	}
	if got := lp.selectedRelay(); got != "" {
		t.Errorf("an untouched station reports %q as the operator's choice; nothing has been tuned, "+
			"and bedPick's zero value is a real relay rather than an absent one", got)
	}
	// AND THE FIRST KEYPRESS IS WHAT MAKES IT AN ANSWER.
	if cmd := lp.stepBedRelay(1); cmd != nil {
		cmd()
	}
	if got := lp.selectedRelay(); got == "" {
		t.Errorf("the operator stepped the selector and it still reports no choice")
	}
}

// AND A STATION WHOSE OPERATOR HAS CHOSEN NOTHING STILL NAMES THE DIRECTOR'S BED.
//
// The fallback is not dead code: until the operator touches the selector, what the
// bed is carrying is the monitor's rotation, and that is the only honest answer.
func TestAnUnchosenRelayLeavesTheDirectorsBedOnTheRow(t *testing.T) {
	var got []tty.BedMsg
	b := newBench(t, &scriptVoice{})
	b.x.publish = func(m tea.Msg) {
		if bm, ok := m.(tty.BedMsg); ok {
			got = append(got, bm)
		}
	}
	b.x.bedLabel = func(string) string { return "OCEANSIDE, CA" }
	b.x.selected = func() string { return "" }
	b.x.run(context.Background(), lineup.Publish{Bed: lineup.BedState{Ref: "oceanside"}})

	if len(got) != 1 || got[0].Relay != "OCEANSIDE, CA" {
		t.Errorf("with no operator choice the row names what the Director's bed is carrying; got %+v", got)
	}
}
