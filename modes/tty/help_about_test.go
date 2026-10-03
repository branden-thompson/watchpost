package tty

import (
	"fmt"
	"github.com/branden-thompson/watchpost/platform/term"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestHelpFloatsOverDashboard(t *testing.T) {
	// UAT 8.3: '?' composites the help panel over the dashboard instead of
	// replacing the view — the dashboard chrome stays visible around it.
	m := dash(t)
	m2, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	v := stripANSITest(m2.View().Content)
	if !strings.Contains(v, "Watchpost Help") {
		t.Fatalf("help panel missing:\n%s", v)
	}
	// At 133 cols the two-column window spans most of the
	// width: the dashboard shows beside it — a row with content left of the
	// panel's border — never replaced by it.
	beside := false
	for _, l := range strings.Split(v, "\n") {
		if i := strings.Index(l, "│"); i > 0 && strings.TrimSpace(l[:i]) != "" {
			beside = true
			break
		}
	}
	if !beside {
		t.Fatalf("dashboard must stay visible beside the floating help:\n%s", v)
	}
	// On a wider terminal the header clears the window too.
	wide, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	wide, _ = wide.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if wv := stripANSITest(wide.View().Content); !strings.Contains(wv, "WATCHPOST Observer") || !strings.Contains(wv, "Quit") {
		t.Fatalf("dashboard header must stay visible beneath the floating help at 200 cols:\n%s", wv)
	}
}

// aboutGroups is a set of credit groups as the app hands them in (W21).
func aboutGroups() []CreditGroup {
	return []CreditGroup{
		{Name: "NATIONAL OCEANIC AND ATMOSPHERIC ADMINISTRATION (NOAA)", Lines: []CreditLine{
			{Abbr: "NWS", What: "National Weather Service", Host: "api.weather.gov"},
			{Abbr: "NDBC", What: "National Data Buoy Center", Host: "ndbc.noaa.gov"}}},
		{Name: "UNITED STATES ENVIRONMENTAL PROTECTION AGENCY", Lines: []CreditLine{
			{Abbr: "AQI", What: "U.S. EPA AirNow", Note: "preliminary data, not fully verified"}}},
		{Name: "OPEN-METEO (CC BY 4.0)", Lines: []CreditLine{{What: "geocoding"}, {What: "Wind data, interpolated"}}},
	}
}

// aboutFrame is the About window's framed lines, as drawn at a terminal size.
func aboutFrame(t *testing.T, cfg Config, w, h int) (Dashboard, string) {
	t.Helper()
	m, err := NewDashboard(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: w, Height: h})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	model, _ = model.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	d := model.(Dashboard)
	if d.modal != modalAbout {
		t.Fatal("[a] must open the About window")
	}
	var frame []string
	for _, l := range strings.Split(stripANSITest(d.View().Content), "\n") {
		if i := strings.Index(l, "│"); i >= 0 && strings.Count(l, "│") >= 2 {
			frame = append(frame, l[i:])
		}
	}
	return d, strings.Join(frame, "\n")
}

// THE ABOUT WINDOW IS THE MOCK'S (W21, about-credits-mock.md): the title and
// the build on one line; the warnings first; the terms; the data sets, a group
// a provider, each source once - its short name, what it is, its host at the
// right margin, a note under it; then what it is built with and who made it.
func TestAboutWindowMatchesMock(t *testing.T) {
	d, body := aboutFrame(t, Config{Version: "0.1.0-test", CreditGroups: aboutGroups(),
		AboutWarnings: []string{"NOT INTENDED AS A SUBSTITUTE FOR OFFICIAL WARNING SOURCES OR DEVICES", "WEATHER RELAYS MAY BE DELAYED"}}, 133, 80)
	for _, want := range []string{
		"│" + strings.Repeat(" ", 25) + "WATCHPOST    v. 0.1.0-test" + strings.Repeat(" ", 25) + "│", // 26 cells centred in 76
		"│   ! NOT INTENDED AS A SUBSTITUTE FOR OFFICIAL WARNING SOURCES OR DEVICES   │",
		"│   ! WEATHER RELAYS MAY BE DELAYED                                          │",
		"│   All sources free to use with attribution.                                │",
		"│   DATA SETS PROVIDED BY:                                                   │",
		"│   NATIONAL OCEANIC AND ATMOSPHERIC ADMINISTRATION (NOAA)                   │",
		"│     NWS     - National Weather Service                   api.weather.gov   │",
		"│     NDBC    - National Data Buoy Center                    ndbc.noaa.gov   │",
		"│     AQI     - U.S. EPA AirNow                                              │",
		"│               preliminary data, not fully verified                         │",
		"│   OPEN-METEO (CC BY 4.0)                                                   │",
		"│     geocoding                                                              │",
		"│   Built with:                                                              │",
		"│   GO " + strings.TrimPrefix(runtime.Version(), "go") + " | BubbleTea | LipGloss | go-tuimaps",
		"│   Stylized Terminal UI Design System (STUDS)                               │",
		"│   github: branden-thompson                                                 │",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("About window missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, "Built with ♥ by Branden R. Thompson") {
		t.Errorf("the maker line is missing:\n%s", body)
	}
	order := []string{"! NOT INTENDED", "All sources free", "DATA SETS PROVIDED BY:", "NATIONAL OCEANIC", "UNITED STATES ENVIRONMENTAL", "OPEN-METEO", "Built with:", "github:"}
	for k := 1; k < len(order); k++ {
		if strings.Index(body, order[k-1]) > strings.Index(body, order[k]) {
			t.Errorf("%q comes after %q; want the mock's order", order[k-1], order[k])
		}
	}
	if strings.Count(body, "National Data Buoy Center") != 1 {
		t.Errorf("a source is credited %d times; want once", strings.Count(body, "National Data Buoy Center"))
	}
	m3, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m3.(Dashboard).modal == modalAbout {
		t.Fatal("esc must close the About window")
	}
}

// THE ABOUT WINDOW TAKES TWO COLUMNS WHERE THEY FIT (W21.2): the data sets
// side by side on a wide terminal, the groups whole in either column, and one
// column, scrolled, where the terminal is short.
func TestAboutTakesTwoColumnsWhereTheyFit(t *testing.T) {
	cfg := Config{Version: "t", CreditGroups: aboutGroups()}
	_, wide := aboutFrame(t, cfg, 220, 60)
	row := ""
	for _, l := range strings.Split(wide, "\n") {
		if strings.Contains(l, "NATIONAL OCEANIC") {
			row = l
		}
	}
	if !strings.Contains(row, "UNITED STATES ENVIRONMENTAL") && !strings.Contains(row, "OPEN-METEO") {
		t.Errorf("on a wide terminal the groups are not side by side:\n%s", wide)
	}
	_, narrow := aboutFrame(t, cfg, 133, 80)
	for _, l := range strings.Split(narrow, "\n") {
		if strings.Contains(l, "NATIONAL OCEANIC") && strings.Contains(l, "OPEN-METEO") {
			t.Errorf("at 133 columns the groups are side by side: %q", l)
		}
	}
	d, short := aboutFrame(t, cfg, 133, 24)
	if strings.Contains(short, "github: branden-thompson") {
		t.Fatalf("a short terminal shows the whole window; the fixture must make it scroll:\n%s", short)
	}
	whole := false // beside the scroll rail a row is still whole: the window widens by the rail
	for _, l := range strings.Split(short, "\n") {
		if strings.Contains(l, "National Weather Service") && strings.Contains(l, "api.weather.gov") {
			whole = true
		}
	}
	if !whole {
		t.Errorf("scrolling, the NWS row is not whole on its line:\n%s", short)
	}
	for _, l := range strings.Split(short, "\n") {
		if n := strings.Count(l, "─"); n > 0 && n != aboutColumn {
			t.Errorf("scrolling, the rule is %d cells; want the column's %d - the rail is the window's to make room for", n, aboutColumn)
		}
	}
	for range 60 { // to the end
		m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		d = m.(Dashboard)
	}
	var frame []string
	for _, l := range strings.Split(stripANSITest(d.View().Content), "\n") {
		if strings.Count(l, "│") >= 2 {
			frame = append(frame, l)
		}
	}
	if !strings.Contains(strings.Join(frame, "\n"), "github: branden-thompson") {
		t.Errorf("scrolled to the end, the last line is not reachable:\n%s", strings.Join(frame, "\n"))
	}
}

func TestHelpModalControlsAreChips(t *testing.T) {
	// UAT 68.2: the help window's controls use the same KeyCap chips as
	// every other modal (not literal brackets).
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	m2, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	d := m2.(Dashboard)
	if d.modal != modalHelp {
		t.Fatal("? opens help")
	}
	o := d.opts()
	lines := d.helpLines(o)
	last := lines[len(lines)-1]
	if want := "  " + o.KeyCap("esc") + " Close   " + o.KeyCap("↑↓") + " Scroll"; last != want || strings.Contains(last, "[esc]") {
		t.Fatalf("help controls must be chips:\n got %q\nwant %q", last, want)
	}
	if !strings.Contains(last, "48;2;86;86;86") { // the chip background — colour is on
		t.Fatalf("chips must carry the KeyChip tone: %q", last)
	}
}

// HUM LEAD UAT 2026-08-27: the Help window groups the bindings by feature,
// every binding appears once, and a rebound key stays in its section.
func TestHelpGroupsBindingsByFeature(t *testing.T) {
	m, err := NewDashboard(Config{Version: "t", KeyOverrides: term.KeyMap{"quit": {Keys: []string{"Q"}, Help: "Quit"}}})
	if err != nil {
		t.Fatal(err)
	}
	lines := m.helpLines(m.opts())
	text := strings.Join(lines, "\n")
	order := []string{"NAVIGATE", "RADIO", "WATCHLIST", "DISPLAY", "APP"} // the registry's order (NAVIGATE and RADIO first: the left column when two fit)
	last := -1
	for _, name := range order {
		i := strings.Index(text, name)
		if i < 0 || i < last {
			t.Fatalf("section %s missing or out of order:\n%s", name, text)
		}
		last = i
	}
	nav := text[strings.Index(text, "NAVIGATE"):strings.Index(text, "RADIO")]
	if !strings.Contains(nav, "Q ") || !strings.Contains(nav, "Quit") {
		t.Fatalf("a rebound quit stays under NAVIGATE:\n%s", nav)
	}
	for _, bind := range m.keys { // every binding listed exactly once, by its rendered row prefix (up and down share a Help text; "-" is a key)
		if row := helpRow(bind); strings.Count(text, row) != 1 {
			t.Fatalf("%q listed %d times", row, strings.Count(text, row))
		}
	}
	if strings.Contains(text, "OTHER") {
		t.Fatal("every default binding has a group")
	}
	if !strings.HasPrefix(lines[len(lines)-3], " Row marks:") { // the mock's one-space inset
		t.Fatalf("the legend follows the groups: %q", lines[len(lines)-3])
	}
}

// The Help window is one column with the panel's scroll on a narrow
// terminal and two columns on a wide one: a blank
// line of air under the title in both; every group whole, in one column;
// every binding once in both layouts.
func TestHelpLaysOutOneOrTwoColumns(t *testing.T) {
	for _, c := range []struct {
		w      int
		twoCol bool
	}{{80, false}, {100, false}, {133, true}, {200, true}} {
		m, err := NewDashboard(Config{Version: "t"})
		if err != nil {
			t.Fatal(err)
		}
		var mm tea.Model = m
		mm, _ = mm.Update(tea.WindowSizeMsg{Width: c.w, Height: 44})
		mm, _ = mm.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
		d := mm.(Dashboard)
		lines := d.helpLines(d.opts())
		if lines[0] != "" {
			t.Fatalf("%d cols: a blank line under the title, got %q", c.w, lines[0])
		}
		text := stripANSITest(strings.Join(lines, "\n"))
		// TWO COLUMNS IS "SOME LINE CARRIES TWO GROUP HEADERS", not "NAVIGATE
		// sits beside WATCHLIST". A named pairing is a proxy for the layout:
		// adding a group (D-135's SURFACES) moves the balance point, and the
		// proxy then reports one column on a window plainly drawing two. It
		// measures the split, not the thing the test is named for.
		var names []string
		for _, g := range helpGroups(d.surface) {
			names = append(names, g.name)
		}
		names = append(names, "OTHER")
		pairs := 0
		for _, l := range strings.Split(text, "\n") {
			n := 0
			for _, name := range names {
				if strings.Contains(l, name) {
					n++
				}
			}
			if n >= 2 {
				pairs++
			}
		}
		if (pairs > 0) != c.twoCol {
			t.Fatalf("%d cols: two columns = %v, want %v:\n%s", c.w, pairs > 0, c.twoCol, text)
		}
		for _, bind := range d.keys { // every binding once, whatever the layout
			if row := helpRow(bind); strings.Count(text, row) != 1 {
				t.Fatalf("%d cols: %q listed %d times", c.w, row, strings.Count(text, row))
			}
		}
		// A group rolls as a unit: the RADIO header's column holds its first row on the next line.
		ls := strings.Split(text, "\n")
		for i, l := range ls {
			if j := strings.Index(l, "RADIO"); j >= 0 {
				if next := ls[i+1]; len(next) <= j || !strings.Contains(next[j:], "space") {
					t.Fatalf("%d cols: RADIO's first row must follow its header in the same column:\n%s", c.w, text)
				}
			}
		}
		frame := stripANSITest(d.View().Content)
		for _, l := range strings.Split(frame, "\n") {
			if render.Width(strings.TrimRight(l, " ")) > c.w {
				t.Fatalf("%d cols: a line overflows the terminal: %q", c.w, l)
			}
		}
	}
}

// helpRow is a binding's row in the Help window, by its rendered key prefix.
func helpRow(bind term.Binding) string {
	return fmt.Sprintf("   %-12s - ", strings.Join(bind.Keys, ", "))
}

// THE ABOUT WINDOW'S MARKS (D-232): the warnings in the focus yellow, a blank
// row under them; each group's title bold white; each host light blue - the
// app's own focus colours, so every theme carries them.
func TestAboutWindowsMarks(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	d, err := NewDashboard(Config{Version: "t", CreditGroups: aboutGroups(), AboutWarnings: []string{"FIRST WARNING", "LAST WARNING"}})
	if err != nil {
		t.Fatal(err)
	}
	lines := d.aboutLines(d.opts())
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		render.Tint("! FIRST WARNING", render.Tok(render.ListFocus)),
		render.Tint("! LAST WARNING", render.Tok(render.ListFocus)),
		render.Tint("NATIONAL OCEANIC AND ATMOSPHERIC ADMINISTRATION (NOAA)", render.Tok(render.FocusPointer)),
		render.Tint("api.weather.gov", render.Tok(render.AboutHost)),
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("the window lacks %q", want)
		}
	}
	last := -1
	for i, l := range lines {
		if strings.Contains(l, "LAST WARNING") {
			last = i
		}
	}
	if last < 0 || last+2 >= len(lines) || stripANSITest(lines[last+1]) != "" || !strings.Contains(lines[last+2], creditsNotice) {
		t.Errorf("under the last warning come %q and %q; want a blank row, then the terms", lines[last+1], lines[last+2])
	}
}
