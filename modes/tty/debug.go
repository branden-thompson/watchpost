package tty

// debug.go — the ctrl+d window (F-21).
//
// TWO FEATURES LIVE HERE AND THEY HAVE OPPOSITE SHIPPING ANSWERS, which is the
// distinction the follow-up was written to preserve:
//
//   - DIAGNOSTICS are read-only checks a listener runs against reality, and are
//     meant to SHIP. The weatherUSA outage is why: the program was correct and
//     an upstream service was failing while APPEARING healthy — HTTP 200,
//     audio/mpeg, correct ICY headers, well-formed MP3, and total silence. Every
//     signal the app checks was green and the listener heard nothing.
//   - INJECTION fabricates an alert to exercise the takeover on demand, and must
//     NEVER ship: a screenshot of a fabricated tornado warning is
//     indistinguishable from a real one.
//
// The window is the same surface; the SECTIONS differ. Injection renders only
// when the app supplied a hook, and a release build never does — the capability
// is absent from the binary (app/inject_release.go), so this cannot offer what
// does not exist.

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

const (
	debugWidth = 84 // the relay-fault window's width: one shape for the app's asides
	// debugPickW is the Alert Type picker's value cell: the widest value this
	// window offers plus room for one longer, so the chips do not move when the
	// value changes and a new scenario does not silently truncate.
	debugPickW = 30
)

// DebugScenario is one thing the window can make happen. Label is what the
// operator reads; Key is what goes back to the app.
type DebugScenario struct {
	Label string
	Key   string
}

// debugState is the open window: which question has the focus, which value that
// question is showing, and whether the confirmation is up.
type debugState struct {
	focus   int
	pick    int
	confirm bool
}

// debugScenarios is what this build offers. Empty in a release build, because
// the app supplies no injector there and a window offering nothing offers
// nothing rather than a disabled row.
func (d Dashboard) debugScenarios() []DebugScenario {
	if d.cfg.InjectAlert == nil {
		return nil
	}
	return d.cfg.DebugScenarios
}

// debugProseWidth is the width this window's prose wraps to: the panel's RAIL
// budget, less the window's own inset.
//
// WRAPPED ONCE, AND WITH A MARGIN LEFT. Wrapping to the box width let the panel
// wrap a second time three columns narrower, and at 80 columns the prose came
// apart into orphan lines reading "a", "and", "correctly". Leaving the wrap to
// the panel fixed that and cost the right margin — the panel wraps to its
// border, and every other window in the app clears it by three. Wrapping to the
// NARROWER of the two budgets does both: the panel's wrap is then a no-op, and
// on a frame with no scroll rail the text is three cells short of what it could
// be, which is invisible.
func debugProseWidth(o render.Opts, box int) int {
	return max(min(o.Width, box)-7-(modalInset-panelSide), 8)
}

// debugTitle is the window's border: its name, and the warning it carries
// wherever it is drawn.
//
// THE WARNING IS IN THE CHROME, not in the body, so it cannot scroll away from
// the control it is about.
func (d Dashboard) debugTitle(o render.Opts, w int) string {
	const name, warn = "DIAGNOSTICS", "*USE RESPONSIBLY*"
	fill := w - severeTitleChrome - render.Width(name) - render.Width(warn)
	tint := func(s string) string { return render.Tint(s, render.Tok(render.ModalTitle)) }
	if fill <= 1 {
		return tint(name)
	}
	return tint(name) + " " + strings.Repeat(o.Glyphs().Rule, fill) + " " + tint(warn)
}

// debugLines is the body.
func (d Dashboard) debugLines(o render.Opts) (out []string, focusAt, focusEnd int) {
	// ONE PARAGRAPH, WRAPPED BY ITS OWNER. Hand-wrapped literals were wrapped a
	// second time by the panel at 80 columns and came apart into orphan lines
	// ("a", "and", "correctly") — a paragraph the window wraps once cannot.
	out = insetModalLines([]string{
		"Tools to verify Watchpost machinery is working as intended. USE RESPONSIBLY. Audio " +
			"diagnostics will have Diagnostic messages attached to the beginning and ending of the " +
			"audio feed for safety purposes to ensure listeners do not confuse the diagnostic event " +
			"as a real weather report, situation, or emergency.",
		"",
	}, debugProseWidth(o, debugWidth))

	sc := d.debugScenarios()
	if len(sc) == 0 {
		// -1: NOTHING TO FOCUS, SO THE BODY SCROLLS (FR-5). This is the window a
		// release build ships. It returned 0 — "hold the top" — and 0 is a
		// focused row, so the scroll never moved: at 80x24 every line of what
		// this window exists to say sat below the fold with no key that reached
		// it, in the build that ships.
		return append(out, insetModalLines([]string{
			"INJECTION IS NOT AVAILABLE IN THIS BUILD.", "",
			"It is compiled out rather than switched off, so a fabricated alert cannot be produced " +
				"here by any means. Build the diagnostics binary with `make build-diag` to enable it."},
			debugProseWidth(o, debugWidth))...), -1, -1
	}
	out = append(out, insetModalLines([]string{
		"ALERT INJECTION - ENSURE ALERTING AND TICKER TAKEOVER WORKS", ""}, debugProseWidth(o, debugWidth))...)
	// THROUGH THE SETTINGS ROW'S OWN CONTROLS (D-1). The mark, the label and the
	// picker are the Settings window's, because this is the same gesture on the
	// same shape of question — and because the --ascii scan and the AA register
	// already know those three.
	focusAt = len(out)
	focusEnd = focusAt
	out = append(out, strings.Repeat(" ", modalInset)+o.ListMark(true)+" "+
		settingLabel("Alert Type:", true)+"   "+
		pickerCellW(sc[d.debugPick()].Label, newArrowChips(o), flashNone, debugPickW))
	return append(out, ""), focusAt, focusEnd
}

// debugPick is the focused value, bounded by what the build offers — the list
// is the app's and this window never trusts an index into it.
func (d Dashboard) debugPick() int {
	n := len(d.debugScenarios())
	if n == 0 {
		return 0
	}
	return ((d.debug.pick % n) + n) % n
}

// debugChips is the pinned footer.
func (d Dashboard) debugChips(o render.Opts) []string {
	if len(d.debugScenarios()) == 0 {
		// ↑↓ SCROLL HERE, AND THE CHIPS SAY SO. There is no question in this
		// build, and a window whose text runs past the fold with no advertised
		// way down reads as a window with nothing more in it.
		return []string{"  " + o.KeyCap("↑↓") + " Scroll    " + o.KeyCap("esc") + " Close"}
	}
	// "TEST ALERT" HERE, "TEST EVENT" ON THE SURFACES. The chip names the ACTION
	// — inject a test alert — and the mark names what the thing IS wherever it
	// is later seen. The mock says both, in those two places.
	return []string{"  " + o.KeyCap("tab") + " Next question    " +
		o.KeyCap("enter") + " Inject **TEST ALERT**    " +
		o.KeyCap("↑↓") + " Pick   " + o.KeyCap("esc") + " Cancel"}
}

// debugConfirmWidth is the confirmation's box, to the mock.
const debugConfirmWidth = 65

// debugConfirmLines is the ARE YOU SURE window (HUM LEAD mock, 2026-09-07).
//
// IT IS A SECOND WINDOW OVER THE FIRST, and it is RED. An injection cannot be
// stopped once it is under way — it enters where a real alert enters and
// crosses every stage a real one does — so the last thing between the operator
// and a fabricated alert on their own broadcast is a question they have to
// answer, on the colour this app uses for exactly one thing.
func (d Dashboard) debugConfirmLines(o render.Opts) []string {
	centre := func(s string) string {
		return strings.Repeat(" ", max((debugConfirmWidth-2-2*modalInset-render.Width(s))/2+modalInset-panelSide, 0)) + s
	}
	out := []string{"", centre("ARE YOU SURE?"), centre("*** ONCE CONFIRMED, YOU CANNOT STOP THIS ACTION ***"), ""}
	out = append(out, insetModalLines([]string{
		"Watchpost has taken every reasonable measure to ensure an injected alert is clearly " +
			"marked as such in the UI, and in the audio read scripts to minimize any potential " +
			"confusion to anyone listening to the audio or your broadcast.  However, as the " +
			"operator, responsibility for the contents of your station broadcast is ultimately " +
			"yours.",
		""}, debugProseWidth(o, debugConfirmWidth))...)
	return append(out, strings.Repeat(" ", modalInset)+o.KeyCap("esc")+"  Cancel   "+
		o.KeyCap("enter")+" CONFIRM: I UNDERSTAND", "")
}

// handleDebugNav walks the window: the questions with tab, the focused
// question's value with the arrows.
func (d Dashboard) handleDebugNav(act term.Action) Dashboard {
	n := len(d.debugScenarios())
	if n == 0 {
		// A BUILD WITH NO LIST STILL HAS A WINDOW TO READ (FR-5): the keys that
		// pick a value scroll the prose instead, bounded by the window's own
		// geometry rather than by a second copy of it.
		switch act {
		case "nav-up":
			d.modalScroll = max(0, d.modalScroll-1)
		case "nav-down":
			d.modalScroll = min(d.modalScroll+1, d.footerModalScrollMax(d.opts()))
		}
		return d
	}
	if d.debug.confirm {
		return d // the question is answered with enter or esc, not walked
	}
	switch act {
	case "nav-up", "alert-prev":
		d.debug.pick = ((d.debugPick()-1)%n + n) % n
	case "nav-down", "alert-next":
		d.debug.pick = (d.debugPick() + 1) % n
	}
	return d
}

// askDebugConfirm puts the confirmation up. ENTER DOES NOT INJECT: the window
// asks first, and the asking is the feature.
func (d Dashboard) askDebugConfirm() Dashboard {
	if len(d.debugScenarios()) == 0 {
		return d
	}
	d.debug.confirm = true
	return d
}

// cancelDebugConfirm takes the confirmation down and injects nothing.
func (d Dashboard) cancelDebugConfirm() Dashboard {
	d.debug.confirm = false
	return d
}

// chooseDebug fires the chosen scenario and closes the window — ONLY from the
// confirmation. Nothing else in this file injects.
func (d Dashboard) chooseDebug() (Dashboard, tea.Cmd) {
	sc := d.debugScenarios()
	if len(sc) == 0 || !d.debug.confirm {
		return d, nil
	}
	inject, key := d.cfg.InjectAlert, sc[d.debugPick()].Key
	d.modal, d.debug = modalNone, debugState{}
	return d, func() tea.Msg {
		if inject != nil {
			inject(key)
		}
		return nil
	}
}

// ModalOpen reports whether ANY of Observer's windows is showing (D-65).
//
// GENERALISED FROM DiagnosticsOpen. The console advertises Settings, About,
// Status and Help in its masthead, and the Router composites whichever the
// operator opened — one door for every window rather than a method per window,
// which is what a second `OverlayAbout` would have become.
func (d Dashboard) ModalOpen() bool { return d.modal != modalNone }

// DiagnosticsOpen reports whether the ctrl+d window is showing (D-58).
//
// EXPORTED FOR THE ROUTER, which composites this window over the console. It is
// a narrow, named seam rather than the Router reaching into `d.modal`: the
// Dashboard owns what "open" means, including the day a second modal state
// exists.
func (d Dashboard) DiagnosticsOpen() bool { return d.modal == modalDebug }

// OverlayWindow lays whichever of Observer's windows is open — and its
// confirmation — over another surface's frame (D-58, generalised at D-65).
//
// IT TAKES THE BASE RATHER THAN RETURNING A PRE-COMPOSITED PAIR, and that is a
// CORRECTNESS requirement, not a style choice. `render.Overlay` centres the
// modal on the TERMINAL width and positions it against the BASE's height:
//
//	x := max(0, (termWidth-Width(modal))/2)
//	y := max(0, (Height(base)-Height(modal))/2)
//
// So compositing the confirmation onto the BARE WINDOW put it at x=70 in a
// 200-column terminal — past the right edge of the 70-wide box it was meant to
// cover — and the two rendered SIDE BY SIDE. Found in UAT, on screen, because
// the test asserted only that the frame CHANGED.
//
// BOTH LAYERS GO ONTO THE SAME FULL-SIZE BASE, which is exactly what
// `Dashboard.View` does with them and why it never had this bug. Keeping that
// rule in one place is the point of the seam (D-56).
func (d Dashboard) OverlayWindow(base string, termWidth int) string {
	if !d.ModalOpen() {
		return base
	}
	o := d.layout().o
	if win := d.modalView(o); win != "" {
		base = render.Overlay(base, win, termWidth)
	}
	// AND THE CONFIRMATION OVER THAT — a second layer rather than a swapped
	// body, which is the HUM LEAD's own mock: the red box sits ON TOP of the
	// diagnostics window and the window underneath is unchanged.
	if box := d.confirmOverlay(o); box != "" {
		base = render.Overlay(base, box, termWidth)
	}
	return base
}

// openDiagnostics opens the window directly, for tests that need it open
// without pressing a key on a particular surface.
func (d Dashboard) openDiagnostics() Dashboard { return d.toggle(modalDebug) }
