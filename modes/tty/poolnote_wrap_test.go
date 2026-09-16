package tty

// poolnote_wrap_test.go — the caveat must keep its colour ALL THE WAY TO THE
// END, in every window that draws it.
//
// HUM LEAD, UAT 2026-09-14 (second report, with a screenshot): "'Broadcast
// Radius' bug is back". The first fix wrapped the text and tinted each line, so
// the tint could survive a wrap. This one was the OTHER half of the same rule:
// the lines were wrapped to the REQUEST window's width (119 cells at 133 cols)
// and handed to a window 56 cells wide, which wrapped them AGAIN — and the
// second wrap is the frame's, after the tint, so the tail came out plain.
//
// A TINT IS TWO ESCAPE CODES AT THE ENDS OF A STRING. Wrapping to the wrong
// width is the same defect as not wrapping at all.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// colouredFrame renders with colour ON, which is what makes this measurable at
// all: the golden fixtures disable it, and a test that inherits that would read
// a plain tail as correct in a frame that has no colour anywhere.
func colouredFrame(t *testing.T, d Dashboard) string {
	t.Helper()
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
	return d.renderModal(d.opts())
}

// everyCaveatLineIsStyled: no line of the wrapped caveat may reach the frame
// without its own escape codes.
func everyCaveatLineIsStyled(t *testing.T, frame, what string) {
	t.Helper()
	var seen int
	for _, line := range strings.Split(frame, "\n") {
		if !strings.Contains(line, "Observer supports") && !strings.Contains(line, "Broadcast Radius") {
			continue
		}
		seen++
		// THE ITALIC, NOT "any escape code". The PANEL tints its own background
		// on every line it draws, so `contains "\x1b["` is true of every line in
		// the window and measures nothing — which is how the first version of
		// this test passed against the build in the HUM LEAD's screenshot.
		// Italic is the caveat's own, and the frame never adds it.
		if !strings.Contains(line, "\x1b[3m") {
			t.Errorf("%s: this line of the caveat lost the caveat's own styling — the tint died on "+
				"a wrap the WINDOW did after we had already coloured it:\n  %q", what, line)
		}
	}
	if seen < 2 {
		t.Fatalf("%s: expected the caveat to WRAP onto at least two lines (it is the wrap that "+
			"breaks it); saw %d", what, seen)
	}
}

func TestTheConsoleLookupsCaveatKeepsItsColourAcrossTheWrap(t *testing.T) {
	d := goldenDash(t, false)
	d.surface, d.addMode, d.addQuery = SurfaceBroadcaster, "lookup", "lone pine"
	// THE HOOK MUST BE WIRED EVEN THOUGH THE VERDICT IS PRE-SETTLED: its
	// presence is what puts the window in the console's scoped mode at all, so
	// a fixture without it draws no caveat and the test measures a blank frame.
	d.cfg.LocateInRadius = func(string) (snapshot.LocationRef, bool, bool, bool) {
		return snapshot.LocationRef{}, false, false, true
	}
	d.addLocate = settledLocate(locateLookup, "lone pine", snapshot.LocationRef{}, false, false)
	d.addLocate.asked = true // the lookup ANSWERED; "no such place" is the answer
	d = d.open(modalAdd)

	everyCaveatLineIsStyled(t, colouredFrame(t, d), "the console's lookup window")
}

// AND THE REQUEST WINDOW, which is where the rule was ruled. Both windows draw
// the same sentence through the same helper, so both are measured here — a fix
// that repaired one and left the other is exactly what happened the first time.
func TestTheRequestWindowsCaveatKeepsItsColourAcrossTheWrap(t *testing.T) {
	d := goldenDash(t, false)
	d.request = requestOpen()
	d.request.query = "Lone Pine, CA"
	d.request.locate = settledLocate(locateRequest, "Lone Pine, CA", snapshot.LocationRef{}, false, false)
	d.request.locate.asked = true
	d = d.open(modalRequest)

	everyCaveatLineIsStyled(t, colouredFrame(t, d), "the request window")
}
