package app

// The deck's half of the merge: what needsRead REPORTS (0.16.0 P3).
//
// WHAT THIS CAN SEE, AND WHAT IT CANNOT (INST-5). The live stage never enters
// startSynth, so the deck can be a bare struct and every assertion here is
// about the report. The off and dark stages DO enter it, and startSynth
// resolves a voice — which on a machine without one installs Piper, minutes of
// network, inside a unit test. So those two stages are exercised here only on
// the stale-generation path, where startSynth's own first line returns, and
// their fresh path is covered structurally instead
// (maintrack_seam_test.go) and by the P3 UAT.

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// recordingDeck is a deck that can only report. Nothing else on it is set, so
// a test that reached the audio would panic rather than quietly install a voice.
func recordingDeck() (*radioDeck, *[]lineup.Event) {
	got := &[]lineup.Event{}
	d := &radioDeck{}
	d.emit = func(ev lineup.Event) { *got = append(*got, ev) }
	return d, got
}

var testRef = snapshot.LocationRef{Label: "OCEANSIDE, CA", Lat: 33.1959, Lon: -117.3795}

func TestTheDeckReportsTheNeedAndLeavesTheAirAlone(t *testing.T) {
	t.Setenv("WATCHPOST_MAINTRACK", "live")
	d, got := recordingDeck()

	d.needsRead(testRef, "no NWR relay in reach", 0) // gen 0 matches a fresh deck

	if len(*got) != 1 {
		t.Fatalf("the deck reports the need, once; got %d events", len(*got))
	}
	ev, ok := (*got)[0].(lineup.NeedsRead)
	if !ok {
		t.Fatalf("the fact reported is that the location needs a read; got %T", (*got)[0])
	}
	if ev.Ref != string(snapshot.Key(testRef)) {
		t.Errorf("the ref is the location's own key, which is what the cut-over and the composer "+
			"both resolve against; got %q", ev.Ref)
	}
	if ev.Headline != testRef.Label {
		t.Errorf("the headline is the location's name — a card is showable from the moment it "+
			"exists (DR-7); got %q", ev.Headline)
	}
	// startSynth's second statement is setMode. A blank mode is proof the deck
	// did not also start the audio, which is the double-speak this batch removes.
	d.mu.Lock()
	mode := d.mode
	d.mu.Unlock()
	if mode != "" {
		t.Errorf("the schedule owns the air at this stage; the deck must not start audio too, got mode %q", mode)
	}
}

func TestAStaleNeedIsNotReported(t *testing.T) {
	for _, stage := range []string{"", "dark", "live"} {
		t.Setenv("WATCHPOST_MAINTRACK", stage)
		d, got := recordingDeck()

		// The listener stopped, or re-tuned, while the fallback was in flight:
		// the deck's generation has moved on and this need is about a location
		// nobody is on.
		d.needsRead(testRef, "relay unavailable", 99)

		if len(*got) != 0 {
			t.Errorf("stage %q: a need that arrived after the listener moved on must not queue a card — "+
				"they would have stopped the station and been read to anyway; got %v", stage, *got)
		}
		d.mu.Lock()
		mode := d.mode
		d.mu.Unlock()
		if mode != "" {
			t.Errorf("stage %q: nor start audio; got mode %q", stage, mode)
		}
	}
}

func TestTheDefaultStageTellsTheDirectorNothing(t *testing.T) {
	t.Setenv("WATCHPOST_MAINTRACK", "")
	d, got := recordingDeck()
	// A stale generation, so the audio half returns at startSynth's own guard:
	// what is under test is that the REPORT does not happen by default, which
	// is what makes every commit before the flip a no-op for a listener.
	d.needsRead(testRef, "no NWR relay in reach", 99)
	if len(*got) != 0 {
		t.Errorf("the merge is off until it is asked for; got %v", *got)
	}
}
