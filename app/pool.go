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
