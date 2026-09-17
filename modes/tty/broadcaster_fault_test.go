package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// F-150, REVIEW — THE OPERATOR SEES A FAULT RUN ON THE CONSOLE, in the station's
// own band while ON AIR: the count and the reason. A read that finished clears
// it (Run 0), and so does STANDBY, which is the operator acting on it.
func TestAFaultRunShowsInTheBandWhileOnAirAndClears(t *testing.T) {
	b := bcStandby(t, true, 0)
	b, _ = b.Update(StationMsg{Power: lineup.Running})
	b, _ = b.Update(StationFaultMsg{Run: 3, Reason: "the report could not be composed: no key"})
	got := stripANSITest(b.View().Content)
	for _, want := range []string{"3 CARD(S) FAILED", "could not perform", "no key"} { // bounded by the phrases (P10-02)
		if !strings.Contains(got, want) {
			t.Errorf("an ON AIR station with three failed cards shows no %q:\n%s", want, got)
		}
	}
	cleared, _ := b.Update(StationFaultMsg{})
	if strings.Contains(stripANSITest(cleared.View().Content), "FAILED") {
		t.Error("a read that finished did not clear the fault band")
	}
	standby, _ := b.Update(StationMsg{Power: lineup.OffAir})
	if strings.Contains(stripANSITest(standby.View().Content), "FAILED") {
		t.Error("STANDBY did not clear the fault band")
	}
}
