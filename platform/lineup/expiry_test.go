package lineup

import (
	"testing"
	"time"
)

// lapsing is a hazard with an expiry on it.
func lapsing(id string, at time.Time, life time.Duration) Arrival {
	a := hazard(id, 33.30, -117.23, at)
	a.Until = at.Add(life)
	return a
}

func standingFence() Fence {
	return Fence{RadiusMi: 150, Lat: refLat, Lon: refLon, HasOrigin: true}
}

// onStandbyWith is a silent station holding the given arrivals.
func onStandbyWith(t *testing.T, now time.Time, arr ...Arrival) Director {
	t.Helper()
	d := New(Settings{Max: 10}, now)
	d, _ = d.Step(Powered{To: OffAir})
	d, _ = d.Step(Aired{To: AirProgramme, Fence: standingFence()})
	d, _ = d.Step(Arrived{Arrivals: arr, Fence: standingFence()})
	return d
}

// A LAPSED HAZARD LEAVES THE RAIL (D-155).
//
// MEASURED BEFORE THE FIX, and every number here is from that run: a tornado
// warning valid for thirty minutes, admitted at a silent station, still sat on
// the rail six hours later. `firstStale` found nothing — it skips a card with a
// zero `BuiltAt`, and a card nobody has built has no words to have gone off.
// `Projection` counted it, so `heldNotice` escalated to its loudest rung and
// told the operator to go ON AIR and read it. `Next()` offered it the air.
//
// THE LISTENER WAS NEVER AT RISK — `eventsFor` declines to build an expired
// alert, so the words never came. What the console asserted was a pending read
// the schedule could never take, forever, which is FR-3.3 read backwards.
func TestALapsedHazardLeavesTheRail(t *testing.T) {
	t0 := time.Now()
	d := onStandbyWith(t, t0, lapsing("twister", t0, 30*time.Minute))
	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Fatalf("the hazard is admitted while it is in force; the rail holds %d", n)
	}

	d, _ = d.Step(Tick{Now: t0.Add(6 * time.Hour)})

	if n := len(d.lineup.Cards(AlertRail)); n != 0 {
		t.Errorf("a hazard that lapsed five and a half hours ago is still on the rail (%d)", n)
	}
	if n := len(d.lineup.Projection(AlertRail)); n != 0 {
		t.Errorf("the notice still counts %d lapsed hazard(s) — heldNotice reads this", n)
	}
	if next, _, ok := d.lineup.Next(); ok {
		t.Errorf("a lapsed hazard %q is offered the air", next.ID)
	}
}

// AND A HAZARD STILL IN FORCE IS NEVER TOUCHED.
//
// THE DANGEROUS DIRECTION. Holding a live tornado warning off the air because a
// drop rule was written a few minutes too eager would be far worse than the
// defect above, which only ever cost the operator a misleading count.
func TestAHazardStillInForceIsNotDropped(t *testing.T) {
	t0 := time.Now()
	d := onStandbyWith(t, t0, lapsing("twister", t0, 2*time.Hour))

	d, _ = d.Step(Tick{Now: t0.Add(90 * time.Minute)})

	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Fatalf("a warning with half an hour left is still in force; the rail holds %d", n)
	}
	if _, _, ok := d.lineup.Next(); !ok {
		t.Error("a hazard in force is offered the air")
	}
}

// AND ONE LAPSED ARRIVAL DOES NOT TAKE THE LIVE ONES WITH IT.
//
// EVERY, NOT ANY. A burst is ONE card carrying many hazards (MVS-D-77). Dropping
// the card when the first of them lapses would take the rest off the air with
// it — the same shape as the `heldNotice` defect that counted cards instead of
// hazards, and with a far worse consequence.
func TestABurstSurvivesWhileAnyOfItsHazardsIsInForce(t *testing.T) {
	t0 := time.Now()
	d := onStandbyWith(t, t0,
		lapsing("short", t0, 15*time.Minute),
		lapsing("long", t0, 4*time.Hour))
	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Fatalf("both hazards burst into one card; the rail holds %d", n)
	}

	d, _ = d.Step(Tick{Now: t0.Add(time.Hour)})

	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Fatalf("one hazard lapsed and took a live one with it; the rail holds %d", n)
	}
	// AND WHEN THE LAST ONE LAPSES, THE CARD GOES.
	d, _ = d.Step(Tick{Now: t0.Add(5 * time.Hour)})
	if n := len(d.lineup.Cards(AlertRail)); n != 0 {
		t.Errorf("nothing on the card is in force and it is still held (%d)", n)
	}
}

// AND THE PER-HAZARD BOUNDARY IS `eventsFor`'S, SAID THE SAME WAY.
//
// IT HAS TO BE. The Director decides what reaches the air and the executor
// decides what can be built; a hazard the one calls live and the other calls
// lapsed is a card offered forever and declined forever — which is the defect
// this file exists to close, reintroduced from the other end.
//
// IT COVERS THE PER-HAZARD DIMENSIONS; the per-card halves agree since F-110,
// where the composer learned to skip a lapsed hazard rather than decline the
// burst carrying it.
func TestThePerHazardExpiryBoundaryMatchesTheBuilders(t *testing.T) {
	t0 := time.Now()

	// AN ALERT WITH NO `Until` NEVER EXPIRES — a quake's instant.
	instant := hazard("quake", 33.30, -117.23, t0) // Until stays zero
	d := onStandbyWith(t, t0, instant)
	d, _ = d.Step(Tick{Now: t0.Add(30 * 24 * time.Hour)})
	if n := len(d.lineup.Cards(AlertRail)); n != 1 {
		t.Errorf("a hazard with no expiry was dropped after a month (%d held)", n)
	}

	// AND ONE EXPIRING EXACTLY NOW IS KEPT: the boundary errs towards telling
	// the listener.
	d2 := onStandbyWith(t, t0, lapsing("edge", t0, time.Hour))
	d2, _ = d2.Step(Tick{Now: t0.Add(time.Hour)})
	if n := len(d2.lineup.Cards(AlertRail)); n != 1 {
		t.Errorf("a hazard expiring exactly now was dropped (%d held); eventsFor keeps it", n)
	}
}
