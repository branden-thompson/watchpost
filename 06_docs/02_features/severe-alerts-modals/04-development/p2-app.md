# P2 — App: publish path, `severeDeck`, narration, seen-store

Depends on P1. `app/` imports `domains/*`, `modes/tty` (UI types) and `platform/*`.

---

## Task 2.1 — `Locate` before the radius; hand the deck the full set

**File:** `app/ticker.go` (MODIFY — `cycle`)

**Test first (RED):** `app/ticker_test.go` (append)
```go
func TestCycleLocatesBeforeTheRadiusFilterAndFeedsTheDeck(t *testing.T) {
	var got []globalfeed.Event
	deck := newSevereDeck(func(tea.Msg) {}, func() []*snapshot.Snapshot { return nil })
	deck.onFeed = func(evs []globalfeed.Event) { got = evs }
	far := globalfeed.Event{ID: "far", Class: globalfeed.ClassQuake, Type: "Earthquake", Place: "55 km NW of Kodāri, Nepal", Lat: 27.9, Lon: 85.6, HasPoint: true, At: time.Now()}
	td := &tickerDeck{send: func(tea.Msg) {}, sources: []globalfeed.Source{fakeSource{name: "stub", evs: []globalfeed.Event{far}}},
		watch: func() []snapshot.LocationRef { return []snapshot.LocationRef{{Label: "San Diego", Lat: 32.7, Lon: -117.2}} },
		seen: loadSeen(t.TempDir(), time.Hour), muted: &atomic.Bool{}, radius: &atomic.Int64{}, severe: deck, done: make(chan struct{})}
	td.radius.Store(50) // a 50-mile radius: the Nepal quake is OFF the tape…
	td.cycle(context.Background())
	if len(got) != 1 || got[0].Location == "" {
		t.Fatalf("…but the deck must receive the full, Locate'd set: %+v", got)
	}
}
```
(`fakeSource{name, evs}` is the existing test double, `app/ticker_test.go:128-134`.)

**Code:** in `tickerDeck` add the field `severe *severeDeck // 0.13.0: the severe-events index (nil in the
old tests = no window)`. In `cycle`, record each source's outcome in the fetch loop:
```go
	var events []globalfeed.Event
	health := make([]SourceHealth, 0, len(t.sources))
	fetchedAt := time.Now()
	for _, s := range t.sources {
		evs, err := s.Fetch(ctx)
		if err != nil {
			health = append(health, SourceHealth{Name: s.Name(), OK: false}) // a dead source: no fetch time to report
			continue // a feed outage leaves its events absent this cycle; the others still show
		}
		health = append(health, SourceHealth{Name: s.Name(), OK: true, FetchedAt: fetchedAt})
		events = append(events, evs...)
	}
```
(no new deck state: the deck's "Updated" is the newest OK fetch across the sources it was last handed —
red-team PLAN-CQ #3 removed a `lastOK` map that would have been nil on the first cycle.) Then move the `Locate` loop above the radius branch and add the
hand-off:
```go
	now := time.Now()
	events = globalfeed.Active(events, now) // drop alerts past their active window (#2)
	watch := t.watch()
	// Tie EVERY event to its representative location before any filter
	// (0.13.0, red-team C-2): the severe-events window lists the pre-radius
	// set and needs the labels; the tape's radius filter applies below.
	for i := range events {
		events[i].Location = globalfeed.Locate(events[i].HasPoint, events[i].Lat, events[i].Lon, events[i].Place, watch, t.nearest)
	}
	if t.severe != nil {
		t.severe.SetFeed(events, health) // the window's half of the index — its own copy (SetFeed clones)
	}
	// The alert-radius filter (HUM LEAD): "Filtered to N Mi of my location"
	// scopes the whole ticker to within N miles of the default location; 0 = All.
	// Filtered but no location set → show nothing, never silently fall back to
	// the global stack the UI says is scoped away (red-team 0.12.0 P4 F7).
	if r := int(t.radius.Load()); r > 0 {
		if len(watch) > 0 {
			events = globalfeed.Within(events, watch[0].Lat, watch[0].Lon, float64(r))
		} else {
			events = nil
		}
	}
```
and delete the old `for i := range events { events[i].Location = … }` loop that followed the radius branch.

**Verify:** `go test ./app -run 'TestCycleLocates' -v`

---

## Task 2.2 — `severeDeck` and the `SevereMsg` mapping

**File:** `app/severe.go` (CREATE)

**Test first (RED):** `app/severe_test.go`
```go
package app

import (
	"fmt"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestSevereDeckPublishesOnlyWhenTheIndexChanges(t *testing.T) {
	var sent []tea.Msg
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "Olathe", TZ: "America/Chicago", Alerts: []snapshot.Alert{{
		ID: "urn:oid:2.49.0.1.840.0.1.001.1", Event: "Tornado Warning", Severity: "extreme", Sent: time.Now(), Expires: time.Now().Add(time.Hour)}}}}}
	deck := newSevereDeck(func(m tea.Msg) { sent = append(sent, m) }, func() []*snapshot.Snapshot { return []*snapshot.Snapshot{snap} })
	deck.Trigger()
	deck.Trigger() // same index → no second message
	if len(sent) != 1 {
		t.Fatalf("sent %d messages for one index, want 1", len(sent))
	}
	msg := sent[0].(tty.SevereMsg)
	if len(msg.Rows) != 1 || msg.Rows[0].Tab != tty.SevereWarnings || msg.Totals[tty.SevereWarnings] != 1 {
		t.Fatalf("rows: %+v", msg)
	}
	if msg.Rows[0].Declared == "" || msg.Rows[0].Declared[len(msg.Rows[0].Declared)-3:] == "UTC" {
		t.Fatalf("a tied row shows the location's clock, got %q", msg.Rows[0].Declared)
	}
	fetched := time.Date(2026, 8, 28, 15, 38, 0, 0, time.UTC)
	deck.SetFeed([]globalfeed.Event{{ID: "us7000tbwb", Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Nepal", At: time.Now(), Quake: &globalfeed.QuakeDetail{}}},
		[]SourceHealth{{Name: "USGS", OK: true, FetchedAt: fetched}, {Name: "NWS", OK: false}})
	if len(sent) != 2 || len(sent[1].(tty.SevereMsg).Rows) != 2 {
		t.Fatalf("a new feed event must publish: %d", len(sent))
	}
	last := sent[1].(tty.SevereMsg)
	if !last.Updated.Equal(fetched) || len(last.Sources) != 2 || last.Sources[1].OK {
		t.Fatalf("Updated must be the newest OK fetch and a dead source must show: %+v", last)
	}
	sameFeed := []globalfeed.Event{{ID: "us7000tbwb", Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Nepal", At: time.Now(), Quake: &globalfeed.QuakeDetail{}}}
	deck.SetFeed(sameFeed, []SourceHealth{{Name: "USGS", OK: true, FetchedAt: fetched}, {Name: "NWS", OK: true, FetchedAt: fetched}})
	if len(sent) != 3 {
		t.Fatal("a source coming back must publish even with the same rows")
	}
	deck.SetFeed(sameFeed, []SourceHealth{{Name: "USGS", OK: true, FetchedAt: fetched.Add(2 * time.Minute)}, {Name: "NWS", OK: true, FetchedAt: fetched.Add(2 * time.Minute)}})
	if len(sent) != 4 || !sent[3].(tty.SevereMsg).Updated.Equal(fetched.Add(2*time.Minute)) {
		t.Fatal("a fresh successful fetch must move Updated even with identical rows (FR-9)")
	}
}

func TestSevereDeckPublishIsSerialised(t *testing.T) {
	var mu sync.Mutex
	var gens []uint64
	deck := newSevereDeck(func(m tea.Msg) { mu.Lock(); gens = append(gens, m.(tty.SevereMsg).Gen); mu.Unlock() }, func() []*snapshot.Snapshot { return nil })
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			deck.SetFeed([]globalfeed.Event{{ID: fmt.Sprintf("us%d", i), Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "x", At: time.Now(), Quake: &globalfeed.QuakeDetail{}}}, nil)
		}(i)
	}
	wg.Wait()
	for i := 1; i < len(gens); i++ {
		if gens[i] <= gens[i-1] {
			t.Fatalf("generations went backwards: %v", gens)
		}
	}
	// The LAST message must carry the LAST feed set (a slow publish never lands after a newer one — B-2).
	if last := deck.lastKey; last != indexKey(severe.Sorted(severe.Union(deck.feed, nil, deck.now())), nil) {
		t.Fatal("the final published key is not the final feed's key")
	}
}

func TestSevereDeckCapsAndCounts(t *testing.T) {
	var last tty.SevereMsg
	deck := newSevereDeck(func(m tea.Msg) { last = m.(tty.SevereMsg) }, func() []*snapshot.Snapshot { return nil })
	var evs []globalfeed.Event
	for i := 0; i < 520; i++ {
		evs = append(evs, globalfeed.Event{ID: "us" + string(rune('a'+i%26)) + string(rune('a'+i/26)), Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "x", At: time.Now().Add(-time.Duration(i) * time.Minute), Quake: &globalfeed.QuakeDetail{}})
	}
	deck.SetFeed(evs, nil)
	if len(last.Rows) != 500 || last.Totals[tty.SevereQuakes] != 520 {
		t.Fatalf("cap/count: %d rows, total %d", len(last.Rows), last.Totals[tty.SevereQuakes])
	}
}
```

**Code:**
```go
package app

// severe.go — the severe-events index deck (0.13.0, SAM-D-26 AX-1 = A): the
// join between the ticker's pre-radius feed events and the tracked locations'
// alerts. The ticker cycle hands over the feed half (SetFeed); the priority and
// recent publishers poke it after every publish (Trigger); it recomputes the
// index through domains/severe and sends a SevereMsg to the dashboard ONLY when
// the set of rows changed — so a 20-second alerts tier never churns the modal
// memo. Pure computation, microseconds; runs off the tea update loop.

import (
	"crypto/sha256"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	zones "github.com/branden-thompson/watchpost/platform/tz"
)

// SourceHealth is one feed's last outcome, for the window's "Updated" stamp
// and its source line (red-team PLAN A-1/B-7: a dead source must show, and the
// stamp is the newest successful fetch — dataAsOf — never the publish time).
type SourceHealth struct {
	Name      string
	OK        bool
	FetchedAt time.Time // the last successful fetch; zero = never
}

type severeDeck struct {
	// publishMu serialises the WHOLE publish (read inputs → compute → compare →
	// send): the ticker goroutine and two publisher timers call it concurrently,
	// and a slow publish must never land after a newer one (red-team PLAN B-2).
	publishMu sync.Mutex
	mu        sync.Mutex // guards feed/sources between SetFeed and publish
	feed      []globalfeed.Event
	sources   []SourceHealth
	snaps     func() []*snapshot.Snapshot // the publishers' last snapshots (nil entries skipped)
	send      func(tea.Msg)
	gen       uint64
	lastKey   [32]byte
	onFeed    func([]globalfeed.Event) // test hook: observe the feed half
	onPublish func()                   // test hook
	now       func() time.Time
}

// The window's tab count and the domain's must agree: an array conversion
// between different lengths does not compile (red-team PLAN S4).
var _ = [severe.NumTabs]int(tty.SevereMsg{}.Totals)

func newSevereDeck(send func(tea.Msg), snaps func() []*snapshot.Snapshot) *severeDeck {
	return &severeDeck{send: send, snaps: snaps, now: time.Now}
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

// Trigger republishes after a snapshot publish (the location half changed).
func (s *severeDeck) Trigger() { s.publish() }

// publish recomputes the index and sends it when the row set (or a source's
// health) changed. Serialised end to end.
func (s *severeDeck) publish() {
	s.publishMu.Lock()
	defer s.publishMu.Unlock()
	if s.onPublish != nil {
		s.onPublish()
	}
	s.mu.Lock()
	feed, sources := s.feed, s.sources
	s.mu.Unlock()
	var locs []snapshot.Location
	if s.snaps != nil {
		for _, sn := range s.snaps() {
			if sn != nil {
				locs = append(locs, sn.Locations...)
			}
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
		return // nothing changed: no message, no memo churn (the 20-second alerts tier lands here — red-team PLAN P4: nothing below runs)
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
	for _, r := range kept { // records are composed only here — on a change — never per trigger (B-5)
		msg.Rows = append(msg.Rows, toSevereRow(r))
	}
	s.send(msg)
}

// indexKey fingerprints the row set (keys + sent times) and the sources'
// health + fetch minute, so an unchanged index publishes nothing while a fresh
// successful fetch still refreshes "Updated" (FR-9). Sent is bounded to the
// second by the feed parsers, so a source cannot make the key churn by itself.
func indexKey(rows []severe.Row, sources []SourceHealth) [32]byte {
	h := sha256.New()
	for _, r := range rows {
		h.Write([]byte(r.Key))
		h.Write([]byte(r.Sent.UTC().Format(time.RFC3339)))
		h.Write([]byte{0})
	}
	for _, src := range sources {
		h.Write([]byte(src.Name))
		if src.OK {
			h.Write([]byte{1})
			h.Write([]byte(src.FetchedAt.UTC().Truncate(time.Minute).Format(time.RFC3339))) // a fresh successful fetch moves "Updated" even with identical rows (minute-coarse — the 2-min cadence)
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
// the right zone (tz lookups happen here, off the render path — red-team P9),
// the record already composed.
func toSevereRow(r severe.Row) tty.SevereRow {
	in := rowClock(r)
	rec := severe.RecordOf(r, in)
	row := tty.SevereRow{
		Key: r.Key, Tab: tty.SevereTab(r.Tab), Product: r.Product, Location: r.Location,
		Declared: r.At.In(in).Format("01/02 15:04 MST"), Severity: tty.TickerSeverity(r.Severity),
		Record: tty.SevereRecord{Title: rec.Title, Meta: rec.Meta, Timing: rec.Timing, Area: rec.Area, Paras: rec.Paras},
	}
	if r.Name != "" {
		row.Product = r.Product + " " + r.Name
	}
	if !r.Until.IsZero() {
		row.Expires = r.Until.In(in).Format("01/02 15:04 MST")
	}
	return row
}
```
(`platform/tz` exposes `Location(name) (*time.Location, error)` — the same import `alerts.go` uses.)

(`severe.Sorted` is `Sort` returning its argument — add the one-line helper in Task 1.8 beside `Sort`.)

**Verify:** `go test -race ./app -run 'TestSevereDeck' -v` (the serialisation test is only meaningful under `-race`)

---

## Task 2.3 — Wire the deck: construction, ticker, publisher triggers

**Files:** `app/dashboard.go` (MODIFY), `app/pipelines.go` (MODIFY `startRecent`), `app/ticker.go` (MODIFY `startTicker`)

Design: no new hook type. `startPriority` already takes `onPublish func(*snapshot.Snapshot)`
(`pipelines.go:170`); `startRecent` gets the same parameter, and `startPipelines` composes the deck's
`Trigger` into both closures.

**Test first (RED):** `app/severe_test.go` (append)
```go
func TestRecentPublishPokesTheDeck(t *testing.T) {
	pokes := 0
	deck := newSevereDeck(func(tea.Msg) {}, func() []*snapshot.Snapshot { return nil })
	deck.onPublish = func() { pokes++ }
	p := tea.NewProgram(nil)
	rp := startRecent(context.Background(), p, nil, []snapshot.LocationRef{{Label: "A", Zip: "00000", Lat: 1, Lon: 1}}, func(*snapshot.Snapshot) { deck.Trigger() })
	rp.pub.Trigger()
	time.Sleep(50 * time.Millisecond) // the first publish runs at once, on a timer goroutine
	rp.stop()
	if pokes != 1 {
		t.Fatalf("recent publish did not poke the deck: %d", pokes)
	}
}
```
(`startRecent` with nil providers builds an assembler over an empty provider set — the existing
`TestRecentPipeline*` tests do the same; if `tea.NewProgram(nil)` is rejected by `p.Send`, use the
`captureProgram` helper the ticker tests use.) Add `onPublish func()` to `severeDeck` (test hook, called at
the top of `publish`).

**Code — `app/pipelines.go`:** `startRecent(ctx, p, providers, refs, onPublish func(*snapshot.Snapshot))`;
inside its `run` closure, after `snap := rp.asm.Snapshot()` and before `p.Send(tty.RecentSnapshotMsg{…})`:
```go
		if onPublish != nil {
			onPublish(snap)
		}
```
Update the two existing callers (`startPipelines`; any test) to pass `nil` or the composed hook.

**Code — `app/dashboard.go`:** in `livePipelines` add `severe *severeDeck // 0.13.0: the severe-events
index`. In `startPipelines`, before `lp.priority = …`:
```go
	lp.severe = newSevereDeck(p.Send, lp.lastSnapshots) // 0.13.0: fed by the ticker cycle, poked by both publishers
```
compose the priority hook — and assign under `lp.mu`, because the publisher's first fire is immediate:
```go
	lp.mu.Lock()
	lp.priority = startPriority(httpx.WithPriority(ctx), p, lp.providers(), refs, func(snap *snapshot.Snapshot) {
		if fullyPopulated(snap) { // M1: CAS records only the first fully-populated publish
			firstFullNanos.CompareAndSwap(0, int64(time.Since(start)))
		}
		lp.severe.Trigger()
	})
	lp.recent = startRecent(ctx, p, lp.providers(), restoreRecent(refsFromConfig(cfg.Recent), refs, seedRecent(idx, refs, tty.RecentCap), tty.RecentCap), func(*snapshot.Snapshot) { lp.severe.Trigger() })
	lp.mu.Unlock()
```
(`lastSnapshots` takes the same lock; `commit` already holds it across `priority.update` — `Trigger` only
schedules a timer there, so no re-entrancy.)
Pass `lp.severe` to `startTicker` (new trailing parameter). Add:
```go
// lastSnapshots are the publishers' most recent snapshots — the severe
// index's location half. Nil entries (no publish yet, or an empty RECENT list
// whose publisher is nil — pipelines.go:209) are skipped by the deck.
func (lp *livePipelines) lastSnapshots() []*snapshot.Snapshot {
	lp.mu.Lock() // the priority publisher's FIRST fire is immediate, on a timer goroutine, and can reach here before startPipelines' assignment lands (red-team PLAN-CQ #7)
	defer lp.mu.Unlock()
	var out []*snapshot.Snapshot
	if lp.priority != nil && lp.priority.pub != nil {
		out = append(out, lp.priority.pub.lastSnapshot())
	}
	if lp.recent != nil && lp.recent.pub != nil {
		out = append(out, lp.recent.pub.lastSnapshot())
	}
	return out
}
```
**Code — `app/ticker.go`:** `startTicker(…, audio tickerAudio, severe *severeDeck)` sets `severe: severe`
in the literal; update the existing call site(s) and tests (pass `nil`).

**Verify:** `go build ./... && go test ./app -run 'TestSevereDeck|TestRecentPublishPokes|TestCycle' -v`

---

## Task 2.4 — Narration: press W; the tape and the sentence name storms

**File:** `app/ticker.go` (MODIFY)

**Test first (RED):** `app/ticker_test.go` (append)
```go
func TestNarrationPointsAtTheWindow(t *testing.T) {
	e := globalfeed.Event{Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "the Oklahoma City area", At: time.Date(2026, 8, 28, 15, 42, 0, 0, time.Local)}
	if got := alertNarration(e); !strings.HasSuffix(got, ". Press W in Watchpost for the full report on this event") {
		t.Fatalf("tail: %q", got)
	}
	if burstClosing != "For the full report on any of these events, press W in Watchpost." {
		t.Fatalf("burst closing: %q", burstClosing)
	}
	s := globalfeed.Event{Class: globalfeed.ClassTropical, Type: "Tropical Storm", Name: "Dolly", Location: "the Atlantic", At: e.At}
	if got := tapeText(s); !strings.HasPrefix(got, "Tropical Storm Dolly · the Atlantic") {
		t.Fatalf("tape: %q", got)
	}
	if got := eventNarration(s); !strings.HasPrefix(got, "Tropical Storm Dolly has been reported for the Atlantic") {
		t.Fatalf("narration: %q", got)
	}
	evil := globalfeed.Event{Class: globalfeed.ClassTropical, Type: "Tropical Storm", Name: "Dolly\x1b]52;c;x\x07", Location: "the Atlantic", At: e.At}
	if got := eventNarration(evil); strings.ContainsAny(got, "\x1b\x07") {
		t.Fatalf("a provider escape reached the speech path: %q", got)
	}
	// Storm names pass the synth's normaliser untouched (pronounced as written).
	for _, name := range []string{"Dolly", "Idalia", "Lala"} {
		if got := synth.Pronounce("Tropical Storm " + name); !strings.Contains(got, name) { // the substitution table, domains/radio/synth/normalize.go:304
			t.Fatalf("normaliser rewrote %q: %q", name, got)
		}
	}
}
```

(`synth.Pronounce` is the exported substitution pass, `domains/radio/synth/normalize.go:304`; `app` already
imports `domains/radio/synth`.)

**Code:** replace the three strings/functions (add `platform/render` to `ticker.go`'s imports if absent).
```go
// alertNarration is a SINGLE event's full narration: its line plus the tail
// directing the listener to the severe-events window (0.13.0, SAM-D-26 N-1).
func alertNarration(e globalfeed.Event) string {
	return eventNarration(e) + ". Press W in Watchpost for the full report on this event"
}

// eventNarration is one event's spoken line (0.13.0: through render.Plain —
// a provider-supplied storm NAME now reaches the synthesiser, and the tape
// already strips at app/ticker.go:365; the speech path did not — red-team
// PLAN S1). ExpandStates in AlertNarration reads "VA" as "Virginia".
func eventNarration(e globalfeed.Event) string {
	s := e.Sentence() + " at " + clock(e.At)
	if !e.Until.IsZero() {
		s += " until " + clock(e.Until)
	}
	return render.Plain(s)
}

// burstClosing is the one tail after a multi-event burst (HUM LEAD script,
// re-pointed at the window in 0.13.0).
const burstClosing = "For the full report on any of these events, press W in Watchpost."
```
and in `tapeText` replace `e.Type + " · "` with `e.Title() + " · "` (the named storm on the tape,
SAM-D-14).

**Verify:** `go test ./app -run 'Narration|Tape' -v` (NAR: 2/2 strings)

---

## Task 2.5 — NFR-1 by construction, and the deck's cost measured

**File:** `app/severe_test.go` (append)

NFR-1 ("no new fetches") holds **by construction**: `severeDeck` has no `httpx.Client` and no `Source`; it
can only read what the ticker already fetched and what the publishers already published. A counting test
on a client the deck cannot reach would be vacuous (red-team PLAN A-3); the guard is the type itself, plus
this comment in `severe.go`'s doc block: "The deck never fetches — adding a client field here needs a new
NFR." What IS measured is the deck's cost per trigger (red-team B-5): Union + Sort + key hash over the
canonical load, and the change-gated record composition.
```go
func BenchmarkSevereDeckTrigger(b *testing.B) {
	snap := &snapshot.Snapshot{}
	for i := 0; i < 60; i++ {
		loc := snapshot.Location{Label: fmt.Sprintf("L%02d", i), TZ: "America/Chicago"}
		for j := 0; j < 3; j++ {
			loc.Alerts = append(loc.Alerts, snapshot.Alert{ID: fmt.Sprintf("urn:oid:2.49.0.1.840.0.%02d%02d.001.1", i, j), Event: "Flood Warning", Severity: "severe", Sent: time.Now(), Expires: time.Now().Add(time.Hour), Description: strings.Repeat("prose ", 300)})
		}
		snap.Locations = append(snap.Locations, loc)
	}
	var feed []globalfeed.Event
	for i := 0; i < 400; i++ {
		feed = append(feed, globalfeed.Event{ID: fmt.Sprintf("https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.f%03d.001.1", i), Class: globalfeed.ClassSevereWx, Type: "Severe Thunderstorm Warning", Location: "x", At: time.Now(), Until: time.Now().Add(time.Hour), Severe: &globalfeed.SevereDetail{Description: strings.Repeat("prose ", 300)}})
	}
	deck := newSevereDeck(func(tea.Msg) {}, func() []*snapshot.Snapshot { return []*snapshot.Snapshot{snap} })
	deck.SetFeed(feed, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		deck.Trigger() // unchanged index: Union + Sort + hash only — the steady-state cost every ≤ 20 s
	}
}
```
Record ns/op and allocs/op in the P2 build log; the expectation is well under 1 ms and no message. A
second benchmark with `deck.SetFeed(feed[:len(feed)-1], nil)` alternating measures the change path
(records composed) — expected single-digit ms for 500 rows, at most once per 2-minute cycle.

**Verify:** `go test ./app -run '^$' -bench SevereDeck -benchmem`

---

## Task 2.6 — Seen-store hardening (NFR-13)

**File:** `app/ticker.go` (MODIFY `seenStore.save` / `loadSeen`)

**Test first (RED):** `app/ticker_test.go` (append)
```go
func TestSeenStoreIsPrivateAndBounded(t *testing.T) {
	dir := t.TempDir()
	s := loadSeen(dir, time.Hour)
	s.mark([]globalfeed.Event{{ID: "a"}}, time.Now())
	s.save()
	fi, err := os.Stat(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("seen.json mode %o, want 0600", fi.Mode().Perm())
	}
	big := make(map[string]time.Time, maxSeenIDs+10)
	for i := 0; i < maxSeenIDs+10; i++ {
		big[fmt.Sprintf("id%d", i)] = time.Now()
	}
	b, _ := json.Marshal(big)
	_ = os.WriteFile(s.path, b, 0o600)
	s2 := loadSeen(dir, time.Hour)
	if n := len(s2.set()); n > maxSeenIDs {
		t.Fatalf("load kept %d ids, cap %d", n, maxSeenIDs)
	}
}
```

**Code:** add `const maxSeenIDs = 20_000 // 7 days × a few hundred ids/day, with room (P10-03)`; in `save()`
use `0o700` / `0o600`; in `loadSeen`, after unmarshalling, if `len(ids) > maxSeenIDs` drop the OLDEST
entries until at the cap:
```go
	if len(ids) > maxSeenIDs {
		type kv struct {
			id string
			at time.Time
		}
		all := make([]kv, 0, len(ids))
		for id, at := range ids {
			all = append(all, kv{id, at})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].at.After(all[j].at) })
		ids = make(map[string]time.Time, maxSeenIDs)
		for _, e := range all[:maxSeenIDs] {
			ids[e.id] = e.at
		}
	}
```
(`sort` is already imported in `ticker.go`.)

**Verify:** `go test ./app -run 'TestSeenStore' -v`

**Batch exit:** `make verify`; `go test -race ./app` (the ticker/deck goroutine seam); commit
`feat(severe): app join — deck, publish path, narration, seen-store hardening`.
