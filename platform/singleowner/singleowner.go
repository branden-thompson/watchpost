// Package singleowner asserts that only named files may do a named thing.
//
// WHY A GATE AND NOT A TEST. A rule about WHICH CODE may do something is not
// observable in the program's behaviour, so no test of output can hold it.
// A mutant that sends the band's message directly instead of through
// mastercontrol produces the identical observable message, so a behavioural
// test can only appear to guard the single-writer rule — passing, when it
// passes, for a reason other than its stated rule (F-22, mutant m50).
//
// AST, NOT GREP. A grep matches the name in a comment, in a string, and in the
// gate itself; and it must be written twice, because a type or function is
// bare inside its own package and qualified outside it.
package singleowner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Check walks the module and fails for every non-test file that contains a node
// the matcher accepts and is not listed in owners.
//
// owners maps a repo-relative path to the REASON that file may do the thing. A
// reason is the point: an owner without one is a bypass nobody argued for.
//
// THE CORPUS IS COUNTED OVER EVERY FILE, TESTS INCLUDED, and only non-test
// files are policed. A matcher that stops matching runs green, which is
// indistinguishable from compliance. A gate that cannot demonstrate it is
// looking at anything must not be allowed to pass.
func Check(t *testing.T, what string, owners map[string]string, match func(ast.Node) bool) {
	t.Helper()
	CheckIn(t, Root(t), what, owners, match)
}

// CheckIn is Check against an explicit root, so the package can test itself
// against a fixture instead of against the repository it polices.
func CheckIn(t *testing.T, root, what string, owners map[string]string, match func(ast.Node) bool) {
	t.Helper()
	var sites []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && info.Name() == ".git" {
			// Not source, and git rewrites it during a walk: a lock file removed
			// between the listing and the lstat would fail the gate.
			return filepath.SkipDir
		}
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.Contains(filepath.ToSlash(path), "third_party/") {
			return nil
		}
		f, perr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if perr != nil {
			return nil // not ours to police
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if n != nil && match(n) {
				sites = append(sites, filepath.ToSlash(path))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("%s: walking %s: %v", what, root, err)
	}
	if len(sites) == 0 {
		t.Fatalf("%s: the matcher found nothing anywhere, tests included — it is broken, "+
			"so a green result here would prove nothing", what)
	}
	prefix := filepath.ToSlash(root) + "/"
	for _, s := range sites {
		if strings.HasSuffix(s, "_test.go") {
			continue // the corpus proves the matcher lives; tests are not the rule's subject
		}
		rel := strings.TrimPrefix(s, prefix)
		if _, ok := owners[rel]; !ok {
			t.Errorf("%s: %s is not an owner. Go through the owner, or add this file to the "+
				"owner set with a written reason", what, rel)
		}
	}
}

// Root is the module root, found by walking up for go.mod.
//
// FOUND RATHER THAN COUNTED: a hand-written "../.." points one directory
// short of the root as soon as the gate sits at another depth, and matches
// nothing. A gate that has to know its own depth is a gate that breaks when it
// moves.
func Root(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot locate the module root: %v", err)
	}
	// THE WALK IS BOUNDED BY THE PATH IT WALKS (P10-02). A bare `for {}` terminates
	// in FACT — `filepath.Dir` reaches a fixed point at the root and the guard below
	// catches it — but not in SHAPE, and this is the one loop in the package a
	// mistake in `filepath.Dir` would hang rather than fail. One separator is one
	// possible step up, so the count of them is the ceiling.
	for range strings.Count(dir, string(filepath.Separator)) + 2 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot locate the module root: no go.mod above the working directory")
		}
		dir = parent
	}
	// THE CEILING IS REACHED ONLY IF THE WALK NEVER TERMINATED, which the guard
	// above should have caught. Falling out of a bounded loop is the failure the
	// bound exists to make visible rather than to hide as a hang.
	t.Fatal("cannot locate the module root: the walk did not reach the filesystem root")
	return ""
}
