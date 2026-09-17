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
// IT COMPARES TWO DERIVED SETS, AND NEITHER IS WRITTEN HERE. A test listing the
// four kinds by hand would be a SECOND closed set, drifting from the first
// exactly when the first changes — which is the defect, moved. The kinds come off
// `report.go`'s own const block; the answered set is every `report.X` the AST
// finds inside a `want.Has(...)` call in `segments`. A COUNT of the two would not
// do: answer a new kind and drop `Seismic` and the count is still four, still
// green. P-9: read the paragraph before adding to a closed set; this is the
// paragraph, made executable.
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
	// A COUNT MISSES THE CASE THAT MATTERS: answer a new kind and drop `Seismic`
	// and the count is still four, still green, with a kind the operator can
	// choose and the composer never reads. A gate measuring a PROXY for the
	// requirement instead of the requirement is this project's own definition of
	// a defect.
	//
	// BOTH SIDES STAY DERIVED, which is what makes the count form tempting. The
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

// AND THE BRANCH MUST REACH THE COMPOSER, not merely name the kind.
//
// THE GUARD ABOVE HAS A HOLE AND THIS IS IT. `TestEveryReportKindReachesTheComposer`
// records a kind as answered when `report.X` appears inside a `want.Has(...)`
// call, so `if want.Has(report.AirQuality) { }` — an empty branch — satisfies it.
// `wires` agrees for the same reason: the registry key is a READER and the call
// argument is a WRITER, and an empty branch supplies both. Two gates, one hole,
// and through it walks F-111's own defect one level down: the operator requests
// a kind, the card is built, and nothing they asked for is in it.
//
// SO THIS ASKS WHERE THE VALUE GOES. Every kind's branch must assign an
// identifier that arrives at the `Compose` call — which is what "answering a
// kind" actually means, and what an empty branch cannot fake.
//
// WHAT IT CANNOT SEE, stated because a number without its blind spots is worse
// than no number (INST-5). This is a STATIC check over identifiers, and every
// bypass found against it has the same shape: a branch that touches SOME value
// reaching Compose without that value being the kind's report — a parameter, an
// unrelated local, a `:=` shadow of the right name, a zero-value assignment. A
// static rule that closed one of those was 36 lines that closed one token and
// missed the next spelling, so it was deleted rather than extended. The last
// step of adding a kind is a FIXTURE proving the right data arrives on the air,
// and this gate is the reminder to write it, not a substitute for it.
func TestEveryKindsBranchReachesTheComposer(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "radio.go", nil, 0)
	if err != nil {
		t.Fatalf("radio.go could not be parsed: %v", err)
	}
	fn := funcNamed(f, "segments")
	if fn == nil {
		t.Fatal("radioDeck.segments not found — this guard has lost its subject and must be re-pointed, not deleted")
	}

	reaching := composeArgs(t, fn)
	for kind, assigned := range branchAssignments(fn) { // bounded by the registry (P10-02)
		var lands bool
		for _, name := range assigned { // bounded by the branch (P10-02)
			if reaching[name] {
				lands = true
				break
			}
		}
		if !lands {
			t.Errorf("`report.%s`'s branch in `segments` assigns %v, and none of it reaches Compose.\n"+
				"A branch that names the kind and hands the Composer nothing is the operator being told "+
				"their report was built while the thing they asked for is absent from it.\n"+
				"Assign the kind's value and pass it to Compose; do not satisfy this by editing the test.", kind, assigned)
		}
	}
}

// composeArgs is every identifier appearing anywhere in the Compose call's
// arguments, composite literals included — `synth.Reports{Fire: fire}` counts
// `fire` as arriving.
func composeArgs(t *testing.T, fn *ast.FuncDecl) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Compose" {
			return true
		}
		for _, arg := range call.Args { // bounded by the signature (P10-02)
			ast.Inspect(arg, func(m ast.Node) bool {
				if id, ok := m.(*ast.Ident); ok {
					out[id.Name] = true
				}
				return true
			})
		}
		return true
	})
	if len(out) == 0 {
		t.Fatal("no Compose call found in segments; this guard has lost its subject")
	}
	return out
}

// branchAssignments maps each kind named in an `if want.Has(report.X)` condition
// to the identifiers its branch assigns.
func branchAssignments(fn *ast.FuncDecl) map[string][]string {
	out := map[string][]string{}
	ast.Inspect(fn, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		kind := kindInCondition(ifs.Cond)
		if kind == "" {
			return true
		}
		var names []string
		ast.Inspect(ifs.Body, func(m ast.Node) bool {
			as, ok := m.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, lhs := range as.Lhs { // bounded by the statement (P10-02)
				switch v := lhs.(type) {
				case *ast.Ident:
					names = append(names, v.Name)
				case *ast.SelectorExpr: // maritime.Forecast = … still lands on `maritime`
					if id, ok := v.X.(*ast.Ident); ok {
						names = append(names, id.Name)
					}
				case *ast.IndexExpr: // products[i].Text = … lands on `products`
					if id, ok := v.X.(*ast.Ident); ok {
						names = append(names, id.Name)
					}
				}
			}
			return true
		})
		out[kind] = names
		return true
	})
	return out
}

// kindInCondition is the `report.X` a condition asks `want.Has` about, or "".
func kindInCondition(cond ast.Expr) string {
	var kind string
	ast.Inspect(cond, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Has" {
			return true
		}
		if arg, ok := call.Args[0].(*ast.SelectorExpr); ok {
			if pkg, ok := arg.X.(*ast.Ident); ok && pkg.Name == "report" {
				kind = arg.Sel.Name
			}
		}
		return true
	})
	return kind
}

// funcNamed is the top-level or method declaration called name.
func funcNamed(f *ast.File, name string) *ast.FuncDecl {
	var out *ast.FuncDecl
	ast.Inspect(f, func(n ast.Node) bool {
		if d, ok := n.(*ast.FuncDecl); ok && d.Name.Name == name {
			out = d
		}
		return out == nil
	})
	return out
}
