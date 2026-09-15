package tty

// help_surface_test.go — the Help window answers for the surface you are ON.
//
// HUM LEAD, UAT 2026-09-15: "we also need to make the ? (help) modal display
// the correct key bindings and hints depending on the mode that currently
// active - rigth now the help window only shows Observer key bindings, and it
// doesnt show the user how to swap between Observer and Broadcaster.  Once the
// user in the Broadcaster UI, they key bindings share/rempapped for that mode
// do not update their help (like 'r')."

import (
	"strings"
	"testing"
)

func helpTextOn(t *testing.T, surface Surface) string {
	t.Helper()
	d := goldenDash(t, false)
	d.surface = surface
	d = d.open(modalHelp)
	return stripANSITest(d.renderModal(d.opts()))
}

// DEFECT 1: the console was handed the listener's manual.
func TestTheConsolesHelpDocumentsTheConsole(t *testing.T) {
	got := helpTextOn(t, SurfaceBroadcaster)
	for _, want := range []string{"ON AIR / STANDBY", "Line-Up Request", "Details / Manage Slot", "Previous Relay"} {
		if !strings.Contains(got, want) {
			t.Errorf("the console's help does not mention %q:\n%s", want, got)
		}
	}
	// AND NOT THE LISTENER'S. A console has no watchlist and no visualizer, and
	// rows for keys that do nothing here are worse than no window at all.
	for _, never := range []string{"Add Location", "Remove from Watchlist", "Visualizer"} {
		if strings.Contains(got, never) {
			t.Errorf("the console's help offers %q, which belongs to Observer:\n%s", never, got)
		}
	}
	// THE SECTION HEADERS, NOT ONLY THE ROWS — and this is the half the first
	// draft missed. `helpBlocks` sweeps anything the grouping does not claim
	// into an OTHER block, so a window built from OBSERVER'S grouping and the
	// CONSOLE'S keymap still contains every console row, just heaped under one
	// heading. The rows passed; the organisation was unmeasured, and the mutant
	// that swapped the grouping survived on exactly that gap.
	for _, want := range []string{"STATION", "LINE UP", "BED"} {
		if !strings.Contains(got, want) {
			t.Errorf("the console's help has no %s section; its controls are not organised for this surface:\n%s", want, got)
		}
	}
	for _, never := range []string{"NAVIGATE", "RADIO", "WATCHLIST", "TICKER"} {
		if strings.Contains(got, never) {
			t.Errorf("the console's help is grouped into Observer's %s section:\n%s", never, got)
		}
	}
}

// DEFECT 2: neither surface said how to reach the other.
func TestBothSurfacesDocumentTheSwap(t *testing.T) {
	for _, tc := range []struct {
		name    string
		surface Surface
	}{{"observer", SurfaceObserver}, {"broadcaster", SurfaceBroadcaster}} {
		got := helpTextOn(t, tc.surface)
		if !strings.Contains(got, "SURFACES") {
			t.Errorf("%s: no SURFACES group — the operator cannot find their way off this surface:\n%s", tc.name, got)
		}
		for _, want := range []string{"ctrl+o", "ctrl+b"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: help never mentions %q, and the Router accepts it on BOTH surfaces", tc.name, want)
			}
		}
	}
}

// DEFECT 3: a key that means two things must read as the RIGHT one on each
// surface. This is D-56 — one key, one meaning per surface — being DOCUMENTED,
// which is the half that was missing.
func TestARemappedKeyReadsAsItsSurfacesMeaning(t *testing.T) {
	console, observer := helpTextOn(t, SurfaceBroadcaster), helpTextOn(t, SurfaceObserver)

	if !strings.Contains(console, "r            - Line-Up Request") {
		t.Errorf("on the console, [r] must read as the request window:\n%s", console)
	}
	if strings.Contains(console, "Repeat: Off / One / Watchlist") {
		t.Errorf("the console's help describes [r] as Observer's repeat:\n%s", console)
	}
	if !strings.Contains(observer, "Repeat: Off / One / Watchlist") {
		t.Errorf("Observer's [r] lost its own meaning:\n%s", observer)
	}
}

// AND THE LEGEND FOLLOWS THE SURFACE TOO. The row marks are a table Observer
// draws and the console does not; the console's ten slot ADDRESSES are
// deliberately not bindings, so the legend is the only place they are explained.
func TestTheLegendBelongsToItsSurface(t *testing.T) {
	console, observer := helpTextOn(t, SurfaceBroadcaster), helpTextOn(t, SurfaceObserver)

	if !strings.Contains(console, "Slot numbers:") {
		t.Errorf("the console never explains what a slot's number does:\n%s", console)
	}
	if strings.Contains(console, "Row marks:") {
		t.Errorf("the console shows Observer's row-mark legend for a table it does not draw:\n%s", console)
	}
	if !strings.Contains(observer, "Row marks:") {
		t.Errorf("Observer lost its row-mark legend:\n%s", observer)
	}
}
