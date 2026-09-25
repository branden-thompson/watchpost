package app

import (
	"context"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/config"
)

// TestTheAppHandsTheWindowItsMap is 0.18.0 W1.1 and W2's wiring (rule 8: a
// test of a thing is not a test of its wiring): the composition root hands the
// map window a constructor, and what it builds draws the basemap from the
// embedded tiles, sized as asked, reaching nothing (FR-3.2).
func TestTheAppHandsTheWindowItsMap(t *testing.T) {
	lp := &livePipelines{maps: newMapBuilder("t", t.TempDir(), &recorded{offline: true}, nil)}
	cfg := lp.ttyConfig("t", Options{}, false, config.Config{}, nil, nil, nil, nil, nil, nil)
	if cfg.NewMap == nil {
		t.Fatal("the window is handed no map constructor, so g would open a window with no map")
	}
	m, err := cfg.NewMap(tuimaps.Size{Cols: 69, Rows: 12})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if _, err := m.Settle(context.Background()); err != nil {
		t.Fatal(err)
	}
	f, err := m.Render(tuimaps.Size{Cols: 69, Rows: 12}, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Lines) != 12 {
		t.Errorf("the map drew %d lines, want 12", len(f.Lines))
	}
	if (&livePipelines{}).newMap() != nil {
		t.Error("a station with no builder handed the window a constructor")
	}
}
