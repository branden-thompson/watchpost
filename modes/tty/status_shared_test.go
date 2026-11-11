package tty

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestASharedHostShowsItsWorstFeedAndItsOldestFetch is 0.19.5 Q2.6 (RN-7): two
// feeds served from one host (nws and nws-marine on api.weather.gov) make one
// row, and the row shows the unhealthier feed and the older fetch, in either
// order, so a fine feed cannot hide a failing one.
func TestASharedHostShowsItsWorstFeedAndItsOldestFetch(t *testing.T) {
	const host = "api.weather.gov"
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	ok := snapshot.ProviderStatus{ID: "nws", Status: snapshot.ProviderOK, FetchedAt: now.Add(-time.Minute)}
	bad := snapshot.ProviderStatus{ID: "nws-marine", Status: snapshot.ProviderDegraded, FetchedAt: now.Add(-time.Hour)}
	st := Stats{Endpoints: map[string][]string{"nws": {host}, "nws-marine": {host}}}
	for _, provs := range [][]snapshot.ProviderStatus{{ok, bad}, {bad, ok}} {
		rows := sourceRows(provs, st, nil, now)
		if len(rows) != 1 {
			t.Fatalf("two feeds on one host make %d rows; want one", len(rows))
		}
		r := rows[0]
		if r.status != snapshot.ProviderDegraded {
			t.Errorf("feeds in order %s, %s: the row says %s; want the degraded feed's status", provs[0].ID, provs[1].ID, r.status)
		}
		if !r.fetchedAt.Equal(bad.FetchedAt) {
			t.Errorf("feeds in order %s, %s: the row's fetch is %v; want the older, %v", provs[0].ID, provs[1].ID, r.fetchedAt, bad.FetchedAt)
		}
	}
}
