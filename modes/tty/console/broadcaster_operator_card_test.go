package console

// broadcaster_operator_card_test.go — who asked for a beat, on its card.

import (
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
