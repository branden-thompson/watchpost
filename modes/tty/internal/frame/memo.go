package frame

// memo.go — the counters every memo slot carries, with the lock the slot
// takes around its lookup (0.19.6 A-21).

import "sync"

// MemoStats is the counter half every memo slot carries. ONE OWNER (metric D,
// 2026-09-08): bodyMemo and modalMemo embed it rather than each carrying its own
// nil-check-lock-return accessor, so the next memo inherits `counts()` instead
// of copying it.
//
// The slot's lock is the embedded mutex: a slot locks it around its lookup,
// and Hit and Miss count with it held (0.19.6 A-21).
type MemoStats struct {
	sync.Mutex
	hits, misses int
}

// Hit counts a lookup the slot answered. The caller holds the lock.
func (m *MemoStats) Hit() { m.hits++ }

// Miss counts a lookup the slot had to build. The caller holds the lock.
func (m *MemoStats) Miss() { m.misses++ }

// Counts reports the slot's hit/miss counters; the zero value of a nil slot is
// (0, 0), which is what both callers expect.
func (m *MemoStats) Counts() (hits, misses int) {
	if m == nil {
		return 0, 0
	}
	m.Lock()
	defer m.Unlock()
	return m.hits, m.misses
}
