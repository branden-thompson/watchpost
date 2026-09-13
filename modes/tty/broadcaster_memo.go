package tty

// broadcaster_memo.go — the console's table memo, Observer's `bodyMemo` one
// surface along.
//
// THE TWO TABLES ARE 95% OF THE FRAME'S ALLOCATIONS AND 60% OF ITS TIME —
// measured on a loaded console before this was written (D-53): 320 µs and 8796
// allocs out of 537 µs and 9298. They change only when one of their inputs does,
// so they are built once per input change and reused on every tick, clock and
// bed-step frame between.
//
// THE KEY IS COMPLETE BY CONSTRUCTION — one field per input the tables read —
// and the guard that makes that true is `broadcaster_memo_completeness_test.go`,
// which DERIVES the fields from the struct rather than listing them. F-30 is why:
// a hand-kept key froze three of Observer's windows in one release while the
// model underneath worked perfectly, and the first took three UAT rounds to find.
//
// A MEMO MAY MISS. IT MUST NEVER WRONGLY HIT. Every choice here takes the miss.

import (
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// consoleKey is every input the running order and the pool read.
//
// THE GENERATIONS ARE FOR THE TWO INPUTS THAT CANNOT BE COMPARED. `lineup.Lineup`
// holds an array of slices and `StationAreaMsg` holds a slice, so neither is a
// `==` type — and hashing either every frame would spend the saving on the
// comparison. Both are assigned WHOLESALE from a message and never mutated in
// place, so a counter bumped by the writer is exact, not approximate.
type consoleKey struct {
	width, height int
	// used is how many rows the frame had already drawn when it asked. It sets
	// the pool's room and the running order's window, so a frame that grew a row
	// above the tables draws different tables.
	used  int
	ascii bool
	// power decides where the line-up is read FROM: on standby it draws from UP
	// NEXT down, which moves every slot by one (D-84, liveOffset).
	power      lineup.Power
	fireBoldMW float64
	// selected drives BOTH the focused row and the pool's scroll window — the
	// window is derived from the selection (poolSpan), not stored beside it.
	selected int
	// recent is the weather behind both tables (D-99). A POINTER, as Observer
	// keys them: a publish hands over a whole new snapshot rather than editing
	// one, so the pointer moving IS the data changing.
	//
	// `snap` IS NOT HERE, AND WAS. The PRIORITY snapshot feeds the masthead's
	// stamp and its API summary — `b.snap` has exactly two readers and neither is
	// a table. Keyed on it, every priority publish threw away a cached pair that
	// was still correct. The guard is what said so: dropping it from the key
	// changed no table, which for a key field means it was never an input.
	recent    *snapshot.Snapshot
	lineupGen uint64
	areaGen   uint64
	// theme is render.ThemeGeneration: every Tok() tint in every cell.
	theme uint64
	// shimmer is the LoadingDots phase, and ONLY while something is loading —
	// carried unconditionally it would miss on every tick for the whole life of
	// the console, which is every frame this exists to save.
	shimmer int
}

// consoleMemo is the single slot, bound to one entry: the last key.
type consoleMemo struct {
	memoStats
	ok          bool
	key         consoleKey
	sched, pool scrollSpan
}

// consoleKeyFor derives the key from the model and this frame's position.
func (b Broadcaster) consoleKeyFor(used int) consoleKey {
	k := consoleKey{
		width: b.width, height: b.height, used: used, ascii: b.ascii,
		power: b.power, fireBoldMW: b.fireBold(), selected: b.selected,
		recent: b.pool, lineupGen: b.lineupGen, areaGen: b.areaGen,
		theme: render.ThemeGeneration(),
	}
	if b.anyLoading() {
		k.shimmer = ((b.frame % 4) + 4) % 4 // the LoadingDots phase (render/units.go)
	}
	return k
}

// spans builds the running order and the pool, or returns the memoised pair.
//
// IT OWNS THE INDEX TOO, and that is deliberate. `locIndex` builds a map over the
// whole pool so the forty joins are not forty linear scans (D-120) — real work,
// and pointless on a hit. Taking it inside means a hit costs the key comparison
// and nothing else; leaving it outside would have paid for the index on every
// frame to save the tables on most of them.
func (b Broadcaster) spans(used int) (sched, pool scrollSpan) {
	m := b.memo
	if m == nil {
		return b.buildSpans(used)
	}
	key := b.consoleKeyFor(used)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ok && m.key == key {
		m.hits++
		return m.sched, m.pool
	}
	sched, pool = b.buildSpans(used)
	m.ok, m.key, m.sched, m.pool = true, key, sched, pool
	m.misses++
	return sched, pool
}

// buildSpans is the miss path: the index, then both tables against it.
func (b Broadcaster) buildSpans(used int) (scrollSpan, scrollSpan) {
	idx := b.locIndex()
	sched := b.scheduledSpan(b.mainTrack(), used, idx)
	return sched, b.poolSpan(used+len(sched.lines), idx)
}

// stats is the nil hop from the slot to its counters, the third of its kind
// beside bodyMemo's and modalMemo's.
//
// IT CANNOT BE INHERITED FROM THE EMBEDDED TYPE, which is why there are three:
// the guard is on the OUTER pointer, and a method promoted from `memoStats`
// would have dereferenced the nil slot to reach itself. Same shape, different
// receiver — the mutex-read accessor the HUM LEAD ratified on 2026-09-13, one
// package along.
func (m *consoleMemo) stats() *memoStats {
	if m == nil {
		return nil
	}
	return &m.memoStats
}

// consoleMemoCounts reports the slot's hit/miss counters (0, 0 without a slot).
func (b Broadcaster) consoleMemoCounts() (hits, misses int) { return b.memo.stats().counts() }

// anyLoading reports whether any location either table draws is still waiting on
// its weather — the shimmer's reason to animate, and so the key's reason to
// carry the frame.
//
// IT ASKS THE POOL'S SNAPSHOT, WHICH IS WHERE BOTH TABLES GET THEIR WEATHER
// (D-99). The running order joins the same locations by key, so a place that
// shimmers in one shimmers in the other.
func (b Broadcaster) anyLoading() bool {
	if b.pool == nil {
		return false
	}
	for i := range b.pool.Locations { // bounded by the snapshot (P10-02)
		if rowLoading(&b.pool.Locations[i]) {
			return true
		}
	}
	return false
}

// joinSpans lays the pool's rows under the running order's.
//
// A SLICE OF ITS OWN, NEVER APPENDED ONTO THE SCHEDULED SPAN. Both spans may be
// the MEMO'S COPIES, and `append(sched.lines, …)` writes into `sched.lines`'
// spare capacity whenever it has any — scribbling the pool's rows into a cached
// running order, to be replayed on every hit afterwards. It would appear one
// frame late and only at some pool sizes, which is close to the worst shape a
// defect can have.
//
// IT IS A FUNCTION SO THAT THE RULE CAN BE TESTED. Written inline, the only way
// to catch a regression was to hope the aliasing happened to become visible;
// here the property — "the result does not share an array with the cache" — is a
// thing a test can assert directly, whether or not today's capacities expose it.
func joinSpans(sched, pool scrollSpan) []string {
	out := make([]string, 0, len(sched.lines)+len(pool.lines))
	out = append(out, sched.lines...)
	return append(out, pool.lines...)
}
