package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// bounded is long held to max bytes. Sixty of its days, a record each, take
// about 1.9 KB on disk.
func bounded(max int64) Dataset {
	d := long
	d.Name, d.MaxBytes = "bounded", max
	return d
}

// dayByDay records a day at a time, a record each, the clock moving on a day
// after each and the store pruned as it goes.
func dayByDay(s *Store, d Dataset, start time.Time, days int, now *time.Time) {
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		s.Put(d.Name, rec(day.Add(time.Hour), 0, float64(i), 1, 2, 3))
		*now = day.AddDate(0, 0, 1)
		for p := 0; p < 4; p++ {
			s.RollUpAndPrune()
		}
	}
}

// A DATASET IS HELD TO ITS BYTE BOUND (D-143): past it, its oldest files go -
// roll-ups, then days - before its retention ends; its newest days stay, and
// the store says the bound cut it short.
func TestADatasetIsHeldToItsByteBound(t *testing.T) {
	const max = 1200
	d := bounded(max)
	dir, now := t.TempDir(), t0
	s := Open(dir, func() time.Time { return now }, d)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	dayByDay(s, d, start, 60, &now)
	if got := s.BytesOf(d.Name); got > max {
		t.Errorf("the dataset holds %d bytes, past its bound of %d", got, max)
	}
	if len(s.Days(d.Name, ndfd, start, start, 1)) != 0 {
		t.Error("the oldest day was kept past the bound")
	}
	last := start.AddDate(0, 0, 59)
	if _, ok := s.Get(d.Name, ndfd, last.Add(time.Hour)); !ok {
		t.Error("the newest day went for the bound: the oldest go first")
	}
	if !s.Bounded(d.Name) {
		t.Error("the bound cut the dataset short and the store does not say so")
	}
}

// A DATASET WITHIN ITS BOUND, OR WITH NONE, KEEPS ITS RETENTION and is not
// said to be cut short.
func TestADatasetWithinItsBoundKeepsItsRetention(t *testing.T) {
	for _, max := range []int64{0, 1 << 30} {
		d := bounded(max)
		dir, now := t.TempDir(), t0
		s := Open(dir, func() time.Time { return now }, d)
		start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		dayByDay(s, d, start, 20, &now)
		if got := len(s.Days(d.Name, ndfd, start, start.AddDate(0, 0, 19), 30)); got < 16 {
			t.Errorf("bound %d: %d of the 16 days past the hours rolled up and kept", max, got)
		}
		if s.Bounded(d.Name) {
			t.Errorf("bound %d: said to be cut short", max)
		}
	}
}

// THE CURRENT DAY IS NEVER REMOVED FOR THE BOUND: it is being recorded.
func TestTheCurrentDayIsNeverRemovedForTheBound(t *testing.T) {
	d := bounded(1)
	dir, now := t.TempDir(), t0
	s := Open(dir, func() time.Time { return now }, d)
	yesterday := t0.Truncate(24*time.Hour).AddDate(0, 0, -1)
	s.Put(d.Name, rec(yesterday.Add(time.Hour), 0, 1, 1, 1, 1))
	s.Put(d.Name, rec(t0, 0, 2, 2, 2, 2))
	s.RollUpAndPrune()
	if _, ok := s.Get(d.Name, ndfd, t0); !ok {
		t.Error("the current day's record went for the bound")
	}
	if _, ok := s.Get(d.Name, ndfd, yesterday.Add(time.Hour)); ok {
		t.Error("yesterday was kept past a bound it passes")
	}
}

// A CHANGED RETENTION CLEARS THE WORDS; THE SAME ONE APPLIED AGAIN DOES NOT:
// Settings applies the retention each time it closes.
func TestAChangedRetentionClearsTheBoundsWords(t *testing.T) {
	d := bounded(1)
	dir, now := t.TempDir(), t0
	s := Open(dir, func() time.Time { return now }, d)
	s.Put(d.Name, rec(t0.AddDate(0, 0, -1), 0, 1, 1, 1, 1))
	s.RollUpAndPrune()
	if !s.Bounded(d.Name) {
		t.Fatal("the bound acted and the store does not say so")
	}
	s.Retain(d.Name, d.Hours, d.Days)
	if !s.Bounded(d.Name) {
		t.Error("the same retention, applied again, cleared the words")
	}
	s.Retain(d.Name, d.Hours, d.Days/2)
	if s.Bounded(d.Name) {
		t.Error("a changed retention kept the old retention's words")
	}
}

// AN OLD VERSION COUNTS, AND GOES FIRST: the bound is on every version, and
// an old version's files hold the oldest days.
func TestAnOldVersionCountsAndGoesFirst(t *testing.T) {
	dir, now := t.TempDir(), t0
	old := bounded(0)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	dayByDay(Open(dir, func() time.Time { return now }, old), old, start, 30, &now)
	d := bounded(1000)
	d.Version = 2
	s := Open(dir, func() time.Time { return now }, d)
	dayByDay(s, d, now, 5, &now)
	if got := s.BytesOf(d.Name); got > d.MaxBytes {
		t.Errorf("every version holds %d bytes, past the bound of %d", got, d.MaxBytes)
	}
	v1 := filepath.Join(s.versionDir(d.Name, 1), ndfd.Source, ndfd.Place)
	if _, err := os.Stat(rollupPart(v1, start, 1)); err == nil {
		t.Error("the old version's oldest roll-up was kept past the bound")
	}
	if !s.Bounded(d.Name) {
		t.Error("the bound trimmed the dataset and the store does not say so")
	}
	if _, ok := s.Get(d.Name, ndfd, now.AddDate(0, 0, -1).Add(time.Hour)); !ok {
		t.Error("the current version's newest day went before the old version's")
	}
}

// A REMOVED PART HIDES NO LATER PART: a month's parts are read from the
// directory, so the bound may take its first part and the rest still read.
func TestARemovedPartHidesNoLaterPart(t *testing.T) {
	defer func(b int) { rollUpBudget = b }(rollUpBudget)
	rollUpBudget = 4 << 10
	dir, now := t.TempDir(), t0
	s := openLong(t, dir, &now)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	dayByDay(s, long, start, 24, &now)
	rollAll(s)
	all := len(s.Days(long.Name, ndfd, start, start.AddDate(0, 0, 29), 40))
	if err := os.Remove(rollupPart(longDir(dir), start, 1)); err != nil {
		t.Fatal(err)
	}
	left := s.Days(long.Name, ndfd, start, start.AddDate(0, 0, 29), 40)
	if len(left) == 0 || len(left) >= all {
		t.Fatalf("%d of %d days read with the first part removed: the later parts were hidden, or nothing went", len(left), all)
	}
}
