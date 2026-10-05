package globalfeed

// feed.go — ONE canonical way this package reads a JSON feed.
//
// NHC, USGS and NWS read a feed in the same six lines: ask the memo for
// events, and give it a closure that pulls the body through httpx with a TTL
// and reports whether the response was a 304. NWS differs by one statement —
// it computes its URL first — which is exactly the near-duplicate a structural
// fingerprint can miss.
//
// HUM LEAD, 2026-09-08: "we should likely only have one .Fetch for API data
// feeds and provide a pointer to the provider — we should always follow the
// principle of, 'There's one canonical way to do a specific thing.'"

import (
	"context"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// jsonFeed is the state every source in this package holds, and the reason it
// is embedded rather than passed: the fields promote, so `n.client`, `n.base`
// and `n.memo` work at every call site.
type jsonFeed struct {
	client *httpx.Client
	base   string
	memo   sourceMemo
}

// fetch reads the feed through the parse memo: the body comes from httpx
// (cache, TTL and conditional GET), and it is decoded only when it changed.
//
// url is a parameter rather than always f.base because NWS builds its query per
// call, and parse is a parameter because each source decodes its own shape.
// Those two are the whole of the difference between the three sources.
func (f *jsonFeed) fetch(ctx context.Context, url string, ttl time.Duration, parse func([]byte) ([]Event, error)) ([]Event, error) {
	return f.memo.events(func() ([]byte, bool, error) {
		var body []byte
		hdr, err := f.client.GetJSON(ctx, url, &body, httpx.TTL(ttl))
		return body, hdr == nil, err // hdr == nil is a 304: the body is unchanged
	}, parse)
}
