//go:build mutants

// Package mutants guards the mutation corpus itself.
//
// WHY THIS IS GO AND NOT SHELL (D-9). Bash fails silently by default: exit
// codes vanish through pipes, counters die in subshells, an unmatched glob
// passes as an empty list, and `--exclude '.git/'` matches a directory but not
// the .git FILE a git worktree uses — so a shell guard can run `git commit`
// against the developer's own repository while reporting OK.
//
// As a Go test a t.Fatalf cannot be lost the way an exit code can, the build
// goes through the go command's own overlay rather than a shell's copy, and the
// guard is testable by ordinary means (judge_test.go) — which a shell control
// is not, being a second implementation of the thing it checks.
//
// IT IS BEHIND A BUILD TAG, and deliberately not part of `go test ./...`. It
// builds the tree once per mutant; inside the default run that cost would
// also land in `make race`, under the race detector, on every commit. The
// LANGUAGE choice and the WHEN-TO-RUN choice are separate, and only the first
// is what D-9 is about.
//
//	make mutant-check
package mutants

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
// AN EMPTY OR MIS-GLOBBED CORPUS IS A FAILURE, NOT A PASS. A loop over nothing
// runs zero times, leaves every counter at zero and exits clean — a green gate
// over nothing at all.
const corpusFloor = 50

// targetRE finds the files a mutant patches: every pathlib.Path literal, in
// either of Python's quotes. Two thirds of the corpus writes 'single' ones.
var targetRE = regexp.MustCompile(`pathlib\.Path\(["']([^"']*)["']\)`)

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

// verdict is what applying one mutant proves.
type verdict string

const (
	applies verdict = "applies and compiles"
	stale   verdict = "STALE"   // the mutation no longer matches the source
	inert   verdict = "INERT"   // it applied and changed nothing
	broken  verdict = "INVALID" // the mutated tree does not compile
	escaped verdict = "ESCAPED" // it wrote, or names a target, outside the files it patches
)

// targetsOf is the files a mutant patches, relative to the root, and the
// first target that is not a relative path inside it, if any.
func targetsOf(mutant string) (targets []string, outside string) {
	b, err := os.ReadFile(mutant)
	if err != nil {
		return nil, ""
	}
	seen := map[string]bool{}
	for _, m := range targetRE.FindAllStringSubmatch(string(b), -1) { // bounded by the script (P10-02)
		rel := m[1]
		if filepath.IsAbs(rel) || !filepath.IsLocal(rel) {
			return nil, rel
		}
		if !seen[rel] {
			seen[rel] = true
			targets = append(targets, rel)
		}
	}
	return targets, ""
}

// copyFile copies one file, creating its directory.
func copyFile(t *testing.T, from, to string) error {
	t.Helper()
	b, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.WriteFile(to, b, 0o644)
}

// judge applies one mutant and classifies it (0.19.6 D-37).
//
// ONLY THE MUTANT'S TARGETS ARE COPIED, into dir, and the mutant runs there.
// The build then runs in the real tree with `-overlay`, which swaps the mutated
// files in for that build alone. So every package the mutant does not touch is
// a build-cache hit, and nothing is written into the tree: a copy of the whole
// tree per mutant was about 56 MB written and read twice over, for 1,059
// mutants a gate.
//
// THE REAL TREE IS THEREFORE WITHIN REACH OF THE SCRIPT, which the copy used to
// prevent. Two checks stand in for it: a target that is not a relative path
// inside the root is refused before the script runs, and the real targets
// must be byte for byte what they were after it ran. Anything else a script
// writes is a mutant that names files it does not patch, and the gate's own
// clean-tree check in TestEveryMutantAppliesAndCompiles catches it.
func judge(t *testing.T, root, mutant, dir string) (verdict, string) {
	t.Helper()
	targets, outside := targetsOf(mutant)
	if outside != "" {
		return escaped, "it names " + outside + ", which is not a path inside the tree"
	}
	if len(targets) == 0 {
		return stale, "it names no target"
	}
	realBefore := map[string]string{}
	for _, rel := range targets { // bounded by the targets (P10-02)
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return stale, "its target " + rel + " is not in the tree"
		}
		realBefore[rel] = string(b)
		if err := copyFile(t, filepath.Join(root, rel), filepath.Join(dir, rel)); err != nil {
			t.Fatalf("copying %s: %v", rel, err)
		}
	}
	// The mutant is COPIED IN and run with dir as its working directory, so a
	// mutant that locates its target relative to its own __file__ resolves
	// inside dir rather than in the real tree.
	local := filepath.Join(dir, "06_docs", "mutants", filepath.Base(mutant))
	if err := copyFile(t, mutant, local); err != nil {
		t.Fatalf("copying the mutant: %v", err)
	}
	before := digest(t, dir)
	cmd := exec.Command("python3", local)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	for _, rel := range targets { // bounded by the targets (P10-02)
		if b, _ := os.ReadFile(filepath.Join(root, rel)); string(b) != realBefore[rel] {
			return escaped, "it wrote to the real tree's " + rel
		}
	}
	if err != nil {
		return stale, strings.TrimSpace(lastLine(string(out)))
	}
	if before == digest(t, dir) {
		return inert, "the mutation applied but changed nothing in the tree"
	}
	overlay, err := writeOverlay(root, dir, targets)
	if err != nil {
		t.Fatalf("writing the overlay: %v", err)
	}
	if out, err := run(root, "go", "build", "-overlay", overlay, "./..."); err != nil {
		return broken, firstLine(out)
	}
	// go build never compiles tests, and the rules these mutants delete are
	// frequently called only from tests — so the patched packages are vetted
	// too, which does compile them.
	if pkgs := packagesOf(mutant); len(pkgs) > 0 {
		if out, err := run(root, "go", append([]string{"vet", "-overlay", overlay}, pkgs...)...); err != nil {
			return broken, firstLine(out)
		}
	}
	return applies, ""
}

// writeOverlay writes the go command's overlay file beside dir: each real
// target replaced by its mutated copy.
func writeOverlay(root, dir string, targets []string) (string, error) {
	replace := map[string]string{}
	for _, rel := range targets { // bounded by the targets (P10-02)
		replace[filepath.Join(root, rel)] = filepath.Join(dir, rel)
	}
	b, err := json.Marshal(map[string]map[string]string{"Replace": replace})
	if err != nil {
		return "", err
	}
	path := dir + ".overlay.json"
	return path, os.WriteFile(path, b, 0o644)
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

// TestEveryMutantNamesItsTarget fails a mutant whose target targetRE cannot
// read. packagesOf decides which packages `go vet` compiles a mutant against,
// and the verdict sweep reads its package the same way, so a target it cannot
// read is a mutant whose test files are never compiled and whose verdict
// cannot be taken.
func TestEveryMutantNamesItsTarget(t *testing.T) {
	root := repoRoot(t)
	for _, mutant := range corpus(t, root) {
		b, err := os.ReadFile(mutant)
		if err != nil {
			t.Fatal(err)
		}
		if len(targetRE.FindAllStringSubmatch(string(b), -1)) == 0 {
			t.Errorf("%s names no target targetRE can read: write it as pathlib.Path(\"<file>\")", filepath.Base(mutant))
		}
	}
}

// mutantLanes is how many mutants may build at once. Four rather than one:
// serial takes several times as long, and four leaves the machine enough to be
// used.
const mutantLanes = 4

// TestEveryMutantAppliesAndCompiles is D-3, mechanised.
//
// A mutant that no longer matches its source is a rule NOBODY IS MEASURING: a
// sweep reports fewer verdicts than it launched and the tally still looks clean.
// A mutant that removes a USE rather than a rule leaves the tree uncompilable,
// and its verdict is no evidence either way. This test finds both before a
// sweep is paid for.
//
// THE COMMIT GATE JUDGES WHAT ITS CHANGE REACHES (0.19.6 D-38): `make verify`
// judges the mutants inScope selects, and MUTANT_SCOPE=full, which `make
// verify-full`, CI and the release job set, judges the whole corpus.
//
// THE TREE IS THE SAME AFTERWARDS. Each mutant builds against the real tree
// through an overlay (judge, 0.19.6 D-37), so `git status` is read before the
// corpus and after it, and any change fails the test, naming what a script
// wrote.
func TestEveryMutantAppliesAndCompiles(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	if testing.Short() {
		t.Skip("builds the tree once per mutant")
	}
	status := func() string {
		out, err := run(root, "git", "status", "--porcelain")
		if err != nil {
			t.Fatalf("reading the tree's status: %v: %s", err, out)
		}
		return out
	}
	was := status()
	t.Cleanup(func() {
		if now := status(); now != was {
			t.Errorf("the corpus changed the real tree:\nbefore:\n%s\nafter:\n%s", was, now)
		}
	})
	// THE GUARD BOUNDS ITS OWN CONCURRENCY, rather than taking whatever
	// -parallel defaults to.
	//
	// Each subtest builds the tree. At one per CPU that is a dozen builds at
	// once, and on a loaded machine — the app under UAT, an editor —
	// they start failing: mutants are reported UNCOMPILABLE with compile errors
	// that name the standard library (`net/netip`). Nothing in this repository
	// can break the standard library, so such verdicts are about the machine,
	// not the code.
	//
	// That is the guard's own failure mode arriving in its own output: it exists
	// to say when a verdict is no evidence. A gate whose reliability falls as the
	// corpus grows gets least trustworthy exactly when it is worth most, so the
	// ceiling is here in the test rather than in whatever command happens to
	// invoke it.
	lane := make(chan struct{}, mutantLanes)
	for _, mutant := range mutantsToJudge(t, root) {
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

// D-4 IS NOT MECHANISABLE.
//
// The rule reads: a mutant must delete a RULE, never weaken an ASSERTION, since
// one that weakens a check measures the check and can never fail. It is true,
// and every static predicate for it misclassifies mutants in the corpus:
//
//	"mentions invariant.Check"      flags mD9 and m67, which DELETE rules —
//	                                in this codebase an invariant IS how a rule
//	                                is written
//	"both sides mention one"        flags m63 and mD5, whose real edits are
//	                                elsewhere and which merely quote a check as
//	                                anchor context
//	"the edit lands inside a check" flags m67, which drops half of
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
// walk gives different answers for the same unchanged tree, and reports a
// modified working tree when nothing changed. An instrument that cannot repeat
// itself is not measuring.
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
