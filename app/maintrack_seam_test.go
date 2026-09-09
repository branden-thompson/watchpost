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
// comparing against a remembered list.
//
// IT WALKS THE WHOLE FILE, NOT ONLY ITS FUNCTIONS (red team 2026-09-09,
// finding 2). The first version iterated `file.Decls` and skipped everything
// that was not an `*ast.FuncDecl`, so a second speaker declared at package
// level —
//
//	var secondSpeaker = func(d *radioDeck, ref snapshot.LocationRef) {
//		d.startSynth(ref, "a retry nobody routed through the seam", 0)
//	}
//
// — was invisible to BOTH walks, and the counts did not move, so the INST-2
// gate could not see it either. That is precisely the defect these tests exist
// to catch, declared in the file they were reading. AT P3(d) THIS CHECK GETS STRICTER, not
// looser — startSynth goes to ZERO callers when the direct path retires, and
// this test is where that is stated.

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

// site is one occurrence, and the top-level declaration it was found in.
//
// THE DECLARATION, NOT THE FUNCTION. A call inside a package-level var is
// inside no function at all, and reporting it as "not in needsRead" is the
// whole point — the walk must have somewhere to put it rather than a reason to
// skip it.
type site struct{ in, at string }

// declName is what a top-level declaration is called, for any kind of it.
func declName(d ast.Decl) string {
	switch v := d.(type) {
	case *ast.FuncDecl:
		return v.Name.Name
	case *ast.GenDecl:
		var names []string
		for _, spec := range v.Specs { // bounded by the declaration (P10-02)
			switch sp := spec.(type) {
			case *ast.ValueSpec:
				for _, n := range sp.Names {
					names = append(names, n.Name)
				}
			case *ast.TypeSpec:
				names = append(names, sp.Name.Name)
			}
		}
		return v.Tok.String() + " " + strings.Join(names, ",")
	}
	return "an unnamed declaration"
}

// walkPackage visits every node of every non-test file, telling the visitor
// which top-level declaration it is inside.
func walkPackage(t *testing.T, visit func(in string, pos func(ast.Node) string, n ast.Node)) {
	t.Helper()
	fset, files, err := declset.Files(".")
	if err != nil {
		t.Fatal(err)
	}
	pos := func(n ast.Node) string { return fset.Position(n.Pos()).String() }
	for _, file := range files { // bounded by the package (P10-02)
		for _, decl := range file.Decls {
			name := declName(decl)
			ast.Inspect(decl, func(n ast.Node) bool {
				visit(name, pos, n)
				return true
			})
		}
	}
}

// callsTo is every call to a method of this name, with where it was made from.
func callsTo(t *testing.T, method string) []site {
	t.Helper()
	var out []site
	walkPackage(t, func(in string, pos func(ast.Node) string, n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != method {
			return
		}
		out = append(out, site{in: in, at: pos(call)})
	})
	return out
}

func TestEveryPathToASynthesisedReadGoesThroughTheOneSeam(t *testing.T) {
	callers := callsTo(t, "startSynth")
	// SILENCE IS A DISTINCT VERDICT (INST-2). Zero callers today would mean the
	// walk broke, not that the tree is clean — and at P3(d), when zero becomes
	// the right answer, this test is rewritten rather than left to pass by
	// accident.
	if len(callers) == 0 {
		t.Fatal("found no call to startSynth — the walk did not run, which is not the same as passing. " +
			"If the direct path has retired (P3(d)), this test is now the one that asserts ZERO, and must say so")
	}
	for _, c := range callers {
		if c.in != "needsRead" {
			t.Errorf("%s: startSynth is called from %s — every path to a synthesised read must go through "+
				"needsRead, which is the one place the merge stage is asked. A site that starts audio "+
				"itself is the second speaker P3 exists to remove, and it is invisible to any test that "+
				"looks at one site at a time", c.at, c.in)
		}
	}
	t.Logf("checked %d call(s) to startSynth; blind to any made from outside package app, or through a "+
		"function value whose call site names something other than startSynth", len(callers))
}

// THE FACT IS REPORTED FROM ONE PLACE TOO. A second site telling the Director a
// location needs a read would queue the same card twice under two different
// staleness rules — and only one of them checks the epoch.
func TestTheNeedIsReportedFromTheOneSeam(t *testing.T) {
	found := 0
	walkPackage(t, func(in string, pos func(ast.Node) string, n ast.Node) {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NeedsRead" {
			return
		}
		if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "lineup" {
			return
		}
		found++
		if in != "needsRead" {
			t.Errorf("%s: lineup.NeedsRead is built in %s — the fact is reported from needsRead "+
				"alone, which is the only place that checks the epoch before reporting it", pos(lit), in)
		}
	})
	if found == 0 {
		t.Fatal("found no lineup.NeedsRead literal — the producer is not wired, or the walk broke")
	}
	t.Logf("checked %d lineup.NeedsRead literal(s); blind to one built outside package app", found)
}
