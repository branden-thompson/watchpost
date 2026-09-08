//go:build watchpost_debug

package app

// inject_scenarios_test.go — FR-4.1, FR-4.2, FR-4.3.
//
// THE DEFECT THAT NAMES THIS FILE. The scenario labelled "An Emergency Order
// (leads the rail, overruns Max)" injected a Tornado Warning, byte-identical to
// the default arm beside it. A Tornado Warning is not in the civil-emergency
// family, so it cannot lead the rail — leading the rail is what being an
// emergency order MEANS. Had that scenario injected an actual Evacuation
// Immediate, an operator pressing ctrl+d would have watched #15 happen: the
// evacuation order reaching the marquee as an ordinary warning, five weeks
// before a listener did.
//
// So the property is not "the window offers three things". It is that every
// lane the feed can produce has a scenario, and that each scenario's payload
// LANDS in the lane it is named for — checked through the app's own classifier,
// not against a second copy of the mapping.

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/closedset"
)

func TestEveryLaneTheFeedProducesHasAScenarioThatLandsInIt(t *testing.T) {
	now := time.Now()
	closedset.EachMember(t, "ctrl+d scenarios", globalfeed.FeedLanes(), nil, func(l globalfeed.Lane) bool {
		key, ok := scenarioKeyFor(l)
		if !ok {
			return false
		}
		evs := injectedEvents(key, now)
		if len(evs) == 0 {
			t.Errorf("%s: the %q scenario fabricates nothing", category.Of(l).Bucket, key)
			return true
		}
		for _, e := range evs {
			if got := globalfeed.LaneOf(e); got != l {
				t.Errorf("%s: the %q scenario injects %q, which the app lanes as %s — a scenario "+
					"that does not reach its own lane exercises the path nobody was worried about",
					category.Of(l).Bucket, key, e.Type, category.Of(got).Bucket)
			}
		}
		return true
	})
}

// AND A TEST EVENT SAYS SO, IN THE EVENT ITSELF. The marquee and the read
// script have to mark it, and they ship — so the mark cannot be the injector's
// id prefix, which is the one string a clean binary is proven not to contain.
func TestEveryFabricatedEventIsMarkedAsOne(t *testing.T) {
	for _, key := range allScenarioKeys() {
		for _, e := range injectedEvents(key, time.Now()) {
			if !e.Fabricated {
				t.Errorf("the %q scenario fabricates %q and it does not say so — nothing "+
					"downstream can tell it from a real hazard", key, e.Type)
			}
		}
	}
}

// AND IT EXPIRES WITHIN TWO MINUTES (FR-4.3). They carried an hour, which is
// thirty times the stated bound: a test run at the top of the hour left a
// fabricated tornado warning in the marquee and the severe window for the rest
// of it, and in the seen store for seven days.
func TestFabricatedEventsExpireWithinTwoMinutes(t *testing.T) {
	now := time.Now()
	for _, key := range allScenarioKeys() {
		for _, e := range injectedEvents(key, now) {
			if e.Until.IsZero() {
				t.Errorf("the %q scenario fabricates %q with NO expiry: Active keeps it "+
					"until the feed drops it, and the feed never had it", key, e.Type)
				continue
			}
			if d := e.Until.Sub(now); d > testEventLife {
				t.Errorf("the %q scenario's %q lives %s, bound %s", key, e.Type, d, testEventLife)
			}
		}
	}
}

// AND A KEY NOTHING OFFERS FABRICATES NOTHING. The default arm handed back a
// Tornado Warning, so a typo or a stale window produced a red alert nobody
// asked for — and produced it identically to the scenario named for the
// emergency path, which is how FR-4.1 went unnoticed.
func TestAnUnknownScenarioKeyFabricatesNothing(t *testing.T) {
	if evs := injectedEvents("no-such-scenario", time.Now()); len(evs) != 0 {
		t.Errorf("an unknown key fabricated %d events, first %q", len(evs), evs[0].Type)
	}
}
