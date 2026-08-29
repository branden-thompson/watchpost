package render

// units.go — units and value formatting: Units, Opts, temperatures, distances, tides, wind, the health and trend glyphs. Split from render.go by the quality pass (Q2, pure move).

import (
	"fmt"

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

// Opts carries the render context every primitive needs.
type Opts struct {
	Width int
	Units Units
	ASCII bool
	Frame int // animation phase (loading dots; ticked by the program loop)
	// ThinBands collapses the group and section bands from three rows (a
	// band-coloured row above and below the label — HUM LEAD UAT 2026-08-27,
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
	Seismic                            [3]string // the felt-band ramp: [0] below feeling, [1] felt, [2] significant (0.11.0)
	OK, Fail, Note, Cursor, Fill       string    // ✔ ✘ ♪ ▌ ░ and their ASCII forms (REVIEW R5-C-13: one owner for every mark)
	Dash, Dot                          string    // — and · as separators
}

// Glyphs resolves the mark set for these options. Under --ascii the play
// mark is its own form (`*`), never the pointer's `>` (HUM LEAD ruling
// 2026-08-29, B-08b).
func (o Opts) Glyphs() Glyphs {
	if o.ASCII {
		return Glyphs{Pointer: ">", Play: "*", Repeat: "R", Fire: "*", Alert: "!", Seismic: [3]string{".", "o", "O"},
			OK: "+", Fail: "x", Note: "~", Cursor: "_", Fill: ".", Dash: "-", Dot: "|"}
	}
	return Glyphs{Pointer: "›", Play: "▶", Repeat: "∞", Fire: "◆", Alert: "⚠", Seismic: [3]string{"○", "●", "◉"},
		OK: "✔", Fail: "✘", Note: "♪", Cursor: "▌", Fill: "░", Dash: "—", Dot: "·"}
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
		return fmt.Sprintf("%.0fºC", *c)
	}
	return fmt.Sprintf("%.0fºF", *c*9/5+32)
}

// Distance renders a kilometres value in the DIST column's fixed "nnn km"
// slot (miles under ºF, following Height); blank when unknown.
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
// under ºF, centimetres under ºC — UAT 61) in a fixed 4-cell numeric slot,
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

// Wind renders a m/s value in the display units (mph under ºF, km/h under ºC).
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
