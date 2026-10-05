package tty

// map_nopicture_test.go — D-267: every key the map window binds changes what
// is on screen, in every description mode and under --ascii. Without a
// picture the Overlays menu is drawn as text in its place, so a listener
// moving through it sees what they change.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// mapKeyMsg is a key the map window binds, as the terminal sends it.
func mapKeyMsg(t *testing.T, key string) tea.KeyPressMsg {
	t.Helper()
	named := map[string]tea.KeyPressMsg{
		"up": {Code: tea.KeyUp}, "down": {Code: tea.KeyDown}, "left": {Code: tea.KeyLeft}, "right": {Code: tea.KeyRight},
		"pgup": {Code: tea.KeyPgUp}, "pgdown": {Code: tea.KeyPgDown}, "space": {Code: tea.KeySpace, Text: " "},
		"enter": {Code: tea.KeyEnter}, "esc": {Code: tea.KeyEscape},
		"shift+left": {Code: tea.KeyLeft, Mod: tea.ModShift}, "shift+right": {Code: tea.KeyRight, Mod: tea.ModShift},
	}
	if msg, ok := named[key]; ok {
		return msg
	}
	if r := []rune(key); len(r) == 1 {
		return tea.KeyPressMsg{Code: r[0], Text: key}
	}
	t.Fatalf("the fixture cannot express the key %q", key)
	return tea.KeyPressMsg{}
}

// screenModes are the window's ways of showing the map: the description's
// three, and --ascii.
var screenModes = []struct {
	name  string
	desc  string
	ascii bool
}{{"with", "with", false}, {"off", "off", false}, {"instead", "instead", false}, {"ascii", "with", true}}

// screenDash is the map open on Oceanside in a mode, every layer group
// registered, its work settled.
func screenDash(t *testing.T, desc string, ascii bool) Dashboard {
	t.Helper()
	var asked []string
	var asks []MapAsk
	cfg := Config{ASCII: ascii, MapDescription: desc, MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, "MRMS", &asked), MapTemperature: tempAnswer(&asks),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar"},
			{Key: TemperatureLayer, Label: "Temperature", On: true}, {Key: WindLayer, Label: "Wind"},
			{Key: RainLayer, Label: "Rain & snow"}, {Key: FeelsLayer, Label: "Feels like"}, {Key: WaveLayer, Label: "Waves"},
			{Key: UVLayer, Label: "UV"}, {Key: AirLayer, Label: "Air quality"}}}
	d := mapDash(t, cfg)
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(mapKeyMsg(t, "g"))
	return settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
}

// screenAfter presses a key and returns the screen before and after it.
func screenAfter(t *testing.T, d Dashboard, key string) (before, after string) {
	t.Helper()
	before = d.View().Content
	m, _ := d.Update(mapKeyMsg(t, key))
	d = settleMap(t, m.(Dashboard))
	return before, d.View().Content
}

// screenPreKeys put the window where a key does something before it is
// pressed: the step back, Now and the days' high or low act on a day, and the
// window opens on Now.
var screenPreKeys = map[string]string{"shift+left": "shift+right", "n": "shift+right", "<": "shift+right", ">": "shift+right"}

// The reasons a key's effect is not on screen.
const (
	onePlace    = "the fixture's list holds one place: there is no other to go to"
	bodyFits    = "the window's body fits it: there is nothing to scroll"
	viewUnseen  = "no picture: the view it moves is drawn nowhere, and the words name the place, which it leaves where it is"
	boxIsWords  = "no picture: the Area Alerts box is the description in a box, and the description is on screen whole already"
	playUnseen  = "no picture: the step it plays shows at the next step, in the badge, and the play state is the timeline's, which is drawn under a picture"
	regionNamed = "no picture: the region it shows is the one the description already names"
)

// invisibleKeys are the window's keys whose effect is not on screen in a
// mode, each with why that is right.
var invisibleKeys = map[string]map[string]string{
	"with": {"[": onePlace, "]": onePlace, "pgup": bodyFits, "pgdown": bodyFits},
	"off":  {"[": onePlace, "]": onePlace, "pgup": bodyFits, "pgdown": bodyFits},
	"instead": {"[": onePlace, "]": onePlace, "pgup": bodyFits, "pgdown": bodyFits,
		"up": viewUnseen, "down": viewUnseen, "left": viewUnseen, "right": viewUnseen, "+": viewUnseen, "=": viewUnseen, "-": viewUnseen,
		"A": boxIsWords, "space": playUnseen, "1": regionNamed},
	"ascii": {"[": onePlace, "]": onePlace, "pgup": bodyFits, "pgdown": bodyFits,
		"up": viewUnseen, "down": viewUnseen, "left": viewUnseen, "right": viewUnseen, "+": viewUnseen, "=": viewUnseen, "-": viewUnseen,
		"A": boxIsWords, "space": playUnseen, "1": regionNamed},
}

// TestTheMenusFocusStaysOnScreenWithoutAPicture is D-267: the menu drawn as
// text is longer than a short terminal's window, and the focused row is kept
// in view wherever the cursor goes.
func TestTheMenusFocusStaysOnScreenWithoutAPicture(t *testing.T) {
	m, _ := screenDash(t, "with", true).Update(tea.WindowSizeMsg{Width: 133, Height: 30})
	d := m.(Dashboard)
	for _, k := range []string{"O", "up"} { // up from the first row is the last: Parks
		m, _ := d.Update(mapKeyMsg(t, k))
		d = m.(Dashboard)
	}
	if text := stripANSITest(d.View().Content); !strings.Contains(text, "> [ ] Parks") {
		t.Errorf("the focused row, Parks, is not on screen:\n%s", text)
	}
	m, _ = d.Update(mapKeyMsg(t, "down")) // and round to the first
	if text := stripANSITest(m.(Dashboard).View().Content); !strings.Contains(text, "> * Temperature") {
		t.Errorf("the focused row, Temperature, is not on screen:\n%s", text)
	}
}

// TestEveryMapKeyChangesTheScreen is D-267: in every description mode and
// under --ascii, every key the window binds - and every key the open
// Overlays menu owns - changes what is on screen, but for the keys named
// with the reason their effect is not there to see. Each key is pressed on
// a window of its own: every copy of one shares its library map.
func TestEveryMapKeyChangesTheScreen(t *testing.T) {
	for _, mode := range screenModes {
		keys := screenDash(t, mode.desc, mode.ascii).mapKeys
		for _, act := range mapActions {
			for _, key := range keys[act].Keys {
				if _, ok := invisibleKeys[mode.name][key]; ok {
					continue
				}
				d := screenDash(t, mode.desc, mode.ascii)
				if pre, ok := screenPreKeys[key]; ok {
					m, _ := d.Update(mapKeyMsg(t, pre))
					d = settleMap(t, m.(Dashboard))
				}
				if before, after := screenAfter(t, d, key); before == after {
					t.Errorf("%s: %s (%s) changed nothing on screen", mode.name, key, act)
				}
			}
		}
		for _, c := range []struct{ key, at string }{ // the open menu's keys, each on a row where it does something
			{"down", ""}, {"up", "down"}, {"right", ""}, {"left", ""}, {"space", ""}, {"enter", ""}, {"esc", ""}} {
			d := screenDash(t, mode.desc, mode.ascii)
			for _, k := range []string{"O", c.at} {
				if k != "" {
					m, _ := d.Update(mapKeyMsg(t, k))
					d = settleMap(t, m.(Dashboard))
				}
			}
			if text := stripANSITest(d.View().Content); !strings.Contains(text, "MAP DETAILS / OVERLAYS") || !strings.Contains(text, d.opts().Glyphs().Pointer+" ") {
				t.Fatalf("%s: O drew no menu with its focused row marked:\n%s", mode.name, text)
			}
			if before, after := screenAfter(t, d, c.key); before == after {
				t.Errorf("%s: %s in the open menu changed nothing on screen", mode.name, c.key)
			}
		}
	}
}
