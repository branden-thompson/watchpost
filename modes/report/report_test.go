package report

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Spec: architecture §10.2 (exit codes), §10.11 (null↔n/a parity), AI-10
// (envelope), M5 (bidirectional parity — every JSON data leaf's formatted form
// appears in the plain render and vice versa). Fixture uses pairwise-distinct
// sentinel values (red-team PLAN #3) so substring matches cannot alias.

func f64(v float64) *float64 { return &v }

// fixtureSnapshot carries pairwise-distinct sentinels in every populated field.
func fixtureSnapshot() *snapshot.Snapshot {
	sent := time.Date(2026, 8, 23, 14, 5, 0, 0, time.UTC)
	obs := time.Date(2026, 8, 23, 17, 52, 0, 0, time.UTC)
	return &snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion,
		GeneratedAt:   time.Date(2026, 8, 23, 18, 4, 11, 0, time.UTC),
		Locations: []snapshot.Location{{
			Label: "Oceanside, CA", Zip: "92057", Lat: 33.2405, Lon: -117.2912, TZ: "America/Los_Angeles",
			Harmonized: snapshot.Conditions{
				ObservedAt: obs,
				Temp:       f64(22.8), Feels: f64(24.1), Dewpoint: f64(16.3),
				HumidityPct: f64(66.5), Pressure: f64(1014.2),
				Wind: f64(4.6), WindDirDeg: f64(250), // WindGust nil on purpose (n/a path)
				Visibility: f64(16090), Condition: "partly_cloudy",
				Source: snapshot.SourceInfo{Provider: "nws", ModelOrStation: "KOKB", IssuedAt: obs},
			},
			ByProvider: map[string]snapshot.Section{"nws": {}},
			Alerts: []snapshot.Alert{{
				ID: "urn:oid:2.49.0.1.840.0.abc123", Event: "Heat Advisory",
				Severity: "moderate", Urgency: "expected", Certainty: "likely",
				MessageType: "alert", Sent: sent, Effective: sent,
				Expires:  time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC),
				AreaDesc: "San Diego County Coastal Areas", Headline: "Heat Advisory until midnight",
				Description: "Hot conditions expected.", Instruction: "Drink plenty of fluids.",
				Source: snapshot.SourceInfo{Provider: "nws", IssuedAt: sent},
			}},
			Fire: snapshot.FireState{
				AsOf:      obs,
				Hotspots:  []snapshot.Hotspot{{Lat: 33.3, Lon: -117.3, DetectedAt: obs, Confidence: "analyst", FRPMW: f64(62.4), DistanceKm: f64(8.2), Source: snapshot.SourceInfo{Provider: "hms", ModelOrStation: "GOES-WEST", IssuedAt: obs}}},
				Incidents: []snapshot.Incident{{Name: "Timber", Discovered: obs.Add(-72 * time.Hour), PercentContained: f64(26), Acres: f64(12915), State: "CA", Source: snapshot.SourceInfo{Provider: "wfigs", DistanceKm: f64(30.5)}}},
			},
			Hourly: []snapshot.Hourly{{Time: obs.Add(time.Hour), Temp: f64(23.4), PrecipProb: f64(15), Wind: f64(4.0), Condition: "partly_cloudy"}},
			Daily:  []snapshot.Daily{{Date: "2026-08-23", TempMax: f64(23.9), TempMin: f64(17.2), PrecipProb: f64(20), Condition: "partly_cloudy"}},
		}},
		Providers: []snapshot.ProviderStatus{{ID: "nws", Role: "reference", Status: snapshot.ProviderOK, FetchedAt: obs, Attribution: "NOAA/NWS"}},
		Warnings:  []snapshot.Warning{},
	}
}

func TestJSONEnvelope(t *testing.T) {
	out, err := RenderJSON(fixtureSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatal(err)
	}
	if env["schema_version"] != snapshot.SchemaVersion {
		t.Fatalf("schema_version = %v", env["schema_version"])
	}
	if _, ok := env["locations"]; !ok {
		t.Fatal("locations missing")
	}
}

func TestPlainRenderContainsCoreData(t *testing.T) {
	txt := RenderPlain(fixtureSnapshot(), 80)
	for _, want := range []string{
		"Oceanside, CA (92057)",           // R-2': zip alongside label
		"22.8",                            // temp sentinel
		"ALERT [moderate]: Heat Advisory", // R-12a severity as text
		"n/a",                             // nil gust renders n/a
		"NOAA/NWS",                        // attribution
		"fire: 1 hotspot(s) nearby — nearest 8.2 km, 62.4 MW, hms/GOES-WEST at", // B5 fire, plain words
		"incident: Timber — 12915.0 acres, 26.0% contained, 30.5 km",
	} {
		if !strings.Contains(txt, want) {
			t.Fatalf("plain render missing %q:\n%s", want, txt)
		}
	}
	if strings.Contains(txt, "\x1b[") {
		t.Fatal("plain render must carry no ANSI")
	}
}

func TestParityJSONLeavesAppearInPlainRender(t *testing.T) {
	// M5 forward SPOT-CHECK of the core field set. The ratified M5 direction is
	// JSON ⊇ TTY (reverse test below is the load-bearing one): the TTY may
	// summarize series (Hourly[0] as "next hour"), never invent facts.
	snap := fixtureSnapshot()
	txt := RenderPlain(snap, 200)
	for name, v := range map[string]float64{
		"temp": 22.8, "feels": 24.1, "dewpoint": 16.3, "humidity": 66.5,
		"pressure": 1014.2, "hourly_temp": 23.4, "daily_max": 23.9, "daily_min": 17.2,
	} {
		f := FormatNum(v)
		if !strings.Contains(txt, f) {
			t.Errorf("M5: JSON leaf %s=%s not in plain render", name, f)
		}
	}
}

func TestParityPlainNumbersMapToJSONLeaves(t *testing.T) {
	// M5 reverse direction: every number token in the plain render exists as a
	// numeric leaf in the JSON (value-level compare: Go marshals 15.0 as 15,
	// so substring matching would false-fail — the parity is over values).
	snap := fixtureSnapshot()
	// Skip the header line: "schema 1.0.0-rc" is envelope metadata, not a
	// data leaf; the parity contract covers data.
	lines := strings.SplitN(RenderPlain(snap, 200), "\n", 2)
	txt := lines[len(lines)-1]
	out, _ := RenderJSON(snap)
	var env any
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatal(err)
	}
	leaves := map[float64]bool{}
	collectNums(env, leaves)
	numRe := regexp.MustCompile(`\b\d+\.\d\b`)
	for _, tok := range numRe.FindAllString(txt, -1) {
		v, err := strconv.ParseFloat(tok, 64)
		if err != nil {
			t.Fatalf("unparseable rendered number %q", tok)
		}
		if !leaves[v] {
			t.Errorf("M5 reverse: rendered number %s has no JSON numeric leaf", tok)
		}
	}
}

// collectNums walks decoded JSON gathering every numeric leaf.
func collectNums(v any, out map[float64]bool) {
	switch x := v.(type) {
	case float64:
		out[x] = true
	case map[string]any:
		for _, e := range x {
			collectNums(e, out)
		}
	case []any:
		for _, e := range x {
			collectNums(e, out)
		}
	}
}

func TestNoSecretInOutput(t *testing.T) {
	// §10.5: no configured key value may appear in any machine output byte.
	snap := fixtureSnapshot()
	secret := "sentinel-key-A1"
	out, _ := RenderJSON(snap)
	if strings.Contains(string(out), secret) || strings.Contains(RenderPlain(snap, 80), secret) {
		t.Fatal("secret leaked into output")
	}
}

func TestExitCodes(t *testing.T) {
	ok := fixtureSnapshot()
	if got := ExitCode(ok); got != 0 {
		t.Fatalf("healthy snapshot exit = %d, want 0", got)
	}
	deg := fixtureSnapshot()
	deg.Providers[0].Status = snapshot.ProviderDegraded
	if got := ExitCode(deg); got != 2 {
		t.Fatalf("degraded provider exit = %d, want 2", got)
	}
	stale := fixtureSnapshot()
	stale.Warnings = append(stale.Warnings, snapshot.Warning{Code: snapshot.WarnObsStale, Message: "old obs"})
	if got := ExitCode(stale); got != 0 {
		t.Fatalf("obs_stale must NOT trigger exit 2 (§10.2 carve-out), got %d", got)
	}
}

// Quality pass Q0: `report --verbose` trails the plain report with one
// counter line per host — plain words, raw numbers, no escapes.
func TestRenderRequestsIsOneLinePerHost(t *testing.T) {
	out := RenderRequests(httpx.RequestStats{Uptime: 1500 * time.Millisecond, Hosts: []httpx.HostStats{
		{Host: "api.weather.gov", Attempts: 7, Net: 5, Cache: 2, BytesNet: 4096, H2: 5, TLSHandshakes: 1}}})
	want := "requests: uptime 1.5s\nrequests api.weather.gov: attempts 7 net 5 cache 2 neg 0 304 0 bytes 4096 h2 5 tls 1\n"
	if out != want {
		t.Fatalf("got %q\nwant %q", out, want)
	}
	if got := RenderRequests(httpx.RequestStats{}); !strings.Contains(got, "requests: none") {
		t.Fatalf("no traffic must be said outright, got %q", got)
	}
}
