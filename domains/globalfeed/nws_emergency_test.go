package globalfeed

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/category"
)

// C-2 — READ RANK 1 MUST BE OCCUPIABLE.
//
// platform/category gives Emergency ReadRank 1 and platform/lineup asserts the
// read order LEADS with it, exempts it from the burst Max and lets it overrun
// the budget. None of that could ever fire: the national query did not ask for
// a single civil-emergency product, so no arrival could carry the category.
//
// THE REGISTRY ITSELF SAYS THIS IS A DEFECT AND NOT A SCOPE DECISION. Spec has
// a Watchlist flag — "a category the national feed cannot produce, whose rows
// arrive only through tracked locations". Advisories and Statements carry it
// and are correctly absent here. Emergency does not, so the registry asserts
// the national feed can produce it.
func TestTheNationalQueryAsksForTheEmergencyOrders(t *testing.T) {
	if _, ok := CivilEmergencyCategory("Evacuation Immediate"); !ok {
		t.Fatal("the civil-emergency family does not know the Evacuation Immediate")
	}
	got := (&NWS{jsonFeed{base: "https://example.test/alerts/active"}}).url()
	if !strings.Contains(got, "Evacuation%20Immediate") {
		t.Errorf("the national query never asks for an evacuation order, so read rank 1 can never be occupied:\n%s", got)
	}
}

// The lane an evacuation order lands in, asked the way the producer asks it.
// Without this it is laned Warnings by LaneOf's default — the rung BELOW the
// one the ladder reserves for it, and indistinguishable on the band from a
// thunderstorm warning.
func TestLaneOfPutsAnEvacuationOrderInTheEmergencyLane(t *testing.T) {
	e := Event{ID: "evac", Class: ClassSevereWx, Type: "Evacuation Immediate"}
	if got := LaneOf(e); got != category.Emergency {
		t.Errorf("an evacuation order lanes as %s; it is the one Emergency Orders product (MVS-D-61)", category.Of(got).Bucket)
	}
	// THE CONTROL. Without it this passes against a LaneOf that answers
	// Emergency for everything.
	if got := LaneOf(Event{ID: "t", Class: ClassSevereWx, Type: "Tornado Warning"}); got != category.Warnings {
		t.Errorf("a tornado warning lanes as %s, want Warnings", category.Of(got).Bucket)
	}
}
