package lineup

// THE DUCK, AS A PURE DECISION (D-32, HUM LEAD 2026-09-09).
//
// > "only 1 [of main and bed] can be active at a time, and the priority ducks
// > the bed only."
//
// THREE THINGS CAN TRIGGER A DUCK TODAY AND ONLY ONE OF THEM SHOULD. This is
// that rule expressed where it can be tested without an audio device: a
// priority card on the air, over a bed the operator has cut to, and nothing
// else. It is answerable from state the Director already holds, which is the
// whole argument for putting it here — P3's four blockers were every one of
// them in the join between this machine and the device.
//
// WHAT IT IS NOT. The duck is not about which medium is AUDIBLE at this
// instant: `Suppress` is inert when nothing is playing, and the engine re-reads
// the source kind every tick so "how to yield" follows the audio. The Director
// says an alert is on the air; the engine decides dip-or-hold. Asking the
// deck's mode here "fixed an answer the audio could outlive" (radio.go), and
// that design is not repeated.

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
)

// aTornado is the one hazard these tests need: a warning, close, and severe
// enough that no ordering rule can keep it out of the burst.
func aTornado() Arrival { return arrival("a1", category.Warnings, 95, time.Minute) }

// onTheBed is a running station the operator has cut over to a relay.
func onTheBed(t *testing.T) Director {
	t.Helper()
	d := New(Settings{Max: 5}, time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(CutOver{ToBed: true})
	if !d.bed.carries {
		t.Fatal("the fixture needs the programme on the bed")
	}
	return d
}

// ducks reports whether this step asked for the bed to give way, and whether it
// asked for it back. Named rather than inlined because every test below asks
// the same question of a different step.
func ducks(fx []Effect) (duck, restore int) {
	for _, f := range fx {
		switch f.(type) {
		case Duck:
			duck++
		case Restore:
			restore++
		}
	}
	return duck, restore
}

func TestAHazardOverACarryingBedDucksIt(t *testing.T) {
	d := onTheBed(t)

	_, fx := d.Step(Arrived{Arrivals: []Arrival{aTornado()}})

	if got, _ := ducks(fx); got != 1 {
		t.Fatalf("a takeover admitted over a live bed gives way once; got %d Duck effect(s)", got)
	}
	// THE ORDERING RULE IS REAL AND CURRENTLY UNREACHABLE, and that is worth a
	// paragraph rather than a check that cannot fail (D-2).
	//
	// The effect set states it: "a duck that landed after the read had started
	// would be the duck-lift bug in a new costume." So the duck must precede the
	// cue. A plant that moved it after the cue SURVIVES, and MEASURING the
	// sequence says why rather than guessing:
	//
	//	Arrived  -> duck    · build   · publish
	//	Built    -> cue     · speak   · publish
	//	Finished -> release · restore · publish
	//
	// The duck's edges NEVER FALL IN THE SAME STEP AS A CUE. The rising edge is
	// at admission, when the card has no words and nothing can air; the falling
	// edge is at the last Finished, and nothing can be cued alongside it.
	//
	// D-82 RELAXED EXCLUSIVITY AND THIS STAYED UNREACHABLE — re-checked rather
	// than assumed, because the sentence above used to rest on it. A hazard can
	// now take the air over a report, so there are two ways a cue could fall
	// beside a falling edge, and both are closed: while the bed carries, the
	// main track does not advance at all (FR-4.2); and with a report reading, it
	// is that report holding the lane's air, so the next one cannot be cued
	// until it leaves.
	//
	// SO THE RULE IS ENFORCED BY THE SCHEDULE, not by the order of this list —
	// and a check asserting the position would be the vacuous class this project
	// has shipped five of. It becomes reachable only if a lane can cue a card
	// while another is still reading on it, which is the invariant OnAir keeps.
}

// THE OTHER HALF OF D-32, and it is the half that broke a release. Nothing ducks
// when there is no programme UNDERNEATH the hazard.
//
// ITS REASON WAS REWRITTEN AT D-82, and the distinction matters. It used to read
// "a chosen read REPLACES the bed rather than playing over it, so there is
// nothing underneath to dip" — true of the bed, and true here because this
// station has nothing on the air at all. What is NOT true any more is the
// generalisation: a report READING is something underneath, and the rail holds
// it for the whole drain (TestTheProgrammeIsHeldForAWholeRailDrainNotPerHazard).
// This case is the one that stays: no bed carrying, no card reading, nothing to
// give way — and a Restore paired to a Duck nobody emitted would lift a bed some
// other owner had put down.
func TestNothingDucksWhenTheBedIsNotCarrying(t *testing.T) {
	d := New(Settings{Max: 5}, time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})

	_, fx := d.Step(Arrived{Arrivals: []Arrival{aTornado()}})

	if got, _ := ducks(fx); got != 0 {
		t.Errorf("with no bed carrying the programme there is nothing to duck; got %d Duck effect(s) — "+
			"the rotation IS the programme, and ducking for it ducks the thing being played", got)
	}
}

// DR-24's pairing, as a property of the Director rather than a promise: the bed
// comes back up, once, when the rail is done with it.
func TestTheBedComesBackWhenTheRailIsDone(t *testing.T) {
	d := onTheBed(t)
	d, _ = d.Step(Arrived{Arrivals: []Arrival{aTornado()}})

	total := 0
	for _, step := range []Event{
		Built{ID: BurstID("a1"), Script: Say("A tornado warning has been declared.")},
		Finished{ID: BurstID("a1")},
	} {
		var fx []Effect
		d, fx = d.Step(step)
		_, r := ducks(fx)
		total += r
	}

	if total != 1 {
		t.Errorf("the bed is given back exactly once when the rail empties; got %d Restore effect(s)", total)
	}
}

// A SECOND HAZARD DOES NOT DIP A SECOND TIME (MVS-D-67): "one duck per RAIL
// DRAIN, never per card." A rail of two cards that dipped, lifted and dipped
// again between them was measured before that ruling existed.
func TestASecondHazardDoesNotDuckAgain(t *testing.T) {
	d := onTheBed(t)
	d, _ = d.Step(Arrived{Arrivals: []Arrival{aTornado()}})

	_, fx := d.Step(Arrived{Arrivals: []Arrival{arrival("a2", category.Warnings, 90, time.Minute)}})

	if got, _ := ducks(fx); got != 0 {
		t.Errorf("the bed is already down; asking again is a second dip the listener hears; got %d", got)
	}
}

// AND NOTHING IS GIVEN BACK THAT WAS NEVER TAKEN. A Restore with no Duck would
// lift a bed some other owner had put down.
func TestNothingIsRestoredThatWasNeverDucked(t *testing.T) {
	d := New(Settings{Max: 5}, time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d, _ = d.Step(Arrived{Arrivals: []Arrival{aTornado()}})
	d, _ = d.Step(Built{ID: BurstID("a1"), Script: Say("A tornado warning has been declared.")})

	_, fx := d.Step(Finished{ID: BurstID("a1")})

	if _, got := ducks(fx); got != 0 {
		t.Errorf("no bed was ducked, so none is given back; got %d Restore effect(s)", got)
	}
}

// THE PROGRAMME GIVES WAY TOO, NOT ONLY THE BED (D-82).
//
// MVS-D-67 is "one duck per RAIL DRAIN, never per card" — a rail of two cards
// that dipped, lifted and dipped again between them was MEASURED. The mechanism
// that holds it is the Director's Duck/Restore pair, and its one condition was
// `bed.carries`: the bed was the only thing a hazard could be speaking over.
//
// Since D-82 a REPORT can be underneath. The arbiter gives way per SEQUENCE and
// takes it back the moment nothing is waiting, so between two hazards the engine
// would un-hold a rendered report for the few milliseconds the schedule takes to
// dispatch the next one — and un-holding a paused report is not a volume bounce,
// it is a fragment of a word.
//
// THE ENGINE DECIDES DIP-OR-HOLD, which is why one condition can serve both and
// why this does not contradict D-32's "the priority ducks the bed only": a relay
// DIPS and a rendered report HOLDS, read from the source kind every 50 ms. The
// Director says only that the rail is speaking over the programme.
func TestTheProgrammeIsHeldForAWholeRailDrainNotPerHazard(t *testing.T) {
	d := New(Settings{Max: 5}, time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d = reading(t, d, "report", LocationReport)
	if _, on := d.lineup.OnAir(MainTrack); !on {
		t.Fatal("the fixture needs the report reading under the hazards")
	}

	// The first hazard is admitted over it: the programme gives way, once. The
	// rising edge is at ADMISSION rather than at the air, which is the rule
	// `givingWay` already states — the card has no words yet, and a report that
	// spoke over the attention tone is worse than a second of held air.
	d, fx := d.Step(Arrived{Arrivals: []Arrival{aTornado()}})
	if duck, restore := ducks(fx); duck != 1 || restore != 0 {
		t.Fatalf("the first hazard asked for %d ducks and %d restores; want one dip and no lift", duck, restore)
	}

	// A second hazard joins the drain behind it.
	d, fx = d.Step(Arrived{Arrivals: []Arrival{arrival("a2", category.Warnings, 90, time.Minute)}})
	if duck, restore := ducks(fx); duck != 0 || restore != 0 {
		t.Errorf("a second hazard over an already-held programme asked for %d ducks and %d restores; want neither", duck, restore)
	}
	held := len(d.lineup.tracks[AlertRail])
	if held < 2 {
		t.Fatalf("the fixture needs two cards on the rail to have a DRAIN at all; got %d", held)
	}

	// The first one is read and ends. The rail is not dry, so nothing lifts.
	d, _ = d.Step(Built{ID: BurstID("a1"), Script: Say("A tornado warning has been declared.")})
	d, fx = d.Step(Finished{ID: BurstID("a1")})
	if duck, restore := ducks(fx); restore != 0 || duck != 0 {
		t.Errorf("between two hazards the programme was lifted (%d) and dipped again (%d) — "+
			"MVS-D-67 is one duck per DRAIN, and a report un-held for a few milliseconds is a fragment of a word",
			restore, duck)
	}

	// The rail runs dry, and only then does the programme come back.
	d, _ = d.Step(Built{ID: BurstID("a2"), Script: Say("A flash flood warning has been declared.")})
	d, fx = d.Step(Finished{ID: BurstID("a2")})
	if _, restore := ducks(fx); restore != 1 {
		t.Errorf("the rail is dry and the programme was not given back: %d restores in %v", restore, describeAll(fx))
	}
}
