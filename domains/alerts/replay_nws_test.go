package alerts_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/sched"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// B1 red-team #7: the stub-provider replay bypassed nws.fetchAlerts/mapAlert —
// exactly where the truncated-AffectedZones bug lived. This variant drives the
// REAL provider against a mutable httptest feed, covering zone mapping,
// multi-zone alerts, and the two-pass fix.

func TestReplayThroughRealNWSProvider(t *testing.T) {
	var mu sync.Mutex
	activeAlerts := []map[string]any{}
	setFeed := func(alerts ...map[string]any) {
		mu.Lock()
		defer mu.Unlock()
		activeAlerts = alerts
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/alerts/active":
			mu.Lock()
			feats := make([]map[string]any, 0, len(activeAlerts))
			for _, a := range activeAlerts {
				feats = append(feats, map[string]any{"properties": a})
			}
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"features": feats})
		case len(r.URL.Path) > 8 && r.URL.Path[:8] == "/points/":
			_, _ = w.Write([]byte(`{"properties":{"forecast":"BASE/f","forecastHourly":"BASE/h","observationStations":"BASE/s","forecastZone":"https://api.weather.gov/zones/forecast/CAZ554","county":"https://api.weather.gov/zones/county/CAC073","timeZone":"America/Los_Angeles"}}`))
		default: // stations
			_, _ = w.Write([]byte(`{"features":[{"properties":{"stationIdentifier":"KOKB"}}]}`))
		}
	}))
	defer srv.Close()

	client, err := httpx.New(httpx.Config{UserAgent: "t", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	provider := nws.New(client, srv.URL)
	ref := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.2405, Lon: -117.2912}
	asm := snapshot.NewAssembler([]snapshot.LocationRef{ref}, []string{"nws"})

	fetch := func() {
		frag, err := provider.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindAlerts, Locations: []snapshot.LocationRef{ref}})
		if err != nil {
			t.Fatal(err)
		}
		asm.Apply(frag)
	}
	_ = sched.RealClock{} // harness parity: cadence covered by the stub replay

	sent := time.Now().UTC().Add(-30 * time.Second)
	multiZone := map[string]any{
		"id": "real-1", "event": "Tornado Warning", "severity": "Extreme",
		"sent": sent.Format(time.RFC3339), "expires": time.Now().Add(time.Hour).Format(time.RFC3339),
		"affectedZones": []string{
			"https://api.weather.gov/zones/forecast/CAZ043", // unrelated zone FIRST
			"https://api.weather.gov/zones/county/CAC073",   // our county second
		},
	}
	setFeed(multiZone)
	fetch()
	snap := asm.Snapshot()
	al := snap.Locations[0].Alerts
	if len(al) != 1 {
		t.Fatalf("alert must map to our location via county UGC, got %d", len(al))
	}
	// The two-pass fix: full zone list survives even though our match was second.
	if len(al[0].AffectedZones) != 2 {
		t.Fatalf("AffectedZones truncated (B1 #1 regression): %v", al[0].AffectedZones)
	}
	// Cancellation through the real path.
	setFeed()
	fetch()
	if n := len(asm.Snapshot().Locations[0].Alerts); n != 0 {
		t.Fatalf("cleared feed must clear alerts, got %d", n)
	}
}
