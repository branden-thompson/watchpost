package stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Spec: B4 / AI-4 — location -> county SAME -> covering transmitter ->
// relay mount; nearest relayed transmitter when the covering one is not
// carried. Fixtures: the vendored NWS table and trimmed live directory
// documents (2026-08-24).

func TestEmbeddedTableParsesAndIndexes(t *testing.T) {
	tb, err := LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	if tb.Len() < 900 {
		t.Fatalf("table has %d transmitters, expected ~1000", tb.Len())
	}
	sd, ok := tb.ByCallsign("KEC62")
	if !ok || sd.Site != "San Diego" || sd.FreqMHz != "162.400" || sd.Lat < 32 || sd.Lat > 34 {
		t.Fatalf("KEC62: %+v", sd)
	}
	if cov := tb.Covering(SAMEFromFIPS("06073")); len(cov) == 0 || cov[0].Callsign != "KEC62" {
		t.Fatalf("San Diego county must be covered by KEC62: %v", cov)
	}
	if near := tb.Nearest(33.2405, -117.2912, 1); near[0].Callsign != "KEC62" || near[0].KM > 60 {
		t.Fatalf("nearest to Oceanside is KEC62: %+v", near[0])
	}
}

func server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/wx/status-json.xsl"):
			b, _ := os.ReadFile("testdata/wxradio.json")
			_, _ = w.Write(b)
		case strings.HasPrefix(r.URL.Path, "/wu/status-json.xsl"):
			b, _ := os.ReadFile("testdata/weatherusa.json")
			_, _ = w.Write(b)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDirectoryMergesRelaysByCallsign(t *testing.T) {
	srv := server(t)
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: -1})
	d := NewDirectory(c, srv.URL+"/wx", srv.URL+"/wu")
	m := d.Mounts(context.Background())
	if len(m["KEC49"]) != 2 { // Monterey is on both relays
		t.Fatalf("KEC49 mounts: %+v", m["KEC49"])
	}
	if got := m["KEC49"][0]; got.Relay != "wxradio.org" || !strings.HasSuffix(got.URL, "/CA-Monterey-KEC49") {
		t.Fatalf("wxradio mount: %+v", got)
	}
	if got := m["KEC49"][1]; got.Relay != "weatherusa.net" || !strings.HasSuffix(got.URL, "/NWR/KEC49.mp3") { // the relay's path is kept
		t.Fatalf("weatherUSA mount: %+v", got)
	}
	if _, ok := m["KEC55"]; !ok { // "KEC55_2.mp3" folds to its callsign
		t.Fatal("suffixed weatherUSA mounts must fold to the callsign")
	}
	if _, ok := m["KEC62"]; ok {
		t.Fatal("San Diego is not relayed by either directory in the fixtures")
	}
}

func TestResolveCoveringFirstThenNearestRelayed(t *testing.T) {
	srv := server(t)
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: -1})
	r, err := NewResolver(NewDirectory(c, srv.URL+"/wx", srv.URL+"/wu"))
	if err != nil {
		t.Fatal(err)
	}
	// Monterey county (06053) is covered by relayed KEC49: it comes first, marked covering.
	st := r.Resolve(context.Background(), 36.6, -121.9, SAMEFromFIPS("06053"))
	if len(st) == 0 || st[0].Callsign != "KEC49" || !st[0].Covering || len(st[0].Mounts) != 2 {
		t.Fatalf("Monterey: %+v", st)
	}
	// Oceanside: KEC62 covers but is not relayed -> nearest relayed transmitters, none covering, distance labelled.
	st = r.Resolve(context.Background(), 33.2405, -117.2912, SAMEFromFIPS("06073"))
	if len(st) == 0 || st[0].Covering || st[0].KM < 50 {
		t.Fatalf("Oceanside must fall back to the nearest relayed transmitter: %+v", st)
	}
	if len(st) > directoryCandidateCap {
		t.Fatal("candidates are capped")
	}
	for i := 1; i < len(st); i++ {
		if st[i].KM < st[i-1].KM {
			t.Fatal("fallbacks are ordered by distance")
		}
	}
	named := false
	for _, tx := range r.CoveringTransmitters(SAMEFromFIPS("06073")) {
		named = named || tx.Callsign == "KEC62"
	}
	if !named {
		t.Fatal("the UI can still name the unrelayed covering transmitter (KEC62)")
	}
}

func TestSAMEFromFIPS(t *testing.T) {
	if SAMEFromFIPS("06073") != "006073" || SAMEFromFIPS("006073") != "006073" || SAMEFromFIPS("73") != "" {
		t.Fatal("SAME = 0 + county FIPS")
	}
	if SAMEFromUGC("CAC073") != "006073" || SAMEFromUGC("NYC061") != "036061" || SAMEFromUGC("CAZ554") != "" || SAMEFromUGC("XXC001") != "" {
		t.Fatal("SAME from county UGC = 0 + state FIPS + county")
	}
}
