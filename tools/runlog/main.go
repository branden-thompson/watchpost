// Command runlog runs a gate lane and records the run: one line in
// 06_docs/verify-runs.md, written by this tool and never by hand, saying
// when it ran, the commit it ran on, the tree it tested, the lane, the
// result and how long it took (0.19.0 BUILD red team H-8). The commit a
// green run is chained to carries its line, so a commit's record says
// which run passed it.
//
//	go run ./tools/runlog -lane verify -- make verify-gates
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// logFile is the record, from the repository's root.
const logFile = "06_docs/verify-runs.md"

// logHead opens the record when the first run makes it.
const logHead = "# Verify runs\n\n`tools/runlog` appends one line here per `make verify` or `make verify-docs` run; a line is\nwritten by the tool, never by hand. **Tree** is the git tree the run tested, this file left out, so a line\nsays exactly what passed.\n\n| When (UTC) | HEAD | Tree | Lane | Result | Seconds |\n|---|---|---|---|---|---|\n"

// The refusals.
var (
	errNoLane       = errors.New("usage: runlog -lane <verify|verify-docs> -- <command> [args...]") // a run with no lane or no command
	errNoTree       = errors.New("runlog: git gave no tree for the working tree")
	errNoGitCommand = errors.New("runlog: git asked with nothing to do")
)

func main() {
	lane := flag.String("lane", "", "the lane run: verify or verify-docs")
	flag.Parse()
	if *lane == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, errNoLane)
		os.Exit(2)
	}
	started := time.Now()
	rc := run(flag.Args())
	if err := record(".", *lane, rc, time.Since(started), time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "runlog: this run is not recorded in "+logFile+": "+err.Error()+" - the gates' result stands; fix the record and run again")
	}
	os.Exit(rc)
}

// run runs the lane's command, its output the terminal's, and is its exit
// code; 1 for a command that could not start.
func run(args []string) int {
	if len(args) == 0 {
		return 2 // no command: nothing ran, so nothing passed
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "runlog: "+err.Error())
		return 1
	}
	return 0
}

// record appends the run's line to the record in the repository at dir,
// opening the record when there is none yet.
func record(dir, lane string, rc int, took time.Duration, now time.Time) error {
	if lane == "" {
		return errNoLane
	}
	head, err := git(dir, nil, "rev-parse", "--short", "HEAD")
	if err != nil {
		return err
	}
	tree, err := testedTree(dir)
	if err != nil {
		return err
	}
	result := "green"
	if rc != 0 {
		result = fmt.Sprintf("FAILED (exit %d)", rc)
	}
	path := filepath.Join(dir, logFile)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(logHead), 0o644); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "| %s | %s | %s | %s | %s | %d |\n", now.UTC().Format(time.RFC3339), head, tree, lane, result, int(took.Seconds()))
	return errors.Join(err, f.Close())
}

// testedTree is the git tree of the working tree as the run saw it - every
// file, tracked or not, and the record itself left out - from an index of
// its own, so the repository's index is never touched.
func testedTree(dir string) (string, error) {
	tmp, err := os.MkdirTemp("", "runlog-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }() // a scratch index; nothing to report if it lingers
	env := []string{"GIT_INDEX_FILE=" + filepath.Join(tmp, "index")}
	for _, args := range [][]string{{"read-tree", "HEAD"}, {"add", "-A"}, {"rm", "-q", "--cached", "--ignore-unmatch", logFile}} {
		if _, err := git(dir, env, args...); err != nil {
			return "", err
		}
	}
	tree, err := git(dir, env, "write-tree")
	if err != nil {
		return "", err
	}
	if len(tree) < 12 {
		return "", errNoTree
	}
	return tree[:12], nil
}

// git runs git in dir with the environment given beside the process's,
// and is its output, trimmed.
func git(dir string, env []string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", errNoGitCommand
	}
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
