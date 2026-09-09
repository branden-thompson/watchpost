package lineup

// rotation.go — the rotation's turn becomes a card (0.16.0 P3, T3.2b).
//
// THE BED IS NOT A TRACK, and this must not flatten that. Tuning to a LIVE
// relay stays a Tune: `Track`'s own comment says the bed is "a selectable
// resource the Director may cut over to, not a queue of cards", and a relay
// is not something the narrator arbiter can own — it is live radio that
// cannot be paused, only ducked.
//
// WHAT BECOMES A CARD IS THE SYNTHESISED READ. That is the half the arbiter
// CAN own, and owning it is what makes the two paths to speech one.

// NeedsRead says a location wants a synthesised read.
//
// A FACT, NOT A REQUEST (the event set's own rule). The deck is the only
// thing that can observe that a relay is unavailable, has failed, or was
// never wanted; whether that becomes a card is the Director's decision, which
// is the same division `Tuned` and `Ended` already use.
type NeedsRead struct {
	isEvent
	Ref      string // the location, as the schedule keys it
	Headline string // what the card is about, in the words a person reads
}

// ReadID is the identity of the rotation card for a location, and the ONE owner
// of that rule.
//
// IT IS A PURE FUNCTION OF THE REF, and that is the whole no-double-speak
// mechanism (FR-2.5) at the schedule level: the lineup already refuses two
// cards with one identity, so a second need for a location the schedule is
// still holding queues nothing, without a second rule to keep in step. Make the
// id unique per call and the property is gone — which is what the duplicate
// test is really pinned to.
//
// Prefixed so a rotation card can never collide with an alert's own id, or with
// a burst's, in the same schedule — the same reason BurstID is prefixed.
func ReadID(ref string) string {
	if ref == "" {
		return ""
	}
	return "read:" + ref
}

// onNeedsRead proposes the location's read as a main-track card.
//
// ADMISSION IS A PROMISE TO READ (DR-3), so a track that cannot advance must
// not accept one. That is why this asks advances rather than testing the power
// directly: a stopped programme would otherwise pile up a rotation nobody can
// drop, and every card of it would be owed a read the moment the operator
// pressed start. One predicate, now three askers — taking the air, preparing
// what follows, and admitting.
func (d Director) onNeedsRead(ev NeedsRead) (Director, []Effect) {
	if !d.advances(MainTrack) {
		return d, nil
	}
	card, err := Propose(Card{ID: ReadID(ev.Ref), Slot: LocationReport, Origin: FromDirector,
		Subject: ev.Ref, Headline: ev.Headline})
	if err != nil {
		// A need with no location, or no headline, is the deck's bug and not
		// the rotation's victim. The step is a no-op, exactly as a malformed
		// burst is.
		return d, nil
	}
	if card, err = card.To(Admitted); err != nil {
		return d, nil
	}
	next, err := d.lineup.Queue(MainTrack, card)
	if err != nil {
		// "Already held": the location is still in the schedule, and asking
		// again is not a second read. See ReadID.
		return d, nil
	}
	d.lineup = next
	return d.settle()
}
