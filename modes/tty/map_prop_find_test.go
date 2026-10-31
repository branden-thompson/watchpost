package tty

// map_prop_find_test.go — 0.19.0 W5.3: finding a place in the Propagation
// mode (FR-1.10; D-25, D-27, D-61, D-79).

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/propanswers"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// reykjavik is a place no weather region covers.
var reykjavik = snapshot.LocationRef{Label: "Reykjavik, IS", Lat: 64.1, Lon: -21.9}

// typeFind presses / and types a query, then Enter, running the find it
// asks for; it returns the dashboard after the answer.
func typeFind(t *testing.T, d Dashboard, query string) Dashboard {
	t.Helper()
	d = pressMap(t, d, "/")
	if d.modal != modalPropFind {
		t.Fatalf("/ opened window %v; want the place finder (FR-1.10)", d.modal)
	}
	for _, r := range query {
		m, _ := d.Update(mapKeyMsg(t, string(r)))
		d = m.(Dashboard)
	}
	m, cmd := d.Update(mapKeyMsg(t, "enter"))
	d = m.(Dashboard)
	if cmd == nil {
		t.Fatal("Enter asked no find")
	}
	m, _ = d.Update(cmd())
	return m.(Dashboard)
}

// placeIDs are the places the map marks.
func placeIDs(d Dashboard) []string {
	var ids []string
	for _, p := range d.mapPane.m.Places() {
		ids = append(ids, p.ID)
	}
	return ids
}

// TestAFoundPlaceIsAnsweredAndNotKept is FR-1.9 and FR-1.10 (D-27, D-204):
// on the Observer a place found anywhere - here one no weather region covers
// - is marked, joins the chart as a target row beside the continents,
// answered from the origin, and is said; the map stays where it was, so the
// origin is still its centre; and nothing is kept: no commit, the recent
// places as they were, and leaving the mode takes the mark and the row off.
func TestAFoundPlaceIsAnsweredAndNotKept(t *testing.T) {
	d := chartDash(t)
	commits := 0
	d.cfg.Commit = func(_, _ []snapshot.LocationRef) error { commits++; return nil }
	asked := ""
	d.cfg.FindPlace = func(_ context.Context, q string) (snapshot.LocationRef, error) { asked = q; return reykjavik, nil }
	recent := len(d.recentLocations())
	before, _ := d.mapPane.m.Centre()
	origin := d.mapPane.propOrigin.at
	d = typeFind(t, d, "Reykjavik")
	if asked != "Reykjavik" {
		t.Errorf("the finder was asked %q", asked)
	}
	if c, _ := d.mapPane.m.Centre(); c != before || d.mapPane.propOrigin.at != origin {
		t.Errorf("the find moved the map to %+v (origin %+v); on the Observer it stays at %+v (D-204)", c, d.mapPane.propOrigin.at, before)
	}
	if !slices.Contains(placeIDs(d), foundPlaceID) {
		t.Errorf("the found place is not marked: %v", placeIDs(d))
	}
	d = workChart(t, d)
	names, rows := d.propChartRows()
	if len(names) != 2+len(propanswers.Continents) || names[len(names)-1] != reykjavik.Label || len(rows[len(rows)-1].Bands) == 0 {
		t.Fatalf("the chart's rows are %v; the continents, then the place found, answered", names)
	}
	if k := d.mapPane.propChart.key; k.origin != origin {
		t.Errorf("the found place is answered from %+v; from the origin %+v", k.origin, origin)
	}
	words := strings.Join(propWordsFrom(t, d), "\n")
	if !strings.Contains(words, "Found Reykjavik, IS: marked on the map and answered with the targets, and not kept.") || !strings.Contains(words, "\nReykjavik, IS: ") {
		t.Errorf("the words do not say the place found, or its line:\n%s", words)
	}
	if commits != 0 || len(d.recentLocations()) != recent {
		t.Errorf("the found place was kept: %d commits, %d recent places (were %d)", commits, len(d.recentLocations()), recent)
	}
	d = pressMap(t, d, "P") // the Propagation mode left
	if d.mapPane.propFound != nil || slices.Contains(placeIDs(d), foundPlaceID) {
		t.Errorf("leaving the mode kept the found place: %v, %v", d.mapPane.propFound, placeIDs(d))
	}
}

// TestFocusReturnsAfterAFind is FR-1.10: the map has the focus again once a
// place is found; a find that fails says why in the finder, which stays open
// to change it; Esc leaves the map as it was; an earlier find's answer that
// arrives late is dropped; an empty Enter takes the mark off.
func TestFocusReturnsAfterAFind(t *testing.T) {
	d := chartDash(t)
	d.cfg.FindPlace = func(context.Context, string) (snapshot.LocationRef, error) {
		return snapshot.LocationRef{}, errors.New("no match in the offline index - try a city, coordinates or a grid locator")
	}
	d = typeFind(t, d, "nowhere at all")
	if d.modal != modalPropFind || !strings.Contains(strings.Join(d.propFindLines(d.opts()), "\n"), "no match in the offline index") {
		t.Fatalf("a failed find: window %v, lines %q", d.modal, d.propFindLines(d.opts()))
	}
	if lines := strings.Join(strings.Fields(strings.Join(d.propFindLines(d.opts()), " ")), " "); !strings.Contains(lines, "press Enter to try") {
		t.Errorf("a failed find does not say Enter tries again (AX-10): %q", lines)
	}
	if m, _ := d.Update(mapKeyMsg(t, "esc")); m.(Dashboard).modal != modalMap {
		t.Errorf("Esc left window %v; the map", m.(Dashboard).modal)
	}
	d.cfg.FindPlace = func(context.Context, string) (snapshot.LocationRef, error) { return reykjavik, nil }
	d = typeFind(t, d, "Reykjavik")
	if d.modal != modalMap {
		t.Errorf("after a place was found the focus is on window %v; the map", d.modal)
	}
	late := snapshot.LocationRef{Label: "An earlier find's answer", Lat: 1, Lon: 1}
	if m, _ := d.Update(propFoundMsg{seq: d.mapPane.propFindSeq - 1, ref: late}); m.(Dashboard).mapPane.propFound.Label != reykjavik.Label {
		t.Errorf("an earlier find's answer, arriving late, replaced the place found: %v", m.(Dashboard).mapPane.propFound)
	}
	d = pressMap(t, d, "/")
	m, _ := d.Update(mapKeyMsg(t, "enter"))
	if d = m.(Dashboard); d.mapPane.propFound != nil || slices.Contains(placeIDs(d), foundPlaceID) || d.modal != modalMap {
		t.Errorf("an empty Enter left the mark: %v, %v, window %v", d.mapPane.propFound, placeIDs(d), d.modal)
	}
}

// TestTheBroadcasterAnswersAFoundPlaceAsItsTarget is FR-1.10 with D-193:
// on the Broadcaster, which answers from its station, the place found is
// the map's centre and so the target its answer is for.
func TestTheBroadcasterAnswersAFoundPlaceAsItsTarget(t *testing.T) {
	d := stationDash(t)
	d.cfg.FindPlace = func(context.Context, string) (snapshot.LocationRef, error) { return reykjavik, nil }
	d = typeFind(t, d, "Reykjavik")
	if words := workCentre(t, d).propCentreWords(); !strings.Contains(words, "64.") {
		t.Errorf("the centre's answer is not for the place found: %q", words)
	}
}

// TestTheCentreIsNamedWhereverItIs is BUILD red team AX-1 (D-193, D-64):
// abroad, the Observer's origin - the map's centre - is named for where it
// is, never for the region last drawn: the place found while the centre
// rests on it, then, panned away, the town nearest the centre, and with no
// town near, its coordinates, said as degrees.
func TestTheCentreIsNamedWhereverItIs(t *testing.T) {
	d := chartDash(t)
	region := d.mapPane.region.Name
	d.cfg.FindPlace = func(context.Context, string) (snapshot.LocationRef, error) { return reykjavik, nil }
	d.cfg.PlaceNear = func(lat, lon float64) (string, string) {
		if lat > 30 && lat < 40 && lon > 130 && lon < 145 {
			return "Tokyo, JP", "JP"
		}
		return "", ""
	}
	pan := func(d Dashboard, lat, lon float64) Dashboard {
		t.Helper()
		if err := d.mapPane.m.Recentre(tuimaps.LonLat{Lon: lon, Lat: lat}); err != nil {
			t.Fatal(err)
		}
		return d.renderMap()
	}
	d = typeFind(t, d, "Reykjavik")
	d = pan(d, reykjavik.Lat, reykjavik.Lon) // the Observer's map stays on a find (D-204); panned onto the place, it is named for it
	words := strings.Join(propWordsFrom(t, d), "\n")
	if d.mapPane.propOrigin.name != reykjavik.Label || !strings.Contains(words, "Reykjavik, IS, now: ") {
		t.Errorf("centred on the place found, the origin is %q:\n%s", d.mapPane.propOrigin.name, words)
	}
	d = pan(d, 35.7, 139.7)
	words = strings.Join(d.describeLinesAll(), "\n")
	if !strings.Contains(words, "Near Tokyo, Japan, now: ") || strings.Contains(words, region) { // said by its country (D-231)
		t.Errorf("panned to Tokyo, the words do not name it, or name %q:\n%s", region, words)
	}
	if title := d.viewName(); title != "near Tokyo, JP" {
		t.Errorf("panned to Tokyo, the view is named %q", title)
	}
	if strings.Contains(words, "marked at the map's centre") {
		t.Errorf("panned away, the words still put the place found at the centre:\n%s", words)
	}
	d = pan(d, -40, -140)
	if words = strings.Join(d.describeLinesAll(), "\n"); !strings.Contains(words, "40.0 degrees south 140.0 degrees west, now: ") || strings.Contains(words, region) {
		t.Errorf("over open sea, the words do not say the centre's coordinates, or name %q:\n%s", region, words)
	}
}

// TestEscCancelsAFindInFlight is BUILD red team AX-11: Esc while a find is
// still being looked up cancels it - its answer, arriving late, neither
// marks a place nor moves the map.
func TestEscCancelsAFindInFlight(t *testing.T) {
	d := chartDash(t)
	d.cfg.FindPlace = func(context.Context, string) (snapshot.LocationRef, error) { return reykjavik, nil }
	before, _ := d.mapPane.m.Centre()
	d = pressMap(t, d, "/")
	for _, r := range "Reykjavik" {
		m, _ := d.Update(mapKeyMsg(t, string(r)))
		d = m.(Dashboard)
	}
	m, cmd := d.Update(mapKeyMsg(t, "enter"))
	d = m.(Dashboard)
	m, _ = d.Update(mapKeyMsg(t, "esc"))
	d = m.(Dashboard)
	if d.mapPane.propFindBusy {
		t.Error("Esc left the find busy")
	}
	m, _ = d.Update(cmd())
	d = m.(Dashboard)
	after, _ := d.mapPane.m.Centre()
	if d.mapPane.propFound != nil || slices.Contains(placeIDs(d), foundPlaceID) || after != before {
		t.Errorf("a find cancelled with Esc still landed: %v, %v, centre %+v (was %+v)", d.mapPane.propFound, placeIDs(d), after, before)
	}
}

// TestACancelledFindCancelsItsRequest is FR-4.9 and AX-11: Esc over a find
// in flight drops its answer and stops its lookup too - a query the
// listener took back is not sent on to the online geocoder.
func TestACancelledFindCancelsItsRequest(t *testing.T) {
	d := chartDash(t)
	stopped := make(chan error, 1)
	d.cfg.FindPlace = func(ctx context.Context, _ string) (snapshot.LocationRef, error) {
		<-ctx.Done()
		stopped <- ctx.Err()
		return snapshot.LocationRef{}, ctx.Err()
	}
	d = pressMap(t, d, "/")
	for _, r := range "Reykjavik" {
		m, _ := d.Update(mapKeyMsg(t, string(r)))
		d = m.(Dashboard)
	}
	m, find := d.Update(mapKeyMsg(t, "enter"))
	d = m.(Dashboard)
	if find == nil {
		t.Fatal("control: Enter asked nothing")
	}
	go find()
	m, _ = d.Update(mapKeyMsg(t, "esc"))
	select {
	case err := <-stopped:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the find stopped with %v; want cancelled", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Esc left the find's lookup running")
	}
	if m.(Dashboard).modal != modalMap {
		t.Errorf("Esc left window %v", m.(Dashboard).modal)
	}
}

// TestAPlaceAbroadIsSaidByItsCountry is D-231 (F-218): the words say the
// centre's town abroad by its country's name - "near Tokyo, Japan" - while
// the box keeps the code; a place found abroad likewise.
func TestAPlaceAbroadIsSaidByItsCountry(t *testing.T) {
	d := chartDash(t)
	d.cfg.PlaceNear = func(float64, float64) (string, string) { return "Tokyo, JP", "JP" }
	d.mapPane.propOrigin = propPlace{at: ionomaps.LatLon{Lat: 35.7, Lon: 139.7}, name: "near Tokyo, JP", country: "JP", ok: true}
	if w := strings.Join(d.propOriginWords(), " "); !strings.Contains(w, "near Tokyo, Japan") || strings.Contains(w, "Tokyo, JP") {
		t.Errorf("the origin abroad is said %q", w)
	}
	d.mapPane.propOrigin = propPlace{at: ionomaps.LatLon{Lat: 33.2, Lon: -117.4}, name: "Oceanside, CA", country: "US", ok: true}
	if w := strings.Join(d.propOriginWords(), " "); !strings.Contains(w, "Oceanside, CA") {
		t.Errorf("a US place lost its state's code: %q", w)
	}
}

// TestLeavingTheModeDropsAFindInFlight is D-27 and RC-5: a find still being
// looked up when the Propagation mode's session ends - by whatever route -
// is cancelled with it: its lookup stops and its answer, arriving late,
// neither marks a place past the mode nor moves the map.
func TestLeavingTheModeDropsAFindInFlight(t *testing.T) {
	d := chartDash(t)
	stopped := make(chan error, 1)
	d.cfg.FindPlace = func(ctx context.Context, _ string) (snapshot.LocationRef, error) {
		<-ctx.Done()
		stopped <- ctx.Err()
		return reykjavik, nil // an answer the lookup had in hand as it was stopped
	}
	before, _ := d.mapPane.m.Centre()
	d = pressMap(t, d, "/")
	for _, r := range "Reykjavik" {
		m, _ := d.Update(mapKeyMsg(t, string(r)))
		d = m.(Dashboard)
	}
	m, find := d.Update(mapKeyMsg(t, "enter"))
	d = m.(Dashboard)
	if find == nil {
		t.Fatal("control: Enter asked nothing")
	}
	answer := make(chan any, 1)
	go func() { answer <- find() }()
	d = d.stopPropagation()
	select {
	case err := <-stopped:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the find stopped with %v; want cancelled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("leaving the mode left the find's lookup running")
	}
	m, _ = d.Update(<-answer)
	d = m.(Dashboard)
	after, _ := d.mapPane.m.Centre()
	if d.mapPane.propFound != nil || slices.Contains(placeIDs(d), foundPlaceID) || after != before {
		t.Errorf("a find in flight as the mode was left still landed: %v, %v, centre %+v (was %+v)", d.mapPane.propFound, placeIDs(d), after, before)
	}
}
