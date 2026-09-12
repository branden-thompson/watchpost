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
