package app

// radio_queue.go — the watchlist queue and the live-relay dwell: [r] Repeat, advancing, the nearest-station pick. Split from radio.go by the quality pass (Q2, pure move).

import (
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// liveDwell is how long Watchlist mode stays on a live relay before moving
// on (UAT 93): a relay never ends, so one NWR cycle (~5 min) is the
// "broadcast" we let it finish. The one knob; Synth advances at its own end.
const liveDwell = 5 * time.Minute

// armDwell keeps the Watchlist dwell consistent with the mode: a live relay
// under Watchlist counts down once (idempotent while it runs — relay title
// changes must not restart it); any other mode or path cancels it. Callers
// hold no lock.
func (d *radioDeck) armDwell(ref snapshot.LocationRef) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.repeat != tty.RepeatWatchlist || d.mode != "live" {
		d.stopDwell()
		return
	}
	if d.dwell != nil {
		return
	}
	d.dwell = time.AfterFunc(liveDwell, func() { d.advanceQueue(ref) })
}

// stopDwell cancels a pending live advance. Callers hold d.mu.
func (d *radioDeck) stopDwell() {
	if d.dwell != nil {
		d.dwell.Stop()
		d.dwell = nil
	}
}

// advanceQueue tunes the location after cur in the Watchlist queue (UAT
// 93) — wrapping at the end, starting at the top when cur is not a
// favourite. A no-op unless the mode is still Watchlist and the queue has
// somewhere to go.
func (d *radioDeck) advanceQueue(cur snapshot.LocationRef) {
	d.mu.Lock()
	mode, queue, playing := d.repeat, d.queue, d.mode != ""
	d.mu.Unlock()
	if mode != tty.RepeatWatchlist || !playing {
		return // the user stopped, or the mode changed: a late dwell timer must not tune anything
	}
	if next, ok := nextInQueue(queue, cur); ok {
		d.Tune(next)
	}
}

// nextInQueue is the location after cur (by key), wrapping; the first
// entry when cur is not in the queue; nothing for an empty queue.
func nextInQueue(queue []snapshot.LocationRef, cur snapshot.LocationRef) (snapshot.LocationRef, bool) {
	if len(queue) == 0 {
		return snapshot.LocationRef{}, false
	}
	for i, r := range queue {
		if snapshot.Key(r) == snapshot.Key(cur) {
			return queue[(i+1)%len(queue)], true
		}
	}
	return queue[0], true
}

// chooseNearest picks the station Nearest Relay mode plays (UAT 97): the
// resolver lists the covering transmitter first when it is relayed, then
// the nearest relayed ones — so the first with a mount is the answer.
func chooseNearest(stations []stream.Station) (stream.Station, bool) {
	for _, st := range stations {
		if len(st.Mounts) > 0 {
			return st, true
		}
	}
	return stream.Station{}, false
}

// SetRepeat implements tty.Radio (UAT 83/93): One loops the synthesized
// broadcast; Watchlist lets the current cycle end and then advances through
// the queue (a live relay advances after liveDwell); Off plays to the end.
func (d *radioDeck) SetRepeat(mode tty.RepeatMode, watchlist []snapshot.LocationRef) {
	d.mu.Lock()
	d.repeat, d.queue = mode, watchlist
	src, ref := d.source, d.ref
	d.mu.Unlock()
	if src != nil {
		src.Loop(mode == tty.RepeatOne)
	}
	if d.engine.Status().State == player.Playing {
		d.armDwell(ref) // a live relay under Watchlist starts its dwell now; any other mode cancels it
	}
}
