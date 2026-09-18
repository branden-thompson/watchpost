package app

// pool_pipeline_test.go — the station's pool stays in the pipeline that fetches
// it (D-112).

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/sched"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// poolPipes is a livePipelines whose recent pipeline can be asked what it holds.
//
// A REAL ASSEMBLER, NOT A FAKE. `update` early-returns on a nil one, so a fake
// that recorded the call would pass while the real path did nothing — which is
// the exact shape of the defect under test.
func poolPipes(t *testing.T, pool []snapshot.LocationRef) *livePipelines {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{poolRefs: pool}
	lp.recent = &recentPipeline{
		asm:     newAssembler(nil, nil),
		publish: func() {},
		// A SCHEDULER THAT IS NOT ONE. `update` calls this for every newcomer and
		// a real one would fetch against the network from a unit test; nil is what
		// it does with a location it cannot schedule, which is the honest stand-in.
		newFor: func(snapshot.LocationRef) *sched.Scheduler { return nil },
	}
	return lp
}

// THE POOL IS PART OF THE LIST ON EVERY COMMIT, not just at the seed.
//
// `withPool` WAS APPLIED ONCE, WHERE THE PIPELINE WAS BUILT, and every commit
// after that reconciled against the LISTENER's list alone — so the first lookup,
// favourite or watchlist edit told `SetLocations` that twenty-five locations were
// no longer wanted. It stopped every one of their schedulers and dropped their
// data, and the pool table went back to shimmering and stayed there.
func TestTheStationsPoolSurvivesACommit(t *testing.T) {
	pool := []snapshot.LocationRef{
		{Label: "Fallbrook, CA", Zip: "92028", Lat: 33.37, Lon: -117.25},
		{Label: "Vista, CA", Zip: "92081", Lat: 33.20, Lon: -117.24},
	}
	lp := poolPipes(t, pool)
	recent := []snapshot.LocationRef{{Label: "Boise, ID", Zip: "83702", Lat: 43.62, Lon: -116.2}}

	if err := lp.commit(nil, recent); err != nil {
		t.Fatal(err)
	}
	held := map[snapshot.LocationKey]bool{}
	for _, l := range lp.recent.asm.Snapshot().Locations { // bounded by the list (P10-02)
		held[snapshot.Key(snapshot.LocationRef{Lat: l.Lat, Lon: l.Lon})] = true
	}
	for _, r := range pool {
		if !held[snapshot.Key(r)] {
			t.Errorf("%s fell out of the pipeline on a commit; nothing will fetch its weather", r.Label)
		}
	}
	// AND THE LISTENER'S OWN LOCATIONS ARE STILL THERE, or the fix is just the
	// defect pointing the other way.
	if !held[snapshot.Key(recent[0])] {
		t.Error("the recent list fell out instead")
	}
}

// AND A MOVED STATION FETCHES ITS NEW CANDIDATES.
//
// THE ORDER IS THE WHOLE OF IT. Reconciling the pipeline BEFORE the station
// moves PUBLISHES a restationed console a new pool and fetches the old one — it
// names a region whose weather nothing is asking for.
func TestARestationedPoolIsWhatGetsFetched(t *testing.T) {
	was := snapshot.LocationRef{Label: "Somewhere Else, NV", Lat: 39.5, Lon: -119.8}
	lp := poolPipes(t, []snapshot.LocationRef{was})
	// THE STATION BORROWS THE WATCHLIST until it has a transmitter of its own
	// (D-72), so committing a watchlist is what moves it.
	lp.station = stationArea{transmitter: was, radiusMi: 25, followsDefault: true}
	// THE REAL EMBEDDED TABLE, because `locations.Pool` answers nil without one
	// and a pool of nothing cannot show that the WRONG one is being fetched.
	idx, err := geodata.Load()
	if err != nil {
		t.Fatalf("the embedded location table is what a station's pool is derived from: %v", err)
	}
	lp.idx = idx
	watch := []snapshot.LocationRef{{Label: "Oceanside, CA", Zip: "92057", Lat: 33.24, Lon: -117.29}}
	if err := lp.commit(watch, nil); err != nil {
		t.Fatal(err)
	}
	// THE PREMISE: the station actually MOVED. Without this the assertions below
	// pass on a pool that never changed, which is the shape mT3 was.
	if len(lp.poolRefs) == 0 || snapshot.Key(lp.poolRefs[0]) == snapshot.Key(was) {
		t.Fatalf("the station did not restation; the pool is still %v", lp.poolRefs)
	}
	held := map[snapshot.LocationKey]bool{}
	for _, l := range lp.recent.asm.Snapshot().Locations { // bounded by the list (P10-02)
		held[snapshot.Key(snapshot.LocationRef{Lat: l.Lat, Lon: l.Lon})] = true
	}
	for _, r := range lp.poolRefs {
		if !held[snapshot.Key(r)] {
			t.Fatalf("the new station's pool is published but not fetched: %s is missing", r.Label)
		}
	}
	// AND THE OLD ONE IS GONE, which is what makes this a MOVE rather than a
	// growing list nobody prunes.
	if held[snapshot.Key(snapshot.LocationRef{Lat: 39.5, Lon: -119.8})] {
		t.Error("the station moved and kept fetching the region it left")
	}
}
