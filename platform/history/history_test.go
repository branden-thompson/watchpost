package history

import (
	"compress/gzip"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
)

var t0 = time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)

// grid is the first datasets' shape: NDFD's hours over a box (D-166).
var grid = Dataset{Name: "ndfd-hourly", Title: "NDFD, hourly", Version: 1, Hours: 72 * time.Hour, Days: 30 * 24 * time.Hour,
	Fields: []Field{{Name: "temp", Label: "Temperature", Unit: "°C", Decimals: 1}, {Name: "wind", Label: "Wind", Unit: "km/h", Decimals: 1}}}

// muf is a point every 15 minutes: an ionosonde's maximum usable frequency,
// the kind of source the store must hold beside the weather's (HUM LEAD).
var muf = Dataset{Name: "ionosonde", Title: "Ionosonde", Version: 1, Step: 15 * time.Minute, Hours: 72 * time.Hour,
	Fields: []Field{{Name: "muf", Label: "Maximum usable frequency", Unit: "MHz", Decimals: 2}}}

// alerts carry documents, not numbers over a shape.
var alerts = Dataset{Name: "nws-alerts", Title: "NWS alerts", Version: 1, Hours: 72 * time.Hour}

var box = Shape{Box: geo.Box{W: -120, S: 32, E: -115, N: 36}, Cols: 2, Rows: 2}
var ndfd = Key{Source: "ndfd", Place: "us-a"}

func open(t *testing.T, dir string, now *time.Time) *Store {
	t.Helper()
	return Open(dir, func() time.Time { return *now }, grid, muf, alerts)
}

func rec(at time.Time, issued time.Duration, temps ...float64) Record {
	return Record{Key: ndfd, At: at, IssuedAt: at.Add(issued), Shape: box, Values: map[string][]float64{"temp": temps, "wind": {1, 2, 3, 4}}}
}

// A RECORD IS READ BACK AS IT WAS WRITTEN - AND BY ANOTHER INSTANCE (D-166):
// values rounded to their field's decimals, a missing one missing, not zero.
func TestARecordIsReadBackByAnotherStore(t *testing.T) {
	dir, now := t.TempDir(), t0
	a, b := open(t, dir, &now), open(t, dir, &now)
	if !a.Put(grid.Name, rec(t0.Add(20*time.Minute), 0, 21.04, math.NaN(), -3.96, 10)) {
		t.Fatal("not written")
	}
	got, ok := b.Get(grid.Name, ndfd, t0.Add(59*time.Minute))
	if !ok {
		t.Fatal("another store did not find it")
	}
	v := got.Values["temp"]
	if !got.At.Equal(t0) || v[0] != 21 || !math.IsNaN(v[1]) || v[2] != -4 || v[3] != 10 {
		t.Errorf("read back at %v as %v", got.At, v)
	}
}

// A NEWER ISSUE REPLACES AN OLDER; AN OLDER NEVER REPLACES A NEWER.
func TestTheNewestIssueStands(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	s.Put(grid.Name, rec(t0, 10*time.Minute, 1, 1, 1, 1))
	s.Put(grid.Name, rec(t0, 5*time.Minute, 2, 2, 2, 2))
	if got, _ := s.Get(grid.Name, ndfd, t0); got.Values["temp"][0] != 1 {
		t.Errorf("an older issue replaced a newer: %v", got.Values["temp"])
	}
	s.Put(grid.Name, rec(t0, 20*time.Minute, 3, 3, 3, 3))
	if got, _ := s.Get(grid.Name, ndfd, t0); got.Values["temp"][0] != 3 {
		t.Errorf("a newer issue did not replace: %v", got.Values["temp"])
	}
}

// SEVERAL INSTANCES WRITING ONE DAY LOSE NO HOUR (design section 4): each
// merges and checks its hour stands after its rename.
func TestSeveralWritersOfOneDayLoseNoHour(t *testing.T) {
	dir, now := t.TempDir(), t0
	var wg sync.WaitGroup
	for h := range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			open(t, dir, &now).Put(grid.Name, rec(t0.Truncate(24*time.Hour).Add(time.Duration(h)*time.Hour), 0, float64(h), 0, 0, 0))
		}()
	}
	wg.Wait()
	got := open(t, dir, &now).Range(grid.Name, ndfd, t0.Truncate(24*time.Hour), t0.Truncate(24*time.Hour).Add(23*time.Hour), 100)
	if len(got) != 12 {
		t.Fatalf("%d of 12 hours stand", len(got))
	}
	for i, r := range got {
		if r.Values["temp"][0] != float64(i) {
			t.Errorf("hour %d holds %v: out of order or another's", i, r.Values["temp"][0])
		}
	}
}

// ANY STEP, NOT ONLY THE HOUR (HUM LEAD: MUF): an ionosonde's 15 minutes,
// read back by range and the newest at a time.
func TestAQuarterHourSeriesIsItsOwnStep(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	st := Key{Source: "giro", Place: "pa836"}
	for q := range 8 {
		at := t0.Add(time.Duration(q) * 15 * time.Minute)
		s.Put(muf.Name, Record{Key: st, At: at.Add(3 * time.Minute), IssuedAt: at, Shape: Shape{Cols: 1, Rows: 1}, Values: map[string][]float64{"muf": {12 + float64(q)}}})
	}
	got := s.Range(muf.Name, st, t0, t0.Add(2*time.Hour), 100)
	if len(got) != 8 || !got[1].At.Equal(t0.Add(15*time.Minute)) {
		t.Fatalf("%d quarter hours, the second at %v", len(got), got[min(1, len(got)-1)].At)
	}
	last, ok := s.Latest(muf.Name, st, t0.Add(80*time.Minute), time.Hour)
	if !ok || !last.At.Equal(t0.Add(75*time.Minute)) || last.Values["muf"][0] != 17 {
		t.Errorf("the newest at 16:20 is %v, %v", last.At, last.Values)
	}
}

// A DOCUMENT IS A PAYLOAD (HUM LEAD: every API): alerts are not numbers over
// a shape; their record is the dataset's own JSON, bounded.
func TestADocumentIsRecorded(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	k := Key{Source: "nws", Place: "ca"}
	doc := json.RawMessage(`{"alerts":[{"id":"urn:1","event":"Heat Advisory"}]}`)
	if !s.Put(alerts.Name, Record{Key: k, At: t0, IssuedAt: t0, Doc: doc}) {
		t.Fatal("a document was not recorded")
	}
	got, ok := s.Get(alerts.Name, k, t0)
	if !ok || string(got.Doc) != string(doc) {
		t.Errorf("read back %s", got.Doc)
	}
	if s.Put(alerts.Name, Record{Key: k, At: t0, IssuedAt: t0.Add(time.Minute), Doc: json.RawMessage(`{not json`)}) {
		t.Error("a document that is not JSON was recorded")
	}
}

// ONE INSTANCE FETCHES AN HOUR (design section 4b): of several, one claim
// wins; a recorded hour is claimed by none; a stale claim is taken over.
func TestAnHourIsClaimedOnce(t *testing.T) {
	dir, now := t.TempDir(), t0
	a, b := open(t, dir, &now), open(t, dir, &now)
	if !a.Claim(grid.Name, ndfd, t0) || b.Claim(grid.Name, ndfd, t0) {
		t.Fatal("not exactly one claim won")
	}
	now = now.Add(claimStale + time.Minute)
	if !b.Claim(grid.Name, ndfd, t0) {
		t.Error("a stale claim was not taken over")
	}
	a.Put(grid.Name, rec(t0.Add(time.Hour), 0, 1, 1, 1, 1))
	if a.Claim(grid.Name, ndfd, t0.Add(time.Hour)) {
		t.Error("a recorded hour was claimed")
	}
}

// PAST ITS HOURS, A DAY IS ROLLED UP AND ITS HOURS GO; PAST ITS DAYS, ITS
// YEAR GOES (D-171, D-176): the roll-up written before the hours are removed.
func TestADayIsRolledUpThenPruned(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	day := t0.Truncate(24 * time.Hour)
	s.Put(grid.Name, rec(day.Add(1*time.Hour), 0, 10, math.NaN(), 0, 0))
	s.Put(grid.Name, rec(day.Add(2*time.Hour), 0, 20, math.NaN(), 0, 0))
	now = day.Add(4 * 24 * time.Hour) // past 72 hours
	s.RollUpAndPrune()
	if _, ok := s.Get(grid.Name, ndfd, day.Add(time.Hour)); ok {
		t.Error("the day's hours outlived their retention")
	}
	days := s.Days(grid.Name, ndfd, day, day, 10)
	if len(days) != 1 || days[0].Hours != 2 {
		t.Fatalf("rolled up: %+v", days)
	}
	if d := days[0]; d.Min["temp"][0] != 10 || d.Max["temp"][0] != 20 || d.Mean["temp"][0] != 15 || !math.IsNaN(d.Mean["temp"][1]) {
		t.Errorf("rolled up to min %v max %v mean %v", d.Min["temp"], d.Max["temp"], d.Mean["temp"])
	}
	now = day.AddDate(2, 0, 0)
	s.RollUpAndPrune()
	if len(s.Days(grid.Name, ndfd, day, day, 10)) != 0 {
		t.Error("a year past its retention was kept")
	}
}

// NOTHING BROKEN IS READ (D-124): a truncated file, another schema, another
// version read as absent and are counted.
func TestABrokenFileReadsAsAbsent(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	s.Put(grid.Name, rec(t0, 0, 1, 1, 1, 1))
	path := filesAt(filepath.Join(dir, grid.Name, "v1", "ndfd", "us-a"), t0).bucket // where an hour lives until its day is compacted
	if err := os.WriteFile(path, []byte("not gzip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get(grid.Name, ndfd, t0); ok || s.Stats().Corrupt == 0 {
		t.Errorf("a broken file was read, or not counted: %+v", s.Stats())
	}
	f, _ := os.Create(path)
	zw := gzip.NewWriter(f)
	_ = json.NewEncoder(zw).Encode(dayDoc{Schema: schema + 1, Dataset: grid.Name, Version: 1})
	_ = zw.Close()
	_ = f.Close()
	if _, ok := s.Get(grid.Name, ndfd, t0); ok || s.Stats().VersionMismatch == 0 {
		t.Errorf("another schema was read, or not counted: %+v", s.Stats())
	}
}

// A KEY IS A PATH SEGMENT OR NOTHING: no key reaches outside its series.
func TestAKeyCannotLeaveTheStore(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	for _, k := range []Key{{"..", "x"}, {"ndfd", "../../etc"}, {"NDFD", "x"}, {"", "x"}} {
		if s.Put(grid.Name, Record{Key: k, At: t0, IssuedAt: t0, Shape: box, Values: map[string][]float64{"temp": {1, 2, 3, 4}}}) {
			t.Errorf("key %+v was written", k)
		}
	}
}

// WHAT IS RECORDED CAN BE BROWSED (HUM LEAD: an Analyst mode to come): the
// catalog holds every dataset version's manifest, even one this store did
// not register; a dataset's series and each one's extent are listed.
func TestWhatIsRecordedCanBeBrowsed(t *testing.T) {
	dir, now := t.TempDir(), t0
	open(t, dir, &now).Put(grid.Name, rec(t0, 0, 1, 1, 1, 1))
	Open(dir, func() time.Time { return now }, Dataset{Name: "later", Title: "Written by another version", Version: 3})
	reader := Open(dir, func() time.Time { return now }, grid)
	titles := map[string]bool{}
	for _, d := range reader.Catalog() {
		titles[d.Title] = true
	}
	if !titles["NDFD, hourly"] || !titles["Written by another version"] || !titles["Ionosonde"] {
		t.Errorf("the catalog holds %v", titles)
	}
	if keys := reader.Series(grid.Name, 10); len(keys) != 1 || keys[0] != ndfd {
		t.Errorf("series %v", keys)
	}
	if first, last, ok := reader.Extent(grid.Name, ndfd); !ok || !first.Equal(t0.Truncate(24*time.Hour)) || !last.Equal(first) {
		t.Errorf("extent %v..%v %v", first, last, ok)
	}
}

// A STORE WITH NO ROOT DOES NOTHING, and says so by what it returns.
func TestAStoreWithNoRootDoesNothing(t *testing.T) {
	now := t0
	s := Open("", func() time.Time { return now }, grid)
	if s.Put(grid.Name, rec(t0, 0, 1, 1, 1, 1)) || len(s.Catalog()) != 0 {
		t.Error("a store with no root recorded")
	}
	s.RollUpAndPrune()
}

// A FINISHED DAY IS COMPACTED INTO ONE FILE AND LOSES NO RECORD - WITH TWO
// INSTANCES COMPACTING IT AT ONCE AND A LATE BUCKET BEING WRITTEN (design
// section 4): one compaction holds the day's claim, a bucket file goes only
// once the day's file holds it, and a bucket the compaction missed is kept for
// the next pass.
func TestADayIsCompactedWithoutLosingARecord(t *testing.T) {
	dir, now := t.TempDir(), t0
	day := t0.Truncate(24 * time.Hour)
	for h := range 20 {
		open(t, dir, &now).Put(grid.Name, rec(day.Add(time.Duration(h)*time.Hour), 0, float64(h), 0, 0, 0))
	}
	now = day.Add(25 * time.Hour) // the day is done; its hours are well within 72
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); open(t, dir, &now).RollUpAndPrune() }()
	}
	wg.Add(1)
	go func() { // late: the day's last hours, issued as the compactions run
		defer wg.Done()
		for h := 20; h < 24; h++ {
			open(t, dir, &now).Put(grid.Name, rec(day.Add(time.Duration(h)*time.Hour), 0, float64(h), 0, 0, 0))
		}
	}()
	wg.Wait()
	open(t, dir, &now).RollUpAndPrune() // the next pass takes what the first missed
	s := open(t, dir, &now)
	got := s.Range(grid.Name, ndfd, day, day.Add(23*time.Hour), 100)
	if len(got) != 24 {
		t.Fatalf("%d of 24 hours stand after compaction", len(got))
	}
	for i, r := range got {
		if r.Values["temp"][0] != float64(i) {
			t.Errorf("hour %d holds %v", i, r.Values["temp"][0])
		}
	}
	series := filepath.Join(dir, grid.Name, "v1", "ndfd", "us-a")
	if _, err := os.Stat(filesAt(series, day).day); err != nil {
		t.Error("the day was not compacted into its file")
	}
	if _, err := os.Stat(filesAt(series, day).buckets); !os.IsNotExist(err) {
		t.Error("the day's buckets outlived its compaction")
	}
}

// A STEP IS HELD TO A MINUTE TO A DAY, AND VALUES NEED A SHAPE: a dataset
// asking for seconds records by the minute - a day's file would otherwise
// hold 86,400 buckets - and one asking for a week records by the day; a
// record whose values cover no shape is refused.
func TestAStepIsHeldAndValuesNeedAShape(t *testing.T) {
	for _, tc := range []struct {
		step, want time.Duration
	}{{time.Second, time.Minute}, {0, time.Hour}, {7 * 24 * time.Hour, 24 * time.Hour}, {6 * time.Minute, 6 * time.Minute}} {
		if got := (Dataset{Step: tc.step}).step(); got != tc.want {
			t.Errorf("a step of %v records every %v; want %v", tc.step, got, tc.want)
		}
	}
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	for _, values := range []map[string][]float64{{"temp": {1}}, {"undeclared": {1}}} {
		if s.Put(grid.Name, Record{Key: ndfd, At: t0, IssuedAt: t0, Values: values}) {
			t.Errorf("values %v over no shape were recorded", values)
		}
	}
}

// RETENTION CHANGES WHILE THE STORE RUNS (D-175): a longer hourly retention
// chosen on the Data tab keeps a day the default would have rolled up; the
// manifest says the new retention to any reader.
func TestRetentionChangesWhileRunning(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := open(t, dir, &now)
	day := t0.Truncate(24 * time.Hour)
	s.Put(grid.Name, rec(day.Add(time.Hour), 0, 1, 1, 1, 1))
	if !s.Retain(grid.Name, 7*24*time.Hour, 90*24*time.Hour) {
		t.Fatal("the retention was not taken")
	}
	now = day.Add(4 * 24 * time.Hour) // past 72 hours, within seven days
	s.RollUpAndPrune()
	if _, ok := s.Get(grid.Name, ndfd, day.Add(time.Hour)); !ok {
		t.Error("an hour within the chosen seven days was pruned by the default's 72 hours")
	}
	for _, d := range Open(dir, func() time.Time { return now }).Catalog() {
		if d.Name == grid.Name && d.Hours != 7*24*time.Hour {
			t.Errorf("the manifest says %v; want the chosen seven days", d.Hours)
		}
	}
	if s.Retain("nothing", time.Hour, 0) {
		t.Error("a dataset not held took a retention")
	}
}

// THE STORE'S SIZE AND ITS CLEARING (D-175, D-177): the Data tab shows what
// it holds on disk; Clear history removes every record and roll-up - the
// store's own, nothing beside it - and recording goes on after.
func TestTheStoreIsSizedAndCleared(t *testing.T) {
	root := filepath.Join(t.TempDir(), "history")
	beside := filepath.Join(filepath.Dir(root), "keep.txt")
	if err := os.WriteFile(beside, []byte("not the store's"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := t0
	s := Open(root, func() time.Time { return now }, grid)
	if s.Bytes() != 0 && s.Bytes() > 4096 {
		t.Errorf("an empty store holds %d bytes", s.Bytes())
	}
	s.Put(grid.Name, rec(t0, 0, 1, 1, 1, 1))
	if s.Bytes() <= 0 {
		t.Error("a store with a record says it holds nothing")
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get(grid.Name, ndfd, t0); ok {
		t.Error("a record survived Clear")
	}
	if _, err := os.Stat(beside); err != nil {
		t.Errorf("Clear removed what lay beside the store: %v", err)
	}
	if !s.Put(grid.Name, rec(t0.Add(time.Hour), 0, 2, 2, 2, 2)) {
		t.Error("the store does not record after Clear")
	}
}
