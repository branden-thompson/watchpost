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

	"github.com/branden-thompson/watchpost/platform/render"
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
	"80x24":  3_598 * 1.05, // 0.13.0 P3-9: the worst-case floor (NFR-2) — measured 3,370; 3,598 after the player facelift's box
}

// frameAllocBudgetHit pins the memo-hit frame (plan §1: ≤ 6,000 at 133×44
// after Q3; measured at Q3 and set at × 1.05 like the miss path).
var frameAllocBudgetHit = map[string]float64{
	"133x44": 6_000,
	"133x70": 6_000,
	"200x60": 6_000,
	"80x24":  962 * 1.05, // measured 962 after red-team round 4 (B-06: the layout builds the player and control rows once per frame — 1,312 before, when the thin-bands re-resolution built them four times; was 996 before the boxes)
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
	// 0.11.0: a recent quake so the budget measures the seismic row mark's
	// per-row Tint on the miss path (REVIEW P5 D2 — the fixture previously
	// carried no hazard marks, leaving that cost unmeasured).
	loc.Seismic = &snapshot.SeismicState{AsOf: loc.Harmonized.Source.IssuedAt, Quakes: []snapshot.Quake{{Mag: 4.2, DistanceKm: 20, Bearing: "NE", DepthKm: 8}}}
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

// --- 0.13.0: the Severe Weather / Disaster Events window (plan P3-9) ---

// severeAllocBudget pins the window's frame: the modal-memo HIT (every tick
// while it is open — the number the radio-on target reads) and the MISS (a
// publish, a key). Measured at P3-9 and set at × 1.05 like the frame pins.
var severeAllocBudget = map[string]float64{"hit": 2_401 * 1.05, "miss": 7_561 * 1.05} // BUILD-exit measurement: hit 2,401 (the overlay compositor is most of it) · miss 7,561 (was 3,067 / 8,061 before the row copies went — R3-B-11)

// severeBench is the window open over the 133×44 fixture with a 60-row
// Warnings index (a busy outbreak day, not the 9-row mock).
func severeBench(tb testing.TB) Dashboard {
	tb.Helper()
	d := benchDash(tb, 133, 44).(Dashboard)
	var rows []SevereRow
	for i := range 60 {
		rows = append(rows, SevereRow{Key: fmt.Sprint(i), Tab: SevereWarnings, Product: "Severe Thunderstorm Warning", Location: fmt.Sprintf("Benchmark County %02d, KS", i), Declared: "08/28 08:45 CDT", Expires: "08/28 09:30 CDT",
			Record: SevereRecord{Title: "SEVERE THUNDERSTORM WARNING", Meta: "[Severe · Immediate · Observed]", Timing: "Declared 08/28 08:45 CDT   Expires 08/28 09:30 CDT   (~45m)", Area: "Area: Benchmark County, KS · NWS Topeka", Paras: []string{"At 845 AM CDT, a severe thunderstorm was located near Benchmark, moving east at 35 mph. HAZARD: 60 mph wind gusts and quarter size hail.", "Instructions: For your protection move to an interior room on the lowest floor of a building."}}})
	}
	var m tea.Model = d
	m, _ = m.Update(SevereMsg{Gen: 1, Rows: rows, Totals: [severeNumTabs]int{60}})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	return m.(Dashboard)
}

func BenchmarkFrame_133x44_Severe(b *testing.B) {
	d := severeBench(b)
	b.ReportAllocs()
	for b.Loop() {
		_ = d.View().Content
	}
}

// BenchmarkOverlayOnly isolates the compositor: render.Overlay alone, over a
// pre-rendered base and window (the number the memo cannot lower).
func BenchmarkOverlayOnly(b *testing.B) {
	d := severeBench(b)
	o := d.opts()
	modal := d.renderModal(o)
	closed := d
	closed.modal = modalNone
	base := closed.View().Content
	b.ReportAllocs()
	for b.Loop() {
		_ = render.Overlay(base, modal, d.width)
	}
}

func TestSevereFrameAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := severeBench(t)
	_ = d.View().Content
	hit := testing.AllocsPerRun(20, func() { _ = d.View().Content })
	miss := testing.AllocsPerRun(20, func() { d.mmemo.ok = false; _ = d.View().Content })
	t.Logf("severe window: hit %.0f allocs (budget %.0f) · miss %.0f allocs (budget %.0f)", hit, severeAllocBudget["hit"], miss, severeAllocBudget["miss"])
	if hit > severeAllocBudget["hit"] {
		t.Errorf("severe window (memo hit) allocates %.0f per View(), budget %.0f", hit, severeAllocBudget["hit"])
	}
	if miss > severeAllocBudget["miss"] {
		t.Errorf("severe window (memo miss) allocates %.0f per View(), budget %.0f", miss, severeAllocBudget["miss"])
	}
}
