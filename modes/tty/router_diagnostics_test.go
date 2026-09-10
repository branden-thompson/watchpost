package tty

// router_diagnostics_test.go — D-58: ONE diagnostics window, composited over
// whichever surface is active.
//
// THE RULE IT SERVES IS D-56, "one canonical way to do a thing". The ctrl+d
// window is entirely Dashboard methods, and giving the console a second one
// would be a second injector UI, a second confirm, and two places for the
// TEST EVENT wording to drift. So the Router composites the EXISTING window
// over the console instead — and the console stays on screen underneath, which
// is the whole point: the operator injects an alert and WATCHES the takeover
// activate and drain.

import (
	"strings"
	"testing"
)

// consoleWith puts the operator on the console, over a REAL Dashboard.
//
// A BARE `Dashboard{}` HAS NO KEYMAP, so a forwarded ctrl+d reaches nothing —
// which the first draft of these tests discovered by failing. The window is
// reached through the Observer's own bindings, so the fixture has to be a
// Dashboard that HAS them.
func consoleWith(t *testing.T, d Dashboard) Router {
	t.Helper()
	r := NewRouter(d)
	r.broadcaster.width, r.broadcaster.height = 150, 74
	r.observer.width, r.observer.height = 150, 74
	r = pressAction(t, r, actSwapBroadcaster)
	if r.active != SurfaceBroadcaster {
		t.Fatal("the test needs to be on the console")
	}
	return r
}

func TestTheOperatorCanOpenDiagnosticsFromTheConsole(t *testing.T) {
	r := consoleWith(t, goldenDash(t, false))

	r = pressAction(t, r, actDiagnostics)

	if !r.observer.DiagnosticsOpen() {
		t.Fatal("ctrl+d on the console must open the diagnostics window")
	}
	if r.active != SurfaceBroadcaster {
		t.Error("and it must not swap surfaces: the operator is watching the console")
	}
}

// THE CONSOLE STAYS VISIBLE UNDERNEATH. A window that replaced the frame would
// make it impossible to watch what the injection does, which is the only reason
// to reach it from here.
func TestTheConsoleIsStillDrawnBeneathTheDiagnosticsWindow(t *testing.T) {
	r := consoleWith(t, goldenDash(t, false))
	before := r.View().Content
	if !strings.Contains(before, "LINE UP") {
		t.Fatalf("fixture: the console must be drawing its lanes; got:\n%s", before)
	}

	r = pressAction(t, r, actDiagnostics)
	got := r.View().Content

	if got == before {
		t.Fatal("the frame did not change: nothing was composited over the console")
	}
	if !strings.Contains(got, "LINE UP") {
		t.Error("the console must remain visible beneath the window; it was replaced instead")
	}
}

// IT IS THE SAME WINDOW THE OBSERVER SHOWS, not a copy of it. If these ever
// diverge there are two injector UIs, which is what D-56 refuses.
func TestItIsTheObserversOwnWindowAndNotACopy(t *testing.T) {
	viaConsole := consoleWith(t, goldenDash(t, false))
	viaConsole = pressAction(t, viaConsole, actDiagnostics)

	onObserver := NewRouter(goldenDash(t, false))
	onObserver.observer.width, onObserver.observer.height = 150, 74
	onObserver.observer = onObserver.observer.openDiagnostics()

	want := onObserver.observer.DiagnosticsOverlay()
	if want == "" {
		t.Fatal("the observer's own window rendered nothing")
	}
	if got := viaConsole.observer.DiagnosticsOverlay(); got != want {
		t.Errorf("the console composites a DIFFERENT window than the observer draws:\n got %q\nwant %q",
			firstLine(got), firstLine(want))
	}
}

// WHILE IT IS OPEN THE KEYS ARE ITS OWN. An arrow that moved a card in the
// line-up while the operator was navigating the injector would be the console
// acting on input meant for the window on top of it.
func TestTheWindowOwnsTheKeysWhileItIsOpen(t *testing.T) {
	r := consoleWith(t, goldenDash(t, false))
	r = pressAction(t, r, actDiagnostics)
	if !r.observer.DiagnosticsOpen() {
		t.Fatal("fixture: the window must be open")
	}

	// esc is the window's own way out, and it must reach it.
	m, _ := r.Update(keyFor(t, "esc"))
	out, ok := m.(Router)
	if !ok {
		t.Fatal("the router must stay the program's model")
	}
	if out.observer.DiagnosticsOpen() {
		t.Error("esc must reach the window and close it, not the console beneath")
	}
	if out.active != SurfaceBroadcaster {
		t.Error("and closing it leaves the operator where they were")
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
