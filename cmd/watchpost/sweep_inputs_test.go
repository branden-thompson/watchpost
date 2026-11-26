package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// sweepScript is the corpus sweep, run from the repository's root. A test may
// point it at another copy of the script, to see a check fail before it lands.
var sweepScript = "scripts/quality/mutant-verdicts.sh"

// TestTheSweepRefusesInputsItCannotHonour is 0.19.6 A-2 and H-4: a killer
// record the sweep is told to read and cannot find, and a mutant a scoped list
// names that the corpus lacks, each stop the run at once and say so. Without
// the first, every mutant starts at its whole package (about 19 hours where
// the record makes it one) and nothing says why; without the second, a scoped
// sweep judges less than its list and reports the rest as never derived.
func TestTheSweepRefusesInputsItCannotHonour(t *testing.T) {
	root := filepath.Join("..", "..")
	list := filepath.Join(t.TempDir(), "list")
	if err := os.WriteFile(list, []byte("06_docs/mutants/m_no_such_mutant.py\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		env  []string
		want string
	}{
		{"a named killer record that is missing", []string{"MUTANT_KILLERS=" + filepath.Join(t.TempDir(), "since-v9.9.9.log"), "MUTANT_ONLY=^$"}, "is missing"},
		{"a listed mutant the corpus lacks", []string{"MUTANT_LIST=" + list}, "MUTANT_LIST names mutants the corpus lacks"},
	}
	for _, c := range cases {
		out := filepath.Join(t.TempDir(), "sweep.log")
		cmd := exec.Command("sh", sweepScript, out)
		cmd.Dir = root
		cmd.Env = append(withoutSweepInputs(os.Environ()), c.env...)
		combined, err := cmd.CombinedOutput()
		if err == nil {
			t.Errorf("%s: the sweep ran; want it refused", c.name)
			continue
		}
		if !strings.Contains(string(combined), c.want) {
			t.Errorf("%s: the sweep said %q; want it to say %q", c.name, strings.TrimSpace(string(combined)), c.want)
		}
	}
}

// TestTheSweepLeavesTheUnionOut is 0.19.6 D-84: the union fails on every
// mutant under app and modes/tty, so a catch of its says only that the source
// changed. The sweep's MUTANT_SKIP must name it, by the name the function has,
// so a rename that the pattern misses fails here.
func TestTheSweepLeavesTheUnionOut(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", sweepScript))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^export MUTANT_SKIP='([^']+)'$`).FindSubmatch(raw)
	if m == nil {
		t.Fatalf("%s sets no MUTANT_SKIP, so the union can catch a mutant", sweepScript)
	}
	full := runtime.FuncForPC(reflect.ValueOf(TestTheTwoTreesDeclarationsAreConserved).Pointer()).Name()
	name := full[strings.LastIndex(full, ".")+1:]
	if !regexp.MustCompile(string(m[1])).MatchString(name) {
		t.Fatalf("%s skips %q, which does not match the union's name %s", sweepScript, m[1], name)
	}
}

// withoutSweepInputs is env with every MUTANT_ variable dropped, so each case
// sets the sweep's inputs alone. A scoped sweep's baseline runs this test in a
// worktree with the sweep's own MUTANT_KILLERS set, a path relative to the
// repository that the worktree lacks: the second case inherited it and was
// refused as a missing record, not as a mutant the corpus lacks (0.19.6 A-48).
func withoutSweepInputs(env []string) []string {
	kept := make([]string, 0, len(env))
	for _, kv := range env { // bounded by the environment (P10-02)
		if !strings.HasPrefix(kv, "MUTANT_") {
			kept = append(kept, kv)
		}
	}
	return kept
}
