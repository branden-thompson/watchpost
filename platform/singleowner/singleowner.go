// Package singleowner asserts that only named files may do a named thing.
//
// WHY A GATE AND NOT A TEST. A rule about WHICH CODE may do something is not
// observable in the program's behaviour, so no test of output can hold it.
// F-22 records what happens when one tries: mutant m50 claimed to guard the
// band's single-writer rule, and when it was re-anchored to send the message
// directly instead of through mastercontrol, it SURVIVED — both forms produce
// the identical observable message. Its earlier CAUGHT verdicts came from an
// unrelated early return. It was passing for a reason other than its stated
// rule, and only moving it revealed that.
//
// AST, NOT GREP. A grep matches the name in a comment, in a string, and in the
// gate itself; and it must be written twice, because a type or function is
// bare inside its own package and qualified outside it. The config gate found
// exactly that blindness in itself the afternoon it was written.
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
// files are policed. Both gates that led to this package hit the same failure
// first — the matcher stopped matching and the run went green, which is
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
// FOUND RATHER THAN COUNTED: the first of these gates used a hand-written
// "../.." and pointed one directory short of the root, so it matched nothing
// and its own liveness guard caught it. A gate that has to know its own depth
// is a gate that breaks when it moves.
func Root(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot locate the module root: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot locate the module root: no go.mod above the working directory")
		}
		dir = parent
	}
}
