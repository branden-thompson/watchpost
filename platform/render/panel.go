package render

// panel.go — panels, bands, modules, blocks, the scroll panel and the modal overlay compositor. Split from render.go by the quality pass (Q2, pure move).

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// Band renders a full-width section band in the group-header chip style
// (UAT 43: the RECENT / SEARCHED separator becomes a band like the column
// groups). Bracketed form when color is off.
func Band(title, short string, width int, bg Token) string {
	if colorOn() {
		return sgrRaw(" "+centered(title, short, width-2)+" ", Tok(GroupText)+";"+Tok(bg))
	}
	return bracketTitle(title, short, width)
}

// centered fits a title into inner cells, degrading gracefully: full title,
// then its short form (UAT 16.1: "E X T E N D E D" instead of a truncated
// "EXTENDEDFORECA…"), then unspread lettering, then truncate.
func centered(title, short string, inner int) string {
	if displayWidth(title) > inner && short != "" {
		title = short
	}
	if displayWidth(title) > inner {
		title = strings.ReplaceAll(title, " ", "") // unspread the lettering
	}
	if displayWidth(title) > inner {
		title = truncate(title, max(0, inner))
	}
	pad := inner - displayWidth(title)
	return strings.Repeat(" ", pad/2) + title + strings.Repeat(" ", pad-pad/2)
}

// bracketTitle is the color-off band form: [ title ] across width cells.
func bracketTitle(title, short string, width int) string {
	return "[" + centered(title, short, width-2) + "]"
}

// AlertBanner renders the M-V1 conditional alert module: severity ALWAYS a
// text label; paging "NN / MM" per the mock.
func (o Opts) AlertBanner(a snapshot.Alert, index, total int) string {
	warn := "⚠"
	if o.ASCII {
		warn = "!"
	}
	// Title carries the event (uppercased per mock); the severity label stays
	// a verbatim lowercase token — same shape as report mode ("ALERT [severe]")
	// so both surfaces read identically (R-12a).
	body := fmt.Sprintf("[%s] %s", a.Severity, a.Headline)
	pager := fmt.Sprintf("%02d / %02d Alerts", index, total)
	return o.Panel(warn+" "+strings.ToUpper(a.Event), body+"\n"+pager)
}

// Panel renders a rounded-border box with a title (mock anatomy), width-bound.
func (o Opts) Panel(title, content string) string { return o.PanelColored(title, content, "") }

// PanelColored is Panel with the border + title tinted (bare 256 fg code) —
// the alert module reads red or yellow by statement class (UAT 4.6).
func (o Opts) PanelColored(title, content, fg string) string {
	w := max(o.Width, 20)
	tl, tr, bl, br, hz, vt := "┌", "┐", "└", "┘", "─", "│" // square corners (UAT 10.5)
	if o.ASCII {
		tl, tr, bl, br, hz, vt = "+", "+", "+", "+", "-", "|"
	}
	tint := func(t string) string {
		if fg == "" {
			return t
		}
		return rendering.WrapSGR(t, fg)
	}
	var b strings.Builder
	head := tl + hz + hz + " " + title + " "
	if title == "" {
		head = tl // untitled panel: an unbroken top rule (About window, UAT 68)
	}
	pad := w - displayWidth(head) - 1
	if pad < 0 {
		pad = 0
	}
	b.WriteString(tint(head+strings.Repeat(hz, pad)+tr) + "\n")
	for _, line := range strings.Split(content, "\n") {
		line = truncate(line, w-4)
		b.WriteString(tint(vt) + "  " + line + strings.Repeat(" ", max(0, w-4-displayWidth(line))) + tint(vt) + "\n")
	}
	b.WriteString(tint(bl + strings.Repeat(hz, max(0, w-2)) + br))
	return b.String()
}

// ModuleInnerWidth is the content width inside a module for its tone:
// visible-bg modules inset 3 cols each side (UAT 19.1a); hidden-bg modules
// run flush with the header edges (19.1b).
func (o Opts) ModuleInnerWidth(bg string) int {
	if BGVisible(bg) {
		return o.Width - 6
	}
	return o.Width
}

// Module renders a module block with the global inset policy (UAT 19.1):
// visible-bg tones get the 3-col left/right inset plus a padded blank line
// top and bottom; hidden-bg tones render flush, no padding lines — the
// single blank between modules comes from the layout (19.1c).
func (o Opts) Module(lines []string, fg, bg string) string {
	if !BGVisible(bg) {
		return o.Block(strings.Join(lines, "\n"), fg, bg)
	}
	padded := make([]string, 0, len(lines)+2)
	padded = append(padded, "")
	for _, l := range lines {
		padded = append(padded, "   "+l)
	}
	padded = append(padded, "")
	return o.Block(strings.Join(padded, "\n"), fg, bg)
}

// ModuleHeight is the rendered line count for a module of n content lines.
func ModuleHeight(n int, bg string) int {
	if BGVisible(bg) {
		return n + 2
	}
	return n
}

// Block renders content as a borderless full-width background block (UAT
// 5.4): every line is padded to the width and painted fg+bg; inner SGR
// resets (chips, temp colors) re-arm the block tone so the background never
// tears mid-line, and every line closes clean so it never bleeds past the
// block. With color off the content passes through untinted (R-12a: the
// module's text signals carry the meaning).
func (o Opts) Block(content, fg, bg string) string {
	lines := strings.Split(content, "\n")
	if !colorOn() {
		for i, line := range lines {
			lines[i] = strings.TrimRight(line, " ")
		}
		return strings.Join(lines, "\n")
	}
	for i, line := range lines {
		line = PadTo(line, o.Width)
		line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0;"+fg+";"+bg+"m")
		lines[i] = "\x1b[" + fg + ";" + bg + "m" + line + "\x1b[0m"
	}
	return strings.Join(lines, "\n")
}

// ScrollPanel renders a floating panel whose body windows to maxLines with
// a visible right-edge scroll rail (UAT 10.4: users must SEE that it
// scrolls on short terminals); when everything fits, it expands and the
// rail disappears.
func (o Opts) ScrollPanel(title string, lines []string, scroll, maxLines int) string {
	if maxLines <= 0 {
		maxLines = 1
	}
	if len(lines) <= maxLines {
		return o.PanelColored(title, strings.Join(lines, "\n"), "")
	}
	maxScroll := len(lines) - maxLines
	scroll = max(0, min(scroll, maxScroll))
	win := make([]string, maxLines)
	inner := o.Width - 7 // panel chrome (4) + rail col + gap
	thumb := 1
	if maxLines > 3 {
		thumb = 1 + scroll*(maxLines-3)/max(1, maxScroll)
	}
	for i := range maxLines {
		glyph := "│"
		switch i {
		case 0:
			glyph = "▲"
		case maxLines - 1:
			glyph = "▼"
		case thumb:
			glyph = "█"
		}
		win[i] = PadTo(truncate(lines[scroll+i], inner), inner) + " " + glyph
	}
	return o.PanelColored(title, strings.Join(win, "\n"), "")
}

// Overlay floats modal centered over base via lipgloss v2 Canvas/Layer
// compositing (UAT 8.3: the '?' help floats over the dashboard instead of
// replacing it). The modal's own cells - spaces included - overwrite the
// base, so the panel is opaque.
func Overlay(base, modal string, termWidth int) string {
	// v2.0.2 note: Layer.Draw ignores X/Y (positioning lives in the
	// Compositor) and Layer.Width/Height are unset fields - measure with
	// lipgloss.Width/Height and composite through the Compositor. Centering
	// uses the TERMINAL width (UAT 35.4): the base's widest line is not the
	// viewport, so a long row must never shove the modal off-center.
	if termWidth <= 0 {
		termWidth = lipgloss.Width(base)
	}
	x := max(0, (termWidth-lipgloss.Width(modal))/2)
	y := max(0, (lipgloss.Height(base)-lipgloss.Height(modal))/2)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(modal).X(x).Y(y).Z(1),
	).Render()
}
