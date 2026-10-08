package history

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// long is a dataset kept a year and more, so a test's days outlive no retention.
var long = Dataset{Name: "long", Version: 1, Hours: 72 * time.Hour, Days: 400 * 24 * time.Hour, Fields: grid.Fields}

func openLong(t *testing.T, dir string, now *time.Time) *Store {
	t.Helper()
	return Open(dir, func() time.Time { return *now }, long)
}

func longDir(dir string) string { return filepath.Join(dir, long.Name, "v1", "ndfd", "us-a") }

// decompressedSize is a roll-up part's size as a read decompresses it.
func decompressedSize(t *testing.T, path string) int64 {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }() // read only: nothing to flush
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	n, err := io.Copy(io.Discard, zr)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// rollAll runs enough passes for every day due to be rolled up.
func rollAll(s *Store) {
	for i := 0; i < 60; i++ {
		s.RollUpAndPrune()
	}
}

// A ROLL-UP NEVER PASSES THE READ CAP (#27, 0.19.0 W1.2): a month's roll-ups are
// written in parts, a new part begun before one would pass the budget, so no
// file the store writes is one it cannot read back - and every day is still
// read, in order.
func TestARollUpNeverPassesTheReadCap(t *testing.T) {
	defer func(b int) { rollUpBudget = b }(rollUpBudget)
	rollUpBudget = 4 << 10 // a few days a part
	dir, now := t.TempDir(), t0
	s := openLong(t, dir, &now)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 40; i++ {
		day := start.AddDate(0, 0, i)
		s.Put(long.Name, rec(day.Add(time.Hour), 0, float64(i), 1, 2, 3))
	}
	now = start.AddDate(0, 0, 44)
	rollAll(s)
	days := s.Days(long.Name, ndfd, start, start.AddDate(0, 0, 39), 100)
	if len(days) != 40 {
		t.Fatalf("read back %d days of 40", len(days))
	}
	for i, d := range days {
		if d.Min["temp"][0] != float64(i) {
			t.Fatalf("day %d read as %v: out of order or lost", i, d.Min["temp"][0])
		}
	}
	parts := 0
	err := filepath.Walk(filepath.Join(longDir(dir), "rollup"), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".json.gz") {
			return err
		}
		parts++
		if n := decompressedSize(t, p); n > int64(rollUpBudget) {
			t.Errorf("%s decompresses to %d bytes, past the budget %d", filepath.Base(p), n, rollUpBudget)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rollupPart(longDir(dir), start, 2)); err != nil || parts < 3 {
		t.Errorf("40 days in %d part(s), September's second part %v: the parts were not split by size", parts, err)
	}
}

// A DAY TOO LARGE FOR ANY PART WAITS: it is never written past the budget, and
// its hours are kept for a later pass.
func TestADayTooLargeForAPartWaits(t *testing.T) {
	defer func(b int) { rollUpBudget = b }(rollUpBudget)
	rollUpBudget = 64 // smaller than any day's entry
	dir, now := t.TempDir(), t0
	s := openLong(t, dir, &now)
	day := t0.Truncate(24 * time.Hour)
	s.Put(long.Name, rec(day.Add(time.Hour), 0, 1, 2, 3, 4))
	now = day.AddDate(0, 0, 4)
	rollAll(s)
	if len(s.Days(long.Name, ndfd, day, day, 10)) != 0 {
		t.Error("a day larger than the budget was written")
	}
	if _, ok := s.Get(long.Name, ndfd, day.Add(time.Hour)); !ok {
		t.Error("the day's hours went although its roll-up waits")
	}
}

// A YEAR'S ROLL-UP FROM BEFORE THE PARTS IS STILL READ (0.18.0's layout,
// rollup/<YYYY>.json.gz): its days come back beside the days written in parts.
func TestALegacyYearRollUpIsStillRead(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := openLong(t, dir, &now)
	legacy := t0.Truncate(24*time.Hour).AddDate(0, 0, -1) // 2026-09-29
	one := func(v float64) values { return values{v, v, v, v} }
	doc := yearDoc{Schema: schema, Dataset: long.Name, Version: long.Version, Key: ndfd, Days: []dayEntry{{
		Date: legacy.Format("2006-01-02"), Shape: box, Hours: 1,
		Stats: map[string]map[string]values{"temp": {"min": one(7), "max": one(7), "mean": one(7)}},
	}}}
	year := filepath.Join(longDir(dir), "rollup", "2026.json.gz")
	if err := os.MkdirAll(filepath.Dir(year), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.writeCounted(year, doc); err != nil {
		t.Fatal(err)
	}
	day := legacy.AddDate(0, 0, 1)
	s.Put(long.Name, rec(day.Add(time.Hour), 0, 9, 9, 9, 9))
	now = day.AddDate(0, 0, 4)
	rollAll(s)
	days := s.Days(long.Name, ndfd, legacy, day, 10)
	if len(days) != 2 || days[0].Min["temp"][0] != 7 || days[1].Min["temp"][0] != 9 {
		t.Fatalf("read back %+v; want the legacy day, then the new one", days)
	}
}

// A PART THAT CANNOT BE READ IS NEVER REWRITTEN (#27, 0.19.0 FR-6.5): a part
// past the read cap, or broken, reads as absent; writing the next day over it
// would replace every day it held. The write is refused and the day's hours are
// kept for a later pass.
func TestAFailedYearReadNeverRewritesTheYear(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := openLong(t, dir, &now)
	day1 := t0.Truncate(24*time.Hour).AddDate(0, 0, -1) // 2026-09-29: both days in one month
	s.Put(long.Name, rec(day1.Add(time.Hour), 0, 10, 10, 10, 10))
	now = day1.AddDate(0, 0, 4)
	rollAll(s)
	if len(s.Days(long.Name, ndfd, day1, day1, 10)) != 1 {
		t.Fatal("the first day was not rolled up")
	}
	part := rollupPart(longDir(dir), day1, 1)
	if err := os.Truncate(part, maxDocBytes+1); err != nil { // past the read cap: unreadable, not absent
		t.Fatal(err)
	}
	day2 := day1.AddDate(0, 0, 1)
	s.Put(long.Name, rec(day2.Add(time.Hour), 0, 20, 20, 20, 20))
	now = day2.AddDate(0, 0, 4)
	rollAll(s)
	if info, err := os.Stat(part); err != nil || info.Size() != maxDocBytes+1 {
		t.Errorf("an unreadable part was rewritten: %v, %v", info.Size(), err)
	}
	if _, ok := s.Get(long.Name, ndfd, day2.Add(time.Hour)); !ok {
		t.Error("the day's hours went although its roll-up was refused")
	}
}
