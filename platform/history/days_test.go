package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A DAY IS READ FROM ITS FILES ONCE WHILE THEY STAND (W14 P-18, D-212): a
// day's hours are kept decoded and read again only when its files change -
// a record this store puts, or one another instance puts in the same
// directory - so a map's replay of the hours before reads no file it has
// read while they stand.
func TestADayIsReadFromItsFilesOnceWhileTheyStand(t *testing.T) {
	dir, now := t.TempDir(), t0
	a, b := open(t, dir, &now), open(t, dir, &now)
	if !a.Put(grid.Name, rec(t0, 0, 1, 2, 3, 4)) {
		t.Fatal("not written")
	}
	reads := func() int64 { return a.Stats().DayReads }
	start := reads()
	for range 3 {
		if _, ok := a.Get(grid.Name, ndfd, t0); !ok {
			t.Fatal("the hour was not found")
		}
	}
	if n := reads() - start; n != 1 {
		t.Errorf("three reads of an unchanged day read its files %d times; want once", n)
	}
	if !a.Put(grid.Name, rec(t0.Add(time.Hour), 0, 5, 6, 7, 8)) {
		t.Fatal("not written")
	}
	if _, ok := a.Get(grid.Name, ndfd, t0.Add(time.Hour)); !ok {
		t.Error("an hour this store put was not read back: the day was kept past its change")
	}
	if !b.Put(grid.Name, rec(t0.Add(2*time.Hour), 0, 9, 9, 9, 9)) {
		t.Fatal("not written")
	}
	if _, ok := a.Get(grid.Name, ndfd, t0.Add(2*time.Hour)); !ok {
		t.Error("an hour another store put was not read: the day was kept past another's change")
	}
}

// AN HOUR FILED UNDER ANOTHER DAY IS NOT READ AS THAT DAY'S: a bucket that
// holds the next day's hour, in this day's directory, is a misfiled or
// damaged document - left out of the day, and counted.
func TestAnHourFiledUnderAnotherDayIsNotRead(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	next := t0.Add(24 * time.Hour)
	if !s.Put(grid.Name, rec(t0, 0, 1, 1, 1, 1)) || !s.Put(grid.Name, rec(next, 0, 2, 2, 2, 2)) {
		t.Fatal("not written")
	}
	series := filepath.Join(dir, grid.Name, "v1", "ndfd", "us-a")
	body, err := os.ReadFile(filesAt(series, next).bucket)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filesAt(series, t0).buckets, "2359.json.gz"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	before := s.Stats().Corrupt
	recs := s.Range(grid.Name, ndfd, t0, t0.Add(23*time.Hour), 10)
	if len(recs) != 1 || !recs[0].At.Equal(t0) {
		t.Errorf("the day's records are %v; want its one hour", recs)
	}
	if s.Stats().Corrupt == before {
		t.Error("the misfiled hour was not counted")
	}
}

// A DAY'S DOCUMENT IS READ FOR ITS OWN HOURS ONLY: an hour in it that is not
// the day's, or more hours than the day's step allows, is a damaged document -
// the stray hour left out, the overfull day read as absent - and counted.
func TestADaysDocumentIsReadForItsOwnHoursOnly(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	series := filepath.Join(dir, grid.Name, "v1", "ndfd", "us-a")
	write := func(hours ...time.Time) {
		t.Helper()
		doc := dayDoc{Schema: schema, Dataset: grid.Name, Version: grid.Version, Key: ndfd, Date: t0.Format("2006-01-02")}
		for _, h := range hours {
			doc.Hours = append(doc.Hours, toEntry(grid, rec(h, 0, 1, 1, 1, 1), h))
		}
		f := filesAt(series, t0)
		if err := os.MkdirAll(filepath.Dir(f.day), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := writeGz(f.day, doc); err != nil {
			t.Fatal(err)
		}
	}
	write(t0, t0.Add(25*time.Hour)) // its own hour, and the next day's
	before := s.Stats().Corrupt
	if recs := s.Range(grid.Name, ndfd, t0, t0.Add(23*time.Hour), 10); len(recs) != 1 {
		t.Errorf("the day read %d hours; want its own one", len(recs))
	}
	if s.Stats().Corrupt == before {
		t.Error("the stray hour was not counted")
	}
	start := time.Date(t0.Year(), t0.Month(), t0.Day(), 0, 0, 0, 0, t0.Location())
	var many []time.Time
	for i := range 48 { // every half hour of the day: twice what an hourly step allows
		many = append(many, start.Add(time.Duration(i)*30*time.Minute))
	}
	write(many...)
	if recs := s.Range(grid.Name, ndfd, start, start.Add(23*time.Hour), 60); len(recs) != 0 {
		t.Errorf("a day holding more hours than its step allows read %d; want it read as absent", len(recs))
	}
}
