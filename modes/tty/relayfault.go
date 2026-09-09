package tty

// relayfault.go — the window that says the relay is dead.
//
// A relay that is UP AND BROADCASTING NOTHING answers every check correctly and
// gives the listener silence (UAT 2026-09-04, weatherusa.net). The station used
// to report PLAYING throughout, which for a weather radio is the worst available
// answer: confident and wrong.
//
// MVS-D-76, HUM LEAD: tell the listener, offer the alternatives, and if nobody
// is at the keyboard fall through to a Synth read rather than sit in silence.

import (
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

// RelayCandidate is one thing the listener may tune to instead. Label is what
// they read; Key is what goes back to the radio.
type RelayCandidate struct {
	Label string
	Key   string
}

// RelaySilentMsg raises the window: the mount the deck was playing has gone
// quiet, and these are the candidates left.
type RelaySilentMsg struct {
	Candidates []RelayCandidate
}

const (
	// relayFaultWidth is the mock's width, and relayFaultLabelW its label
	// column — "Recommended" is the longest, and the colons line up under it.
	relayFaultWidth  = 84
	relayFaultLabelW = 11

	// relayFaultPad is the air this window adds on top of the panel's own, so
	// every line clears the border by modalInset on the LEFT.
	relayFaultPad = modalInset - panelSide

	// relayFaultSeconds is how long the window waits for an answer before it
	// takes the fall-through itself (MVS-D-76, ratified).
	relayFaultSeconds = 10
)

// relayFaultContentFor is the room a line has between the two margins, AT THE
// WIDTH THE PANEL WILL ACTUALLY BE.
//
// THE WINDOW DOES NOT STRETCH, BUT IT DOES SHRINK (red team 2026-09-05).
// floatModalFooter clamps the panel to the terminal, so below about 94 columns
// every measurement taken against the mock's 84 is wrong: the footer's right
// margin came out at two cells while the body's was three — the very defect
// this file had just fixed at 84 — the body was wrapped twice, and the second
// wrap went through WrapLines, which cannot preserve a hanging indent.
//
// THE RIGHT MARGIN IS A BOUND, NOT A HOPE (HUM LEAD, UAT 2026-09-05). The prose
// is fixed and happened to clear the edge; the ROWS carry a station's real
// label and run straight into the border.
func relayFaultContentFor(o render.Opts) int {
	return max(min(o.Width, relayFaultWidth)-2-2*modalInset, 8)
}

// relayFaultState is the open window: what it offers, what is focused, and how
// long is left on the clock.
type relayFaultState struct {
	candidates []RelayCandidate
	focus      int
	left       int       // seconds remaining; the footer reads it
	last       time.Time // when the countdown last stepped, so a 300 ms tick counts seconds

	// held stops the countdown for a listener who has started choosing
	// (FR-6.4). Ten seconds is enough to read the four ways out and not enough
	// to read them and decide, so the person the auto-close takes the choice
	// away from is the one who was in the middle of making it. MVS-D-76 is
	// narrowed, not removed: doing nothing still falls through, through the
	// same one door enter uses — what changes is that pressing a key is no
	// longer doing nothing.
	held bool
}

// relayFaultRows is what the window offers, in the mock's order: the two best
// remaining relays, then the fall-through that is always available.
//
// THE FALL-THROUGH IS ALWAYS LAST AND ALWAYS PRESENT. It is what the countdown
// chooses, so a window that could omit it would have a default it did not show.
func (d Dashboard) relayFaultRows() []struct{ label, text, key string } {
	out := []struct{ label, text, key string }{}
	names := []string{"Recommended", "Alternate"}
	for i, c := range d.relayFault.candidates { // bounded by the candidates (P10-02)
		if i >= len(names) {
			break
		}
		out = append(out, struct{ label, text, key string }{names[i], "Tune to " + c.Label, c.Key})
	}
	return append(out, struct{ label, text, key string }{"Fall-Thru", "Read the Watchpost Report instead", ""})
}

// relayFaultInsetLines puts a line inside the window's margins: modalInset cells
// of air on the left, and WRAPPED — never cut — so it never reaches within
// modalInset of the right.
//
// WRAPPED, NOT TRUNCATED (HUM LEAD, UAT 2026-09-05). An earlier pass cut these
// to fit, which is the class UAT 25 ruled out in as many words on WrapLines:
// "floating windows wrap, never truncate". In THIS window it is worse than
// untidy — what gets cut is the address of the station the listener is being
// told to tune to, in the one window that exists because something is already
// wrong. Information must not be lost here of all places.
func relayFaultInsetLines(text string, content int) []string {
	return insetModalLines([]string{text}, content)
}

// relayFaultLines is the body, to the mock exactly. The span the focused row
// occupies goes with it, so the window can be SCROLLED to its own focus.
func (d Dashboard) relayFaultLines(o render.Opts) (out []string, focusAt, focusEnd int) {
	// THE PANEL INSETS BY TWO, so every offset here is the mock's less two.
	// relayFaultInset is that inset named once rather than subtracted at four
	// call sites — the mock is the specification and the arithmetic between it
	// and the string is the part that goes wrong silently.
	content := relayFaultContentFor(o)
	// CENTRED FROM THE WIDTH IT IS DRAWN AT, not from a hand-computed column:
	// the mock's 33 was right at exactly 84 and visibly off at every other.
	title := "*** ERROR ***"
	lines := []string{
		strings.Repeat(" ", max((content-render.Width(title))/2+relayFaultPad, 0)) + title,
		"   TL;DR",
		"   The radio stream is either unavailable or broadcasting silence (not useful)",
		"   Select either a different stream, or Watchpost will attempt to connect to",
		"   the next best relay, or failing that fall through to a Synth Read.",
		"",
		"   NEXT ACTIONS:",
		"",
	}
	body := make([]string, 0, len(lines))
	for i, l := range lines { // bounded by the lines just built (P10-02)
		if i == 0 || l == "" {
			body = append(body, l)
			continue
		}
		body = append(body, relayFaultInsetLines(strings.TrimLeft(l, " "), content)...)
	}
	lines = body
	// THROUGH THE LIST'S ONE OWNER (D-1). render/list.go exists so that every
	// list-shaped surface marks focus the same way — and two more surfaces were
	// found still marking it by hand after this one was fixed (ctrl+d, then the
	// Settings suggestions), so the rule needs the owner, not the intention.
	// This window was marking
	// it with a bare "›" and NO TINT AT ALL — the pointer moved and nothing on
	// the line changed colour, on a window that (until the tick was armed) never
	// redrew itself either. From a listener's chair that reads as arrows that do
	// not work, which is exactly how it was reported.
	//
	// A ROW WRAPS UNDER ITS OWN VALUE, and the rows are separated by a blank
	// line (HUM LEAD, UAT 2026-09-05). A station's full address — callsign,
	// site, state, frequency, distance — is longer than the row, and the
	// continuation belongs under the VALUE rather than back at the margin,
	// where it would read as another way out.
	//
	// The mock's geometry is unchanged: ListMark is two cells, so one space
	// either side puts the pointer and the label in exactly the columns the HUM
	// LEAD drew them in.
	for i, r := range d.relayFaultRows() { // bounded by the rows (P10-02)
		if i > 0 {
			lines = append(lines, "") // the air between the ways out
		}
		focused := i == d.relayFault.focus
		if focused {
			focusAt = len(lines)
		}
		head := o.ListMark(focused) + " " + render.ListLabel(render.PadTo(r.label, relayFaultLabelW)+":", focused) + "  "
		for _, l := range render.WrapHanging(head, r.text, content) { // bounded by the wrap (P10-02)
			lines = append(lines, strings.Repeat(" ", relayFaultPad)+l)
		}
		if focused {
			focusEnd = len(lines) - 1 // the WHOLE row, including what it wrapped to
		}
	}
	return append(lines, "", ""), focusAt, focusEnd
}

// relayFaultChips is the pinned footer: the key on the left, the clock on the
// right. The countdown is SHOWN because it acts on its own — a default that
// fires silently is one the listener cannot choose against.
func (d Dashboard) relayFaultChips(o render.Opts) []string {
	// THE SAME MARGINS AS THE BODY. The footer sat at two cells either side
	// while every line above it sat at three, which is the one place in the
	// window a listener could see the inset was not a rule.
	//
	// THE PAD IS ADDED AFTER THE MEASUREMENT, NOT BEFORE IT. PlainLine TRIMS,
	// so measuring a string that starts with its own padding under-counts by
	// exactly that padding and the right margin comes out one cell short.
	content := relayFaultContentFor(o)
	left := o.KeyCap("esc") + " Close"
	right := "Auto Close in <" + strconv.Itoa(d.relayFault.left) + ">"
	if d.relayFault.held {
		// AND IT SAYS SO. A countdown that stops with no explanation reads as a
		// frozen window, which is the state this window exists to report.
		right = "Auto Close held"
	}
	gap := content - render.Width(render.PlainLine(left)) - render.Width(render.PlainLine(right))
	if gap < 1 {
		gap = 1
	}
	return []string{strings.Repeat(" ", relayFaultPad) + left + strings.Repeat(" ", gap) + right}
}

// stepRelayFault moves the clock on. It takes the WALL TIME rather than counting
// ticks: the tick that drives it is 300 ms and its rate is not this window's
// business, and a window that counted ticks would run at whatever speed the
// shimmer happened to be redrawing at.
func (d Dashboard) stepRelayFault(now time.Time) (Dashboard, bool) {
	if d.modal != modalRelayFault || d.relayFault.left <= 0 || d.relayFault.held {
		return d, false
	}
	if d.relayFault.last.IsZero() {
		d.relayFault.last = now
		return d, false
	}
	if now.Sub(d.relayFault.last) < time.Second {
		return d, false
	}
	d.relayFault.last = now
	d.relayFault.left--
	return d, d.relayFault.left <= 0 // true when the clock has run out
}

// handleRelayFaultNav walks the actions. ↑↓ only: there is nothing else in the
// window to reach, and the wrap matches every other list (UAT 2026-08-30 #11 —
// stopping dead at an end reads as a stuck key).
func (d Dashboard) handleRelayFaultNav(act term.Action) Dashboard {
	// NO EMPTY GUARD. relayFaultRows always appends the fall-through, so n is
	// never zero and the branch that checked it could not be falsified (D-2,
	// red team 2026-09-05). The rule is stated where it is true instead.
	n := len(d.relayFaultRows())
	switch act {
	case "nav-up":
		d.relayFault.focus, d.relayFault.held = (d.relayFault.focus-1+n)%n, true
	case "nav-down":
		d.relayFault.focus, d.relayFault.held = (d.relayFault.focus+1)%n, true
	}
	return d
}

// chooseRelayFault acts on the focused row and closes the window.
//
// THE COUNTDOWN AND ENTER GO THROUGH ONE DOOR. The window's whole purpose is
// that doing nothing has the same shape as choosing Fall-Thru; two paths to it
// would be two chances for them to drift.
func (d Dashboard) chooseRelayFault() (Dashboard, tea.Cmd) {
	// THE FOCUS CANNOT BE OUT OF RANGE, so the range check that used to sit
	// here was a branch with no failing input (D-2): it is reset to zero on
	// open, moved only by a modulo, and openRelayFault now refuses to touch an
	// open window, so the row list cannot shrink underneath it.
	rows := d.relayFaultRows()
	if err := invariant.Check(d.relayFault.focus >= 0 && d.relayFault.focus < len(rows),
		"the focused way out is one the window is offering"); err != nil {
		return d, nil
	}
	return d.takeRelayFault(rows[d.relayFault.focus].key)
}

// takeRelayFault tunes to key, or reads the report when key is empty, and shuts
// the window either way.
func (d Dashboard) takeRelayFault(key string) (Dashboard, tea.Cmd) {
	tune, synth := d.cfg.TuneRelay, d.cfg.ReadReport
	d.modal, d.relayFault = modalNone, relayFaultState{}
	return d, func() tea.Msg {
		switch {
		case key != "" && tune != nil:
			tune(key)
		case synth != nil:
			synth()
		}
		return nil
	}
}

// fallThroughRelayFault is what the clock running out does: the Fall-Thru row,
// whatever the cursor is on. A listener who has walked to another row and not
// pressed enter has not chosen it.
func (d Dashboard) fallThroughRelayFault() (Dashboard, tea.Cmd) { return d.takeRelayFault("") }

// openRelayFault raises the window for a mount that has gone quiet.
//
// A SECOND REPORT WHILE IT IS OPEN IS NOT A SECOND EVENT (UAT 2026-09-05), and
// this is the defect that made the window unusable rather than merely untidy.
//
// The silence detector fires ONCE PER STREAM. The engine falls through to the
// next mount on the tune list, and a station broadcasting silence on every mount
// — which is the outage this whole window was built for — gives each new mount
// its own reader, its own detection and its own report, one about every five
// seconds. Each one used to rebuild the state: THE LISTENER'S CURSOR SNAPPED
// BACK TO THE FIRST ROW AND THE COUNTDOWN RESTARTED AT TEN. From the chair that
// is arrows that do not work and a clock that never moves, which is exactly how
// it was reported — and no test could see it, because every one of them opened
// the window once.
//
// The ways out do not go stale: the candidates are the OTHER mounts on the tune
// list, and one more of them going quiet does not make the rest worse. So the
// open window keeps its cursor and its clock, and a report that arrives after
// the listener has chosen — which closed it — opens a fresh one.
func (d Dashboard) openRelayFault(v RelaySilentMsg) Dashboard {
	if d.modal == modalRelayFault {
		return d
	}
	d.relayFault = relayFaultState{candidates: v.Candidates, left: relayFaultSeconds}
	return d.open(modalRelayFault)
}
