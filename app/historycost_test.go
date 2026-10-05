package app

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/history"
)

// A LONGER RETENTION'S COST IS MEASURED (D-231): from the rate each dataset has
// grown since the store began - hourly detail every dataset whole; trends the
// values a day's roll-up, the documents whole, as the store keeps them; a store
// too new to measure says so.
func TestALongerRetentionsCostIsMeasured(t *testing.T) {
	now := time.Date(2026, 10, 3, 15, 0, 0, 0, time.UTC)
	store := history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly, nwsAlerts)
	if got := historyCostOf(store, false, "72h", "30d", now); !strings.Contains(got, "too new") {
		t.Errorf("an empty store's cost is %q; want it said too new to measure", got)
	}
	for h := 0; h < 48; h++ {
		at := now.Add(-time.Duration(h) * time.Hour)
		store.Put(ndfdHourly.Name, history.Record{Key: history.Key{Source: "ndfd", Place: "box"}, At: at, IssuedAt: at, Shape: pointShape(1, 1), Values: map[string][]float64{"temp": {20}}})
		store.Put(nwsAlerts.Name, history.Record{Key: alertKey("a" + at.String()), At: at, IssuedAt: at, Doc: []byte(`{"event":"` + strings.Repeat("x", 400) + `"}`)})
	}
	hourly := historyCostOf(store, false, "72h", "30d", now)
	trends := historyCostOf(store, true, "30d", "1y", now)
	if !strings.Contains(hourly, "more on disk") || !strings.Contains(hourly, "since") || !strings.Contains(trends, "more on disk") {
		t.Fatalf("the costs read %q and %q", hourly, trends)
	}
	if historyExtra(store, false, "72h", "30d", now) <= historyExtra(store, false, "72h", "7d", now) {
		t.Error("a longer raise does not cost more")
	}
	valuesOnly := history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly)
	for h := 0; h < 48; h++ {
		at := now.Add(-time.Duration(h) * time.Hour)
		valuesOnly.Put(ndfdHourly.Name, history.Record{Key: history.Key{Source: "ndfd", Place: "box"}, At: at, IssuedAt: at, Shape: pointShape(1, 1), Values: map[string][]float64{"temp": {20}}})
	}
	hourlyDay, trendDay := historyExtra(valuesOnly, false, "72h", "7d", now), historyExtra(valuesOnly, true, "30d", "90d", now)
	if trendDay*24 > hourlyDay*60/4*2 || trendDay == 0 { // 56 trend days at a 24th against 4 hourly days
		t.Errorf("a values dataset's trends cost %d for 60 days against %d for 4 days of hours; a day's roll-up is a 24th of its hours", trendDay, hourlyDay)
	}
}
