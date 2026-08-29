package globalfeed

// memo.go — the parse memo (0.13.0, NFR-3; red-team P7/S8): a source's body is
// fetched through httpx every cycle (its TTL/conditional GET decide whether the
// network is touched), but it is DECODED only when the bytes changed. Keyed on
// httpx's own "served from cache" fact first (hdr == nil), else on a sha256 of
// the body — 1–2 ms on the 1 MB national pull, against a 2-minute cadence.
//
// Not domains/fire's Memo[T]: that one memoises errors until Forget (right for
// a parsed archive); a feed parse error must NOT be memoised — the next cycle
// retries — and the caller needs its own slice (Locate writes into elements).

import (
	"crypto/sha256"
	"sync"
)

type sourceMemo struct {
	mu   sync.Mutex
	ok   bool
	hash [32]byte
	last []Event // the parsed events for hash — a field and a method may not share a name, so not "events"
}

// events returns the parsed events for the current body, decoding only when
// the body differs from the memoised one. get returns the body and whether
// httpx served it from cache untouched; parse decodes it.
func (m *sourceMemo) events(get func() ([]byte, bool, error), parse func([]byte) ([]Event, error)) ([]Event, error) {
	body, cached, err := get()
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ok && cached {
		return cloneEvents(m.last), nil
	}
	h := sha256.Sum256(body)
	if m.ok && h == m.hash {
		return cloneEvents(m.last), nil
	}
	evs, err := parse(body)
	if err != nil {
		return nil, err // an error is never memoised — the next cycle retries (S8)
	}
	if evs == nil {
		evs = []Event{} // a successful parse of an empty feed memoises "no events", not "unknown"
	}
	m.ok, m.hash, m.last = true, h, evs
	return cloneEvents(evs), nil
}

// cloneEvents hands callers their own slice (Locate writes into elements). — the elements are copied; the per-class
// detail pointers (Quake/Tropical/Severe) are shared by every caller and
// every published row, and are immutable after parse.
func cloneEvents(in []Event) []Event {
	out := make([]Event, len(in))
	copy(out, in)
	return out
}
