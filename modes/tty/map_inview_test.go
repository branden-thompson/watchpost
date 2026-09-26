package tty

// map_inview_test.go — 0.18.0 D-66 (UAT-1 U1-15) and D-76: the map draws
// every alert in view, and there is no scope to choose. The ask carries the
// view's box; the feed is asked again once panning stops - 600 ms after the
// last move - never on every key.

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestTheMapsTabHasNoAlertScope is D-76 (UAT-1 U1-37): what the map draws is
// what is real in view, switched at the map; Settings offers no scope.
func TestTheMapsTabHasNoAlertScope(t *testing.T) {
	d, _ := uiDash(t, rowMapsOn)
	body, _, _ := d.focusBody(d.opts())
	text := stripANSITest(strings.Join(body, "\n"))
	for _, never := range []string{"Alerts -", "Alerts in view", "Station's places", "regional severe"} {
		if strings.Contains(text, never) {
			t.Errorf("the Maps tab still offers a scope (%q):\n%s", never, text)
		}
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
	closed, _ := pressKey(d, "g")
	gen := closed.mapPane.feedGen
	m, _ = closed.Update(mapViewSettledMsg{gen: closed.mapPane.viewGen})
	if m.(Dashboard).mapPane.feedGen != gen {
		t.Error("a settling that lands after the map closed asked the feed")
	}
}
