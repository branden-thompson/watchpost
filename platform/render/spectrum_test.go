package render

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestSpectrumRowsFillTheWidthWithFractionalBlocks(t *testing.T) {
	// UAT 92 (CLIAmp Bars): ten bands separated by one cell, each band as wide
	// as the room allows; every row is exactly width cells; a full band is █
	// on every row, an empty band is blank, a partial band shows a fractional
	// block on the row it ends in.
	bands := []float64{1, 0, 0.5, 0, 0, 0, 0, 0, 0, 1}
	rows := Spectrum(bands, 39, 3) // 10 bands × 3 cells + 9 gaps
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	for i, r := range rows {
		if Width(r) != 39 {
			t.Fatalf("row %d is %d cells, want 39: %q", i, Width(r), r)
		}
	}
	if !strings.HasPrefix(rows[0], "███ ") || !strings.HasSuffix(rows[0], " ███") {
		t.Fatalf("full bands are solid on the top row: %q", rows[0])
	}
	if !strings.HasPrefix(rows[2], "███ ") || !strings.HasSuffix(rows[2], " ███") {
		t.Fatalf("full bands are solid on the bottom row: %q", rows[2])
	}
	// Band 2 at 0.5: bottom row (0–⅓) solid, middle row (⅓–⅔) half a block, top row blank.
	cells := func(r string) []rune { return []rune(r) }
	if string(cells(rows[2])[8:11]) != "███" {
		t.Fatalf("half band is solid on the bottom row: %q", rows[2])
	}
	if mid := string(cells(rows[1])[8:11]); mid != "▄▄▄" {
		t.Fatalf("half band shows the fractional block on its middle row, got %q", mid)
	}
	if top := string(cells(rows[0])[8:11]); top != "   " {
		t.Fatalf("half band is blank on the top row, got %q", top)
	}
	if string(cells(rows[2])[4:7]) != "   " {
		t.Fatalf("an empty band stays blank: %q", rows[2])
	}
}

func TestSpectrumWidthSharing(t *testing.T) {
	// Uneven room is shared left-first; too narrow for gaps drops them; a
	// single row still spans the width; nil bands render blank rows.
	if r := Spectrum(make([]float64, 10), 25, 1); len(r) != 1 || Width(r[0]) != 25 {
		t.Fatalf("25 cells, 1 row: %q", r)
	}
	if r := Spectrum(nil, 12, 2); len(r) != 2 || strings.TrimSpace(r[0]) != "" || Width(r[1]) != 12 {
		t.Fatalf("nil bands are blank rows of the width: %q", r)
	}
	if r := Spectrum([]float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, 5, 1); r[0] != "█████" {
		t.Fatalf("narrower than the bands: no gaps, one cell each while room lasts: %q", r[0])
	}
	if r := Spectrum([]float64{1}, 0, 1); len(r) != 1 || r[0] != "" {
		t.Fatalf("zero width is an empty row: %q", r)
	}
	if r := Spectrum([]float64{1}, 4, 0); r != nil {
		t.Fatalf("zero rows is nothing: %q", r)
	}
}

// sgr256 is the SGR parameter list go-studs emits for a bare 256-color code.
func sgr256(code string) string { return "38;5;" + code }

func TestSpectrumGradientByHeightAndByLevel(t *testing.T) {
	// CLIAmp's spectrum gradient: rows are tinted by their height — the
	// bottom third green, the middle yellow, the top red. A one-row
	// visualizer has no height to speak of, so each band takes the tier of
	// its own level instead.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	full := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	rows := Spectrum(full, 19, 3)
	for i, tok := range []Token{SpectrumHigh, SpectrumMid, SpectrumLow} {
		if !strings.HasPrefix(rows[i], "\x1b["+sgr256(Tok(tok))+"m") {
			t.Fatalf("row %d carries the %s tint: %q", i, tok, rows[i])
		}
	}
	one := Spectrum([]float64{0.2, 0.5, 0.9, 0, 0, 0, 0, 0, 0, 0}, 19, 1)[0]
	for _, want := range []string{
		"\x1b[" + sgr256(Tok(SpectrumLow)) + "m▁",
		"\x1b[" + sgr256(Tok(SpectrumMid)) + "m▄",
		"\x1b[" + sgr256(Tok(SpectrumHigh)) + "m▇",
	} {
		if !strings.Contains(one, want) {
			t.Fatalf("one row tints each band by its level; missing %q in %q", want, one)
		}
	}
	if Width(one) != 19 {
		t.Fatalf("tinted row keeps its width: %d", Width(one))
	}
	if strings.Contains(one, "m \x1b[0m") {
		t.Fatalf("blank bands are not tinted (no SGR noise): %q", one)
	}
}
