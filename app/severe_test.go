package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/branden-thompson/watchpost/platform/render"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestSevereDeckPublishesOnlyWhenTheIndexChanges(t *testing.T) {
	var sent []tea.Msg
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "Olathe", TZ: "America/Chicago", Alerts: []snapshot.Alert{{
		ID: "urn:oid:2.49.0.1.840.0.1.001.1", Event: "Tornado Warning", Severity: "extreme", Sent: time.Now(), Expires: time.Now().Add(time.Hour)}}}}}
	deck := newSevereDeck(func(m tea.Msg) { sent = append(sent, m) })
	deck.SetLocations(0, snap)
	deck.SetLocations(0, snap) // same index → no second message
	if len(sent) != 1 {
		t.Fatalf("sent %d messages for one index, want 1", len(sent))
	}
	msg := sent[0].(tty.SevereMsg)
	if len(msg.Rows) != 1 || msg.Rows[0].Tab != tty.SevereWarnings || msg.Totals[tty.SevereWarnings] != 1 {
		t.Fatalf("rows: %+v", msg)
	}
	if msg.Rows[0].Declared == "" || strings.HasSuffix(msg.Rows[0].Declared, "UTC") {
		t.Fatalf("a tied row shows the location's clock, got %q", msg.Rows[0].Declared)
	}
	fetched := time.Date(2026, 8, 28, 15, 38, 0, 0, time.UTC)
	quake := []globalfeed.Event{{ID: "us7000tbwb", Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Nepal", At: time.Now(), Quake: &globalfeed.QuakeDetail{}}}
	deck.SetFeed(quake, []SourceHealth{{Name: "USGS", OK: true, FetchedAt: fetched}, {Name: "NWS", OK: false}})
	if len(sent) != 2 || len(sent[1].(tty.SevereMsg).Rows) != 2 {
		t.Fatalf("a new feed event must publish: %d", len(sent))
	}
	last := sent[1].(tty.SevereMsg)
	if !last.Updated.Equal(fetched) || len(last.Sources) != 2 || last.Sources[1].OK {
		t.Fatalf("Updated must be the newest OK fetch and a dead source must show: %+v", last)
	}
	deck.SetFeed(quake, []SourceHealth{{Name: "USGS", OK: true, FetchedAt: fetched}, {Name: "NWS", OK: true, FetchedAt: fetched}})
	if len(sent) != 3 {
		t.Fatal("a source coming back must publish even with the same rows")
	}
	deck.SetFeed(quake, []SourceHealth{{Name: "USGS", OK: true, FetchedAt: fetched.Add(2 * time.Minute)}, {Name: "NWS", OK: true, FetchedAt: fetched.Add(2 * time.Minute)}})
	if len(sent) != 4 || !sent[3].(tty.SevereMsg).Updated.Equal(fetched.Add(2*time.Minute)) {
		t.Fatal("a fresh successful fetch must move Updated even with identical rows (FR-9)")
	}
}

func TestSevereDeckCapsAndCounts(t *testing.T) {
	var last tty.SevereMsg
	deck := newSevereDeck(func(m tea.Msg) { last = m.(tty.SevereMsg) })
	var evs []globalfeed.Event
	for i := 0; i < 520; i++ {
		evs = append(evs, globalfeed.Event{ID: fmt.Sprintf("us%04d", i), Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "x", At: time.Now().Add(-time.Duration(i) * time.Minute), Quake: &globalfeed.QuakeDetail{}})
	}
	deck.SetFeed(evs, nil)
	if len(last.Rows) != 500 || last.Totals[tty.SevereDisasters] != 520 {
		t.Fatalf("cap/count: %d rows, total %d", len(last.Rows), last.Totals[tty.SevereDisasters])
	}
}

func TestSevereDeckPublishIsSerialised(t *testing.T) {
	var mu sync.Mutex
	var gens []uint64
	deck := newSevereDeck(func(m tea.Msg) { mu.Lock(); gens = append(gens, m.(tty.SevereMsg).Gen); mu.Unlock() })
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
	// The LAST published key is the LAST feed's key (a slow publish never lands after a newer one — B-2).
	deck.mu.Lock()
	feed := deck.feed
	deck.mu.Unlock()
	if deck.lastKey != indexKey(severe.Sorted(severe.Union(feed, nil, deck.now())), nil) {
		t.Fatal("the final published key is not the final feed's key")
	}
}

func TestCycleLocatesBeforeTheRadiusFilterAndFeedsTheDeck(t *testing.T) {
	var got []globalfeed.Event
	deck := newSevereDeck(func(tea.Msg) {})
	deck.onFeed = func(evs []globalfeed.Event) { got = evs }
	far := globalfeed.Event{ID: "far", Class: globalfeed.ClassQuake, Type: "Earthquake", Place: "55 km NW of Kodāri, Nepal", Lat: 27.9, Lon: 85.6, HasPoint: true, At: time.Now()}
	td := &tickerDeck{send: func(tea.Msg) {}, sources: []globalfeed.Source{fakeSource{name: "stub", evs: []globalfeed.Event{far}}},
		watch: func() []snapshot.LocationRef {
			return []snapshot.LocationRef{{Label: "San Diego", Lat: 32.7, Lon: -117.2}}
		},
		seen: loadSeen(t.TempDir(), time.Hour), muted: &atomic.Bool{}, radius: &atomic.Int64{}, severe: deck, done: make(chan struct{})}
	td.radius.Store(50) // a 50-mile radius: the Nepal quake is OFF the tape…
	td.cycle(context.Background())
	if len(got) != 1 || got[0].Location == "" {
		t.Fatalf("…but the deck must receive the full, Locate'd set: %+v", got)
	}
}

func TestRecentPublishPokesTheDeck(t *testing.T) {
	var pokes atomic.Int32
	deck := newSevereDeck(func(tea.Msg) {})
	deck.onPublish = func() { pokes.Add(1) } // the publisher fires on a timer goroutine
	p := tea.NewProgram(nil)
	t.Cleanup(p.Kill) // a never-run program's Send blocks; Kill releases the publisher goroutine
	rp := startRecent(context.Background(), p, nil, []snapshot.LocationRef{{Label: "A", Zip: "00000", Lat: 1, Lon: 1}}, func(snap *snapshot.Snapshot) { deck.SetLocations(1, snap) })
	t.Cleanup(rp.stop)
	rp.pub.Trigger()
	deadline := time.Now().Add(2 * time.Second)
	for pokes.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if pokes.Load() == 0 {
		t.Fatal("a recent publish did not poke the deck")
	}
}

func TestNarrationPointsAtTheWindow(t *testing.T) {
	e := globalfeed.Event{Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "the Oklahoma City area", At: time.Date(2026, 8, 28, 15, 42, 0, 0, time.Local)}
	if got := alertNarration(nil, e, render.Clock12, sameDay); !strings.HasSuffix(got, ". Press W in Watchpost for the full report on this event") {
		t.Fatalf("tail: %q", got)
	}
	if got := burstClosingLine(nil, 0, "alerts"); got != "For more details on any of these alerts, press W in Watchpost." {
		t.Fatalf("burst closing: %q", got)
	}
	s := globalfeed.Event{Class: globalfeed.ClassTropical, Type: "Tropical Storm", Name: "Dolly", Location: "the Atlantic", At: e.At}
	if got := tapeHead(s); !strings.HasPrefix(got, "Tropical Storm Dolly · the Atlantic") {
		t.Fatalf("tape: %q", got)
	}
	if got := eventNarration(s, render.Clock12, sameDay); !strings.HasPrefix(got, "Tropical Storm Dolly has been reported for the Atlantic") {
		t.Fatalf("narration: %q", got)
	}
	evil := globalfeed.Event{Class: globalfeed.ClassTropical, Type: "Tropical Storm", Name: "Dolly\x1b]52;c;x\x07", Location: "the Atlantic", At: e.At}
	if got := eventNarration(evil, render.Clock12, sameDay); strings.ContainsAny(got, "\x1b\x07") {
		t.Fatalf("a provider escape reached the speech path: %q", got)
	}
	for _, name := range []string{"Dolly", "Idalia", "Lala"} {
		if got := synth.Pronounce("Tropical Storm " + name); !strings.Contains(got, name) {
			t.Fatalf("normaliser rewrote %q: %q", name, got)
		}
	}
}

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
		big[fmt.Sprintf("id%d", i)] = time.Now().Add(-time.Duration(i) * time.Second)
	}
	b, _ := json.Marshal(big)
	_ = os.WriteFile(s.path, b, 0o600)
	s2 := loadSeen(dir, time.Hour)
	if n := len(s2.set()); n > maxSeenIDs {
		t.Fatalf("load kept %d ids, cap %d", n, maxSeenIDs)
	}
}

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
	deck := newSevereDeck(func(tea.Msg) {})
	deck.SetFeed(feed, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		deck.SetLocations(0, snap) // unchanged index: Union + Sort + hash only — the steady-state cost every ≤ 20 s
	}
}

// TestSevereEscapesNeverReachTheFrameEndToEnd (P4-2): a hostile feed event —
// escapes in every field, the OSC clipboard write included — through the
// deck, the record composer, the message and the window's browse and detail
// renders. The ESC and BEL bytes never reach the frame on either path.
func TestSevereEscapesNeverReachTheFrameEndToEnd(t *testing.T) {
	evil := "x\x1b]52;c;aGVsbG8=\x07y\x1b[31mz"
	var got tty.SevereMsg
	deck := newSevereDeck(func(m tea.Msg) {
		if v, ok := m.(tty.SevereMsg); ok {
			got = v
		}
	})
	deck.now = func() time.Time { return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC) }
	deck.SetFeed([]globalfeed.Event{{
		ID: "https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.evil", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Name: evil, Place: evil,
		At: deck.now().Add(-10 * time.Minute), Until: deck.now().Add(time.Hour), Source: "NWS",
		Severe: &globalfeed.SevereDetail{Headline: evil, Description: evil, Instruction: evil, Severity: evil, Certainty: evil, Urgency: evil, SenderName: evil, AffectedZones: []string{evil}},
	}}, []SourceHealth{{Name: evil, OK: false}})
	if len(got.Rows) != 1 {
		t.Fatalf("rows %d", len(got.Rows))
	}
	m, err := tty.NewDashboard(tty.Config{Version: "t"})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 44})
	model, _ = model.Update(got)
	model, _ = model.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	browse := model.View().Content
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	detail := model.View().Content
	for name, frame := range map[string]string{"browse": browse, "detail": detail} {
		if strings.Contains(frame, "\x1b]") || strings.Contains(frame, "\x07") || strings.Contains(frame, "\x1b[31m") {
			t.Fatalf("%s: a feed escape reached the frame", name)
		}
	}
}

// The deck sees the snapshot the publisher is ABOUT to send, not the previous
// one (R3-A-02: reading pb.last back from inside the hook lagged the tables
// by one publish — a 0-row first publish, then ≤ 20 s / ≤ 2 min behind).
func TestDeckSeesTheSnapshotBeingPublished(t *testing.T) {
	var sent []tea.Msg
	deck := newSevereDeck(func(m tea.Msg) { sent = append(sent, m) })
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "Olathe", TZ: "America/Chicago", Alerts: []snapshot.Alert{{
		ID: "urn:oid:2.49.0.1.840.0.1.001.1", Event: "Tornado Warning", Severity: "extreme", Sent: time.Now(), Expires: time.Now().Add(time.Hour)}}}}}
	pub := &publisher{run: func() *snapshot.Snapshot { // startPriority's run shape: the hook runs before pb.last is stored
		deck.SetLocations(0, snap)
		return snap
	}}
	pub.fire()
	if len(sent) != 1 || len(sent[0].(tty.SevereMsg).Rows) != 1 {
		t.Fatalf("the publish that carried the warning must reach the window as a 1-row publish: %d messages", len(sent))
	}
}

// A same-id revision republishes (R3-A-03): a USGS magnitude/time update, and
// a row flipping from untied (national feed) to tied (a tracked location's
// alerts tier) — the zone rule and the prose depend on it.
func TestSevereDeckRepublishesOnContentChange(t *testing.T) {
	var sent []tea.Msg
	deck := newSevereDeck(func(m tea.Msg) { sent = append(sent, m) })
	m58, m61 := 5.8, 6.1
	at := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	deck.SetFeed([]globalfeed.Event{{ID: "us7000tbwb", Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Nepal", At: at, Quake: &globalfeed.QuakeDetail{Mag: &m58}}}, nil)
	deck.SetFeed([]globalfeed.Event{{ID: "us7000tbwb", Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Nepal", At: at, Quake: &globalfeed.QuakeDetail{Mag: &m61}}}, nil)
	if len(sent) != 2 {
		t.Fatalf("a revised magnitude must republish: %d publishes", len(sent))
	}
	sent = nil
	sentAt := time.Date(2026, 8, 28, 13, 0, 0, 0, time.UTC)
	deck.SetFeed([]globalfeed.Event{{ID: "https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.1.001.1", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
		Location: "Johnson County", At: sentAt, Until: time.Now().Add(time.Hour), Severe: &globalfeed.SevereDetail{SenderName: "NWS Kansas City", Sent: sentAt}}}, nil)
	deck.SetLocations(0, &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "Olathe", TZ: "America/Chicago", Alerts: []snapshot.Alert{{
		ID: "urn:oid:2.49.0.1.840.0.1.001.1", Event: "Tornado Warning", Severity: "extreme", SenderName: "NWS Kansas City", Sent: sentAt, Effective: sentAt, Expires: time.Now().Add(time.Hour), Description: "prose"}}}}})
	if len(sent) != 2 || sent[1].(tty.SevereMsg).Rows[0].Location != "Olathe" {
		t.Fatalf("the tie flip must republish with the location's record: %d publishes", len(sent))
	}
}

// A seen store left by 0.12.0 at 0644 is tightened on the next save (R3-D-01:
// WriteFile applies the mode only on create).
func TestSeenStoreUpgradeTightensAnOldMode(t *testing.T) {
	dir := t.TempDir()
	s := loadSeen(dir, time.Hour)
	if err := os.WriteFile(s.path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.mark([]globalfeed.Event{{ID: "a"}}, time.Now())
	s.save()
	if fi, _ := os.Stat(s.path); fi.Mode().Perm() != 0o600 {
		t.Fatalf("seen.json mode after save %o, NFR-13 wants 0600", fi.Mode().Perm())
	}
}

// The deck keeps its OWN copy of the locations and their alerts (REVIEW
// R5-B-01): the tty sorts the published snapshot's alerts in place on its
// loop while the deck re-reads them on the ticker's — the race probe, and
// the plain fact that a later mutation of the caller's slice never reaches
// the index.
func TestDeckKeepsItsOwnCopyOfTheLocations(t *testing.T) {
	deck := newSevereDeck(func(tea.Msg) {})
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "Olathe, KS", Alerts: []snapshot.Alert{
		{ID: "urn:oid:2.49.0.1.840.0.a1", Event: "Heat Advisory", Severity: "minor"},
		{ID: "urn:oid:2.49.0.1.840.0.a2", Event: "Tornado Warning", Severity: "extreme"},
	}}}}
	deck.SetLocations(0, snap)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { // the tty's in-place sort, over and over
		defer wg.Done()
		for i := 0; i < 200; i++ {
			sort.SliceStable(snap.Locations[0].Alerts, func(a, b int) bool { return snap.Locations[0].Alerts[a].ID < snap.Locations[0].Alerts[b].ID })
			snap.Locations[0].Alerts[0], snap.Locations[0].Alerts[1] = snap.Locations[0].Alerts[1], snap.Locations[0].Alerts[0]
		}
	}()
	go func() { // the deck's re-read on every feed publish
		defer wg.Done()
		for i := 0; i < 50; i++ {
			deck.SetFeed(nil, nil)
		}
	}()
	wg.Wait()
	snap.Locations[0].Alerts = nil // the caller drops its alerts: the index still has both
	deck.SetFeed(nil, nil)
	deck.mu.Lock()
	rows := len(deck.locs[0].Locations[0].Alerts)
	deck.mu.Unlock()
	if rows != 2 {
		t.Fatalf("the deck's copy has %d alerts, want 2", rows)
	}
}

// An alert's own update replaces it in the tables too (REVIEW R5-A-08): the
// publisher drops the superseded alert before the snapshot is sent, so the
// [A] modal and the module never page an alert beside its replacement.
func TestDropSupersededKeepsOnlyTheReplacement(t *testing.T) {
	older := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	newer := older.Add(10 * time.Minute)
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "L", Alerts: []snapshot.Alert{
		{ID: "urn:oid:1.1", Event: "Tornado Warning", SenderName: "NWS A", Sent: older},
		{ID: "urn:oid:1.2", Event: "Tornado Warning", SenderName: "NWS A", Sent: newer, References: []string{"https://api.weather.gov/alerts/urn:oid:1.1"}},
		{ID: "urn:oid:2.1", Event: "Heat Advisory", SenderName: "NWS A", Sent: older},
	}}}}
	dropSuperseded(snap)
	got := snap.Locations[0].Alerts
	if len(got) != 2 || got[0].ID != "urn:oid:1.2" || got[1].ID != "urn:oid:2.1" {
		t.Fatalf("want the update and the unrelated advisory, got %+v", got)
	}
	dropSuperseded(nil) // inert
}

// ALERTS - EVENTS scopes the WINDOW, not only the tape (HUM LEAD, UAT
// 2026-08-30).
//
// The window listed the pre-radius set while the tape listed the filtered one,
// so the two disagreed — and the STATEMENTS and ADVISORIES tabs, whose rows come
// only from the tracked locations and never from the national feed, were bounded
// by nothing but which locations happened to be on the watchlist. One preference
// governs both now.
func TestTheAlertRadiusScopesTheSevereWindowsLocationRows(t *testing.T) {
	near := snapshot.Location{Label: "Oceanside, CA", Lat: 33.24, Lon: -117.30,
		Alerts: []snapshot.Alert{{ID: "sps-near", Event: "Special Weather Statement", Effective: time.Now()}}}
	far := snapshot.Location{Label: "Lone Pine, CA", Lat: 36.61, Lon: -118.06, // ~240 mi away
		Alerts: []snapshot.Alert{{ID: "sps-far", Event: "Special Weather Statement", Effective: time.Now()}}}

	statements := func(radiusMi int) []string {
		d := newSevereDeck(func(tea.Msg) {})
		if radiusMi > 0 {
			d.radius = &atomic.Int64{}
			d.radius.Store(int64(radiusMi))
		}
		d.locs[0] = &snapshot.Snapshot{Locations: []snapshot.Location{near, far}}
		feed, locs := d.scope(defaultLocation(d.locs[0]), nil, d.locs[0].Locations)
		var out []string
		for _, r := range severe.Union(feed, locs, time.Now()) {
			if r.Tab == severe.TabStatements {
				out = append(out, r.Location)
			}
		}
		return out
	}

	if got := statements(0); len(got) != 2 {
		t.Errorf("All locations keeps every statement, got %v", got)
	}
	// Within 50 mi of the DEFAULT location — the first of the priority snapshot,
	// which is the location the setting names.
	got := statements(50)
	if len(got) != 1 || got[0] != "Oceanside, CA" {
		t.Errorf("Within 50 mi keeps only the near statement, got %v", got)
	}
}

// Filtered with NO default location set shows nothing rather than silently
// falling back to the unscoped set — the rule the tape already follows. A window
// that says it is scoped must not quietly show everything.
func TestAScopedWindowWithNoDefaultShowsNothing(t *testing.T) {
	d := newSevereDeck(func(tea.Msg) {})
	d.radius = &atomic.Int64{}
	d.radius.Store(25)
	feed, locs := d.scope(nil, []globalfeed.Event{{ID: "e", HasPoint: true}}, []snapshot.Location{{Label: "x"}})
	if len(feed) != 0 || len(locs) != 0 {
		t.Errorf("scoped with no default: feed %d rows, locations %d — both must be empty", len(feed), len(locs))
	}
}

// A RADIUS MUST NOT DELETE AN ALERT FOR HAVING COARSE GEOMETRY.
//
// Many NWS products are zone-only — watches, most flood warnings, heat
// advisories — and carry no polygon, so there is nothing to measure a radius
// against. Dropping them silenced the tape, the breaking takeover and the
// spoken alert for a warning the app was already tracking.
//
// THE INPUT IS BUILT BY THE PIPELINE, not by the test. globalfeed.Locate is
// what fills Location on the real path, and it deliberately skips the watchlist
// tie without a point — so a zone-only alert's Location is the feed's area
// description. A test that hand-sets Location to a watchlist label asserts a
// rule against an input the pipeline cannot produce, and passes while the
// product does not work.
func TestAZoneOnlyAlertTheAppIsTrackingSurvivesTheRadius(t *testing.T) {
	here := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.3}
	now := time.Now()
	zoneOnly := globalfeed.Event{
		ID: "urn:oid:2.49.0.1.840.0.abc123", Class: globalfeed.ClassSevereWx,
		Severity: globalfeed.SevRed, Type: "Tornado Warning",
		Place: "San Diego County", HasPoint: false,
		At: now, Until: now.Add(time.Hour),
	}
	zoneOnly.Location = globalfeed.Locate(zoneOnly.HasPoint, zoneOnly.Lat, zoneOnly.Lon,
		zoneOnly.Place, []snapshot.LocationRef{here}, nil)
	if zoneOnly.Location == here.Label {
		t.Fatalf("Locate tied a point-less alert to the watchlist (%q) — rewrite this test rather than trusting it", zoneOnly.Location)
	}

	tracked := []snapshot.Location{{Label: here.Label, Lat: here.Lat, Lon: here.Lon,
		Alerts: []snapshot.Alert{{ID: zoneOnly.ID, Event: "Tornado Warning"}}}}
	if kept := scopeEvents([]globalfeed.Event{zoneOnly}, here.Lat, here.Lon, 100, alertKeysOf(tracked)); len(kept) != 1 {
		t.Errorf("a zone-only warning the app is tracking must reach every surface, got %d kept", len(kept))
	}
	// The radius still means something.
	if stray := scopeEvents([]globalfeed.Event{zoneOnly}, here.Lat, here.Lon, 100, nil); len(stray) != 0 {
		t.Errorf("a zone-only alert nothing is tracking must not be pulled in, got %d", len(stray))
	}
	far := zoneOnly
	far.ID, far.HasPoint, far.Lat, far.Lon = "urn:oid:2.49.0.1.840.0.far", true, 44.0, -93.0
	if out := scopeEvents([]globalfeed.Event{far}, here.Lat, here.Lon, 100, alertKeysOf(tracked)); len(out) != 0 {
		t.Errorf("a distant polygonal alert must still be filtered out, got %d", len(out))
	}
}

// THE RADIUS BRANCH UNDER CONCURRENCY (red-team BUILD exit, CQ-1/SC-3).
//
// scope() used to re-read s.locs[0] outside s.mu while SetLocations wrote it —
// three race sites on the alert path. The -race gate could not see it: every
// test that set a radius called scope() DIRECTLY, single-goroutine, so the
// branch was never entered concurrently. This test enters it the way the app
// does, through publish, which is the only reason -race now covers it.
func TestScopeIsRaceFreeUnderConcurrentLocationUpdates(t *testing.T) {
	d := newSevereDeck(func(tea.Msg) {})
	d.radius = &atomic.Int64{}
	d.radius.Store(100) // the branch: ALERTS-EVENTS is not "All"
	d.SetFeed([]globalfeed.Event{{
		ID: "e1", Type: "Tornado Warning", Lat: 33.2, Lon: -117.3, HasPoint: true,
		At: time.Now(), Until: time.Now().Add(time.Hour),
	}}, nil)

	var wg sync.WaitGroup
	for i := range 40 {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			d.SetLocations(0, &snapshot.Snapshot{Locations: []snapshot.Location{
				{Label: "Oceanside, CA", Lat: 33.0 + float64(i)/100, Lon: -117.3},
			}})
		}(i)
		go func() { defer wg.Done(); d.publish() }()
	}
	wg.Wait()
}

// THE TIE IS SCOPED BY THE RADIUS TOO, driven through cycle.
//
// A point-less alert is kept when the app is already tracking it — but "already
// tracking" must mean a location INSIDE the radius. The RECENT table is seeded
// with the fifty largest US cities and each fetches its own alerts, so an
// unscoped tie hands the marquee and the spoken read to a warning a thousand
// miles from the listener, defeating their own setting from the other side.
func TestCycleScopesTheZoneOnlyTieByTheRadiusToo(t *testing.T) {
	const (
		nearID = "urn:oid:2.49.0.1.840.0.aaa111.1.1"
		farID  = "urn:oid:2.49.0.1.840.0.bbb222.1.1"
		// The feed serves the URL form and a tracked location carries the bare
		// OID, so the tie is a join across the two — which is the thing worth
		// testing, and the thing a same-string fixture never touches.
		feedPrefix = "https://api.weather.gov/alerts/"
	)
	oceanside := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.3}
	now := time.Now()
	zoneOnly := func(id, place string) globalfeed.Event {
		return globalfeed.Event{ID: id, Class: globalfeed.ClassSevereWx, Severity: globalfeed.SevRed,
			Type: "Tornado Warning", Place: place, HasPoint: false, At: now, Until: now.Add(time.Hour)}
	}

	deck := newSevereDeck(func(tea.Msg) {})
	// Both are tracked: one where the listener lives, one a seeded RECENT row
	// 1,700 miles away.
	deck.SetLocations(1, &snapshot.Snapshot{Locations: []snapshot.Location{
		{Label: oceanside.Label, Lat: oceanside.Lat, Lon: oceanside.Lon,
			Alerts: []snapshot.Alert{{ID: nearID, Event: "Tornado Warning"}}},
		{Label: "Chicago, IL", Lat: 41.9, Lon: -87.6,
			Alerts: []snapshot.Alert{{ID: farID, Event: "Tornado Warning"}}},
	}})

	var tape []tty.TickerItem
	td := &tickerDeck{
		send: func(m tea.Msg) {
			if v, ok := m.(tty.TickerMsg); ok {
				tape = v.Items
			}
		},
		sources: []globalfeed.Source{fakeSource{name: "stub",
			evs: []globalfeed.Event{zoneOnly(feedPrefix+nearID, "San Diego County"), zoneOnly(feedPrefix+farID, "Cook County")}}},
		watch: func() []snapshot.LocationRef { return []snapshot.LocationRef{oceanside} },
		seen:  loadSeen(t.TempDir(), time.Hour),
		muted: &atomic.Bool{}, radius: &atomic.Int64{}, severe: deck, done: make(chan struct{}),
	}
	td.radius.Store(100)
	td.warm.Store(true)
	td.cycle(context.Background())

	var onTape []string
	for _, it := range tape {
		onTape = append(onTape, it.Head)
	}
	near, far := false, false
	for _, head := range onTape {
		if strings.Contains(head, "San Diego County") {
			near = true
		}
		if strings.Contains(head, "Cook County") {
			far = true
		}
	}
	if !near {
		t.Errorf("a zone-only warning the listener is tracking where they live must reach the tape: %v", onTape)
	}
	if far {
		t.Errorf("a zone-only warning 1,700 miles away must NOT reach the tape or the spoken read — the radius is the listener's setting: %v", onTape)
	}
}

// EACH LOCATION-ONLY LANE KEEPS ITS OWN BUDGET, AND ITS WORST ROWS.
//
// Advisories and Special Weather Statements reach the marquee only through the
// tracked locations, and they shared one budget — so a day of heat advisories
// emptied the Statements lane while its statements were live, and the rotation
// lost a whole lane. Driven through publish, which is what fills LaneRows.
func TestEachLocationOnlyLaneKeepsItsOwnBudget(t *testing.T) {
	now := time.Now()
	alert := func(id, event, severity string, at time.Time) snapshot.Alert {
		return snapshot.Alert{ID: id, Event: event, Severity: severity, Sent: at,
			Effective: at, Expires: now.Add(6 * time.Hour), AreaDesc: "San Diego County"}
	}
	// One old but SEVERE advisory against a flood of fresher, milder ones, so
	// the cut has to choose on severity within the lane — plus three statements,
	// which must survive the advisory flood entirely.
	alerts := []snapshot.Alert{alert("urn:oid:2.49.0.1.840.0.aaa111.1.1", "Heat Advisory", "Severe", now.Add(-6*time.Hour))}
	for i := range 2 * maxLaneRows {
		alerts = append(alerts, alert(fmt.Sprintf("urn:oid:2.49.0.1.840.0.a%05x.1.1", i),
			"Heat Advisory", "Minor", now.Add(-time.Duration(i)*time.Minute)))
	}
	for i := range 3 {
		alerts = append(alerts, alert(fmt.Sprintf("urn:oid:2.49.0.1.840.0.b%05x.1.1", i),
			"Special Weather Statement", "Minor", now.Add(-time.Duration(i)*time.Minute)))
	}

	deck := newSevereDeck(func(tea.Msg) {})
	deck.SetLocations(0, &snapshot.Snapshot{Locations: []snapshot.Location{
		{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.3, Alerts: alerts}}})
	deck.publish()

	perTab := map[severe.Tab]int{}
	var advisories []severe.Row
	for _, r := range deck.LaneRows() {
		perTab[r.Tab]++
		if r.Tab == severe.TabAdvisories {
			advisories = append(advisories, r)
		}
	}
	if perTab[severe.TabAdvisories] != maxLaneRows {
		t.Errorf("an over-full advisories lane is cut to exactly maxLaneRows=%d, got %d", maxLaneRows, perTab[severe.TabAdvisories])
	}
	if perTab[severe.TabStatements] != 3 {
		t.Errorf("the statements lane has its own budget and keeps all 3, got %d", perTab[severe.TabStatements])
	}
	// The cut is by severity: the six-hour-old severe advisory outranks sixty
	// fresher minor ones and must survive.
	if !slices.ContainsFunc(advisories, func(r severe.Row) bool { return r.Severity == globalfeed.SevOrange }) {
		t.Error("the lane's most severe row was evicted by fresher, milder ones")
	}
	// …and what survives reads most-recent-first, like every other lane.
	for i := 1; i < len(advisories); i++ {
		if advisories[i].At.After(advisories[i-1].At) {
			t.Fatalf("the kept rows must read most-recent-first: %v before %v", advisories[i-1].At, advisories[i].At)
		}
	}
}

// AN UNIDENTIFIABLE ALERT TIES TO NOTHING.
//
// The tie matches a point-less feed event to a tracked alert by normalised id.
// An id the app cannot identify must key nothing, or one such tracked alert
// would pull the national feed's zone-only products past a listener's radius,
// onto the tape and into the spoken read.
//
// Driven through cycle, because that is where a real feed's ids arrive. The
// live arm is the NON-EMPTY unrecognised id: an event with no id at all is
// already dropped by globalfeed.Merge and by severe's index before the tie is
// reached, so that arm is belt-and-braces here and is pinned where it can
// actually fail, in TestAlertKeysOfRefusesAnythingItCannotIdentify.
func TestAnUnidentifiableAlertTiesToNothing(t *testing.T) {
	now := time.Now()
	oceanside := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.3}
	zoneOnly := func(id, place string) globalfeed.Event {
		return globalfeed.Event{ID: id, Class: globalfeed.ClassSevereWx, Severity: globalfeed.SevRed,
			Type: "Tornado Warning", Place: place, HasPoint: false, At: now, Until: now.Add(time.Hour)}
	}

	deck := newSevereDeck(func(tea.Msg) {})
	// A tracked location whose alerts carry ids nothing can normalise: one
	// empty, one that is not a CAP id at all.
	deck.SetLocations(0, &snapshot.Snapshot{Locations: []snapshot.Location{
		{Label: oceanside.Label, Lat: oceanside.Lat, Lon: oceanside.Lon, Alerts: []snapshot.Alert{
			{ID: "", Event: "Tornado Warning"},
			{ID: "not-an-oid", Event: "Tornado Warning"},
		}}}})

	var tape []tty.TickerItem
	td := &tickerDeck{
		send: func(m tea.Msg) {
			if v, ok := m.(tty.TickerMsg); ok {
				tape = v.Items
			}
		},
		sources: []globalfeed.Source{fakeSource{name: "stub", evs: []globalfeed.Event{
			zoneOnly("", "Cook County"),           // no id either
			zoneOnly("not-an-oid", "Erie County"), // an id, but not one NormalizeID accepts
		}}},
		watch: func() []snapshot.LocationRef { return []snapshot.LocationRef{oceanside} },
		seen:  loadSeen(t.TempDir(), time.Hour),
		muted: &atomic.Bool{}, radius: &atomic.Int64{}, severe: deck, done: make(chan struct{}),
	}
	td.radius.Store(100)
	td.warm.Store(true)
	td.cycle(context.Background())

	for _, it := range tape {
		if strings.Contains(it.Head, "Cook County") || strings.Contains(it.Head, "Erie County") {
			t.Errorf("an alert the app cannot identify must tie to nothing, yet %q defeated the radius", it.Head)
		}
	}
}

// THE TIE SET REFUSES ANYTHING IT CANNOT IDENTIFY.
//
// Asserted directly, because through the pipeline it cannot be: globalfeed.Merge
// and severe's own index both drop an id-less event before the tie is reached,
// so an end-to-end test of the empty-id case passes whatever this function does.
// The guard is real all the same — one tracked alert keying the empty string
// would tie every point-less event that also lacks an id — and this is the only
// place it can be shown to work.
func TestAlertKeysOfRefusesAnythingItCannotIdentify(t *testing.T) {
	const real = "urn:oid:2.49.0.1.840.0.aaa111.1.1"
	keys := alertKeysOf([]snapshot.Location{{Label: "Oceanside, CA", Alerts: []snapshot.Alert{
		{ID: "", Event: "Tornado Warning"},           // no id at all
		{ID: "not-an-oid", Event: "Tornado Warning"}, // an id, but not a CAP one
		{ID: "urn:oid:", Event: "Tornado Warning"},   // the prefix alone
	}}})
	if len(keys) != 0 {
		t.Errorf("an alert the app cannot identify must key nothing, got %v", keys)
	}

	// The positive half: without it this passes on a function that keys nothing
	// at all, and the tie would be silently dead rather than selective.
	keys = alertKeysOf([]snapshot.Location{{Label: "Oceanside, CA",
		Alerts: []snapshot.Alert{{ID: real, Event: "Tornado Warning"}}}})
	if !keys[real] {
		t.Errorf("a real CAP alert id must key the tie set, got %v", keys)
	}
}

// THE LANE CUT MUST NOT REORDER THE WINDOW.
//
// publish sorts its rows once, hands the same slice to setLaneRows, and then
// reuses it for the window's own cap and totals. setLaneRows cuts each
// location-only lane by SEVERITY, which is a different order from the window's
// most-recent-first — so if that cut reached the caller's slice, the window
// would silently start listing by severity, and the two surfaces the listener
// compares would disagree about the same alerts.
//
// The fixture is built so the two orders genuinely differ: an old but severe
// advisory against newer mild ones. Sorted the window's way it comes last;
// sorted the lane's way it comes first.
func TestTheLaneCutDoesNotReorderTheWindow(t *testing.T) {
	now := time.Now()
	alert := func(id, event, severity string, at time.Time) snapshot.Alert {
		return snapshot.Alert{ID: id, Event: event, Severity: severity, Sent: at,
			Effective: at, Expires: now.Add(6 * time.Hour), AreaDesc: "San Diego County"}
	}
	alerts := []snapshot.Alert{
		alert("urn:oid:2.49.0.1.840.0.aaa111.1.1", "Heat Advisory", "Severe", now.Add(-6*time.Hour)), // old, severe
	}
	for i := range 4 { // newer, milder
		alerts = append(alerts, alert(fmt.Sprintf("urn:oid:2.49.0.1.840.0.b%05x.1.1", i),
			"Heat Advisory", "Minor", now.Add(-time.Duration(i)*time.Minute)))
	}

	var msg tty.SevereMsg
	deck := newSevereDeck(func(m tea.Msg) {
		if v, ok := m.(tty.SevereMsg); ok {
			msg = v
		}
	})
	deck.SetLocations(0, &snapshot.Snapshot{Locations: []snapshot.Location{
		{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.3, Alerts: alerts}}})
	deck.publish()

	if len(msg.Rows) != len(alerts) {
		t.Fatalf("the window must list every alert: got %d of %d", len(msg.Rows), len(alerts))
	}
	// Fixture validity: the severe row must be the OLDEST, so that severity-first
	// and recency-first genuinely disagree about where it goes. Without that the
	// test proves nothing whichever order the window is in.
	var keys []string
	for _, r := range msg.Rows {
		keys = append(keys, r.Key)
	}
	severe := "urn:oid:2.49.0.1.840.0.aaa111.1.1"
	if !slices.Contains(keys, severe) {
		t.Fatalf("the severe row must reach the window at all: %v", keys)
	}
	// The window reads most-recent-first, so the oldest row is last. The lane's
	// own cut is severity-first and would put it first.
	if keys[len(keys)-1] != severe {
		t.Errorf("the lane's severity cut has reordered the window — the oldest row must be last, order was %v", keys)
	}
}

// THE TWO TAB ENUMS ARE ONE ORDERING, NOT TWO THAT LOOK ALIKE.
//
// toSevereRow converts with a numeric cast — tty.SevereTab(r.Tab) — so the
// domain's Tab and the window's SevereTab must agree position by position. A
// compile-time check already asserts they are the same LENGTH; nothing asserted
// they are in the same ORDER, and reordering one without the other would file
// every row under a neighbouring category with no error anywhere.
func TestTheDomainAndWindowTabsAgreePositionByPosition(t *testing.T) {
	for _, c := range []struct {
		domain severe.Tab
		window tty.SevereTab
		name   string
	}{
		{severe.TabEmergency, tty.SevereEmergency, "Emergency"},
		{severe.TabWarnings, tty.SevereWarnings, "Warnings"},
		{severe.TabWatches, tty.SevereWatches, "Watches"},
		{severe.TabAdvisories, tty.SevereAdvisories, "Advisories"},
		{severe.TabStatements, tty.SevereStatements, "Statements"},
		{severe.TabDisasters, tty.SevereDisasters, "Disasters"},
		{severe.TabMarine, tty.SevereMarine, "Marine"},
		{severe.TabForecasts, tty.SevereForecasts, "Forecasts"},
	} {
		if got := tty.SevereTab(c.domain); got != c.window {
			t.Errorf("%s: the domain says %d and the window says %d — a row would land in the wrong tab",
				c.name, c.domain, c.window)
		}
	}
	if int(severe.NumTabs) != 8 {
		t.Errorf("a tab was added or removed without this list: %d", severe.NumTabs)
	}
}
