package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

var updateUnion = flag.Bool("update-union", false, "re-capture testdata/union-decls.txt from the tree")

// isSplitPackage reports whether a name is one of the packages of the two
// trees 0.19.6 splits: a qualifier naming one is what a move adds or drops
// while the code is the same.
func isSplitPackage(name string) bool {
	switch name {
	case "tty", "console", "mapview", "frame", "controls", "ttytest", "deck", "mapdata", "run", "apptest":
		return true
	}
	return false
}

// testSupport is the two trees' test-support packages, which test files alone
// may import (scripts/lint-imports.sh): test code by role wherever its files
// live, so the union leaves them out as it leaves out _test.go files. The
// guard's engine and helpers leaving modes/tty's _test.go files for ttytest's
// are no new code (0.19.6 D-63).
var testSupport = []string{"app/internal/apptest", "modes/tty/internal/ttytest"}

// inTestSupport reports whether a source file, by its path under root, is in
// one of the test-support packages.
func inTestSupport(root, p string) bool {
	rel, err := filepath.Rel(root, filepath.Dir(p))
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, d := range testSupport { // bounded by the list (P10-02)
		if rel == d || strings.HasPrefix(rel, d+"/") {
			return true
		}
	}
	return false
}

// unionDecls lists every top-level declaration of the non-test code under app
// and modes/tty, as "kind name hash": the hash is of the declaration's tokens,
// so its comments and layout do not count, with a qualifier naming a split
// package dropped, so a move that changes only where a declaration lives, and
// how its callers qualify it, leaves its line the same. The test-support
// packages are not counted (testSupport).
func unionDecls(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	for _, tree := range []string{"app", "modes/tty"} {
		paths, err := declset.SourcePaths(filepath.Join(root, tree))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range paths {
			if inTestSupport(root, p) {
				continue
			}
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, p, src, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range f.Decls {
				out = append(out, declLines(fset, src, d)...)
			}
		}
	}
	sort.Strings(out)
	return out
}

// declLines is one declaration's lines: one for a function, one per spec for
// a general declaration.
func declLines(fset *token.FileSet, src []byte, d ast.Decl) []string {
	hash := func(n ast.Node) string {
		from, to := fset.Position(n.Pos()).Offset, fset.Position(n.End()).Offset
		sum := sha256.Sum256([]byte(tokensOf(src[from:to])))
		return fmt.Sprintf("%x", sum[:6])
	}
	switch d := d.(type) {
	case *ast.FuncDecl:
		name := d.Name.Name
		if d.Recv != nil && len(d.Recv.List) > 0 {
			name = "(" + recvTypeName(d.Recv.List[0].Type) + ")." + name
		}
		return []string{"func " + name + " " + hash(d)}
	case *ast.GenDecl:
		var out []string
		for _, s := range d.Specs {
			switch s := s.(type) {
			case *ast.TypeSpec:
				out = append(out, "type "+s.Name.Name+" "+hash(s))
			case *ast.ValueSpec:
				for _, n := range s.Names {
					out = append(out, d.Tok.String()+" "+n.Name+" "+hash(s))
				}
			}
		}
		return out
	}
	return nil
}

// tokensOf is a declaration's tokens, one space apart, comments and layout
// dropped, and a split package's qualifier dropped with its dot.
func tokensOf(src []byte) string {
	var sc scanner.Scanner
	fs := token.NewFileSet()
	sc.Init(fs.AddFile("", -1, len(src)), src, nil, 0)
	var toks []string
	for {
		_, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.PERIOD && len(toks) > 0 && isSplitPackage(toks[len(toks)-1]) {
			toks = toks[:len(toks)-1]
			continue
		}
		if lit == "" || tok == token.SEMICOLON {
			lit = tok.String()
		}
		toks = append(toks, lit)
	}
	return strings.Join(toks, " ")
}

// recvTypeName is a receiver's type name, through a pointer and type
// parameters.
func recvTypeName(e ast.Expr) string {
	if s, ok := e.(*ast.StarExpr); ok {
		e = s.X
	}
	switch x := e.(type) {
	case *ast.IndexExpr:
		e = x.X
	case *ast.IndexListExpr:
		e = x.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return "?"
}

// TestTheTwoTreesDeclarationsAreConserved is 0.19.6 RX-10, TG-7 and B-10: a
// move commit may change where a declaration lives and how it is qualified,
// and nothing else. Every top-level declaration under app and modes/tty, with
// a hash of its source, is held against testdata/union-decls.txt, so a move
// that edited a body, dropped a declaration or added one fails here. A prep
// commit that renames, exports or reshapes re-captures the golden with
// -update-union, and its diff is a row of the batch's changed-line table.
func TestTheTwoTreesDeclarationsAreConserved(t *testing.T) {
	// THE CONTROLS: a split package's qualifier, a comment and the layout do not
	// change a declaration's tokens; another qualifier and a statement do.
	same := tokensOf([]byte("func f() tty.Config {\n\treturn tty.Config{}\n}"))
	if tokensOf([]byte("func f() Config {\n\t// a comment\n\n\treturn Config{}\n}")) != same {
		t.Fatal("control: a dropped split-package qualifier, a comment or a blank line changed the tokens")
	}
	if tokensOf([]byte("func f() render.Config {\n\treturn render.Config{}\n}")) == tokensOf([]byte("func f() Config {\n\treturn Config{}\n}")) ||
		tokensOf([]byte("func f() Config {\n\t_ = 0\n\treturn Config{}\n}")) == tokensOf([]byte("func f() Config {\n\treturn Config{}\n}")) {
		t.Fatal("control: another package's qualifier or an added statement left the tokens the same")
	}
	// THE LEFT-OUT PACKAGES ARE lint-imports' TEST SUPPORT UNDER THE TWO TREES,
	// both ways: a package the binary links is never left out, and a
	// test-support package is never counted as code.
	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "lint-imports.sh"))
	if err != nil {
		t.Fatal(err)
	}
	var named []string
	for _, m := range regexp.MustCompile(`check_none "\$r" (\S+) "test support`).FindAllStringSubmatch(string(script), -1) {
		if strings.HasPrefix(m[1], "app/") || strings.HasPrefix(m[1], "modes/tty/") { // the two trees' own
			named = append(named, m[1])
		}
	}
	sort.Strings(named)
	if strings.Join(named, " ") != strings.Join(testSupport, " ") {
		t.Fatalf("the union leaves out %v and scripts/lint-imports.sh holds %v to test files: the two lists are one", testSupport, named)
	}
	if !inTestSupport("r", filepath.Join("r", "modes", "tty", "internal", "ttytest", "main.go")) || inTestSupport("r", filepath.Join("r", "modes", "tty", "internal", "frame", "row.go")) {
		t.Fatal("control: a test-support file was counted, or a linked package's file left out")
	}
	got := unionDecls(t, filepath.Join("..", ".."))
	if len(got) < 3000 { // the two trees declare 3,319 at 0.19.6 B0
		t.Fatalf("the walk found %d declarations; it is broken", len(got))
	}
	golden := filepath.Join("testdata", "union-decls.txt")
	if *updateUnion {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(strings.Join(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("captured %d declarations", len(got))
		return
	}
	raw, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("no golden yet: run go test ./cmd/watchpost -run TheTwoTreesDeclarations -args -update-union: %v", err)
	}
	want := strings.Split(strings.TrimSpace(string(raw)), "\n")
	added, removed := declset.Diff(want, got)
	if len(added)+len(removed) > 0 {
		t.Errorf("the two trees' declarations changed: %d new or edited, %d gone or edited.\n  new: %v\n  gone: %v\n"+
			"A move commit changes neither; a prep commit re-captures with -update-union and lists the diff in the changed-line table",
			len(added), len(removed), head(added), head(removed))
	}
}

// head is the first few lines of a list, for a failure message.
func head(l []string) []string {
	if len(l) > 12 {
		return append(l[:12:12], fmt.Sprintf("… %d more", len(l)-12))
	}
	return l
}
