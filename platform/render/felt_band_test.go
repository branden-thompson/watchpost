package render

import "testing"

// TestTheSeismicLevelAtEveryBoundary is 0.19.5 Q5.1's pin, written before the
// felt bands get one owner: the glyph ramp's level either side of each felt
// boundary (DR-1).
func TestTheSeismicLevelAtEveryBoundary(t *testing.T) {
	for _, c := range []struct {
		mag  float64
		want int
	}{{0, 1}, {3.49, 1}, {3.5, 2}, {4.49, 2}, {4.5, 2}, {4.99, 2}, {5.0, 3}, {7.8, 3}} {
		if got := SeismicLevel(c.mag); got != c.want {
			t.Errorf("SeismicLevel(%v) = %d; want %d", c.mag, got, c.want)
		}
	}
}

// TestEveryFeltBandHasAGlyphLevel is 0.19.5 JM-15: a table indexed by the felt band
// has one entry per band, so a band added to FeltBand cannot index past it or
// read an empty entry.
func TestEveryFeltBandHasAGlyphLevel(t *testing.T) {
	if len(seismicLevels) != FeltBands {
		t.Errorf("seismicLevels has %d entries for %d felt bands", len(seismicLevels), FeltBands)
	}
	for b := range FeltBands { // bounded (P10-02)
		if seismicLevels[b] < 1 || seismicLevels[b] > 3 {
			t.Errorf("felt band %d has glyph level %d; the ramp is 1 to 3", b, seismicLevels[b])
		}
	}
}
