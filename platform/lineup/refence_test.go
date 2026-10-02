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
// New arrivals are scoped by the fence in force, and so is everything ALREADY
// ON THE RAIL: otherwise it goes on being read under the fence that admitted
// it, and a hundred-mile hazard survives a narrowing to twenty-five.
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
// they changed surfaces would be worse than reading one from outside the fence.
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
// THE SCHEDULE MUST SEND IT. The console's own tests feed it a `BedMsg` and
// prove it DRAWS one; without this test, deleting the bed from the effect leaves
// every test green and the row a constant — the defect F-79 names.
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

	// THE CUT-OVER IS WHAT SETTLES — `lineup.CutOver`'s production caller is
	// the console's `[b]` (D-78) — and the publish it produces carries all
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

// TestAZoneOnlyCardIsHeldWhenTheNewSCOPEDoesNotFollowIt.
//
// THE SAME RULING AS ABOVE, FOR THE HAZARD THAT CANNOT BE MEASURED. D-75 holds
// what the new fence does not admit — and a zone-only alert has no point, so
// "admit" cannot mean a distance. It means the app follows this hazard at a
// watched location, which is a fact about the SCOPE, and the scope is exactly
// what changed.
//
// A VERDICT FROZEN ON THE ARRIVAL by whichever scope planned it would let
// `Admits` wave the card through: a zone-only alert tied to a watched location
// in Observer would be read out over a console whose transmitter is four
// hundred miles away — through a fence, past a radius the HUM LEAD set, with
// nothing ever measured. No narrower fence could close that bypass, because
// the narrowing is not what it consults.
func TestAZoneOnlyCardIsHeldWhenTheNewScopeDoesNotFollowIt(t *testing.T) {
	now := time.Now()
	zoneOnly := Arrival{ID: "NWS.zone.1", TrackedAs: "zone1", Headline: "FLOOD WARNING · zone",
		Subject: "zone", At: now, Severity: 3} // no point: HasPoint is false

	// OBSERVER FOLLOWS IT. The listener has this alert at a watched location.
	watching := Fence{RadiusMi: 50, Lat: refLat, Lon: refLon, HasOrigin: true,
		Tracked: map[string]bool{"zone1": true}}
	// THE CONSOLE DOES NOT. Same radius, a transmitter four hundred miles north,
	// and no watched location there carrying this hazard.
	elsewhere := Fence{RadiusMi: 50, Lat: 39.0, Lon: -121.0, HasOrigin: true}

	d := New(Settings{Max: 10}, now)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Arrived{Arrivals: []Arrival{zoneOnly}, Fence: watching})
	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Fatalf("the scope that follows it admits it; the rail holds %d", n)
	}

	d, _ = d.Step(Aired{To: AirProgramme, Fence: elsewhere})
	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Errorf("HELD, not dropped — DR-3 — and the rail holds %d", n)
	}
	if _, _, ok := d.lineup.Next(); ok {
		t.Error("a zone-only alert the station's scope does not follow was offered the air")
	}

	// AND CROSSING BACK RELEASES IT, the same both-ways rule.
	d, _ = d.Step(Aired{To: AirMonitor, Fence: watching})
	if _, _, ok := d.lineup.Next(); !ok {
		t.Error("returning to the scope that follows it must release the card")
	}
}

// AND THE FENCE MOVES WHEN THE AIR DOES NOT (D-154).
//
// THE GAP THE TWO TESTS ABOVE LEAVE, and they leave it in the same shape: both
// move the air on every step, so both only ever exercise the trigger `Aired`
// already serves. The operator who narrows the STATION'S SERVICE AREA in
// Settings moves no air at all, and a fence assigned only behind the air-moved
// guard never sees that narrowing.
//
// THE SAME RULE, THROUGH THE OTHER DOOR: a hundred-mile hazard must not
// survive a narrowing to twenty-five, whichever of the two ways the fence
// changes.
func TestTheRailIsReScopedWhenTheServiceAreaNarrowsUnderAStandingAir(t *testing.T) {
	now := time.Now()
	wide := Fence{RadiusMi: 150, Lat: refLat, Lon: refLon, HasOrigin: true}
	narrow := Fence{RadiusMi: 25, Lat: refLat, Lon: refLon, HasOrigin: true}

	d := New(Settings{Max: 10}, now)
	d, _ = d.Step(Powered{To: Running})
	// THE OPERATOR IS ON THE CONSOLE, service area a hundred and fifty miles,
	// and a hazard a hundred miles out is admitted under it.
	d, _ = d.Step(Aired{To: AirProgramme, Fence: wide})
	d, _ = d.Step(Arrived{Arrivals: []Arrival{hazard("lancaster", 34.60, -118.20, now)}, Fence: wide})
	if _, _, ok := d.lineup.Next(); !ok {
		t.Fatal("the wide service area admits it: it is offered the air")
	}

	// THEY NARROW THE SERVICE AREA TO TWENTY-FIVE. The air does not move —
	// there is no surface change here, only a setting.
	d, _ = d.Step(Refenced{Fence: narrow})

	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Errorf("the card is HELD, not dropped — DR-3 — and the rail holds %d", n)
	}
	if next, _, ok := d.lineup.Next(); ok {
		t.Errorf("a hazard %s outside the narrowed service area is offered the air anyway", next.ID)
	}

	// AND WIDENING IT AGAIN RELEASES THE CARD, with the air still standing.
	d, _ = d.Step(Refenced{Fence: wide})
	if _, _, ok := d.lineup.Next(); !ok {
		t.Error("widening the service area releases what it held")
	}
}

// AND A REPEATED `Aired` CARRIES ITS FENCE THROUGH THE REPEAT GUARD.
//
// THE OTHER HALF OF THE SAME RULE. Returning `d, nil` the moment the air
// matches is right for the AIR and wrong for the FENCE riding the same event:
// the operator who changes a setting and then keys the surface they are
// already on is not sending a no-op.
func TestARepeatedAirStillCarriesANarrowedFence(t *testing.T) {
	now := time.Now()
	wide := Fence{RadiusMi: 150, Lat: refLat, Lon: refLon, HasOrigin: true}
	narrow := Fence{RadiusMi: 25, Lat: refLat, Lon: refLon, HasOrigin: true}

	d := New(Settings{Max: 10}, now)
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme, Fence: wide})
	d, _ = d.Step(Arrived{Arrivals: []Arrival{hazard("lancaster", 34.60, -118.20, now)}, Fence: wide})
	if _, _, ok := d.lineup.Next(); !ok {
		t.Fatal("the wide service area admits it: it is offered the air")
	}

	// THE SAME AIR, A NARROWER FENCE.
	d, _ = d.Step(Aired{To: AirProgramme, Fence: narrow})
	if next, _, ok := d.lineup.Next(); ok {
		t.Errorf("the repeat guard dropped the fence: %s is still offered the air", next.ID)
	}
}
