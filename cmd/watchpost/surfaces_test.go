package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

// updateExported rewrites the roots' exported-name lists from the tree.
var updateExported = flag.Bool("update-exported", false, "rewrite 06_docs/graph/exported-*.txt from the tree")

// roots are the two packages 0.19.6 splits: each directory, its list, and a
// floor under the list against one that reads as empty or truncated, below
// the 154 and 11 names each lists at 0.19.6 B0.12, so a batch that moves
// names out does not trip it.
var roots = []struct {
	dir, list string
	floor     int
}{
	{"modes/tty", "06_docs/graph/exported-tty.txt", 100},
	{"app", "06_docs/graph/exported-app.txt", 5},
}

// exportedNames is a package's exported surface from declset's lines: its
// exported top-level names, and the exported methods of its exported types.
func exportedNames(t *testing.T, dir string) []string {
	t.Helper()
	decls, err := declset.Set(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range decls { // bounded by the declarations (P10-02)
		kind, name, _ := strings.Cut(d, " ")
		if rest, ok := strings.CutPrefix(name, "("); ok {
			recv, m, _ := strings.Cut(rest, ").")
			if exported(strings.TrimPrefix(recv, "*")) && exported(m) {
				out = append(out, kind+" "+strings.TrimPrefix(recv, "*")+"."+m)
			}
			continue
		}
		if exported(name) {
			out = append(out, d)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// exported says whether a name starts with an upper-case letter.
func exported(s string) bool { return s != "" && s[0] >= 'A' && s[0] <= 'Z' }

// TestTheRootsExportOnlyWhatIsListed is 0.19.6 B0.12 (M5, JN P2-4): the
// exported surface of modes/tty and of app grows only by what a batch lists,
// the new packages' entry points. A name the tree exports and the list lacks
// fails; a listed name that is gone, moved out with its code, is reported.
// -update-exported rewrites the lists, a deliberate act in the commit that
// adds an entry point.
func TestTheRootsExportOnlyWhatIsListed(t *testing.T) {
	for _, r := range roots { // bounded by the two roots (P10-02)
		dir, list := r.dir, r.list
		got := exportedNames(t, filepath.Join("..", "..", dir))
		path := filepath.Join("..", "..", list)
		if *updateExported {
			head := "# the exported surface of " + dir + " (0.19.6 B0.12, M5): exported top-level names and exported types' methods;\n# a batch adds its new packages' entry points here; TestTheRootsExportOnlyWhatIsListed\n"
			if err := os.WriteFile(path, []byte(head+strings.Join(got, "\n")+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v: record it with -update-exported", err)
		}
		var want []string
		for _, l := range strings.Split(string(raw), "\n") { // bounded by the list (P10-02)
			if l != "" && !strings.HasPrefix(l, "#") {
				want = append(want, l)
			}
		}
		if len(want) < r.floor {
			t.Fatalf("COULD NOT RUN: %s lists %d names, under its floor of %d: the list is not the root's", list, len(want), r.floor)
		}
		for _, n := range got { // bounded by the surface (P10-02)
			if !slices.Contains(want, n) {
				t.Errorf("%s exports %q and %s does not list it: a root's surface grows only by a batch's listed entry points (M5)", dir, n, list)
			}
		}
		for _, n := range want { // bounded by the list (P10-02)
			if !slices.Contains(got, n) {
				t.Logf("%s no longer exports %q, listed in %s", dir, n, list)
			}
		}
	}
}

// testOnlyExports are 0.19.6 D-26's three (design MS C-6): exported names the map's
// files declare whose only users outside modes/tty are tests, pre-existing
// cross-package contract checks a 0.19.7 follow-up decides. The map's names
// are the ones the split moves into a package of their own, where M5 counts
// each export the split adds; a root name's test-only uses stay what they
// were.
var testOnlyExports = []string{"ChipKnown", "FireAll", "SameOverlay"}

// TestOnlyD26sThreeExportsAreUsedOutsideByTestsAlone is 0.19.6 D-26: of the
// exported package-level names the map's non-test files declare, those that
// files outside modes/tty name only in tests are exactly the three, so none
// is added silently. A name is used where a file outside modes/tty, importing
// it, selects it from the import. Seen failing on a planted use.
func TestOnlyD26sThreeExportsAreUsedOutsideByTestsAlone(t *testing.T) {
	inTests, inCode := outsideUses(t, "../..", module+"/modes/tty")
	var got []string
	mapNames := mapDeclared(t, "../../modes/tty", "map_")
	for _, n := range testOnly(inTests, inCode) { // bounded by the uses (P10-02)
		if mapNames[n] {
			got = append(got, n)
		}
	}
	if !slices.Equal(got, testOnlyExports) {
		t.Errorf("the exports of modes/tty used outside it by tests alone are %v; D-26 holds exactly %v", got, testOnlyExports)
	}
	if planted := testOnly(map[string]bool{"Planted": true, "ChipKnown": true}, map[string]bool{"ChipKnown": true}); !slices.Equal(planted, []string{"Planted"}) {
		t.Errorf("COULD NOT RUN: a planted test-only use read as %v", planted)
	}
}

// mapDeclared is the package-level names declared in a directory's non-test
// files whose names start with prefix.
func mapDeclared(t *testing.T, dir, prefix string) map[string]bool {
	t.Helper()
	fset, files, err := declset.Files(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, f := range files { // bounded by the package (P10-02)
		if !strings.HasPrefix(filepath.Base(fset.Position(f.Pos()).Filename), prefix) {
			continue
		}
		for _, d := range f.Decls { // bounded by the file (P10-02)
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					out[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, sp := range d.Specs { // bounded by the declaration (P10-02)
					switch sp := sp.(type) {
					case *ast.TypeSpec:
						out[sp.Name.Name] = true
					case *ast.ValueSpec:
						for _, n := range sp.Names { // bounded by the spec (P10-02)
							out[n.Name] = true
						}
					}
				}
			}
		}
	}
	if len(out) < 50 {
		t.Fatalf("COULD NOT RUN: %s's %s* files declare %d names; the map declares hundreds", dir, prefix, len(out))
	}
	return out
}

// module is this repository's import path.
const module = "github.com/branden-thompson/watchpost"

// testOnly is the names in tests and not in code, in order.
func testOnly(inTests, inCode map[string]bool) []string {
	var out []string
	for n := range inTests { // bounded by the uses (P10-02)
		if !inCode[n] && exported(n) {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out
}

// outsideUses is every name selected from pkg by files outside its
// directory, split into those named in a test file and those named in code.
func outsideUses(t *testing.T, root, pkg string) (inTests, inCode map[string]bool) {
	t.Helper()
	inTests, inCode = map[string]bool{}, map[string]bool{}
	own := filepath.Join(root, strings.TrimPrefix(pkg, module+"/"))
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && (d.Name() == "third_party" || d.Name() == ".git" || d.Name() == ".claude" || d.Name() == "dist" || p == own):
			return filepath.SkipDir
		case d.IsDir() || !strings.HasSuffix(p, ".go"):
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		into := inCode
		if strings.HasSuffix(p, "_test.go") {
			into = inTests
		}
		for name := range selectedFrom(f, pkg) { // bounded by the file's uses (P10-02)
			into[name] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inCode) < 50 {
		t.Fatalf("COULD NOT RUN: code outside %s names %d of its names; app alone names hundreds", pkg, len(inCode))
	}
	return inTests, inCode
}

// selectedFrom is the names a file selects from its import of pkg.
func selectedFrom(f *ast.File, pkg string) map[string]bool {
	local := ""
	for _, imp := range f.Imports { // bounded by the file's imports (P10-02)
		if p, err := strconv.Unquote(imp.Path.Value); err == nil && p == pkg {
			local = filepath.Base(pkg)
			if imp.Name != nil {
				local = imp.Name.Name
			}
		}
	}
	out := map[string]bool{}
	if local == "" {
		return out
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == local {
				out[sel.Sel.Name] = true
			}
		}
		return true
	})
	return out
}
