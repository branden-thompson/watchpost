package usgs

// boxmemo.go — the parsed-box memo (seismic P2, plan §3 Approach B, the Q5
// FIRMS-tile precedent). Many locations share one regional box URL, and a
// box's body rarely changes between ticks; the memo decodes each distinct
// body once, revalidated by hash, so the shared request is also a shared
// parse. Bounded LRU (P10-03) and gauged (the [S] modal).

import (
	"crypto/sha256"
	"math"
	"sync"
)

// maxBoxes bounds the memo: the distinct query URLs the current location set
// can touch — at most one near box per location (≤ 60) plus a handful of
// shared regional boxes. Past the bound the least-recently-used box is
// dropped, so a churny location set cannot grow it without limit.
const maxBoxes = 160

type boxEntry struct {
	sum   [sha256.Size]byte
	feats []feature
	used  uint64
}

// boxMemo holds the parsed features of the boxes most recently fetched, keyed
// by URL and revalidated by body hash.
type boxMemo struct {
	mu     sync.Mutex
	tick   uint64
	items  map[string]*boxEntry
	parses int
}

func newBoxMemo() *boxMemo { return &boxMemo{items: make(map[string]*boxEntry, 64)} }

// features returns the box's parsed features, decoding only when the body has
// changed since it was last seen (a shared or repeated body parses once).
func (m *boxMemo) features(url string, raw []byte) ([]feature, error) {
	sum := sha256.Sum256(raw)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tick++
	if e, ok := m.items[url]; ok && e.sum == sum {
		e.used = m.tick
		return e.feats, nil
	}
	feats, err := parseFeatures(raw)
	if err != nil {
		return nil, err
	}
	m.parses++
	if _, ok := m.items[url]; !ok && len(m.items) >= maxBoxes {
		m.evictLocked()
	}
	m.items[url] = &boxEntry{sum: sum, feats: feats, used: m.tick}
	return feats, nil
}

// evictLocked drops the least-recently-used box (caller holds mu).
func (m *boxMemo) evictLocked() {
	var victim string
	oldest := uint64(math.MaxUint64)
	for k, e := range m.items {
		if e.used < oldest {
			victim, oldest = k, e.used
		}
	}
	delete(m.items, victim)
}

// stats reports the memo's live size and the parses since launch (the gauge:
// a bounded size and a low parse rate prove the shared-box path is working).
func (m *boxMemo) stats() (boxes, parses int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.items), m.parses
}
