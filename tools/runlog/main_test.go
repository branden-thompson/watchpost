package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// repo is a throwaway repository with one commit and an untracked file.
func repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "--allow-empty", "-m", "first"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "06_docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "work.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestEachRunIsRecorded is H-8: a run, green or failed, appends its line -
// when, HEAD, the tree it tested, the lane, the result, the seconds - and
// the first run opens the record with its header.
func TestEachRunIsRecorded(t *testing.T) {
	dir := repo(t)
	at := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	tested, err := testedTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rc, err := record(dir, "verify", 0, tested, 95*time.Second, at); err != nil || rc != 0 {
		t.Fatal(rc, err)
	}
	if rc, err := record(dir, "verify-docs", 2, tested, 3*time.Second, at.Add(time.Hour)); err != nil || rc != 2 {
		t.Fatal(rc, err)
	}
	b, err := os.ReadFile(filepath.Join(dir, logFile))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if !strings.HasPrefix(lines[0], "# Verify runs") {
		t.Errorf("the record opens with %q", lines[0])
	}
	green, failed := lines[len(lines)-2], lines[len(lines)-1]
	if !strings.HasPrefix(green, "| 2026-10-17T12:00:00Z | ") || !strings.HasSuffix(green, " | verify | green | 95 |") {
		t.Errorf("the green run is %q", green)
	}
	if !strings.HasSuffix(failed, " | verify-docs | FAILED (exit 2) | 3 |") {
		t.Errorf("the failed run is %q", failed)
	}
}

// TestTheTreeIsWhatTheRunSaw is H-8: the tree recorded is the working tree,
// untracked files in, the record left out - so a change to a file changes
// it, a line written to the record does not, and the repository's own index
// is untouched.
func TestTheTreeIsWhatTheRunSaw(t *testing.T) {
	dir := repo(t)
	before, err := testedTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := record(dir, "verify", 0, before, time.Second, time.Now()); err != nil {
		t.Fatal(err)
	}
	if after, _ := testedTree(dir); after != before {
		t.Errorf("writing the record changed the tree: %s, then %s", before, after)
	}
	if err := os.WriteFile(filepath.Join(dir, "work.go"), []byte("package y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, _ := testedTree(dir); changed == before {
		t.Error("a changed file left the tree as it was")
	}
	if staged, _ := git(dir, nil, "diff", "--cached", "--name-only"); staged != "" {
		t.Errorf("the repository's index was touched: %q", staged)
	}
}

// TestATreeChangedDuringTheRunIsNoPass is the REVIEW red team's RC-3: the
// tree is taken before the run; a file changed while it ran makes the line
// FAILED, both trees named, and the run's exit code not 0, whatever the
// gates said - the commit chained to it does not happen.
func TestATreeChangedDuringTheRunIsNoPass(t *testing.T) {
	dir := repo(t)
	tested, err := testedTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "work.go"), []byte("package edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc, err := record(dir, "verify", 0, tested, time.Second, time.Now())
	if err != nil || rc == 0 {
		t.Fatalf("a tree edited during a green run: exit %d, %v; want not 0", rc, err)
	}
	b, err := os.ReadFile(filepath.Join(dir, logFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "| "+tested+" | verify | FAILED (the tree changed during the run: "+tested+" tested, ") {
		t.Errorf("the record says:\n%s", b)
	}
}

// TestARunWithNoRecordIsNoPass is the REVIEW red team's RC-3: a green run
// whose line cannot be written, or that runs outside a repository, exits
// failing - a run with no record is no pass - and a failed run keeps its own
// code.
func TestARunWithNoRecordIsNoPass(t *testing.T) {
	dir := repo(t)
	tested, err := testedTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, logFile), 0o755); err != nil { // the record a directory: no line can be written
		t.Fatal(err)
	}
	if rc, err := record(dir, "verify", 0, tested, time.Second, time.Now()); err == nil || rc != 1 {
		t.Errorf("a green run whose line cannot be written: exit %d, %v; want 1 and the reason", rc, err)
	}
	if rc, err := record(dir, "verify", 2, tested, time.Second, time.Now()); err == nil || rc != 2 {
		t.Errorf("a failed run whose line cannot be written: exit %d, %v; want its own 2", rc, err)
	}
	if rc, err := record(t.TempDir(), "verify", 0, tested, time.Second, time.Now()); err == nil || rc != 1 {
		t.Errorf("a green run outside a repository: exit %d, %v; want 1 and the reason", rc, err)
	}
	// R3C-8: the tree read again at the run's end, and the line's write,
	// each fail the run when they fail.
	unreadable := repo(t)
	before, err := testedTree(unreadable)
	if err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(unreadable, "locked.go")
	if err := os.WriteFile(locked, []byte("package x\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(locked, 0o644) }()
	if rc, err := record(unreadable, "verify", 0, before, time.Second, time.Now()); err == nil || rc != 1 {
		t.Errorf("a green run whose tree cannot be read at its end: exit %d, %v; want 1 and the reason", rc, err)
	}
	writable := repo(t)
	tree, err := testedTree(writable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(writable, logFile), []byte(logHead), 0o644); err != nil {
		t.Fatal(err)
	}
	was := openLog
	openLog = func(name string, _ int, _ os.FileMode) (*os.File, error) { return os.Open(name) } // read-only: the write fails
	defer func() { openLog = was }()
	if rc, err := record(writable, "verify", 0, tree, time.Second, time.Now()); err == nil || rc != 1 {
		t.Errorf("a green run whose line cannot be written: exit %d, %v; want 1 and the reason", rc, err)
	}
}
