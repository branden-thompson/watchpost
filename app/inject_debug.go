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
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
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

// debugScenarios are what the ctrl+d window offers. Deliberately FEW and
// deliberately DIFFERENT: one alert reads as a single event with its own tail, a
// burst exercises the head/lines/tail structure and the Max bound, and an
// emergency order leads the rail whatever else is queued.
func debugScenarios() []tty.DebugScenario {
	return []tty.DebugScenario{
		{Label: "One Tornado Warning (single read)", Key: "one"},
		{Label: "A burst of six alerts (head, lines, tail, divert)", Key: "burst"},
		{Label: "An Emergency Order (leads the rail, overruns Max)", Key: "emergency"},
	}
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
		now := time.Now()
		mk := func(n int, typ string, sev globalfeed.Severity) globalfeed.Event {
			return globalfeed.Event{
				ID:       fmt.Sprintf("watchpost-injected-%s-%d-%d", key, now.UnixNano(), n),
				Type:     typ,
				Location: "Injected Test Location",
				Class:    globalfeed.ClassSevereWx,
				Severity: sev,
				Source:   "NWS",
				At:       now.Add(-time.Minute),
				Until:    now.Add(time.Hour),
			}
		}
		var evs []globalfeed.Event
		switch key {
		case "burst":
			for i := range 6 {
				evs = append(evs, mk(i, "Flood Advisory", globalfeed.SevOrange))
			}
		case "emergency":
			evs = append(evs, mk(0, "Tornado Warning", globalfeed.SevRed))
		default:
			evs = append(evs, mk(0, "Tornado Warning", globalfeed.SevRed))
		}
		radioDebugLog("inject:" + key)
		tk.Inject(evs...)
	}
}
