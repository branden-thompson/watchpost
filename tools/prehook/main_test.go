package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestTheHookRunsOnlyItsOwnTree is FR-10.2 with the BUILD red team's H-9:
// the hook runs tools/prepush from the tree git pushes from, and in a tree
// with none - a clone whose core.hooksPath names another's hook - it says so
// rather than run anything.
func TestTheHookRunsOnlyItsOwnTree(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	t.Chdir(dir)
	if err := run(); err == nil || !strings.Contains(err.Error(), "belongs to another clone") {
		t.Errorf("in a tree with no tools/prepush the hook says %v", err)
	}
}
