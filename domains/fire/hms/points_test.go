package hms

// points_test.go — 0.18.0 D-121: the map is handed every detection, however
// far from the station's places, from the places' own coalesced read.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

func TestTheMapReadsEveryHotspot(t *testing.T) {
	asked := 0
	body := kmz(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked++
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	p := New(c, srv.URL+"/fireAllSats.kmz", fire.DefaultRules())
	pts, err := p.Points(context.Background())
	if err != nil || len(pts) != 4 {
		t.Fatalf("%d detections (%v); want all four, the one in Canada too", len(pts), err)
	}
	if _, err := p.Points(context.Background()); err != nil || asked != 1 {
		t.Errorf("a second read within the window asked %d times (%v); want the coalesced read", asked, err)
	}
}

// TestATruncatedArchiveStillGivesTheMapItsHotspots: an archive over the
// placemark cap is served as read, without the error the places' fire
// reports - the map draws what was read.
func TestATruncatedArchiveStillGivesTheMapItsHotspots(t *testing.T) {
	var b strings.Builder
	b.WriteString("<kml><Document>\n")
	pm := "<Placemark><description><![CDATA[Lon: -117.1<br>Lat: 33.1]]></description><Point><coordinates>-117.1,33.1,0</coordinates></Point></Placemark>\n"
	for range maxPlacemarks + 1 {
		b.WriteString(pm)
	}
	b.WriteString("</Document></kml>")
	body := []byte(b.String())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) }))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0, MaxBodyBytes: 64 << 20})
	pts, err := New(c, srv.URL+"/fireAllSats.kml", fire.DefaultRules()).Points(context.Background())
	if err != nil || len(pts) != maxPlacemarks {
		t.Errorf("a truncated archive gave %d detections (%v); want the cap's, with no error", len(pts), err)
	}
}
