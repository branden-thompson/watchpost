// Package debounce holds the rule for "wait until the operator stops typing".
//
// WHY IT IS A PACKAGE AND NOT A FIELD ON ONE WINDOW (HUM LEAD, 2026-09-14):
// "let's ensure we build the debounce as a helper that can be applied to *any*
// user input we want in the future, or retrofit into existing ones later to
// help improve performance." The first use is the location fields, whose check
// can reach the network; the shape is the same for any input whose answer costs
// more than a keystroke.
//
// IT IS FRAMEWORK-FREE ON PURPOSE. Nothing here knows about bubbletea, a timer,
// or a surface — the caller owns the clock. What lives here is the part that is
// easy to get wrong and identical everywhere: WHICH answer is still the one the
// operator is waiting for.
package debounce

import "time"

// Pause is how long a field waits after the last keystroke before it asks.
//
// 300ms, RULED (HUM LEAD, 2026-09-14): "Types 'Rainbo' / waits 300ms / *then*
// we can do a resolve check." Long enough that ordinary typing never reaches
// the resolver, short enough that a pause reads as an answer arriving rather
// than as a lag.
const Pause = 300 * time.Millisecond

// Gate is one input's debounce. The zero value is an untouched field.
//
// A SEQUENCE NUMBER, NOT A TIMER HANDLE. Cancelling a pending timer is the
// obvious design and it is the one that goes wrong: the cancel races the fire,
// and an answer already in flight has no timer left to cancel at all. Counting
// edits instead makes staleness a COMPARISON — every pause and every answer
// carries the sequence it was asked for, and anything that does not match the
// current one is discarded on arrival. Nothing has to be stopped, so nothing
// can fail to stop.
type Gate struct {
	seq     int
	settled bool
}

// Seq is the sequence a pause or an answer must carry to still be wanted.
func (g Gate) Seq() int { return g.seq }

// Edit records a keystroke: every pause and answer outstanding is now stale,
// and whatever the field last knew is no longer about what it holds.
func (g Gate) Edit() Gate { return Gate{seq: g.seq + 1} }

// Admits reports whether a pause or an answer carrying seq is still the current
// one. An answer for an older sequence is an answer to a question the operator
// has already changed.
func (g Gate) Admits(seq int) bool { return seq == g.seq }

// Settle records that an answer arrived for seq, and reports whether it was
// wanted. A refused answer leaves the gate exactly as it was.
func (g Gate) Settle(seq int) (Gate, bool) {
	if !g.Admits(seq) {
		return g, false
	}
	g.settled = true
	return g, true
}

// Settled reports whether the field has an answer for what it currently holds.
// False covers both "not asked yet" and "asked, still waiting" — which are the
// same thing to a window deciding what to draw: it does not know.
func (g Gate) Settled() bool { return g.settled }
