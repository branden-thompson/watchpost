package lineup

import (
	"slices"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Track is one of the two queues a card can sit on (DR-3). The bed — the live
// NOAA relays — is deliberately not here: it is a selectable resource the
// Director may cut over to, not a queue of cards, and modelling it as a third
// track would invite something to be scheduled onto it.
type Track int

const (
	MainTrack Track = iota
	AlertRail

	// numTracks bounds the lineup's storage; it is not itself a track.
	numTracks
)

// String names the track for the transition log (DR-23).
func (t Track) String() string {
	if t < 0 || t >= numTracks {
		return ""
	}
	if t == AlertRail {
		return "ALERT RAIL"
	}
	return "MAIN TRACK"
}

// Lineup is the schedule: what is queued, on which track, in what order.
//
// EXACTLY ONE WRITER (DR-1). The storage is unexported, so nothing outside this
// package can assign to it, and every write returns a NEW Lineup rather than
// changing this one — the Director holds the current value and Step returns the
// next. Readers get copies, so a reader cannot become a second writer by
// accident.
//
// The zero value is a usable empty lineup with both tracks present, which is the
// ordinary resting state: the alert rail is always there and usually empty.
type Lineup struct {
	tracks [numTracks][]Card

	// discarded is the operator's undo pile — NOT a track (D-35, discard.go).
	// It is outside `tracks` on purpose: held() counts tracks, and a pile that
	// counted would mean the schedule never reads as stopped.
	discarded []Card
}

// clone copies the storage so a write can be made against the copy. Every
// mutator starts here: a Lineup is a value and the pump holds more than one at a
// time, so two lineups built from one must never share a backing array.
func (l Lineup) clone() Lineup {
	var out Lineup
	for t := range l.tracks { // bounded by the array (P10-02)
		out.tracks[t] = slices.Clone(l.tracks[t])
		// A CLONE THAT LOST A TRACK LOSES A SCHEDULE, silently: every mutator
		// starts here, so a short copy would drop cards that were promised a
		// read and nothing downstream would know they had ever existed.
		if err := invariant.Check(len(out.tracks[t]) == len(l.tracks[t]), "a clone keeps every card on every track"); err != nil {
			return Lineup{}
		}
	}
	// AND IT KEEPS THE PILE. Every mutator starts here, so a short copy would
	// lose the operator's undo silently — the same defect the track check above
	// exists for, one field along.
	out.discarded = slices.Clone(l.discarded)
	if err := invariant.Check(len(out.discarded) == len(l.discarded), "a clone keeps the discard pile"); err != nil {
		return Lineup{}
	}
	return out
}

// find locates a card by identity, returning its track, its index there, and
// whether the lineup holds it at all.
func (l Lineup) find(id string) (Track, int, bool) {
	if id == "" {
		return MainTrack, -1, false
	}
	for t := range l.tracks { // bounded by the array, then by each track (P10-02)
		for i, c := range l.tracks[t] {
			if c.ID == id {
				return Track(t), i, true
			}
		}
	}
	return MainTrack, -1, false
}

// Cards is one track's cards in order — A COPY, always. A slice handed out would
// alias the Director's own storage, and a reader that can write into the
// schedule is the second writer DR-1 exists to make impossible.
func (l Lineup) Cards(t Track) []Card {
	if err := invariant.Check(t >= 0 && t < numTracks, "a track is one of the declared two"); err != nil {
		return nil
	}
	out := slices.Clone(l.tracks[t])
	if err := invariant.Check(len(out) == len(l.tracks[t]), "the copy holds every card the track holds"); err != nil {
		return nil
	}
	return out
}

// Queue puts an admitted card at the end of a track.
//
// ENTRY TO THE LINEUP IS ADMISSION (DR-3). The lineup accepts nothing that has
// not passed the pre-screen, and once a card is here it will be read: bounds
// apply here and nowhere later. That is what removes `breakingCap`'s defect
// rather than moving it — there is no subsequent moment at which a queued hazard
// can be cut, so no hazard can be silenced by sorting last.
func (l Lineup) Queue(t Track, c Card) (Lineup, error) {
	if err := invariant.Check(t >= 0 && t < numTracks, "a card is queued onto one of the declared two tracks"); err != nil {
		return l, err
	}
	if err := invariant.Check(c.State == Admitted, "the lineup holds admitted cards only"); err != nil {
		return l, err
	}
	if err := c.check(); err != nil {
		return l, err
	}
	_, _, taken := l.find(c.ID)
	if err := invariant.Check(!taken, "no two cards in the lineup share an identity"); err != nil {
		return l, err
	}
	out := l.clone()
	out.tracks[t] = append(out.tracks[t], c)
	if err := invariant.Check(len(out.tracks[t]) == len(l.tracks[t])+1, "queueing adds exactly one card"); err != nil {
		return l, err
	}
	return out, nil
}

// insertBeside puts a card immediately ahead of or behind one the lineup already
// holds.
//
// ITS OWN MUTATOR, for the reason Reorder is one: `Queue` appends, and the
// Director's transitions are the one thing that must land at a PLACE rather than
// at the end. It takes the position from an identity rather than an index so no
// caller has to hold a number the schedule may already have changed.
func (l Lineup) insertBeside(t Track, id string, c Card, before bool) (Lineup, error) {
	if err := invariant.Check(t >= 0 && t < numTracks, "a card is inserted onto one of the declared two tracks"); err != nil {
		return l, err
	}
	if err := invariant.Check(c.State == Admitted, "the lineup holds admitted cards only"); err != nil {
		return l, err
	}
	if err := c.check(); err != nil {
		return l, err
	}
	_, _, taken := l.find(c.ID)
	if err := invariant.Check(!taken, "no two cards in the lineup share an identity"); err != nil {
		return l, err
	}
	at, _, held := l.find(id)
	if err := invariant.Check(held && at == t, "a card is inserted ahead of one the track is holding"); err != nil {
		return l, err
	}
	_, i, _ := l.find(id)
	if !before {
		i++
	}
	out := l.clone()
	out.tracks[t] = slices.Insert(out.tracks[t], i, c)
	if err := invariant.Check(len(out.tracks[t]) == len(l.tracks[t])+1, "inserting adds exactly one card"); err != nil {
		return l, err
	}
	return out, nil
}

// held is how many cards the schedule is holding, on both tracks. It is what
// bounds any walk over the schedule (P10-02).
func (l Lineup) held() int {
	n := 0
	for t := range l.tracks { // bounded by the array (P10-02)
		n += len(l.tracks[t])
	}
	return n
}

// toPrepare is the next card preparation should look at, and the track it came
// from — walking the schedule in read order, the alert rail first (DR-3).
//
// IT IS NOT Next. Once a card has been promoted to standby, Next keeps offering
// IT, because it is still on its way to the air; preparation has to look past
// what it has already prepared to find what it has not.
//
// IT STOPS AT A CARD WHOSE WORDS WERE ASKED FOR AND HAVE NOT COME BACK, and
// that is "one ahead, and only one" — the bound expressed as a question about
// the schedule rather than as a counter. A card already on the air, or one
// standing by with its words on it, needs nothing: the walk looks behind it.
func (l Lineup) toPrepare() (Card, Track, bool) {
	for _, t := range []Track{AlertRail, MainTrack} { // the precedence, in one line
		for _, c := range l.tracks[t] { // bounded by the track (P10-02)
			if c.State == Admitted {
				return c, t, true
			}
			// A card standing by whose words were FIXED AT PROPOSAL needs
			// nothing and will be read from where it stands, so the walk looks
			// behind it — that is what lets a burst opening with a head and a
			// transition still get its first alert built in time.
			if c.State == Standby && !c.Slot.textAtStandby() {
				continue
			}
			// A REPORT STANDING BY STOPS THE WALK, whether its words are on the
			// way or already back. Building past it is running ahead of the
			// air, and a report composed several reads before it plays speaks
			// data that was true when it was built (DR-7). This is "one ahead,
			// and only one", asked as a question about the schedule.
			if c.State == Standby {
				return Card{}, t, false
			}
		}
	}
	return Card{}, MainTrack, false
}

// Next is the card that takes the air next, and the track it came from.
//
// THE ALERT RAIL DRAINS FIRST (DR-3). When it holds anything, those cards are
// read first and in order; normal programming resumes only when it is dry.
func (l Lineup) Next() (Card, Track, bool) {
	for _, t := range []Track{AlertRail, MainTrack} { // the precedence, in one line
		for _, c := range l.tracks[t] { // bounded by the track (P10-02)
			// STATED POSITIVELY. Only a card still on its way to the air is
			// offered — a card already on the air is not offered twice, and a
			// finished or discarded one is never re-read. A list of states to
			// SKIP would grow a hole every time a state was added.
			if c.State == Admitted || c.State == Standby {
				return c, t, true
			}
		}
	}
	return Card{}, MainTrack, false
}

// OnAir is the card being read, if one is.
//
// DERIVED, NEVER STORED. The Director could keep the identity beside the lineup,
// and then the two could disagree — which is the shape of every rule this
// release has had to un-split. AT MOST ONE CARD HOLDS THE AIR: two would mean
// two voices, and the invariant says so where it can be seen to fail rather than
// in a comment.
func (l Lineup) OnAir() (Card, bool) {
	var found Card
	seen := 0
	for t := range l.tracks { // bounded by the array, then by each track (P10-02)
		for _, c := range l.tracks[t] {
			if c.State == OnAir {
				found, seen = c, seen+1
			}
		}
	}
	if err := invariant.Check(seen <= 1, "at most one card holds the air"); err != nil {
		return Card{}, false
	}
	return found, seen == 1
}

// Set replaces a card the lineup already holds, in place. This is how a card
// advances, gets its words and gets its voice: the schedule's order is the
// planner's decision and a state change must not reorder it.
func (l Lineup) Set(c Card) (Lineup, error) {
	t, i, held := l.find(c.ID)
	if err := invariant.Check(held, "Set replaces a card the lineup is holding"); err != nil {
		return l, err
	}
	if err := c.check(); err != nil {
		return l, err
	}
	have := l.tracks[t][i]
	// Swapping what a card IS under a live identity would change what is read
	// without changing what the lineup says is read.
	if err := invariant.Check(have.Slot == c.Slot && have.Origin == c.Origin, "a card keeps its slot and its origin for life"); err != nil {
		return l, err
	}
	edited := have.Words() != c.Words() || have.ReadBy != c.ReadBy || have.Max != c.Max ||
		have.Subject != c.Subject || have.Headline != c.Headline
	// THE ON-AIR LOCK (DR-6), and this is the only place it can be enforced:
	// an edit is real when it reaches the schedule. The state is written out
	// rather than calling Locked() because P10-05 counts only call-free
	// conditions; Locked() is the readable predicate for everyone else.
	if err := invariant.Check(have.State != OnAir || !edited, "a card on the air refuses edits; only its state may move"); err != nil {
		return l, err
	}
	out := l.clone()
	out.tracks[t][i] = c
	if err := invariant.Check(len(out.tracks[t]) == len(l.tracks[t]), "Set replaces a card and never adds one"); err != nil {
		return l, err
	}
	return out, nil
}

// Remove takes a card off its track. The Director does this once a card is
// finished or discarded; it is not the Operator's DROP control, which arrives
// with the Broadcaster UI.
func (l Lineup) Remove(id string) (Lineup, error) {
	t, i, held := l.find(id)
	if err := invariant.Check(held, "Remove takes a card the lineup is holding"); err != nil {
		return l, err
	}
	out := l.clone()
	out.tracks[t] = slices.Delete(out.tracks[t], i, i+1)
	if err := invariant.Check(len(out.tracks[t]) == len(l.tracks[t])-1, "Remove takes exactly one card"); err != nil {
		return l, err
	}
	_, _, still := out.find(id)
	if err := invariant.Check(!still, "the card Remove took is gone"); err != nil {
		return l, err
	}
	return out, nil
}
