package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// beat is one message through the Router and the frame Bubble Tea then draws:
// what the app does on every tick and every visualiser frame.
type beat struct {
	name  string
	model func(testing.TB) tea.Model
	msg   tea.Msg
}

// beats are the three the brief's M2 measures (0.19.5 D-4, D-22): the idle
// Observer's 300 ms tick, the radio playing with the visualiser on, and the
// console's tick at 160x74.
func beats() []beat {
	return []beat{
		{"idle Observer 133x44", func(tb testing.TB) tea.Model { return NewRouter(benchDash(tb, 133, 44).(Dashboard)) }, tickMsg{}},
		{"radio playing 133x44", func(tb testing.TB) tea.Model { return NewRouter(playingDash(tb, 133, 44)) }, vizTickMsg{}},
		{"console 160x74", func(tb testing.TB) tea.Model {
			r := NewRouter(benchDash(tb, 160, 74).(Dashboard))
			r.broadcaster = loadedConsole(tb, loadedPoolSize)
			r.active = SurfaceBroadcaster
			return r
		}, tickMsg{}},
	}
}

// beatCost is a beat's allocations and bytes, from testing.Benchmark.
func beatCost(tb testing.TB, b beat) (allocs, bytes int64) {
	m := b.model(tb)
	_ = m.View().Content // warm the memos and the theme
	r := testing.Benchmark(func(bb *testing.B) {
		bb.ReportAllocs()
		for bb.Loop() {
			next, _ := m.Update(b.msg)
			_ = next.View().Content
		}
	})
	return r.AllocsPerOp(), r.AllocedBytesPerOp()
}

// beatBudget is each beat's pin: allocations and bytes, measured × 1.05. The
// bytes are pinned as well as the count, because the costliest parts of a
// beat (a copy of the model into an interface, a rune slice of the tape) are
// one allocation each and many kilobytes (0.19.5 PF-6). The batches that cut a
// beat lower its pin to what lands; M2's targets are 80, 90 and 140 KB.
var beatBudget = map[string]struct{ allocs, bytes float64 }{
	"idle Observer 133x44": {401 * 1.05, 104_843 * 1.05},
	"radio playing 133x44": {626 * 1.05, 112_447 * 1.05},
	"console 160x74":       {537 * 1.05, 183_812 * 1.05},
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
// publish that changes nothing shown, which today re-renders both tables. PF-9's
// content generation is measured against it.
func TestPublishAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := benchDash(t, 133, 44).(Dashboard)
	_ = d.View().Content
	r := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			same := *d.snap
			m, _ := d.Update(SnapshotMsg{Snap: &same})
			_ = m.View().Content
		}
	})
	const allocs, bytes = 5_595 * 1.05, 388_674 * 1.05
	t.Logf("a no-change publish's frame: %d allocs (budget %.0f), %d bytes (budget %.0f)", r.AllocsPerOp(), allocs, r.AllocedBytesPerOp(), bytes)
	if float64(r.AllocsPerOp()) > allocs || float64(r.AllocedBytesPerOp()) > bytes {
		t.Errorf("a no-change publish's frame costs %d allocations and %d bytes; its budget is %.0f and %.0f", r.AllocsPerOp(), r.AllocedBytesPerOp(), allocs, bytes)
	}
}
