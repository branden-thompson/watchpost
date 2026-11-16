// Package declset lists a Go package directory's top-level declarations so
// a test can pin them against a golden: a pure file move (quality pass Q2)
// must not add, drop or rename one, and `git diff --stat` cannot certify a
// 1→13 split (rename detection needs whole-file similarity — red-team
// R2-22). Test-only in practice: every package the pass splits calls Pin
// from one small test; nothing in the product imports it.
package declset

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Set lists the non-test top-level declarations under dir as sorted
// "kind name" lines — methods as "func (Recv).Name", vars and consts one
// line per name.
func Set(dir string) ([]string, error) {
	_, files, err := Files(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range files { // bounded by the package's files (P10-02)
		out = append(out, declsOf(f)...)
	}
	sort.Strings(out)
	return out, nil
}

// Files parses every non-test Go file in dir, with the FileSet that positions
// them. It is what any check that WALKS a package for a structural rule should
// build on — "startSynth has exactly one caller", "the program's model is
// always the router" — so those checks derive their subject list rather than
// enumerating it (INST-1).
//
// NOT go/parser.ParseDir, WHICH IS DEPRECATED. Its replacement,
// x/tools/go/packages, is a dependency and a type-check for a job that is a
// directory listing; this is that listing, once, where the package that
// already did it lives.
//
// TEST FILES ARE EXCLUDED, and every caller so far wants that: a rule about
// production code must not be satisfied by a test that happens to mention the
// right identifier.
func Files(dir string) (*token.FileSet, []*ast.File, error) {
	if err := invariant.Check(dir != "", "declset: a package directory is required"); err != nil {
		return nil, nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries { // bounded by the directory (P10-02)
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, f)
	}
	return fset, files, nil
}

func declsOf(f *ast.File) []string {
	if err := invariant.Check(f != nil, "declset: parsed file required"); err != nil {
		return nil
	}
	var out []string
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			name := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				name = "(" + recvType(d.Recv.List[0].Type) + ")." + name
			}
			out = append(out, "func "+name)
		case *ast.GenDecl:
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					out = append(out, "type "+s.Name.Name)
				case *ast.ValueSpec:
					for _, n := range s.Names {
						out = append(out, d.Tok.String()+" "+n.Name)
					}
				}
			}
		}
	}
	return out
}

// recvType names a receiver type: stars kept, type parameters dropped.
// Iterative (P10-01): a receiver nests at most a star and an index.
func recvType(e ast.Expr) string {
	prefix := ""
	for i := 0; i < 4; i++ { // bounded: *T, T[P], *T[P] are the legal shapes
		switch t := e.(type) {
		case *ast.StarExpr:
			prefix, e = prefix+"*", t.X
			continue
		case *ast.IndexExpr:
			e = t.X
			continue
		case *ast.Ident:
			return prefix + t.Name
		}
		break
	}
	return prefix + fmt.Sprintf("%T", e)
}

// Diff reports the lines in got that are not in want and vice versa.
func Diff(want, got []string) (added, removed []string) {
	w := map[string]bool{}
	for _, s := range want {
		w[s] = true
	}
	g := map[string]bool{}
	for _, s := range got {
		g[s] = true
		if !w[s] {
			added = append(added, s)
		}
	}
	for _, s := range want {
		if !g[s] {
			removed = append(removed, s)
		}
	}
	return added, removed
}

// Golden is the conventional golden path for a package directory.
func Golden(dir string) string { return filepath.Join(dir, "testdata", "declset.txt") }

// Write captures the current set into the golden file.
func Write(dir string) (int, error) {
	set, err := Set(dir)
	if err != nil {
		return 0, err
	}
	if err := invariant.Check(len(set) > 0, "declset: refusing to write an empty golden for "+dir); err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(Golden(dir)), 0o755); err != nil {
		return 0, err
	}
	return len(set), os.WriteFile(Golden(dir), []byte(strings.Join(set, "\n")+"\n"), 0o644)
}

// Compare reads the golden and returns what changed; ok is false when the
// golden is missing (the caller says how to create it).
func Compare(dir string) (added, removed []string, ok bool, err error) {
	raw, err := os.ReadFile(Golden(dir))
	if err != nil {
		return nil, nil, false, nil
	}
	got, err := Set(dir)
	if err != nil {
		return nil, nil, true, err
	}
	want := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if err := invariant.Check(len(want) > 0 && want[0] != "", "declset: golden "+Golden(dir)+" is empty"); err != nil {
		return nil, nil, true, err
	}
	added, removed = Diff(want, got)
	return added, removed, true, nil
}

// Tree parses every non-test Go file under root, root's own package and every
// package below it (0.19.6 TG-2). A structural rule that walks one directory
// passes emptier the moment a split moves code below it, and nothing reports
// the loss; rooted at the package the rule was written for, Tree keeps its
// scope whatever the split does. A testdata directory and a nested module are
// left out: neither is this package's code.
func Tree(root string) (*token.FileSet, []*ast.File, error) {
	paths, err := walk(root, func(name string) bool {
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	})
	if err != nil {
		return nil, nil, err
	}
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(paths))
	for _, p := range paths { // bounded by the tree's files (P10-02)
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, f)
	}
	return fset, files, nil
}

// TestPaths lists every test file under root, for a rule about the tests
// themselves; the same scope as Tree, sorted.
func TestPaths(root string) ([]string, error) {
	return walk(root, func(name string) bool { return strings.HasSuffix(name, "_test.go") })
}

// walk lists the files under root that keep accepts, sorted, skipping
// testdata, hidden directories and any directory with a go.mod of its own.
func walk(root string, keep func(string) bool) ([]string, error) {
	if err := invariant.Check(root != "", "declset: a root directory is required"); err != nil {
		return nil, err
	}
	var out []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipDir(root, p, d.Name())
		}
		if keep(d.Name()) {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

// skipDir is filepath.SkipDir for a directory that is not the tree's code.
func skipDir(root, p, name string) error {
	if p == root {
		return nil
	}
	if name == "testdata" || strings.HasPrefix(name, ".") {
		return filepath.SkipDir
	}
	if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
		return filepath.SkipDir
	}
	return nil
}

// SourcePaths lists every non-test Go file under root, the scope of Tree, for
// a rule that reads source as text rather than as syntax.
func SourcePaths(root string) ([]string, error) {
	return walk(root, func(name string) bool {
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	})
}
