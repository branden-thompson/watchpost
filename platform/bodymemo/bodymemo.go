// Package bodymemo caches a PARSE, keyed by whatever the caller fetches by and
// revalidated by the body's own hash.
//
// WHY IT EXISTS (F-53). domains/fire/firms keys it by (source, tile) over
// parsed FIRMS points, and domains/seismic/usgs by URL over decoded GeoJSON
// features — the same tick counter, the same hash revalidation, the same
// least-recently-used eviction. Two implementations of one operation is a
// defect that has not happened yet: the day one is corrected and the other is
// not, they disagree and both look right in isolation.
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

	"github.com/branden-thompson/watchpost/platform/invariant"
	"sync"
)

// Memo is a bounded, hash-revalidated parse cache. The zero value is not
// usable; call New.
type Memo[K comparable, V any] struct {
	mu        sync.Mutex
	tick      uint64
	max       int
	items     map[K]*entry[V]
	parses    int
	keepsErrs bool // a failed parse is kept with its value, and answered again for the same body
}

type entry[V any] struct {
	sum  [sha256.Size]byte
	val  V
	err  error
	used uint64
}

// New builds a memo holding at most max entries.
//
// A CAP BELOW ONE IS CLAMPED TO ONE AND NOT REPORTED. Returning nil would move a
// construction-time mistake into a nil dereference at the first read, and a memo
// is built at start-up where nothing is watching; there is no error channel here
// to name it on. The bound that CAN break — that the memo never exceeds max — is
// checked in `Parsed`, against the clamped value, which is the only value that
// governs eviction.
func New[K comparable, V any](max int) *Memo[K, V] {
	if max < 1 {
		// THE CLAMP IS ALL THERE IS, AND SAYING SO IS THE POINT. `platform/invariant`
		// is SIDE-EFFECT-FREE — `Check` builds an error and the CALLER'S return is the
		// recovery — and this function returns no error, so an
		// `invariant.Check(false, …)` here would have no observable effect. A check
		// that satisfies a density metric and produces no observable effect is the
		// proxy-gate pattern this codebase treats as a defect.
		//
		// RETURNING AN ERROR INSTEAD IS NOT TAKEN: a memo is built at start-up where
		// nothing is watching, and turning a sizing mistake into a nil dereference at
		// the first read is worse than clamping. The bound that MATTERS — that the
		// memo never exceeds max — is checked in `Parsed`, where it can actually break.
		max = 1
	}
	return &Memo[K, V]{max: max, items: make(map[K]*entry[V], 64)}
}

// NewKeepingErrors builds a memo that keeps a failed parse too: the value and
// the error the parse gave are answered again for the same body, so a bad body
// is not parsed again for the rest of its cache life - the caller forgets it
// at the cache - and a soft error's value comes back with it.
func NewKeepingErrors[K comparable, V any](max int) *Memo[K, V] {
	m := New[K, V](max)
	m.keepsErrs = true
	return m
}

// Last is the last parse kept for k, whatever body comes next: for a caller
// that knows the body unchanged without hashing it, or that reads what it
// holds without fetching. ok is false before a parse was kept.
func (m *Memo[K, V]) Last(k K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.items[k]
	if !ok {
		var zero V
		return zero, false
	}
	return e.val, true
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
		return e.val, e.err
	}
	val, err := parse(raw)
	if err != nil && !m.keepsErrs {
		var zero V
		return zero, err
	}
	m.parses++
	if _, ok := m.items[k]; !ok && len(m.items) >= m.max {
		m.evictLocked()
	}
	m.items[k] = &entry[V]{sum: sum, val: val, err: err, used: m.tick}
	// THE BOUND IS THIS PACKAGE'S THIRD RULE, CHECKED WHERE IT CAN BREAK (P10-05).
	// OQ-9 states it in the package doc — "It is bounded. At most max entries,
	// least-recently-used out" — and this asserts it: the eviction is
	// conditional on a miss, so the day that condition is wrong the memo grows
	// without bound and every observable (a hit returns what a parse would,
	// parses counts misses) goes on reading correct.
	//
	// ON THE MISS PATH ONLY, deliberately. Rule 2 is that a HIT does not parse,
	// and the providers' allocation pins hold a hit at ZERO — so a check on the
	// hit path would be measured by those pins rather than by this package.
	// A miss has already parsed and allocated; this costs nothing it did not
	// already spend.
	if bound := invariant.Check(len(m.items) <= m.max, "the memo holds at most max entries"); bound != nil {
		return val, bound
	}
	return val, err
}

// evictLocked drops the least-recently-used entry (caller holds mu).
func (m *Memo[K, V]) evictLocked() {
	// AN EVICTION WITH NOTHING TO EVICT DELETES THE ZERO KEY, which is silent:
	// `delete` on an absent key is a no-op, so an empty memo would "evict"
	// for ever and the caller's bound would never be reached.
	if len(m.items) == 0 {
		return
	}
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
