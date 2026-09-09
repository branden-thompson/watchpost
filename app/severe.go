package app

// severe.go — the severe-events index deck (0.13.0, SAM-D-26 AX-1 = A): the
// join between the ticker's pre-radius feed events and the tracked locations'
// alerts. The ticker cycle hands over the feed half (SetFeed); the priority and
// recent publishers poke it after every publish (Trigger); it recomputes the
// index through domains/severe and sends a SevereMsg to the dashboard only when
// the row set (or a row's content), a source's health or the fetch minute
// changed — at most once per ticker cycle plus on any real change, so the
// 20-second alerts tier never churns the modal memo by itself. Pure computation off the tea update
// loop. The deck never fetches — adding a client field here needs a new NFR
// (NFR-1 holds by construction).

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	zones "github.com/branden-thompson/watchpost/platform/tz"
)

// The [S] gauge's cap and the domain's must agree too (a negative array
// length does not compile).
var (
	_ [severe.MaxRows - tty.SevereMaxRows]struct{}
	_ [tty.SevereMaxRows - severe.MaxRows]struct{}
)

// SourceHealth is one feed's last outcome, for the window's "Updated" stamp
// and its source line: a dead source must show, and the stamp is the newest
// successful fetch — dataAsOf — never the publish time.
type SourceHealth struct {
	Name      string
	OK        bool
	FetchedAt time.Time // the last successful fetch; zero = never
}

type severeDeck struct {
	// publishMu serialises the WHOLE publish (read inputs → compute → compare →
	// send): the ticker goroutine and two publisher timers call it concurrently,
	// and a slow publish must never land after a newer one.
	publishMu sync.Mutex
	mu        sync.Mutex // guards feed/sources/locs between the setters and publish
	feed      []globalfeed.Event
	sources   []SourceHealth
	locs      [2]*snapshot.Snapshot // the publishers' CURRENT snapshots, handed in by their hooks (0 priority, 1 recent); nil = no publish yet
	send      func(tea.Msg)
	gen       uint64
	lastKey   [32]byte
	rows      map[string]tty.SevereRow // the last publish, by key — the event reader's lookup
	onFeed    func([]globalfeed.Event) // test hook: observe the feed half
	onPublish func()                   // test hook
	now       func() time.Time

	// laneRows are the Advisories and Statements rows, for the ticker's lanes.
	laneRows []severe.Row

	// radius is the ALERTS - EVENTS preference in miles, live (0 = All).
	//
	// The window used to list the PRE-radius set while the tape listed the
	// filtered one, which meant the two disagreed and the STATEMENTS and
	// ADVISORIES tabs — whose rows come only from the tracked locations, never
	// from the national feed — were bounded by nothing but which locations
	// happened to be on the watchlist. "Filtered by what data is saved", as the
	// HUM LEAD put it. One preference governs both now.
	radius *atomic.Int64
}

func newSevereDeck(send func(tea.Msg)) *severeDeck {
	return &severeDeck{send: send, now: time.Now}
}

// scope applies the ALERTS - EVENTS preference to both halves of the index.
//
// "All locations" (0) keeps everything. "Within N mi of Default Location" keeps
// the feed events within N miles of the default and the tracked locations within
// N miles of it — the DEFAULT location being the one the setting names, and the
// first of the priority snapshot being it.
//
// Filtered with no default set shows NOTHING rather than silently falling back
// to the unscoped set, which is the rule the tape already follows: a window that
// says it is scoped must not quietly show everything (red-team 0.12.0 P4 F7).
// The default location is a PARAMETER, never re-read from s.locs: publish
// resolves it inside the same critical section it takes the snapshots in, so
// the centre and the rows being scoped always come from one consistent view,
// and this runs on the alert path with no lock of its own.
func (s *severeDeck) scope(def *snapshot.Location, feed []globalfeed.Event, locs []snapshot.Location) ([]globalfeed.Event, []snapshot.Location) {
	r := 0
	if s.radius != nil {
		r = int(s.radius.Load())
	}
	if r <= 0 {
		return feed, locs
	}
	if def == nil {
		return nil, nil
	}
	kept := locs[:0:0]
	for _, l := range locs {
		if globalfeed.WithinMiles(def.Lat, def.Lon, l.Lat, l.Lon, float64(r)) {
			kept = append(kept, l)
		}
	}
	return scopeEvents(feed, def.Lat, def.Lon, float64(r), alertKeysOf(kept)), kept
}

// scopeEvents applies the ALERTS-EVENTS radius to feed events.
//
// ONE OWNER for every surface. The severe window, the tape, the breaking
// takeover and the spoken alert all ask the same question about the same
// hazard, and two answers means the app shows a warning in one place while
// staying silent in another.
//
// UNKNOWN DISTANCE IS NOT OUT OF RANGE. Many NWS products are zone-only and
// carry no polygon at all — watches, most flood warnings, heat advisories — so
// there is nothing to measure a radius against. Such an alert is kept when the
// app is separately tracking the SAME alert on a location in scope: that
// location's own zone query fetched it, which is the app already saying this
// hazard applies here.
//
// The tie is the alert's ID, normalised the way severe.Guard normalises it.
// It cannot be the event's Location, because globalfeed.Locate deliberately
// skips the watchlist tie when there is no point — a zone-only alert's Location
// is the feed's area description, never a tracked location's label.
func scopeEvents(evs []globalfeed.Event, lat, lon, radiusMi float64, tracked map[string]bool) []globalfeed.Event {
	out := evs[:0:0]
	for _, e := range evs {
		switch {
		case e.HasPoint:
			// INSIDE THE RADIUS, OR CARRIED IN BY ITS OWN SIGNIFICANCE (BD-6).
			// The second clause is the whole reason the exception reaches a
			// listener at all: this filter runs BEFORE anything becomes an
			// arrival, so a bare radius test here fenced the M7.5 out of the
			// app entirely and lineup.Fence never got to admit it (C-2/C-3).
			// Measured with the same function as the radius, so the exception
			// cannot drift from the rule it excepts.
			if globalfeed.WithinMiles(lat, lon, e.Lat, e.Lon, radiusMi) ||
				globalfeed.WithinMiles(lat, lon, e.Lat, e.Lon, reachMiOf(e)) {
				out = append(out, e)
			}
		default:
			// No id, no tie: NormalizeID rejects anything it does not recognise
			// as a CAP alert — an empty id included — so an event that cannot be
			// identified is never shown to be one the app is already tracking.
			//
			// The `ok` here is DEFENCE IN DEPTH and cannot be observed to fail:
			// alertKeysOf already refuses to key an id NormalizeID rejects, so
			// no unusable key is ever in `tracked` to match. It is kept because
			// this is the hazard path and one guard should not be the only
			// thing between a listener and a warning a thousand miles away —
			// but deleting it changes no behaviour, and no test can catch that,
			// which is stated here rather than left to look like coverage.
			if key, ok := severe.NormalizeID(e.ID); ok && tracked[key] {
				out = append(out, e)
			}
		}
	}
	return out
}

// reachMiOf is how far an event's own significance carries it PAST the
// listener's radius (BD-6, ratified 2026-09-02), in miles. Zero is the ordinary
// case: almost nothing buys an exception.
//
// ONE OWNER (D-1), and it has two callers for a reason: the producer's radius
// filter must not drop what the Director's fence would admit, and the arrival
// the producer hands over must carry the same number the filter used. Splitting
// them is exactly how the ruling came to be implemented, pinned, mutant-guarded
// and connected to nothing — arrivalsOf set nine fields and not this one, so
// Fence.Admits compared every distance against zero (red team 2026-09-05, C-3).
//
// A QUAKE ONLY, matching lineup.Fence.Admits' own Disasters test: such a
// disaster has proximal effects, so it carries its own reach; a warning a
// thousand miles away is still a warning a thousand miles away, however severe.
// The SCALE is not restated here — lineup.QuakeReachMi owns it, including the
// M9.5 clamp that stops a malformed feed row buying unbounded reach.
func reachMiOf(e globalfeed.Event) float64 {
	if e.Class != globalfeed.ClassQuake || e.Quake == nil || e.Quake.Mag == nil {
		return 0
	}
	return lineup.QuakeReachMi(*e.Quake.Mag)
}

// alertKeysOf is the normalised ids of the alerts the given locations carry —
// what a point-less feed event can be tied to.
//
// NormalizeID's own acceptance is the whole guard: it returns ok only for a
// string matching the CAP alert grammar, which is never empty. An id it rejects
// is of unknown provenance and must key nothing — an empty key would tie every
// point-less event that also lacks an id, which is one match on nothing at all.
func alertKeysOf(locs []snapshot.Location) map[string]bool {
	keys := map[string]bool{}
	for i := range locs {
		for _, a := range locs[i].Alerts {
			if key, ok := severe.NormalizeID(a.ID); ok {
				keys[key] = true
			}
		}
	}
	return keys
}

// defaultLocation is the watchlist's first entry — the "Default Location" the
// ALERTS - EVENTS setting names.
func defaultLocation(sn *snapshot.Snapshot) *snapshot.Location {
	if sn == nil || len(sn.Locations) == 0 {
		return nil
	}
	return &sn.Locations[0]
}

// SetLocations installs one publisher's snapshot (slot 0 priority, 1 recent)
// and republishes. The publisher hands the snapshot it is about to send, so
// the index never lags the tables by a publish (R3-A-02: reading the
// publisher's last snapshot back from the hook returned the PREVIOUS one).
// The deck keeps its OWN copy of the locations and their alerts: the same
// pointer goes to the tty, which sorts each location's alerts in place on
// its loop while the deck re-reads them on the ticker's (REVIEW R5-B-01, a
// race on the main data path).
func (s *severeDeck) SetLocations(slot int, snap *snapshot.Snapshot) {
	if slot < 0 || slot >= len(s.locs) {
		return
	}
	var own *snapshot.Snapshot
	if snap != nil {
		locs := make([]snapshot.Location, len(snap.Locations))
		for i, l := range snap.Locations {
			locs[i] = l
			locs[i].Alerts = append([]snapshot.Alert(nil), l.Alerts...)
		}
		own = &snapshot.Snapshot{Locations: locs}
	}
	s.mu.Lock()
	s.locs[slot] = own
	s.mu.Unlock()
	s.publish()
}

// SetFeed replaces the feed half with its own copy plus the sources' health,
// and republishes.
func (s *severeDeck) SetFeed(evs []globalfeed.Event, sources []SourceHealth) {
	cp := make([]globalfeed.Event, len(evs))
	copy(cp, evs)
	sh := make([]SourceHealth, len(sources))
	copy(sh, sources)
	s.mu.Lock()
	s.feed, s.sources = cp, sh
	s.mu.Unlock()
	if s.onFeed != nil {
		s.onFeed(cp)
	}
	s.publish()
}

// publish recomputes the index and sends it when the row set (or a source's
// health) changed. Serialised end to end.
func (s *severeDeck) publish() {
	s.publishMu.Lock()
	defer s.publishMu.Unlock()
	if s.onPublish != nil {
		s.onPublish()
	}
	rows, sources := s.currentRows()
	severe.Sort(rows)
	s.setLaneRows(rows)            // after the sort: the lanes keep the worst, not the first built
	key := indexKey(rows, sources) // the pre-cap set: a change past the 500th row still changes the totals
	if key == s.lastKey {
		return // nothing changed: no message, no memo churn (the 20-second alerts tier lands here)
	}
	s.lastKey = key
	s.gen++
	kept, _ := severe.Cap(rows, severe.MaxRows)
	byTab := severe.ByTab(rows) // totals count the PRE-cap rows per tab (honest "showing N of M")
	msg := tty.SevereMsg{Gen: s.gen, Rows: make([]tty.SevereRow, 0, len(kept))}
	for _, src := range sources { // "Updated" = the newest SUCCESSFUL fetch (dataAsOf, FR-9), never the publish time
		if src.OK && src.FetchedAt.After(msg.Updated) {
			msg.Updated = src.FetchedAt
		}
		msg.Sources = append(msg.Sources, tty.SevereSource{Name: src.Name, OK: src.OK})
	}
	for i := range byTab {
		msg.Totals[i] = len(byTab[i])
	}
	byKey := make(map[string]tty.SevereRow, len(kept))
	for _, r := range kept { // records are composed only here — on a change — never per trigger
		row := toSevereRow(r)
		msg.Rows = append(msg.Rows, row)
		byKey[row.Key] = row
	}
	s.mu.Lock()
	s.rows = byKey
	s.mu.Unlock()
	s.send(msg)
}

// Row is the last published row with this key (the reader's lookup).
func (s *severeDeck) Row(key string) (tty.SevereRow, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rows[key]
	return r, ok
}

// currentRows joins this instant's feed and tracked locations into the row set,
// and returns the source health that came with them.
//
// Everything it reads is taken in ONE critical section, so the default location
// the scope is centred on, the events being scoped and the locations they are
// tied to all describe the same moment. Reading any of them again later is how
// a centre from one snapshot ends up scoping another one's rows.
func (s *severeDeck) currentRows() ([]severe.Row, []SourceHealth) {
	s.mu.Lock()
	feed, sources, snaps := s.feed, s.sources, s.locs
	s.mu.Unlock()

	var tracked []snapshot.Location
	for _, sn := range snaps {
		if sn != nil {
			tracked = append(tracked, sn.Locations...)
		}
	}
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	return s.union(defaultLocation(snaps[0]), feed, tracked, now), sources
}

// union scopes both halves by the ALERTS - EVENTS preference and joins them
// into the row set everything downstream reads. Ordering and distribution are
// publish's, so this does one thing.
func (s *severeDeck) union(def *snapshot.Location, feed []globalfeed.Event, locs []snapshot.Location, now time.Time) []severe.Row {
	feed, locs = s.scope(def, feed, locs)
	return severe.Union(feed, locs, now)
}

// setLaneRows keeps the rows of the two tabs the NATIONAL FEED cannot supply —
// Advisories and Special Weather Statements come only from the tracked
// locations (SAM-D-10) — so the ticker can give them lanes.
//
// The ticker reads the feed; these rows are not in it. Without this the window
// promised two categories the ticker could never mention, which is the gap the
// HUM LEAD found at UAT (2026-08-30).
func (s *severeDeck) setLaneRows(rows []severe.Row) {
	keep := laneRowsPerTab(rows)
	s.mu.Lock()
	s.laneRows = keep
	s.mu.Unlock()
}

// laneRowsPerTab keeps each location-only lane's own share.
//
// PER LANE, for the same reason the national stack caps per lane: the marquee
// rotates, so a lane with nothing in it drops out of the rotation. Advisories
// and Statements sharing one budget lets a day of heat advisories empty the
// Statements lane while its events are live.
//
// The cut is by severity — an over-full lane gives up its mildest rows, not its
// oldest — and what survives is then ordered most-recent-first, the order every
// other lane reads in. Cutting and reading are different questions, and the
// marquee must not answer them differently in one lane than in the next.
func laneRowsPerTab(rows []severe.Row) []severe.Row {
	laneTabs := []severe.Tab{severe.TabEmergency, severe.TabAdvisories, severe.TabStatements}
	inLane := map[severe.Tab]bool{}
	for _, t := range laneTabs {
		inLane[t] = true
	}
	byTab := map[severe.Tab][]severe.Row{}
	for _, r := range rows {
		if inLane[r.Tab] {
			byTab[r.Tab] = append(byTab[r.Tab], r)
		}
	}
	out := make([]severe.Row, 0, len(laneTabs)*maxLaneRows)
	for _, tab := range laneTabs {
		held := byTab[tab]
		sort.SliceStable(held, func(i, j int) bool {
			if held[i].Severity != held[j].Severity {
				return held[i].Severity > held[j].Severity
			}
			return held[i].At.After(held[j].At)
		})
		if len(held) > maxLaneRows {
			held = held[:maxLaneRows]
		}
		sort.SliceStable(held, func(i, j int) bool { return held[i].At.After(held[j].At) })
		out = append(out, held...)
	}
	return out
}

// maxLaneRows caps each location-only lane, mirroring globalfeed.MaxPerLane on
// the national stack: one bound per lane, so the marquee's whole input is
// bounded however busy the weather is and no lane can empty another.
const maxLaneRows = globalfeed.MaxPerLane

// AlertKeysWithin is the tie set for a scoped feed: the normalised ids of the
// alerts carried by tracked locations INSIDE the radius.
//
// radiusMi must be positive. Callers answer "All" before asking — a scoped
// surface is the only thing that needs a tie set, and treating a non-positive
// radius as "every tracked alert" here would rebuild, quietly, the unscoped set
// this method exists to replace.
//
// Scoped, because the tape and the window must answer one question the same
// way. An unscoped set defeats the listener's own setting from the other
// direction: the RECENT table is seeded with the fifty largest US cities and
// each of them fetches its own alerts, so a zone-only warning a thousand miles
// away would be tied, take the marquee over and be read aloud — while the
// window, which scopes its half, silently dropped it.
func (s *severeDeck) AlertKeysWithin(lat, lon, radiusMi float64) map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	var near []snapshot.Location
	for _, sn := range s.locs {
		if sn == nil {
			continue
		}
		for _, l := range sn.Locations {
			if globalfeed.WithinMiles(lat, lon, l.Lat, l.Lon, radiusMi) {
				near = append(near, l)
			}
		}
	}
	return alertKeysOf(near)
}

// LaneRows is a copy of those rows, for the ticker's cycle.
func (s *severeDeck) LaneRows() []severe.Row {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]severe.Row(nil), s.laneRows...)
}

// indexKey fingerprints the row set — each row's key, times, location, path
// and the cheap scalars a same-id revision changes (a USGS magnitude update,
// a new NHC advisory, a row flipping from untied to tied: R3-A-03) — plus the
// sources' health + fetch minute, so an unchanged index publishes nothing
// while a fresh successful fetch still refreshes "Updated" (FR-9). The times
// are hashed at second precision (RFC3339 drops fractions here), so a source
// cannot make the key churn by itself.
func indexKey(rows []severe.Row, sources []SourceHealth) [32]byte {
	h := sha256.New()
	stamp := func(t time.Time) { h.Write([]byte(t.UTC().Format(time.RFC3339))) }
	for _, r := range rows {
		h.Write([]byte(r.Key))
		stamp(r.Sent)
		stamp(r.At)
		stamp(r.Until)
		h.Write([]byte(r.Location))
		h.Write([]byte(r.Source))
		h.Write([]byte{byte(r.Severity)})
		if r.Tied != nil {
			h.Write([]byte(r.Tied.Label + "\x00" + r.Tied.TZ))
		}
		if q := r.Detail.Quake; q != nil {
			stamp(q.UpdatedAt)
			if q.Mag != nil {
				_, _ = fmt.Fprintf(h, "%.2f", *q.Mag) // a hash.Hash never returns an error (P10-07: explicit)
			}
		}
		if tr := r.Detail.Tropical; tr != nil {
			_, _ = fmt.Fprintf(h, "%s|%d", tr.AdvisoryNum, tr.WindKt) // likewise
		}
		h.Write([]byte{0})
	}
	for _, src := range sources {
		h.Write([]byte(src.Name))
		if src.OK {
			h.Write([]byte{1})
			h.Write([]byte(src.FetchedAt.UTC().Truncate(time.Minute).Format(time.RFC3339)))
		} else {
			h.Write([]byte{0})
		}
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// rowClock is the zone rule (FR-9): a row tied to a tracked location reads in
// that location's clock (the F17 precedent), any other row in the viewer's.
func rowClock(r severe.Row) *time.Location {
	if r.Tied != nil && r.Tied.TZ != "" {
		if z, err := zones.Location(r.Tied.TZ); err == nil {
			return z
		}
	}
	return time.Local
}

// toSevereRow maps a domain row onto the UI type: the times pre-formatted in
// the right zone (tz lookups happen here, off the render path), the record
// already composed.
func toSevereRow(r severe.Row) tty.SevereRow {
	in := rowClock(r)
	rec := severe.RecordOf(r, in)
	row := tty.SevereRow{
		// Tab needs no conversion: the domain's and the window's are one type
		// since F-21, so there is no cast that could be wrong.
		Key: r.Key, Tab: r.Tab, Product: r.Product, Location: r.Location, Detection: severe.Detection(r),
		Severity: tty.TickerSeverity(r.Severity), Test: r.Test,
		Record: tty.SevereRecord{Title: rec.Title, Meta: rec.Meta, Timing: rec.Timing, Area: rec.Area, Paras: rec.Paras},
	}
	if !r.At.IsZero() { // a zero clock reads blank, as the record's stamp does — never "01/01 00:00" (R3-A-06)
		row.Declared = r.At.In(in).Format("01/02 15:04 MST")
	}
	if r.Name != "" {
		row.Product = r.Product + " " + r.Name
	}
	if !r.Until.IsZero() {
		row.Expires = r.Until.In(in).Format("01/02 15:04 MST")
	}
	return row
}

// dropSuperseded removes from a snapshot the alerts a newer message from the
// same sender and product has replaced (severe.Guard — the window's rule),
// so the [A] modal and the alert module never page an alert beside its own
// update (REVIEW R5-A-08; the window applied the guard, the tables did not).
// Runs on the publisher's own copy, before it is sent.
func dropSuperseded(snap *snapshot.Snapshot) {
	if snap == nil {
		return
	}
	superseded := severe.Guard(snap.Locations)
	if len(superseded) == 0 {
		return
	}
	for i := range snap.Locations {
		kept := snap.Locations[i].Alerts[:0]
		for _, a := range snap.Locations[i].Alerts {
			if key, _ := severe.NormalizeID(a.ID); !superseded[key] {
				kept = append(kept, a)
			}
		}
		snap.Locations[i].Alerts = kept
	}
}
