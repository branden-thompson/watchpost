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
)

// helpModal renders live from the merged KeyMap so it is truthful after any
// swap (D-15 guarantee 3).
func (d Dashboard) helpModal(o render.Opts) string {
	return d.floatModal(o, d.modalWidth(), "Watchpost Help", d.helpLines(o)) // UAT 8.3/10.1/10.4
}

// helpLines renders the merged KeyMap as modal body lines (truthful after
// any swap - D-15 guarantee 3).
func (d Dashboard) helpLines(o render.Opts) []string {
	lines := make([]string, 0, len(d.keys)+4)
	for act, bind := range d.keys {
		lines = append(lines, fmt.Sprintf("%-14s - %s", strings.Join(bind.Keys, ", "), orDefault(bind.Help, string(act))))
	}
	sort.Strings(lines)
	// Row marks legend (red-team B5 U8): the glyphs beside a location, in words.
	g := o.Glyphs() // the table's own set, ASCII included (A11-10)
	lines = append(lines, "", fmt.Sprintf("Row marks: %s playing   %s on repeat   n%s fires nearby (bold = burning hard)   n%s alerts", g.Play, g.Repeat, g.Fire, g.Alert))
	return append(lines, "", "  "+o.KeyCap("esc")+" Close   "+o.KeyCap("↑↓")+" Scroll") // chips like every other modal (UAT 68.2)
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
