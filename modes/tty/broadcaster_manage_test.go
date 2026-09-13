package tty

// broadcaster_manage_test.go — the operator's two acts on a scheduled card
// (D-118).
//
// HUM LEAD, 2026-09-13: "Every line up position from UP NEXT -> Pos. 14 need the
// following controls in the Modal - these are **specific and unique** to the
// Broadcaster UI: [P] Change Position [k] Drop from Line-Up."

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// openCard is the router with a card window open on the pointed row.
func openCard(t *testing.T, at int) Router {
	t.Helper()
	r := consoleRouter(t)
	r.broadcaster.selected = at
	out, ok := r.openPointedCard()
	if !ok {
		t.Fatalf("row %d opened no card window", at)
	}
	out.observer.width, out.observer.height = 150, 74
	return out
}

func press(t *testing.T, r Router, key string) Router {
	t.Helper()
	out, taken := r.cardWindowKey(keyPress(t, key))
	if !taken {
		t.Fatalf("the card window did not take %q", key)
	}
	return out
}

// THE TWO CONTROLS ARE ON THE CARD, and they are the CONSOLE'S: Observer has no
// line-up to reorder and nothing to drop from one.
func TestTheCardWindowOffersTheConsolesTwoControls(t *testing.T) {
	r := openCard(t, 0)
	body := stripANSITest(strings.Join(r.observer.cardLines(r.observer.opts()), "\n"))
	for _, want := range []string{"Change Position", "Drop from Line-Up"} {
		if !strings.Contains(body, want) {
			t.Errorf("the card window does not offer %q:\n%s", want, body)
		}
	}
}

// AND NOT ON A CARD THE SCHEDULE WOULD REFUSE.
//
// D-45 RULES A LIVE CARD "Management Locked", and the card's own STATUS line has
// said so since D-87 — so the window that shows that line must not also offer the
// two keys it rules out. A control that is not offered and still WORKS is the
// same lie as one that is offered and does not, so the keys are refused too.
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
	// question about STATE alone: `manageable` reads the id and the state, and a
	// fixture that carried a whole read to ask it would be testing the lifecycle.
	if manageable(lineup.Card{ID: "live", State: lineup.OnAir}) {
		t.Error("a card on the air is offered the management controls")
	}
	if !manageable(live) {
		t.Error("an admitted card is refused them")
	}
	if manageable(lineup.Card{}) {
		t.Error("an empty slot is offered them")
	}
}

// [P] TAKES A POSITION AND TELLS THE SCHEDULE.
func TestChangePositionSendsTheMove(t *testing.T) {
	var gotID string
	var gotTo int
	r := openCard(t, 0)
	r.observer.cfg.MoveCard = func(id string, to int) { gotID, gotTo = id, to }
	want := r.observer.cardID

	r = press(t, r, "P")
	if r.observer.cardAct != cardActionMove {
		t.Fatal("[P] opened no position window")
	}
	for _, k := range []string{"0", "7"} {
		r = press(t, r, k)
	}
	if r.observer.cardMoveTo != "07" {
		t.Errorf("the field holds %q, want 07", r.observer.cardMoveTo)
	}
	r = press(t, r, "enter")

	if gotID != want || gotTo != 7 {
		t.Errorf("the schedule was told (%q, %d); the operator moved %q to 7", gotID, gotTo, want)
	}
	// AND THE CARD WINDOW CLOSES WITH IT: the card is no longer at the position
	// it was opened from, so a window still showing it would have moved under the
	// operator's eyes.
	if r.observer.modal != modalNone || r.observer.cardAct != cardActionNone {
		t.Errorf("after the move the window is %v/%v", r.observer.modal, r.observer.cardAct)
	}
}

// A POSITION THE RUNNING ORDER DOES NOT HAVE IS REFUSED WHERE IT CAN BE FIXED.
//
// FR-3.3: "an action must never be shown as taken unless the schedule took it".
// The schedule would refuse it out of sight; the honest form of that rule here is
// to not send it at all, and to say why on the window the operator is looking at.
func TestChangePositionRefusesWhatIsOutOfRange(t *testing.T) {
	for _, typed := range []string{"1", "15", "99", ""} {
		sent := false
		r := openCard(t, 0)
		r.observer.cfg.MoveCard = func(string, int) { sent = true }
		r = press(t, r, "P")
		r.observer.cardMoveTo = typed
		r = press(t, r, "enter")

		if sent {
			t.Errorf("%q was sent to the schedule; the table draws %d to %d",
				typed, bcScheduledFrom, MainTrackSlots-1)
		}
		if r.observer.cardErr == "" {
			t.Errorf("%q was refused with no reason on the window", typed)
		}
		if r.observer.cardAct != cardActionMove {
			t.Errorf("%q closed the window instead of letting the operator fix it", typed)
		}
	}
}

// [k] ASKS FIRST, AND SAYS WHAT DROPPING ACTUALLY DOES.
//
// AN OPERATOR WHO THINKS DROPPING A CARD REMOVES A TOWN would stop using the
// control — so the window says where the card goes and where the place stays, in
// the HUM LEAD's own terms.
func TestDropAsksAndExplainsWhatItDoes(t *testing.T) {
	r := openCard(t, 0)
	r = press(t, r, "k")
	if r.observer.cardAct != cardActionDrop {
		t.Fatal("[k] opened no confirmation")
	}
	got := stripANSITest(strings.Join(r.observer.dropPrompt(r.observer.opts()), "\n"))
	for _, want := range []string{"ARE YOU SURE", "discard pile", "location pool", "Producer"} {
		if !strings.Contains(got, want) {
			t.Errorf("the confirmation does not mention %q:\n%s", want, got)
		}
	}
}

// AND IT DROPS ONLY ON A YES.
func TestDropSendsOnlyOnConfirmation(t *testing.T) {
	dropped := ""
	r := openCard(t, 0)
	r.observer.cfg.DropCard = func(id string) { dropped = id }
	want := r.observer.cardID

	r = press(t, r, "k")
	r = press(t, r, "esc")
	if dropped != "" {
		t.Fatalf("esc dropped %q", dropped)
	}
	if r.observer.cardAct != cardActionNone || r.observer.modal != modalCard {
		t.Error("esc left the confirmation up, or closed the card behind it")
	}

	r = press(t, r, "k")
	r = press(t, r, "enter")
	if dropped != want {
		t.Errorf("enter dropped %q, want %q", dropped, want)
	}
}

// AND AN OPEN QUESTION OWNS THE KEYBOARD (D-58, one window deeper).
//
// A DIGIT TYPED INTO THE POSITION FIELD MUST NOT REACH THE RUNNING ORDER, and
// `enter` must settle the question rather than closing the card behind it — which
// is what it does with no question open (D-109).
func TestAnOpenQuestionOwnsTheKeyboard(t *testing.T) {
	r := openCard(t, 0)
	r = press(t, r, "P")
	// A DIGIT IS THE FIELD'S. Without the overlay it addresses a slot and opens
	// that card instead (D-88).
	r = press(t, r, "3")
	if r.observer.cardMoveTo != "3" {
		t.Errorf("the digit did not reach the field: %q", r.observer.cardMoveTo)
	}
	if r.observer.modal != modalCard {
		t.Error("the digit reached the running order and changed the window")
	}
	// AND A STRAY KEY OVER THE DROP QUESTION IS SWALLOWED: it is the last thing
	// between the operator and a card leaving the running order.
	r = press(t, r, "esc")
	r = press(t, r, "k")
	if _, taken := r.cardWindowKey(tea.KeyPressMsg{Code: 'x', Text: "x"}); !taken {
		t.Error("a stray key fell through the confirmation to the console beneath it")
	}
}
