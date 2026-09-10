package app

// The deck's half of the merge: what needsRead REPORTS (0.16.0 P3).
//
// THE STAGE IS GONE (P3(d)). While the merge was landing, a three-state switch
// kept the air single-owner between the producer arriving and the flip; it was
// deleted with startSynth's direct path, because a switch that outlived the
// merge would be a second way for the station to behave — the thing being
// removed. What is left is the rule it protected: the deck REPORTS, and starts
// no audio of its own.

import (
	"os"
	"strings"
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
		t.Errorf("the ref is the location's own key, which is what the cut-over, the composer and the "+
			"reader all resolve against; got %q", ev.Ref)
	}
	if ev.Headline != testRef.Label {
		t.Errorf("the headline is the location's name — a card is showable from the moment it "+
			"exists (DR-7); got %q", ev.Headline)
	}
	// setMode is readReport's first statement. A blank mode is proof the deck
	// did not also start the audio, which is the double-speak this batch
	// removed — and the deck here has no engine, so it could not have.
	d.mu.Lock()
	mode := d.mode
	d.mu.Unlock()
	if mode != "" {
		t.Errorf("the schedule owns the air; the deck must not start audio when it reports, got mode %q", mode)
	}
}

// THE REASON IS FILED FOR THE READ THAT FOLLOWS, and taken exactly once.
//
// It is the deck's own string — "the relay was silent" is not a fact the
// schedule has any use for — so it travels beside the event rather than on it,
// and the read that starts minutes later puts it on the player's detail line.
func TestTheReasonIsKeptForTheReadAndTakenOnce(t *testing.T) {
	d, _ := recordingDeck()
	key := string(snapshot.Key(testRef))

	d.needsRead(testRef, "the relay was silent", 0)

	if got := d.takeWhy(key); got != "the relay was silent" {
		t.Errorf("the read must be able to say WHY it is happening; got %q", got)
	}
	if got := d.takeWhy(key); got != "" {
		t.Errorf("a reason is about ONE read: taken twice it would explain the wrong one; got %q", got)
	}
}

func TestTheReasonStoreIsBounded(t *testing.T) {
	d, _ := recordingDeck()
	for i := range needWhyCap * 3 {
		d.rememberWhy(string(rune('a'+i%26))+string(rune('a'+i/26)), "why")
	}
	d.mu.Lock()
	n := len(d.needWhy)
	d.mu.Unlock()
	if n > needWhyCap {
		t.Errorf("the store holds %d reasons against a cap of %d — a 24/7 station whose watchlist "+
			"churns must not accumulate strings for the life of the process", n, needWhyCap)
	}
	if n == 0 {
		t.Error("bounding it to nothing is not bounding it: the next read would have no reason to give")
	}
}

func TestAStaleNeedIsNotReported(t *testing.T) {
	d, got := recordingDeck()

	// The listener stopped, or re-tuned, while the fallback was in flight: the
	// deck's generation has moved on and this need is about a location nobody
	// is on.
	d.needsRead(testRef, "relay unavailable", 99)

	if len(*got) != 0 {
		t.Errorf("a need that arrived after the listener moved on must not queue a card — "+
			"they would have stopped the station and been read to anyway; got %v", *got)
	}
	if got := d.takeWhy(string(snapshot.Key(testRef))); got != "" {
		t.Errorf("nor leave a reason behind for the next read to explain itself with; got %q", got)
	}
}

// THE DARK RUN'S INSTRUMENT OUTLIVED THE DARK RUN, and deliberately: "which
// location did the deck decide needed a read, when, and was it fresh" is the
// first question anyone asks of a station that read the wrong thing.
func TestTheDeckRecordsTheNeedItActedOn(t *testing.T) {
	path := radioDebugTo(t, "1")
	d, _ := recordingDeck()

	// A stale generation, so nothing follows: what is under test is the
	// record, which is written before any of that is decided.
	d.needsRead(testRef, "no NWR relay in reach", 99)

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the need was recorded nowhere: %v", err)
	}
	line := strings.TrimSpace(string(b))
	for _, w := range []string{
		"needs-read",
		"fresh=false", // and it says WHY nothing followed
		"ref=" + string(snapshot.Key(testRef)),
		"why=no NWR relay in reach",
	} {
		if !strings.Contains(line, w) {
			t.Errorf("the record must carry %q, or it cannot be paired with what the station did; got %q", w, line)
		}
	}
}

// THE DIAGNOSTIC IS OFF BY DEFAULT, and a station running without it must not
// pay for a line nobody collects.
func TestTheNeedIsNotRecordedWhenTheDiagnosticIsOff(t *testing.T) {
	path := radioDebugTo(t, "1")
	t.Setenv("WATCHPOST_DEBUG_RADIO", "")
	d, _ := recordingDeck()
	d.needsRead(testRef, "no NWR relay in reach", 99)
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		t.Errorf("the diagnostic is opt-in; got %q", b)
	}
}

// THE LISTENER IS TOLD WHY THE STATION IS READING (0.16.0 P3(d)).
//
// A report starts for one of three reasons — no relay covers this location,
// the relay failed, or the relay went silent — and the detail line is the only
// place any of that reaches a person. Asserted here rather than inside
// readReport because readReport resolves a voice, which on a machine without
// one is a 63 MB download inside a unit test.
func TestTheStationSaysWhyItIsReadingRatherThanRelaying(t *testing.T) {
	d, _ := recordingDeck()
	d.needsRead(testRef, "the relay was silent", 0)

	d.announceReport(testRef)

	d.mu.Lock()
	mode, station, detail := d.mode, d.station, d.detail
	d.mu.Unlock()
	if mode != "synth" {
		t.Errorf("the station is on its own broadcast; got mode %q", mode)
	}
	if station != "Watchpost Synth · "+testRef.Label {
		t.Errorf("the player row names the station and the location; got %q", station)
	}
	if detail != "the relay was silent" {
		t.Errorf("the detail line must say WHY this read is happening; got %q — a listener whose "+
			"relay just died is owed the reason, and it is the only place it appears", detail)
	}

	// AND ONLY FOR THIS READ. The next report has its own reason, or none.
	d.announceReport(testRef)
	d.mu.Lock()
	again := d.detail
	d.mu.Unlock()
	if again != "" {
		t.Errorf("a reason explains ONE read; carried forward it explains the wrong one; got %q", again)
	}
}
