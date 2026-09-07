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
	// debugContent is the room between the two margins — the box less its
	// borders, less modalInset either side.
	debugContent = debugWidth - 2 - 2*modalInset
)

// DebugScenario is one thing the window can make happen. Label is what the
// operator reads; Key is what goes back to the app.
type DebugScenario struct {
	Label string
	Key   string
}

// debugState is the open window.
type debugState struct {
	focus int
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

// debugLines is the body.
// debugTitle heads the window, centred from the width it is drawn at rather
// than from a hand-computed column.
const debugTitle = "*** DIAGNOSTICS ***"

func (d Dashboard) debugLines(o render.Opts) (out []string, focusAt, focusEnd int) {
	out = []string{
		strings.Repeat(" ", max((debugWidth-2-2*modalInset-render.Width(debugTitle))/2+modalInset-panelSide, 0)) + debugTitle,
		"",
		" This window is not part of a release build's alerting surface. It exists so a",
		" person can check whether the app is working when the answer is not obvious —",
		" the case that produced it was a relay that answered every check correctly and",
		" broadcast silence.",
		"",
	}
	// THE PROSE THROUGH THE ONE OWNER (red team 2026-09-05). These are
	// hand-wrapped literals and they reached two cells from the right border —
	// the same defect the relay-fault window had, in the window next door, and
	// found by measuring every window instead of the one that was reported.
	out = insetModalLines(out, debugContent)

	sc := d.debugScenarios()
	if len(sc) == 0 {
		// -1: NOTHING TO FOCUS, SO THE BODY SCROLLS (FR-5). This is the window a
		// release build ships. It returned 0 — "hold the top" — and 0 is a
		// focused row, so the scroll never moved: at 80x24 every line of what
		// this window exists to say sat below the fold with no key that reached
		// it, in the build that ships.
		return append(out, insetModalLines([]string{
			"INJECTION IS NOT AVAILABLE IN THIS BUILD.", "",
			"It is compiled out rather than switched off, so a fabricated alert cannot be",
			"produced here by any means. Build with -tags watchpost_debug to enable it.", ""},
			debugContent)...), -1, -1
	}
	out = append(out, insetModalLines([]string{"INJECT AN ALERT:", ""}, debugContent)...)
	// THROUGH THE LIST'S ONE OWNER (D-1), like every other list. This drew a
	// bare "›" with no tint — which
	// --ascii could not turn into ">" — and the cursor moved with nothing on
	// the line changing colour. The same three defects the relay-fault window
	// had, in the window next door (red team 2026-09-05).
	//
	// IT WAS NOT THE LAST ONE, which this comment claimed until 2026-09-06. The
	// Settings window's location suggestions had the same three defects, in the
	// most-opened window in the app, and the --ascii scan that should have
	// caught it could not: its fixture focuses a cast row, so the suggestion
	// list never rendered. A gate whose fixture cannot reach the branch is
	// vacuous on that branch.
	for i, s := range sc { // bounded by the scenarios (P10-02)
		focused := i == d.debug.focus
		if focused {
			focusAt, focusEnd = len(out), len(out)
		}
		out = append(out, strings.Repeat(" ", modalInset)+o.ListMark(focused)+" "+render.ListLabel(s.Label, focused))
	}
	return append(out, insetModalLines([]string{"",
		"These are FABRICATED and enter where a real alert enters, so what you hear is",
		"what the pipeline does — not a shortcut that would prove nothing.", ""},
		debugContent)...), focusAt, focusEnd
}

// debugChips is the pinned footer.
func (d Dashboard) debugChips(o render.Opts) []string {
	if len(d.debugScenarios()) == 0 {
		// ↑↓ SCROLL HERE, AND THE CHIPS SAY SO. There is no list in this build,
		// and a window whose text runs past the fold with no advertised way
		// down reads as a window with nothing more in it.
		return []string{"  " + o.KeyCap("↑↓") + " Scroll    " + o.KeyCap("esc") + " Close"}
	}
	return []string{"  " + o.KeyCap("↑↓") + " Choose    " + o.KeyCap("enter") + " Inject    " + o.KeyCap("esc") + " Close"}
}

// handleDebugNav walks the scenarios, wrapping like every other list.
func (d Dashboard) handleDebugNav(act term.Action) Dashboard {
	n := len(d.debugScenarios())
	if n == 0 {
		// A BUILD WITH NO LIST STILL HAS A WINDOW TO READ (FR-5): the keys that
		// choose a scenario scroll the prose instead, bounded by the window's
		// own geometry rather than by a second copy of it.
		switch act {
		case "nav-up":
			d.modalScroll = max(0, d.modalScroll-1)
		case "nav-down":
			d.modalScroll = min(d.modalScroll+1, d.footerModalScrollMax(d.opts()))
		}
		return d
	}
	switch act {
	case "nav-up":
		d.debug.focus = (d.debug.focus - 1 + n) % n
	case "nav-down":
		d.debug.focus = (d.debug.focus + 1) % n
	}
	return d
}

// chooseDebug fires the focused scenario and closes the window.
func (d Dashboard) chooseDebug() (Dashboard, tea.Cmd) {
	sc := d.debugScenarios()
	if d.debug.focus < 0 || d.debug.focus >= len(sc) {
		return d, nil
	}
	inject, key := d.cfg.InjectAlert, sc[d.debug.focus].Key
	d.modal, d.debug = modalNone, debugState{}
	return d, func() tea.Msg {
		if inject != nil {
			inject(key)
		}
		return nil
	}
}
