package app

// The program's model is the ROUTER, and nothing else (P0, FR-1.1).
//
// DERIVED, NOT ENUMERATED (INST-1). The check WALKS the package's AST for
// every tea.NewProgram call rather than asserting against a remembered list
// of call sites, so a THIRD one added later cannot slip past a list nobody
// updated — which is exactly how the two sites here came to differ from the
// plan's own count in the first place.

import (
	"go/ast"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

func TestTheProgramsModelIsAlwaysTheRouter(t *testing.T) {
	// THROUGH declset.Files, NOT go/parser.ParseDir. ParseDir is deprecated
	// (Go 1.25) and three checks in this tree had copied the same shape; the
	// package that already walked a package's non-test files owns it now.
	fset, files, err := declset.Files(".")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "NewProgram" {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "tea" {
				return true
			}
			found++
			pos := fset.Position(call.Pos())
			if len(call.Args) == 0 {
				t.Errorf("%s: tea.NewProgram with no model", pos)
				return true
			}
			inner, ok := call.Args[0].(*ast.CallExpr)
			if !ok {
				t.Errorf("%s: the program's model must be tty.NewRouter(...), got a bare value — "+
					"a surface handed straight to the program is a seam nothing can swap", pos)
				return true
			}
			isel, ok := inner.Fun.(*ast.SelectorExpr)
			if !ok || isel.Sel.Name != "NewRouter" {
				t.Errorf("%s: the program's model must be tty.NewRouter(...), got %s", pos, exprName(inner.Fun))
			}
			return true
		})
	}
	// SILENCE IS A DISTINCT VERDICT (INST-2): finding no call at all means the
	// walk broke, not that the package is clean.
	if found == 0 {
		t.Fatal("found no tea.NewProgram call in this package — the check did not run, which is not the same as passing")
	}
	t.Logf("checked %d tea.NewProgram call(s); blind to any created outside package app", found)
}

func exprName(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprName(v.X) + "." + v.Sel.Name
	}
	return "?"
}
