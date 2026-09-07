package tty

// setup_layout.go — HOW the Settings window is arranged: a group becomes a
// block, blocks become one or two columns, and the body scrolls to keep the
// focused row on screen.
//
// It owns geometry and nothing else. What a group SAYS is in the file named for
// it (setup_cast.go, setup_ui.go, setup_relay.go, setup_tones.go, and
// setup_form.go for the three questions that have no group file of their own);
// what a KEY does is in setup.go.
//
// SPLIT FROM setup.go (2026-09-06), a pure move. That file was 1,210 lines
// holding three separable things — the window's state and keys, this, and the
// form's rows — and the package already named its files after what they hold.

import (
	"github.com/branden-thompson/watchpost/platform/render"
)

// setupGroup is a settings-group header: white like the questions, set off by
// blank lines above and below (the section pattern — see docs). More groups
// join as more configurability is added.
func setupGroup(text string) string {
	// ModalTitle, as every other window's section headers use (statusHeader,
	// the Help groups): bold, so the groups read as headings rather than as
	// slightly brighter rows.
	return "  " + render.Tint(text, render.Tok(render.ModalTitle))
}

// setupWidth is the window's width: wide enough for two columns when the
// terminal allows, the one-column floor when it does not — the Help modal's
// rule, applied to the mock's layout (Task 4.4).
//
// Setup needs its own width for the same reason Help does: its content decides
// how wide it wants to be, and a fixed width would either waste a wide terminal
// or force a stack on one that could hold both columns.
func (d Dashboard) setupWidth() int {
	o := d.opts()
	blocks := d.setupBlocks(o)
	if plan, ok := d.columnPlan(blocks, o); ok {
		return plan.width
	}
	return max(setupOneColWidth, min(o.Width, widestBlock(blocks)+panelFrame+panelRail+columnMargin))
}

// setupOneColWidth is the stacked layout's floor: today's window width, which
// the DATA group's lines were written for.
const setupOneColWidth = 78

// setupLines is the Setup window body: the groups, laid in two BALANCED columns
// when they fit and stacked when they do not.
func (d Dashboard) setupLines(o render.Opts) []string {
	lines, _, _ := d.setupBody(o)
	return lines // the chips are a pinned footer, drawn by floatModalFooter
}

// setupBlock is one group's lines, and where the focus sits inside them when it
// is in this group. Height and width travel with the lines because the column
// balance needs both.
type setupBlock struct {
	lines   []string
	w       int // the widest line, measured ONCE — see columnPlan
	at, end int // the focused row's span within lines
	focused bool

	// noteH is how many of lines are a focused row's NOTE.
	//
	// The note appears only under the row the cursor is on, so a block's drawn
	// height depends on where the cursor is. The split is balanced on heights,
	// so that made the SPLIT depend on the cursor too, and the window changed
	// width as you moved through it — 118 cells to 121 at 133x44. The note was
	// already wrapped so it could not widen a block; nothing stopped it
	// reshaping the layout by making one taller. The plan discounts it.
	noteH int
}

// setupBlocks builds every group, in draw order.
//
// One block per group, rather than two hard-coded columns, is what makes the
// balance possible: the layout can put any group in either column because no
// group knows which one it is in.
func (d Dashboard) setupBlocks(o render.Opts) []setupBlock {
	groups := setupGroups()
	out := make([]setupBlock, 0, len(groups))
	for _, g := range groups {
		out = append(out, d.setupBlock(o, g))
	}
	return out
}

// setupBlock builds one group: the blank line, its heading, the blank under it
// and its rows. Each row records where it began, so the scroll reads the same
// geometry the renderer produced rather than a second guess at it.
func (d Dashboard) setupBlock(o render.Opts, g setupGroupID) setupBlock {
	focus := d.setup.focus
	b := setupBlock{lines: []string{"", setupGroup(setupGroupTitle(g)), ""}, focused: setupTable()[focus].group == g}
	at := len(b.lines)
	switch g {
	case groupData:
		b.lines = append(b.lines, d.setupLocationLines(o, setupMark(o, focus == rowLocation))...)
		if focus == rowLocation {
			b.at, b.end = at, len(b.lines)
		}
		b.lines = append(b.lines, "") // the separator between the two DATA rows
		at = len(b.lines)
		b.lines = append(b.lines, d.setupKeyLines(setupMark(o, focus == rowFIRMSKey))...)
		if focus == rowFIRMSKey {
			b.at, b.end = at, len(b.lines)
		}
	case groupUI:
		ui, uiAt := d.uiLines(o)
		b.lines = append(b.lines, ui...)
		b.at, b.end = at+uiAt, len(b.lines)
	case groupEvents:
		b.lines = append(b.lines, d.setupAlertLines(o)...)
		b.at, b.end = at+int(focus-rowEventsAll), len(b.lines)
	case groupTone:
		b.lines = append(b.lines, d.toneLines(o)...)
		b.at, b.end = at+toneLineOf(classRowOrder(), focus), len(b.lines)
	case groupRelay:
		b.lines = append(b.lines, d.relayLines(o)...)
		// The focused ROW, not the whole group: the mark is on one of the two
		// rows and a span covering both puts the scroll's anchor on the wrong
		// line — which reads as a dead keyboard on the row below.
		b.at, b.end = at+relayLineOf(focus), at+relayLineOf(focus)+relayRowH

	case groupCast:
		cast := d.castLines(o)
		b.lines = append(b.lines, cast...)
		b.at = at + castLineOf(focus)
		b = d.appendCastNote(b, focus, widest(cast))
		b.end = len(b.lines) // the note, when there is one, is part of the row
	}
	if !b.focused {
		b.at, b.end = 0, 0
	}
	b.end = max(b.end, b.at)
	// Measured HERE and carried, because the split search asks for it once per
	// candidate: re-measuring made the 80x24 rebuild allocate half again as much
	// as the hand-assigned columns it replaced. render.Width walks the escapes in
	// a styled line, and a block is measured against every split that could put
	// it in a column.
	b.w = widest(b.lines)
	return b
}

// columns is a two-column layout: where the groups were cut, and what the
// window must be to hold them.
type columns struct {
	split        int // blocks[:split] left, blocks[split:] right
	leftW, width int
}

// columnPlan picks the split, and reports whether two columns fit at all.
//
// THE SPLIT IS COMPUTED, not written down. The window had its groups assigned to
// columns by hand, and every group added since made that assignment worse: by
// 0.14.0 four groups stood against one, so the right column ended a dozen rows
// short and the window was a dozen rows taller than it needed to be — which buys
// a scroll rail nobody wanted and pays for those rows on every frame that draws
// them.
//
// The rule is the SPLIT POINT that leaves the two columns most nearly equal in
// height, reading order preserved: groups fill the left column top to bottom,
// then the right. Order matters more than perfect balance — a listener walking
// ↓ through the window must not find the groups shuffled — so this chooses where
// to cut the sequence rather than which groups to pair up.
//
// Every split is costed and the most balanced one that FITS wins, rather than
// the most balanced one full stop: the widest group in a column sets that
// column's width, so where the cut falls changes the total width as well as the
// heights, and a narrower pairing can fit a terminal the best-balanced one
// cannot.
func (d Dashboard) columnPlan(blocks []setupBlock, o render.Opts) (columns, bool) {
	best, bestImbalance, found := columns{}, 0, false
	total := blockHeight(blocks)
	left, leftW := 0, 0
	for split := 1; split < len(blocks); split++ {
		left, leftW = left+blockHeight(blocks[split-1:split]), max(leftW, blocks[split-1].w)
		imbalance := abs(2*left - total)
		if found && imbalance >= bestImbalance {
			continue // a worse balance than one that already fits
		}
		w := twoColumnsWidth(leftW, widestBlock(blocks[split:]), panelChromeFor(left+4, d.modalMax()))
		if w > o.Width {
			continue
		}
		best, bestImbalance, found = columns{split: split, leftW: leftW, width: w}, imbalance, true
	}
	return best, found
}

// blockHeight is the total lines a run of blocks draws; widestBlock the widest
// line among them, from the widths they measured when they were built.

// appendCastNote adds the focused cast row's note, with the blank that belongs
// to it. Extracted from setupBlock, which the P10-01 statement bound caught
// growing past 40 as the relay group landed — the note is a self-contained
// step and reads better with a name on it.
func (d Dashboard) appendCastNote(b setupBlock, focus setupRowID, castW int) setupBlock {
	note := d.castNote(focus)
	if note == "" || !b.focused {
		return b
	}
	b.lines = append(b.lines, "")
	b.noteH++ // the blank belongs to the note, and goes with it
	// Notes belong to the row that raised them and are WRAPPED, never
	// truncated: a reason a listener cannot read is not a reason. They stay no
	// wider than the picker rows, so focusing one cannot flip the two-column
	// layout out from under the reader, and FLUSH with the row labels rather
	// than inset under them (HUM LEAD, UAT 2026-08-30).
	for _, l := range render.WrapLines([]string{note}, max(20, castW-len(castNoteIndent))) { // bounded by the wrap (P10-02)
		b.lines = append(b.lines, castNoteIndent+l)
		b.noteH++
	}
	return b
}

// blockHeight is the height the SPLIT is planned against: the lines a run of
// blocks draws, less any note. A note is transient — it belongs to whichever row
// the cursor is on — and planning against it moves the layout as the cursor
// moves. Use len(b.lines) for what is actually drawn.
func blockHeight(blocks []setupBlock) int {
	n := 0
	for _, b := range blocks {
		n += len(b.lines) - b.noteH
	}
	return n
}

func widestBlock(blocks []setupBlock) int {
	w := 0
	for _, b := range blocks {
		w = max(w, b.w)
	}
	return w
}

// joinBlocks lays a run of blocks end to end and returns the focused row's span
// within the result.
func joinBlocks(blocks []setupBlock) (lines []string, at, end int) {
	for _, b := range blocks {
		if b.focused {
			at, end = len(lines)+b.at, len(lines)+b.end
		}
		lines = append(lines, b.lines...)
	}
	return lines, at, end
}

// setupBody returns the window's lines AND the line the focused row starts on.
//
// The two travel together because they must agree: the scroll keeps the focused
// row on screen (RS-19), and a scroll computed against a different layout than
// the one drawn would put the mark just off the edge — which reads as a dead
// keyboard, since the listener sees nothing move.
func (d Dashboard) setupBody(o render.Opts) (lines []string, focusAt, focusEnd int) {
	blocks := d.setupBlocks(o)
	// NOT modalWidth: that asks this function how wide it wants to be.
	if plan, ok := d.columnPlan(blocks, o); ok {
		left, leftAt, leftEnd := joinBlocks(blocks[:plan.split])
		right, rightAt, rightEnd := joinBlocks(blocks[plan.split:])
		lines = sideBySide(left, right, plan.leftW)
		if anyFocused(blocks[plan.split:]) {
			return lines, rightAt, rightEnd
		}
		return lines, leftAt, leftEnd
	}
	return joinBlocks(blocks)
}

// anyFocused reports whether the focused row is in this run of blocks.
func anyFocused(blocks []setupBlock) bool {
	for _, b := range blocks {
		if b.focused {
			return true
		}
	}
	return false
}

// focusBody is the OPEN WINDOW's lines and the span its focused row occupies.
//
// IT ASKS THE OPEN WINDOW, NOT SETUP (red team 2026-09-05). modalScroll called
// setupBody unconditionally, so every other pinned-footer window scrolled by
// SETUP's focus against SETUP's line count — a permanent zero for the
// relay-fault and ctrl+d windows, because nothing else writes d.modalScroll.
// At 80x24, the app's documented floor, that put every one of the relay-fault
// window's ways out below the fold: the rail drew its arrows, the cursor moved,
// and the screen did not change. The reported "arrows do not work", arrived at
// by geometry instead of by the memo.
//
// at IS -1 WHEN THERE IS NOTHING TO FOCUS (FR-5). A release build compiles the
// injector out, so the shipped ctrl+d window is prose and no list — and a
// focus-following scroll has nothing to follow there. -1 says "this body
// scrolls on its own"; 0 would say "hold the top", which is what pinned the
// whole of that window's message below the fold at 80x24.
func (d Dashboard) focusBody(o render.Opts) (lines []string, at, end int) {
	switch d.modal {
	case modalSetup:
		return d.setupBody(o)
	case modalRelayFault:
		return d.relayFaultLines(o)
	case modalDebug:
		return d.debugLines(o)
	}
	return nil, -1, -1
}

// wrappedIndex is where line i of a body lands once the body is wrapped to w.
// WrapLines is per-line, so the prefix wraps exactly as the prefix of the whole.
func wrappedIndex(lines []string, i, w int) int {
	return len(render.WrapLines(lines[:min(i, len(lines))], w))
}

// focusScroll is the offset arithmetic, with no rendering in it: keep at..end
// inside a window of `window` lines, holding `cur` when the span is already
// there. at < 0 means nothing is focused and `cur` simply rules, clamped.
func focusScroll(n, at, end, window, cur int) int {
	if window <= 0 || n <= window {
		return 0
	}
	if at < 0 {
		return max(0, min(cur, n-window))
	}
	// The whole span, not just the first line: the row AND everything it draws
	// — its hint, its value, its note, its reason — must be on screen (RS-19).
	// If the span is taller than the window the top wins: a row whose head is
	// off screen cannot be identified at all.
	want := min(end, n-1)
	switch {
	case want >= cur+window:
		return max(0, min(want-window+1, at))
	case at < cur:
		return at
	}
	return min(cur, n-window)
}

// castLineOf is a cast row's offset within the group's block, following the
// mock's layout (the blank line and the "...except" line included).
func castLineOf(id setupRowID) int {
	for i, r := range castRowOrder() {
		if r == id {
			return i
		}
	}
	return 0
}

// castRowIndent is the cast rows' left inset and castNoteIndent a note's — past
// the focus mark, so a note sits flush with the label of the row that raised it.
const (
	castRowIndent  = "  "
	castNoteIndent = castRowIndent + "  "
)

// setupChips is the footer: the keys, as the mock draws them, PINNED under the
// scroll window rather than scrolling away with the body (OP-5) — at 80x24 a
// scrolling chip row would be invisible exactly when a lost listener needs it.
func (d Dashboard) setupChips(o render.Opts) []string {
	action := "Next"
	if enterSaves(d.setup.focus) {
		action = "Save"
	}
	segs := []string{
		o.KeyCap("tab") + " Next question",
		o.KeyCap("enter") + " " + action,
		o.KeyCap("↑↓") + " Move",
	}
	// OP-4: with one keyboard rule, `space` operates most of the window's rows
	// and the mock's chip row names it nowhere. It is named here, and only
	// where it does something.
	switch setupTable()[d.setup.focus].kind {
	case rowRadio:
		segs = append(segs, o.KeyCap("space")+" Select")
	case rowCheck:
		segs = append(segs, o.KeyCap("space")+" Toggle")
	}
	if setupTable()[d.setup.focus].picker {
		// OP-1: the mock's chip row does not name the key that changes a voice.
		//
		// The THEME picker names no preview key, because ←→ already preview it:
		// the whole app repaints as the picker moves. Offering `p Preview` there
		// would name a key for something that has already happened.
		if d.setup.focus == rowTheme {
			segs = append(segs, o.KeyCap("←→")+" Theme (live)")
		} else {
			segs = append(segs, o.KeyCap("←→")+" Voice", o.KeyCap("p")+" Preview")
		}
	}
	if d.setup.focus == rowFIRMSKey && d.setup.key != "" {
		segs = append(segs, o.KeyCap("ctrl+r")+" Reveal typed key")
	}
	// The cast and tone groups AUTO-SAVE, so esc closes rather than cancels
	// there; the typed DATA rows still need enter, and esc still discards them.
	// A chip that said Cancel over an auto-saving group would be lying.
	closeLabel := "Cancel"
	if g := setupTable()[d.setup.focus].group; g == groupCast || g == groupTone || g == groupUI {
		closeLabel = "Close"
	}
	segs = append(segs, o.KeyCap("esc")+" "+closeLabel)
	inner := min(o.Width, d.modalWidth()) - 7 - 2 // wrapModal's rail allowance, then the 2-cell inset
	out := []string{""}
	for _, row := range render.WrapSegments(segs, inner, "   ") {
		out = append(out, "  "+row)
	}
	return out
}
