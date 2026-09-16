package app

// locate_radius_test.go — what "a valid location" means on the console.
//
// HUM LEAD, UAT 2026-09-14: "location search should accept any value WITHIN the
// service radius, not just the 25 slot location pool.  Example: 'Rainbow, CA'
// is a valid location within a 25 mi radius of Oceanside, but now it says that
// it's not a valid location."

import (
	"context"
	"testing"

	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// fakeGeocoder stands in for the network. Rainbow, CA is the HUM LEAD's own
// example and its coordinates are the REAL ones the live geocoder returned on
// 2026-09-14 — 14.7 miles from Oceanside, inside a 25-mile radius and in
// neither the city nor the zip table.
type fakeGeocoder struct {
	asked []string
	known map[string]snapshot.LocationRef
}

func (f *fakeGeocoder) Resolve(_ context.Context, q string) (snapshot.LocationRef, error) {
	f.asked = append(f.asked, q)
	if ref, ok := f.known[q]; ok {
		return ref, nil
	}
	return snapshot.LocationRef{}, context.Canceled // any error: "no such place"
}

func locateFixture(t *testing.T) (*livePipelines, *locations.Resolver, *fakeGeocoder) {
	t.Helper()
	idx, err := geodata.Load()
	if err != nil {
		t.Fatalf("the embedded table is what the fence is measured against: %v", err)
	}
	geo := &fakeGeocoder{known: map[string]snapshot.LocationRef{
		"Rainbow, CA":   {Label: "Rainbow, CA", Lat: 33.41031, Lon: -117.14781},
		"Lone Pine, CA": {Label: "Lone Pine, CA", Lat: 36.606, Lon: -118.0629},
	}}
	r, err := locations.New(idx, geo)
	if err != nil {
		t.Fatal(err)
	}
	oceanside := snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2407, Lon: -117.3025}
	lp := &livePipelines{idx: idx}
	lp.setStation(stationArea{transmitter: oceanside, radiusMi: 25})
	return lp, r, geo
}

// THE DEFECT ITSELF. Rainbow is inside the radius and outside the pool.
func TestAPlaceInsideTheRadiusButOutsideThePoolIsValid(t *testing.T) {
	lp, r, _ := locateFixture(t)
	if _, _, inPool := lp.lookInPool("Rainbow, CA"); inPool {
		t.Fatal("fixture: Rainbow is supposed to be OUTSIDE the 25-slot pool")
	}

	ref, within, found, _ := lp.locateInRadius(r)("Rainbow, CA")

	if !found {
		t.Fatal("a real place the geocoder knows answered 'no such location'")
	}
	if !within {
		t.Errorf("%s is 14.7 mi from a 25 mi station and was refused as out of range", ref.Label)
	}
	if ref.Tag == "" {
		t.Error("the ref reached the caller with no tag; a card names one")
	}
}

// AND THE FENCE STILL HOLDS. Accepting everything would be the same bug pointed
// the other way.
func TestARealPlaceOutsideTheRadiusIsNamedButNotWithin(t *testing.T) {
	lp, r, _ := locateFixture(t)

	ref, within, found, _ := lp.locateInRadius(r)("Lone Pine, CA")

	if !found {
		t.Fatal("a real place must resolve; 'outside the radius' is not 'does not exist'")
	}
	if within {
		t.Errorf("%s is 200 miles away and was accepted as broadcastable", ref.Label)
	}
}

// A NAME THAT MEANS NOTHING IS THE THIRD ANSWER, and it must not read as the
// second: the window points the operator at Observer for one and asks them to
// retype for the other.
func TestANameThatResolvesToNothingIsNotFound(t *testing.T) {
	lp, r, _ := locateFixture(t)
	if _, _, found, _ := lp.locateInRadius(r)("zzzzzzzz"); found {
		t.Error("a name nothing knows was reported as a real place")
	}
}

// THE POOL SHORT-CIRCUITS THE NETWORK. It is the common case and it is free;
// reaching the geocoder for a place the Director already offers would be a
// round trip to re-learn what is in hand.
func TestAPooledLocationNeverReachesTheGeocoder(t *testing.T) {
	lp, r, geo := locateFixture(t)

	_, within, found, _ := lp.locateInRadius(r)("Vista")

	if !found || !within {
		t.Fatalf("a pooled location must answer reachable: found=%v within=%v", found, within)
	}
	if len(geo.asked) != 0 {
		t.Errorf("the pool arm reached the network: %v", geo.asked)
	}
}

// A STATION WITH NOWHERE TO TRANSMIT FROM REACHES NOTHING — the same reading
// locations.Pool and lineup.Fence both take of an unset epicentre.
func TestAStationWithNoTransmitterReachesNothing(t *testing.T) {
	lp, r, _ := locateFixture(t)
	lp.setStation(stationArea{radiusMi: 25})

	_, within, found, _ := lp.locateInRadius(r)("Rainbow, CA")

	if !found {
		t.Fatal("the place still exists; only the station's reach is in question")
	}
	if within {
		t.Error("a station with no epicentre claimed to reach somewhere")
	}
}

// A LOOKUP THAT COULD NOT BE MADE IS NOT "NO SUCH PLACE" (D-151).
//
// A 5-second timeout, a DNS blip or a cancelled context must not read as
// `found=false`: that is identical to a genuine no-match, and it tells the
// operator a real location does not exist while disabling the key that would
// have retried it.
//
// THE MAPPING IS THE APP'S, AND THIS IS WHERE IT IS EXERCISED. A console-side
// test that builds the fourth state from a verdict message directly proves the
// window and not the mapping of an error onto it; `mCL3` tells the two apart.
func TestATimedOutLookupReportsThatItCouldNotAsk(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatalf("the embedded table: %v", err)
	}
	// A FALLBACK THAT NEVER ANSWERS. The name is absent from the offline index,
	// so the resolver reaches this and the context expires.
	r, err := locations.New(idx, hangingGeocoder{})
	if err != nil {
		t.Fatal(err)
	}
	oceanside := snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057", Lat: 33.2407, Lon: -117.3025}
	lp := &livePipelines{idx: idx}
	lp.setStation(stationArea{transmitter: oceanside, radiusMi: 25})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the question cannot be put at all
	lp.ctx = ctx

	_, _, found, asked := lp.locateInRadius(r)("Rainbow, CA")
	if found {
		t.Fatal("a lookup that never answered reported a place")
	}
	if asked {
		t.Error("a failed lookup is reported as a genuine no-match: the operator is told a real " +
			"location does not exist, and the retry key is disabled by the same answer")
	}
}

// hangingGeocoder never answers; the context decides.
type hangingGeocoder struct{}

func (hangingGeocoder) Resolve(ctx context.Context, _ string) (snapshot.LocationRef, error) {
	<-ctx.Done()
	return snapshot.LocationRef{}, ctx.Err()
}
