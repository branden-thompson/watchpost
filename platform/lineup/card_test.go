package lineup

import (
	"slices"
	"testing"
	"time"
)

// report is a proposal for a location report — the main track's ordinary card,
// and the shape most of the broadcast has.
func report(id string) Card {
	return Card{ID: id, Slot: LocationReport, Origin: FromObserver,
		Subject: id, Headline: "Conditions for " + id}
}

// alert is a proposal for the alert rail.
func alert(id string) Card {
	return Card{ID: id, Slot: BreakingAlert, Origin: FromObserver,
		Subject: id, Headline: "Severe thunderstorm warning for " + id}
}

// proposed runs a card literal through the real constructor. Every test starts
// here rather than from a struct literal, so a rule Propose enforces cannot be
// stepped around by the tests that are meant to check it.
func proposed(t *testing.T, c Card) Card {
	t.Helper()
	out, err := Propose(c)
	if err != nil {
		t.Fatalf("Propose(%s): %v", c.ID, err)
	}
	return out
}

// at walks a card to a state along the declared path, failing the test at the
// first refusal. The path is spelled out by the caller so a test reads as the
// sequence it means.
func at(t *testing.T, c Card, path ...State) Card {
	t.Helper()
	for _, s := range path {
		next, err := c.To(s)
		if err != nil {
			t.Fatalf("%s: To(%v) from %v: %v", c.ID, s, c.State, err)
		}
		c = next
	}
	return c
}

// TestTheCardStatesAdvanceOnlyAlongTheDeclaredPath restates the transition
// table in the test, deliberately. It is not proving the table is right — it is
// making the table a CONTRACT: changing what a card may do next has to be done
// in two places, on purpose, and a change made in only one fails here. The
// mutants for this task are what prove the assertion can fail at all.
func TestTheCardStatesAdvanceOnlyAlongTheDeclaredPath(t *testing.T) {
	legal := map[State][]State{
		Proposed:  {Admitted, Refused},
		Admitted:  {Standby, Discarded},
		Standby:   {OnAir, Discarded},
		OnAir:     {Done, Discarded},
		Refused:   nil,
		Done:      nil,
		Discarded: nil,
	}
	for from := State(0); from < numStates; from++ {
		for to := State(0); to < numStates; to++ {
			want := slices.Contains(legal[from], to)
			if got := from.CanBecome(to); got != want {
				t.Errorf("%v.CanBecome(%v) = %v, want %v", from, to, got, want)
			}
		}
	}
}

// TestNoCardReachesTheAirWithoutPassingThroughStandby is DR-7's reason stated as
// a path property rather than a table row: standby is where the text
// materialises, so a card that could skip it could take the air with nothing to
// say.
func TestNoCardReachesTheAirWithoutPassingThroughStandby(t *testing.T) {
	for from := State(0); from < numStates; from++ {
		if from == Standby {
			continue
		}
		if from.CanBecome(OnAir) {
			t.Errorf("%v.CanBecome(OnAir) = true; only STANDBY leads to the air", from)
		}
	}
}

// TestEveryExitFromOnAirIsReachable is DR-24. The architecture's state diagram
// drew only ON AIR -> DONE, but the requirement enumerates five exits — read in
// full, discarded, superseded, cancelled, context ended — and the last four are
// all the same transition. A card that could only leave the air by finishing
// would make a superseded takeover unexpressible, and the release effect DR-24
// pairs with the cue would have nowhere to hang.
func TestEveryExitFromOnAirIsReachable(t *testing.T) {
	for _, want := range []State{Done, Discarded} {
		if !OnAir.CanBecome(want) {
			t.Errorf("OnAir.CanBecome(%v) = false; every exit from the air must be expressible", want)
		}
	}
}

// TestATerminalStateIsTheEndOfTheCard: nothing leaves REFUSED, DONE or
// DISCARDED. A card that could be revived is a card that can be read twice.
func TestATerminalStateIsTheEndOfTheCard(t *testing.T) {
	for _, from := range []State{Refused, Done, Discarded} {
		for to := State(0); to < numStates; to++ {
			if from.CanBecome(to) {
				t.Errorf("%v.CanBecome(%v) = true; %v is terminal", from, to, from)
			}
		}
	}
}

// TestAProposalCarriesItsShapeAndNoWords is DR-7's first half: a card is born
// with its slot, subject and headline, and with the text still empty.
func TestAProposalCarriesItsShapeAndNoWords(t *testing.T) {
	c := proposed(t, report("Bonsall"))
	if c.State != Proposed {
		t.Errorf("State = %v, want %v", c.State, Proposed)
	}
	if c.Script.Text() != "" {
		t.Errorf("Text = %q, want empty until standby", c.Script.Text())
	}
	if c.Headline == "" || c.Subject == "" {
		t.Errorf("Headline = %q, Subject = %q; both are carried from creation", c.Headline, c.Subject)
	}
}

// TestAMalformedProposalIsRefused. Each case is one rule of Propose, and each
// starts from a card that is otherwise valid, so a case can only fail for the
// reason it names.
func TestAMalformedProposalIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(Card) Card
	}{
		{"no identity", func(c Card) Card { c.ID = ""; return c }},
		{"no headline", func(c Card) Card { c.Headline = ""; return c }},
		{"slot outside the registry", func(c Card) Card { c.Slot = numSlots; return c }},
		{"origin outside the registry", func(c Card) Card { c.Origin = numOrigins; return c }},
		{"negative max duration", func(c Card) Card { c.Max = -time.Second; return c }},
		{"already past proposal", func(c Card) Card { c.State = OnAir; return c }},
		{"a voice it cannot yet have", func(c Card) Card { c.ReadBy = "Samantha"; return c }},
		{"a report's words, too early", func(c Card) Card { c.Script = Say("Conditions are fair."); return c }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Propose(tc.edit(report("Bonsall"))); err == nil {
				t.Fatal("Propose accepted it; want a refusal")
			}
		})
	}
}

// TestOnlyAReportComposesItsTextAtStandby pins Propose's DR-7 rule against the
// registry rather than against a list written twice. The rule is spelled out
// call-free inside Propose (P10-05 counts only call-free conditions), so this is
// what catches the two drifting apart when a slot is added.
func TestOnlyAReportComposesItsTextAtStandby(t *testing.T) {
	// BOTH DIRECTIONS, FROM THE REGISTRY. Propose names the structural slots
	// literally, twice, because P10-05 counts only call-free conditions — so
	// this walk is what stops those lists drifting from the registry they
	// stand in for. Pinning only the with-words direction left the second list
	// unpinned: a slot added to the registry and to one list but not the other
	// passed the whole suite, and a wordless structural card then wedged the
	// rail behind it.
	for slot := Slot(0); slot < numSlots; slot++ {
		card := Card{ID: "x", Slot: slot, Headline: "a headline", Subject: "a subject"}

		withWords := card
		withWords.Script = Say("some words")
		_, err := Propose(withWords)
		if slot.textAtStandby() && err == nil {
			t.Errorf("%v was proposed carrying words; a report composes them at standby (DR-7)", slot)
		}
		if !slot.textAtStandby() && err != nil {
			t.Errorf("%v was refused its own words at proposal: %v", slot, err)
		}

		_, err = Propose(card)
		if slot.textAtStandby() && err != nil {
			t.Errorf("%v was refused without words, which is how a report is proposed: %v", slot, err)
		}
		if !slot.textAtStandby() && err == nil {
			t.Errorf("%v was accepted WITHOUT words; it can never be built and would wedge the rail behind it", slot)
		}
	}
}

// TestAReportsTextMaterialisesAtStandbyAndNowhereElse is DR-7's second half.
func TestAReportsTextMaterialisesAtStandbyAndNowhereElse(t *testing.T) {
	c := proposed(t, report("Bonsall"))
	for _, s := range []State{Proposed, Admitted} {
		walked := c
		if s == Admitted {
			walked = at(t, c, Admitted)
		}
		if _, err := walked.WithScript(Say("Conditions are fair."), testBuiltAt); err == nil {
			t.Errorf("WithText accepted at %v; a report's words arrive at standby", s)
		}
	}
	standby := at(t, c, Admitted, Standby)
	built, err := standby.WithScript(Say("Conditions are fair."), testBuiltAt)
	if err != nil {
		t.Fatalf("WithText at standby: %v", err)
	}
	if built.Script.Text() != "Conditions are fair." {
		t.Errorf("Text = %q, want the composed script", built.Script.Text())
	}
	if c.Script.Text() != "" {
		t.Errorf("the proposal it came from now reads %q; a card is a value", c.Script.Text())
	}
}

// TestAStructuralCardCarriesItsWordsFromTheStart: the divert notice's count is
// known when the burst is planned (DR-14), so its text is fixed at proposal and
// the standby rule does not apply to it.
func TestAStructuralCardCarriesItsWordsFromTheStart(t *testing.T) {
	notice := Card{ID: "divert", Slot: Transition, Origin: FromDirector,
		Headline: "and 16 other alerts", Script: Say("For more details about these and 16 other alerts, press w.")}
	c := proposed(t, notice)
	if c.Script.Text() == "" {
		t.Fatal("the notice lost its words at proposal")
	}
	standby := at(t, c, Admitted, Standby)
	if _, err := standby.WithScript(Say("something else"), testBuiltAt); err == nil {
		t.Error("WithText accepted on a structural card; its words are fixed when it is proposed")
	}
}

// TestACardTakesTheAirWithItsWordsAlreadyOnIt. A takeover that reaches the air
// with nothing to say is silence where the ticker has already promised a
// callout (DR-24).
func TestACardTakesTheAirWithItsWordsAlreadyOnIt(t *testing.T) {
	standby := at(t, proposed(t, alert("a1")), Admitted, Standby)
	if _, err := standby.To(OnAir); err == nil {
		t.Fatal("an empty card took the air")
	}
	spoken, err := standby.WithScript(Say("A severe thunderstorm warning is in effect."), testBuiltAt)
	if err != nil {
		t.Fatalf("WithText: %v", err)
	}
	if _, err := spoken.To(OnAir); err != nil {
		t.Fatalf("a card with its words refused the air: %v", err)
	}
}

// TestTheVoiceIsResolvedBeforeTheAirAndNotAfter is DR-6's `read by`: a plain
// display name, put on the card by the radio domain when it resolves the slot
// against this machine's cast. Empty means unresolved.
func TestTheVoiceIsResolvedBeforeTheAirAndNotAfter(t *testing.T) {
	c := proposed(t, report("Bonsall"))
	if c.ReadBy != "" {
		t.Fatalf("ReadBy = %q at proposal, want empty", c.ReadBy)
	}
	admitted := at(t, c, Admitted)
	voiced, err := admitted.WithReadBy("Samantha")
	if err != nil {
		t.Fatalf("WithReadBy at admitted: %v", err)
	}
	if voiced.ReadBy != "Samantha" {
		t.Errorf("ReadBy = %q, want Samantha", voiced.ReadBy)
	}
	onAir := at(t, at(t, voiced, Standby).mustText(t), OnAir)
	if _, err := onAir.WithReadBy("Rishi"); err == nil {
		t.Error("the voice changed while the card was on the air")
	}
}

// mustText fills a card's script so it can be walked to the air.
func (c Card) mustText(t *testing.T) Card {
	t.Helper()
	out, err := c.WithScript(Say("the script"), testBuiltAt)
	if err != nil {
		t.Fatalf("%s: WithText: %v", c.ID, err)
	}
	return out
}

// TestOnlyACardOnTheAirIsLocked. The lock is derived from the state, never
// stored beside it — two carriers of one rule is how the duck regression was
// possible (PD-1's lesson applied to the card).
func TestOnlyACardOnTheAirIsLocked(t *testing.T) {
	for s := State(0); s < numStates; s++ {
		c := report("Bonsall")
		c.State = s
		if got, want := c.Locked(), s == OnAir; got != want {
			t.Errorf("a card at %v: Locked() = %v, want %v", s, got, want)
		}
	}
}

// TestOnlyAnAlertReadSpendsTheMax is DR-15: heads, transitions and the divert
// notice are the Director's arrangement, and Max counts alert reads.
func TestOnlyAnAlertReadSpendsTheMax(t *testing.T) {
	spend := map[Slot]bool{BreakingAlert: true, SevereRead: true}
	for s := Slot(0); s < numSlots; s++ {
		if got, want := s.CountsAgainstMax(), spend[s]; got != want {
			t.Errorf("%v.CountsAgainstMax() = %v, want %v", s, got, want)
		}
	}
}

// TestEveryRowOfEveryRegistryIsFilled, and everything outside them reads as
// empty rather than panicking: a slot, state or origin arrives from a config
// file and a hand-edited file must never crash the station.
func TestEveryRowOfEveryRegistryIsFilled(t *testing.T) {
	for s := Slot(0); s < numSlots; s++ {
		if s.String() == "" {
			t.Errorf("Slot(%d) has no row in the registry", s)
		}
	}
	for s := State(0); s < numStates; s++ {
		if s.String() == "" {
			t.Errorf("State(%d) has no row in the registry", s)
		}
	}
	for o := Origin(0); o < numOrigins; o++ {
		if o.String() == "" {
			t.Errorf("Origin(%d) has no row in the registry", o)
		}
	}
	for _, bad := range []int{-1, -99, 1 << 20} {
		if got := Slot(bad).String(); got != "" {
			t.Errorf("Slot(%d).String() = %q, want empty", bad, got)
		}
		if got := State(bad).String(); got != "" {
			t.Errorf("State(%d).String() = %q, want empty", bad, got)
		}
		if got := Origin(bad).String(); got != "" {
			t.Errorf("Origin(%d).String() = %q, want empty", bad, got)
		}
		if Slot(bad).CountsAgainstMax() {
			t.Errorf("Slot(%d) spends the Max", bad)
		}
	}
	if Slot(numSlots).String() != "" || State(numStates).String() != "" || Origin(numOrigins).String() != "" {
		t.Error("a count sentinel named itself; it is not a member")
	}
}

// TestTheThreeOriginsAreDistinct is DR-4: Observer proposes, the Operator
// proposes, and the Director generates its own structural cards.
func TestTheThreeOriginsAreDistinct(t *testing.T) {
	want := []string{"OBSERVER", "OPERATOR", "DIRECTOR"}
	got := []string{FromObserver.String(), FromOperator.String(), FromDirector.String()}
	if !slices.Equal(got, want) {
		t.Errorf("origins = %v, want %v", got, want)
	}
	if int(numOrigins) != len(want) {
		t.Errorf("the registry holds %d origins, want %d", numOrigins, len(want))
	}
}

// testBuiltAt is a non-zero build stamp for tests that are not about staleness.
// PD-3 makes the stamp mandatory at WithText precisely so it cannot be
// forgotten; a test may not care when, but it may not pretend there was no when.
var testBuiltAt = time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
