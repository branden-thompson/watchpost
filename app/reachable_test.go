package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryLivePipelinesMethodIsReachedFromProductionCode.
//
// **This is a gate against one defect, and that defect has happened four
// times.** Seeding the watched places' zones was measured, argued, ruled on,
// built and tested - and nothing called it. It passed every test it had and
// did nothing. The same shape then recurred three more times in the round that
// was fixing it: a cap whose call site nothing exercised, counters nothing
// read, and a panic guard in the wrong goroutine.
//
// A test of a thing is not a test of its wiring, and unit tests cannot tell
// the difference: they call the method themselves, so the method works and the
// program does not. `run` builds the whole station from the network and a
// terminal, so no unit test reaches it either.
//
// So this reads the source. Every method on `*livePipelines` must be called
// somewhere outside a test. It is a weak statement - being called is not being
// called correctly - but it is exactly the statement that was missing, and it
// fails the moment a call site is deleted.
func TestEveryLivePipelinesMethodIsReachedFromProductionCode(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]token.Position{}
	referencedInProduction := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		isTest := strings.HasSuffix(name, "_test.go")
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			if !isTest && receiverIsLivePipelines(fn.Recv.List[0].Type) {
				declared[fn.Name.Name] = fset.Position(fn.Pos())
			}
		}
		if isTest {
			continue
		}
		// Every mention of a method in this file, by name - **a reference, not
		// only a call**. A method handed over as a value is wired just as
		// surely as one invoked (`newDumper(..., lp.ttyStats)`), and counting
		// calls alone reported eleven live methods as dead. A gate that cries
		// wolf is a gate people learn to skip.
		//
		// The receiver is not resolved, so a same-named method on another type
		// would let one through. That makes this weaker than it looks, and
		// never a false alarm - the right way round for a gate.
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				referencedInProduction[sel.Sel.Name] = true
			}
			return true
		})
	}
	if len(declared) == 0 {
		t.Fatal("no methods on *livePipelines were found; this gate is not looking at anything")
	}
	for name, at := range declared {
		if !referencedInProduction[name] {
			t.Errorf("%s: (*livePipelines).%s is called by nothing but tests - "+
				"a ruling that does not run is a ruling that was not kept", at, name)
		}
	}
}

// receiverIsLivePipelines is true for `(lp *livePipelines)` and
// `(lp livePipelines)`.
func receiverIsLivePipelines(expr ast.Expr) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "livePipelines"
}
