package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// plant makes a committed repository with one Go file and one Markdown file:
// the lane decides from what has changed since the last commit.
func plant(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "main.go", "package main\n")
	write(t, root, "README.md", "# Planted\n")
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=lane", "-c", "user.email=lane@example.com", "commit", "-q", "-m", "planted"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLaneTakesAMarkdownOnlyChange: changed, new and nested Markdown is the
// lane's to check, and the lane says how much of it there is.
func TestLaneTakesAMarkdownOnlyChange(t *testing.T) {
	root := plant(t)
	write(t, root, "README.md", "# Planted, edited\n")
	write(t, root, "06_docs/new.md", "# New\n")
	var out bytes.Buffer
	if err := run(root, &out); err != nil {
		t.Fatalf("a Markdown-only change was refused: %v", err)
	}
	if !strings.Contains(out.String(), "2 Markdown file(s)") {
		t.Fatalf("the lane did not count the change: %q", out.String())
	}
}

// TestLaneWithNothingChangedSaysSo: an unchanged tree is not a failure, and
// is not silently a pass either.
func TestLaneWithNothingChangedSaysSo(t *testing.T) {
	var out bytes.Buffer
	if err := run(plant(t), &out); err != nil {
		t.Fatalf("an unchanged tree was refused: %v", err)
	}
	if !strings.Contains(out.String(), "nothing has changed") {
		t.Fatalf("the lane did not say nothing changed: %q", out.String())
	}
}

// TestLaneRefusesAnyOtherFile is the positive control: one file that is not
// Markdown, in any state, sends the whole change to make verify.
func TestLaneRefusesAnyOtherFile(t *testing.T) {
	cases := []struct {
		name, want string
		change     func(t *testing.T, root string)
	}{
		{"a changed source file", "main.go", func(t *testing.T, root string) { write(t, root, "main.go", "package main\n\n// changed\n") }},
		{"a new untracked file", "specimens/frame.txt", func(t *testing.T, root string) { write(t, root, "specimens/frame.txt", "a frame\n") }},
		{"a deleted tracked file", "main.go", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "main.go")); err != nil {
				t.Fatal(err)
			}
		}},
		{"a staged file", "main.go", func(t *testing.T, root string) {
			write(t, root, "main.go", "package main\n\n// staged\n")
			cmd := exec.Command("git", "add", "main.go")
			cmd.Dir = root
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git add: %v\n%s", err, out)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := plant(t)
			write(t, root, "README.md", "# Planted, edited\n")
			c.change(t, root)
			var out bytes.Buffer
			err := run(root, &out)
			if err == nil {
				t.Fatalf("the lane accepted a change with %s in it: %q", c.want, out.String())
			}
			if !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "make verify") {
				t.Fatalf("the refusal must name %s and send the change to make verify: %v", c.want, err)
			}
		})
	}
}

// TestLaneRefusesATreeWithNoCommit: with nothing to compare against there is
// no change to judge, and the lane does not guess.
func TestLaneRefusesATreeWithNoCommit(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	var out bytes.Buffer
	if err := run(root, &out); err == nil || !strings.Contains(err.Error(), "make verify") {
		t.Fatalf("a tree with no commit was not sent to make verify: %v", err)
	}
}
