package tty

// broadcaster_scroll_test.go — one scroll control over two tables, in Observer's
// own column (D-104), and the pointer as an address (D-105).

import (
	"fmt"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// manyPool is a console whose station reaches `n` candidates.
func manyPool(t *testing.T, n int) Broadcaster {
	t.Helper()
	pool := make([]snapshot.LocationRef, 0, n)
	for i := range n { // bounded by the ask (P10-02)
		pool = append(pool, snapshot.LocationRef{
			Label: fmt.Sprintf("Place %02d, CA", i+1), Zip: fmt.Sprintf("920%02d", i),
			Lat: 33.28 + float64(i)/100, Lon: -117.23, Population: 1000 * (i + 1)})
	}
	b := bcWith(t, card(t, "a", "Oceanside, CA"))
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    100, Pool: pool})
	return b
}

// consoleRouter is a Router on the console with a running order to point at.
func consoleRouter(t *testing.T) Router {
	t.Helper()
	cards := make([]lineup.Card, 0, MainTrackSlots)
	for i := range MainTrackSlots { // bounded by the track (P10-02)
		cards = append(cards, card(t, fmt.Sprintf("c%02d", i), fmt.Sprintf("Place %02d, CA", i)))
	}
	b := bcWith(t, cards...)
	b.ascii = true
	// WITH THE KEYMAP THE REAL ROUTER IS BUILT WITH. A fixture without it skips
	// the action switch entirely, so every test through it exercises the
	// fall-through and none of the bindings — which is F-72's shape ("ASSIGNED AT
	// CONSTRUCTION, and it was not") wearing a test's clothes.
	return Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster,
		keys: broadcasterKeyMap()}
}

// THE CONTROL SITS WHERE OBSERVER'S SITS (HUM LEAD, UAT 2026-09-12): "the control
// is immediately to left of the 2 col right global inset."
//
// MEASURED AGAINST OBSERVER, NOT AGAINST A NUMBER. Both surfaces are asked for
// the same terminal and their rails have to land in the same column — a constant
// here would agree with the mock and drift from the app the mock is about.
func TestTheScrollControlSitsInObserversColumn(t *testing.T) {
	b := poolConsole(t)
	b.width, b.height, b.ascii = 150, 58, true
	if got, want := railAt(b), b.width-bcRightInset-1; got != want {
		t.Errorf("the control is at %d; the two-column right margin puts it at %d", got, want)
	}
	// AND THE TABLE RUNS UP TO IT, with Observer's single blank column between
	// (UAT 9.2) and nothing else.
	if got, want := b.tableWidth(), b.frameWidth()-2; got != want {
		t.Errorf("the table is %d cells inside a %d-cell frame; it runs to the control", got, want)
	}
}

// THE CONTROL BELONGS TO WHAT SCROLLS (D-106).
//
// HUM LEAD, UAT 2026-09-12: "Location Pool Scrolls, Line-up doesnt — if the
// line-up table isnt going to scroll, then it needs to follow the Observer
// pattern where the scroll is anchored only to the location pool table, and the
// top of the vertical scroll aligns with the headers of the table (so they dont
// disappear when I scroll down)."
//
// D-104 SPANNED IT OVER BOTH on the strength of the shared pointer. The running
// order does not actually move — fifteen slots is the whole list — so the control
// was claiming a scroll that never happens, and the pool's own headings were
// inside the window it drew.
func TestTheScrollControlIsThePoolsAlone(t *testing.T) {
	b := manyPool(t, 25)
	b.width, b.height, b.ascii = 150, 58, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")

	up, down, sched, header, footer := -1, -1, -1, -1, -1
	for i, r := range rows { // bounded by the frame (P10-02)
		if at := railAt(b); len([]rune(r)) > at {
			switch []rune(r)[at] {
			case '^':
				up = i
			case 'v':
				down = i
			}
		}
		switch {
		case strings.Contains(r, "S C H E D U L E D"):
			sched = i
		case strings.Contains(r, "POPULATION"):
			header = i
		case strings.Contains(r, "Location Pool Locations"):
			footer = i
		}
	}
	if up < 0 || down < 0 || sched < 0 || header < 0 || footer < 0 {
		t.Fatalf("frame incomplete: up %d down %d sched %d header %d footer %d", up, down, sched, header, footer)
	}
	// ▲ ON THE POOL'S COLUMN TITLES and ▼ on its "Showing" line — Observer's own
	// anchoring, and the reason the titles survive a scroll.
	if up != header {
		t.Errorf("the control opens on row %d; the pool's headings are row %d", up, header)
	}
	if down != footer {
		t.Errorf("the control closes on row %d; the pool's footer is row %d", down, footer)
	}
	if up < sched {
		t.Fatal("the control opens above the running order; it belongs to the pool")
	}
	// AND THE RUNNING ORDER CARRIES NO MARK, which is what "anchored only to the
	// location pool table" means. Asked of the rows between the running order's
	// heading and the pool's, because ABOVE those the boxes' own right borders
	// legitimately stand in this column — and in the no-colour form a border and
	// a rail are the same character.
	for i := sched; i < up; i++ { // bounded by the running order (P10-02)
		if at := railAt(b); len([]rune(rows[i])) > at && []rune(rows[i])[at] != ' ' {
			t.Errorf("row %d marks the rail column over the running order:\n%s", i, rows[i])
		}
	}
}

// AND THE POOL'S HEADINGS SURVIVE A SCROLL, which is the point of anchoring it
// there: the operator moving down the list keeps the names of the columns.
func TestThePoolsHeadingsDoNotScroll(t *testing.T) {
	b := manyPool(t, 25)
	b.width, b.height, b.ascii = 150, 58, true
	for range 40 { // bounded by the walk (P10-02)
		b = b.scrollQueue(1)
	}
	got := stripANSITest(b.View().Content)
	for _, want := range []string{"L O C A T I O N     P O O L", "POPULATION", "CONDITIONS"} {
		if !strings.Contains(got, want) {
			t.Errorf("scrolled to the bottom, the pool has lost %q:\n%s", want, got)
		}
	}
	// AND IT IS ACTUALLY AT THE BOTTOM, or this proves nothing.
	if !strings.Contains(got, "of 25 Location Pool Locations") || strings.Contains(got, "Showing 1 - ") {
		t.Errorf("the pool did not scroll:\n%s", got)
	}
}

// THE POOL GETS A FLOOR, so it is not merely what the running order leaves.
func TestThePoolIsNotWhateverIsLeftOver(t *testing.T) {
	b := manyPool(t, 25)
	b.width, b.height, b.ascii = 150, 58, true
	got := stripANSITest(b.View().Content)
	if !strings.Contains(got, "010. ") {
		t.Errorf("the pool shows fewer than its ten rows:\n%s", got)
	}
	if !strings.Contains(got, "of 25 Location Pool Locations") {
		t.Error("the footer does not count the whole pool")
	}
}

// AND THE POINTER OPENS WHAT IT IS ON, at any position (D-105).
//
// TEN DIGITS CANNOT ADDRESS FIFTEEN SLOTS, which is the HUM LEAD's own reason:
// "this is more important now for positions [10-14]".
func TestEnterOpensTheRowThePointerIsOn(t *testing.T) {
	r := consoleRouter(t)
	// THE PREMISE, ASSERTED. A loop that only checks the rows that DID open
	// passes just as well on a console where none of them do — which is the shape
	// mT3 was, and the reason a test states what it needed before it reports.
	beyondADigit := 0
	for at := range MainTrackSlots - bcScheduledFrom { // bounded by the table (P10-02)
		r.broadcaster.selected = at
		out, opened := r.openPointedCard()
		if !opened {
			continue // the Director has not filled that slot; a digit refuses too
		}
		if at+bcScheduledFrom > 9 {
			beyondADigit++
		}
		if out.observer.modal != modalCard {
			t.Fatalf("row %d opened no card window", at)
		}
		want, _, ok := r.broadcaster.cardDetail(at + bcScheduledFrom)
		if !ok || out.observer.cardID != want {
			t.Errorf("row %d opened card %q, want %q", at, out.observer.cardID, want)
		}
	}
	if beyondADigit == 0 {
		t.Fatal("no slot past [9] opened, so this proves nothing about the rows that have no digit")
	}
	// AND IT REFUSES QUIETLY IN THE POOL, where that window is not built: the key
	// is not consumed, so it falls through as an unbound key does.
	r.broadcaster.selected = MainTrackSlots - bcScheduledFrom
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
// console stops owning the keys, so `enter` reached Observer — where it means
// "open the details for the row I have selected", and that row is Observer's own.
// The operator pressed enter on one location and was shown another.
func TestASecondEnterClosesTheCardWindow(t *testing.T) {
	r := consoleRouter(t)
	r.broadcaster.selected = 0

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
