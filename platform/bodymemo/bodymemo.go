// Package bodymemo caches a PARSE, keyed by whatever the caller fetches by and
// revalidated by the body's own hash.
//
// WHY IT EXISTS. Two providers had written this: domains/fire/firms keyed by
// (source, tile) over parsed FIRMS points, and domains/seismic/usgs keyed by
// URL over decoded GeoJSON features. F-53 records them as byte-identical in
// their stats and the same shape around it — the same tick counter, the same
// hash revalidation, the same least-recently-used eviction, written twice. Two
// implementations of one operation is a defect that has not happened yet: the
// day one is corrected and the other is not, they disagree and both look right
// in isolation.
//
// WHAT A MEMO OWES, ruled at OQ-9 (HUM LEAD, 2026-09-07):
//
//  1. A hit returns what a parse would. Revalidation by body hash is how it
//     earns that: a changed body MISSES.
//  2. A hit does not parse. Parses is the observable, and the providers' own
//     allocation pins hold a hit at zero.
//  3. It is bounded. At most max entries, least-recently-used out.
//
// And the non-property that shapes all three: a memo may MISS spuriously —
// that costs time, not correctness — but it must NEVER wrongly HIT. The same
// asymmetry the frame memo's key guard is built on, one layer down.
package bodymemo

import (
	"crypto/sha256"
	"math"
	"sync"
)

// Memo is a bounded, hash-revalidated parse cache. The zero value is not
// usable; call New.
type Memo[K comparable, V any] struct {
	mu     sync.Mutex
	tick   uint64
	max    int
	items  map[K]*entry[V]
	parses int
}

type entry[V any] struct {
	sum  [sha256.Size]byte
	val  V
	used uint64
}

// New builds a memo holding at most max entries.
func New[K comparable, V any](max int) *Memo[K, V] {
	if max < 1 {
		max = 1
	}
	return &Memo[K, V]{max: max, items: make(map[K]*entry[V], 64)}
}

// Parsed is the value for this key and body, parsing only when the body has
// changed since it was last seen.
//
// parse IS A TOP-LEVEL FUNCTION AT BOTH CALL SITES, deliberately: a closure
// that captures anything allocates on every call, including the hits this
// exists to make free, and the providers' pins hold a hit at zero allocations.
func (m *Memo[K, V]) Parsed(k K, raw []byte, parse func([]byte) (V, error)) (V, error) {
	sum := sha256.Sum256(raw)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tick++
	if e, ok := m.items[k]; ok && e.sum == sum {
		e.used = m.tick
		return e.val, nil
	}
	val, err := parse(raw)
	if err != nil {
		var zero V
		return zero, err
	}
	m.parses++
	if _, ok := m.items[k]; !ok && len(m.items) >= m.max {
		m.evictLocked()
	}
	m.items[k] = &entry[V]{sum: sum, val: val, used: m.tick}
	return val, nil
}

// evictLocked drops the least-recently-used entry (caller holds mu).
func (m *Memo[K, V]) evictLocked() {
	var victim K
	oldest := uint64(math.MaxUint64)
	for k, e := range m.items {
		if e.used < oldest {
			victim, oldest = k, e.used
		}
	}
	delete(m.items, victim)
}

// Prune drops every entry the caller no longer considers live.
//
// THE SECOND WAY A MEMO IS BOUNDED, and both are real. The tile and box memos
// are bounded by COUNT — at most max, least-recently-used out — because their
// keys are geometry and there is no outside authority on which ones matter. The
// NWS grid memo is bounded by LIVENESS: its keys are the grid URLs of watched
// locations, and when a location leaves the watchlist its grid stops existing
// as far as the app is concerned. Forcing that caller onto an LRU would keep a
// removed location's grid until 240 others pushed it out.
//
// keep runs under the memo's lock, so it must not call back into anything that
// takes it.
func (m *Memo[K, V]) Prune(keep func(K) bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.items {
		if !keep(k) {
			delete(m.items, k)
		}
	}
}

// Stats is the memo's live size and the parses since it was made — the gauge a
// bounded size and a low parse rate are read from.
func (m *Memo[K, V]) Stats() (entries, parses int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.items), m.parses
}
