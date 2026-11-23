package tty

// console_enter_test.go — enter on the console's pointer opens what it is on: a
// card's window in the running order, a place's details in the pool (D-105,
// D-109).

import (
	"fmt"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// runningOrder is a full running order to point at, a place a card.
func runningOrder(t *testing.T) lineup.Lineup {
	t.Helper()
	cards := make([]lineup.Card, 0, MainTrackSlots)
	for i := range MainTrackSlots { // bounded by the track (P10-02)
		cards = append(cards, ttytest.Card(t, fmt.Sprintf("c%02d", i), fmt.Sprintf("Place %02d, CA", i)))
	}
	return mainTrackOf(t, cards...)
}

// consoleRouter is a Router on the console with a running order to point at.
func consoleRouter(t *testing.T) Router {
	t.Helper()
	return consoleRouterOn(t, runningOrder(t))
}

// consoleRouterOn is a Router on an --ascii console holding the line-up l.
func consoleRouterOn(t *testing.T, l lineup.Lineup) Router {
	t.Helper()
	b := consoleOf(t, Opening{ASCII: true}, l)
	// WITH THE KEYMAP THE REAL ROUTER IS BUILT WITH. A fixture without it skips
	// the action switch entirely, so every test through it exercises the
	// fall-through and none of the bindings — which is F-72's shape (a field
	// ASSIGNED AT CONSTRUCTION that is not) wearing a test's clothes.
	return Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster,
		keys: broadcasterKeyMap()}
}

// The pointer opens what it is on, at any position.
//
// TEN DIGITS CANNOT ADDRESS FIFTEEN SLOTS, which is the HUM LEAD's own reason:
// "this is more important now for positions [10-14]".
func TestEnterOpensTheRowThePointerIsOn(t *testing.T) {
	r := consoleRouter(t)
	// THE PREMISE, ASSERTED. A loop that only checks the rows that DID open
	// passes just as well on a console where none of them do — so a test states
	// what it needed before it reports.
	beyondADigit := 0
	for at := range MainTrackSlots - ScheduledFrom { // bounded by the table (P10-02)
		r.broadcaster = pointAt(t, r.broadcaster, at)
		out, opened := r.openPointedCard()
		if !opened {
			continue // the Director has not filled that slot; a digit refuses too
		}
		if at+ScheduledFrom > 9 {
			beyondADigit++
		}
		if out.observer.modal != modalCard {
			t.Fatalf("row %d opened no card window", at)
		}
		want, _, ok := r.broadcaster.CardDetail(at + ScheduledFrom)
		if !ok || out.observer.cardID != want {
			t.Errorf("row %d opened card %q, want %q", at, out.observer.cardID, want)
		}
	}
	if beyondADigit == 0 {
		t.Fatal("no slot past [9] opened, so this proves nothing about the rows that have no digit")
	}
	// The key refuses quietly in the pool, where that window is not built: the key
	// is not consumed, so it falls through as an unbound key does. The pointer
	// reaches the pool's first row while it holds one, and the pool then empties
	// under it, as a narrowed service area empties it.
	r.broadcaster, _ = r.broadcaster.Update(StationAreaMsg{Pool: []snapshot.LocationRef{{Label: "Vista, CA"}}})
	r.broadcaster = pointAt(t, r.broadcaster, MainTrackSlots-ScheduledFrom)
	r.broadcaster, _ = r.broadcaster.Update(StationAreaMsg{})
	if _, opened := r.openPointedCard(); opened {
		t.Error("enter opened a card window from the pool")
	}
}

// A SECOND PRESS CLOSES THE WINDOW THE FIRST ONE OPENED (D-109).
//
// HUM LEAD, UAT 2026-09-12: "<enter> (again) flows to Oceanside, CA location card
// from Observer … hitting <enter> a 2nd time should just close that modal for now
// (until we write the management controls)."
//
// THE CARD WINDOW IS THE CONSOLE'S AND OBSERVER ONLY DRAWS IT. With one open the
// console stops owning the keys, so an `enter` let through reaches Observer —
// where it means "open the details for the row I have selected", and that row
// is Observer's own: the operator presses enter on one location and is shown
// another.
func TestASecondEnterClosesTheCardWindow(t *testing.T) {
	r := consoleRouter(t)
	r.broadcaster = pointAt(t, r.broadcaster, 0)

	opened, ok := r.openPointedCard()
	if !ok || opened.observer.modal != modalCard {
		t.Fatal("the first press opened no card window")
	}
	// THE PREMISE: Observer has a selection of its own, and it is not the row the
	// console's pointer is on. Without that this test cannot see the defect.
	opened.observer.selected = 3

	closed := pressAction(t, opened, actQueueOpen)
	if closed.observer.modal != modalNone {
		t.Errorf("a second enter left the window %v open", closed.observer.modal)
	}
	// AND IT DID NOT REACH OBSERVER, which is the defect itself: a modal of
	// Observer's own would mean the key crossed the surface.
	if closed.observer.modal == modalDetails {
		t.Error("the key fell through to Observer and opened its own location details")
	}
	if closed.active != SurfaceBroadcaster {
		t.Error("and the operator is still on the console")
	}
}

// A pool row opens the location's details.
//
// HUM LEAD, UAT 2026-09-12: "make it so <enter> on a location pool row opens the
// location details (like observer)."
//
// ONE KEY, WHATEVER THE POINTER IS ON. The pointer walks the running order and
// the pool as one list, so `enter` opens a card above and a place below —
// never OBSERVER'S OWN selection.
func TestEnterOnAPoolRowOpensThatLocation(t *testing.T) {
	r := consoleRouter(t)
	pool := []snapshot.LocationRef{
		{Label: "Fallbrook, CA", Zip: "92028", Lat: 33.37, Lon: -117.25},
		{Label: "Vista, CA", Zip: "92081", Lat: 33.20, Lon: -117.24},
	}
	r.broadcaster, _ = r.broadcaster.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50, Pool: pool})

	for want, at := range map[string]int{"Fallbrook, CA": 0, "Vista, CA": 1} {
		r.broadcaster = pointAt(t, r.broadcaster, MainTrackSlots-ScheduledFrom+at)
		out, opened := r.openPointedCard()
		if !opened {
			t.Fatalf("%s: enter opened nothing on a pool row", want)
		}
		if out.observer.modal != modalDetails {
			t.Errorf("%s: enter opened %v, not the location details", want, out.observer.modal)
		}
		// THE LOCATION THE POINTER IS ON, not Observer's own selected row.
		if out.observer.lookupRef == nil || out.observer.lookupRef.Label != want {
			t.Errorf("the details are about %v, want %q", out.observer.lookupRef, want)
		}
	}
	// A second press closes the window, the same rule the card window follows.
	r.broadcaster = pointAt(t, r.broadcaster, MainTrackSlots-ScheduledFrom)
	opened, _ := r.openPointedCard()
	if closed := pressAction(t, opened, actQueueOpen); closed.observer.modal != modalNone {
		t.Errorf("a second enter left the location details %v open", closed.observer.modal)
	}
}
