package lineup

// cadence.go — what the Director remembers about what it has READ (D-48, F-76).
//
// THE PROBLEM: the Director had no memory at all. Its state is the current
// schedule and the current clock; a card that finishes is REMOVED outright. So
// "it's been six minutes since we've done a location report" — the HUM LEAD's
// own tie-break in D-47 — was a question nothing could answer.
//
// THE SHAPE IS RULED, AND IT IS THE SMALL ONE:
//
//	"it's bounded by the types of cards/reports we have (so it's bounded) and
//	it's just a 'since last read of this type' — READ HISTORY IS TOO OVERWEIGHT
//	and violates our 'done/discarded cards can pile up' concern."
//
// So it is ONE TIMESTAMP PER SLOT, in an array bounded by the registry. It
// cannot grow, there is nothing to cap, nothing to evict, and no second owner:
// a log would have wanted all four.
//
// AND IT MUST BE ABLE TO FAIL WITHOUT TAKING THE DIRECTOR WITH IT:
//
//	"architect this in a way where the Director can still function and make
//	other evaluations in the event this particular stack fails or is somehow
//	disabled (maybe a broadcaster specific setting — where the Operator does /
//	does not want 'last read' to be a factor in Director prioritization)."
//
// That is why the term is ADDITIVE AND SEPARABLE rather than woven through the
// comparison: `overdue` answers zero for every slot when the operator has turned
// it off, when nothing has been read yet, and when the slot is outside the
// registry. A term that answers the same for everything DISCRIMINATES NOTHING,
// so the ranking falls straight through to what it used before this existed.
// Failing soft is a property of the arithmetic here, not of a branch someone has
// to remember to write.

import (
	"math"
	"time"
)

// neverRead is the overdue-ness of a kind the station has NEVER put out.
//
// THE LARGEST THERE IS, not zero, and a test caught it being zero. "We have
// never done one" is the strongest possible case that one is due — stronger
// than any elapsed time — which is exactly the HUM LEAD's example read to its
// conclusion: credits went out six minutes ago, a location report has not gone
// out at all, and the location report takes the slot.
//
// IT DOES NOT BREAK THE COLD START, because at a cold start EVERY kind is
// never-read: they all answer this, the term discriminates between none of
// them, and the order falls through to the watchlist.
const neverRead = time.Duration(math.MaxInt64)

// noteRead records that a card of this kind has just been read in full.
//
// READ, NOT REMOVED. Only a card that reached DONE was actually spoken; a
// discarded one was taken away precisely so the listener would NOT hear it, and
// counting it would tell the Director the station had covered something it never
// did. The state table makes DONE reachable only from ON AIR, so the caller's
// `to == Done` is the whole test.
func (d Director) noteRead(s Slot) Director {
	if s < 0 || s >= numSlots {
		return d // a card outside the registry teaches the Director nothing
	}
	d.lastRead[s] = d.now
	return d
}

// lastReadOf is when a kind was last read in full, and whether it ever was.
//
// THE ZERO TIME IS "NEVER", not "long ago". The distinction matters at a cold
// start, where every slot is unread: treating that as infinitely overdue would
// have the term discriminate on a difference that does not exist yet.
func (d Director) lastReadOf(s Slot) (time.Time, bool) {
	if s < 0 || s >= numSlots {
		return time.Time{}, false
	}
	at := d.lastRead[s]
	return at, !at.IsZero()
}

// overdue is how long this kind has gone unread — the cadence term, and the ONE
// place it is computed.
//
// IT ANSWERS THE SAME FOR EVERYTHING ON EVERY DEGRADED PATH, and that is the
// whole degradation design — zero when the operator has switched it off or the
// slot is outside the registry, `neverRead` when nothing of that kind has gone
// out yet. A term equal for every candidate DISCRIMINATES BETWEEN NONE OF THEM,
// so the ranking falls through to the watchlist, which is exactly what it ranked
// by before this term existed. Failing soft is a property of the arithmetic, not
// of a branch anyone has to remember to write.
func (d Director) overdue(s Slot) time.Duration {
	if !d.settings.WeighLastRead {
		return 0 // the operator does not want this weighed (D-48)
	}
	// A CORRUPT SLOT COSTS THE DIRECTOR ITS OPINION, NOT ITS ABILITY TO CHOOSE.
	// Checked HERE and not left to lastReadOf, which cannot tell the two apart:
	// it answers `false` for "never read" and for "not a slot", and those want
	// OPPOSITE answers — maximally due, and no view at all.
	if s < 0 || s >= numSlots {
		return 0
	}
	at, ever := d.lastReadOf(s)
	if !ever {
		return neverRead // nothing has ever put this out; nothing is more due
	}
	if d.now.Before(at) {
		return 0 // a clock that went backwards is not a reason to reorder the station
	}
	return d.now.Sub(at)
}
