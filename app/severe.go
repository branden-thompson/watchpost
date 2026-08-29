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
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	zones "github.com/branden-thompson/watchpost/platform/tz"
)

// The window's tab count and the domain's must agree: an array conversion
// between different lengths does not compile.
var _ = [severe.NumTabs]int(tty.SevereMsg{}.Totals)

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
}

func newSevereDeck(send func(tea.Msg)) *severeDeck {
	return &severeDeck{send: send, now: time.Now}
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
	s.mu.Lock()
	feed, sources, snaps := s.feed, s.sources, s.locs
	s.mu.Unlock()
	var locs []snapshot.Location
	for _, sn := range snaps {
		if sn != nil {
			locs = append(locs, sn.Locations...)
		}
	}
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	rows := severe.Union(feed, locs, now)
	severe.Sort(rows)
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
		Key: r.Key, Tab: tty.SevereTab(r.Tab), Product: r.Product, Location: r.Location, Detection: severe.Detection(r),
		Severity: tty.TickerSeverity(r.Severity),
		Record:   tty.SevereRecord{Title: rec.Title, Meta: rec.Meta, Timing: rec.Timing, Area: rec.Area, Paras: rec.Paras},
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
