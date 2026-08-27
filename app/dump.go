package app

// The diagnostic dump (quality pass Q0, plan §2.1; OQ-2): a running
// dashboard — one that was NOT launched with WATCHPOST_DEBUG_PPROF — can be
// attributed after the fact. A trigger (SIGUSR1 on Unix, /debug/dump on
// the opt-in loopback server everywhere) writes one directory under the
// cache dir:
//
//	profiles/<UTC timestamp>/
//	  heap.pb.gz allocs.pb.gz goroutine.pb.gz threadcreate.pb.gz
//	  counters.json  — post-GC MemStats, goroutines/threads/fds, request
//	                   counters, publish counters, every Gauge
//
// Bounds (red-team IS-7, PH-11, R2-19): one dump in flight, at least
// dumpMinInterval apart, the newest dumpKeep kept, directory 0700 and
// files 0600, never debug.WriteHeapDump (it contains object bytes). Zero
// cost while idle: nothing runs until a trigger arrives.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

const (
	dumpMinInterval = 60 * time.Second
	dumpKeep        = 12
	dumpTimeLayout  = "20060102T150405Z"
)

// profileNames are the runtime profiles a dump writes, in file order.
func profileNames() []string { return []string{"heap", "allocs", "goroutine", "threadcreate"} }

// errDumpTooSoon is the rate bound's answer.
var errDumpTooSoon = errors.New("dump skipped: the previous one is less than a minute old")

// dumper owns the profiles directory and the rate bound.
type dumper struct {
	dir     string
	sources func() diagSources
	stats   func() tty.Stats
	started time.Time

	mu       sync.Mutex
	last     time.Time
	busy     bool
	lastNote string // "<ts> ok <path>" | "<ts> failed: <reason>"
}

func newDumper(dir string, started time.Time, sources func() diagSources, stats func() tty.Stats) *dumper {
	return &dumper{dir: dir, sources: sources, stats: stats, started: started}
}

// dumpRecord is counters.json.
type dumpRecord struct {
	At         time.Time              `json:"at"`
	UptimeS    float64                `json:"uptime_s"`
	Mem        memRecord              `json:"memstats_after_gc"` // read after runtime.GC() in record()
	Goroutines int                    `json:"goroutines"`
	Threads    int                    `json:"threads"`
	FDs        int                    `json:"fds"`
	Requests   httpx.RequestStats     `json:"requests"`
	Pipelines  map[string]publishView `json:"pipelines"`
	Gauges     []Gauge                `json:"gauges"`
}

// memRecord is the MemStats subset the soak statistic reads (plan §1:
// heap_alloc after runtime.GC() is the growth series).
type memRecord struct {
	HeapAlloc   uint64 `json:"heap_alloc"`
	HeapInuse   uint64 `json:"heap_inuse"`
	HeapObjects uint64 `json:"heap_objects"`
	HeapSys     uint64 `json:"heap_sys"`
	StackInuse  uint64 `json:"stack_inuse"`
	Sys         uint64 `json:"sys"`
	NumGC       uint32 `json:"num_gc"`
	TotalAlloc  uint64 `json:"total_alloc"` // cumulative bytes allocated: the soak's allocation-rate series (Q3 gate, R2-7)
	Mallocs     uint64 `json:"mallocs"`     // cumulative heap objects allocated
}

// publishView is one pipeline's publish counters plus the size of its last
// snapshot, measured only here (marshalling every publish would be the
// churn the pass is removing — red-team R2-7).
type publishView struct {
	Publishes     int64 `json:"publishes"`
	Folded        int64 `json:"folded"`
	SnapshotBytes int   `json:"snapshot_bytes"`
}

// Dump writes one dump set and returns its directory.
func (d *dumper) Dump(now time.Time) (string, error) {
	if err := invariant.Check(d.dir != "", "dump: no profiles directory (the OS offered no cache dir)"); err != nil {
		return "", err
	}
	if err := d.begin(now); err != nil {
		return "", err
	}
	defer d.end()
	dir, err := d.write(now)
	d.setNote(now, dir, err)
	return dir, err
}

// begin applies the rate bound and the one-in-flight rule.
func (d *dumper) begin(now time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.busy {
		return errors.New("dump skipped: one is already in progress")
	}
	if !d.last.IsZero() && now.Sub(d.last) < dumpMinInterval {
		return errDumpTooSoon
	}
	d.busy, d.last = true, now
	return nil
}

func (d *dumper) end() {
	d.mu.Lock()
	d.busy = false
	d.mu.Unlock()
}

func (d *dumper) setNote(now time.Time, dir string, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err != nil {
		d.lastNote = now.UTC().Format(dumpTimeLayout) + " failed: " + err.Error()
		return
	}
	d.lastNote = now.UTC().Format(dumpTimeLayout) + " ok " + dir
}

// note is the last outcome for the [S] modal ("" before the first dump).
func (d *dumper) note() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastNote
}

// hint tells the [S] reader how to trigger a dump on this platform.
func (d *dumper) hint() string { return dumpHint(os.Getpid(), d.dir) }

// write produces the directory, the four profiles and counters.json.
func (d *dumper) write(now time.Time) (string, error) {
	dir := filepath.Join(d.dir, now.UTC().Format(dumpTimeLayout))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("dump: %w", err)
	}
	for _, name := range profileNames() {
		if err := writeProfile(filepath.Join(dir, name+".pb.gz"), name); err != nil {
			return dir, err
		}
	}
	raw, err := json.MarshalIndent(d.record(now), "", "  ")
	if err != nil {
		return dir, fmt.Errorf("dump: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "counters.json"), raw, 0o600); err != nil {
		return dir, fmt.Errorf("dump: %w", err)
	}
	return dir, pruneDumps(d.dir, dumpKeep)
}

func writeProfile(path, name string) error {
	p := pprof.Lookup(name)
	if err := invariant.Check(p != nil, "dump: unknown runtime profile "+name); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("dump: %w", err)
	}
	werr := p.WriteTo(f, 0) // debug=0: gzipped protobuf, what pprof -base diffs
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return fmt.Errorf("dump: %s: %w", name, werr)
	}
	return nil
}

// record builds counters.json's content (also served by /debug/counters).
// It runs a GC first: the memory rows describe live memory, not garbage
// waiting for a cycle — the series the soak statistic reads (plan §1).
// Quality pass Q7 found the GC on the dump path only, so every 5-minute
// sample the soaks took through /debug/counters before 0.10.1 was a
// pre-GC reading; the hourly dumps were the post-GC truth.
func (d *dumper) record(now time.Time) dumpRecord {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	g, t, fds := runtimeCounts()
	st := d.stats()
	rec := dumpRecord{
		At: now.UTC(), UptimeS: now.Sub(d.started).Seconds(),
		Mem:        memRecord{HeapAlloc: ms.HeapAlloc, HeapInuse: ms.HeapInuse, HeapObjects: ms.HeapObjects, HeapSys: ms.HeapSys, StackInuse: ms.StackInuse, Sys: ms.Sys, NumGC: ms.NumGC, TotalAlloc: ms.TotalAlloc, Mallocs: ms.Mallocs},
		Goroutines: g, Threads: t, FDs: fds,
		Requests:  st.Requests,
		Pipelines: map[string]publishView{},
	}
	src := d.sources()
	rec.Gauges = src.gauges()
	for i, pub := range [...]*publisher{src.priorityPub, src.recentPub} {
		name := [...]string{"priority", "recent"}[i]
		var last *snapshot.Snapshot
		if pub != nil {
			last = pub.lastSnapshot()
		}
		rec.Pipelines[name] = publishView{Publishes: st.Pipelines[i].Publishes, Folded: st.Pipelines[i].Folded, SnapshotBytes: snapshotBytes(last)}
	}
	return rec
}

// snapshotBytes is the JSON size of a snapshot — the per-publish cost the
// allocation target attributes (plan §1); 0 when nothing has published.
func snapshotBytes(s *snapshot.Snapshot) int {
	if s == nil {
		return 0
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return 0
	}
	return len(raw)
}

// pruneDumps keeps the newest keep dump directories (names sort by time).
func pruneDumps(root string, keep int) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("dump: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for i, extra := 0, len(names)-keep; i < extra; i++ { // bounded: exactly the surplus, oldest first (P10-02)
		if err := os.RemoveAll(filepath.Join(root, names[i])); err != nil {
			return fmt.Errorf("dump: %w", err)
		}
	}
	return nil
}
