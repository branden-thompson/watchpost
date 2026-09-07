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
