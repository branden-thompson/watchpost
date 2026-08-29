package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	if len(last.Rows) != 500 || last.Totals[tty.SevereQuakes] != 520 {
		t.Fatalf("cap/count: %d rows, total %d", len(last.Rows), last.Totals[tty.SevereQuakes])
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
	if got := alertNarration(nil, e); !strings.HasSuffix(got, ". Press W in Watchpost for the full report on this event") {
		t.Fatalf("tail: %q", got)
	}
	if got := burstClosingLine(nil); got != "For the full report on any of these events, press W in Watchpost." {
		t.Fatalf("burst closing: %q", got)
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
