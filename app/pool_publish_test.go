package app

// pool_publish_test.go — D-93: the console is TOLD the station's pool.
//
// THE LINE-UP TABLE NEEDS WHAT THE CARD DOES NOT CARRY. `lineup.Card` is
// domain-free by DR-1 — it holds a subject and a headline, never a zip or a
// distance — so the console joins the card against the pool the Producer itself
// offered. That join is only honest if the two arrive together.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// THE AREA AND ITS POOL ARE ONE DERIVATION, so they are one message.
//
// A console told them separately could hold a pool belonging to an area it has
// stopped showing — D-59's torn pair, one fact along, and the reason the lineup
// and the power already travel together.
func TestTheAreaAndItsPoolTravelTogether(t *testing.T) {
	lp := bedPipelines(t) // a station at Bonsall, with the real geodata index
	s := lp.currentStation()
	pool := lp.currentPool()
	if len(pool) == 0 {
		t.Fatalf("the fixture's station reaches no locations, so this measures nothing")
	}

	var got []tty.StationAreaMsg
	publishTo(t, &got, s, pool)

	if len(got) != 1 {
		t.Fatalf("one area change, one message; got %d", len(got))
	}
	if got[0].Transmitter != s.transmitter || got[0].RadiusMi != s.radiusMi {
		t.Errorf("the message does not carry the area it was published for")
	}
	if len(got[0].Pool) != len(pool) {
		t.Errorf("the area arrived with %d pool entries and the station has %d: a console that has to "+
			"ask for the pool separately can hold one that belongs to an area it is no longer showing",
			len(got[0].Pool), len(pool))
	}
	// AND THE ENTRIES CARRY WHAT THE TABLE NEEDS. A ref with no zip and no
	// coordinates is a row the line-up cannot place.
	for _, r := range got[0].Pool {
		if r.Label == "" || (r.Lat == 0 && r.Lon == 0) {
			t.Errorf("a pool entry reached the console unusable: %+v", r)
			break
		}
	}
}

// THE POOL ARRIVES IN READ-PRIORITY ORDER, WITH HOME FIRST — and that order is
// what the eviction ruling rests on (D-93).
//
// IT IS NOT GLOBALLY DISTANCE-SORTED, and a first draft of this test asserted
// that it was and FAILED: "San Luis Rey, CA at 6.9 mi follows one at 24.3 mi".
// `locations.Pool` is THREE TIERS, each nearest-first, concatenated —
//
//  1. HOME, the transmitter itself: "the most local report it has, and the one
//     its listeners are standing in"
//  2. the region's CITIES, nearest first, from a population-filtered table so
//     "distance order IS major locations first"
//  3. the HYPER-LOCAL places, nearest first, "filling in behind"
//
// — so tier three restarts at near distances behind tier two. That tiering is
// already a population preference, which is why the eviction rule is "the tail of
// this order" rather than a distance-and-population rule invented beside it.
func TestThePoolReachesTheConsoleHomeFirst(t *testing.T) {
	lp := bedPipelines(t)
	pool := lp.currentPool()
	if len(pool) < 3 {
		t.Fatalf("the fixture needs a few locations to have an order; got %d", len(pool))
	}
	tx := lp.currentStation().transmitter
	if snapshot.Key(pool[0]) != snapshot.Key(tx) {
		t.Errorf("the pool does not open on the transmitter's own location: got %q, want %q.  HOME is "+
			"tier one, and the eviction rule drops the TAIL — so a pool that does not start at home "+
			"could evict the place the station is standing in", pool[0].Label, tx.Label)
	}
}

// AND A STATION WITH NO EPICENTRE OFFERS NOTHING, rather than the whole country.
func TestAStationWithNoTransmitterPublishesAnEmptyPool(t *testing.T) {
	lp := &livePipelines{idx: indexForTest(t)}
	lp.setStation(stationFrom(config.Config{}))

	var got []tty.StationAreaMsg
	publishTo(t, &got, lp.currentStation(), lp.currentPool())

	if len(got) != 1 || len(got[0].Pool) != 0 {
		t.Errorf("an unset epicentre has no region; got %d entries", len(got[0].Pool))
	}
}

// publishTo drives the REAL publishArea and captures what it sent.
//
// A SECOND COPY OF ITS BODY WOULD MEASURE THE COPY. The first draft of this
// helper rebuilt the message by hand, which would have passed with `publishArea`
// deleted — the same shape as D-74's `y4`, where the predicate was asserted and
// the behaviour was not.
func publishTo(t *testing.T, out *[]tty.StationAreaMsg, s stationArea, pool []snapshot.LocationRef) {
	t.Helper()
	publishArea(func(m tea.Msg) {
		if a, ok := m.(tty.StationAreaMsg); ok {
			*out = append(*out, a)
		}
	}, s, pool)
}

// miBetween is how far a pool entry sits from the transmitter, in miles.
func miBetween(tx, r snapshot.LocationRef) float64 {
	return geo.HaversineKM(tx.Lat, tx.Lon, r.Lat, r.Lon) * 0.621371
}
