package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// overlayShipped is the console's window and its cursor as 0.19.3 drew them,
// each working out the Observer's layout for itself: the reference the
// one-layout overlay is held to.
func overlayShipped(d Dashboard, base string, termWidth int) (out string, x, y int, ok bool) {
	if !d.ModalOpen() {
		return base, 0, 0, false
	}
	o := d.layout().o
	x, y, ok = d.frameCursor(o, base, d.modalView(o), -1, d.confirmOverlay(o), termWidth)
	o = d.layout().o
	if win := d.modalView(o); win != "" {
		base = render.Overlay(base, win, termWidth)
	}
	if box := d.confirmOverlay(o); box != "" {
		base = render.Overlay(base, box, termWidth)
	}
	return base, x, y, ok
}

// TestTheConsolesWindowIsTheSameLaidOnce is 0.19.5 Q3.6 (PF-11): an Observer
// window over the console, and the cursor in it, come out the same with the
// layout worked out once as with it worked out for each: Settings, Diagnostics
// and Diagnostics' confirmation, and no window at all.
func TestTheConsolesWindowIsTheSameLaidOnce(t *testing.T) {
	r := consoleRouter(t)
	base := r.broadcaster.View().Content
	d := mapDash(t, Config{ASCII: true})
	m, _ := d.Update(tea.WindowSizeMsg{Width: 150, Height: 74})
	d = m.(Dashboard)
	confirming := d.openDiagnostics()
	confirming.debug.confirm = true
	for name, w := range map[string]Dashboard{
		"no window": d, "Settings": d.openSetupAt(rowTheme),
		"Diagnostics": d.openDiagnostics(), "the confirmation": confirming,
	} {
		got, gx, gy, gok := w.overlay(base, 150)
		want, wx, wy, wok := overlayShipped(w, base, 150)
		if got != want {
			line, a, b := firstDiffLine(want, got)
			t.Errorf("%s: the frame differs at line %d:\n want %q\n  got %q", name, line, a, b)
		}
		if gx != wx || gy != wy || gok != wok {
			t.Errorf("%s: the cursor is at %d,%d (%v); laid twice it is at %d,%d (%v)", name, gx, gy, gok, wx, wy, wok)
		}
	}
}
