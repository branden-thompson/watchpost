package tty

// map_settings_test.go — 0.18.0 W1.5, W1.6, W1.8, W1.10: the map's Settings
// rows, what they change, and what the window shows at every size.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"
	"github.com/mattn/go-runewidth"
)

// TestTheMapRowsSaveWithTheDisplayPreferences is W1.10 (FR-9.1): maps on or
// off and the description's mode are rows of the WATCHPOST UI group, live
// when chosen and written with the group on close.
func TestTheMapRowsSaveWithTheDisplayPreferences(t *testing.T) {
	d, got := uiDash(t, rowMapDesc)
	m0, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m0.(Dashboard)
	if d.mapDesc != mapDescInstead {
		t.Errorf("→ on the description's picker did not move it on: %v", d.mapDesc)
	}
	d.setup.focus = rowMapsOn
	d = d.setupSpace()
	if !d.mapsOff {
		t.Error("space on the maps row did not switch maps off")
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapDescription != "instead" || got.Maps != "off" {
		t.Errorf("esc wrote %+v; want the map description instead and maps off", *got)
	}
	body, _, _ := d.focusBody(d.opts())
	lines := strings.Join(body, "\n")
	for _, want := range []string{"Maps -", "Disabled", "Map description -", "Instead of the picture"} {
		if !strings.Contains(lines, want) {
			t.Errorf("the WATCHPOST UI group does not show %q", want)
		}
	}
}

// TestTheConfigWordsOpenTheWindowAsChosen is W1.10's round trip: what the file
// says is what the window opens with.
func TestTheConfigWordsOpenTheWindowAsChosen(t *testing.T) {
	d := mapDash(t, Config{Maps: "off", MapDescription: "off"})
	if !d.mapsOff || d.mapDesc != mapDescOff {
		t.Errorf("maps off and description off in the file opened as %v, %v", d.mapsOff, d.mapDesc)
	}
	if d := mapDash(t, Config{}); d.mapsOff || d.mapDesc != mapDescWith {
		t.Errorf("an empty file opened as maps off %v, description %v; want on, with the picture", d.mapsOff, d.mapDesc)
	}
}

// TestMapsOffSaysSoAndBuildsNothing is W1.8 (FR-1.6, NFR-3): with maps off, g
// says so in one line and no map is built.
func TestMapsOffSaysSoAndBuildsNothing(t *testing.T) {
	built := 0
	d := mapDash(t, Config{Maps: "off", NewMap: func(size tuimaps.Size) (*tuimaps.Map, error) { built++; return embeddedMap(size) }})
	d, _ = pressKey(d, "g")
	out := stripANSITest(d.View().Content)
	if !strings.Contains(out, mapsOffText) {
		t.Errorf("with maps off the window does not say so:\n%s", out)
	}
	if built != 0 {
		t.Errorf("a map was built %d times with maps off", built)
	}
}

// TestTheDescriptionComesFirstWithThePicture is W1.6 (D-55): with the
// description's mode "with the picture", the description comes first in
// reading order, above the braille; "instead" shows no braille; "off" shows
// no description.
func TestTheDescriptionComesFirstWithThePicture(t *testing.T) {
	for _, c := range []struct {
		mode           string
		words, braille bool
	}{{"", true, true}, {"instead", true, false}, {"off", false, true}} {
		t.Run("mode "+c.mode, func(t *testing.T) {
			d := mapDash(t, Config{MapDescription: c.mode, MapFeed: boxFeed(-117.6, -117.1, false)})
			d, _ = pressKey(d, "g")
			d = feedAndSettle(t, d)
			body := d.mapBodyLines()
			text := stripANSITest(strings.Join(body, "\n"))
			words := strings.Index(text, "Oceanside, CA:")
			braille := strings.IndexFunc(text, func(r rune) bool { return r > 0x2800 && r <= 0x28ff })
			if (words >= 0) != c.words || (braille >= 0) != c.braille {
				t.Fatalf("words %v, braille %v; want %v, %v:\n%s", words >= 0, braille >= 0, c.words, c.braille, text)
			}
			if c.words && c.braille && words > braille {
				t.Error("the description does not come first in reading order")
			}
		})
	}
}

// TestBelowTheFloorTheNoticeCarriesTheDescription is W1.5 (FR-1.4): a map
// body under 69x12 is never drawn; the window names the size it needs and
// has, and carries the description.
func TestBelowTheFloorTheNoticeCarriesTheDescription(t *testing.T) {
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false)})
	m, _ := d.Update(tea.WindowSizeMsg{Width: 70, Height: 22})
	d = m.(Dashboard)
	calls := &[]string{}
	d.mapPane.calls = calls
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	text := stripANSITest(strings.Join(d.mapBodyLines(), "\n"))
	if !strings.Contains(text, "69 × 12") || !strings.Contains(text, "Wind Warning") {
		t.Errorf("below the floor the window says:\n%s", text)
	}
	if strings.IndexFunc(text, func(r rune) bool { return r > 0x2800 && r <= 0x28ff }) >= 0 {
		t.Error("a map smaller than the floor was drawn")
	}
	for _, c := range *calls {
		if c == "Render" {
			t.Fatal("a map under the floor was rendered, to be thrown away")
		}
	}
}

// TestTheMapDrawsAtTheDocumentedFloor is W1.5 against the app's own floor:
// at 80x24 a 69x12 map fits, the description off and on (scrolling then).
func TestTheMapDrawsAtTheDocumentedFloor(t *testing.T) {
	for _, mode := range []string{"off", "with"} {
		d := mapDash(t, Config{MapDescription: mode, MapFeed: boxFeed(-117.6, -117.1, false)})
		m, _ := d.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		d = m.(Dashboard)
		d, _ = pressKey(d, "g")
		d = feedAndSettle(t, d)
		if !d.mapFits() {
			t.Errorf("description %s: at 80x24 the map is %+v, under the floor", mode, d.mapBodySize())
		}
		text := stripANSITest(strings.Join(d.mapBodyLines(), "\n"))
		if strings.IndexFunc(text, func(r rune) bool { return r > 0x2800 && r <= 0x28ff }) < 0 {
			t.Errorf("description %s: no map at 80x24:\n%s", mode, text)
		}
	}
}

// TestEverySizeShowsAStatedStateAndNeverOverflows is W1.5 and W1.6's sweep
// (FR-1.4, FR-1.9): from 20x10 to 200x80 every frame is a map at least 69x12
// or words, and no line is wider than the terminal.
func TestEverySizeShowsAStatedStateAndNeverOverflows(t *testing.T) {
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false)})
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	for w := 20; w <= 200; w += 9 {
		for h := 10; h <= 80; h += 7 {
			m, _ := d.Update(tea.WindowSizeMsg{Width: w, Height: h})
			sized := m.(Dashboard)
			body := stripANSITest(strings.Join(sized.mapBodyLines(), "\n"))
			if strings.TrimSpace(body) == "" {
				t.Fatalf("%dx%d: the window body is empty", w, h)
			}
			hasMap := strings.IndexFunc(body, func(r rune) bool { return r > 0x2800 && r <= 0x28ff }) >= 0
			if hasMap && (sized.mapBodySize().Cols < mapMinBody.Cols || sized.mapBodySize().Rows < mapMinBody.Rows) {
				t.Errorf("%dx%d: a %+v map was drawn under the floor", w, h, sized.mapBodySize())
			}
			if !hasMap && !strings.ContainsAny(body, "abcdefghijklmnopqrstuvwxyz") {
				t.Errorf("%dx%d: no map and no words", w, h)
			}
			for _, l := range strings.Split(stripANSITest(sized.renderModal(sized.opts())), "\n") {
				if n := runewidth.StringWidth(l); n > w {
					t.Fatalf("%dx%d: a line is %d wide", w, h, n)
				}
			}
		}
	}
}

// TestTheDescriptionPickerGoesBothWays: → and ← walk the three modes in
// opposite directions, and wrap.
func TestTheDescriptionPickerGoesBothWays(t *testing.T) {
	d, _ := uiDash(t, rowMapDesc)
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if got := m.(Dashboard).mapDesc; got != mapDescOff {
		t.Errorf("← from with the picture went to %v, want off", got)
	}
}

// TestTheArrowsSwitchTabsUnlessTheRowTakesThem is D-62: on a row that does not
// operate with the arrows, → and ← switch tabs (as in [w]); a focused picker
// keeps them; tab and shift+tab switch tabs from any row.
func TestTheArrowsSwitchTabsUnlessTheRowTakesThem(t *testing.T) {
	d, _ := uiDash(t, rowUnitsImperial) // a radio row, on General
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	if got := m.(Dashboard).setupTab(); got != tabRadio {
		t.Errorf("→ on a radio row went to %s, want Watchpost Radio", got.Label())
	}
	m, _, _ = d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if got := m.(Dashboard).setupTab(); got != tabMaps {
		t.Errorf("← on General went to %s, want Maps (wrapping)", got.Label())
	}
	p, _ := uiDash(t, rowTheme) // a picker keeps the arrows
	m, _, _ = p.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	if got := m.(Dashboard).setupTab(); got != tabGeneral {
		t.Errorf("→ on the theme picker left for %s", got.Label())
	}
	m, _ = p.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.(Dashboard).setupTab(); got != tabRadio {
		t.Errorf("tab on the theme picker went to %s, want Watchpost Radio", got.Label())
	}
}

// TestEachSurfaceHasItsTabs is D-62 exactly: Observer's Settings are General,
// Watchpost Radio and Maps; the console's General, Watchpost Radio and
// Broadcaster (D-18/D-92: a mode's own settings appear only in its mode).
func TestEachSurfaceHasItsTabs(t *testing.T) {
	for surface, want := range map[Surface][]setupTab{
		SurfaceObserver:    {tabGeneral, tabRadio, tabMaps},
		SurfaceBroadcaster: {tabGeneral, tabRadio, tabBroadcaster},
	} {
		d, _ := uiDash(t, rowTheme)
		d.surface = surface
		got := d.tabsShown()
		if len(got) != len(want) {
			t.Errorf("surface %v shows %v, want %v", surface, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("surface %v shows %v, want %v", surface, got, want)
				break
			}
		}
	}
}
