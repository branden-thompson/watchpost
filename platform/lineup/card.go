// Package lineup is the Director's schedule as PURE VALUES — the cards, the
// states they move through, and the two tracks they sit on. Nothing here starts
// a goroutine, reads a clock or performs I/O. The Director decides with these
// values and a separate pump does the work (Approach C, director-architecture.md),
// which is what makes the schedule assertable directly: a test states the
// arrangement it wants and reads back what would be spoken, with no timers.
//
// It lives under platform/ because BOTH surfaces need it. Observer schedules a
// watchlist rotation and an alert rail; a Broadcaster station will schedule a
// whole day. scripts/lint-imports.sh forbids anything under modes/ from
// importing domains/*, and it now forbids the same from platform/ — so a card
// is DOMAIN-FREE by construction. A card names a SLOT; the radio domain maps
// that slot to a cast.Role and thence to a voice, from the cast settings it
// already owns (cast.Resolve, app/cast.go:radioDeck.resolveVoice). ReadBy is the plain display
// name that resolution puts back on the card, and empty means unresolved.
//
// The fields are exported and the lineup's storage is not, deliberately. A Card
// is what the Operator must see (DR-6), so it reads as data; the rules that
// matter are enforced where an edit becomes REAL, which is Lineup.Set. Editing
// a local copy of a value harms nobody. Editing the schedule is the thing that
// has to be refused.
package lineup

import (
	"slices"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// State is where a card is in its life. The zero value is Proposed, which is
// where every card genuinely starts.
type State int

const (
	Proposed State = iota
	Admitted
	Refused
	Standby
	OnAir
	Done
	Discarded

	// numStates bounds every table below; it is not itself a state.
	numStates
)

// stateRow is one row of the state registry: what the state is called, and the
// complete set of states a card may move to from it. The table IS the machine —
// there is no second place where a transition is decided.
type stateRow struct {
	name string
	next []State
}

// states is the registry, indexed by State. A function rather than a package
// variable (P10-06), and the only description of the card's life.
//
// REFUSED, DONE and DISCARDED lead nowhere: a card that could be revived is a
// card that can be read twice.
func states() [numStates]stateRow {
	return [numStates]stateRow{
		Proposed: {name: "PROPOSED", next: []State{Admitted, Refused}},
		Admitted: {name: "ADMITTED", next: []State{Standby, Discarded}},
		Refused:  {name: "REFUSED"},
		Standby:  {name: "STANDBY", next: []State{OnAir, Discarded}},
		// BOTH EXITS FROM THE AIR ARE REAL (DR-24). Read in full is DONE;
		// discarded, superseded, cancelled and context-ended are all the same
		// transition. The architecture's first state diagram drew only DONE,
		// which would have left a superseded takeover unexpressible and the
		// paired release effect with nothing to hang on.
		OnAir:     {name: "ON AIR", next: []State{Done, Discarded}},
		Done:      {name: "DONE"},
		Discarded: {name: "DISCARDED"},
	}
}

// String is the state's name, and empty for anything outside the registry —
// a State crosses package boundaries and a hand-edited file must never crash
// the station.
func (s State) String() string {
	if s < 0 || s >= numStates {
		return ""
	}
	return states()[s].name
}

// CanBecome reports whether a card at s may move to next. Both ends are
// checked: an out-of-range state moves nowhere and is reached from nowhere.
func (s State) CanBecome(next State) bool {
	if s < 0 || s >= numStates {
		return false
	}
	if next < 0 || next >= numStates {
		return false
	}
	return slices.Contains(states()[s].next, next)
}

// Slot is what kind of read a card is. The zero value is a location report,
// which is most of the broadcast.
type Slot int

const (
	LocationReport Slot = iota
	SevereRead
	BreakingAlert
	Transition

	// numSlots bounds the registry; it is not itself a slot.
	numSlots
)

// slotRow is one row of the slot registry.
//
// A field left at its zero value MEANS something: a slot that neither spends the
// Max nor composes at standby is one of the Director's own structural cards.
type slotRow struct {
	label string

	// alertRead marks a slot that SPENDS THE MAX (DR-15). Heads, transitions
	// and the divert notice are the Director's arrangement, not alert reads, so
	// a Max of five reads five alerts however many structural cards surround
	// them. A location report is not an alert read either — the Max bounds the
	// burst, not the rotation.
	alertRead bool

	// structural marks one of THE DIRECTOR'S OWN CARDS — a card the schedule
	// reads but the operator never asked for and never sees (D-44).
	//
	// STATED, NOT INFERRED. This row's own comment already said "a slot that
	// neither spends the Max nor composes at standby is one of the Director's
	// own structural cards" — a rule carried by the CONJUNCTION OF TWO ZERO
	// VALUES, which is a rule nobody can find and any new slot can break by
	// accident. It is the filter the operator's whole running order is derived
	// from, so it gets a field.
	structural bool

	// announced marks a kind that does NOT introduce itself, so the listener is
	// told what is coming — "please standby for station identification" (D-49).
	//
	// MVS-D-80'S REASON, GENERALISED. It rules that a transition does not fire
	// location-to-location "because the location scripts already announce their
	// location", which makes self-announcement the property that matters, and a
	// property of the KIND rather than of a pair.
	//
	// NOTHING SETS IT YET, deliberately: station credits will, and credits have
	// no slot until D-31. A rule nobody has made is not a rule.
	announced bool

	// handsBack marks a kind that INTERRUPTED the programme, so the listener is
	// handed back when it ends — "we now return to our regularly scheduled
	// programming" (MVS-D-80).
	handsBack bool

	// textAtStandby marks a slot whose words are composed as the card nears the
	// air (DR-7), rather than when it is proposed. Reports go this way: a report
	// composed at admission says what the weather was when it was queued, not
	// when it plays. Structural cards go the other way — the divert notice's
	// count is decided when the burst is planned (DR-14), so its words are fixed
	// at proposal and nothing may rewrite them later.
	textAtStandby bool
}

// slots is the registry, indexed by Slot.
//
// THESE ARE THE SLOTS 0.14.0 ACTUALLY PRODUCES, and no others. A Marine Report
// as its own card, an operator-requested read, a station identification — all
// arrive with the Broadcaster surface that proposes them. A slot nobody proposes
// is dead code (AP-DEAD-01).
func slots() [numSlots]slotRow {
	return [numSlots]slotRow{
		LocationReport: {label: "Location Report", textAtStandby: true},
		SevereRead:     {label: "Severe-event Read", alertRead: true, textAtStandby: true},
		BreakingAlert:  {label: "Breaking Alert", alertRead: true, textAtStandby: true, handsBack: true},
		Transition:     {label: "Transition", structural: true},
	}
}

// row is the one safe read of the slot registry (metric D, 2026-09-08).
//
// THREE ACCESSORS CARRIED THIS, not two: String, CountsAgainstMax and
// textAtStandby each repeated the range guard and the same invariant, and a
// fourth would have repeated it again. The duplicate detector found two of
// them; the third differed only in the field it returned.
//
// A HOLE IN THE TABLE IS CAUGHT WHERE IT IS USED: a slot added to the enum
// without a row beside it would otherwise be an unnamed card in the log and a
// blank line in the rail, with nothing to say which slot it was.
func (s Slot) row() (slotRow, bool) {
	if s < 0 || s >= numSlots {
		return slotRow{}, false
	}
	r := slots()[s]
	if err := invariant.Check(r.label != "", "every slot in the enum has a row in the registry"); err != nil {
		return slotRow{}, false
	}
	return r, true
}

// String is the slot's name as the Operator reads it.
func (s Slot) String() string {
	r, ok := s.row()
	if !ok {
		return ""
	}
	return r.label
}

// CountsAgainstMax reports whether this slot spends the burst's budget (DR-15).
func (s Slot) CountsAgainstMax() bool {
	r, ok := s.row()
	return ok && r.alertRead
}

// structural reports whether this is one of the Director's own cards, which the
// operator neither sees nor addresses. Unexported for the reason textAtStandby
// is: it is the card model's own rule, asked through Lineup.Projection.
func (s Slot) structural() bool {
	r, ok := s.row()
	return ok && r.structural
}

// announced reports whether the listener is told this kind is coming.
func (s Slot) announced() bool {
	r, ok := s.row()
	return ok && r.announced
}

// handsBack reports whether the listener is handed back to the programme when a
// card of this kind ends.
func (s Slot) handsBack() bool {
	r, ok := s.row()
	return ok && r.handsBack
}

// textAtStandby reports whether this slot's words are composed as it nears the
// air. Unexported: it is the card model's own rule, not a question anyone
// outside asks.
func (s Slot) textAtStandby() bool {
	r, ok := s.row()
	return ok && r.textAtStandby
}

// Origin is who put the card forward (DR-4).
type Origin int

const (
	// FromObserver is the zero value because Observer proposes most of what is
	// read: it fetches, normalises and composes.
	FromObserver Origin = iota
	FromOperator
	FromDirector

	// numOrigins bounds the registry; it is not itself an origin.
	numOrigins
)

// originNames is the registry, indexed by Origin.
func originNames() [numOrigins]string {
	return [numOrigins]string{
		FromObserver: "OBSERVER",
		FromOperator: "OPERATOR",
		FromDirector: "DIRECTOR",
	}
}

// String is the origin's name, and empty for anything outside the registry.
func (o Origin) String() string {
	if o < 0 || o >= numOrigins {
		return ""
	}
	name := originNames()[o]
	if err := invariant.Check(name != "", "every origin in the enum has a row in the registry"); err != nil {
		return ""
	}
	return name
}

// Card is one read: what it is, who wants it, what will be said, and where in
// its life it is. Everything the Operator must see (DR-6) is here and readable.
type Card struct {
	// ID addresses the card for the life of the lineup. Two cards may not share
	// one, because an ambiguous address is a card read twice.
	ID string

	// Slot is what kind of read this is; Origin is who proposed it. Neither
	// changes once the lineup is holding the card.
	Slot   Slot
	Origin Origin

	// Subject is what the card is about — the location, the alert. Headline is
	// the one line that names it before there is a script, so a queued card can
	// be shown, logged and counted from the moment it exists (DR-7).
	Subject, Headline string

	// Refs are the PRODUCER'S RECORDS this card reads, in read order — for a
	// takeover, the alert ids the burst is made of (MVS-D-77).
	//
	// THE CARD STAYS DOMAIN-FREE (DR-1), so these are identifiers and nothing
	// more. The Director owns the ORDER (it planned it); whoever proposed the
	// card owns what each id IS, exactly as the [space] read keeps its own row.
	// The Composer needs both halves and can hold neither: it is handed the
	// order on the effect and asks the producer for the events.
	//
	// EMPTY FOR A CARD THAT IS ABOUT ITSELF — a location report, a transition.
	// Only a card assembled FROM producer records carries them. (It said "a
	// burst head" too; BurstHead was retired when a burst became one card.)
	Refs []string

	// Divert is how many alerts this burst COULD have read and did not — the
	// figure the listener is told aloud (DR-14).
	//
	// IT TRAVELS ON THE CARD because only the Director knows it: it is the
	// difference between what the fence admitted and what the Max allowed, and
	// a Composer that recomputed it would be a second planner disagreeing with
	// the first about a number that is SPOKEN. Zero means nothing was left out,
	// which is a real answer and not an absent one.
	//
	// Counted against the CANDIDATES, not the raw arrivals: a Forecast the
	// listener never lost must not be spoken as an alert they did (m72).
	Divert int

	// Max bounds this one card's read. ZERO MEANS FULL LENGTH, which is the
	// default and, in 0.14.0, the only value: the per-card control arrives with
	// the Broadcaster UI, and the field is carried now because the card model is
	// the part of that surface this release builds (recorded in the build plan's
	// "not built" table, ratified).
	Max time.Duration

	// ReadBy is the display name of the voice that will read this card, resolved
	// against what THIS machine has. Empty means unresolved — the Broadcaster
	// surface renders that as `N/A`. The card stays domain-free: the radio
	// domain resolves Slot to a cast role and puts the resulting name back here.
	ReadBy string

	// Script is the card's words, IN PARTS (MVS-D-77, T3.8) — what the Reader
	// says and what the Broadcaster displays, from one representation so the
	// two cannot drift. Empty until standby for a report (DR-7), fixed at
	// proposal for a structural card; slotRow.textAtStandby says which.
	//
	// A string would not do: MVS-D-72 gives a takeover an internal shape the
	// listener hears — tone, header, lines, tail — and inferring those breaks
	// from newlines works until an alert contains one.
	Script Script

	// BuiltAt is when this card's words came home, and ZERO when they never
	// did — a structural card whose text was fixed at proposal was never built
	// and can never go stale (PD-3). It is the only reason the staleness check
	// cannot discard the very transition it raises.
	BuiltAt time.Time

	// State is where the card is in its life, and the only thing about it that
	// may change while it is on the air.
	State State
}

// Propose returns a well-formed card at PROPOSED, or the reason it is not one.
// Every card enters through here, so the rules below hold for anything the
// Director will ever look at.
func Propose(c Card) (Card, error) {
	if err := invariant.Check(c.State == Proposed, "a proposal starts at PROPOSED"); err != nil {
		return Card{}, err
	}
	// The voice is resolved against this machine as the card nears the air, so a
	// proposal naming one has resolved it too early — against a cast that may
	// have changed by the time it plays.
	if err := invariant.Check(c.ReadBy == "", "a proposal names no voice; the voice is resolved before the air"); err != nil {
		return Card{}, err
	}
	// DR-7's first half: a report carries no words yet. The second half — a
	// card whose words are fixed at proposal must arrive carrying them — lives
	// in check, which every write goes through, so it is NOT repeated here.
	// Its mutant survived while it was: deleting a guard that another guard
	// already enforces changes nothing, which is the rule written twice rather
	// than an invariant.
	//
	// It names the structural slot LITERALLY rather than calling
	// Slot.textAtStandby(), because P10-05 counts only call-free conditions;
	// that trade is the one ratified at T1.2, and
	// TestOnlyAReportComposesItsTextAtStandby walks every slot in the registry
	// so the two drifting apart fails a test rather than passing silently.
	// prepareNext, which has no such constraint, asks the registry directly.
	//
	// ONE STRUCTURAL SLOT NOW, not three. BurstHead and DivertNotice were
	// retired at the T3.10 red team: nothing proposed either, and the registry
	// says in as many words that a slot nobody proposes is dead code
	// (AP-DEAD-01). A burst's head and its divert tail are INTRA-CARD content
	// under MVS-D-77 — parts of the takeover's script — which is what T3.7 said
	// they would become and what the Composer now builds.
	if err := invariant.Check(c.Words() == "" || c.Slot == Transition,
		"a report's words materialise at standby, never at proposal"); err != nil {
		return Card{}, err
	}
	if err := invariant.Check(c.Subject != "" || c.Words() != "", "a card says what it is about, or says its own words"); err != nil {
		return Card{}, err
	}
	if err := c.check(); err != nil {
		return Card{}, err
	}
	return c, nil
}

// check is the well-formedness every card holds for its whole life. Propose
// applies it at the door and the Lineup applies it at every write, so a card
// built as a struct literal cannot get into a schedule.
func (c Card) check() error {
	if err := invariant.Check(c.ID != "", "every card carries an identity"); err != nil {
		return err
	}
	if err := invariant.Check(c.Slot >= 0 && c.Slot < numSlots, "every card names a slot in the registry"); err != nil {
		return err
	}
	if err := invariant.Check(c.Origin >= 0 && c.Origin < numOrigins, "every card names one of the three origins"); err != nil {
		return err
	}
	if err := invariant.Check(c.State >= 0 && c.State < numStates, "every card is in a declared state"); err != nil {
		return err
	}
	if err := invariant.Check(c.Max >= 0, "a card's max duration is never negative"); err != nil {
		return err
	}
	if err := invariant.Check(c.Headline != "", "every card carries a headline from the moment it is proposed"); err != nil {
		return err
	}
	// DR-7's structural half, HERE RATHER THAN ONLY AT Propose. Queue and Set
	// are doors too: a wordless burst head queued directly reaches standby,
	// describes no build, and stands there for ever with the rail stopped
	// behind it. check runs on every write, so this closes all of them.
	if err := invariant.Check(c.Words() != "" || c.Slot != Transition,
		"a card whose words are fixed at proposal never exists without them"); err != nil {
		return err
	}
	// D-42 (HUM LEAD, 2026-09-10): "transition cards are NEVER
	// Origin.fromOperator." A transition is the DIRECTOR'S one additive act —
	// the role model's own words — so an operator-originated one is a category
	// error: the human asks for a report, and the hand-off around it is the
	// Director's consequence of that choice, never the request itself.
	//
	// IT HELD BY ACCIDENT BEFORE THIS LINE. The guard above refuses a wordless
	// transition, and the undo deliberately drops the words, so the one path
	// that could have built such a card failed for an unrelated reason — a rule
	// held by a DIFFERENT rule, which is the shape this package keeps having to
	// un-split. Stated here, it survives the day a transition carries its words
	// through.
	if err := invariant.Check(c.Slot != Transition || c.Origin != FromOperator,
		"a transition is the Director's own structural card; the operator never originates one"); err != nil {
		return err
	}
	return nil
}

// Locked reports whether the card refuses edits.
//
// DERIVED FROM THE STATE, NEVER STORED BESIDE IT. A card is locked exactly while
// it is on the air, and a stored flag would be a second carrier of one rule —
// which is how the duck came to be lifted by one spelling of tune and not the
// other. An unknown state locks: the safe direction is to refuse the edit.
func (c Card) Locked() bool {
	if err := invariant.Check(c.State >= 0 && c.State < numStates, "a card is in a declared state"); err != nil {
		return true
	}
	return c.State == OnAir
}

// To moves the card along the declared path, or refuses.
//
// THE LOCK DOES NOT APPLY HERE. A card on the air must be able to leave it —
// finished, discarded or superseded (DR-24) — and a lock that could hold a card
// on the air would wedge the station, which is the failure this package exists
// to make impossible.
func (c Card) To(next State) (Card, error) {
	if err := invariant.Check(next >= 0 && next < numStates, "a card only moves to a declared state"); err != nil {
		return c, err
	}
	// A takeover that reaches the air with nothing to say is silence where the
	// ticker has already promised a callout (DR-18, DR-24).
	if err := invariant.Check(next != OnAir || c.Words() != "", "a card takes the air with its words already on it"); err != nil {
		return c, err
	}
	if err := c.check(); err != nil {
		return c, err
	}
	if err := invariant.Check(c.State.CanBecome(next), "a card moves only along the declared path"); err != nil {
		return c, err
	}
	c.State = next
	return c, nil
}

// WithText fills in the script. This is DR-7's moment: the card is at standby,
// its data was fetched just now, and what it will say is decided from that
// rather than from whatever was true when it was queued.
func (c Card) WithScript(script Script, builtAt time.Time) (Card, error) {
	if err := invariant.Check(!script.Empty(), "a card's words are never set to nothing"); err != nil {
		return c, err
	}
	// THE STAMP IS NOT OPTIONAL. A build that records no time is a card that can
	// never be judged stale, which is the failure PD-3 exists to prevent and
	// would be invisible — it reads correctly and goes off quietly months later.
	// Taking it as an argument here means a caller cannot forget it.
	if err := invariant.Check(!builtAt.IsZero(), "a build records when it came home"); err != nil {
		return c, err
	}
	if err := invariant.Check(c.State == Standby, "a report's words materialise at standby"); err != nil {
		return c, err
	}
	if err := invariant.Check(c.Slot != Transition,
		"a structural card's words are fixed when it is proposed"); err != nil {
		return c, err
	}
	if err := c.check(); err != nil {
		return c, err
	}
	c.Script, c.BuiltAt = script, builtAt
	return c, nil
}

// WithReadBy records which voice will read this card, once the radio domain has
// resolved the slot against this machine's cast. Before the air, and only
// before: the voice a listener is hearing does not change mid-read.
func (c Card) WithReadBy(voice string) (Card, error) {
	if err := invariant.Check(voice != "", "a card's voice is never set to nothing; unresolved is the empty string"); err != nil {
		return c, err
	}
	if err := invariant.Check(c.State == Admitted || c.State == Standby, "the voice is resolved before the card takes the air"); err != nil {
		return c, err
	}
	if err := c.check(); err != nil {
		return c, err
	}
	c.ReadBy = voice
	return c, nil
}

// Words is the card's words as one string — for a log line, a summary, an
// invariant. COMPUTED from the parts rather than stored beside them, so there
// is no second copy to fall out of step.
func (c Card) Words() string { return c.Script.Text() }
