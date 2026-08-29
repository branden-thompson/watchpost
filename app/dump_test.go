package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Quality pass Q0 (plan §2.1, Q0 task 4): the diagnostic dump — files,
// permissions, counters.json content, the rate bound, retention, and the
// invariant that a missing directory is an error, never a panic.

// testDumper builds a dumper whose process started 90 s before `started`
// and whose disk gauges point at the temp dir (never the developer's
// real cache).
func testDumper(t *testing.T, dir string, started time.Time) *dumper {
	t.Helper()
	c, err := httpx.New(httpx.Config{UserAgent: "t (t@example.com)"})
	if err != nil {
		t.Fatal(err)
	}
	return newDumper(dir, started.Add(-90*time.Second),
		func() diagSources {
			return diagSources{clients: []*httpx.Client{c}, cacheDir: dir, profilesDir: dir, voicesDir: dir}
		},
		func() tty.Stats { return tty.Stats{Requests: c.RequestStats()} })
}

func TestDumpWritesProfilesAndCounters(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	d := testDumper(t, root, now)
	dir, err := d.Dump(now)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "20260826T120000Z" {
		t.Fatalf("dump dirs are UTC timestamps, got %s", dir)
	}
	for _, name := range []string{"heap.pb.gz", "allocs.pb.gz", "goroutine.pb.gz", "threadcreate.pb.gz", "counters.json"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s must be written: %v", name, err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
			t.Fatalf("%s must be 0600, got %v", name, info.Mode().Perm())
		}
		if info.Size() == 0 {
			t.Fatalf("%s must not be empty", name)
		}
	}
	if info, _ := os.Stat(dir); runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		t.Fatalf("the dump dir must be 0700, got %v", info.Mode().Perm())
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "counters.json"))
	var rec dumpRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("counters.json must parse: %v", err)
	}
	if rec.Mem.HeapAlloc == 0 || rec.Goroutines == 0 || rec.Threads == 0 || rec.UptimeS < 89 {
		t.Fatalf("counters must carry live memory, goroutines, threads and uptime: %+v", rec)
	}
	want := map[string]bool{"httpx.mem.entries": false, "httpx.neg.entries": false, "tz.cache": false, "disk.cache": false, "disk.profiles": false, "disk.voices": false}
	for _, g := range rec.Gauges {
		if _, ok := want[g.Name]; ok {
			want[g.Name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("gauge %s missing from counters.json", name)
		}
	}
	if _, ok := rec.Pipelines["priority"]; !ok {
		t.Fatal("pipelines must be present even before a publish")
	}
	if d.note() == "" || d.note()[len(d.note())-len(dir):] != dir {
		t.Fatalf("the [S] note must name the dump: %q", d.note())
	}
}

func TestDumpRateBound(t *testing.T) {
	d := testDumper(t, t.TempDir(), time.Now())
	now := time.Now()
	if _, err := d.Dump(now); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Dump(now.Add(30 * time.Second)); err != errDumpTooSoon {
		t.Fatalf("a second dump within a minute must be refused, got %v", err)
	}
	if _, err := d.Dump(now.Add(61 * time.Second)); err != nil {
		t.Fatalf("a dump a minute later must succeed: %v", err)
	}
}

func TestDumpKeepsOnlyTheNewest(t *testing.T) {
	root := t.TempDir()
	for i := range dumpKeep + 3 {
		if err := os.Mkdir(filepath.Join(root, time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC).Format(dumpTimeLayout)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	d := testDumper(t, root, time.Now())
	dir, err := d.Dump(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != dumpKeep {
		t.Fatalf("retention keeps %d dumps, found %d", dumpKeep, len(entries))
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal("the newest dump must survive pruning")
	}
	if _, err := os.Stat(filepath.Join(root, "20260101T000000Z")); err == nil {
		t.Fatal("the oldest dump must be pruned")
	}
}

func TestDumpRequiresADirectory(t *testing.T) {
	d := testDumper(t, "", time.Now())
	if _, err := d.Dump(time.Now()); err == nil {
		t.Fatal("no cache directory must be an error, not a panic")
	}
	if d.note() != "" {
		t.Fatal("a refused dump leaves no note")
	}
}

func TestDirGauge(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a"), []byte("12345"), 0o600)
	_ = os.Mkdir(filepath.Join(dir, "sub"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "sub", "b"), []byte("123"), 0o600)
	if g := dirGauge("x", dir, false); g.Len != 1 || g.Bytes != 5 {
		t.Fatalf("flat gauge counts regular files in the directory only: %+v", g)
	}
	if g := dirGauge("x", dir, true); g.Len != 2 || g.Bytes != 8 {
		t.Fatalf("recursive gauge counts the subtree: %+v", g)
	}
	if g := dirGauge("x", filepath.Join(dir, "missing"), true); g.Len != 0 || g.Bytes != 0 {
		t.Fatalf("a missing directory is a zero gauge: %+v", g)
	}
	if g := dirGauge("x", "", false); g.Len != 0 {
		t.Fatal("an empty path is a zero gauge")
	}
}

func TestPublisherCountsPublishesAndFoldedTriggers(t *testing.T) {
	pb := &publisher{run: func() *snapshot.Snapshot { return nil }}
	pb.Trigger()
	pb.Trigger() // inside the window: folds (a loaded runner may let the window fire between calls)
	pb.Trigger()
	// Every trigger ends as a publish or a fold, but a publish counts only
	// after its run — and a loaded runner (ubuntu CI under -race, PR #4) can
	// have a second one in flight when the first is done. Wait for the sum,
	// not for the first publish.
	waitUntil(t, "three triggers published or folded", func() bool {
		st := pb.stats()
		return st.Publishes+st.Folded == 3
	})
	st := pb.stats()
	if st.Publishes < 1 || st.Publishes+st.Folded != 3 {
		t.Fatalf("every trigger is either published or folded, at least one publish: got %+v", st)
	}
	if st.Publishes == 1 && st.Folded != 2 {
		t.Fatalf("triggers inside one window fold into it, got %+v", st)
	}
}

// Quality pass Q7: the counters' memory rows are read after a GC — the
// post-GC series the soak statistic is defined on — not just the dump's.
func TestCountersRecordRunsAGCFirst(t *testing.T) {
	d := testDumper(t, t.TempDir(), time.Now())
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	rec := d.record(time.Now())
	if rec.Mem.NumGC <= before.NumGC {
		t.Fatalf("record must collect before it reads: NumGC %d -> %d", before.NumGC, rec.Mem.NumGC)
	}
}
