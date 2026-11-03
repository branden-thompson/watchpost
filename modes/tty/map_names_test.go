package tty

// map_names_test.go — F-205 AX-6 and AX-9 (FR-7.4; D-365, A-107 (8)): every
// map control says its state, the description has one name, and the line
// that cuts it short names the remedy.

import (
	"strings"
	"testing"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TestEveryMapControlSaysItsState is AX-6: the Description chip says
// whether its box is open, every chip's state is whole on the smallest
// window, and the region row says which region the map is bound to.
func TestEveryMapControlSaysItsState(t *testing.T) {
	chips := func(d Dashboard) string { return stripANSITest(d.mapStatusLine()) }
	for _, size := range [][2]int{{133, 44}, {80, 24}} {
		d := openMap(t, Config{}, size[0], size[1])
		for _, want := range []string{"A] Description Off", "R] Radar On", "P] Propagation Off", "O] Overlays"} {
			if !strings.Contains(chips(d), want) {
				t.Errorf("%dx%d: on opening the chips %q do not say %q", size[0], size[1], chips(d), want)
			}
		}
		if got := chips(pressCode(d, 'A', "A")); !strings.Contains(got, "A] Description On") {
			t.Errorf("%dx%d: with the box open the chips say %q", size[0], size[1], got)
		}
	}
	d := openMap(t, Config{}, 133, 44)
	row := stripANSITest(d.regionsRow(200))
	if strings.Count(row, "(shown)") != 1 || !strings.Contains(row, "CONT. US (shown)") {
		t.Errorf("the region row does not say the region shown once, after CONT. US: %q", row)
	}
	if row := stripANSITest(d.regionsRow(60)); !strings.Contains(row, "(shown)") {
		t.Errorf("the short region row drops the region shown: %q", row)
	}
}

// TestTheDescriptionHasOneName is AX-9 (D-365): the box, its chip, its key's
// help, Help and Settings all call the text "Description"; the Settings
// choice says the box waits for A.
func TestTheDescriptionHasOneName(t *testing.T) {
	d := pressCode(openMap(t, Config{}, 133, 44), 'A', "A")
	if lines := strings.Split(bodyText(d), "\n"); !strings.Contains(lines[0], "Description") {
		t.Errorf("the box's title is %q", lines[0])
	}
	if got := d.mapKeys[actMapAlerts].Help; got != "Description" {
		t.Errorf("A's help is %q", got)
	}
	help := ""
	for _, r := range mapHelpRows(d.mapKeys, false) {
		help += r.keys + " " + r.help + "\n"
	}
	if !strings.Contains(help, "A Description") {
		t.Errorf("Help does not name A's text Description:\n%s", help)
	}
	all := bodyText(d) + stripANSITest(d.mapStatusLine()) + help
	for _, other := range []string{"Area Alerts", "A Alerts"} {
		if strings.Contains(all, other) {
			t.Errorf("the text is still called %q", other)
		}
	}
	if got := mapDescWith.Label(); got != "With the map, on A" {
		t.Errorf("the Setting's choice is %q", got)
	}
}

// TestTheOverflowLineNamesItsRemedy is AX-9 (D-365): a description longer
// than the box ends with a line naming where the whole text is.
func TestTheOverflowLineNamesItsRemedy(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	box := d.descriptionBox(tuimaps.Size{Cols: 40, Rows: 6})
	text := strings.Join(strings.Fields(strings.NewReplacer("│", " ", "┌", " ", "┐", " ", "└", " ", "┘", " ", "─", " ").Replace(stripANSITest(strings.Join(box, "\n")))), " ") // the box's words, its wrapping undone
	if !strings.Contains(text, "the rest: Settings, Maps, Description, Instead of the map") {
		t.Errorf("the cut box does not name its remedy:\n%s", text)
	}
}

// TestTheOutlineDigitsAreKeyed is AX-8 (A-107 (9)): while the picture draws
// an alert's outline with its severity digit, a badge under the map keys
// each digit drawn to its word, and Help keys them all.
func TestTheOutlineDigitsAreKeyed(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	badges := stripANSITest(strings.Join(d.badgeRows(200), "\n"))
	if !strings.Contains(badges, outlineKeyLabel) || !strings.Contains(badges, "3 severe") {
		t.Errorf("the badges do not key the severe outline's digit: %q", badges)
	}
	if strings.Contains(badges, "4 extreme") {
		t.Errorf("the key names a digit the map did not draw: %q", badges)
	}
	if instead := openMap(t, Config{MapDescription: "instead"}, 133, 44); strings.Contains(stripANSITest(strings.Join(instead.badgeRows(200), "\n")), outlineKeyLabel) {
		t.Error("without the picture the badges key digits nobody sees")
	}
	help := strings.Join(d.helpLines(render.Opts{Width: 200}), "\n")
	for _, want := range []string{"4 extreme", "3 severe", "2 moderate", "1 minor", "? unknown"} {
		if !strings.Contains(help, want) {
			t.Errorf("Help does not key %q", want)
		}
	}
}
