package lineup

// topoff.go — the Producer tops off the lineup (D-40, HUM LEAD 2026-09-10).
//
// THE GAP: nothing read the track's depth. Two things queued a main-track card —
// the deck reporting that a location needs a read, and the operator's undo — so
// the schedule held about ONE card while the console drew ten slots, and a
// dropped card left its slot empty for ever. 0.14.0's role model predicted it in
// as many words: "Nothing produces location reports or credits; the rotation
// does that implicitly."
//
// THE ROLE SPLIT IS THE DESIGN, not a comment on it:
//
//	the Producer PROPOSES — several at once, cheaply, because a proposal is a
//	name and DR-7 already says the words materialise at standby;
//	the Director CHOOSES — it owns the lineup (DR-1), and choosing is
//	arrangement, which is exactly what the role model gives it.
//
// AND IT CHOOSES BY THE WATCHLIST. That is what makes "chooses" a behaviour
// rather than a label: taking the first thing offered would put the producer in
// the chair. The watchlist is the OPERATOR'S stated rotation order, so ranking
// by it is the charter itself — "execute the will of the human operator".
//
// WHICH ALSO DEFINES "IN DOUBT" rather than leaving it to be invented later. The
// HUM LEAD's ruling says the Director "WHEN IN DOUBT may surface a choice modal
// to the Operator", and doubt is now a fact about the schedule: the Director is
// in doubt exactly when its own criterion cannot separate the candidates — when
// the proposals it is choosing between are all off the watchlist and therefore
// all rank the same. That surfacing is the UI half and is not built here.

import (
	"cmp"
	"slices"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Proposal is one cheap template the Producer puts forward: a location, and the
// one line that names it.
//
// A NAME AND NOTHING ELSE, and that is what makes proposing four of them cost
// nothing. A proposal carrying words would be a Composer decision made by a
// Producer, and it would be composed against data that was true when it was
// offered rather than when it plays (DR-7).
type Proposal struct {
	Ref      string // the location, as the schedule keys it
	Headline string // what the card is about, in the words a person reads

	// Slot is WHAT KIND OF READ this would be (D-46, D-48).
	//
	// THE PRODUCER'S TO SAY, because "what cards should exist" is its whole
	// charter, and the kind is most of what a card IS. The Director needs it to
	// choose: D-48's cadence term discriminates between KINDS — "we haven't had
	// a location report in a while" is not a statement about a location.
	//
	// THE ZERO VALUE IS A LOCATION REPORT, which is most of the broadcast and
	// what every proposal was before this field existed.
	Slot Slot
}

// Offered is the Producer offering the Director something to fill the lineup
// with.
//
// A FACT, NOT A REQUEST, like every other member of the set: it says what the
// producer has, and whether any of it is scheduled is the Director's. The
// producer may offer more than the Director needs, may offer what is already
// scheduled, and may offer every time the lineup is published — none of that
// costs anything, because the Director is the one holding the depth.
type Offered struct {
	isEvent
	Proposals []Proposal
}

// onOffered fills the main track up to the Director's depth.
//
// ADMISSION IS A PROMISE TO READ. IT IS NOT A PROMISE TO READ *RIGHT NOW*.
//
// THE DISTINCTION IS THE WHOLE OF D-84 (HUM LEAD, 2026-09-11), and the first
// draft of this comment got it wrong by saying DR-3 had been inverted. It has
// not: nothing admitted is dropped unread, and the line-up the operator builds
// on standby is that promise being KEPT — it goes to air when they press the
// key. What was wrong was the INFERENCE, "therefore do not admit while
// stopped".
//
//	"being able to see, manage, and change the line up PRIOR to going on air is
//	 a fundamental requirement — otherwise the user might as well just use
//	 Observer."
//
// IT ASKED `advances` FIRST, under "admission is a promise to read (DR-3)", so a
// station on standby admitted NOTHING: the console drew ten empty slots and the
// operator had nothing to inspect, reorder or drop until after they had gone on
// the air. The reasoning was that a stopped programme "would accumulate a
// rotation nobody can drop" — and the answer is that the operator CAN drop it.
// That is the whole point of the surface.
//
// THE RULE THIS PACKAGE ALREADY HAD, one file along: `reconcileJoins` is "NOT
// GATED ON `advances` … A stopped station still has a running order; it simply
// is not reading it." Admission now says the same thing, and the gate lives in
// ONE place — `airOnce`, which is where READING happens.
//
// AND EVERY CARD OF IT IS OWED A READ WHEN THE PROGRAMME COMES BACK, which the
// old comment raised as the hazard. It is the REQUIREMENT.
func (d Director) onOffered(ev Offered) (Director, []Effect) {
	// COUNTED IN THE LINE-UP, NOT THE SCHEDULE (D-44). The depth is a promise
	// about what the OPERATOR sees: counting the Director's own cards against it
	// would top a ten-slot console off at about five real reports and leave the
	// rest empty for ever.
	//
	// HOW MANY SLOTS ARE OWED, and the loop below is the ONE thing that enforces
	// it. An early `if need <= 0 { return }` stood here and its mutant SURVIVED:
	// with the walk's own bound, flipping it to `< 0` changed no outcome, which
	// is the rule written twice rather than an invariant — the same verdict, and
	// the same remedy, as the DR-7 guard card.go's Propose records.
	need := d.settings.Depth - len(d.lineup.Projection(MainTrack))
	took := 0
	for _, p := range d.chosen(ev.Proposals) { // bounded by the offer (P10-02)
		if took >= need {
			break
		}
		next, ok := d.admit(p)
		if !ok {
			// A malformed proposal, or a location the schedule is already
			// holding. THE PRODUCER'S BUG MUST NOT COST THE OTHER SLOTS: the
			// walk carries on, and the next candidate takes the slot this one
			// could not — the same call onNeedsRead makes about a malformed
			// need.
			continue
		}
		d.lineup = next
		took++
	}
	// NEVER MORE THAN THE SLOTS THAT WERE OWED. The walk's bound is the one
	// carrier of the depth and this is what says so out loud: a bound expressed
	// only as a `break` is a bound that a later `continue` can walk straight
	// past, which is exactly how the refused-proposal path could have spent a
	// slot it never filled.
	if err := invariant.Check(took <= need, "a top-off fills the slots that were owed and no more"); err != nil {
		return d, nil
	}
	if took == 0 {
		return d, nil
	}
	// FILLING NEVER OVERSHOOTS. The console draws a fixed number of slots and a
	// track deeper than the depth is a running order the operator cannot see the
	// end of — cards promised a read with nothing on screen admitting they exist.
	if err := invariant.Check(len(d.lineup.Projection(MainTrack)) <= d.settings.Depth,
		"topping off fills the track to its depth and never past it"); err != nil {
		return d, nil
	}
	return d.settle()
}

// chosen is the Director's decision: the proposals in the order IT would take
// them, which is the operator's watchlist order.
//
// STABLE, so the producer's own order survives among equals — two locations the
// operator never listed are offered in the order the producer thought best, and
// the Director has no reason to disagree with that half.
func (d Director) chosen(ps []Proposal) []Proposal {
	out := slices.Clone(ps)
	slices.SortStableFunc(out, func(a, b Proposal) int {
		// THE CADENCE TERM FIRST, AND IT IS SEPARABLE (D-48). It answers zero
		// for every proposal when the operator has switched it off, when
		// nothing has been read yet, or when a slot is outside the registry —
		// and a term equal for everything discriminates nothing, so the
		// comparison falls straight through to the watchlist below. That is the
		// degradation the HUM LEAD asked for, and it is arithmetic rather than
		// a branch anyone has to remember.
		if n := cmp.Compare(d.overdue(b.Slot), d.overdue(a.Slot)); n != 0 {
			return n // longest unread first
		}
		return cmp.Compare(d.rank(a.Ref), d.rank(b.Ref))
	})
	// A SORT THAT LOST A CANDIDATE would silently narrow the choice, and the
	// narrowing would look exactly like the producer having offered less.
	if err := invariant.Check(len(out) == len(ps), "choosing reorders the proposals and never drops one"); err != nil {
		return nil
	}
	// AND THE DECISION ACTUALLY HAPPENED. A comparator that stopped
	// discriminating would leave the producer's own order in place and nothing
	// downstream could tell — the cards would still be well-formed, the depth
	// still respected, and the operator's watchlist quietly ignored. That is the
	// failure this whole file exists to prevent, so it is asserted rather than
	// assumed.
	if err := invariant.Check(slices.IsSortedFunc(out, func(a, b Proposal) int {
		if n := cmp.Compare(d.overdue(b.Slot), d.overdue(a.Slot)); n != 0 {
			return n
		}
		return cmp.Compare(d.rank(a.Ref), d.rank(b.Ref))
	}), "the chosen order runs in the Director's own order of preference"); err != nil {
		return nil
	}
	return out
}

// rank is where a location sits in the operator's rotation, and OFF THE LIST
// RANKS LAST.
//
// Not "is excluded". A location the producer offers is inside the service radius
// and is worth reading; it simply does not outrank something the operator asked
// for by name. Every off-list candidate gets the SAME rank, which is what makes
// the doubt above a fact rather than a judgement.
func (d Director) rank(ref string) int {
	for i, listed := range d.settings.Watchlist { // bounded by the watchlist (P10-02)
		if listed == ref {
			// A RANK INSIDE THE LIST IS A POSITION IN IT. An off-by-one here
			// would not fail anything — it would silently read the operator's
			// rotation in the wrong order, which is the class of defect that
			// shows up as "the station keeps leading with the wrong town".
			if err := invariant.Check(i < len(d.settings.Watchlist), "a listed location ranks at its own position"); err != nil {
				break
			}
			return i
		}
	}
	return len(d.settings.Watchlist)
}

// admit turns one proposal into a queued card, or says it could not.
//
// IT TAKES THE ROTATION'S IDENTITY for the location (ReadID), and that is
// deliberate: ReadID is a pure function of the ref, and the lineup refuses two
// cards with one identity — so a top-off can never schedule a location the
// rotation is already holding, without a second rule to keep in step (FR-2.5).
// The refusal is what the caller reads as "this slot goes to the next candidate".
//
// THE ORIGIN IS THE DIRECTOR'S because the Director chose it. The producer put
// it forward; DR-4 records who put the card in the running order, and that is
// the chair this one was decided in.
func (d Director) admit(p Proposal) (Lineup, bool) {
	// THE TOP-OFF FILLS THE MAIN TRACK, so it admits only what belongs there.
	// A takeover proposed here would drain onto the rail, leaving the slot it
	// was meant to fill still empty while the walk counted it spent.
	if trackFor(p.Slot) != MainTrack {
		return d.lineup, false
	}
	// AND A CARD THAT JUST FAILED IS NOT PUT STRAIGHT BACK (see retry.go). The
	// producer offers what it always offers and cannot know a report would not
	// compose a moment ago; the Director can, and this is the one place a
	// proposal becomes a card.
	if d.sittingOut(p.Ref) {
		return d.lineup, false
	}
	// AND THE DIRECTOR'S OWN CARDS ARE NOT THE PRODUCER'S TO PROPOSE (D-43).
	// Transitions are derived from the running order; one arriving as a proposal
	// would be a second author of the same thing.
	//
	// A TRIPWIRE, NOT A DECISION, and its mutant SURVIVES BY DESIGN: a
	// structural card's words are fixed at proposal, `Proposal` carries none,
	// and `check` refuses a wordless transition — so today the rule holds for an
	// UNRELATED reason. That is the D-42 shape exactly, and it is stated here
	// for the same reason: a rule held by a different rule is one that vanishes
	// silently the day the other rule moves.
	if err := invariant.Check(!p.Slot.structural(),
		"the Director's own structural cards are derived from the order, never proposed"); err != nil {
		return d.lineup, false
	}
	card, err := Propose(Card{ID: ReadID(p.Ref), Slot: p.Slot, Origin: FromDirector,
		Subject: p.Ref, Headline: p.Headline})
	if err != nil {
		return d.lineup, false
	}
	admitted, err := card.To(Admitted)
	if err != nil {
		return d.lineup, false
	}
	next, err := d.lineup.Queue(MainTrack, admitted)
	if err != nil {
		return d.lineup, false
	}
	return next, true
}
