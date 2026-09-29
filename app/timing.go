package app

// timing.go — the timing instrument's keeper (0.18.0 W14, D-154): M5's time to
// picture, W8.12's radar arms and M6's two halves, as the dashboard measures
// them (modes/tty/timing.go), kept for /debug/counters.
//
// THERE ONLY UNDER WATCHPOST_DEBUG_TIMING=1, the switch M1's launch timer
// already answers to: without it there is no keeper, the dashboard is handed
// no hook, and nothing is measured.

import (
	"os"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty"
)

// timingsKept bounds the log: the newest intervals, enough for a reader
// polling the counters once a minute while the loop ticks a few times a second.
const timingsKept = 1024

// timingRecord is one interval, as the counters say it.
type timingRecord struct {
	Seq     uint64    `json:"seq"` // rising: a poller keeps what it has not seen
	At      time.Time `json:"at"`  // when it was measured: a reader places it in its phase
	Trigger string    `json:"trigger"`
	Event   string    `json:"event"`
	MS      float64   `json:"ms"`
}

// timingLog keeps the newest intervals.
type timingLog struct {
	mu   sync.Mutex
	seq  uint64
	held []timingRecord
}

// newTimingLog is a keeper under WATCHPOST_DEBUG_TIMING=1, and nil otherwise.
func newTimingLog() *timingLog {
	if os.Getenv("WATCHPOST_DEBUG_TIMING") != "1" {
		return nil
	}
	return &timingLog{}
}

// hook is the dashboard's Config.Timed: nil without a keeper.
func (l *timingLog) hook() func(tty.Timing) {
	if l == nil {
		return nil
	}
	return l.note
}

// note keeps one interval, dropping the oldest past timingsKept.
func (l *timingLog) note(t tty.Timing) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seq++
	l.held = append(l.held, timingRecord{Seq: l.seq, At: time.Now().UTC(), Trigger: t.Trigger, Event: t.Event, MS: float64(t.After.Microseconds()) / 1000})
	if len(l.held) > timingsKept {
		l.held = append(l.held[:0:0], l.held[len(l.held)-timingsKept:]...)
	}
}

// last is a copy of the intervals held; nil without a keeper.
func (l *timingLog) last() []timingRecord {
	if l == nil {
		return nil
	}
	return lockedCopy(&l.mu, &l.held)
}
