package app

// mapquakes_test.go — 0.18.0 D-80 (UAT-1 U1-43): the map draws the alerts by
// [w]'s categories, never a forecast; and D-122, D-123: the quakes the
// listener chose, drawn as USGS draws them.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestTheAlertsAreFiledAsTheSevereWindowFilesThem(t *testing.T) {
	square := geo.Shape{{{{Lon: -117.5, Lat: 33.0}, {Lon: -117.3, Lat: 33.0}, {Lon: -117.3, Lat: 33.2}, {Lon: -117.5, Lat: 33.0}}}}
	var alerts []snapshot.Alert
	for i, ev := range []string{"Tornado Warning", "Flood Watch", "Small Craft Advisory", "Special Weather Statement", "Marine Weather Statement", "Hydrologic Outlook", "Air Quality Alert"} {
		alerts = append(alerts, snapshot.Alert{ID: string(rune('a' + i)), Event: ev, Severity: "Moderate", Area: square})
	}
	here := snapshot.Location{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.38}
	in := mapInputs{snap: &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "x", Alerts: alerts}}}, place: &here}
	out := (&livePipelines{}).mapFeedWith(context.Background(), in, func(snapshot.Location) []string { return nil })
	var ids []string
	for _, o := range out.Overlays {
		ids = append(ids, o.ID)
	}
	want := []string{"alert/warnings/a", "alert/watches/b", "alert/advisories/c", "alert/statements/d", "alert/marine/e", "alert/advisories/g"}
	for _, w := range want {
		if !slices.Contains(ids, w) {
			t.Errorf("no %s among %v", w, ids)
		}
	}
	if slices.ContainsFunc(ids, func(id string) bool { return id[len(id)-2:] == "/f" }) {
		t.Errorf("the forecast was drawn: %v", ids)
	}
}

// TestQuakesAreRingsSizedByMagnitudeColouredByAge is D-123: each quake in
// view a ring fixed on the screen, larger with each magnitude as USGS's; its
// colour its age - the past hour, the past day, older; labelled with its
// magnitude and local time, and the day when not today - NEW within the
// radar loop's hours past (D-129); never an alert.
func TestQuakesAreRingsSizedByMagnitudeColouredByAge(t *testing.T) {
	now := time.Date(2026, 9, 29, 21, 0, 0, 0, time.UTC) // a Tuesday
	mag := func(m float64) *globalfeed.QuakeDetail { return &globalfeed.QuakeDetail{Mag: &m} }
	feed := []globalfeed.Event{
		{ID: "hour", Class: globalfeed.ClassQuake, Lat: 33.5, Lon: -117.0, HasPoint: true, At: now.Add(-20 * time.Minute), Quake: mag(4.1)},
		{ID: "day", Class: globalfeed.ClassQuake, Lat: 33.6, Lon: -117.1, HasPoint: true, At: now.Add(-5 * time.Hour), Quake: mag(2.5)},
		{ID: "older", Class: globalfeed.ClassQuake, Lat: 33.7, Lon: -117.2, HasPoint: true, At: now.Add(-50 * time.Hour), Quake: mag(6.2)},
		{ID: "alaska", Class: globalfeed.ClassQuake, Lat: 61.0, Lon: -150.0, HasPoint: true, At: now, Quake: mag(7.0)},
		{ID: "nopoint", Class: globalfeed.ClassQuake, At: now, Quake: mag(5.0)},
	}
	v := tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}
	in := quakesIn(feed, v)
	if len(in) != 3 {
		t.Fatalf("the quakes in view are %d, want three", len(in))
	}
	got := map[string]tuimaps.Feature{}
	o, _ := quakeOverlay(in, now, render.ClockByKey("12h"))
	for _, f := range o.Features {
		got[f.ID] = f
	}
	for id, want := range map[string]struct {
		role  tuimaps.Token
		label string
	}{"hour": {tuimaps.QuakeHour, "NEW M4.1 8:40 PM"}, "day": {tuimaps.QuakeDay, "M2.5 4:00 PM"}, "older": {tuimaps.QuakeOlder, "M6.2 Sun 7:00 PM"}} {
		f := got[id]
		if f.Kind != tuimaps.Circle || f.RadiusKm != 0 || f.RadiusDots == 0 || f.Role != want.role || f.Label != want.label || f.Severity != 0 {
			t.Errorf("%s is drawn %+v; want a ring on the screen in %v, labelled %q, no severity", id, f, want.role, want.label)
		}
	}
	if got["day"].RadiusDots >= got["hour"].RadiusDots || got["hour"].RadiusDots >= got["older"].RadiusDots {
		t.Errorf("the rings are M2.5 %d, M4.1 %d, M6.2 %d dots; want larger with each magnitude", got["day"].RadiusDots, got["hour"].RadiusDots, got["older"].RadiusDots)
	}
	if quakeRingDots(0.5) < 2 || quakeRingDots(10) > 48 {
		t.Errorf("the rings run %d to %d dots; want 2 to 48", quakeRingDots(0.5), quakeRingDots(10))
	}
}

// TestTheMapAsksTheQuakesChosen is D-122's wiring: the feed the listener
// chose, M2.5+ over the past week when none is - not the ticker's
// significant feed - kept to the view, and its size in the cost line.
func TestTheMapAsksTheQuakesChosen(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)
		_, _ = w.Write([]byte(`{"features":[{"id":"q1","properties":{"mag":3.1,"time":1790000000000},"geometry":{"coordinates":[-117.0,33.5,5]}},{"id":"q2","properties":{"mag":4.0,"time":1790000000000},"geometry":{"coordinates":[-150.0,61.0,5]}}]}`))
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	lp := &livePipelines{mapQuakes: newMapQuakes(c, srv.URL+"/")}
	view := tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}
	in := lp.mapInputsFetching(context.Background(), tty.MapAsk{View: view})
	if len(in.quakes) != 1 || in.quakes[0].ID != "q1" || len(asked) != 1 || asked[0] != "/2.5_week.geojson" {
		t.Fatalf("the inputs hold %v, asked %v; want q1 alone, from M2.5+ past week", in.quakes, asked)
	}
	_ = lp.mapInputsFetching(context.Background(), tty.MapAsk{View: view, QuakeFeed: "1.0_day"})
	if asked[len(asked)-1] != "/1.0_day.geojson" {
		t.Errorf("M1.0+ past day asked %v", asked)
	}
	for feed, want := range map[string]int64{"2.5_day": 33_000, "2.5_week": 221_000, "1.0_day": 113_000, "1.0_week": 918_000, "": 221_000} {
		if b, r := quakeLayerCost(mapInputs{region: geo.RegionContiguous, quakeFeed: feed}); b != want || r != 1 {
			t.Errorf("%q costs %d bytes in %d requests; want %d in one", feed, b, r, want)
		}
	}
}

// TestTheStatusWindowNamesTheQuakesHost is FR-9.4 for D-122: the map asks
// USGS itself now, and the Status window says so.
func TestTheStatusWindowNamesTheQuakesHost(t *testing.T) {
	for _, s := range mapSourceList() {
		if s.Host == "earthquake.usgs.gov" && s.Layers == "quakes" { // a MAP STATUS row (D-150)
			return
		}
	}
	t.Error("the Status window's map list does not name earthquake.usgs.gov")
}

// TestEveryQuakeIsAcceptedByTheLibrary is UAT-2 U2-29: a quake of the past
// week handed in as current for eight days was refused - the library keeps
// a thing current for at most seven - and the week's feed flooded the map
// with refusals. Every quake the feeds can hold, a minute old to a full
// week, is handed to a real map and accepted.
func TestEveryQuakeIsAcceptedByTheLibrary(t *testing.T) {
	m, err := tuimaps.New(tuimaps.WithSize(69, 12))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()
	now := time.Now()
	mag := 3.1
	var feed []globalfeed.Event
	for i, age := range []time.Duration{time.Minute, 5 * time.Hour, 3 * 24 * time.Hour, 7*24*time.Hour - time.Minute} {
		feed = append(feed, globalfeed.Event{ID: string(rune('a' + i)), Class: globalfeed.ClassQuake, Lat: 33.5, Lon: -117, HasPoint: true, At: now.Add(-age), Quake: &globalfeed.QuakeDetail{Mag: &mag}})
	}
	o, _ := quakeOverlay(feed, now, render.ClockByKey("12h"))
	if _, err := m.Set(o); err != nil {
		t.Errorf("%s was refused: %v", o.ID, err)
	}
}
