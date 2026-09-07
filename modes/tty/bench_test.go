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
// Every pin is x1.05 of the measurement taken against benchDash's LIVE MARQUEE
// — the state the app actually runs in. A fixture with an empty ticker reads
// about 10 % under the real cost, and it is the hit path that the band keeps
// drawing around the clock.
var frameAllocBudget = map[string]float64{
	"133x44": 5_608 * 1.05,
	"133x70": 11_477 * 1.05,
	"200x60": 14_477 * 1.05,
	"80x24":  2_188 * 1.05,
}

// frameAllocBudgetHit pins the memo-hit frame — THE PIN THAT MATTERS.
//
// The marquee scrolls on every tick, so tickNeeded holds whenever there are
// active events, which is nearly always: this frame is drawn about three times
// a second for as long as the app is up, and every allocation on it is a
// permanent cost. Treat a failure here as a regression to fix, not a budget to
// raise. Measured at x1.05 against benchDash's live marquee, whose showing lane
// is FULL — the state that costs the most per frame, and the one the earlier
// numbers (313/331/334/325, a tape spread thin across six lanes) did not reach.
var frameAllocBudgetHit = map[string]float64{
	"133x44": 365 * 1.05,
	"133x70": 383 * 1.05,
	"200x60": 386 * 1.05,
	"80x24":  377 * 1.05,
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
	model, _ = model.Update(TickerMsg{Items: benchTicker()})
	return model
}

// benchTicker is the marquee the pins measure against.
//
// THE BAND IS WHY THE FRAME RUNS AT ALL. tickNeeded holds while the ticker has
// events, and there is essentially always an active national hazard — so the
// state every allocation pin should be guarding is a dashboard with a LIVE
// tape, not an empty one. A fixture without it measures a program the app does
// not run, and reads ~10 % under the real cost at every size.
//
// ONE LANE, FULL. Only the showing lane is formatted into the frame, so the
// most expensive tape is not the largest one — it is the one whose visible lane
// is full. Spreading the same events across every lane leaves five in view and
// measures a cheaper program than the app runs.
func benchTicker() []TickerItem {
	const activeAlerts = 30 // globalfeed.MaxPerLane: one lane's own cap
	items := make([]TickerItem, 0, activeAlerts)
	const lane = CatWarning // a warnings outbreak: the realistic full lane
	for i := range activeAlerts {
		items = append(items, TickerItem{
			ID:       fmt.Sprintf("bench-%02d", i),
			Category: lane,
			Head:     fmt.Sprintf("Severe Thunderstorm Warning · Benchmark County %02d, KS", i),
			Verb:     "issued",
			At:       time.Date(2026, 8, 24, 0, int(i), 0, 0, time.UTC),
		})
	}
	return items
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
// The hit path here is dominated by render.Overlay's compositor, which is an
// accepted cost — docs/accepted-costs.md §1 states why and what would re-open it.
var severeAllocBudget = map[string]float64{"hit": 1_740 * 1.05, "miss": 5_248 * 1.05} // was 2_401 / 7_561 at the 0.13.0 BUILD exit

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
	m, _ = m.Update(SevereMsg{Gen: 1, Rows: rows, Totals: [severeNumTabs]int{SevereWarnings: 60}})
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

// --- P0 (0.14.0 multi-voice-support): the Setup window ---
//
// The "before" pins for the Setup window, taken at P0 against TODAY'S form so
// the two new groups (ALERTS - TONE, WATCHPOST RADIO - CORRESPONDENTS) are added
// against a recorded baseline rather than a guess. P4 Task 4.11 re-measures the
// new window and re-pins these numbers — the pin is always a measurement, never
// a formula (07-readiness/perf-protocol.md §2).

// setupAllocBudget pins the Setup window's frame at both sizes: the modal-memo
// HIT (every tick while it is open) and the MISS (a key, a publish). Set at
// × 1.05 of the measurement, as the frame and severe pins are.
//
// RE-PINNED at 0.14.0 P4 Task 4.11, against the NEW window (four groups, two
// columns, the correspondents focused). P0 recorded the old window as an
// informational baseline: 133×44 hit 3,046 / miss 3,476 · 80×24 hit 1,978 /
// miss 2,478.
//
// What moved, and why it is acceptable:
//
//   - The HIT path got CHEAPER (3,046 → 2,114 and 1,978 → 1,799), because the
//     modal memo now serves a window that is rebuilt far less often.
//   - The MISS path grew (3,476 → 3,895 = 1.12x, and 2,478 → 3,474 = 1.40x).
//     The plan's rule is that a miss beyond TWICE the baseline means the
//     builders are fixed rather than the pin raised; both are well inside it,
//     and the growth buys twenty rows where there were three.
//   - 80×24 grew most because that is where the window STACKS: it builds both
//     columns and then lays them one after the other, so a miss there does the
//     two-column work and the stacking.
//
// Measured against the same live-marquee fixture as the frame pins.
var setupAllocBudget = map[string]float64{
	"133x44-hit": 2_630 * 1.05, "133x44-miss": 3_808 * 1.05,
	// 80x24-miss re-measured at 0.14.0 T3.2b for the RELAY REPLAY group's
	// second row: 2_678 -> 2_800 (+4.6%), the cost of drawing two more lines
	// and a spacer at the width where the window still scrolls. The 133x44
	// numbers went DOWN over the same change (2_545 against a 2_762 budget),
	// because sizing the pickers' cells to their own values took cells out of
	// every row that has one.
	"80x24-hit": 1_604 * 1.05, "80x24-miss": 2_800 * 1.05,
}

// Re-measured after the columns were BALANCED automatically (HUM LEAD, UAT
// 2026-08-30). The window is much better and the numbers barely moved, and the
// reason is worth writing down because it is the opposite of what was expected.
//
// The window at 133×44 went from 39 body lines to 25 and from 121 cells wide to
// 118 — short enough to need no scroll rail at all. The hypothesis was that a
// shorter window would allocate less, since fewer rows are drawn.
//
// It allocates MORE on the hit path: 2,391 → 2,828. All 437 of that is
// render.Overlay, measured directly (1,808 → 2,245 for the same frame). Overlay
// costs what it does NOT cover: a dashboard row the modal sits over is replaced
// outright, while a row beside or below it has to be spliced around, and
// splicing a styled line means walking its escapes. A smaller window uncovers
// more of the dashboard, so it costs more to composite.
//
// Which puts the real lever in view: Overlay is 2,245 of this window's 2,828
// allocations per frame, on every tick it is open. Nothing else in the frame is
// close, and no amount of trimming the window's own content will touch it.
// That is a job for the performance pass, not for this change.
//
// The rebuild paths did move the right way where it matters: 80×24, which
// stacks and so does the most work, went 4,464 → 4,484 (flat) after the split
// search was made to cost nothing — see columnPlan and setupBlock.w. Before
// that it was 6,904, because every candidate split re-measured every block.

// Re-measured after the WATCHPOST UI group joined the window (HUM LEAD, UAT
// 2026-08-30) — a picker and five radio rows, twelve lines.
//
// The MISS grew as the extra rows would predict (4,936 → 5,477 at 133×44,
// 4,176 → 4,464 at 80×24): a miss builds the whole body, and the body is a
// group longer.
//
// The HIT fell on both (2,761 → 2,391 and 2,018 → 1,832), which the group did
// not do. The header did: [t] Theme and [M] Mute Severe Alerts left the row, so
// the masthead builds five fewer styled chips on every frame — and unlike the
// body, the header is rebuilt whether or not the modal memo hits. The two paths
// moving in opposite directions is the same lesson as the tone group's: what
// this window costs per frame and what it costs to rebuild are different
// questions with different answers.
//
// Against P0's 3,476 the miss is now 1.58x, inside the plan's twice-the-baseline
// rule but a third of the way through what is left of it.

// Re-measured after the tone group went to ONE CLASS PER LINE (HUM LEAD, UAT
// 2026-08-30). Every path got cheaper — 133×44 hit 2,932 → 2,761 and miss
// 5,461 → 4,936; 80×24 hit 2,411 → 2,018 and miss 4,768 → 4,176 — and the miss
// is now 1.42x P0's 3,476, down from 1.57x.
//
// The saving is not really the group. Six short rows made the LEFT COLUMN
// narrow enough that the window lays its groups side by side again, as the mock
// draws them, and a two-column body is short enough to need no scroll rail. The
// window stopped wrapping lines and stopped drawing a rail.
//
// Worth keeping in mind, because the intermediate arrangement proved the
// converse: laying the classes two abreast with each sub-column sized to its own
// labels made the group narrower AND made the 133×44 hit path DEARER (3,288),
// because it left the window at a middling width where neither the two-column
// body nor the rail-free body applied. This window's frame cost is not a
// monotone function of how much it draws.

// WHERE THIS IS AGAINST THE PRE-FEATURE WINDOW, honestly: the miss path is now
// 4,936 against P0's 3,476 — 1.42x. The plan's rule is that beyond TWICE the
// baseline the BUILDERS are fixed rather than the pin raised, so there is
// headroom, but it is being spent and the next addition should look at the
// builders first.
//
// Two things were already tried and are kept because they are right, not
// because they paid: the ←→ chips are built once per frame and shared by all
// thirteen controls (they were being built twenty-six times), and the modal
// memo keys on the Setup GENERATION rather than fmt.Sprintf("%+v") over a
// struct carrying two maps. Neither moved the number much — the cost is spread
// across the frame rather than concentrated anywhere — which is itself worth
// knowing before someone else goes looking.

// Re-measured again after the UAT round that tinted every focused label
// (settingLabel) — one more styled span on the focused row, and the group
// headers moved to the bold ModalTitle tone.
//
// Re-measured before that after the UAT change that made each picker two key CHIPS
// instead of a dropdown cell (HUM LEAD, 2026-08-30). Seven pickers x two chips
// is fourteen more styled spans per frame; the hit path moved 2,114 -> 2,392
// and the miss 3,895 -> 4,405. Still far inside the twice-the-baseline rule,
// and the chips are what make the control honest about the keys that move it.

// setupBench is the Setup window open over the fixture dashboard at w×h, with
// the CORRESPONDENTS group focused — the heaviest state: the pickers draw, the
// focused row's note is built and wrapped, and the two-column decision runs.
func setupBench(tb testing.TB, w, h int) Dashboard {
	tb.Helper()
	d := benchDash(tb, w, h).(Dashboard)
	return d.openSetupAt(rowCastAlerts)
}

func BenchmarkSetup_133x44(b *testing.B) {
	d := setupBench(b, 133, 44)
	b.ReportAllocs()
	for b.Loop() {
		_ = d.View().Content
	}
}

func BenchmarkSetup_80x24(b *testing.B) {
	d := setupBench(b, 80, 24)
	b.ReportAllocs()
	for b.Loop() {
		_ = d.View().Content
	}
}

// BenchmarkSetup_133x44_Miss is the re-render path: the modal memo slot is
// invalidated before every frame, as a keypress inside the form does.
func BenchmarkSetup_133x44_Miss(b *testing.B) {
	d := setupBench(b, 133, 44)
	b.ReportAllocs()
	for b.Loop() {
		d.mmemo.ok = false
		_ = d.View().Content
	}
}

func TestSetupAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	for _, size := range []struct {
		name string
		w, h int
	}{{"133x44", 133, 44}, {"80x24", 80, 24}} {
		d := setupBench(t, size.w, size.h)
		_ = d.View().Content // warm the kit's probes and the theme
		hit := testing.AllocsPerRun(20, func() { _ = d.View().Content })
		miss := testing.AllocsPerRun(20, func() { d.mmemo.ok = false; _ = d.View().Content })
		hitBudget, missBudget := setupAllocBudget[size.name+"-hit"], setupAllocBudget[size.name+"-miss"]
		t.Logf("setup %s: hit %.0f allocs (budget %.0f) · miss %.0f allocs (budget %.0f)", size.name, hit, hitBudget, miss, missBudget)
		if hit > hitBudget {
			t.Errorf("setup %s (memo hit) allocates %.0f per View(), budget %.0f", size.name, hit, hitBudget)
		}
		if miss > missBudget {
			t.Errorf("setup %s (memo miss) allocates %.0f per View(), budget %.0f", size.name, miss, missBudget)
		}
	}
}
