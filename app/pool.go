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
	"context"
	"errors"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/report"
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
	// AND THE RAIL IS RE-TESTED AGAINST THE NEW FENCE (D-154). The station's
	// service area IS the rail's fence on the console, so moving the region
	// moves the fence — and everything already admitted was admitted under the
	// old one. Without this a narrowing from 100 miles to 25 left a 100-mile
	// hazard sitting on the rail, which is the sentence `Aired.Fence` claims to
	// have fixed and did not: its guard is written for the air, and the air does
	// not move when a setting changes.
	//
	// AFTER `setStation`, NOT BEFORE. The fence is ASKED of the scope in force,
	// and the scope in force is the one that was just installed a few lines up —
	// asking first would re-fence the rail to the region the station has left.
	//
	// SYNCHRONOUS, UNLIKE THE BED. This is an event handed to a pump, not
	// network work; the operator waiting on the save is not waiting on it.
	if mc := lp.masterControl(); mc != nil {
		mc.Refence()
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

// lookInPool answers what the operator typed into the request window.
//
// THREE ANSWERS, NOT TWO (ruling 2). `found` is whether the place exists at all;
// `inPool` is whether the station can broadcast about it. A location outside the
// service radius is NOT a lookup failure — it is a real place the window names
// and points at Observer for, which is only possible if the two are kept apart.
//
// THE POOL AND NOTHING ELSE, which is what makes it safe to run on a keystroke.
func (lp *livePipelines) lookInPool(query string) (snapshot.LocationRef, bool, bool) {
	if lp == nil {
		return snapshot.LocationRef{}, false, false
	}
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return snapshot.LocationRef{}, false, false
	}
	for _, ref := range lp.currentPool() { // bounded by the station's pool (P10-02)
		if matchesQuery(ref, q) {
			return ref, true, true
		}
	}
	return snapshot.LocationRef{}, false, false
}

// radiusLookupBudget bounds the resolve behind a location field. It is the
// Resolve hook's own budget, for the same reason: the operator is waiting.
const radiusLookupBudget = 5 * time.Second

// locateInRadius answers the ONE question both location fields ask: CAN THE
// STATION BROADCAST ABOUT THE PLACE THIS NAMES?
//
// THE POOL IS NOT THE TEST, AND THAT WAS THE DEFECT (D-130). The pool is capped
// at 25 (locations.PoolCap) and is a DELIBERATE subset of the fence — so a real
// place inside the service radius answered "not found" merely for being the
// 26th. HUM LEAD, UAT 2026-09-14: "location search should accept any value
// WITHIN the service radius, not just the 25 slot location pool. Example:
// 'Rainbow, CA' is a valid location within a 25 mi radius of Oceanside, but now
// it says that it's not a valid location."
//
// AND THE OFFLINE DATA CANNOT ANSWER IT ALONE. Measured 2026-09-14: the
// embedded index holds 34,106 cities and 41,490 zips, and Rainbow, CA is in
// NEITHER — it is an unincorporated community with no postal code of its own.
// The geocoder places it 14.7 miles from Oceanside, inside a 25-mile radius.
// So the network is not an optimisation to avoid here; for a whole class of
// small places it is the only thing that knows they exist.
//
// WHICH IS WHY THE CALLER DEBOUNCES (platform/debounce). `setup.go` already
// carries the rule this would otherwise break — never the network per keystroke
// (AI-8, ToS) — and a 300 ms pause is what reconciles "ask the geocoder" with
// it. This function is therefore SLOW BY CONTRACT and must never be called from
// a render path or a key handler.
//
// AND THE HYPER-LOCAL CASE IS THE POINT, NOT AN EDGE (HUM LEAD, ratified
// 2026-09-14): "the human operator should be able to lookup any valid location
// within their service radius, even if the initial sorting didn't include it
// into the default location pool. That value here again is exactly the
// hyper-local (Rainbow, CA) use case for human operator broadcasting a short
// range FRS/GMRS/CBRS station."
//
// THE POOL'S SORT IS WHY IT CANNOT BE THE TEST. `locations.Pool` fills from a
// POPULATION-FILTERED table, nearest first — so the 25 it keeps are structurally
// the BIGGEST places in range, and a short-range station's listeners are
// standing in the small ones. Rainbow is not an unlucky 26th; a pool ordered
// that way can never surface it, however large the cap.
//
// THREE ANSWERS, NOT TWO. `within` distinguishes a real place the station
// cannot reach from a name that means nothing — the first points the operator
// at Observer, the second asks them to try again.
func (lp *livePipelines) locateInRadius(r *locations.Resolver) func(string) (snapshot.LocationRef, bool, bool, bool) {
	return func(query string) (snapshot.LocationRef, bool, bool, bool) {
		if lp == nil {
			return snapshot.LocationRef{}, false, false, true
		}
		q := strings.TrimSpace(query)
		if q == "" {
			return snapshot.LocationRef{}, false, false, true // nothing typed: answered, and the answer is nothing
		}
		// THE POOL FIRST, because it is free and it is what the Director already
		// offers. A prefix match here is the common case and never leaves the
		// machine.
		if ref, _, ok := lp.lookInPool(q); ok {
			return ref, true, true, true
		}
		if r == nil {
			return snapshot.LocationRef{}, false, false, false // no resolver: the question cannot be put
		}
		ctx, done := context.WithTimeout(lp.lookupCtx(), radiusLookupBudget)
		defer done()
		ref, _, err := r.Resolve(ctx, q)
		if err != nil {
			// A TIMEOUT IS NOT AN ANSWER (D-151). `found=false` is what a genuine
			// no-match returns, and reporting a failed lookup the same way told
			// the operator a real place does not exist — then disabled the key
			// that would have retried it.
			return snapshot.LocationRef{}, false, false, !isLookupFailure(err)
		}
		if ref.Tag == "" {
			ref.Tag = deriveTag(ref.Label) // the same backfill Resolve does; a card names one
		}
		// AND THEN THE FENCE. A station with nowhere to transmit from reaches
		// nothing, which is the same reading `locations.Pool` and `lineup.Fence`
		// both take of an unset epicentre.
		s := lp.currentStation()
		if s.transmitter.Lat == 0 && s.transmitter.Lon == 0 {
			return ref, false, true, true
		}
		within := globalfeed.WithinMiles(s.transmitter.Lat, s.transmitter.Lon, ref.Lat, ref.Lon, s.radiusMi)
		return ref, within, true, true
	}
}

// lookupCtx is the pipeline's context when it has one, so a lookup in flight is
// cancelled at shutdown rather than holding the teardown for its budget.
func (lp *livePipelines) lookupCtx() context.Context {
	if lp.ctx != nil {
		return lp.ctx
	}
	return context.Background()
}

// matchesQuery is how a typed string names a pooled location: its label or its
// zip, case-folded, matched as a prefix so a half-typed city still resolves.
func matchesQuery(ref snapshot.LocationRef, q string) bool {
	return strings.HasPrefix(strings.ToLower(ref.Label), q) || strings.HasPrefix(ref.Zip, q)
}

// requestCard carries the operator's request to the Director.
//
// IT REMEMBERS THE REF BEFORE IT ASKS, and that ordering is the fix (D-140).
// The card the Director mints carries only a KEY; the Composer turns that key
// back into a location by looking it up. Until 2026-09-15 it looked only in the
// station's POOL — so a request for somewhere inside the service radius but
// outside the capped 25 minted fine, closed the window as though scheduled, and
// then failed to build. `Failed{Routed:true}` is treated as deliberate and
// self-healing, so nothing surfaced: the operator was shown an action nothing
// took, which is FR-3.3's named trap.
//
// IT REMEMBERS EVEN WITH NO DIRECTOR. The remembering is about what this
// process must be able to RESOLVE, not about what it managed to schedule.
func (lp *livePipelines) requestCard(ref snapshot.LocationRef, kinds report.Set, at int) {
	if lp == nil {
		return
	}
	lp.rememberRequested(ref)
	if lp.director == nil {
		return
	}
	lp.director.mc.RequestCard(ref, kinds, at)
}

// requestedCap bounds what the operator's requests may cost in memory.
//
// NOT THE MAIN TRACK'S DEPTH, AND THE FIRST VERSION SAID IT WAS (D-152). That
// justification — "a request that has fallen off the bottom of the running order
// can no longer be built" — described an eviction this code does not perform:
// it drops the OLDEST REMEMBERED, while `Insert` sheds the LAST VISIBLE. With
// the request window's default slot those are opposite ends, so sixteen requests
// at the bottom would evict the ref of the card sitting at the TOP, still
// unbuilt — and D-140's defect returns silently. Found by red team's second
// round.
//
// SO THE CAP IS SIZED TO MAKE EVICTION UNREACHABLE IN A SESSION rather than
// pretending to track the schedule. A `snapshot.LocationRef` is about a hundred
// bytes; sixty-four of them is single-figure kilobytes, and an operator would
// have to request sixty-four DISTINCT locations — repeats are deduped by
// identity — before the oldest is forgotten. The cost of being wrong the other
// way is a card the Composer cannot resolve, which is silent.
//
// TYING IT TO THE RUNNING ORDER PROPERLY would mean dropping a ref when its card
// leaves the schedule, which needs the schedule here. That is a wiring decision,
// not a constant, and it is recorded rather than guessed at.
const requestedCap = 64

// rememberRequested records a location the operator asked for, so the Composer
// can resolve it later.
//
// DEDUPED BY IDENTITY, NOT BY LABEL. `snapshot.Key` is what the schedule uses
// to refuse a duplicate card, so it is what this must agree with; two zip
// centroids of one town are two locations to everything downstream.
func (lp *livePipelines) rememberRequested(ref snapshot.LocationRef) {
	if lp == nil || ref.Label == "" {
		return
	}
	lp.mu.Lock()
	defer lp.mu.Unlock()
	key := snapshot.Key(ref)
	for _, have := range lp.requestedRefs { // bounded by requestedCap (P10-02)
		if snapshot.Key(have) == key {
			return
		}
	}
	lp.requestedRefs = append(lp.requestedRefs, ref)
	if n := len(lp.requestedRefs) - requestedCap; n > 0 {
		lp.requestedRefs = lp.requestedRefs[n:]
	}
}

// resolvable is what the COMPOSER may turn a card's key back into: the station's
// pool, plus whatever the operator has asked for.
//
// TWO QUESTIONS, NOT ONE (D-140). `schedule.go` said pool was "what its Producer
// may offer AND what its Composer resolves against" — one list serving two
// questions, which held only while the operator could request nothing else.
// D-130 made that false. The Producer is still bounded by the pool: widening
// what may be PROPOSED would let the station offer a location it never chose.
func (lp *livePipelines) resolvable() []snapshot.LocationRef {
	if lp == nil {
		return nil
	}
	pool := lp.currentPool()
	lp.mu.Lock()
	defer lp.mu.Unlock()
	out := make([]snapshot.LocationRef, 0, len(pool)+len(lp.requestedRefs))
	out = append(out, pool...)
	return append(out, lp.requestedRefs...)
}

// isLookupFailure separates "the question could not be put" from "there is no
// such place" (D-151).
//
// A CANCELLED OR TIMED-OUT CONTEXT IS THE FORMER, and so is a resolver that
// failed to build. Anything else — the offline index refusing a name, the
// geocoder answering with nothing — is a real no-match, which the window is
// right to draw as one.
func isLookupFailure(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
