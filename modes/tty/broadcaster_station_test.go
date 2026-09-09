package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// P2: the station's state comes from the DIRECTOR's power, never a UI flag
// (FR-5.1). A console that kept its own idea of whether it was on the air
// could show ON AIR while nothing was broadcasting — and the swap gate is
// about to depend on this value, so a local copy would be a safety bug, not a
// display bug.

func bcPowered(p lineup.Power) Broadcaster {
	b := NewBroadcaster()
	b.width, b.height = 150, 74
	b, _ = b.Update(StationMsg{Power: p})
	return b
}

func TestTheBannerReadsTheDirectorsPowerNotALocalFlag(t *testing.T) {
	if got := bcPowered(lineup.Running).power; got != lineup.Running {
		t.Errorf("the console's power must be what the Director published; got %v", got)
	}
	if got := bcPowered(lineup.OffAir).power; got != lineup.OffAir {
		t.Errorf("the console's power must be what the Director published; got %v", got)
	}
}

// D-21: Variant C — the state as a labelled field with the transition in
// parentheses, and the bed row's third use of the word replaced.
func TestTheBannerIsVariantC(t *testing.T) {
	on := bcPowered(lineup.Running).View().Content
	if !strings.Contains(on, "STATION:") {
		t.Error("Variant C names the state as a labelled field: the frame carries no STATION: label")
	}
	if !strings.Contains(on, "ON AIR") {
		t.Error("a running station reads ON AIR")
	}
	if !strings.Contains(on, "STANDBY") {
		t.Error("Variant C puts the transition in parentheses, so the destination state is named")
	}
	off := bcPowered(lineup.OffAir).View().Content
	if !strings.Contains(off, "STANDBY") || strings.Contains(off, "*** ON AIR") {
		t.Error("a station OffAir must not read as emphasised ON AIR")
	}
}

// FR-5.3: the state is legible WITHOUT colour. A background alone fails the
// --ascii path and any terminal without colour, so the words carry it too.
func TestTheStateIsLegibleWithoutColour(t *testing.T) {
	b := bcPowered(lineup.OffAir)
	b.ascii = true
	got := b.View().Content
	if !strings.Contains(got, "STANDBY") {
		t.Error("under --ascii, with no colour at all, the state must still be readable in words")
	}
}

// FR-5.5: Watchpost has NO radio path. "On the air" can only mean the
// software is putting programme out, never that an antenna is radiating — and
// an operator who infers otherwise has been misled by us. The boundary is
// stated where they read it, not only in a design document.
func TestTheOnAirBoundaryIsStatedToTheOperator(t *testing.T) {
	got := bcPowered(lineup.Running).View().Content
	if !strings.Contains(strings.ToUpper(got), "AUDIO OUT") {
		t.Error("FR-5.5: the console must say what ON AIR means — that it is audio out of this " +
			"program, not a transmitter it cannot observe")
	}
}
