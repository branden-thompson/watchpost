package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// The HUM LEAD's own admit-case, at the producer (BD-6, ratified 2026-09-02):
// "an M7.5 in Los Angeles (~120 mi)" reaches a listener whose radius is 50.
var (
	reachHome = snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.2886, Lon: -117.2247}
	reachLA   = [2]float64{34.0522, -118.2437}
)

func quakeAt(id string, mag float64, at [2]float64, when time.Time) globalfeed.Event {
	m := mag
	return globalfeed.Event{
		ID: id, Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Los Angeles, CA",
		Severity: globalfeed.SevYellow, At: when, HasPoint: true, Lat: at[0], Lon: at[1],
		Quake: &globalfeed.QuakeDetail{Mag: &m},
	}
}

func reachDeck(t *testing.T, radiusMi int64) *tickerDeck {
	t.Helper()
	nar := testDirector(&scriptVoice{}, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	r := &atomic.Int64{}
	r.Store(radiusMi)
	return &tickerDeck{
		send:   func(tea.Msg) {},
		muted:  &atomic.Bool{},
		voice:  nar,
		seen:   loadSeen(t.TempDir(), time.Hour),
		radius: r,
		watch:  func() []snapshot.LocationRef { return []snapshot.LocationRef{reachHome} },
	}
}

// C-3 — THE SIGNIFICANCE REACH REACHES THE LISTENER.
//
// BD-6 was implemented in platform/lineup, pinned there, and guarded by two
// mutants — and connected to nothing. arrivalsOf never set Arrival.ReachMi, so
// the fence compared every distance against zero; and the producer's own radius
// filter dropped the distant quake before the Director ever saw it. The one
// assignment of ReachMi in the whole tree was in fence_test.go, which built the
// Arrival itself — D-12, a pin that does not start where the human starts.
//
// This starts at the events the feed yields and ends at words in the air.
func TestASignificantQuakeReachesAListenerOutsideTheirRadius(t *testing.T) {
	deck := reachDeck(t, 50)
	when := time.Now().Add(-2 * time.Minute)
	// A LOCAL WARNING RIDES ALONG so an empty burst cannot pass for a read one.
	local := globalfeed.Event{
		ID: "local", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "Bonsall, CA",
		Severity: globalfeed.SevRed, At: when, HasPoint: true, Lat: 33.29, Lon: -117.22,
	}
	big := quakeAt("m75", 7.5, reachLA, when)

	// THE FIXTURE IS ASSERTED VALID FIRST: the quake really is outside the
	// radius, so admitting it can only come from the exception.
	if globalfeed.WithinMiles(reachHome.Lat, reachHome.Lon, big.Lat, big.Lon, 50) {
		t.Fatal("the fixture's quake is inside the radius; it pins nothing")
	}

	// THE PRODUCER'S OWN FILTER. This is the half that dropped it first.
	scoped := deck.scopeToRadius([]globalfeed.Event{local, big})
	if len(scoped) != 2 {
		t.Fatalf("the producer's radius filter dropped the significant quake before the Director could see it: kept %d of 2", len(scoped))
	}

	// THE TRANSLATION. A fence with no reach on the arrival compares against zero.
	for _, a := range arrivalsOf(scoped) {
		if a.ID == "m75" && a.ReachMi <= 0 {
			t.Errorf("the producer hands the M7.5 over with no reach, so the fence measures it against zero miles")
		}
	}
	if !deck.fence().Admits(arrivalsOf(scoped)[1]) {
		t.Error("the Director's fence refuses the M7.5 the ruling admits")
	}

	newStation(t, deck).takeover(context.Background(), scoped)
	if !deck.seen.set()["m75"] {
		t.Error("the M7.5 in Los Angeles was never read aloud; BD-6 admits it")
	}
}

// THE CONTROL, and without it the fix above could simply admit everything.
// G-7: the exception is scaled by significance, not unbounded.
func TestAnOrdinaryQuakeOutsideTheRadiusStaysOut(t *testing.T) {
	deck := reachDeck(t, 50)
	small := quakeAt("m40", 4.0, reachLA, time.Now())
	if lineup.QuakeReachMi(4.0) != 0 {
		t.Fatal("the fixture's magnitude buys reach; it pins nothing")
	}
	if got := deck.scopeToRadius([]globalfeed.Event{small}); len(got) != 0 {
		t.Errorf("an M4.0 a hundred miles away reached a 50-mile listener: the exception is not scaled by significance")
	}
}
