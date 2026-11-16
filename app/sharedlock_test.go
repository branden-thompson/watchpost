package app

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

// sharedOwners are the types whose `mu` is livePipelines' mutex: the root's
// own, and the deck's and the map data's, which hold it by pointer (0.19.6
// D-16).
var sharedOwners = map[string]bool{"livePipelines": true, "Deck": true, "Feeds": true}

// funcValueCalls are the calls under the shared lock through a function value,
// each by the function it is made in and its text (0.19.6 A-32), recorded at
// 0.19.6 B0. In production each is asynchronous or takes no lock of the
// tree's: the publisher's trigger (rp.publish), its timer (pb.after, nil in
// production, so time.AfterFunc), a scheduler's constructor (rp.newFor) and
// the clock (now). Those are design AP-3.5's four values; rp.newFor is called
// in two declarations, so five rows. GOTCHA: a method call on a value whose
// type the walk cannot read resolves to the tree's methods of that name, or to
// nothing when the tree has none, so a func field on such a value is not
// seen. That is the check's one blind spot, and the deadlock test runs the
// paths it would hide.
var funcValueCalls = map[string]bool{
	"publisher.Trigger: pb.after":       true,
	"publisher.windowNow: now":          true,
	"recentPipeline.update: rp.newFor":  true,
	"recentPipeline.update: rp.publish": true,
	"startRecent: rp.newFor":            true,
}

// lockTree is the parsed app tree, its functions by key ("Recv.Name" or
// "Name"), and the planned package of each.
type lockTree struct {
	fset    *token.FileSet
	funcs   map[string]*ast.FuncDecl
	file    map[string]string // function key → its file
	planned map[string]string // function key → its planned package
	methods map[string][]string
	pkgs    map[string]bool              // the package names the tree's files import
	fields  map[string]map[string]string // struct type → field → the field's type, "" for a func or other
}

// loadLockTree parses every non-test file under root and reads the planned
// layout's map, failing on a file the map does not name.
func loadLockTree(t *testing.T, root, plannedFile string) *lockTree {
	t.Helper()
	paths, err := declset.SourcePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := readPlan(t, plannedFile)
	lt := &lockTree{fset: token.NewFileSet(), funcs: map[string]*ast.FuncDecl{}, file: map[string]string{},
		planned: map[string]string{}, methods: map[string][]string{}, pkgs: map[string]bool{}, fields: map[string]map[string]string{}}
	for _, p := range paths {
		f, err := parser.ParseFile(lt.fset, p, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		rel := "app/" + filepath.ToSlash(strings.TrimPrefix(p, root+string(filepath.Separator)))
		pkg, ok := plan["file "+rel]
		if !ok {
			t.Errorf("%s is in no row of %s: a file the planned layout does not place is a file this check cannot judge", rel, plannedFile)
		}
		for _, im := range f.Imports {
			path := strings.Trim(im.Path.Value, `"`)
			name := filepath.Base(path)
			if im.Name != nil {
				name = im.Name.Name
			}
			lt.pkgs[name] = true
		}
		for _, d := range f.Decls {
			lt.noteFields(d)
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			key := funcKey(fn)
			lt.funcs[key], lt.file[key] = fn, rel
			lt.planned[key] = pkg
			if over, ok := plan["method "+key]; ok {
				lt.planned[key] = over
			}
			if fn.Recv != nil {
				lt.methods[fn.Name.Name] = append(lt.methods[fn.Name.Name], key)
			}
		}
	}
	return lt
}

// noteFields records each struct type's fields and their types.
func (lt *lockTree) noteFields(d ast.Decl) {
	g, ok := d.(*ast.GenDecl)
	if !ok || g.Tok != token.TYPE {
		return
	}
	for _, sp := range g.Specs {
		ts := sp.(*ast.TypeSpec)
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}
		m := map[string]string{}
		for _, f := range st.Fields.List {
			for _, n := range f.Names {
				m[n.Name] = typeName(f.Type)
			}
			if len(f.Names) == 0 {
				m[typeName(f.Type)] = typeName(f.Type) // embedded
			}
		}
		lt.fields[ts.Name.Name] = m
	}
}

// typeName is a type expression's name, through a pointer: "T" for one of the
// tree's types, "pkg.T" for another package's, "" for a func, map or the like.
func typeName(e ast.Expr) string {
	if s, ok := e.(*ast.StarExpr); ok {
		e = s.X
	}
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
	case *ast.IndexExpr:
		return typeName(e.X)
	}
	return ""
}

// env is a function's identifiers and their types: the receiver, the
// parameters, and the locals assigned from a field, a literal or a
// conversion the tree can read.
func (lt *lockTree) env(fn *ast.FuncDecl) map[string]string {
	vars := map[string]string{}
	if fn.Recv != nil && recvName(fn) != "" {
		vars[recvName(fn)] = recvOf(fn)
	}
	for _, f := range fn.Type.Params.List {
		for _, n := range f.Names {
			vars[n.Name] = typeName(f.Type)
		}
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != len(as.Rhs) {
			return true
		}
		for i, l := range as.Lhs {
			if id, ok := l.(*ast.Ident); ok {
				if typ := lt.exprType(vars, as.Rhs[i]); typ != "" || vars[id.Name] == "" {
					vars[id.Name] = typ // a local the walk cannot type is still a local
				}
			}
		}
		return true
	})
	return vars
}

// exprType is the type of an expression where the tree can read it: an
// identifier, a field, a literal, an address of one.
func (lt *lockTree) exprType(vars map[string]string, e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return vars[e.Name]
	case *ast.SelectorExpr:
		if t := lt.exprType(vars, e.X); t != "" {
			return lt.fields[t][e.Sel.Name]
		}
	case *ast.UnaryExpr:
		return lt.exprType(vars, e.X)
	case *ast.CompositeLit:
		return typeName(e.Type)
	case *ast.ParenExpr:
		return lt.exprType(vars, e.X)
	}
	return ""
}

// readPlan reads the planned layout: "file <path> <pkg>" and
// "method <Type.name> <pkg>" rows.
func readPlan(t *testing.T, file string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Split(line, "\t")
		if len(f) == 3 {
			out[f[0]+" "+f[1]] = f[2]
		}
	}
	return out
}

// funcKey is "Recv.Name" for a method and "Name" for a function.
func funcKey(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return recvOf(fn) + "." + fn.Name.Name
}

// recvOf is a method's receiver type name.
func recvOf(fn *ast.FuncDecl) string {
	e := fn.Recv.List[0].Type
	if s, ok := e.(*ast.StarExpr); ok {
		e = s.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return "?"
}

// recvName is a method's receiver variable, or "".
func recvName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 || len(fn.Recv.List[0].Names) == 0 {
		return ""
	}
	return fn.Recv.List[0].Names[0].Name
}

// isSharedCall reports whether a statement is `<recv>.mu.<op>()` in a method
// of a shared owner, op Lock or Unlock, deferred when defer is true.
func isSharedCall(fn *ast.FuncDecl, s ast.Stmt, op string, deferred bool) bool {
	if fn.Recv == nil || !sharedOwners[recvOf(fn)] {
		return false
	}
	var call *ast.CallExpr
	switch s := s.(type) {
	case *ast.ExprStmt:
		if deferred {
			return false
		}
		call, _ = s.X.(*ast.CallExpr)
	case *ast.DeferStmt:
		if !deferred {
			return false
		}
		call = s.Call
	}
	if call == nil {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != op {
		return false
	}
	mu, ok := sel.X.(*ast.SelectorExpr)
	if !ok || mu.Sel.Name != "mu" {
		return false
	}
	x, ok := mu.X.(*ast.Ident)
	return ok && x.Name == recvName(fn)
}

// section is the statements one shared-lock section runs.
type section struct {
	fn    *ast.FuncDecl
	stmts []ast.Stmt
	at    string
}

// sections finds every shared-lock section in a function: from a Lock to the
// Unlock in the same block, or to the block's end when the Unlock is deferred.
func sections(lt *lockTree, fn *ast.FuncDecl) []section {
	var out []section
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		b, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		for i, s := range b.List {
			if !isSharedCall(fn, s, "Lock", false) {
				continue
			}
			end := len(b.List)
			for j := i + 1; j < len(b.List); j++ {
				if isSharedCall(fn, b.List[j], "Unlock", false) {
					end = j
					break
				}
			}
			out = append(out, section{fn: fn, stmts: b.List[i+1 : end], at: lt.fset.Position(s.Pos()).String()})
		}
		return true
	})
	return out
}

// takesSharedLock reports whether a function locks a shared owner's mutex.
func takesSharedLock(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if s, ok := n.(ast.Stmt); ok && isSharedCall(fn, s, "Lock", false) {
			found = true
		}
		return !found
	})
	return found
}

// calls lists the functions a node calls in the tree, and the calls it makes
// through a value. A function literal's body and a goroutine run later,
// outside the section, and the lock's own Lock and Unlock are not calls under
// it. A method call resolves by the type of what it is called on; one on
// another package's type calls nothing of the tree's; one whose type the walk
// cannot read resolves to every method of the tree with its name.
func calls(lt *lockTree, in *ast.FuncDecl, nodes []ast.Stmt) (callees []string, values []string) {
	vars := lt.env(in)
	for _, st := range nodes {
		ast.Inspect(st, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncLit, *ast.GoStmt:
				return false
			case *ast.CallExpr:
				c, v := lt.resolve(vars, in, n)
				callees = append(callees, c...)
				values = append(values, v...)
			}
			return true
		})
	}
	return callees, values
}

// resolve is one call's callees, or its text when it is a call through a value.
func (lt *lockTree) resolve(vars map[string]string, in *ast.FuncDecl, call *ast.CallExpr) ([]string, []string) {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		if _, ok := lt.funcs[f.Name]; ok {
			return []string{f.Name}, nil
		}
		if _, ok := vars[f.Name]; ok {
			return nil, []string{funcKey(in) + ": " + f.Name}
		}
		return nil, nil // a builtin or a conversion
	case *ast.SelectorExpr:
		if mu, ok := f.X.(*ast.SelectorExpr); ok && mu.Sel.Name == "mu" && (f.Sel.Name == "Lock" || f.Sel.Name == "Unlock") {
			return nil, nil
		}
		if x, ok := f.X.(*ast.Ident); ok && lt.pkgs[x.Name] && vars[x.Name] == "" {
			return nil, nil // a package's function
		}
		typ := lt.exprType(vars, f.X)
		if strings.Contains(typ, ".") {
			return nil, nil // a method of another package's type
		}
		if typ != "" {
			if _, ok := lt.funcs[typ+"."+f.Sel.Name]; ok {
				return []string{typ + "." + f.Sel.Name}, nil
			}
			if ft, ok := lt.fields[typ][f.Sel.Name]; ok && ft == "" {
				return nil, []string{funcKey(in) + ": " + lockExprText(f)}
			}
			return nil, nil // a method the type gets from elsewhere
		}
		if cands := lt.methods[f.Sel.Name]; len(cands) > 0 {
			return cands, nil
		}
		return nil, nil // a method of a value the walk cannot type, which the tree does not declare
	}
	switch call.Fun.(type) {
	case *ast.ArrayType, *ast.MapType, *ast.ChanType, *ast.FuncType, *ast.InterfaceType, *ast.StructType:
		return nil, nil // a conversion
	}
	return nil, []string{funcKey(in) + ": a call through an expression"}
}

// lockExprText prints a selector for a message.
func lockExprText(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return lockExprText(e.X) + "." + e.Sel.Name
	}
	return "?"
}

// seenValues collects the recorded value calls a walk met, so a record no
// section reaches any more fails as stale.
var seenValues = map[string]bool{}

// lockProblems judges every shared-lock section of a tree (0.19.6 D-16, A-19,
// A-32): nothing it reaches takes the lock again, nothing it calls is in
// another planned package, and every call it makes through a value is
// recorded. It returns the sections found too.
func lockProblems(lt *lockTree) (found int, probs []string) {
	keys := make([]string, 0, len(lt.funcs))
	for k := range lt.funcs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, sec := range sections(lt, lt.funcs[k]) {
			found++
			probs = append(probs, judgeSection(lt, k, sec)...)
		}
		if strings.HasSuffix(k, "Locked") {
			if by := reachesLock(lt, k); by != "" {
				probs = append(probs, fmt.Sprintf("%s: %s is named for a caller holding the lock, and %s takes it", lt.file[k], k, by))
			}
		}
	}
	sort.Strings(probs)
	return found, probs
}

// reachesLock is the first function, from key through its static calls, that
// takes the shared lock, or "". Bounded by the tree's functions, each visited
// once (P10-02).
func reachesLock(lt *lockTree, key string) string {
	seen := map[string]bool{}
	queue := []string{key}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if seen[c] {
			continue
		}
		seen[c] = true
		if takesSharedLock(lt.funcs[c]) {
			return c
		}
		next, _ := calls(lt, lt.funcs[c], lt.funcs[c].Body.List)
		queue = append(queue, next...)
	}
	return ""
}

// judgeSection is one section's problems.
func judgeSection(lt *lockTree, key string, sec section) []string {
	var probs []string
	direct, values := calls(lt, sec.fn, sec.stmts)
	for _, v := range values {
		seenValues[v] = true
		if !funcValueCalls[v] {
			probs = append(probs, fmt.Sprintf("%s: under the lock, a call through a value not recorded at B0: %s", sec.at, v))
		}
	}
	seen := map[string]bool{}
	queue := append([]string(nil), direct...)
	for len(queue) > 0 { // bounded by the tree's functions, each visited once (P10-02)
		c := queue[0]
		queue = queue[1:]
		if seen[c] {
			continue
		}
		seen[c] = true
		fn := lt.funcs[c]
		if takesSharedLock(fn) {
			probs = append(probs, fmt.Sprintf("%s: under the lock, %s takes it again: a deadlock", sec.at, c))
		}
		if lt.planned[c] != lt.planned[key] {
			probs = append(probs, fmt.Sprintf("%s: under the lock, %s (planned %s) calls %s (planned %s): a call that crosses a package", sec.at, key, lt.planned[key], c, lt.planned[c]))
		}
		next, more := calls(lt, fn, fn.Body.List)
		queue = append(queue, next...)
		for _, v := range more {
			seenValues[v] = true
			if !funcValueCalls[v] {
				probs = append(probs, fmt.Sprintf("%s: under the lock, a call through a value not recorded at B0: %s", sec.at, v))
			}
		}
	}
	return probs
}

// TestNoSectionUnderTheSharedLockTakesItOrLeavesItsPackage is 0.19.6 D-16,
// A-19 and A-32. livePipelines' mutex is shared by pointer with the deck and
// the map data once they move, and a lock taken in one package and assumed in
// another is a deadlock -race cannot see. So no section under it reaches a
// function that takes it again, no section calls into another package of the
// planned layout, and every call it makes through a function value is one
// recorded at B0. The planted fixture fails each rule; the tree's sections are
// counted against a fixed floor.
func TestNoSectionUnderTheSharedLockTakesItOrLeavesItsPackage(t *testing.T) {
	fixture := loadLockTree(t, filepath.Join("testdata", "sharedlock", "fixture"), filepath.Join("testdata", "sharedlock", "fixture.tsv"))
	_, planted := lockProblems(fixture)
	got := strings.Join(planted, "\n")
	for _, want := range []string{"takes it again", "crosses a package", "a call through a value not recorded", "is named for a caller holding the lock"} {
		if !strings.Contains(got, want) {
			t.Errorf("the planted fixture was not caught by its %q case:\n%s", want, got)
		}
	}
	lt := loadLockTree(t, ".", filepath.Join("testdata", "sharedlock", "planned.tsv"))
	found, probs := lockProblems(lt)
	if found < 25 { // the shared lock's sections at 0.19.6 B0 (design AP-3)
		t.Fatalf("found %d sections under the shared lock; fewer than the 25 this check was fixed on", found)
	}
	t.Logf("%d sections under the shared lock", found)
	for _, p := range probs {
		t.Error(p)
	}
	for v := range funcValueCalls {
		if !seenValues[v] {
			t.Errorf("funcValueCalls records %q, which no section reaches any more: drop it", v)
		}
	}
}
