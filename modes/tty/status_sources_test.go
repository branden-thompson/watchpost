package tty

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// groupedStatus is the Status window over a station and a map that share
// api.weather.gov: a weather host the station alone uses, a fire host, the
// map's own tiles, and a host the geocoder counts that no provider claims.
func groupedStatus(t *testing.T) Dashboard {
	t.Helper()
	now := time.Now()
	st := Stats{
		Requests: httpx.RequestStats{Uptime: time.Hour, Hosts: []httpx.HostStats{
			{Host: "api.weather.gov", Attempts: 10, Net: 8, LastOK: now},
			{Host: "www.ndbc.noaa.gov", Attempts: 4, Net: 4, LastOK: now},
			{Host: "www.ospo.noaa.gov", Attempts: 3, Net: 3, LastOK: now},
			{Host: "geocoding-api.open-meteo.com", Attempts: 2, Net: 2, LastOK: now}}},
		MapRequests: httpx.RequestStats{Hosts: []httpx.HostStats{
			{Host: "api.weather.gov", Attempts: 5, Net: 5, LastOK: now},
			{Host: "tiles.openfreemap.org", Attempts: 212, Net: 180, LastOK: now}}},
		Endpoints: map[string][]string{"nws": {"api.weather.gov"}, "ndbc": {"www.ndbc.noaa.gov"}, "hms": {"www.ospo.noaa.gov"}},
		ProviderSources: map[string]StatusSource{
			"nws":  {Feature: FeatureWeather, Badge: "NWS"},
			"ndbc": {Feature: FeatureWeather, Badge: "NDBC"},
			"hms":  {Feature: FeatureAdverse, Badge: "HMS"}},
		HostSources: map[string]StatusSource{"geocoding-api.open-meteo.com": {Feature: FeatureWeather, Badge: "O-METEO"}},
	}
	m, err := NewDashboard(Config{Version: "t", Stats: func() Stats { return st }, MapSources: []MapSource{
		{Name: "OpenFreeMap", Host: "tiles.openfreemap.org", Layers: "basemap", Badge: "OFM", Sent: "tiles"},
		{Name: "National Weather Service", Host: "api.weather.gov", Layers: "alert areas", Badge: "NWS"}}})
	if err != nil {
		t.Fatal(err)
	}
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 160, Height: 60})
	sn := snap()
	sn.Providers = []snapshot.ProviderStatus{{ID: "nws", Status: snapshot.ProviderOK, FetchedAt: now},
		{ID: "ndbc", Status: snapshot.ProviderOK, FetchedAt: now}, {ID: "hms", Status: snapshot.ProviderOK, FetchedAt: now}}
	mm, _ = mm.Update(SnapshotMsg{Snap: sn})
	return mm.(Dashboard)
}

// TestTheStatusWindowGroupsHostsByWhatTheyServe is D-170 (A-43): every host
// is one row in one group - COMMON DATA when two features use it, else its
// feature's - the groups in order, one header row of columns over them all,
// a shared host's counters once, the station's and the map's together.
func TestTheStatusWindowGroupsHostsByWhatTheyServe(t *testing.T) {
	d := groupedStatus(t)
	text := stripANSITest(strings.Join(d.statusBlocks(d.statusInner()).sources, "\n"))
	order := []string{"DATA SOURCES", "COMMON DATA", "api.weather.gov", "WEATHER DATA", "www.ndbc.noaa.gov", "geocoding-api.open-meteo.com",
		"ADVERSE EVENTS", "www.ospo.noaa.gov", "MAP DRAWING DATA", "tiles.openfreemap.org"}
	at := 0
	for _, w := range order {
		i := strings.Index(text[at:], w)
		if i < 0 {
			t.Fatalf("%q is not after %q:\n%s", w, text[:at], text)
		}
		at += i + len(w)
	}
	for _, host := range []string{"api.weather.gov", "www.ndbc.noaa.gov", "www.ospo.noaa.gov", "tiles.openfreemap.org", "geocoding-api.open-meteo.com"} {
		if n := strings.Count(text, host); n != 1 {
			t.Errorf("%s is %d rows; want one", host, n)
		}
	}
	for _, gone := range []string{"API STATUS", "MAP STATUS"} {
		if strings.Contains(text, gone) {
			t.Errorf("the window still has %s", gone)
		}
	}
	if n := strings.Count(text, "ENDPOINT"); n != 1 {
		t.Errorf("the columns are headed %d times; want once over every group", n)
	}
	for _, l := range strings.Split(text, "\n") {
		if strings.Contains(l, "api.weather.gov") && !strings.Contains(l, " 15 ") {
			t.Errorf("api.weather.gov's tries are the station's and the map's together, 15: %q", l)
		}
	}
	if !strings.Contains(text, "(5 hosts in 4 groups)") {
		t.Errorf("the headline does not count hosts and groups:\n%s", text)
	}
}

// TestTheSourcesAreTheirBadges is D-170: a row's sources are the badges the
// map shows, each a chip, once a row, the columns still lined up.
func TestTheSourcesAreTheirBadges(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	d := groupedStatus(t)
	lines := d.statusBlocks(d.statusInner()).sources
	var nws string
	for _, l := range lines {
		if strings.Contains(stripANSITest(l), "api.weather.gov") {
			nws = l
		}
	}
	if !strings.Contains(nws, chipFace("NWS")) || strings.Count(stripANSITest(nws), "NWS") != 1 {
		t.Errorf("api.weather.gov's sources are not its NWS chip, once: %q", nws)
	}
	col := -1
	for _, l := range lines {
		if p := stripANSITest(l); strings.Contains(p, "ENDPOINT") {
			col = strings.Index(p, "STATUS")
		}
	}
	for _, l := range lines {
		p := []rune(stripANSITest(l))
		host := strings.Contains(string(p), ".gov") || strings.Contains(string(p), ".org") || strings.Contains(string(p), ".com")
		if host && (col < 1 || col >= len(p) || p[col-1] != ' ' || p[col] == ' ') {
			t.Errorf("a row's status does not start under STATUS (column %d): %q", col, string(p))
		}
	}
}
