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
	if err := invariant.Check(dir != "", "declset: a package directory is required"); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			return nil, err
		}
		out = append(out, declsOf(f)...)
	}
	sort.Strings(out)
	return out, nil
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
