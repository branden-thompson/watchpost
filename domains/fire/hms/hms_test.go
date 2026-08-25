package hms

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// kml is the shape live-probed on 2026-08-25 (hms_fire20260825.kml): a
// strong fire 6 km from Oceanside, a weak GOES point in the same ring (no
// FRP), a second pass of the strong one, and a far one.
const kml = `<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2"><Document>
<Placemark><description><![CDATA[Lon: -117.240000<br>Lat: 33.290000<br>YearDay: 2026237<br>Time: 0201UTC<br>Satellite: GOES-WEST<br>Method: NGFS<br>Ecosystem: 22<br>FRP: 61.500MW]]></description><Point><coordinates>-117.240000,33.290000</coordinates></Point></Placemark>
<Placemark><description><![CDATA[Lon: -117.241000<br>Lat: 33.291000<br>YearDay: 2026237<br>Time: 0601UTC<br>Satellite: NOAA-20<br>Method: VIIRS<br>Ecosystem: 22<br>FRP: 20.000MW]]></description><Point><coordinates>-117.241000,33.291000</coordinates></Point></Placemark>
<Placemark><description><![CDATA[Lon: -117.300000<br>Lat: 33.200000<br>YearDay: 2026237<br>Time: 0301UTC<br>Satellite: GOES-EAST<br>Method: NGFS<br>Ecosystem: 22<br>FRP: -999.000MW]]></description><Point><coordinates>-117.300000,33.200000</coordinates></Point></Placemark>
<Placemark><description><![CDATA[Lon: -121.552498<br>Lat: 49.891666<br>YearDay: 2026237<br>Time: 0201UTC<br>Satellite: GOES-EAST<br>Method: NGFS<br>Ecosystem: 22<br>FRP: 10.980MW]]></description><Point><coordinates>-121.552498,49.891666</coordinates></Point></Placemark>
<Placemark><description><![CDATA[garbage]]></description><Point><coordinates>bad</coordinates></Point></Placemark>
</Document></kml>`

func kmz(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"doc.kml", "kmls/GOES-EASTfire20260825.kml", "kmls/hms_fire20260825.kml"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		body := kml
		if name != "kmls/hms_fire20260825.kml" {
			body = `<kml><Document></Document></kml>` // the merged file is the answer; the others are ignored when it exists
		}
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func TestParseReadsTheMergedFileFromTheKMZ(t *testing.T) {
	pts, err := Parse(kmz(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 4 {
		t.Fatalf("4 parseable placemarks (the garbage one skipped), got %d", len(pts))
	}
	p := pts[0]
	if p.Lat != 33.29 || p.Lon != -117.24 || p.Satellite != "GOES-WEST" || p.FRPMW == nil || *p.FRPMW != 61.5 {
		t.Fatalf("fields: %+v", p)
	}
	if p.At.Format("2006-01-02 15:04") != "2026-08-25 02:01" {
		t.Fatalf("YearDay 2026237 + 0201UTC is 2026-08-25 02:01 UTC, got %s", p.At)
	}
	if pts[2].FRPMW != nil {
		t.Fatal("FRP -999 means unknown")
	}
	if _, err := Parse([]byte("not a zip")); err == nil {
		t.Fatal("garbage is refused")
	}
}

func TestFetchAnswersEveryLocationFromOneArchive(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/vnd.google-earth.kmz")
		_, _ = w.Write(kmz(t))
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: -1})
	p := New(c, srv.URL+"/fireAllSats.kmz", fire.DefaultRules())
	oceanside := snapshot.LocationRef{Label: "Oceanside, CA", Lat: 33.24, Lon: -117.29}
	boise := snapshot.LocationRef{Label: "Boise, ID", Lat: 43.62, Lon: -116.2}
	frag, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindFire, Locations: []snapshot.LocationRef{oceanside, boise}})
	if err != nil || frag.Err != nil {
		t.Fatalf("fetch: %v %v", err, frag.Err)
	}
	if hits != 1 {
		t.Fatalf("one archive download serves the whole watchlist, got %d", hits)
	}
	o := frag.PerLocation[snapshot.Key(oceanside)].Fire
	if o == nil || len(o.Hotspots) != 2 {
		t.Fatalf("Oceanside: the strong fire (two passes clustered) and the unknown-FRP point, got %+v", o)
	}
	// Nearest first: the unknown-FRP GOES point (~5 km), then the strong
	// fire (~7 km) whose two passes clustered to the stronger reading.
	if o.Hotspots[0].FRPMW != nil || *o.Hotspots[0].DistanceKm > *o.Hotspots[1].DistanceKm {
		t.Fatalf("nearest first: %+v", o.Hotspots)
	}
	strong := o.Hotspots[1]
	if strong.FRPMW == nil || *strong.FRPMW != 61.5 || strong.Source.ModelOrStation != "GOES-WEST" || *strong.DistanceKm > 8 {
		t.Fatalf("the cluster keeps the strongest pass: %+v", strong)
	}
	if strong.Source.Provider != "hms" || strong.Confidence != "analyst" {
		t.Fatalf("source and confidence: %+v", strong.Source)
	}
	b := frag.PerLocation[snapshot.Key(boise)].Fire
	if b == nil || len(b.Hotspots) != 0 {
		t.Fatalf("Boise: nothing nearby is an empty (not missing) FireState: %+v", b)
	}
	if _, err := p.Fetch(context.Background(), snapshot.FetchReq{Kind: snapshot.KindObs}); err == nil {
		t.Fatal("the kind is checked")
	}
}

func TestParseRefusesNonKMLAndSkipsBadPlacemarks(t *testing.T) {
	// Red-team B5 F6/F7: an HTML outage page must not read as "no fires";
	// one malformed placemark must not lose the continent.
	if _, err := Parse([]byte("<html><body>maintenance</body></html>")); err == nil || !strings.Contains(err.Error(), "not <kml>") {
		t.Fatalf("HTML must be refused: %v", err)
	}
	if _, err := Parse([]byte("   ")); err == nil {
		t.Fatal("an empty body is an error, not zero fires")
	}
	kml := `<?xml version="1.0"?><kml><Document>
<Placemark><description><![CDATA[Lon: -117.1<br>Lat: 33.1<br>YearDay: 2026237<br>Time: 0201UTC<br>Satellite: GOES-WEST<br>Method: NGFS<br>FRP: 12.5MW]]></description><Point><coordinates>-117.1,33.1,0</coordinates></Point></Placemark>
<Placemark><Point><coordinates>bad</coordinates></Point><description><![CDATA[Lon: x<br>Lat: y]]></description></Placemark>
<Placemark><description><![CDATA[Lon: -117.2<br>Lat: 33.2]]></description><Point><coordinates>-117.2,33.2,0</coordinates></Point></Placemark>
</Document></kml>`
	pts, err := Parse([]byte(kml))
	if err != nil || len(pts) != 2 {
		t.Fatalf("two good points survive the bad one: %d, %v", len(pts), err)
	}
}

func TestParseCapCountsPlacemarksAndSaysSo(t *testing.T) {
	// Red-team B5 F2: the cap counts placemarks (not XML tokens) and hitting
	// it is reported with what was read.
	var b strings.Builder
	b.WriteString("<kml><Document>\n")
	pm := "<Placemark>\n  <description><![CDATA[Lon: -117.1<br>Lat: 33.1]]></description>\n  <Point><coordinates>-117.1,33.1,0</coordinates></Point>\n</Placemark>\n"
	for i := 0; i < maxPlacemarks+5; i++ {
		b.WriteString(pm)
	}
	b.WriteString("</Document></kml>")
	pts, err := Parse([]byte(b.String()))
	if !errors.Is(err, ErrTruncated) {
		t.Fatalf("over the cap: ErrTruncated, got %v", err)
	}
	if len(pts) != maxPlacemarks {
		t.Fatalf("exactly the cap is read (whitespace tokens do not count): %d", len(pts))
	}
}

func TestParsedIsMemoizedByContent(t *testing.T) {
	// Red-team B5 P1: fifty RECENT schedulers ask for the same archive — it
	// is parsed once per change, not once per asker.
	kml := []byte(`<kml><Document><Placemark><description><![CDATA[Lon: -117.1<br>Lat: 33.1]]></description><Point><coordinates>-117.1,33.1,0</coordinates></Point></Placemark></Document></kml>`)
	p := New(nil, "", fire.DefaultRules())
	a, err := p.parsed(kml)
	if err != nil || len(a) != 1 {
		t.Fatal(err)
	}
	b, _ := p.parsed(kml)
	if &a[0] != &b[0] {
		t.Fatal("same bytes must return the memoized points")
	}
	c, _ := p.parsed(append([]byte(nil), append(kml, ' ')...))
	if len(c) != 1 || &a[0] == &c[0] {
		t.Fatal("different bytes parse afresh")
	}
}
