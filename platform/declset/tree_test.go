package declset

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestTreeWalksEveryPackageBelowItsRoot is 0.19.6 TG-2: a structural rule
// that walks only its own directory goes blind to code a split moves below it.
// Tree keeps the rule's scope whatever the split does: the root and every
// package under it, test files and testdata left out, and a nested module
// left to its own rules.
func TestTreeWalksEveryPackageBelowItsRoot(t *testing.T) {
	dir := t.TempDir()
	for rel, src := range map[string]string{
		"a.go":              "package a\n",
		"a_test.go":         "package a\n",
		"sub/b.go":          "package sub\n",
		"sub/b_test.go":     "package sub\n",
		"sub/testdata/c.go": "package c\n",
		"other/go.mod":      "module other\n",
		"other/d.go":        "package d\n",
	} {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fset, files, err := Tree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(t, dir, fset, files); got != "a.go sub/b.go" {
		t.Errorf("Tree parsed %q; want a.go and sub/b.go", got)
	}
	fs2, flat, err := Files(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(t, dir, fs2, flat); got != "a.go" {
		t.Errorf("Files parsed %q; want a.go alone, the directory and nothing below", got)
	}
	tests, err := TestPaths(dir)
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, p := range tests {
		r, _ := filepath.Rel(dir, p)
		rels = append(rels, r)
	}
	if len(rels) != 2 || rels[0] != "a_test.go" || rels[1] != filepath.Join("sub", "b_test.go") {
		t.Errorf("TestPaths found %v; want a_test.go and sub/b_test.go", rels)
	}
}

func names(t *testing.T, dir string, fset *token.FileSet, files []*ast.File) string {
	t.Helper()
	var out []string
	for _, f := range files {
		r, err := filepath.Rel(dir, fset.Position(f.Package).Filename)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, filepath.ToSlash(r))
	}
	sort.Strings(out)
	s := ""
	for i, o := range out {
		if i > 0 {
			s += " "
		}
		s += o
	}
	return s
}
