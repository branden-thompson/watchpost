// Package agememo keeps values by key for an age: what a source answered,
// answered again while it is fresh, asked once however many ask together, and
// standing in when a refresh fails while it is not too old (W14 S-12, D-212).
//
// It is the one shape of the map's keyed, time-bounded memos - an area's
// alerts, a day's station lists, an archive's coalesced parse - so their
// rules are written once. `platform/bodymemo` is the other half: a parse kept
// by its body's hash, where this keeps a value by its age.
package agememo

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// errFetchPanicked is what callers waiting on a fetch get when it panicked.
var errFetchPanicked = errors.New("agememo: the fetch panicked")

// Options are a memo's rules.
type Options struct {
	// Fresh is how long a value answers without asking again.
	Fresh time.Duration
	// StandIn is how old a value may be and still answer when a refresh
	// fails; zero, a failure is answered as a failure.
	StandIn time.Duration
	// Max is how many keys are held, the least recently used out; one when
	// unset.
	Max int
	// Now is the clock; the wall clock when nil.
	Now func() time.Time
}

// Memo keeps values by key for an age. It is safe for concurrent use.
type Memo[K comparable, V any] struct {
	o      Options
	mu     sync.Mutex
	tick   uint64
	items  map[K]*entry[V]
	flying map[K]*call[V]
}

type entry[V any] struct {
	val  V
	at   time.Time
	used uint64
}

// call is a fetch in flight: the callers that ask while it runs wait for it.
type call[V any] struct {
	done chan struct{}
	val  V
	err  error
}

// New builds a memo with the rules o.
func New[K comparable, V any](o Options) *Memo[K, V] {
	o.Max = max(o.Max, 1)
	if o.Now == nil {
		o.Now = time.Now
	}
	return &Memo[K, V]{o: o, items: map[K]*entry[V]{}, flying: map[K]*call[V]{}}
}

// Get is k's value while it is fresh.
func (m *Memo[K, V]) Get(k K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.freshLocked(k); ok {
		return e.val, true
	}
	var zero V
	return zero, false
}

// Last is k's value whatever its age, and when it was stored.
func (m *Memo[K, V]) Last(k K) (V, time.Time, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.items[k]
	if !ok {
		var zero V
		return zero, time.Time{}, false
	}
	return e.val, e.at, true
}

// Put stores v as k's value now.
func (m *Memo[K, V]) Put(k K, v V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storeLocked(k, v)
}

// Forget drops k's value.
func (m *Memo[K, V]) Forget(k K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, k)
}

// Do is k's value while it is fresh, else fetch's - one fetch a key at a
// time, its answer shared by the callers that asked while it ran; a caller
// waiting on another's fetch gives up when its own ctx ends. A failed fetch
// is answered by the last value while that is younger than StandIn, else by
// the failure; a failure is never kept.
func (m *Memo[K, V]) Do(ctx context.Context, k K, fetch func() (V, error)) (V, error) {
	var zero V
	if err := invariant.Check(fetch != nil, "agememo: Do needs a fetch"); err != nil {
		return zero, err
	}
	// A STAND-IN UNDER THE FRESHNESS NEVER STANDS IN: a value is only asked
	// again once it is older than Fresh, so it is already too old for StandIn.
	if err := invariant.Check(m.o.StandIn == 0 || m.o.StandIn >= m.o.Fresh, "agememo: StandIn is shorter than Fresh"); err != nil {
		return zero, err
	}
	m.mu.Lock()
	if e, ok := m.freshLocked(k); ok {
		m.mu.Unlock()
		return e.val, nil
	}
	if c, ok := m.flying[k]; ok {
		m.mu.Unlock()
		select {
		case <-c.done:
			return c.val, c.err
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}
	c := &call[V]{done: make(chan struct{}), err: errFetchPanicked} // what the waiting callers get if fetch never returns
	m.flying[k] = c
	m.mu.Unlock()
	defer m.land(k, c) // also when fetch panics: the callers waiting are let go
	c.val, c.err = fetch()
	return m.answer(k, c)
}

// answer settles a finished fetch under the memo's lock: a value is kept; a
// failure stands on the last value while it is young enough.
func (m *Memo[K, V]) answer(k K, c *call[V]) (V, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.err == nil {
		m.storeLocked(k, c.val)
		return c.val, nil
	}
	if e, ok := m.items[k]; ok && m.o.StandIn > 0 && m.o.Now().Sub(e.at) < m.o.StandIn {
		c.val, c.err = e.val, nil
	}
	return c.val, c.err
}

// land ends a fetch: it is no longer in flight, and its callers are let go.
func (m *Memo[K, V]) land(k K, c *call[V]) {
	m.mu.Lock()
	delete(m.flying, k)
	m.mu.Unlock()
	close(c.done)
}

// freshLocked is k's entry while it is fresh, marked used.
func (m *Memo[K, V]) freshLocked(k K) (*entry[V], bool) {
	e, ok := m.items[k]
	if !ok || m.o.Now().Sub(e.at) >= m.o.Fresh {
		return nil, false
	}
	m.tick++
	e.used = m.tick
	return e, true
}

// storeLocked keeps v as k's value, dropping the least recently used key
// when the memo is full.
func (m *Memo[K, V]) storeLocked(k K, v V) {
	m.tick++
	if _, ok := m.items[k]; !ok && len(m.items) >= m.o.Max {
		m.evictLocked()
	}
	m.items[k] = &entry[V]{val: v, at: m.o.Now(), used: m.tick}
	// THE BOUND, CHECKED WHERE IT CAN BREAK: eviction is conditional on a new
	// key, and the day that condition is wrong the memo grows without limit
	// while every answer it gives stays right.
	if err := invariant.Check(len(m.items) <= m.o.Max, "the memo holds at most Max keys"); err != nil {
		m.evictLocked()
	}
}

// evictLocked drops the least recently used key.
func (m *Memo[K, V]) evictLocked() {
	var victim K
	oldest, found := uint64(0), false
	for k, e := range m.items { // at most Max (P10-02)
		if !found || e.used < oldest {
			victim, oldest, found = k, e.used, true
		}
	}
	if found {
		delete(m.items, victim)
	}
}
