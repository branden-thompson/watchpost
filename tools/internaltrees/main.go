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
	"fmt"
	"os"

	"github.com/branden-thompson/watchpost/tools/internaltrees/trees"
)

func main() {
	root, home := ".", os.Getenv("HOME")
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if len(os.Args) > 2 {
		home = os.Args[2]
	}
	expr, err := trees.Expr(root, home)
	if err != nil {
		fmt.Fprintln(os.Stderr, "internaltrees:", err)
		os.Exit(1)
	}
	fmt.Println(expr)
}
