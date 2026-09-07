package lineup

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
)

// DR-24 — EVERY EXIT FROM ON AIR CARRIES ITS RELEASE.
//
// The cue and its release are paired, with the same guarantee the duck already
// has. What made this a requirement rather than a tidy-up: the band was
// released on ONE path, with five early returns above it that sent nothing,
// while the audio side was released unconditionally. So the audio came back and
// the ticker kept a callout for a card that had stopped existing — a station
// showing an alert it is no longer reading.
//
// IT IS A PROPERTY OF Step's OUTPUT, not a discipline about call sites. That
// distinction is the requirement: "remember to send the release" is exactly what
// failed before, five times in one function. So this asserts the property
// directly — after any step, a card that WAS on air and now is not must have a
// release in that step's effects — and every exit is driven through it rather
// than each being spot-checked.
//
//	exit            | reached by
//	----------------+----------------------------------------------------
//	read in full    | Finished
//	cancelled       | Failed
//	discarded       | Failed for a card the schedule is still holding
//	superseded      | a second card taking the air (Observer never drops an
//	                | admitted card, DR-3 — so this is asserted as its ABSENCE:
//	                | nothing supersedes, and nothing is left holding a callout)
//	context ended   | the pump's own cancellation, which reaches Step as Failed

// onAirDirector returns a director with the first burst's takeover reading and
// a second one waiting behind it, and the reading card's ID.
//
// TWO BURSTS, because a burst is one card (MVS-D-77): the waiting card is what
// the discarded-before-the-air case needs, and the alerts inside a burst are
// content rather than cards.
func onAirDirector(t *testing.T) (Director, string) {
	t.Helper()
	d := New(Settings{Max: 10}, planNow)
	d, _ = run(d, Powered{To: Running},
		Arrived{Arrivals: many("a", category.Warnings, 2)},
		Arrived{Arrivals: many("b", category.Warnings, 2)})
	d, _ = run(d, Built{ID: BurstID("a00"), Script: Say("words")})
	id, on := d.Lineup().OnAir()
	if !on {
		t.Fatal("the fixture must have a card on the air")
	}
	return d, id.ID
}

// stepReleasesTheAir is the PROPERTY: whatever the event, a card that leaves the
// air leaves a release behind it.
func stepReleasesTheAir(t *testing.T, d Director, ev Event) {
	t.Helper()
	before, wasOn := d.Lineup().OnAir()
	next, fx := d.Step(ev)
	_, stillOn := next.Lineup().OnAir()
	if !wasOn || stillOn {
		return // it did not leave the air on this step; nothing is owed
	}
	for _, f := range fx {
		if r, ok := f.(ReleaseTicker); ok && r.ID == before.ID {
			return
		}
	}
	t.Errorf("%T took %s off the air and released nothing: %v", ev, before.ID, describeAll(fx))
}

func TestDR24ReadInFullReleasesTheBand(t *testing.T) {
	d, id := onAirDirector(t)
	stepReleasesTheAir(t, d, Finished{ID: id})
}

func TestDR24AFailureReleasesTheBand(t *testing.T) {
	d, id := onAirDirector(t)
	stepReleasesTheAir(t, d, Failed{ID: id, Reason: "cancelled"})
}

// THE RELEASE NAMES THE CARD, and it is the card that was reading. A release
// carrying the wrong id clears whatever callout the band is legitimately
// showing — which is the same defect in the other direction.
func TestDR24TheReleaseNamesTheCardThatWasReading(t *testing.T) {
	d, id := onAirDirector(t)
	_, fx := d.Step(Finished{ID: id})
	seen := 0
	for _, f := range fx {
		if r, ok := f.(ReleaseTicker); ok {
			seen++
			if r.ID != id {
				t.Errorf("the release names %q, the card that was reading is %q", r.ID, id)
			}
		}
	}
	if seen != 1 {
		t.Errorf("exactly one release per exit, got %d in %v", seen, describeAll(fx))
	}
}

// NOTHING THAT DOES NOT LEAVE THE AIR RELEASES ANYTHING. A release sent while a
// card is still reading clears the callout for the thing being read — the same
// stale-band defect, arrived at from the opposite side.
func TestDR24AnEventThatKeepsTheAirReleasesNothing(t *testing.T) {
	d, id := onAirDirector(t)
	for _, ev := range []Event{
		Tick{Now: planNow.Add(time.Minute)},
		Arrived{Arrivals: many("c", category.Warnings, 1)},
		Built{ID: BurstID("b00"), Script: Say("more words")},
		Powered{To: Running},
		Programme{Watchlist: []string{"x"}, Dwell: time.Minute},
		Failed{ID: "not-a-card", Reason: "gone"},
	} {
		next, fx := d.Step(ev)
		if _, on := next.Lineup().OnAir(); !on {
			t.Fatalf("%T took the card off the air; this case is about events that do not", ev)
		}
		for _, f := range fx {
			if r, ok := f.(ReleaseTicker); ok {
				t.Errorf("%T released %q while it was still reading", ev, r.ID)
			}
		}
		_ = id
	}
}

// AND THE PROPERTY HOLDS FOR EVERY EVENT THE DIRECTOR ACCEPTS, not only the two
// that happen to take a card off the air today. An event added later that ends a
// read without releasing the band fails here rather than in a listener's session
// — which is what "a property of Step's output" means.
func TestDR24EveryEventLeavesTheBandConsistent(t *testing.T) {
	events := func(id string) []Event {
		return []Event{
			Tick{Now: planNow.Add(time.Minute)},
			Arrived{Arrivals: many("c", category.Warnings, 1)},
			Built{ID: BurstID("b00"), Script: Say("more words")},
			Finished{ID: id},
			Failed{ID: id, Reason: "cancelled"},
			Powered{To: Running},
			Powered{To: Stopped},
			Tuned{Ref: "x", Live: true},
			Programme{Watchlist: []string{"x"}, Dwell: time.Minute},
			Ended{},
		}
	}
	// A fresh director per event: the property is about ONE step, and reusing
	// one would let an earlier event decide what a later one is asked.
	for _, ev := range events(BurstID("a00")) {
		d, id := onAirDirector(t)
		if id != BurstID("a00") {
			t.Fatalf("the fixture's card changed id to %q; the table above names %s", id, BurstID("a00"))
		}
		stepReleasesTheAir(t, d, ev)
	}
}

// A CARD THAT NEVER HELD THE BAND RELEASES NOTHING WHEN IT LEAVES.
//
// The release is paired with the CUE, not with the card. A card discarded from
// standby was never cued, so releasing on its way out clears the callout for
// whatever IS reading — the same stale-band defect as DR-24's original, with
// the band cleared too early instead of too late.
//
// The code says this in as many words and nothing measured it: the mutant that
// released on every exit, on air or not, SURVIVED the rest of this file. Every
// case here had the leaving card ON the air, so the condition being deleted was
// true in all of them.
func TestDR24ACardDiscardedBeforeTheAirReleasesNothing(t *testing.T) {
	d, onAir := onAirDirector(t)

	// The second burst's takeover is held but has never read: it is behind the
	// one on the air.
	var waiting string
	for tr := Track(0); tr < numTracks; tr++ {
		for _, c := range d.lineup.Cards(tr) {
			if c.ID != onAir {
				waiting = c.ID
			}
		}
	}
	if waiting == "" {
		t.Fatal("the fixture must hold a card that is not on the air")
	}

	next, fx := d.Step(Failed{ID: waiting, Reason: "the voice could not render"})
	for _, f := range fx {
		if r, ok := f.(ReleaseTicker); ok {
			t.Errorf("discarding %s released the band (%q) — it never held it, and %s is still reading",
				waiting, r.ID, onAir)
		}
	}
	// And the card that IS reading keeps the air: the failure was routed around.
	if id, on := next.Lineup().OnAir(); !on || id.ID != onAir {
		t.Errorf("the reading card must keep the air, got %v/%v", id.ID, on)
	}
}
