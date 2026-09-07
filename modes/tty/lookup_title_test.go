package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// F-42 — DETAILS OPENED FROM A LOOKUP NAMES THE PLACE FROM THE FIRST FRAME.
//
// lookupIndex is "-1 while it waits" for the rebuilt RECENT list to carry the
// looked-up row, so selectedLocation has nothing to return until then and this
// window titled itself the literal "Location" — about one run in three, caught
// by the PTY journey on a step named for the property that was not holding.
//
// The ref is right there: lookupRef is "the location a lookup opened Details
// for, until its data lands", added when the modal used to show the old top
// RECENT row instead. The title just never consulted it.
func TestDetailsFromALookupNamesThePlaceBeforeItsDataLands(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	d := dash(t).(Dashboard)
	d.modal = modalDetails
	// The state during the wait: a lookup is pending, and the selection points
	// past every drawn row, so selectedLocation() is nil.
	d.lookupRef = &snapshot.LocationRef{Label: "Vista, CA", Zip: "92084", Lat: 33.2, Lon: -117.24}
	d.selected = d.numPriority() + d.numRecent() + 5
	if d.selectedLocation() != nil {
		t.Fatal("the fixture resolves a location, so it does not pose the wait this test is about")
	}

	got := stripANSITest(d.detailsModal(render.Opts{Width: 133}))
	if !strings.Contains(got, "Vista, CA") {
		t.Errorf("Details opened from a lookup does not name the place while its data lands:\n%s",
			firstLines(got, 3))
	}
	if strings.Contains(firstLines(got, 2), "Location ") {
		t.Errorf("the window still falls back to the literal \"Location\":\n%s", firstLines(got, 3))
	}
}

// THE CONTROL: with no lookup pending and nothing selected, the fallback is
// still the generic title. Without this, the assertion above would be satisfied
// by a window that names Vista in every circumstance.
func TestDetailsWithNoLookupAndNoSelectionKeepsTheGenericTitle(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	d := dash(t).(Dashboard)
	d.modal = modalDetails
	d.lookupRef = nil
	d.selected = d.numPriority() + d.numRecent() + 5
	got := stripANSITest(d.detailsModal(render.Opts{Width: 133}))
	if !strings.Contains(got, "Location") {
		t.Errorf("with nothing to name, the window should still say Location:\n%s", firstLines(got, 3))
	}
}

func firstLines(s string, n int) string {
	ls := strings.Split(s, "\n")
	if len(ls) > n {
		ls = ls[:n]
	}
	return strings.Join(ls, "\n")
}

// ISSUE #11 — A SECOND LOOKUP MUST NOT REPLAY THE FIRST ONE'S DETAILS.
//
// Reported against 0.14.0 from a real session: look up Miami and Lake Henshaw —
// the previous lookup — is on screen until Miami's data lands, then "magically
// swaps in". A regression of the 0.13.0 fix.
//
// THE MEMO KEY IS THE WHOLE STORY, and `selected` cannot stand in for the
// lookup. modal_location.go focuses every lookup at the same index
// (`d.selected = len(watch)`, the first RECENT row), so two lookups in a row
// produce an identical modalKey while the location differs, and the single-slot
// modal memo replays the cached frame. bodyKey has carried lookupKey since the
// original fix; modalKey never did.
//
// This asserts the KEY rather than the rendered frame, because the key is what
// broke: a frame assertion would pass the moment anything else moved the key
// and would not say why.
func TestASecondLookupDoesNotShareTheFirstsModalKey(t *testing.T) {
	d := dash(t).(Dashboard)
	d.modal = modalDetails
	o := render.Opts{Width: 133}

	first := snapshot.LocationRef{Label: "Lake Henshaw, CA", Zip: "92070", Lat: 33.24, Lon: -116.76}
	second := snapshot.LocationRef{Label: "Miami, FL", Zip: "33101", Lat: 25.77, Lon: -80.19}

	// Both lookups focus the same row, exactly as the commit path does.
	d.selected = d.numPriority()
	d.lookupRef = &first
	if d.lookupIndex() >= 0 {
		t.Fatal("the fixture already carries the looked-up row, so this does not pose the wait")
	}
	k1 := d.modalKeyFor(o)

	d.lookupRef = &second
	k2 := d.modalKeyFor(o)

	if k1 == k2 {
		t.Error("two different lookups share one modal key — the memo will replay the first one's Details")
	}
	// CONTROL: the same lookup twice must still hit, or the fix has simply
	// disabled the memo for this window and traded a stale frame for a rebuild
	// on every keystroke.
	d.lookupRef = &second
	if d.modalKeyFor(o) != k2 {
		t.Error("control: the same pending lookup produced two different keys; the memo can never hit")
	}
}
