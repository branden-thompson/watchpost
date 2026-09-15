package lineup

// fence_prepare_test.go — an out-of-fence card must not block the rail at
// PREPARATION, the way it already cannot block it at the air.
//
// FOUND BY RED TEAM AT BUILD EXIT, 0.16.0 (2026-09-15), reproduced through this
// release's own headline journey: Observer queues alerts under a wide fence,
// the operator swaps to Broadcaster, and `refence` narrows to the station's
// own radius. The far card was already STANDBY, so it stayed STANDBY — and
// `toPrepare` stops at any report standing by.
//
// `Next` HAS CARRIED THE SKIP SINCE D-75 AND SAYS WHY IN AS MANY WORDS:
// "refusing it there would let it block every admissible card behind it, and a
// hazard in the operator's own town would wait on one that is not." That is
// exactly what happened one function along, because the rule was taught to the
// air and not to the walk that feeds it.

import "testing"

// A HAZARD FIVE MILES AWAY MUST NOT WAIT ON ONE IN KANSAS.
func TestAnOutOfFenceCardDoesNotBlockPreparation(t *testing.T) {
	far := aBurst(t, "far", 37.2, -99.8)     // Kansas, from an Oceanside station
	near := aBurst(t, "near", 33.31, -117.3) // ~5 miles from the transmitter

	l := Lineup{}
	for _, c := range []Card{far, near} { // the far one FIRST: it is what blocks
		next, err := l.Queue(AlertRail, c)
		if err != nil {
			t.Fatalf("seeding %s: %v", c.ID, err)
		}
		l = next
	}
	// THE FAR CARD REACHES STANDBY WHILE THE FENCE IS STILL WIDE, which is the
	// state the journey actually produces — Observer's fence admitted it.
	l = advanceTo(t, l, "far", Standby)

	d := Director{lineup: l, settings: Settings{
		Fence: Fence{RadiusMi: 25, Lat: 33.24, Lon: -117.29, HasOrigin: true}}}
	d = d.refence()

	if !d.lineup.cardByID(t, "far").OutOfFence {
		t.Fatal("fixture: the far card must be out of fence after the narrowing")
	}
	c, track, ok := d.lineup.toPrepare()
	if !ok {
		t.Fatal("preparation offered NOTHING: the out-of-fence card standing by stopped the walk, " +
			"so the near hazard is never built and can never take the air")
	}
	if c.ID != "near" || track != AlertRail {
		t.Errorf("preparation offered %q on track %v; the fence admits only the near hazard", c.ID, track)
	}
}

// AND THE AIR AGREES, which is the property the two functions must share: a
// card the air will never offer must never be the card preparation waits on.
func TestPreparationAndTheAirAgreeAboutTheFence(t *testing.T) {
	far := aBurst(t, "far", 37.2, -99.8)
	near := aBurst(t, "near", 33.31, -117.3)

	l := Lineup{}
	for _, c := range []Card{far, near} {
		next, err := l.Queue(AlertRail, c)
		if err != nil {
			t.Fatal(err)
		}
		l = next
	}
	l = advanceTo(t, l, "far", Standby)
	d := Director{lineup: l, settings: Settings{
		Fence: Fence{RadiusMi: 25, Lat: 33.24, Lon: -117.29, HasOrigin: true}}}
	d = d.refence()

	air, _, airOK := d.lineup.Next()
	prep, _, prepOK := d.lineup.toPrepare()
	if !airOK || !prepOK {
		t.Fatalf("air ok=%v prep ok=%v; both must find the near hazard", airOK, prepOK)
	}
	if air.ID != prep.ID {
		t.Errorf("the air offers %q and preparation offers %q — the two disagree about the fence", air.ID, prep.ID)
	}
}

// advanceTo moves one card on the rail to a state, through the real machine.
func advanceTo(t *testing.T, l Lineup, id string, to State) Lineup {
	t.Helper()
	for i, c := range l.tracks[AlertRail] {
		if c.ID != id {
			continue
		}
		next, err := c.To(to)
		if err != nil {
			t.Fatalf("moving %s to %v: %v", id, to, err)
		}
		l.tracks[AlertRail] = append(append([]Card{}, l.tracks[AlertRail][:i]...),
			append([]Card{next}, l.tracks[AlertRail][i+1:]...)...)
		return l
	}
	t.Fatalf("no card %q on the rail", id)
	return l
}

// cardByID reads one card back out of the rail.
func (l Lineup) cardByID(t *testing.T, id string) Card {
	t.Helper()
	for _, c := range l.tracks[AlertRail] {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("no card %q on the rail", id)
	return Card{}
}
