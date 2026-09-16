package app

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestTheLookupAnswersFromThePoolAlone.
//
// HUM LEAD, 2026-09-14: "Whatever helps performance - the end result is
// transparent to the end user - either what they type is a valid location within
// the service radius or not."
//
// SO THE POOL IS THE WHOLE ANSWER, with no fall-through to the RESOLVER when the
// pool has no match. Telling "outside the radius" from "nowhere at all" costs one
// network call PER KEYSTROKE, on a field the operator types into, and buys two
// helper sentences both pointing at Observer.
//
// AND THAT MAKES THIS CHECK THE ONLY THING between a typo and a scheduled card,
// which is why a lookup that admits anything is a mutant worth having.
func TestTheLookupAnswersFromThePoolAlone(t *testing.T) {
	vista := snapshot.LocationRef{Label: "Vista, CA", Zip: "92084"}
	lp := &livePipelines{poolRefs: []snapshot.LocationRef{vista}}

	for _, tc := range []struct {
		query         string
		inPool, found bool
	}{
		{"vista", true, true},
		{"Vista, CA", true, true},
		{"92084", true, true},
		{"denver", false, false},
		{"", false, false},
	} {
		ref, inPool, found := lp.lookInPool(tc.query)
		if inPool != tc.inPool || found != tc.found {
			t.Errorf("%q: inPool=%v found=%v, want %v/%v", tc.query, inPool, found, tc.inPool, tc.found)
		}
		if found && ref.Label != vista.Label {
			t.Errorf("%q resolved to %q, want %q", tc.query, ref.Label, vista.Label)
		}
	}
}
