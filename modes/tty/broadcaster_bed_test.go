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

func bedConsole(t *testing.T, msg BedMsg, told bool) Broadcaster {
	t.Helper()
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	if told {
		b, _ = b.Update(msg)
	}
	return b
}

// A STATION NOTHING STREAMS TO SAYS SO, AND OFFERS NO CONTROL.
//
// DEAD AIR IS THE OUTCOME THIS PREVENTS. The selector used to walk the embedded
// transmitter table — every NOAA tower in the country — so the operator could
// choose a callsign no directory carries, and cutting to it put silence on the
// transmitter with the row still reading as tuned.
func TestABedWithNothingToCarryIsNotOffered(t *testing.T) {
	b := bedConsole(t, BedMsg{Relays: 0}, true)
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
	b := bedConsole(t, BedMsg{Relays: 3}, true)
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
