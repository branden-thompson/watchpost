package tty

import (
	"fmt"
	"github.com/branden-thompson/watchpost/platform/term"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestHelpFloatsOverDashboard(t *testing.T) {
	// UAT 8.3: '?' composites the help panel over the dashboard instead of
	// replacing the view — the dashboard chrome stays visible around it.
	m := dash(t)
	m2, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	v := m2.View().Content
	if !strings.Contains(v, "Watchpost Help") {
		t.Fatalf("help panel missing:\n%s", v)
	}
	if !strings.Contains(v, "W A T C H P O S T") || !strings.Contains(v, "Quit") {
		t.Fatalf("dashboard must stay visible beneath the floating help:\n%s", v)
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
	order := []string{"NAVIGATE", "WATCHLIST", "RADIO", "DISPLAY", "APP"}
	last := -1
	for _, name := range order {
		i := strings.Index(text, name)
		if i < 0 || i < last {
			t.Fatalf("section %s missing or out of order:\n%s", name, text)
		}
		last = i
	}
	nav := text[strings.Index(text, "NAVIGATE"):strings.Index(text, "WATCHLIST")]
	if !strings.Contains(nav, "Q ") || !strings.Contains(nav, "Quit") {
		t.Fatalf("a rebound quit stays under NAVIGATE:\n%s", nav)
	}
	for _, bind := range m.keys { // every binding listed exactly once, by its rendered row prefix (up and down share a Help text; "-" is a key)
		if row := fmt.Sprintf("  %-12s - ", strings.Join(bind.Keys, ", ")); strings.Count(text, row) != 1 {
			t.Fatalf("%q listed %d times", row, strings.Count(text, row))
		}
	}
	if strings.Contains(text, "OTHER") {
		t.Fatal("every default binding has a group")
	}
	if !strings.HasPrefix(lines[len(lines)-3], "Row marks:") {
		t.Fatalf("the legend follows the groups: %q", lines[len(lines)-3])
	}
}
