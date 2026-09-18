package lineup

// bed_fence_test.go — the bed must come back up when nothing on the rail can
// ever be read.
//
// `givingWay` ASKS THE PROJECTION, like every other rail reader, and not the RAW
// TRACK. The projection drops out-of-fence cards precisely because they are not
// READ; asking the track, a rail holding only cards the fence excludes keeps the
// answer true for ever, so Duck is emitted and Restore never is and the station
// broadcasts the relay at duck gain indefinitely with no voice over it.

import (
	"testing"
	"time"
)

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

// AND THE FOURTH CALL SITE OF THE SAME RULE (D-150).
//
// The fence belongs to `Next`, `Projection`, `toPrepare`, `givingWay` AND
// `refreshStandby` — five sites, and `refreshStandby` is the one easiest to miss
// because it is the last raw-track "does the rail hold anything" question in the
// package. Four of five is the same defect as none.
//
// THE INTERACTION IS WHAT MAKES IT WORSE THAN A MISSED SITE. After D-139 an
// out-of-fence rail card is IMMORTAL and INVISIBLE: `Next` skips it, `toPrepare`
// skips it so it never reaches Standby with a BuiltAt and can never be dropped
// as stale, and `Projection` hides it so the operator cannot drop it either. So
// this guard stayed permanently true and the main track's standing-by report was
// never re-hydrated for the life of the fence — until it aged past StaleAfter
// and the listener heard "That report is out of date and has been dropped."
func TestAnUnreadableRailDoesNotFreezeTheMainTracksRefresh(t *testing.T) {
	far := aBurst(t, "far", 37.2, -99.8) // Kansas, from an Oceanside station
	rep, err := Propose(Card{ID: "rep", Slot: LocationReport, Subject: "rep",
		Headline: "Oceanside, CA", State: Proposed})
	if err != nil {
		t.Fatal(err)
	}
	adm, err := rep.To(Admitted)
	if err != nil {
		t.Fatal(err)
	}
	standby, err := adm.To(Standby)
	if err != nil {
		t.Fatal(err)
	}
	standby.BuiltAt = time.Now().Add(-14 * time.Minute) // past RefreshAfter

	var l Lineup
	if l, err = l.Queue(AlertRail, far); err != nil {
		t.Fatal(err)
	}
	l.tracks[MainTrack] = append(l.tracks[MainTrack], standby)

	d := Director{lineup: l, settings: Settings{
		Fence: Fence{RadiusMi: 25, Lat: 33.24, Lon: -117.29, HasOrigin: true}}}
	d = d.refence()
	d.now = time.Now()

	if len(d.lineup.Projection(AlertRail)) != 0 {
		t.Fatal("fixture: nothing on this rail is readable")
	}
	_, fx := d.refreshStandby()
	if len(fx) == 0 {
		t.Error("the main track's standing-by report is never refreshed while an unreadable card " +
			"sits on the rail — and that card can never leave, so the report ages out and the " +
			"listener is told a report was dropped")
	}
}
