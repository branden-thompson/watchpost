package render

import (
	"strings"
	"testing"
)

func TestThemeTokensAllResolve(t *testing.T) {
	// UAT 16.4: every declared token has a value - a theme swap can never
	// leave a role unpainted.
	for _, tk := range []Token{TextBase, TextBright, TempHi, TempLo, TrendUp, TrendDown,
		FocusName, FocusCell, FocusPointer, NameAdvisory, NameWarning, ProviderOK, ProviderDown, KeyChip, KeyChipMuted, ChipFlashUp, ChipFlashDown, GroupText,
		GroupLocationBG, GroupTodayBG, GroupTomorrowBG, GroupExtendedBG, GroupSectionBG,
		AlertLabel, AlertDanger,
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
	if a.Play != "*" || a.Repeat != "R" || a.Fire != "*" || a.Alert != "!" || a.Pointer != ">" {
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
		for _, tok := range []Token{TableMuted, TableName, TextBase, TextBright, ModalTitle} {
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

// The category tints are theme-INDEPENDENT (SAM-D-7): every theme except
// Monochrome resolves them to the default. The planted override is the
// positive control — a guard that passes on an empty set proves nothing.
func TestEventCategoryTokensAreThemeIndependent(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	def := defaultTheme()
	for _, name := range ThemeNames() {
		if name == "Monochrome" || name == LightThemeName { // greys, and the pale tints of the one light theme (HUM LEAD 2026-08-29)
			continue
		}
		if !SetTheme(name) {
			t.Fatal(name)
		}
		for _, tok := range []Token{EventCatRedBG, EventCatOrangeBG, EventCatYellowBG, EventCatWatchBG, EventCatStmtBG, EventCatBlueBG} {
			if Tok(tok) != def[tok] {
				t.Errorf("%s overrides %s: %q (must inherit %q)", name, tok, Tok(tok), def[tok])
			}
		}
	}
	// Positive control: a registered theme that DOES override must be caught.
	RegisterTheme("planted-control", map[Token]string{EventCatRedBG: "48;2;1;2;3"})
	t.Cleanup(func() { UnregisterTheme("planted-control") })
	SetTheme("planted-control")
	if Tok(EventCatRedBG) == def[EventCatRedBG] {
		t.Fatal("the control override was not applied — the guard is vacuous")
	}
	UnregisterTheme("planted-control")
	if ThemeName() != DefaultThemeName {
		t.Fatal("unregistering the active theme must restore the default")
	}
}

// Every token the window paints — the substrate's text and the table's own
// (SevereTableTokens, R3-B-03) — on every category tint, on both modal
// substrates, in every theme, at AA.
func TestCategoryToneContrastAA(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	for _, name := range ThemeNames() {
		SetTheme(name)
		for _, dark := range []bool{true, false} {
			for _, hue := range []Token{EventCatRedBG, EventCatOrangeBG, EventCatYellowBG, EventCatWatchBG, EventCatStmtBG, EventCatBlueBG} {
				fg, bg := CategoryTone(hue, dark)
				bl, ok := bgLuminance(bg)
				if !ok {
					t.Fatalf("%s %s dark=%v: unreadable tone %q", name, hue, dark, bg)
				}
				fgs := map[string]string{"substrate": fg}
				for _, tok := range SevereTableTokens() {
					fgs[string(tok)] = Tok(tok)
				}
				for label, v := range fgs {
					fl, ok := fgLuminance(v)
					if !ok {
						t.Fatalf("%s %s dark=%v: unreadable fg %s=%q", name, hue, dark, label, v)
					}
					hi, lo := max(fl, bl), min(fl, bl)
					if ratio := (hi + 0.05) / (lo + 0.05); ratio < 4.5 {
						t.Errorf("%s %s dark=%v %s: %.2f:1 below AA", name, hue, dark, label, ratio)
					}
				}
				// The column-header band: bold white on the lifted tint of the hue.
				hl, ok1 := bgLuminance(SevereHeaderTone(hue))
				gl, ok2 := fgLuminance(severeHeaderFG(SevereHeaderTone(hue)))
				if !ok1 || !ok2 {
					t.Fatalf("%s %s: unreadable header tone", name, hue)
				}
				hi, lo := max(gl, hl), min(gl, hl)
				if ratio := (hi + 0.05) / (lo + 0.05); ratio < 4.5 {
					t.Errorf("%s %s header band: %.2f:1 below AA", name, hue, ratio)
				}
			}
		}
	}
}
