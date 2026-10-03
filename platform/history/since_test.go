package history

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// THE STORE SAYS SINCE WHEN, AND HOW MUCH A DATASET HOLDS (D-231): the oldest
// day it holds anything for, and one dataset's bytes alone - what a longer
// retention's cost is measured from.
func TestTheStoreSaysSinceWhenAndHowMuch(t *testing.T) {
	now := time.Date(2026, 10, 3, 15, 0, 0, 0, time.UTC)
	vals := Dataset{Name: "vals", Version: 1, Step: time.Hour, Fields: []Field{{Name: "v", Unit: "x"}}, Hours: 72 * time.Hour}
	docs := Dataset{Name: "docs", Version: 1, Step: time.Hour, Hours: 72 * time.Hour}
	s := Open(t.TempDir(), func() time.Time { return now }, vals, docs)
	if _, ok := s.Since(); ok {
		t.Fatal("an empty store says it holds something")
	}
	shape := Shape{Box: geo.Box{W: 1, S: 1, E: 1, N: 1}, Cols: 1, Rows: 1}
	early := time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC)
	for _, at := range []time.Time{early.AddDate(0, 0, 1), early, now.Add(-time.Hour)} { // two days of one month: the earlier is its first
		if !s.Put("vals", Record{Key: Key{Source: "a", Place: "b"}, At: at, IssuedAt: at, Shape: shape, Values: map[string][]float64{"v": {1}}}) {
			t.Fatal("put failed")
		}
	}
	s.Put("docs", Record{Key: Key{Source: "a", Place: "c"}, At: now, IssuedAt: now, Doc: []byte(`{"x":1}`)})
	since, ok := s.Since()
	if !ok || !since.Equal(time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("since %v, %v; want 28 September", since, ok)
	}
	if v, d := s.BytesOf("vals"), s.BytesOf("docs"); v <= 0 || d <= 0 || v+d > s.Bytes() {
		t.Errorf("vals %d, docs %d, all %d: each its own, within the whole", v, d, s.Bytes())
	}
}
