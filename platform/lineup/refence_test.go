package lineup

import (
	"testing"
	"time"
)

// bonsallLat/Lon is the station the fence rulings were measured against.
const refLat, refLon = 33.2881, -117.2256

func hazard(id string, lat, lon float64, at time.Time) Arrival {
	return Arrival{ID: id, Headline: "TORNADO WARNING · " + id, Subject: id,
		At: at, Lat: lat, Lon: lon, HasPoint: true, Severity: 3}
}

// THE RAIL FILTERS AND EXPANDS WITH THE SURFACE (D-75).
//
// THE HUM LEAD ASKED WHETHER IT DID, and the honest answer was half: new
// arrivals were scoped correctly and everything ALREADY ON THE RAIL went on
// being read under the fence that admitted it. Measured before the fix — a
// hundred-mile hazard survived a narrowing to twenty-five.
//
// HELD, NOT DROPPED. DR-3 says nothing admitted is dropped unread; the ruling
// says nothing outside the service area is heard on the console. The card waits
// for a surface whose fence admits it, and both rules stay true.
func TestTheRailHoldsWhatTheNewFenceDoesNotAdmit(t *testing.T) {
	now := time.Now()
	wide := Fence{RadiusMi: 150, Lat: refLat, Lon: refLon, HasOrigin: true}
	narrow := Fence{RadiusMi: 25, Lat: refLat, Lon: refLon, HasOrigin: true}

	d := New(Settings{Max: 10}, now)
	d, _ = d.Step(Powered{To: Running})
	// A HAZARD A HUNDRED MILES OUT, admitted while the listener's wide filter
	// was the one in force.
	d, _ = d.Step(Arrived{Arrivals: []Arrival{hazard("lancaster", 34.60, -118.20, now)}, Fence: wide})
	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Fatalf("the wide fence admits it; the rail holds %d", n)
	}

	// THE OPERATOR MOVES TO THE CONSOLE, whose service area is twenty-five miles.
	d, _ = d.Step(Aired{To: AirProgramme, Fence: narrow})
	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Errorf("the card is HELD, not dropped — DR-3 — and the rail holds %d", n)
	}
	if _, _, ok := d.lineup.Next(); ok {
		t.Error("a card outside the service area is not offered the air")
	}

	// AND GOING BACK RELEASES IT, which is the "expand itself" half.
	d, _ = d.Step(Aired{To: AirMonitor, Fence: wide})
	next, track, ok := d.lineup.Next()
	if !ok || track != AlertRail || next.ID == "" {
		t.Errorf("widening the fence releases what it holds; got %v / %v / %v", next.ID, track, ok)
	}
}

// AND A CARD INSIDE THE NEW FENCE IS NEVER HELD — which is the half that would
// be dangerous to get wrong. Holding a hazard in the operator's own town because
// they changed surfaces would be worse than the defect this fixes.
func TestTheRailKeepsWhatTheNewFenceStillAdmits(t *testing.T) {
	now := time.Now()
	d := New(Settings{Max: 10}, now)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Arrived{
		Arrivals: []Arrival{hazard("bonsall", 33.30, -117.23, now)},
		Fence:    Fence{RadiusMi: 150, Lat: refLat, Lon: refLon, HasOrigin: true},
	})
	d, _ = d.Step(Aired{To: AirProgramme, Fence: Fence{RadiusMi: 25, Lat: refLat, Lon: refLon, HasOrigin: true}})
	if _, _, ok := d.lineup.Next(); !ok {
		t.Error("a hazard five miles from the transmitter is still the operator's business")
	}
}

// A BURST IS HELD ONLY IF NONE OF ITS HAZARDS IS INSIDE. One card carries
// several; silencing it because a companion alert was farther out would silence
// a hazard in the operator's own town.
func TestABurstSurvivesIfAnyOfItIsInside(t *testing.T) {
	now := time.Now()
	d := New(Settings{Max: 10}, now)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Arrived{
		Arrivals: []Arrival{hazard("lancaster", 34.60, -118.20, now), hazard("bonsall", 33.30, -117.23, now)},
		Fence:    Fence{RadiusMi: 150, Lat: refLat, Lon: refLon, HasOrigin: true},
	})
	d, _ = d.Step(Aired{To: AirProgramme, Fence: Fence{RadiusMi: 25, Lat: refLat, Lon: refLon, HasOrigin: true}})
	if _, _, ok := d.lineup.Next(); !ok {
		t.Error("a burst with one hazard inside the service area is still read")
	}
}

// A HELD CARD DOES NOT BLOCK THE ONES BEHIND IT.
//
// IT IS SKIPPED AT THE SELECTION, not refused at the air — refusing it there
// would leave a hazard in the operator's own town waiting on one that is not.
func TestAHeldCardDoesNotBlockTheRail(t *testing.T) {
	now := time.Now()
	wide := Fence{RadiusMi: 150, Lat: refLat, Lon: refLon, HasOrigin: true}
	d := New(Settings{Max: 10}, now)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Arrived{Arrivals: []Arrival{hazard("lancaster", 34.60, -118.20, now)}, Fence: wide})
	d, _ = d.Step(Arrived{Arrivals: []Arrival{hazard("bonsall", 33.30, -117.23, now.Add(time.Second))}, Fence: wide})
	if n := len(d.lineup.Cards(AlertRail)); n != 2 {
		t.Fatalf("two bursts, two cards; got %d", n)
	}
	d, _ = d.Step(Aired{To: AirProgramme, Fence: Fence{RadiusMi: 25, Lat: refLat, Lon: refLon, HasOrigin: true}})
	next, _, ok := d.lineup.Next()
	if !ok {
		t.Fatal("the admissible card is still offered")
	}
	if next.OutOfFence {
		t.Errorf("the held card was offered anyway: %q", next.ID)
	}
}

// THE BED'S STATE TRAVELS ON THE PUBLISH (F-79, closed at D-78).
//
// A PLANT SAID THIS WAS MISSING. The console's own tests feed it a `BedMsg` and
// prove it DRAWS one; nothing proved the schedule ever SENDS one, so deleting
// the bed from the effect left every test green and the row a constant again —
// which is the defect F-79 was filed for in the first place.
//
// ALL THREE FACTS TRAVEL TOGETHER, which is why they are on one effect: D-62
// consolidated the line-up, the power and the bed into ONE region so the
// operator reads the air state in a glance, and three facts drawn together and
// published apart would undo that at the seam.
func TestThePublishCarriesTheBedsState(t *testing.T) {
	now := time.Now()
	d := New(Settings{Max: 5}, now)
	d, _ = d.Step(Monitored{Running: true})
	d, _ = d.Step(Programme{Watchlist: []string{"a", "b"}, Dwell: time.Minute})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})

	// THE CUT-OVER IS WHAT SETTLES — `lineup.CutOver`'s first production caller
	// is the console's `[b]` (D-78) — and the publish it produces carries all
	// three facts the station section draws.
	_, fx := d.Step(CutOver{ToBed: true})
	pub, ok := lastPublish(fx)
	if !ok {
		t.Fatalf("a cut-over settles and publishes; got %v", fx)
	}
	if pub.Bed.Ref != "a" || !pub.Bed.Live {
		t.Errorf("the publish names what the bed is tuned to; got %+v", pub.Bed)
	}
	if !pub.Bed.Carrying {
		t.Errorf("and that it is carrying; got %+v", pub.Bed)
	}

	// AND CUTTING BACK SAYS SO, or the row would learn ACTIVE and never unlearn
	// it — the same constant one state along.
	d, _ = d.Step(CutOver{ToBed: true})
	_, fx = d.Step(CutOver{ToBed: false})
	if pub, ok := lastPublish(fx); !ok || pub.Bed.Carrying {
		t.Errorf("cutting back publishes a bed that is not carrying; got %+v / %v", pub.Bed, ok)
	}
}

func lastPublish(fx []Effect) (Publish, bool) {
	for i := len(fx) - 1; i >= 0; i-- {
		if p, ok := fx[i].(Publish); ok {
			return p, true
		}
	}
	return Publish{}, false
}
