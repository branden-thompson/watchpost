package tty

// map_events_test.go — 0.18.0 W2.3 (guard 2, D-45: every event that can
// change the frame draws it) and W2.5 (the freshness property: after every
// event the stored frame's counters are the library's).

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// openSettledMap is the window open on Oceanside with one alert that goes
// stale in ten minutes, every piece of work landed.
func openSettledMap(t *testing.T) Dashboard {
	t.Helper()
	now := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	feed := func(ctx context.Context, ask MapAsk) MapFeed {
		f := boxFeed(-117.6, -117.1, false)(ctx, ask)
		f.Overlays[0].Valid, f.Overlays[0].Keeps = now.Add(-50*time.Minute), time.Hour
		return f
	}
	d := mapDash(t, Config{MapFeed: feed})
	d.now = func() time.Time { return now }
	d, _ = pressKey(d, "g")
	return feedAndSettle(t, d)
}

// assertFresh is W2.5's property: the stored frame's counters are the
// library's as they stand.
func assertFresh(t *testing.T, d Dashboard, after string) {
	t.Helper()
	m := d.mapPane.m
	if d.mapPane.changed != m.Changed() || d.mapPane.ticks != m.FrameTicks() {
		t.Errorf("after %s the stored frame is at %d/%d and the library at %d/%d", after,
			d.mapPane.changed, d.mapPane.ticks, m.Changed(), m.FrameTicks())
	}
}

// TestEveryEventThatChangesTheFrameDrawsIt is W2.3's table, one row per P1-a
// event, with W2.5's property checked after each. Theme and colour depth join
// with W7, which gives the map the theme's palette; frame advance and
// playback with W8.10.
func TestEveryEventThatChangesTheFrameDrawsIt(t *testing.T) {
	for _, row := range []struct {
		name  string
		event func(t *testing.T, d Dashboard) Dashboard
	}{
		{"data: the feed lands", func(t *testing.T, d Dashboard) Dashboard {
			m, _ := d.Update(SnapshotMsg{Snap: placedSnap()})
			d = m.(Dashboard)
			m, _ = d.Update(d.mapFeedCmd()())
			return m.(Dashboard)
		}},
		{"Work landing a tile", func(t *testing.T, d Dashboard) Dashboard {
			m, _ := d.Update(mapWorkedMsg{did: true})
			return m.(Dashboard)
		}},
		{"size", func(t *testing.T, d Dashboard) Dashboard {
			m, _ := d.Update(tea.WindowSizeMsg{Width: 150, Height: 46})
			return m.(Dashboard)
		}},
		{"selection", func(t *testing.T, d Dashboard) Dashboard { return pressCode(d, ']', "]") }},
		{"pan", func(t *testing.T, d Dashboard) Dashboard {
			m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
			return m.(Dashboard)
		}},
		{"zoom", func(t *testing.T, d Dashboard) Dashboard { return pressCode(d, '+', "+") }},
		{"units", func(t *testing.T, d Dashboard) Dashboard { return pressCode(d, 'c', "c") }},
		{"the library's tick", func(t *testing.T, d Dashboard) Dashboard {
			if d.mapPane.tickAt.IsZero() {
				t.Fatal("no tick is outstanding: this row measures nothing")
			}
			m, _ := d.Update(mapTickMsg{at: d.mapPane.tickAt})
			return m.(Dashboard)
		}},
		{"a map Setting, on the next open", func(t *testing.T, d Dashboard) Dashboard {
			d, _ = pressKey(d, "g")
			d.mapScale = mapScaleCounty
			d, _ = pressKey(d, "g")
			return d
		}},
	} {
		t.Run(row.name, func(t *testing.T) {
			d := openSettledMap(t)
			gen := d.mapPane.gen
			d = row.event(t, d)
			if d.mapPane.gen == gen {
				t.Errorf("%s did not draw the map", row.name)
			}
			assertFresh(t, d, row.name)
		})
	}
}
