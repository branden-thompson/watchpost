package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// site is where a mutant edits at one commit: the file, the function its first
// changed line sits in, and a hash of that function's source.
type site struct {
	target, fn, hash string
}

// tree reads files at one commit, each once.
type tree struct {
	root, rev string
	files     map[string][]byte
}

func newTree(root, rev string) *tree {
	return &tree{root: root, rev: rev, files: map[string][]byte{}}
}

// read is a file's content at the tree's commit; a file the commit lacks
// reads as nil, not as an error, because a batch adds and removes files.
func (t *tree) read(path string) ([]byte, error) {
	if b, ok := t.files[path]; ok {
		return b, nil
	}
	cmd := exec.Command("git", "-C", t.root, "show", t.rev+":"+path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil && (strings.Contains(stderr.String(), "does not exist") || strings.Contains(stderr.String(), "but not in")) {
		b, err = nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %v: %s", t.rev, path, err, stderr.String())
	}
	t.files[path] = b
	return b, nil
}

// mutants lists the corpus at the tree's commit.
func (t *tree) mutants() ([]string, error) {
	out, err := exec.Command("git", "-C", t.root, "ls-tree", "--name-only", t.rev, "06_docs/mutants/").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree %s: %v", t.rev, err)
	}
	var list []string
	for _, p := range strings.Fields(string(out)) {
		if strings.HasPrefix(filepath.Base(p), "m") && strings.HasSuffix(p, ".py") {
			list = append(list, p)
		}
	}
	return list, nil
}

// driverSource runs each mutant against the files of one commit with its read
// served from git and its write caught, and prints the file it edits and the
// first line its edit changes. It writes nothing.
const driverSource = `
import pathlib, subprocess, sys
root, rev = sys.argv[1], sys.argv[2]
seen = {}
def read(self, *a, **k):
    key = self.as_posix()
    if key not in seen:
        seen[key] = subprocess.run(["git", "-C", root, "show", rev + ":" + key], capture_output=True, text=True, check=True).stdout
    return seen[key]
def write(self, text, *a, **k):
    old = seen[self.as_posix()].splitlines()
    new = text.splitlines()
    n = next((i for i, (x, y) in enumerate(zip(old, new)) if x != y), min(len(old), len(new)))
    print("SITE\t%s\t%s\t%d" % (label, self.as_posix(), n + 1))
pathlib.Path.read_text = read
pathlib.Path.write_text = write
for arg in sys.argv[3:]:
    label, _, path = arg.partition("=")
    try:
        exec(compile(open(path, encoding="utf-8").read(), label, "exec"), {"__name__": "__mutant__"})
    except Exception as e:
        print("FAIL\t%s\t%s: %s" % (label, type(e).__name__, str(e).replace("\n", " ")))
print("DONE")
`

// sites finds where each mutant of the list edits at the tree's commit. A
// mutant the commit lacks is left out; one whose anchor no longer matches is
// an error, because a drifted mutant measures nothing.
func (t *tree) sites(list []string) (map[string]site, error) {
	dir, err := os.MkdirTemp("", "mutantmap")
	if err != nil {
		return nil, err
	}
	// The copies are scratch; a failure to remove them loses nothing.
	defer func() { _ = os.RemoveAll(dir) }()
	args := []string{"-I", "-c", driverSource, t.root, t.rev}
	for i, m := range list {
		src, err := t.read(m)
		if err != nil {
			return nil, err
		}
		if src == nil {
			continue
		}
		p := filepath.Join(dir, strconv.Itoa(i)+".py")
		if err := os.WriteFile(p, src, 0o600); err != nil {
			return nil, err
		}
		args = append(args, m+"="+p)
	}
	cmd := exec.Command("python3", args...)
	cmd.Dir = t.root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("the mutant driver at %s: %v", t.rev, err)
	}
	return t.parseSites(out)
}

// parseSites reads the driver's lines, refusing an output with no DONE: a
// driver that died part way measured nothing past the point it died.
func (t *tree) parseSites(out []byte) (map[string]site, error) {
	found := map[string]site{}
	done := false
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Split(line, "\t")
		switch {
		case f[0] == "DONE":
			done = true
		case f[0] == "FAIL" && len(f) == 3:
			return nil, fmt.Errorf("mutant %s at %s: %s", f[1], t.rev, f[2])
		case f[0] == "SITE" && len(f) == 4:
			line, err := strconv.Atoi(f[3])
			if err != nil {
				return nil, err
			}
			src, err := t.read(f[2])
			if err != nil {
				return nil, err
			}
			fn, hash := enclosing(f[2], src, line)
			found[f[1]] = site{target: f[2], fn: fn, hash: hash}
		}
	}
	if !done {
		return nil, fmt.Errorf("the mutant driver at %s stopped before its end", t.rev)
	}
	return found, nil
}

// enclosing names the declaration a line sits in and hashes its source: a
// function as Name or Recv.Name, a type, var or const by its first name, and
// anything else, a Makefile included, as the whole file.
func enclosing(path string, src []byte, line int) (string, string) {
	whole := fmt.Sprintf("%x", sha256.Sum256(src))
	if !strings.HasSuffix(path, ".go") {
		return "-", whole
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return "-", whole
	}
	for _, d := range f.Decls {
		from, to := fset.Position(d.Pos()), fset.Position(d.End())
		if line < from.Line || line > to.Line {
			continue
		}
		return declName(d), fmt.Sprintf("%x", sha256.Sum256(src[from.Offset:to.Offset]))
	}
	return "-", whole
}

// declName is Name, Recv.Name, or a general declaration's first name.
func declName(d ast.Decl) string {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil || len(d.Recv.List) == 0 {
			return d.Name.Name
		}
		return recvName(d.Recv.List[0].Type) + "." + d.Name.Name
	case *ast.GenDecl:
		for _, s := range d.Specs {
			switch s := s.(type) {
			case *ast.TypeSpec:
				return s.Name.Name
			case *ast.ValueSpec:
				return s.Names[0].Name
			}
		}
	}
	return "-"
}

// recvName is a receiver's type name, through a pointer and type parameters.
func recvName(e ast.Expr) string {
	if s, ok := e.(*ast.StarExpr); ok {
		e = s.X
	}
	switch x := e.(type) {
	case *ast.IndexExpr:
		e = x.X
	case *ast.IndexListExpr:
		e = x.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return "?"
}

// testDef is where a top-level test is declared, and a hash of its source.
type testDef struct {
	file, hash string
}

// tests finds each named test's declarations at the tree's commit. A name
// declared in more than one package keeps every declaration.
func (t *tree) tests(names map[string]bool) (map[string][]testDef, error) {
	out, err := exec.Command("git", "-C", t.root, "grep", "-n", "-E", `^func Test[A-Za-z0-9_]+\(`, t.rev, "--", "*_test.go").Output()
	if err != nil && len(out) == 0 {
		return map[string][]testDef{}, nil
	}
	found := map[string][]testDef{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(strings.TrimPrefix(line, t.rev+":"), ":", 3)
		if len(parts) != 3 {
			continue
		}
		name := strings.TrimPrefix(strings.SplitN(parts[2], "(", 2)[0], "func ")
		if !names[name] {
			continue
		}
		n, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		src, err := t.read(parts[0])
		if err != nil {
			return nil, err
		}
		_, hash := enclosing(parts[0], src, n)
		found[name] = append(found[name], testDef{file: parts[0], hash: hash})
	}
	return found, nil
}
