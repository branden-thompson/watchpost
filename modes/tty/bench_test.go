package tty

// Frame benchmarks and the allocation pin (quality pass Q0, plan §1 and
// Q0 task 5). The fixture is DISCOVER lens L5's canonical one — 10
// favourites + 50 recent, 7 forecast days, alerts on every third row —
// with colour forced ON and TERM set, because that is how the dashboard
// renders for a person (the kit gates its palette on $TERM; `go test`
// otherwise measures a colour-off frame — red-team PA-6, CQ-4, R2-8).
//
// Wall-clock numbers are recorded (`make quality-bench`), never gated.
// The allocation count is deterministic and IS gated: TestFrameAllocBudget
// runs in the non-race CI step (`make alloc-budget`).

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// frameAllocBudget is the pin per fixture size (allocations per View()).
// Q0 sets it at DISCOVER's baseline × 1.05; Q3 lowers it to ≤ 6,000 at
// 133×44 and Q4b to ≤ 3,300 (plan §1). A change that raises a count above
// its budget is a regression the gate refuses.
//
// Since Q3 there are two paths: the memo HIT (every tick, marquee and
// visualizer frame between input changes — the number §1's radio-on target
// reads) and the MISS (a snapshot, key or resize re-renders the tables).
var frameAllocBudget = map[string]float64{
	"133x44": 10_044 * 1.05, // Q0 measurement: 10,044 — the Q3 hit path is pinned in frameAllocBudgetHit
	"133x70": 15_539 * 1.05,
	"200x60": 20_031 * 1.05,
}

// frameAllocBudgetHit pins the memo-hit frame (plan §1: ≤ 6,000 at 133×44
// after Q3; measured at Q3 and set at × 1.05 like the miss path).
var frameAllocBudgetHit = map[string]float64{
	"133x44": 6_000,
	"133x70": 6_000,
	"200x60": 6_000,
}

func benchLoc(i int, days int, alert bool) snapshot.Location {
	loc := snapshot.Location{
		Label: fmt.Sprintf("Benchmark City %02d", i+1), Zip: fmt.Sprintf("9%04d", i),
		Lat: 33.0 + float64(i)/10, Lon: -117.0 - float64(i)/10,
		Harmonized: snapshot.Conditions{Temp: f64(22.8 + float64(i%7)), Condition: "partly_cloudy",
			Source: snapshot.SourceInfo{Provider: "nws", ModelOrStation: "KCRQ", DistanceKm: f64(12.5)}},
	}
	for d := range days {
		loc.Daily = append(loc.Daily, snapshot.Daily{Date: fmt.Sprintf("2026-08-%02d", 24+d), TempMax: f64(23.9 + float64(d)), TempMin: f64(17.2), Condition: "clear"})
	}
	if alert {
		loc.Alerts = []snapshot.Alert{{Event: "Extreme Heat Watch", Severity: "severe", Headline: "until Friday"}}
	}
	return loc
}

// benchDash is the canonical fixture at a terminal size: 10 favourites,
// 50 recent, colour on.
func benchDash(tb testing.TB, w, h int) tea.Model {
	tb.Helper()
	tb.Setenv("TERM", "xterm-256color")
	rendering.SetColorEnabledForTest(true)
	tb.Cleanup(rendering.ResetColorEnabledForTest)
	m, err := NewDashboard(Config{Version: "bench"})
	if err != nil {
		tb.Fatal(err)
	}
	obs := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	sn := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion, GeneratedAt: obs,
		Providers: []snapshot.ProviderStatus{{ID: "nws", Status: snapshot.ProviderOK, FetchedAt: obs}, {ID: "firms", Status: snapshot.ProviderOK, FetchedAt: obs}}}
	for i := range 10 {
		sn.Locations = append(sn.Locations, benchLoc(i, 7, i%3 == 0))
	}
	rs := &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion, GeneratedAt: obs}
	for i := range 50 {
		rs.Locations = append(rs.Locations, benchLoc(100+i, 7, i%5 == 0))
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: w, Height: h})
	model, _ = model.Update(SnapshotMsg{Snap: sn})
	model, _ = model.Update(RecentSnapshotMsg{Snap: rs})
	return model
}

func benchFrame(b *testing.B, w, h int) {
	m := benchDash(b, w, h)
	v := m.View().Content
	b.ReportMetric(float64(len(v)), "bytes/frame")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = m.View().Content
	}
}

func BenchmarkFrame_133x44(b *testing.B) { benchFrame(b, 133, 44) }
func BenchmarkFrame_133x70(b *testing.B) { benchFrame(b, 133, 70) }
func BenchmarkFrame_200x60(b *testing.B) { benchFrame(b, 200, 60) }

// BenchmarkFrame_Help is the modal path (Overlay compositor — L5-F9).
func BenchmarkFrame_133x44_Help(b *testing.B) {
	m := benchDash(b, 133, 44)
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	b.ReportAllocs()
	for b.Loop() {
		_ = m.View().Content
	}
}

// TestFrameAllocBudget pins allocations per frame at each size. It skips
// under the race detector (counts are not comparable there) and runs in
// `make alloc-budget`. A miss prints the measured value so the build log
// can record it.
func TestFrameAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	for size, budget := range frameAllocBudget {
		var w, h int
		if _, err := fmt.Sscanf(size, "%dx%d", &w, &h); err != nil {
			t.Fatal(err)
		}
		d := benchDash(t, w, h).(Dashboard)
		_ = d.View().Content // warm the kit's probes and the theme
		hit := testing.AllocsPerRun(20, func() { _ = d.View().Content })
		miss := testing.AllocsPerRun(20, func() { d.memo.ok = false; _ = d.View().Content })
		t.Logf("frame %s: hit %.0f allocs (budget %.0f) · miss %.0f allocs (budget %.0f)", size, hit, frameAllocBudgetHit[size], miss, budget)
		if hit > frameAllocBudgetHit[size] {
			t.Errorf("frame %s (memo hit) allocates %.0f per View(), budget %.0f — a render regression (plan §1)", size, hit, frameAllocBudgetHit[size])
		}
		if miss > budget {
			t.Errorf("frame %s (memo miss) allocates %.0f per View(), budget %.0f — a render regression (plan §1)", size, miss, budget)
		}
	}
}

// BenchmarkFrame_133x44_Miss is the table re-render path (a snapshot, a
// key, a resize): the memo slot is invalidated before every frame.
func BenchmarkFrame_133x44_Miss(b *testing.B) {
	d := benchDash(b, 133, 44).(Dashboard)
	b.ReportAllocs()
	for b.Loop() {
		d.memo.ok = false
		_ = d.View().Content
	}
}
