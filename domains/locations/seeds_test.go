package locations

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
)

func TestSeedsProduceCompleteRefs(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	refs := Seeds(idx, 50) // UAT 48: 50 most-recent seeds
	if len(refs) != 50 {
		t.Fatalf("want 50 refs, got %d", len(refs))
	}
	for _, r := range refs {
		// Every seed must be dashboard-renderable AND fetchable: label+zip for
		// the table, lat/lon+tz for the NWS pipeline (B2 PTY tz='' regression).
		if r.Label == "" || r.Zip == "" || r.TZ == "" || (r.Lat == 0 && r.Lon == 0) {
			t.Fatalf("incomplete seed ref: %+v", r)
		}
	}
}
