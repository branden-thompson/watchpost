package tty

// broadcaster_detail_test.go — F-97, the card's own window (D-88).
//
// THE CARD'S CHIP OPENS ITS WINDOW (UAT 2026-09-11: "Pressing [1] doesn't open
// the details modal."). The chip is drawn on every card (D-52), and the card is
// a manifest that defers to the window (D-87), so a chip bound to nothing leaves
// the full read unreachable.

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// broadcasterWithOneCard is a station on standby holding one composed card —
// the fixture the memo-completeness guard opens the window with, and the one
// every test below starts from.
func broadcasterWithOneCard(t *testing.T) Broadcaster {
	t.Helper()
	b := New(Opening{})
	b, _ = b.Update(LineupMsg{Lineup: ttytest.Seed(t, ttytest.OneCard(t))})
	b.width, b.height = 150, 74
	b.now = func() time.Time { return ttytest.DetailAt().Add(2 * time.Minute) }
	return b
}

func TestTheCardWindowCarriesWhatTheCardDefersTo(t *testing.T) {
	b := broadcasterWithOneCard(t)
	_, rows, ok := b.CardDetail(1)
	if !ok {
		t.Fatalf("slot 1 holds the only card on a standby station and opened nothing")
	}
	head, body := rows(b.opts())
	if !strings.Contains(head, "LOCATION REPORT") || !strings.Contains(head, "Oceanside, CA") {
		t.Errorf("the window is titled %q; it must name the KIND and the SUBJECT, as the card does", head)
	}
	text := strings.Join(body, "\n")
	for _, want := range []string{
		"Ready for Read-Out",                     // the card's own status, same function
		"Friday September 11, 2026 @ 5:45:00 PM", // the stamp, in the default 12-hour clock (0.19.5 D-29)
		"(2 MIN AGO)",                            // and the age, which is what it is trusted on
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
	_, rows, _ := b.CardDetail(1)
	_, body := rows(b.opts())
	if !strings.Contains(strings.Join(body, "\n"), "Watchpost Air Quality Report") {
		t.Errorf("the fifth source reached neither the card nor the window, so nothing can read it")
	}
}

// AN EMPTY SLOT OPENS NOTHING, and the key is NOT consumed — it falls through
// exactly as an unbound key does.
func TestAnEmptySlotOpensNoWindow(t *testing.T) {
	if _, _, ok := broadcasterWithOneCard(t).CardDetail(7); ok {
		t.Errorf("slot 7 is undecided and opened a window onto the absence of a card")
	}
}

// THE ALERT BOX'S WINDOW IS THE ONE A DIGIT WOULD HAVE BUILT, which is the half
// an id check cannot see; the Router's [A] test is the other half (0.19.6
// D-52). Mutant mAU2 gives the hazard window a builder of its own that
// composes the same title TODAY — so the two agree by coincidence rather than
// by construction, and the next change to `cardTitle` moves one and not the
// other. The Router's test proves they are the same CARD; this proves they are
// the same WINDOW.
//
// UNDER THE WINDOW'S OPTS, NOT THE CONSOLE'S. The window is drawn by Observer,
// and here the console is built --ascii and the window is not, so a builder
// drawing in the console's own glyphs (mutant mW5) draws another title.
func TestTheAlertBoxsWindowIsTheOneADigitBuilds(t *testing.T) {
	alert := ttytest.Card(t, "brk1", "Vista, CA")
	b := bcWith(t)
	l, err := b.lineup.Queue(lineup.AlertRail, alert)
	if err != nil {
		t.Fatalf("seeding the rail: %v", err)
	}
	b, _ = b.Update(LineupMsg{Lineup: l})
	b.width, b.height, b.ascii = 150, 74, true

	o := render.Opts{Width: 38} // a bare Observer's window: its 40-cell floor less the frame, not --ascii
	_, draw, ok := b.AlertDetail()
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
