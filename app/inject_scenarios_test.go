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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/closedset"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// testHere is a watchlist location for the scenarios to happen at.
var testHere = snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23}

func TestEveryLaneTheFeedProducesHasAScenarioThatLandsInIt(t *testing.T) {
	now := time.Now()
	closedset.EachMember(t, "ctrl+d scenarios", globalfeed.FeedLanes(), nil, func(l globalfeed.Lane) bool {
		key, ok := scenarioKeyFor(l)
		if !ok {
			return false
		}
		evs := injectedEvents(key, now, testHere)
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
		for _, e := range injectedEvents(key, time.Now(), testHere) {
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
		for _, e := range injectedEvents(key, now, testHere) {
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
	if evs := injectedEvents("no-such-scenario", time.Now(), testHere); len(evs) != 0 {
		t.Errorf("an unknown key fabricated %d events, first %q", len(evs), evs[0].Type)
	}
}

// A TEST ALERT HAPPENS WHERE THE LISTENER IS (HUM LEAD 2026-09-07).
//
// It carried Location: "Injected Test Location" and no point, which exercises
// neither the D5 location tie nor the radius fence — the two stages most likely
// to be the reason a real alert never reached someone. The whole tool exists to
// test the machinery, and a fabricated event that skips two stages of it tests
// less of the machinery than it appears to.
//
// The mark is what keeps it identifiable, and there are now four of them: the
// event says so, the tape says so at both ends, the window's row says so, and
// the read says so in words.
func TestAFabricatedAlertHappensAtTheListenersOwnLocation(t *testing.T) {
	here := snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23}
	for _, e := range injectedEvents("warn", time.Now(), here) {
		if e.Location != here.Label {
			t.Errorf("the fabricated alert is for %q, not the listener's own location %q", e.Location, here.Label)
		}
		if !e.HasPoint || e.Lat != here.Lat || e.Lon != here.Lon {
			t.Errorf("the fabricated alert carries no point (%v %v %v), so the radius fence and the "+
				"location tie are never exercised", e.HasPoint, e.Lat, e.Lon)
		}
	}

	// AND WITH AN EMPTY WATCHLIST IT STILL SAYS WHERE IT IS. A listener with no
	// locations can still press ctrl+d, and a blank location on the band reads
	// as a rendering fault rather than as a test.
	for _, e := range injectedEvents("warn", time.Now(), snapshot.LocationRef{}) {
		if e.Location == "" {
			t.Error("with no watchlist the fabricated alert has no location at all")
		}
		if e.HasPoint {
			t.Error("with no watchlist the fabricated alert claims a point at 0,0 — the Gulf of Guinea")
		}
	}
}

// THE COPY AND THE CLOCK ARE ONE FACT (FR-4.3 × FR-4.5).
//
// The test read promises "effective for 2 minutes" in words while testEventLife
// says it in a constant, and nothing but this holds them together: change the
// constant and the read goes on promising two minutes, which is the release's
// own subject — one fact with two owners, disagreeing quietly.
func TestTheTestScriptPromisesTheExpiryTheEventCarries(t *testing.T) {
	want := strconv.Itoa(int(testEventLife.Minutes())) + " minutes"
	got := testLine(script.New(""), globalfeed.Event{Type: "Tornado Warning"})
	if !strings.Contains(got, want) {
		t.Errorf("the test read says %q and the event lives %s: the read promises %q", got, testEventLife, want)
	}
}

// A DEBUG BUILD ACTUALLY OFFERS THE INJECTION (F-21b).
//
// THE WINDOW SAYS "NOT AVAILABLE IN THIS BUILD" WHENEVER THE HOOK IS NIL, and
// it is right to: a scenario list with no injector behind it is a row that
// fabricates nothing. But that message is also what a correctly-built
// DIAGNOSTICS binary would show if the wiring ever came loose, and nothing here
// asserted the wiring — only that a release build has none of it. UAT hit the
// mirror of this on 2026-09-07: a clean binary and a diagnostics binary that
// differed only in a filename, and the clean one was run.
func TestADebugBuildWiresTheInjector(t *testing.T) {
	lp := &livePipelines{}
	if lp.injectHook() == nil {
		t.Error("a debug build supplies no injector: the ctrl+d window will say injection is not " +
			"available in a build that was made specifically to have it")
	}
	if len(debugScenarios()) == 0 {
		t.Error("a debug build offers no scenarios, so the window has no question to ask")
	}
}
