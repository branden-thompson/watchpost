package synth

// children.go — the say processes watchpost starts, kept so none outlives
// it (0.19.0 W9.2, FR-9.1, FR-9.2, F-187). Each live child is held in
// memory, for EndChildren at exit, and recorded on disk - its process ID,
// its parent's and when it started - so a watchpost killed outright leaves
// a record the next start can sweep. A sweep ends a recorded process only
// when every proof holds (I-4); a record whose watchpost still runs is kept.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Proc is what the sweep is told of a process: its owner, its parent, when
// it started and its arguments.
type Proc struct {
	UID, PPID int
	Started   time.Time
	Args      string
}

// Procs is the process table the sweep reads and ends from.
type Procs interface {
	Inspect(pid int) (Proc, bool)
	End(pid int) error
}

// sayMarker is in every say child's arguments: its files are in a
// watchpost-say- directory.
const sayMarker = "watchpost-say-"

// startSlack is how far a process's start time may lie from the one
// recorded and still be the process recorded: the record is written just
// after the start, and ps reads to the second.
const startSlack = 5 * time.Second

// children are the say processes running now, and where they are recorded.
var children = struct {
	mu     sync.Mutex
	live   map[int]*exec.Cmd
	record string
}{live: map[int]*exec.Cmd{}}

// SetRecordDir is where children are recorded; empty records none.
func SetRecordDir(dir string) {
	children.mu.Lock()
	defer children.mu.Unlock()
	children.record = dir
}

// startTracked starts a child, holds it and records it.
func startTracked(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	children.mu.Lock()
	defer children.mu.Unlock()
	children.live[pid] = cmd
	if children.record != "" {
		_ = writeRecord(children.record, pid, os.Getpid(), time.Now()) // no record is no sweep later, never a wrong one
	}
	return nil
}

// untrack lets an ended child go.
func untrack(pid int) {
	children.mu.Lock()
	defer children.mu.Unlock()
	delete(children.live, pid)
	if children.record != "" {
		_ = os.Remove(recordPath(children.record, pid))
	}
}

// LiveChildren are the say processes running now, by process ID.
func LiveChildren() []int {
	children.mu.Lock()
	defer children.mu.Unlock()
	out := make([]int, 0, len(children.live))
	for pid := range children.live {
		out = append(out, pid)
	}
	slices.Sort(out)
	return out
}

// EndChildren ends every say process running now (FR-9.1): the app calls it
// as it exits and when it is told to terminate. It says how many it ended.
func EndChildren() int {
	children.mu.Lock()
	defer children.mu.Unlock()
	n := 0
	for _, cmd := range children.live {
		if cmd.Process != nil && cmd.Process.Kill() == nil {
			n++
		}
	}
	return n
}

// recordPath is one child's record.
func recordPath(dir string, pid int) string { return filepath.Join(dir, "say-"+strconv.Itoa(pid)) }

// writeRecord records a child: its parent and when it started.
func writeRecord(dir string, pid, parent int, started time.Time) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	line := fmt.Sprintf("%d %d\n", parent, started.Unix())
	return os.WriteFile(recordPath(dir, pid), []byte(line), 0o600)
}

// SweepOrphans ends the say processes a watchpost killed outright left
// behind (FR-9.2): a recorded process is ended only when it is the user's
// own, orphaned (its parent gone, it is the init process's), its arguments
// name a watchpost-say- path, and it started when the record says. A record
// whose process is gone, or that was ended, is let go; one whose watchpost
// still runs is kept. It says how many it ended.
func SweepOrphans(dir string, procs Procs, uid int) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	ended := 0
	for _, e := range entries {
		pid, ok := strings.CutPrefix(e.Name(), "say-")
		n, err := strconv.Atoi(pid)
		if !ok || err != nil || n <= 1 {
			continue
		}
		path := filepath.Join(dir, e.Name())
		started, ok := recordedStart(path)
		p, alive := procs.Inspect(n)
		switch {
		case !alive || !ok:
			_ = os.Remove(path) // gone, or a record that cannot be read
		case orphanOf(p, uid, started) && procs.End(n) == nil:
			ended++
			_ = os.Remove(path)
		case p.PPID != 1:
			// its watchpost still runs: the record is kept
		default:
			_ = os.Remove(path) // not provably watchpost's: never ended, its record let go
		}
	}
	return ended
}

// recordedStart is when a record says its child started.
func recordedStart(path string) (time.Time, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, false
	}
	fields := strings.Fields(string(raw))
	if len(fields) != 2 {
		return time.Time{}, false
	}
	secs, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(secs, 0), true
}

// orphanOf is whether a process is provably a say watchpost started and
// left (I-4): the same user, orphaned, a watchpost-say- path in its
// arguments, and started when its record says.
func orphanOf(p Proc, uid int, started time.Time) bool {
	gap := p.Started.Sub(started)
	return p.UID == uid && p.PPID == 1 && strings.Contains(p.Args, sayMarker) && gap <= startSlack && gap >= -startSlack
}
