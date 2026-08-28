package globalfeed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

func client(t *testing.T) *httpx.Client {
	t.Helper()
	c, err := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func serve(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestUSGSFetchMapsSeverityAndFields(t *testing.T) {
	body := `{"features":[
	  {"id":"us1","properties":{"mag":7.1,"place":"12 km SE of Sendai, Japan","time":1787712730000,"type":"earthquake","tsunami":1},"geometry":{"type":"Point","coordinates":[141.0,38.3,10]}},
	  {"id":"us2","properties":{"mag":5.0,"place":"4 km N of Toride, Japan","time":1787700000000,"type":"earthquake","tsunami":0},"geometry":{"type":"Point","coordinates":[140.1,35.9,50]}},
	  {"id":"","properties":{"mag":6.0,"place":"nowhere","time":1,"type":"earthquake","tsunami":0},"geometry":{"type":"Point","coordinates":[0,0]}}
	]}`
	evs, err := NewUSGS(client(t), serve(t, body)).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("the id-less feature is dropped: %d events", len(evs))
	}
	if evs[0].ID != "us1" || evs[0].Class != ClassQuake || evs[0].Severity != SevRed || evs[0].Type != "Earthquake" || evs[0].Lat != 38.3 {
		t.Fatalf("M7.1 + tsunami is a red Earthquake with the point: %+v", evs[0])
	}
	if evs[1].Severity != SevYellow {
		t.Fatalf("M5.0 is yellow: %+v", evs[1])
	}
}

func TestNHCFetchMapsClassAndBasin(t *testing.T) {
	body := `{"activeStorms":[
	  {"id":"al042026","name":"Dolly","classification":"HU","latitudeNumeric":25.1,"longitudeNumeric":-70.2,"lastUpdate":"2026-08-27T15:00:00.000Z"},
	  {"id":"ep052026","name":"Lala","classification":"TS","latitudeNumeric":15.0,"longitudeNumeric":-110.0,"lastUpdate":"2026-08-27T15:00:00.000Z"},
	  {"id":"al992026","name":"Gone","classification":"EX","latitudeNumeric":40,"longitudeNumeric":-40,"lastUpdate":"2026-08-27T15:00:00.000Z"}
	]}`
	evs, err := NewNHC(client(t), serve(t, body)).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("the post-tropical (EX) storm is skipped: %d events", len(evs))
	}
	if evs[0].Type != "Hurricane" || evs[0].Severity != SevRed || evs[0].Place != "the Atlantic" {
		t.Fatalf("HU al… is a red Hurricane in the Atlantic: %+v", evs[0])
	}
	if evs[1].Type != "Tropical Storm" || evs[1].Severity != SevOrange || evs[1].Place != "the Eastern Pacific" {
		t.Fatalf("TS ep… is an orange Tropical Storm in the E-Pacific: %+v", evs[1])
	}
}

func TestNWSFetchMapsEventAndSeverity(t *testing.T) {
	body := `{"features":[
	  {"id":"urn:oid:tor1","properties":{"event":"Tornado Warning","areaDesc":"Oklahoma County, OK; Cleveland County, OK","onset":"2026-08-27T20:00:00+00:00"}},
	  {"id":"urn:oid:svr1","properties":{"event":"Severe Thunderstorm Warning","areaDesc":"Dallas County, TX","onset":"2026-08-27T19:30:00+00:00"}}
	]}`
	src := NewNWS(client(t), serve(t, body))
	// The query filters by the curated event list.
	if u := src.url(); !strings.Contains(u, "event=Tornado%20Warning") {
		t.Fatalf("the national query filters by event name: %s", u)
	}
	evs, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 || evs[0].Type != "Tornado Warning" || evs[0].Severity != SevRed || evs[0].Class != ClassSevereWx {
		t.Fatalf("a Tornado Warning is a red severe-wx event: %+v", evs)
	}
	if evs[1].Severity != SevOrange {
		t.Fatalf("a Severe Thunderstorm Warning is orange: %+v", evs[1])
	}
	// Its place ties via areaDesc (no coords): the first area, cleaned.
	if loc := Locate(evs[0].HasPoint, evs[0].Lat, evs[0].Lon, evs[0].Place, nil, nil); loc != "Oklahoma County, OK" {
		t.Fatalf("the areaDesc names the location: %q", loc)
	}
}

func TestNWSDropsSupersededAlerts(t *testing.T) {
	// svr2 UPDATES tor1 (references it): tor1 is superseded and must not show,
	// so the same real-world warning isn't listed/announced twice (follow-up).
	body := `{"features":[
	  {"id":"urn:oid:tor1","properties":{"event":"Tornado Warning","areaDesc":"Oklahoma County, OK","onset":"2026-08-27T20:00:00+00:00"}},
	  {"id":"urn:oid:tor2","properties":{"event":"Tornado Warning","areaDesc":"Oklahoma County, OK","onset":"2026-08-27T20:10:00+00:00","references":[{"@id":"urn:oid:tor1"}]}}
	]}`
	evs, err := NewNWS(client(t), serve(t, body)).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Both ride along (tor1 flagged Superseded so the ticker seen-marks it and
	// it can't resurface), but only the update tor2 is displayable.
	byID := map[string]Event{}
	for _, e := range evs {
		byID[e.ID] = e
	}
	if !byID["urn:oid:tor1"].Superseded {
		t.Fatal("the superseded alert is flagged")
	}
	if byID["urn:oid:tor2"].Superseded {
		t.Fatal("the update is not superseded")
	}
}

// Live proof of all three feeds (WATCHPOST_LIVE=1; skipped in CI).
func TestLiveGlobalFeeds(t *testing.T) {
	if os.Getenv("WATCHPOST_LIVE") == "" {
		t.Skip("set WATCHPOST_LIVE=1 to read the real USGS/NHC/NWS feeds")
	}
	c, _ := httpx.New(httpx.Config{UserAgent: "watchpost-test (t@example.com)", RatePerSec: 2, MaxRetries: 1})
	for _, s := range []Source{NewUSGS(c, ""), NewNHC(c, ""), NewNWS(c, "")} {
		evs, err := s.Fetch(context.Background())
		if err != nil {
			t.Fatalf("%s live: %v", s.Name(), err)
		}
		t.Logf("%s: %d events", s.Name(), len(evs))
		for _, e := range evs[:min(3, len(evs))] {
			t.Logf("  [%s] %s — %s (%s)", sevName(e.Severity), e.Type, e.Place, e.At.Format(time.RFC3339))
		}
	}
}

func sevName(s Severity) string {
	return map[Severity]string{SevRed: "RED", SevOrange: "ORG", SevYellow: "YEL"}[s]
}
