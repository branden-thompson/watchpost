// gcprobe holds what the compiler decides about the two trees 0.19.6 splits
// (0.19.6 B0.7, PB-2, PB-3, A-26, D-28): every value that escapes to the heap,
// by the function it is in and its type, and every inlined call, by its caller
// and callee. A move that boxes a model behind an interface, or turns a method
// into a call through a func field, changes what the compiler does and nothing
// a test asserts: this is where it shows.
//
// WHAT IT READS. `go build -gcflags=-m` over the packages the binary links
// under ./modes/tty/... and ./app/... (test support is not linked, so it is
// not judged), each escape keyed by the package, the enclosing function and the
// escaping value's type, typed with go/types over the build's export data, and
// each "inlining call to" line keyed by package, caller and callee. A position
// in another package's file, a body inlined from there, is keyed by that
// file's function and the compiler's own text for the value.
//
// WHAT IT HOLDS, against the baselines recorded on `e57afeaf`:
//   - no escape key counts more than its baseline, and no type escapes more
//     often across the trees (the backstop, 0.19.6 RR-12): a new escape fails;
//   - no inlining key counts fewer, and the trees' total of inlined calls is
//     no lower: a lost inlining fails, and stands only by a HUM LEAD ruling.
//
// A move renames keys, so `-map` names a file of "old<TAB>new" lines, each
// old name replaced by the new in every baseline key's fields before the
// comparison: a function re-homed onto a model, a package moved. A name is
// replaced only where it stands whole, so `tick` renames the function and
// not `tickerRow`; a line's optional third field names the one package whose
// keys it renames, so modes/tty's `tick` leaves app's `schedule.tick`
// (0.19.6 A-49). Mapping many
// old names to one new one counts the union, which is 0.19.6 A-26's rule for a split
// function: its entry points' inlined calls are matched by callee across the
// union of the old function and the functions carved from it.
//
// A key no rename can carry, because the code it names was carved into new
// functions or replaced by a field read, is a line of `-excused`, the
// reviewed list (0.19.6 D-82): the key, the count it may reach, its class
// ("moved" or "seam") and why. Anything else still fails, and so does a line
// no key needs any more.
//
// A package move is `-moves`, a file of "from<TAB>to" lines applied after the
// renames (0.19.6 A-52). The names that moved are read from to's source, so
// the map names two packages and nothing that could drift: a moved function's
// keys become to's, from's calls to it read as the compiler writes a call into
// another package (frame.Plural, frame.(*MastheadSlot).Get), a third package's
// keys name it through to (console.BedMsg for tty.BedMsg), and a key no
// function owns, a compiler wrapper, a body from another module's file or a
// package-scope initializer, is counted across both packages on either side
// (0.19.6 A-60).
//
// usage:
//
//	go run ./tools/gcprobe record [-dir <tree>] [-tree <commit>] -escapes <file> -inlining <file>
//	go run ./tools/gcprobe compare [-dir <tree>] -escapes <file> -inlining <file> [-map <file>] [-moves <file>] [-excused <file>]
//
// The baselines name the platform and Go they were recorded under; the
// compiler's decisions are both's, so on another the comparison is refused,
// with the reason, and exits 0 on another platform and 1 under another Go.
// A go whose standard library is outside the module cache's toolchain is
// refused too, recording or comparing, and exits 1: its keys name the library
// by base name, not std/ (0.19.6 D-75).
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// module is the import path the keys strip, so a key reads as a repository
// path.
const module = "github.com/branden-thompson/watchpost/"

// maxLines bounds the build's output (9,950 lines for modes/tty alone at
// e57afeaf) and maxPackages the listing (a few hundred with every dependency).
const (
	maxLines    = 1 << 22
	maxPackages = 1 << 14
)

// trees are the packages probed, less those the binary does not link.
var trees = []string{"./modes/tty/...", "./app/..."}

// binary is the command whose links decide what is judged.
const binary = "./cmd/watchpost"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main without the process. A verdict that could not be written
// fails.
func run(args []string, stdoutW, stderrW io.Writer) int {
	stdout, stderr := &out{w: stdoutW}, &out{w: stderrW}
	code := probeRun(args, stdout, stderr)
	if stdout.err != nil || stderr.err != nil {
		return max(code, 1)
	}
	return code
}

// out is a writer that keeps its first error, so a run whose verdict could
// not be written is not taken for one that passed.
type out struct {
	w   io.Writer
	err error
}

// say writes one formatted line, once nothing before it failed.
func (o *out) say(format string, a ...any) {
	if o.err == nil {
		_, o.err = fmt.Fprintf(o.w, format+"\n", a...)
	}
}

// probeRun is run's work, writing through the two outs.
func probeRun(args []string, stdout, stderr *out) int {
	if len(args) == 0 || (args[0] != "record" && args[0] != "compare") {
		stderr.say("gcprobe: say record or compare")
		return 2
	}
	fs := flag.NewFlagSet("gcprobe "+args[0], flag.ContinueOnError)
	fs.SetOutput(stderr.w)
	dir := fs.String("dir", ".", "the tree to probe, its module root")
	escapes := fs.String("escapes", "", "the escapes baseline")
	inlining := fs.String("inlining", "", "the inlining baseline")
	mapFile := fs.String("map", "", "compare: a file of old<TAB>new renames applied to the baselines' keys")
	movesFile := fs.String("moves", "", "compare: a file of from<TAB>to package moves applied to the keys")
	excusedFile := fs.String("excused", "", "compare: the reviewed list of keys no rename carries (0.19.6 D-82)")
	tree := fs.String("tree", "", "record: the commit the baselines name; the tree's HEAD when empty")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *escapes == "" || *inlining == "" {
		stderr.say("gcprobe: say -escapes <file> and -inlining <file>")
		return 2
	}
	if args[0] == "record" {
		return record(*dir, *tree, *escapes, *inlining, stdout, stderr)
	}
	return compare(*dir, *escapes, *inlining, *mapFile, *movesFile, *excusedFile, stdout, stderr)
}

// ---- the probe ----------------------------------------------------------------

// counts is a multiset of keys, each "pkg<TAB>function<TAB>type" or
// "pkg<TAB>caller<TAB>callee".
type counts map[string]int

// probe is the compiler's decisions over the trees under dir.
type probe struct {
	escapes, inlining counts
}

// listed is the part of `go list -json -export` the probe reads.
type listed struct {
	ImportPath, Dir, Export string
	GoFiles                 []string
	DepOnly                 bool
}

// probeTree builds the trees with -m and keys every escape and inlined call.
func probeTree(dir string) (*probe, error) {
	linked, err := linkedPackages(dir)
	if err != nil {
		return nil, err
	}
	pkgs, err := listPackages(dir)
	if err != nil {
		return nil, err
	}
	out, err := goCmd(dir, append([]string{"build", "-gcflags=-m"}, trees...)...)
	if err != nil {
		return nil, fmt.Errorf("go build -gcflags=-m: %v: %.400s", err, out)
	}
	p := &probe{escapes: counts{}, inlining: counts{}}
	r := newResolver(dir, pkgs)
	pkg := ""
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for range maxLines {
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		if strings.HasPrefix(line, "# ") {
			pkg = strings.TrimPrefix(line, "# ")
			continue
		}
		if linked[pkg] {
			p.take(r, pkg, line)
		}
	}
	if err := invariant.Check(len(p.escapes) > 0 && len(p.inlining) > 0, "the build reported no escape or no inlined call: the probe read nothing"); err != nil {
		return nil, err
	}
	return p, nil
}

// take keys one line of the compiler's output.
func (p *probe) take(r *resolver, pkg, line string) {
	file, lineNo, col, msg, ok := splitDiagnostic(line)
	if !ok {
		return
	}
	short := strings.TrimPrefix(pkg, module)
	switch {
	case strings.HasPrefix(msg, "inlining call to "):
		p.inlining[short+"\t"+r.function(pkg, file, lineNo, col)+"\t"+strings.TrimPrefix(msg, "inlining call to ")]++
	case strings.HasPrefix(msg, "moved to heap: "), strings.HasSuffix(msg, " escapes to heap"):
		p.escapes[short+"\t"+r.function(pkg, file, lineNo, col)+"\t"+r.typeOf(pkg, file, lineNo, col, msg)]++
	}
}

// splitDiagnostic reads "file:line:col: message", and "<autogenerated>:line:
// message", which has no column.
func splitDiagnostic(line string) (file string, lineNo, col int, msg string, ok bool) {
	parts := strings.SplitN(line, ": ", 2)
	if len(parts) != 2 {
		return "", 0, 0, "", false
	}
	pos, msg := strings.Split(parts[0], ":"), parts[1]
	switch len(pos) {
	case 3:
		lineNo, _ = strconv.Atoi(pos[1])
		col, _ = strconv.Atoi(pos[2])
		return pos[0], lineNo, col, msg, lineNo > 0
	case 2:
		lineNo, _ = strconv.Atoi(pos[1])
		return pos[0], lineNo, 0, msg, pos[0] == "<autogenerated>"
	}
	return "", 0, 0, "", false
}

// linkedPackages is the import paths the binary links.
func linkedPackages(dir string) (map[string]bool, error) {
	out, err := goCmd(dir, "list", "-deps", binary)
	if err != nil {
		return nil, fmt.Errorf("go list -deps %s: %v: %.400s", binary, err, out)
	}
	linked := map[string]bool{}
	for _, p := range strings.Fields(out) { // bounded by the binary's packages (P10-02)
		linked[p] = true
	}
	listed := linked[module+strings.TrimPrefix(binary, "./")]
	if err := invariant.Check(listed, "the binary is not among its own packages: the listing is broken"); err != nil {
		return nil, err
	}
	return linked, nil
}

// listPackages is the trees' packages and every dependency, with the export
// data the type checker reads.
func listPackages(dir string) (map[string]listed, error) {
	out, err := goCmd(dir, append([]string{"list", "-e", "-export", "-deps", "-json"}, trees...)...)
	if err != nil {
		return nil, fmt.Errorf("go list -export: %v: %.400s", err, out)
	}
	pkgs := map[string]listed{}
	dec := json.NewDecoder(strings.NewReader(out))
	for range maxPackages {
		if !dec.More() {
			break
		}
		var p listed
		if err := dec.Decode(&p); err != nil {
			return nil, err
		}
		pkgs[p.ImportPath] = p
	}
	return pkgs, nil
}

// goCmd runs go in dir and returns what it printed, both streams.
func goCmd(dir string, args ...string) (string, error) { return command(dir, "go", args...) }

// command runs a program in dir and returns what it printed, both streams.
func command(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	return string(b), err
}

// ---- resolving a position -----------------------------------------------------

// resolver finds a position's enclosing function and an escaping value's type,
// type-checking each package the first time one of its files is asked about.
type resolver struct {
	dir     string
	fset    *token.FileSet
	pkgs    map[string]listed
	byFile  map[string]string // a file's absolute path -> its package's import path
	checked map[string]*checkedPkg
	imp     types.Importer
}

// checkedPkg is one package's parsed files and its types.
type checkedPkg struct {
	files map[string]*ast.File // by absolute path
	info  *types.Info
	pkg   *types.Package
	src   map[string][]byte
}

func newResolver(dir string, pkgs map[string]listed) *resolver {
	// The compiler names a file relative to dir and go list names packages'
	// directories absolutely, so dir is made absolute before the two meet.
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	r := &resolver{dir: dir, fset: token.NewFileSet(), pkgs: pkgs, byFile: map[string]string{}, checked: map[string]*checkedPkg{}}
	for _, p := range pkgs { // bounded by the packages (P10-02)
		for _, f := range p.GoFiles { // bounded by the package (P10-02)
			r.byFile[filepath.Join(p.Dir, f)] = p.ImportPath
		}
	}
	r.imp = importer.ForCompiler(r.fset, "gc", func(path string) (io.ReadCloser, error) {
		p, ok := pkgs[path]
		if !ok || p.Export == "" {
			return nil, errors.New("no export data for " + path)
		}
		return os.Open(p.Export)
	})
	return r
}

// at is the package and file a diagnostic's path names, or nil for a file
// outside the listed packages (a dependency outside the module, a wrapper).
func (r *resolver) at(file string) (*checkedPkg, *ast.File) {
	abs := file
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(r.dir, file)
	}
	abs = filepath.Clean(abs)
	path, ok := r.byFile[abs]
	if !ok || !strings.HasPrefix(path, module) {
		return nil, nil
	}
	c := r.check(path)
	if c == nil {
		return nil, nil
	}
	return c, c.files[abs]
}

// check type-checks one package, once. A package that does not check is nil:
// its keys fall back to the compiler's text.
func (r *resolver) check(path string) *checkedPkg {
	if c, done := r.checked[path]; done {
		return c
	}
	r.checked[path] = nil
	p := r.pkgs[path]
	c := &checkedPkg{files: map[string]*ast.File{}, src: map[string][]byte{}}
	var files []*ast.File
	for _, name := range p.GoFiles { // bounded by the package (P10-02)
		abs := filepath.Join(p.Dir, name)
		src, err := os.ReadFile(abs)
		if err != nil {
			return nil
		}
		f, err := parser.ParseFile(r.fset, abs, src, parser.SkipObjectResolution)
		if err != nil {
			return nil
		}
		c.files[abs], c.src[abs] = f, src
		files = append(files, f)
	}
	c.info = &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: r.imp, Error: func(error) {}}
	c.pkg, _ = conf.Check(path, r.fset, files, c.info)
	r.checked[path] = c
	return c
}

// function is the enclosing function of a position: "Recv.Name" for a
// method, "Name" for a function, "package scope" outside one; for a file
// outside the module, its path below the module cache.
func (r *resolver) function(pkg, file string, line, col int) string {
	if file == "<autogenerated>" {
		return "<autogenerated>"
	}
	_, f := r.at(file)
	if f == nil {
		return external(file)
	}
	pos := r.offset(f, line, col)
	for _, d := range f.Decls { // bounded by the file (P10-02)
		fn, ok := d.(*ast.FuncDecl)
		if !ok || pos < fn.Pos() || pos >= fn.End() {
			continue
		}
		if fn.Recv != nil && len(fn.Recv.List) == 1 {
			return receiverName(fn.Recv.List[0].Type) + "." + fn.Name.Name
		}
		return fn.Name.Name
	}
	return "package scope"
}

// external is a file outside the module, by its path from the module cache's
// root with the version dropped, so the key reads the same on every machine
// and through a dependency's upgrade: the standard library as std/<path>, a
// module by its path. A version stays out of the key for a second reason: the
// toolchain's directory, its name, an at sign and its version, is what the
// published tree's privacy check takes for an email address.
func external(file string) string {
	f := filepath.ToSlash(file)
	i := strings.Index(f, "/pkg/mod/")
	if i < 0 {
		return filepath.Base(file)
	}
	f = f[i+len("/pkg/mod/"):]
	if strings.HasPrefix(f, "golang.org/toolchain@") {
		if _, std, ok := strings.Cut(f, "/src/"); ok {
			return "std/" + std
		}
	}
	return version.ReplaceAllString(f, "")
}

// version is a module path's "@<version>" element.
var version = regexp.MustCompile(`@[^/]+`)

// receiverName is a receiver's type name, pointer and type parameters
// dropped.
func receiverName(e ast.Expr) string {
	for range 4 { // a pointer to a generic type is three deep (P10-02)
		switch t := e.(type) {
		case *ast.StarExpr:
			e = t.X
		case *ast.IndexExpr:
			e = t.X
		case *ast.IndexListExpr:
			e = t.X
		case *ast.Ident:
			return t.Name
		default:
			return "?"
		}
	}
	return "?"
}

// offset is a line and column as a position in f.
func (r *resolver) offset(f *ast.File, line, col int) token.Pos {
	tf := r.fset.File(f.Pos())
	if line > tf.LineCount() {
		return token.NoPos
	}
	return tf.LineStart(line) + token.Pos(max(col-1, 0))
}

// typeOf is the escaping value's type: the type go/types gives the
// expression the compiler names at the position, or, where none resolves,
// the compiler's own text for it.
func (r *resolver) typeOf(pkg, file string, line, col int, msg string) string {
	text := strings.TrimSuffix(strings.TrimPrefix(msg, "moved to heap: "), " escapes to heap")
	c, f := r.at(file)
	if c == nil || f == nil || c.pkg == nil {
		return "~" + text
	}
	// THE COMPILER'S POSITION IS NOT WHERE THE EXPRESSION STARTS: a binary
	// expression is placed at its operator, a call at its bracket, a literal at
	// its brace. So every expression whose span holds the position is a
	// candidate: the innermost whose text is the compiler's wins, and where the
	// compiler printed a rewritten form (a method as a function, an inlined
	// result as ~r0) no text matches, and the innermost the position anchors
	// stands in.
	pos := r.offset(f, line, col)
	var found, anchored ast.Expr
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil || pos < n.Pos() || pos > n.End() {
			return false
		}
		e, ok := n.(ast.Expr)
		if id, isID := e.(*ast.Ident); isID {
			if _, builtin := c.info.Uses[id].(*types.Builtin); builtin {
				return true // "append escapes" is the call's result, not the builtin's name
			}
		}
		switch {
		case ok && r.matches(c, f, e, text):
			found = e
		case ok && anchor(e) == pos:
			anchored = e
		}
		return true
	})
	if found == nil {
		found = anchored
	}
	if found == nil {
		return "~" + text
	}
	if _, ok := found.(*ast.FuncLit); ok {
		return "func literal"
	}
	t := c.info.TypeOf(found)
	if id, ok := found.(*ast.Ident); ok {
		if v, isVar := c.info.ObjectOf(id).(*types.Var); isVar {
			t = v.Type() // a variable moved to the heap is named where it is declared
		}
	}
	if t == nil || t == types.Typ[types.Invalid] {
		return "~" + text // a package name, a label: nothing the heap holds
	}
	return types.TypeString(t, func(p *types.Package) string {
		if p == c.pkg {
			return ""
		}
		return p.Name()
	})
}

// anchor is where the compiler places an expression: a binary or unary
// expression at its operator, a call at its opening bracket, a composite
// literal at its brace, an index or slice at its bracket, anything else at its
// start.
func anchor(e ast.Expr) token.Pos {
	switch x := e.(type) {
	case *ast.BinaryExpr:
		return x.OpPos
	case *ast.CallExpr:
		return x.Lparen
	case *ast.CompositeLit:
		return x.Lbrace
	case *ast.IndexExpr:
		return x.Lbrack
	case *ast.SliceExpr:
		return x.Lbrack
	case *ast.SelectorExpr:
		return x.Sel.Pos()
	}
	return e.Pos()
}

// matches says whether e, which holds the compiler's position, is the
// value it names: the same source text, or, for the forms the compiler
// abbreviates, the same opening.
func (r *resolver) matches(c *checkedPkg, f *ast.File, e ast.Expr, text string) bool {
	src := c.src[r.fset.File(f.Pos()).Name()]
	tf := r.fset.File(f.Pos())
	lo, hi := tf.Offset(e.Pos()), tf.Offset(e.End())
	if lo < 0 || hi > len(src) || lo >= hi {
		return false
	}
	got := strings.Join(strings.Fields(string(src[lo:hi])), " ")
	if got == text {
		return true
	}
	switch e.(type) {
	case *ast.FuncLit:
		return text == "func literal"
	case *ast.CompositeLit, *ast.UnaryExpr, *ast.CallExpr:
		for _, cut := range []string{"{...}", "(...)"} { // the compiler's two abbreviations (P10-02)
			if open, ok := strings.CutSuffix(text, cut); ok && strings.HasPrefix(got, open) {
				return true
			}
		}
		// A builtin's call is named by the builtin alone: "append escapes".
		return strings.HasPrefix(got, text+"(")
	}
	return false
}

// ---- the baselines ------------------------------------------------------------

// header is a baseline's first lines: what it was recorded on.
type header struct{ platform, goVer, tree string }

// writeCounts writes a baseline: the header, then one "key<TAB>count" row a
// key, in order.
func writeCounts(path, what string, h header, c counts) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# gcprobe: %s (0.19.6 B0.7), read by tools/gcprobe\n# platform\t%s\n# go\t%s\n# tree\t%s\n", what, h.platform, h.goVer, h.tree)
	for _, k := range slices.Sorted(maps.Keys(c)) { // bounded by the keys (P10-02)
		fmt.Fprintf(&b, "%s\t%d\n", k, c[k])
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// readCounts reads a baseline.
func readCounts(path string) (header, counts, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return header{}, nil, err
	}
	var h header
	c := counts{}
	for _, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") { // bounded by the file (P10-02)
		f := strings.Split(line, "\t")
		switch {
		case len(f) == 2 && f[0] == "# platform":
			h.platform = f[1]
		case len(f) == 2 && f[0] == "# go":
			h.goVer = f[1]
		case len(f) == 2 && f[0] == "# tree":
			h.tree = f[1]
		case len(f) == 4:
			n, err := strconv.Atoi(f[3])
			if err != nil {
				return h, nil, fmt.Errorf("%s: %q: %v", path, line, err)
			}
			c[strings.Join(f[:3], "\t")] = n
		}
	}
	complete := h.platform != "" && h.goVer != "" && len(c) > 0
	if err := invariant.Check(complete, path+" names no platform, go or key: it is not a gcprobe baseline"); err != nil {
		return h, nil, err
	}
	return h, c, nil
}

// record probes the tree and writes both baselines.
func record(dir, tree, escapes, inlining string, stdout, stderr *out) int {
	if err := stdlibKeyed(dir); err != nil {
		stderr.say("gcprobe: %v", err)
		return 1
	}
	p, err := probeTree(dir)
	if err != nil {
		stderr.say("gcprobe: %v", err)
		return 1
	}
	if tree == "" {
		head, err := command(dir, "git", "rev-parse", "--short", "HEAD")
		if err != nil {
			stderr.say("gcprobe: git rev-parse: %v: %s; say -tree", err, head)
			return 1
		}
		tree = head
	}
	h := header{platform: platform(), goVer: runtime.Version(), tree: strings.TrimSpace(tree)}
	if err := writeCounts(escapes, "every escape to the heap, by package, function and type", h, p.escapes); err != nil {
		stderr.say("gcprobe: %v", err)
		return 1
	}
	if err := writeCounts(inlining, "every inlined call, by package, caller and callee", h, p.inlining); err != nil {
		stderr.say("gcprobe: %v", err)
		return 1
	}
	stdout.say("gcprobe: recorded %d escapes in %d keys and %d inlined calls in %d keys on %s", total(p.escapes), len(p.escapes), total(p.inlining), len(p.inlining), h.tree)
	return 0
}

// stdlibKeyed refuses a go in dir whose standard library the keys would not
// name as the baselines do (0.19.6 D-75).
func stdlibKeyed(dir string) error {
	root, err := goCmd(dir, "env", "GOROOT")
	if err != nil {
		return fmt.Errorf("go env GOROOT: %v: %s", err, root)
	}
	return keyedRoot(strings.TrimSpace(root))
}

// keyedRoot is whether a GOROOT's standard library is keyed std/<path>, which
// external does only under the module cache's toolchain. A GOROOT elsewhere,
// of the same version or not, has its files keyed by base name, so every
// generic the trees instantiate from it reads as new and lost: 154 keys under
// a system go 1.27.4 (0.19.6 D-75).
func keyedRoot(root string) error {
	if !strings.HasPrefix(external(filepath.Join(root, "src", "slices", "slices.go")), "std/") {
		return fmt.Errorf("the go on the path keeps its standard library at %s, outside the module cache's toolchain, where the baselines key it as std/: put that toolchain first on the path", root)
	}
	return nil
}

// platform is the system the probe runs on.
func platform() string { return runtime.GOOS + "/" + runtime.GOARCH }

// ---- the comparison -----------------------------------------------------------

// compare probes the tree and holds it to the baselines.
func compare(dir, escapes, inlining, mapFile, movesFile, excusedFile string, stdout, stderr *out) int {
	he, be, err := readCounts(escapes)
	if err != nil {
		stderr.say("gcprobe: %v", err)
		return 2
	}
	_, bi, err := readCounts(inlining)
	if err != nil {
		stderr.say("gcprobe: %v", err)
		return 2
	}
	if he.platform != platform() {
		stdout.say("gcprobe: the baselines are %s's; the compiler decides by system, so nothing is compared on %s", he.platform, platform())
		return 0
	}
	if he.goVer != runtime.Version() {
		stdout.say("gcprobe: FAIL the baselines were recorded under %s and this is %s: record them again on the commit that moves the toolchain", he.goVer, runtime.Version())
		return 1
	}
	renames, err := readMap(mapFile)
	if err != nil {
		stderr.say("gcprobe: %v", err)
		return 2
	}
	moves, err := readMoves(dir, movesFile)
	if err != nil {
		stderr.say("gcprobe: %v", err)
		return 2
	}
	excused, err := readExcused(excusedFile)
	if err != nil {
		stderr.say("gcprobe: %v", err)
		return 2
	}
	if err := stdlibKeyed(dir); err != nil {
		stdout.say("gcprobe: FAIL %v", err)
		return 1
	}
	p, err := probeTree(dir)
	if err != nil {
		stderr.say("gcprobe: %v", err)
		return 1
	}
	now := &probe{escapes: moved(p.escapes, moves, false), inlining: moved(p.inlining, moves, false)}
	failures := judge(moved(renamed(be, renames), moves, true), moved(renamed(bi, renames), moves, true), now, excused)
	for _, f := range failures { // bounded by the verdict (P10-02)
		stdout.say("gcprobe: FAIL %s", f)
	}
	if len(failures) > 0 {
		return 1
	}
	stdout.say("gcprobe: OK - %d escapes in %d keys, none new; %d inlined calls in %d keys, none lost (baselines on %s; %d keys excused)", total(p.escapes), len(p.escapes), total(p.inlining), len(p.inlining), he.tree, len(excused))
	return 0
}

// judge is every way the probe breaks the baselines.
func judge(baseEsc, baseInl counts, p *probe, excused []excuse) []string {
	allowed := map[string]map[string]int{"escape": {}, "type": {}, "inlining": {}}
	for _, e := range excused { // bounded by the list (P10-02)
		allowed[e.kind][e.key] = e.count
	}
	var out []string
	for _, k := range slices.Sorted(maps.Keys(p.escapes)) { // bounded by the keys (P10-02)
		if n, was := p.escapes[k], baseEsc[k]; n > ceiling(allowed["escape"], k, was) {
			out = append(out, fmt.Sprintf("escape %s: %d, against %d at baseline: a value now escapes that did not", strings.ReplaceAll(k, "\t", " | "), n, was))
		}
	}
	nowType, wasType := byType(p.escapes), byType(baseEsc)
	for _, t := range slices.Sorted(maps.Keys(nowType)) { // bounded by the types (P10-02)
		if nowType[t] > ceiling(allowed["type"], t, wasType[t]) {
			out = append(out, fmt.Sprintf("type %s escapes %d times across the trees, against %d at baseline", t, nowType[t], wasType[t]))
		}
	}
	for _, k := range slices.Sorted(maps.Keys(baseInl)) { // bounded by the keys (P10-02)
		floor, ok := allowed["inlining"][k]
		if !ok {
			floor = baseInl[k]
		}
		if n, was := p.inlining[k], baseInl[k]; n < floor {
			out = append(out, fmt.Sprintf("inlining %s: %d, against %d at baseline: a lost inlining stands only by a HUM LEAD ruling (A-26)", strings.ReplaceAll(k, "\t", " | "), n, was))
		}
	}
	out = append(out, stale(excused, baseEsc, baseInl, p, nowType, wasType)...)
	if now, was := total(p.inlining), total(baseInl); now < was {
		out = append(out, fmt.Sprintf("the trees inline %d calls, against %d at baseline", now, was))
	}
	return out
}

// ceiling is the most a key may reach: its excused count, else its baseline.
func ceiling(allowed map[string]int, k string, was int) int {
	if n, ok := allowed[k]; ok {
		return n
	}
	return was
}

// stale is every excuse no key needs: its key now within its baseline, so the
// line would hide whatever came to use it next.
func stale(excused []excuse, baseEsc, baseInl counts, p *probe, nowType, wasType counts) []string {
	var out []string
	for _, e := range excused { // bounded by the list (P10-02)
		var needed bool
		switch e.kind {
		case "escape":
			needed = p.escapes[e.key] > baseEsc[e.key]
		case "type":
			needed = nowType[e.key] > wasType[e.key]
		case "inlining":
			needed = p.inlining[e.key] < baseInl[e.key]
		}
		if !needed {
			out = append(out, fmt.Sprintf("excused %s %s needs no excuse: the key is within its baseline, so remove the line (0.19.6 D-82)", e.kind, strings.ReplaceAll(e.key, "\t", " | ")))
		}
	}
	return out
}

// excuse is one line of the reviewed list (0.19.6 D-82): a key no rename can
// carry, the count it may reach - an escape's or a type's most, an inlined
// call's least - and its class.
type excuse struct {
	kind, key string
	count     int
	class     string
}

// excuseClasses are the two kinds of key 0.19.6 D-82 lets the list hold: code carved
// into new functions or under a renamed type, and calls the map's seam made
// field reads or asks the Dashboard applies.
var excuseClasses = map[string]bool{"moved": true, "seam": true}

// readExcused reads the reviewed list: "escape|inlining<TAB>pkg<TAB>function
// <TAB>type-or-callee<TAB>count<TAB>class<TAB>why" or "type<TAB>type<TAB>count
// <TAB>class<TAB>why"; "" is no file and no excuse.
func readExcused(path string) ([]excuse, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []excuse
	seen := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the file (P10-02)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		var e excuse
		var count string
		switch {
		case len(f) == 7 && (f[0] == "escape" || f[0] == "inlining"):
			e.kind, e.key, count, e.class = f[0], strings.Join(f[1:4], "\t"), f[4], f[5]
		case len(f) == 5 && f[0] == "type":
			e.kind, e.key, count, e.class = f[0], f[1], f[2], f[3]
		default:
			return nil, fmt.Errorf("%s: %q is not an excuse line", path, line)
		}
		n, err := strconv.Atoi(count)
		if err != nil || !excuseClasses[e.class] || strings.TrimSpace(f[len(f)-1]) == "" {
			return nil, fmt.Errorf("%s: %q needs a count, a class (moved or seam) and why", path, line)
		}
		if err := invariant.Check(!seen[e.kind+"\t"+e.key], path+": "+line+" excuses a key a line above it excuses: the later count would replace the earlier unseen"); err != nil {
			return nil, err
		}
		seen[e.kind+"\t"+e.key] = true
		e.count = n
		out = append(out, e)
	}
	return out, nil
}

// byType is the escapes' counts by type alone.
func byType(c counts) counts {
	out := counts{}
	for k, n := range c { // bounded by the keys (P10-02)
		out[k[strings.LastIndex(k, "\t")+1:]] += n
	}
	return out
}

// readMap reads "old<TAB>new" lines; "" is no file and no renames.
func readMap(path string) ([]rename, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []rename
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the file (P10-02)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 2 || len(f) > 3 || f[0] == "" {
			return nil, fmt.Errorf("%s: %q is not old<TAB>new, or old<TAB>new<TAB>package", path, line)
		}
		r := rename{old: f[0], new: f[1]}
		if len(f) == 3 {
			r.pkg = f[2]
		}
		out = append(out, r)
	}
	return out, nil
}

// rename is one line of the map: old's whole name made new, in every key, or
// in the keys of pkg alone when it is named.
type rename struct{ old, new, pkg string }

// renamed is c with every rename applied to each key's fields; keys that
// become one are summed.
func renamed(c counts, renames []rename) counts {
	if len(renames) == 0 {
		return c
	}
	out := counts{}
	for k, n := range c { // bounded by the keys (P10-02)
		fields := strings.Split(k, "\t")
		for i := range fields { // the key's three fields (P10-02)
			for _, r := range renames { // bounded by the map (P10-02)
				if r.pkg == "" || r.pkg == fields[0] {
					fields[i] = replaceWhole(fields[i], r.old, r.new)
				}
			}
		}
		out[strings.Join(fields, "\t")] += n
	}
	return out
}

// replaceWhole is s with every occurrence of old that no identifier
// character touches on either side replaced by new: `tick` in `tick` and in
// `tick.func1`, not in `tickerRow` or `armtick`.
func replaceWhole(s, old, new string) string {
	var b strings.Builder
	for i := 0; i < len(s); { // advances by at least one byte a pass (P10-02)
		j := strings.Index(s[i:], old)
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		at, end := i+j, i+j+len(old)
		b.WriteString(s[i:at])
		if (at == 0 || !identByte(s[at-1])) && (end == len(s) || !identByte(s[end])) {
			b.WriteString(new)
		} else {
			b.WriteString(old)
		}
		i = end
	}
	return b.String()
}

// identByte reports whether c can be part of a Go identifier written in ASCII,
// which every name the compiler reports in these keys is.
func identByte(c byte) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// total is a multiset's size.
func total(c counts) int {
	n := 0
	for _, v := range c { // bounded by the keys (P10-02)
		n += v
	}
	return n
}

// ---- the move map -------------------------------------------------------------

// move is one line of the move map: declarations that left the package from
// for the package to, both module-relative paths. Which ones is read from
// to's source, never listed by hand, so the map cannot fall out of step with
// the code it describes.
type move struct {
	from, to string
	fromPkg  string          // from's package name, the qualifier a third package wrote
	pkg      string          // to's package name, the qualifier its callers write
	names    map[string]bool // every top-level name to's non-test files declare
}

// readMoves reads "from<TAB>to" lines and each destination's names; "" is no
// file and no moves.
func readMoves(dir, path string) ([]move, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []move
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the file (P10-02)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 2 || f[0] == "" || f[1] == "" || f[0] == f[1] {
			return nil, fmt.Errorf("%s: %q is not from<TAB>to, two packages", path, line)
		}
		pkg, names, err := declaredNames(filepath.Join(dir, f[1]))
		if err != nil {
			return nil, fmt.Errorf("%s: %q: %w", path, line, err)
		}
		fromPkg, err := packageName(filepath.Join(dir, f[0]))
		if err != nil {
			return nil, fmt.Errorf("%s: %q: %w", path, line, err)
		}
		out = append(out, move{from: f[0], to: f[1], fromPkg: fromPkg, pkg: pkg, names: names})
	}
	return out, nil
}

// declaredNames is a package's name and every top-level name its non-test
// files declare: its functions, types, constants and variables. A method goes
// with its type, and a closure with its function, by the name a key's function
// begins with. The package's name is its files', which must agree: it is the
// qualifier the compiler writes, and a directory's last element is that name
// only by convention.
func declaredNames(pkgDir string) (string, map[string]bool, error) {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return "", nil, err
	}
	pkg, names := "", map[string]bool{}
	for _, e := range entries { // bounded by the directory (P10-02)
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(pkgDir, e.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			return "", nil, err
		}
		if err := invariant.Check(pkg == "" || f.Name.Name == pkg, pkgDir+"'s files name two packages: "+pkg+" and "+f.Name.Name); err != nil {
			return "", nil, err
		}
		pkg = f.Name.Name
		for _, d := range f.Decls { // bounded by the file (P10-02)
			addDeclared(names, d)
		}
	}
	if err := invariant.Check(len(names) > 0, pkgDir+" declares nothing: a move's destination holds the code that moved"); err != nil {
		return "", nil, err
	}
	return pkg, names, nil
}

// packageName is the package a directory's Go files declare, read from the
// first one's clause: the qualifier a third package's keys wrote for it before
// the move. A directory with no Go file has no package to name.
func packageName(pkgDir string) (string, error) {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries { // bounded by the directory (P10-02)
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(pkgDir, e.Name()), nil, parser.PackageClauseOnly)
		if err != nil {
			return "", err
		}
		return f.Name.Name, nil
	}
	return "", fmt.Errorf("%s holds no Go file: a move's source names the package it left", pkgDir)
}

// addDeclared adds one top-level declaration's names; a method adds none.
func addDeclared(names map[string]bool, d ast.Decl) {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil {
			names[d.Name.Name] = true
		}
	case *ast.GenDecl:
		for _, sp := range d.Specs { // bounded by the declaration (P10-02)
			switch sp := sp.(type) {
			case *ast.TypeSpec:
				names[sp.Name.Name] = true
			case *ast.ValueSpec:
				for _, n := range sp.Names { // bounded by the spec (P10-02)
					names[n.Name] = true
				}
			}
		}
	}
}

// moved is c with every move applied, to a baseline's keys or to the tree's
// (0.19.6 A-52). In a baseline, a key of from's whose function is a moved
// name's becomes to's, as it reads once moved; any other key of from's names
// the moved declarations qualified, as from's code calls them; a key of a
// third package's names them through to, as that package's code now does. A
// key no function owns, the compiler's <autogenerated> wrappers, bodies
// inlined from another module's file and package-scope initializers, is
// counted across both packages, in the baseline and in the tree alike: a move
// takes some of them with it, and which ones no name says (0.19.6 A-60).
func moved(c counts, moves []move, baseline bool) counts {
	if len(moves) == 0 {
		return c
	}
	out := counts{}
	for k, n := range c { // bounded by the keys (P10-02)
		fields := strings.Split(k, "\t")
		for _, m := range moves { // bounded by the map (P10-02)
			fields = movedKey(fields, m, baseline)
		}
		out[strings.Join(fields, "\t")] += n
	}
	return out
}

// movedKey is one key's fields under one move.
func movedKey(fields []string, m move, baseline bool) []string {
	q := m.pkg
	switch {
	case unowned(fields[1]) && fields[0] == m.to:
		fields[0] = m.from
		fields[1], fields[2] = qualified(fields[1], m.names, q), qualified(fields[2], m.names, q)
	case baseline && fields[0] == m.from && !unowned(fields[1]) && m.names[leadingName(fields[1])]:
		fields[0] = m.to
	case baseline && fields[0] == m.from:
		fields[1], fields[2] = qualified(fields[1], m.names, q), qualified(fields[2], m.names, q)
	case baseline && fields[0] != m.to:
		fields[1], fields[2] = requalified(fields[1], m, q), requalified(fields[2], m, q)
	}
	return fields
}

// unowned reports whether a key's function field names no function of the
// package: a compiler wrapper, another module's file a body came from, or a
// package-level variable's initializer (0.19.6 A-60).
func unowned(fn string) bool {
	return fn == "<autogenerated>" || fn == "package scope" || strings.Contains(fn, "/")
}

// leadingName is the name a key's function begins with: a function's, or a
// method's type, with or without its pointer.
func leadingName(fn string) string {
	fn = strings.TrimPrefix(fn, "(*")
	return fn[:nameEnd(fn, 0)]
}

// qualified is s with each moved name that stands alone, after no dot,
// qualified as the compiler writes a name from another package: q.Name,
// *q.Name, and q.(*Name) for a method on its pointer.
func qualified(s string, names map[string]bool, q string) string {
	var b strings.Builder
	for i := 0; i < len(s); { // advances by at least one byte a pass (P10-02)
		star := strings.HasPrefix(s[i:], "(*")
		at := i
		if star {
			at = i + 2
		}
		end := nameEnd(s, at)
		free := i == 0 || (!identByte(s[i-1]) && s[i-1] != '.')
		if end > at && free && names[s[at:end]] {
			b.WriteString(q + "." + s[i:end])
			i = end
			continue
		}
		if end > at && !star {
			b.WriteString(s[i:end]) // a name, not moved or not free, whole
			i = end
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// requalified is s with each moved name a third package wrote through from,
// from.Name or from.(*Name), written through to instead, as the compiler
// writes it once the name has moved; a name that did not move keeps from.
func requalified(s string, m move, q string) string {
	var b strings.Builder
	lead := m.fromPkg + "."
	for i := 0; i < len(s); { // advances by at least one byte a pass (P10-02)
		free := i == 0 || (!identByte(s[i-1]) && s[i-1] != '.')
		if free && strings.HasPrefix(s[i:], lead) {
			at := i + len(lead)
			if strings.HasPrefix(s[at:], "(*") {
				at += 2
			}
			if end := nameEnd(s, at); end > at && m.names[s[at:end]] {
				b.WriteString(q + "." + s[i+len(lead):end])
				i = end
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// nameEnd is where the identifier starting at from ends: from itself when no
// identifier starts there.
func nameEnd(s string, from int) int {
	for i := from; i < len(s); i++ { // bounded by the string (P10-02)
		if !identByte(s[i]) {
			return i
		}
	}
	return len(s)
}
