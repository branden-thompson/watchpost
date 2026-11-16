//go:build mutants

package mutants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// listedPackage is the part of `go list -json` the commit scope reads.
type listedPackage struct {
	ImportPath, Dir                 string
	Deps, TestImports, XTestImports []string
}

// wholeModuleFiles are the changes that reach every package, so a change to
// one selects the whole corpus.
var wholeModuleFiles = []string{"go.mod", "go.sum", "go.work", "go.work.sum"}

// reach is every import path a package's build and its tests' builds read:
// itself, its dependencies, and each test import with that import's own
// dependencies.
func reach(p listedPackage, byPath map[string]listedPackage) map[string]bool {
	out := map[string]bool{p.ImportPath: true}
	for _, d := range p.Deps { // bounded by the imports (P10-02)
		out[d] = true
	}
	for _, ti := range slices.Concat(p.TestImports, p.XTestImports) { // bounded by the test imports (P10-02)
		out[ti] = true
		for _, d := range byPath[ti].Deps { // bounded by the imports (P10-02)
			out[d] = true
		}
	}
	return out
}

// ownerOf is the deepest package directory holding a path, or "".
func ownerOf(path string, pkgs []listedPackage) string {
	best := ""
	for _, p := range pkgs { // bounded by the module (P10-02)
		if (path == p.Dir || strings.HasPrefix(path, p.Dir+string(filepath.Separator))) && len(p.Dir) > len(best) {
			best = p.Dir
		}
	}
	return best
}

// inScope is the commit gate's mutants (0.19.6 D-38): those whose target's
// package, or a package that package or its tests import, holds a changed
// file, and those whose own script changed. A file belongs to the deepest
// package directory that holds it, so a testdata file counts for its package.
// A change to go.mod or go.sum reaches every package and selects the whole
// corpus. changed is relative to root; the packages' directories and the
// mutants' paths are absolute.
//
// GOTCHA: a change outside every package (a document, a script) selects
// nothing, so a commit of those runs no mutant at the commit gate. The full
// gate, at each batch's close, judges the whole corpus.
func inScope(root string, changed []string, pkgs []listedPackage, mutants []string) []string {
	for _, f := range changed { // bounded by the change (P10-02)
		if slices.Contains(wholeModuleFiles, f) {
			return mutants
		}
	}
	byPath := map[string]listedPackage{}
	for _, p := range pkgs { // bounded by the module (P10-02)
		byPath[p.ImportPath] = p
	}
	touched := map[string]bool{} // import paths of the packages holding a changed file
	changedFile := map[string]bool{}
	for _, f := range changed { // bounded by the change (P10-02)
		changedFile[f] = true
		owner := ownerOf(filepath.Join(root, f), pkgs)
		for _, p := range pkgs { // bounded by the module (P10-02)
			if owner != "" && p.Dir == owner {
				touched[p.ImportPath] = true
			}
		}
	}
	var out []string
	for _, m := range mutants { // bounded by the corpus (P10-02)
		rel, _ := filepath.Rel(root, m)
		targets, _ := targetsOf(m)
		if changedFile[filepath.ToSlash(rel)] || len(targets) == 0 {
			out = append(out, m) // a changed script is judged; one naming no target is judged, and fails
			continue
		}
		if reachesTouched(filepath.Join(root, targets[0]), pkgs, byPath, touched) {
			out = append(out, m)
		}
	}
	return out
}

// reachesTouched reports whether the package owning a file reads a touched
// package, itself included.
func reachesTouched(file string, pkgs []listedPackage, byPath map[string]listedPackage, touched map[string]bool) bool {
	owner := ownerOf(file, pkgs)
	for _, p := range pkgs { // bounded by the module (P10-02)
		if p.Dir != owner {
			continue
		}
		for path := range reach(p, byPath) { // bounded by the package's reach (P10-02)
			if touched[path] {
				return true
			}
		}
	}
	return false
}

// changedFiles is what the commit being gated changes: the working tree
// against HEAD, and every file git does not yet track.
func changedFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	for _, args := range [][]string{{"diff", "--name-only", "HEAD"}, {"ls-files", "--others", "--exclude-standard"}} { // the two (P10-02)
		got, err := run(root, "git", args...)
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, got)
		}
		for _, l := range strings.Split(strings.TrimSpace(got), "\n") { // bounded by the change (P10-02)
			if l != "" {
				out = append(out, l)
			}
		}
	}
	return out
}

// modulePackages is `go list -json ./...` over the root.
func modulePackages(t *testing.T, root string) []listedPackage {
	t.Helper()
	out, err := run(root, "go", "list", "-json", "./...")
	if err != nil {
		t.Fatalf("go list: %v: %s", err, firstLine(out))
	}
	var pkgs []listedPackage
	dec := json.NewDecoder(bytes.NewReader([]byte(out)))
	for dec.More() { // bounded by the module's packages (P10-02)
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("reading go list: %v", err)
		}
		pkgs = append(pkgs, p)
	}
	if len(pkgs) < 50 { // the module holds over a hundred packages at 0.19.6 B0
		t.Fatalf("go list found %d packages; the reading is broken", len(pkgs))
	}
	return pkgs
}

// mutantsToJudge is the corpus the scope selects: MUTANT_SCOPE=full, which
// `make verify-full`, CI and the release job set, is every mutant; anything
// else is the commit gate's scope.
func mutantsToJudge(t *testing.T, root string) []string {
	t.Helper()
	all := corpus(t, root)
	if os.Getenv("MUTANT_SCOPE") == "full" {
		t.Logf("mutant scope: full, %d mutants", len(all))
		return all
	}
	changed := changedFiles(t, root)
	picked := inScope(root, changed, modulePackages(t, root), all)
	t.Logf("mutant scope: commit, %d of %d mutants, from %d changed file(s)", len(picked), len(all), len(changed))
	return picked
}

// TestTheCommitScopeSelectsWhatAChangeReaches is 0.19.6 D-38 on a planted
// module: a change selects the mutants of its package and of every package
// that imports it, through code or through tests, and of none other; a
// testdata change counts for its package; a changed script selects itself;
// a change to go.mod selects every mutant.
func TestTheCommitScopeSelectsWhatAChangeReaches(t *testing.T) {
	root := "/m"
	pkgs := []listedPackage{
		{ImportPath: "m/base", Dir: "/m/base"},
		{ImportPath: "m/user", Dir: "/m/user", Deps: []string{"m/base"}},
		{ImportPath: "m/tested", Dir: "/m/tested", TestImports: []string{"m/user"}},
		{ImportPath: "m/apart", Dir: "/m/apart"},
		{ImportPath: "m/apart/sub", Dir: "/m/apart/sub"},
	}
	dir := t.TempDir()
	mutant := func(name, target string) string {
		p := filepath.Join(dir, "06_docs", "mutants", name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(fmt.Sprintf("import pathlib\np = pathlib.Path('%s')\n", target)), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	mBase, mUser, mTested, mApart, mSub := mutant("mb.py", "base/b.go"), mutant("mu.py", "user/u.go"), mutant("mt.py", "tested/t.go"), mutant("ma.py", "apart/a.go"), mutant("ms.py", "apart/sub/s.go")
	all := []string{mBase, mUser, mTested, mApart, mSub}
	// The mutants live under the temporary directory; inScope reads their
	// targets from their scripts and resolves the targets against root.
	for _, c := range []struct {
		name    string
		changed []string
		want    []string
	}{
		{"a change reaches its importers, through code and tests", []string{"base/b.go"}, []string{mBase, mUser, mTested}},
		{"a change reaches only what reads it", []string{"user/u.go"}, []string{mUser, mTested}},
		{"a testdata file counts for its package", []string{"apart/testdata/x.json"}, []string{mApart}},
		{"the deepest package owns a file", []string{"apart/sub/s.go"}, []string{mSub}},
		{"a document selects nothing", []string{"README.md"}, nil},
		{"go.mod selects everything", []string{"go.mod"}, all},
	} {
		if got := inScope(root, c.changed, pkgs, all); !slices.Equal(got, c.want) {
			t.Errorf("%s: selected %v; want %v", c.name, names(got), names(c.want))
		}
	}
	// A changed script selects itself, wherever its target is.
	rel, err := filepath.Rel(root, mApart)
	if err != nil {
		t.Fatal(err)
	}
	if got := inScope(root, []string{filepath.ToSlash(rel)}, pkgs, all); !slices.Contains(got, mApart) {
		t.Errorf("a changed script did not select itself: %v", names(got))
	}
}

// names is the base names of mutant paths, for a failure message.
func names(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths { // bounded by the list (P10-02)
		out = append(out, filepath.Base(p))
	}
	return out
}
