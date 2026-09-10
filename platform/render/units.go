package render

// units.go — units and value formatting: Units, Opts, temperatures, distances, tides, wind, the health and trend glyphs. Split from render.go by the quality pass (Q2, pure move).

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// Units selects display units (D-19: global, live-swappable).
type Units int

// Unit values. UnitF is the v0.1 default per the mocks.
const (
	UnitF Units = iota
	UnitC
)

// The config words for the units, a closed set like the clock's. An unknown one
// reads as Imperial rather than failing a load.
const (
	unitsKeyImperial = "imperial"
	unitsKeyMetric   = "metric"
)

// UnitsByKey is the config word's units, UnitF for anything unrecognised.
func UnitsByKey(key string) Units {
	if key == unitsKeyMetric {
		return UnitC
	}
	return UnitF
}

// Key is the config word for u.
func (u Units) Key() string {
	if u == UnitC {
		return unitsKeyMetric
	}
	return unitsKeyImperial
}

// Label is how the Settings row names u — the system AND the units it means, so
// nobody has to remember which way round Imperial runs.
//
// The units are PADDED so the parentheticals line up under each other, the way
// the clock's do. Two rows is few enough to align by hand and the alignment is
// what makes the pair scannable: the eye reads down the bracketed column.
func (u Units) Label() string {
	if u == UnitC {
		return "Metric    (°C/Km)"
	}
	return "Imperial  (°F/Mi)"
}

// UnitsOrder is the two systems in the order the Settings group draws them.
func UnitsOrder() []Units { return []Units{UnitF, UnitC} }

// Opts carries the render context every primitive needs.
type Opts struct {
	Width int
	Units Units
	Clock Clock // how times of day are written (clock.go) — one owner, one setting
	ASCII bool
	Frame int // animation phase (loading dots; ticked by the program loop)
	// ThinBands collapses the group and section bands from three rows (a
	// band-coloured row above and below the label,
	// "so they breathe") back to one: the layout's last resort on a terminal
	// too short for the table's floor.
	ThinBands bool
}

// BandHeight is the height of a band under these options.
func (o Opts) BandHeight() int {
	if o.ThinBands {
		return 1
	}
	return 3
}

// Glyphs are the row-mark and legend marks in the active glyph set: the
// Unicode marks of the mocks, or their ASCII stand-ins under --ascii
// (A11-10: one owner, so the table and the Help legend cannot disagree).
type Glyphs struct {
	Pointer, Play, Repeat, Fire, Alert string
	// Pause is the row mark for a read a LISTENER has paused (MVS-D-74). It
	// replaces Play on that row rather than sitting beside it: the row is still
	// the one being read, and two marks would say two things are happening.
	Pause                        string
	Seismic                      [3]string // the felt-band ramp: [0] below feeling, [1] felt, [2] significant (0.11.0)
	OK, Fail, Note, Cursor, Fill string    // ✔ ✘ ♪ ▌ ░ and their ASCII forms (REVIEW R5-C-13: one owner for every mark)
	Dash, Dot                    string    // — and · as separators
	// Arrow is a rightward arrow in running TEXT — "( SHIFT + ENTER → STANDBY )".
	// Not a keycap: KeyCap draws a KEY, and this is the direction between two
	// states. One owner, so --ascii needs no special case at the call site.
	Arrow string
	// The CARD's own corners (0.16.0). Rounded, which is what the reference
	// mock draws for a card — the app's WINDOWS use the heavy box `BoxTitled`
	// owns, and a card is not a window. Through the glyph set so --ascii needs
	// no special case at the call site.
	CornerTL, CornerTR, CornerBL, CornerBR string
	// Idle and Live are a thing's own state where it is NAMED — the bed's
	// ACTIVE / INACTIVE chip (0.16.0, D-62). Not the seismic ramp, which an
	// early draft borrowed: that ramp means FELT INTENSITY and reusing it here
	// would give one glyph two meanings.
	Idle, Live string
	// 0.14.0: the Setup window's marks. Down is a picker's dropdown arrow;
	// Rail and RailCar draw the scroll rail; Ellipsis and Bullet are used where
	// text is cut or listed. All go through the glyph set so --ascii needs no
	// special case anywhere (Task 4.9's glyph parity).
	Up, Down, DropDown, Rail, RailCar, Ellipsis, Bullet, Stop string
	// Rule is a horizontal rule a WINDOW draws inside itself — a divider over
	// an alert, the fill between a modal's title and its stamp. It is not the
	// panel's border, which PanelColored owns and switches for itself.
	// Minus pairs with a plain "+" on the watchlist chips and is U+2212, not a
	// hyphen, so the two marks are the same width. Heart is the About line.
	Rule, Minus, Heart string
}

// Glyphs resolves the mark set for these options. Under --ascii the play
// mark is its own form (`*`), never the pointer's `>` (HUM LEAD ruling
// 2026-08-29, B-08b).
func (o Opts) Glyphs() Glyphs {
	if o.ASCII {
		return Glyphs{Pointer: ">", Play: "*", Pause: "=", Repeat: "R", Fire: "*", Alert: "!", Seismic: [3]string{".", "o", "O"},
			OK: "+", Fail: "x", Note: "~", Cursor: "_", Fill: ".", Dash: "-", Dot: "|",
			Up: "^", Down: "v", DropDown: "v", Rail: "|", RailCar: "#", Ellipsis: "...", Bullet: "*", Stop: "#", Rule: "-", Minus: "-", Heart: "<3", Arrow: "->", CornerTL: "+", CornerTR: "+", CornerBL: "+", CornerBR: "+", Idle: "o", Live: "*"}
	}
	return Glyphs{Pointer: "›", Play: "▶", Pause: "‖", Repeat: "∞", Fire: "◆", Alert: "⚠", Seismic: [3]string{"○", "●", "◉"},
		OK: "✔", Fail: "✘", Note: "♪", Cursor: "▌", Fill: "░", Dash: "—", Dot: "·",
		Up: "▲", Down: "▼", DropDown: "▾", Rail: "│", RailCar: "█", Ellipsis: "…", Bullet: "•", Stop: "■", Rule: "─", Minus: "−", Heart: "♥", Arrow: "→", CornerTL: "╭", CornerTR: "╮", CornerBL: "╰", CornerBR: "╯", Idle: "○", Live: "●"}
}

// asciiKey names an arrow key in words for a chip under --ascii — the one
// place every chip's key name crosses the glyph boundary (R5-C-13).
func asciiKey(key string) string {
	switch key {
	case "↑↓":
		return "up/down"
	case "←→":
		return "left/right"
	case "←":
		return "left"
	case "→":
		return "right"
	case "↑":
		return "up"
	case "↓":
		return "down"
	}
	return key
}

// SeismicLevel maps a magnitude to the felt-band glyph-ramp level — the single
// owner the table row mark and the detail section share (0.11.0): 1 = below
// feeling (○), 2 = felt (●), 3 = significant (◉). Only the strongest quake at a
// location wears the mark, so the caller passes that magnitude; 0 means none.
func SeismicLevel(mag float64) int {
	switch {
	case mag >= 5.0:
		return 3
	case mag >= 3.5:
		return 2
	default:
		return 1
	}
}

// LoadingDots is the loading shimmer (UAT 18.2b): a 4-phase dot sweep shown
// where data has not arrived yet — "n/a" is reserved for data that is truly
// absent after load. Upstream candidate (M6): a go-studs spinner style.
func (o Opts) LoadingDots() string {
	frames := [4]string{"...", "\u00b7..", ".\u00b7.", "..\u00b7"}
	if o.ASCII {
		frames = [4]string{"...", " ..", ". .", ".. "}
	}
	return frames[((o.Frame%4)+4)%4]
}

// Temp renders a Celsius value in the display units; nil renders n/a.
func (o Opts) Temp(c *float64) string {
	if c == nil {
		return "n/a"
	}
	if o.Units == UnitC {
		return fmt.Sprintf("%.0f°C", *c)
	}
	return fmt.Sprintf("%.0f°F", *c*9/5+32)
}

// Distance renders a kilometres value in the DIST column's fixed "nnn km"
// slot (miles under °F, following Height); blank when unknown.
func (o Opts) Distance(km *float64) string {
	if km == nil {
		return ""
	}
	if o.Units == UnitC {
		return fmt.Sprintf("%3.0f km", *km)
	}
	return fmt.Sprintf("%3.0f mi", *km*0.621371)
}

// TideHeight renders a metres value at tide precision (tenths of a foot
// under °F, centimetres under °C — UAT 61) in a fixed 4-cell numeric slot,
// so a negative low ("-0.1 ft") never shifts the column (UAT 62).
func (o Opts) TideHeight(m *float64) string {
	if m == nil {
		return "n/a"
	}
	if o.Units == UnitC {
		return fmt.Sprintf("%4.2f m", *m)
	}
	return fmt.Sprintf("%4.1f ft", *m*3.28084)
}

// Knots renders a m/s current speed in knots — the convention under both
// unit systems (UAT 61) — in the same fixed 4-cell slot as TideHeight.
func (o Opts) Knots(mps *float64) string {
	if mps == nil {
		return "n/a"
	}
	return fmt.Sprintf("%4.1f kt", *mps/0.514444)
}

// Wind renders a m/s value in the display units (mph under °F, km/h under °C).
func (o Opts) Wind(mps *float64) string {
	if mps == nil {
		return "n/a"
	}
	if o.Units == UnitC {
		return fmt.Sprintf("%.0f km/h", *mps*3.6)
	}
	return fmt.Sprintf("%.0f mph", *mps*2.23694)
}

// HealthGlyph renders one provider's header status (mock M-V1: ✔/⚠/✘ + name;
// glyph is textual, color additive elsewhere; ASCII fallback keeps the signal).
func (o Opts) HealthGlyph(name, status string) string {
	glyph := "✔"
	if status != snapshot.ProviderOK {
		glyph = "✘"
	}
	if o.ASCII {
		glyph = "OK"
		if status != snapshot.ProviderOK {
			glyph = "XX"
		}
	}
	fg := Tok(ProviderOK) // UAT 4.8: healthy provider reads green
	if status != snapshot.ProviderOK {
		fg = Tok(ProviderDown)
	}
	if status == snapshot.ProviderOff { // not a source right now (FIRMS without a key): neutral, never red (UAT 100)
		glyph, fg = "—", Tok(TextBase)
		if o.ASCII {
			glyph = "--"
		}
	}
	return rendering.WrapSGR(glyph+" "+name, fg)
}

// TrendGlyph renders a trend arrow in its muted tone (UAT 14.2) - the one
// owner for every view that shows ↗/↘ (tables, forecast modal, future).
func (o Opts) TrendGlyph(trend string) string {
	up, down := "↗", "↘"
	if o.ASCII {
		up, down = "^", "v"
	}
	switch trend {
	case "up":
		return Tint(up, Tok(TrendUp))
	case "down":
		return Tint(down, Tok(TrendDown))
	}
	return ""
}

// asciiMarks maps every unicode mark to its ASCII form, derived from the two
// sets rather than listed beside them: a mark added to Glyphs is covered the
// day it lands, which is the failure F-47 describes.
var asciiMarks = sync.OnceValue(func() *strings.Replacer {
	uni, asc := reflect.ValueOf(Opts{}.Glyphs()), reflect.ValueOf(Opts{ASCII: true}.Glyphs())
	var pairs []string
	for i := range uni.NumField() {
		switch u, a := uni.Field(i), asc.Field(i); u.Kind() {
		case reflect.String:
			pairs = append(pairs, u.String(), a.String())
		case reflect.Array:
			for j := range u.Len() {
				pairs = append(pairs, u.Index(j).String(), a.Index(j).String())
			}
		}
	}
	return strings.NewReplacer(pairs...)
})

// Marks rewrites text that was COMPOSED WITHOUT VIEW OPTIONS into this
// frame's mark set — a keymap's static help string, a Producer's ticker head.
// Those are the strings that cross the glyph boundary already built (F-47),
// and they are the only ones that need this: anything a renderer composes
// itself takes its marks from Glyphs directly, where the mark is chosen once
// with the layout around it rather than substituted afterwards.
func (o Opts) Marks(s string) string {
	if !o.ASCII {
		return s
	}
	return asciiMarks().Replace(s)
}
