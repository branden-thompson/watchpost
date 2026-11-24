package tty

import (
	"strconv"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/console"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// TestChangePositionSaysItsRangeAndItsError is 0.19.5 Q2.5 (RN-6): the move
// window asks for a position, names the range the running order draws, and
// shows why a typed position was refused.
func TestChangePositionSaysItsRangeAndItsError(t *testing.T) {
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	d.cardMoveTo, d.cardErr = "9", "positions run 2 to 14"
	got := ttytest.StripANSI(strings.Join(d.movePrompt(d.opts()), "\n"))
	for _, want := range []string{
		"MOVE THIS CARD TO POSITION", "[ 9 ",
		"Positions " + strconv.Itoa(console.ScheduledFrom) + " to " + strconv.Itoa(console.MainTrackSlots-1) + ".",
		d.cardErr, "Cancel", "Confirm",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the move window is missing %q:\n%s", want, got)
		}
	}
	d.cardErr = ""
	if strings.Contains(ttytest.StripANSI(strings.Join(d.movePrompt(d.opts()), "\n")), "positions run") {
		t.Error("the move window shows an error line with no error")
	}
}
