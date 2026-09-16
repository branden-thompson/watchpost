package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"
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

	// THE KIND NAMES, DERIVED FROM THE REGISTRY'S OWN SOURCE — not listed here.
	//
	// misses: answer a new kind and drop `Seismic` and the count is still four,
	// still green, with a kind the operator can choose and the composer never
	// reads. A gate measuring a PROXY for the requirement instead of the
	// requirement is this project's own definition of a defect — and this one was
	// written the same day that defect was made the centrepiece of a red-team
	// round.
	//
	// BOTH SIDES STAY DERIVED, which is what kept the count form tempting. The
	// registry's identifiers come from `report.go`'s own const block, so a fifth
	// kind is picked up here with no edit; the answered set comes from the AST.
	// Neither is a hand-written list that can drift from the other.
	want := kindIdents(t)
	for _, k := range want {
		if !asked[k] {
			t.Errorf("`report.%s` is a kind the operator can choose and `segments` never reads.\n"+
				"A report requested with it is built and silently missing what was asked for.\n"+
				"Add the branch in radio.go; do not add a name to this test.", k)
		}
	}
	for k := range asked {
		if !slices.Contains(want, k) {
			t.Errorf("`segments` reads `report.%s`, which the registry does not declare", k)
		}
	}
}

// kindIdents is every `Kind` the registry declares, read off its own const block.
//
// DERIVED, NOT LISTED. A hand-written list here would be a SECOND closed set,
// drifting from the first exactly when the first changes — which is the defect
// this guard exists to catch, moved one file along.
func kindIdents(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../platform/report/report.go", nil, 0)
	if err != nil {
		t.Fatalf("the kind registry could not be parsed: %v", err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		gd, ok := n.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			return true
		}
		// THE BLOCK THAT DECLARES `Kind`, NOT EVERY CONST IN THE FILE. The first
		// spec names the type — `NWS Kind = iota` — and the rest inherit it.
		// Without this the walk also collected the display-string constants and
		// reported them as kinds nobody composes, which is a guard failing for a
		// reason that has nothing to do with what it guards.
		first, ok := gd.Specs[0].(*ast.ValueSpec)
		if !ok || first.Type == nil {
			return true
		}
		if id, ok := first.Type.(*ast.Ident); !ok || id.Name != "Kind" {
			return true
		}
		for _, sp := range gd.Specs {
			vs, ok := sp.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, id := range vs.Names {
				// `numKinds` is the sentinel, not a kind.
				if id.Name == "numKinds" {
					return false
				}
				out = append(out, id.Name)
			}
		}
		return false
	})
	if len(out) == 0 {
		t.Fatal("no Kind constants found; this guard has lost its subject and must be re-pointed, not deleted")
	}
	return out
}
