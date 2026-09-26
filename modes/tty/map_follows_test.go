package tty

// map_follows_test.go — 0.18.0 D-78 (UAT-1 U1-41): the Area Alerts box follows
// the view. While the selected place is in view the box begins with its
// conditions; once the view leaves it, the box names the view instead and
// says nothing of the place - "No information is better than incorrect or
// non matching information with the map." Either way it lists the alerts in
// view, most severe first, and a full box ends with how many more there are.

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// alertAt is one alert overlay: a small square at a point, of a severity.
func alertAt(id, label string, lon, lat float64, sev tuimaps.Severity) tuimaps.Overlay {
	ring := []tuimaps.LonLat{{Lon: lon, Lat: lat}, {Lon: lon + 0.2, Lat: lat}, {Lon: lon + 0.2, Lat: lat + 0.2}, {Lon: lon, Lat: lat + 0.2}, {Lon: lon, Lat: lat}}
	return tuimaps.Overlay{ID: "alert/" + id, Valid: time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC), Keeps: 6 * time.Hour,
		Features: []tuimaps.Feature{{Kind: tuimaps.Polygon, Rings: [][]tuimaps.LonLat{ring}, Role: tuimaps.AlertSevere, Label: label, Severity: sev, ID: id}}}
}

// viewFeed is Oceanside's wind warning, a gale warning off Santa Monica, and
// a red flag warning in Imperial County - all three in view at the state
// scale - and a flood warning due north, at the view's longitudes but none
// of its latitudes.
func viewFeed(_ context.Context, _ MapAsk) MapFeed {
	return MapFeed{
		Overlays: []tuimaps.Overlay{alertAt("w1", "Wind Warning", -117.5, 33.1, tuimaps.SeveritySevere),
			alertAt("g1", "Gale Warning", -119.2, 33.8, tuimaps.SeverityModerate),
			alertAt("r1", "Red Flag Warning", -115.8, 33.0, tuimaps.SeverityExtreme),
			alertAt("f1", "Flood Warning", -117.5, 38.0, tuimaps.SeveritySevere)},
		InView: []snapshot.Alert{
			{ID: "g1", Event: "Gale Warning", Severity: "Moderate", AreaDesc: "Santa Monica Basin"},
			{ID: "r1", Event: "Red Flag Warning", Severity: "Extreme", AreaDesc: "Imperial County"},
		},
	}
}

// boxText is the Area Alerts box's words as one run.
func boxText(d Dashboard) string {
	return unwrapped(stripANSITest(strings.Join(d.areaAlertsBox(d.mapBodySize()), "\n")))
}

func TestTheBoxListsTheAlertsInViewMostSevereFirst(t *testing.T) {
	d := openMap(t, Config{MapFeed: viewFeed}, 133, 44)
	text := boxText(d)
	for _, want := range []string{"Oceanside, CA - Currently:", "Red Flag Warning in effect for Imperial County", "Gale Warning in effect for Santa Monica Basin"} {
		if !strings.Contains(text, want) {
			t.Errorf("the box does not say %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Flood Warning") {
		t.Errorf("an alert north of the view is listed:\n%s", text)
	}
	if red, gale := strings.Index(text, "Red Flag"), strings.Index(text, "Gale"); red > gale {
		t.Errorf("the extreme alert is not before the moderate one:\n%s", text)
	}
}

func TestTheBoxFollowsTheViewAwayFromThePlace(t *testing.T) {
	namer := func(c tuimaps.LonLat, _ float64) string {
		if c.Lon > -100 {
			return "the Carolinas"
		}
		return "Southern California"
	}
	d := openMap(t, Config{MapFeed: viewFeed, MapAreaName: namer}, 133, 44)
	d = pressCode(d, '1', "1")
	for range 30 { // east, until Oceanside is behind
		m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
		d = m.(Dashboard)
		if !d.viewBox(d.mapBodySize()).Contains(33.2, -117.38) {
			break
		}
	}
	d = pressCode(d, '+', "+")
	d = pressCode(d, '+', "+")
	if d.viewBox(d.mapBodySize()).Contains(33.2, -117.38) {
		t.Fatal("the view never left Oceanside")
	}
	text := boxText(d)
	if strings.Contains(text, "Oceanside") || strings.Contains(text, "Currently") {
		t.Errorf("away from the place, the box still speaks of it:\n%s", text)
	}
	if !strings.Contains(text, "the Carolinas") && !strings.Contains(text, "The Carolinas") {
		t.Errorf("the box does not name the view:\n%s", text)
	}
	if strings.Contains(text, "Wind Warning") {
		t.Errorf("an alert out of view is still listed:\n%s", text)
	}
	if !strings.Contains(text, "No alerts in view") {
		t.Errorf("an empty view does not say so:\n%s", text)
	}
	if got := d.mapTitle(); strings.Contains(got, "Oceanside") {
		t.Errorf("away from the place, the title still names it: %q", got)
	}
}

func TestAFullBoxSaysHowManyMore(t *testing.T) {
	feed := func(ctx context.Context, ask MapAsk) MapFeed {
		f := MapFeed{}
		for i := range 30 {
			id := "s" + strconv.Itoa(i)
			f.Overlays = append(f.Overlays, alertAt(id, "Small Craft Advisory", -118.5+float64(i%6)*0.3, 32.8+float64(i/6)*0.3, tuimaps.SeverityMinor))
			f.InView = append(f.InView, snapshot.Alert{ID: id, Event: "Small Craft Advisory", Severity: "Minor", AreaDesc: "Waters " + id})
		}
		return f
	}
	d := openMap(t, Config{MapFeed: feed}, 133, 44)
	text := boxText(d)
	if !strings.Contains(text, "more in view") {
		t.Errorf("thirty alerts fill the box and it does not say how many more:\n%s", text)
	}
	full := strings.Join(d.describeLinesAll(), " ")
	if n := strings.Count(full, "Small Craft Advisory"); n != 30 {
		t.Errorf("the full description names %d of the 30 alerts in view", n)
	}
}
