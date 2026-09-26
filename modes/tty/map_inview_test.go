package tty

// map_inview_test.go — 0.18.0 D-66 (UAT-1 U1-15): "Alerts in view", the
// default scope. The ask carries the view's box; the feed is asked again once
// panning stops - 600 ms after the last move - never on every key.

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestAlertsInViewIsTheDefaultScope(t *testing.T) {
	d := mapDash(t, Config{})
	if d.mapScope != ScopeInView || alertScopeByKey("") != ScopeInView || ScopeInView.Key() != "view" {
		t.Errorf("an empty file opens with scope %v; want alerts in view (D-66)", d.mapScope)
	}
	if alertScopeByKey("station") != ScopeStation || alertScopeByKey("national") != ScopeNational {
		t.Error("the other scopes' words do not read back")
	}
}

func TestTheAskCarriesTheView(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	v := d.mapAsk().View
	if !(v.W < v.E && v.S < v.N) {
		t.Fatalf("the ask's view is %+v", v)
	}
	if !(v.W < -117.38 && -117.38 < v.E && v.S < 33.2 && 33.2 < v.N) {
		t.Errorf("the view %+v does not hold the place it is centred on", v)
	}
}

func TestTheFeedIsAskedOnceThePanningStops(t *testing.T) {
	asked := 0
	feed := func(ctx context.Context, ask MapAsk) MapFeed {
		asked++
		return boxFeed(-117.6, -117.1, false)(ctx, ask)
	}
	d := openMap(t, Config{MapFeed: feed}, 133, 44)
	before := asked
	m, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	m, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	if cmd == nil || d.mapPane.viewGen < 2 {
		t.Fatalf("panning scheduled nothing (view generation %d)", d.mapPane.viewGen)
	}
	m, _ = d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen - 1}) // the first pan's settling: superseded
	if c := m.(Dashboard).mapFeedCmd(); c != nil && m.(Dashboard).mapPane.feedGen != d.mapPane.feedGen {
		t.Error("a superseded settling asked the feed")
	}
	m, cmd = d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen})
	d = m.(Dashboard)
	if d.mapPane.feedGen == 0 || cmd == nil {
		t.Fatal("the settled view did not ask the feed")
	}
	d = feedAndSettle(t, d)
	if asked <= before {
		t.Error("the feed was not asked once the panning stopped")
	}
	station := openMap(t, Config{MapFeed: feed, MapAlertScope: "station"}, 133, 44)
	gen := station.mapPane.feedGen
	m, _ = station.Update(mapViewSettledMsg{gen: station.mapPane.viewGen})
	if m.(Dashboard).mapPane.feedGen != gen {
		t.Error("with the station's scope a settled view asked the feed again")
	}
}
