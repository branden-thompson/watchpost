package tty

// map_timing.go — the map's half of the host's timing instrument (0.18.0 W14,
// D-154): M5's time to picture, W8.12's radar arms, and how late the map's
// clock lands (M6), on the map model (0.19.6 design MS-1).
//
// OFF UNLESS THE APP SUPPLIES Hooks.Timed, which it does only under
// WATCHPOST_DEBUG_TIMING=1: every site below returns at its first line when it
// is nil, and the frame is the same either way
// (TestTheInstrumentChangesNothingItMeasures). It measures the picture the
// listener sees, from inside the host, because the screen does not say "whole"
// in a way a script can time: the status line just stops speaking.

import (
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// mapClock is the map's running clock: when the listener last asked for
// something, what it was, and which of its moments have been said.
type mapClock struct {
	from   time.Time
	what   string
	seen   uint8
	feedAt uint64 // the alerts' generation when the clock started: M5 waits for a newer one drawn
}

const (
	seenComplete uint8 = 1 << iota
	seenM5
	seenSettled
	seenFeed
	seenRadar
	seenTemp
)

// answerSeen is each answer's bit: an answer is timed once a kind per ask.
var answerSeen = map[string]uint8{"answered:feed": seenFeed, "answered:radar": seenRadar, "answered:temp": seenTemp}

// timeFrom starts the map's clock at an ask: g, a map key, a switch.
func (mm mapModel) timeFrom(what string, e mapEnv) mapModel {
	if mm.mapHooks.Timed == nil {
		return mm
	}
	mm.mapPane.clock = mapClock{from: e.Now(), what: what, feedAt: mm.mapPane.feedGen}
	return mm
}

// timed says an event against the running clock. AN ASK STOPS LISTENING
// ONCE SETTLED, and an answer is timed once a kind: a trigger that outlives
// its ask (space, for the loop) would otherwise time every later refresh
// from the key press (W14).
func (mm mapModel) timed(event string, e mapEnv) mapModel {
	c := &mm.mapPane.clock
	if mm.mapHooks.Timed == nil || c.from.IsZero() || c.seen&seenSettled != 0 {
		return mm
	}
	if bit, ok := answerSeen[event]; ok {
		if c.seen&bit != 0 {
			return mm
		}
		c.seen |= bit
	}
	mm.mapHooks.Timed(Timing{Trigger: c.what, Event: event, After: e.Now().Sub(c.from)})
	return mm
}

// timeDraw says, once each per ask:
//   - complete: the library's first Complete frame - the basemap whole;
//   - m5: M5 as D-46 defines it - the first complete frame holding every
//     alert's area: complete, the view still, no work waiting, and an answer
//     asked since the ask drawn (an ask that asks no alerts has no m5);
//   - settled: nothing left to come - the view still, no work, radar,
//     temperature or alerts in flight.
func (mm mapModel) timeDraw(e mapEnv) mapModel {
	if mm.mapHooks.Timed == nil || mm.mapPane.clock.from.IsZero() {
		return mm
	}
	p := &mm.mapPane
	complete := p.status == tuimaps.Complete
	if complete && p.clock.seen&seenComplete == 0 {
		mm = mm.timed("complete", e)
		p = &mm.mapPane
		p.clock.seen |= seenComplete
	}
	// A move asks for its view only once it has stood still (D-66): until its
	// settle tick has asked, nothing about the new view is in.
	still := p.viewAsked == p.viewGen
	// M5 is the alerts asked since the ask, drawn - not the newest data: with
	// new data landing while the feed runs, the answer drawn is one generation
	// behind, and holds what was asked (D-157). Settled is nothing in flight
	// and nothing wanted again.
	alerts, layers := p.lanes[laneAlerts], p.lanes[laneLayers]
	feedIn := !alerts.busy && !alerts.again && !layers.busy && !layers.again
	if complete && still && !p.pending && alerts.applied > p.clock.feedAt && p.clock.seen&seenM5 == 0 {
		mm = mm.timed("m5", e)
		p = &mm.mapPane
		p.clock.seen |= seenM5
	}
	if complete && still && !p.pending && !p.radarBusy && !p.tempBusy && feedIn && p.clock.seen&seenSettled == 0 {
		mm = mm.timed("settled", e)
		p = &mm.mapPane
		p.clock.seen |= seenSettled
	}
	return mm
}

// timeBelowFloor says a draw fell under the map's floor and showed the notice
// in its place: the render check a workload run reads (W14 - "everything
// that's supposed to render, ACTUALLY renders"). It carries no interval.
func (mm mapModel) timeBelowFloor() {
	if mm.mapHooks.Timed != nil {
		mm.mapHooks.Timed(Timing{Trigger: "render", Event: "below-floor"})
	}
}

// timeTick says how late the map's clock landed against the moment the
// library asked for (M6: the loop's pace is the ticks').
func (mm mapModel) timeTick(at time.Time, e mapEnv) {
	if mm.mapHooks.Timed == nil {
		return
	}
	mm.mapHooks.Timed(Timing{Trigger: "tick", Event: "tick", After: e.Now().Sub(at)})
}
