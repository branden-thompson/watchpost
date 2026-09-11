package lineup

// stale.go — PD-3: an interruption that outlasts the report's validity.
//
// A card's words are built as it nears the air (DR-7), and then it waits. If it
// waits long enough the words stop being true — the observation they quote has
// been superseded, the alert they read may have been cancelled — and reading
// them is worse than not reading them, because a station that says something
// false with confidence is the failure this whole release is about.
//
// OBSERVER CANNOT REACH THIS. The alert rail's drain is bounded by Max (5) plus
// an emergency overrun, which at ~8 s a read is well under two minutes, and no
// report goes stale in two minutes. It becomes reachable in Broadcaster, where
// an operator can hold the rail. It is implemented here anyway because the
// alternative is that Broadcaster inherits a latent stale-read on the SAFETY
// path and a listener is the one who finds it (PD-3).

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// StaleAfter is how long a built card's words stay good.
//
// FIFTEEN MINUTES, RATIFIED BY THE HUM LEAD 2026-09-01. It sits far beyond any
// drain Observer can produce, so it never fires there, while bounding the
// Broadcaster case. NWS observations update hourly, so fifteen minutes is
// conservative against the data's own cadence.
//
// A constant rather than a Setting: it is a safety bound, not a preference, and
// nobody asked for a knob. If the Broadcaster surface ever needs to show or move
// it, it moves to Settings then — one owner either way.
const StaleAfter = 15 * time.Minute

// staleTransitionText is what the listener hears in place of the dropped read.
//
// They must be TOLD. A read that simply vanishes is indistinguishable from the
// station breaking, which is the impression the transition exists to prevent.
const staleTransitionText = "That report is out of date and has been dropped."

// staleTransitionID is the transition's card ID. One at a time is all that can
// exist: it is proposed only when the air is free, and it holds the air until it
// is read, so a second can never be proposed while the first is outstanding.
const staleTransitionID = "stale-transition"

// dropStale discards every STANDBY card whose words have gone off, and returns a
// transition card to say so when it dropped any.
//
// IT RUNS WHERE THE AIR IS TAKEN, not on a timer. Staleness is a question about
// a card that is ABOUT TO BE READ; a card sitting in standby while the
// programme is stopped is not stale, it is waiting, and judging it early would
// discard a schedule the listener has not resumed yet.
func (d Director) dropStale() (Director, Card, bool) {
	dropped := false
	// Bounded by the schedule: each pass discards one card, and a discarded
	// card is never held again (P10-02).
	for range d.lineup.held() {
		card, ok := d.firstStale()
		if !ok {
			break
		}
		gone, err := card.To(Discarded)
		if err != nil {
			return d, Card{}, false
		}
		if err := invariant.Check(gone.State == Discarded, "a stale card leaves the schedule discarded"); err != nil {
			return d, Card{}, false
		}
		next, err := d.lineup.Remove(gone.ID)
		if err != nil {
			return d, Card{}, false
		}
		if err := invariant.Check(next.held() < d.lineup.held(), "dropping a stale card shortens the schedule"); err != nil {
			return d, Card{}, false
		}
		// ON THE OPERATOR'S PILE, because this is a DELIBERATE removal (D-35).
		// PD-3's window exists to stop the station asserting something untrue,
		// so a card dropped here was taken away on purpose — and before this it
		// simply vanished, with nothing able to say what had gone or why.
		//
		// A FAILED card takes a different path and does NOT land here: routing
		// around a fault is not something the operator did.
		d.lineup = next.discard(gone)
		dropped = true
	}
	if !dropped {
		return d, Card{}, false
	}
	// The transition carries its own words, so it is never itself a candidate
	// for this — which is what stops a stale drain from looping.
	notice, err := Propose(Card{
		ID: staleTransitionID, Slot: Transition, Origin: FromDirector,
		Subject: "stale read", Headline: "Report out of date", Script: Say(staleTransitionText),
	})
	if err != nil {
		return d, Card{}, false
	}
	return d, notice, true
}

// firstStale is the first STANDBY card whose build has aged out.
//
// A card that was never built — a structural card, whose words were fixed at
// proposal — has a zero BuiltAt and can never be stale. That is the rule that
// keeps the transition itself out of the set it belongs to.
func (d Director) firstStale() (Card, bool) {
	for t := Track(0); t < numTracks; t++ { // bounded by the registry (P10-02)
		for _, c := range d.lineup.Cards(t) { // bounded by the schedule (P10-02)
			if c.State != Standby || c.BuiltAt.IsZero() {
				continue
			}
			if d.now.Sub(c.BuiltAt) > StaleAfter {
				return c, true
			}
		}
	}
	return Card{}, false
}

// RefreshAfter is how long a card's words may sit BEFORE the schedule asks for
// them again, and it exists because D-84 lets a card wait indefinitely.
//
// THE DEFECT IT CLOSES IS THE ONE THE RULING CREATES. The Composer now works on
// standby so the line is ready the instant the operator goes on air — and a
// station can sit on standby for an hour. At `StaleAfter` the prepared card is
// dropped as it takes the air (PD-3) and the listener's first words are "That
// report is out of date and has been dropped." The operator sets up a line-up,
// goes for coffee, presses the key, and the station opens by apologising.
//
// HALF THE WINDOW, AND THE DERIVATION IS THE POINT. A rebuild is ~1.03 s of
// network, so refreshing at half leaves 7.5 minutes of margin — four hundred
// times the cost of the thing it is racing, which is what makes it impossible
// for a refresh to be the reason a card goes stale. Refreshing AT StaleAfter
// would race `takeTheAir`, which runs first in the same settle and would drop
// the card on the very tick the refresh was due.
//
// AND IT IS NOT A SECOND STALENESS RULE. `StaleAfter` still decides what may go
// on the air; this only decides when to ask for better words while nothing can.
// Derived from it rather than chosen beside it, so the two cannot drift.
const RefreshAfter = StaleAfter / 2

// refreshStandby asks for a standing-by card's words again when they have aged
// past RefreshAfter, or does nothing.
//
// IT RE-HYDRATES; IT DOES NOT REPLACE (HUM LEAD, 2026-09-11):
//
//	"It's still the same card, it's just like the composer 'filling it' for the
//	 first time, it's just stale … No need to re-check admission — it's already
//	 been admitted. No need for the director to choose / move another card — it's
//	 still the same card, the order has been decided. It just needs to have its
//	 data updated."
//
// So this emits the SAME `BuildCard` the first fill emits, for the same ID. The
// card is not proposed again, not re-admitted, not re-ordered, and does not move
// in the line-up: `Built` comes home and `WithScript` overwrites the words and
// the stamp in place. Nothing else about it changes.
//
// ONLY WHILE THE TRACK CANNOT ADVANCE, which bounds the new behaviour to exactly
// the case that created it. A running station replaces its cards as it reads
// them, so nothing sits; a card that DOES sit on a running station is being held
// by a rail drain, and PD-3's drop is the right answer there — mid-broadcast
// there is no time to rebuild, which is the whole reason `readInstead` exists.
//
// ONE CARD, because only one is ever built: `toPrepare` is "one ahead, and only
// one", so every other card in the line-up has no words to go stale.
func (d Director) refreshStandby() (Director, []Effect) {
	if d.advances(MainTrack) {
		return d, nil
	}
	// AND NEVER WHILE THE RAIL HOLDS ANYTHING (DR-3, and the merge property test
	// caught this within a minute of the refresh being wired).
	//
	// THE COMPOSER IS ONE BOUNDED RESOURCE. `toPrepare` stops one card ahead, so
	// a rail holding [standing-by, admitted] leaves it idle — and this would have
	// spent the idle moment on a REPORT. The hazard behind it becomes eligible
	// the instant the one in front leaves the air, and its build would then queue
	// behind a report's. "The rail is prepared first, not merely aired first" is
	// property 5, and it was written because a plant that reversed the precedence
	// in `toPrepare` SURVIVED.
	//
	// BLUNTER THAN IT STRICTLY NEEDS TO BE, deliberately: the precise rule is
	// "no rail card still needs composing", and this refuses while the rail holds
	// anything at all. On the safety path the cost of being early is a report
	// whose words are a few minutes older, and the cost of being clever is a
	// hazard that waits.
	if len(d.lineup.tracks[AlertRail]) > 0 {
		return d, nil
	}
	for _, c := range d.lineup.tracks[MainTrack] { // bounded by the track (P10-02)
		if c.State != Standby || c.BuiltAt.IsZero() {
			continue
		}
		if d.now.Sub(c.BuiltAt) < RefreshAfter {
			return d, nil
		}
		// THE STAMP MOVES WITH THE ASK, NOT WITH THE ANSWER. A build takes a
		// second or more and the schedule ticks every second; without this the
		// same card is asked for again on every tick until its words come home,
		// which is a build storm against the provider the moment a refresh is
		// due. `Built` overwrites it with the true time when it lands.
		bumped := c
		bumped.BuiltAt = d.now
		next, err := d.lineup.Set(bumped)
		if err != nil {
			return d, nil
		}
		d.lineup = next
		return d, []Effect{BuildCard{ID: c.ID, Slot: c.Slot, Subject: c.Subject, Refs: c.Refs}}
	}
	return d, nil
}
