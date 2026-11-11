package app

import (
	"reflect"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
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
	lp := &livePipelines{provider: prov, providerFamilies: newProviderFamilies(c, prov, tides, fireProvs, config.Config{})}
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

// TestAllAsksEveryFamily reads providerFamilies' fields by reflection, so a
// family added to the struct without its line in all fails here: both fetch
// paths ask all, and a family it leaves out is fetched by neither.
func TestAllAsksEveryFamily(t *testing.T) {
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost/test (t@example.com)", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	prov := nws.New(c, "")
	fireProvs, _ := fireProviders(c, config.Config{})
	fam := newProviderFamilies(c, prov, coops.New(c, ""), fireProvs, config.Config{})
	asked := map[uintptr]bool{} // by identity: a family's fields are unexported, so reflect can read them but not call them
	for _, p := range fam.all(prov) {
		asked[reflect.ValueOf(p).Pointer()] = true
	}
	v := reflect.ValueOf(fam)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i).Name
		f := v.Field(i)
		if f.Len() == 0 {
			t.Errorf("family %s is empty from newProviderFamilies; it is built nowhere", field)
		}
		for j := 0; j < f.Len(); j++ {
			if !asked[f.Index(j).Elem().Pointer()] {
				t.Errorf("family %s's provider %d is not in all, so neither fetch path asks it", field, j)
			}
		}
	}
}
