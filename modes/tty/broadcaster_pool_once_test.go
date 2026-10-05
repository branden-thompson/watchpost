package tty

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// A POOL ROW FINDS A WATCHED PLACE IN THE PRIORITY SNAPSHOT (D-208): a place
// the watchlist and the pool share is fetched once, by the priority pipeline,
// so its row reads it there; the pool's own places come from the recent
// snapshot, which wins where both hold a place.
func TestAPoolRowReadsAWatchedPlaceFromThePriorityPipeline(t *testing.T) {
	home := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.24, Lon: -117.29}
	vista := snapshot.LocationRef{Label: "Vista, CA", Lat: 33.20, Lon: -117.24}
	b := NewBroadcaster()
	b.snap = &snapshot.Snapshot{Locations: []snapshot.Location{
		{Label: "Oceanside (priority)", Lat: home.Lat, Lon: home.Lon},
		{Label: "Vista (priority)", Lat: vista.Lat, Lon: vista.Lon},
	}}
	b.pool = &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "Vista (recent)", Lat: vista.Lat, Lon: vista.Lon}}}
	idx := b.locIndex()
	if l := idx.at(home); l == nil || l.Label != "Oceanside (priority)" {
		t.Errorf("the watched place's row reads %+v; want the priority snapshot's", l)
	}
	if l := idx.at(vista); l == nil || l.Label != "Vista (recent)" {
		t.Errorf("a pool place's row reads %+v; want the recent snapshot's, which wins", l)
	}
}
