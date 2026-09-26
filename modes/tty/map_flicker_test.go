package tty

// map_flicker_test.go — 0.18.0 UAT-1 U1-28: an alert's area "periodically
// flickers". Every new snapshot asks the feed again, and the same alert comes
// back; the frame drawn at once must still show it - and so must every frame
// the library's clock and a landed Work draw after.

import (
	"context"
	"strings"
	"testing"
)

// hasArea reports whether the drawn lines carry the alert's severity digit,
// which the outline repeats.
func hasArea(d Dashboard) bool {
	return strings.Contains(stripANSITest(strings.Join(d.mapPane.lines, "\n")), "3")
}

func TestTheSameAlertAgainNeverDropsFromTheFrame(t *testing.T) {
	feed := boxFeed(-117.6, -117.1, false)
	d := openMap(t, Config{MapFeed: feed, MapDescription: "off"}, 133, 44)
	if !hasArea(d) {
		t.Fatal("the settled map does not draw the alert: this measures nothing")
	}
	for i := range 20 {
		m, _ := d.Update(SnapshotMsg{Snap: placedSnap()})
		d = m.(Dashboard)
		m, _ = d.Update(mapFeedMsg{gen: d.mapPane.feedGen, feed: feed(context.Background(), d.mapAsk())})
		d = m.(Dashboard)
		if !hasArea(d) {
			t.Fatalf("refresh %d: the frame drawn as the same alert came back does not show it", i)
		}
		if at := d.mapPane.tickAt; !at.IsZero() {
			m, _ = d.Update(mapTickMsg{at: at})
			d = m.(Dashboard)
			if !hasArea(d) {
				t.Fatalf("refresh %d: the clock's frame does not show it", i)
			}
		}
		for w := d.mapWorkCmd(); w != nil; w = d.mapWorkCmd() {
			m, _ = d.Update(w())
			d = m.(Dashboard)
			if !hasArea(d) {
				t.Fatalf("refresh %d: a frame drawn as work landed does not show it", i)
			}
		}
	}
}

// TestAChangedAlertIsHandedInAgain: skipping the unchanged is not skipping
// the changed - an alert whose words or shape move is set again.
func TestAChangedAlertIsHandedInAgain(t *testing.T) {
	d := openMap(t, Config{MapDescription: "off"}, 133, 44)
	calls := &[]string{}
	d.mapPane.calls = calls
	same := boxFeed(-117.6, -117.1, false)(context.Background(), d.mapAsk())
	m, _ := d.Update(mapFeedMsg{gen: d.mapPane.feedGen, feed: same})
	d = m.(Dashboard)
	if strings.Contains(strings.Join(*calls, " "), "Set") {
		t.Fatalf("an unchanged alert was handed in again: %v", *calls)
	}
	moved := boxFeed(-117.7, -117.2, false)(context.Background(), d.mapAsk())
	d.Update(mapFeedMsg{gen: d.mapPane.feedGen, feed: moved})
	if !strings.Contains(strings.Join(*calls, " "), "Set") {
		t.Errorf("an alert whose area moved was not handed in: %v", *calls)
	}
}
