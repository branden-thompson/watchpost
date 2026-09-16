package lineup

// projection.go — the LINE-UP the operator sees, from the SCHEDULE the roles
// work from (D-44).
//
// THE DISTINCTION IS RULED, AND IT IS OLDER THAN THIS FILE (HUM LEAD,
// 2026-09-05, director-build-log.md:1945):
//
//	"The operator's view is a PROJECTION of the one Lineup, not a second
//	schedule … The one-card model REDUCES the gap between what the operator sees
//	and what the machine holds, rather than creating it."
//
// IT WAS THE IDENTITY FUNCTION UNTIL NOW, which is why no code carried it: one
// card per burst and nothing invisible meant `Cards` already WAS the operator's
// view. The Director's structural cards are the first thing that makes it a real
// function — and one of them has been in production since 0.14.0, because
// director.go queues the staleness notice straight onto the main track.
//
// THE VOCABULARY, because the type is named for the wrong half of it:
//
//	Cards(t)      — THE SCHEDULE.  Every card, structural ones included.  What
//	                the Reader, the Composer and the staleness check work from.
//	Projection(t) — THE LINE-UP.  What the operator sees, numbers and addresses.
//
// AND THE GAP IS THE THING TO WATCH. The 2026-09-05 ruling's rationale is that
// the projection should SHRINK the distance between the two, so every structural
// card is a small debt against it. That is why `structural` is a field on the
// slot registry rather than a property anything may claim: the set of things the
// operator cannot see is closed, and adding to it is a deliberate act.

import "github.com/branden-thompson/watchpost/platform/invariant"

// Projection is the running order as the OPERATOR sees it — the schedule with
// the Director's own structural cards taken out.
//
// A COPY, like Cards, and for the same reason: a reader that could write into
// the schedule is the second writer DR-1 exists to make impossible.
func (l Lineup) Projection(t Track) []Card {
	if err := invariant.Check(t >= 0 && t < numTracks, "a track is one of the declared two"); err != nil {
		return nil
	}
	out := make([]Card, 0, len(l.tracks[t]))
	for _, c := range l.tracks[t] { // bounded by the track (P10-02)
		if c.Slot.structural() {
			continue
		}
		// AND THE CURRENT FENCE ADMITS IT (D-114). `nextForAir` has skipped
		// out-of-fence cards since D-75 — they are not READ — and the projection
		// went on returning them, so the console DREW hazards the station would
		// never broadcast.
		//
		// HUM LEAD, 2026-09-13, describing the path exactly: "Starting watchpost →
		// Defaults to Observer → Alerts queue according to Observer's alert radius
		// → User ctrl+b → Broadcaster UI loads → Meanwhile the alerts from Observer
		// carry over". `refence` marks them on the swap; nothing acted on the mark
		// where the operator could see it.
		//
		// A CARD ON THE AIR IS STILL DRAWN, for the same reason `refence` will not
		// mark one: cutting a hazard out of the frame mid-sentence is a worse
		// answer than the one this fixes. `refence` never sets the flag on an
		// OnAir card, so this needs no case of its own — recorded because that is
		// a rule held HERE by a rule stated THERE.
		if c.OutOfFence {
			continue
		}
		out = append(out, c)
	}
	// THE PROJECTION NEVER INVENTS A CARD. It is a filter, and a filter that
	// returned more than it was given would be the "second schedule" the
	// 2026-09-05 ruling refuses by name.
	if err := invariant.Check(len(out) <= len(l.tracks[t]), "the line-up is a filter of the schedule, never an addition to it"); err != nil {
		return nil
	}
	return out
}

// scheduleIndex is where projection position `to` sits in the schedule.
//
// THE ONE OWNER OF THE TRANSLATION, and the defect it exists to prevent is
// FR-3.3's: `Moved`'s own comment says "the console already knows every slot
// number it drew", and those are PROJECTION numbers. Indexing the schedule with
// one puts the card somewhere the operator did not ask for — an action shown as
// taken that the schedule took differently, which is silent because both numbers
// are valid.
//
// A position PAST the last visible card means the end of the schedule; anything
// else means "immediately before the card the operator can see there", which
// leaves the Director's structural cards ahead of it undisturbed. Whether they
// SHOULD stay there is the join derivation's business, not this function's.
func (l Lineup) scheduleIndex(t Track, to int) (int, bool) {
	seen := 0
	for i, c := range l.tracks[t] { // bounded by the track (P10-02)
		if c.Slot.structural() {
			continue
		}
		if seen == to {
			return i, true
		}
		seen++
	}
	// AN EXACT MATCH IS THE END; ANYTHING FURTHER IS NOT A POSITION (D-149).
	//
	// THIS FUNCTION SERVES TWO DIFFERENT QUESTIONS and must stay strict for one
	// of them. `Reorder` asks where an EXISTING card goes among the cards the
	// operator can SEE — a slot the running order never drew is meaningless and
	// is refused (TestAMoveBeyondTheProjectionIsRefused). `Insert` asks where a
	// NEW card goes, and there "past the last card" has an obvious meaning: the
	// bottom.
	//
	// SO THE CLAMP LIVES AT THE REQUEST PATH, NOT HERE. A first fix relaxed this
	// to `to >= seen` and broke the move rule in the same edit — one function
	// answering two questions, given one answer.
	if to == seen {
		return len(l.tracks[t]), true // the end of the running order
	}
	return 0, false
}
