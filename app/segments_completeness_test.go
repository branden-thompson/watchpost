package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/branden-thompson/watchpost/platform/report"
)

// EVERY REPORT KIND THE OPERATOR CAN CHOOSE REACHES THE COMPOSER (F-111).
//
// THE GAP THIS CLOSES IS SILENT AND OPERATOR-FACING. `report.Kind` is a CLOSED
// SET with a `numKinds` sentinel and one registry, and 0.16.0 put that set in
// front of the operator: the Line-Up Request window lists the kinds and they
// choose (FR-3.4). `radioDeck.segments` answers each one with its OWN typed hook
// — `d.fire`, `d.seismic`, `d.marine`, and the NWS products path — so the
// branches cannot be table-driven without erasing the types that make them
// readable, and they were right to be written out.
//
// WHAT WAS MISSING IS THE GUARD, NOT A REFACTOR. Add a fifth kind and it appears
// in the registry, in the modal, and in the running order's labels — and
// composes to NOTHING, because nothing here asks it to. The operator requests a
// report, the card is built, and the thing they asked for is simply absent from
// what goes out. No error, no fault, no gap in the log.
//
// IT COUNTS RATHER THAN NAMES. A test listing the four kinds by hand would be a
// SECOND closed set, drifting from the first exactly when the first changes —
// which is the defect, moved. `report.All()` is the registry's own enumeration,
// and the count of distinct `want.Has(...)` conditions in `segments` is what the
// composer actually answers. P-9: read the paragraph before adding to a closed
// set; this is the paragraph, made executable.
//
// FOUND BY RED TEAM ROUND 3 (code-quality axis) as "four hard-coded branches".
// The branches are not the defect and are not changed. The absence of anything
// that notices a fifth is.
func TestEveryReportKindReachesTheComposer(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "radio.go", nil, 0)
	if err != nil {
		t.Fatalf("radio.go could not be parsed: %v", err)
	}

	var fn *ast.FuncDecl
	ast.Inspect(f, func(n ast.Node) bool {
		if d, ok := n.(*ast.FuncDecl); ok && d.Name.Name == "segments" {
			fn = d
		}
		return fn == nil
	})
	if fn == nil {
		t.Fatal("radioDeck.segments not found — this guard has lost its subject and must be re-pointed, not deleted")
	}

	// EVERY `report.X` NAMED INSIDE A `want.Has(...)` CALL, whatever else the
	// condition does. `Has` is the one way a kind is asked for, so the set of
	// kinds this function answers is the set of selectors it passes to it.
	asked := map[string]bool{}
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Has" || len(call.Args) != 1 {
			return true
		}
		if arg, ok := call.Args[0].(*ast.SelectorExpr); ok {
			if pkg, ok := arg.X.(*ast.Ident); ok && pkg.Name == "report" {
				asked[arg.Sel.Name] = true
			}
		}
		return true
	})

	if got, want := len(asked), len(report.All()); got != want {
		t.Errorf("the registry holds %d report kinds and `segments` answers %d (%v).\n"+
			"A kind the operator can choose and the composer never reads is a report\n"+
			"that is requested, built, and silently missing the thing it was asked for.\n"+
			"Add the branch in radio.go; do not add a name to this test.",
			want, got, keysOf(asked))
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
