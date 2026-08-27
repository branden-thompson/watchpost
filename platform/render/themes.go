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
// (GradStart/GradMid/GradEnd): the W A T C H P O S T wordmark is part of
// the palette, never a leftover from the default (HUM LEAD, UAT 107;
// pinned by TestEveryThemeOwnsItsTitleGradient).
func builtinOverrides() map[string]map[Token]string {
	return map[string]map[Token]string{
		"High Contrast": {
			TextBase: "255", TextBright: "231", FocusCell: "159", FocusName: "1;227",
			KeyChip: "1;38;5;16;48;5;252", KeyChipMuted: "38;5;250;48;2;70;70;70",
			GroupLocationBG: "48;2;120;120;120", GroupTodayBG: "48;2;70;110;160",
			GroupTomorrowBG: "48;2;60;150;150", GroupExtendedBG: "48;2;120;110;170",
			GroupSectionBG: "48;2;60;60;60", TempHi: "214", TempLo: "87", FireMark: "214", SeismicMark: "177", // bright light-purple, high legibility (0.11.0)
			TableHeader: "231", TableMuted: "255", TableName: "231", ModalTitle: "1;231", // Q4a-004: the table reads as bright as the rest
			GradStart: "#FFFFFF", GradMid: "#FFFF5F", GradEnd: "#5FFFFF", // white → its focus yellow → its low cyan
		},
		"Monochrome": {
			TempHi: "255", TempLo: "250", TrendUp: "250", TrendDown: "250",
			FocusName: "1;255", FocusCell: "255", NameAdvisory: "250", NameWarning: "255",
			ProviderOK: "250", ProviderDown: "255", RadioAccent: "250", RadioStation: "1;255",
			StatePlaying: "1;255", RepeatOn: "1;255", VizOn: "1;255",
			SpectrumLow: "245", SpectrumMid: "250", SpectrumHigh: "255", FireMark: "255", SeismicMark: "252", // greyscale on a monochrome theme — the glyph, not colour, distinguishes it (0.11.0)
			GroupLocationBG: "48;2;70;70;70", GroupTodayBG: "48;2;70;70;70",
			GroupTomorrowBG: "48;2;70;70;70", GroupExtendedBG: "48;2;70;70;70",
			AlertWarnFG: "38;5;255", AlertAdvFG: "38;5;250", AlertLabel: "250", AlertDanger: "255",
			AlertModalWarnFG: "38;2;235;235;235", AlertModalAdvFG: "38;2;200;200;200",
			AlertModalWarnBG: "48;2;40;40;40", AlertModalAdvBG: "48;2;30;30;30",
			GradStart: "#FFFFFF", GradMid: "#C0C0C0", GradEnd: "#808080",
			ChipFlashUp: "1;38;5;16;48;5;255", ChipFlashDown: "1;38;5;255;48;5;240",
			TableHeader: "1;255", TableMuted: "250", TableName: "255", ModalTitle: "1;255", // Q4a-004: no purple header on a monochrome theme
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
			AlertWarnFG: "38;5;203", AlertAdvFG: "38;5;221", AlertLabel: "221", AlertDanger: "203",
			AlertModalWarnFG: "38;2;254;68;80", AlertModalAdvFG: "38;2;254;222;93", AlertModalText: "231",
			AlertModalWarnBG: "48;2;60;20;40", AlertModalAdvBG: "48;2;60;50;20", ConfirmBG: "48;2;120;40;80",
			ModalFG: "38;5;231", ModalBGDark: "48;2;36;27;47", ModalBGLight: "48;2;52;41;79",
			WindowBGDark: "#262335", GradStart: "#FF7EDB", GradMid: "#36F9F6", GradEnd: "#FEDE5D",
			TableHeader: "213", TableMuted: "146", TableName: "231", ModalTitle: "1;231", // Q4a-004: neon pink headers, lavender attributes (≥ 4.5:1 on #262335)
		},
		"Solarized Night": {
			TextBase: "247", TextBright: "254", TempHi: "166", TempLo: "37", TrendUp: "136", TrendDown: "33",
			FocusName: "1;136", FocusCell: "109", NameAdvisory: "136", NameWarning: "160",
			ProviderOK: "64", ProviderDown: "160", RadioAccent: "64", RadioStation: "1;136",
			SpectrumLow: "64", SpectrumMid: "136", SpectrumHigh: "160", FireMark: "166", SeismicMark: "61", // solarized violet (0.11.0)
			GroupLocationBG: "48;2;7;54;66", GroupTodayBG: "48;2;38;79;120",
			GroupTomorrowBG: "48;2;42;107;103", GroupExtendedBG: "48;2;108;83;132",
			GroupSectionBG: "48;2;7;54;66", ModalBGDark: "48;2;0;43;54",
			WindowBGDark: "#002b36", GradStart: "#D33682", GradMid: "#268BD2", GradEnd: "#2AA198",
			TableHeader: "178", TableMuted: "247", TableName: "254", ModalTitle: "1;254", // Q4a-004: solarized yellow headers (≥ 4.5:1 on #002b36)
		},
	}
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
	themeMu.Lock()
	defer themeMu.Unlock()
	themeTable[name] = full
	themeGen.Add(1)
}

// ThemeNames lists the registered themes, default first, then sorted.
func ThemeNames() []string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	names := make([]string, 0, len(themeTable))
	for n := range themeTable {
		if n != DefaultThemeName {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return append([]string{DefaultThemeName}, names...)
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
