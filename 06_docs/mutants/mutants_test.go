//go:build mutants

// Package mutants guards the mutation corpus itself.
//
// WHY THIS IS GO AND NOT SHELL (D-9). The guards this replaces were bash, and
// bash fails silently by default: exit codes vanish through pipes, counters die
// in subshells, an unmatched glob passes as an empty list, and `--exclude
// '.git/'` matches a directory but not the .git FILE a git worktree uses — which
// made a guard run `git commit` against the developer's own repository while
// reporting OK. Seventeen of one session's thirty-four defects were in those
// scripts, against ten in the product.
//
// As a Go test a t.Fatalf cannot be lost the way an exit code can, the copy is a
// walk rather than rsync semantics, `.git` is skipped by NAME so a worktree's
// .git FILE cannot come along, and the guard is testable by ordinary means —
// which the shell controls never were, being a second implementation of the
// thing they checked.
//
// IT IS BEHIND A BUILD TAG, and deliberately not part of `go test ./...`. It
// copies and compiles the tree once per mutant, which is ~143s; inside the
// default run that cost would also land in `make race`, under the race detector,
// on every commit. The LANGUAGE choice and the WHEN-TO-RUN choice are separate,
// and only the first is what D-9 is about.
//
//	make mutant-check
package mutants

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// corpusFloor is the smallest corpus this project is ever expected to have.
//
// AN EMPTY OR MIS-GLOBBED CORPUS IS A FAILURE, NOT A PASS. The shell version
// looped zero times, left every counter at zero and exited clean — a green gate
// over nothing at all, while printing its own controls' success as reassurance.
const corpusFloor = 50

// targetRE finds the files a mutant patches: every pathlib.Path("...") literal.
var targetRE = regexp.MustCompile(`pathlib\.Path\("([^"]*)"\)`)

// repoRoot is the module root, two directories above this package.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("locating the working directory: %v", err)
	}
	root := filepath.Dir(filepath.Dir(wd))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("expected the module root at %s: %v", root, err)
	}
	return root
}

// corpus is every mutant, by path relative to the root.
func corpus(t *testing.T, root string) []string {
	t.Helper()
	found, err := filepath.Glob(filepath.Join(root, "06_docs", "mutants", "m*.py"))
	if err != nil {
		t.Fatalf("reading the corpus: %v", err)
	}
	if len(found) < corpusFloor {
		t.Fatalf("found %d mutants under 06_docs/mutants/m*.py, expected at least %d — "+
			"a corpus this small means the glob is wrong, not that the tree is clean", len(found), corpusFloor)
	}
	return found
}

// copyTree copies the working tree into dir, skipping what no build reads.
//
// .git IS SKIPPED BY NAME, whether it is a directory or a FILE. In a git
// worktree it is a file pointing at the real gitdir, and a copy that inherits
// it is a copy that can write to the developer's repository.
func copyTree(t *testing.T, root, dir string) {
	t.Helper()
	skip := map[string]bool{".git": true, ".claude": true, "dist": true, "docs": true, "node_modules": true}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skip[filepath.Base(path)] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		dst := filepath.Join(dir, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil // symlinks and sockets are not source
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copying the tree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		t.Fatal("the copy contains a .git — it could write to the real repository")
	}
}

// verdict is what applying one mutant to a copy proves.
type verdict string

const (
	applies verdict = "applies and compiles"
	stale   verdict = "STALE"   // the mutation no longer matches the source
	inert   verdict = "INERT"   // it applied and changed nothing
	broken  verdict = "INVALID" // the mutated tree does not compile
	escaped verdict = "ESCAPED" // it wrote outside the copy
)

// judge applies one mutant to its own copy of the tree and classifies it.
func judge(t *testing.T, root, mutant, dir string) (verdict, string) {
	t.Helper()
	copyTree(t, root, dir)
	before := digest(t, dir)

	// The mutant is COPIED IN and run with the copy as its working directory, so
	// a mutant that locates its target relative to its own __file__ resolves
	// inside the copy rather than in the real tree.
	local := filepath.Join(dir, "06_docs", "mutants", filepath.Base(mutant))
	cmd := exec.Command("python3", local)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return stale, strings.TrimSpace(lastLine(string(out)))
	}
	after := digest(t, dir)
	if before == after {
		return inert, "the mutation applied but changed nothing in the tree"
	}
	if out, err := run(dir, "go", "build", "./..."); err != nil {
		return broken, firstLine(out)
	}
	// go build never compiles tests, and the rules these mutants delete are
	// frequently called only from tests — so the patched packages are vetted
	// too, which does compile them.
	if pkgs := packagesOf(mutant); len(pkgs) > 0 {
		if out, err := run(dir, "go", append([]string{"vet"}, pkgs...)...); err != nil {
			return broken, firstLine(out)
		}
	}
	return applies, ""
}

// packagesOf is the Go packages a mutant patches. A mutant that patches no Go
// file has none, and vetting a directory that is not a package would report a
// compile failure for a tree that compiles perfectly well.
func packagesOf(mutant string) []string {
	b, err := os.ReadFile(mutant)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, m := range targetRE.FindAllStringSubmatch(string(b), -1) {
		if !strings.HasSuffix(m[1], ".go") {
			continue
		}
		p := "./" + filepath.Dir(m[1])
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// TestEveryMutantAppliesAndCompiles is D-3, mechanised.
//
// A mutant that no longer matches its source is a rule NOBODY IS MEASURING: a
// sweep reports fewer verdicts than it launched and the tally still looks clean.
// A mutant that removes a USE rather than a rule leaves the tree uncompilable,
// and its verdict is no evidence either way. Six of each in one release, every
// one of them found by paying for a sweep first.
// mutantLanes is how many trees may compile at once. Four rather than one:
// serial took 429 s when the corpus was 159 mutants (it is larger now, so the
// envelope is understated); four keeps the run near three minutes while leaving
// the machine enough to be used.
const mutantLanes = 4

func TestEveryMutantAppliesAndCompiles(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	if testing.Short() {
		t.Skip("copies and compiles the tree once per mutant")
	}
	// THE GUARD BOUNDS ITS OWN CONCURRENCY, rather than taking whatever
	// -parallel defaults to.
	//
	// Each subtest compiles the whole tree. At one per CPU that is a dozen full
	// builds at once, and on a loaded machine — the app under UAT, an editor —
	// they start failing: 2026-09-04 produced a run where nine mutants were
	// reported UNCOMPILABLE and one of the compile errors named `net/netip`.
	// Nothing in this repository can break the standard library, so those
	// verdicts were about the machine, not the code.
	//
	// That is the guard's own failure mode arriving in its own output: it exists
	// to say when a verdict is no evidence, and it was producing verdicts that
	// were no evidence. A gate whose reliability falls as the corpus grows gets
	// least trustworthy exactly when it is worth most, so the ceiling is here in
	// the test rather than in whatever command happens to invoke it.
	lane := make(chan struct{}, mutantLanes)
	for _, mutant := range corpus(t, root) {
		name := filepath.Base(mutant)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			lane <- struct{}{}
			defer func() { <-lane }()
			got, detail := judge(t, root, mutant, t.TempDir())
			switch got {
			case applies:
			case stale:
				t.Errorf("%s no longer matches the tip, so the rule it guards is UNMEASURED: %s", name, detail)
			case inert:
				t.Errorf("%s %s — it cannot catch anything", name, detail)
			case broken:
				t.Errorf("%s makes the tree uncompilable, so its verdict is no evidence either way: %s", name, detail)
			default:
				t.Errorf("%s: %s", name, detail)
			}
		})
	}
}

// D-4 IS NOT MECHANISABLE, AND THIS IS THE RECORD OF FINDING THAT OUT.
//
// The rule reads: a mutant must delete a RULE, never weaken an ASSERTION, since
// one that weakens a check measures the check and can never fail. It is true,
// and three static predicates were written for it, each disproved by running it
// against the corpus in under a second:
//
//	"mentions invariant.Check"      flagged mD9 and m67, which DELETE rules —
//	                                in this codebase an invariant IS how a rule
//	                                is written
//	"both sides mention one"        flagged m63 and mD5, whose real edits are
//	                                elsewhere and which merely quote a check as
//	                                anchor context
//	"the edit lands inside a check" flagged m67, which drops half of
//	                                `have.Slot == c.Slot && have.Origin == c.Origin`
//	                                — a genuine rule deletion that is CAUGHT
//
// m67 is the counter-example that settles it. Weakening an invariant's condition
// is a legitimate mutation when the code under it can violate the weakened form,
// and a vacuous one when it cannot — and THAT is semantic, not syntactic. No
// static predicate separates them.
//
// The decidable form is a triage rule at sweep time rather than a check here: a
// mutant that SURVIVES while editing only an invariant's condition means either
// an unpinned rule or a vacuous check, and both need a person. It is recorded in
// 06_docs/defect-classes.md as D-4 and is not enforced by a test, because the
// quality plan says a rule that cannot be made mechanical should not pretend to
// be. The list is supposed to shrink.

// digest is a stable fingerprint of the tree's regular files: sorted paths and
// their contents.
//
// SORTED AND CONTENT-ONLY, DELIBERATELY. A fingerprint built from an unsorted
// walk piped through xargs gave a different answer for the same unchanged tree
// on three consecutive runs, and told me a script had modified the working tree
// when it had not. An instrument that cannot repeat itself is not measuring.
func digest(t *testing.T, dir string) string {
	t.Helper()
	h := sha256.New()
	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("digesting the copy: %v", err)
	}
	sort.Strings(paths)
	for _, p := range paths {
		rel, _ := filepath.Rel(dir, p)
		h.Write([]byte(rel))
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("digesting %s: %v", p, err)
		}
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// run executes a command in dir and returns its combined output.
func run(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			return strings.TrimSpace(l)
		}
	}
	return ""
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
