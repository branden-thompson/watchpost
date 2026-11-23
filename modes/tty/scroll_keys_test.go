package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// TestTheConsolesArrowsMoveItsPointer is 0.19.5 Q2.13 (RN's minor list): on
// the console ↓ moves the running order's pointer down a row and ↑ back up;
// the sign of each key is what this pins.
func TestTheConsolesArrowsMoveItsPointer(t *testing.T) {
	r := NewRouter(benchDash(t, 160, 74).(Dashboard))
	r.broadcaster = loadedConsoleOf(t, Opening{}, ttytest.LoadedPoolSize)
	r.active = SurfaceBroadcaster
	// THE RUNNING ORDER'S ROW, which the pointer stays in for 0, 1, 0 (0.19.6 D-14).
	at := r.broadcaster.LineupSelection()
	down := step(t, r, tea.KeyPressMsg{Code: tea.KeyDown})
	if down.broadcaster.LineupSelection() != at+1 {
		t.Fatalf("↓ moved the console's pointer from %d to %d; want %d", at, down.broadcaster.LineupSelection(), at+1)
	}
	if up := step(t, down, tea.KeyPressMsg{Code: tea.KeyUp}); up.broadcaster.LineupSelection() != at {
		t.Errorf("↑ moved the console's pointer from %d to %d; want %d", at+1, up.broadcaster.LineupSelection(), at)
	}
}

// TestDiagnosticsScrollsItsProse is 0.19.5 Q2.13 (RN's minor list): in a build
// with nothing to inject, ↓ scrolls the Diagnostics window's prose down to its
// last line and no further, and ↑ back to the top and no further.
func TestDiagnosticsScrollsItsProse(t *testing.T) {
	d := goldenWindow(t, 80, 20, false, modalDebug) // 20 rows: the window's prose runs past it
	most := d.footerModalScrollMax(d.opts())
	if most == 0 {
		t.Fatal("at 80x20 the Diagnostics window fits whole; the fixture cannot scroll it")
	}
	for range most + 3 {
		d = d.handleDebugNav("nav-down")
	}
	if d.modalScroll != most {
		t.Errorf("↓ past the end leaves the scroll at %d; want its last line, %d", d.modalScroll, most)
	}
	for range most + 3 {
		d = d.handleDebugNav("nav-up")
	}
	if d.modalScroll != 0 {
		t.Errorf("↑ past the top leaves the scroll at %d; want 0", d.modalScroll)
	}
}
