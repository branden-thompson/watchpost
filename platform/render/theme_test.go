package render

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestThemeTokensAllResolve(t *testing.T) {
	// UAT 16.4: every declared token has a value - a theme swap can never
	// leave a role unpainted.
	for _, tk := range []Token{TextBase, TextBright, TempHi, TempLo, TrendUp, TrendDown,
		FocusName, FocusCell, FocusPointer, NameAdvisory, NameWarning, ProviderOK, ProviderDown, KeyChip, KeyChipMuted, ChipFlashUp, ChipFlashDown, GroupText,
		GroupLocationBG, GroupTodayBG, GroupTomorrowBG, GroupExtendedBG, GroupSectionBG,
		AlertWarnFG, AlertWarnBG, AlertAdvFG, AlertAdvBG, AlertLabel, AlertDanger,
		RadioFG, RadioBG, RadioAccent, StateStopped, StatePlaying, RadioStation, RepeatOn, VizOn,
		ConfirmBG, AlertModalText, AlertModalWarnFG, AlertModalAdvFG, AlertModalWarnBG, AlertModalAdvBG, ModalFG, ModalBGDark, ModalBGLight, WindowBGDark, WindowBGLight,
		GradStart, GradMid, GradEnd} {
		if Tok(tk) == "" {
			t.Fatalf("token %q unresolved", tk)
		}
	}
	if r, g, b := hexRGB(GradStart); r != 0xDD || g != 0x51 || b != 0xD6 {
		t.Fatalf("hexRGB(GradStart) = %d,%d,%d", r, g, b)
	}
}

func TestThemeRegistrySwitchesLive(t *testing.T) {
	// UAT 53: themes register as overrides over the default; SetTheme flips
	// the live table so Tok() resolves the new value; unknown names refuse.
	defer SetTheme(DefaultThemeName)
	if got := ThemeNames()[0]; got != DefaultThemeName {
		t.Fatalf("default must list first, got %q", got)
	}
	RegisterTheme("Test Ember", map[Token]string{TempHi: "196"})
	if !SetTheme("Test Ember") || Tok(TempHi) != "196" {
		t.Fatalf("SetTheme must resolve live: %q", Tok(TempHi))
	}
	if Tok(TempLo) != "51" {
		t.Fatal("unlisted tokens inherit the default")
	}
	if SetTheme("nope") || ThemeName() != "Test Ember" {
		t.Fatal("unknown theme must be refused and leave the active theme")
	}
	for _, n := range []string{"High Contrast", "Monochrome", "Solarized Night", "Synthwave '84"} {
		if !SetTheme(n) {
			t.Fatalf("built-in %q must register", n)
		}
	}
}

func TestEveryThemeOwnsItsTitleGradient(t *testing.T) {
	// UAT 107 (HUM LEAD): the wordmark gradient follows the theme — every
	// built-in sets all three stops itself, and they differ from the default
	// (a new theme that forgets them fails here, not at UAT).
	def := builtinOverrides()
	for name, over := range def {
		for _, tok := range []Token{GradStart, GradMid, GradEnd} {
			v, ok := over[tok]
			if !ok || len(v) != 7 || v[0] != '#' {
				t.Fatalf("theme %q must set %s as #RRGGBB, got %q", name, tok, v)
			}
		}
		if def := defaultTheme(); over[GradStart] == def[GradStart] && over[GradMid] == def[GradMid] && over[GradEnd] == def[GradEnd] {
			t.Fatalf("theme %q reuses the default gradient", name)
		}
	}
}

func TestCompositeThemeTokensAreLegalRawSGR(t *testing.T) {
	// UAT 108 (High Contrast chips read white on a light chip): a composite
	// token rides the raw path, so a bare palette index inside it ("1;16;…")
	// is silently dropped by the terminal. Every composite value in every
	// theme must be fully qualified.
	check := func(theme string, table map[Token]string) {
		for tok, v := range table {
			raw := strings.HasPrefix(v, "38;") || strings.HasPrefix(v, "48;") || strings.Contains(v, ";38;") || strings.Contains(v, ";48;") // the raw path: composite colour lists (Tint-path tokens like "1;220" are qualified by SGR())
			if raw && !validRawSGR(v) {
				t.Errorf("theme %q token %s = %q is not a legal raw SGR list (qualify palette codes as 38;5;n / 48;5;n)", theme, tok, v)
			}
		}
	}
	check(DefaultThemeName, defaultTheme())
	for name, over := range builtinOverrides() {
		check(name, over)
	}
}

// Quality pass Q3 (R2-4): the body memo keys on the theme generation, so
// a switch or a (re)registration must move it.
func TestThemeGenerationMovesOnSwitchAndRegistration(t *testing.T) {
	g0 := ThemeGeneration()
	if !SetTheme("Monochrome") {
		t.Fatal("Monochrome is a built-in")
	}
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	g1 := ThemeGeneration()
	if g1 <= g0 {
		t.Fatalf("SetTheme must bump the generation: %d -> %d", g0, g1)
	}
	if SetTheme("no-such-theme") || ThemeGeneration() != g1 {
		t.Fatal("an unknown theme changes nothing, the generation included")
	}
	RegisterTheme("gen-test", map[Token]string{TempHi: "201"})
	if ThemeGeneration() <= g1 {
		t.Fatal("registering (or re-registering) a theme bumps the generation: its table may be the active one")
	}
}

// A11-10: the table's marks and the Help legend read one glyph set.
func TestGlyphsSwapAsOneSetUnderASCII(t *testing.T) {
	u, a := Opts{}.Glyphs(), Opts{ThinBands: true, ASCII: true}.Glyphs()
	if u.Play != "▶" || u.Repeat != "∞" || u.Fire != "◆" || u.Alert != "⚠" || u.Pointer != "›" {
		t.Fatalf("unicode set: %+v", u)
	}
	if a.Play != ">" || a.Repeat != "R" || a.Fire != "*" || a.Alert != "!" || a.Pointer != ">" {
		t.Fatalf("ascii set: %+v", a)
	}
	row := LocationRow{Index: 1, Name: "X", Playing: true, Repeat: true, Fire: 2, HasAlert: true, AlertCount: 1, Selected: true}
	plain := Opts{ThinBands: true, Width: 120, ASCII: true}.LocationTable([]LocationRow{row}, 0)
	for _, g := range []string{"▶", "∞", "◆", "⚠", "›"} {
		if strings.Contains(plain, g) {
			t.Fatalf("an ASCII table must carry no unicode mark %q:\n%s", g, plain)
		}
	}
	for _, g := range []string{"R", "2*", "1!", ">"} {
		if !strings.Contains(plain, g) {
			t.Fatalf("the ASCII table carries %q:\n%s", g, plain)
		}
	}
}

// xterm256RGB is the standard 256-colour palette: 16 basics, a 6×6×6 cube
// (levels 0, 95, 135, 175, 215, 255), 24 greys (8 + 10·i).
func xterm256RGB(i int) (r, g, b int) {
	basics := [16][3]int{{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0}, {0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
		{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0}, {0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255}}
	switch {
	case i < 16:
		return basics[i][0], basics[i][1], basics[i][2]
	case i < 232:
		levels := [6]int{0, 95, 135, 175, 215, 255}
		i -= 16
		return levels[i/36], levels[(i/6)%6], levels[i%6]
	default:
		v := 8 + 10*(i-232)
		return v, v, v
	}
}

// luminance is WCAG relative luminance.
func luminance(r, g, b int) float64 {
	lin := func(c int) float64 {
		v := float64(c) / 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// fgLuminance reads a theme value: a bare 256 index, "1;n", "38;5;n" or
// "38;2;r;g;b" (bold ignored; anything else is not a foreground value).
func fgLuminance(v string) (float64, bool) {
	parts := strings.Split(strings.TrimPrefix(v, "1;"), ";")
	switch {
	case len(parts) == 1:
		n, err := strconv.Atoi(parts[0])
		if err != nil || n > 255 {
			return 0, false
		}
		if n >= 90 && n <= 97 { // bright basics
			return luminance(xterm256RGB(n - 90 + 8)), true
		}
		if n >= 30 && n <= 37 {
			return luminance(xterm256RGB(n - 30)), true
		}
		return luminance(xterm256RGB(n)), true
	case len(parts) == 3 && parts[0] == "38" && parts[1] == "5":
		n, err := strconv.Atoi(parts[2])
		if err != nil || n > 255 {
			return 0, false
		}
		return luminance(xterm256RGB(n)), true
	case len(parts) == 5 && parts[0] == "38" && parts[1] == "2":
		r, _ := strconv.Atoi(parts[2])
		g, _ := strconv.Atoi(parts[3])
		b, _ := strconv.Atoi(parts[4])
		return luminance(r, g, b), true
	}
	return 0, false
}

// Quality pass Q4a-004 (A11-2): the table tokens every theme now owns must
// read at WCAG AA (≥ 4.5:1) against the theme's own window background;
// the text tokens ride the same check.
func TestThemeTokenContrastAA(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	for _, name := range ThemeNames() {
		if !SetTheme(name) {
			t.Fatal(name)
		}
		r, g, b := hexRGB(WindowBGDark)
		bg := luminance(r, g, b)
		for _, tok := range []Token{TableHeader, TableMuted, TableName, TextBase, TextBright, ModalTitle} {
			fg, ok := fgLuminance(Tok(tok))
			if !ok {
				t.Fatalf("%s %s: %q is not a foreground value", name, tok, Tok(tok))
			}
			hi, lo := max(fg, bg), min(fg, bg)
			if ratio := (hi + 0.05) / (lo + 0.05); ratio < 4.5 {
				t.Errorf("%s: %s = %q reads %.2f:1 on %s — below AA (4.5:1)", name, tok, Tok(tok), ratio, Tok(WindowBGDark))
			}
		}
	}
}
