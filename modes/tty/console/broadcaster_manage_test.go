package console

// broadcaster_manage_test.go — the operator's two acts on a scheduled card
// (0.16.0 D-118).
//
// HUM LEAD, 2026-09-13: "Every line up position from UP NEXT -> Pos. 14 need the
// following controls in the Modal - these are **specific and unique** to the
// Broadcaster UI: [P] Change Position [k] Drop from Line-Up."

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// AND NOT ON A CARD THE SCHEDULE WOULD REFUSE.
//
// 0.16.0 D-45 RULES A LIVE CARD "Management Locked", and the card's own
// STATUS line says so (D-87) — so the window that shows that line must not
// also offer the two keys it rules out. A control that is not offered and
// still WORKS is the same lie as one that is offered and does not, so the
// keys are refused too.
func TestALiveCardOffersNeitherControl(t *testing.T) {
	live, err := lineup.Propose(lineup.Card{ID: "live", Slot: lineup.LocationReport,
		Subject: "here", Headline: "here", State: lineup.Proposed})
	if err != nil {
		t.Fatal(err)
	}
	if live, err = live.To(lineup.Admitted); err != nil {
		t.Fatal(err)
	}
	// BUILT DIRECTLY, because `To(OnAir)` demands a card's words and this is a
	// question about STATE alone: `Manageable` reads the id and the state, and a
	// fixture that carried a whole read to ask it would be testing the lifecycle.
	if Manageable(lineup.Card{ID: "live", State: lineup.OnAir}) {
		t.Error("a card on the air is offered the management controls")
	}
	if !Manageable(live) {
		t.Error("an admitted card is refused them")
	}
	if Manageable(lineup.Card{}) {
		t.Error("an empty slot is offered them")
	}
}
