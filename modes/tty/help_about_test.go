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
	// At 133 cols the two-column window (UAT 2026-08-28) spans most of the
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
	if wv := stripANSITest(wide.View().Content); !strings.Contains(wv, "W A T C H P O S T") || !strings.Contains(wv, "Quit") {
		t.Fatalf("dashboard header must stay visible beneath the floating help at 200 cols:\n%s", wv)
	}
}

func TestAboutWindowMatchesMock(t *testing.T) {
	// UAT 68/70/75: [a] floats the About window (60 cols): centred title +
	// version, the app-owned credits list + licence notice and the build
	// stack inset 3 from the frame, maker lines centred; esc closes.
	m, err := NewDashboard(Config{Version: "0.1.0-test", Credits: []string{"NOAA National Weather Service (api.weather.gov)", "GeoNames.org cities & postal codes (CC BY 4.0)"}})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	m2, _ := model.Update(SnapshotMsg{Snap: snap()})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	d := m2.(Dashboard)
	if d.modal != modalAbout {
		t.Fatal("[a] must open the About window")
	}
	v := stripANSITest(d.View().Content)
	var frame []string
	for _, l := range strings.Split(v, "\n") {
		if i := strings.Index(l, "│"); i >= 0 && strings.Contains(l, "│") && strings.Count(l, "│") >= 2 {
			frame = append(frame, l[i:])
		}
	}
	body := strings.Join(frame, "\n")
	for _, want := range []string{
		"│                    W A T C H P O S T                     │",
		"│                       v 0.1.0-test                       │", // 12 chars centred on the 58-cell interior (the mock's {v 0.0.0-dev} is 13)
		"│   Data Provided by:                                      │",
		"│   NOAA National Weather Service (api.weather.gov)        │",
		"│   GeoNames.org cities & postal codes (CC BY 4.0)         │", // the app's list renders as given (UAT 75)
		"│   All sources free to use with attribution.              │",
		"│   Built with:                                            │",
		"│   STUDS - Stylized Terminal UI Design System             │",
		"│            Made with ♥ by Branden R. Thompson            │",
		"│                 github: branden-thompson                 │",
		"│             Make CLIs Great for Humans Again             │",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("About window missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, "│   GO "+strings.TrimPrefix(runtime.Version(), "go")+" | BubbleTea | LipGloss |") {
		t.Fatalf("build line must carry the running Go version:\n%s", body)
	}
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(l, "┌") && strings.Contains(l, "┐") && strings.Contains(l, "─ ") {
			t.Fatalf("About window has no title: %q", l)
		}
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m3.(Dashboard).modal == modalAbout {
		t.Fatal("esc must close the About window")
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
		if row := fmt.Sprintf("   %-12s - ", strings.Join(bind.Keys, ", ")); strings.Count(text, row) != 1 {
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
// terminal and two columns on a wide one (HUM LEAD UAT 2026-08-28): a blank
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
		pairs := 0
		for _, l := range strings.Split(text, "\n") {
			if strings.Contains(l, "NAVIGATE") && strings.Contains(l, "WATCHLIST") {
				pairs++
			}
		}
		if (pairs == 1) != c.twoCol {
			t.Fatalf("%d cols: two columns = %v, want %v:\n%s", c.w, pairs == 1, c.twoCol, text)
		}
		for _, bind := range d.keys { // every binding once, whatever the layout
			if row := fmt.Sprintf("   %-12s - ", strings.Join(bind.Keys, ", ")); strings.Count(text, row) != 1 {
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
