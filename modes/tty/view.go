package tty

// view.go — frame assembly: View, modal sizing and overlay. The map of where
// things happen is docs/where-things-happen.md.

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// View implements tea.Model — a transcription of dashboard-mock-rev2-125col
// (feedback-mock-fidelity: the mock IS the spec). The viewport is terminal-
// width aware: 4-col padding all around (UAT-2C), content resizing per
// UAT-2D/E is delegated to the render seam.
func (d Dashboard) View() tea.View {
	fl := d.layout()                 // once per frame (Q3, L5-F6)
	priority, recent := d.tables(fl) // from the memo on every frame between input changes (Q3)
	o := fl.o
	var b strings.Builder
	b.Grow(len(priority) + len(recent) + 8192) // one buffer for the frame, no growth copies (Q3)
	b.WriteString("\n\n")                      // top padding: 2 blank lines (UAT 10.3)
	b.WriteString(d.header(o))
	b.WriteString("\n")                                            // 0.12.0: the ticker band's top row is the header/ticker separator
	d.writeBody(&b, fl, priority, recent)                          // UAT 57: no footer - every control lives where it acts
	content := frameText(b.String(), viewPadLeft, render.TextBase) // UAT 4.10: base grey; no stray trailing row (UAT 58)
	if overlay := d.modalView(o); overlay != "" {
		content = render.Overlay(content, overlay, d.width) // the one open window (UAT 8.3: lipgloss compositing)
	}
	// AND THE CONFIRMATION OVER THAT (HUM LEAD mock, 2026-09-07). A second
	// layer rather than a swapped body: the mock shows the red box ON TOP of
	// the diagnostics window, and the window underneath is unchanged — which is
	// also why it stays outside the modal memo. It is composited here, on the
	// terminal's own centre, because a box taller than the window it covers
	// would hang off the bottom of a composite centred on the window.
	if box := d.confirmOverlay(o); box != "" {
		content = render.Overlay(content, box, d.width)
	}
	v := tea.NewView(content)
	if x, y, ok := d.cursorAt(content); ok {
		v.Cursor = tea.NewCursor(x, y) // D-81: on the acknowledgement's first line; nowhere else yet (F-205)
	}
	v.AltScreen = true
	v.BackgroundColor = render.WindowBG(d.darkBG) // UAT 10.2: blue-grey window
	return v
}

// confirmOverlay is the window that floats over another window, "" when none.
//
// ONE TODAY: the ctrl+d window's ARE YOU SURE, on the red confirm tile this app
// uses for exactly one thing — a question whose answer cannot be taken back.
func (d Dashboard) confirmOverlay(o render.Opts) string {
	fg, _ := render.ModalTone(d.darkBG)
	// THE CARD WINDOW'S OWN TWO (D-118), asked on top of the card so the operator
	// can still read what they are about to move or discard.
	//
	// THE DROP IS ON THE CONFIRM TILE and the MOVE is not: red is what this app
	// uses for exactly one thing — "a question whose answer cannot be taken back"
	// — and a card in the wrong position is moved again, while a dropped one is
	// in the discard pile.
	if d.modal == modalCard {
		switch d.cardAct {
		case cardActionMove:
			return d.floatModal(o, bcManageWidth, "Change Position", d.movePrompt(o))
		case cardActionDrop:
			return d.floatModalToned(o, bcManageWidth, "Drop from Line-Up", d.dropPrompt(o), fg, render.Tok(render.ConfirmBG))
		}
		return ""
	}
	if d.modal == modalSetup && d.setup.raise != nil { // a longer retention's question (D-231): kept or not, nothing is lost
		return d.floatModal(o, debugConfirmWidth, "", d.raiseLines(o))
	}
	if d.modal == modalSetup && d.setup.confirmClear { // Clear history's (D-177)
		return d.floatModalToned(o, debugConfirmWidth, "", d.historyConfirmLines(o), fg, render.Tok(render.ConfirmBG))
	}
	if d.modal != modalDebug || !d.debug.confirm {
		return ""
	}
	return d.floatModalToned(o, debugConfirmWidth, "", d.debugConfirmLines(o), fg, render.Tok(render.ConfirmBG))
}

// renderModal renders the open window, "" when none (Q6: one switch, one
// overlay). modalView (memo.go) memoises it per input change (FR-10).
func (d Dashboard) renderModal(o render.Opts) string {
	switch d.modal {
	case modalHelp:
		return d.helpModal(o)
	case modalDetails:
		return d.detailsModal(o) // UAT 10.6
	case modalAdd:
		title := "Add Location"
		if d.addMode == "lookup" {
			title = "Lookup Location" // UAT 26.4
		}
		return d.floatModal(o, d.modalWidth(), title, d.addLines(o))
	case modalRemove:
		fg, _ := render.ModalTone(d.darkBG)
		return d.floatModalToned(o, d.modalWidth(), "Remove Location", d.removeLines(o), fg, render.Tok(render.ConfirmBG)) // UAT 26.2
	case modalAlerts:
		return d.alertDetailsModal(o) // UAT 22
	case modalStatus:
		return d.floatModal(o, d.modalWidth(), "Watchpost Status", d.statusLines()) // UAT 24.2; the window covers more than the APIs (0.14.0)
	case modalAbout:
		return d.floatModal(o, d.modalWidth(), "", d.aboutLines(o)) // UAT 68
	case modalMap:
		return d.mapWindow(o)
	case modalPropAck:
		return d.floatModal(o, d.modalWidth(), "Propagation", d.propAckLines(o)) // D-46, D-81
	case modalSevere:
		return d.severeModal(o) // 0.13.0
	case modalCard:
		// THE WINDOW WEARS THE CARD'S GROUND (D-127). `floatModal` is the same
		// call with the standard tone, so a card that carries no ground of its
		// own is unchanged.
		fg, bg := render.ModalTone(d.darkBG)
		if d.cardGround != "" {
			bg = d.cardGround
		}
		return d.floatModalToned(o, d.modalWidth(), d.cardTitleOf(o), d.cardLines(o), fg, bg) // D-88
	case modalSetup, modalDebug, modalRelayFault, modalRequest:
		// The chips are a PINNED FOOTER (OP-5): they render after the scroll
		// window, so at 80x24 they cannot scroll away exactly when a lost
		// listener needs them.
		return d.floatModalFooter(o)
	}
	return ""
}

// modalMax is the modal body height budget (UAT 10.4: expand to fit tall
// terminals, window + rail on short ones).
func (d Dashboard) modalMax() int {
	if d.modal == modalMap {
		// U1-13: about 80% of the terminal, so the dashboard shows round it.
		// D-159: never less than the map needs to keep its 69x12 floor under
		// its own status, notes and radar timeline, up to the terminal's height
		// minus 8.
		base := max(d.height*80/100-5, mapMinBody.Rows+1)
		need := mapMinBody.Rows + d.mapChromeRows(len(d.noteLines(d.mapTextW())))
		return max(5, min(d.height-8, max(base, need)))
	}
	return max(5, d.height-12)
}

// modalWidth is the open modal's width — ONE source for the render sites
// and the scroll bounds.
func (d Dashboard) modalWidth() int {
	// Content-heavy modals stretch to 60% of the terminal on wide screens
	// (UAT 31.2); their base widths are the floor.
	stretch := func(base int) int { return max(base, d.width*60/100) }
	switch d.modal {
	case modalDetails:
		return stretch(85) // location-detail-mock.txt width
	case modalAlerts:
		return stretch(76)
	case modalSetup:
		return d.setupWidth()
	case modalRelayFault:
		return relayFaultWidth // the mock's width exactly; it does not stretch
	case modalDebug:
		return debugWidth
	case modalStatus:
		return d.statusWidth() // providers beside requests when they fit, else the stretch
	case modalAbout:
		_, w, _ := d.aboutPlan(d.opts(), d.opts().Width) // two columns of data sets when they fit (W21.2)
		return w
	case modalMap:
		return d.mapWindowCols()
	case modalPropAck:
		return propAckWidth
	case modalHelp:
		return d.helpWidth(d.opts(), d.opts().Width) // two columns when they fit, else the single column
	case modalCard:
		// THE CARD'S CLASS, WHICH IS THE LOCATION WINDOW'S (D-88). Both are one
		// report read at length, and the console lays its tables out against this
		// window's FLOOR (bcDetailRoom = 85 - 7) — so the number is shared rather
		// than chosen twice.
		return stretch(85)
	case modalSevere:
		return 130 // every column at 133 cols, DETECTION included; the ladder below
	}
	return 56 // help, add/lookup, remove, theme
}

// mapWindowCols is the map window's width: U1-13's about 80% of the terminal,
// at least a 69-column map and its borders (FR-1.4, U1-27), never past the
// frame. Its own function, so the map's sizing reads it without the switch
// above - whose Help arm leads back to the window's height, a cycle P10
// forbids (W14, D-159).
func (d Dashboard) mapWindowCols() int {
	return max(min(d.width-2, max(d.width*80/100, mapMinBody.Cols+2)), 10)
}

// modalLines is the open modal's full body, wrapped exactly as the
// component renders it — scroll bounds always match what is on screen.
//
// EVERY WINDOW HAS A CASE (FR-2.3). A default arm that handed back HELP's
// lines would scroll a window with no case of its own by help's line count —
// a failure mode that GROWS with every window Broadcaster adds, since each new
// one would inherit it silently. So there is no default: the two values that
// are not open windows are named, and a window added later fails to compile
// here instead of quietly reading as help.
func (d Dashboard) modalLines() []string {
	o := d.opts()
	raw, w := []string(nil), d.modalWidth()
	switch d.modal {
	case modalHelp:
		raw = d.helpLines(o)
	case modalDetails:
		raw = d.detailLines()
	case modalAdd:
		raw = d.addLines(o)
	case modalRemove:
		raw = d.removeLines(o)
	case modalAlerts:
		raw = d.alertDetailLines()
	case modalStatus:
		raw = d.statusLines()
	case modalAbout:
		raw = d.aboutLines(o)
	case modalMap:
		raw, o.Width = d.mapBodyLines(), w // measured at the width the map window is drawn at, not the dashboard's
	case modalPropAck:
		raw = d.propAckLines(o)
	case modalSevere:
		raw = d.severeDetailLines(o) // only the record scrolls; the table windows itself
	case modalCard:
		raw = d.cardLines(o) // asked of the console, at THIS window's opts (D-88)
	case modalSetup, modalDebug, modalRelayFault, modalRequest:
		// The pinned-footer windows are laid out at their own box width and
		// their scroll follows the focus. Asked at the dashboard's width they
		// would report a body nobody draws.
		fw, _, _, _ := d.footerModalChrome(o)
		fo := o
		fo.Width, w = min(o.Width, fw), fw
		raw, _, _ = d.focusBody(fo)
	case modalNone, numModals:
		return nil // not open windows
	}
	return d.wrapModal(raw, min(o.Width, w))
}

// wrapModal wraps a modal body for a panel of width w exactly as the
// component will draw it (UAT 68): to the full content width (w-4) when
// everything fits without the scroll rail, else to the rail budget (w-7).
// Single owner — the renderer and the scroll bounds both use it.
func (d Dashboard) wrapModal(lines []string, w int) []string {
	out, _ := d.wrapModalAt(lines, w)
	return out
}

// wrapModalAt is that, and the width it wrapped at, in ONE pass — because the
// scroll has to count in the coordinates the panel draws in (FR-5), and asking
// twice costs a second wrap of the whole body on the memo-miss frame.
func (d Dashboard) wrapModalAt(lines []string, w int) ([]string, int) {
	used := 4 // the two borders and the panel's inset
	if d.modal == modalMap {
		used = 2 // the map window is flush: its borders alone (UAT-1 U1-27)
	}
	if full := render.WrapLines(lines, w-used); len(full) <= d.modalMax() {
		return full, w - used
	}
	return render.WrapLines(lines, w-used-3), w - used - 3
}

// footerModalScrollMax is how far a pinned-footer window scrolls: its wrapped
// body less the window it is drawn in. THE KEYBOARD ASKS THIS, so a window with
// nothing to focus scrolls exactly as far as it has lines and no further —
// the same arithmetic floatModalFooter renders with, not a second copy of it.
func (d Dashboard) footerModalScrollMax(o render.Opts) int {
	width, _, footer, give := d.footerModalChrome(o)
	if width == 0 {
		return 0
	}
	o.Width = min(o.Width, width)
	lines, _, _ := d.focusBody(o)
	body := d.wrapModal(lines, o.Width)
	foot := fitFooter(d.wrapModal(footer, o.Width), give, len(body), d.modalMax())
	return max(0, len(body)-max(1, d.modalMax()-len(foot)))
}

// footerModalChrome is the open pinned-footer window's frame: its width, its
// title, its footer rows, and how many of the footer's first rows are room
// held that may give way to the body. One owner for the windows that pin a
// footer, so the renderer and the keyboard cannot disagree about the geometry.
func (d Dashboard) footerModalChrome(o render.Opts) (width int, title string, footer []string, give int) {
	switch d.modal {
	case modalSetup:
		// SETTINGS, not the mock's "Setup / Configs": setup is what you do once,
		// and this is where the cast, the tones and the alert scope are changed
		// whenever. The CLI keeps `watchpost setup` — the run-it-once meaning is
		// the right one there, and it is the pattern people expect of a tool's
		// first run.
		w := d.modalWidth() // every tab laid out to measure it: asked once
		notices, held := d.setupNotices(o, w)
		return w, "Settings", append(notices, d.setupChips(o)...), held // the notices over the controls (D-237)
	case modalDebug:
		// THE WARNING RIDES THE BORDER (HUM LEAD mock, 2026-09-07), so it cannot
		// scroll away from the control it is about.
		return debugWidth, d.debugTitle(o, min(o.Width, debugWidth)), d.debugChips(o), 0
	case modalRequest:
		return d.modalWidth(), requestTitle, d.requestChips(o), 0
	case modalRelayFault:
		// No title in the frame: the mock puts *** ERROR *** on its own line
		// inside the box, over a plain top border.
		return relayFaultWidth, "", d.relayFaultChips(o), 0
	}
	return 0, "", nil, 0
}

// fitFooter is a footer as drawn under a body of bodyLen lines: its held rows
// give way, from the top, as far as the body needs them, so the window keeps
// one height - its contents' while they fit, the screen's once they do not.
func fitFooter(foot []string, give, bodyLen, maxLines int) []string {
	return foot[min(give, max(0, bodyLen+len(foot)-maxLines)):]
}

// detailsModal renders the floating detail view (location-detail-mock.txt):
// title carries the location + a right-aligned Updated stamp; the body
// lengthens/shortens with terminal height via the ScrollPanel budget.
func (d Dashboard) detailsModal(o render.Opts) string {
	loc := d.selectedLocation()
	title := "Location"
	switch {
	case loc != nil:
		title = loc.Label + " " + loc.Zip // labels are text by the time they are published (the assembler, R5-C-05)
	case d.lookupRef != nil:
		// THE REF THE LOOKUP OPENED THIS WITH, while RECENT catches up (F-42).
		//
		// lookupIndex is "-1 while it waits" for the rebuilt list to carry the
		// row, and until then selectedLocation has nothing to return — so without
		// this the window would title itself the literal "Location" for a frame or
		// more after a lookup, breaking the PTY journey's "Lookup opens Details on
		// Vista FROM THE FIRST FRAME".
		//
		// The field exists for precisely this — "the location a lookup opened
		// Details for, until its data lands" — and the title must consult it, or
		// the modal shows the old top RECENT row instead (UAT 2026-08-28).
		title = d.lookupRef.Label + " " + d.lookupRef.Zip
	}
	if d.snap != nil {
		stamp := "Updated: " + o.Clock.Stamp(dataAsOf(d.snap).Local())
		fill := min(o.Width, d.modalWidth()) - 10 - len([]rune(title)) - len([]rune(stamp))
		if fill > 1 { // the name and the stamp bold white, the fill in the panel's tone (the panel leaves a tinted title as it is)
			title = render.Tint(title, render.Tok(render.ModalTitle)) + " " + strings.Repeat(o.Glyphs().Rule, fill) + " " + render.Tint(stamp, render.Tok(render.ModalTitle))
		}
	}
	return d.floatModal(o, d.modalWidth(), title, d.detailLines())
}

// floatModal is THE floating-window renderer (help, forecast details, About
// and the rest): scrollable panel body, blue-grey tile background per
// terminal mode (UAT 12.4), base-grey text.
func (d Dashboard) floatModal(o render.Opts, width int, title string, lines []string) string {
	fg, bg := render.ModalTone(d.darkBG)
	return d.floatModalToned(o, width, title, lines, fg, bg)
}

// floatModalToned renders a floating window with an explicit tile tone —
// the [A] alert modal carries its severity tint (UAT 22). Body lines WRAP
// to the modal width here, in the component (UAT 25: truncation is not a
// bug any caller can reintroduce).

// floatModalFooter is floatModal with rows PINNED below the scroll window: the
// body scrolls, the footer does not.
//
// Setup is the one window that needs it. Its body is four groups tall and its
// footer names the keys that operate them — a footer that scrolled with the
// body would be missing precisely when the reader has scrolled far enough to
// be lost.
func (d Dashboard) floatModalFooter(o render.Opts) string {
	width, title, footer, give := d.footerModalChrome(o)
	fg, bg := render.ModalTone(d.darkBG)
	o.Width = min(o.Width, width)
	// THE BODY IS BUILT AT THE WIDTH IT IS DRAWN AT. A width passed in by the
	// caller is computed from the unnarrowed opts, which lays setup out for one
	// width and measures it at another.
	lines, at, end := d.focusBody(o)
	wrapped, wrapAt := d.wrapModalAt(lines, o.Width)
	foot := fitFooter(d.wrapModal(footer, o.Width), give, len(wrapped), d.modalMax())
	// The scroll FOLLOWS THE FOCUS: at 80x24 most of the window is off screen,
	// and a focused row the listener cannot see reads as a dead keyboard.
	//
	// IN THE COORDINATES THE PANEL SCROLLS IN (FR-5). The bodies arrive
	// hand-inset to their own window's width and are RE-WRAPPED here, so every
	// line below a paragraph that wrapped moves down. A focus counted on the
	// unwrapped body lands short: at 80x24 a row at unwrapped 11 sits at wrapped
	// 14, an offset of 1 draws lines 1..11, and the cursor moves while the screen
	// does not change — a dead keyboard.
	if at >= 0 {
		at, end = wrappedIndex(lines, at, wrapAt), wrappedIndex(lines, end+1, wrapAt)-1
	}
	scroll := focusScroll(len(wrapped), at, end, max(1, d.modalMax()-len(foot)), d.modalScroll)
	panel := o.ScrollPanelFooter(title, wrapped, foot, scroll, d.modalMax())
	return o.Block(panel, fg, bg)
}

// mapWindow is the map's window: its panel, drawn at the window's own width
// border to border (UAT-1 U1-27), the lines drawn in Update (D-41), with the
// badge a tab on its frame's upper right (D-120).
func (d Dashboard) mapWindow(o render.Opts) string {
	o.Width = d.modalWidth() // the panel at the map window's own width, not the dashboard's (which reserves a rail)
	o.Flush = true
	fg, bg := render.ModalTone(d.darkBG)
	body, title, words := d.wrapModal(d.mapBodyLines(), o.Width), d.mapTitle(), d.mapBadgeWords()
	panel := o.ScrollPanel(title, body, d.modalScroll, d.modalMax())
	rows := strings.Split(panel, "\n")
	if room := tabCol(rows, words) - titleChrome; words != "" && render.Width(title) > room && room > 0 {
		// THE TITLE GIVES WAY, NEVER THE TAB: the tab says the moment and
		// STALE, which are never hidden (FR-5.4).
		dots := o.Glyphs().Ellipsis
		short := strings.TrimRight(render.TruncateCells(title, max(room-render.Width(dots), 1)), " ") + dots
		rows = strings.Split(o.ScrollPanel(short, body, d.modalScroll, d.modalMax()), "\n")
	}
	return o.Block(strings.Join(withTab(rows, words, render.LightBox(o.ASCII)), "\n"), fg, bg)
}

func (d Dashboard) floatModalToned(o render.Opts, width int, title string, lines []string, fg, bg string) string {
	o.Width = min(o.Width, width)
	lines = d.wrapModal(lines, o.Width)
	// Block alone arms BOTH the base-grey text and the tile background and
	// re-arms them after every inner reset. Running TintDefault first would
	// consume the resets Block re-arms on, so every styled span (chips, temp
	// tints) would drop the tile background for the rest of its line.
	return o.Block(o.ScrollPanel(title, lines, d.modalScroll, d.modalMax()), fg, bg)
}

// opts sizes the layout: content width = terminal - 2x2-col padding, minus
// a 2-col gutter reserved for the recent rail. The NAME fill column makes
// the tables span this width exactly (UAT 11.1), so every section stays
// flush and aligned at any terminal size.
func (d Dashboard) opts() render.Opts {
	raw := max(d.width-viewPadLeft-viewPadRight, 40)
	return render.Opts{Width: raw - 2, Units: d.units, Clock: d.clockFmt, Frame: d.frame, ASCII: d.cfg.ASCII} // --ascii (A11-10, Q3)
}

// tableBreakpoint is the total table rows (favourites + recent window)
// the full layout must keep before the modules minimize (UAT 49).
const tableBreakpoint = 20

// frameText finishes the frame in one pass (Q3: TrimRight, indent and
// TintDefault together): trailing newlines dropped, every
// non-empty line indented by pad, and, with colour on, the base grey armed
// at the start and re-armed after every SGR reset so explicitly-tinted
// spans keep their colours and everything else reads grey (UAT 4.10).
func frameText(s string, pad int, tok render.Token) string {
	s = strings.TrimRight(s, "\n")
	base := ""
	if render.ColorOn() {
		base = "\x1b[0;" + render.FgSGR(render.Tok(tok)) + "m" // a bare index or a truecolor token alike (the Light theme's TextBase is truecolor)
	}
	const reset = "\x1b[0m"
	var b strings.Builder
	b.Grow(len(s) + len(base)*(2+strings.Count(s, reset)) + pad*(1+strings.Count(s, "\n")))
	b.WriteString(base)
	first := true
	for line := range strings.SplitSeq(s, "\n") {
		if !first {
			b.WriteByte('\n')
		}
		first = false
		if line != "" {
			for range pad {
				b.WriteByte(' ')
			}
		}
		if base == "" {
			b.WriteString(line)
			continue
		}
		for i := strings.Index(line, reset); i >= 0; i = strings.Index(line, reset) {
			b.WriteString(line[:i])
			b.WriteString(base)
			line = line[i+len(reset):]
		}
		b.WriteString(line)
	}
	if base != "" {
		b.WriteString(reset)
	}
	return b.String()
}
