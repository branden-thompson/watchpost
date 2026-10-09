package tty

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

// modeConstants are the map modes, read from the package's own declarations:
// every constant of type mapMode, the type carried down a const block as Go
// carries it to a spec that repeats the one before.
func modeConstants(files []*ast.File) []string {
	var out []string
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			typed := false
			for _, s := range gd.Specs {
				spec := s.(*ast.ValueSpec)
				switch {
				case spec.Type != nil:
					id, ok := spec.Type.(*ast.Ident)
					typed = ok && id.Name == "mapMode"
				case spec.Values != nil:
					typed = false
				}
				for _, name := range spec.Names {
					if typed {
						out = append(out, name.Name)
					}
				}
			}
		}
	}
	return out
}

// modeSlips are the places a file decides by mode without naming each mode
// it means: a mode compared with !=, a comparison of modes negated, a switch
// on the mode that leaves a mode out or has a default, and the old boolean.
func modeSlips(fset *token.FileSet, f *ast.File, modes []string) []string {
	var out []string
	isMode := func(e ast.Expr) bool {
		id, ok := e.(*ast.Ident)
		return ok && slices.Contains(modes, id.Name)
	}
	names := func(e ast.Expr) bool {
		found := false
		ast.Inspect(e, func(n ast.Node) bool {
			if x, ok := n.(ast.Expr); ok && isMode(x) {
				found = true
			}
			return !found
		})
		return found
	}
	at := func(n ast.Node, why string) { out = append(out, fset.Position(n.Pos()).String()+": "+why) }
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BinaryExpr:
			if x.Op == token.NEQ && (isMode(x.X) || isMode(x.Y)) {
				at(x, "a mode compared with != also means every mode added later")
			}
		case *ast.UnaryExpr:
			if x.Op == token.NOT && names(x.X) {
				at(x, "a negated mode comparison also means every mode added later")
			}
		case *ast.Ident:
			if x.Name == "radarMode" {
				at(x, "the Radar-or-not boolean: name the mode")
			}
		case *ast.IfStmt:
			if x.Else != nil && names(x.Cond) {
				at(x, "an else after a mode comparison: a mode added later lands in it")
			}
		case *ast.SwitchStmt:
			if x.Tag == nil {
				decides, hasDefault := false, false
				for _, s := range x.Body.List {
					cc := s.(*ast.CaseClause)
					hasDefault = hasDefault || cc.List == nil
					for _, e := range cc.List {
						decides = decides || names(e)
					}
				}
				if decides && hasDefault {
					at(x, "a default in a switch that decides by mode: a mode added later lands in it")
				}
				return true
			}
			call, ok := x.Tag.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); !ok || sel.Sel.Name != "mapMode" {
				return true
			}
			var listed []string
			for _, s := range x.Body.List {
				cc := s.(*ast.CaseClause)
				if cc.List == nil {
					at(cc, "a default in a switch on the mode: a mode added later lands in it")
				}
				for _, e := range cc.List {
					if id, ok := e.(*ast.Ident); ok {
						listed = append(listed, id.Name)
					}
				}
			}
			for _, m := range modes {
				if !slices.Contains(listed, m) {
					at(x, "a switch on the mode leaves out "+m)
				}
			}
		}
		return true
	})
	return out
}

// TestEveryMapModeIsHandledAtEverySite is W2.0 (C-M2, A-29): every place the
// map decides by its mode names the modes it means - == a mode, or a switch
// listing every mode with no default - so a mode added later (Propagation,
// W2.1) falls into no other mode's behaviour unless a site says so. Planted
// slips are caught.
func TestEveryMapModeIsHandledAtEverySite(t *testing.T) {
	fset, files, err := declset.Files(".")
	if err != nil {
		t.Fatal(err)
	}
	modes := modeConstants(files)
	if len(modes) < 2 {
		t.Fatalf("found the modes %v: the walk broke, which is not the same as passing", modes)
	}
	sites := 0
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "mapMode" {
				sites++
			}
			return true
		})
		for _, slip := range modeSlips(fset, f, modes) {
			t.Error(slip)
		}
	}
	if sites < 27 {
		t.Errorf("%d sites decide by the mode; the boolean had 27 (the plan's 28 counted its definition)", sites)
	}
	planted := `package p
func (d D) a() bool { return d.mapMode() != modeRadar }
func (d D) b() bool { return !(d.mapMode() == modeForecast) }
func (d D) c() bool { return d.radarMode() }
func (d D) e() { switch d.mapMode() { case modeRadar: } }
func (d D) g() { switch d.mapMode() { default: } }
func (d D) h() { if d.mapMode() == modeRadar { d.x() } else { d.y() } }
func (d D) i() { switch { case d.mapMode() == modeRadar: d.x(); default: d.y() } }
func (d D) ok() bool { if d.mapMode() == modeForecast { return true }; switch { case d.mapMode() == modeRadar: }; return false }
`
	pf, err := parser.ParseFile(fset, "planted.go", planted, 0)
	if err != nil {
		t.Fatal(err)
	}
	flagged := map[int]bool{}
	for _, f := range modeSlips(fset, pf, modes) {
		var line int
		if _, err := fmt.Sscanf(strings.TrimPrefix(f, "planted.go:"), "%d", &line); err == nil {
			flagged[line] = true
		}
	}
	for line := 2; line <= 8; line++ {
		if !flagged[line] {
			t.Errorf("planted slip on line %d was not caught", line)
		}
	}
	if flagged[9] {
		t.Error("the planted function that names its modes was flagged")
	}
}
