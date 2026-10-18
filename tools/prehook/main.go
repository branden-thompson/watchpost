// Command prehook is the pre-push hook git runs (make hooks builds it into
// the clone's git folder): it runs tools/prepush from the tree with go run, so
// the record's tests are always the tree's own, never a binary older than it
// (0.19.0 FR-10.2, BUILD red team H-9). It passes git's lines on stdin through.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	err := run()
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "prehook:", err)
	os.Exit(1)
}

// run runs tools/prepush from the root of the working tree git is pushing
// from.
func run() error {
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return fmt.Errorf("not in a git working tree: %w", err)
	}
	dir := strings.TrimSpace(string(root))
	if dir == "" {
		return errors.New("git named no working tree to push from")
	}
	_, missing := os.Stat(filepath.Join(dir, "tools", "prepush"))
	if missing != nil {
		return errors.New("this tree has no tools/prepush, so the hook belongs to another clone: run make hooks in this one")
	}
	cmd := exec.Command("go", "run", "./tools/prepush")
	cmd.Dir, cmd.Stdin, cmd.Stdout, cmd.Stderr = dir, os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
