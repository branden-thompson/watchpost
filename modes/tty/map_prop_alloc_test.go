package tty

// map_prop_alloc_test.go — the Propagation mode's frame pinned as the
// weather modes' is (FR-8.3): a memo hit, a miss, colour off and on, and a
// draw.

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// propAllocBudget pins the Propagation window over the 133×44 fixture, its
// block under the map drawn: measured and set at × 1.05, as the other pins
// are. D-194's window, inset to inset and 85% wide, draws more map rows than
// U1-13's, and every row costs its allocations.
var propAllocBudget = map[string]float64{"hit": 1_270 * 1.05, "miss": 1_733 * 1.05, "hit colour": 1_751 * 1.05, "miss colour": 3_409 * 1.05}

// propBench is the map in the Propagation mode, a snapshot drawn, its clock
// pinned so the age holds still, every piece of work landed.
func propBench(tb testing.TB) Dashboard {
	tb.Helper()
	t := tb
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	s := &seam{answer: PropagationResult{Snapshot: propSnapAt(at, 18)}}
	var asked []string
	d := openMap(t, Config{PropagationUpdate: s.update, PropagationAck: propAckVersion, MapRadar: radarFeed(t, "MRMS", &asked)}, 133, 44)
	d.now = func() time.Time { return at.Add(7 * time.Minute) }
	d, cmd := keyCmd(t, d, "P")
	d = runProp(t, d, cmd)
	for cmd := d.mapWorkCmd(); cmd != nil; cmd = d.mapWorkCmd() {
		m, _ := d.Update(cmd())
		d = m.(Dashboard)
	}
	return d
}

func BenchmarkFrame_133x44_MapPropagation(b *testing.B) {
	d := propBench(b)
	b.ReportAllocs()
	for b.Loop() {
		d.mmemo.ok = false
		_ = d.View().Content
	}
}

func BenchmarkPropRenderMap(b *testing.B) {
	d := propBench(b)
	b.ReportAllocs()
	for b.Loop() {
		_ = d.renderMap()
	}
}

func TestPropagationFrameAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := propBench(t)
	if !strings.Contains(stripANSITest(d.View().Content), "MODE:") {
		t.Fatal("the fixture drew no Propagation block: this measures nothing")
	}
	for _, colour := range []bool{false, true} {
		rendering.SetColorEnabledForTest(colour)
		name := map[bool]string{false: "", true: " colour"}[colour]
		calls := &[]string{}
		d.mapPane.calls = calls
		hit := testing.AllocsPerRun(20, func() { _ = d.View().Content })
		miss := testing.AllocsPerRun(20, func() { d.mmemo.ok = false; _ = d.View().Content })
		if len(*calls) != 0 {
			t.Errorf("a frame called the library: %v", *calls)
		}
		t.Logf("Propagation window%s: hit %.0f allocs (budget %.0f) · miss %.0f allocs (budget %.0f)", name, hit, propAllocBudget["hit"+name], miss, propAllocBudget["miss"+name])
		if hit > propAllocBudget["hit"+name] {
			t.Errorf("the Propagation window%s (memo hit) allocates %.0f per View(), budget %.0f", name, hit, propAllocBudget["hit"+name])
		}
		if miss > propAllocBudget["miss"+name] {
			t.Errorf("the Propagation window%s (memo miss) allocates %.0f per View(), budget %.0f", name, miss, propAllocBudget["miss"+name])
		}
	}
	rendering.SetColorEnabledForTest(false)
}
