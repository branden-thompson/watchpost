package lineup

// operator.go — the operator's three acts on a card (FR-3, P4).
//
// "EXECUTE THE WILL OF THE HUMAN OPERATOR" is the Director's charter, and this
// is the half that was missing: the card model has carried the fields for two
// releases — `director-requirements.md` says in as many words that "the card
// model carries the fields; the controls arrive with the Broadcaster UI" — and
// nothing could move a card.
//
// FR-3.3 IS WHY THESE ARE EVENTS AND NOT SETTERS. "An action must never be
// shown as taken unless the schedule took it", and the named trap is that
// `Lineup.Set` refuses to reorder BY DESIGN — so a promote routed through it
// updates a display and leaves `Next()` answering the old order, with nothing
// to see. Reordering needs its own mutator, and it is below.

import (
	"errors"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Moved is the operator putting a card at a position on its own track.
//
// A POSITION, NOT A DIRECTION. Promote and demote are what the OPERATOR does;
// what the schedule is told is where the card ended up, because the console
// already knows every slot number it drew and a relative move would make the
// Director re-derive something the surface had in hand.
//
// IT NEVER CHANGES TRACKS. The rail is the producer's ordering and the main
// track is the rotation; a card that could hop between them would let an
// operator schedule a hazard as programming, or a report as a takeover.
type Moved struct {
	isEvent
	ID string
	To int
}

// Dropped is the operator taking a card out of the running order.
//
// IT IS THE DESTRUCTIVE ONE (FR-3.7), and it is recoverable two ways: a confirm
// guards the accidental keypress, an undo recovers the considered-but-wrong
// decision. Two different failures, two different remedies — and the undo
// carries NO TIMER, because "an expiring undo is a hidden clock, and a hidden
// clock under pressure is a trap."
type Dropped struct {
	isEvent
	ID string
}

// Restored is the operator undoing a drop.
type Restored struct {
	isEvent
	ID string
}

// Requested is the operator asking for a report, at a position (R3).
//
// THE CARD ARRIVES BUILT. The Director does not mint cards — D-40's whole split
// is "the Producer proposes, the Director chooses" — so the app composes and
// admits it, and this says WHERE the operator wants it. The same shape `Moved`
// has, one field wider.
//
// `To` IS A RUNNING-ORDER POSITION, the same number `Moved.To` carries, and
// PRIORITIZE is simply zero: the front of the running order, which is UP NEXT.
//
// THE DIRECTOR DOES NOT REFUSE IT (HUM LEAD, 2026-09-14): "The Director also
// executes the will of the Operator, so the only time a Director would refuse is
// if the card request is not valid … For 0.16.0 that answer should be NO, it
// doesn't refuse the human operator. The only thing that would supersede an
// operator action is again, a valid alert within the broadcast service radius."
// A hazard still interrupts, because the rail outranks the main track by
// construction — not by refusing this.
type Requested struct {
	isEvent
	Card Card
	To   int
}

// onRequested puts the operator's card where they asked for it.
//
// IT CANNOT FAIL ON THE OPERATOR'S ACCOUNT. `Insert` refuses only a card that is
// not a card — unadmitted, identity-less, or one whose id the lineup already
// holds — and every one of those is the Producer having built it wrong rather
// than the operator having asked for something unreasonable.
func (d Director) onRequested(ev Requested) (Director, []Effect) {
	next, err := d.lineup.Insert(MainTrack, ev.Card, ev.To)
	if err != nil {
		return d, nil
	}
	d.lineup = next
	return d.settle()
}

// onMoved puts the card where the operator put it.
func (d Director) onMoved(ev Moved) (Director, []Effect) {
	next, err := d.lineup.Reorder(ev.ID, ev.To)
	if err != nil {
		return d, nil // a card the schedule does not hold, or one being read
	}
	d.lineup = next
	return d.settle()
}

// onDropped takes the card out of the running order and puts it where the undo
// can reach it (D-35).
func (d Director) onDropped(ev Dropped) (Director, []Effect) {
	card, held := d.find(ev.ID)
	if !held {
		return d, nil
	}
	// NOT THE DIRECTOR'S OWN CARDS (D-44). The operator never saw it, so the
	// drop cannot have meant it — and a structural card on the undo pile would
	// offer them a restore of something they never scheduled.
	if card.Slot.structural() {
		return d, nil
	}
	// THE ON-AIR CARD IS NOT DROPPABLE HERE. Stopping a read in progress is a
	// different act with a different sound, and DR-24 pairs its own release;
	// routing it through the running order would take a card off the air with
	// no cue released and the band still holding a callout.
	if card.State == OnAir {
		return d, nil
	}
	gone, err := card.To(Discarded)
	if err != nil {
		return d, nil
	}
	next, err := d.lineup.Remove(gone.ID)
	if err != nil {
		return d, nil
	}
	d.lineup = next.discard(gone)
	return d.settle()
}

// onRestored puts a dropped card back, as a NEW card.
//
// NOT A REVIVAL, and the card model is explicit: "REFUSED, DONE and DISCARDED
// lead nowhere: a card that could be revived is a card that can be read twice."
// So this proposes afresh from what the pile kept — and deliberately WITHOUT
// the old words, because they were composed against data that may since have
// become untrue, which is the whole reason the staleness window exists.
//
// AND IT IS THE OPERATOR'S (FR-3.4). A human put this card back, and the card
// says so: `Origin.FromOperator` has existed for two releases with nothing ever
// constructing it.
func (d Director) onRestored(ev Restored) (Director, []Effect) {
	// NOTHING IS COMMITTED UNTIL ALL OF IT SUCCEEDS (F-74). The pile entry used
	// to be spent here, on the line that takes it, and every refusal below
	// returned the Director that had ALREADY LOST IT — so a restore the schedule
	// would not accept destroyed the operator's one recovery and put nothing
	// back, silently.
	//
	// IT WAS REACHED THE ORDINARY WAY, which is why this is a defect and not a
	// hardening: ReadID is a pure function of the ref, so a location dropped and
	// then re-queued by the rotation is holding the very identity the undo wants.
	// `next` is therefore carried down as a LOCAL and assigned to the Director
	// only at the end.
	was, next, ok := d.lineup.takeDiscarded(ev.ID)
	if !ok {
		return d, nil
	}
	card, err := Propose(Card{ID: was.ID, Slot: was.Slot, Origin: FromOperator,
		Subject: was.Subject, Headline: was.Headline, Refs: was.Refs})
	if err != nil {
		return d, nil // it cannot be re-proposed — a structural card, whose words the undo drops
	}
	admitted, err := card.To(Admitted)
	if err != nil {
		return d, nil
	}
	queued, err := next.Queue(trackFor(was.Slot), admitted)
	if err != nil {
		return d, nil // the schedule is already holding one under that identity
	}
	d.lineup = queued
	return d.settle()
}

// Reorder moves a card to a position on the track it is already on.
//
// ITS OWN MUTATOR, and FR-3.3 is the reason: `Set` replaces in place and says
// so — "the schedule's order is the planner's decision and a state change must
// not reorder it" — which is right for a state change and useless for a
// promote. Routing the operator's intent through it is the silent no-op the
// requirement exists to make impossible.
func (l Lineup) Reorder(id string, to int) (Lineup, error) {
	t, from, held := l.find(id)
	if err := invariant.Check(held, "Reorder moves a card the lineup is holding"); err != nil {
		return l, err
	}
	// A CARD BEING READ IS NOT IN THE RUNNING ORDER. Giving it a queue position
	// would have the schedule describe a place for something that has left the
	// queue.
	if err := invariant.Check(l.tracks[t][from].State != OnAir, "a card on the air is not in the running order"); err != nil {
		return l, err
	}
	// AND A STRUCTURAL CARD IS NOT IN IT EITHER (D-44). The operator cannot see
	// one, so they cannot have meant one: a move naming it would be the console
	// addressing a slot it never drew.
	if err := invariant.Check(!l.tracks[t][from].Slot.structural(), "the operator moves cards they can see; the Director's own cards are not among them"); err != nil {
		return l, err
	}
	out := l.clone()
	track := out.tracks[t]
	card := track[from]
	track = append(track[:from], track[from+1:]...)
	out.tracks[t] = track
	// `to` IS A POSITION IN THE LINE-UP, NOT IN THE SCHEDULE, and this is its
	// ONE bound. A `to >= 0 && to < len(Projection(t))` guard stood above and its
	// mutant SURVIVED: the lookup refuses exactly the same set, so the guard
	// could never decide anything — the rule written twice, which is the second
	// time today (see topoff.go, and card.go's Propose before it).
	//
	// Using the schedule's own index here instead would be a SILENT off-by-N,
	// because both numbers are valid indices and neither errors.
	at, ok := out.scheduleIndex(t, to)
	if err := invariant.Check(ok, "a card moves to a position the RUNNING ORDER has"); err != nil {
		return l, err
	}
	track = append(track[:at], append([]Card{card}, track[at:]...)...)
	out.tracks[t] = track
	// MOVES ONE, ADDS NONE, LOSES NONE. A slice reorder written by hand is
	// exactly where a card goes missing, and a schedule that lost one has
	// promised a read it will never make (DR-3).
	if err := invariant.Check(len(out.tracks[t]) == len(l.tracks[t]), "reordering moves a card and never adds or drops one"); err != nil {
		return l, err
	}
	return out, nil
}

// MainTrackCap is how many cards the running order holds.
//
// THE SCHEDULE'S NUMBER, NOT THE SCREEN'S. `modes/tty` had it as
// `MainTrackSlots` and the console is where it was first needed, but "the last
// card falls off" is a SCHEDULE rule — the discard pile it falls into belongs to
// the Lineup, and a cap the UI owned would be a bound the domain could not
// enforce. The console reads this now, so there is one number (the D-124
// standing: a test that prevents drift is not the same as a fact with one owner).
//
// SIXTEEN: the card on the air, UP NEXT, and fourteen behind them.
const MainTrackCap = 16

// Insert puts a card INTO the running order at `to`, pushing the rest down.
//
// HUM LEAD, 2026-09-14: "Line-Up Slot: ___ / Cards from this position will be
// pushed down by 1", and PRIORITIZE is the same act at position zero — "Move to
// 'UP NEXT' / Pushes all existing line-up cards down by 1".
//
// THE LAST CARD FALLS OFF INTO THE DISCARD PILE, which is the ruling: "Last Card
// fall off, can go into our discard pile and the producer can re-request a copy
// of that card if needed, otherwise if the user really wants that location they
// can look it up and manually re-place it into the bottom slot."
//
// A FULL TRACK IS NOT A REASON TO REFUSE THE OPERATOR. Refusing would be the
// console saying no to a request the schedule could honour, and FR-3.3's rule
// runs the other way: an action must never be SHOWN as taken unless the schedule
// took it. This takes it, and says what it cost.
//
// `to` IS A RUNNING-ORDER POSITION, the same number `Reorder` takes, resolved by
// the same `scheduleIndex` — so an operator who types 4 gets the fourth thing
// they can SEE, whatever structural cards the Director has between them.
func (l Lineup) Insert(t Track, c Card, to int) (Lineup, error) {
	if err := invariant.Check(t >= 0 && t < numTracks, "a card is inserted onto one of the declared two tracks"); err != nil {
		return l, err
	}
	if err := invariant.Check(c.State == Admitted, "the lineup holds admitted cards only"); err != nil {
		return l, err
	}
	if err := c.check(); err != nil {
		return l, err
	}
	if _, _, taken := l.find(c.ID); taken {
		// NOT AN INVARIANT, A REFUSAL. Asking twice for the same report is a
		// thing an operator can reasonably do by accident, and the schedule
		// already holding it is the honest answer rather than a broken rule.
		return l, errors.New("the running order already holds " + c.ID)
	}
	// PAST THE END MEANS THE END (D-149). The Line-Up Request window opens at
	// slot 15 — the HUM LEAD's ruled default, "default to the bottom" — and a
	// running order of three cards is the ordinary case, so the position the
	// window offers by default named no visible card.
	//
	// IT WAS REFUSED SILENTLY, AND THE WINDOW HAD ALREADY CLOSED. `Insert`'s
	// invariant failed, `onRequested` returned no effects, nothing was queued,
	// and `requestSchedule` closes on `valid()` — so the operator was shown a
	// scheduled request the schedule never took, which is FR-3.3's named trap on
	// this release's headline control. The console compounds it: it draws slots
	// 02..15 as empty ADDRESSES, and every one of them was refused.
	//
	// CLAMPED HERE RATHER THAN IN `scheduleIndex`, because that function also
	// serves `Reorder`, where a slot the running order never drew is meaningless
	// and must stay refused.
	if n := l.visible(t); to > n {
		to = n
	}
	at, ok := l.scheduleIndex(t, to)
	if err := invariant.Check(ok, "a card is inserted at a position the RUNNING ORDER has"); err != nil {
		return l, err
	}
	out := l.clone()
	track := out.tracks[t]
	track = append(track[:at], append([]Card{c}, track[at:]...)...)
	out.tracks[t] = track

	// AND THE OVERFLOW FALLS OFF THE BOTTOM, one card for the one that came in.
	// Counted over what the operator can SEE: the Director's structural cards are
	// not in the running order and must not be pushed out of it.
	for out.visible(t) > MainTrackCap {
		last, ok := out.lastVisible(t)
		if !ok {
			break // nothing left to shed; the cap is smaller than the structure
		}
		fallen := out.tracks[t][last]
		out.tracks[t] = append(out.tracks[t][:last], out.tracks[t][last+1:]...)
		out = out.discard(fallen)
	}
	return out, nil
}

// visible is how many cards of a track the operator can see — the running
// order's own length, which is what the cap counts.
func (l Lineup) visible(t Track) int {
	n := 0
	for _, c := range l.tracks[t] { // bounded by the track (P10-02)
		if !c.Slot.structural() {
			n++
		}
	}
	return n
}

// lastVisible is the schedule index of the last card in the running order.
func (l Lineup) lastVisible(t Track) (int, bool) {
	for i := len(l.tracks[t]) - 1; i >= 0; i-- { // bounded by the track (P10-02)
		if !l.tracks[t][i].Slot.structural() {
			return i, true
		}
	}
	return 0, false
}
