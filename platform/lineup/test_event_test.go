package lineup

// test_event_test.go — FR-4.4's selection half: a fabricated alert must not
// cost a real one its place in the burst.
//
// THE HAZARD IS ARITHMETIC. The ctrl+d window's burst scenario injects six
// events against a Max of five, so an operator exercising the station while
// real weather is arriving pushes a real hazard out of the read budget and
// hears the divert count absorb it. The burst is the one place in the app where
// something is DROPPED for lack of room, and a fabricated event taking that
// room is a fabricated event silencing a real alert.

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
)

// A REAL ALERT KEEPS ITS PLACE WHEN THE BUDGET IS FULL OF TEST EVENTS.
//
// The real one here is the MILDEST and the OLDEST of the seven, so today's
// ordering — rung, then severity, then recency — puts it last and the budget
// drops it. That is the arrangement the operator cannot see: the alert that
// loses its slot is by definition the one at the bottom of the read order.
func TestAFabricatedAlertNeverTakesARealOnesPlaceInTheBurst(t *testing.T) {
	var arrivals []Arrival
	for i, a := range many("test", category.Warnings, 6) {
		a.Test, a.Severity = true, 90-i // all above the real one
		arrivals = append(arrivals, a)
	}
	real := arrival("real-tornado", category.Warnings, 10, time.Hour)
	arrivals = append(arrivals, real)

	b := mustPlan(t, arrivals, Settings{Max: 5})
	if !b.HasTakeover() {
		t.Fatal("seven arrivals and no takeover")
	}
	var read bool
	for _, ref := range b.Takeover.Refs {
		if ref == real.ID {
			read = true
		}
	}
	if !read {
		t.Errorf("the real alert was diverted by six fabricated ones: the burst read %v — an "+
			"operator testing the station at the wrong moment silenced a live hazard", b.Takeover.Refs)
	}
}

// AND A FABRICATED EMERGENCY DOES NOT SPEND THE BUDGET EITHER. An emergency
// order is the one category read past Max, because an evacuation is never the
// thing that gets dropped. That exemption belongs to real ones: a fabricated
// evacuation order taking it costs a real warning its place, in the window
// whose whole purpose is to be safe to press.
func TestAFabricatedEmergencyDoesNotSpendTheBudget(t *testing.T) {
	arrivals := many("real", category.Warnings, 5)
	evac := arrival("test-evac", category.Emergency, 99, 0)
	evac.Test = true
	arrivals = append(arrivals, evac)

	b := mustPlan(t, arrivals, Settings{Max: 5})
	read := map[string]bool{}
	for _, ref := range b.Takeover.Refs {
		read[ref] = true
	}
	for _, a := range arrivals {
		if a.Test {
			continue
		}
		if !read[a.ID] {
			t.Errorf("%s was dropped from the burst (read %v): a fabricated emergency spent the "+
				"slot a real warning was owed", a.ID, b.Takeover.Refs)
		}
	}
}
