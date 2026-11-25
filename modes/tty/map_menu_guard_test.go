package tty

// map_menu_guard_test.go — the Overlays menu's derived guards (0.19.6 A-33):
// what TestTheMenuIsTheHUMLEADsLayout's hand-listed rows cannot see.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"
)

// TestEveryRegistryTheMenuIsBuiltFromHasItsRow is the layout's derived half:
// each registered layer, each alert category and each detail switch has
// exactly one row, so one the menu drops is seen here, where the hand-listed
// layout, written from the same rows, passes it. The radar is a mode (0.18.0
// D-117) and Feels like is Temperature's choice (0.18.0 U2-39), never rows.
func TestEveryRegistryTheMenuIsBuiltFromHasItsRow(t *testing.T) {
	d := menuMap(t)
	rows := map[string]int{}
	for _, r := range d.overlayRows() { // bounded by the menu's rows (P10-02)
		rows[r.key]++
	}
	var want []string
	for _, l := range d.cfg.MapLayers { // bounded by the registered layers (P10-02)
		if l.Key != RadarLayer && l.Key != FeelsLayer {
			want = append(want, l.Key)
		}
	}
	for _, c := range AlertCategories() { // bounded by the categories (P10-02)
		want = append(want, AlertCategorySwitch(c.Key))
	}
	want = append(want, detailLevelKey)
	for _, l := range mapDetailLayers() { // bounded by the detail switches (P10-02)
		want = append(want, l.key)
	}
	if len(want) < 20 {
		t.Fatalf("COULD NOT RUN: %d registry keys, fewer than the fixture registers", len(want))
	}
	for _, k := range want { // bounded by the registry keys (P10-02)
		if rows[k] != 1 {
			t.Errorf("%q has %d rows in the Overlays menu; want 1", k, rows[k])
		}
	}
}

// TestOnlyADetailRowReachesSwitchRowsFallThrough holds switchRow's last line,
// which saves the row as a detail switch, to the one kind it is for: every
// other rowKind is named by a case of its own. A new kind without its space
// case would otherwise write its key into the saved map_detail word, silently
// (0.19.6 A-33). Read from map_boxes.go's source, so the guard needs no row of
// the new kind to exist.
func TestOnlyADetailRowReachesSwitchRowsFallThrough(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "map_boxes.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	kinds, cased := rowKinds(f), switchRowCases(t, f)
	if len(kinds) < 6 {
		t.Fatalf("COULD NOT RUN: %d rowKind constants read from map_boxes.go", len(kinds))
	}
	var unnamed []string
	for _, k := range kinds { // bounded by the kinds (P10-02)
		if !slices.Contains(cased, k) {
			unnamed = append(unnamed, k)
		}
	}
	if !slices.Equal(unnamed, []string{"menuDetail"}) {
		t.Errorf("the kinds switchRow's fall-through takes are %v; want only menuDetail, every other kind its own case", unnamed)
	}
}

// rowKinds are the rowKind constants map_boxes.go declares, in order: the
// first typed rowKind, and each name after it in the same block.
func rowKinds(f *ast.File) []string {
	var out []string
	for _, decl := range f.Decls { // bounded by the file's declarations (P10-02)
		g, ok := decl.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		in := false
		for _, spec := range g.Specs { // bounded by the block (P10-02)
			v := spec.(*ast.ValueSpec)
			if id, ok := v.Type.(*ast.Ident); ok {
				in = id.Name == "rowKind"
			}
			if in {
				for _, n := range v.Names { // bounded by the spec's names (P10-02)
					out = append(out, n.Name)
				}
			}
		}
	}
	return out
}

// switchRowCases are the names switchRow's top-level switch has a case for.
func switchRowCases(t *testing.T, f *ast.File) []string {
	t.Helper()
	for _, decl := range f.Decls { // bounded by the file's declarations (P10-02)
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "switchRow" {
			continue
		}
		for _, st := range fn.Body.List { // bounded by the body's statements (P10-02)
			sw, ok := st.(*ast.SwitchStmt)
			if !ok {
				continue
			}
			var out []string
			for _, c := range sw.Body.List { // bounded by the cases (P10-02)
				for _, e := range c.(*ast.CaseClause).List { // bounded by the case's names (P10-02)
					if id, ok := e.(*ast.Ident); ok {
						out = append(out, id.Name)
					}
				}
			}
			return out
		}
	}
	t.Fatal("COULD NOT RUN: no switchRow with a switch in map_boxes.go")
	return nil
}
