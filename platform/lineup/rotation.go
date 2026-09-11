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
// ADMISSION IS A PROMISE TO READ, NOT A PROMISE TO READ RIGHT NOW (D-84). This
// asked `advances` too, for the reason onOffered did, and with the same
// consequence: a station on standby accepted no cards at all. The predicate now
// has ONE asker — `airOnce` — and the schedule is planned whether or not it is
// being read.
func (d Director) onNeedsRead(ev NeedsRead) (Director, []Effect) {
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
