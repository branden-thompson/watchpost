package tty

// map_prop_find_test.go — 0.19.0 W5.3: finding a place in the Propagation
// mode (FR-1.10; D-25, D-27, D-61, D-79).

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

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

// TestAFoundPlaceIsAnsweredAndNotKept is FR-1.10 (D-27): a place found
// anywhere - here one no weather region covers - is centred, so the centre
// answers for it; marked; said; and never kept: nothing is committed, the
// recent places are as they were, and leaving the mode takes the mark off.
func TestAFoundPlaceIsAnsweredAndNotKept(t *testing.T) {
	d := chartDash(t)
	commits := 0
	d.cfg.Commit = func(_, _ []snapshot.LocationRef) error { commits++; return nil }
	asked := ""
	d.cfg.FindPlace = func(q string) (snapshot.LocationRef, error) { asked = q; return reykjavik, nil }
	recent := len(d.recentLocations())
	d = typeFind(t, d, "Reykjavik")
	if asked != "Reykjavik" {
		t.Errorf("the finder was asked %q", asked)
	}
	c, _ := d.mapPane.m.Centre()
	if math.Abs(c.Lat-reykjavik.Lat) > 0.5 || math.Abs(c.Lon-reykjavik.Lon) > 0.5 {
		t.Errorf("the map's centre is %+v; the found place is %+v", c, reykjavik)
	}
	if !slices.Contains(placeIDs(d), foundPlaceID) {
		t.Errorf("the found place is not marked: %v", placeIDs(d))
	}
	if words := strings.Join(d.describeLinesAll(), " "); !strings.Contains(words, "Found Reykjavik, IS: marked at the map's centre, and not kept.") {
		t.Errorf("the words do not say the place found:\n%s", words)
	}
	if o := d.mapPane.propOrigin.at; math.Abs(o.Lat-reykjavik.Lat) > 0.5 || math.Abs(o.Lon-reykjavik.Lon) > 0.5 {
		t.Errorf("the Observer answers from %+v; the map's centre, the found place, is %+v (D-193)", o, reykjavik)
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
	d.cfg.FindPlace = func(string) (snapshot.LocationRef, error) {
		return snapshot.LocationRef{}, errors.New("no match in the offline index - try a city, coordinates or a grid locator")
	}
	d = typeFind(t, d, "nowhere at all")
	if d.modal != modalPropFind || !strings.Contains(strings.Join(d.propFindLines(d.opts()), "\n"), "no match in the offline index") {
		t.Fatalf("a failed find: window %v, lines %q", d.modal, d.propFindLines(d.opts()))
	}
	if m, _ := d.Update(mapKeyMsg(t, "esc")); m.(Dashboard).modal != modalMap {
		t.Errorf("Esc left window %v; the map", m.(Dashboard).modal)
	}
	d.cfg.FindPlace = func(string) (snapshot.LocationRef, error) { return reykjavik, nil }
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
	d.cfg.FindPlace = func(string) (snapshot.LocationRef, error) { return reykjavik, nil }
	d = typeFind(t, d, "Reykjavik")
	if words := workCentre(t, d).propCentreWords(); !strings.Contains(words, "64.") {
		t.Errorf("the centre's answer is not for the place found: %q", words)
	}
}
