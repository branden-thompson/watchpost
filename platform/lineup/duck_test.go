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
	// edge is at the last Finished, and while the bed carries, exclusivity means
	// no main-track card can take the air to be cued alongside it.
	//
	// SO THE RULE IS ENFORCED BY EXCLUSIVITY, not by the order of this list —
	// and a check asserting the position would be the vacuous class this project
	// has shipped five of. IT BECOMES REACHABLE IF EXCLUSIVITY IS EVER RELAXED,
	// which is recorded in the track model for whoever does that.
}

// THE OTHER HALF OF D-32, and it is the half that broke a release. Nothing
// ducks for the programme: a chosen read REPLACES the bed rather than playing
// over it, so there is nothing underneath to dip.
func TestNothingDucksWhenTheBedIsNotCarrying(t *testing.T) {
	d := New(Settings{Max: 5}, time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC))
	d, _ = d.Step(Powered{To: Running})

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
	d, _ = d.Step(Arrived{Arrivals: []Arrival{aTornado()}})
	d, _ = d.Step(Built{ID: BurstID("a1"), Script: Say("A tornado warning has been declared.")})

	_, fx := d.Step(Finished{ID: BurstID("a1")})

	if _, got := ducks(fx); got != 0 {
		t.Errorf("no bed was ducked, so none is given back; got %d Restore effect(s)", got)
	}
}
