package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// P1(b): the console shows what the schedule PUBLISHED, and nothing it made up
// itself. A console that painted its own idea of the running order would be
// asserting an order the schedule does not follow — RS-1, the release's
// UI-integrity risk.

func bcWith(t *testing.T, cards ...lineup.Card) Broadcaster {
	t.Helper()
	var l lineup.Lineup
	for _, c := range cards {
		next, err := l.Queue(lineup.MainTrack, c)
		if err != nil {
			t.Fatalf("seeding the lineup: %v", err)
		}
		l = next
	}
	b := NewBroadcaster()
	b, _ = b.Update(LineupMsg{Lineup: l})
	b.width, b.height = 150, 74
	return b
}

func card(t *testing.T, id, subject string) lineup.Card {
	t.Helper()
	c, err := lineup.Propose(lineup.Card{ID: id, Slot: lineup.LocationReport, Subject: subject, Headline: subject, State: lineup.Proposed})
	if err != nil {
		t.Fatalf("proposing %s: %v", id, err)
	}
	// The lineup holds ADMITTED cards only — its own invariant, and it refused
	// a proposed one. The fixture follows the real lifecycle rather than
	// side-stepping it.
	c, err = c.To(lineup.Admitted)
	if err != nil {
		t.Fatalf("admitting %s: %v", id, err)
	}
	return c
}

func TestTheConsoleShowsTheMainTrackItWasPublished(t *testing.T) {
	b := bcWith(t, card(t, "a", "OCEANSIDE"), card(t, "b", "BONSALL"))
	got := b.View().Content
	for _, want := range []string{"OCEANSIDE", "BONSALL"} {
		if !strings.Contains(got, want) {
			t.Errorf("the console must show the card the schedule published; %q is missing from the frame", want)
		}
	}
}

func TestTheConsoleShowsNothingItWasNotPublished(t *testing.T) {
	b := bcWith(t) // an empty lineup
	if got := b.View().Content; strings.Contains(got, "OCEANSIDE") {
		t.Error("the console invented a card the schedule never published")
	}
}

func TestTheConsoleNumbersTheMainTrackSlots(t *testing.T) {
	b := bcWith(t, card(t, "a", "ONE"), card(t, "b", "TWO"))
	got := stripANSITest(b.View().Content)

	// THE READ CARDS KEEP THEIR CHIP, AND THE TABLE KEEPS ITS NUMBER (D-94).
	//
	// Slots 0 and 1 are the two the operator READS from and they are still cards,
	// so `[1]` is still a chip on one. Everything below is a table row now, and
	// the reference addresses those by the `##.` column exactly as Observer's
	// table does — the number IS the handle, so a chip beside it would be the
	// address written twice.
	if !strings.Contains(got, chipFor("1")) {
		t.Error("the UP NEXT card lost its handle (FR-2.4)")
	}
	if strings.Contains(got, chipFor("0")) {
		t.Error("the standby box carries a handle: [0] addresses nothing while the station is at rest (D-89)")
	}
	for _, want := range []string{"02.", "03.", "04."} {
		if !strings.Contains(got, want) {
			t.Errorf("the running order does not number slot %q; the ##. column IS the address", want)
		}
	}
}

func TestTheConsoleShowsAtMostFifteenMainTrackSlots(t *testing.T) {
	var cs []lineup.Card
	for i := 0; i < 20; i++ {
		cs = append(cs, card(t, string(rune('a'+i)), "LOC"+string(rune('A'+i))))
	}
	got := stripANSITest(bcWith(t, cs...).View().Content)

	// FIFTEEN, RULED 2026-09-12 (was ten). LIVE is 0, UP NEXT is 1, and the table
	// draws 2..14 — so a `15.` row is a sixteenth slot that must not exist.
	if strings.Contains(got, "15.") {
		t.Error("the main track is a ROLLING view of FIFTEEN; a sixteenth slot reached the frame")
	}
	if !strings.Contains(got, "14.") {
		t.Error("the fifteenth slot did not reach the frame")
	}
}

// FR-2.6: external text in the NEW lanes goes through the EXISTING clamp.
// One owner, no second path — a card's headline is provider prose and a
// provider can send anything.
func TestAHostileHeadlineIsClampedInTheNewLanes(t *testing.T) {
	b := bcWith(t, card(t, "a", "NORMAL\x1b[31mRED\x1b[0m\x07"))
	got := b.View().Content
	if strings.Contains(got, "\x1b[31m") || strings.Contains(got, "\x07") {
		t.Error("a card's headline reached the frame with escapes intact — the new lanes must route " +
			"external text through the existing plaintext clamp, not a second path")
	}
	if !strings.Contains(got, "NORMAL") {
		t.Error("clamping must strip the escapes and KEEP the words")
	}
}

// notice is the Director's own structural card — the staleness replacement read,
// which director.go has queued straight onto the main track since 0.14.0.
func notice(t *testing.T, id string) lineup.Card {
	t.Helper()
	c, err := lineup.Propose(lineup.Card{ID: id, Slot: lineup.Transition, Origin: lineup.FromDirector,
		Subject: "stale read", Headline: "Report out of date",
		Script: lineup.Say("That report is out of date and has been dropped.")})
	if err != nil {
		t.Fatalf("proposing the notice: %v", err)
	}
	c, err = c.To(lineup.Admitted)
	if err != nil {
		t.Fatalf("admitting the notice: %v", err)
	}
	return c
}

// D-44: the console draws the LINE-UP, not the SCHEDULE.
//
// The two differ by the Director's own structural cards, which are read on air
// and never shown — the operator did not ask for them, and a slot number spent
// on one is a number they cannot address. This is a LIVE difference: the
// staleness notice reaches the main track in production.
//
// IT IS ALSO WHAT KEEPS THE SLOT NUMBERS HONEST. The console numbers rows by
// their position in what it drew, and `Moved` carries that number back to the
// schedule; a hidden card between two visible ones shifts every number below it.
func TestTheConsoleDrawsTheLineUpAndNotTheSchedule(t *testing.T) {
	b := bcWith(t, card(t, "a", "OCEANSIDE"), notice(t, "n1"), card(t, "b", "BONSALL"))
	got := b.View().Content

	if strings.Contains(got, "Report out of date") {
		t.Error("the Director's own structural card must not appear in the operator's running order")
	}
	for _, want := range []string{"OCEANSIDE", "BONSALL"} {
		if !strings.Contains(got, want) {
			t.Errorf("and the cards the operator DID schedule must still be drawn; %q is missing", want)
		}
	}
	// THE SLOT NUMBERS ARE THE POINT. With the notice hidden, BONSALL is the
	// operator's slot 1 — and slot 1 is the number `Moved` would carry back.
	// Drawing the schedule instead would number it 2 and move the wrong card.
	oceanside := strings.Index(got, "OCEANSIDE")
	bonsall := strings.Index(got, "BONSALL")
	if oceanside < 0 || bonsall < 0 {
		t.Fatal("both cards must be on the frame")
	}
	between := got[oceanside:bonsall]
	if strings.Contains(between, "2") && !strings.Contains(between, "1") {
		t.Errorf("BONSALL must be numbered as the operator's slot 1, not the schedule's 2:\n%s", between)
	}
}
