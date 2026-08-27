package tty

// view.go — frame assembly: View, modal sizing and overlay. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

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
	b.WriteString("\n\n")                      // top padding: 2 blank lines (UAT 10.3, was 3 per UAT-3.1)
	b.WriteString(d.header(o))
	b.WriteString("\n\n")                                          // UAT-3.2: blank line between header and alert section
	d.writeBody(&b, fl, priority, recent)                          // UAT 57: no footer - every control lives where it acts
	content := frameText(b.String(), viewPadLeft, render.TextBase) // UAT 4.10: base grey; no stray trailing row (UAT 58)
	if d.showHelp {
		// UAT 8.3: help floats over the dashboard (lipgloss compositing).
		content = render.Overlay(content, d.helpModal(o), d.width)
	}
	if d.showDetails {
		content = render.Overlay(content, d.detailsModal(o), d.width) // UAT 10.6
	}
	if d.showAdd {
		title := "Add Location"
		if d.addMode == "lookup" {
			title = "Lookup Location" // UAT 26.4
		}
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), title, d.addLines(o)), d.width)
	}
	if d.showRemove {
		fg, _ := render.ModalTone(d.darkBG)
		content = render.Overlay(content, d.floatModalToned(o, d.modalWidth(), "Remove Location",
			d.removeLines(o), fg, render.Tok(render.ConfirmBG)), d.width) // UAT 26.2
	}
	if d.showAlerts {
		content = render.Overlay(content, d.alertDetailsModal(o), d.width) // UAT 22
	}
	if d.showStatus {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "API Status", d.statusLines()), d.width) // UAT 24.2
	}
	if d.showAbout {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "", d.aboutLines()), d.width) // UAT 68
	}
	if d.showTheme {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "Color Theme", d.themeLines(o)), d.width) // UAT 53
	}
	if d.showVoice {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "Correspondent Voice", d.voiceLines(o)), d.width) // UAT 84
	}
	if d.showSetup {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "Setup", d.setupLines(o)), d.width) // UAT 100
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = render.WindowBG(d.darkBG) // UAT 10.2: blue-grey window
	return v
}

// modalMax is the modal body height budget (UAT 10.4: expand to fit tall
// terminals, window + rail on short ones).
func (d Dashboard) modalMax() int { return max(5, d.height-12) }

// modalWidth is the open modal's width — ONE source for the render sites
// and the scroll bounds.
func (d Dashboard) modalWidth() int {
	// Content-heavy modals stretch to 60% of the terminal on wide screens
	// (UAT 31.2); their base widths are the floor.
	stretch := func(base int) int { return max(base, d.width*60/100) }
	switch {
	case d.showDetails:
		return stretch(85) // location-detail-mock.txt width
	case d.showAlerts:
		return stretch(76)
	case d.showStatus:
		return stretch(68)
	case d.showAbout:
		return aboutWidth
	case d.showVoice, d.showSetup:
		return 68 // the four chip controls fit on one line (UAT 86); the FIRMS address fits (UAT 100)
	case d.showAdd, d.showRemove, d.showTheme:
		return 56
	}
	return 56 // help
}

// modalLines is the open modal's full body, wrapped exactly as the
// component renders it — scroll bounds always match what is on screen.
func (d Dashboard) modalLines() []string {
	var raw []string
	switch {
	case d.showDetails:
		raw = d.detailLines()
	case d.showAlerts:
		raw = d.alertDetailLines()
	case d.showStatus:
		raw = d.statusLines()
	case d.showAbout:
		raw = d.aboutLines()
	default:
		raw = d.helpLines(d.opts())
	}
	o := d.opts()
	return d.wrapModal(raw, min(o.Width, d.modalWidth()))
}

// wrapModal wraps a modal body for a panel of width w exactly as the
// component will draw it (UAT 68): to the full content width (w-4) when
// everything fits without the scroll rail, else to the rail budget (w-7).
// Single owner — the renderer and the scroll bounds both use it.
func (d Dashboard) wrapModal(lines []string, w int) []string {
	if full := render.WrapLines(lines, w-4); len(full) <= d.modalMax() {
		return full
	}
	return render.WrapLines(lines, w-7)
}

// detailsModal renders the floating detail view (location-detail-mock.txt):
// title carries the location + a right-aligned Updated stamp; the body
// lengthens/shortens with terminal height via the ScrollPanel budget.
func (d Dashboard) detailsModal(o render.Opts) string {
	loc := d.selectedLocation()
	title := "Location"
	if loc != nil {
		title = loc.Label + " " + loc.Zip
	}
	if d.snap != nil {
		stamp := "Updated: " + dataAsOf(d.snap).Local().Format("01/02/2006 15:04:05 MST")
		fill := min(o.Width, d.modalWidth()) - 10 - len([]rune(title)) - len([]rune(stamp))
		if fill > 1 {
			title = title + " " + strings.Repeat("─", fill) + " " + stamp
		}
	}
	return d.floatModal(o, d.modalWidth(), title, d.detailLines())
}

// floatModal is THE floating-window renderer (help, forecast details, and
// the coming About/setup modals): scrollable panel body, blue-grey tile
// background per terminal mode (UAT 12.4), base-grey text.
func (d Dashboard) floatModal(o render.Opts, width int, title string, lines []string) string {
	fg, bg := render.ModalTone(d.darkBG)
	return d.floatModalToned(o, width, title, lines, fg, bg)
}

// floatModalToned renders a floating window with an explicit tile tone —
// the [A] alert modal carries its severity tint (UAT 22). Body lines WRAP
// to the modal width here, in the component (UAT 25: truncation is not a
// bug any caller can reintroduce).
func (d Dashboard) floatModalToned(o render.Opts, width int, title string, lines []string, fg, bg string) string {
	o.Width = min(o.Width, width)
	lines = d.wrapModal(lines, o.Width)
	// Block alone arms BOTH the base-grey text and the tile background and
	// re-arms them after every inner reset. Running TintDefault first was
	// the session-12 color bug: it consumed the resets Block re-arms on, so
	// every styled span (chips, temp tints) dropped the tile background for
	// the rest of its line.
	return o.Block(o.ScrollPanel(title, lines, d.modalScroll, d.modalMax()), fg, bg)
}

// opts sizes the layout: content width = terminal - 2x2-col padding, minus
// a 2-col gutter reserved for the recent rail. The NAME fill column makes
// the tables span this width exactly (UAT 11.1), so every section stays
// flush and aligned at any terminal size.
func (d Dashboard) opts() render.Opts {
	raw := max(d.width-viewPadLeft-viewPadRight, 40)
	return render.Opts{Width: raw - 2, Units: d.units, Frame: d.frame, ASCII: d.cfg.ASCII} // --ascii (A11-10, Q3)
}

// tableBreakpoint is the total table rows (favourites + recent window)
// the full layout must keep before the modules minimize (UAT 49).
const tableBreakpoint = 20

// frameText finishes the frame in one pass (Q3: was three copies —
// TrimRight, indent, TintDefault): trailing newlines dropped, every
// non-empty line indented by pad, and, with colour on, the base grey armed
// at the start and re-armed after every SGR reset so explicitly-tinted
// spans keep their colours and everything else reads grey (UAT 4.10).
func frameText(s string, pad int, tok render.Token) string {
	s = strings.TrimRight(s, "\n")
	base := ""
	if render.ColorOn() {
		base = "\x1b[0;38;5;" + render.Tok(tok) + "m"
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
