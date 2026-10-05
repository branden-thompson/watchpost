package app

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/domains/fire/firms"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// AN EMPTY RECENT LIST IS NO PANIC (W14, C-6): with no seeds the recent
// pipeline has no assembler, and every reader of the pipelines' assemblers -
// the radio's fire, quakes and sea, the FIRMS status - reads past it.
func TestAnEmptyRecentListIsNoPanic(t *testing.T) {
	lp := &livePipelines{recent: &recentPipeline{}, firms: firms.New(nil, "", "", fire.Rules{})}
	ref := snapshot.LocationRef{Label: "Nowhere", Zip: "00000"}
	if got := lp.fireFor(ref); got.Known {
		t.Errorf("fire for a place nothing holds: %+v", got)
	}
	_ = lp.seismicFor(ref)
	_ = lp.marineFor(ref)
	lp.markFIRMS()
	if len(lp.assemblers()) != 0 {
		t.Error("an absent assembler was listed")
	}
}
