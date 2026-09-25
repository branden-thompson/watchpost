package tty

// map_alloc_test.go — 0.18.0 W2.7 (FR-8.3): an allocation pin on the map
// window's frame, and a memo hit that does not re-render.

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"testing"
)

// mapAllocBudget pins the map window over the 133×44 fixture: the modal-memo
// HIT (every frame between events - the marquee's ticks keep drawing it) and
// the MISS (a draw, a key). Measured at W2.7 and set at × 1.05, as the other
// pins are.
//
// The hit is above the severe window's (1,740): the map's braille lines carry
// their colour spans, and the overlay compositor walks every one. Measured and
// recorded, not optimised yet (D-53: function before performance).
var mapAllocBudget = map[string]float64{"hit": 2_508 * 1.05, "miss": 2_919 * 1.05}

// mapBench is the map window open over the benchmark fixture, one alert on
// the map, every piece of work landed.
func mapBench(tb testing.TB) Dashboard {
	tb.Helper()
	d := benchDash(tb, 133, 44).(Dashboard)
	d.cfg.NewMap, d.cfg.MapFeed = embeddedMap, boxFeed(-117.6, -117.1, false)
	m, _ := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = m.(Dashboard)
	if cmd := d.mapFeedCmd(); cmd != nil {
		m, _ = d.Update(cmd())
		d = m.(Dashboard)
	}
	for cmd := d.mapWorkCmd(); cmd != nil; cmd = d.mapWorkCmd() {
		m, _ = d.Update(cmd())
		d = m.(Dashboard)
	}
	return d
}

func BenchmarkFrame_133x44_Map(b *testing.B) {
	d := mapBench(b)
	b.ReportAllocs()
	for b.Loop() {
		_ = d.View().Content
	}
}

func TestMapFrameAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	d := mapBench(t)
	if !strings.ContainsRune(d.View().Content, '⠀') && !strings.ContainsAny(d.View().Content, "⣿⠿") {
		t.Fatal("the fixture drew no map: this measures nothing")
	}
	calls := &[]string{}
	d.mapPane.calls = calls
	hit := testing.AllocsPerRun(20, func() { _ = d.View().Content })
	if len(*calls) != 0 {
		t.Errorf("a memo hit called the library: %v", *calls)
	}
	miss := testing.AllocsPerRun(20, func() { d.mmemo.ok = false; _ = d.View().Content })
	if len(*calls) != 0 {
		t.Errorf("a memo miss re-rendered the map in View: %v", *calls)
	}
	t.Logf("map window: hit %.0f allocs (budget %.0f) · miss %.0f allocs (budget %.0f)", hit, mapAllocBudget["hit"], miss, mapAllocBudget["miss"])
	if hit > mapAllocBudget["hit"] {
		t.Errorf("the map window (memo hit) allocates %.0f per View(), budget %.0f", hit, mapAllocBudget["hit"])
	}
	if miss > mapAllocBudget["miss"] {
		t.Errorf("the map window (memo miss) allocates %.0f per View(), budget %.0f", miss, mapAllocBudget["miss"])
	}
}
