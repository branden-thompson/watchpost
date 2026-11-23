package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/internal/allocpin"
	"github.com/branden-thompson/watchpost/modes/tty/internal/frame"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// beat is one message through the Router and the frame Bubble Tea then draws:
// what the app does on every tick and every visualiser frame.
type beat struct {
	name  string
	model func(testing.TB) tea.Model
	msg   tea.Msg
}

// beats are the three 0.19.5's M2 measures (0.19.5 D-4, D-22): the idle
// Observer's 300 ms tick, the radio playing with the visualiser on, and the
// console's tick at 160x74; and the idle tick with Settings open, which
// 0.19.6 M1 adds (A-31) because the map's seam derives a window's sizes from
// what is open.
func beats() []beat {
	return []beat{
		{"idle Observer 133x44", func(tb testing.TB) tea.Model { return NewRouter(benchDash(tb, 133, 44).(Dashboard)) }, frame.TickMsg{}},
		{"idle Observer 133x44 with Settings open", func(tb testing.TB) tea.Model { return NewRouter(setupBench(tb, 133, 44)) }, frame.TickMsg{}},
		{"radio playing 133x44", func(tb testing.TB) tea.Model { return NewRouter(playingDash(tb, 133, 44)) }, vizTickMsg{}},
		{"console 160x74", func(tb testing.TB) tea.Model {
			r := NewRouter(benchDash(tb, 160, 74).(Dashboard))
			r.broadcaster = loadedConsoleOf(tb, Opening{}, ttytest.LoadedPoolSize)
			r.active = SurfaceBroadcaster
			return r
		}, frame.TickMsg{}},
	}
}

// beatCost is a beat's allocations and bytes, the mean over 100 beats.
//
// Each beat's model is fed back in, as Bubble Tea does, so every beat draws
// the frame after the one before. Measured on one model over and over, each
// beat would draw the same frame, and the work that runs only on a changed
// frame would go uncounted: the marquee moves on every tick, and the cursor's
// cell is painted again on every new frame (16 KB at 133x44).
//
// THE COUNT OF BEATS IS FIXED, so every run averages the same frames.
// testing.Benchmark picks its count from the machine's speed, and with it
// which frames the mean covers: the beat with Settings open moved 0.4 % in
// bytes between runs measured that way, past the 0.1 % tools/pins allows.
func beatCost(tb testing.TB, b beat) (allocs, bytes int64) {
	m := b.model(tb)
	_ = m.View().Content // warm the memos and the theme
	step := func() {
		m, _ = m.Update(b.msg)
		_ = m.View().Content
	}
	c := allocpin.MeasureEach(tb, 100, func() func() { return step })
	allocpin.Report(tb, "beat "+b.name, c)
	return int64(c.Allocs), int64(c.Bytes)
}

// beatBudget is each beat's pin: allocations and bytes, measured × 1.05. The
// bytes are pinned as well as the count, because the costliest parts of a
// beat (a copy of the model into an interface, a rune slice of the tape) are
// one allocation each and many kilobytes (0.19.5 PF-6). The batches that cut a
// beat lower its pin to what lands; M2's targets are 80, 90 and 140 KB.
//
// THE BEAT WITH SETTINGS OPEN COSTS SIXTEEN TIMES THE IDLE ONE, 2,642
// allocations and 870,677 bytes at 0.19.6 B0: its frame lays every tab's groups
// out to measure the window's width, which is most of the window's own pin on
// a memo hit (setupAllocBudget). Recorded, not cut (0.16.0 D-53): 0.19.6
// moves code and changes no cost.
var beatBudget = map[string]struct{ allocs, bytes float64 }{
	"idle Observer 133x44":                    {168 * 1.05, 75_990 * 1.05},
	"radio playing 133x44":                    {327 * 1.05, 68_475 * 1.05},
	"console 160x74":                          {377 * 1.05, 122_387 * 1.05},
	"idle Observer 133x44 with Settings open": {2_642 * 1.05, 870_677 * 1.05},
}

// TestBeatAllocBudget is 0.19.5 Q2.12 (PF-6): what a beat costs, the message
// through the Router and the frame after it, for the three beats M2 measures.
// The frame pins above count only View, and only allocations.
func TestBeatAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	for _, b := range beats() {
		budget, ok := beatBudget[b.name]
		if !ok {
			t.Fatalf("beat %q has no budget", b.name)
		}
		allocs, bytes := beatCost(t, b)
		t.Logf("%s: %d allocs (budget %.0f), %d bytes (budget %.0f)", b.name, allocs, budget.allocs, bytes, budget.bytes)
		if float64(allocs) > budget.allocs || float64(bytes) > budget.bytes {
			t.Errorf("%s costs %d allocations and %d bytes a beat; its budget is %.0f and %.0f", b.name, allocs, bytes, budget.allocs, budget.bytes)
		}
	}
}

// TestPublishAllocBudget is 0.19.5 Q2.12 (A-15, DV-11): the frame after a
// publish that changes nothing shown. The publish is a fresh snapshot over a
// fresh copy of the locations, as the assembler makes one, so the measurement
// carries the comparison that keeps the tables from being drawn again (D-31).
func TestPublishAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := benchDash(t, 133, 44).(Dashboard)
	_ = d.View().Content
	// The copy is built once, so its cost is not counted as the publish's.
	// Each publish is still a snapshot of its own, as the assembler sends: a
	// memo that answered by pointer would hit on a snapshot it had seen.
	again := republished(t, d.snap, nil)
	r := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			publish := *again
			m, _ := d.Update(SnapshotMsg{Snap: &publish})
			_ = m.View().Content
		}
	})
	allocpin.PinBenchmark(t, "publish no-change frame 133x44", r)
	const allocs, bytes = 203 * 1.05, 82_322 * 1.05
	t.Logf("a no-change publish's frame: %d allocs (budget %.0f), %d bytes (budget %.0f)", r.AllocsPerOp(), allocs, r.AllocedBytesPerOp(), bytes)
	if float64(r.AllocsPerOp()) > allocs || float64(r.AllocedBytesPerOp()) > bytes {
		t.Errorf("a no-change publish's frame costs %d allocations and %d bytes; its budget is %.0f and %.0f", r.AllocsPerOp(), r.AllocedBytesPerOp(), allocs, bytes)
	}
}

// TestUpdateWithTheMapOpenAllocBudget is 0.19.6 M1's Update pin (A-31): the
// idle tick through the Router with the map window open, the message alone
// and no frame, each model fed back. The map's seam builds its Env on every
// message, so a size derived there, or a window laid out to answer a tick,
// shows here before any frame is drawn.
func TestUpdateWithTheMapOpenAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	var m tea.Model = NewRouter(mapBench(t))
	_ = m.View().Content
	r := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			m, _ = m.Update(frame.TickMsg{})
		}
	})
	c := allocpin.PinBenchmark(t, "Update with the map open 133x44", r)
	const allocs, bytes = 15 * 1.05, 9_512 * 1.05
	if float64(c.Allocs) > allocs || float64(c.Bytes) > bytes {
		t.Errorf("a tick through the Router with the map open costs %d allocations and %d bytes; its budget is %.0f and %.0f", c.Allocs, c.Bytes, allocs, bytes)
	}
}
