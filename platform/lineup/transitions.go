package lineup

// transitions.go — the inter-card transition, DERIVED FROM WHAT EACH KIND NEEDS
// AROUND IT (D-49; MVS-D-80, HUM LEAD 2026-09-05).
//
// THE DIRECTOR'S ONE ADDITIVE ACT (role-model.md) — the only place it puts a
// card into the running order because of what the running order IS:
//
//	"only the Director knows that two adjacent cards came from DIFFERENT
//	COMPOSERS and need a handoff between them."
//
// THE TRIGGER IS RULED (MVS-D-80, follow-ups.md F-27). A transition "fires when
// something that INTERRUPTED THE PROGRAMME leaves the air and the programme
// resumes", and it does NOT fire location-to-location — because "the location
// scripts already announce their location". **That reason is the general rule**,
// and it is a property of the KIND rather than of a pair:
//
//	announced — this kind does NOT introduce itself, so the listener is told
//	            what is coming ("please standby for station identification").
//	handsBack — this kind INTERRUPTED the programme, so the listener is handed
//	            back when it ends ("we now return to our regularly scheduled
//	            programming").
//
// SO ADDING AN INTER-CARD CARD IS A ROW, which is the HUM LEAD's requirement in
// their own words: "the system needs to be flexible enough that adding
// additional transition or inter-card cards is low cost, and doesn't require a
// complete rewiring of the line-up flow and logic."
//
// AN EARLIER VERSION FIRED ON `Origin == FromOperator` AND WAS WRONG. D-43's
// bookended operator card was "an example to show the function of the DIRECTOR
// understanding how a card fits into the line-up" — not a rule — and as a rule
// it fired location-to-location, which MVS-D-80 forbids and S-5 independently
// calls "jarring to a listening audience". It was written without reading F-27,
// which is the failure 00-REQUIRED-READING.md exists for.
//
// THE WORDS ARE NOT THE DIRECTOR'S. It owns ARRANGEMENT; the script library owns
// CONTENT, and that is the S-7 boundary T-3 draws. The station hands the lines in
// through Settings, composed by the app from `transition/resume.txt` — so there
// is ONE owner of the sentence and the Director never invents one. With no words
// it arranges nothing, which is the same degradation shape as D-48's cadence
// term: better to move on than to speak a line nobody wrote.
//
// RE-DERIVING IS FREE. `slots()` gives Transition `textAtStandby: false`, so a
// transition's words are fixed at proposal — no composer, no network, nothing to
// wait for. That is what makes recomputing the whole order on every change the
// cheap option rather than the expensive one.
//
// PLACEMENT, AND IT RE-OPENS A RULING. MVS-D-80 put the transition "at the duck"
// in `mastercontrol`, because the lift "is the one point that sees the takeover,
// the [w] read and the relay ALIKE". With the relay case ruled out of scope
// (HUM LEAD 2026-09-10), two of those three are cards — and T-3 says inter-card
// transitions ARE cards, where the Director can adjust and remove them. So the
// takeover's hand-back is a card at the end of the rail. **Flagged for
// ratification**: if it is wrong it costs one registry row.

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// joinPrefix marks the cards THIS FILE OWNS, and ownership by identity is
// load-bearing rather than cosmetic: the reconcile deletes transitions it no
// longer wants, and the staleness notice is a structural card it never made. A
// reconcile that owned every structural card would delete that notice silently.
const joinPrefix = "join:"

const (
	leadKind = "lead" // said BEFORE the card it names
	tailKind = "tail" // said AFTER it
)

// leadID and tailID are the identities of a card's transitions, and both are
// PURE FUNCTIONS OF THE CARD. That is what makes the reconcile idempotent: it
// runs on every settle, so a non-deterministic id would grow the schedule by one
// transition per event.
func leadID(id string) string { return joinPrefix + leadKind + ":" + id }
func tailID(id string) string { return joinPrefix + tailKind + ":" + id }

// isJoinID reports whether an id names a card this file minted.
func isJoinID(id string) bool { return strings.HasPrefix(id, joinPrefix) }

// splitJoinID reads back what a transition was minted for — the last reader of
// the id's shape, and all of them are in this block.
func splitJoinID(id string) (kind, card string, ok bool) {
	if !isJoinID(id) {
		return "", "", false
	}
	kind, card, ok = strings.Cut(strings.TrimPrefix(id, joinPrefix), ":")
	if !ok || card == "" || (kind != leadKind && kind != tailKind) {
		return "", "", false
	}
	return kind, card, true
}

// around is what this card needs said before and after it — THE ONE PLACE THE
// RULE IS APPLIED, reading the one place it is declared.
func (d Director) around(c Card) (lead Card, hasLead bool, tail Card, hasTail bool) {
	// A TRANSITION NEVER BOOKENDS A TRANSITION. Two structural cards in a row is
	// the Director talking to itself, and the registry says as much by leaving
	// Transition's own row empty — this is the tripwire for the day it does not.
	if err := invariant.Check(!c.Slot.structural(),
		"a transition is asked about the running order, where the Director's own cards do not appear"); err != nil {
		return Card{}, false, Card{}, false
	}
	if c.Slot.announced() {
		lead, hasLead = d.mint(leadID(c.ID), c, d.settings.Announcement)
	}
	// THE HAND-BACK IS MINTED WHILE THE INTERRUPTING READ IS STILL ON AIR, and
	// F-27's own design note asks for exactly that: "enqueue it while the
	// interrupting read is still on air, and the duck stays down by itself …
	// the hazard to design against is the DUCK-BOUNCE" — enqueue it after the
	// read has ended and the bed lifts, dips again for the transition, then
	// lifts, which is the dip-lift-dip MVS-D-67 removed and MEASURED.
	//
	// AND IT IS WHAT TELLS "READ" FROM "NEVER PLAYED" WITH NO STORED
	// ASSOCIATION. A takeover dropped or declined before it aired never had a
	// hand-back to strand; one that aired leaves its hand-back at the head of
	// the track, where `stillHolds` keeps it. A first attempt derived the tail
	// from admission instead, and a dropped takeover left a stray "we now
	// return to our regularly scheduled programming" with nothing before it.
	if c.Slot.handsBack() && c.State == OnAir {
		tail, hasTail = d.mint(tailID(c.ID), c, d.settings.ProgrammeReturn)
	}
	return lead, hasLead, tail, hasTail
}

// mint builds one transition card, or says the station gave it nothing to say.
func (d Director) mint(id string, about Card, words string) (Card, bool) {
	if words == "" {
		return Card{}, false // the station has no line for this; say nothing
	}
	card, err := Propose(Card{
		ID: id, Slot: Transition, Origin: FromDirector,
		Subject: about.Subject, Headline: about.Headline, Script: Say(words),
	})
	if err != nil {
		return Card{}, false
	}
	return card, true
}

// placement is one transition and where it goes: the card it attaches to, and
// whether it sits before or after it.
type placement struct {
	card   Card
	anchor string
	before bool
}

// wanted is every transition the running order now calls for, per track.
func (d Director) wanted(t Track) []placement {
	view := d.lineup.Projection(t)
	out := make([]placement, 0, 2*len(view))
	for _, c := range view { // bounded by the running order (P10-02)
		lead, hasLead, tail, hasTail := d.around(c)
		if hasLead {
			out = append(out, placement{card: lead, anchor: c.ID, before: true})
		}
		if hasTail {
			out = append(out, placement{card: tail, anchor: c.ID})
		}
	}
	// AT MOST TWO PER CARD. More would mean a kind had asked for the same side
	// twice, which is a schedule that reads one handoff to itself.
	if err := invariant.Check(len(out) <= 2*len(view), "a running order of n cards needs at most 2n transitions"); err != nil {
		return nil
	}
	return out
}

// reconcileJoins brings the schedule's transitions into line with what the
// running order now calls for. It is the whole of "they move with the card" and
// "they simply go away" — neither is implemented, both fall out, because there
// is no association for `Reorder`, `onDropped` or the undo to honour.
//
// IT RUNS FIRST IN `settle`, before anything reads the order, because a
// transition added now may be the very next thing spoken.
//
// IT IS NOT GATED ON `advances`. These are consequences of cards the schedule
// has ALREADY admitted, so DR-3's promise was made when the card was let in. A
// stopped station still has a running order; it simply is not reading it.
func (d Director) reconcileJoins() Director {
	for t := Track(0); t < numTracks; t++ { // bounded by the registry (P10-02)
		d.lineup = d.lineup.dropStaleJoins(t)
		for _, p := range d.wanted(t) { // bounded by the running order (P10-02)
			d.lineup = d.lineup.addJoin(t, p)
		}
	}
	return d
}

// stillHolds reports whether the transition at index i still sits where it was
// minted to sit.
//
// ASKED OF THE SCHEDULE, NOT OF THE RUNNING ORDER, and that distinction is a
// DEFECT THIS FILE ALREADY HAD. Deriving the wanted set from the running order
// alone prunes a hand-back the moment the card it hands back FROM is read: the
// takeover finishes, leaves the schedule, and the transition is deleted in the
// same settle — one step before it would have been spoken. The listener hears
// the programme resume with no hand-back, which is the precise thing MVS-D-80
// exists to prevent.
//
// So a TAIL survives at the head of its track: everything before it has been
// read, which is the only way its card can legitimately have vanished. One whose
// card was DROPPED from the middle is not at the head, and goes — the difference
// between "already read" and "taken away", as a position rather than a flag
// anyone maintains.
func (l Lineup) stillHolds(t Track, i int) bool {
	track := l.tracks[t]
	kind, about, ok := splitJoinID(track[i].ID)
	if !ok {
		return false
	}
	if kind == leadKind {
		// A LEAD MUST STILL INTRODUCE WHAT FOLLOWS IT. Announcing a card that is
		// no longer next is worse than announcing nothing.
		return i+1 < len(track) && track[i+1].ID == about
	}
	if i == 0 {
		return true // the card it hands back from has been read; this is due
	}
	return track[i-1].ID == about
}

// dropStaleJoins removes the transitions this file minted that the order no
// longer calls for.
//
// TWO THINGS IT WILL NOT TOUCH. A structural card it did not mint — the
// staleness notice — because ownership is by identity. And a card ON THE AIR
// (D-45): the listener is mid-sentence.
func (l Lineup) dropStaleJoins(t Track) Lineup {
	out := l.clone()
	kept := make([]Card, 0, len(out.tracks[t]))
	for i, c := range out.tracks[t] { // bounded by the track (P10-02)
		if isJoinID(c.ID) && c.State != OnAir && !l.stillHolds(t, i) {
			continue
		}
		kept = append(kept, c)
	}
	out.tracks[t] = kept
	// IT ONLY EVER SHORTENS. A prune that added a card would be inventing a
	// running order rather than tidying one.
	if err := invariant.Check(len(kept) <= len(l.tracks[t]), "pruning the transitions never adds a card"); err != nil {
		return l
	}
	return out
}

// addJoin puts one transition beside the card it belongs to, if the schedule is
// not already holding it.
func (l Lineup) addJoin(t Track, p placement) Lineup {
	if _, _, held := l.find(p.card.ID); held {
		return l // already there; the id is a pure function of the card
	}
	admitted, err := p.card.To(Admitted)
	if err != nil {
		return l
	}
	next, err := l.insertBeside(t, p.anchor, admitted, p.before)
	if err != nil {
		return l
	}
	return next
}
