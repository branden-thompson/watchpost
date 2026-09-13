package app

// pool.go — the station's own location pool, and what re-derives it (D-72).
//
// THE OBSERVER'S WATCHLIST IS NOT THE BROADCASTER'S LINE-UP. The HUM LEAD ruled
// the split on 2026-09-10; until then the Producer offered `lp.currentWatch`, so
// the station could only ever read the places the LISTENER happened to be
// watching — three of them in his UAT, against a console that draws ten slots
// (F-81, F-82).
//
// DERIVED, NOT PERSISTED. The pool is a pure function of the transmitter, the
// service radius and the embedded tables; storing it would create a second
// answer that could drift from the settings that produced it, and re-deriving
// costs about forty milliseconds on the two occasions it happens — startup, and
// a change to either setting.

import (
	tea "charm.land/bubbletea/v2"
	"context"

	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// stationArea is where the broadcast comes from and how far it reaches.
//
// NOT NAMED `station`, WHICH THIS PACKAGE'S TESTS ALREADY USE for a wired-up
// fixture. The word is right and it was taken; the service area is what this
// actually describes.
type stationArea struct {
	transmitter snapshot.LocationRef
	radiusMi    float64

	// followsDefault is whether the transmitter came from the LISTENER's default
	// location rather than from a setting of the station's own. It has to be
	// remembered, not re-derived: once the fallback has been taken, nothing
	// downstream can tell a borrowed epicentre from a chosen one — and only a
	// borrowed one should move when the listener changes their watchlist.
	followsDefault bool
}

// stationFrom reads the station's settings out of the configuration, with the
// transmitter falling back to the listener's default location (config.Station).
func stationFrom(cfg config.Config) stationArea {
	loc, ok := cfg.Station()
	if !ok {
		return stationArea{radiusMi: cfg.Broadcaster.ServiceRadius()}
	}
	refs := refsFromConfig([]config.Location{loc})
	if len(refs) == 0 {
		return stationArea{radiusMi: cfg.Broadcaster.ServiceRadius()}
	}
	own := cfg.Broadcaster.Transmitter.Lat != 0 || cfg.Broadcaster.Transmitter.Lon != 0
	return stationArea{transmitter: refs[0], radiusMi: cfg.Broadcaster.ServiceRadius(), followsDefault: !own}
}

// setStation records where the station transmits from and re-derives its pool.
//
// THE POOL IS REPLACED WHOLE. A pool edited in place would let the Producer see
// a half-built list on the update that follows a settings change — and the one
// thing the Producer must never do is offer a location the station cannot reach.
// IT DOES NOT PUBLISH, AND THAT IS LOAD-BEARING. It is called from
// `startPipelines`, BEFORE `p.Run()` — and `tea.Program.Send` on a program whose
// loop has not started blocks for ever. The first version of this deadlocked the
// entire app at launch, and `TestRunWithoutArgsStartsTheDashboard` sat on it for
// ten minutes under `-race` rather than failing with a reason. The console opens
// with the area on `tty.Config` instead; only CHANGES are sent.
func (lp *livePipelines) setStation(s stationArea) {
	pool := lp.poolFor(s)
	lp.mu.Lock()
	lp.station, lp.poolRefs = s, pool
	lp.mu.Unlock()
}

// moveCard and dropCard are the console's two card controls, routed to the one
// thing that may change a schedule (D-118).
//
// THROUGH MASTERCONTROL, LIKE EVERY OTHER DECLARATION. It is the band's and the
// bed's one owner already (T2.3), and a second path into the Director would be a
// second answer to who may change the running order.
func (lp *livePipelines) moveCard(id string, to int) {
	if lp == nil || lp.director == nil {
		return
	}
	lp.director.mc.MoveCard(id, to)
}

func (lp *livePipelines) dropCard(id string) {
	if lp == nil || lp.director == nil {
		return
	}
	lp.director.mc.DropCard(id)
}

// rebed re-resolves the station's relays, through a seam a test can hold.
//
// A SEAM BECAUSE THE REAL ONE IS NETWORK WORK AND A GOROUTINE. A mutant that
// stopped the re-resolve on a move SURVIVED — nothing could observe whether it
// happened — and "a moved station goes on offering the relays of the region it
// left" is exactly the rule that must not be unpinned. `rp.newFor` is the same
// shape for the same reason.
func (lp *livePipelines) rebed(ctx context.Context) {
	lp.mu.Lock()
	f := lp.bedRefresh
	lp.mu.Unlock()
	if f == nil {
		f = lp.refreshBedRelays
	}
	f(ctx)
}

// transmitterOf is the station's OWN transmitter, or nil when it is borrowing
// the listener's default location (D-115).
//
// NIL IS THE POINT. "Not set" and "set to the same place the listener happens to
// have chosen" are different states — the first MOVES with the watchlist and the
// second does not — and the window says which one the operator is looking at.
func transmitterOf(cfg config.Config) *snapshot.LocationRef {
	tx := cfg.Broadcaster.Transmitter
	if tx.Lat == 0 && tx.Lon == 0 {
		return nil
	}
	refs := refsFromConfig([]config.Location{tx})
	if len(refs) == 0 {
		return nil
	}
	return &refs[0]
}

// setTransmitter records where the station transmits from, re-derives its pool
// and tells the console (D-115).
//
// THE THREE ARE ONE ACT. A transmitter written without re-deriving would leave
// the Producer offering the old region for the life of the process — which is
// exactly the defect `reStationOnCommit` exists for on the borrowed path — and a
// pool re-derived without publishing would leave the console naming a region the
// station has left.
//
// IT STOPS BORROWING. Writing a transmitter of the station's own is what ends
// the D-72 fallback, so `followsDefault` goes false and the listener's watchlist
// no longer moves the station.
func (lp *livePipelines) setTransmitter(ref snapshot.LocationRef) {
	if err := config.Mutate(func(cfg *config.Config) error {
		// THROUGH THE ONE CONVERTER, so the station's transmitter is stored in
		// exactly the shape every other saved location is — the derived tag
		// included, which is what the config's own reader expects.
		if out := configLocations([]snapshot.LocationRef{ref}); len(out) == 1 {
			cfg.Broadcaster.Transmitter = out[0]
		}
		return nil
	}); err != nil {
		return // the console keeps showing what is in force; nothing was written
	}
	lp.restationTo(stationArea{transmitter: ref, radiusMi: lp.currentStation().radiusMi})
}

// setServiceRadius records how far the station serves, and re-derives with it.
//
// THE RADIUS IS HALF THE POOL. `locations.Pool` is a function of the transmitter
// AND the radius, so a radius written without a re-derivation is a console whose
// candidate list disagrees with its own stated service area.
func (lp *livePipelines) setServiceRadius(mi int) {
	if err := config.Mutate(func(cfg *config.Config) error {
		cfg.Broadcaster.ServiceRadiusMi = float64(mi)
		return nil
	}); err != nil {
		return
	}
	now := lp.currentStation()
	lp.restationTo(stationArea{transmitter: now.transmitter, radiusMi: float64(mi),
		followsDefault: now.followsDefault})
}

// restationTo installs a station area, re-derives its pool and publishes both.
//
// EXTRACTED AT THE SECOND CALLER, which is the standing rule: the transmitter
// and the radius each change one half of the same derivation, and two copies of
// "set, derive, publish" would be two places for one of the three to be
// forgotten.
func (lp *livePipelines) restationTo(s stationArea) {
	lp.setStation(s)
	lp.mu.Lock()
	p, pool, ctx := lp.p, lp.poolRefs, lp.ctx
	lp.mu.Unlock()
	if p != nil {
		publishArea(p.Send, s, pool)
	}
	// AND THE BED RE-RESOLVES WITH IT (D-117). Which relays actually stream is a
	// fact about the station's REGION, so moving the region asks the question
	// again — otherwise the selector goes on offering the relays of the place the
	// station has left.
	//
	// ON ITS OWN GOROUTINE, because this is network work and the caller is a
	// settings save the operator is waiting on.
	if ctx != nil {
		go lp.rebed(ctx)
	}
}

// publishArea tells the console where the station transmits from.
//
// PUBLISHED RATHER THAN READ. The console draws both facts and holds neither —
// the same rule the line-up and the power already follow — so there is one
// answer to what the station's service area is, and it is the one the Producer
// just used.
// IT IS FOR CHANGES ONLY. At launch the console reads the area off `tty.Config`
// — see setStation for why sending it then cannot work.
// IT TAKES A SEND SEAM, NOT A PROGRAM (D-93). `*tea.Program` cannot be driven by
// a test without running one, so the only way to check what this publishes was to
// write a second copy of it in the test — which measures the copy. `func(tea.Msg)`
// is what the executors' own `publish` seam already is.
func publishArea(send func(tea.Msg), s stationArea, pool []snapshot.LocationRef) {
	if send == nil {
		return // no program (a test), or none attached yet: nothing to tell
	}
	// THE POOL GOES WITH IT (D-93). It is derived from the very two fields above,
	// so sending them apart would let the console hold a pool belonging to an
	// area it has stopped showing.
	send(tty.StationAreaMsg{Transmitter: s.transmitter, RadiusMi: s.radiusMi, Pool: pool})
}

// poolFor derives a station's pool and takes NO LOCK, so a caller already
// holding `lp.mu` can compute it before it needs one. The scan reads only the
// embedded tables and its argument, which is what makes that safe to state.
func (lp *livePipelines) poolFor(s stationArea) []snapshot.LocationRef {
	return locations.Pool(lp.idx, s.transmitter, s.radiusMi, locations.PoolCap)
}

// currentPool is what the Producer offers the Director — the station's pool,
// never the listener's watchlist.
func (lp *livePipelines) currentPool() []snapshot.LocationRef {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.poolRefs
}

// producer is what the station's Producer may offer, and it is THE ONE PLACE
// that choice is made (D-72).
//
// A FUNCTION RATHER THAN A CALL SITE, deliberately. `startSchedule` takes three
// seams that all read the same list — what may be proposed, what a ref resolves
// against, and what the bed cuts to — and the defect this release keeps
// producing is a wiring nothing drives. A call site cannot be asserted; this
// can, and a test does.
func (lp *livePipelines) producer() func() []snapshot.LocationRef { return lp.currentPool }

// currentStation is where the console says the station transmits from.
func (lp *livePipelines) currentStation() stationArea {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.station
}

// reStationOnCommit re-derives the pool when the watchlist's FIRST entry moves
// and no transmitter of its own has been set.
//
// IT EXISTS BECAUSE THE FALLBACK IS REAL. `config.Station` reads the listener's
// default location when the station has no transmitter, so changing that
// default silently moves the station's epicentre — and a pool derived once at
// startup would go on offering the old region for the life of the process.
// reStation is the station and pool a commit should leave behind, computed
// WITHOUT the lock so `commit` can assign them under the one it already holds.
//
// IT EXISTS BECAUSE THE FALLBACK IS REAL. A station with no transmitter of its
// own borrows the listener's default location, so changing that default moves
// the station's epicentre — and a pool derived once at startup would go on
// offering the old region for the life of the process.
//
// A STATION WITH ITS OWN EPICENTRE DOES NOT MOVE, which is the whole point of
// the split: the operator's watchlist is not the station's service area.
func (lp *livePipelines) reStation(watch []snapshot.LocationRef) (stationArea, []snapshot.LocationRef, bool) {
	now := lp.currentStation()
	if !now.followsDefault || len(watch) == 0 {
		return stationArea{}, nil, false
	}
	if snapshot.Key(watch[0]) == snapshot.Key(now.transmitter) {
		return stationArea{}, nil, false // the default did not move
	}
	next := stationArea{transmitter: watch[0], radiusMi: now.radiusMi, followsDefault: true}
	return next, lp.poolFor(next), true
}

// withPool adds the station's pool to a location set, skipping what is already
// there (D-99).
//
// THE POOL JOINS THE RECENT PIPELINE RATHER THAN GETTING ONE OF ITS OWN. That
// pipeline already fetches a bounded set at the slow cadence, publishes into one
// assembler, and coalesces — building a third of it for twenty-five locations
// would be a second copy of all of that.
//
// THE CADENCE IS THAT PIPELINE'S, AND IT IS SLOWER THAN THE RULING ASKED FOR IN
// THE HALF THAT MATTERS. The HUM LEAD approved fifteen minutes (2026-09-12);
// `recentTiers` gives observations every TEN — fresher — and the forecast every
// HOUR, which is the source's own refresh interval and what R-8.2 bounds a cadence
// by. Fifteen minutes on a forecast NWS republishes hourly would re-read cache
// four times for the same bytes. Recorded rather than silently substituted.
//
// DEDUPED BY KEY, because a watched location inside the service area is one
// location, and fetching it twice would double its cost for no new data.
func withPool(recent, pool []snapshot.LocationRef) []snapshot.LocationRef {
	seen := make(map[snapshot.LocationKey]bool, len(recent)+len(pool))
	out := make([]snapshot.LocationRef, 0, len(recent)+len(pool))
	for _, r := range append(append([]snapshot.LocationRef(nil), recent...), pool...) { // bounded (P10-02)
		k := snapshot.Key(r)
		if r.Label == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, r)
	}
	return out
}
