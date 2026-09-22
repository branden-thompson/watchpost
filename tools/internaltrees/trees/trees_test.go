package trees

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func mustExpr(t *testing.T, root, home string) *regexp.Regexp {
	t.Helper()
	expr, err := Expr(root, home)
	if err != nil {
		t.Fatalf("Expr(%q, %q): %v", root, home, err)
	}
	return regexp.MustCompile(expr)
}

// THE SHAPES HOLD WITH NO WORKSPACE AROUND THE CHECKOUT, which is what a CI
// runner is. Every line is a positive or a negative control: a pattern that
// fired on nothing would pass the whole-index scan exactly when the tree had
// become clean enough to trust it.
func TestShapesFireWithoutAWorkspace(t *testing.T) {
	re := mustExpr(t, ".", t.TempDir())
	for _, leak := range []string{ // bounded by the table (P10-02)
		"LI_PROJECTS/somewhere",        // a retired name, still in older notes
		"WORK-CODE/04__TEAM/some-tool", // a numbered bucket under a renamed root
		"07__SOME_BUCKET/some-repo",    // a numbered bucket nobody has created yet
		"cd ~/x/09-ARCHIVE/y && make",  // a dash-numbered root as a path segment
	} {
		if !re.MatchString(leak) {
			t.Errorf("the internal-tree rule did not fire on %q", leak)
		}
	}
	for _, clean := range []string{ // bounded by the table (P10-02)
		"ratified 2026-07-14",
		"P10-02 bounds this loop",
		"06_docs/p10-ledger.md",
		"platform/render/units.go",
	} {
		if re.MatchString(clean) {
			t.Errorf("the internal-tree rule fired on %q — a gate that flags ordinary text trains authors to add exemptions", clean)
		}
	}
}

// THE NAMES ARE READ FROM THE DISK, so a workspace nobody has seen before is
// described on its first run. The fixture holds no numbered folder, so only
// derivation — not the static shapes — can make these fire.
func TestNamesAreReadFromTheDisk(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "Desk", "WORKSPACE", "SIDE-PROJ", "demo-repo")
	for _, d := range []string{ // bounded by the table (P10-02)
		repo,
		filepath.Join(home, "Desk", "WORKSPACE", "ACME-INTERNAL"),
		filepath.Join(home, "Library"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	re := mustExpr(t, repo, home)
	staticRe := mustExpr(t, repo, t.TempDir())

	for _, leak := range []string{"WORKSPACE/a", "ACME-INTERNAL/b", "see SIDE-PROJ/c"} { // bounded (P10-02)
		if !re.MatchString(leak) {
			t.Errorf("the derived rule did not fire on %q", leak)
		}
		if staticRe.MatchString(leak) {
			t.Errorf("the static rule fired on %q, so this control does not prove derivation", leak)
		}
	}
	for _, clean := range []string{"demo-repo/cmd", "Library/Caches", "ACME-INTERNAL as a word"} { // bounded (P10-02)
		if re.MatchString(clean) {
			t.Errorf("the derived rule fired on %q — the repository's own name, a first-level home folder, "+
				"or a bare word is not a leak", clean)
		}
	}
}

// A CHECKOUT OUTSIDE HOME, OR TOO SHALLOW TO HAVE A WORKSPACE, DERIVES NOTHING —
// and still gets the static shapes rather than an error.
func TestNoWorkspaceDerivesNothing(t *testing.T) {
	home := t.TempDir()
	shallow := filepath.Join(home, "Desk", "repo")
	if err := os.MkdirAll(shallow, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ root, home string }{ // bounded by the table (P10-02)
		{shallow, home},
		{t.TempDir(), home},
		{shallow, ""},
	} {
		names, err := Derived(c.root, c.home)
		if err != nil || len(names) != 0 {
			t.Errorf("Derived(%q, %q) = %v, %v; want nothing", c.root, c.home, names, err)
		}
	}
}

// TestAnEmptyPathIsRefusedRatherThanGuessed is the positive control for the
// guards added with them: an empty root would otherwise resolve to whatever
// directory the process is in, and the rule would describe a workspace nobody
// asked about. Each case must FAIL.
func TestAnEmptyPathIsRefusedRatherThanGuessed(t *testing.T) {
	home := t.TempDir()

	if _, err := Expr("", home); err == nil {
		t.Error("Expr accepted an empty repository root")
	}
	if _, err := Derived("", home); err == nil {
		t.Error("Derived accepted an empty repository root")
	}
	if _, err := resolve(""); err == nil {
		t.Error("resolve accepted an empty path")
	}

	// The control the other way: a real root is still accepted, so the guards
	// above cannot pass by refusing everything.
	if _, err := Expr(".", home); err != nil {
		t.Errorf("Expr refused a real root: %v", err)
	}
}
