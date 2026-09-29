package tty

// timing.go — the host's own timing instrument (0.18.0 W14, D-154): M5's time
// to picture and W8.12's radar arms, and M6's two halves - how late the map's
// clock lands, and a key to the frame it produced.
//
// OFF UNLESS THE APP SUPPLIES Config.Timed, which it does only under
// WATCHPOST_DEBUG_TIMING=1: every site below returns at its first line when it
// is nil, and the frame is the same either way
// (TestTheInstrumentChangesNothingItMeasures). It measures the picture the
// listener sees, from inside the host, because the screen does not say "whole"
// in a way a script can time: the status line just stops speaking.

import (
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// Timing is one measured interval: what started the clock, what it reached,
// and how long that took.
type Timing struct {
	Trigger string // "open", a map action's name, "overlay", "tick", "key"
	Event   string // "answered:feed|radar|temp", "complete", "settled", "tick", "key"
	After   time.Duration
}

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
func (d Dashboard) timeFrom(what string) Dashboard {
	if d.cfg.Timed == nil {
		return d
	}
	d.mapPane.clock = mapClock{from: d.now(), what: what, feedAt: d.mapPane.feedGen}
	return d
}

// timed says an event against the running clock. AN ASK STOPS LISTENING
// ONCE SETTLED, and an answer is timed once a kind: a trigger that outlives
// its ask (space, for the loop) otherwise timed every later refresh from the
// key press - the baseline's 300 s "answers" (W14).
func (d Dashboard) timed(event string) Dashboard {
	c := &d.mapPane.clock
	if d.cfg.Timed == nil || c.from.IsZero() || c.seen&seenSettled != 0 {
		return d
	}
	if bit, ok := answerSeen[event]; ok {
		if c.seen&bit != 0 {
			return d
		}
		c.seen |= bit
	}
	d.cfg.Timed(Timing{Trigger: c.what, Event: event, After: d.now().Sub(c.from)})
	return d
}

// timeDraw says, once each per ask:
//   - complete: the library's first Complete frame - the basemap whole;
//   - m5: M5 as D-46 defines it - the first complete frame holding every
//     alert's area: complete, no work waiting, and the alerts asked since the
//     ask drawn (an ask that asks no alerts has no m5);
//   - settled: nothing left to come - the view still, no work, radar,
//     temperature or alerts in flight.
func (d Dashboard) timeDraw() Dashboard {
	if d.cfg.Timed == nil || d.mapPane.clock.from.IsZero() {
		return d
	}
	p := &d.mapPane
	complete := p.status == tuimaps.Complete
	if complete && p.clock.seen&seenComplete == 0 {
		d = d.timed("complete")
		p = &d.mapPane
		p.clock.seen |= seenComplete
	}
	// A move asks for its view only once it has stood still (D-66): until its
	// settle tick has asked, nothing about the new view is in.
	still := p.viewAsked == p.viewGen
	feedIn := p.feedApplied == p.feedGen
	if complete && still && !p.pending && feedIn && p.feedApplied > p.clock.feedAt && p.clock.seen&seenM5 == 0 {
		d = d.timed("m5")
		p = &d.mapPane
		p.clock.seen |= seenM5
	}
	if complete && still && !p.pending && !p.radarBusy && !p.tempBusy && feedIn && p.clock.seen&seenSettled == 0 {
		d = d.timed("settled")
		p = &d.mapPane
		p.clock.seen |= seenSettled
	}
	return d
}

// timeTick says how late the map's clock landed against the moment the
// library asked for (M6: the loop's pace is the ticks').
func (d Dashboard) timeTick(at time.Time) {
	if d.cfg.Timed == nil {
		return
	}
	d.cfg.Timed(Timing{Trigger: "tick", Event: "tick", After: d.now().Sub(at)})
}

// keyClock is the Router's: when the last key arrived, until the frame it
// produced is drawn. A pointer, so View - a value method - can clear it.
type keyClock struct{ at time.Time }

// keyArrived starts the key's clock.
func (r Router) keyArrived() {
	if r.observer.cfg.Timed != nil && r.keyTime != nil {
		r.keyTime.at = time.Now()
	}
}

// frameDrawn says the key's time to its frame, once (M6's second half).
func (r Router) frameDrawn() {
	if r.keyTime == nil || r.keyTime.at.IsZero() || r.observer.cfg.Timed == nil {
		return
	}
	r.observer.cfg.Timed(Timing{Trigger: "key", Event: "key", After: time.Since(r.keyTime.at)})
	r.keyTime.at = time.Time{}
}
