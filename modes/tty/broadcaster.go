package tty

// broadcaster.go — the operator console (P1, FR-2).
//
// THE SECOND SURFACE. Observer is built for a person watching the weather;
// Broadcaster is built for a person putting it on the air. It reads the
// schedule the Director PUBLISHES and never keeps a second copy — a console
// that painted its own idea of the running order would be asserting an order
// the schedule does not follow, which is the release's UI-integrity risk.
//
// P1 IS THE SHELL. The lanes, the breakpoints and the size notice arrive here;
// the station state is P2 and operator control is P4.

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/plaintext"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

// LineupMsg carries the schedule the Director PUBLISHED.
//
// IT CARRIES THE LINEUP BY VALUE, as the effect that produces it does, and for
// the same reason: the pump dispatches asynchronously, so a console that
// fetched the current lineup when the message arrived would get whatever it
// had become by then, not what was published.
type LineupMsg struct{ Lineup lineup.Lineup }

// mainTrackSlots is how many cards the rolling main-track view shows (FR-3.1).
const mainTrackSlots = 10

// Broadcaster is the operator console's model.
type Broadcaster struct {
	width, height int
	darkBG        bool

	// lineup is the last PUBLISHED schedule. It is never mutated here — the
	// console names an intent and the Director owns the order (D-23).
	lineup lineup.Lineup
}

// NewBroadcaster builds the console.
func NewBroadcaster() Broadcaster { return Broadcaster{} }

func (b Broadcaster) Init() tea.Cmd { return nil }

// Update takes the surface's own messages. The program-scoped ones arrive
// through the Router's fan-out, which is why they are handled here as well as
// in Observer: BOTH surfaces must know the size, including while inactive.
func (b Broadcaster) Update(msg tea.Msg) (Broadcaster, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		b.width, b.height = v.Width, v.Height
	case tea.BackgroundColorMsg:
		b.darkBG = v.IsDark()
	case LineupMsg:
		b.lineup = v.Lineup
	}
	return b, nil
}

// View renders the console: the priority track, the main track, the bed.
//
// STRUCTURE, NOT YET THE MOCK'S PIXELS. P1 is the shell — it renders the three
// lanes in the mock's own vocabulary (the handles, the badges, the lane names)
// from the PUBLISHED schedule. Byte-exact fidelity to
// `01-objectives/mock-broadcaster-v1.txt` is NOT claimed here and is not
// pretended: the mock is 150x74 with a column budget this does not yet honour,
// and layout is the HUM LEAD's pass.
func (b Broadcaster) View() tea.View {
	lines := b.lanes()
	if b.tooSmall() {
		lines = b.notice()
	}
	v := tea.NewView(b.clamp(lines))
	v.AltScreen = true
	v.BackgroundColor = render.WindowBG(b.darkBG)
	return v
}

// minSize is the floor below which the console refuses to draw (FR-7.3).
//
// 44 LINES IS MEASURED, NOT CHOSEN: the fixed chrome plus one readable card,
// counted off the mock at DISCOVER wave 1. The mock draws 74, so 30 of its
// lines exist to show MORE cards, not to work at all.
func (b Broadcaster) minSize() (cols, rows int) { return bcMinCols, bcMinRows }

const (
	// bcMinRows is MEASURED: fixed chrome plus one readable card, counted off
	// the mock at wave 1. The mock draws 74, so 30 of its lines show MORE
	// cards rather than making it work.
	bcMinRows = 44

	// bcMinCols is the start of the platform's STANDARD class. Below it the
	// console has no honest layout — and D-13 chose the platform vocabulary
	// precisely so this number is a boundary someone already reasoned about
	// rather than one invented here.
	bcMinCols = 80
)

// tooSmall reports whether the terminal is below the floor.
func (b Broadcaster) tooSmall() bool {
	c, r := b.minSize()
	return b.width < c || b.height < r
}

// clamp cuts the frame to the terminal, in BOTH directions.
//
// SILENT OVERFLOW IS A DEFECT, NOT A DEGRADATION (FR-7.3). F-55 measured the
// other surface rendering 57 cells into a 20-cell terminal with no clamp and
// no notice; this one does not inherit that. The clamp is applied to the
// NOTICE as well as to the lanes — a notice that overflows would let a sweep
// report zero overflows, which is the anti-solution the red team drove
// through M5.
func (b Broadcaster) clamp(lines []string) string {
	if b.height > 0 && len(lines) > b.height {
		lines = lines[:b.height]
	}
	if b.width > 0 {
		for i, l := range lines {
			lines[i] = render.TruncateCells(l, b.width)
		}
	}
	return strings.Join(lines, "\n")
}

// notice is what the operator sees below the floor.
func (b Broadcaster) notice() []string {
	c, r := b.minSize()
	return []string{
		"TERMINAL TOO SMALL",
		"",
		"Broadcaster needs " + strconv.Itoa(c) + "x" + strconv.Itoa(r) + ".",
		"This terminal is " + strconv.Itoa(b.width) + "x" + strconv.Itoa(b.height) + ".",
	}
}

// lanes builds the three lanes from the last published schedule.
func (b Broadcaster) lanes() []string {
	out := []string{"WATCHPOST Broadcaster"}
	out = append(out, "")

	// THE PRIORITY TRACK IS DRAWN FIRST because it DRAINS first, in every
	// state. Drawing it below the rotation would put the lane that interrupts
	// everything under the lane it interrupts.
	out = append(out, "PRIORITY")
	rail := b.lineup.Cards(lineup.AlertRail)
	if len(rail) == 0 {
		out = append(out, "  (clear)")
	}
	for _, c := range rail {
		out = append(out, "  "+cardRow(c, "T", "PRIORITY", term.BreakpointFor(b.width) >= term.BreakWide))
	}
	out = append(out, "")

	// THE BREAKPOINT SELECTS THE LAYOUT (FR-7.1, D-13). Not a call whose
	// result is discarded: the class decides what a lane row can afford, and
	// changing the class changes the frame.
	wide := term.BreakpointFor(b.width) >= term.BreakWide
	if wide {
		out = append(out, "SCHEDULED LINE UP")
	} else {
		out = append(out, "LINE UP")
	}
	main := b.lineup.Cards(lineup.MainTrack)
	// A ROLLING VIEW OF TEN (FR-3.1). An eleventh card exists in the schedule
	// and does not reach the frame; the console shows a window onto the
	// lineup, never a second copy of it.
	if len(main) > mainTrackSlots {
		main = main[:mainTrackSlots]
	}
	if len(main) == 0 {
		out = append(out, "  (nothing scheduled)")
	}
	for i, c := range main {
		out = append(out, "  "+cardRow(c, strconv.Itoa(i), "STANDARD", wide))
	}
	out = append(out, "")
	out = append(out, "BED   (no relay tuned)")
	return out
}

// cardRow is one lane row: what it is, and the handle that addresses it.
func cardRow(c lineup.Card, handle, badge string, wide bool) string {
	// A narrower class affords less headline. The width is not decoration: it
	// is what the class BUYS, and it is why a discarded classify call would
	// not satisfy FR-7.1.
	w := 60
	if !wide {
		w = 30
	}
	// THE HEADLINE, NOT THE SUBJECT. The headline is what the card is ABOUT in
	// the words a person reads; the subject is its key.
	return render.PadTo(render.TruncateCells(plaintext.Text(c.Headline), w), w) + " •" + badge + "•  [ " + handle + " ]"
}
