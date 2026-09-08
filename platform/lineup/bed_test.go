package lineup

import (
	"testing"
	"time"
)

func atNoon() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC) }

// tunesIn is the Tune effects among a step's output.
//
// SETTLE PUBLISHES ON EVERY TICK, so a step is never empty and "len(fx) == 0"
// would assert the wrong thing — it did, in the first version of these tests,
// and failed on the Publish rather than on anything about the bed.
func tunesIn(fx []Effect) []Tune {
	var out []Tune
	for _, f := range fx {
		if t, ok := f.(Tune); ok {
			out = append(out, t)
		}
	}
	return out
}

// bedDirector is a Director carrying a watchlist and a dwell, with the bed on
// the first entry as a live relay.
func bedDirector(t *testing.T, dwell time.Duration) Director {
	t.Helper()
	// POWERED ON FIRST, because Stopped is the zero value on purpose: a station
	// starts silent and nothing plays until the listener asks. Four of these
	// tests were written without it and passed only because the bed did not yet
	// consult the power at all.
	d := New(Settings{Max: 5, Watchlist: []string{"a", "b", "c"}, Dwell: dwell}, atNoon())
	d, _ = d.Step(Powered{To: Running})
	d, fx := d.Step(Tuned{Ref: "a", Live: true})
	if n := len(tunesIn(fx)); n != 0 {
		t.Fatalf("tuning the bed asked for another tune: %v", fx)
	}
	return d
}

// A LIVE RELAY HOLDS THE BED FOR ITS DWELL AND THEN THE NEXT ONE TAKES IT
// (T3.2b, UAT 93's rule, moved off a timer).
//
// Stated as a schedule rather than observed with a clock: the dwell used to be a
// time.AfterFunc inside the radio deck, so the only way to see it was to wait
// five real minutes. Here it is a pure function of the bed, the settings and
// now.
func TestALiveRelayAdvancesWhenItsDwellElapses(t *testing.T) {
	d := bedDirector(t, 5*time.Minute)

	// A tick inside the dwell moves nothing.
	d, fx := d.Step(Tick{Now: atNoon().Add(4*time.Minute + 59*time.Second)})
	if n := len(tunesIn(fx)); n != 0 {
		t.Fatalf("the bed moved before its dwell elapsed: %v", fx)
	}

	// The tick that reaches it advances to the next in the listener's order.
	d, fx = d.Step(Tick{Now: atNoon().Add(5 * time.Minute)})
	tunes := tunesIn(fx)
	if len(tunes) != 1 || tunes[0].Ref != "b" {
		t.Fatalf("want one Tune to b, got %v", fx)
	}

	// AND THE DWELL RESTARTS AT THE DECISION, not when the new relay arrives.
	// Waiting for the Tuned that follows would leave the deadline in the past and
	// every tick between would emit another advance.
	_, fx = d.Step(Tick{Now: atNoon().Add(5*time.Minute + time.Second)})
	if n := len(tunesIn(fx)); n != 0 {
		t.Fatalf("a second advance fired while the first tune was still in flight: %v", fx)
	}
}

// THE THREE WAYS THE BED DOES NOT ADVANCE, each the rule it stands for.
func TestTheBedHoldsWhenItShould(t *testing.T) {
	for _, tc := range []struct {
		what string
		d    func(*testing.T) Director
	}{
		{"repeat is not Watchlist, which reaches the Director as a zero dwell",
			func(t *testing.T) Director { return bedDirector(t, 0) }},
		{"the bed is the synthesised broadcast, which ends on its own and needs no turn",
			func(t *testing.T) Director {
				d := New(Settings{Max: 5, Watchlist: []string{"a", "b"}, Dwell: time.Minute}, atNoon())
				d, _ = d.Step(Powered{To: Running})
				d, _ = d.Step(Tuned{Ref: "a", Live: false})
				return d
			}},
		{"the watchlist is empty, so there is nowhere to move to",
			func(t *testing.T) Director {
				d := New(Settings{Max: 5, Dwell: time.Minute}, atNoon())
				d, _ = d.Step(Powered{To: Running})
				d, _ = d.Step(Tuned{Ref: "a", Live: true})
				return d
			}},
	} {
		t.Run(tc.what, func(t *testing.T) {
			d := tc.d(t)
			if _, fx := d.Step(Tick{Now: atNoon().Add(time.Hour)}); len(tunesIn(fx)) != 0 {
				t.Errorf("the bed advanced when %s: %v", tc.what, fx)
			}
		})
	}
}

// A BED ON SOMETHING OUTSIDE THE QUEUE REJOINS AT THE TOP — the deck's
// nextInQueue rule, moved rather than rewritten. A listener who removed the
// location they are listening to keeps rotating instead of stopping.
func TestABedOutsideTheWatchlistRejoinsAtTheTop(t *testing.T) {
	d := New(Settings{Max: 5, Watchlist: []string{"a", "b"}, Dwell: time.Minute}, atNoon())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Tuned{Ref: "gone", Live: true})
	_, fx := d.Step(Tick{Now: atNoon().Add(time.Minute)})
	tunes := tunesIn(fx)
	if len(tunes) != 1 {
		t.Fatalf("want one Tune, got %v", fx)
	}
	if tunes[0].Ref != "a" {
		t.Errorf("a bed outside the queue rejoins at the top, got %q", tunes[0].Ref)
	}
}

// THE DWELL COUNTS FROM WHEN AUDIO PLAYS, not from when the tune was asked for.
// A resolve and a connect can take seconds, and charging those to the listener's
// five minutes would cut every turn short by however slow the network was.
func TestTheDwellRunsFromTheMomentTheBedTookIt(t *testing.T) {
	d := New(Settings{Max: 5, Watchlist: []string{"a", "b"}, Dwell: time.Minute}, atNoon())
	d, _ = d.Step(Powered{To: Running})
	// Three minutes pass with nothing tuned, then the bed takes it.
	d, _ = d.Step(Tick{Now: atNoon().Add(3 * time.Minute)})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})
	// One second short of its own minute: it keeps the bed.
	d, fx := d.Step(Tick{Now: atNoon().Add(3*time.Minute + 59*time.Second)})
	if n := len(tunesIn(fx)); n != 0 {
		t.Fatalf("the dwell was charged time before the bed took the air: %v", fx)
	}
	if _, fx = d.Step(Tick{Now: atNoon().Add(4 * time.Minute)}); len(tunesIn(fx)) != 1 {
		t.Fatalf("the dwell did not elapse a minute after the bed took the air: %v", fx)
	}
}

// A ONE-ENTRY WATCHLIST DOES NOT RE-TUNE ITSELF EVERY DWELL.
//
// The wrap makes the next entry the current one, so an unguarded advance would
// cut the audio and re-tune the same relay every five minutes — the listener
// with a single station hears it restart on a timer, for no reason they could
// name. Found by the P10 density gate sending me back through nextInWatchlist,
// which is the second time a gate has produced a defect rather than a chore.
func TestAWatchlistOfOneNeverRetunesItself(t *testing.T) {
	d := New(Settings{Max: 5, Watchlist: []string{"a"}, Dwell: time.Minute}, atNoon())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})

	for i := 1; i <= 4; i++ { // four dwells' worth: it must never move
		var fx []Effect
		d, fx = d.Step(Tick{Now: atNoon().Add(time.Duration(i) * time.Minute)})
		if tunes := tunesIn(fx); len(tunes) != 0 {
			t.Fatalf("dwell %d: the only station in the rotation was re-tuned to itself: %v", i, tunes)
		}
	}

	// And it still advances the moment there is somewhere else to go.
	d, _ = d.Step(Powered{To: Running}) // a stopped station rotates nowhere; without this these pass vacuously
	d, _ = d.Step(Programme{Watchlist: []string{"a", "b"}, Dwell: time.Minute})
	_, fx := d.Step(Tick{Now: atNoon().Add(5 * time.Minute)})
	if tunes := tunesIn(fx); len(tunes) != 1 || tunes[0].Ref != "b" {
		t.Errorf("a second station was added and the bed did not move to it: %v", fx)
	}
}

// A ROTATION WITH A BLANK ENTRY IS REFUSED, not carried.
//
// A blank would be a tune to nowhere: the bed asked for a location with no name,
// and the deck resolving nothing. The rotation would stop there without
// reporting anything.
func TestARotationWithABlankEntryIsRefused(t *testing.T) {
	d := New(Settings{Max: 5, Watchlist: []string{"a", "b"}, Dwell: time.Minute}, atNoon())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})
	before := d.settings.Watchlist

	d, _ = d.Step(Programme{Watchlist: []string{"a", "", "b"}, Dwell: time.Minute})
	if got := d.settings.Watchlist; len(got) != len(before) {
		t.Errorf("a rotation carrying a blank entry was taken: %q", got)
	}
}

// A STOPPED PROGRAMME DOES NOT ADVANCE THE BED.
//
// The listener pressed stop. The bed moving on afterwards would be the station
// starting itself again five minutes later — and the deck's own rule says
// nothing follows a stop (`d.mode = ""`). Found by writing out where the bed can
// be and what moves it, BEFORE the wiring, rather than by a fifth review round.
func TestAStoppedProgrammeDoesNotAdvanceTheBed(t *testing.T) {
	d := New(Settings{Max: 5, Watchlist: []string{"a", "b"}, Dwell: time.Minute}, atNoon())
	d, _ = d.Step(Tuned{Ref: "a", Live: true})
	d, _ = d.Step(Powered{To: Stopped})

	if _, fx := d.Step(Tick{Now: atNoon().Add(time.Hour)}); len(tunesIn(fx)) != 0 {
		t.Errorf("a stopped programme advanced the bed: %v", fx)
	}
	if _, fx := d.Step(Ended{}); len(tunesIn(fx)) != 0 {
		t.Errorf("a stopped programme advanced on a cycle end: %v", fx)
	}

	// And it advances again once the listener starts it.
	d, _ = d.Step(Powered{To: Running})
	if _, fx := d.Step(Tick{Now: atNoon().Add(2 * time.Hour)}); len(tunesIn(fx)) != 1 {
		t.Errorf("the programme was started again and the bed still would not move: %v", fx)
	}
}

// THE SYNTHESISED BROADCAST MOVES ON WHEN ITS CYCLE ENDS, NOT ON A DWELL.
//
// `advanceQueue` had two callers, and the absorb needs both: the five-minute
// dwell that moves a live relay, and the cycle end that moves the synthesised
// broadcast. A relay never ends, which is why it has a dwell at all; the synth
// broadcast ends on its own and moves on then — a cycle that ran two minutes
// moves at two minutes.
func TestTheSynthBroadcastAdvancesWhenItsCycleEnds(t *testing.T) {
	d := New(Settings{Max: 5, Watchlist: []string{"a", "b"}, Dwell: time.Minute}, atNoon())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Tuned{Ref: "a", Live: false}) // the synth broadcast

	// Well inside the dwell: a tick moves nothing, because the synth does not dwell.
	d, fx := d.Step(Tick{Now: atNoon().Add(10 * time.Second)})
	if n := len(tunesIn(fx)); n != 0 {
		t.Fatalf("the synthesised broadcast was moved by a dwell: %v", fx)
	}
	// Its cycle ends: it moves on at once.
	_, fx = d.Step(Ended{})
	tunes := tunesIn(fx)
	if len(tunes) != 1 || tunes[0].Ref != "b" {
		t.Fatalf("a finished cycle did not move to the next location: %v", fx)
	}
}

// AND A CYCLE ENDING OUTSIDE WATCHLIST MOVES NOTHING — the rotation only
// advances by itself when the listener asked for a rotation.
func TestACycleEndOutsideWatchlistHoldsTheBed(t *testing.T) {
	d := New(Settings{Max: 5, Watchlist: []string{"a", "b"}}, atNoon()) // Dwell 0 = not Watchlist
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Tuned{Ref: "a", Live: false})
	if _, fx := d.Step(Ended{}); len(tunesIn(fx)) != 0 {
		t.Errorf("a cycle ended outside Watchlist and the bed moved anyway: %v", fx)
	}
}

// A REPEATED REPORT DOES NOT RESTART THE TURN.
//
// A live relay sends a status every time its title changes, and every one says
// Playing — so a dwell restarted on each report would never elapse, and the
// rotation would stop dead on whichever station talks most. The deck's armDwell
// was idempotent for exactly this reason; the rule had to survive the move, and
// it did not until the corpus guard noticed m53 had nothing left to anchor to.
func TestARepeatedTunedReportDoesNotRestartTheTurn(t *testing.T) {
	d := New(Settings{Max: 5, Watchlist: []string{"a", "b"}, Dwell: time.Minute}, atNoon())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})

	// The relay's title changes twice while its turn runs.
	d, _ = d.Step(Tick{Now: atNoon().Add(30 * time.Second)})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})
	d, _ = d.Step(Tick{Now: atNoon().Add(50 * time.Second)})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})

	// Its minute is still its minute.
	d, fx := d.Step(Tick{Now: atNoon().Add(time.Minute)})
	tunes := tunesIn(fx)
	if len(tunes) != 1 || tunes[0].Ref != "b" {
		t.Fatalf("the turn was restarted by a status report; the rotation would never move: %v", fx)
	}

	// And a REAL move does restart it.
	d, _ = d.Step(Tuned{Ref: "b", Live: true})
	if _, fx := d.Step(Tick{Now: atNoon().Add(time.Minute + 30*time.Second)}); len(tunesIn(fx)) != 0 {
		t.Errorf("the new station did not get a full turn of its own: %v", fx)
	}
}

// A TUNE THAT NEVER LANDS IS REPORTED (FR-9.3).
//
// The rotation is told to move and the station does not: the deck reports Tuned
// when audio actually PLAYS, so a tune that never plays is silence that nothing
// else observes. Nothing is on the air and nothing is coming — which is exactly
// the fault fault.go says a window is for.
func TestATuneThatNeverLandsIsReported(t *testing.T) {
	d := New(Settings{Max: 5}, planNow)
	d, _ = d.Step(Powered{To: Running}) // a stopped station rotates nowhere; without this these pass vacuously
	d, _ = d.Step(Programme{Watchlist: []string{"a", "b"}, Dwell: time.Minute})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})

	// The turn ends and the rotation is told to move to "b".
	d, fx := d.Step(Ended{})
	if !hasTune(fx, "b") {
		t.Fatalf("the rotation did not move on: %v", fx)
	}

	// The station never plays it. Ticks up to the bound report nothing.
	d, fx = d.Step(Tick{Now: planNow.Add(tuneLands - time.Second)})
	if hasEscalate(fx) {
		t.Errorf("the rotation was reported stalled before its bound: %v", fx)
	}
	// Past the bound it is reported, ONCE.
	d, fx = d.Step(Tick{Now: planNow.Add(tuneLands + time.Second)})
	if !hasEscalate(fx) {
		t.Fatalf("a tune that never landed was never reported: %v", fx)
	}
	// A SECOND TICK, ONE SECOND LATER — not at 2 x the bound, which is where
	// the one-minute dwell elapses and issues a NEW tune, resetting the clock
	// and making this assertion pass for the wrong reason. It did, until the
	// planted defect it exists to catch went unnoticed.
	_, fx = d.Step(Tick{Now: planNow.Add(tuneLands + 2*time.Second)})
	if hasEscalate(fx) {
		t.Errorf("the stall was reported twice; a tick every second would raise a window every "+
			"second, which is the noise regression fault.go exists to avoid: %v", fx)
	}
}

// AND A TUNE THAT LANDS IS NOT REPORTED, however long the station then plays.
func TestATuneThatLandsIsNotReported(t *testing.T) {
	d := New(Settings{Max: 5}, planNow)
	d, _ = d.Step(Powered{To: Running}) // a stopped station rotates nowhere; without this these pass vacuously
	d, _ = d.Step(Programme{Watchlist: []string{"a", "b"}, Dwell: time.Minute})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})
	d, _ = d.Step(Ended{})
	d, _ = d.Step(Tuned{Ref: "b", Live: true}) // the station moved

	_, fx := d.Step(Tick{Now: planNow.Add(10 * tuneLands)})
	if hasEscalate(fx) {
		t.Errorf("a station that moved when asked was reported as stalled: %v", fx)
	}
}

// AND SILENCE THE OPERATOR CHOSE IS NOT A FAULT (FR-9.3's rewording; I-2).
//
// A stop while a tune is in flight is a listener saying stop, not a station
// failing to move. Reporting it would raise a window for something the listener
// did on purpose — the exact reversal the HUM LEAD's rewording exists to
// prevent.
func TestAStopWhileATuneIsInFlightIsNotAFault(t *testing.T) {
	d := New(Settings{Max: 5}, planNow)
	d, _ = d.Step(Powered{To: Running}) // a stopped station rotates nowhere; without this these pass vacuously
	d, _ = d.Step(Programme{Watchlist: []string{"a", "b"}, Dwell: time.Minute})
	d, _ = d.Step(Tuned{Ref: "a", Live: true})
	d, _ = d.Step(Ended{})
	d, _ = d.Step(Powered{To: Stopped}) // the listener stopped it

	_, fx := d.Step(Tick{Now: planNow.Add(tuneLands + time.Second)})
	if hasEscalate(fx) {
		t.Errorf("stopping the station raised a fault for the tune it was mid-way through: %v", fx)
	}
}

func hasTune(fx []Effect, ref string) bool {
	for _, f := range fx {
		if t, ok := f.(Tune); ok && t.Ref == ref {
			return true
		}
	}
	return false
}

func hasEscalate(fx []Effect) bool {
	for _, f := range fx {
		if _, ok := f.(Escalate); ok {
			return true
		}
	}
	return false
}
