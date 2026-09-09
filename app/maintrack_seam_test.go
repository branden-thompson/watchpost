package app

// THE MERGE'S SAFETY IS STRUCTURAL, so the check is too (0.16.0 P3).
//
// Two owners of the audio device speaking at once is this batch's defect, and
// what prevents it is that every path to a synthesised read goes through ONE
// function that asks the stage. A fourth call site added later — a new fallback,
// a retry, a settings action — would reintroduce the second speaker silently,
// and no behavioural test would notice, because the site would be correct in
// isolation and wrong only in company.
//
// DERIVED, NOT ENUMERATED (INST-1): the walk finds the call sites rather than
// comparing against a remembered list. AT P3(d) THIS CHECK GETS STRICTER, not
// looser — startSynth goes to ZERO callers when the direct path retires, and
// this test is where that is stated.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// callersOf walks the package's non-test files and reports, for every call to
// a method with this name, the name of the function it was called from.
func callersOf(t *testing.T, method string) []string {
	t.Helper()
	fset := token.NewFileSet()
	pkgMap, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, pkg := range pkgMap {
		for name, file := range pkg.Files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				ast.Inspect(fn, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != method {
						return true
					}
					out = append(out, fn.Name.Name+" ("+fset.Position(call.Pos()).String()+")")
					return true
				})
			}
		}
	}
	return out
}

func TestEveryPathToASynthesisedReadGoesThroughTheOneSeam(t *testing.T) {
	callers := callersOf(t, "startSynth")
	// SILENCE IS A DISTINCT VERDICT (INST-2). Zero callers today would mean the
	// walk broke, not that the tree is clean — and at P3(d), when zero becomes
	// the right answer, this test is rewritten rather than left to pass by
	// accident.
	if len(callers) == 0 {
		t.Fatal("found no call to startSynth — the walk did not run, which is not the same as passing. " +
			"If the direct path has retired (P3(d)), this test is now the one that asserts ZERO, and must say so")
	}
	for _, c := range callers {
		if !strings.HasPrefix(c, "needsRead ") {
			t.Errorf("startSynth is called from %s — every path to a synthesised read must go through "+
				"needsRead, which is the one place the merge stage is asked. A site that starts audio "+
				"itself is the second speaker P3 exists to remove, and it is invisible to any test that "+
				"looks at one site at a time", c)
		}
	}
	t.Logf("checked %d call(s) to startSynth; blind to any made from outside package app or through a function value", len(callers))
}

// THE FACT IS REPORTED FROM ONE PLACE TOO. A second site telling the Director a
// location needs a read would queue the same card twice under two different
// staleness rules — and only one of them checks the epoch.
func TestTheNeedIsReportedFromTheOneSeam(t *testing.T) {
	fset := token.NewFileSet()
	pkgMap, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, pkg := range pkgMap {
		for name, file := range pkg.Files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				ast.Inspect(fn, func(n ast.Node) bool {
					lit, ok := n.(*ast.CompositeLit)
					if !ok {
						return true
					}
					sel, ok := lit.Type.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "NeedsRead" {
						return true
					}
					if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "lineup" {
						return true
					}
					found++
					if fn.Name.Name != "needsRead" {
						t.Errorf("%s: lineup.NeedsRead is built in %s — the fact is reported from needsRead "+
							"alone, which is the only place that checks the epoch before reporting it",
							fset.Position(lit.Pos()), fn.Name.Name)
					}
					return true
				})
			}
		}
	}
	if found == 0 {
		t.Fatal("found no lineup.NeedsRead literal — the producer is not wired, or the walk broke")
	}
	t.Logf("checked %d lineup.NeedsRead literal(s); blind to one built outside package app", found)
}
