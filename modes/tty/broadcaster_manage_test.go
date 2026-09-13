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

	// THE SLOT THE OPERATOR TYPED, TRANSLATED TO A LINE-UP INDEX (D-119). On a
	// station at STANDBY the two differ by one — LIVE is empty and the line-up is
	// drawn from UP NEXT down (D-84) — and `Reorder` takes the INDEX. Sending the
	// typed number straight through moved the card one place further down than
	// the operator asked, silently, on the surface's normal state.
	wantTo := 7 - r.broadcaster.liveOffset()
	if gotID != want || gotTo != wantTo {
		t.Errorf("the schedule was told (%q, %d); slot 7 is line-up index %d", gotID, gotTo, wantTo)
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
	for _, typed := range []string{"1", "16", "99", ""} {
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

// AND THE TABLE REDRAWS ON THE SCHEDULE'S ANSWER (D-119).
//
// HUM LEAD, 2026-09-13: "Once I hit a number and <enter> on re-ordering the
// lineup - that table DOES need to redraw/update - otherwise the UI lies to me."
//
// IT IS THE SCHEDULE'S ANSWER AND NOT THE CONSOLE'S GUESS, which is FR-3.3 and
// the whole reason these are events: the console does not reorder its own rows
// and then hope. `Reorder` moves the card, `settle` publishes the new order, and
// the table draws what it was published. A console that reordered locally would
// show a move the schedule had refused — which is the "UI lies to me" failure in
// its worse form, because it would look right.
//
// DRIVEN THROUGH THE WHOLE PATH: the operator's keys, the schedule's own
// `Reorder`, the published line-up, the rows the table builds.
func TestMovingACardRedrawsTheTable(t *testing.T) {
	r := openCard(t, 0) // the pointer is on the first row the table draws
	l := r.broadcaster.lineup
	moved := r.observer.cardID

	// THE SCHEDULE IS THE ONE THAT MOVES IT. `MoveCard` is wired to this in the
	// app; here it is wired to the schedule directly, so the test exercises the
	// same function the Director calls rather than a stand-in.
	// THE NEW ORDER IS CAPTURED, NOT ASSIGNED THROUGH THE ROUTER. `press` acts on
	// a COPY and returns it, so a closure writing to `r` mid-press has its work
	// overwritten by the value press hands back — which is a test that reports the
	// feature broken for a reason the feature has nothing to do with.
	var settled lineup.Lineup
	r.observer.cfg.MoveCard = func(id string, to int) {
		next, err := l.Reorder(id, to)
		if err != nil {
			t.Fatalf("the schedule refused (%q -> %d): %v", id, to, err)
		}
		settled = next
	}

	before := rowNumberOf(t, r.broadcaster, moved)
	r = press(t, r, "P")
	for _, k := range []string{"0", "9"} {
		r = press(t, r, k)
	}
	r = press(t, r, "enter")

	// AND THE SCHEDULE'S ANSWER REACHES THE CONSOLE THE WAY IT DOES IN THE APP:
	// as a published line-up, not as a local edit.
	r.broadcaster, _ = r.broadcaster.Update(LineupMsg{Lineup: settled})

	after := rowNumberOf(t, r.broadcaster, moved)
	if after == before {
		t.Fatalf("the card is still at position %s after being moved to 9; the table did not redraw", after)
	}
	if after != "09." {
		t.Errorf("the card is at %s; the operator moved it to 9", after)
	}
	// AND THE FRAME AGREES WITH THE ROWS, which is the claim the operator is
	// actually making: what they READ has changed.
	if !strings.Contains(stripANSITest(r.broadcaster.View().Content), "09.") {
		t.Error("position nine is not on the frame at all")
	}
}

// rowNumberOf is the `##.` cell of the row a card is drawn on.
func rowNumberOf(t *testing.T, b Broadcaster, id string) string {
	t.Helper()
	// THE SLOT, NOT THE INDEX. The table numbers its rows by SLOT and reads the
	// card at `slot - liveOffset()` (D-84), so a test that compared indices would
	// be asking a different question from the one the operator can see.
	for i, c := range b.mainTrack() { // bounded by the track (P10-02)
		if c.ID == id {
			return pad2(i + b.liveOffset())
		}
	}
	t.Fatalf("card %q is not on the main track at all", id)
	return ""
}
