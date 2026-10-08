package main

// AP-OK-01: a value taken from a module function together with its bool, with
// the bool thrown away.
//
// THE BUG IT CATCHES (#27, 0.19.0 D-142). `year, _ := readAs(...)` carried on
// with an empty year when the read failed, and wrote it over a year of real
// roll-ups. P10-07 cannot see this: it checks `error` returns only, and treats
// `_` as a deliberate discard. A bool that says "the value is good" is as much a
// return to check as an error.
//
// A SYNTAX TREE HAS NO TYPES, so the callee is known by name: the module's
// functions and methods whose results are a value and a `bool`. A bare name is
// judged only when EVERY declaration of it agrees; a name some declaration
// returns otherwise is ambiguous and left alone, because flagging it would be
// guessing. Test files are not judged: a test that discards a bool asserts the
// value another way.

import (
	"go/ast"
	"go/token"
	"strings"
)

// okFuncs is the module's declarations by name: those returning a value and a
// bool, and those returning anything else.
type okFuncs struct{ ok, other map[string]bool }

func newOKFuncs() okFuncs { return okFuncs{ok: map[string]bool{}, other: map[string]bool{}} }

// add records a file's function and method declarations.
func (o okFuncs) add(f *ast.File) {
	for _, d := range f.Decls { // a file's declarations (P10-02)
		fd, isFunc := d.(*ast.FuncDecl)
		if !isFunc {
			continue
		}
		if valueAndBool(fd.Type.Results) {
			o.ok[fd.Name.Name] = true
		} else {
			o.other[fd.Name.Name] = true
		}
	}
}

// returnsOK reports whether every declaration of name returns a value and a bool.
func (o okFuncs) returnsOK(name string) bool { return o.ok[name] && !o.other[name] }

// valueAndBool reports whether a result list is exactly two results, the second
// the predeclared bool.
func valueAndBool(results *ast.FieldList) bool {
	if results == nil {
		return false
	}
	var types []ast.Expr
	for _, r := range results.List { // a result list (P10-02)
		n := len(r.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ { // its names (P10-02)
			types = append(types, r.Type)
		}
	}
	if len(types) != 2 {
		return false
	}
	id, isIdent := types[1].(*ast.Ident)
	return isIdent && id.Name == "bool"
}

// calleeName is the name a call is made by: `f`, `x.f`, `f[T]`.
func calleeName(call *ast.CallExpr) string {
	fn := call.Fun
	switch ix := fn.(type) {
	case *ast.IndexExpr:
		fn = ix.X
	case *ast.IndexListExpr:
		fn = ix.X
	}
	switch f := fn.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// checkDiscardedOK reports `v, _ := f(...)` and `v, _ = f(...)` where f is a
// module function returning a value and a bool (AP-OK-01).
func checkDiscardedOK(fset *token.FileSet, f *ast.File, path string, fns okFuncs) []Finding {
	if strings.HasSuffix(path, "_test.go") {
		return nil
	}
	var out []Finding
	ast.Inspect(f, func(n ast.Node) bool {
		as, isAssign := n.(*ast.AssignStmt)
		if !isAssign || len(as.Lhs) != 2 || len(as.Rhs) != 1 {
			return true
		}
		if blank, isIdent := as.Lhs[1].(*ast.Ident); !isIdent || blank.Name != "_" {
			return true
		}
		call, isCall := as.Rhs[0].(*ast.CallExpr)
		if !isCall || !fns.returnsOK(calleeName(call)) {
			return true
		}
		out = append(out, Finding{
			Rule: "AP-OK-01", File: path, Line: fset.Position(as.Pos()).Line,
			Text: calleeName(call) + "'s bool is thrown away",
			Why:  "handle the bool, or give callers that need only the value a function that returns only the value",
		})
		return true
	})
	return out
}
