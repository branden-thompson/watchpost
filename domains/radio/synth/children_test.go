package synth

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// sayStub, set in its environment, makes this test binary a stand-in say
// that sleeps in place of speaking.
const sayStub = "WATCHPOST_SAY_STUB"

// TestMain runs the package's tests, or, as the stand-in say, sleeps.
func TestMain(m *testing.M) {
	if os.Getenv(sayStub) == "1" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// fakeProcs is the process table a sweep sees, and what it ended; refuse
// makes every end fail, and swap changes a process between its two reads.
type fakeProcs struct {
	procs  map[int]Proc
	ended  []int
	refuse bool
	swap   map[int]Proc
	reads  int
}

func (f *fakeProcs) Inspect(pid int) (Proc, bool) {
	f.reads++
	if q, ok := f.swap[pid]; ok && f.reads > 1 {
		return q, true
	}
	p, ok := f.procs[pid]
	return p, ok
}

func (f *fakeProcs) End(pid int) error {
	if f.refuse {
		return errors.New("operation not permitted")
	}
	f.ended = append(f.ended, pid)
	return nil
}

// sweepNow is the clock the sweep tests age records by: a day after none of
// them, so the age-out applies only where a test says.
var sweepNow = time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)

// recordAt writes a child's record as a running watchpost would.
func recordAt(t *testing.T, dir string, pid, parent int, started time.Time) {
	t.Helper()
	if err := writeRecord(dir, pid, parent, started); err != nil {
		t.Fatal(err)
	}
}

// TestAnOrphanedSayIsEndedAtStart is FR-9.2 (F-187, I-4): a say a killed
// watchpost left behind - recorded, the same user, its arguments naming a
// watchpost-say- path, its start time the one recorded, and orphaned - is
// ended by the next start, and its record let go.
func TestAnOrphanedSayIsEndedAtStart(t *testing.T) {
	dir, started := t.TempDir(), time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	recordAt(t, dir, 4242, 4000, started)
	procs := &fakeProcs{procs: map[int]Proc{4242: {UID: 501, PPID: 1, Started: started.Add(time.Second), Args: "say -o /tmp/watchpost-say-123/out.wav -f /tmp/watchpost-say-123/in.txt"}}}
	if n, _ := SweepOrphans(dir, procs, 501, sweepNow); n != 1 || len(procs.ended) != 1 || procs.ended[0] != 4242 {
		t.Fatalf("the orphan: %d ended, %v", n, procs.ended)
	}
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Errorf("the ended orphan's record is kept: %v", left)
	}
}

// TestAnUnrelatedSayIsNeverEnded is FR-9.2's I-4: a process is ended only
// when every proof holds. Another user's, one whose watchpost still runs, one
// whose arguments name no watchpost-say- path, one whose start time is not
// the one recorded (its process ID reused), and a say nothing recorded are
// each left alone; a record whose process is gone is let go, and one whose
// watchpost still runs is kept.
func TestAnUnrelatedSayIsNeverEnded(t *testing.T) {
	started := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	ours := Proc{UID: 501, PPID: 1, Started: started, Args: "say -o /tmp/watchpost-say-9/out.wav"}
	for name, c := range map[string]struct {
		proc Proc
		kept bool
	}{
		"another user's":           {Proc{UID: 502, PPID: 1, Started: started, Args: ours.Args}, false},
		"its watchpost still runs": {Proc{UID: 501, PPID: 4000, Started: started, Args: ours.Args}, true},
		"no watchpost-say- path":   {Proc{UID: 501, PPID: 1, Started: started, Args: "say hello"}, false},
		"not the say program":      {Proc{UID: 501, PPID: 1, Started: started, Args: "/usr/bin/tail -f /tmp/watchpost-say-9/out.wav"}, false},
		"its process ID reused":    {Proc{UID: 501, PPID: 1, Started: started.Add(time.Hour), Args: ours.Args}, false},
	} {
		dir := t.TempDir()
		recordAt(t, dir, 4242, 4000, started)
		procs := &fakeProcs{procs: map[int]Proc{4242: c.proc}}
		if n, _ := SweepOrphans(dir, procs, 501, sweepNow); n != 0 || len(procs.ended) != 0 {
			t.Errorf("%s: ended %v", name, procs.ended)
		}
		if left, _ := os.ReadDir(dir); (len(left) == 1) != c.kept {
			t.Errorf("%s: %d records left; kept %v", name, len(left), c.kept)
		}
	}
	dir := t.TempDir()
	procs := &fakeProcs{procs: map[int]Proc{4242: ours}}
	if n, _ := SweepOrphans(dir, procs, 501, sweepNow); n != 0 || len(procs.ended) != 0 {
		t.Errorf("a say nothing recorded was ended: %v", procs.ended)
	}
	recordAt(t, dir, 5151, 4000, started)
	SweepOrphans(dir, &fakeProcs{procs: map[int]Proc{}}, 501, sweepNow)
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Errorf("the record of a process that is gone is kept: %v", left)
	}
}

// TestNoSayOutlivesAnOrderlyExit is FR-9.1 (F-187): a child started through
// the tracker is ended by EndChildren, as the app's exit and its signal
// handler call it, and its record goes with it.
func TestNoSayOutlivesAnOrderlyExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("say runs on macOS; the tracker's process handling is unix's")
	}
	dir := t.TempDir()
	SetRecordDir(dir)
	t.Cleanup(func() { SetRecordDir("") })
	cmd := exec.Command("sleep", "30")
	if err := startTracked(cmd); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	if live := LiveChildren(); len(live) != 1 || live[0] != cmd.Process.Pid {
		t.Fatalf("the live children are %v", live)
	}
	if left, _ := os.ReadDir(dir); len(left) != 1 {
		t.Fatalf("%d records for one child", len(left))
	}
	if n := EndChildren(); n != 1 {
		t.Fatalf("EndChildren ended %d", n)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the child outlived EndChildren")
	}
	untrack(cmd.Process.Pid)
	if left, _ := os.ReadDir(dir); len(left) != 0 || len(LiveChildren()) != 0 {
		t.Errorf("after the child ended: %d records, %v live", len(left), LiveChildren())
	}
}

// TestSayIsTrackedWhileItRuns is FR-9.1 through Say itself: the say child
// is held while it runs - so EndChildren reaches it - and let go when its
// read is cancelled. This test binary stands in for say on PATH.
func TestSayIsTrackedWhileItRuns(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Say runs macOS's say")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.Symlink(exe, filepath.Join(bin, "say")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(sayStub, "1") // this test binary, run as say, sleeps (TestMain)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := SayVoice{}.Say(ctx, "held"); done <- err }()
	deadline := time.Now().Add(5 * time.Second)
	for len(LiveChildren()) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if len(LiveChildren()) != 1 {
		cancel()
		t.Fatalf("while say runs, the live children are %v", LiveChildren())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("say outlived its cancelled read")
	}
	if live := LiveChildren(); len(live) != 0 {
		t.Errorf("after say ended, it is still held: %v", live)
	}
}

// TestAnOrphanThatCannotBeEndedIsTriedAgain is the BUILD red team's I-3 and
// I-4: an orphan whose end fails keeps its record, so the next start tries
// again, and is counted; a process that changed between the sweep's read and
// the end - its process ID reused - is never ended; a record a day old whose
// process runs under another parent is let go.
func TestAnOrphanThatCannotBeEndedIsTriedAgain(t *testing.T) {
	started := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	orphan := Proc{UID: 501, PPID: 1, Started: started, Args: "/usr/bin/say -o /tmp/watchpost-say-9/out.wav"}
	dir := t.TempDir()
	recordAt(t, dir, 4242, 4000, started)
	if ended, failed := SweepOrphans(dir, &fakeProcs{procs: map[int]Proc{4242: orphan}, refuse: true}, 501, sweepNow); ended != 0 || failed != 1 {
		t.Errorf("an end that failed: %d ended, %d failed", ended, failed)
	}
	if left, _ := os.ReadDir(dir); len(left) != 1 {
		t.Errorf("the record of an orphan not ended was let go: %v", left)
	}
	reused := Proc{UID: 501, PPID: 1, Started: started.Add(time.Hour), Args: orphan.Args}
	procs := &fakeProcs{procs: map[int]Proc{4242: orphan}, swap: map[int]Proc{4242: reused}}
	if ended, _ := SweepOrphans(dir, procs, 501, sweepNow); ended != 0 || len(procs.ended) != 0 {
		t.Errorf("a process that changed before it was ended was ended: %v", procs.ended)
	}
	recordAt(t, dir, 5151, 4000, started)
	running := Proc{UID: 501, PPID: 4000, Started: started, Args: orphan.Args}
	SweepOrphans(dir, &fakeProcs{procs: map[int]Proc{5151: running}}, 501, started.Add(recordMaxAge+time.Minute))
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Errorf("a record a day old whose process runs under another parent is kept: %v", left)
	}
}
