package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// module is the repository's import path.
const module = "github.com/branden-thompson/watchpost"

// trees are the packages loaded, with their tests.
var trees = []string{"./modes/tty/...", "./app/..."}

// use is one identifier that resolves to an object the module declares.
type use struct {
	file string // the using file, relative to the root
	test bool
	obj  types.Object
	decl string // the object's declaring file
	encl string // the enclosing top-level function, "Type.Name" for a method; "" outside one
}

// testFunc is a function declared in a test file of package tty: its name,
// its file, and the module objects its body names.
type testFunc struct {
	name, file string
	uses       []types.Object
}

// index is the loaded trees: every use, every method declaration on a named
// type, and package tty's test functions.
type index struct {
	dir        string // the tree measured
	inputs     string // the root the committed inputs are read from
	uses       []use
	methodDecl map[string]map[string]int // "pkgpath.Type" -> file -> methods declared on it
	fieldOwner map[*types.Var]string     // a struct field -> its struct's name, package tty and app
	testFuncs  map[types.Object]*testFunc
	tests      []*testFunc // package tty's Test functions
}

// load type-checks the two trees with their tests and indexes them. A file
// is read once, from the first package that compiles it: a package and its
// test variant share their files.
func load(dir string) (*index, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	cfg := &packages.Config{Dir: abs, Tests: true, Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo}
	pkgs, err := packages.Load(cfg, trees...)
	if err != nil {
		return nil, err
	}
	ix := &index{dir: abs, methodDecl: map[string]map[string]int{}, fieldOwner: map[*types.Var]string{}, testFuncs: map[types.Object]*testFunc{}}
	seen := map[string]bool{}
	for _, p := range pkgs { // bounded by the packages (P10-02)
		if len(p.Errors) > 0 {
			return nil, fmt.Errorf("%s: %v", p.ID, p.Errors[0])
		}
		ix.owners(p)
		for _, f := range p.Syntax { // bounded by the package (P10-02)
			name := ix.rel(p.Fset.Position(f.Pos()).Filename)
			if seen[name] {
				continue
			}
			seen[name] = true
			ix.file(p, f, name)
		}
	}
	if len(ix.uses) == 0 || len(ix.tests) == 0 {
		return nil, fmt.Errorf("the trees under %s indexed no use or no test: the load is broken", abs)
	}
	return ix, nil
}

// rel is a path relative to the root, slash-separated.
func (ix *index) rel(path string) string {
	r, err := filepath.Rel(ix.dir, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}

// owners records the struct each field of a package-level struct type
// belongs to.
func (ix *index) owners(p *packages.Package) {
	if p.Types == nil {
		return
	}
	sc := p.Types.Scope()
	for _, n := range sc.Names() { // bounded by the package's scope (P10-02)
		tn, ok := sc.Lookup(n).(*types.TypeName)
		if !ok {
			continue
		}
		if st, ok := tn.Type().Underlying().(*types.Struct); ok {
			for i := range st.NumFields() { // bounded by the struct (P10-02)
				ix.fieldOwner[st.Field(i)] = n
			}
		}
	}
}

// file indexes one file: its uses, its method declarations, and, in a test
// file of package tty, its functions.
func (ix *index) file(p *packages.Package, f *ast.File, name string) {
	test := strings.HasSuffix(name, "_test.go")
	for _, d := range f.Decls { // bounded by the file (P10-02)
		encl := ""
		fd, isFunc := d.(*ast.FuncDecl)
		fn, _ := p.TypesInfo.Defs[nameOf(fd)].(*types.Func)
		if isFunc && fn != nil {
			encl = ix.symName(fn)
		}
		ast.Inspect(d, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if o := p.TypesInfo.Uses[id]; o != nil && o.Pkg() != nil && strings.HasPrefix(o.Pkg().Path(), module) {
					ix.uses = append(ix.uses, use{file: name, test: test, obj: o, decl: ix.rel(p.Fset.Position(o.Pos()).Filename), encl: encl})
				}
			}
			return true
		})
		if !isFunc {
			continue
		}
		if recv := recvOf(fn); fn != nil && recv != "" && !test {
			recv = p.PkgPath + "." + recv
			if ix.methodDecl[recv] == nil {
				ix.methodDecl[recv] = map[string]int{}
			}
			ix.methodDecl[recv][name]++
		}
		if test && p.Types.Name() == "tty" && fd.Recv == nil && fd.Body != nil {
			ix.testFunc(p, fd, name)
		}
	}
}

// testFunc indexes one function of a tty test file.
func (ix *index) testFunc(p *packages.Package, fd *ast.FuncDecl, name string) {
	tf := &testFunc{name: fd.Name.Name, file: name}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			if o := p.TypesInfo.Uses[id]; o != nil && o.Pkg() == p.Types {
				tf.uses = append(tf.uses, o)
			}
		}
		return true
	})
	if o := p.TypesInfo.Defs[fd.Name]; o != nil {
		ix.testFuncs[o] = tf
	}
	if strings.HasPrefix(fd.Name.Name, "Test") {
		ix.tests = append(ix.tests, tf)
	}
}

// nameOf is a function declaration's name, or nil for any other
// declaration.
func nameOf(fd *ast.FuncDecl) *ast.Ident {
	if fd == nil {
		return nil
	}
	return fd.Name
}

// symName is an object's name as the design documents write it: a method or
// a field qualified by its type, anything else bare.
func (ix *index) symName(o types.Object) string {
	if f, ok := o.(*types.Func); ok {
		if recv := recvOf(f); recv != "" {
			return recv + "." + f.Name()
		}
	}
	if v, ok := o.(*types.Var); ok && v.IsField() {
		return ix.fieldOwner[v] + "." + v.Name()
	}
	return o.Name()
}

// ownerOf is the type a method or field belongs to, or "".
func (ix *index) ownerOf(o types.Object) string {
	if f, ok := o.(*types.Func); ok {
		return recvOf(f)
	}
	if v, ok := o.(*types.Var); ok && v.IsField() {
		return ix.fieldOwner[v]
	}
	return ""
}

// recvOf is a method's receiver type name, or "" for a function or for nil.
func recvOf(f *types.Func) string {
	if f == nil {
		return ""
	}
	sig, ok := f.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return ""
	}
	t := sig.Recv().Type()
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	if n, ok := t.(*types.Named); ok {
		return n.Obj().Name()
	}
	return ""
}
