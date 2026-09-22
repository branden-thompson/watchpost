// Command internaltrees prints the one expression for the "internal project
// tree" identity class, for the gates that are not Go: lint-ledger.sh and
// p10-ledger-mirror.py. The Go gate imports package trees directly.
//
//	go run ./tools/internaltrees [REPO_ROOT [HOME]]
//
// REPO_ROOT defaults to the working directory and HOME to $HOME. A failure
// exits non-zero with nothing on stdout, so a caller can never mistake a
// missing rule for an empty one.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/branden-thompson/watchpost/tools/internaltrees/trees"
)

func main() {
	if err := run(os.Args[1:], os.Getenv("HOME"), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "internaltrees:", err)
		os.Exit(1)
	}
}

// run writes the expression for args to out. It is separate from main so the
// failures below are errors a caller sees rather than an exit code, and so
// they can be tested.
//
// THE GUARDS REFUSE SILENCE, NOT MALFORMED INPUT. Each one names a way this
// command could otherwise print something a gate would accept: a third
// argument quietly ignored, an empty root that resolves to whatever directory
// the caller happened to be in, or an empty expression - which lint-ledger.sh
// and p10-ledger-mirror.py both refuse to run without, and which they can only
// refuse if it never reaches them as an empty line on standard output.
func run(args []string, home string, out io.Writer) error {
	if out == nil {
		return errors.New("no destination to write the rule to")
	}
	if len(args) > 2 {
		return fmt.Errorf("at most two arguments (REPO_ROOT, HOME), got %d", len(args))
	}
	root := "."
	if len(args) > 0 {
		root = args[0]
	}
	if root == "" {
		return errors.New("empty repository root: pass a path, or none for the working directory")
	}
	if len(args) > 1 {
		home = args[1]
	}
	expr, err := trees.Expr(root, home)
	if err != nil {
		return err
	}
	if expr == "" {
		return errors.New("empty rule: the gates that read this refuse to run without one")
	}
	// THE WRITE IS CHECKED because a short write is the one way this command
	// could still hand a gate half a rule and exit zero.
	if _, err := fmt.Fprintln(out, expr); err != nil {
		return err
	}
	return nil
}
