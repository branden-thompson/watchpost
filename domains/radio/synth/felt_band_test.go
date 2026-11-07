package synth

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TestTheSpokenLikelihoodAtEveryBoundary is 0.19.5 Q5.1's pin, written before
// the felt bands get one owner: the felt likelihood the radio says either side
// of each boundary (DR-1).
func TestTheSpokenLikelihoodAtEveryBoundary(t *testing.T) {
	for _, c := range []struct {
		mag  float64
		want string
	}{{0, "low"}, {3.49, "low"}, {3.5, "moderate"}, {4.49, "moderate"}, {4.5, "strong"}, {4.99, "strong"}, {5.0, "strong"}, {7.8, "strong"}} {
		if got := feltLikelihood(c.mag); got != c.want {
			t.Errorf("a magnitude %v quake's likelihood is said %q; want %q", c.mag, got, c.want)
		}
	}
}

// TestEveryFeltBandHasSpokenWords is JM-15 for the radio's likelihoods.
func TestEveryFeltBandHasSpokenWords(t *testing.T) {
	if len(feltLikelihoods) != render.FeltBands {
		t.Errorf("feltLikelihoods has %d entries for %d felt bands", len(feltLikelihoods), render.FeltBands)
	}
}

// TestADepthIsSaidInWholeKilometres is 0.19.5 Q5.5's pin (DR-2), written
// before depthWords calls distanceWords: a quake's depth is said in whole
// kilometres whatever the listener's units, and one kilometre is singular.
func TestADepthIsSaidInWholeKilometres(t *testing.T) {
	for km, want := range map[float64]string{0.6: "1 kilometer", 1.4: "1 kilometer", 1.5: "2 kilometers", 9.4: "9 kilometers", 0: "0 kilometers"} {
		if got := depthWords(km); got != want {
			t.Errorf("depthWords(%v) = %q; want %q", km, got, want)
		}
	}
}
