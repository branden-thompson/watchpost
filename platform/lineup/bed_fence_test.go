package lineup

// bed_fence_test.go — the bed must come back up when nothing on the rail can
// ever be read.
//
// FOUND BY RED TEAM AT BUILD EXIT, 0.16.0 (2026-09-15). `givingWay` asked the
// RAW TRACK — "does the rail hold anything" — where every other rail reader
// asks the PROJECTION, which drops out-of-fence cards precisely because they
// are not READ. A rail holding only cards the fence excludes kept the answer
// true for ever: Duck was emitted and Restore never was, so the station
// broadcast the relay at duck gain indefinitely with no voice over it.

import "testing"

// A RAIL THAT CAN NEVER SPEAK MUST NOT HOLD THE BED DOWN.
func TestARailOfUnreadableCardsGivesTheBedBack(t *testing.T) {
	far := aBurst(t, "far", 37.2, -99.8) // Kansas, from an Oceanside station

	l := Lineup{}
	next, err := l.Queue(AlertRail, far)
	if err != nil {
		t.Fatal(err)
	}
	d := Director{lineup: next, settings: Settings{
		Fence: Fence{RadiusMi: 25, Lat: 33.24, Lon: -117.29, HasOrigin: true}}}
	d = d.refence()

	// THE BED MUST BE CARRYING, or this passes for the wrong reason: with
	// nothing underneath there is nothing to duck and `givingWay` is false
	// whatever the rail holds. The first draft of this test omitted it and
	// passed against the defect.
	d.bed.carries = true

	if !d.lineup.cardByID(t, "far").OutOfFence {
		t.Fatal("fixture: the card must be out of fence")
	}
	if len(d.lineup.Projection(AlertRail)) != 0 {
		t.Fatal("fixture: nothing on this rail is readable")
	}
	if d.givingWay() {
		t.Error("the bed is held down for a rail whose every card the fence excludes — " +
			"nothing can ever read, so Restore never comes and the relay stays ducked for ever")
	}
}

// AND A RAIL THAT CAN SPEAK STILL TAKES IT. Fixing the stall by never ducking
// would be the same defect pointed the other way: a hazard read over an
// undipped relay is the thing MVS-D-67 exists to prevent.
func TestAReadableRailStillTakesTheBed(t *testing.T) {
	near := aBurst(t, "near", 33.31, -117.3)

	l := Lineup{}
	next, err := l.Queue(AlertRail, near)
	if err != nil {
		t.Fatal(err)
	}
	d := Director{lineup: next, settings: Settings{
		Fence: Fence{RadiusMi: 25, Lat: 33.24, Lon: -117.29, HasOrigin: true}}}
	d = d.refence()
	d.bed.carries = true

	if len(d.lineup.Projection(AlertRail)) != 1 {
		t.Fatal("fixture: the near hazard is readable")
	}
	if !d.givingWay() {
		t.Error("a readable hazard on the rail must duck the bed under it (MVS-D-67)")
	}
}

// AND THE EDGE IS WHAT IS EMITTED, which is the half MVS-D-67 rests on: a
// drain of several cards dips ONCE.
//
// PINNED HERE BECAUSE THIS PATH WAS BELIEVED DEAD UNTIL 2026-09-15. A comment
// in app/executors.go asserted that nothing in production constructs Duck or
// Restore; P5 wired them through `givingWay` and the comment was not revisited,
// so the effect pair ran for a release with a note on it saying it could not.
// The executor half was covered (TestTheDuckAndItsRestoreReachTheEffector); the
// DIRECTOR's emission was not.
func TestTheBedGivesWayOnceAndTakesItBackOnce(t *testing.T) {
	near := aBurst(t, "near", 33.31, -117.3)
	l := Lineup{}
	next, err := l.Queue(AlertRail, near)
	if err != nil {
		t.Fatal(err)
	}
	d := Director{lineup: next, settings: Settings{
		Fence: Fence{RadiusMi: 25, Lat: 33.24, Lon: -117.29, HasOrigin: true}}}
	d = d.refence()
	d.bed.carries = true

	d, fx := d.giveOrTakeBack()
	if len(fx) != 1 {
		t.Fatalf("a readable hazard over a carrying bed emitted %v, want one Duck", fx)
	}
	if _, ok := fx[0].(Duck); !ok {
		t.Fatalf("emitted %T, want Duck", fx[0])
	}
	// THE SECOND ASK EMITS NOTHING: only the change is sent.
	d, again := d.giveOrTakeBack()
	if len(again) != 0 {
		t.Errorf("a second ask emitted %v; the pair is edge-triggered", again)
	}
	// AND THE RAIL GOING QUIET GIVES IT BACK, exactly once.
	d.lineup.tracks[AlertRail] = nil
	_, back := d.giveOrTakeBack()
	if len(back) != 1 {
		t.Fatalf("an empty rail emitted %v, want one Restore", back)
	}
	if _, ok := back[0].(Restore); !ok {
		t.Fatalf("emitted %T, want Restore", back[0])
	}
}
