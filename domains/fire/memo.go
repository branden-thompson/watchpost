package fire

// memo.go — the one-entry parse memo the fire feeds share (quality pass
// Q3, L4-F6, red-team B5 P1): a feed's whole-country body is fetched by
// every location's scheduler, so the parsed form is kept by content hash
// and parsed once per change, whoever asks. Bound: the last body's parse.

import (
	"crypto/sha256"
	"sync"
)

// Memo holds the parse of the last body seen. T is the parsed form.
type Memo[T any] struct {
	mu     sync.Mutex
	ok     bool
	sum    [sha256.Size]byte
	val    T
	err    error
	parses int
}

// Get returns the parse of raw, running parse only when the bytes differ
// from the last call's. A parse error is memoised too: the same bad body
// is not re-parsed for the rest of its cache life (the caller Forgets it).
func (m *Memo[T]) Get(raw []byte, parse func([]byte) (T, error)) (T, error) {
	sum := sha256.Sum256(raw)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ok && sum == m.sum {
		return m.val, m.err
	}
	m.val, m.err = parse(raw)
	m.sum, m.ok = sum, true
	m.parses++
	return m.val, m.err
}

// Peek returns the memoised value without parsing (false before the first Get).
func (m *Memo[T]) Peek() (T, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.val, m.ok
}

// Parses counts the parses run since construction (the diagnostic view:
// one per body change).
func (m *Memo[T]) Parses() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.parses
}
