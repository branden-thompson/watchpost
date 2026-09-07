package app

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// EVERY REGISTERED PROVIDER HAS AN ENDPOINT (0.14.0).
//
// [S] keys its table by host, so a provider with no entry here shows under its
// own id with no counters beside it — which reads as "this provider makes no
// requests" rather than "nobody wrote its host down". The map is written by
// hand because it cannot be derived: the snapshot carries no host, and the
// attribution strings are not it — FIRMS credits earthdata.nasa.gov while its
// API is firms.modaps.eosdis.nasa.gov.
func TestEveryProviderHasAnEndpoint(t *testing.T) {
	// The ids come from the PRODUCTION construction, not a list written beside
	// the map. A hardcoded list can only ever agree with the map it was written
	// from: a provider added to one and not the other passes, which is the whole
	// failure this test exists to catch.
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost/test (t@example.com)", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	prov := nws.New(c, "")
	tides := coops.New(c, "")
	fireProvs, _ := fireProviders(c, config.Config{})
	lp := &livePipelines{
		provider: prov,
		marine:   []snapshot.Provider{nws.NewMarine(prov), ndbc.New(c, ""), tides, coops.NewObs(tides)},
		fire:     fireProvs,
		seismic:  seismicProviders(c, config.Config{}),
	}
	var ids []string
	for _, p := range lp.providers() {
		ids = append(ids, p.ID())
	}
	if len(ids) < 2 {
		t.Fatal("the production provider set is empty — this test is checking nothing")
	}
	eps := providerEndpoints()
	for _, id := range ids {
		hosts, ok := eps[id]
		if !ok || len(hosts) == 0 {
			t.Errorf("provider %q has no endpoint", id)
			continue
		}
		for _, h := range hosts {
			if strings.Contains(h, "/") || strings.Contains(h, ":") || !strings.Contains(h, ".") {
				t.Errorf("provider %q: %q is not a bare host", id, h)
			}
		}
	}
}
