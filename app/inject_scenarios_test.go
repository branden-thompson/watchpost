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
	"context"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

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

// WHAT THE WINDOW SENDS BACK IS WHAT THE HOOK INJECTS (F-21b).
//
// THE PIPELINE TEST CANNOT SEE THIS. TestAnInjectedAlertCrossesTheWholePipeline
// — "the test the whole tool stands on" — calls deck.Inject with a
// hand-built event, so it proves the SEAM and says nothing about the path from
// the window's key to that call. A key the hook does not recognise, or a hook
// that returns before it queues anything, is invisible to it: the tool reports
// that everything works and pressing the button does nothing at all.
//
// UAT 2026-09-07: the confirmation was accepted and no alert arrived — no
// audio, no takeover, nothing in [w].
func TestTheHookInjectsWhatTheWindowSendsBack(t *testing.T) {
	for _, sc := range debugScenarios() {
		deck := &tickerDeck{}
		lp := &livePipelines{ticker: deck}
		lp.injectHook()(sc.Key)
		got := deck.takeInjected()
		if len(got) == 0 {
			t.Errorf("the window's %q scenario (%s) queued NOTHING: the operator confirmed an "+
				"injection and nothing happened", sc.Key, sc.Label)
			continue
		}
		for _, e := range got {
			if !e.Fabricated || e.Type == "" {
				t.Errorf("the %q scenario queued %+v", sc.Key, e)
			}
		}
	}
}

// AN INJECTION RUNS A CYCLE AT ONCE (UAT 2026-09-07).
//
// THE DIAGNOSTIC WAITED FOR THE WEATHER. Injected events are drained by the
// ticker's fetch cycle, which runs every two minutes — so an operator who
// confirmed an injection watched nothing happen for up to two minutes and
// reasonably reported that it does not work.
//
// AND IT IS WORSE THAN SLOW. A test event is effective for two minutes
// (FR-4.3), the same two minutes, so in the worst case globalfeed.Active drops
// it in the very cycle that would have shown it: the injection then does
// nothing at all, silently, and the tool that exists to prove the machinery
// works is the one thing in the app that cannot be trusted.
func TestAnInjectionRunsACycleAtOnce(t *testing.T) {
	deck := &tickerDeck{}
	select {
	case <-deck.inject.wake():
		t.Fatal("the deck asked for a cycle before anything was injected")
	default:
	}
	deck.Inject(globalfeed.Event{ID: "x", Type: "Tornado Warning"})
	select {
	case <-deck.inject.wake():
	default:
		t.Error("an injection did not ask for a cycle: the operator waits up to two minutes for " +
			"the fetch that drains it, and the event may expire first")
	}
}

// AND IT IS EFFECTIVE FOR TWO MINUTES FROM WHEN IT ARRIVES, not from when it
// was queued. A cycle in flight, a slow fetch, or a wake that lands mid-cycle
// all spend part of the event's life before anyone can see it — and the read
// promises two minutes out loud.
func TestAQueuedEventIsFreshWhenTheCycleTakesIt(t *testing.T) {
	deck := &tickerDeck{}
	stale := time.Now().Add(-90 * time.Second)
	deck.Inject(globalfeed.Event{ID: "x", Type: "Tornado Warning", Fabricated: true,
		At: stale.Add(-time.Minute), Until: stale.Add(testEventLife)})

	got := deck.takeInjected()
	if len(got) != 1 {
		t.Fatalf("queued 1, took %d", len(got))
	}
	if left := time.Until(got[0].Until); left < testEventLife-time.Second {
		t.Errorf("the event arrives with %s left of its %s: it was stamped when it was queued, "+
			"not when the cycle took it", left.Round(time.Second), testEventLife)
	}
	if got[0].At.After(time.Now()) {
		t.Errorf("the event was declared in the future: %s", got[0].At)
	}
}

// EVERY SCENARIO THE WINDOW OFFERS REACHES THE [w] WINDOW (UAT 2026-09-07).
//
// "STILL NO ENTRIES IN [w]" is the report this covers, and the path it covers
// is the one no test had: the window's own KEY, through the real hook, through
// a real cycle, into the severe index. The pipeline test hands a hand-built
// event to deck.Inject — so a scenario whose payload the index cannot classify,
// or whose key the hook does not know, was invisible to every gate in the repo
// while the tool reported that everything works.
func TestEveryScenarioReachesTheSevereIndex(t *testing.T) {
	for _, sc := range debugScenarios() {
		t.Run(sc.Key, func(t *testing.T) {
			sev := newSevereDeck(func(tea.Msg) {})
			deck := &tickerDeck{
				send:   func(tea.Msg) {},
				muted:  &atomic.Bool{},
				mc:     newMastercontrol(nil, func(tea.Msg) {}),
				voice:  testDirector(&scriptVoice{}, nil),
				seen:   loadSeen(t.TempDir(), time.Hour),
				severe: sev,
				watch:  func() []snapshot.LocationRef { return []snapshot.LocationRef{testHere} },
			}
			deck.warm.Store(true)
			lp := &livePipelines{ticker: deck, watchRefs: []snapshot.LocationRef{testHere}}

			lp.injectHook()(sc.Key)
			deck.cycle(context.Background())

			rows, _ := sev.currentRows()
			if len(rows) == 0 {
				t.Fatalf("%q (%s) reached no row in the severe index: [w] is empty after an "+
					"injection the operator confirmed", sc.Key, sc.Label)
			}
			for _, r := range rows {
				if !r.Test {
					t.Errorf("%q produced an UNMARKED row in [w]: %s", sc.Key, r.Product)
				}
			}
		})
	}
}
