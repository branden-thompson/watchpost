package tty

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

// TestTheSettingsTableIsNeverWritten is 0.19.6 D-88: setupRows is one table
// the whole package reads in place, so nothing may assign through it, step it,
// or take its address. Every Go file of the modes/tty tree, tests included, is
// read for the three.
func TestTheSettingsTableIsNeverWritten(t *testing.T) {
	// THE CONTROLS: each write is seen, and the reads are not.
	planted := `package tty
func f() {
	setupRows[0].kind = rowInput
	setupRows[1].picker = true
	setupRows = setupTable()
	setupRows[2].id++
	p := &setupRows[3]
	q := &setupRows
	_, _ = p, q
	_ = setupRows[4].group
	_ = setupRowOf(5)
	for _, r := range setupRows {
		_ = r
	}
}`
	if got := writesOfSetupRows(t, "planted.go", planted); len(got) != 6 {
		t.Fatalf("control: the planted file's six writes read as %d: %v", len(got), got)
	}
	files, err := declset.SourcePaths(".") // the whole modes/tty tree (0.19.6 TG-2)
	if err != nil {
		t.Fatal(err)
	}
	tests, err := declset.TestPaths(".")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, tests...)
	if len(files) < 100 {
		t.Fatalf("the tree's files are %d; the walk is broken", len(files))
	}
	for _, f := range files { // bounded by the package's files (P10-02)
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range writesOfSetupRows(t, f, string(src)) { // bounded by the file (P10-02)
			t.Errorf("%s writes the Settings table, which every caller shares", w)
		}
	}
}

// writesOfSetupRows is each place in one file that assigns through setupRows,
// steps it, or takes its address, by position.
func writesOfSetupRows(t *testing.T, name, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	note := func(e ast.Expr) {
		if rootIdent(e) == "setupRows" {
			out = append(out, fset.Position(e.Pos()).String())
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, l := range n.Lhs { // bounded by the statement (P10-02)
				note(l)
			}
		case *ast.IncDecStmt:
			note(n.X)
		case *ast.UnaryExpr:
			if n.Op == token.AND {
				note(n.X)
			}
		}
		return true
	})
	return out
}

// rootIdent is the variable an expression reaches through indexes, fields,
// parentheses and dereferences, or "" when it reaches none.
func rootIdent(e ast.Expr) string {
	for i := 0; i < 32; i++ { // bounded by any expression's depth here (P10-02)
		switch x := e.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.IndexExpr:
			e = x.X
		case *ast.SelectorExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		default:
			return ""
		}
	}
	return ""
}
