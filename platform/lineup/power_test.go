package lineup

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
)

// programme is a director holding one location report on the main track and
// nothing else — the watchlist rotation, which is most of the broadcast.
//
// The main track's filling is the Phase 3 absorb of armDwell/advanceQueue; here
// it is seeded directly, which is the state that absorb will produce.
func programme(t *testing.T, d Director, ids ...string) Director {
	t.Helper()
	for _, id := range ids {
		card := at(t, proposed(t, Card{ID: id, Slot: LocationReport, Origin: FromObserver,
			Subject: id, Headline: "Conditions for " + id}), Admitted)
		l, err := d.lineup.Queue(MainTrack, card)
		if err != nil {
			t.Fatalf("seeding %s: %v", id, err)
		}
		d.lineup = l
	}
	return d
}

// TestANewDirectorIsStopped. The zero value is how a station starts: nothing is
// playing until the listener asks for it, which is today's `d.mode == ""`
// (app/radio.go:radioDeck.setMode). A director that came up running would put a report to air
// on launch, which no listener asked for.
func TestANewDirectorIsStopped(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	if d.Power() != Stopped {
		t.Errorf("a new director is %v, want %v", d.Power(), Stopped)
	}
	var zero Director
	if zero.Power() != Stopped {
		t.Errorf("the zero director is %v, want %v", zero.Power(), Stopped)
	}
}

// TestTheMainTrackDoesNotAdvanceWhileTheRadioIsStopped is PD-1, and the whole
// reason a running state is built for Observer's own merits: the listener
// stopped the radio, and the programme stays stopped.
//
// It is not built, either. A card built while stopped would spend 1.03 s of
// network on a report that has no cutover to be ready for, and might be stale
// by the time one comes.
func TestTheMainTrackDoesNotAdvanceWhileTheRadioIsStopped(t *testing.T) {
	d := programme(t, New(Settings{Max: 10}, planNow), "bonsall", "oceanside")

	d, idle := run(d, Tick{Now: planNow.Add(time.Minute)})
	for _, f := range idle {
		if strings.HasPrefix(f, "build(") || strings.HasPrefix(f, "speak(") {
			t.Errorf("a stopped radio produced %v", idle)
		}
	}
	if _, on := d.Lineup().OnAir(); on {
		t.Error("something took the air while the radio was stopped")
	}

	// And the moment the listener starts it, the programme picks up.
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, started := run(d, Powered{To: Running})
	if !has(started, "build(bonsall)") {
		t.Fatalf("starting the radio produced %v, want the report's build", started)
	}
	_, air := run(d, Built{ID: "bonsall", Script: Say("conditions are fair")})
	if !has(air, "speak(bonsall)") {
		t.Errorf("effects %v do not put the report on the air", air)
	}
}

// TestAStoppedRadioStillReadsTheAlertRail — the asymmetry, and it is deliberate.
//
// STOPPING THE RADIO STOPS THE PROGRAMME, NOT THE HAZARDS. That is today's
// behaviour and it is not being changed: Stop halts the engine, which silences
// the broadcast, while a takeover reads through the narrator's own path and is
// never asked about the deck's mode (app/executors.go asks only whether the
// voice is silent or muted). Mute is the control for "do not speak to me"; stop
// is the control for "do not play me a programme".
func TestAStoppedRadioStillReadsTheAlertRail(t *testing.T) {
	alert := BurstID("a00")
	d := programme(t, New(Settings{Max: 10}, planNow), "bonsall")
	d, fx := run(d, Arrived{Arrivals: many("a", category.Warnings, 2)})
	if !has(fx, "build("+alert+")") {
		t.Fatalf("a stopped radio refused an alert: %v", fx)
	}
	if has(fx, "build(bonsall)") {
		t.Errorf("effects %v built the programme while stopped", fx)
	}
	d, air := run(d, Built{ID: alert, Script: Say("a severe thunderstorm warning is in effect")})
	if !has(air, "speak("+alert+")") {
		t.Errorf("effects %v keep an alert off the air because the radio is stopped", air)
	}
	if on, _ := d.Lineup().OnAir(); on.ID != alert {
		t.Errorf("the air is held by %q, want the alert", on.ID)
	}
}

// TestStoppingTakesTheProgrammeOffTheAirAndReleasesTheBand. Stop is a listener
// action with an immediate effect — the broadcast goes quiet — and the band must
// not be left holding a callout for a read that has ended (DR-24).
func TestStoppingTakesTheProgrammeOffTheAirAndReleasesTheBand(t *testing.T) {
	d := programme(t, New(Settings{Max: 10}, planNow), "bonsall", "oceanside")
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, _ = run(d, Powered{To: Running}, Built{ID: "bonsall", Script: Say("conditions are fair")})
	if on, _ := d.Lineup().OnAir(); on.ID != "bonsall" {
		t.Fatalf("the air is held by %q; the stop under test is not being exercised", on.ID)
	}

	d, fx := run(d, Powered{To: Stopped})
	if !has(fx, "release(bonsall)") {
		t.Errorf("effects %v leave the band holding a callout for a read that stopped", fx)
	}
	if on, live := d.Lineup().OnAir(); live {
		t.Errorf("%q is still on the air after the listener stopped the radio", on.ID)
	}
	// WHAT WAS PROMISED IS STILL PROMISED. Stopping is not a discard: the rest of
	// the rotation waits, and starting again picks it up (DR-3).
	if got := ids(d.Lineup().Cards(MainTrack)); !equal(got, []string{"oceanside"}) {
		t.Errorf("the main track holds %v, want the rest of the rotation waiting", got)
	}
}

// TestStoppingDoesNotCutAnAlertShort. "Whether an alert is on the air is the
// takeover's to say, and it pairs its own" (app/radio.go:radioDeck.setMode). A hazard being
// read is not programme, so the listener's stop does not silence it — and NO
// ADMITTED CARD IS EVER DROPPED UNREAD (DR-3).
func TestStoppingDoesNotCutAnAlertShort(t *testing.T) {
	// TWO BURSTS: "the rail keeps draining" needs a card BEHIND the one reading,
	// and a burst is one card (MVS-D-77).
	reading, behind := BurstID("a00"), BurstID("b00")
	d, _ := rail(t, "a", "b")
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, _ = run(d, Powered{To: Running}, Built{ID: reading, Script: Say("words")})
	if on, _ := d.Lineup().OnAir(); on.ID != reading {
		t.Fatalf("the air is held by %q; the stop under test is not being exercised", on.ID)
	}
	d, fx := run(d, Powered{To: Stopped})
	for _, f := range fx {
		if strings.HasPrefix(f, "release(") {
			t.Errorf("effects %v cut an alert short because the listener stopped the programme", fx)
		}
	}
	if on, _ := d.Lineup().OnAir(); on.ID != reading {
		t.Errorf("the air is held by %q, want the alert still reading", on.ID)
	}
	// And the rail keeps draining while the radio is stopped.
	_, after := run(d, Built{ID: behind, Script: Say("more")}, Finished{ID: reading})
	if !has(after, "speak("+behind+")") {
		t.Errorf("effects %v stop the rail draining; nothing admitted is dropped unread", after)
	}
}

// TestTheRunningStateIsTheOnlyCarrier. Today the rule is `d.mode != ""` PLUS an
// epoch counter — one rule in two places, which is the shape that produced the
// duck-lift bug (RD-2). Here it is one field and one predicate, and this is the
// whole truth table.
func TestTheRunningStateIsTheOnlyCarrier(t *testing.T) {
	for _, tc := range []struct {
		power Power
		track Track
		want  bool
	}{
		{Running, MainTrack, true},
		{Running, AlertRail, true},
		{Stopped, MainTrack, false},
		{Stopped, AlertRail, true},
	} {
		d := New(Settings{Max: 10}, planNow)
		// THE STATION HOLDS THE AIR (D-74). This table is about the POWER, so
		// the other half of the gate is set out of the way of what it measures.
		d.power, d.air = tc.power, AirProgramme
		if got := d.advances(tc.track); got != tc.want {
			t.Errorf("%v %v: advances = %v, want %v", tc.power, tc.track, got, tc.want)
		}
	}
	// An out-of-range track advances nothing: the safe direction is silence.
	d := New(Settings{Max: 10}, planNow)
	d.power, d.air = Running, AirProgramme
	if d.advances(numTracks) {
		t.Error("a track outside the registry advances")
	}

	// AND A POWER OUTSIDE THE REGISTRY ADVANCES NOTHING EITHER. onPowered already
	// refuses to store one, so this cannot be reached through any public path —
	// it is set directly here because that is the only place it CAN be observed,
	// and a guard nothing can exercise is decoration rather than defence. It is
	// kept because advances is the single predicate deciding whether programme
	// goes to air, and one guard should not be the only thing between a
	// corrupted state and audio nobody asked for.
	for _, bad := range []Power{-1, numPowers, 1 << 20} {
		d := New(Settings{Max: 10}, planNow)
		d.power = bad
		if d.advances(MainTrack) {
			t.Errorf("Power(%d) advanced the programme; the safe direction is silence", bad)
		}
	}
}

// TestPoweringToWhatItAlreadyIsChangesNothing. The listener can press stop on a
// stopped radio, and a pump that batches can deliver the same command twice.
func TestPoweringToWhatItAlreadyIsChangesNothing(t *testing.T) {
	d := programme(t, New(Settings{Max: 10}, planNow), "bonsall")
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, _ = run(d, Powered{To: Running}, Built{ID: "bonsall", Script: Say("conditions are fair")})
	before := ids(d.Lineup().Cards(MainTrack))

	// NO EFFECTS AT ALL, not merely no release. Every effect is dispatched, so a
	// no-op command that still published would re-notify every subscriber — the
	// synth layer, the fetch layer, the ticker, the deck — about a lineup that
	// did not change, on every idle keypress.
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, again := run(d, Powered{To: Running})
	if len(again) != 0 {
		t.Errorf("starting an already-running radio produced %v, want nothing", again)
	}
	if on, _ := d.Lineup().OnAir(); on.ID != "bonsall" {
		t.Errorf("the air is held by %q after a repeated start", on.ID)
	}
	if got := ids(d.Lineup().Cards(MainTrack)); !equal(got, before) {
		t.Errorf("the main track went from %v to %v on a repeated start", before, got)
	}

	stopped, _ := run(d, Powered{To: Stopped})
	twice, fx := run(stopped, Powered{To: Stopped})
	if len(fx) != 0 {
		t.Errorf("stopping an already-stopped radio produced %v, want nothing", fx)
	}
	if _, on := twice.Lineup().OnAir(); on {
		t.Error("something is on the air after two stops")
	}
}

// TestAnUnknownPowerIsRefused — a hand-edited config or a later enum value
// arriving from a producer that was not rebuilt must not silently mean
// "running".
func TestAnUnknownPowerIsRefused(t *testing.T) {
	d := programme(t, New(Settings{Max: 10}, planNow), "bonsall")
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, _ = run(d, Powered{To: Running})
	after, _ := run(d, Powered{To: numPowers})
	if after.Power() != Running {
		t.Errorf("an unknown power set the director to %v", after.Power())
	}
}

// TestBothPowersNameThemselves — the transition line DR-23 asks for says why the
// schedule is doing nothing, and "the radio is stopped" is the commonest answer.
func TestBothPowersNameThemselves(t *testing.T) {
	if got, want := Running.String(), "RUNNING"; got != want {
		t.Errorf("Running = %q, want %q", got, want)
	}
	if got, want := Stopped.String(), "STOPPED"; got != want {
		t.Errorf("Stopped = %q, want %q", got, want)
	}
	for _, bad := range []Power{-1, numPowers, 1 << 20} {
		if got := bad.String(); got != "" {
			t.Errorf("Power(%d).String() = %q, want empty", bad, got)
		}
	}
}

// TestAStepPublishesOnceAndLast — stopping the radio takes the programme off
// the air AND settles, and those were two separate settles: one step described
// two publishes, with the first no longer last.
//
// It matters because the pump runs publishes CONCURRENTLY — a publish holds no
// resource — so a reader could apply the older snapshot after the newer one and
// show a card that has already left the schedule.
func TestAStepPublishesOnceAndLast(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	report := at(t, proposed(t, Card{ID: "bonsall", Slot: LocationReport, Origin: FromObserver,
		Subject: "Bonsall, CA", Headline: "Conditions for Bonsall"}), Admitted)
	l, err := d.Lineup().Queue(MainTrack, report)
	if err != nil {
		t.Fatalf("seeding the main track: %v", err)
	}
	d.lineup = l
	d, _ = d.Step(Aired{To: AirProgramme}) // the console holds the air (D-74)
	d, _ = run(d, Powered{To: Running})
	d, _ = run(d, Built{ID: "bonsall", Script: Say("conditions are fair")})

	// The programme is on the air; the listener stops the radio.
	_, fx := run(d, Powered{To: Stopped})
	if n := countHas(fx, "publish("); n != 1 {
		t.Errorf("stopping described %d publishes in %v; a step tells the readers once", n, fx)
	}
	if len(fx) == 0 || !strings.HasPrefix(fx[len(fx)-1], "publish(") {
		t.Errorf("stopping described %v; the readers are told LAST", fx)
	}
}

// MVS-D-78 — THE THREE POWERS, AND WHAT EACH LETS THROUGH.
//
// The table IS the rule. Stopped and OffAir differ on exactly one cell, and it
// is the cell that matters: stopping the radio stops the PROGRAMME and lets
// hazards through, because mute is the control for "do not speak to me" and
// stop for "do not play me a programme". Standby is a listener saying the first
// about everything.
//
// Walked over the registry rather than a hand-written list, so a power added
// later fails here rather than defaulting into whatever the last case happened
// to be — the sentinel exists for exactly that (numNarrationClasses' lesson).
func TestMVSD78EachPowerLetsThroughWhatItShould(t *testing.T) {
	want := map[Power]map[Track]bool{
		Running: {MainTrack: true, AlertRail: true},
		Stopped: {MainTrack: false, AlertRail: true},  // the programme stops; hazards do not
		OffAir:  {MainTrack: false, AlertRail: false}, // dead air means dead air
	}
	if len(want) != int(numPowers) {
		t.Fatalf("the table covers %d powers, the registry declares %d — a power was added without a row", len(want), numPowers)
	}
	for p := Power(0); p < numPowers; p++ {
		// THE STATION HOLDS THE AIR. This walks the POWER registry, so the
		// air is held constant at the value that lets the power be measured
		// at all (D-74).
		d := Director{power: p, air: AirProgramme}
		for tr := Track(0); tr < numTracks; tr++ {
			if got := d.advances(tr); got != want[p][tr] {
				t.Errorf("%v on %v advances=%v, want %v", p, tr, got, want[p][tr])
			}
		}
		if p.String() == "" {
			t.Errorf("%d has no name for the timeline", p)
		}
	}
}

// A power outside the registry advances nothing. FAIL CLOSED: a value from a
// producer built against a later enum must not read as "running" and start
// putting programme to air nobody asked for.
func TestAnUndeclaredPowerAdvancesNothing(t *testing.T) {
	for _, p := range []Power{-1, numPowers, numPowers + 7} {
		d := Director{power: p}
		for tr := Track(0); tr < numTracks; tr++ {
			if d.advances(tr) {
				t.Errorf("power %d advanced %v", p, tr)
			}
		}
	}
}

// STANDBY IS NOT A LOUDER STOPPED, and the rail is where they part. This says
// it directly, because the two differ in one cell of the table above and a
// reader skimming it could take them for the same state.
func TestStandbyHoldsTheRailWhereStoppedDoesNot(t *testing.T) {
	if !(Director{power: Stopped}).advances(AlertRail) {
		t.Error("a stopped radio still reads hazards")
	}
	if (Director{power: OffAir}).advances(AlertRail) {
		t.Error("dead air reads nothing, hazards included")
	}
}
