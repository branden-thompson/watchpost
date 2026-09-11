package app

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

var bonsallCfg = config.Location{Label: "Bonsall, CA", Zip: "92003", Lat: 33.2881, Lon: -117.2256, TZ: "America/Los_Angeles"}

// THE SCHEDULE IS FED BY THE STATION'S POOL, NOT THE LISTENER'S WATCHLIST.
//
// THE RULING (HUM LEAD, 2026-09-10): "the Observer watchlist is its lineup, and
// a different rolling window / stack / list needs to serve as Broadcaster
// Location Pool for producers to create the lineup."
//
// IT ASKS THE FUNCTION THE WIRING USES. `startSchedule` takes three seams that
// all read one list, and the defect this release keeps producing is a wiring no
// test drives — so the choice lives in `producer()` and this is what asserts it.
func TestTheProducerOffersTheStationsPoolNotTheWatchlist(t *testing.T) {
	lp := &livePipelines{idx: indexForTest(t)}
	// A WATCHLIST DELIBERATELY NOWHERE NEAR THE STATION. If the schedule were
	// still fed by it, every name below would come back.
	lp.setWatch([]snapshot.LocationRef{
		{Label: "Anchorage, AK", Lat: 61.2181, Lon: -149.9003},
		{Label: "Miami, FL", Lat: 25.7617, Lon: -80.1918},
	})
	lp.setStation(stationFrom(config.Config{Locations: []config.Location{bonsallCfg}}))

	got := lp.producer()()
	if len(got) < 10 {
		t.Fatalf("the pool fills the console's ten slots; got %d", len(got))
	}
	for _, r := range got {
		if r.Label == "Anchorage, AK" || r.Label == "Miami, FL" {
			t.Errorf("the listener's watchlist reached the station's pool: %q", r.Label)
		}
	}
	if got[0].Label != bonsallCfg.Label {
		t.Errorf("the station reads where it transmits from first; got %q", got[0].Label)
	}
	// AND THE TWO LISTS ARE GENUINELY DIFFERENT THINGS.
	if len(lp.currentWatch()) != 2 {
		t.Errorf("the listener still has their own watchlist; got %d", len(lp.currentWatch()))
	}
}

// A BORROWED EPICENTRE FOLLOWS THE LISTENER; A CHOSEN ONE DOES NOT.
//
// THE FALLBACK IS WHAT MAKES THE SPLIT FREE for an install that exists today —
// and it is also what would make a station silently wander if nothing re-derived
// the pool when the default location moved.
func TestTheStationFollowsTheDefaultOnlyWhileItIsBorrowingIt(t *testing.T) {
	moved := []snapshot.LocationRef{{Label: "Reno, NV", Lat: 39.5296, Lon: -119.8138}}

	borrowing := &livePipelines{idx: indexForTest(t)}
	borrowing.setStation(stationFrom(config.Config{Locations: []config.Location{bonsallCfg}}))
	next, pool, ok := borrowing.reStation(moved)
	if !ok {
		t.Fatal("a borrowed epicentre moves with the listener's default location")
	}
	if next.transmitter.Label != "Reno, NV" || len(pool) == 0 || pool[0].Label != "Reno, NV" {
		t.Errorf("and the pool is re-derived around it; got %q / %d", next.transmitter.Label, len(pool))
	}

	own := &livePipelines{idx: indexForTest(t)}
	own.setStation(stationFrom(config.Config{
		Locations:   []config.Location{{Label: "Vista, CA", Lat: 33.2, Lon: -117.24}},
		Broadcaster: config.Broadcaster{Transmitter: bonsallCfg},
	}))
	if _, _, ok := own.reStation(moved); ok {
		t.Error("a station with its own transmitter does not move because the listener did")
	}
}

// AND A STATION WITH NOWHERE TO TRANSMIT FROM OFFERS NOTHING, rather than the
// whole country.
func TestAStationWithNoEpicentreOffersNothing(t *testing.T) {
	lp := &livePipelines{idx: indexForTest(t)}
	lp.setStation(stationFrom(config.Config{}))
	if got := lp.producer()(); len(got) != 0 {
		t.Errorf("no transmitter is no region; got %d", len(got))
	}
}

func indexForTest(t *testing.T) *geodata.Index {
	t.Helper()
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

// AND THE POOL FILLS THE CONSOLE'S WINDOW, END TO END.
//
// THE DEFECT THIS CLOSES IS F-82, MEASURED: `ReadID` is a pure function of the
// ref and the lineup refuses a duplicate identity, so the schedule could never
// be deeper than the number of DISTINCT places its producer offered. With the
// listener's watchlist that was three; with the station's pool it is twenty-five,
// and the console draws ten.
//
// IT DRIVES `startSchedule` ITSELF, through the same `producer()` the wiring
// uses — the D-54 shape, because a unit test that hands the Director its own
// proposals proves nothing about what the station will actually offer.
func TestTheStationsPoolFillsTheConsolesWindow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lp := &livePipelines{idx: indexForTest(t)}
	lp.setStation(stationFrom(config.Config{Locations: []config.Location{bonsallCfg}}))

	var mu sync.Mutex
	deepest := 0
	publish := func(m tea.Msg) {
		if lm, ok := m.(tty.LineupMsg); ok {
			mu.Lock()
			deepest = max(deepest, len(lm.Lineup.Projection(lineup.MainTrack)))
			mu.Unlock()
		}
	}
	nar := testDirector(nil, func(tea.Msg) {})
	tick := &tickerDeck{muted: &atomic.Bool{}, seen: loadSeen(t.TempDir(), time.Hour), alerts: newAlertStore()}
	s := startSchedule(ctx, nar, nil, func() render.Clock { return render.Clock12 }, nil,
		lp.producer(), lp.currentWatch, tick, publish, nil)
	if s == nil {
		t.Fatal("the schedule refused to start")
	}
	s.carry(lineup.Aired{To: lineup.AirProgramme}) // the console holds the air (D-74)
	s.carry(lineup.Powered{To: lineup.Running})

	deadline := time.Now().Add(10 * time.Second)
	got := 0
	for time.Now().Before(deadline) {
		mu.Lock()
		got = deepest
		mu.Unlock()
		if got >= tty.MainTrackSlots {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got != tty.MainTrackSlots {
		t.Errorf("the station's pool fills the console's %d slots; it reached %d", tty.MainTrackSlots, got)
	}
}

// THE MONITOR'S ROTATION RESOLVES AGAINST THE LISTENER'S WATCHLIST (D-76).
//
// THE REGRESSION THIS PINS WAS MINE, AND IT WAS FOUND BY DRAWING THE FLOW rather
// than by a gate. D-72 moved all three of `startSchedule`'s list-reading seams
// from the watchlist to the station's pool, on the reasoning that a Director
// scheduling a location its own Composer cannot resolve gets it benched by
// D-67's cool-off. True of `propose` and `compose`. NOT true of `cutTo`, which
// serves `advanceBed` — the operator's own rotation, moving through their own
// watchlist. A watched location outside the station's pool stopped resolving,
// and the tune died as `schedule:tune-unknown` with nothing said.
func TestTheCutOverResolvesAgainstTheWatchlistNotThePool(t *testing.T) {
	// A LISTENER WATCHING SOMEWHERE THE STATION DOES NOT REACH — the HUM LEAD's
	// own case: Lone Pine, listened to from a station in Bonsall.
	lonePine := snapshot.LocationRef{Label: "Lone Pine, CA", Lat: 36.6060, Lon: -118.0640}
	bishop := snapshot.LocationRef{Label: "Bishop, CA", Lat: 37.3614, Lon: -118.3951}
	watch := func() []snapshot.LocationRef { return []snapshot.LocationRef{bishop, lonePine} }

	lp := &livePipelines{idx: indexForTest(t)}
	lp.setStation(stationFrom(config.Config{Locations: []config.Location{bonsallCfg}}))
	pool := lp.producer()

	// THE FIXTURE IS ASSERTED FIRST: Lone Pine really is outside the station's
	// pool, so resolving it can only come from the watchlist.
	for _, r := range pool() {
		if r.Label == lonePine.Label {
			t.Fatal("the fixture's location is inside the pool; it pins nothing")
		}
	}
	if _, ok := refFor(pool, string(snapshot.Key(lonePine))); ok {
		t.Fatal("the pool must not resolve it, or this test proves nothing")
	}
	if _, ok := refFor(watch, string(snapshot.Key(lonePine))); !ok {
		t.Fatal("the monitor's rotation resolves a watched location against the WATCHLIST")
	}

	// AND THE WIRING USES THE RIGHT ONE, driven through `startSchedule` itself.
	//
	// A PLANT SAID THIS WAS NEEDED. The assertions above compare the two lists
	// and prove nothing about which one `cutTo` is built from — so swapping the
	// wiring back to the pool SURVIVED them. The same shape as `producer()`'s: a
	// call site cannot be asserted, so the test has to drive it.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.Setenv("WATCHPOST_MAINTRACK", "dark") // so a tune the deck accepts reports it
	deck, _ := offlineDeck(t)
	deck.pref = tty.ModeSynth
	tick := &tickerDeck{muted: &atomic.Bool{}, seen: loadSeen(t.TempDir(), time.Hour), alerts: newAlertStore()}
	s := startSchedule(ctx, testDirector(nil, func(tea.Msg) {}), nil,
		func() render.Clock { return render.Clock12 }, deck, pool, watch, tick, func(tea.Msg) {}, nil)
	if s == nil {
		t.Fatal("the schedule refused to start")
	}
	// OBSERVED AT THE DECK, after the schedule has wired itself: a tune that
	// RESOLVED reports the need it found, and a tune that did not resolve
	// reports nothing at all (`schedule:tune-unknown`, silently — which is
	// exactly how the regression hid).
	var mu sync.Mutex
	var tuned []string
	deck.mu.Lock()
	deck.emit = func(ev lineup.Event) {
		if n, ok := ev.(lineup.NeedsRead); ok {
			mu.Lock()
			tuned = append(tuned, n.Ref)
			mu.Unlock()
		}
	}
	deck.mu.Unlock()
	// THE MONITOR ASKS FOR THE WATCHED LOCATION, which only the watchlist holds.
	// A ROTATION OF TWO, so there is somewhere to move ON to: the bed advances
	// from Bishop to Lone Pine, and only the watchlist can resolve either.
	s.carry(lineup.Programme{
		Watchlist: []string{string(snapshot.Key(bishop)), string(snapshot.Key(lonePine))},
		Dwell:     time.Minute,
	})
	s.carry(lineup.Monitored{Running: true})
	s.carry(lineup.Tuned{Ref: string(snapshot.Key(bishop)), Live: true})
	s.carry(lineup.Ended{})

	deadline := time.Now().Add(5 * time.Second)
	want := string(snapshot.Key(lonePine))
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(tuned)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	mu.Lock()
	got := append([]string(nil), tuned...)
	mu.Unlock()
	if len(got) == 0 || got[0] != want {
		t.Errorf("the cut-over never reached the watched location; the deck was asked for %v, want %q", got, want)
	}
}
