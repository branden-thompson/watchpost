package lineup

import (
	"strings"
	"testing"
	"time"
)

// PD-3 — A CARD WHOSE WORDS OUTLIVED THEIR VALIDITY IS NOT READ.
//
// The table below is the whole rule, written before the code was. Each row is a
// STANDBY card at the moment the air comes free; the outcome is what a listener
// gets.
//
//	# | built            | age        | programme | outcome
//	--+------------------+------------+-----------+---------------------------
//	1 | never (zero)     | —          | running   | reads — it was never built,
//	  |                  |            |           | so it cannot have gone off
//	2 | yes              | ≤ 15 min   | running   | reads
//	3 | yes              | = 15 min   | running   | reads — EXCEEDED, not reached
//	4 | yes              | > 15 min   | running   | dropped; the notice reads
//	5 | yes              | > 15 min   | stopped   | nothing; it is waiting, not
//	  |                  |            |           | stale, and is judged on resume
//	6 | several, all old | > 15 min   | running   | all dropped, ONE notice
//	7 | the notice       | —          | running   | never stale (row 1), which is
//	  |                  |            |           | what stops this looping
//
// Row 7 is the one worth writing the table for. The notice is a card on the same
// schedule; if it could go stale it would raise a second notice, which could go
// stale in turn. Its words are fixed at proposal, so BuiltAt is zero, so row 1
// covers it — the loop is closed by the model rather than by a special case.

// stagedOn is a director holding ONE built card in standby on the given track,
// built `age` ago, with the programme running.
//
// The track matters and is not a detail: advances(AlertRail) is unconditionally
// true, because the rail drains whatever the listener has done to the
// programme. So a stopped-programme case can only be posed on the MAIN TRACK —
// posing it on the rail tests the rail's always-advance rule instead, which is
// the mistake the first version of row 5 made.
func stagedOn(t *testing.T, track Track, age time.Duration) Director {
	t.Helper()
	d := New(Settings{Max: 10}, planNow)
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, _ = run(d, Powered{To: Running})

	slot := LocationReport
	if track == AlertRail {
		slot = BreakingAlert
	}
	c, err := Propose(Card{ID: "staged", Slot: slot, Origin: FromDirector,
		Subject: "Oceanside, CA", Headline: "Oceanside, CA"})
	if err != nil {
		t.Fatalf("staging: %v", err)
	}
	admitted, err := c.To(Admitted)
	if err != nil {
		t.Fatalf("staging: %v", err)
	}
	queued, err := d.lineup.Queue(track, admitted)
	if err != nil {
		t.Fatalf("staging: %v", err)
	}
	d.lineup = queued
	standby, err := admitted.To(Standby)
	if err != nil {
		t.Fatalf("staging: %v", err)
	}
	built, err := standby.WithScript(Say("the report"), planNow.Add(-age))
	if err != nil {
		t.Fatalf("staging: %v", err)
	}
	set, err := d.lineup.Set(built)
	if err != nil {
		t.Fatalf("staging: %v", err)
	}
	d.lineup = set
	return d
}

func staged(t *testing.T, age time.Duration) Director {
	t.Helper()
	return stagedOn(t, AlertRail, age)
}

func TestPD3AStaleCardIsDroppedAndTheListenerIsTold(t *testing.T) {
	// Row 4: the card is older than the window.
	d := staged(t, StaleAfter+time.Second)
	d, fx := d.takeTheAir()

	spoke := ""
	for _, f := range fx {
		if s, ok := f.(Speak); ok {
			spoke = s.Script.Text()
		}
	}
	if spoke != staleTransitionText {
		t.Fatalf("the listener hears the notice, not the stale report; heard %q", spoke)
	}
	if _, busy := onAirAnywhere(d.lineup); !busy {
		t.Error("the notice takes the air in the same step, or the gap it explains comes first")
	}
	// The stale card is GONE, not merely skipped: a card left in the schedule
	// is a card the next pass offers again.
	for tr := Track(0); tr < numTracks; tr++ {
		for _, c := range d.lineup.Cards(tr) {
			if c.ID == "staged" {
				t.Errorf("the stale card is still held: %s in %v", c.ID, c.State)
			}
		}
	}
}

func TestPD3AFreshCardIsReadNormally(t *testing.T) {
	// Rows 2 and 3: inside the window, and exactly on it.
	for _, age := range []time.Duration{time.Minute, StaleAfter} {
		d := staged(t, age)
		_, fx := d.takeTheAir()
		spoke := ""
		for _, f := range fx {
			if s, ok := f.(Speak); ok {
				spoke = s.Script.Text()
			}
		}
		if spoke == staleTransitionText {
			t.Errorf("a card built %v ago is fresh — the window is EXCEEDED, not reached", age)
		}
		if spoke != "the report" {
			t.Errorf("a card built %v ago reads its own words, heard %q", age, spoke)
		}
	}
}

func TestPD3AStoppedProgrammeDoesNotJudgeStaleness(t *testing.T) {
	// Row 5: stopped. The card is waiting, not stale, and must survive to be
	// judged when the listener resumes — dropping it here would discard a
	// schedule nobody abandoned.
	d := stagedOn(t, MainTrack, StaleAfter+time.Hour)
	d, _ = run(d, Powered{To: Stopped})
	d, fx := d.takeTheAir()
	for _, f := range fx {
		if _, ok := f.(Speak); ok {
			t.Fatal("a stopped programme puts nothing to air")
		}
	}
	found := false
	for tr := Track(0); tr < numTracks; tr++ {
		for _, c := range d.lineup.Cards(tr) {
			if c.ID == "staged" {
				found = true
			}
		}
	}
	if !found {
		t.Error("the card must still be held: it is waiting, not stale")
	}
}

func TestPD3TheNoticeItselfCanNeverGoStale(t *testing.T) {
	// Row 7, and the reason the table exists. The notice is proposed with its
	// words already on it, so it was never built, so BuiltAt is zero.
	notice, err := Propose(Card{
		ID: staleTransitionID, Slot: Transition, Origin: FromDirector,
		Subject: "stale read", Headline: "Report out of date", Script: Say(staleTransitionText),
	})
	if err != nil {
		t.Fatalf("the notice must be a well-formed card: %v", err)
	}
	if !notice.BuiltAt.IsZero() {
		t.Fatal("a card whose words are fixed at proposal was never built")
	}

	// And a director whose clock has run far past anything still does not find
	// it stale — which is what closes the loop.
	d := New(Settings{Max: 10}, planNow.Add(72*time.Hour))
	standby, _ := notice.To(Admitted)
	standby, _ = standby.To(Standby)
	admittedNotice, err := notice.To(Admitted)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := d.lineup.Queue(MainTrack, admittedNotice)
	if err != nil {
		t.Fatal(err)
	}
	d.lineup = queued
	set, err := d.lineup.Set(standby)
	if err != nil {
		t.Fatal(err)
	}
	d.lineup = set
	if _, ok := d.firstStale(); ok {
		t.Error("the notice is stale, so raising it would raise another: the loop is open")
	}
}

// A BUILD MUST RECORD WHEN IT CAME HOME.
//
// The stamp is taken as an argument rather than defaulted, because a build that
// records no time yields a card that can never be judged stale — and that failure
// is invisible: it reads correctly today and goes off quietly, months later, on
// the safety path.
func TestPD3ABuildWithoutAStampIsRefused(t *testing.T) {
	c, err := Propose(Card{ID: "x", Slot: LocationReport, Origin: FromDirector, Subject: "Oceanside, CA", Headline: "Oceanside, CA"})
	if err != nil {
		t.Fatal(err)
	}
	c, _ = c.To(Admitted)
	c, _ = c.To(Standby)
	if _, err := c.WithScript(Say("words"), time.Time{}); err == nil {
		t.Error("a build with no stamp must be refused")
	}
	got, err := c.WithScript(Say("words"), planNow)
	if err != nil {
		t.Fatalf("a stamped build is accepted: %v", err)
	}
	if !got.BuiltAt.Equal(planNow) {
		t.Errorf("the card carries the stamp it was built with, got %v", got.BuiltAt)
	}
}

// The notice says something a listener can act on: that a report was dropped,
// and why. Silence would be indistinguishable from the station breaking.
func TestPD3TheNoticeSaysWhatHappened(t *testing.T) {
	for _, want := range []string{"out of date", "dropped"} {
		if !strings.Contains(strings.ToLower(staleTransitionText), want) {
			t.Errorf("the notice must say %q; it reads %q", want, staleTransitionText)
		}
	}
}

// A STANDING-BY CARD IS RE-HYDRATED, NOT TOSSED (D-84, HUM LEAD 2026-09-11).
//
//	"It's still the same card, it's just like the composer 'filling it' for the
//	 first time, it's just stale … No need to re-check admission — it's already
//	 been admitted. No need for the director to choose / move another card — it's
//	 still the same card, the order has been decided."
//
// THE DEFECT THIS CLOSES IS ONE D-84 ITSELF CREATED. The Composer now works on
// standby so the line is ready the instant the operator goes on air — and a
// station can sit on standby for an hour. Without a refresh the prepared card is
// dropped as it takes the air (PD-3) and the station opens by apologising.
func TestAStandingByCardsWordsAreAskedForAgainRatherThanDropped(t *testing.T) {
	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5, Depth: 3}, base) // STOPPED: the operator is setting up
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})
	d, _ = d.Step(Built{ID: ReadID("oceanside"), Script: Say("Currently sixty-one degrees.")})

	before, ok := d.find(ReadID("oceanside"))
	if !ok || before.State != Standby {
		t.Fatalf("the fixture needs the card standing by with its words; got %+v", before)
	}

	// It sits. Nothing happens until its words have aged.
	d, fx := d.Step(Tick{Now: base.Add(RefreshAfter - time.Minute)})
	if builds(fx) != 0 {
		t.Errorf("words a few minutes old were asked for again: %v", describeAll(fx))
	}

	d, fx = d.Step(Tick{Now: base.Add(RefreshAfter + time.Second)})

	if builds(fx) != 1 {
		t.Fatalf("aged words were not asked for again: %v", describeAll(fx))
	}
	// THE SAME CARD, IN THE SAME PLACE. Not a new proposal, not a re-admission,
	// not a reorder — the order has already been decided.
	after, ok := d.find(ReadID("oceanside"))
	if !ok {
		t.Fatal("the card was tossed; it only needed new data")
	}
	if after.State != Standby {
		t.Errorf("the card left standby to be re-filled; it is %v", after.State)
	}
	if got := ids(d.lineup.Cards(MainTrack)); len(got) != 1 || got[0] != ReadID("oceanside") {
		t.Errorf("the line-up changed around a re-fill: %v", got)
	}
}

// AND IT IS ASKED ONCE, NOT ONCE A SECOND. The schedule ticks every second and a
// build takes a second or more; without the stamp moving with the ASK, the same
// card is re-asked on every tick until its words come home — a build storm
// against the provider the moment a refresh falls due.
func TestARefreshIsAskedForOnceWhileItIsInFlight(t *testing.T) {
	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5, Depth: 3}, base)
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})
	d, _ = d.Step(Built{ID: ReadID("oceanside"), Script: Say("Currently sixty-one degrees.")})

	d, fx := d.Step(Tick{Now: base.Add(RefreshAfter + time.Second)})
	if builds(fx) != 1 {
		t.Fatalf("the refresh did not fire: %v", describeAll(fx))
	}
	for i := range 5 {
		d, fx = d.Step(Tick{Now: base.Add(RefreshAfter + time.Duration(2+i)*time.Second)})
		if builds(fx) != 0 {
			t.Fatalf("tick %d asked again while the first was still out: %v", i, describeAll(fx))
		}
	}
}

// A RUNNING STATION DOES NOT REFRESH: it replaces its cards as it reads them, so
// nothing sits. A card that DOES sit on a running station is being held by a
// rail drain, and PD-3's drop is the right answer there — mid-broadcast there is
// no time to rebuild, which is the whole reason readInstead exists.
// THE FIRST VERSION OF THIS TEST COULD NOT FAIL, and a plant said so. It put ONE
// card on a running station and ticked — but a running station reads that card
// immediately, so by the tick there was no standing-by card left to refresh and
// the assertion held whatever the code did. Deleting the guard it exists to pin
// SURVIVED.
//
// The state it has to construct is a card WAITING: one on the air, and one
// standing by behind it with its words already aged.
func TestARunningStationDoesNotRefreshTheCardWaitingBehindTheOneOnAir(t *testing.T) {
	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	d := New(Settings{Max: 5, Depth: 3}, base)
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(NeedsRead{Ref: "oceanside", Headline: "OCEANSIDE, CA"})
	d, _ = d.Step(Built{ID: ReadID("oceanside"), Script: Say("Currently sixty-one degrees.")})
	d, _ = d.Step(NeedsRead{Ref: "bonsall", Headline: "BONSALL, CA"})
	d, _ = d.Step(Built{ID: ReadID("bonsall"), Script: Say("Currently fifty-nine degrees.")})

	if c, on := d.lineup.OnAir(MainTrack); !on || c.ID != ReadID("oceanside") {
		t.Fatalf("the fixture needs the first card reading; the air holds %+v", c)
	}
	waiting, ok := d.find(ReadID("bonsall"))
	if !ok || waiting.State != Standby {
		t.Fatalf("the fixture needs the second card standing by behind it; got %+v", waiting)
	}

	_, fx := d.Step(Tick{Now: base.Add(RefreshAfter + time.Second)})

	for _, f := range describeAll(fx) {
		if strings.HasPrefix(f, "build(read:bonsall)") {
			t.Errorf("a running station refreshed the card waiting behind the read: %v — "+
				"it replaces its cards as it reads them, and a card that DOES sit is held by a "+
				"rail drain, where PD-3's drop is the right answer", describeAll(fx))
		}
	}
}

// builds counts the cards sent to the Composer in one step.
func builds(fx []Effect) int {
	n := 0
	for _, f := range fx {
		if _, ok := f.(BuildCard); ok {
			n++
		}
	}
	return n
}
