// Semantic color tokens (UAT 16.4): every color the UI emits resolves
// through the theme table below — the theming surface. Swapping palettes
// (the 't' theme chooser, CLIAmp-style) replaces this table; call sites
// never change. Values are SGR parameter lists (256-palette codes ride
// go-studs ColorSequence; truecolor entries are pre-expanded 38;2/48;2
// params for the raw-SGR path; window/gradient entries are #RRGGBB hex).
package render

// Token names a semantic color role.
type Token string

// The token vocabulary. Add roles, not colors: a new UI element gets a new
// token here (and a value in every theme), never an inline code.
const (
	TextBase   Token = "text.base"   // default body text
	TextBright Token = "text.bright" // emphasized plain text (timestamps)

	TempHi    Token = "temp.hi"    // high temperatures
	TempLo    Token = "temp.lo"    // low temperatures
	TrendUp   Token = "trend.up"   // ↗ (muted orange)
	TrendDown Token = "trend.down" // ↘ (muted cyan)

	FocusName    Token = "name.focus"    // focused row name
	FocusCell    Token = "cell.focus"    // focused row: grey data cells read light blue (UAT 50.1)
	FocusPointer Token = "pointer.focus" // focused row pointer: bold white (UAT 50.2)
	NameAdvisory Token = "name.advisory" // location under advisory/statement
	NameWarning  Token = "name.warning"  // location under warning/alert

	ProviderOK   Token = "provider.ok"
	ProviderDown Token = "provider.down"

	KeyChip       Token = "chip.key"        // key-cap chips (composite style)
	KeyChipMuted  Token = "chip.key.muted"  // disabled controls: ~50% opacity (UAT 21.1)
	ChipFlashUp   Token = "chip.flash.up"   // [+] acknowledged: green blink (UAT 41)
	ChipFlashDown Token = "chip.flash.down" // [-] acknowledged: red blink
	GroupText     Token = "group.text"      // group band text (composite style)

	GroupLocationBG Token = "group.location.bg"
	GroupTodayBG    Token = "group.today.bg"
	GroupTomorrowBG Token = "group.tomorrow.bg"
	GroupExtendedBG Token = "group.extended.bg"
	GroupSectionBG  Token = "group.section.bg" // RECENT / SEARCHED section band (UAT 43)

	AlertWarnFG Token = "alert.warning.fg"
	AlertWarnBG Token = "alert.warning.bg"
	AlertAdvFG  Token = "alert.advisory.fg"
	AlertAdvBG  Token = "alert.advisory.bg"
	AlertLabel  Token = "alert.label"  // watch/advisory yellow (panel tints)
	AlertDanger Token = "alert.danger" // warning red (panel tints, provider down)

	RadioFG      Token = "radio.fg"
	RadioBG      Token = "radio.bg"
	RadioAccent  Token = "radio.accent"        // title, VOL fill
	StateStopped Token = "radio.stopped"       // ■ STOPPED
	StatePlaying Token = "radio.playing"       // playing/paused (B4)
	RadioStation Token = "radio.station"       // location name in the player: bold bright yellow (UAT 40.4, CLIAmp)
	RepeatOn     Token = "radio.repeat.on"     // 'Repeat: On' label: yellow bold (UAT 52.1)
	VizOn        Token = "radio.viz.on"        // 'Viz: On' label: green bold (UAT 52.2)
	SpectrumLow  Token = "radio.spectrum.low"  // visualizer gradient: bottom third / quiet bands (UAT 92, CLIAmp)
	SpectrumMid  Token = "radio.spectrum.mid"  // middle third / mid bands
	SpectrumHigh Token = "radio.spectrum.high" // top third / loud bands

	FireMark Token = "fire.mark" // ▲ in the row marks and the FIRE section: orange (B5)

	ConfirmBG Token = "confirm.bg" // destructive-action confirmation tile (UAT 26.2)

	AlertModalWarnFG Token = "alert.modal.warning.fg"  // modal alert text, warning-grade (UAT 28.4)
	AlertModalAdvFG  Token = "alert.modal.advisory.fg" // modal alert text, advisory-grade (UAT 28.3)

	AlertModalText   Token = "alert.modal.text"       // [A] modal body text: white for contrast (UAT 55)
	AlertModalWarnBG Token = "alert.modal.warning.bg" // [A] details tile, warning-grade
	AlertModalAdvBG  Token = "alert.modal.advisory.bg"

	ModalFG      Token = "modal.fg"
	ModalBGDark  Token = "modal.bg.dark"
	ModalBGLight Token = "modal.bg.light"

	WindowBGDark  Token = "window.bg.dark"  // hex
	WindowBGLight Token = "window.bg.light" // hex

	GradStart Token = "title.grad.start" // hex — gradient interpolation stops
	GradMid   Token = "title.grad.mid"
	GradEnd   Token = "title.grad.end"
)

// theme is the active palette. Default = the HUM-LEAD-directed B3 palette;
// theming (the 't' chooser) swaps this map wholesale.
var theme = map[Token]string{
	TextBase:   "250",
	TextBright: "97",

	TempHi:    "208",
	TempLo:    "51",
	TrendUp:   "137", // ~#A98D40
	TrendDown: "73",  // ~#409FA9

	FocusName:    "1;220",
	FocusCell:    "117", // light blue
	FocusPointer: "1;97",
	NameAdvisory: "186", // ~#D0CF89
	NameWarning:  "174", // ~#D08989

	ProviderOK:   "77",
	ProviderDown: "196",

	KeyChip:       "1;97;48;2;86;86;86",     // bg #565656 (UAT 18.4: contrast)
	KeyChipMuted:  "38;5;245;48;2;43;43;43", // half-tone text + bg #2b2b2b (UAT 21.1)
	ChipFlashUp:   "1;97;48;5;28",           // bold white on green
	ChipFlashDown: "1;97;48;5;124",          // bold white on red
	GroupText:     "1;97",

	GroupLocationBG: "48;2;97;97;97",
	GroupTodayBG:    "48;2;66;94;122",
	GroupTomorrowBG: "48;2;66;122;122",
	GroupExtendedBG: "48;2;94;94;122",
	GroupSectionBG:  "48;2;34;34;34", // #222 (UAT 44.2)

	AlertWarnFG: "38;5;196",
	AlertWarnBG: "49", // UAT 17.2: tile bg hidden for evaluation (was 48;2;20;0;0 ~5% red)
	AlertAdvFG:  "38;5;220",
	AlertAdvBG:  "49", // UAT 17.2: tile bg hidden for evaluation (was 48;2;32;28;0 ~10% yellow)
	AlertLabel:  "220",
	AlertDanger: "196",

	RadioFG:      "38;5;250",
	RadioBG:      "49", // UAT 18.5: hidden for evaluation; re-enable value: 48;2;48;48;48 (#303030)
	RadioAccent:  "77",
	StateStopped: "1;245",
	StatePlaying: "1;77",
	RadioStation: "1;226",
	RepeatOn:     "1;220",
	VizOn:        "1;77",
	SpectrumLow:  "77",  // green
	SpectrumMid:  "220", // yellow
	SpectrumHigh: "196", // red
	FireMark:     "208", // orange

	ConfirmBG: "48;2;79;12;12", // #4F0C0C — deep red under light text (UAT 109; was #AE7D7E, UAT 26.2)

	AlertModalWarnFG: "38;2;190;84;84",   // #BE5454 (UAT 28.4)
	AlertModalAdvFG:  "38;2;172;174;125", // #ACAE7D (UAT 28.3)

	AlertModalText:   "97",           // white (UAT 55)
	AlertModalWarnBG: "48;2;40;0;0",  // muted red tile (UAT 22)
	AlertModalAdvBG:  "48;2;32;28;0", // muted yellow tile (UAT 22)

	ModalFG:      "38;5;250",
	ModalBGDark:  "48;2;29;40;48", // #1D2830
	ModalBGLight: "48;2;59;81;99", // #3B5163

	WindowBGDark:  "#131313",
	WindowBGLight: "#ECECEC", // placeholder until theming

	GradStart: "#DD51D6", // the reference-CLI gradient
	GradMid:   "#378FE9",
	GradEnd:   "#7CE3B3",
}

// Tok resolves a semantic token to its SGR params (or hex for window/
// gradient tokens). Resolution happens at render time, so a theme swap
// takes effect on the next frame.
func Tok(t Token) string { return activeTable()[t] }

// hexRGB parses a theme hex entry (#RRGGBB) into components.
func hexRGB(t Token) (r, g, b int) {
	h := activeTable()[t]
	if len(h) != 7 || h[0] != '#' {
		return 0, 0, 0
	}
	v := func(s string) int {
		n := 0
		for _, c := range s {
			n <<= 4
			switch {
			case c >= '0' && c <= '9':
				n += int(c - '0')
			case c >= 'a' && c <= 'f':
				n += int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				n += int(c-'A') + 10
			}
		}
		return n
	}
	return v(h[1:3]), v(h[3:5]), v(h[5:7])
}
