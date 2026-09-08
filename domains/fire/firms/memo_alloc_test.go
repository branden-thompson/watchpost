package firms

// memo_alloc_test.go — B4's instrument, built BEFORE the change it measures.
//
// THE PIN IS WHERE THE MEMO IS. The plan's first revision named
// `make alloc-budget` as this batch's instrument; all four of its tests are
// FRAME pins in modes/tty, and this memo is on the provider FETCH path, which
// no frame draws. A frame pin would have returned a green false pass for a
// change the frame cannot see — the same error, one layer up, as a test that
// enters the system past the seam the user does (quality-observations 16).
//
// WHAT IT MEASURES: the memo's whole reason for existing — a HIT. A hit
// revalidates by body hash, finds the entry and returns the parsed points; it
// must not parse, and it must not allocate in proportion to the body. The
// budget is the measurement × 1.05, the house rule.

import "testing"

// ZERO, AND THE HOUSE RULE PUTS IT THERE: the budget is the measurement × 1.05,
// the measurement is 0, and a hit that allocates nothing is what a hash-keyed
// memo over an already-parsed value should cost. It is also the number the
// consolidation must not move — a generic memo taking a parse closure is the
// obvious way to allocate one per call and never notice.
const tileMemoHitAllocBudget float64 = 0

func TestTileMemoHitAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	m := newTileMemo()
	k := tileKey{src: "a", tile: tile{x: 1, y: 0, pitch: tileDeg}}
	raw := []byte(csvBody)
	if _, err := m.points(k, raw); err != nil {
		t.Fatal(err)
	}
	// THE FIXTURE MUST HIT. Measured on a miss this pins the PARSER, which is a
	// different thing that is allowed to allocate.
	_, before := m.stats()
	got := testing.AllocsPerRun(50, func() {
		if _, err := m.points(k, raw); err != nil {
			t.Fatal(err)
		}
	})
	if _, after := m.stats(); after != before {
		t.Fatalf("the fixture parsed %d more times: it is missing, not hitting", after-before)
	}
	t.Logf("tileMemo hit: %.0f allocs (budget %.0f)", got, tileMemoHitAllocBudget)
	if got > tileMemoHitAllocBudget {
		t.Errorf("a tile-memo HIT allocates %.0f, budget %.0f — the memo exists to make this "+
			"cheaper than the parse it replaces", got, tileMemoHitAllocBudget)
	}
}
