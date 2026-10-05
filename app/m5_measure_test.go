package app

// m5_measure_test.go — M5 MEASURED IN PROCESS (D-280, D-46, D-251).
//
//	WATCHPOST_VALIDATE_M5=1 go test -run '^TestMeasureM5InProcess$' -count=1 -v -timeout 20m ./app
//
// It runs only under WATCHPOST_VALIDATE_M5=1, so neither `make verify` nor CI
// ever runs it: its numbers are read, never asserted.
//
// Twenty cold opens at 149x38. Each open builds its own station: the real
// tty Dashboard under its Router in a Bubble Tea program, the renderer
// writing to nowhere, wired by the app's own mapConfig to the app's feed
// (livePipelines.mapFeed, both lanes, D-268), a new map builder over an empty
// tile directory, a new zone store and a new NWS provider - so every cache
// starts empty. The recorded answers come from a local server: M1 scenario
// 01's Oak Ridge Flash Flood Warning, its times moved to now, as the answer
// to every active-alerts ask (the view's areas included); the place's
// /points and its stations; anything else is not found. The basemap comes
// from memory through the map builder's transport (recorded, in
// maps_basemap_test.go): the recorded OpenFreeMap TileJSON, and an embedded
// tile's bytes for every tile asked. The timing keeper (timing.go) records
// m5 - from g to the first complete frame holding every alert's area - and
// the other events of the open; each open's line also counts the TileJSON
// and tile requests its map made, and the log names every request the
// recorded server answered.
//
// The results go to the test log and, when WATCHPOST_VALIDATE_M5_OUT names a
// file, to that file, one open a line.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

const (
	m5Opens = 20 // D-46's n
	// m5Wait is the longest an open is waited on for its m5; an open that
	// never reaches it is reported as missing with the events it said.
	m5Wait = 30 * time.Second
)

func TestMeasureM5InProcess(t *testing.T) {
	if os.Getenv("WATCHPOST_VALIDATE_M5") != "1" {
		t.Skip("measures M5 in process over twenty cold opens; WATCHPOST_VALIDATE_M5=1 runs it")
	}
	t.Setenv("WATCHPOST_DEBUG_TIMING", "1")
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	srv := recordedNWS(t)
	var m5s []float64
	var lines []string
	for i := 1; i <= m5Opens; i++ {
		events := coldOpenInProcess(t, srv.URL, idx)
		line := fmt.Sprintf("open %2d: %s", i, m5Line(events))
		if ms, ok := events["m5"]; ok {
			m5s = append(m5s, ms)
		} else {
			line += " (no m5)"
		}
		lines = append(lines, line)
		t.Log(line)
	}
	s := slices.Clone(m5s)
	slices.Sort(s)
	rank := func(p float64) float64 {
		if len(s) == 0 {
			return math.NaN()
		}
		return s[max(int(math.Ceil(p*float64(len(s))))-1, 0)]
	}
	summary := fmt.Sprintf("M5 in process: n=%d of %d, p50=%.1f p90=%.1f max=%.1f ms (target p90 <= 3500 ms)", len(s), m5Opens, rank(0.5), rank(0.9), rank(1))
	t.Log(summary)
	if path := os.Getenv("WATCHPOST_VALIDATE_M5_OUT"); path != "" {
		if err := os.WriteFile(path, []byte(strings.Join(append(lines, summary), "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// coldOpenInProcess builds one station with empty caches, shows it the
// recorded place, presses g and returns the first time of each event the
// open said, in milliseconds from g.
func coldOpenInProcess(t *testing.T, base string, idx *geodata.Index) map[string]float64 {
	t.Helper()
	client, err := httpx.New(httpx.Config{UserAgent: "watchpost-measure"})
	if err != nil {
		t.Fatal(err)
	}
	provider := nws.New(client, base)
	tiles := &recorded{}
	lp := &livePipelines{ctx: context.Background(), provider: provider, weather: provider, timings: newTimingLog(), idx: idx,
		zoneShapes: zoneStore(t, base), areaAlerts: provider.AlertsInAreas,
		maps: newMapBuilder("0.18.0-measure", filepath.Join(t.TempDir(), "map"), tiles, nil)}
	c := tty.Config{Version: "0.18.0-measure"}
	lp.mapConfig(&c, config.Default())
	d, err := tty.NewDashboard(c)
	if err != nil {
		t.Fatal(err)
	}
	p := tea.NewProgram(tty.NewRouter(d), tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithWindowSize(149, 38),
		tea.WithoutSignalHandler())
	ended := make(chan tea.Model, 1)
	go func() {
		m, err := p.Run()
		if err != nil {
			t.Errorf("the program ended with %v", err)
		}
		ended <- m
	}()
	defer func() {
		p.Quit()
		if r, ok := (<-ended).(tty.Router); ok {
			r.CloseMap()
		}
	}()
	p.Send(tty.SnapshotMsg{Snap: oakRidgeNow(t)})
	time.Sleep(500 * time.Millisecond) // the snapshot is drawn before g; M5's clock starts at g
	p.Send(tea.KeyPressMsg{Code: 'g', Text: "g"})
	events := map[string]float64{}
	for deadline := time.Now().Add(m5Wait); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		for _, r := range lp.timings.last() {
			if _, said := events[r.Event]; r.Trigger == "open" && !said {
				events[r.Event] = r.MS
			}
		}
		if _, ok := events["m5"]; ok {
			break
		}
	}
	tileJSON, tileCount := 0, 0
	for _, u := range tiles.requests() {
		if strings.HasSuffix(u, ".pbf") {
			tileCount++
		} else {
			tileJSON++
		}
	}
	events["tilejson-asked"], events["tiles-asked"] = float64(tileJSON), float64(tileCount)
	return events
}

// oakRidgeNow is M1 scenario 01's place with its recorded alert in force now.
func oakRidgeNow(t *testing.T) *snapshot.Snapshot {
	t.Helper()
	loc, _ := m1Fixture(t, "01-covers-oak-ridge")
	loc.Zip, loc.TZ = "37830", "America/New_York"
	now := time.Now()
	for i := range loc.Alerts {
		loc.Alerts[i].Effective, loc.Alerts[i].Expires = now.Add(-time.Hour), now.Add(2*time.Hour)
	}
	return &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion, GeneratedAt: now, Locations: []snapshot.Location{loc}}
}

// recordedNWS answers as the weather service did for scenario 01: the
// recorded alert for every active-alerts ask, the place's point and one
// station; anything else is not found.
func recordedNWS(t *testing.T) *httptest.Server {
	t.Helper()
	recorded, err := os.ReadFile(filepath.Join("testdata", "maps", "m1", "01-covers-oak-ridge", "alerts.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Type     string           `json:"type"`
		Features []map[string]any `json:"features"`
	}
	if err := json.Unmarshal(recorded, &doc); err != nil {
		t.Fatal(err)
	}
	past, future := time.Now().Add(-time.Hour).Format(time.RFC3339), time.Now().Add(2*time.Hour).Format(time.RFC3339)
	for _, f := range doc.Features {
		props, _ := f["properties"].(map[string]any)
		for k, v := range map[string]string{"sent": past, "effective": past, "onset": past, "expires": future, "ends": future} {
			props[k] = v
		}
	}
	alerts, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("recorded server asked %s", r.URL.RequestURI())
		base := srv.URL
		switch {
		case r.URL.Path == "/alerts/active":
			_, _ = w.Write(alerts)
		case strings.HasPrefix(r.URL.Path, "/points/"):
			_, _ = fmt.Fprintf(w, `{"properties":{"forecast":"%[1]s/gridpoints/MRX/1,1/forecast","forecastHourly":"%[1]s/gridpoints/MRX/1,1/forecast/hourly",`+
				`"forecastGridData":"%[1]s/gridpoints/MRX/1,1","observationStations":"%[1]s/gridpoints/MRX/1,1/stations",`+
				`"county":"%[1]s/zones/county/TNC001","timeZone":"America/New_York"}}`, base)
		case r.URL.Path == "/gridpoints/MRX/1,1/stations":
			_, _ = w.Write([]byte(`{"features":[{"geometry":{"coordinates":[-84.2696,36.0104]},"properties":{"stationIdentifier":"KOQT"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// m5Line is an open's events, sorted by time, for the log.
func m5Line(events map[string]float64) string {
	keys := make([]string, 0, len(events))
	for k := range events {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int { return int(events[a] - events[b]) })
	var b []string
	for _, k := range keys {
		b = append(b, k+"="+strconv.FormatFloat(events[k], 'f', 1, 64))
	}
	return strings.Join(b, " ")
}
