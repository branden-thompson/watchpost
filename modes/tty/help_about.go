package tty

// help_about.go — the help and about modals. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

// helpModal renders live from the merged KeyMap so it is truthful after any
// swap (D-15 guarantee 3).
func (d Dashboard) helpModal(o render.Opts) string {
	return d.floatModal(o, d.modalWidth(), "Watchpost Help", d.helpLines(o)) // UAT 8.3/10.1/10.4
}

// The Help window's geometry (HUM LEAD UAT 2026-08-28): a blank line of air
// under the title; the groups in two columns when the terminal is wide
// enough, one column with the panel's scroll when it is not — each group
// rolls as a unit, never split across columns. columnGap is the air between
// the columns; helpOneColWidth is the single-column window (the 0.12.0
// width); the two-column window is sized from the widest group line.
const helpOneColWidth = 56

// helpBlock is one group as rendered: the header line, then its rows.
type helpBlock struct{ lines []string }

// helpBlocks renders the merged KeyMap as groups (truthful after any swap -
// D-15 guarantee 3): the registry's groups in order, then OTHER for any
// action no group names (a future binding: still listed, never lost).
func (d Dashboard) helpBlocks() []helpBlock {
	var blocks []helpBlock
	seen := map[term.Action]bool{}
	row := func(bind term.Binding, act term.Action) string {
		return fmt.Sprintf("   %-12s - %s", strings.Join(bind.Keys, ", "), orDefault(bind.Help, string(act)))
	}
	for _, g := range helpGroups() {
		var rows []string
		for _, act := range g.actions {
			if bind, ok := d.keys[act]; ok {
				rows = append(rows, row(bind, act))
				seen[act] = true
			}
		}
		if len(rows) > 0 {
			blocks = append(blocks, helpBlock{append([]string{" " + render.Tint(g.name, render.Tok(render.ModalTitle))}, rows...)})
		}
	}
	var other []string
	for act, bind := range d.keys {
		if !seen[act] {
			other = append(other, row(bind, act))
		}
	}
	if len(other) > 0 {
		sort.Strings(other)
		blocks = append(blocks, helpBlock{append([]string{" " + render.Tint("OTHER", render.Tok(render.ModalTitle))}, other...)})
	}
	return blocks
}

// helpColumnWidth is the widest group line — one column's width.
func helpColumnWidth(blocks []helpBlock) int {
	lines := make([][]string, len(blocks))
	for i, b := range blocks {
		lines[i] = b.lines
	}
	return widest(lines...)
}

// helpPlan decides the layout for a terminal content width: two columns
// when they fit (the chrome charged by whether the two-column body scrolls),
// else the single column; it returns the window's width with it.
func (d Dashboard) helpPlan(avail int) (twoCol bool, width int) {
	blocks := d.helpBlocks()
	colW := helpColumnWidth(blocks)
	body := 1 + len(helpTwoColumns(blocks, colW)) + 4 // air · columns · legend (≤2) · blank · chips
	if w := twoColumnsWidth(colW, colW, panelChromeFor(body, d.modalMax())); w <= avail {
		return true, w
	}
	return false, helpOneColWidth
}

// helpWidth is the window's width for a terminal content width.
func (d Dashboard) helpWidth(avail int) int {
	_, w := d.helpPlan(avail)
	return w
}

// helpLines renders the modal body: the air under the title, the groups in
// one or two columns, the row-marks legend, the chips.
func (d Dashboard) helpLines(o render.Opts) []string {
	blocks := d.helpBlocks()
	twoCol, _ := d.helpPlan(o.Width)
	lines := []string{""} // air under the title (UAT 2026-08-28 item 1)
	if twoCol {
		lines = append(lines, helpTwoColumns(blocks, helpColumnWidth(blocks))...)
	} else {
		for _, b := range blocks {
			lines = append(lines, b.lines...)
			lines = append(lines, "")
		}
	}
	// Row marks legend (red-team B5 U8): the glyphs beside a location, in words.
	g := o.Glyphs() // the table's own set, ASCII included (A11-10)
	lines = append(lines, fmt.Sprintf(" Row marks: %s playing   %s on repeat   %s%s%s recent quake (below/felt/significant)   n%s fires nearby (bold = burning hard)   n%s alerts",
		g.Play, g.Repeat, g.Seismic[0], g.Seismic[1], g.Seismic[2], g.Fire, g.Alert))
	return append(lines, "", "  "+o.Controls("   ", render.Ctl("esc", "Close"), render.Ctl("↑↓", "Scroll"))) // chips like every other modal (UAT 68.2)
}

// helpTwoColumns lays the groups out side by side: the registry order is
// kept and split once, at the point that balances the two columns' heights
// best, so every group stays whole and the columns read top to bottom.
func helpTwoColumns(blocks []helpBlock, colW int) []string {
	height := func(bs []helpBlock) int {
		n := 0
		for _, b := range bs {
			n += len(b.lines) + 1 // the blank after each group
		}
		return n
	}
	split, best := 1, -1
	for k := 1; k < len(blocks); k++ {
		if diff := abs(height(blocks[:k]) - height(blocks[k:])); best < 0 || diff < best {
			split, best = k, diff
		}
	}
	column := func(bs []helpBlock) []string {
		var out []string
		for _, b := range bs {
			out = append(out, b.lines...)
			out = append(out, "")
		}
		return out
	}
	return sideBySide(column(blocks[:split]), column(blocks[split:]), colW)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// helpGroup is one section of the Help window: the app's features, in the
// order a person meets them (HUM LEAD UAT 2026-08-27).
type helpGroup struct {
	name    string
	actions []term.Action
}

// helpGroups is the one owner of the grouping; a binding's group is its
// action, so a rebound key stays in its section (D-15: keys are data).
func helpGroups() []helpGroup {
	return []helpGroup{ // NAVIGATE and RADIO first: the two tall groups make the left column of the two-column layout (UAT mock 2026-08-28)
		{"NAVIGATE", []term.Action{"nav-up", "nav-down", "details", "alert-details", "severe", "alert-prev", "alert-next", "close", term.HelpAction, "quit"}},
		{"RADIO", []term.Action{"radio-play", "radio-repeat", "radio-mode", "radio-viz", "voice", "radio-size", "radio-vol-up", "radio-vol-dn"}},
		{"WATCHLIST", []term.Action{"add-location", "remove", "lookup"}},
		{"DISPLAY", []term.Action{"units-f", "units-c", "theme"}},
		{"TICKER", []term.Action{"ticker-mute"}},
		{"APP", []term.Action{"setup", "status", "about"}},
	}
}

func orDefault(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

// About window (UAT 68/70 mock, 60 cols): title + version centred, the
// data providers and the build stack inset 3, the maker lines centred. Lines
// are composed on the mock's 58-cell interior and handed to the panel
// minus the two cells its chrome already draws, so every offset matches
// the mock exactly. Providers come from the live provider registry so a
// new data source lists itself.
const aboutWidth = 60

func (d Dashboard) aboutLines() []string {
	interior := aboutWidth - 2
	centre := func(text string) string {
		return strings.Repeat(" ", max(0, (interior-render.Width(text))/2)) + text
	}
	inset := func(text string) string { return "   " + text } // 3-cell inset (UAT 70): the NOAA line splits 3-52-3
	lines := []string{
		centre(render.TitleGradient("W A T C H P O S T")),
		centre("v " + d.cfg.Version),
		"",
		inset("Data Provided by:"),
		"",
	}
	for _, p := range d.cfg.Credits {
		lines = append(lines, inset(p))
	}
	lines = append(lines,
		"",
		inset(creditsNotice), // UAT 75
		"",
		inset("Built with:"),
		inset("GO "+strings.TrimPrefix(runtime.Version(), "go")+" | BubbleTea | LipGloss |"),
		inset("STUDS - Stylized Terminal UI Design System"),
		"",
		centre("Made with ♥ by Branden R. Thompson"),
		centre("github: branden-thompson"),
		centre("Make CLIs Great for Humans Again"),
	)
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, strings.TrimPrefix(l, "  ")) // the panel chrome draws these two cells
	}
	return out
}

// creditsNotice states the terms every listed source shares: NOAA data is
// public domain, GeoNames and Open-Meteo are CC BY 4.0 — all free to use
// with attribution (UAT 75).
const creditsNotice = "All sources free to use with attribution."
