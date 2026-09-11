package lineup

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
)

// run feeds a sequence of events through Step and returns the Director that
// results and every effect it described, in order.
//
// THIS IS THE WHOLE POINT OF APPROACH C. There is no goroutine here, no sleep,
// no clock to advance and nothing to synchronise with: a test states the events
// and reads back the work. Three previous attempts on this code could not get
// that property, and every fixture written against them was vacuous.
func run(d Director, evs ...Event) (Director, []string) {
	var out []string
	for _, ev := range evs {
		var fx []Effect
		d, fx = d.Step(ev)
		for _, f := range fx {
			out = append(out, Describe(f))
		}
	}
	return d, out
}

// burst is a director holding ONE planned takeover, made from n breaking alerts,
// not started. The first effect list is discarded so each test reads only its own.
//
// n ALERTS, ONE CARD (MVS-D-77). The alerts are the burst's CONTENT — the
// Producer's ordering, which the Composer turns into one script — and what the
// schedule holds is the takeover. A test that wants several cards ON THE RAIL
// wants several BURSTS; that is what rail is for.
func burst(t *testing.T, n int) (Director, []string) {
	t.Helper()
	d := New(Settings{Max: 10}, planNow)
	return run(d, Arrived{Arrivals: many("a", category.Warnings, n)})
}

// rail is a director holding one takeover per prefix, in the order given. Each
// burst is two alerts, so none of them is a single alert wearing a burst's name.
func rail(t *testing.T, prefixes ...string) (Director, []string) {
	t.Helper()
	d := New(Settings{Max: 10}, planNow)
	var fx []string
	for _, p := range prefixes {
		var out []string
		d, out = run(d, Arrived{Arrivals: many(p, category.Warnings, 2)})
		fx = append(fx, out...)
	}
	return d, fx
}

func has(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}
	return false
}

func indexOf(got []string, want string) int {
	for i, g := range got {
		if g == want {
			return i
		}
	}
	return -1
}

// TestAnArrivalIsPlannedQueuedPreparedAndPublished — one event, and the whole
// front of the schedule falls out of it.
func TestAnArrivalIsPlannedQueuedPreparedAndPublished(t *testing.T) {
	d, fx := rail(t, "a", "b")
	if got := ids(d.Lineup().Cards(AlertRail)); !equal(got, []string{BurstID("a00"), BurstID("b00")}) {
		t.Fatalf("the rail holds %v, want one card per burst", got)
	}
	if !has(fx, "build("+BurstID("a00")+")") {
		t.Errorf("effects %v do not start the first card's build", fx)
	}
	if has(fx, "build("+BurstID("b00")+")") {
		t.Error("two cards were built at once; only the next one is prepared")
	}
	// PUBLISH IS LAST, ALWAYS. A subscriber reads the lineup to decide what to
	// pre-load, so it must never see a state the same step is still changing.
	if got := fx[len(fx)-1]; !strings.HasPrefix(got, "publish(") {
		t.Errorf("the last effect is %q, want the publish", got)
	}
}

// TestABurstOfManyAlertsIsOneCardOnTheRail is MVS-D-77, and it is where the
// human starts: the operator sees ONE takeover panel with one slot number, and
// DROP / DELAY / PROMOTE act on the burst, not on an alert inside it.
//
// The alerts are still there and still ordered — that is the Producer's output
// and the Composer's input, and every ordering rule is asserted against it — but
// the SCHEDULE holds one card, because there is nothing inside a takeover the
// operator can address separately.
func TestABurstOfManyAlertsIsOneCardOnTheRail(t *testing.T) {
	d, _ := burst(t, 3)
	got := ids(d.Lineup().Cards(AlertRail))
	if !equal(got, []string{BurstID("a00")}) {
		t.Fatalf("three alerts put %v on the rail, want the one takeover they compose", got)
	}
	// The lead names it, and the headline says how many follow: the operator
	// reads the card before there is any script to read.
	card, ok := d.find(BurstID("a00"))
	if !ok {
		t.Fatalf("the rail holds %v; the takeover is not findable by its own id", got)
	}
	if card.Subject != "Bonsall, CA" {
		t.Errorf("the takeover is subjected %q, want the lead alert's own subject", card.Subject)
	}
	if !strings.Contains(card.Headline, "+ 2 more") {
		t.Errorf("the takeover reads %q; it must say how many alerts follow the lead", card.Headline)
	}
	// And a burst of one says nothing about "more", because there are none.
	one, _ := burst(t, 1)
	solo, ok := one.find(BurstID("a00"))
	if !ok {
		t.Fatal("a burst of one alert put no takeover on the rail")
	}
	if strings.Contains(solo.Headline, "more") {
		t.Errorf("a single-alert burst reads %q; there is nothing behind it", solo.Headline)
	}
}

// TestTheCueAlwaysPrecedesTheWords is DR-18 turned from an accident into a
// property. Today the ordering is whatever the call sites happen to do — Phase 0
// pinned it (T0.3, m49) precisely because nothing enforced it. Here it is the
// order of a returned list, which no call site can get wrong.
func TestTheCueAlwaysPrecedesTheWords(t *testing.T) {
	d, _ := burst(t, 2)
	lead := BurstID("a00")
	_, fx := run(d, Built{ID: lead, Script: Say("a severe thunderstorm warning is in effect")})
	cue, speak := indexOf(fx, "cue("+lead+")"), indexOf(fx, "speak("+lead+")")
	if cue < 0 || speak < 0 {
		t.Fatalf("effects %v do not both cue and speak", fx)
	}
	if cue > speak {
		t.Errorf("effects %v speak before they cue; the band would promise a callout after the read", fx)
	}
}

// TestTheNextCardIsBuiltWhileThisOneReads is DR-7 and PD-2, structural.
//
// A cold card build costs 1.03 s of network (perf-protocol §6) against a ~0.7 s
// output buffer, and it is NOT latency-parallelisable. The remedy is to have it
// already done, so the build of the next card must start when this one takes the
// air — not when it finishes, which is exactly one build too late.
func TestTheNextCardIsBuiltWhileThisOneReads(t *testing.T) {
	// TWO BURSTS, because a burst is one card (MVS-D-77): the one-ahead is
	// between cards, and reading it off two alerts of the same burst would be
	// asserting it at a scale the schedule no longer has.
	d, _ := rail(t, "a", "b")
	first, second := BurstID("a00"), BurstID("b00")
	d, fx := run(d, Built{ID: first, Script: Say("words")})
	if !has(fx, "build("+second+")") {
		t.Errorf("effects %v do not start the next build while %s takes the air", fx, first)
	}
	if _, on := onAirAnywhere(d.Lineup()); !on {
		t.Fatal("nothing took the air")
	}
	// And when the first finishes, the second's words are already there to
	// speak: no second build, and no silence waiting for one.
	_, after := run(d, Built{ID: second, Script: Say("more words")}, Finished{ID: first})
	if has(after, "build("+second+")") {
		t.Errorf("%s was built twice", second)
	}
	if !has(after, "speak("+second+")") {
		t.Errorf("effects %v do not put %s on the air; its words were ready", after, second)
	}
}

// TestEveryExitFromTheAirReleasesTheTicker is DR-24, and the reason it is a
// property of Step's output rather than a discipline about call sites: today
// TickerBreakingDoneMsg is sent on ONE path with five early returns above it
// that send nothing, while the audio side is released unconditionally. Under the
// Lineup a discarded or superseded card makes that ordinary.
func TestEveryExitFromTheAirReleasesTheTicker(t *testing.T) {
	lead := BurstID("a00")
	for _, tc := range []struct {
		name string
		exit Event
	}{
		{"read in full", Finished{ID: lead}},
		{"failed mid-read", Failed{ID: lead, Reason: "the voice could not render"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := burst(t, 2)
			d, _ = run(d, Built{ID: lead, Script: Say("words")})
			if _, on := onAirAnywhere(d.Lineup()); !on {
				t.Fatalf("%s never took the air; the exit under test is not being exercised", lead)
			}
			after, fx := run(d, tc.exit)
			if !has(fx, "release("+lead+")") {
				t.Errorf("effects %v leave the band holding a stale callout", fx)
			}
			if id, on := onAirAnywhere(after.Lineup()); on {
				t.Errorf("%q is still on the air after it left it", id.ID)
			}
		})
	}
}

// TestACardThatNeverTookTheAirReleasesNothing. The pairing is with the CUE, not
// with the card: releasing a band that was never cued would clear whatever
// callout it is legitimately showing.
func TestACardThatNeverTookTheAirReleasesNothing(t *testing.T) {
	// TWO BURSTS: the second card is admitted and waiting, and it has never been
	// cued. One burst would give a card that does not exist, which tests a
	// stranger's event instead of this rule.
	d, _ := rail(t, "a", "b")
	_, fx := run(d, Failed{ID: BurstID("b00"), Reason: "fetch failed"})
	for _, f := range fx {
		if strings.HasPrefix(f, "release(") {
			t.Errorf("effects %v release a band that was never cued", fx)
		}
	}
}

// TestOneCardHoldsTheAirAtATime, through a whole run rather than at one moment.
func TestOneCardHoldsTheAirAtATime(t *testing.T) {
	d, _ := rail(t, "a", "b", "c")
	script := []Event{
		Built{ID: BurstID("a00"), Script: Say("one")},
		Built{ID: BurstID("b00"), Script: Say("two")},
		Tick{Now: planNow.Add(time.Minute)},
		Built{ID: BurstID("c00"), Script: Say("three")},
		Finished{ID: BurstID("a00")},
		Finished{ID: BurstID("b00")},
	}
	for _, ev := range script {
		var fx []Effect
		d, fx = d.Step(ev)
		onAir := 0
		for _, track := range []Track{AlertRail, MainTrack} {
			for _, c := range d.Lineup().Cards(track) {
				if c.State == OnAir {
					onAir++
				}
			}
		}
		if onAir > 1 {
			t.Fatalf("after %T, %d cards hold the air (effects %v)", ev, onAir, fx)
		}
	}
}

// TestNothingIsBuiltTwiceHoweverManyTicksArrive. The state transition IS the
// guard: a card moves to standby when its build is described, so the next tick
// no longer sees a card waiting to be built. Nothing counts, and nothing
// remembers.
func TestNothingIsBuiltTwiceHoweverManyTicksArrive(t *testing.T) {
	d, first := burst(t, 3)
	if !has(first, "build("+BurstID("a00")+")") {
		t.Fatal("the first build never started")
	}
	_, fx := run(d,
		Tick{Now: planNow.Add(time.Minute)},
		Tick{Now: planNow.Add(2 * time.Minute)},
		Tick{Now: planNow.Add(3 * time.Minute)})
	for _, f := range fx {
		if strings.HasPrefix(f, "build(") {
			t.Errorf("three idle ticks produced %v; the takeover's build was already described", fx)
		}
	}
}

// TestTheRailDrainsBeforeTheMainTrackThroughStep. The precedence is the
// Lineup's, and this is the Director asking it the same question: a report that
// is READY still waits behind an alert that is not.
//
// That last part is the sharp end of it. takeTheAir looks at the HEAD of the
// queue and stops if it is not ready — it does not look past it for something
// that is. Skipping ahead would read the schedule out of order, which is the one
// thing the rail exists to prevent.
func TestTheRailDrainsBeforeTheMainTrackThroughStep(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	// The main track's filling is the Phase 3 absorb of armDwell/advanceQueue;
	// here it is seeded directly, which is the state that absorb will produce.
	report := at(t, proposed(t, Card{ID: "bonsall", Slot: LocationReport, Origin: FromObserver,
		Subject: "Bonsall, CA", Headline: "Conditions for Bonsall"}), Admitted)
	l, err := d.Lineup().Queue(MainTrack, report)
	if err != nil {
		t.Fatalf("seeding the main track: %v", err)
	}
	d.lineup = l

	// The listener starts the radio, and the report's build begins (PD-1).
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, started := run(d, Powered{To: Running})
	if !has(started, "build(bonsall)") {
		t.Fatalf("starting the radio produced %v, want the report's build", started)
	}

	// An alert arrives and goes to the head of the queue, so its build starts too.
	alert := BurstID("a00")
	d, arrived := run(d, Arrived{Arrivals: many("a", category.Warnings, 1)})
	if !has(arrived, "build("+alert+")") {
		t.Errorf("effects %v do not prepare the alert", arrived)
	}

	// The report's words come back first — and it waits, because the alert is
	// ahead of it on the rail and is not ready yet.
	d, ready := run(d, Built{ID: "bonsall", Script: Say("conditions are fair")})
	if has(ready, "speak(bonsall)") {
		t.Errorf("effects %v read the report over a waiting alert", ready)
	}
	if _, on := onAirAnywhere(d.Lineup()); on {
		t.Error("a ready report took the air while an alert was still being built")
	}

	// The alert's words arrive, and it takes the air over the ready report.
	d, air := run(d, Built{ID: alert, Script: Say("words")})
	if on, _ := onAirAnywhere(d.Lineup()); on.ID != alert {
		t.Errorf("the air is held by %q, want the alert", on.ID)
	}
	if has(air, "speak(bonsall)") {
		t.Errorf("effects %v put the report on the air alongside the alert", air)
	}

	// And once the rail is dry the report resumes — with no second build, because
	// its words have been ready since before the alert started.
	_, after := run(d, Finished{ID: alert})
	if !has(after, "speak(bonsall)") {
		t.Errorf("effects %v do not resume the report once the rail is dry", after)
	}
	if has(after, "build(bonsall)") {
		t.Errorf("effects %v build the report a second time", after)
	}
	if indexOf(after, "release("+alert+")") > indexOf(after, "cue(bonsall)") {
		t.Errorf("effects %v cue the report before releasing the alert's callout", after)
	}
}

// TestABurstArrivingWhileTheRailDrainsAddsToIt is DR-3's guarantee at the
// Director's level: NO ADMITTED CARD IS EVER DROPPED UNREAD. Bounds apply at
// admission and nowhere later, so a second burst joins the queue rather than
// replacing what is already promised.
//
// Observer evaluates each burst on its own — read the Max, divert the rest to
// [w] — which is what handles large bursts and bursts-of-bursts alike (Q-2,
// Q-3); the ladder that would ride or divert them is Broadcaster machinery and
// is deliberately not built.
//
// This is m99: the mutant cleared the rail before queueing, and it survived. It
// also walked straight past the post-condition meant to catch it, because
// clearing the rail BEFORE the count is taken makes "the rail never got shorter"
// true. An invariant that the mutation can reorder itself around is not a pin.
func TestABurstArrivingWhileTheRailDrainsAddsToIt(t *testing.T) {
	reading := BurstID("a00")
	d, _ := rail(t, "a", "b")
	d, _ = run(d, Built{ID: reading, Script: Say("words")})
	if on, _ := onAirAnywhere(d.Lineup()); on.ID != reading {
		t.Fatalf("the air is held by %q; the drain under test is not happening", on.ID)
	}
	before := ids(d.Lineup().Cards(AlertRail))

	// TWO MORE BURSTS, because a burst is one card (MVS-D-77) and "adds to it"
	// is a rule about several arrivals, not about several alerts in one.
	d, _ = run(d,
		Arrived{Arrivals: many("c", category.Warnings, 2)},
		Arrived{Arrivals: many("d", category.Warnings, 1)})
	after := ids(d.Lineup().Cards(AlertRail))

	for _, id := range before {
		if !has(after, id) {
			t.Errorf("%q was promised a read and is gone; the rail now holds %v", id, after)
		}
	}
	for _, id := range []string{BurstID("c00"), BurstID("d00")} {
		if !has(after, id) {
			t.Errorf("%q arrived and was not queued; the rail holds %v", id, after)
		}
	}
	if len(after) != len(before)+2 {
		t.Errorf("the rail holds %v, want the %d already promised plus the two that arrived", after, len(before))
	}
	if on, _ := onAirAnywhere(d.Lineup()); on.ID != reading {
		t.Errorf("the arriving burst took the air from %q mid-read", reading)
	}
}

// TestAnEventForACardTheLineupDoesNotHoldChangesNothing. A completion can arrive
// for a card that was discarded while its build was in flight — ordinary under
// the Lineup, and it must not disturb whatever is on the air now.
func TestAnEventForACardTheLineupDoesNotHoldChangesNothing(t *testing.T) {
	d, _ := burst(t, 2)
	before := ids(d.Lineup().Cards(AlertRail))
	after, fx := run(d,
		Built{ID: "ghost", Script: Say("words")},
		Finished{ID: "ghost"},
		Failed{ID: "ghost", Reason: "gone"})
	if len(fx) != 0 {
		t.Errorf("a stranger's events produced %v", fx)
	}
	if got := ids(after.Lineup().Cards(AlertRail)); !equal(got, before) {
		t.Errorf("the rail changed from %v to %v", before, got)
	}
}

// TestTheSameEventsProduceTheSameEffects is DR-2 at the level the pump runs at.
// Step takes a clock only as an event, so there is nothing else for a second run
// to differ by.
func TestTheSameEventsProduceTheSameEffects(t *testing.T) {
	script := func() []Event {
		return []Event{
			Arrived{Arrivals: many("a", category.Warnings, 4)},
			Arrived{Arrivals: many("b", category.Warnings, 2)},
			Built{ID: BurstID("a00"), Script: Say("one")},
			Tick{Now: planNow.Add(time.Minute)},
			Built{ID: BurstID("b00"), Script: Say("two")},
			Finished{ID: BurstID("a00")},
			Failed{ID: BurstID("b00"), Reason: "the voice could not render"},
		}
	}
	_, first := run(New(Settings{Max: 10}, planNow), script()...)
	_, second := run(New(Settings{Max: 10}, planNow), script()...)
	if !equal(first, second) {
		t.Errorf("two identical runs differ:\n%v\n%v", first, second)
	}
}

// TestStepDescribesWorkAndPerformsNone — the Director's state changes, and the
// only thing that leaves is a list. This is what makes the pump replaceable and
// the deadlock class non-existent: Step cannot block, because it cannot perform.
func TestStepDescribesWorkAndPerformsNone(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	before := d
	_, fx := d.Step(Arrived{Arrivals: many("a", category.Warnings, 2)})
	if got := len(before.Lineup().Cards(AlertRail)); got != 0 {
		t.Errorf("the director Step was called on now holds %d cards; it is a value", got)
	}
	if len(fx) == 0 {
		t.Fatal("a burst arrived and no work was described")
	}
	for _, f := range fx {
		if Describe(f) == "" {
			t.Errorf("an effect of type %T does not say what it is; the debug line DR-23 asks for would be blank", f)
		}
	}
}

// TestTheClockOnlyMovesForward, and only by a Tick. Determinism is structural
// because there is no other way for time to enter (DR-20).
func TestTheClockOnlyMovesForward(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	if !d.Now().Equal(planNow) {
		t.Fatalf("a new director reads %v, want the clock it was given", d.Now())
	}
	ahead, _ := run(d, Tick{Now: planNow.Add(time.Hour)})
	if !ahead.Now().Equal(planNow.Add(time.Hour)) {
		t.Errorf("after a tick the clock reads %v, want an hour on", ahead.Now())
	}
	back, _ := run(ahead, Tick{Now: planNow.Add(-time.Hour)})
	if back.Now().Before(ahead.Now()) {
		t.Errorf("the clock went backwards to %v; a tick out of order would re-age every disaster", back.Now())
	}
}

// TestTheEffectSetIsClosed. The vocabulary IS the architecture's interface
// (PL-6), so every member renders a line a person can act on.
//
// IT NO LONGER CARRIES THE LIST. This test enumerated eight members and omitted
// Escalate — live since fault.go was written — so the one guard whose job is to
// notice the set growing a member had already missed one (red team 2026-09-05,
// I-6). everyEffect() is now the single list and effectset_test.go derives the
// truth from the package source, so it cannot be short again.
func TestTheEffectSetIsClosed(t *testing.T) {
	want := []string{
		"build(x)", "speak(x)", "cue(x)", "release(x)",
		"duck()", "restore()", "tune(KEC62)", "escalate(x)", "publish(rail=[] main=[])",
	}
	got := []string{}
	for _, f := range everyEffect() {
		got = append(got, Describe(f))
	}
	if !equal(got, want) {
		t.Errorf("the effect set renders as %v, want %v", got, want)
	}
	// An effect Describe has no line for renders empty, which is what catches
	// the set growing a member without the timeline learning to say it.
	if got := Describe(nil); got != "" {
		t.Errorf("Describe(nil) = %q, want empty", got)
	}
	for _, e := range []Effect{BuildCard{}, Speak{}, CueTicker{}, ReleaseTicker{}, Tune{}} {
		if got := Describe(e); !strings.HasSuffix(got, "(?)") {
			t.Errorf("an effect with no identity renders as %q; a line nobody can act on should say so", got)
		}
	}
}

// TestEffectsAboutACardCarryItsSlot (BD-8). An executor is chosen by what kind
// of read the card is, and the effect is the description of that work — so the
// slot rides on it. Looking it up from the published lineup would race the
// dispatch: the publish for the same step runs concurrently with the build.
func TestEffectsAboutACardCarryItsSlot(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	d, fx := d.Step(Arrived{Arrivals: many("a", category.Warnings, 1)})
	var build BuildCard
	for _, f := range fx {
		if b, ok := f.(BuildCard); ok {
			build = b
		}
	}
	if build.ID == "" {
		t.Fatalf("the step described %v; no build in it", fx)
	}
	if build.Slot != BreakingAlert {
		t.Errorf("the build carries slot %q, want the card's %q", build.Slot, BreakingAlert)
	}
	_, fx = d.Step(Built{ID: build.ID, Script: Say("words")})
	var speak Speak
	for _, f := range fx {
		if s, ok := f.(Speak); ok {
			speak = s
		}
	}
	if speak.ID != build.ID {
		t.Fatalf("the step described %v; no speak for %s in it", fx, build.ID)
	}
	if speak.Slot != BreakingAlert {
		t.Errorf("the speak carries slot %q, want the card's %q", speak.Slot, BreakingAlert)
	}
}

// admittedCard is a card proposed and admitted the way the planner does it
// (plan.go cardsOf), for the slots the planner does not produce yet.
func admittedCard(t *testing.T, c Card) Card {
	t.Helper()
	card, err := Propose(c)
	if err != nil {
		t.Fatal(err)
	}
	card, err = card.To(Admitted)
	if err != nil {
		t.Fatal(err)
	}
	return card
}

// onTheRail queues admitted cards in order, the way a planned burst does.
func onTheRail(t *testing.T, d Director, cards ...Card) Director {
	t.Helper()
	for _, c := range cards {
		l, err := d.lineup.Queue(AlertRail, c)
		if err != nil {
			t.Fatal(err)
		}
		d.lineup = l
	}
	return d
}

// TestACardThatArrivesWithItsWordsTakesTheAirAndIsNeverBuilt — F-D1.
//
// A burst head, a transition and the divert notice carry their words from the
// moment they are proposed (DR-7): the divert notice's count is decided when
// the burst is planned, so there is nothing to build for them.
//
// prepareNext used to REFUSE such a card — its invariant said a card waiting to
// be built has no words yet — so the card never left ADMITTED. Next offers a
// card at ADMITTED or STANDBY, and only a STANDBY card can take the air: the
// card sat at the head of the queue for ever and EVERY CARD BEHIND IT WENT
// UNREAD. That is DR-3's guarantee failing from the other side, and no fixture
// could see it, because the planner proposes only breaking alerts today.
func TestACardThatArrivesWithItsWordsTakesTheAirAndIsNeverBuilt(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	d = onTheRail(t, d,
		admittedCard(t, Card{ID: "head", Slot: Transition, Headline: "who declared them",
			Script: Say("The National Weather Service has issued the following.")}),
		admittedCard(t, Card{ID: "a1", Slot: BreakingAlert, Headline: "a tornado warning", Subject: "a1"}),
	)
	d, fx := run(d, Tick{Now: planNow.Add(time.Minute)})

	if c, ok := d.find("head"); !ok || c.State != OnAir {
		t.Fatalf("the head is %v after a tick; a card that cannot be built must still reach the air", c.State)
	}
	if !has(fx, "cue(head)") || !has(fx, "speak(head)") {
		t.Fatalf("the step described %v; the head must be cued and read", fx)
	}
	if has(fx, "build(head)") {
		t.Errorf("the step described %v; a card whose words are fixed at proposal is never built (DR-7)", fx)
	}
	if indexOf(fx, "cue(head)") > indexOf(fx, "speak(head)") {
		t.Errorf("the step described %v; the cue must precede the words (DR-18)", fx)
	}

	// AND THE CARD BEHIND IT BUILDS WHILE IT READS (DR-7), in this same step:
	// the one-ahead is not lost just because the card on the air needed no
	// build of its own, or the 1.03 s would land as a silence after the head.
	if !has(fx, "build(a1)") {
		t.Errorf("the step described %v; the alert behind the head must build while the head reads", fx)
	}

	// AND THE RAIL DRAINS BEHIND IT. This is the half the wedge destroyed: the
	// alert queued behind the head must still be prepared and read.
	d2, after := run(d, Finished{ID: "head"})
	if !has(after, "release(head)") {
		t.Errorf("the head left the air and described %v; the band was never released (DR-24)", after)
	}
	// It does NOT take the air here: its build has not come home yet, and a card
	// takes the air with its words already on it. It reads when they arrive —
	// which is the whole point of having started that build a step early.
	if has(after, "speak(a1)") {
		t.Errorf("the head left the air and described %v; a1 was aired before its words arrived", after)
	}
	d2, read := run(d2, Built{ID: "a1", Script: Say("a tornado warning is in effect")})
	if !has(read, "cue(a1)") || !has(read, "speak(a1)") {
		t.Errorf("a1's words arrived and the step described %v; the rail did not drain", read)
	}
	if c, ok := d2.find("a1"); !ok || c.State != OnAir {
		t.Errorf("a1 is %v once its words arrived; the rail did not drain behind the head", c.State)
	}
}

// TestTheOneAheadSurvivesTwoStructuralCards — the shape a real burst opens
// with: a head, a transition, then the first alert.
//
// Preparation walks PAST a card that needs no build, because stopping at the
// first one would leave the alert behind them unprepared and its 1.03 s build
// would land as dead air rather than being ready when its turn came (DR-7). It
// still describes at most one build: the walk stops at a card whose words were
// asked for and have not come back.
func TestTheOneAheadSurvivesTwoStructuralCards(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	d = onTheRail(t, d,
		admittedCard(t, Card{ID: "head", Slot: Transition, Headline: "h", Script: Say("the service has issued the following")}),
		admittedCard(t, Card{ID: "trans", Slot: Transition, Headline: "t", Script: Say("we now return")}),
		admittedCard(t, Card{ID: "a1", Slot: BreakingAlert, Headline: "a", Subject: "a1"}),
	)
	_, fx := run(d, Tick{Now: planNow.Add(time.Minute)})
	if !has(fx, "speak(head)") {
		t.Fatalf("the step described %v; the head must take the air", fx)
	}
	if !has(fx, "build(a1)") {
		t.Errorf("the step described %v; the alert behind two structural cards must build while the head reads", fx)
	}
	// AT MOST ONE BUILD, however many cards were walked past to reach it.
	if n := countHas(fx, "build("); n != 1 {
		t.Errorf("the step described %d builds in %v; one ahead, and only one", n, fx)
	}
}

// TestAStructuralCardIsRefusedWithoutItsWords — the door, not the schedule.
//
// A burst head, a transition and the divert notice carry their words from
// proposal (DR-7). Accepted without them, such a card is promoted to standby,
// handed a build it can never absorb — WithText refuses a structural card's
// words — and left standing by for ever with the rail stopped behind it. The
// same DR-3 failure as the wedge above, reached from the other side, so it is
// closed where every card enters.
func TestAStructuralCardIsRefusedWithoutItsWords(t *testing.T) {
	for _, slot := range []Slot{Transition} { // the one structural slot left (T3.10 red team)
		t.Run(slot.String(), func(t *testing.T) {
			if _, err := Propose(Card{ID: "x", Slot: slot, Headline: "h", Subject: "something"}); err == nil {
				t.Errorf("a %v with no words was accepted; it can never be built and would wedge the rail", slot)
			}
			// The control: with its words, the same card is fine.
			if _, err := Propose(Card{ID: "x", Slot: slot, Headline: "h", Script: Say("the words")}); err != nil {
				t.Errorf("a %v carrying its words was refused: %v", slot, err)
			}
		})
	}
	// And the other half of DR-7 still holds: a report may NOT carry words yet.
	if _, err := Propose(Card{ID: "r", Slot: LocationReport, Headline: "h", Subject: "Bonsall", Script: Say("early")}); err == nil {
		t.Error("a report was proposed carrying words; they materialise at standby")
	}
}

// countHas is how many of the described effects begin with prefix.
func countHas(got []string, prefix string) int {
	n := 0
	for _, g := range got {
		if strings.HasPrefix(g, prefix) {
			n++
		}
	}
	return n
}

// TestPreparationNeverRunsAheadOfTheAir — "one ahead, and only one", as a
// property of the schedule rather than a claim in a comment.
//
// The walk that lets preparation look past a card needing no build must not
// also look past a report that is standing by. It did: each completed build
// triggered the next, so with the first card still ON AIR three more were
// built and a fourth was building. A report composed several reads before it
// plays speaks data that was true when it was built, which is the staleness
// DR-7 exists to prevent — the freshness guarantee traded away for the
// structural-card fix, silently.
func TestPreparationNeverRunsAheadOfTheAir(t *testing.T) {
	first, second := BurstID("a00"), BurstID("b00")
	d, _ := rail(t, "a", "b", "c", "d")
	d, _ = run(d, Built{ID: first, Script: Say("the first alert")})

	// The first takeover is on the air and the second was prepared behind it.
	// Its words coming back must not start a third card's build.
	d, after := run(d, Built{ID: second, Script: Say("the second alert")})
	if countHas(after, "build(") != 0 {
		t.Errorf("a completed build described %v; preparation ran ahead of the air", after)
	}

	waiting := 0
	for _, c := range d.Lineup().Cards(AlertRail) {
		if c.State == Standby {
			waiting++
		}
	}
	if waiting != 1 {
		t.Errorf("%d cards are standing by while one reads; one ahead means one", waiting)
	}
	if on, ok := onAirAnywhere(d.Lineup()); !ok || on.ID != first {
		t.Errorf("the air holds %q, want %s", on.ID, first)
	}
}

// P2: Publish carries the station's power alongside the schedule, so a reader
// can never hold a torn pair — a new lineup beside a stale power.
//
// SWEPT ACROSS EVERY POWER, derived from where Power.String() ends, because a
// version that carried a constant would pass a single-value check.
func TestPublishCarriesThePowerWithTheLineup(t *testing.T) {
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	for p := Power(0); p.String() != ""; p++ {
		d := New(Settings{Max: 5}, base)
		// MAKE THE TRANSITION REAL. A Director starts STOPPED, so stepping to
		// STOPPED changes nothing, settles nothing and publishes nothing — and
		// the sweep would have asserted nothing for that value. The "no
		// Publish" fatal below caught exactly that.
		from := Running
		if p == Running {
			from = OffAir
		}
		d, _ = d.Step(Powered{To: from})
		_, fx := d.Step(Powered{To: p})
		found := false
		for _, e := range fx {
			pub, ok := e.(Publish)
			if !ok {
				continue
			}
			found = true
			if pub.Power != p {
				t.Errorf("Publish must carry the power the Director holds; got %v want %v", pub.Power, p)
			}
		}
		// SILENCE IS A DISTINCT VERDICT (INST-2): no Publish means the check
		// did not run, which is not the same as it passing.
		if !found {
			t.Fatalf("power=%v: no Publish was emitted, so this asserted nothing", p)
		}
	}
}
