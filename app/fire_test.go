package app

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestFireProvidersRegisterFIRMSAndKeyItLive(t *testing.T) {
	// B5 / UAT 100: HMS and WFIGS serve everyone; FIRMS is always registered
	// (so the Setup window can key it without a relaunch) but is a source
	// only with a MAP_KEY — unkeyed it reads "off", never "ok".
	client, err := httpx.New(httpx.Config{UserAgent: "test"})
	if err != nil {
		t.Fatal(err)
	}
	provs, f := fireProviders(client, config.Config{})
	if len(provs) != 3 || provs[0].ID() != "hms" || provs[1].ID() != "wfigs" || provs[2].ID() != "firms" {
		t.Fatalf("want [hms wfigs firms], got %d providers", len(provs))
	}
	if f.Enabled() {
		t.Fatal("unkeyed FIRMS must be disabled")
	}
	if err := f.SetKey("0123456789abcdef0123456789abcdef"); err != nil || !f.Enabled() {
		t.Fatalf("a well-formed key enables it: %v", err)
	}
	if err := f.SetKey("short"); err == nil || !f.Enabled() {
		t.Fatal("a malformed key is refused and the provider keeps its key")
	}
	keyed := config.Config{Providers: map[string]config.Provider{"firms": {Key: "0123456789abcdef0123456789abcdef"}}}
	if _, f2 := fireProviders(client, keyed); !f2.Enabled() {
		t.Fatal("a stored key enables FIRMS at launch")
	}
	// The [fire] rules reach the providers with the defaults filled in.
	if r := fireRules(config.Fire{RadiusKm: 40}); r.RadiusKm != 40 || r.BoldFRPMW != 50 || r.MinConfidence != "nominal" {
		t.Fatalf("rules: %+v", r)
	}
}

func TestFireReportOf(t *testing.T) {
	// UAT 114 / REVIEW C4: the broadcast's fire report — Known only once a
	// fire feed has answered; the [fire] rings and the contributing feeds
	// ride along; FIRMS is credited only when it answered ok.
	rules := fireRules(config.Fire{})
	if fr := fireReportOf(snapshot.FireState{}, 33.24, -117.29, rules, true); fr.Known {
		t.Fatal("no feed yet: not Known")
	}
	fs := snapshot.FireState{AsOf: time.Now(), Hotspots: []snapshot.Hotspot{{Lat: 33.3, Lon: -117.3}}}
	fr := fireReportOf(fs, 33.24, -117.29, rules, true)
	if !fr.Known || fr.RadiusKm != 25 || fr.IncidentRadiusKm != 50 || len(fr.State.Hotspots) != 1 || fr.Lat != 33.24 {
		t.Fatalf("report: %+v", fr)
	}
	if got := fr.Sources; len(got) != 3 || got[2] != "NASA FIRMS" {
		t.Fatalf("firms ok: three feeds named, got %v", got)
	}
	if got := fireReportOf(fs, 0, 0, rules, false).Sources; len(got) != 2 {
		t.Fatalf("firms not ok (unkeyed, rejected, degraded): two feeds, got %v", got)
	}
}
