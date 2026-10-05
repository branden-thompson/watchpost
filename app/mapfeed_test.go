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
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// m1Fixture loads a recorded M1 scenario: its place, its alerts as watchpost
// holds them, and a zone server that answers every zone the scenario did not
// withhold. A scenario whose failure is "offline" has the network down: its
// zone server is closed, so every zone asked is refused, as the alerts
// watchpost already holds are kept.
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
		Failure  string   `json:"failure"`
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
	switch sc.Failure {
	case "":
	case "offline":
		srv.Close()
	default:
		t.Fatalf("%s: no way to apply the failure %q", name, sc.Failure)
	}
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
// zones that came, its label says so, and the diagnostics' words name the
// missing zone and say Fort Davis lies in it - never a note under the map
// (D-124, D-240).
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
	if len(feed.Notes) != 0 {
		t.Errorf("an undrawn area is a note under the map: %q", feed.Notes)
	}
	notes := strings.Join(feed.Undrawn, "\n")
	if !strings.Contains(notes, "Davis Mountains") || strings.Contains(notes, "TXZ277") {
		t.Errorf("the note does not name the missing zone in words: %q", notes)
	}
	if !strings.Contains(notes, loc.Label+" lies in it") {
		t.Errorf("the note does not say the place lies in the missing zone: %q", notes)
	}
	_, outside := feedFor(t, "05-partial-fort-davis", []string{"TXZ280", "TXC377"})
	if n := strings.Join(outside.Undrawn, "\n"); !strings.Contains(n, "does not lie in it") {
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
// parts (W9.2's answer-key test, D-53): for every recorded scenario, its
// failure applied, the feed is built from the recorded alerts and zones and
// handed to the window, whose map is built as the station builds it; the
// window's description - the words the listener reads - is then held to the
// key. Each sentence's relation word must be the key's exactly, every alert
// the description names must be in the key, and every alert in the key must
// be named. A covering alert whose zone holding the place was withheld is
// "covers-missing": it covers, and the description says its outline could
// not be drawn. The key is computed from the recorded geometry; the HUM LEAD's
// confirmation of it is owed, and M1b is scored on the description that
// ships.
func TestTheDescriptionAnswersTheM1Key(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("testdata", "maps", "m1", "*", "scenario.json"))
	if err != nil || len(dirs) < 9 {
		t.Fatalf("%d scenarios found: %v", len(dirs), err)
	}
	var scored atomic.Int32
	t.Run("scenarios", func(t *testing.T) {
		for _, p := range dirs {
			name := filepath.Base(filepath.Dir(p))
			var sc struct {
				Key map[string]struct {
					Answer string `json:"answer"`
					Event  string `json:"event"`
					Area   string `json:"area"`
				} `json:"answer_key"`
				Withheld []string `json:"withheld_zones"`
			}
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &sc); err != nil {
				t.Fatal(err)
			}
			if len(sc.Key) == 0 {
				t.Fatalf("%s has no answer key", name)
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel() // each waits out the window's clock tick
				loc, said := describedAlerts(t, name)
				used := map[string]bool{}
				ids := make([]string, 0, len(sc.Key))
				for id := range sc.Key {
					ids = append(ids, id)
				}
				sort.Strings(ids) // one order every run: map order differed between machines
				wantOf := func(id string) string {
					want := sc.Key[id].Answer
					if want == "covers" && coversByWithheldZone(loc, id, sc.Withheld, placeZonesOf(name)) {
						want = "covers-missing"
					}
					return want
				}
				claim := func(s describedAlert, byArea bool) bool {
					fits := func(id string) bool {
						k := sc.Key[id]
						return !used[id] && k.Event == s.event && (!byArea || strings.EqualFold(keyAreaWords(k.Area), s.areas))
					}
					// TWO ALERTS OF ONE EVENT OVER ONE AREA (03, Harlan) ARE TOLD APART BY
					// THEIR ANSWER: the sentence takes the key it agrees with where one
					// fits, and only then any key that fits - which it then fails.
					pick := ""
					for _, id := range ids {
						if fits(id) && wantOf(id) == s.relation {
							pick = id
							break
						}
					}
					for _, id := range ids {
						if pick == "" && fits(id) {
							pick = id
						}
					}
					for _, id := range []string{pick} {
						if id == "" {
							continue
						}
						k := sc.Key[id]
						used[id] = true
						want := wantOf(id)
						if s.relation != want {
							t.Errorf("%s (%s): the description says %q - %q; the key says %q", k.Event, id, s.relation, s.sentence, want)
						}
						scored.Add(1)
						return true
					}
					return false
				}
				var unplaced []describedAlert
				for _, s := range said { // an alert named by its areas is matched on them
					if s.areas == "" || !claim(s, true) {
						unplaced = append(unplaced, s)
					}
				}
				for _, s := range unplaced { // "this area" names no area: matched on its event
					if !claim(s, false) {
						t.Errorf("the description names an alert that is not in the key: %q", s.sentence)
					}
				}
				for id, k := range sc.Key {
					if !used[id] {
						t.Errorf("%s (%s, %s) is in the key and the description does not name it", k.Event, id, k.Answer)
					}
				}
			})
		}
	})
	if scored.Load() < int32(len(dirs)) {
		t.Fatalf("%d answers scored over %d scenarios, so this proves too little", scored.Load(), len(dirs))
	}
}

// describedAlert is one alert sentence of the description: the sentence, the
// event it names, the areas it names ("" for "this area") and its relation in
// M1's words.
type describedAlert struct {
	sentence, event, areas, relation string
}

// alertSentence is the description's sentence for an alert (D-74):
// "<event> in effect for <where>", then its end, if it has one.
var alertSentence = regexp.MustCompile(`^(.+?) in effect for (.+?)(?: until .+)?\.$`)

// describedAlerts opens the map window on a scenario's place, the feed built
// through the real parts with the scenario's failure applied, the
// description shown in place of the picture; and reads the description's
// alert sentences back from the window's frame.
func describedAlerts(t *testing.T, name string) (snapshot.Location, []describedAlert) {
	t.Helper()
	loc, srv := m1Fixture(t, name)
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)}
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{loc}}
	b := newMapBuilder("t", t.TempDir(), &recorded{offline: true}, nil)
	m, err := tty.NewDashboard(tty.Config{Version: "t", NewMap: b.build, MapDescription: "instead",
		MapFeed: func(ctx context.Context, _ tty.MapAsk) tty.MapFeed {
			return lp.mapFeedWith(ctx, mapInputs{snap: snap, place: &loc}, func(snapshot.Location) []string { return placeZonesOf(name) })
		}})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 240, Height: 60})
	model, _ = model.Update(tty.SnapshotMsg{Snap: snap})
	model, cmd := model.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	model = runCommands(model, cmd)
	frame := ansiSeq.ReplaceAllString(model.View().Content, "")
	var out []describedAlert
	for _, line := range strings.Split(frame, "\n") {
		cell := boxCell.FindStringSubmatch(line)
		if cell == nil {
			continue
		}
		s := strings.TrimSpace(cell[1])
		parts := alertSentence.FindStringSubmatch(s)
		if parts == nil {
			continue
		}
		d := describedAlert{sentence: s, event: parts[1]}
		switch where := parts[2]; {
		case where == "this area; its outline could not be drawn":
			d.relation = "covers-missing"
		case where == "this area":
			d.relation = "covers"
		case strings.HasPrefix(where, "nearby "):
			d.relation, d.areas = "stops short", strings.TrimPrefix(where, "nearby ")
		default:
			d.relation, d.areas = "lies to one side", where
		}
		out = append(out, d)
	}
	return loc, out
}

// coversByWithheldZone reports whether an alert names a withheld zone that
// holds the place.
func coversByWithheldZone(loc snapshot.Location, id string, withheld, placeZones []string) bool {
	for _, a := range loc.Alerts {
		if a.ID != id {
			continue
		}
		for _, z := range a.AffectedZones {
			if slices.Contains(withheld, z) && slices.Contains(placeZones, z) {
				return true
			}
		}
	}
	return false
}

// ansiSeq is a terminal escape sequence; boxCell is a line's text inside the
// map window's border, where it holds an alert sentence.
var (
	ansiSeq = regexp.MustCompile(`\x1b\[[0-9;:?]*[A-Za-z]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
	boxCell = regexp.MustCompile(`│ ([^│]* in effect for [^│]*?)\s*│`)
)

// keyAreaWords is a key's area as the description names it: its first two
// places joined by "and" (D-74); case aside, as the description lowers the
// Weather Service's generic words.
func keyAreaWords(area string) string {
	var places []string
	for _, p := range strings.Split(area, ";") {
		if p = strings.TrimSpace(p); p != "" && len(places) < 2 {
			places = append(places, p)
		}
	}
	return strings.Join(places, " and ")
}

// runCommands runs a window's commands as Bubble Tea would, each one's message
// fed back through Update, until none is outstanding. A command that has not
// answered within cmdWait is a timer - the clock's tick - and its message is
// dropped, which ends its chain.
func runCommands(model tea.Model, cmd tea.Cmd) tea.Model {
	const cmdWait = 3 * time.Second
	results := make(chan tea.Msg)
	pending := 0
	launch := func(c tea.Cmd) {
		if c == nil {
			return
		}
		pending++
		go func() {
			done := make(chan tea.Msg, 1)
			go func() { done <- c() }()
			select {
			case msg := <-done:
				results <- msg
			case <-time.After(cmdWait):
				results <- nil
			}
		}()
	}
	launch(cmd)
	for pending > 0 {
		msg := <-results
		pending--
		switch m := msg.(type) {
		case nil:
		case tea.BatchMsg:
			for _, c := range m {
				launch(c)
			}
		default:
			var next tea.Cmd
			model, next = model.Update(m)
			launch(next)
		}
	}
	return model
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

// TestAnUncheckedCategoryIsGoneFromTheMap is D-160 ("unchecked = gone"): an
// alert whose category is unchecked is neither drawn, noted, named in view
// nor fetched - its zones are not asked; with the whole layer off, no alert
// is. Before, the feed resolved them all and the window dropped the drawing
// only, keeping the notes and the names.
func TestAnUncheckedCategoryIsGoneFromTheMap(t *testing.T) {
	loc, srv := m1Fixture(t, "04-watch-and-warning-roswell")
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{loc}}
	var off []string
	for _, a := range loc.Alerts {
		if cat, ok := alertCategory(a); ok {
			off = append(off, cat)
		}
	}
	if len(off) == 0 {
		t.Fatal("control: the fixture has no alert the map draws")
	}
	for name, in := range map[string]mapInputs{
		"every category unchecked": {snap: snap, place: &loc, alertCategoriesOff: off},
		"the layer off":            {snap: snap, place: &loc, alertsOff: true},
	} {
		lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)}
		feed := lp.mapFeedWith(context.Background(), in, func(snapshot.Location) []string { return nil })
		if len(feed.Overlays) != 0 || len(feed.Notes) != 0 || len(feed.InView) != 0 {
			t.Errorf("%s: %d overlays, %d notes, %d named in view; want none", name, len(feed.Overlays), len(feed.Notes), len(feed.InView))
		}
		if got := lp.zoneShapes.Stats().Fetched; got != 0 {
			t.Errorf("%s: %d zones fetched for alerts the map will not draw", name, got)
		}
	}
	// Through the ask, as the dashboard sends it: the switches reach the inputs.
	if in := (&livePipelines{}).inputsFor(context.Background(), tty.MapAsk{AlertsOff: true, AlertCategoriesOff: off}, false); !in.alertsOff || len(in.alertCategoriesOff) != len(off) {
		t.Errorf("the ask's switches did not reach the inputs: layer off %v, categories off %v", in.alertsOff, in.alertCategoriesOff)
	}
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)} // control: all on, the same fixture draws - by its zones
	if feed := lp.mapFeedWith(context.Background(), mapInputs{snap: snap, place: &loc}, func(snapshot.Location) []string { return nil }); len(feed.Overlays) == 0 || lp.zoneShapes.Stats().Fetched == 0 {
		t.Error("control: with every category on the fixture draws nothing, or asks no zones, so this proves nothing")
	}
}

// TestAClosedMapLetsItsZonesGo is D-162: zone geometry is held once. The
// map's own outlines keep a warm reopen instant; the zone store's parsed copy
// is let go when the map closes - and the next ask still gets its zones, from
// the HTTP cache's copy, not from nothing.
func TestAClosedMapLetsItsZonesGo(t *testing.T) {
	loc, srv := m1Fixture(t, "04-watch-and-warning-roswell")
	lp := &livePipelines{zoneShapes: zoneStore(t, srv.URL)}
	in := mapInputs{snap: &snapshot.Snapshot{Locations: []snapshot.Location{loc}}, place: &loc}
	first := lp.mapFeedWith(context.Background(), in, func(snapshot.Location) []string { return nil })
	if lp.zoneShapes.Stats().Held == 0 || len(first.Overlays) == 0 {
		t.Fatal("control: the feed held no zones, so there is nothing to let go")
	}
	lp.mapClosed()
	if held := lp.zoneShapes.Stats().Held; held != 0 {
		t.Errorf("the map closed and the zone store still holds %d shapes", held)
	}
	again := lp.mapFeedWith(context.Background(), in, func(snapshot.Location) []string { return nil })
	if len(again.Overlays) != len(first.Overlays) {
		t.Errorf("reopened, the feed drew %d alerts where it drew %d: the zones did not come back", len(again.Overlays), len(first.Overlays))
	}
}
