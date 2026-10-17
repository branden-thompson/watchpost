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
	if err := record(dir, "verify", 0, 95*time.Second, at); err != nil {
		t.Fatal(err)
	}
	if err := record(dir, "verify-docs", 2, 3*time.Second, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
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
	if err := record(dir, "verify", 0, time.Second, time.Now()); err != nil {
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
	if err := record(dir, "", 0, 0, time.Now()); err == nil {
		t.Error("a run with no lane was recorded")
	}
	if rc := run(nil); rc != 2 {
		t.Errorf("a run with no command exited %d; nothing ran, so 2", rc)
	}
	if _, err := git(dir, nil); err == nil {
		t.Error("git ran with nothing to do")
	}
}
