package app

// mapfire_test.go — 0.18.0 D-121: the map's fire - the active perimeters,
// each named incident, the satellite hotspots - one layer, on by default.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/domains/fire/hms"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func pf(v float64) *float64 { return &v }

// fireView is a view over southern California.
var fireView = tty.MapView{W: -121, S: 32, E: -114, N: 36}

// someFire is one of each: a perimeter with a hole, an incident in view and
// one far out of it, and hotspots strong, weak, too weak and unmeasured.
func someFire() fireInView {
	outline := [][2]float64{{-117.2, 33.4}, {-117.0, 33.4}, {-117.0, 33.6}, {-117.2, 33.6}, {-117.2, 33.4}}
	hole := [][2]float64{{-117.15, 33.45}, {-117.1, 33.45}, {-117.1, 33.5}, {-117.15, 33.45}}
	return fireInView{
		perimeters: []wfigs.Perimeter{{Name: "Timber", Acres: pf(12915), Contained: pf(26), Areas: [][][][2]float64{{outline, hole}}}},
		incidents: []snapshot.Incident{{Name: "Timber", Lat: 33.5, Lon: -117.1, Acres: pf(12915), PercentContained: pf(26)},
			{Name: "Bug", Lat: 39.72, Lon: -120.03, Acres: pf(93733)}},
		hotspots: []hms.Point{{Lat: 33.5, Lon: -117.05, FRPMW: pf(80)}, {Lat: 33.52, Lon: -117.06, FRPMW: pf(12)},
			{Lat: 33.53, Lon: -117.07, FRPMW: pf(2)}, {Lat: 33.54, Lon: -117.08}},
	}
}

// TestFireIsDrawnFromItsThreeSources is D-121: the perimeter an outline and
// its hole, in fire's role; the incident in view a marker with its name,
// acres and containment, the one out of view not drawn; the hotspots the
// rules keep as dots, the strong in fire's role and the rest fainter; each
// credited; none an alert.
func TestFireIsDrawnFromItsThreeSources(t *testing.T) {
	got := fireOverlays(someFire(), fireView, fire.DefaultRules())
	var perimeter, incidents, hotspots []tuimaps.Feature
	for _, o := range got {
		if !strings.HasPrefix(o.ID, tty.FireLayer+"/") || o.Credit == "" {
			t.Errorf("%s: credit %q; want a fire overlay, credited", o.ID, o.Credit)
		}
		for _, f := range o.Features {
			if f.Role != tuimaps.Fire && f.Role != tuimaps.FireFaint {
				t.Errorf("%s has a feature in role %v; fire is drawn in its own", o.ID, f.Role)
			}
			switch {
			case f.Kind == tuimaps.Polygon:
				perimeter = append(perimeter, f)
			case strings.HasSuffix(o.ID, "/incidents"):
				incidents = append(incidents, f)
			default:
				hotspots = append(hotspots, f)
			}
		}
	}
	if len(perimeter) != 1 || len(perimeter[0].Rings) != 2 {
		t.Errorf("the perimeter is %d areas; want one, its outline and its hole", len(perimeter))
	}
	if len(incidents) != 1 || incidents[0].Label != "Timber 12,915 ac, 26%" {
		t.Errorf("the incidents drawn are %+v; want Timber alone, labelled with its acres and containment", incidents)
	}
	strong, faint := 0, 0
	for _, h := range hotspots {
		if h.Role == tuimaps.Fire {
			strong++
		} else {
			faint++
		}
	}
	if strong != 1 || faint != 2 {
		t.Errorf("the hotspots are %d strong and %d faint; want the 80 MW strong, the 12 MW and the unmeasured faint, the 2 MW dropped", strong, faint)
	}
}

// TestFireIsOnByDefaultAndItsPerimetersAreCosted is D-121 with D-23: the Fire
// row on, its perimeters a request a field box counted; the incidents and
// hotspots are what watchpost already fetches.
func TestFireIsOnByDefaultAndItsPerimetersAreCosted(t *testing.T) {
	for _, l := range mapLayers {
		if l.key == tty.FireLayer {
			b, r := l.cost(mapInputs{region: geo.RegionContiguous, view: fireView})
			if !l.on || r != len(fieldBoxes(geo.RegionContiguous, fireView)) || b <= 0 {
				t.Errorf("fire is on %v, costs %d bytes in %d requests; want on, a request a field box", l.on, b, r)
			}
			return
		}
	}
	t.Fatal("fire is not registered")
}

// TestTheFeedCarriesTheFire is D-121's wiring: the feed, asked from the
// composition root's providers, carries the perimeters, incidents and
// hotspots, each timed as a thing that is so now - drawn through the loop,
// and on Now alone in Forecast mode.
func TestTheFeedCarriesTheFire(t *testing.T) {
	perims, err := os.ReadFile("../domains/fire/wfigs/testdata/perimeters.json")
	if err != nil {
		t.Fatal(err)
	}
	incidents := `{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-120.9,44.2]},"properties":{"IncidentName":"Miners","IncidentSize":150,"PercentContained":0,"IncidentTypeCategory":"WF"}}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("geometry") != "" {
			_, _ = w.Write(perims)
			return
		}
		_, _ = w.Write([]byte(incidents))
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	lp := &livePipelines{fire: []snapshot.Provider{wfigs.New(c, srv.URL+"/query", fire.DefaultRules())}, rules: fire.DefaultRules()}
	ask := tty.MapAsk{Snap: &snapshot.Snapshot{}, Region: geo.RegionContiguous, View: tty.MapView{W: -125, S: 42, E: -116, N: 49}}
	feed := lp.mapFeedWith(context.Background(), lp.mapInputsFetching(context.Background(), ask), func(snapshot.Location) []string { return nil })
	n := 0
	for _, o := range feed.Overlays {
		if strings.HasPrefix(o.ID, tty.FireLayer+"/") {
			n += len(o.Features)
			if tm := feed.Times[o.ID]; !tm.Happened || !tm.From.IsZero() {
				t.Errorf("%s is timed %+v; want a thing so now, from always", o.ID, tm)
			}
		}
	}
	if n < 5 {
		t.Errorf("the feed carries %d fire features; want the fixture's four perimeters and its incident", n)
	}
}

// TestTheStatusWindowNamesTheFiresHosts is FR-9.4 for D-121: the map's
// fire hosts are in the Status window's list - the perimeters' and
// incidents' host, and the hotspots'.
func TestTheStatusWindowNamesTheFiresHosts(t *testing.T) {
	hosts := map[string]bool{}
	for _, s := range mapSourceList() {
		hosts[s.Host] = true
	}
	for _, h := range []string{"services3.arcgis.com", "www.ospo.noaa.gov"} {
		if !hosts[h] {
			t.Errorf("the Status window's map list lacks %s: %v", h, hosts)
		}
	}
}

// TestAPerimeterInTwoBoxesIsDrawnOnce: Alaska's map is two boxes, split at
// the antimeridian; a perimeter in both answers is one fire.
func TestAPerimeterInTwoBoxesIsDrawnOnce(t *testing.T) {
	perims, err := os.ReadFile("../domains/fire/wfigs/testdata/perimeters.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("geometry") != "" {
			_, _ = w.Write(perims)
			return
		}
		_, _ = w.Write([]byte(`{"type":"FeatureCollection","features":[]}`))
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	lp := &livePipelines{fire: []snapshot.Provider{wfigs.New(c, srv.URL+"/query", fire.DefaultRules())}}
	ask := tty.MapAsk{Region: geo.RegionAlaska}
	if n := len(fieldBoxes(ask.Region, ask.View)); n != 2 {
		t.Fatalf("Alaska is %d field boxes; the test needs its two", n)
	}
	if got := lp.fireIn(context.Background(), ask); len(got.perimeters) != 4 {
		t.Errorf("%d perimeters from two boxes' answers; want the four fires once each", len(got.perimeters))
	}
}
