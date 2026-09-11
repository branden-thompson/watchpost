package locations

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// bonsall is the HUM LEAD's own station, and the epicentre this was designed
// against (D-72).
var bonsall = snapshot.LocationRef{Label: "Bonsall, CA", Zip: "92003", Lat: 33.2881, Lon: -117.2256, TZ: "America/Los_Angeles"}

// THE POOL IS THE BROADCASTER'S OWN LIST, AND IT IS NOT THE WATCHLIST (D-72).
//
//	"the Observer watchlist is its lineup, and a different rolling window /
//	 stack / list needs to serve as Broadcaster Location Pool for producers to
//	 create the lineup"
//
// THREE TIERS, in this order: the transmitter's own place, the region's cities
// nearest-first, then the hyper-local zip places that fill in behind them —
// "the big value here is the hyper local station reports."
func TestThePoolIsThreeTiersInsideTheFence(t *testing.T) {
	idx := indexForTest(t)
	got := Pool(idx, bonsall, 20, 25)
	if len(got) < 10 {
		t.Fatalf("a 20-mile fence around Bonsall fills a ten-slot line-up; got %d", len(got))
	}
	// TIER ONE IS HOME. A station reads where it transmits from first; it is
	// the most local report it has.
	if got[0].Label != bonsall.Label {
		t.Errorf("the transmitter's own place leads the pool; got %q", got[0].Label)
	}
	// TIER TWO IS THE CITIES, nearest first — the two nearest to Bonsall.
	head := got[1].Label + " " + got[2].Label
	if head != "Vista, CA Fallbrook, CA" && head != "Fallbrook, CA Vista, CA" {
		t.Errorf("the cities follow home, nearest first; got %q", head)
	}
	// TIER THREE REACHES WHERE THE CITY TABLE CANNOT. San Luis Rey is a zip
	// place at seven miles and no city at all.
	if !hasLabel(got, "San Luis Rey, CA") {
		t.Errorf("the hyper-local tier is missing from the pool:\n%s", labels(got))
	}
	// AND EVERY ENTRY IS USABLE AS A CARD: something to say, somewhere to say
	// it about, and an identity the schedule can key on.
	seen := map[snapshot.LocationKey]bool{}
	for _, r := range got {
		// SOMETHING TO SAY, SOMEWHERE TO SAY IT ABOUT, A POSTAL CODE THE CARD
		// NAMES, AND A CLOCK. The hyper-local tier came back with no timezone
		// at all on the first build — every place the city table does not hold,
		// which is the whole reason that tier exists.
		if r.Label == "" || r.Lat == 0 || r.Lon == 0 || r.Zip == "" || r.TZ == "" {
			t.Errorf("a pool entry the station cannot read: %+v", r)
		}
		if k := snapshot.Key(r); seen[k] {
			t.Errorf("two entries share one identity, so the schedule can hold only one: %q", r.Label)
		} else {
			seen[k] = true
		}
	}
}

// ONE ENTRY PER PLACE. Vista has four zip centroids inside twenty miles of
// Bonsall, and a line-up that read Vista four times would be the station
// repeating itself while the region went unheard.
func TestThePoolHoldsEachPlaceOnce(t *testing.T) {
	idx := indexForTest(t)
	seen := map[string]bool{}
	for _, r := range Pool(idx, bonsall, 25, 25) {
		if seen[r.Label] {
			t.Errorf("%q is in the pool twice", r.Label)
		}
		seen[r.Label] = true
	}
}

// THE CAP IS A CAP, and it is the HUM LEAD's number: 25.
func TestThePoolIsCapped(t *testing.T) {
	idx := indexForTest(t)
	wide := Pool(idx, bonsall, 50, 25)
	if len(wide) != 25 {
		t.Errorf("a fifty-mile fence overflows the cap and is cut to it; got %d", len(wide))
	}
	// AND THE CAP KEEPS THE TIERS' ORDER: what is dropped is the far end of the
	// last tier, never the station's own place.
	if wide[0].Label != bonsall.Label {
		t.Errorf("the cap dropped home; got %q", wide[0].Label)
	}
}

// A FENCE THAT HOLDS ALMOST NOTHING STILL HOLDS HOME.
//
// MEASURED AND IT IS WHY THE FLOOR MATTERS: a two-mile fence around Bonsall
// holds no city and one zip place — its own. The station can still read where
// it stands, and the console is left to say the rest honestly (F-83).
func TestATightFenceStillKnowsWhereTheStationIs(t *testing.T) {
	idx := indexForTest(t)
	got := Pool(idx, bonsall, 2, 25)
	if len(got) == 0 || got[0].Label != bonsall.Label {
		t.Fatalf("a station always knows where it transmits from; got %s", labels(got))
	}
	if len(got) > 3 {
		t.Errorf("a two-mile fence holds almost nothing; got %d:\n%s", len(got), labels(got))
	}
}

// AND NO FENCE IS NOT "EVERYWHERE".
func TestAPoolWithNoFenceIsJustTheStation(t *testing.T) {
	idx := indexForTest(t)
	got := Pool(idx, bonsall, 0, 25)
	if len(got) != 1 || got[0].Label != bonsall.Label {
		t.Errorf("no radius admits no region; got %s", labels(got))
	}
	if got := Pool(idx, snapshot.LocationRef{}, 25, 25); len(got) != 0 {
		t.Errorf("and no transmitter admits nothing at all; got %s", labels(got))
	}
}

func hasLabel(refs []snapshot.LocationRef, want string) bool {
	for _, r := range refs {
		if r.Label == want {
			return true
		}
	}
	return false
}

func labels(refs []snapshot.LocationRef) string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Label+" ("+r.Zip+")")
	}
	return "  " + strings.Join(out, "\n  ")
}

func indexForTest(t *testing.T) *geodata.Index {
	t.Helper()
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	return idx
}
