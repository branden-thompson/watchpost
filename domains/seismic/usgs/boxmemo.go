package usgs

// boxmemo.go — the parsed-box memo (seismic P2, plan §3 Approach B, the Q5
// FIRMS-tile precedent). Many locations share one regional box URL, and a
// box's body rarely changes between ticks; the memo decodes each distinct
// body once, revalidated by hash, so the shared request is also a shared
// parse. Bounded LRU (P10-03) and gauged (the [S] modal).

import "github.com/branden-thompson/watchpost/platform/bodymemo"

// maxBoxes bounds the memo: the distinct query URLs the current location set
// can touch — at most one near box per location (≤ 60) plus a handful of
// shared regional boxes. Past the bound the least-recently-used box is
// dropped, so a churny location set cannot grow it without limit.
const maxBoxes = 160

// boxMemo is the parsed-box cache.
//
// THE CACHE IS platform/bodymemo (F-53). This held its own tick counter, hash
// revalidation and least-recently-used eviction, and domains/fire/firms held a
// byte-identical copy of all three. What is left here is the name the seismic
// path calls it by, and the box's own key: the query URL.
type boxMemo struct {
	cache *bodymemo.Memo[string, []feature]
}

func newBoxMemo() *boxMemo { return &boxMemo{cache: bodymemo.New[string, []feature](maxBoxes)} }

// features returns the box's parsed features, decoding only when the body has
// changed since it was last seen (a shared or repeated body parses once).
//
// parseFeatures IS PASSED AS A TOP-LEVEL FUNCTION, not wrapped: a closure that
// captures anything allocates on every call, hits included, and
// TestBoxMemoHitAllocBudget holds a hit at zero.
func (m *boxMemo) features(url string, raw []byte) ([]feature, error) {
	return m.cache.Parsed(url, raw, parseFeatures)
}

// stats reports the memo's live size and the parses since launch (the gauge:
// a bounded size and a low parse rate prove the shared-box path is working).
func (m *boxMemo) stats() (boxes, parses int) { return m.cache.Stats() }
