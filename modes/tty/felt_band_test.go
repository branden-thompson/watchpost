package tty

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TestTheSeismicLabelAtEveryBoundary is 0.19.5 Q5.1's pin, written before the
// felt bands get one owner: the Details section's felt label either side of
// each boundary (DR-1).
func TestTheSeismicLabelAtEveryBoundary(t *testing.T) {
	o := dash(t).(Dashboard).opts()
	for _, c := range []struct {
		mag  float64
		want string
	}{{0, "Below feeling"}, {3.49, "Below feeling"}, {3.5, "Might feel it"}, {4.49, "Might feel it"},
		{4.5, "Almost certainly felt"}, {4.99, "Almost certainly felt"}, {5.0, "Significant"}, {7.8, "Significant"}} {
		if _, got := seismicBand(o, c.mag); got != c.want {
			t.Errorf("a magnitude %v quake is labelled %q; want %q", c.mag, got, c.want)
		}
	}
}

// TestEveryFeltBandHasALabel is JM-15 for the Details section's labels.
func TestEveryFeltBandHasALabel(t *testing.T) {
	if len(feltLabels) != render.FeltBands {
		t.Errorf("feltLabels has %d entries for %d felt bands", len(feltLabels), render.FeltBands)
	}
}
