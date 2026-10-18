package main

import (
	"strings"
	"testing"
)

// TestThePushedTreeIsTheTested is the BUILD red team's H-9: the hook reads
// the working tree, so a tree with changes to tracked files is refused - the
// tests would read what the push does not carry - and a push of commits other
// than the one checked out is let through with a note; a push of the
// checked-out commit, or of a deletion, runs the tests.
func TestThePushedTreeIsTheTested(t *testing.T) {
	head := "1111111111111111111111111111111111111111"
	if _, err := treeIsPushed(" M app/app.go", head, ""); err == nil || !strings.Contains(err.Error(), "commit or stash") {
		t.Errorf("a changed tree: %v; refused", err)
	}
	other := "refs/heads/x 2222222222222222222222222222222222222222 refs/heads/x 3333333333333333333333333333333333333333"
	if note, err := treeIsPushed("", head, other); err != nil || !strings.Contains(note, "not the checked-out commit") {
		t.Errorf("another commit pushed: note %q, %v", note, err)
	}
	for _, line := range []string{"refs/heads/main " + head + " refs/heads/main " + zeroSHA, "(delete) " + zeroSHA + " refs/heads/old 4444444444444444444444444444444444444444", ""} {
		if note, err := treeIsPushed("", head, line); err != nil || note != "" {
			t.Errorf("%q: note %q, %v; the tests run", line, note, err)
		}
	}
}

// TestHooksAreNeverSilenced is the BUILD red team's I-8: make hooks refuses
// where another core.hooksPath, or hooks in git's own folder, would stop
// running, and says how to keep both; with nothing in place, or watchpost's
// own path already set, it installs.
func TestHooksAreNeverSilenced(t *testing.T) {
	ours := "/clone/.git/watchpost-hooks"
	if err := mayInstall("/elsewhere", ours, nil); err == nil || !strings.Contains(err.Error(), "go run ./tools/prepush") {
		t.Errorf("another core.hooksPath: %v; refused with the way to keep both", err)
	}
	if err := mayInstall("", ours, []string{"pre-commit"}); err == nil || !strings.Contains(err.Error(), "pre-commit") {
		t.Errorf("a hook in git's own folder: %v; refused, naming it", err)
	}
	for _, cur := range []string{"", ours, ".git/watchpost-hooks"} {
		if err := mayInstall(cur, ours, nil); err != nil {
			t.Errorf("core.hooksPath %q: %v; installed", cur, err)
		}
	}
}
