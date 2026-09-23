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
// THE GUARDS REFUSE SILENCE, NOT MALFORMED INPUT: a third argument quietly
// ignored, and an empty root that would resolve to whatever directory the
// caller happened to be in.
//
// THE SET IS DELIBERATELY SMALL. A check a static reader can prove will never
// fail is not a guard — P10 Rule 5 refuses it — so each of these is one a
// caller can really trip. The consumers' own refusal of an empty rule is the
// guard that matters, and it lives in them.
func run(args []string, home string, out io.Writer) error {
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
	// AN EXPLICITLY EMPTY HOME IS A CALLER'S MISTAKE. Empty means "derive
	// nothing", so passing it as an argument asks for a rule with the derived
	// half missing, and a gate handed a shortened rule still prints a pass. An
	// unset HOME in the environment is different and stays allowed: that is a
	// machine with no home directory, not a caller asking for less.
	if len(args) > 1 && args[1] == "" {
		return errors.New("empty HOME argument: omit it to use $HOME, which is what a caller almost always means")
	}
	if len(args) > 1 {
		home = args[1]
	}
	expr, err := trees.Expr(root, home)
	if err != nil {
		return err
	}
	// A SILENT STATIC-ONLY RULE IS THE SHAPE THIS COMMAND REFUSES ELSEWHERE.
	// The derived half is empty on a machine with no workspace above the
	// checkout — a CI runner, which is expected — and also on one whose
	// workspace directories do not carry the shouting convention, which is not.
	// Both print the static half and pass; only the note tells them apart.
	names, err := trees.Derived(root, home)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "internaltrees: no workspace names derived — the static half applies alone "+
			"(expected on a runner with no workspace above the checkout; on a development machine it means "+
			"the workspace's directories do not match the convention, and the rule is weaker than it reads)")
	}
	// THE WRITE IS CHECKED because a short write is the one way this command
	// could still hand a gate half a rule and exit zero.
	if _, err := fmt.Fprintln(out, expr); err != nil {
		return err
	}
	return nil
}
