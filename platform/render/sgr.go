package render

// sgr.go — colour: raw SGR wrapping, tints, key caps, alert and radio tones, the window and modal palette. Split from render.go by the quality pass (Q2, pure move).

import (
	"fmt"
	"image/color"
	"strings"

	lipgloss "charm.land/lipgloss/v2"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// Color pass (HUM LEAD, UAT session 3): HI temps orange, LO temps cyan;
// chips bold-white on light grey; group headers carry 50%-muted backgrounds
// (terminal cells have no alpha - nearest 256-palette pastels). All styling
// flows through go-studs rendering.WrapSGR, which no-ops when color is off
// (NO_COLOR / pipes / tests), so text signals never depend on color (R-12a).

// sgrRaw wraps text in a raw SGR sequence, gated by go-studs' color switch.
// The sanctioned constructor (SGR/ColorSequence) classifies bare numeric
// codes as 256-palette FOREGROUNDS only — background chips are outside its
// contract (upstream candidate M6: background support in ColorSequence).
// Until that lands, the seam emits the one escape shape it needs.
func sgrRaw(text, params string) string {
	if !colorOn() {
		return text
	}
	return "\x1b[" + params + "m" + text + "\x1b[0m"
}

// colorOn probes the go-studs color gate (single source of truth: NO_COLOR +
// tty detection, test-overridable via SetColorEnabledForTest).
func colorOn() bool { return rendering.ColorsEnabled() }

// ColorOn is the gate for callers outside the package that compose raw
// SGR themselves (the tty's frame finisher, Q3).
func ColorOn() bool { return colorOn() }

// TitleGradient renders the app title bold with the reference-CLI interpolated
// truecolor gradient (#DD51D6 -> #378FE9 -> #7CE3B3) — UAT 4.9, standing in
// until theming lands. Plain text when color is off.
func TitleGradient(text string) string {
	if !colorOn() {
		return text
	}
	runes := []rune(text)
	if len(runes) < 2 {
		return text
	}
	r1, g1, b1 := hexRGB(GradStart)
	r2, g2, b2 := hexRGB(GradMid)
	r3, g3, b3 := hexRGB(GradEnd)
	lerp := func(a, b int, t float64) float64 { return float64(a)*(1-t) + float64(b)*t }
	var b strings.Builder
	for i, ch := range runes {
		pos := float64(i) / float64(len(runes)-1)
		var r, g, bl float64
		if pos <= 0.5 {
			t := pos * 2
			r, g, bl = lerp(r1, r2, t), lerp(g1, g2, t), lerp(b1, b2, t)
		} else {
			t := (pos - 0.5) * 2
			r, g, bl = lerp(r2, r3, t), lerp(g2, g3, t), lerp(b2, b3, t)
		}
		fmt.Fprintf(&b, "\x1b[1;38;2;%d;%d;%dm%c", int(r), int(g), int(bl), ch)
	}
	b.WriteString("\x1b[0m")
	return b.String()
}

// TintDefault repaints all un-tinted text in the base grey (UAT 4.10): the
// frame opens with the base foreground and every SGR reset re-arms it, so
// explicitly-colored spans keep their colors and everything else drops to
// grey 250 (~9:1 on black — comfortably AA). No-op with color off.
func TintDefault(s string) string {
	if !colorOn() {
		return s
	}
	base := "\x1b[0;38;5;" + Tok(TextBase) + "m"
	return base + strings.ReplaceAll(s, "\x1b[0m", base) + "\x1b[0m"
}

// Exported tint codes for module text (views compose, the seam styles).

// TintRaw wraps text in a full SGR parameter list (truecolor 38;2/48;2 and
// bold composites - the shapes go-studs WrapSGR cannot express), gated by
// the same color switch. Theme tokens with truecolor values ride this.
func TintRaw(text, params string) string { return sgrRaw(text, params) }

// Tint wraps text in a fg code (bare 256 or basic SGR; "1;"-prefixed for
// bold) through the go-studs gate - plain text when color is off.
func Tint(text, code string) string {
	if !colorOn() {
		return text
	}
	return rendering.WrapSGR(text, code)
}

// KeyCapIf renders a key chip in its enabled or muted state (UAT 21.1):
// a control that cannot act in the current model state reads at ~50%
// opacity. THE chip entry point for stateful controls - views pass their
// model-derived enabled flags (ELM: state in, view out) and every future
// control inherits the behavior.
func (o Opts) KeyCapIf(key string, enabled bool) string {
	if enabled {
		return o.KeyCap(key)
	}
	if o.ASCII || !colorOn() {
		return "[" + key + "]" // color-off keeps the textual affordance (RS-14)
	}
	return sgrRaw(" "+key+" ", Tok(KeyChipMuted))
}

// KeyCapWith renders a chip in an explicit tone (composite SGR token) —
// feedback states such as the volume blink (UAT 41). Color-off keeps [key].
func (o Opts) KeyCapWith(key string, tone Token) string {
	if o.ASCII || !colorOn() {
		return "[" + key + "]"
	}
	return sgrRaw(" "+key+" ", Tok(tone))
}

// KeyCap renders a key binding as a CLIAmp-style button: grey background,
// bold white text (UAT session 2B; upstream candidate M6: tui.KeyCap token).
// With color off (NO_COLOR, pipes, tests) it degrades to the mock's [key]
// form so the affordance never disappears (RS-14).
func (o Opts) KeyCap(key string) string {
	if o.ASCII {
		return "[" + key + "]"
	}
	if !colorOn() {
		return "[" + key + "]"
	}
	return sgrRaw(" "+key+" ", Tok(KeyChip))
}

// AlertTone maps an alert to its display class (UAT 4.6): warning-grade
// statements read red, watch/advisory-grade read yellow. Returned as the
// bare 256 fg code PanelColored expects.
func AlertTone(event, severity string) string {
	if AlertIsWarning(event, severity) {
		return Tok(AlertDanger)
	}
	return Tok(AlertLabel)
}

// AlertIsWarning is THE warning-vs-advisory classifier (single owner:
// module tones, row-name tints, and future views all ride it).
func AlertIsWarning(event, severity string) bool {
	sev := strings.ToLower(severity)
	return sev == "severe" || sev == "extreme" || strings.Contains(event, "Warning")
}

// Block tones (UAT 5.4): borderless modules carry a ~10%-opacity background
// (truecolor - the 256 palette has no tints that dark) with a matching text
// tone. Dark-terminal variants; the light-bg variants land with theming.

// AlertBlockTone returns the fg/bg pair for an alert module (UAT 5.4a/b).
func AlertBlockTone(event, severity string) (fg, bg string) {
	if AlertIsWarning(event, severity) {
		return Tok(AlertWarnFG), Tok(AlertWarnBG)
	}
	return Tok(AlertAdvFG), Tok(AlertAdvBG)
}

// RadioBlockTone returns the fg/bg pair for the radio module (UAT 5.4c).
func RadioBlockTone() (fg, bg string) { return Tok(RadioFG), Tok(RadioBG) }

// BGVisible reports whether a tone paints a real background ("49" = the
// terminal default = hidden; theming toggles module chrome by value alone).
func BGVisible(bg string) bool { return bg != "" && bg != "49" }

// Window + floating-tile backgrounds (UAT 12.4): the base viewport sits on
// a near-black; the blue-grey pair belongs to floating modal tiles only.

// WindowBG returns the viewport background for the terminal's mode (the
// dashboard passes bubbletea's BackgroundColorMsg.IsDark verdict).
func WindowBG(dark bool) color.Color {
	if dark {
		return lipgloss.Color(Tok(WindowBGDark))
	}
	return lipgloss.Color(Tok(WindowBGLight))
}

// ModalTone returns the fg/bg pair for floating modal tiles (UAT 12.4:
// the blue-grey tile treatment; text stays the base grey).
func ModalTone(dark bool) (fg, bg string) {
	if dark {
		return Tok(ModalFG), Tok(ModalBGDark)
	}
	return Tok(ModalFG), Tok(ModalBGLight)
}
