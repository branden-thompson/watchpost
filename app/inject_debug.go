//go:build watchpost_debug

package app

// inject_debug.go — the injection seam (F-21b), PRESENT ONLY IN A DEBUG BUILD.
//
// IT IS BUILD-TAGGED RATHER THAN RUNTIME-GATED, and that is the safety
// decision. A screenshot of a fabricated tornado warning is indistinguishable
// from a real one, so the capability must be ABSENT from a release binary — a
// runtime gate is one config mistake away from being reachable, a build tag is
// not in the file the linker reads.
//
//	go build -tags watchpost_debug ./cmd/watchpost
//
// IT ENTERS WHERE A REAL ALERT ENTERS. The events go in immediately after the
// source fetch and before globalfeed.Active, so an injected alert crosses every
// stage a real one does: the active-window filter, the D5 location tie, the
// severe index, the radius scope, fresh-detection, the rail's plan, the
// Composer and the Reader. An injector that shortcut any of those would
// validate the machinery and say nothing about the wiring — which is the defect
// this whole tool exists to make findable.

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/category"
)

// injectQueue holds what is waiting for the next cycle. It is a TYPE rather
// than two fields on the deck so that the release build can define it as an
// empty struct — otherwise those fields are dead weight there, which P10 reports
// and AP-DEAD-01 forbids.
type injectQueue struct {
	mu  sync.Mutex
	evs []globalfeed.Event
}

// Inject queues events for the next ticker cycle. Safe from any goroutine.
func (t *tickerDeck) Inject(evs ...globalfeed.Event) {
	if t == nil || len(evs) == 0 {
		return
	}
	t.inject.mu.Lock()
	t.inject.evs = append(t.inject.evs, evs...)
	t.inject.mu.Unlock()
	radioDebugLog("inject:queued:" + itoaN(len(evs)))
}

// takeInjected drains the queue into the cycle.
//
// DRAINED, NOT HELD. An injected alert is a one-shot: leaving it in place would
// make it arrive every cycle, and "it keeps re-firing" is not a behaviour any
// real alert has.
func (t *tickerDeck) takeInjected() []globalfeed.Event {
	if t == nil {
		return nil
	}
	return t.inject.take()
}

// take drains the queue. DRAINED, NOT HELD: an injected alert is a one-shot,
// and leaving it in place would make it arrive every cycle — which is not a
// behaviour any real alert has.
func (q *injectQueue) take() []globalfeed.Event {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := q.evs
	q.evs = nil
	return out
}

func itoaN(n int) string {
	if n <= 0 {
		return "0"
	}
	var b []byte
	for n > 0 { // bounded by the digits (P10-02)
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// scenario is one lane's fabricated payload. THE LANE IS THE SUBJECT: what a
// tester wants to exercise is a path through the app, and the product is
// whichever one reaches it.
type scenario struct {
	typ    string
	class  globalfeed.Class
	sev    globalfeed.Severity
	source string
}

// laneScenarios is one payload per lane the feed can produce.
//
// DERIVED FROM globalfeed.FeedLanes, NOT HAND-LISTED (FR-4.2). The window used
// to offer three fixed rows, and the row named for the emergency path injected
// a Tornado Warning — so the one path the operator most wants to exercise was
// the one the window could not reach. A lane added to the feed now arrives here
// as a missing key, loudly, in a test.
//
// THE PAYLOADS ARE NOT CHECKED AGAINST A SECOND COPY OF THE MAPPING. The test
// runs each one through globalfeed.LaneOf — the app's own classifier — and
// fails if it lands anywhere but the lane it is filed under.
func laneScenarios() map[globalfeed.Lane]scenario {
	return map[globalfeed.Lane]scenario{
		category.Emergency: {"Evacuation Immediate", globalfeed.ClassSevereWx, globalfeed.SevRed, "NWS"},
		category.Disasters: {"Earthquake", globalfeed.ClassQuake, globalfeed.SevOrange, "USGS"},
		category.Marine:    {"Tropical Storm", globalfeed.ClassTropical, globalfeed.SevOrange, "NHC"},
		category.Warnings:  {"Tornado Warning", globalfeed.ClassSevereWx, globalfeed.SevRed, "NWS"},
		category.Watches:   {"Tornado Watch", globalfeed.ClassSevereWx, globalfeed.SevYellow, "NWS"},
	}
}

// burstKey is the one scenario that is not a lane: six events at once, which is
// what exercises the head, the lines, the tail and the divert.
const burstKey = "burst"

// scenarioKeyFor is a lane's key — its own short tab word, so the key a
// listener sees in the debug log names the lane it exercised.
func scenarioKeyFor(l globalfeed.Lane) (string, bool) {
	if _, ok := laneScenarios()[l]; !ok {
		return "", false
	}
	return strings.ToLower(category.Of(l).TabShort), true
}

// allScenarioKeys is every key the window can send back.
func allScenarioKeys() []string {
	out := []string{burstKey}
	for _, l := range globalfeed.FeedLanes() { // bounded by the feed's lanes (P10-02)
		if k, ok := scenarioKeyFor(l); ok {
			out = append(out, k)
		}
	}
	return out
}

// laneForKey is the reverse, over the same closed set.
func laneForKey(key string) (globalfeed.Lane, bool) {
	for _, l := range globalfeed.FeedLanes() { // bounded by the feed's lanes (P10-02)
		if k, ok := scenarioKeyFor(l); ok && k == key {
			return l, true
		}
	}
	return category.None, false
}

// debugScenarios are what the ctrl+d window offers: one per lane the feed can
// produce, and a burst.
//
// EACH LABEL NAMES THE PRODUCT IT INJECTS. "An Emergency Order (leads the rail,
// overruns Max)" was true of nothing the scenario did, and no test could have
// caught that, because a label is prose. A label built from the payload cannot
// drift from it.
func debugScenarios() []tty.DebugScenario {
	out := make([]tty.DebugScenario, 0, len(globalfeed.FeedLanes())+1)
	for _, l := range globalfeed.FeedLanes() { // bounded by the feed's lanes (P10-02)
		k, ok := scenarioKeyFor(l)
		if !ok {
			continue
		}
		out = append(out, tty.DebugScenario{
			Label: category.Of(l).Bucket + " - " + laneScenarios()[l].typ,
			Key:   k,
		})
	}
	return append(out, tty.DebugScenario{
		Label: "A burst of six Warnings (head, lines, tail, divert)",
		Key:   burstKey,
	})
}

// injectHook turns a scenario key into fabricated events.
//
// THEY ARE MARKED AS FABRICATED IN THEIR OWN TEXT. A tester holding a
// screenshot must be able to tell, and the id prefix carries into the seen
// store, the severe window and the debug log.
func (lp *livePipelines) injectHook() func(string) {
	return func(key string) {
		lp.mu.Lock()
		tk := lp.ticker
		lp.mu.Unlock()
		if tk == nil {
			return
		}
		evs := injectedEvents(key, time.Now())
		radioDebugLog("inject:" + key)
		tk.Inject(evs...)
	}
}

// testEventLife is how long a fabricated event stays active (FR-4.3).
//
// TWO MINUTES, NOT AN HOUR. They carried an hour, which is thirty times the
// bound the requirement states: a test run at the top of the hour left a
// fabricated tornado warning in the marquee and the severe window for the rest
// of it, and in the seen store for seven days after that.
const testEventLife = 2 * time.Minute

// injectedEvents is what a scenario key fabricates, as a pure function of the
// key and the clock — so what the window offers can be checked without a deck.
//
// AN UNKNOWN KEY FABRICATES NOTHING. The arm that used to catch one produced a
// Tornado Warning identical to the "emergency" scenario's, so the two could not
// be told apart — which is precisely how the emergency scenario went five weeks
// without exercising the emergency path.
func injectedEvents(key string, now time.Time) []globalfeed.Event {
	mk := func(n int, sc scenario) globalfeed.Event {
		return globalfeed.Event{
			ID:         fmt.Sprintf("watchpost-injected-%s-%d-%d", key, now.UnixNano(), n),
			Type:       sc.typ,
			Location:   "Injected Test Location",
			Class:      sc.class,
			Severity:   sc.sev,
			Source:     sc.source,
			At:         now.Add(-time.Minute),
			Until:      now.Add(testEventLife),
			Fabricated: true,
		}
	}
	if key == burstKey {
		evs := make([]globalfeed.Event, 0, 6)
		for i := range 6 {
			evs = append(evs, mk(i, laneScenarios()[category.Warnings]))
		}
		return evs
	}
	l, ok := laneForKey(key)
	if !ok {
		radioDebugLog("inject:unknown:" + key)
		return nil
	}
	return []globalfeed.Event{mk(0, laneScenarios()[l])}
}
