package render

import "strings"

// Spectrum renders band levels (0..1) as CLIAmp's default "Bars"
// visualizer (UAT 92): every band gets an equal share of width with one
// blank cell between bands (dropped when there is no room), heights are
// drawn with the fractional block glyphs, and the rows carry the spectrum
// gradient — bottom third green, middle yellow, top red. With a single row
// there is no height to grade, so each band takes the tier of its own
// level. Each row is exactly width cells; nil bands are blank rows.
func Spectrum(bands []float64, width, rows int) []string {
	if rows <= 0 {
		return nil
	}
	out := make([]string, rows)
	if width <= 0 {
		return out
	}
	if len(bands) == 0 {
		bands = make([]float64, 10)
	}
	cols, gaps := bandColumns(len(bands), width)
	for row := 0; row < rows; row++ { // bounded by rows
		bottom := float64(rows-1-row) / float64(rows)
		top := float64(rows-row) / float64(rows)
		var b strings.Builder
		for i, level := range bands {
			if cols[i] == 0 {
				continue
			}
			if i > 0 && i <= gaps {
				b.WriteByte(' ')
			}
			cell := strings.Repeat(fracBlock(level, bottom, top), cols[i])
			if rows == 1 && strings.TrimSpace(cell) != "" {
				cell = Tint(cell, Tok(spectrumTier(level)))
			}
			b.WriteString(cell)
		}
		line := PadTo(b.String(), width)
		if rows > 1 {
			line = Tint(line, Tok(spectrumTier(bottom)))
		}
		out[row] = line
	}
	return out
}

// bandColumns shares width among n bands with one-cell gaps: the leading
// bands take any remainder; when the width cannot seat every band, only the
// first width bands get a cell and the gaps go. Returns the cells per band
// and how many gaps (after bands 1..gaps) the width affords.
func bandColumns(n, width int) ([]int, int) {
	cols := make([]int, n)
	visible := min(n, width)
	gaps := min(visible-1, max(0, width-visible))
	cells := width - gaps
	base, extra := cells/visible, cells%visible
	for i := 0; i < visible; i++ { // bounded by visible ≤ n
		cols[i] = base
		if i < extra {
			cols[i]++
		}
	}
	return cols, gaps
}

// fracBlock is the glyph for a band of level on the row spanning
// bottom..top: solid when the level clears the row, blank when it does not
// reach it, a partial block in between.
func fracBlock(level, bottom, top float64) string {
	const blocks = " ▁▂▃▄▅▆▇█"
	glyphs := []rune(blocks)
	if level >= top {
		return "█"
	}
	if level <= bottom {
		return " "
	}
	idx := int((level - bottom) / (top - bottom) * float64(len(glyphs)-1))
	return string(glyphs[max(0, min(idx, len(glyphs)-1))])
}

// spectrumTier picks the gradient token for a height or level in 0..1.
func spectrumTier(v float64) Token {
	switch {
	case v >= 0.6:
		return SpectrumHigh
	case v >= 0.3:
		return SpectrumMid
	}
	return SpectrumLow
}
