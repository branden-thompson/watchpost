package temperature

import (
	"context"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// The temperature client is the gate's cache: a client the gate could not read
// would refuse what is paid for, which is what D-217 ends.
var _ cachedReader = (*httpx.Client)(nil)

// cachingGetter is a gateGetter with a response cache: what it answered once
// it serves again without asking.
type cachingGetter struct {
	gateGetter
	kept map[string][]byte
}

func (c *cachingGetter) GetText(ctx context.Context, url string, opts ...httpx.Option) ([]byte, error) {
	if b, ok := c.kept[url]; ok {
		return b, nil
	}
	b, err := c.gateGetter.GetText(ctx, url, opts...)
	if err == nil {
		c.kept[url] = b
	}
	return b, err
}

func (c *cachingGetter) Cached(url string) ([]byte, bool) {
	b, ok := c.kept[url]
	return b, ok
}

// WHAT IS PAID FOR IS SERVED WHILE THE QUOTA IS SPENT (D-217): a held host's
// answers already in the cache are served, never asked for; only an answer
// from the network frees the host - a cached one says nothing about the quota.
func TestAHeldHostServesWhatIsPaidFor(t *testing.T) {
	now := quotaNow
	refuse := false
	under := &cachingGetter{kept: map[string][]byte{}, gateGetter: gateGetter{answer: func(string) ([]byte, error) {
		if refuse {
			return nil, spent("Daily")
		}
		return []byte("ok"), nil
	}}}
	g := NewQuotaGate(under, func() time.Time { return now })
	ctx := context.Background()
	paid, other, third := "https://api.open-meteo.com/v1/forecast?a", "https://api.open-meteo.com/v1/forecast?b", "https://api.open-meteo.com/v1/forecast?c"
	if _, err := g.GetText(ctx, paid); err != nil {
		t.Fatal(err)
	}
	refuse = true
	if _, err := g.GetText(ctx, other); err == nil {
		t.Fatal("the refusal was not passed on")
	}
	asked := len(under.asked)
	if body, err := g.GetText(ctx, paid); err != nil || string(body) != "ok" {
		t.Errorf("a held host refused an answer already paid for: %q, %v", body, err)
	}
	if _, err := g.GetText(ctx, third); err == nil {
		t.Error("a held host's uncached ask went through")
	}
	if len(under.asked) != asked {
		t.Errorf("a held host was asked %d more times; want none", len(under.asked)-asked)
	}

	now = now.Add(quotaProbeEvery)
	if _, err := g.GetText(ctx, paid); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.Refused(); !ok {
		t.Error("a cached answer freed the host: it says nothing about the quota")
	}
	refuse = false
	if body, err := g.GetText(ctx, third); err != nil || string(body) != "ok" || len(under.asked) != asked+1 {
		t.Fatalf("the probe was spent on the cached answer: %q, %v, %d asks", body, err, len(under.asked)-asked)
	}
	if _, ok := g.Refused(); ok {
		t.Error("an answered probe left the quota said spent")
	}
}
