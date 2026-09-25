package app

// mapfeed_test.go — 0.18.0 W5: the alerts the map draws, from the recorded M1
// scenarios (W0.2). Zones are served from the scenario's recorded files on a
// loopback server, as the zone store's own tests do; the withheld zone is
// not served.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// m1Fixture loads a recorded M1 scenario: its place, its alerts as watchpost
// holds them, and a zone server that answers every zone the scenario did not
// withhold.
func m1Fixture(t *testing.T, name string) (snapshot.Location, *httptest.Server) {
	t.Helper()
	dir := filepath.Join("testdata", "maps", "m1", name)
	var sc struct {
		Place struct {
			Name string  `json:"name"`
			Lat  float64 `json:"lat"`
			Lon  float64 `json:"lon"`
		} `json:"place"`
		Withheld []string `json:"withheld_zones"`
	}
	b, err := os.ReadFile(filepath.Join(dir, "scenario.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &sc); err != nil {
		t.Fatal(err)
	}
	var fc struct {
		Features []struct {
			Geometry   json.RawMessage `json:"geometry"`
			Properties struct {
				ID        string    `json:"id"`
				Event     string    `json:"event"`
				Severity  string    `json:"severity"`
				AreaDesc  string    `json:"areaDesc"`
				Effective time.Time `json:"effective"`
				Expires   time.Time `json:"expires"`
				Sender    string    `json:"senderName"`
				Geocode   struct {
					UGC []string `json:"UGC"`
				} `json:"geocode"`
			} `json:"properties"`
		} `json:"features"`
	}
	b, err = os.ReadFile(filepath.Join(dir, "alerts.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &fc); err != nil {
		t.Fatal(err)
	}
	loc := snapshot.Location{Label: sc.Place.Name, Lat: sc.Place.Lat, Lon: sc.Place.Lon}
	for _, f := range fc.Features {
		p := f.Properties
		a := snapshot.Alert{ID: p.ID, Event: p.Event, Severity: p.Severity, AreaDesc: p.AreaDesc,
			Effective: p.Effective, Expires: p.Expires, SenderName: p.Sender, AffectedZones: p.Geocode.UGC}
		if len(f.Geometry) > 0 && string(f.Geometry) != "null" {
			if a.Area, err = geo.ReadGeometry(f.Geometry); err != nil { // the alert's own polygon, as the NWS reader keeps it
				t.Fatal(err)
			}
		}
		loc.Alerts = append(loc.Alerts, a)
	}
	withheld := map[string]bool{}
	for _, z := range sc.Withheld {
		withheld[z] = true
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		body, err := os.ReadFile(filepath.Join(dir, "zones", id+".json"))
		if err != nil || withheld[id] {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return loc, srv
}

// feedFor is the map's feed for one scenario, with the place's own zones as
// given.
func feedFor(t *testing.T, name string, placeZones []string) (snapshot.Location, tty.MapFeed) {
	t.Helper()
	loc, srv := m1Fixture(t, name)
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)}
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{loc}}
	return loc, lp.mapFeedWith(context.Background(), mapInputs{snap: snap, place: &loc}, func(snapshot.Location) []string { return placeZones })
}

// TestEveryAlertIsOneOverlay is W5.2 (FR-4.2, D-55): one overlay per alert,
// one feature per area, the severity carried as data and in the role, the
// times carried; and the library reads it back as the same alert.
func TestEveryAlertIsOneOverlay(t *testing.T) {
	loc, feed := feedFor(t, "01-covers-oak-ridge", nil)
	if len(feed.Overlays) != len(loc.Alerts) || len(loc.Alerts) == 0 {
		t.Fatalf("%d overlays for %d alerts", len(feed.Overlays), len(loc.Alerts))
	}
	m, err := tuimaps.New(tuimaps.WithSize(69, 12))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	for i, o := range feed.Overlays {
		a := loc.Alerts[i]
		if len(o.Features) == 0 {
			t.Fatalf("%s has no features", a.Event)
		}
		wantSev, wantRole := map[string]tuimaps.Severity{"Extreme": tuimaps.SeverityExtreme, "Severe": tuimaps.SeveritySevere,
			"Moderate": tuimaps.SeverityModerate, "Minor": tuimaps.SeverityMinor}[a.Severity], map[string]tuimaps.Token{
			"Extreme": tuimaps.AlertExtreme, "Severe": tuimaps.AlertSevere, "Moderate": tuimaps.AlertModerate, "Minor": tuimaps.AlertMinor}[a.Severity]
		for _, f := range o.Features {
			if f.ID != a.ID || f.Label != a.Event || !f.Expires.Equal(a.Expires) || f.Severity != wantSev || f.Role != wantRole || wantSev == 0 {
				t.Errorf("a feature of %s (%s) carries %q %q %v severity %v role %v", a.Event, a.Severity, f.ID, f.Label, f.Expires, f.Severity, f.Role)
			}
		}
		if !o.Valid.Add(o.Keeps).Equal(a.Expires) {
			t.Errorf("%s is kept until %v; it expires %v", a.Event, o.Valid.Add(o.Keeps), a.Expires)
		}
		if _, err := m.Set(o); err != nil {
			t.Fatalf("the library refused %s: %v", a.Event, err)
		}
	}
	rep, err := m.Report([]tuimaps.Place{{Name: loc.Label, At: tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Alerts) == 0 || rep.Alerts[0].Severity == 0 || !strings.Contains(rep.Alerts[0].Label, loc.Alerts[0].Event) {
		t.Errorf("the library's reading lost the alert or its severity: %+v", rep.Alerts)
	}
}

// TestAPartialAreaIsDrawnAsFoundAndSaid is W5.4, M4's instrument (FR-4.4,
// D-42, specimen 31): the Flood Watch over Fort Davis is drawn from the four
// zones that came, its label says so, and the note names the missing zone in
// words and says Fort Davis lies in it.
func TestAPartialAreaIsDrawnAsFoundAndSaid(t *testing.T) {
	loc, feed := feedFor(t, "05-partial-fort-davis", []string{"TXZ277", "TXC043"})
	if len(feed.Overlays) != 1 {
		t.Fatalf("%d overlays; want the Flood Watch", len(feed.Overlays))
	}
	o := feed.Overlays[0]
	if len(o.Features) == 0 {
		t.Fatal("the partial area was withheld instead of drawn as found")
	}
	if !strings.Contains(o.Features[0].Label, "4 of 5 zones") {
		t.Errorf("the label %q does not say how much is drawn", o.Features[0].Label)
	}
	notes := strings.Join(feed.Notes, "\n")
	if !strings.Contains(notes, "Davis Mountains") || strings.Contains(notes, "TXZ277") {
		t.Errorf("the note does not name the missing zone in words: %q", notes)
	}
	if !strings.Contains(notes, loc.Label+" lies in it") {
		t.Errorf("the note does not say the place lies in the missing zone: %q", notes)
	}
	_, outside := feedFor(t, "05-partial-fort-davis", []string{"TXZ280", "TXC377"})
	if n := strings.Join(outside.Notes, "\n"); !strings.Contains(n, "does not lie in it") {
		t.Errorf("a place outside the missing zone is not said to be: %q", n)
	}
}

// TestTheFeedIsReachedFromProduction is W5.1 (FR-4.1, C-1): the composition
// root hands the window the feed, through a livePipelines method.
func TestTheFeedIsReachedFromProduction(t *testing.T) {
	lp := &livePipelines{maps: newMapBuilder("t", t.TempDir(), &recorded{offline: true}, nil)}
	cfg := lp.ttyConfig("t", Options{}, false, config.Config{}, nil, nil, nil, nil, nil, nil)
	if cfg.MapFeed == nil {
		t.Fatal("the window is handed no alert feed")
	}
}

// TestAnAlertOnTwoPlacesIsOneOverlay is W5.2 (FR-4.2): an alert attached to
// every location it covers is drawn once.
func TestAnAlertOnTwoPlacesIsOneOverlay(t *testing.T) {
	loc, srv := m1Fixture(t, "01-covers-oak-ridge")
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)}
	twin := loc
	twin.Label = "Knoxville, TN"
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{loc, twin}}
	feed := lp.mapFeedWith(context.Background(), mapInputs{snap: snap, place: &loc}, func(snapshot.Location) []string { return nil })
	if len(feed.Overlays) != len(loc.Alerts) {
		t.Errorf("%d overlays for %d alerts on two places", len(feed.Overlays), len(loc.Alerts))
	}
}

// TestEverySeverityHasItsRole is W5.2 (FR-4.2): CAP's five severities map one
// to one onto the library's, and onto the role each is drawn in.
func TestEverySeverityHasItsRole(t *testing.T) {
	for cap, want := range map[string]struct {
		sev  tuimaps.Severity
		role tuimaps.Token
	}{
		"Extreme": {tuimaps.SeverityExtreme, tuimaps.AlertExtreme}, "Severe": {tuimaps.SeveritySevere, tuimaps.AlertSevere},
		"Moderate": {tuimaps.SeverityModerate, tuimaps.AlertModerate}, "Minor": {tuimaps.SeverityMinor, tuimaps.AlertMinor},
		"Unknown": {tuimaps.SeverityUnknown, tuimaps.AlertUnknown}, "": {tuimaps.SeverityUnknown, tuimaps.AlertUnknown},
	} {
		if sev, role := severityOf(cap); sev != want.sev || role != want.role {
			t.Errorf("%q is %v in %v, want %v in %v", cap, sev, role, want.sev, want.role)
		}
	}
}

// TestTheDescriptionAnswersTheM1Key is W1.4's acceptance through the real
// parts (W9.2's answer-key test, D-53): for every recorded scenario the
// feed's overlays, set on a map built as the station builds it, give the
// library's Report, and the description's word for each alert - covers,
// stops short, lies to one side - is the scenario's answer. The key is
// computed from the recorded geometry; the HUM LEAD's confirmation of it is
// owed, and M1b is scored on the description that ships.
func TestTheDescriptionAnswersTheM1Key(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("testdata", "maps", "m1", "*", "scenario.json"))
	if err != nil || len(dirs) < 8 {
		t.Fatalf("%d scenarios found: %v", len(dirs), err)
	}
	scored := 0
	for _, p := range dirs {
		name := filepath.Base(filepath.Dir(p))
		var sc struct {
			Failure json.RawMessage `json:"failure"`
			Key     map[string]struct {
				Answer string `json:"answer"`
			} `json:"answer_key"`
			Place struct {
				Zones []string `json:"zones"`
			} `json:"place"`
		}
		b, _ := os.ReadFile(p)
		if err := json.Unmarshal(b, &sc); err != nil {
			t.Fatal(err)
		}
		if len(sc.Failure) > 0 && string(sc.Failure) != "null" {
			continue // the failure scenario is the offline one: nothing to answer
		}
		t.Run(name, func(t *testing.T) {
			loc, srv := m1Fixture(t, name)
			lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)}
			snap := &snapshot.Snapshot{Locations: []snapshot.Location{loc}}
			feed := lp.mapFeedWith(context.Background(), mapInputs{snap: snap, place: &loc}, func(snapshot.Location) []string { return placeZonesOf(name) })
			m, err := newMapBuilder("t", t.TempDir(), &recorded{offline: true}, nil).build(tuimaps.Size{Cols: 69, Rows: 12})
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			m.Units(false, false)
			for _, o := range feed.Overlays {
				if _, err := m.Set(o); err != nil {
					t.Fatal(err)
				}
			}
			rep, err := m.Report([]tuimaps.Place{{Name: loc.Label, At: tuimaps.LonLat{Lon: loc.Lon, Lat: loc.Lat}}})
			if err != nil || len(rep.Places) != 1 {
				t.Fatalf("report: %v", err)
			}
			got := map[string]string{}
			for _, pa := range rep.Places[0].Alerts {
				got[pa.Feature] = tty.Relation(pa, feed.InMissing[pa.Feature])
			}
			for id, want := range sc.Key {
				if rel, ok := got[id]; !ok || !strings.HasPrefix(rel, want.Answer) {
					t.Errorf("%s: the description says %q, the key says %q", id, rel, want.Answer)
				}
				scored++
			}
		})
	}
	if scored == 0 {
		t.Fatal("no answer was scored, so this proves nothing")
	}
}

// placeZonesOf is each scenario place's own zone codes, as the weather
// service gives them: only the partial scenario needs them, to know its place
// lies in the withheld zone.
func placeZonesOf(name string) []string {
	if name == "05-partial-fort-davis" {
		return []string{"TXZ277", "TXC043"}
	}
	return nil
}
