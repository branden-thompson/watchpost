package tty

// help_about.go — the help and about modals. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"maps"
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

// The Help window's geometry: a blank line of air
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
func (d Dashboard) helpBlocks(o render.Opts) []helpBlock {
	var blocks []helpBlock
	seen := map[term.Action]bool{}
	row := func(bind term.Binding, act term.Action) string {
		return fmt.Sprintf("   %-12s - %s", strings.Join(bind.Keys, ", "), o.Marks(orDefault(bind.Help, string(act))))
	}
	keys := d.helpKeys()
	for _, g := range helpGroups(d.surface) {
		var rows []string
		if g.name == "MAP" {
			for _, pr := range mapHelpRows(keys) {
				rows = append(rows, fmt.Sprintf("   %-12s - %s", pr.keys, o.Marks(pr.help)))
			}
			for _, act := range g.actions {
				seen[act] = true
			}
		}
		for _, act := range g.actions {
			if g.name == "MAP" {
				break // listed above, a pair to a row
			}
			if bind, ok := keys[act]; ok {
				rows = append(rows, row(bind, act))
				seen[act] = true
			}
		}
		if len(rows) > 0 {
			blocks = append(blocks, helpBlock{append([]string{" " + render.Tint(g.name, render.Tok(render.ModalTitle))}, rows...)})
		}
	}
	var other []string
	for act, bind := range keys {
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
// THE PLAN SEES WHAT THE RENDER SEES. The mark swap changes widths — an
// ellipsis is one cell in the set and three in ASCII — so a plan measured
// under one set and drawn under the other clips (F-47).
func (d Dashboard) helpPlan(o render.Opts, avail int) (twoCol bool, width int) {
	blocks := d.helpBlocks(o)
	colW := helpColumnWidth(blocks)
	body := 1 + len(helpTwoColumns(blocks, colW)) + 4 // air · columns · legend (≤2) · blank · chips
	if w := twoColumnsWidth(colW, colW, panelChromeFor(body, d.modalMax())); w <= avail {
		return true, w
	}
	return false, helpOneColWidth
}

// helpWidth is the window's width for a terminal content width.
func (d Dashboard) helpWidth(o render.Opts, avail int) int {
	_, w := d.helpPlan(o, avail)
	return w
}

// helpLines renders the modal body: the air under the title, the groups in
// one or two columns, the row-marks legend, the chips.
func (d Dashboard) helpLines(o render.Opts) []string {
	blocks := d.helpBlocks(o)
	twoCol, _ := d.helpPlan(o, o.Width)
	lines := []string{""} // air under the title (item 1)
	if twoCol {
		lines = append(lines, helpTwoColumns(blocks, helpColumnWidth(blocks))...)
	} else {
		for _, b := range blocks {
			lines = append(lines, b.lines...)
			lines = append(lines, "")
		}
	}
	lines = append(lines, d.helpLegend(o))
	lines = append(lines, d.helpWithheld()...)
	return append(lines, "", "  "+o.Controls("   ", render.Ctl("esc", "Close"), render.Ctl("↑↓", "Scroll"))) // chips like every other modal (UAT 68.2)
}

// helpWithheld names the `[keys]` entries that did not reach THIS surface.
//
// D-15 REFUSES A SILENT WIN, and withholding is only not-silent if the operator
// is shown it. The console is the surface the entry did not reach, so it is the
// one that has to report it; Observer got the binding the file asked for and has
// nothing to say. Without this the reconciliation is a conflicting override
// losing quietly, which is the outcome D-15 exists to forbid.
func (d Dashboard) helpWithheld() []string {
	if d.surface != SurfaceBroadcaster || len(d.keysWithheld) == 0 {
		return nil
	}
	out := []string{"", " Your [keys] file, not applied here (this surface keeps its own):"}
	for _, w := range d.keysWithheld { // bounded by the override table (P10-02)
		out = append(out, "   "+w)
	}
	return out
}

// helpLegend is the line under the groups: what the operator will SEE that no
// keybinding explains.
//
// IT IS PER SURFACE FOR THE SAME REASON THE GROUPS ARE (D-135). The row marks
// are Observer's table glyphs and mean nothing on a console, which draws no
// such table — a legend for a surface you are not on is noise in the one window
// a lost operator opens.
//
// AND THE CONSOLE'S LEGEND CARRIES WHAT THE KEYMAP CANNOT. A slot's number
// opens its card, and those ten digits are deliberately NOT bindings — "ten
// ADDRESSES of one action, not ten actions", which would otherwise put ten rows
// in this window for one control. The chips on the cards say what the keys are;
// this says what they MEAN, which is the one thing the chips cannot.
func (d Dashboard) helpLegend(o render.Opts) string {
	g := o.Glyphs() // the table's own set, ASCII included (A11-10)
	if d.surface == SurfaceBroadcaster {
		return fmt.Sprintf(" Slot numbers: 0 opens the LIVE card   1-9 open a line-up slot   A opens the %s ALERT takeover   (an empty slot refuses quietly)", g.Alert)
	}
	// Row marks legend (red-team B5 U8): the glyphs beside a location, in words.
	return fmt.Sprintf(" Row marks: %s playing   %s on repeat   %s%s%s recent quake (below/felt/significant)   n%s fires nearby (bold = burning hard)   n%s alerts",
		g.Play, g.Repeat, g.Seismic[0], g.Seismic[1], g.Seismic[2], g.Fire, g.Alert)
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
// order a person meets them.
type helpGroup struct {
	name    string
	actions []term.Action
}

// helpGroups is the one owner of the grouping; a binding's group is its
// action, so a rebound key stays in its section (D-15: keys are data).
//
// AND THE SURFACE CHOOSES THE GROUPING (D-135). Building Observer's sections
// whatever the operator is looking at hands a console operator pressing `?` the
// listener's manual — HUM LEAD, UAT
// 2026-09-15: "right now the help window only shows Observer key bindings …
// Once the user in the Broadcaster UI, they key bindings share/rempapped for
// that mode do not update their help (like 'r')."
func helpGroups(surface Surface) []helpGroup {
	// SURFACES LEADS ON BOTH, because it is the one group whose absence leaves
	// the operator stuck. The swap is live on EITHER surface — the Router looks
	// it up before either one sees the key — and it was documented on NEITHER:
	// "it doesnt show the user how to swap between Observer and Broadcaster."
	surfaces := helpGroup{"SURFACES", []term.Action{actSwapObserver, actSwapBroadcaster}}
	if surface == SurfaceBroadcaster {
		return []helpGroup{
			surfaces,
			// THE CONSOLE'S OWN SECTIONS, in the order the operator meets them:
			// put the station on the air, order the line-up, choose the bed.
			{"STATION", []term.Action{actStationToggle, actGainUp, actGainDown}},
			{"LINE UP", []term.Action{actQueuePrev, actQueueNext, actQueueOpen, actRequest}},
			{"BED", []term.Action{actBedCut, actBedPrev, actBedNext}},
			{"APP", []term.Action{actLookup, actSettings, actStatus, actAbout, term.HelpAction, actDiagnostics, actQuit}},
		}
	}
	return []helpGroup{ // NAVIGATE and RADIO first: the two tall groups make the left column of the two-column layout (UAT mock 2026-08-28)
		surfaces,
		{"NAVIGATE", []term.Action{"nav-up", "nav-down", "details", "alert-details", "severe", actMap, "alert-prev", "alert-next", "close", term.HelpAction, "quit"}},
		{"RADIO", []term.Action{"radio-play", "radio-repeat", "radio-mode", "radio-viz", "voice", "radio-vol-up", "radio-vol-dn"}},
		{"WATCHLIST", []term.Action{"add-location", "remove", "lookup"}},
		{"DISPLAY", []term.Action{"units-f", "units-c", "theme"}},
		{"TICKER", []term.Action{"ticker-mute"}},
		{"APP", []term.Action{"setup", "status", "about", "debug"}},
		{"MAP", mapActions}, // 0.18.0 D-61: the open map window's own keys
	}
}

// helpKeys is the map the window documents: the ACTIVE surface's.
//
// THE CONSOLE'S LIVES ON THE ROUTER, and the Help window is Observer's — which
// is how the two came apart. `d.surface` is already mirrored on every update
// (D-92) for exactly this class of question, so the window can ask it rather
// than being told.
//
// AND THE SWAP IS ADDED TO OBSERVER'S, because it is real there and absent from
// its map: the Router intercepts it before either surface sees the key, so
// neither keymap carries it and neither help could show it.
func (d Dashboard) helpKeys() term.KeyMap {
	if d.surface == SurfaceBroadcaster {
		return d.consoleKeyMap()
	}
	out := term.KeyMap{}
	for act, bind := range d.keys {
		out[act] = bind
	}
	maps.Copy(out, d.mapKeys) // the map window's own scope, listed in its own group (D-61)
	bc := d.consoleKeyMap()
	for _, act := range []term.Action{actSwapObserver, actSwapBroadcaster} {
		if bind, ok := bc[act]; ok {
			out[act] = bind
		}
	}
	return out
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

func (d Dashboard) aboutLines(o render.Opts) []string {
	interior := aboutWidth - 2
	centre := func(text string) string {
		return strings.Repeat(" ", max(0, (interior-render.Width(text))/2)) + text
	}
	// The window's own margin, from the one owner (UAT 70; D-1 at the T3.10 red team).
	inset := func(text string) string { return strings.Repeat(" ", modalInset) + text }
	lines := []string{
		centre(render.Wordmark(render.EditionObserver)),
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
		centre("Made with "+o.Glyphs().Heart+" by Branden R. Thompson"),
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
