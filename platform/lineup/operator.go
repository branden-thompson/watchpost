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

import "github.com/branden-thompson/watchpost/platform/invariant"

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
	was, next, ok := d.lineup.takeDiscarded(ev.ID)
	if !ok {
		return d, nil
	}
	d.lineup = next
	card, err := Propose(Card{ID: was.ID, Slot: was.Slot, Origin: FromOperator,
		Subject: was.Subject, Headline: was.Headline, Refs: was.Refs})
	if err != nil {
		return d, nil
	}
	admitted, err := card.To(Admitted)
	if err != nil {
		return d, nil
	}
	queued, err := d.lineup.Queue(trackFor(was.Slot), admitted)
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
	if err := invariant.Check(to >= 0 && to < len(l.tracks[t]), "a card moves to a position the track has"); err != nil {
		return l, err
	}
	out := l.clone()
	track := out.tracks[t]
	card := track[from]
	track = append(track[:from], track[from+1:]...)
	track = append(track[:to], append([]Card{card}, track[to:]...)...)
	out.tracks[t] = track
	// MOVES ONE, ADDS NONE, LOSES NONE. A slice reorder written by hand is
	// exactly where a card goes missing, and a schedule that lost one has
	// promised a read it will never make (DR-3).
	if err := invariant.Check(len(out.tracks[t]) == len(l.tracks[t]), "reordering moves a card and never adds or drops one"); err != nil {
		return l, err
	}
	return out, nil
}
