package lineup

// discard.go — the discard pile (D-35, HUM LEAD 2026-09-09).
//
// A STACK WITH A TRAPDOOR: the newest on top, and the oldest falls through when
// another is set on it. The operator's undo reaches for what they just dropped,
// which is what makes it a stack rather than a queue.
//
// IT IS NOT A TRACK, AND THE REASON IS MEASURED. `stopped()` is
// `lineup.held() == 0` and `held()` counts every card on every track — so a
// discarded card parked in one would mean the schedule NEVER READS AS STOPPED
// and the relay-fault window could never fire. Modelling the pile as a third
// track would have been the same mistake `Track`'s own comment refuses for the
// bed: "modelling it as a third track would invite something to be scheduled
// onto it."
//
// AND THE CAP IS WHY IT CANNOT BECOME ONE BY ACCIDENT. The HUM LEAD's framing
// is the argument: "once it's refused, it's in a final state that is temporally
// unbounded." A 24/7 station accumulates every card it ever dropped unless
// something takes them away, and nothing else would.
//
// WHAT IT HOLDS IS CONTENT, NOT CARDS AWAITING RESURRECTION. The card model
// says "REFUSED, DONE and DISCARDED lead nowhere: a card that could be revived
// is a card that can be read twice", so an undo MINTS A NEW CARD from what is
// here — which is also the HUM LEAD's standing ruling that a live card's origin
// never changes and a copy is how you reuse one.

import "github.com/branden-thompson/watchpost/platform/invariant"

// discardDepth is how many dropped cards the pile keeps.
//
// FIVE, RATIFIED (D-35). It is an UNDO BUFFER, not a history: the operator
// reaches for what they just dropped, under weather pressure, and a pile deep
// enough to browse is a pile deep enough to pick the wrong card out of. The
// number is the HUM LEAD's; what this comment owes is the reason it is small.
const discardDepth = 5

// discard puts a card on the pile and drops the oldest through the trapdoor.
//
// A DELIBERATE REMOVAL ONLY. A card that FAILED is not a card that was dropped:
// a routed decline is the schedule routing around a fault (DR-21), and putting
// those here would fill the operator's undo with things they never did — one
// per rotation turn on some paths. The callers are the staleness drop, where
// reading the card would assert something untrue, and the operator's own DROP
// when it arrives.
func (l Lineup) discard(c Card) Lineup {
	if err := invariant.Check(c.ID != "", "a discarded card is still a card, and carries its identity"); err != nil {
		return l
	}
	out := l.clone()
	// NEWEST FIRST, so the top of the stack is index 0 and the console's undo
	// needs no arithmetic to find it.
	out.discarded = append([]Card{c}, out.discarded...)
	if len(out.discarded) > discardDepth {
		out.discarded = out.discarded[:discardDepth]
	}
	if err := invariant.Check(len(out.discarded) <= discardDepth, "the pile never grows past its cap"); err != nil {
		return l
	}
	return out
}

// Discarded is the pile, newest first — what the console offers as undo.
//
// A COPY, like Cards: a reader may hold it as long as it likes, and the
// schedule's own pile cannot be edited through it.
func (l Lineup) Discarded() []Card {
	return append([]Card(nil), l.discarded...)
}

// takeDiscarded lifts one card off the pile, by identity.
//
// IT REMOVES IT. An undo that left the entry behind would let a second press
// queue a second copy of the same card, which is the read-twice defect arriving
// through the recovery control rather than through the schedule.
func (l Lineup) takeDiscarded(id string) (Card, Lineup, bool) {
	for i, c := range l.discarded { // bounded by the cap (P10-02)
		if c.ID != id {
			continue
		}
		out := l.clone()
		out.discarded = append(out.discarded[:i], out.discarded[i+1:]...)
		return c, out, true
	}
	return Card{}, l, false
}

// trackFor is the lane a slot belongs to, and it is the ONE owner of that rule.
//
// DERIVED RATHER THAN REMEMBERED. The pile could have stored the track each
// card came from, and then a restore would depend on a field nobody else keeps
// in step. What lane a card belongs to is a property of WHAT IT IS: a takeover
// is the priority rail's, and everything else — a report, a transition — is the
// programme's.
//
// IT IS ALSO WHY `Moved` CANNOT CHANGE TRACKS. A card that could hop lanes
// would let an operator schedule a hazard as programming, or a report as a
// takeover.
func trackFor(s Slot) Track {
	if s == BreakingAlert {
		return AlertRail
	}
	return MainTrack
}
