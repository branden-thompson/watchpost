package tty

// timing.go — the host's own timing instrument (0.18.0 W14, D-154): what a
// measured interval is, and the Router's half of M6 - a key to the frame it
// produced. The map's half, M5's time to picture and W8.12's radar arms, is
// map_timing.go's (0.19.6 design MS-1).
//
// OFF UNLESS THE APP SUPPLIES Config.FrameTimed, which it does only under
// WATCHPOST_DEBUG_TIMING=1: every site below returns at its first line when it
// is nil, and the frame is the same either way
// (TestTheInstrumentChangesNothingItMeasures).

import "time"

// Timing is one measured interval: what started the clock, what it reached,
// and how long that took.
type Timing struct {
	Trigger string // "open", a map action's name, "overlay", "tick", "key"
	Event   string // what it reached: "answered:<lane>", "complete", "m5", "settled", "tick", "key" and the rest perf-measurement.md lists
	After   time.Duration
}

// keyClock is the Router's: when the last key arrived, until the frame it
// produced is drawn - and the frames drawn, a hundred at a time, for the
// frame rate (W14). A pointer, so View - a value method - can keep them.
type keyClock struct {
	at      time.Time
	frames  int
	hundred time.Time // when the current hundred began
}

// keyArrived starts the key's clock.
func (r Router) keyArrived() {
	if r.observer.cfg.FrameTimed != nil && r.keyTime != nil {
		r.keyTime.at = time.Now()
	}
}

// frameDrawn says the key's time to its frame, once (M6's second half), and
// every hundredth frame how long the hundred took - the frame rate.
func (r Router) frameDrawn() {
	if r.keyTime == nil || r.observer.cfg.FrameTimed == nil {
		return
	}
	now := time.Now()
	if r.keyTime.frames == 0 {
		r.keyTime.hundred = now
	}
	if r.keyTime.frames++; r.keyTime.frames == 100 {
		r.observer.cfg.FrameTimed(Timing{Trigger: "frame", Event: "frames100", After: max(now.Sub(r.keyTime.hundred), time.Nanosecond)})
		r.keyTime.frames = 0
	}
	if r.keyTime.at.IsZero() {
		return
	}
	r.observer.cfg.FrameTimed(Timing{Trigger: "key", Event: "key", After: now.Sub(r.keyTime.at)})
	r.keyTime.at = time.Time{}
}
