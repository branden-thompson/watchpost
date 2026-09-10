package lineup

// retry.go — how long a card that FAILED sits out before it is offered again.
//
// THE DEFECT THIS EXISTS FOR (HUM LEAD, UAT 2026-09-10): "the lineup is FLYING
// through locations rapidly even on standby — so it seems like cards are
// constantly getting discarded and proposed / accepted." It was, and the loop
// had nothing in it to slow it down:
//
//	decline → Failed{Routed} → the card is discarded → the step PUBLISHES →
//	the publish executor asks the producer to top the line-up off → the producer
//	offers the same watchlist → ReadID gives the failed location the identity it
//	had a microsecond ago → admit → build → the same fault → decline
//
// at pump speed, for as long as the fault lasts. Every executor refusal reaches
// it: a muted listener, a report that would not compose, a script that rendered
// nothing to say. The executor's comment says the work "will be offered again"
// on the producer's "NEXT CYCLE", which was the right intent — what nobody
// noticed is that a publish IS a cycle, so "next" meant "now".
//
// THE RATE LIMIT LIVES ON THE DIRECTOR, not on the producer, because it is a
// decision about the SCHEDULE. The Director already refuses a duplicate identity
// and already weighs how long it has been since a kind of card was read; "this
// one just failed" is the same sort of fact, and a producer holding it would be
// a second author of what the line-up may contain (D-40).
//
// IT IS A RATE LIMIT AND NOT A BLACKLIST. A fault that clears must not silence a
// location for the rest of the run, so the memory is a TIME, not a flag.

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

const (
	// retryAfter is how long a location that failed to reach the air waits
	// before the Director will schedule it again.
	//
	// FIVE MINUTES, AND THE NUMBER IS THE HUM LEAD'S TO RULE. What it must do
	// is stop a busy loop, and any number above zero does that; what it trades
	// is how quickly a TRANSIENT fault recovers against how often a PERSISTENT
	// one is retried for nothing. Five minutes is the dwell the station already
	// uses for its rotation, so it is a pace this project has reasoned about
	// once rather than a number invented here. The rest of the watchlist keeps
	// filling the depth meanwhile, so a location sitting out never leaves the
	// line-up short.
	retryAfter = 5 * time.Minute

	// retryMemory is how many refused refs the Director remembers.
	//
	// FIXED, FOR D-48'S REASON, WHICH WAS RULED: "read history is too
	// overweight and violates our 'done/discarded cards can pile up' concern."
	// A log of failures is a log. This is a ring — oldest evicted — so it
	// cannot grow, and there is nothing to cap, own or clear at runtime.
	//
	// THIRTY-TWO IS THREE TIMES THE DEEPEST LINE-UP the console can show. If a
	// station somehow fails more distinct locations than that inside one
	// cool-off, the oldest is forgotten and retried early — which is the
	// behaviour that existed before this file, for one card, rather than a new
	// failure of its own.
	retryMemory = 32
)

// declineNote is one refused ref and when it was refused.
type declineNote struct {
	ref string
	at  time.Time
}

// noteDeclined records that a ref's card left the schedule without reaching the
// air, so the Director can keep it out of the next few offers.
//
// AN EMPTY REF IS NOT RECORDED. A card with no subject cannot be proposed again
// by ref, so remembering it would spend a slot of the ring on nothing.
func (d Director) noteDeclined(ref string) Director {
	if ref == "" {
		return d
	}
	if err := invariant.Check(!d.now.IsZero(), "the clock was set before a decline was timed"); err != nil {
		return d
	}
	// ONE ENTRY PER REF. A location that fails repeatedly must move its own
	// timestamp forward rather than fill the ring with copies of itself — which
	// would evict every OTHER location's cool-off and let them all back in.
	out := append([]declineNote(nil), d.declined...)
	for i := range out { // bounded by the ring (P10-02)
		if out[i].ref == ref {
			out[i].at = d.now
			d.declined = out
			return d
		}
	}
	out = append(out, declineNote{ref: ref, at: d.now})
	if len(out) > retryMemory {
		out = out[len(out)-retryMemory:]
	}
	d.declined = out
	return d
}

// sittingOut reports whether a ref failed recently enough that the Director
// should leave it alone.
//
// IT DEGRADES TO FALSE, like every other term the Director weighs (D-48): a
// clock that was never set, or a ref never refused, answers "go ahead". A rate
// limit that could not answer must never be the reason a station goes quiet.
func (d Director) sittingOut(ref string) bool {
	if ref == "" || d.now.IsZero() {
		return false
	}
	for _, n := range d.declined { // bounded by the ring (P10-02)
		if n.ref == ref {
			return d.now.Sub(n.at) < retryAfter
		}
	}
	return false
}
