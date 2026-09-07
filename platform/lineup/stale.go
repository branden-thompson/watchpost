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
		d.lineup = next
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
