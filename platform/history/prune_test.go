package history

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// brief keeps its hours 72 hours and rolls up nothing: a dataset whose every
// day past retention is removed outright.
var brief = Dataset{Name: "brief", Title: "Brief", Version: 1, Hours: 72 * time.Hour,
	Fields: []Field{{Name: "temp", Label: "Temperature", Unit: "°C", Decimals: 1}}}

// briefKey is the i-th of many series, each a place of one source.
func briefKey(i int) Key { return Key{Source: "src", Place: "p" + strconv.Itoa(1000+i)} }

func briefRec(i int, at time.Time) Record {
	return Record{Key: briefKey(i), At: at, IssuedAt: at, Shape: Shape{Cols: 1, Rows: 1}, Values: map[string][]float64{"temp": {float64(i)}}}
}

// fastWrites has the test's writes skip the disk's full flush, which takes
// milliseconds a file on macOS.
func fastWrites(t *testing.T) {
	syncFile = func(*os.File) error { return nil }
	t.Cleanup(func() { syncFile = (*os.File).Sync })
}

// storeShape is what a dataset's directory holds: its record files (every
// regular file but the manifest), its directories, and the day files and day
// directories dated before cut.
func storeShape(t *testing.T, vdir string, cut time.Time) (files, dirs, expired int) {
	t.Helper()
	_ = filepath.WalkDir(vdir, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() {
			dirs++
		} else if e.Name() != "manifest.json.gz" {
			files++
		}
		rel, _ := filepath.Rel(vdir, path)
		parts := strings.Split(filepath.ToSlash(rel), "/") // source, place, month, day
		if len(parts) != 4 {
			return nil
		}
		if d, err := time.Parse("2006-01/02", parts[2]+"/"+trimExt(parts[3])); err == nil && d.Before(cut) {
			expired++
		}
		return nil
	})
	return files, dirs, expired
}

// EVERY SERIES PAST RETENTION IS PRUNED, HOWEVER MANY THERE ARE (QA-4, PF-1):
// a pass is bounded, so the next begins where it stopped, and a series emptied
// of its days leaves no directory behind - 600 one-record series all expire
// within a few bounded passes, and only the dataset's own directories stand.
func TestEverySeriesPastRetentionIsPruned(t *testing.T) {
	fastWrites(t)
	dir, now := t.TempDir(), t0
	s := Open(dir, func() time.Time { return now }, brief)
	const n = 600
	for i := range n {
		if !s.Record(brief.Name, briefRec(i, t0)) {
			t.Fatal("not written")
		}
	}
	now = t0.Add(5 * 24 * time.Hour)
	for range 40 { // each pass removes at most maxPruneRemovals: 600 need ten
		s.RollUpAndPrune()
	}
	vdir := filepath.Join(dir, brief.Name, "v1")
	files, dirs, _ := storeShape(t, vdir, now)
	if files != 0 {
		t.Errorf("%d expired record files of %d outlived 40 passes", files, n)
	}
	if dirs > 2 { // the version's directory and its source's
		t.Errorf("%d directories stand after every series expired; want the dataset's own", dirs)
	}
}

// A STORE PRUNED HOURLY STAYS BOUNDED (QA-4, PF-1): on an accelerated clock,
// 40 series each recorded every 10 hours and a pass each hour for four days,
// no day lingers a day past its retention, a month emptied goes and the
// directories stay a series' few. Its counts bound it to seconds; its time is
// logged, not judged, as a loaded machine stretches any wall clock.
func TestAPruneSoakStaysBounded(t *testing.T) {
	start := time.Now()
	fastWrites(t)
	dir, now := t.TempDir(), t0
	s := Open(dir, func() time.Time { return now }, brief)
	const n, perHour, hours = 40, 4, 4 * 24
	vdir := filepath.Join(dir, brief.Name, "v1")
	next := 0
	for h := range hours {
		now = t0.Add(time.Duration(h) * time.Hour)
		for range perHour {
			if !s.Record(brief.Name, briefRec(next%n, now)) {
				t.Fatal("not written")
			}
			next++
		}
		s.RollUpAndPrune()
	}
	cut := now.Add(-brief.Hours).Truncate(24*time.Hour).AddDate(0, 0, -1) // a day's lag allowed
	_, dirs, expired := storeShape(t, vdir, cut)
	if expired != 0 {
		t.Errorf("%d days lingered more than a day past their retention", expired)
	}
	if most := n*(1+2+5) + 2; dirs > most { // a series, two months, a few days' buckets
		t.Errorf("%d directories stand; want at most %d", dirs, most)
	}
	if _, err := os.Stat(filepath.Join(vdir, "src", "p1000", "2026-09")); !os.IsNotExist(err) {
		t.Error("a month emptied of its days was left standing")
	}
	t.Logf("the soak took %v (bounded by its counts: under 10 s on an idle machine, with -race)", time.Since(start))
}

// A PUT DOES NOT READ ITS DAY AGAIN (PF-8): the day's hours as recording reads
// them are held, and the record written is added - a day's 24 puts read its
// files once. Recording holds no document (D-242), so reading the day back
// reads it once more, and once only: every hour reads back as it was put. A
// record another instance puts in the same day is still seen.
func TestAPutDoesNotReadItsDayAgain(t *testing.T) {
	dir, now := t.TempDir(), t0
	a, b := open(t, dir, &now), open(t, dir, &now)
	day := t0.Truncate(24 * time.Hour)
	for h := range 24 {
		if !a.Record(grid.Name, rec(day.Add(time.Duration(h)*time.Hour), 0, float64(h), 0, 0, 0)) {
			t.Fatal("not written")
		}
	}
	if got := a.Stats().DayReads; got != 1 {
		t.Errorf("24 puts read their day %d times; want once", got)
	}
	for h := range 24 {
		r, ok := a.Get(grid.Name, ndfd, day.Add(time.Duration(h)*time.Hour))
		if !ok || r.Values["temp"][0] != float64(h) {
			t.Errorf("hour %d reads back as %v (%v)", h, r.Values["temp"], ok)
		}
	}
	if reads := a.Stats().DayReads; reads != 2 {
		t.Errorf("reading the day's 24 hours back: %d reads in all; want the puts' one and one more", reads)
	}
	if days, bytes, indexed := a.Memory(); days != 1 || bytes <= 0 || indexed != 1 {
		t.Errorf("held: %d days weighing %d, %d indexed; want the day once for readers and once for recording", days, bytes, indexed)
	}
	if !a.Record(grid.Name, rec(day.Add(5*time.Hour), time.Minute, 50, 0, 0, 0)) {
		t.Fatal("a newer issue was not written")
	}
	if r, _ := a.Get(grid.Name, ndfd, day.Add(5*time.Hour)); r.Values["temp"][0] != 50 {
		t.Errorf("a newer issue reads back as %v", r.Values["temp"][0])
	}
	if !b.Record(grid.Name, rec(day.Add(6*time.Hour), time.Hour, 60, 0, 0, 0)) {
		t.Fatal("not written")
	}
	if r, _ := a.Get(grid.Name, ndfd, day.Add(6*time.Hour)); r.Values["temp"][0] != 60 {
		t.Errorf("another instance's record reads as %v: the day was kept past its change", r.Values["temp"][0])
	}
}

// THE STORE KEEPS ITS SIZE AS IT GOES (PF-2): its bytes, each dataset's and the
// oldest day it holds are kept as records are put, rolled up, pruned and
// cleared - what a walk of the store finds - and reading them walks nothing:
// a file laid beside the records afterwards is not seen until the store is
// measured again.
func TestTheStoreKeepsItsSizeAsItGoes(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	if s.Bytes() != treeBytes(dir) {
		t.Errorf("an empty store holds %d bytes; a walk finds %d", s.Bytes(), treeBytes(dir))
	}
	day := t0.Truncate(24 * time.Hour)
	for h := range 6 {
		s.Record(grid.Name, rec(day.Add(time.Duration(h)*time.Hour), 0, float64(h), 0, 0, 0))
		s.Record(grid.Name, rec(day.Add(time.Duration(h)*time.Hour), time.Minute, float64(h), 1, 0, 0)) // a newer issue replaces its bucket
	}
	s.Record(muf.Name, Record{Key: Key{Source: "giro", Place: "pa836"}, At: day.Add(-48 * time.Hour), IssuedAt: day, Shape: Shape{Cols: 1, Rows: 1}, Values: map[string][]float64{"muf": {12}}})
	check := func(when string, since bool) {
		t.Helper()
		if got, want := s.Bytes(), treeBytes(dir); got != want {
			t.Errorf("%s: the store says %d bytes; a walk finds %d", when, got, want)
		}
		for _, d := range []Dataset{grid, muf, alerts} {
			if got, want := s.BytesOf(d.Name), treeBytes(filepath.Join(dir, d.Name)); got != want {
				t.Errorf("%s: %s says %d bytes; a walk finds %d", when, d.Name, got, want)
			}
		}
		if !since {
			return
		}
		held, ok := s.Since()
		walked, wok := walkSince(dir)
		if ok != wok || !held.Equal(walked) {
			t.Errorf("%s: the store says it holds since %v (%v); a walk finds %v (%v)", when, held, ok, walked, wok)
		}
	}
	check("after puts", true)
	now = day.Add(25 * time.Hour) // the day is done: compacted
	s.measured = now              // measured now: the pass's own count is what is checked
	s.RollUpAndPrune()
	check("after a compaction", true)
	now = day.Add(4 * 24 * time.Hour) // past 72 hours: rolled up and pruned
	s.measured = now                  // measured now: the pass's own count is what is checked
	s.RollUpAndPrune()
	check("after a roll-up and prune", false)
	s.measured = now.Add(-remeasureEvery) // due to be measured again
	s.RollUpAndPrune()
	check("measured again after a prune", true)
	if err := os.WriteFile(filepath.Join(dir, "stray.bin"), make([]byte, 4096), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := s.Bytes(); got == treeBytes(dir) {
		t.Error("the size was walked again on reading it")
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	check("after Clear", true)
}

// ONE DATASET IS FORGOTTEN, NOTHING ELSE (D-269): Clear map data forgets EPA's
// UV readings by city - its records and roll-ups, counted - and every other
// dataset's records stand; the dataset goes on recording.
func TestOneDatasetIsForgotten(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	s.Record(grid.Name, rec(t0, 0, 1, 1, 1, 1))
	st := Key{Source: "giro", Place: "pa836"}
	for q := range 3 {
		at := t0.Add(time.Duration(q) * 15 * time.Minute)
		s.Record(muf.Name, Record{Key: st, At: at, IssuedAt: at, Shape: Shape{Cols: 1, Rows: 1}, Values: map[string][]float64{"muf": {12}}})
	}
	n, err := s.ForgetDataset(muf.Name)
	if err != nil || n != 3 {
		t.Errorf("forgot %d records (%v); want the 3 written", n, err)
	}
	if got := s.Range(muf.Name, st, t0, t0.Add(time.Hour), 10); len(got) != 0 {
		t.Errorf("%d records survived", len(got))
	}
	if _, ok := s.Get(grid.Name, ndfd, t0); !ok {
		t.Error("another dataset's record was forgotten too")
	}
	if s.BytesOf(muf.Name) != treeBytes(filepath.Join(dir, muf.Name)) || s.Bytes() != treeBytes(dir) {
		t.Errorf("the size is not kept: %d against %d", s.Bytes(), treeBytes(dir))
	}
	if !s.Record(muf.Name, Record{Key: st, At: t0, IssuedAt: t0, Shape: Shape{Cols: 1, Rows: 1}, Values: map[string][]float64{"muf": {13}}}) {
		t.Error("the dataset does not record after it was forgotten")
	}
	if n, err := s.ForgetDataset("not-held"); n != 0 || err != nil {
		t.Errorf("a dataset not held forgot %d (%v)", n, err)
	}
}

// TestADatasetsSizeIsNeverBelowNothing: another instance may remove files
// this one counted; the size it keeps is held at nothing, never below.
func TestADatasetsSizeIsNeverBelowNothing(t *testing.T) {
	s := Open(t.TempDir(), time.Now)
	s.sizes = map[string]int64{"ds": 10}
	s.grow(filepath.Join(s.root, "ds", "x.gz"), -25)
	if got := s.sizes["ds"]; got != 0 {
		t.Errorf("the size is %d after removing more than was counted; want 0", got)
	}
}

// RECORDING HOLDS NO DOCUMENT, AND THE DAYS HELD FOR READERS ARE WEIGHED
// (D-242): a day of puts is indexed for recording, its hours and their issue
// times alone, and no day is held for readers until one is read; the days held
// for readers stay within dayRules' weight, the least recently read out.
func TestRecordingHoldsNoDocument(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	day := t0.Truncate(24 * time.Hour)
	for h := range 24 {
		if !s.Record(grid.Name, rec(day.Add(time.Duration(h)*time.Hour), 0, float64(h), 0, 0, 0)) {
			t.Fatal("not written")
		}
	}
	if days, bytes, indexed := s.Memory(); days != 0 || bytes != 0 || indexed != 1 {
		t.Errorf("after a day of puts: %d days held for readers weighing %d, %d indexed; want none held, one indexed", days, bytes, indexed)
	}
	if !s.Record(grid.Name, rec(day.Add(3*time.Hour), 0, 99, 0, 0, 0)) {
		t.Fatal("an equal issue was not skipped quietly")
	}
	if r, _ := s.Get(grid.Name, ndfd, day.Add(3*time.Hour)); r.Values["temp"][0] != 3 {
		t.Errorf("an equal issue replaced the hour: %v", r.Values["temp"][0])
	}
	if days, _, _ := s.Memory(); days != 1 {
		t.Fatalf("a read holds %d days for readers; want the one read", days)
	}
	// A store that must read the day to record, and then skips the record as
	// stale, holds the day for no reader.
	fresh := open(t, dir, &now)
	if !fresh.Record(grid.Name, rec(day.Add(3*time.Hour), 0, 77, 0, 0, 0)) {
		t.Fatal("an equal issue was not skipped quietly")
	}
	if days, _, indexed := fresh.Memory(); days != 0 || indexed != 1 {
		t.Errorf("a store that read the day to record holds %d days for readers, %d indexed; want none, one", days, indexed)
	}
	if !s.Record(grid.Name, rec(day.Add(4*time.Hour), time.Minute, 44, 0, 0, 0)) {
		t.Fatal("a newer issue was not written")
	}
	if days, bytes, _ := s.Memory(); days != 0 || bytes != 0 {
		t.Errorf("a write after a read leaves %d days weighing %d held for readers: the day's documents are no longer its files", days, bytes)
	}
	// Another instance's newer issue is seen by recording: this instance's
	// older issue of that hour is then skipped.
	other := open(t, dir, &now)
	if !other.Record(grid.Name, rec(day.Add(6*time.Hour), time.Hour, 60, 0, 0, 0)) {
		t.Fatal("not written")
	}
	if !s.Record(grid.Name, rec(day.Add(6*time.Hour), 30*time.Minute, 30, 0, 0, 0)) {
		t.Fatal("an older issue was not skipped quietly")
	}
	if r, _ := s.Get(grid.Name, ndfd, day.Add(6*time.Hour)); r.Values["temp"][0] != 60 {
		t.Errorf("an older issue replaced another instance's newer one: %v", r.Values["temp"][0])
	}
	big := heldDay{entries: []hourEntry{{Doc: make([]byte, dayRules.MaxBytes/2+1)}}}
	s.days.Put("a", big)
	s.days.Put("b", big)
	if n, w := s.days.Held(); n != 1 || w != big.weight() {
		t.Errorf("two days of over half the bound each: %d held, weighing %d", n, w)
	}
}
