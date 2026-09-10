package lineup

// transitions.go — the inter-card transition, DERIVED FROM THE JOIN (D-43).
//
// THE DIRECTOR'S ONE ADDITIVE ACT (role-model.md).  Everything else in this
// package arranges cards other roles proposed; this is the only place the
// Director puts a card into the running order because of what the running order
// IS.
//
// IT BELONGS TO THE JOIN, NOT TO THE CARD, and the HUM LEAD's own sentence is
// the argument: "the Director will re-evaluate if transitions are needed for
// that card AT THAT TIME, and the answer MAY BE DIFFERENT depending on what the
// line looks like at that time."  That is as true of a DROP as of a
// resurrection — dropping the card between A and C changes what A→C needs.  Card
// ownership would leave adjacency artifacts (two tails in a row, a lead with
// nothing before it) and something would have to clean them up, which is the
// recompute it was trying to avoid.
//
// AND RE-DERIVING IS FREE.  `slots()` gives Transition `textAtStandby: false`,
// so a transition's words are FIXED AT PROPOSAL — no composer, no network,
// nothing to wait for.  Creating and destroying them costs nothing, which is
// what makes "recompute the whole running order on every change" the cheap
// option rather than the expensive one.
//
// WHAT THIS BUYS, AND IT IS THE HUM LEAD'S STATED REQUIREMENT: "the system needs
// to be flexible enough that adding additional transition or inter-card cards is
// low cost, and doesn't require a complete rewiring of the line-up flow and
// logic."  Adding a kind means adding a rule to `between` — ONE function that
// looks at (previous, next).  `Reorder`, `onDropped` and the undo never change,
// because there is no association for them to honour.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// joinPrefix marks the cards THIS FILE OWNS.
//
// OWNERSHIP BY IDENTITY, and it is load-bearing rather than cosmetic: the
// reconcile deletes structural cards it no longer wants, and the staleness
// notice is a structural card it never made.  A reconcile that owned every
// structural card would delete that notice silently, leaving the listener the
// gap it exists to explain.  The prefix is the house pattern — `ReadID` prefixes
// "read:", `BurstID` prefixes its own, `staleTransitionID` is its own constant.
const joinPrefix = "join:"

// joinID is the identity of the transition between two cards, and it is a PURE
// FUNCTION OF THE PAIR.
//
// THAT IS WHAT MAKES THE RECONCILE IDEMPOTENT.  It runs on every settle, so a
// non-deterministic id would grow the schedule by one transition per event; with
// this, "is it already there?" is a lookup, and the lineup's own refusal of two
// cards under one identity is the backstop.
func joinID(prev, next string) string {
	return joinPrefix + prev + ">" + next
}

// isJoinID reports whether an id names a card this file minted. It sits beside
// joinID deliberately: the shape is defined once and read once.
func isJoinID(id string) bool { return strings.HasPrefix(id, joinPrefix) }

// splitJoinID reads back the pair a join was minted for — the third and last
// reader of the id's shape, all of them in this block.
func splitJoinID(id string) (prev, next string, ok bool) {
	if !isJoinID(id) {
		return "", "", false
	}
	prev, next, ok = strings.Cut(strings.TrimPrefix(id, joinPrefix), ">")
	return prev, next, ok && prev != "" && next != ""
}

// between is THE ONE PLACE THE RULE LIVES: what, if anything, belongs at the
// join from prev to next.
//
// THE RULE IS D-43's, STATED BY THE HUM LEAD — "Location Report Card from
// Operator needs to be bookended by transition cards" — and it is deliberately
// the whole rule for now.  What ELSE makes a join need something said (a change
// of subject, a report following an alert, elapsed time since the last one) is a
// CONTENT ruling and is not yet made; when it is, it is arms in this function
// and nothing else moves.
func between(prev, next Card) (Card, bool) {
	if prev.Origin != FromOperator && next.Origin != FromOperator {
		return Card{}, false
	}
	// A TRANSITION NEVER BOOKENDS A TRANSITION — two structural cards in a row is
	// the Director talking to itself, and it is exactly the adjacency artifact
	// card-ownership would have produced by design.
	//
	// A TRIPWIRE, NOT A GUARD, and the difference is the one this release keeps
	// re-learning. It was written as a guard and its mutant SURVIVED: `between`
	// is asked only about cards in the RUNNING ORDER, and the running order
	// excludes structural cards by construction (D-44), so the branch could
	// never decide anything. Deleting it would have thrown away a real
	// precondition; leaving it as a decision was the rule written twice. It
	// watches instead — and it is what fails, loudly, if a second caller ever
	// asks this question about the schedule rather than the line-up.
	if err := invariant.Check(!prev.Slot.structural() && !next.Slot.structural(),
		"a join is asked about the running order, where the Director's own cards do not appear"); err != nil {
		return Card{}, false
	}
	// ITS WORDS ARE FIXED HERE, WHICH IS WHY THEY NAME WHAT COMES NEXT. The
	// Director is the only role that knows the order, and DR-7's other half says
	// a card whose text is fixed at proposal must arrive carrying it.
	card, err := Propose(Card{
		ID: joinID(prev.ID, next.ID), Slot: Transition, Origin: FromDirector,
		Subject:  next.Subject,
		Headline: "Coming up: " + next.Headline,
		Script:   Say("Coming up, " + next.Headline + "."),
	})
	if err != nil {
		return Card{}, false
	}
	return card, true
}

// wantedJoins is the transitions the CURRENT running order calls for, in order,
// each with the card it must sit in front of.
//
// THE HEAD IS NOT A JOIN. A card at the top of the running order has nothing
// before it to hand off from — what precedes it is the bed, or silence, and
// neither is a card. (A hand-off out of the BED is the cut-over's business, not
// this one; D-11 and FR-4.2 own that lane.)
func (d Director) wantedJoins() []Card {
	view := d.lineup.Projection(MainTrack)
	out := make([]Card, 0, len(view))
	for i := 1; i < len(view); i++ { // bounded by the running order (P10-02)
		card, ok := between(view[i-1], view[i])
		if !ok {
			continue
		}
		out = append(out, card)
	}
	// ONE JOIN PER GAP, at most. More would mean `between` had produced two
	// cards for one place in the order, which is a schedule that reads the same
	// hand-off twice.
	if err := invariant.Check(len(out) < max(len(view), 1), "a running order of n cards has at most n-1 joins"); err != nil {
		return nil
	}
	return out
}

// reconcileJoins brings the schedule's transitions into line with what the
// running order now calls for. It is the whole of "they move with the card" and
// "they simply go away" — neither is implemented, both fall out.
//
// IT RUNS FIRST IN `settle`, before anything reads the order, because a
// transition inserted now may be the very next thing spoken.
//
// IT IS NOT GATED ON `advances`. These are consequences of cards the schedule
// has ALREADY admitted, not new admissions, so DR-3's promise was made when the
// report was let in. A stopped station still has a running order; it simply is
// not reading it.
func (d Director) reconcileJoins() Director {
	d.lineup = d.lineup.dropStaleJoins()
	for _, c := range d.wantedJoins() { // bounded by the running order (P10-02)
		d.lineup = d.lineup.addJoin(c)
	}
	return d
}

// stillHolds reports whether the join at index i still describes the hand-off it
// was minted for.
//
// ASKED OF THE SCHEDULE, NOT OF THE PROJECTION, and that distinction is a
// DEFECT THIS FILE ALREADY HAD. Deriving the wanted set from the running order
// alone prunes a join the moment its lead-in card is READ: `a` finishes, leaves
// the schedule, and the transition introducing what follows it is deleted in the
// same settle — one step before it would have been spoken. The listener loses
// the hand-off and hears the next report begin cold.
//
// So a join survives when it is AT THE HEAD OF THE TRACK: everything before it
// has already been read, which is the only way its lead-in can legitimately have
// vanished. A join whose lead-in was DROPPED from the middle is not at the head,
// and is pruned — which is the difference between "already read" and "taken
// away" expressed as a position rather than as a flag anyone has to maintain.
func (l Lineup) stillHolds(i int) bool {
	track := l.tracks[MainTrack]
	prev, next, ok := splitJoinID(track[i].ID)
	if !ok {
		return false
	}
	// IT MUST STILL INTRODUCE WHAT FOLLOWS IT. A transition read in front of the
	// wrong card is worse than none: it announces something that is not next.
	if i+1 >= len(track) || track[i+1].ID != next {
		return false
	}
	if i == 0 {
		return true // its lead-in has been read; this is the pending hand-off
	}
	return track[i-1].ID == prev
}

// dropStaleJoins removes the transitions this file minted that the order no
// longer calls for.
//
// TWO THINGS IT WILL NOT TOUCH. A structural card it did not mint — the
// staleness notice — because ownership is by identity. And a card ON THE AIR
// (D-45): the listener is mid-sentence, and taking it back is the one thing a
// schedule must never do.
func (l Lineup) dropStaleJoins() Lineup {
	out := l.clone()
	kept := make([]Card, 0, len(out.tracks[MainTrack]))
	for i, c := range out.tracks[MainTrack] { // bounded by the track (P10-02)
		if isJoinID(c.ID) && c.State != OnAir && !l.stillHolds(i) {
			continue
		}
		kept = append(kept, c)
	}
	out.tracks[MainTrack] = kept
	// IT ONLY EVER SHORTENS. A reconcile that added a card here would be
	// inventing a running order rather than pruning one.
	if err := invariant.Check(len(kept) <= len(l.tracks[MainTrack]), "pruning the joins never adds a card"); err != nil {
		return l
	}
	return out
}

// addJoin puts one transition in front of the card it introduces, if the
// schedule is not already holding it.
func (l Lineup) addJoin(c Card) Lineup {
	if _, _, held := l.find(c.ID); held {
		return l // already there; the id is a pure function of the join
	}
	admitted, err := c.To(Admitted)
	if err != nil {
		return l
	}
	_, after, ok := splitJoinID(c.ID)
	if !ok {
		return l
	}
	next, err := l.insertBefore(MainTrack, after, admitted)
	if err != nil {
		return l
	}
	return next
}
