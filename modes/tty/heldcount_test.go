package tty

// heldcount_test.go — the held-hazard band must count HAZARDS, and only the
// ones the station could actually read.
//
// TWO WAYS TO GET ONE LINE WRONG, on the console's loudest safety surface:
//
//  1. It counted CARDS. A burst is ONE card carrying many arrivals (MVS-D-77),
//     so five hazards held read "1 HAZARD(S) HELD" — and the escalation ladder
//     keys off that number.
//  2. It read `Cards()` where every other rail reader reads `Projection()`,
//     which drops out-of-fence cards precisely because they are not READ. So an
//     out-of-fence burst raised "1 HAZARD(S) HELD … Go ON AIR to read them"
//     while the takeover box was empty and going on air would read nothing —
//     a false instruction that escalates to "may be dropped unread".

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// standbyWith puts a station on standby holding exactly these rail cards.
func standbyWith(t *testing.T, after time.Duration, fence lineup.Fence, cards ...lineup.Card) Broadcaster {
	t.Helper()
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	b := NewBroadcaster()
	b.width, b.height = 150, 74
	b.now = func() time.Time { return base }

	var l lineup.Lineup
	for _, c := range cards {
		next, err := l.Queue(lineup.AlertRail, c)
		if err != nil {
			t.Fatalf("seeding %s: %v", c.ID, err)
		}
		l = next
	}
	b, _ = b.Update(LineupMsg{Lineup: lineup.RefencedForTest(l, fence)})
	b, _ = b.Update(StationMsg{Power: lineup.OffAir})
	b.now = func() time.Time { return base.Add(after) }
	return b
}

// A BURST OF FIVE IS FIVE HAZARDS HELD, not one.
func TestTheHeldBandCountsHazardsNotCards(t *testing.T) {
	burst := burstOfForTest(t, "b", 5, 33.31, -117.3)
	b := standbyWith(t, 90*time.Second, wideFenceForTest(), burst)

	got := stripANSITest(strings.Join(b.heldNotice(), " "))
	if !strings.Contains(got, "5 HAZARD(S) HELD") {
		t.Errorf("a burst of five reads %q; a burst is ONE card carrying many hazards, and the "+
			"operator is being told how many are being withheld", strings.TrimSpace(got))
	}
}

// AND A RAIL THE FENCE EXCLUDES HOLDS NOTHING. Saying otherwise instructs the
// operator to go ON AIR to read something going on air will not read.
func TestTheHeldBandIsSilentWhenNothingIsReadable(t *testing.T) {
	far := burstOfForTest(t, "far", 1, 37.2, -99.8) // Kansas, from an Oceanside station
	b := standbyWith(t, 90*time.Second, narrowFenceForTest(), far)

	if rows := b.heldNotice(); rows != nil {
		t.Errorf("the band claims a hazard is held that the fence excludes — going ON AIR "+
			"would read nothing:\n%q", stripANSITest(strings.Join(rows, " ")))
	}
}

// wideFenceForTest admits everything; narrowFenceForTest is the station's own
// 25 miles around Oceanside.
func wideFenceForTest() lineup.Fence { return lineup.Fence{} }
func narrowFenceForTest() lineup.Fence {
	return lineup.Fence{RadiusMi: 25, Lat: 33.24, Lon: -117.29, HasOrigin: true}
}

// burstOfForTest is one rail card carrying n arrivals — the shape MVS-D-77
// rules: a burst is ONE card.
func burstOfForTest(t *testing.T, id string, n int, lat, lon float64) lineup.Card {
	t.Helper()
	from := make([]lineup.Arrival, 0, n)
	for i := range n {
		from = append(from, lineup.Arrival{
			ID: id + ":" + string(rune('a'+i)), Headline: "SEVERE THUNDERSTORM WARNING",
			Subject: id, HasPoint: true, Lat: lat, Lon: lon,
		})
	}
	c, err := lineup.Propose(lineup.Card{ID: id, Slot: lineup.BreakingAlert, Subject: id,
		Headline: id, State: lineup.Proposed, From: from})
	if err != nil {
		t.Fatalf("proposing %s: %v", id, err)
	}
	c, err = c.To(lineup.Admitted)
	if err != nil {
		t.Fatalf("admitting %s: %v", id, err)
	}
	return c
}
