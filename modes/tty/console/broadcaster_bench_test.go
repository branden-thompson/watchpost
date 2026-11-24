package console

// broadcaster_bench_test.go — the console's frame, timed.

import (
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// BenchmarkConsoleFrame is the console's frame on the path that COSTS: a full
// pool, a full running order, and real snapshots behind both, so every row
// joins its weather (0.16.0 D-120). A console with no pool measures the cheap
// path, which is the mistake an alloc budget most easily makes.
func BenchmarkConsoleFrame(b *testing.B) {
	c := loadedConsole(b, ttytest.LoadedPoolSize)
	if loadedJoins(c) == 0 {
		b.Fatal("the fixture joined no weather; this benchmark is about the cheap path")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = c.View().Content
	}
}

// BenchmarkConsoleFrame_Miss is the memo's OTHER path, and it is the one a
// hit-only number would hide.
//
// THE OPERATOR HOLDING AN ARROW KEY IS THE REAL WORST CASE: the slot holds ONE
// entry, so a selection that alternates never finds it. Every frame here rebuilds
// both tables and the location index — which is what the console costs on every
// frame without the memo, so this number says how much the memo can cost when it
// never helps.
func BenchmarkConsoleFrame_Miss(b *testing.B) {
	c := loadedConsole(b, ttytest.LoadedPoolSize)
	if loadedJoins(c) == 0 {
		b.Fatal("the fixture joined no weather; this benchmark is about the cheap path")
	}
	b.ReportAllocs()
	b.ResetTimer()
	i := 0
	for b.Loop() {
		c.selected = i & 1
		_ = c.View().Content
		i++
	}
}
