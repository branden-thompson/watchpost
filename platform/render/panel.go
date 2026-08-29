package render

// panel.go — panels, bands, modules, blocks, the scroll panel and the modal overlay compositor. Split from render.go by the quality pass (Q2, pure move).

import (
	"strings"

	lipgloss "charm.land/lipgloss/v2"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// Band renders a full-width section band in the group-header chip style
// (UAT 43: the RECENT / SEARCHED separator becomes a band like the column
// groups). Bracketed form when color is off.
// BandRows is a band at the height the options set: the label row alone
// when thin, else a blank band row above and below it (UAT 2026-08-27).
func (o Opts) BandRows(title, short string, width int, bg Token) []string {
	if o.ThinBands {
		return []string{Band(title, short, width, bg)}
	}
	blank := Band("", "", width, bg)
	return []string{blank, Band(title, short, width, bg), blank}
}

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
	// The title reads bold white against the tile (ModalTitle; HUM LEAD UAT
	// 2026-08-27) — unless the caller tinted it already — while the rule
	// around it keeps the panel's own tone.
	if title == "" {
		head := tl // untitled panel: an unbroken top rule (About window, UAT 68)
		b.WriteString(tint(head+strings.Repeat(hz, max(0, w-displayWidth(head)-1))+tr) + "\n")
	} else {
		if displayWidth(title) > w-6 {
			title = truncate(title, w-6) // a label longer than the window (R5-C-11): cut, never overflowed
		}
		if !strings.Contains(title, "\x1b") {
			title = Tint(title, Tok(ModalTitle))
		}
		pad := max(0, w-displayWidth(tl+hz+hz+" "+title+" ")-1)
		b.WriteString(tint(tl+hz+hz+" ") + title + tint(" "+strings.Repeat(hz, pad)+tr) + "\n")
	}
	for _, line := range strings.Split(content, "\n") {
		line = truncate(line, w-4)
		b.WriteString(tint(vt) + "  " + line + strings.Repeat(" ", max(0, w-4-displayWidth(line))) + tint(vt) + "\n")
	}
	b.WriteString(tint(bl + strings.Repeat(hz, max(0, w-2)) + br))
	return b.String()
}

// Box renders a module as a bordered box (the radio player's facelift, HUM
// LEAD UAT 2026-08-28): the HEAVY box-drawing rules, so the outline reads
// thicker than the light rails inside it; square corners; the rows inset
// boxInset cells from each border; the whole painted fg+bg like a Module.
// Always a frame, whatever the background — the box is the module's
// identity. --ascii keeps the plain +-| forms.
func (o Opts) Box(lines []string, fg, bg string) string { return o.BoxTitled(lines, "", "", fg, bg) }

// BoxTitled is a Box whose top rule carries a title at the left and a stamp
// at the right (the header's facelift, HUM LEAD UAT 2026-08-28):
//
//	┏━━ TITLE ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  STAMP ━━┓
//
// Either may be empty. The caller fits them to BoxRuleWidth; a rule that
// cannot carry both drops the stamp, then the title, so the corners always
// land.
func (o Opts) BoxTitled(lines []string, title, stamp, fg, bg string) string {
	w := max(o.Width, 20)
	tl, tr, bl, br, hz, vt := "┏", "┓", "┗", "┛", "━", "┃"
	if o.ASCII {
		tl, tr, bl, br, hz, vt = "+", "+", "+", "+", "-", "|"
	}
	inset := strings.Repeat(" ", boxInset)
	out := make([]string, 0, len(lines)+2)
	out = append(out, tl+boxRule(hz, title, stamp, w-2)+tr)
	for _, l := range lines {
		out = append(out, vt+inset+PadTo(truncate(l, o.BoxInnerWidth()), o.BoxInnerWidth())+inset+vt) // a row too wide is cut, never the corner (round 4, B-10)
	}
	out = append(out, bl+strings.Repeat(hz, w-2)+br)
	return o.Block(strings.Join(out, "\n"), fg, bg)
}

// boxRule fills w cells of a box's top rule around a title and a stamp:
// two rule cells, a space, the title, a space, the fill, two spaces, the
// stamp, a space, two rule cells — or the plain rule when both are empty.
func boxRule(hz, title, stamp string, w int) string {
	left, right := "", ""
	if title != "" {
		left = hz + hz + " " + title + " "
	}
	if stamp != "" {
		right = "  " + stamp + " " + hz + hz
	}
	if Width(left)+Width(right) > w {
		right = ""
	}
	if Width(left) > w {
		left = ""
	}
	return left + strings.Repeat(hz, w-Width(left)-Width(right)) + right
}

// boxInset is the air between a box's border and its rows, each side.
const boxInset = 3

// BoxRuleWidth is the cell count of a box's top rule between its corners —
// what a title and a stamp share (BoxTitled).
func (o Opts) BoxRuleWidth() int { return max(o.Width, 20) - 2 }

// BoxInnerWidth is the row width inside a Box: the borders and both insets off.
func (o Opts) BoxInnerWidth() int { return max(0, max(o.Width, 20)-2-2*boxInset) }

// BoxHeight is the rendered line count for a Box of n rows: the two rules.
func BoxHeight(n int) int { return n + 2 }

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
	if fg == "" && !BGVisible(bg) { // no tone of its own (the frame's base tone paints it): padded, no SGR — nothing for the compositor to style (round 4, B-01)
		for i, line := range lines {
			lines[i] = PadTo(line, o.Width)
		}
		return strings.Join(lines, "\n")
	}
	// fg and bg are SGR parameters ("38;5;250", "97", "48;2;…"), never a bare
	// 256 index: a bare number here is ambiguous with the basic codes (B-01).
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
