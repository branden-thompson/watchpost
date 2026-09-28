package tty

// map_quake_test.go — 0.18.0 D-122 at the window: the quakes the map draws
// are a Setting - M2.5+ or M1.0+, the past week or the past day - and the
// ask carries it, with the listener's clock for the quakes' times.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TestTheQuakesAreASetting is D-122: M2.5+ over the past week by default;
// → steps the past day, then M1.0+'s week and day, and round again; the
// file's word opens as chosen, anything else as the default; the ask
// carries the feed's name and the clock.
func TestTheQuakesAreASetting(t *testing.T) {
	d, got := uiDash(t, rowMapQuakes)
	body, _, _ := d.focusBody(d.opts())
	if text := stripANSITest(strings.Join(body, "\n")); !strings.Contains(text, "Quakes -") || !strings.Contains(text, "M2.5+, past week") {
		t.Fatalf("the Maps tab has no quakes row at M2.5+, past week:\n%s", text)
	}
	for _, want := range []string{"2.5_day", "1.0_week", "1.0_day", "2.5_week"} {
		m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
		d = m.(Dashboard)
		if d.mapQuakeFeed != want || d.mapAsk().QuakeFeed != want {
			t.Errorf("→ gave %q (asked %q); want %q", d.mapQuakeFeed, d.mapAsk().QuakeFeed, want)
		}
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapQuakeFeed != "2.5_day" {
		t.Errorf("esc wrote %q, want 2.5_day", got.MapQuakeFeed)
	}
	if mapDash(t, Config{MapQuakeFeed: "1.0_day"}).mapQuakeFeed != "1.0_day" || mapDash(t, Config{}).mapQuakeFeed != "2.5_week" || mapDash(t, Config{MapQuakeFeed: "all_month"}).mapQuakeFeed != "2.5_week" {
		t.Error("the file's word does not open as chosen, or a word not offered is not the default")
	}
	d.clockFmt = render.ClockByKey("24h")
	if d.mapAsk().Clock != render.ClockByKey("24h") {
		t.Error("the ask does not carry the listener's clock")
	}
}
