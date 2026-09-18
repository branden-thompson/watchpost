package app

import (
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// THE ALERT RAIL IS SCOPED TO THE SURFACE THE OPERATOR IS LOOKING AT (D-73).
//
// THE HUM LEAD'S OWN ACCEPTANCE TEST (2026-09-10): "as long as when I switch to
// Broadcaster I don't hear alerts outside my service radius, then that's fine
// ('it just works') and when I switch back to Observer, that alert track has to
// re-adapt to whatever my filter settings dictate ('it just works')."
//
// BOTH DIRECTIONS, because "re-adapt" is the half that is easy to leave out: a
// fence that widened on the way in and stayed wide on the way back would pass a
// test written only for the first sentence.
func TestTheRailIsScopedToWhicheverSurfaceIsActive(t *testing.T) {
	radius := &atomic.Int64{}
	radius.Store(150) // the listener watches a wide area
	watch := func() []snapshot.LocationRef {
		return []snapshot.LocationRef{{Label: "Bonsall, CA", Lat: 33.2881, Lon: -117.2256}}
	}
	st := func() stationArea {
		return stationArea{transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.2881, Lon: -117.2256}, radiusMi: 25}
	}
	var owner airOwner
	scope := func() airScope {
		return scopeFor(&owner, func() airScope { return listenerScope(radius, watch) }, st)
	}

	if got := scope(); got.radiusMi != 150 {
		t.Errorf("Observer shows the listener's filter; got %v mi", got.radiusMi)
	}
	owner.set(tty.SurfaceBroadcaster)
	if got := scope(); got.radiusMi != 25 {
		t.Errorf("the console shows the station's service area; got %v mi", got.radiusMi)
	}
	owner.set(tty.SurfaceObserver)
	if got := scope(); got.radiusMi != 150 {
		t.Errorf("and going back re-adapts to the listener's filter; got %v mi", got.radiusMi)
	}
}

// A CONSOLE WITH NO TRANSMITTER DOES NOT WIDEN THE LISTENER'S ALERTS.
//
// THE SAFE DIRECTION IS THE LISTENER'S. A station with no epicentre has no
// region of its own, and the alternative — falling through to "All" — would
// silently hand the operator every hazard in the country the moment they looked
// at a console that has not been set up.
func TestAConsoleWithNoEpicentreKeepsTheListenersFence(t *testing.T) {
	radius := &atomic.Int64{}
	radius.Store(40)
	watch := func() []snapshot.LocationRef {
		return []snapshot.LocationRef{{Label: "Bonsall, CA", Lat: 33.2881, Lon: -117.2256}}
	}
	var owner airOwner
	owner.set(tty.SurfaceBroadcaster)
	got := scopeFor(&owner, func() airScope { return listenerScope(radius, watch) }, func() stationArea { return stationArea{radiusMi: 25} })
	if got.radiusMi != 40 || got.lat == 0 {
		t.Errorf("an unset station keeps the listener's fence; got %+v", got)
	}
}

// AND THE DECK ASKS ITS SCOPE, NOT THE LISTENER'S SETTING.
//
// IT DRIVES THE DECK'S OWN `fence()`, which is what the planner reads — a test
// that called `scopeFor` alone would prove the answer and nothing about who asks
// for it, which is the wiring shape this release keeps rebuilding.
func TestTheDecksFenceFollowsTheScope(t *testing.T) {
	deck := &tickerDeck{}
	deck.setScope(func() airScope { return airScope{lat: 33.2881, lon: -117.2256, radiusMi: 25, set: true} })
	f := deck.fence()
	if !f.InForce() || f.RadiusMi != 25 || !f.HasOrigin {
		t.Errorf("the rail's fence is the scope's; got %+v", f)
	}
	deck.setScope(func() airScope { return airScope{} })
	if deck.fence().InForce() {
		t.Error("an unscoped rail is All, which is what it has always been")
	}
	// A RADIUS WITH NOWHERE TO MEASURE FROM ADMITS NOTHING, which is the
	// filtered-with-no-default rule stated as a fence.
	deck.setScope(func() airScope { return airScope{radiusMi: 25, set: true} })
	if deck.fence().InForce() {
		t.Error("a fence with no origin is not in force")
	}
}

// THE STATION'S SCOPE IS THE ONE THE POOL WAS BUILT FROM. Two answers to "how
// far does this station reach" is the shape that put the console's transmitter
// and the listener's default location in one field to begin with.
func TestTheRailAndThePoolShareOneServiceArea(t *testing.T) {
	lp := &livePipelines{idx: indexForTest(t)}
	lp.setStation(stationFrom(config.Config{Locations: []config.Location{bonsallCfg}}))
	lp.owner.set(tty.SurfaceBroadcaster)
	got := scopeFor(&lp.owner, func() airScope { return airScope{} }, lp.currentStation)
	if got.radiusMi != config.DefaultServiceRadiusMi {
		t.Errorf("the rail is fenced at the station's service radius; got %v", got.radiusMi)
	}
	if got.lat != bonsallCfg.Lat || got.lon != bonsallCfg.Lon {
		t.Errorf("and measured from its transmitter; got %v,%v", got.lat, got.lon)
	}
}

// A SWAP DOES NOT WAIT OUT THE CYCLE TIMER (D-73).
//
// THE TICKER CYCLES EVERY TWO MINUTES. Without the nudge the tape would go on
// showing the other surface's alerts for up to that long after a swap — which is
// the opposite of "it just works", in both directions.
func TestTakingTheAirAsksTheRailToReScopeAtOnce(t *testing.T) {
	lp := &livePipelines{ticker: &tickerDeck{rescope: make(chan struct{}, 1)}}
	if cmd := lp.takeTheAir(tty.SurfaceBroadcaster); cmd != nil {
		cmd()
	}
	if lp.owner.get() != tty.SurfaceBroadcaster {
		t.Error("the owner moved with the surface")
	}
	select {
	case <-lp.ticker.rescope:
	default:
		t.Error("the rail was not asked to re-scope")
	}

	// AND A FLURRY OF SWAPS IS ONE PENDING RE-SCOPE, not a queue of cycles
	// doing network work.
	for range 20 {
		if cmd := lp.takeTheAir(tty.SurfaceObserver); cmd != nil {
			cmd()
		}
	}
	if got := len(lp.ticker.rescope); got != 1 {
		t.Errorf("twenty swaps left %d pending cycles, want 1", got)
	}

	// AND IT NEVER BLOCKS THE PROGRAM'S GOROUTINE. A deck with no channel at
	// all — the older tests — takes the air without a nudge rather than panicking.
	_ = (&livePipelines{ticker: &tickerDeck{}}).takeTheAir(tty.SurfaceObserver)
	_ = (&livePipelines{}).takeTheAir(tty.SurfaceObserver)
}

// AND THE FEED'S FILTER FOLLOWS THE SCOPE TOO — which is the half the operator
// actually HEARS.
//
// IT SURVIVED THE FIRST PLANT RUN. `fence()` was asserted and `scopeToRadius`
// was not, and the two do different jobs: the fence decides ORDER, the filter
// decides WHAT REACHES THE RAIL AT ALL. A rail correctly ordered around alerts
// that should not be on it is the defect the HUM LEAD named — "when I switch to
// Broadcaster I don't hear alerts outside my service radius" — and only this
// function can keep that promise.
func TestTheFeedsFilterFollowsTheScope(t *testing.T) {
	when := time.Now().Add(-2 * time.Minute)
	near := globalfeed.Event{ID: "near", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
		Location: "Bonsall, CA", Severity: globalfeed.SevRed, At: when, HasPoint: true, Lat: 33.30, Lon: -117.23}
	far := globalfeed.Event{ID: "far", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
		Location: "Lancaster, CA", Severity: globalfeed.SevRed, At: when, HasPoint: true, Lat: 34.60, Lon: -118.20} // ~100 mi out
	deck := &tickerDeck{}

	// THE LISTENER'S WIDE FILTER KEEPS BOTH.
	deck.setScope(func() airScope { return airScope{lat: 33.2881, lon: -117.2256, radiusMi: 150, set: true} })
	if got := deck.scopeToRadius([]globalfeed.Event{near, far}); len(got) != 2 {
		t.Errorf("a 150-mile filter keeps both; got %d", len(got))
	}
	// THE STATION'S SERVICE AREA DROPS WHAT IS OUTSIDE IT.
	deck.setScope(func() airScope { return airScope{lat: 33.2881, lon: -117.2256, radiusMi: 25, set: true} })
	got := deck.scopeToRadius([]globalfeed.Event{near, far})
	if len(got) != 1 || got[0].ID != "near" {
		t.Errorf("a 25-mile service area keeps only what is inside it; got %v", got)
	}
	// AND GOING BACK RE-ADAPTS, which is the second half of the ruling.
	deck.setScope(func() airScope { return airScope{lat: 33.2881, lon: -117.2256, radiusMi: 150, set: true} })
	if got := deck.scopeToRadius([]globalfeed.Event{near, far}); len(got) != 2 {
		t.Errorf("the filter widens again with the listener's own; got %d", len(got))
	}
	// UNSCOPED IS ALL, and a radius with nowhere to measure from is nothing.
	deck.setScope(func() airScope { return airScope{} })
	if got := deck.scopeToRadius([]globalfeed.Event{near, far}); len(got) != 2 {
		t.Errorf("an unscoped rail is All; got %d", len(got))
	}
	deck.setScope(func() airScope { return airScope{radiusMi: 25, set: true} })
	if got := deck.scopeToRadius([]globalfeed.Event{near, far}); len(got) != 0 {
		t.Errorf("filtered with no origin shows nothing; got %d", len(got))
	}
}

// TestTheDecksFenceCarriesTheScopesTieSet.
//
// A ZONE-ONLY ALERT HAS NO POINT, so the tie set is the ONLY thing that can
// admit one — and `Fence.Admits` can only ask a set it was given. This drives
// `deck.fence()`, not `Admits`, because the platform tests already pin the rule
// and would stay green with the wiring cut: the seam that can be dropped is the
// handing over, so that is the seam a test has to take (P-1).
//
// SCOPED, AND FROM THE SCOPE'S OWN ORIGIN. The set the fence gets is the set the
// FEED's filter gets, asked with the same latitude, longitude and radius — that
// agreement is what stops the tape and the burst answering differently about one
// zone-only hazard (D-122).
func TestTheDecksFenceCarriesTheScopesTieSet(t *testing.T) {
	const id = "urn:oid:2.49.0.1.840.0.1.001.1"
	key, ok := severe.NormalizeID(id)
	if !ok {
		t.Fatalf("the fixture's id must normalise; %q did not", id)
	}
	near := snapshot.Location{Label: "Bonsall", TZ: "America/Los_Angeles", Lat: 33.2881, Lon: -117.2256,
		Alerts: []snapshot.Alert{{ID: id, Event: "Flood Warning", Severity: "severe",
			Sent: time.Now(), Expires: time.Now().Add(time.Hour)}}}

	sev := newSevereDeck(func(tea.Msg) {})
	sev.SetLocations(0, &snapshot.Snapshot{Locations: []snapshot.Location{near}})
	deck := &tickerDeck{severe: sev}

	deck.setScope(func() airScope { return airScope{lat: near.Lat, lon: near.Lon, radiusMi: 25, set: true} })
	if f := deck.fence(); !f.Tracked[key] {
		t.Errorf("the scope follows this alert and its fence does not carry the tie; got %v", f.Tracked)
	}

	// THE TRANSMITTER MOVES. Same alert, same radius, four hundred miles north —
	// nothing watched there carries it, so the fence follows nothing and the
	// zone-only alert has no way in.
	deck.setScope(func() airScope { return airScope{lat: 39.0, lon: -121.0, radiusMi: 25, set: true} })
	if f := deck.fence(); f.Tracked[key] {
		t.Error("a fence four hundred miles away inherited the tie set of a scope it is not")
	}
}

// TestTheArrivalsKeyAndTheTieSetsKeyAreTheSameKey.
//
// THE OVER-CORRECTION IS THE DANGEROUS FAILURE HERE, and it is silent in the
// wrong direction: if the arrival's key and the tie set's key are normalised
// differently, NO zone-only alert ever matches, every one of them is fenced out,
// and a real flood warning at the listener's own watched location is never read.
// The bypass this replaced admitted too much; getting the halves out of step
// would admit nothing, which is worse.
//
// So the two normalisations are checked against ONE REAL CAP ID, end to end:
// the id the watched location carries, through `alertKeysOf`, and the id the
// feed's event carries, through `arrivalsOf` — meeting at `Admits`.
func TestTheArrivalsKeyAndTheTieSetsKeyAreTheSameKey(t *testing.T) {
	// THE TWO FORMS ONE ALERT ACTUALLY TAKES, which is the whole reason
	// NormalizeID exists: the location path carries the bare OID and the ticker
	// path carries the same OID under the feature URL. A fixture using one form
	// on both sides normalises to itself and proves NOTHING — it passed against
	// a build with the normalisation cut out, which is how this was caught.
	const bare = "urn:oid:2.49.0.1.840.0.1.001.1"
	const feedID = "https://api.weather.gov/alerts/" + bare
	watched := snapshot.Location{Label: "Bonsall", TZ: "America/Los_Angeles", Lat: 33.2881, Lon: -117.2256,
		Alerts: []snapshot.Alert{{ID: bare, Event: "Flood Warning", Severity: "severe",
			Sent: time.Now(), Expires: time.Now().Add(time.Hour)}}}

	sev := newSevereDeck(func(tea.Msg) {})
	sev.SetLocations(0, &snapshot.Snapshot{Locations: []snapshot.Location{watched}})
	deck := &tickerDeck{severe: sev}
	deck.setScope(func() airScope { return airScope{lat: watched.Lat, lon: watched.Lon, radiusMi: 25, set: true} })

	// The SAME alert off the feed, zone-only: no point to measure by.
	zoneOnly := globalfeed.Event{ID: feedID, Type: "Flood Warning", Place: "San Diego County",
		Location: "Bonsall, CA", Source: "NWS", At: time.Now(), Severity: globalfeed.SevOrange}
	if zoneOnly.HasPoint {
		t.Fatal("the fixture must be zone-only or it proves nothing")
	}

	a := arrivalsOf([]globalfeed.Event{zoneOnly})[0]
	if a.TrackedAs == "" {
		t.Fatalf("a real CAP id must key; arrivalsOf produced %q from %q", a.TrackedAs, feedID)
	}
	if !deck.fence().Admits(a) {
		t.Errorf("the location the app watches carries this exact alert and the fence refused it — "+
			"the arrival keys as %q and the tie set holds %v", a.TrackedAs, deck.fence().Tracked)
	}
}
