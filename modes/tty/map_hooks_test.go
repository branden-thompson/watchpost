package tty

// map_hooks_test.go — the map's hooks, held once (0.19.6 A-31, D-77): the
// helpers a test changes a hook through, and the guard that nothing else
// writes one.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

// ownHooks gives d hooks of its own to change, a copy of the ones it holds.
// Copies of a Dashboard share one *Hooks, so a write through the pointer
// would reach every copy, the one a test compares against included; a write
// to an unset Config field reached only the copy written (0.19.6 A-31).
func ownHooks(d *Dashboard) *Hooks {
	h := *d.mapModel.mapHooks
	d.mapModel.mapHooks = &h
	return &h
}

// configHooks gives cfg hooks of its own to change before NewDashboard: a
// copy of the ones it carries, or new ones.
func configHooks(cfg *Config) *Hooks {
	var h Hooks
	if cfg.Map != nil {
		h = *cfg.Map
	}
	cfg.Map = &h
	return &h
}

// TestTheHooksAreTakenOnceAndWrittenOnACopy holds the hooks' one owner: no
// production code but hookOwners names Config.Map, none replaces a mapHooks,
// and no code writes a hook through a held pointer (mapHooks or Map). A hook
// is written on a copy: its own literal, a local value, or a helper's result.
func TestTheHooksAreTakenOnceAndWrittenOnACopy(t *testing.T) {
	names := hookNames(t)
	files := append(treeFiles(t, "."), treeFiles(t, filepath.Join("..", "..", "app"))...) // the two trees, whole (0.19.6 TG-2)
	if len(files) < 500 || len(names) < 15 {
		t.Fatalf("COULD NOT RUN: %d files, %d hooks read", len(files), len(names))
	}
	for _, name := range files { // bounded by the files (P10-02)
		for _, bad := range hookMisuses(t, name, names) { // bounded by the file's misuses (P10-02)
			t.Error(bad)
		}
	}
}

// TestTheHookGuardSeesEachMisuse is the guard's control: each misuse it
// forbids, planted in a file of its own, is reported, and each allowed form
// is not.
func TestTheHookGuardSeesEachMisuse(t *testing.T) {
	names := hookNames(t)
	for _, c := range []struct { // bounded by the cases (P10-02)
		src  string
		want int
	}{
		{"func (d Dashboard) f() { d.mapHooks.MapFeed = nil }", 1},
		{"func f(c Config) { c.Map.Timed = nil }", 2}, // the write, and Config.Map named
		{"func f(c Config) *Hooks { return c.Map }", 1},
		{"func (d Dashboard) f() Dashboard { d.mapHooks = &Hooks{}; return d }", 1},
		{"func (d Dashboard) f() bool { return d.mapHooks.MapFeed == nil }", 0},
		{"func f(d Dashboard) { ownHooks(&d).MapFeed = nil }", 0},
		{"func f() Config { return Config{Map: &Hooks{MapFeed: nil}} }", 0},
		{"import tuimaps \"github.com/branden-thompson/go-tuimaps\"\n\nfunc f(m *tuimaps.Map) bool { return m == nil }", 0},
	} {
		name := filepath.Join(t.TempDir(), "plant.go")
		if err := os.WriteFile(name, []byte("package tty\n\n"+c.src+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := hookMisuses(t, name, names); len(got) != c.want {
			t.Errorf("%s: %d misuses %v; want %d", c.src, len(got), got, c.want)
		}
	}
}

// treeFiles are every Go file under root, its tests included.
func treeFiles(t *testing.T, root string) []string {
	t.Helper()
	fset, code, err := declset.Tree(root)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range code { // bounded by the tree's files (P10-02)
		out = append(out, fset.Position(f.Package).Filename)
	}
	tests, err := declset.TestPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	return append(out, tests...)
}

// hookNames are Hooks' field names.
func hookNames(t *testing.T) map[string]bool {
	t.Helper()
	ty := reflect.TypeFor[Hooks]()
	names := map[string]bool{}
	for i := range ty.NumField() { // bounded by the fields (P10-02)
		names[ty.Field(i).Name] = true
	}
	return names
}

// hookOwners are the functions that may name Config.Map and replace a
// mapHooks: app's ttyConfig, which fills both once at launch; NewDashboard,
// which takes the pointer; and the two test helpers above.
var hookOwners = map[string]bool{"ttyConfig": true, "NewDashboard": true, "ownHooks": true, "configHooks": true}

// hookMisuses are one file's reads of Config.Map, writes of a hook through
// a held pointer, and replacements of the pointer, outside the functions
// allowed each.
func hookMisuses(t *testing.T, name string, hooks map[string]bool) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	pkgs := map[string]bool{}
	for _, im := range f.Imports { // bounded by the file's imports (P10-02)
		p := strings.Trim(im.Path.Value, `"`)
		pkgs[p[strings.LastIndex(p, "/")+1:]] = true
		if im.Name != nil {
			pkgs[im.Name.Name] = true
		}
	}
	var out []string
	for _, decl := range f.Decls { // bounded by the file's declarations (P10-02)
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		owner := hookOwners[fn.Name.Name]
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				for _, l := range n.Lhs { // bounded by the statement (P10-02)
					if s, ok := l.(*ast.SelectorExpr); ok && hooks[s.Sel.Name] && heldPointer(s.X) {
						out = append(out, fset.Position(l.Pos()).String()+": a hook written through the held pointer, which every copy shares")
					}
					if s, ok := l.(*ast.SelectorExpr); ok && s.Sel.Name == "mapHooks" && !owner {
						out = append(out, fset.Position(l.Pos()).String()+": mapHooks replaced outside the hooks' owners")
					}
				}
			case *ast.SelectorExpr:
				if n.Sel.Name == "Map" && !owner && !strings.HasSuffix(name, "_test.go") && !packageName(n.X, pkgs) {
					out = append(out, fset.Position(n.Pos()).String()+": Config.Map read outside NewDashboard")
				}
			}
			return true
		})
	}
	return out
}

// packageName says whether x is an imported package's name, as tuimaps is
// in tuimaps.Map.
func packageName(x ast.Expr, pkgs map[string]bool) bool {
	id, ok := x.(*ast.Ident)
	return ok && pkgs[id.Name]
}

// heldPointer says whether x is a held hooks pointer: x.mapHooks or x.Map.
func heldPointer(x ast.Expr) bool {
	s, ok := x.(*ast.SelectorExpr)
	return ok && (s.Sel.Name == "mapHooks" || s.Sel.Name == "Map")
}
