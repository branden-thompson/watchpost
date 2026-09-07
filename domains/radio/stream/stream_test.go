package stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
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
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
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

// Quality pass Q1 (plan Q1 task 1; DISCOVER LR-1, red-team PR-1/IS-2,
// RT-9/R2-18, PR-9): weatherUSA over plain HTTP end to end, mounts pinned
// to the directory host (port-agnostic), and a failing directory asked at
// most once per directoryTTL.

func TestWeatherUSAIsPlainHTTPEndToEnd(t *testing.T) {
	if !strings.HasPrefix(weatherUSAStatus, "http://") || !strings.HasPrefix(weatherUSAListen, "http://") {
		t.Fatal("both weatherUSA constants must be plain HTTP: the directory only offers RSA-kex TLS and the mounts are http:// already (LR-1)")
	}
	if !strings.HasPrefix(wxradioStatus, "https://") || !strings.HasPrefix(wxradioListen, "https://") {
		t.Fatal("wxradio stays HTTPS")
	}
	srv := server(t)
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000})
	m := NewDirectory(c, srv.URL+"/wx", srv.URL+"/wu").Mounts(context.Background())
	for _, mt := range m["KEC49"] {
		if !strings.HasPrefix(mt.URL, srv.URL) {
			t.Fatalf("a mount's scheme and host come from the directory base, got %s", mt.URL)
		}
	}
}

func TestMountsAcceptedOnlyOnTheDirectoryHost(t *testing.T) {
	list := []source{
		{ListenURL: "http://127.0.0.1:8000/NWR/KEC49.mp3", ServerType: "audio/mpeg"}, // Icecast advertises :8000 — the port is not the host
		{ListenURL: "http://evil.example/NWR/KEC55.mp3", ServerType: "audio/mpeg"},   // another host: dropped
		{ListenURL: "http://LOCALHOST/NWR/KIH20.mp3", ServerType: "audio/mpeg"},      // case-insensitive host
	}
	got := mountsOf(list, "http://localhost/", "http://radio.weatherusa.net/", "weatherusa.net", weatherUSACallsign)
	if len(got) != 1 || got[0].Callsign != "KIH20" || got[0].URL != "http://localhost/NWR/KIH20.mp3" {
		t.Fatalf("only the directory's own host is accepted, URL rebuilt from the base: %+v", got)
	}
	got = mountsOf(list, "http://127.0.0.1/", "http://radio.weatherusa.net/", "weatherusa.net", weatherUSACallsign)
	if len(got) != 1 || got[0].Callsign != "KEC49" || got[0].URL != "http://127.0.0.1/NWR/KEC49.mp3" {
		t.Fatalf("a matching host on another port is accepted and the URL keeps the base's port: %+v", got)
	}
	// The relay's canonical host is always accepted (the live documents
	// advertise it), whatever base the directory was fetched from.
	live := []source{{ListenURL: "http://radio.weatherusa.net:80/NWR/KEC49.mp3"}}
	if got := mountsOf(live, "http://127.0.0.1/", "http://radio.weatherusa.net/", "weatherusa.net", weatherUSACallsign); len(got) != 1 {
		t.Fatalf("the canonical relay host must be accepted: %+v", got)
	}
}

func TestFailingDirectoryIsAskedAtMostOncePerTTL(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/wu/") {
			hits.Add(1)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		b, _ := os.ReadFile("testdata/wxradio.json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000})
	d := NewDirectory(c, srv.URL+"/wx", srv.URL+"/wu")
	now := time.Now()
	d.now = func() time.Time { return now }
	for range 3 { // Tune, advance, SetMode — each resolves
		m, st := d.MountsWithStatus(context.Background())
		if len(m) == 0 {
			t.Fatal("the healthy relay still contributes")
		}
		if len(st) != 2 || st[0].Err != nil || st[1].Relay != "weatherusa.net" || st[1].Err == nil || st[1].Since.IsZero() {
			t.Fatalf("statuses must name the down relay with its reason and first-seen time: %+v", st)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("a failing directory is hit at most once per %v, got %d", directoryTTL, hits.Load())
	}
	now = now.Add(directoryTTL + time.Second)
	d.MountsWithStatus(context.Background())
	if hits.Load() != 2 {
		t.Fatalf("after the window it is asked again, got %d", hits.Load())
	}
}

// CO-LOCATED TRANSMITTERS RANK IN A SPECIFIED ORDER.
//
// Coachella (KIG78) and Coachella / Spanish (WNG712) share a mast, so their
// distance from any listener compares exactly equal. With a KM-only comparator
// the winner was whatever the sort's internals produced — it was WNG712, which
// is how Vista, CA got the Spanish feed (HUM LEAD, UAT 2026-09-04).
//
// This asserts the ORDER, not that the order is repeatable. Go's sort is
// deterministic for a given input, so an "is it the same twice" test passes
// against the very comparator that has no tie-break at all — the first version
// of this test did exactly that and had to be thrown away. What is actually
// fragile is the tie moving when the table changes underneath it, and only
// naming the expected winner catches that.
//
// The callsign is a tie-break, NOT a language policy: KIG78 winning is luck.
// When the language preference lands, this test's expectation is what it
// changes.
func TestCoLocatedTransmittersRankInASpecifiedOrder(t *testing.T) {
	tbl, err := LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	const vistaLat, vistaLon = 33.2000, -117.2425

	var order []string
	var kmA, kmB float64
	for _, n := range tbl.Nearest(vistaLat, vistaLon, tbl.Len()) {
		switch n.Callsign {
		case "KIG78":
			kmA, order = n.KM, append(order, n.Callsign)
		case "WNG712":
			kmB, order = n.KM, append(order, n.Callsign)
		}
	}
	if len(order) != 2 {
		t.Fatalf("both Coachella transmitters must rank; got %v", order)
	}
	// The premise: they really are a tie. Without this the test could pass on
	// distance alone and prove nothing about the tie-break.
	if kmA != kmB {
		t.Fatalf("the pair no longer shares a mast (%.6f vs %.6f); this test's premise is gone", kmA, kmB)
	}
	if order[0] != "KIG78" {
		t.Errorf("the tie must break by callsign, got %v first", order[0])
	}
}

// LANGUAGE IS READ FROM THE SITE NAME, because it is recorded nowhere else.
//
// Both forms the real table uses are covered, and so is the trap: a suffix
// match, not a substring, because "Spanish Fork" is a place.
func TestATransmittersLanguageIsReadFromItsSite(t *testing.T) {
	for _, tc := range []struct{ site, want string }{
		{"Coachella / Spanish", LangSpanish}, // the / form
		{"El Paso Spanish", LangSpanish},     // the bare-suffix form
		{"Hialeah Spanish", LangSpanish},
		{"Coachella", LangEnglish},
		{"San Diego", LangEnglish},
		{"Spanish Fork", LangEnglish}, // a place, not a language
		{"", LangEnglish},
	} {
		if got := (&Transmitter{Site: tc.site}).Lang(); got != tc.want {
			t.Errorf("%q reads as %q, want %q", tc.site, got, tc.want)
		}
	}

	// EVERY Spanish row in the shipped table is found. A rule read off five
	// rows should be checked against those five rows, or the next naming form
	// silently becomes an English station.
	tbl, err := LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	got := 0
	for _, n := range tbl.Nearest(0, 0, tbl.Len()) {
		if n.Lang() == LangSpanish {
			got++
		}
		if strings.Contains(strings.ToLower(n.Site), "spanish") && n.Lang() != LangSpanish {
			t.Errorf("%s %q names Spanish and does not read as it", n.Callsign, n.Site)
		}
	}
	if got != 5 {
		t.Errorf("the table carries 5 Spanish transmitters, found %d — if the table grew one, update this count deliberately", got)
	}
}
