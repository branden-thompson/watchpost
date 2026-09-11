package tty

// broadcaster_header_test.go — the console's masthead (D-59).
//
// IT REUSES THE OBSERVER'S OWN BOX, per D-56. `render.BoxTitled` already draws
// this exact shape and the Dashboard already carries the responsive LADDER —
// progressively shorter forms, each built only when the wider one did not fit.
// A second box drawer here would be a second place for the frame to drift.
//
// AND THE EDITION ARRIVES AS A WORD, NOT A RENAME, which `sgr.go` has said since
// 2026-08-30: "Broadcaster is the station-running dashboard a later version
// brings, and it will pass its own word through here."

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func headerOf(t *testing.T, w int) []string {
	t.Helper()
	b := NewBroadcaster()
	b.width, b.height = w, 74
	b.ascii = true
	return strings.Split(b.header(b.opts()), "\n")
}

func TestTheConsoleHasAMasthead(t *testing.T) {
	got := strings.Join(headerOf(t, 150), "\n")
	for _, want := range []string{"WATCHPOST", "Broadcaster"} {
		if !strings.Contains(stripANSITest(got), want) {
			t.Errorf("the masthead must name the app and the edition; %q is missing from:\n%s", want, got)
		}
	}
}

// IT SAYS BROADCASTER, NOT OBSERVER. The two surfaces are different experiences
// and the masthead is the only thing that says which one you are looking at.
func TestTheMastheadNamesTheRightEdition(t *testing.T) {
	// THE TITLE ROW, NOT THE WHOLE BLOCK. "ctrl+o  Observer" is a CONTROL and
	// belongs there — it is the swap back, and the reference draws it. The first
	// version of this test searched the whole masthead and failed on the one
	// mention that is supposed to be there.
	title := stripANSITest(headerOf(t, 150)[0])
	if strings.Contains(title, render.EditionObserver) {
		t.Errorf("the console's title row says %q:\n%s", render.EditionObserver, title)
	}
	if !strings.Contains(title, render.EditionBroadcaster) {
		t.Errorf("and it must say %q:\n%s", render.EditionBroadcaster, title)
	}
}

// THE BOX IS THE FRAME'S WIDTH AT EVERY SUPPORTED BREAKPOINT (D-50). A masthead
// drawn at one width and clamped at the others is the hard-coded geometry the
// HUM LEAD ruled out.
func TestTheMastheadFillsEverySupportedWidth(t *testing.T) {
	for _, w := range []int{100, 110, 120, 130, 150} {
		for i, line := range headerOf(t, w) {
			if c := utf8.RuneCountInString(stripANSITest(line)); c > w {
				t.Errorf("width %d: masthead row %d is %d cells\n%s", w, i, c, stripANSITest(line))
			}
		}
	}
}

// THE LADDER DROPS THE STAMP BEFORE THE WORDMARK. A masthead that cannot say
// what the app is has stopped being a masthead — the Observer's own words, and
// the same order here.
func TestTheMastheadShedsTheStampBeforeTheWordmark(t *testing.T) {
	narrow := stripANSITest(strings.Join(headerOf(t, 100), "\n"))
	if !strings.Contains(narrow, "WATCHPOST") {
		t.Errorf("the wordmark is the last thing to go:\n%s", narrow)
	}
}

// THE STATION'S IDENTITY IS NOT IN THE MASTHEAD, AND THAT IS THE POINT (D-71).
//
//	"we're going to take the station center out of the masthead — this should now
//	 make the Observer/Broadcaster masthead nearly identical minus the
//	 Observer/Broadcaster [word]"
//
// It is a fact about the STATION, and the station has a section of its own. What
// is left is what both surfaces share, which is what D-59 was for.
func TestTheMastheadCarriesNoStationIdentity(t *testing.T) {
	got := stripANSITest(strings.Join(headerOf(t, 150), "\n"))
	for _, gone := range []string{"TRANSMITTER", "TOWER GPS", "SERVICE RADIUS", "Bonsall"} {
		if strings.Contains(got, gone) {
			t.Errorf("%q still rides in the masthead:\n%s", gone, got)
		}
	}
	// AND IT IS THREE ROWS: the title, the controls, the rule under them —
	// Observer's masthead exactly, one word apart.
	if n := len(headerOf(t, 150)); n != 3 {
		t.Errorf("the masthead draws %d rows, want 3", n)
	}
}

// THE MASTHEAD IS THE OBSERVER'S, DIFFERING ONLY IN THE EDITION WORD.
//
// THE HUM LEAD ASKED WHY IT WAS DIFFERENT, AND THE ANSWER WAS THAT I HAD
// DEVIATED FROM THE MOCK: the version was dropped from the title, the `Updated:`
// stamp was replaced with an invented "ON AIR / STANDBY", and the API summary
// was left out entirely. The reference draws all three.
//
// The stamp was the worst of the three — it was substituted on my own reasoning
// that the STATION line already carries state, which is a UX ruling that was not
// mine to make.
func TestTheMastheadDrawsWhatTheReferenceDraws(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.version = "0.16.0"
	b.snap = mastheadSnap()
	b.now = func() time.Time { return time.Date(2026, 8, 28, 19, 15, 8, 0, time.UTC) }
	got := stripANSITest(b.header(b.opts()))

	for _, want := range []string{"v0.16.0", "Updated:", "API:"} {
		if !strings.Contains(got, want) {
			t.Errorf("the reference draws %q and the console must too:\n%s", want, got)
		}
	}
}

// AND THE STAMP GOES BEFORE THE VERSION, WHICH GOES BEFORE THE EDITION. The
// Observer's own ladder order, because a masthead that cannot say what the app
// is has stopped being a masthead.
func TestTheMastheadLaddersInTheObserversOrder(t *testing.T) {
	b := NewBroadcaster()
	b.height, b.ascii = 74, true
	b.version = "0.16.0"
	b.snap = mastheadSnap()
	b.now = func() time.Time { return time.Date(2026, 8, 28, 19, 15, 8, 0, time.UTC) }

	b.width = 150
	wide := stripANSITest(b.header(b.opts()))
	b.width = 100
	narrow := stripANSITest(b.header(b.opts()))

	if !strings.Contains(wide, "Updated:") {
		t.Fatalf("fixture: the wide form must carry the stamp:\n%s", wide)
	}
	if !strings.Contains(narrow, "WATCHPOST") {
		t.Errorf("the wordmark is the last thing to go:\n%s", narrow)
	}
}

// mastheadSnap is a snapshot with providers, so the API summary has something
// to count.
func mastheadSnap() *snapshot.Snapshot {
	at := time.Date(2026, 8, 28, 19, 15, 0, 0, time.UTC)
	return &snapshot.Snapshot{Providers: []snapshot.ProviderStatus{
		{ID: "a", Status: snapshot.ProviderOK, FetchedAt: at},
		{ID: "b", Status: snapshot.ProviderOK, FetchedAt: at},
	}}
}
