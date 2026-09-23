// Command docslane decides whether a change may take the docs lane: every
// module's tests and the document lints, instead of the whole of make verify
// (go-tuiMaps v0.2.0 D-15, applied here as watchpost 0.18.0 D-38).
//
//	go run ./tools/docslane [REPO_ROOT]
//
// It exits zero only when every file changed since the last commit is
// Markdown. The change is everything since that commit, staged or not, deleted
// or untracked, so a file cannot slip past by not having been added yet.
//
// THE LANE IS A RULE THE TOOL HOLDS, NOT ONE ANYONE HAS TO REMEMBER. Markdown
// is not free: tests and lints read it, and a ledger file turned the gate red
// at HEAD during 0.18.0 DISCOVER. So the lane still runs what reads documents;
// what it skips is everything that cannot, and it refuses any change that
// could reach one of those.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if err := run(root, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "docslane:", err)
		os.Exit(1)
	}
}

// run writes the lane's decision for the repository at root, and returns an
// error when the change must go to make verify instead.
func run(root string, out io.Writer) error {
	if _, err := git(root, "rev-parse", "--verify", "-q", "HEAD"); err != nil {
		return errors.New("no commit to compare with, so there is no change to judge: run make verify")
	}
	changed, err := changedSince(root)
	if err != nil {
		return err
	}
	var others []string
	for _, p := range changed { // bounded by the change (P10-02)
		if !strings.HasSuffix(p, ".md") {
			others = append(others, p)
		}
	}
	if len(others) > 0 {
		return fmt.Errorf("the docs lane is for a change of Markdown and nothing else; not Markdown:\n  %s\nrun make verify",
			strings.Join(others, "\n  "))
	}
	if len(changed) == 0 {
		_, err := fmt.Fprintln(out, "docslane: nothing has changed since the last commit")
		return err
	}
	_, err = fmt.Fprintf(out, "docslane: %d Markdown file(s) changed; the docs lane applies\n", len(changed))
	return err
}

// changedSince lists every path that differs from the last commit: tracked
// changes staged or not (deletions included), and untracked files git does
// not ignore.
func changedSince(root string) ([]string, error) {
	tracked, err := git(root, "diff", "--name-only", "--no-renames", "HEAD")
	if err != nil {
		return nil, err
	}
	untracked, err := git(root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, p := range strings.Split(tracked+"\n"+untracked, "\n") { // bounded by git's output (P10-02)
		if p != "" {
			seen[p] = true
		}
	}
	paths := make([]string, 0, len(seen))
	for p := range seen { // bounded by the change (P10-02)
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

// git runs one git command in root and returns what it printed.
func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(b), nil
}
