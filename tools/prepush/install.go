package main

// install.go - `make hooks` (`go run ./tools/prepush -install`): points this
// clone's git at watchpost's pre-push hook (0.19.0 W0.1, FR-10.2).
//
// It refuses where it would silence a hook already in place - another
// core.hooksPath, or hooks in git's own folder - and says how to keep both
// (BUILD red team I-8). It writes an absolute path, so a linked worktree finds
// the hook too, and the hook it builds, tools/prehook, runs this tool from the
// tree with go run, so the tests are never a binary older than the tree (H-9).

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitOut is a git command's output, trimmed.
func gitOut(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out)), err
}

// install writes the hook and points git at it, unless another hook would
// stop running.
func install() error {
	dir, err := gitOut("rev-parse", "--path-format=absolute", "--git-path", "watchpost-hooks")
	if err != nil {
		return fmt.Errorf("not in a git working tree: %w", err)
	}
	if dir == "" {
		return errors.New("git named no folder for the hook")
	}
	cur, _ := gitOut("config", "--get", "core.hooksPath") // unset is no error here: git exits 1
	defaults, err := gitOut("rev-parse", "--path-format=absolute", "--git-path", "hooks")
	if err != nil {
		return fmt.Errorf("git's hooks folder could not be found: %w", err)
	}
	if err := mayInstall(cur, dir, activeHooks(defaults)); err != nil {
		return err
	}
	root, err := gitOut("rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("the working tree's root could not be found: %w", err)
	}
	hook := filepath.Join(dir, "pre-push")
	if err := os.Remove(hook); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("the hook there before could not be replaced: %w", err) // watchpost's own folder: what is there is an older hook of its own
	}
	build := exec.Command("go", "build", "-trimpath", "-o", hook, "./tools/prehook")
	build.Dir, build.Stdout, build.Stderr = root, os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("the hook could not be built: %w", err)
	}
	if _, err := gitOut("config", "core.hooksPath", dir); err != nil {
		return fmt.Errorf("core.hooksPath could not be set: %w", err)
	}
	fmt.Println("hooks: pre-push runs the record's tests (tools/prepush)")
	return nil
}

// activeHooks are the hooks in git's own folder that would run: every file
// but git's samples.
func activeHooks(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && !strings.HasSuffix(e.Name(), ".sample") {
			out = append(out, e.Name())
		}
	}
	return out
}

// mayInstall says whether the hook can be installed without silencing
// another: refused when core.hooksPath names another folder, or when hooks in
// git's own folder would stop running; each refusal says how to keep both.
func mayInstall(cur, dir string, active []string) error {
	switch {
	case cur != "" && cur != dir && cur != ".git/watchpost-hooks":
		return errors.New("core.hooksPath is " + cur + ", and watchpost's hook would replace it: call 'go run ./tools/prepush' from the pre-push hook there, or unset core.hooksPath and run make hooks again")
	case cur == "" && len(active) > 0:
		return errors.New("these hooks would stop running: " + strings.Join(active, ", ") + ": call 'go run ./tools/prepush' from your own pre-push hook instead, or move them aside and run make hooks again")
	}
	return nil
}
