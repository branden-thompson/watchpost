package tty

// broadcaster_bed_test.go — the bed is offered only when the station has a
// relay it could actually carry (D-117).
//
// HUM LEAD, 2026-09-13: "If none exist in that area - we should probably tell
// the broadcaster there is no valid relays for their area and disable the BED
// option so the Operator cannot choose something that will broadcast dead air."

import (
	"strings"
	"testing"
)

// bedConsole is a console told `relays` reach it, or told nothing at all.
//
// THE COUNT ARRIVES ON ITS OWN MESSAGE (D-125), so the fixture sends both — the
// bed's state and, when the Producer has answered, how many relays stream.
func bedConsole(t *testing.T, msg BedMsg, told bool) Broadcaster {
	t.Helper()
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	if told {
		b, _ = b.Update(msg)
		b, _ = b.Update(BedRelaysMsg{Count: bedFixtureRelays})
	}
	return b
}

// bedFixtureRelays is what the fixture's next `bedConsole` will be told.
var bedFixtureRelays int

// A STATION NOTHING STREAMS TO SAYS SO, AND OFFERS NO CONTROL.
//
// DEAD AIR IS THE OUTCOME THIS PREVENTS. A selector walking the embedded
// transmitter table — every NOAA tower in the country — lets the operator choose
// a callsign no directory carries, and cutting to it puts silence on the
// transmitter with the row still reading as tuned.
func TestABedWithNothingToCarryIsNotOffered(t *testing.T) {
	bedFixtureRelays = 0
	b := bedConsole(t, BedMsg{}, true)
	if b.bedAvailable() {
		t.Fatal("a station with no streaming relay still offers the bed")
	}
	row := stripANSITest(b.bedLine(b.opts()))
	if !strings.Contains(row, bcNoRelaysHere) {
		t.Errorf("the row does not say why there is nothing to choose:\n%q", row)
	}
	// A DIFFERENT SENTENCE FROM "not chosen yet", which is the point: one is
	// actionable and the other is a state the operator is in the middle of.
	if strings.Contains(row, bcNoRelay) {
		t.Errorf("the row says the operator has not chosen, when there is nothing to choose:\n%q", row)
	}
	// AND NO SELECTOR OVER AN EMPTY LIST.
	for _, chip := range []string{chipFor("⇧←"), chipFor("⇧→")} {
		if strings.Contains(row, chip) {
			t.Errorf("the row offers %q with nothing to step through:\n%q", chip, row)
		}
	}
}

// AND ONE THAT DOES KEEPS ITS CONTROL.
func TestABedWithRelaysIsOffered(t *testing.T) {
	bedFixtureRelays = 3
	b := bedConsole(t, BedMsg{}, true)
	if !b.bedAvailable() {
		t.Fatal("a station with three streaming relays is refused the bed")
	}
	row := stripANSITest(b.bedLine(b.opts()))
	if !strings.Contains(row, chipFor("⇧←")) || !strings.Contains(row, chipFor("⇧→")) {
		t.Errorf("the selector is missing from a station that has relays:\n%q", row)
	}
}

// UNTOLD IS AVAILABLE. The resolve is network work and lands after the console
// opens; greying the control out until then would refuse a key that is about to
// work, which reads as a broken button rather than as a pending answer.
func TestABedNobodyHasAnsweredAboutIsStillOffered(t *testing.T) {
	if !bedConsole(t, BedMsg{}, false).bedAvailable() {
		t.Error("the bed is refused before the Producer has answered how many relays reach the station")
	}
}

// TestTheRelayCountSurvivesALaterBedMessage.
//
// THE DEFECT THE ARCHITECTURE AUDIT FOUND (2026-09-13). `BedMsg` has THREE
// publishers and only one of them sets `Relays`:
//
//	app/bedrelay.go:125   {Relay, Carrying, Relays: len(st)}   ← the resolver
//	app/bedrelay.go:191   {Relay, Carrying}                    ← the selector
//	app/executors.go:288  {Relay, Carrying}                    ← the deck's state
//
// The console stores the WHOLE message, so either of the other two zeroes the
// count the resolver established — and `bedAvailable()` is `!bedTold ||
// Relays > 0`. The BED control then disables itself while relays are streaming.
//
// THAT IS D-117 FIRING BACKWARDS. The rule exists so the operator cannot choose
// something that will broadcast dead air; this tells them there is nothing to
// choose when there is.
func TestTheRelayCountSurvivesALaterBedMessage(t *testing.T) {
	b := NewBroadcaster()
	b, _ = b.Update(BedMsg{Relay: "WXM66 Victorville, CA"})
	b, _ = b.Update(BedRelaysMsg{Count: 3})
	if !b.bedAvailable() {
		t.Fatal("three relays in reach and the bed is not offered")
	}
	// The operator steps the selector, or the deck republishes its state. Neither
	// says anything about how many relays exist.
	b, _ = b.Update(BedMsg{Relay: "WXM66 Victorville, CA", Carrying: true})
	if !b.bedAvailable() {
		t.Error("a later BedMsg zeroed the relay count and disabled the bed — " +
			"the operator is told there is no relay while one is carrying")
	}
}
