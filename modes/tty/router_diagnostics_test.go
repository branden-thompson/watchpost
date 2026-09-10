package tty

// router_diagnostics_test.go — D-58: ONE diagnostics window, composited over
// whichever surface is active.
//
// THE RULE IT SERVES IS D-56, "one canonical way to do a thing". The ctrl+d
// window is entirely Dashboard methods, and giving the console its own would be
// a second injector UI, a second confirm, and two places for the TEST EVENT
// wording to drift. So the Router composites the EXISTING window over the
// console instead — and the console stays on screen underneath, which is the
// whole point: the operator injects an alert and WATCHES the takeover activate
// and drain.

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
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
	const base = "x"
	viaConsole := consoleWith(t, goldenDash(t, false))
	viaConsole = pressAction(t, viaConsole, actDiagnostics)

	onObserver := NewRouter(goldenDash(t, false))
	onObserver.observer.width, onObserver.observer.height = 150, 74
	onObserver.observer = onObserver.observer.openDiagnostics()

	want := onObserver.observer.OverlayDiagnostics(base, 150)
	if want == base {
		t.Fatal("the observer's own window composited nothing")
	}
	if got := viaConsole.observer.OverlayDiagnostics(base, 150); got != want {
		t.Errorf("the console composites a DIFFERENT window than the observer draws:\n got %q\nwant %q",
			firstLine(got), firstLine(want))
	}
}

// THE WINDOW SITS ON TOP OF THE FRAME, NOT BESIDE IT — and this is the test the
// first version of these did not have.
//
// It asserted only that the composited frame CHANGED, which a window rendered
// off to the RIGHT satisfies perfectly. In UAT the diagnostics box and its
// confirmation appeared side by side, both pinned to the top rail, because
// `render.Overlay` centres on the TERMINAL width and positions against the
// BASE's height — so compositing the confirmation onto the bare 70-wide window
// put it at x=70 in a 200-column terminal, past the right edge of the thing it
// was covering.
//
// THE SHAPE THIS PINS: the frame keeps its size, and every layer lands INSIDE
// it. A composite that grows the frame has placed something beside it.
func TestTheWindowsStackOnTheFrameRatherThanBesideIt(t *testing.T) {
	r := consoleWith(t, goldenDash(t, false))
	before := r.View().Content
	baseW, baseH := frameWidth(before), strings.Count(before, "\n")

	r = pressAction(t, r, actDiagnostics)
	got := r.View().Content

	if w := frameWidth(got); w > baseW {
		t.Errorf("the composite is %d cells wide against a %d-cell frame: something landed BESIDE it, not on it", w, baseW)
	}
	if h := strings.Count(got, "\n"); h > baseH {
		t.Errorf("the composite grew to %d rows from %d: a layer landed below the frame", h, baseH)
	}
	// AND THE FRAME MUST BE THE VIEWPORT. A short frame pins every overlay to
	// the top rail, because Overlay centres vertically on the base's height.
	if baseH < r.broadcaster.height-1 {
		t.Errorf("the console frame is %d rows in a %d-row terminal; overlays cannot centre on it",
			baseH, r.broadcaster.height)
	}
}

// frameWidth is the widest line in a rendered frame, in cells.
func frameWidth(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if c := utf8.RuneCountInString(line); c > n {
			n = c
		}
	}
	return n
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

// EVERY LAYER CENTRES ON THE TERMINAL, WHICH IS WHAT "ON TOP OF" MEANS HERE.
//
// THE UAT BUG THIS PINS. `render.Overlay` centres on the TERMINAL width and
// positions against the BASE's height, so compositing the confirmation onto the
// bare 84-cell window put it at x=(150-65)/2=42 INSIDE AN 84-CELL BASE. The
// pair then went onto the frame as one 107-cell block, landing the confirmation
// at column 63 in a 150-column terminal — 20 cells off centre, and visibly
// BESIDE the window rather than on it.
//
// The first attempt at a test asserted only that the composite was no wider
// than the frame, and the broken layout FIT INSIDE a 150-cell frame, so it
// passed. Width was never the property. POSITION is.
func TestEveryLayerCentresOnTheTerminal(t *testing.T) {
	// `wiredDebug` opens the window itself, so pressing ctrl+d here would
	// TOGGLE IT SHUT — the fixture and the keypress fighting each other.
	r := consoleWith(t, wiredDebug(t, func(string) {}))
	if !r.observer.DiagnosticsOpen() {
		t.Fatal("fixture: the window must be open")
	}
	// enter raises the confirmation over the window, injecting nothing.
	m, _ := r.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	r = m.(Router)
	if !r.observer.debug.confirm {
		t.Fatal("fixture: the confirmation must be up")
	}

	got := stripANSITest(r.View().Content)
	want := r.broadcaster.width / 2
	c, ok := centreOfLineContaining(got, "ARE YOU SURE")
	if !ok {
		t.Fatal("the confirmation must be drawn")
	}
	if d := c - want; d > 3 || d < -3 {
		t.Errorf("the confirmation's centre is column %d in a %d-column terminal (want ~%d): "+
			"it was composited against something narrower than the frame, so it sits BESIDE what it covers",
			c, r.broadcaster.width, want)
	}
}

// centreOfLineContaining is the mid-column of the drawn text on the first line
// holding `want` — enough to say where a box sits without parsing its borders.
func centreOfLineContaining(frame, want string) (int, bool) {
	for _, line := range strings.Split(frame, "\n") {
		i := strings.Index(line, want)
		if i < 0 {
			continue
		}
		return i + utf8.RuneCountInString(want)/2, true
	}
	return 0, false
}

// THE MASTHEAD'S DATA REACHES THE CONSOLE THROUGH THE ROUTER, AND THAT IS THE
// PART UNIT TESTS KEEP MISSING.
//
// THE THIRD TIME THIS SHAPE HAS BITTEN in this release. The console's own tests
// set `b.version` and `b.snap` themselves, so deleting BOTH production wirings —
// `b.version = o.cfg.Version` in NewRouter, and SnapshotMsg's fan-out — changed
// no assertion anywhere. Same as the top-off's depth and producer seam.
//
// SO IT IS DRIVEN THE WAY PRODUCTION DRIVES IT: a Router built over a Dashboard
// that HAS a version, and a snapshot delivered as a MESSAGE.
func TestTheConsoleLearnsTheVersionAndTheSnapshotThroughTheRouter(t *testing.T) {
	d := goldenDash(t, true)
	d.cfg.Version = "9.9.9"
	r := NewRouter(d)
	r.broadcaster.width, r.broadcaster.height = 150, 74
	r.observer.width, r.observer.height = 150, 74

	m, _ := r.Update(SnapshotMsg{Snap: mastheadSnap()})
	r = m.(Router)
	r = pressAction(t, r, actSwapBroadcaster)

	got := stripANSITest(r.View().Content)
	if !strings.Contains(got, "9.9.9") {
		t.Errorf("the console never learned the build's version:\n%s", firstLine(got))
	}
	if !strings.Contains(got, "Updated:") {
		t.Errorf("the console never learned the snapshot, so its masthead has no stamp:\n%s", firstLine(got))
	}
}

// THE GAIN THE CONSOLE DRAWS IS THE LEVEL OBSERVER OWNS, AND PRESSING IT FROM
// THE CONSOLE MOVES THAT ONE LEVEL.
//
// PLANTED FIRST THIS TIME. The wiring has been the survivor three times running
// in this release — the top-off's depth, its producer seam, the masthead's
// version and snapshot — every time because a unit test set the field itself.
// So this drives the KEY, through the Router, and reads the level back from the
// surface that owns it.
func TestGainPressedOnTheConsoleMovesObserversOwnLevel(t *testing.T) {
	r := consoleWith(t, goldenDash(t, true))
	before := r.observer.radioVolume
	if r.broadcaster.gain != before {
		t.Fatalf("the console must mirror the level it draws: console %d, observer %d",
			r.broadcaster.gain, before)
	}

	r = pressAction(t, r, actGainUp)

	if r.observer.radioVolume <= before {
		t.Errorf("the press must reach the surface that owns the level: %d -> %d", before, r.observer.radioVolume)
	}
	if r.broadcaster.gain != r.observer.radioVolume {
		t.Errorf("and the console must draw that same level, not a second copy: console %d, observer %d",
			r.broadcaster.gain, r.observer.radioVolume)
	}
	// AND IT DOES NOT SWAP SURFACES. The operator is watching the console.
	if r.active != SurfaceBroadcaster {
		t.Error("the gain keys must not move the operator off the console")
	}
}

func TestGainDownReachesTheSameLevel(t *testing.T) {
	r := consoleWith(t, goldenDash(t, true))
	before := r.observer.radioVolume

	r = pressAction(t, r, actGainDown)

	if r.observer.radioVolume >= before {
		t.Errorf("gain down must lower the one level: %d -> %d", before, r.observer.radioVolume)
	}
	if r.broadcaster.gain != r.observer.radioVolume {
		t.Errorf("console %d, observer %d", r.broadcaster.gain, r.observer.radioVolume)
	}
}
