package tty

import (
	"strconv"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// TestACardSaysWhoAskedForIt is 0.19.5 Q2.5 (RN-6): who asked for a beat, in
// the operator's words, not the enum's.
func TestACardSaysWhoAskedForIt(t *testing.T) {
	for _, c := range []struct {
		origin lineup.Origin
		want   string
	}{
		{lineup.FromOperator, "Station Operator"},
		{lineup.FromDirector, "Director"},
		{lineup.FromObserver, "Producer"},
	} {
		if got := requestedByOf(lineup.Card{Origin: c.origin}); got != c.want {
			t.Errorf("a card from %v says it was requested by %q; want %q", c.origin, got, c.want)
		}
	}
}

// TestChangePositionSaysItsRangeAndItsError is 0.19.5 Q2.5 (RN-6): the move
// window asks for a position, names the range the running order draws, and
// shows why a typed position was refused.
func TestChangePositionSaysItsRangeAndItsError(t *testing.T) {
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	d.cardMoveTo, d.cardErr = "9", "positions run 2 to 14"
	got := stripANSITest(strings.Join(d.movePrompt(d.opts()), "\n"))
	for _, want := range []string{
		"MOVE THIS CARD TO POSITION", "[ 9 ",
		"Positions " + strconv.Itoa(bcScheduledFrom) + " to " + strconv.Itoa(MainTrackSlots-1) + ".",
		d.cardErr, "Cancel", "Confirm",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the move window is missing %q:\n%s", want, got)
		}
	}
	d.cardErr = ""
	if strings.Contains(stripANSITest(strings.Join(d.movePrompt(d.opts()), "\n")), "positions run") {
		t.Error("the move window shows an error line with no error")
	}
}
