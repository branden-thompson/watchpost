package globalfeed

import (
	"fmt"
	"testing"
	"time"
)

// WITHIN A LANE, THE CAP KEEPS THE WORST — not merely the newest.
//
// Sort reaches Severity only when two events share an exactly equal instant,
// which real feed data never produces, so ordering the cut by time alone throws
// away the most severe. A tornado warning from forty minutes ago must not be
// evicted by fresher, milder events in its own lane.
//
// That a whole LANE cannot vanish is a different guarantee, asserted by
// TestAnOutbreakInOneLaneDoesNotEmptyTheOthers — this test is captioned for the
// one it actually makes.
func TestTheCapKeepsTheMostSevereNotTheMostRecent(t *testing.T) {
	now := time.Now()
	// ONE LANE, so the cap has to choose: a tornado warning and heat advisories
	// are both LaneWarning. A fixture that puts the severe event alone in its
	// own lane cannot test this — nothing there is ever evicted.
	var fetched []Event
	for i := range MaxPerLane + 12 { // newer and milder, more than the lane holds
		fetched = append(fetched, Event{ID: fmt.Sprint("adv", i), Class: ClassSevereWx, Severity: SevYellow,
			Type: "Heat Advisory", At: now.Add(-time.Duration(i) * time.Second), HasPoint: true})
	}
	// The severe one is BURIED, not first. At index 0 it survives any truncation
	// by insertion order alone, and the test then passes whatever the cut does —
	// which is how it passed while the severity rule was not being applied.
	fetched = append(fetched, Event{ID: "tornado", Class: ClassSevereWx, Severity: SevRed,
		Type: "Tornado Warning", At: now.Add(-40 * time.Minute), HasPoint: true})
	stack, _ := Merge(fetched, map[string]bool{})
	// EXACTLY the cap: one lane, over-full, so the bound itself is asserted and
	// not merely an upper limit no fixture reaches.
	if len(stack) != MaxPerLane {
		t.Fatalf("one over-full lane must be cut to exactly MaxPerLane=%d, got %d", MaxPerLane, len(stack))
	}
	kept := false
	for _, e := range stack {
		if e.ID == "tornado" {
			kept = true
		}
	}
	if !kept {
		t.Error("the most severe event was evicted by fresher, milder ones in its own lane — the listener loses the tornado warning and keeps thirty heat advisories")
	}
	// And the marquee's own order is unchanged: still most recent first.
	for i := 1; i < len(stack); i++ {
		if stack[i].At.After(stack[i-1].At) {
			t.Fatalf("the surviving stack must still read most-recent-first: %v before %v", stack[i-1].At, stack[i].At)
		}
	}
}

// AN OUTBREAK IN ONE LANE MUST NOT EMPTY THE OTHERS.
//
// The marquee rotates through the lanes, so a lane with no events disappears
// from the rotation entirely. A national tornado outbreak routinely issues more
// warnings than the whole stack can hold, and every one of them is top-severity
// — so ordering the cut by severity does not help: a live hurricane and a live
// significant quake are evicted by volume alone, and the listener is never told
// about either.
func TestAnOutbreakInOneLaneDoesNotEmptyTheOthers(t *testing.T) {
	now := time.Now()
	var fetched []Event
	for i := range MaxPerLane + 10 { // more warnings than the whole budget
		fetched = append(fetched, Event{ID: fmt.Sprint("warn", i), Class: ClassSevereWx, Severity: SevRed,
			Type: "Tornado Warning", At: now.Add(-time.Duration(i) * time.Second), HasPoint: true})
	}
	fetched = append(fetched,
		Event{ID: "quake", Class: ClassQuake, Severity: SevRed, Type: "Earthquake",
			At: now.Add(-30 * time.Minute), HasPoint: true},
		Event{ID: "storm", Class: ClassTropical, Severity: SevRed, Type: "Hurricane Dolly",
			At: now.Add(-45 * time.Minute), HasPoint: true})

	stack, _ := Merge(fetched, map[string]bool{})
	byClass := map[Class]int{}
	for _, e := range stack {
		byClass[e.Class]++
	}
	for _, want := range []struct {
		class Class
		name  string
	}{{ClassQuake, "Disasters"}, {ClassTropical, "Marine"}, {ClassSevereWx, "Warnings"}} {
		if byClass[want.class] == 0 {
			t.Errorf("the %s lane is empty while its events are live — it vanishes from the rotation (stack=%d, by class %v)",
				want.name, len(stack), byClass)
		}
	}
}

// A HAZARD THAT ARRIVES DURING AN OUTBREAK IS STILL ANNOUNCED.
//
// fresh is what the app sounds a tone for and reads aloud, and the caller
// seen-marks everything active afterwards — so an event dropped by the cap
// before fresh is computed is not merely delayed, it is silenced permanently.
func TestANewHazardDuringAnOutbreakIsStillAnnounced(t *testing.T) {
	now := time.Now()
	var fetched []Event
	for i := range MaxPerLane + 10 {
		fetched = append(fetched, Event{ID: fmt.Sprint("warn", i), Class: ClassSevereWx, Severity: SevRed,
			Type: "Tornado Warning", At: now.Add(-time.Duration(i+60) * time.Second), HasPoint: true})
	}
	// A special weather statement issued THIS MINUTE. It is less severe than a
	// tornado warning, which is exactly why ordering the cut by severity drops
	// it — while ordering by recency, as the previous cap did, kept it.
	fetched = append(fetched, Event{ID: "statement", Class: ClassSevereWx, Severity: SevYellow,
		Type: "Special Weather Statement", At: now, HasPoint: true})

	_, fresh := Merge(fetched, map[string]bool{})
	announced := false
	for _, e := range fresh {
		if e.ID == "statement" {
			announced = true
		}
	}
	if !announced {
		t.Errorf("a statement issued this minute must be announced; the cap silenced it before fresh was computed (fresh=%d)", len(fresh))
	}
}

// EVERY LANE SURVIVES THE CAP.
//
// capPerLane iterates laneOrder, so a lane missing from it is not capped — it
// is dropped, and its alerts never reach the stack at all. Nothing else would
// say so: no error, no panic, and a marquee that looks like a quiet day.
func TestEveryLaneReachesTheStack(t *testing.T) {
	now := time.Now()
	// One event per lane, built through the real classifier so the fixture
	// cannot claim a lane the product would not actually land in.
	byLane := map[Lane]Event{}
	for _, e := range []Event{
		{ID: "e", Class: ClassSevereWx, Type: "Evacuation Immediate", At: now, HasPoint: true},
		{ID: "q", Class: ClassQuake, Type: "Earthquake", At: now, HasPoint: true},
		{ID: "t", Class: ClassTropical, Type: "Hurricane", At: now, HasPoint: true},
		{ID: "w", Class: ClassSevereWx, Type: "Tornado Warning", At: now, HasPoint: true},
		{ID: "c", Class: ClassSevereWx, Type: "Tornado Watch", At: now, HasPoint: true},
	} {
		byLane[LaneOf(e)] = e
	}
	if len(byLane) != len(laneOrder()) {
		t.Fatalf("the fixture must cover every lane the feed can produce: %d of %d", len(byLane), len(laneOrder()))
	}
	// LaneOf must never return a category laneOrder does not carry — that event
	// would be dropped by the cap with nothing to say so.
	carried := map[Lane]bool{}
	for _, l := range laneOrder() {
		carried[l] = true
	}
	for l := range byLane {
		if !carried[l] {
			t.Errorf("LaneOf produced %v, which laneOrder does not carry — its events never reach the stack", l)
		}
	}
	var fetched []Event
	for _, e := range byLane {
		fetched = append(fetched, e)
	}
	stack, _ := Merge(fetched, map[string]bool{})
	got := map[Lane]bool{}
	for _, e := range stack {
		got[LaneOf(e)] = true
	}
	for _, l := range laneOrder() {
		if !got[l] {
			t.Errorf("lane %v never reached the stack — laneOrder does not carry it", l)
		}
	}
}
