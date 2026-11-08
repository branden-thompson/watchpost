package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// goIndex is every name the tree's Go code declares, tests included (the page
// names the tests to extend) and go-studs too (the page names its table API):
// each package name's directories, each directory's top-level names, each
// type's methods and fields, and every file by its path.
type goIndex struct {
	files   map[string]bool            // every file by its path from the root, and a .go file by its base name too
	types   map[string]bool            // every type name
	pkgDirs map[string][]string        // package name → its directories
	top     map[string]map[string]bool // directory → its funcs, types, consts and vars
	members map[string]map[string]bool // type name → its methods and fields, in any package
	any     map[string]bool            // every name above
}

// indexModule parses every Go file under root, skipping what is not the
// tree's code: mutated, built, and the agents' worktrees.
func indexModule(t *testing.T, root string) goIndex {
	t.Helper()
	ix := goIndex{files: map[string]bool{}, types: map[string]bool{}, pkgDirs: map[string][]string{}, top: map[string]map[string]bool{},
		members: map[string]map[string]bool{}, any: map[string]bool{}}
	member := func(typ, name string) {
		if ix.members[typ] == nil {
			ix.members[typ] = map[string]bool{}
		}
		ix.members[typ][name], ix.any[name] = true, true
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".claude", "mutants", "dist", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		ix.files[filepath.ToSlash(rel)] = true
		if !strings.HasSuffix(path, ".go") || strings.Contains(filepath.ToSlash(rel), "testdata/") {
			return nil // a golden or fixture can be named; testdata's Go is fixture, not the module's code
		}
		ix.files[filepath.Base(path)] = true
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		dir := filepath.ToSlash(filepath.Dir(rel))
		if ix.top[dir] == nil {
			ix.top[dir] = map[string]bool{}
		}
		if !strings.HasSuffix(f.Name.Name, "_test") && !slices.Contains(ix.pkgDirs[f.Name.Name], dir) {
			ix.pkgDirs[f.Name.Name] = append(ix.pkgDirs[f.Name.Name], dir)
		}
		declare := func(name string) { ix.top[dir][name], ix.any[name] = true, true }
		for _, decl := range f.Decls {
			switch x := decl.(type) {
			case *ast.FuncDecl:
				if x.Recv == nil {
					declare(x.Name.Name)
					continue
				}
				member(receiverType(x.Recv.List[0].Type), x.Name.Name)
			case *ast.GenDecl:
				for _, sp := range x.Specs {
					switch s := sp.(type) {
					case *ast.TypeSpec:
						declare(s.Name.Name)
						ix.types[s.Name.Name] = true
						indexMembers(s, member)
					case *ast.ValueSpec:
						for _, n := range s.Names {
							declare(n.Name)
						}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return ix
}

// receiverType is a method receiver's type name, without its pointer or type
// parameters.
func receiverType(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.StarExpr:
		return receiverType(x.X)
	case *ast.IndexExpr:
		return receiverType(x.X)
	case *ast.IndexListExpr:
		return receiverType(x.X)
	case *ast.Ident:
		return x.Name
	}
	return ""
}

// indexMembers records a struct's fields, embedded ones by their type's name,
// and an interface's methods, as members of the type.
func indexMembers(s *ast.TypeSpec, member func(typ, name string)) {
	var fields *ast.FieldList
	switch x := s.Type.(type) {
	case *ast.StructType:
		fields = x.Fields
	case *ast.InterfaceType:
		fields = x.Methods
	default:
		return
	}
	for _, fl := range fields.List {
		for _, n := range fl.Names {
			member(s.Name.Name, n.Name)
		}
		if len(fl.Names) == 0 {
			if name := receiverType(fl.Type); name != "" {
				member(s.Name.Name, name)
			}
		}
	}
}

// codeSpan is a backticked span on a page.
var codeSpan = regexp.MustCompile("`([^`\n]+)`")

// symbolShaped is a span written as a Go symbol: an identifier, dotted
// through packages, types and members, optionally path-qualified
// (`platform/geo.ReadGeometry`) and optionally called (`httpx.TTL(d)`).
var symbolShaped = regexp.MustCompile(`^(?:([a-z0-9_]+(?:/[a-z0-9_]+)+)\.)?([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)(?:\([^()]*\))?$`)

// checkable reports whether a symbol-shaped span names code, as opposed to a
// word, a key or a variable of the environment:
//   - a lowercase word, snake_case included, is a config key, a JSON field or
//     prose (`off`, `units`, `by_provider`);
//   - an ALL-CAPS word is an environment variable (`NO_COLOR`, `PATH`);
//   - a lowercase dotted span whose first part is neither a package nor a type
//     is a theme or config key (`table.header`).
//
// Each is left to the reader.
func (ix goIndex) checkable(path, sym string) bool {
	if path != "" {
		return true
	}
	if !strings.ContainsAny(sym, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		first, _, dotted := strings.Cut(sym, ".")
		return dotted && (len(ix.pkgDirs[first]) > 0 || ix.types[first])
	}
	return strings.ContainsAny(sym, "abcdefghijklmnopqrstuvwxyz")
}

// goFile is a span naming a Go file, by its path or by its name.
var goFile = regexp.MustCompile(`^[a-z0-9_/]*[a-z0-9_]+\.go$`)

// dirPath is a span shaped like a directory of the tree, `platform/snapshot`
// or `06_docs/02_features/`. It is one when its first segment is one of the
// tree's top-level directories (so `n/a` is a word), and then it must exist.
var dirPath = regexp.MustCompile(`^[a-z0-9_]+(?:/[a-z0-9_.-]+)+/?$`)

// isDir reports whether path is a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// otherFile is a span naming another kind of file. With a path it must exist;
// a bare name (`ci.yml`, a file a command downloads) is left to the reader.
var otherFile = regexp.MustCompile(`^[\w./-]+\.(?:md|ya?ml|jsonl?|csv|js|toml|txt|sh|py|golden|expect)$`)

// resolves reports whether the index declares a symbol-shaped span: a bare
// name anywhere; `pkg.Name` in that package; `Type.Member` on that type;
// `path.Name` in that directory; and the rest of a longer chain as members.
func (ix goIndex) resolves(path, sym string) bool {
	parts := strings.Split(sym, ".")
	if path != "" {
		return ix.top[path][parts[0]] && ix.membersOf(parts[0], parts[1:])
	}
	if len(parts) == 1 {
		return ix.any[parts[0]]
	}
	for _, dir := range ix.pkgDirs[parts[0]] {
		if ix.top[dir][parts[1]] && ix.membersOf(parts[1], parts[2:]) {
			return true
		}
	}
	if ix.types[parts[0]] {
		return ix.membersOf(parts[0], parts[1:])
	}
	// A FIELD OR VARIABLE FIRST (`Harmonized.UVIndex`): its type is not
	// followed, so each later part need only be declared.
	if !ix.any[parts[0]] {
		return false
	}
	for _, name := range parts[1:] {
		if !ix.any[name] {
			return false
		}
	}
	return true
}

// membersOf reports whether each name in chain is a member of the one
// before it, starting at typ. A member's own type is not followed, so past
// the first step any declared member counts.
func (ix goIndex) membersOf(typ string, chain []string) bool {
	if len(chain) == 0 {
		return true
	}
	if !ix.members[typ][chain[0]] {
		return false
	}
	for _, name := range chain[1:] {
		if !ix.any[name] {
			return false
		}
	}
	return true
}

// architectureCite is an "architecture §n" citation, and architecturePath the
// file it must name on the same line: there are several architecture.md
// files, so a citation without one names none of them (0.19.5 JM-13).
var (
	architectureCite = regexp.MustCompile("(?i)architecture(?:\\.md)?`?\\s*§")
	architecturePath = regexp.MustCompile(`[\w-]+/architecture\.md`)
)

// TestExtendingNamesRealSymbols is 0.19.5 Q7.4 (JM-3, DC-14, JR-9):
// docs/extending.md's banner says everything it names exists, and this holds
// it. Every symbol-shaped span on the page resolves to a declaration in the
// module's own code.
func TestExtendingNamesRealSymbols(t *testing.T) {
	pageNamesRealSymbols(t, "extending.md", 150)
}

// TestArchitectureNamesRealSymbols holds docs/architecture.md, the program's
// shape as built (0.19.5 A-3), to the same check.
func TestArchitectureNamesRealSymbols(t *testing.T) {
	pageNamesRealSymbols(t, "architecture.md", 25)
}

// pageNamesRealSymbols checks one page under docs/: every symbol-shaped span
// resolves, every file path exists, at least floor spans are checked, and an
// architecture citation names its file.
func pageNamesRealSymbols(t *testing.T, page string, floor int) {
	t.Helper()
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "docs", page))
	if err != nil {
		t.Fatal(err)
	}
	ix := indexModule(t, root)
	checked := 0
	for _, m := range codeSpan.FindAllStringSubmatch(string(raw), -1) {
		if goFile.MatchString(m[1]) {
			checked++
			if !ix.files[m[1]] {
				t.Errorf("docs/%s names the file `%s`, which does not exist", page, m[1])
			}
			continue
		}
		if otherFile.MatchString(m[1]) {
			if strings.Contains(m[1], "/") {
				checked++
				if !ix.files[m[1]] {
					t.Errorf("docs/%s names the file `%s`, which does not exist", page, m[1])
				}
			}
			continue
		}
		if top, _, _ := strings.Cut(m[1], "/"); dirPath.MatchString(m[1]) && !strings.Contains(m[1], ".") && isDir(filepath.Join(root, top)) {
			checked++
			if !isDir(filepath.Join(root, m[1])) {
				t.Errorf("docs/%s names the directory `%s`, which does not exist", page, m[1])
			}
			continue
		}
		s := symbolShaped.FindStringSubmatch(m[1])
		if s == nil || !ix.checkable(s[1], s[2]) {
			continue
		}
		checked++
		if !ix.resolves(s[1], s[2]) {
			t.Errorf("docs/%s names `%s`, which the module's code does not declare", page, m[1])
		}
	}
	t.Logf("checked %d spans on docs/%s", checked, page)
	// THE MATCHER IS ITSELF PINNED. Narrowed by accident, it would check a
	// handful of spans and pass, and the page would claim what nothing holds.
	if checked < floor {
		t.Errorf("only %d symbol spans were checked on docs/%s; the page names more, so the matcher has narrowed", checked, page)
	}
	for n, line := range strings.Split(string(raw), "\n") {
		if architectureCite.MatchString(line) && !architecturePath.MatchString(line) {
			t.Errorf("docs/%s:%d cites an architecture section with no file; name the architecture.md it means", page, n+1)
		}
	}
}

// TestTheSymbolCheckRefusesWhatIsNotDeclared is the check's own control: over
// the same index, a span naming nothing is refused in every form the page
// uses, a real one in each form passes, the words it leaves to the reader are
// left, and an architecture citation without its file is caught.
func TestTheSymbolCheckRefusesWhatIsNotDeclared(t *testing.T) {
	ix := indexModule(t, filepath.Join("..", ".."))
	for _, c := range []struct {
		span string
		want bool
	}{
		{"firms.CheckKey", true}, {"firms.NoSuchKey", false},
		{"Conditions.UVIndex", true}, {"Conditions.NoSuchField", false}, {"Conditions.CheckKey", false},
		{"livePipelines.markFIRMS", true}, {"livePipelines.noSuchMethod", false},
		{"Harmonized.UVIndex", true}, {"Harmonized.NoSuchField", false},
		{"detailLines", true}, {"noSuchHelper", false},
		{"platform/geo.ReadGeometry", true}, {"platform/geo.NoSuchReader", false},
		{"httpx.TTL(d)", true}, {"httpx.NoSuchTTL(d)", false},
	} {
		s := symbolShaped.FindStringSubmatch(c.span)
		if s == nil || !ix.checkable(s[1], s[2]) {
			t.Errorf("`%s` was not checked at all", c.span)
			continue
		}
		if got := ix.resolves(s[1], s[2]); got != c.want {
			t.Errorf("`%s` resolves %v, want %v", c.span, got, c.want)
		}
	}
	for _, word := range []string{"off", "by_provider", "NO_COLOR", "table.header"} {
		if s := symbolShaped.FindStringSubmatch(word); s != nil && ix.checkable(s[1], s[2]) {
			t.Errorf("`%s` is a word or a key, and was checked as a symbol", word)
		}
	}
	root := filepath.Join("..", "..")
	if !isDir(filepath.Join(root, "platform/snapshot")) || isDir(filepath.Join(root, "platform/no_such")) || isDir(filepath.Join(root, "n")) {
		t.Error("the directory check does not tell a directory that exists from one that does not, or a word from a path")
	}
	if !ix.files["app/credits.go"] || ix.files["app/no_such.go"] || !ix.files["credits.go"] {
		t.Error("the file index does not tell a file that exists from one that does not")
	}
	bad, good := "see architecture §1.1", "see [architecture §1.1](../06_docs/x/architecture.md)"
	if !architectureCite.MatchString(bad) || architecturePath.MatchString(bad) || !architecturePath.MatchString(good) {
		t.Error("an architecture citation without its file is not told from one with it")
	}
}
