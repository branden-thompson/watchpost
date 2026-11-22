package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

// testMainOf reads a package directory's test files and reports whether one
// declares TestMain, whether that TestMain calls <helper>.Main, and whether it
// reads WATCHPOST_TEST_GOOS.
func testMainOf(t *testing.T, dir, helper string) (has, calls, goos bool) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "TestMain" || fn.Recv != nil || fn.Body == nil {
				continue
			}
			has = true
			body := string(src[fset.Position(fn.Body.Pos()).Offset:fset.Position(fn.Body.End()).Offset])
			calls = calls || strings.Contains(body, helper+".Main(")
			goos = goos || strings.Contains(body, "WATCHPOST_TEST_GOOS")
		}
	}
	return has, calls, goos
}

// readsGOOSSeam reports whether a package's production code calls the
// platform seam.
func readsGOOSSeam(t *testing.T, dir string) bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "runtimeGOOS()") {
			return true
		}
	}
	return false
}

// testMainProblems judges every package with tests under one tree: each has a
// TestMain that calls the tree's helper, and one whose code reads the platform
// seam applies WATCHPOST_TEST_GOOS too.
func testMainProblems(t *testing.T, root, tree, helper string) (pkgs, probs []string) {
	t.Helper()
	paths, err := declset.TestPaths(filepath.Join(root, tree))
	if err != nil {
		t.Fatal(err)
	}
	dirs := map[string]bool{}
	for _, p := range paths {
		dirs[filepath.Dir(p)] = true
	}
	for dir := range dirs {
		rel, _ := filepath.Rel(root, dir)
		rel = filepath.ToSlash(rel)
		pkgs = append(pkgs, rel)
		has, calls, goos := testMainOf(t, dir, helper)
		switch {
		case !has:
			probs = append(probs, rel+" has tests and no TestMain: call "+helper+".Main from one")
		case !calls:
			probs = append(probs, rel+"'s TestMain does not call "+helper+".Main")
		}
		if readsGOOSSeam(t, dir) && !goos {
			probs = append(probs, rel+" reads the platform seam and its TestMain does not apply WATCHPOST_TEST_GOOS (0.19.6 D-24)")
		}
	}
	sort.Strings(pkgs)
	sort.Strings(probs)
	return pkgs, probs
}

// TestEveryPackageUnderTheTwoTreesCarriesItsParentsTestMain is 0.19.6 TG-3,
// RX-6 and D-24. Each package's TestMain holds an invariant its tests rely on
// (modes/tty's zone pinned to UTC, app's config and state in directories of the
// run's own), and a package a split creates would lack it silently. So every
// package with tests under modes/tty calls ttytest.Main, every one under app
// calls apptest.Main, and one whose code reads the platform seam applies
// WATCHPOST_TEST_GOOS. A planted package is reported, and the packages that
// exist at 0.19.6 B0 are the floor, raised in each batch that adds one.
func TestEveryPackageUnderTheTwoTreesCarriesItsParentsTestMain(t *testing.T) {
	planted := t.TempDir()
	if err := os.MkdirAll(filepath.Join(planted, "app", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planted, "app", "x", "x_test.go"), []byte("package x\n\nimport \"testing\"\n\nfunc TestMain(m *testing.M) { m.Run() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, probs := testMainProblems(t, planted, "app", "apptest"); len(probs) != 1 {
		t.Fatalf("a planted package without the helper gave %v; want one problem", probs)
	}
	root := filepath.Join("..", "..")
	var all []string
	for _, tree := range []struct{ dir, helper string }{{"modes/tty", "ttytest"}, {"app", "apptest"}} {
		pkgs, probs := testMainProblems(t, root, tree.dir, tree.helper)
		all = append(all, pkgs...)
		for _, p := range probs {
			t.Error(p)
		}
	}
	// The packages with tests at 0.19.6 B0, and the two B2 adds.
	for _, must := range []string{"app", "modes/tty", "modes/tty/internal/frame", "modes/tty/internal/ttytest"} {
		if !slices.Contains(all, must) {
			t.Errorf("the walk found no tests in %s; it is broken, or the package lost its tests", must)
		}
	}
}
