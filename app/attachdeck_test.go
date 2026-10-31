package app

import (
	"context"
	"testing"
	"time"
)

// TestARunWithNoDeckStarts is C-6's split of RunDashboard: attachRadio
// returns no deck when one cannot be built, and attaching it then tells it
// nothing - where the deck's clock and air were set without asking.
func TestARunWithNoDeckStarts(t *testing.T) {
	lp := &livePipelines{}
	lp.attachDeck(context.Background(), nil, nil, nil, tickerPrefs{}, "v0", time.Now())
	if lp.deck != nil || lp.release == nil {
		t.Errorf("with no deck, the run holds deck %v and release watch %v", lp.deck, lp.release)
	}
}
