//go:build mutants

package mutants

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The harness decides whether a rule is pinned, and nothing checked that IT
// works. Three of its verdicts were wrong at least once on 0.14.0: a crash read
// as SURVIVED, a traceback read as a verdict, and a scope too narrow to run the
// tests that catch a mutant.
//
// THE SHELL CONTROLS THAT CAME BEFORE THESE CERTIFIED ALMOST NOTHING. An
// independent review sabotaged run.sh ten realistic ways and NINE passed all
// four of them green — including deleting the crash-is-CAUGHT rule that the
// controls' own header cited as a wrong they prevent, deleting the green-baseline
// gate, and re-arming the restore trap above the clean-tree check, which is the
// regression that once ate a developer's uncommitted work. Four controls
// exercising one scope, one clean tree and one code path do not pin an
// instrument with that many branches.
//
// These run against a PURPOSE-BUILT MODULE rather than the real repository, so a
// scenario can make the baseline red, or make a test panic, without any of that
// being true of watchpost.

// probeRepo builds a tiny git repository with one rule, one test that pins it,
// and a copy of the harness under test.
func probeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module harnessprobe\n\ngo 1.25.0\n")
	write("rule.go", `package harnessprobe

// Allowed is the rule the probe's mutants delete.
func Allowed(x int) bool { return x > 0 }
`)
	write("rule_test.go", `package harnessprobe

import "testing"

func TestAllowedRefusesNegatives(t *testing.T) {
	if Allowed(-1) {
		t.Fatal("a negative was allowed")
	}
}
`)
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "06_docs", "mutants", "run.sh"))
	if err != nil {
		t.Fatalf("reading the harness: %v", err)
	}
	write("run.sh", string(src))
	if err := os.Chmod(filepath.Join(dir, "run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q", "."},
		{"add", "-A"},
		{"-c", "user.email=p@p", "-c", "user.name=p", "commit", "-qm", "probe"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

// commitAll stages everything, including UNTRACKED files, and commits.
//
// `commit -a` does not stage an untracked file, so a scenario that adds a new
// test left the probe dirty and the harness refused it before deciding anything
// — every case reported SKIPPED.
func commitAll(t *testing.T, dir, msg string) {
	t.Helper()
	for _, args := range [][]string{
		{"add", "-A"},
		{"-c", "user.email=p@p", "-c", "user.name=p", "commit", "-qm", msg},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil && !strings.Contains(string(out), "nothing to commit") {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// ask runs the harness on one mutant and returns its verdict and exit code.
func ask(t *testing.T, dir, mutant string) (string, int) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "m.py"), []byte(mutant), 0o644); err != nil {
		t.Fatal(err)
	}
	// The mutant file must not make the tree dirty, or the harness refuses before
	// it has decided anything.
	commitAll(t, dir, "mutant")
	run := exec.Command("./run.sh", "m.py", "./...")
	run.Dir = dir
	out, err := run.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("running the harness: %v", err)
	}
	verdict := ""
	for _, line := range strings.Split(string(out), "\n") {
		for _, v := range []string{"CAUGHT", "SURVIVED", "INVALID", "UNAPPLIED", "SKIPPED"} {
			if strings.HasPrefix(line, v) {
				verdict = v
			}
		}
	}
	if verdict == "" {
		t.Fatalf("the harness printed no verdict at all:\n%s", out)
	}
	return verdict, code
}

// patch is a mutant that replaces old with new in rule.go.
func patch(old, new string) string {
	return "import pathlib\np = pathlib.Path(\"rule.go\"); s = p.read_text()\nold = " +
		quote(old) + "\nnew = " + quote(new) + "\nassert old in s, \"probe\"\np.write_text(s.replace(old, new, 1))\n"
}

func quote(s string) string { return `"""` + s + `"""` }

// TestTheHarnessReportsEachVerdictWithItsExitCode.
//
// BOTH THE TEXT AND THE CODE. A driver that branches on the exit status rather
// than on stdout would have been misled by a harness whose text was right and
// whose codes were all zero, and the shell controls checked only the text.
func TestTheHarnessReportsEachVerdictWithItsExitCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, mutant, verdict string
		code                  int
	}{
		{"a deleted rule is caught", patch("return x > 0", "return true"), "CAUGHT", 0},
		{"a no-op survives", patch("// Allowed is the rule", "// Allowed is still the rule"), "SURVIVED", 1},
		{"an absent anchor is unapplied", patch("this text is not in the file", "x"), "UNAPPLIED", 2},
		{"a tree that will not compile is invalid", patch("return x > 0", "return undefinedIdentifier"), "INVALID", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := probeRepo(t)
			got, code := ask(t, dir, tc.mutant)
			if got != tc.verdict || code != tc.code {
				t.Errorf("got %s exit %d, want %s exit %d", got, code, tc.verdict, tc.code)
			}
		})
	}
}

// TestACrashIsCaughtRatherThanSurvived is the rule the shell controls did not
// cover, and the one a sabotage removed while all four still passed.
//
// A panic on a goroutine takes the test PROCESS down, so no test ever prints
// "--- FAIL" and a harness that decides by grepping for that line reports
// SURVIVED — understating coverage, which sends someone hunting for a test that
// already exists. The exit code is the honest signal.
func TestACrashIsCaughtRatherThanSurvived(t *testing.T) {
	t.Parallel()
	dir := probeRepo(t)
	// A rule whose deletion makes a test panic on another goroutine.
	if err := os.WriteFile(filepath.Join(dir, "rule_test.go"), []byte(`package harnessprobe

import "testing"

func TestAllowedRefusesNegatives(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if Allowed(-1) {
			panic("a negative was allowed")
		}
	}()
	<-done
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir, "panic probe")
	got, code := ask(t, dir, patch("return x > 0", "return true"))
	if got != "CAUGHT" || code != 0 {
		t.Errorf("a mutation that crashes the test binary reported %s exit %d, want CAUGHT exit 0 — "+
			"a crash is the suite failing, not surviving", got, code)
	}
}

// TestTheHarnessRefusesAnUnmutatedTreeThatIsAlreadyRed.
//
// A mutant is only evidence against a GREEN baseline: with a failing test in the
// tree, every mutant looks caught and the whole sweep reports coverage that is
// not there. Deleting that gate passed all four shell controls.
func TestTheHarnessRefusesAnUnmutatedTreeThatIsAlreadyRed(t *testing.T) {
	t.Parallel()
	dir := probeRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "red_test.go"), []byte(`package harnessprobe

import "testing"

func TestAlreadyRed(t *testing.T) { t.Fatal("this tree is red before any mutation") }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir, "red")
	got, _ := ask(t, dir, patch("return x > 0", "return true"))
	if got != "SKIPPED" {
		t.Errorf("against an already-red tree the harness reported %s; it must refuse, because "+
			"every mutant looks caught when something already fails", got)
	}
}

// TestTheHarnessRefusesADirtyTreeAndLeavesItAlone.
//
// It refuses because a mutant needs a clean base. It must also LEAVE THE DIRT:
// the restore trap was once armed above this check and fired on the refusal
// path, reverting the uncommitted work of whoever ran it seconds after telling
// them nothing would be done.
func TestTheHarnessRefusesADirtyTreeAndLeavesItAlone(t *testing.T) {
	t.Parallel()
	dir := probeRepo(t)
	precious := filepath.Join(dir, "rule.go")
	before, err := os.ReadFile(precious)
	if err != nil {
		t.Fatal(err)
	}
	dirty := string(before) + "\n// uncommitted work nobody asked to lose\n"
	if err := os.WriteFile(precious, []byte(dirty), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "m.py"), []byte(patch("return x > 0", "return true")), 0o644); err != nil {
		t.Fatal(err)
	}
	run := exec.Command("./run.sh", "m.py", "./...")
	run.Dir = dir
	out, _ := run.CombinedOutput()
	if !strings.Contains(string(out), "SKIPPED") {
		t.Errorf("against a dirty tree the harness said %q, want SKIPPED", strings.TrimSpace(string(out)))
	}
	after, err := os.ReadFile(precious)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != dirty {
		t.Error("the harness reverted uncommitted work on the path where it declined to do anything")
	}
}

// TestTheHarnessRestoresWhatItMutates. A sweep runs hundreds of these in a row,
// so a mutation left behind would contaminate every verdict after it.
func TestTheHarnessRestoresWhatItMutates(t *testing.T) {
	t.Parallel()
	dir := probeRepo(t)
	ask(t, dir, patch("return x > 0", "return true"))
	status := exec.Command("git", "status", "--porcelain")
	status.Dir = dir
	out, err := status.CombinedOutput()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("the harness left the tree dirty:\n%s", out)
	}
}

// TestAFlakyFailureIsNotReadAsCaught.
//
// THE GREEN-BASELINE GATE SAMPLES ONCE, AND THAT IS NOT ENOUGH. A test that
// fails 1-in-N passes the baseline and then fails on the mutated run for its own
// reasons; the harness read that as evidence about the RULE. It happened on
// 0.14.0 — mH0 was reported CAUGHT by a panicking-executor test while the
// machine was loaded, and mH0 mutates a remainder guard on a hold. The rule was
// unpinned, and the false verdict is what hid it, for long enough that the
// mutation was committed and the whole suite went green over it.
//
// The flake is modelled deterministically with a counter on disk: pass once (the
// baseline), fail afterwards (the mutated run, and the attribution re-run). A
// harness that trusts the first failure calls this CAUGHT; one that asks whether
// the same test fails WITHOUT the mutation cannot.
func TestAFlakyFailureIsNotReadAsCaught(t *testing.T) {
	t.Parallel()
	dir := probeRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "flaky_test.go"), []byte(`package harnessprobe

import (
	"os"
	"strconv"
	"testing"
)

// TestFlaky passes the first time it is ever run in this tree and fails after,
// which is what an intermittent test looks like to a single-sample gate.
func TestFlaky(t *testing.T) {
	const counter = "flaky.count"
	n := 0
	if b, err := os.ReadFile(counter); err == nil {
		n, _ = strconv.Atoi(string(b))
	}
	if err := os.WriteFile(counter, []byte(strconv.Itoa(n+1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if n > 0 {
		t.Fatalf("flaky failure on run %d, nothing to do with any mutation", n+1)
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	// The counter file must not make the tree dirty when it appears.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("flaky.count\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir, "flaky")

	// A mutation the probe's own pin DOES catch, so the only thing deciding the
	// verdict is which failure the harness believes.
	got, _ := ask(t, dir, patch("return x > 0", "return true"))
	if got == "CAUGHT" {
		t.Error("a flaky failure was reported as a catch: the harness credited the mutation with a " +
			"failure that had nothing to do with it, which is how an unpinned rule reads as covered")
	}
	if got != "INVALID" {
		t.Errorf("an unattributable failure is not evidence either way; the harness said %s", got)
	}
}
