package render

import (
	"regexp"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Theme registry (UAT 53): the 't' chooser swaps the active token table
// at runtime — every emission resolves through Tok() on the next frame,
// so users never restart to see new colors. Built-ins ship here; user
// theme files (token -> value) register through RegisterTheme.

// DefaultThemeName is the palette the B3 UAT sessions directed.
const DefaultThemeName = "Watchpost"

// themeValue is the shape of a token value: SGR parameters ("1;38;5;220",
// "48;2;66;94;122") or a #RRGGBB hex for the window/gradient tokens.
var themeValue = regexp.MustCompile(`^(\d+(;\d+)*|#[0-9A-Fa-f]{6})$`)

var (
	themeMu    sync.RWMutex
	themeName  = DefaultThemeName
	themeTable = map[string]map[Token]string{DefaultThemeName: defaultTheme()}
	themeGen   atomic.Uint64 // bumps on every SetTheme/RegisterTheme: the body memo's theme key (quality pass Q3, R2-4)
)

// ThemeGeneration counts theme changes since launch. A renderer that
// memoises tinted output keys on it: any switch or (re)registration
// changes every Tok() value it may have baked in.
func ThemeGeneration() uint64 { return themeGen.Load() }

// builtinOverrides defines the shipped alternates as deltas over the
// default table (unlisted tokens inherit) — one source per theme.
//
// Every theme, built-in or user file, sets its own title gradient
// (GradStart/GradMid/GradEnd): the WATCHPOST wordmark is part of
// the palette, never a leftover from the default (HUM LEAD, UAT 107;
// pinned by TestEveryThemeOwnsItsTitleGradient).
func builtinOverrides() map[string]map[Token]string {
	all := map[string]map[Token]string{
		LightThemeName: lightOverrides(),
		"High Contrast": {
			TextBase: "255", TextBright: "231", FocusCell: "159", FocusName: "1;227",
			KeyChip: "1;38;5;16;48;5;252", KeyChipMuted: "38;5;250;48;2;70;70;70",
			GroupLocationBG: "48;2;120;120;120", GroupTodayBG: "48;2;70;110;160",
			GroupTomorrowBG: "48;2;60;150;150", GroupExtendedBG: "48;2;120;110;170",
			RailLiveBG: "48;2;170;70;70", RailNextBG: "48;2;180;120;60", RailQueueBG: "48;2;70;110;160",
			CardBG: "48;2;45;45;45", CardOperatorBG: "48;2;75;65;40", CardText: "255",
			CardEmptyBG:    "48;2;100;100;100",                                                                // this theme separates by LIGHTNESS, so its grey is the brightest
			GroupSectionBG: "48;2;60;60;60", TempHi: "214", TempLo: "87", FireMark: "214", SeismicMark: "177", // bright light-purple, high legibility (0.11.0)
			TableMuted: "255", TableName: "231", ModalTitle: "1;231", // Q4a-004: the table reads as bright as the rest
			GradStart: "#FFFFFF", GradMid: "#FFFF5F", GradEnd: "#5FFFFF", // white → its focus yellow → its low cyan
			TitleEdition: "1;159", // its own pale blue, at this theme's brightness
		},
		"Monochrome": {
			TempHi: "255", TempLo: "250", TrendUp: "250", TrendDown: "250",
			FocusName: "1;255", FocusCell: "255", NameAdvisory: "250", NameWarning: "255",
			ProviderOK: "250", ProviderDown: "255", RadioAccent: "250", RadioStation: "1;255",
			StatePlaying: "1;255", RepeatOn: "1;255", VizOn: "1;255",
			SpectrumLow: "245", SpectrumMid: "250", SpectrumHigh: "255", FireMark: "255", SeismicMark: "252", // greyscale on a monochrome theme — the glyph, not colour, distinguishes it (0.11.0)
			// The lane ramp, EVENLY SPACED IN L* across the six lanes (HUM LEAD, UAT
			// 2026-08-30). Shade is the only thing telling lanes apart on this
			// theme, and the ramp was cut for four: adding two put Watches and
			// Advisories four bytes apart, which is 1.06:1 — the same band twice.
			//
			// Spaced in CIE L*, not in bytes, because even byte steps are not even
			// STEPS to the eye. Six lanes over the range the four already used
			// (95..30) is 5.8 L* apart each: 95 81 68 55 42 30, in rotation order,
			// brightest first. Three of the four original values land unchanged.
			TickerEmergencyBG: "48;2;110;110;110", TickerDisasterBG: "48;2;95;95;95", TickerMarineBG: "48;2;81;81;81", TickerWarningBG: "48;2;68;68;68", TickerWatchBG: "48;2;55;55;55",
			TickerAdvisoryBG: "48;2;42;42;42", TickerStatementBG: "48;2;30;30;30", EventCatEmergencyBG: "48;2;70;70;70",
			// The tint ramp, evened in L* like the lane ramp above and running one
			// rung darker throughout: 70 64 58 52 46 40, in the same severity
			// order. Category by shade on monochrome; the tab glyph carries
			// identity (0.13.0).
			EventCatDisasterBG: "48;2;70;70;70", EventCatMarineBG: "48;2;64;64;64", EventCatWarningBG: "48;2;58;58;58",
			EventCatWatchBG: "48;2;52;52;52", EventCatAdvisoryBG: "48;2;46;46;46", EventCatStmtBG: "48;2;40;40;40", EventCatForecastBG: "48;2;34;34;34",
			GroupLocationBG: "48;2;70;70;70", GroupTodayBG: "48;2;70;70;70",
			GroupTomorrowBG: "48;2;70;70;70", GroupExtendedBG: "48;2;70;70;70",
			// MONOCHROME KEEPS THE THREE APART BY LIGHTNESS, because it has no
			// hue to keep them apart with — LIVE is the brightest, which is the
			// same ordering the eye reads from red/orange/blue.
			RailLiveBG: "48;2;96;96;96", RailNextBG: "48;2;72;72;72", RailQueueBG: "48;2;52;52;52",
			CardBG: "48;2;30;30;30", CardOperatorBG: "48;2;44;44;44", CardText: "250",
			// AND MONOCHROME HAS ONLY LIGHTNESS, so the empty slot sits above both
			// card grounds and below the rail's brightest band.
			CardEmptyBG: "48;2;62;62;62",
			AlertLabel:  "250", AlertDanger: "255",
			AlertModalWarnFG: "38;2;235;235;235", AlertModalAdvFG: "38;2;200;200;200",
			AlertModalWarnBG: "48;2;40;40;40", AlertModalAdvBG: "48;2;30;30;30",
			// The modal TILE and the destructive-confirm tile, which every other
			// theme paints and this one was inheriting: the default's blue slate
			// and its dark red showed through on a greyscale theme, in every
			// window.
			//
			// #272727 is the blue slate's own LUMINANCE as a grey, so a modal sits
			// at the same visual depth here as everywhere else — raised off the
			// #131313 window by the same amount the colour was raising it.
			//
			// The confirm tile goes LIGHTER rather than matching, because its job
			// is to say "this one is different" and it was saying it in red. Shade
			// is the only voice this theme has for that.
			ModalBGDark: "48;2;39;39;39", ModalBGLight: "48;2;39;39;39", ConfirmBG: "48;2;58;58;58",
			GradStart: "#FFFFFF", GradMid: "#C0C0C0", GradEnd: "#808080",
			// No blue to be had: BOLD WHITE says "edition" the only way this
			// theme can say anything, the same choice its list focus makes.
			TitleEdition: "1;255",
			ChipFlashUp:  "1;38;5;16;48;5;255", ChipFlashDown: "1;38;5;255;48;5;240",
			// A LIST's focus, in this theme's vocabulary. The default carries it in
			// yellow and leaves the label unbolded, because the colour is doing the
			// work; here there is no colour, so BOLD does it — the same
			// distinction said the only way this theme can say it. Without these
			// the Settings window kept its yellow pointer on a greyscale theme.
			ListPointer: "1;255", ListFocus: "1;255",
			TableMuted: "250", TableName: "255", ModalTitle: "1;255", // Q4a-004: a monochrome theme
		},
		// Synthwave '84 (UAT 105; palette from robb0wen/synthwave-vscode:
		// bg #262335 / #241b2f, neon pink #ff7edb, cyan #36f9f6, yellow
		// #fede5d, orange #ff8b39, mint #72f1b8, red #fe4450, comment
		// #848bbd). 256-palette nearest for foregrounds, truecolor for tiles.
		"Synthwave '84": {
			TextBase: "146", TextBright: "231", TempHi: "215", TempLo: "87", TrendUp: "215", TrendDown: "87",
			FocusName: "1;213", FocusCell: "159", FocusPointer: "1;231", NameAdvisory: "221", NameWarning: "203",
			ProviderOK: "121", ProviderDown: "203", RadioAccent: "51", StateStopped: "1;103", StatePlaying: "1;121",
			RadioStation: "1;213", RepeatOn: "1;221", VizOn: "1;51", SpectrumLow: "121", SpectrumMid: "213", SpectrumHigh: "51",
			FireMark: "215", SeismicMark: "171", KeyChip: "1;38;5;231;48;2;52;41;79", KeyChipMuted: "38;5;103;48;2;36;27;47", GroupText: "1;231", // neon magenta-purple, on-palette (0.11.0)
			ChipFlashUp: "1;38;5;16;48;5;121", ChipFlashDown: "1;38;5;231;48;5;203",
			GroupLocationBG: "48;2;52;41;79", GroupTodayBG: "48;2;54;30;90",
			GroupTomorrowBG: "48;2;30;70;90", GroupExtendedBG: "48;2;80;40;90", GroupSectionBG: "48;2;36;27;47",
			RailLiveBG: "48;2;110;35;60", RailNextBG: "48;2;110;70;40", RailQueueBG: "48;2;45;55;100",
			CardBG: "48;2;36;27;47", CardOperatorBG: "48;2;60;48;40", CardText: "231",
			CardEmptyBG: "48;2;62;56;70", // desaturated toward grey, still in this theme's family
			AlertLabel:  "221", AlertDanger: "203",
			AlertModalWarnFG: "38;2;254;68;80", AlertModalAdvFG: "38;2;254;222;93", AlertModalText: "231",
			AlertModalWarnBG: "48;2;60;20;40", AlertModalAdvBG: "48;2;60;50;20", ConfirmBG: "48;2;120;40;80",
			ModalFG: "38;5;231", ModalBGDark: "48;2;36;27;47", ModalBGLight: "48;2;52;41;79",
			WindowBGDark: "#262335", GradStart: "#FF7EDB", GradMid: "#36F9F6", GradEnd: "#FEDE5D",
			TitleEdition: "1;159",                                      // pale neon blue, on-palette beside the cyan
			TableMuted:   "146", TableName: "231", ModalTitle: "1;231", // Q4a-004: lavender attributes (≥ 4.5:1 on #262335)
		},
		"Solarized Night": {
			TextBase: "247", TextBright: "254", TempHi: "166", TempLo: "37", TrendUp: "136", TrendDown: "33",
			FocusName: "1;136", FocusCell: "109", NameAdvisory: "136", NameWarning: "160",
			ProviderOK: "64", ProviderDown: "160", RadioAccent: "64", RadioStation: "1;136",
			SpectrumLow: "64", SpectrumMid: "136", SpectrumHigh: "160", FireMark: "166", SeismicMark: "61", // solarized violet (0.11.0)
			GroupLocationBG: "48;2;7;54;66", GroupTodayBG: "48;2;38;79;120",
			GroupTomorrowBG: "48;2;42;107;103", GroupExtendedBG: "48;2;108;83;132",
			RailLiveBG: "48;2;110;44;40", RailNextBG: "48;2;115;74;30", RailQueueBG: "48;2;38;79;120",
			CardBG: "48;2;7;54;66", CardOperatorBG: "48;2;70;62;30", CardText: "254",
			CardEmptyBG:    "48;2;46;62;68", // solarized's own grey-slate, one step off base02
			GroupSectionBG: "48;2;7;54;66", ModalBGDark: "48;2;0;43;54",
			WindowBGDark: "#002b36", GradStart: "#D33682", GradMid: "#268BD2", GradEnd: "#2AA198",
			TitleEdition: "1;109",                                      // solarized's readable blue-grey, its own light blue
			TableMuted:   "247", TableName: "254", ModalTitle: "1;254", // Q4a-004: solarized attributes (≥ 4.5:1 on #002b36)
		},
	}
	// The Omarchy Quattro palettes, mapped systematically (quattro.go).
	for name, p := range quattroThemes() {
		all[name] = quattroOverrides(p)
	}
	return all
}

func init() {
	for name, over := range builtinOverrides() {
		RegisterTheme(name, over)
	}
}

// RegisterTheme adds (or replaces) a theme as overrides over the default
// table; unlisted tokens inherit. User theme files land here from app.
func RegisterTheme(name string, overrides map[Token]string) {
	if err := invariant.Check(name != "", "theme name is required"); err != nil {
		return
	}
	base := defaultTheme()
	full := make(map[Token]string, len(base))
	for k, v := range base {
		full[k] = v
	}
	for k, v := range overrides {
		if v != "" && themeValue.MatchString(v) { // SGR params or a hex colour only (red-team 0.9.0 S-F6): a theme file must not smuggle escape sequences into the frame
			full[k] = v
		}
	}
	// A theme paints its own ground whatever the terminal reports: a theme
	// that sets only the dark ground/tile gets the same for the light slot
	// (HUM LEAD 2026-08-29 — light terminals are served by Watchpost Light).
	if _, ok := overrides[WindowBGLight]; !ok {
		full[WindowBGLight] = full[WindowBGDark]
	}
	if _, ok := overrides[ModalBGLight]; !ok {
		full[ModalBGLight] = full[ModalBGDark]
	}
	themeMu.Lock()
	defer themeMu.Unlock()
	themeTable[name] = withAA(full)
	themeGen.Add(1)
}

// UnregisterTheme removes a registered theme (tests plant and remove control
// themes); the default is restored if the removed one was active.
func UnregisterTheme(name string) {
	themeMu.Lock()
	defer themeMu.Unlock()
	if name == DefaultThemeName {
		return
	}
	delete(themeTable, name)
	if themeName == name {
		themeName = DefaultThemeName
	}
	themeGen.Add(1)
}

// ThemeNames lists the registered themes, default first, then sorted.
func ThemeNames() []string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	names := make([]string, 0, len(themeTable))
	for n := range themeTable {
		if n != DefaultThemeName && n != LightThemeName {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	head := []string{DefaultThemeName}
	if _, ok := themeTable[LightThemeName]; ok {
		head = append(head, LightThemeName) // second in the picker (HUM LEAD 2026-08-29)
	}
	return append(head, names...)
}

// SetTheme activates a registered theme; false when unknown (the active
// theme is untouched). Takes effect on the next frame.
func SetTheme(name string) bool {
	themeMu.Lock()
	defer themeMu.Unlock()
	if _, ok := themeTable[name]; !ok {
		return false
	}
	themeName = name
	themeGen.Add(1)
	return true
}

// ThemeName is the active theme.
func ThemeName() string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	return themeName
}

// activeTable resolves the live token table.
func activeTable() map[Token]string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	return themeTable[themeName]
}

// lightOverrides is Watchpost Light (HUM LEAD ruling 2026-08-29, R5-C-03):
// the one light theme — a light ground, dark text, the bands as pale
// tints; every pair passes the same AA lift as the dark themes (toward
// black here). The alert tile and the severe window keep their dark
// grounds: their text and marks are painted on the theme-independent
// category tints too, and no one tone reads on both a dark tint and a
// light window.
func lightOverrides() map[Token]string {
	return map[Token]string{
		WindowBGDark: "#ECECEC", WindowBGLight: "#ECECEC",
		ModalBGDark: "48;2;220;226;232", ModalBGLight: "48;2;220;226;232",
		TextBase: "38;2;40;40;40", TextBright: "1;38;2;0;0;0", ModalTitle: "1;38;2;0;0;0", ModalFG: "38;2;40;40;40",
		TableMuted: "38;2;95;95;95", TableName: "38;2;0;0;0",
		GroupText: "1;38;2;20;20;20", GroupLocationBG: "48;2;200;200;200", GroupTodayBG: "48;2;169;196;224",
		GroupTomorrowBG: "48;2;169;224;224", GroupExtendedBG: "48;2;196;196;224", GroupSectionBG: "48;2;221;221;221",
		// THE LIGHT THEME INVERTS THE RELATIONSHIP, NOT THE HUE. A band on a
		// light ground is a PALE wash of the same colour: dark values here would
		// read as holes punched in the page.
		RailLiveBG: "48;2;240;200;200", RailNextBG: "48;2;245;215;185", RailQueueBG: "48;2;169;196;224",
		CardBG: "48;2;235;235;238", CardOperatorBG: "48;2;245;238;215", CardText: "38;2;40;40;40",
		// AND THE LIGHT THEME GOES DARKER, not lighter: on a light ground the
		// dormant slot is the one that recedes, and recede means grey-toward-ink.
		CardEmptyBG: "48;2;209;209;212",
		KeyChip:     "1;38;2;0;0;0;48;2;190;190;190", KeyChipMuted: "38;2;120;120;120;48;2;225;225;225",
		ChipFlashUp: "1;38;2;255;255;255;48;2;0;120;40", ChipFlashDown: "1;38;2;255;255;255;48;2;170;0;0",
		FocusName: "1;38;2;120;80;0", FocusCell: "38;2;0;70;140", FocusPointer: "1;38;2;0;0;0",
		// DARK blue on the light ground: "light blue" is a relationship to the
		// paper, not an absolute, and a pale one here would vanish.
		TitleEdition: "1;38;2;0;70;140",
		NameAdvisory: "38;2;110;100;0", NameWarning: "38;2;150;30;30", ProviderOK: "38;2;0;110;40", ProviderDown: "38;2;170;0;0",
		AlertLabel: "38;2;120;90;0", AlertDanger: "38;2;170;0;0",
		RadioFG: "38;2;40;40;40", RadioAccent: "38;2;0;110;40", StateStopped: "1;38;2;110;110;110", StatePlaying: "1;38;2;0;110;40",
		RadioStation: "1;38;2;120;90;0", RepeatOn: "1;38;2;0;110;40", VizOn: "1;38;2;0;110;40",
		SpectrumLow: "38;2;0;110;40", SpectrumMid: "38;2;150;110;0", SpectrumHigh: "38;2;170;0;0",
		TempHi: "38;2;190;80;0", TempLo: "38;2;0;100;140", TrendUp: "38;2;130;90;20", TrendDown: "38;2;20;110;120",
		FireMark: "38;2;190;80;0", SeismicMark: "38;2;110;60;170",
		TickerMutedFG:    "38;2;90;90;90",
		ConfirmBG:        "48;2;250;205;205",
		AlertModalWarnFG: "38;2;150;30;30", AlertModalAdvFG: "38;2;100;90;0", AlertModalText: "38;2;20;20;20",
		AlertModalWarnBG: "48;2;250;215;215", AlertModalAdvBG: "48;2;245;238;195",
		// The category tints and the ticker lanes, pale (the dark themes share the
		// default's deep tints; a light theme paints no dark ground — HUM LEAD 2026-08-29).
		// The [w] tints mirror the lanes here too — same hue, stepped TOWARD white
		// rather than away from it, because this theme's bands are pale and its
		// text is dark. The step is 45 % of the way, further on the two lanes
		// whose ruled colour is saturated enough that 45 % left the row text
		// under AA.
		EventCatDisasterBG: "48;2;252;225;225", EventCatWarningBG: "48;2;236;225;218", EventCatWatchBG: "48;2;252;247;214",
		EventCatAdvisoryBG: "48;2;252;236;214", EventCatStmtBG: "48;2;215;232;219", EventCatForecastBG: "48;2;228;230;233", EventCatEmergencyBG: "48;2;246;218;238", EventCatMarineBG: "48;2;225;236;252",
		TickerEmergencyBG: "48;2;250;200;200", TickerDisasterBG: "48;2;250;195;235", TickerWarningBG: "48;2;250;220;180", TickerWatchBG: "48;2;250;240;180", TickerMarineBG: "48;2;200;220;250",
		TickerAdvisoryBG: "48;2;205;178;159", TickerStatementBG: "48;2;121;179;135", // #CDB29F / #79B387
		TickerFG:  "1;38;2;20;20;20",
		GradStart: "#A0269A", GradMid: "#1F5FA8", GradEnd: "#2E8B6B", // the default gradient, deepened for a light ground
	}
}

// LightThemeName is the one light theme (HUM LEAD ruling 2026-08-29).
const LightThemeName = "Watchpost Light"
