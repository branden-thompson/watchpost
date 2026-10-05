package globalfeed

// memo.go — the parse memo (0.13.0, NFR-3; red-team P7/S8): a source's body is
// fetched through httpx every cycle (its TTL/conditional GET decide whether the
// network is touched), but it is DECODED only when the bytes changed - the
// shared `platform/bodymemo`, one entry. httpx's own "served unchanged" fact
// (a 304) answers from the last parse without hashing the body; otherwise the
// body's hash decides. A parse error is not kept - the next cycle retries
// (S8) - and each caller gets its own slice (Locate writes into elements).

import (
	"sync"

	"github.com/branden-thompson/watchpost/platform/bodymemo"
)

type sourceMemo struct {
	once sync.Once
	m    *bodymemo.Memo[struct{}, []Event]
}

// events returns the parsed events for the current body, decoding only when
// the body differs from the memoised one. get returns the body and whether
// httpx served it unchanged; parse decodes it.
func (s *sourceMemo) events(get func() ([]byte, bool, error), parse func([]byte) ([]Event, error)) ([]Event, error) {
	body, unchanged, err := get()
	if err != nil {
		return nil, err
	}
	s.once.Do(func() { s.m = bodymemo.New[struct{}, []Event](1) })
	if unchanged {
		if evs, ok := s.m.Last(struct{}{}); ok {
			return cloneEvents(evs), nil
		}
	}
	evs, err := s.m.Parsed(struct{}{}, body, parse)
	if err != nil {
		return nil, err
	}
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
