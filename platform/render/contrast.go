package render

import (
	"math"
	"strconv"
	"strings"
)

// contrast.go — WCAG contrast as a render-time fact, not only a test's: the
// alert tones are LIFTED to AA against their tints when a theme registers
// (HUM LEAD 2026-08-28, red-team round 4 B-05: the module title read 4.20:1
// by default and below AA in ten of twelve themes). A lift keeps the hue —
// the colour is mixed toward white (or toward black on a light tint) only as
// far as 4.5:1 needs — so a theme's intention survives while every pair
// reads.

// AAContrast is WCAG AA for normal text.
const AAContrast = 4.5

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

// fgRGB reads a foreground value: a bare 256 index or basic code, "1;n",
// "38;5;n" or "38;2;r;g;b" (bold ignored; anything else is not a foreground).
func fgRGB(v string) (r, g, b int, ok bool) {
	parts := strings.Split(strings.TrimPrefix(v, "1;"), ";")
	switch {
	case len(parts) == 1:
		return bareRGB(parts[0])
	case len(parts) == 3 && parts[0] == "38" && parts[1] == "5":
		n, err := strconv.Atoi(parts[2])
		if err != nil || n < 0 || n > 255 {
			return 0, 0, 0, false
		}
		r, g, b = xterm256RGB(n)
		return r, g, b, true
	case len(parts) == 5 && parts[0] == "38" && parts[1] == "2":
		r, _ = strconv.Atoi(parts[2])
		g, _ = strconv.Atoi(parts[3])
		b, _ = strconv.Atoi(parts[4])
		return r, g, b, true
	}
	return 0, 0, 0, false
}

// bareRGB reads a bare foreground code: a basic (30–37, 90–97) or a 256 index.
func bareRGB(code string) (r, g, b int, ok bool) {
	n, err := strconv.Atoi(code)
	if err != nil || n < 0 || n > 255 {
		return 0, 0, 0, false
	}
	switch {
	case n >= 90 && n <= 97: // bright basics
		n = n - 90 + 8
	case n >= 30 && n <= 37:
		n -= 30
	}
	r, g, b = xterm256RGB(n)
	return r, g, b, true
}

// fgLuminance is the luminance of a foreground value.
func fgLuminance(v string) (float64, bool) {
	r, g, b, ok := fgRGB(v)
	if !ok {
		return 0, false
	}
	return luminance(r, g, b), true
}

// bgLuminance reads a "48;2;r;g;b" or "48;5;n" background value.
func bgLuminance(v string) (float64, bool) {
	parts := strings.Split(v, ";")
	if len(parts) == 3 && parts[0] == "48" && parts[1] == "5" {
		n, err := strconv.Atoi(parts[2])
		if err != nil || n < 0 || n > 255 {
			return 0, false
		}
		return luminance(xterm256RGB(n)), true
	}
	r, g, b, ok := bgRGB(v)
	if !ok {
		return 0, false
	}
	return luminance(r, g, b), true
}

// contrastRatio is the WCAG ratio of two luminances.
func contrastRatio(a, b float64) float64 {
	hi, lo := max(a, b), min(a, b)
	return (hi + 0.05) / (lo + 0.05)
}

// Contrast reads the WCAG ratio of a foreground value on a background
// value; 0 when either is not a colour.
func Contrast(fg, bg string) float64 {
	f, ok1 := fgLuminance(fg)
	b, ok2 := bgLuminance(bg)
	if !ok1 || !ok2 {
		return 0
	}
	return contrastRatio(f, b)
}

// LiftToAA returns fg mixed toward white — toward black when the background
// is light — just far enough to read at AA on bg, as a truecolor value; the
// hue is kept, only its lightness moves. fg comes back unchanged when the
// pair already reads, when either is not a colour, or when even the limit
// cannot reach AA (the caller's palette is the problem then). A "1;" bold
// prefix is kept.
func LiftToAA(fg, bg string) string {
	bold := strings.HasPrefix(fg, "1;")
	r, g, b, ok := fgRGB(fg)
	bl, ok2 := bgLuminance(bg)
	if !ok || !ok2 || contrastRatio(luminance(r, g, b), bl) >= AAContrast {
		return fg
	}
	tr, tg, tb := 255, 255, 255
	if bl > 0.18 { // a light tint: lift toward black instead
		tr, tg, tb = 0, 0, 0
	}
	for step := 1; step <= 50; step++ { // bounded: 2 % of the way per step (P10-02)
		t := float64(step) / 50
		mr, mg, mb := int(float64(r)*(1-t)+float64(tr)*t+0.5), int(float64(g)*(1-t)+float64(tg)*t+0.5), int(float64(b)*(1-t)+float64(tb)*t+0.5)
		if contrastRatio(luminance(mr, mg, mb), bl) >= AAContrast {
			out := "38;2;" + strconv.Itoa(mr) + ";" + strconv.Itoa(mg) + ";" + strconv.Itoa(mb)
			if bold {
				return "1;" + out
			}
			return out
		}
	}
	return fg
}

// withAA lifts every painted fg × bg pair a theme must read at AA when it
// registers, so a theme's own values are the hue and the lift is the floor
// — toward white on a dark ground, toward black on a light one (the
// Watchpost Light theme rides the same pass). The pairs live in ONE place:
// aaPairs (HUM LEAD rulings 2026-08-28/29: B-05, then R5-C-04 widened).
func withAA(t map[Token]string) map[Token]string {
	for _, p := range aaPairs() {
		bgs := make([]string, 0, len(p.on))
		for _, bg := range p.on {
			bgs = append(bgs, bgOf(t, bg))
		}
		if lifted, ok := liftAll(t[p.fg], bgs); ok {
			t[p.fg] = lifted
			continue
		}
		// No one text tone reads on every ground it shares (bold white on the
		// gold ticker lane, black on the deep blue): the ground gives instead —
		// each failing background deepens toward black until the text reads.
		for _, bg := range p.on {
			if !strings.HasPrefix(t[bg], "48;") || isCategoryTint(bg) {
				continue // a #hex ground (the window) is the theme's own; the category tints are the HUM LEAD's, theme-independent
			}
			t[bg] = deepenToAA(t[bg], t[p.fg])
		}
	}
	return t
}

// liftAll moves fg toward white, else toward black, until it reads at AA on
// EVERY bg — the smaller move wins; false when neither side can (the caller
// then moves the grounds). Bounded: two directions × 50 steps (P10-02).
func liftAll(fg string, bgs []string) (string, bool) {
	if passesAll(fg, bgs) {
		return fg, true
	}
	r, g, b, ok := fgRGB(fg)
	if !ok {
		return fg, false
	}
	bold := strings.HasPrefix(fg, "1;")
	best, bestSteps := "", 0
	for _, target := range [][3]int{{255, 255, 255}, {0, 0, 0}} {
		for step := 1; step <= 50; step++ {
			cand := mixToward(r, g, b, target, float64(step)/50, bold)
			if passesAll(cand, bgs) {
				if best == "" || step < bestSteps {
					best, bestSteps = cand, step
				}
				break
			}
		}
	}
	return best, best != ""
}

// passesAll reports whether fg reads at AA on every bg.
func passesAll(fg string, bgs []string) bool {
	for _, bg := range bgs {
		if Contrast(fg, bg) < AAContrast {
			return false
		}
	}
	return true
}

// mixToward is fg mixed t of the way toward target, as a truecolor value.
func mixToward(r, g, b int, target [3]int, t float64, bold bool) string {
	m := func(x, y int) int { return int(float64(x)*(1-t) + float64(y)*t + 0.5) }
	out := "38;2;" + strconv.Itoa(m(r, target[0])) + ";" + strconv.Itoa(m(g, target[1])) + ";" + strconv.Itoa(m(b, target[2]))
	if bold {
		return "1;" + out
	}
	return out
}

// deepenToAA moves a "48;…" ground away from the text — toward black under
// light text, toward white under dark text — until fg reads on it; unchanged
// when it already does or is not truecolor. Bounded (P10-02).
func deepenToAA(bg, fg string) string {
	if _, _, _, ok := bgRGB(bg); !ok {
		return bg
	}
	target := "48;2;0;0;0"
	if fl, ok := fgLuminance(fg); ok && fl < 0.18 {
		target = "48;2;255;255;255" // dark text: the ground lightens
	}
	for step := 0; step < 50 && Contrast(fg, bg) < AAContrast; step++ {
		bg = mixBG(target, bg, 0.04)
	}
	return bg
}

// aaPair is a foreground token and every background token it is painted on.
type aaPair struct {
	fg Token
	on []Token
}

// aaPairs is the register of painted pairs — what the AA gate checks and
// what registration lifts. A function, not a global (P10-06).
func aaPairs() []aaPair {
	win := []Token{WindowBGDark, WindowBGLight}
	bands := []Token{GroupLocationBG, GroupTodayBG, GroupTomorrowBG, GroupExtendedBG, GroupSectionBG}
	lanes := []Token{TickerRedBG, TickerOrangeBG, TickerYellowBG, TickerBlueBG}
	tints := []Token{EventCatRedBG, EventCatOrangeBG, EventCatYellowBG, EventCatWatchBG, EventCatStmtBG, EventCatBlueBG}
	modal := []Token{ModalBGDark, ModalBGLight}
	pairs := []aaPair{
		{GroupText, bands}, {TickerFG, lanes}, {TickerMutedFG, append(append([]Token{}, lanes...), GroupSectionBG)},
		{StatePlaying, append(append([]Token{}, tints...), win...)}, {FocusPointer, append(append([]Token{}, tints...), win...)}, {ModalTitle, tints},
		{ModalFG, modal}, {ModalTitle, modal}, {AlertDanger, append([]Token{ModalBGDark}, win...)}, {AlertModalText, append(append([]Token{}, tints...), AlertModalWarnBG, AlertModalAdvBG)},
		{AlertModalWarnFG, []Token{AlertModalWarnBG}}, {AlertModalAdvFG, []Token{AlertModalAdvBG}},
	}
	for _, fg := range []Token{TextBase, TextBright, TableMuted, TableName, ProviderOK, ProviderDown, NameAdvisory, NameWarning,
		StateStopped, RadioFG, RadioStation, RadioAccent, RepeatOn, VizOn, AlertLabel, TrendUp, TrendDown, TempHi, TempLo,
		FireMark, SeismicMark, FocusName, FocusCell} {
		pairs = append(pairs, aaPair{fg, win})
	}
	return pairs
}

// bgOf reads a background token as SGR: a "48;…" value as it is, a #RRGGBB
// window value as truecolor.
func bgOf(t map[Token]string, bg Token) string {
	v := t[bg]
	if strings.HasPrefix(v, "#") && len(v) == 7 {
		r, _ := strconv.ParseInt(v[1:3], 16, 0)
		g, _ := strconv.ParseInt(v[3:5], 16, 0)
		b, _ := strconv.ParseInt(v[5:7], 16, 0)
		return "48;2;" + strconv.Itoa(int(r)) + ";" + strconv.Itoa(int(g)) + ";" + strconv.Itoa(int(b))
	}
	return v
}

// isCategoryTint reports one of the severe window's fixed category tints.
func isCategoryTint(bg Token) bool {
	switch bg {
	case EventCatRedBG, EventCatOrangeBG, EventCatYellowBG, EventCatWatchBG, EventCatStmtBG, EventCatBlueBG:
		return true
	}
	return false
}
