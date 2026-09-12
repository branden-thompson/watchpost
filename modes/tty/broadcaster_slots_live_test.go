package tty

// broadcaster_slots_live_test.go — where the line-up starts, and what LIVE holds
// (D-84).
//
// THIS FILE HELD THE LIVE CARD'S SCRIPT WINDOW (D-83) AND NO LONGER DOES. Seven
// tests pinned a card that showed the words it was about to read; D-87 replaced
// that card with a MANIFEST, on the HUM LEAD's own reasoning — "the full script
// on the top level card doesn't make sense when I can drill down to read the
// whole thing" — so the rules those tests held are gone rather than weakened,
// and the tests went with them. One batch's work retired by the next is the
// UAT loop doing its job; a test kept alive against a retired design asserts
// the past.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// reading is a card with words on it, the way one arrives at standby (DR-7).
func readingCard(t *testing.T, lines ...string) lineup.Card {
	t.Helper()
	parts := make([]lineup.Part, 0, len(lines))
	for _, l := range lines {
		parts = append(parts, lineup.Part{Kind: lineup.PartLine, Text: l})
	}
	c, err := lineup.Propose(lineup.Card{ID: "c", Slot: lineup.LocationReport,
		Subject: "oceanside", Headline: "LOCATION REPORT • OCEANSIDE, CA 92057"})
	if err != nil {
		t.Fatalf("proposing: %v", err)
	}
	c.Script = lineup.Script{Parts: parts}
	return c
}

// LIVE IS WHAT IS ON THE AIR, AND ON STANDBY NOTHING IS (D-84, HUM LEAD
// 2026-09-11).
//
//	"1. When the user enters Broadcaster Mode, the station is in STANDBY (DEAD
//	 AIR) … 3. The director should be choosing and populating the line-up. 4.
//	 LIVE should remain EMPTY … 5. The UP NEXT card should be getting the
//	 attention of the Composer … it will be the first thing that goes ON AIR when
//	 the human operator hits SHIFT+ENTER."
func TestOnStandbyTheLineUpStartsAtUpNextAndLiveIsEmpty(t *testing.T) {
	var l lineup.Lineup
	for _, id := range []string{"first", "second", "third"} {
		c := readingCard(t, "the words for "+id)
		c.ID, c.Subject = id, id
		c.Headline = "LOCATION REPORT • " + strings.ToUpper(id)
		admitted, err := c.To(lineup.Admitted)
		if err != nil {
			t.Fatalf("admitting %s: %v", id, err)
		}
		if l, err = l.Queue(lineup.MainTrack, admitted); err != nil {
			t.Fatalf("seeding %s: %v", id, err)
		}
	}
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l}) // power is STOPPED: the station has not gone on air

	// LIVE holds nothing…
	if c, decided := b.slotCard(l.Projection(lineup.MainTrack), 0); decided {
		t.Errorf("LIVE holds %q on a station that is not on the air", c.ID)
	}
	// …and the head of the queue is UP NEXT, where the operator can see the words
	// that will go out first.
	if c, decided := b.slotCard(l.Projection(lineup.MainTrack), 1); !decided || c.ID != "first" {
		t.Errorf("UP NEXT holds %+v; want the head of the line-up", c)
	}
	// AND THE OPERATOR CAN SEE WHAT IT IS. The card is a MANIFEST since D-87 —
	// its name, its status and what it contains — so what UP NEXT shows is the
	// card, not its words.
	got := stripANSITest(b.View().Content)
	if !strings.Contains(got, "FIRST") {
		t.Error("the operator cannot see what will go out first; UP NEXT drew no card")
	}
}

// AND GOING ON AIR MOVES EVERYTHING UP ONE. That is the same movement D-40
// already rules for a dropped card — "everything below moves up one" — so the
// operator has seen it before.
func TestGoingOnAirMovesTheHeadOfTheLineUpIntoLive(t *testing.T) {
	var l lineup.Lineup
	c := readingCard(t, "the words for first")
	c.ID, c.Subject, c.Headline = "first", "first", "LOCATION REPORT • FIRST"
	admitted, err := c.To(lineup.Admitted)
	if err != nil {
		t.Fatalf("admitting: %v", err)
	}
	if l, err = l.Queue(lineup.MainTrack, admitted); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l})
	b.power = lineup.Running

	if c, decided := b.slotCard(l.Projection(lineup.MainTrack), 0); !decided || c.ID != "first" {
		t.Errorf("on air, LIVE holds %+v; want the head of the line-up", c)
	}
}

// THE OFFSET FOLLOWS THE POWER, NOT WHETHER A CARD HAPPENS TO BE PLAYING.
//
// A running station between two reads — the next card still building — has
// nothing on the air for a moment. Keying the offset off that would shunt every
// card down a row and back for the length of one build, which the operator would
// read as the console losing its place.
func TestTheLiveSlotDoesNotShuntWhileARunningStationIsBetweenReads(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	if got := b.liveOffset(); got != 0 {
		t.Errorf("a running station with nothing on the air offset by %d; the power is what decides", got)
	}
	b.power = lineup.Stopped
	if got := b.liveOffset(); got != 1 {
		t.Errorf("a stopped station offset by %d; LIVE is empty and the queue starts at UP NEXT", got)
	}
}
