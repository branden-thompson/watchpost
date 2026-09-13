package lineup

// fence_view_test.go — a hazard the fence keeps out is not DRAWN either (D-114).

import "testing"

// THE CARRYOVER THE HUM LEAD DESCRIBED, 2026-09-13:
//
//	Starting watchpost → Defaults to Observer → Alerts queue according to
//	Observer's alert radius → User ctrl+b → Broadcaster UI loads → Meanwhile the
//	alerts from Observer carry over
//
// `refence` MARKED THEM AND NOTHING ACTED ON THE MARK where the operator could
// see it. `nextForAir` has skipped out-of-fence cards since D-75 — so they were
// never READ — and `Projection` went on returning them, so the console drew a
// takeover full of hazards the station would never broadcast.
func TestAnOutOfFenceCardLeavesTheProjection(t *testing.T) {
	near := aBurst(t, "near", 33.24, -117.29)
	far := aBurst(t, "far", 37.2, -99.8) // Kansas, from an Oceanside station

	l := Lineup{}
	for _, c := range []Card{near, far} {
		next, err := l.Queue(AlertRail, c)
		if err != nil {
			t.Fatalf("seeding %s: %v", c.ID, err)
		}
		l = next
	}
	if got := len(l.Projection(AlertRail)); got != 2 {
		t.Fatalf("both cards are on the rail before the fence narrows; got %d", got)
	}

	d := Director{lineup: l, settings: Settings{
		Fence: Fence{RadiusMi: 25, Lat: 33.24, Lon: -117.29, HasOrigin: true}}}
	d = d.refence()

	got := d.lineup.Projection(AlertRail)
	if len(got) != 1 {
		t.Fatalf("the projection draws %d cards inside a 25-mile fence, want 1", len(got))
	}
	if got[0].ID != "near" {
		t.Errorf("the projection kept %q; the fence admits only the near hazard", got[0].ID)
	}
	// AND THE AIR AGREES WITH THE FRAME, which is the point: the two used to
	// disagree, and the frame was the one the operator was reading.
	c, _, ok := d.lineup.Next()
	if !ok || c.ID != "near" {
		t.Errorf("the air offers %v (ok=%v); the frame and the air must name one card", c.ID, ok)
	}
}

// AND A WIDENING GIVES THEM BACK, because `refence` runs both ways and a rail
// that only ever held would go quiet and stay quiet.
func TestAWideningReturnsThemToTheProjection(t *testing.T) {
	l := Lineup{}
	next, err := l.Queue(AlertRail, aBurst(t, "far", 37.2, -99.8))
	if err != nil {
		t.Fatal(err)
	}
	d := Director{lineup: next, settings: Settings{
		Fence: Fence{RadiusMi: 25, Lat: 33.24, Lon: -117.29, HasOrigin: true}}}
	if d = d.refence(); len(d.lineup.Projection(AlertRail)) != 0 {
		t.Fatal("the narrow fence did not hold it, so the widening proves nothing")
	}
	d.settings.Fence = Fence{} // All
	if got := len(d.refence().lineup.Projection(AlertRail)); got != 1 {
		t.Errorf("a widened fence returned %d cards to the frame, want 1", got)
	}
}

// aBurst is a takeover carrying one pointed hazard.
func aBurst(t *testing.T, id string, lat, lon float64) Card {
	t.Helper()
	c, err := Propose(Card{ID: id, Slot: BreakingAlert, Subject: id, Headline: id, State: Proposed,
		From: []Arrival{{ID: id + ":a", Headline: "SEVERE THUNDERSTORM WARNING", Subject: id,
			HasPoint: true, Lat: lat, Lon: lon}}})
	if err != nil {
		t.Fatalf("proposing %s: %v", id, err)
	}
	c, err = c.To(Admitted)
	if err != nil {
		t.Fatalf("admitting %s: %v", id, err)
	}
	return c
}
