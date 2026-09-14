package tty

// broadcaster_detail_test.go — F-97, the card's own window (D-88).
//
// THE DEFECT THIS RETIRES, reported in UAT 2026-09-11: "Pressing [1] doesn't open
// the details modal." The chip had been drawn on every card since D-52 and bound
// to nothing, and D-87 made that load-bearing by turning the card into a manifest
// that defers to it.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// bcDetailAt is a fixed moment, so the window's age line is a fact rather than
// a race.
var bcDetailAt = time.Date(2026, 9, 11, 17, 45, 0, 0, time.UTC)

// composedCard is a card the Composer has finished with: words, a manifest and
// a stamp — and the ADMITTED card it grew from, because the lineup takes only
// those.
//
// THE REAL CHAIN, NOT A STRUCT LITERAL. `Director.onBuilt` queues an admitted
// card and `Set`s the composed one over it; a fixture that assembled the end
// state by hand would keep passing on the day an invariant along that chain
// started refusing it.
func composedCard(t *testing.T, id, subject string, contents []lineup.Content, at time.Time) (lineup.Card, lineup.Card) {
	t.Helper()
	base := card(t, id, subject)
	c, err := base.To(lineup.Standby)
	if err != nil {
		t.Fatalf("standby %s: %v", id, err)
	}
	c, err = c.WithScript(
		lineup.Script{Parts: []lineup.Part{
			{Kind: lineup.PartLine, Text: "The forecast for " + subject + " calls for sun."},
			{Kind: lineup.PartLine, Text: "Seas are two to three feet."},
		}},
		contents, at,
	)
	if err != nil {
		t.Fatalf("composing %s: %v", id, err)
	}
	return base, c
}

// fiveSources is a manifest one row LONGER than the card can draw.
//
// bcReadLines IS FOUR, so the fifth row exists only in the window — which is the
// claim the window is for, and a fixture of four could not tell the two apart.
func fiveSources() []lineup.Content {
	return []lineup.Content{
		{Name: "NWS Weather Forecast", Detail: "09/11 - 09/17"},
		{Name: "Watchpost Marine Report", Detail: "09/11 - 09/13"},
		{Name: "Watchpost Fire Report", Detail: "3 hotspots / 10 incidents"},
		{Name: "Watchpost Seismic Report", Detail: "2 quakes"},
		{Name: "Watchpost Air Quality Report", Detail: "AQI 41"},
	}
}

// seed is a lineup holding composed cards, queued and Set exactly as the
// Director does it.
func seed(t *testing.T, pairs ...[2]lineup.Card) lineup.Lineup {
	t.Helper()
	var l lineup.Lineup
	for _, p := range pairs {
		next, err := l.Queue(lineup.MainTrack, p[0])
		if err != nil {
			t.Fatalf("queueing %s: %v", p[0].ID, err)
		}
		if next, err = next.Set(p[1]); err != nil {
			t.Fatalf("setting %s: %v", p[1].ID, err)
		}
		l = next
	}
	return l
}

// oneCard is the fixture every test below starts from: one composed card with
// five sources.
func oneCard(t *testing.T) [2]lineup.Card {
	t.Helper()
	a, c := composedCard(t, "a", "Oceanside, CA", fiveSources(), bcDetailAt)
	return [2]lineup.Card{a, c}
}

// broadcasterWithOneCard is a station on standby holding one composed card —
// the fixture the memo-completeness guard opens the window with, and the one
// every test below starts from.
func broadcasterWithOneCard(t *testing.T) Broadcaster {
	t.Helper()
	b := NewBroadcaster()
	b, _ = b.Update(LineupMsg{Lineup: seed(t, oneCard(t))})
	b.width, b.height = 150, 74
	b.now = func() time.Time { return bcDetailAt.Add(2 * time.Minute) }
	return b
}

func TestTheCardWindowCarriesWhatTheCardDefersTo(t *testing.T) {
	b := broadcasterWithOneCard(t)
	_, rows, ok := b.cardDetail(1)
	if !ok {
		t.Fatalf("slot 1 holds the only card on a standby station and opened nothing")
	}
	head, body := rows(b.opts())
	if !strings.Contains(head, "LOCATION REPORT") || !strings.Contains(head, "Oceanside, CA") {
		t.Errorf("the window is titled %q; it must name the KIND and the SUBJECT, as the card does", head)
	}
	text := strings.Join(body, "\n")
	for _, want := range []string{
		"Ready for Read-Out",                // the card's own status, same function
		"Friday September 11, 2026 @ 17:45", // the stamp
		"(2 MIN AGO)",                       // and the age, which is what it is trusted on
		"READ CONTENTS",
		"Watchpost Air Quality Report", // THE FIFTH ROW — the card cannot show it
		"FULL READ",
		"The forecast for Oceanside, CA calls for sun.",
		"Seas are two to three feet.",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the window does not carry %q:\n%s", want, text)
		}
	}
}

// THE WINDOW IS WHERE THE WHOLE READ LIVES, and the card is where four lines of
// it do. Asserting the difference is asserting the drill-down.
func TestTheCardShowsFourSourcesAndTheWindowShowsThemAll(t *testing.T) {
	b := broadcasterWithOneCard(t)
	if got := b.View().Content; strings.Contains(got, "Watchpost Air Quality Report") {
		t.Errorf("the fifth source reached the CARD; bcReadLines is %d and the card's height is fixed", bcReadLines)
	}
	_, rows, _ := b.cardDetail(1)
	_, body := rows(b.opts())
	if !strings.Contains(strings.Join(body, "\n"), "Watchpost Air Quality Report") {
		t.Errorf("the fifth source reached neither the card nor the window, so nothing can read it")
	}
}

// AN EMPTY SLOT OPENS NOTHING, and the key is NOT consumed — it falls through
// exactly as an unbound key does.
func TestAnEmptySlotOpensNoWindow(t *testing.T) {
	if _, _, ok := broadcasterWithOneCard(t).cardDetail(7); ok {
		t.Errorf("slot 7 is undecided and opened a window onto the absence of a card")
	}
}

// routerOnConsole is the Router with the console active and one composed card.
func routerOnConsole(t *testing.T) Router {
	t.Helper()
	r := consoleWith(t, goldenDash(t, false))
	m, _ := r.Update(LineupMsg{Lineup: seed(t, oneCard(t))})
	r = m.(Router)
	r.broadcaster.now = func() time.Time { return bcDetailAt.Add(2 * time.Minute) }
	return r
}

// THE DEFECT, DRIVEN THROUGH THE REAL KEY PATH. A test that called cardDetail
// directly would have passed on the broken build: the body was never the missing
// half, the BINDING was.
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
	bA, bC := composedCard(t, "b", "Bonsall, CA", fiveSources(), bcDetailAt)
	aA, aC := composedCard(t, "a", "Oceanside, CA",
		[]lineup.Content{{Name: "NWS Weather Forecast", Detail: "09/12 - 09/18"}}, bcDetailAt.Add(time.Hour))
	m, _ = r.Update(LineupMsg{Lineup: seed(t, [2]lineup.Card{bA, bC}, [2]lineup.Card{aA, aC})})
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
// THE CONTROL WAS DRAWN AND NOTHING BOUND IT. `burstBody` appends
// "[A]  Details / Full Read / Manage" at the bottom of the takeover box, and no
// handler anywhere took an `A` — so the box advertised a way in that did not
// exist. A painted control that does nothing is worse than an absent one: the
// operator reasonably concludes the report has nothing to show.
//
// THE SAME DOOR AS [1], which is the whole of the ruling. `cardWindowFor` is the
// one builder of a card window; the digit reaches it through the main track and
// `A` reaches it through the alert rail, so the two windows cannot come to
// disagree about what a card looks like.
func TestTheAlertBoxsWayInOpensThePriorityCard(t *testing.T) {
	alert := card(t, "brk1", "Vista, CA")
	b := bcWith(t)
	l, err := b.lineup.Queue(lineup.AlertRail, alert)
	if err != nil {
		t.Fatalf("seeding the rail: %v", err)
	}
	b, _ = b.Update(LineupMsg{Lineup: l})
	b.width, b.height, b.ascii = 150, 74, true

	// THE PREMISE: the box actually draws the control this test is about.
	if frame := stripANSITest(b.View().Content); !strings.Contains(frame, "Details / Full Read / Manage") {
		t.Fatal("the alert box does not draw its way in; this test would prove nothing")
	}

	// THROUGH `Update`, THE WAY A TERMINAL DOES. Driving `openCardWindow`
	// directly is the seam D-121 was caught on this morning: twelve tests passed
	// against a build whose controls did nothing, because they called the handler
	// and the operator presses a KEY (P-1).
	r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	out := press(t, r, "A")
	if out.observer.modal != modalCard {
		t.Fatalf("[A] opened %v, want the card window — the control the box draws does nothing", out.observer.modal)
	}
	if out.observer.cardID != alert.ID {
		t.Errorf("[A] opened card %q, want the rail's own %q", out.observer.cardID, alert.ID)
	}

	// AND THE WINDOW IS THE ONE A DIGIT WOULD HAVE BUILT, which is the half an
	// id check cannot see. Mutant mAU2 gives the hazard window a builder of its
	// own that composes the same title TODAY — so the two agree by coincidence
	// rather than by construction, and the next change to `cardTitle` moves one
	// and not the other. Comparing the id proved they were the same CARD; this
	// proves they are the same WINDOW.
	o := out.observer.opts()
	_, draw, ok := b.alertDetail()
	if !ok {
		t.Fatal("there is no hazard window to compare")
	}
	gotTitle, gotBody := draw(o)
	wantTitle, wantBody := cardTitle(alert, o.Glyphs()), b.detailBody(o, alert)
	if gotTitle != wantTitle {
		t.Errorf("the hazard window's title is %q, the one builder says %q", gotTitle, wantTitle)
	}
	if len(gotBody) != len(wantBody) {
		t.Errorf("the hazard window draws %d rows, the one builder draws %d", len(gotBody), len(wantBody))
	}
}

// AND WITH NO HAZARD THERE IS NOTHING TO OPEN.
//
// It refuses QUIETLY, exactly as a digit on an empty slot does and for the same
// reason: `false` means the key was not consumed, so it falls through rather
// than opening a window onto a report that does not exist.
func TestTheAlertBoxsWayInRefusesWhenTheRailIsEmpty(t *testing.T) {
	b := bcWith(t)
	b.width, b.height, b.ascii = 150, 74, true
	r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	if out := press(t, r, "A"); out.observer.ModalOpen() {
		t.Errorf("[A] opened %v with no hazard on the rail", out.observer.modal)
	}
}

// TestTheConsolesTwoDrawnControlsAreBound.
//
// `[l] Lookup Location from Pool` AND `[r] Request for Line-Up` have been DRAWN
// in the console's control row since D-102 with nothing bound to either (R4b).
// HUM LEAD, 2026-09-14: "i'll need r and l to be bound in broadcaster before I
// can UAT."
//
// THROUGH `Update`, THE WAY A TERMINAL DOES — the seam D-121 was caught on, and
// the reason twelve tests once passed against a build whose controls did nothing.
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
	// slot 2, which is the second card. The same off-by-one that once put a move
	// one slot low, and it is a FIXTURE fact rather than a behaviour — which is
	// why it is written down here instead of being discovered again.
	b := bcWith(t,
		card(t, "c0", "first"),
		card(t, "c1", string(snapshot.Key(vista))),
		card(t, "c2", "third"))
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50,
		Pool:        []snapshot.LocationRef{vista},
	})

	// THE PREMISE: the row really does draw both, or this proves nothing.
	frame := stripANSITest(b.View().Content)
	for _, cap := range []string{"Lookup Location from Pool", "Request for Line-Up"} {
		if !strings.Contains(frame, cap) {
			t.Fatalf("the control row does not draw %q; this test measures nothing", cap)
		}
	}

	r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	if out := press(t, r, "r"); out.observer.modal != modalRequest {
		t.Errorf("[r] opened %v, want the Line-Up Request window", out.observer.modal)
	}
	if out := press(t, r, "l"); out.observer.modal != modalDetails {
		t.Errorf("[l] opened %v, want the location's details", out.observer.modal)
	}
}

// AND `a` STAYS ABOUT ON BOTH SURFACES. `[r]` and `[l]` are the console's own,
// but About is not — taking a key that already works would be a regression
// dressed as a feature (D-56: one key, one meaning PER SURFACE).
func TestTheConsolesNewKeysDoNotTakeAboutsKey(t *testing.T) {
	b := bcWith(t)
	b.width, b.height, b.ascii = 150, 74, true
	r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	if out := press(t, r, "a"); out.observer.modal == modalRequest {
		t.Error("[a] opened the request window; it is About on both surfaces")
	}
}
