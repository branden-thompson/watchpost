package render

import (
	"strings"
	"testing"
)

// Every theme's alert tones read at AA on their tints — lifted at
// registration when the theme's own value does not (HUM LEAD 2026-08-28,
// round 4 B-05). The lift keeps the hue: the default's warning red stays a
// red, brighter.
func TestAlertTonesReadAAInEveryTheme(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	for _, name := range ThemeNames() {
		if !SetTheme(name) {
			t.Fatal(name)
		}
		for _, pair := range [][2]Token{{AlertModalWarnFG, AlertModalWarnBG}, {AlertModalAdvFG, AlertModalAdvBG}} {
			if c := Contrast("1;"+Tok(pair[0]), Tok(pair[1])); c < AAContrast {
				t.Errorf("%s: %s %q on %s %q reads %.2f:1", name, pair[0], Tok(pair[0]), pair[1], Tok(pair[1]), c)
			}
		}
	}
}

func TestLiftToAAKeepsTheHueAndStopsAtAA(t *testing.T) {
	raw, tint := "38;2;190;84;84", "48;2;85;9;9" // the default's warning pair before the lift: 4.20:1
	if c := Contrast(raw, tint); c >= AAContrast {
		t.Fatalf("the control must start below AA: %.2f", c)
	}
	lifted := LiftToAA(raw, tint)
	if lifted == raw || !strings.HasPrefix(lifted, "38;2;") {
		t.Fatalf("lifted: %q", lifted)
	}
	if c := Contrast(lifted, tint); c < AAContrast || c > AAContrast+0.6 {
		t.Fatalf("lifted just past AA, not to white: %.2f (%q)", c, lifted)
	}
	r, g, b, _ := fgRGB(lifted)
	if r <= g || r <= b || r <= 190 {
		t.Fatalf("the hue stays red, brighter: %d %d %d", r, g, b)
	}
	// Already AA, a bold prefix, a non-colour: unchanged.
	if got := LiftToAA("1;97", tint); got != "1;97" {
		t.Fatalf("bold white on the tint already reads: %q", got)
	}
	if got := LiftToAA("1;"+raw, tint); !strings.HasPrefix(got, "1;38;2;") {
		t.Fatalf("the bold prefix is kept: %q", got)
	}
	if got := LiftToAA("38;2;200;200;200", "49"); got != "38;2;200;200;200" {
		t.Fatalf("no tint, no lift: %q", got)
	}
	// A light tint lifts toward black.
	dark := LiftToAA("38;2;200;180;60", "48;2;240;230;150")
	if r, g, b, _ := fgRGB(dark); r+g+b >= 200+180+60 {
		t.Fatalf("on a light tint the tone darkens: %q", dark)
	}
}

// The column-title row's dip reads at AA in every theme under bold white
// (REVIEW R5-A-06: the gate checked the SGR, not the ratio).
func TestTableHeaderDipReadsAAInEveryTheme(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	for _, name := range ThemeNames() {
		if !SetTheme(name) {
			t.Fatal(name)
		}
		for _, band := range []Token{GroupLocationBG, GroupTodayBG, GroupTomorrowBG, GroupExtendedBG} {
			if c := Contrast(Tok(GroupText), TableHeaderTone(band)); c < AAContrast {
				t.Errorf("%s: the title text %q on the %s dip %q reads %.2f:1", name, Tok(GroupText), band, TableHeaderTone(band), c)
			}
		}
	}
}

// Every painted pair in the register reads at AA in every theme — the same
// list registration lifts (HUM LEAD ruling 2026-08-29, R5-C-04 widened;
// the Watchpost Light theme rides the same gate toward black).
func TestEveryPaintedPairReadsAAInEveryTheme(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	for _, name := range ThemeNames() {
		if !SetTheme(name) {
			t.Fatal(name)
		}
		table := activeTable()
		for _, p := range aaPairs() {
			for _, bg := range p.on {
				if c := Contrast(Tok(p.fg), bgOf(table, bg)); c < AAContrast {
					t.Errorf("%s: %s %q on %s %q reads %.2f:1", name, p.fg, Tok(p.fg), bg, bgOf(table, bg), c)
				}
			}
		}
	}
	if !SetTheme("Watchpost Light") {
		t.Fatal("the light theme is registered")
	}
	if l, _ := bgLuminance(bgOf(activeTable(), WindowBGDark)); l < 0.5 {
		t.Fatalf("the light theme's ground is light: %.2f", l)
	}
}

// FgSGR: a bare index becomes 38;5;N, a basic code and a full value stay —
// the frame base once read "38;5;38;2;40;40;40" (index 38, faint, black
// background) for a truecolor TextBase.
func TestFgSGRFormsAValidEscape(t *testing.T) {
	for in, want := range map[string]string{"250": "38;5;250", "97": "97", "1;97": "1;97", "38;2;40;40;40": "38;2;40;40;40", "": ""} {
		if got := FgSGR(in); got != want {
			t.Errorf("FgSGR(%q) = %q, want %q", in, got, want)
		}
	}
}

// Watchpost Light paints no dark ground: every truecolor background in its
// table is light, and every text token dark (HUM LEAD 2026-08-29 screenshot).
func TestTheLightThemePaintsNoDarkGround(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	if !SetTheme(LightThemeName) {
		t.Fatal("registered")
	}
	for tok, v := range activeTable() {
		if l, ok := bgLuminance(v); ok && l < 0.35 {
			t.Errorf("%s %q is a dark ground (luminance %.2f)", tok, v, l)
		}
	}
	for _, tok := range []Token{TextBase, TextBright, ModalFG, ModalTitle, GroupText, TickerFG, AlertModalText, TableName, TableMuted} {
		if l, ok := fgLuminance(Tok(tok)); !ok || l > 0.3 {
			t.Errorf("%s %q is not dark text (luminance %.2f)", tok, Tok(tok), l)
		}
	}
	if names := ThemeNames(); len(names) < 2 || names[1] != LightThemeName {
		t.Fatalf("the light theme is second in the picker: %v", names)
	}
}
