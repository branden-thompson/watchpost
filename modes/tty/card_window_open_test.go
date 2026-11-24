package tty

// card_window_open_test.go — the console's ways into the card window, driven
// through the Router the way a terminal drives it: a slot's digit, the alert
// box's [A], and the console's two drawn controls (D-88, D-126, D-102). What the
// console draws in the window is its own detail tests' (0.19.6 D-52).

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/branden-thompson/watchpost/modes/tty/console"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// routerOnConsole is the Router with the console active and one composed card.
func routerOnConsole(t *testing.T) Router {
	t.Helper()
	d := goldenDash(t, false)
	d.now = func() time.Time { return ttytest.DetailAt().Add(2 * time.Minute) } // the console takes the Dashboard's clock (0.19.6 D-21)
	r := consoleWith(t, d)
	m, _ := r.Update(console.LineupMsg{Lineup: ttytest.Seed(t, ttytest.OneCard(t))})
	return m.(Router)
}

// THE KEY, DRIVEN THROUGH THE REAL KEY PATH. A test that calls CardDetail
// directly passes whether or not the key is bound: the BINDING is the half that
// can go missing, not the body.
func TestPressingTheSlotNumberOpensTheWindow(t *testing.T) {
	r := routerOnConsole(t)
	if r.observer.ModalOpen() {
		t.Fatalf("a window is open before the key; this test would measure nothing")
	}
	m, _ := r.Update(keyFor(t, "1"))
	out := m.(Router)
	if out.observer.modal != modalCard {
		t.Fatalf("[1] left modal=%v; the chip on the card says Report Details and must open one", out.observer.modal)
	}
	if !strings.Contains(out.View().Content, "FULL READ") {
		t.Errorf("the window is open and its body is not composited over the console")
	}
	// AND ESC CLOSES IT, through the same path every other window's esc takes.
	m, _ = out.Update(keyFor(t, "esc"))
	if back := m.(Router); back.observer.ModalOpen() {
		t.Errorf("esc left the window open; a window with no way out is a trap (FR-5)")
	}
}

// A DIGIT WITH NO CARD BEHIND IT IS NOT CONSUMED.
func TestAnEmptySlotNumberOpensNothing(t *testing.T) {
	m, _ := routerOnConsole(t).Update(keyFor(t, "7"))
	if out := m.(Router); out.observer.ModalOpen() {
		t.Errorf("[7] addresses an undecided slot and opened %v", out.observer.modal)
	}
}

// THE WINDOW FOLLOWS THE CARD, NOT THE SLOT (D-88). A card promoted while the
// operator reads it must not become the card that took its place.
func TestTheOpenWindowRefreshesWithTheCardAndNotWithTheSlot(t *testing.T) {
	r := routerOnConsole(t)
	m, _ := r.Update(keyFor(t, "1"))
	r = m.(Router)
	gen := r.observer.cardGen

	// A SECOND CARD AHEAD OF IT, so the one being read moves from slot 1 to 2 —
	// AND the card itself re-hydrates in place: same card, fresher data.
	bA, bC := ttytest.ComposedCard(t, "b", "Bonsall, CA", ttytest.FiveSources(), ttytest.DetailAt())
	aA, aC := ttytest.ComposedCard(t, "a", "Oceanside, CA",
		[]lineup.Content{{Name: "NWS Weather Forecast", Detail: "09/12 - 09/18"}}, ttytest.DetailAt().Add(time.Hour))
	m, _ = r.Update(console.LineupMsg{Lineup: ttytest.Seed(t, [2]lineup.Card{bA, bC}, [2]lineup.Card{aA, aC})})
	out := m.(Router)

	if out.observer.cardID != "a" {
		t.Fatalf("the window followed the SLOT and is now open on %q; the operator was reading card a",
			out.observer.cardID)
	}
	body := strings.Join(out.observer.cardLines(out.observer.opts()), "\n")
	if !strings.Contains(body, "09/12 - 09/18") {
		t.Errorf("the card re-hydrated and the open window kept its first frame (F-30):\n%s", body)
	}
	if out.observer.cardGen == gen {
		t.Errorf("the body changed and the generation did not; the memo will replay the stale frame")
	}
}

// AND A REFRESH THAT CHANGES NOTHING MUST NOT MOVE THE GENERATION, or the memo
// is defeated for as long as the window is open.
func TestAnUnchangedCardDoesNotMoveTheGeneration(t *testing.T) {
	r := routerOnConsole(t)
	m, _ := r.Update(keyFor(t, "1"))
	r = m.(Router)
	gen := r.observer.cardGen
	for range 3 {
		m, _ = r.Update(tea.WindowSizeMsg{Width: 150, Height: 74})
		r = m.(Router)
	}
	if r.observer.cardGen != gen {
		t.Errorf("three updates with an unchanged card moved the generation %d -> %d; "+
			"the window would re-render every frame it is open", gen, r.observer.cardGen)
	}
}

// TestTheAlertBoxsWayInOpensThePriorityCard.
//
// HUM LEAD, UAT 2026-09-13: "[A] Details / Full Read / Manage in the alert window
// doesn't currently work, it should function just like [1] in the Up Next card."
//
// A DRAWN CONTROL MUST BE BOUND. `burstBody` appends
// "[A]  Details / Full Read / Manage" at the bottom of the takeover box, so a
// handler must take the `A` or the box advertises a way in that does not
// exist. A painted control that does nothing is worse than an absent one: the
// operator reasonably concludes the report has nothing to show.
//
// THE SAME DOOR AS [1], which is the whole of the ruling. `cardWindowFor` is the
// one builder of a card window; the digit reaches it through the main track and
// `A` reaches it through the alert rail, so the two windows cannot come to
// disagree about what a card looks like. That the window is the one a digit
// builds is the console's own detail tests' (0.19.6 D-52).
func TestTheAlertBoxsWayInOpensThePriorityCard(t *testing.T) {
	alert := ttytest.Card(t, "brk1", "Vista, CA")
	l, err := lineup.Lineup{}.Queue(lineup.AlertRail, alert)
	if err != nil {
		t.Fatalf("seeding the rail: %v", err)
	}
	b := consoleOf(t, console.Opening{ASCII: true}, l)

	// THE PREMISE: the box actually draws the control this test is about.
	if frame := ttytest.StripANSI(b.View().Content); !strings.Contains(frame, "Details / Full Read / Manage") {
		t.Fatal("the alert box does not draw its way in; this test would prove nothing")
	}

	// THROUGH `Update`, THE WAY A TERMINAL DOES. Driving `openCardWindow`
	// directly calls the handler, and the operator presses a KEY (P-1): a test
	// that skips the key passes against a build whose controls do nothing
	// (D-121).
	r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	out := press(t, r, "A")
	if out.observer.modal != modalCard {
		t.Fatalf("[A] opened %v, want the card window — the control the box draws does nothing", out.observer.modal)
	}
	if out.observer.cardID != alert.ID {
		t.Errorf("[A] opened card %q, want the rail's own %q", out.observer.cardID, alert.ID)
	}

}

// AND WITH NO HAZARD THERE IS NOTHING TO OPEN.
//
// It refuses QUIETLY, exactly as a digit on an empty slot does and for the same
// reason: `false` means the key was not consumed, so it falls through rather
// than opening a window onto a report that does not exist.
func TestTheAlertBoxsWayInRefusesWhenTheRailIsEmpty(t *testing.T) {
	b := consoleOf(t, console.Opening{ASCII: true}, lineup.Lineup{})
	r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	if out := press(t, r, "A"); out.observer.ModalOpen() {
		t.Errorf("[A] opened %v with no hazard on the rail", out.observer.modal)
	}
}

// TestTheConsolesTwoDrawnControlsAreBound.
//
// `[l] Lookup Location from Pool` AND `[r] Request for Line-Up` are DRAWN in
// the console's control row (D-102), so both must be bound (R4b).
// HUM LEAD, 2026-09-14: "i'll need r and l to be bound in broadcaster before I
// can UAT."
//
// THROUGH `Update`, THE WAY A TERMINAL DOES — a test that calls the handler
// passes against a build whose controls do nothing (D-121).
func TestTheConsolesTwoDrawnControlsAreBound(t *testing.T) {
	// THE CARD'S SUBJECT IS THE POOL ENTRY'S KEY, which is how the running order
	// finds the weather for a card — so the fixture uses the same key rather
	// than a label, or `[l]` has nothing to resolve from the line-up.
	vista := snapshot.LocationRef{Label: "Vista, CA", Zip: "92084", Lat: 33.2, Lon: -117.24}
	// ENOUGH CARDS THAT A SCHEDULED ROW EXISTS. The line-up pointer addresses
	// positions 2..15 — LIVE and UP NEXT are read FROM, not navigated to — so a
	// fixture with one card leaves the pointer on an empty slot and every
	// control refuses, correctly and uninformatively.
	// AND THE POOL-KEYED CARD IS THE SECOND ONE, because at standby the line-up
	// draws from UP NEXT down (D-84's `liveOffset`): the pointer's position 0 is
	// slot 2, which is the second card. It is a FIXTURE fact rather than a
	// behaviour, and the same off-by-one can put a move one slot low.
	b := consoleOf(t, console.Opening{ASCII: true}, mainTrackOf(t,
		ttytest.Card(t, "c0", "first"),
		ttytest.Card(t, "c1", string(snapshot.Key(vista))),
		ttytest.Card(t, "c2", "third")))
	b, _ = b.Update(console.StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50,
		Pool:        []snapshot.LocationRef{vista},
	})

	// THE PREMISE: the row really does draw both, or this proves nothing.
	frame := ttytest.StripANSI(b.View().Content)
	for _, cap := range []string{"Lookup Location from Pool", "Request for Line-Up"} {
		if !strings.Contains(frame, cap) {
			t.Fatalf("the control row does not draw %q; this test measures nothing", cap)
		}
	}

	// A REAL OBSERVER BEHIND THE CONSOLE, because `[l]` is FORWARDED to it and a
	// zero Dashboard has no keymap to forward into — the fixture would report
	// the key as dead when the routing is fine.
	obs, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	r := Router{observer: obs, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	if out := press(t, r, "r"); out.observer.modal != modalRequest {
		t.Errorf("[r] opened %v, want the Line-Up Request window", out.observer.modal)
	}
	// `[l]` IS THE LOCATION SEARCH BOX, NOT THE POINTED ROW'S DETAILS.
	//
	// HUM LEAD, UAT 2026-09-14: "the <l> press is in Broadcaster UI EXPECTING the
	// location search modal, not the line-up location details modal." The label
	// "Lookup Location from Pool" does not mean "open the location the pointer is
	// on": it is a LOOKUP, which is a search.
	//
	// IT IS OBSERVER'S OWN WINDOW, forwarded rather than reimplemented, so there
	// is one search box with one behaviour.
	if out := press(t, r, "l"); out.observer.modal != modalAdd {
		t.Errorf("[l] opened %v, want the location search box", out.observer.modal)
	}
	if out := press(t, r, "l"); out.observer.addMode != "lookup" {
		t.Errorf("[l] opened the search box in %q mode, want lookup", out.observer.addMode)
	}
}

// AND `a` STAYS ABOUT ON BOTH SURFACES. `[r]` and `[l]` are the console's own,
// but About is not — taking a key that already works would be a regression
// dressed as a feature (D-56: one key, one meaning PER SURFACE).
func TestTheConsolesNewKeysDoNotTakeAboutsKey(t *testing.T) {
	b := consoleOf(t, console.Opening{ASCII: true}, lineup.Lineup{})
	r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	if out := press(t, r, "a"); out.observer.modal == modalRequest {
		t.Error("[a] opened the request window; it is About on both surfaces")
	}
}
