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
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// injectQueue holds what is waiting for the next cycle. It is a TYPE rather
// than two fields on the deck so that the release build can define it as an
// empty struct — otherwise those fields are dead weight there, which P10 reports
// and AP-DEAD-01 forbids.
type injectQueue struct {
	mu    sync.Mutex
	evs   []globalfeed.Event
	ready chan struct{} // buffered 1: "there is something to drain, cycle now"
}

// Inject queues events and ASKS FOR A CYCLE AT ONCE. Safe from any goroutine.
//
// IT USED TO WAIT FOR THE WEATHER (UAT 2026-09-07). The queue is drained by the
// fetch cycle, which runs every two minutes, so an operator who confirmed an
// injection watched nothing happen for up to two minutes and reasonably
// reported that it does not work. Worse than slow: a test event is effective
// for two minutes — the same two minutes — so in the worst case Active drops it
// in the very cycle that would have shown it, and the tool that exists to prove
// the machinery works is the one thing in the app that cannot be trusted.
func (t *tickerDeck) Inject(evs ...globalfeed.Event) {
	if t == nil || len(evs) == 0 {
		return
	}
	t.inject.mu.Lock()
	t.inject.evs = append(t.inject.evs, evs...)
	ready := t.inject.readyLocked()
	t.inject.mu.Unlock()
	select {
	case ready <- struct{}{}:
	default: // a cycle is already asked for; one is enough for any number of events
	}
	radioDebugLog("inject:queued:" + itoaN(len(evs)))
}

// wake is the deck's "cycle now" signal. NIL IN A RELEASE BUILD, where a select
// arm on a nil channel blocks forever and costs nothing — the capability is
// absent there rather than switched off, like everything else in this file.
func (q *injectQueue) wake() <-chan struct{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.readyLocked()
}

// readyLocked returns the channel, making it on first use. The deck is built as
// a zero value in half a dozen places, so the queue cannot rely on a
// constructor.
func (q *injectQueue) readyLocked() chan struct{} {
	if q.ready == nil {
		q.ready = make(chan struct{}, 1)
	}
	return q.ready
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
	// STAMPED WHEN THE CYCLE TAKES IT, NOT WHEN IT WAS QUEUED. The read promises
	// two minutes out loud, and a cycle in flight, a slow fetch or a wake that
	// lands mid-cycle all spend part of that before anyone can see it. The
	// event's life starts where the listener's does.
	now := time.Now()
	for i := range out { // bounded by the queue (P10-02)
		out[i].At, out[i].Until = now.Add(-time.Minute), now.Add(testEventLife)
	}
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
	// label is what the window's Alert Type picker reads. THE OPERATOR'S WORDS,
	// not the product string: "Emergency Evacuation Order" is what the mock
	// asks for, and the product a scenario injects ("Evacuation Immediate") is
	// the feed's name for the same thing.
	label  string
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
		category.Emergency: {"Emergency Evacuation Order", "Evacuation Immediate", globalfeed.ClassSevereWx, globalfeed.SevRed, "NWS"},
		category.Disasters: {"Earthquake", "Earthquake", globalfeed.ClassQuake, globalfeed.SevOrange, "USGS"},
		category.Marine:    {"Tropical Storm", "Tropical Storm", globalfeed.ClassTropical, globalfeed.SevOrange, "NHC"},
		category.Warnings:  {"Tornado Warning", "Tornado Warning", globalfeed.ClassSevereWx, globalfeed.SevRed, "NWS"},
		category.Watches:   {"Tornado Watch", "Tornado Watch", globalfeed.ClassSevereWx, globalfeed.SevYellow, "NWS"},
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
		out = append(out, tty.DebugScenario{Label: laneScenarios()[l].label, Key: k})
	}
	// THE BURST IS AN ALERT TYPE TOO, as far as the picker is concerned: it is
	// the value that exercises the head, the lines, the tail and the divert,
	// which no single alert can.
	return append(out, tty.DebugScenario{Label: "Burst of six Tornado Warnings", Key: burstKey})
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
		evs := injectedEvents(key, time.Now(), lp.testLocation())
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

// testLocation is where a fabricated alert happens: the listener's own first
// watched location (HUM LEAD 2026-09-07).
//
// NOT A PLACEHOLDER STRING. It carried "Injected Test Location" with no point,
// which exercises neither the D5 location tie nor the radius fence — two of the
// stages most likely to be the reason a real alert never reached someone, and
// the two this tool exists to test.
func (lp *livePipelines) testLocation() snapshot.LocationRef {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if len(lp.watchRefs) == 0 {
		return snapshot.LocationRef{}
	}
	return lp.watchRefs[0]
}

// injectedEvents is what a scenario key fabricates, as a pure function of the
// key and the clock — so what the window offers can be checked without a deck.
//
// AN UNKNOWN KEY FABRICATES NOTHING. The arm that used to catch one produced a
// Tornado Warning identical to the "emergency" scenario's, so the two could not
// be told apart — which is precisely how the emergency scenario went five weeks
// without exercising the emergency path.
func injectedEvents(key string, now time.Time, here snapshot.LocationRef) []globalfeed.Event {
	// A LISTENER WITH NO LOCATIONS CAN STILL PRESS ctrl+d, and a blank location
	// on the band reads as a rendering fault rather than as a test.
	where := here.Label
	if where == "" {
		where = "Injected Test Location"
	}
	mk := func(n int, sc scenario) globalfeed.Event {
		return globalfeed.Event{
			ID:         fmt.Sprintf("watchpost-injected-%s-%d-%d", key, now.UnixNano(), n),
			Type:       sc.typ,
			Location:   where,
			Lat:        here.Lat,
			Lon:        here.Lon,
			HasPoint:   here.Label != "",
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
