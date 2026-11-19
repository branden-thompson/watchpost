package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plantedModule writes a module with the repository's layout: a model in
// modes/tty, a caller in app, the binary linking both, and a test-support
// package the binary does not link.
func plantedModule(t *testing.T, tty, appSrc string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":                          "module github.com/branden-thompson/watchpost\n\ngo 1.26\n",
		"modes/tty/model.go":              tty,
		"app/app.go":                      appSrc,
		"app/internal/apptest/apptest.go": "package apptest\n\nvar Sink any\n\nfunc Box(n int) { Sink = &n }\n",
		"cmd/watchpost/main.go":           "package main\n\nimport \"github.com/branden-thompson/watchpost/app\"\n\nfunc main() { app.Run() }\n",
	}
	for name, src := range files { // bounded by the planted files (P10-02)
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// ttySrc is a model whose Update returns it by value and whose small
// method inlines.
const ttySrc = `package tty

// Model is a planted model.
type Model struct{ n int }

func (m Model) width() int { return m.n * 2 }

// Update steps the model.
func (m Model) Update() (Model, int) {
	m.n++
	return m, m.width()
}
`

// appSrc calls the model.
const appSrc = `package app

import "github.com/branden-thompson/watchpost/modes/tty"

var sink any

// Run steps a model.
func Run() {
	var m tty.Model
	m, w := m.Update()
	sink = w
	_ = m
}
`

// TestTheProbeHoldsEscapesAndInlining is 0.19.6 B0.7's probe on a planted
// module: recorded, the same tree compares clean; a value boxed where it was
// not fails as a new escape of its type; a method that stops inlining fails
// as a lost inlining; and a rename map carries a moved function's keys.
func TestTheProbeHoldsEscapesAndInlining(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a planted module")
	}
	dir := plantedModule(t, ttySrc, appSrc)
	esc, inl := filepath.Join(t.TempDir(), "esc.tsv"), filepath.Join(t.TempDir(), "inl.tsv")
	var out, errs bytes.Buffer
	if code := run([]string{"record", "-dir", dir, "-tree", "planted", "-escapes", esc, "-inlining", inl}, &out, &errs); code != 0 {
		t.Fatalf("record exited %d: %s%s", code, out.String(), errs.String())
	}
	_, baseInl, err := readCounts(inl)
	if err != nil {
		t.Fatal(err)
	}
	if baseInl["app\tRun\ttty.Model.Update"] == 0 || baseInl["modes/tty\tModel.Update\tModel.width"] == 0 {
		t.Fatalf("the planted inlined calls were not keyed: %v", baseInl)
	}
	_, baseEsc, err := readCounts(esc)
	if err != nil {
		t.Fatal(err)
	}
	for k := range baseEsc { // bounded by the keys (P10-02)
		if strings.Contains(k, "apptest") {
			t.Errorf("a package the binary does not link was judged: %q", k)
		}
	}
	compare := func(dir, mapFile string) (int, string) {
		var out, errs bytes.Buffer
		args := []string{"compare", "-dir", dir, "-escapes", esc, "-inlining", inl}
		if mapFile != "" {
			args = append(args, "-map", mapFile)
		}
		code := run(args, &out, &errs)
		return code, out.String() + errs.String()
	}
	if code, said := compare(dir, ""); code != 0 {
		t.Fatalf("the recorded tree did not compare clean: %s", said)
	}
	boxed := plantedModule(t, strings.Replace(ttySrc, "\tm.n++\n", "\tm.n++\n\tkeep = &m\n", 1)+"\nvar keep *Model\n", appSrc)
	if code, said := compare(boxed, ""); code == 0 || !strings.Contains(said, "escape modes/tty | Model.Update | Model") {
		t.Errorf("a model moved to the heap was not a new escape of its type: exit %d, %s", code, said)
	}
	lost := plantedModule(t, strings.Replace(ttySrc, "func (m Model) width()", "//go:noinline\nfunc (m Model) width()", 1), appSrc)
	if code, said := compare(lost, ""); code == 0 || !strings.Contains(said, "inlining modes/tty | Model.Update | Model.width") {
		t.Errorf("a method that stopped inlining was not a lost inlining: exit %d, %s", code, said)
	}
	renamedSrc := strings.ReplaceAll(strings.ReplaceAll(ttySrc, "Update", "Step"), "// Step steps", "// Step steps")
	moved := plantedModule(t, renamedSrc, strings.ReplaceAll(appSrc, "m.Update()", "m.Step()"))
	if code, _ := compare(moved, ""); code == 0 {
		t.Error("a renamed function compared clean with no map: its keys are new")
	}
	m := filepath.Join(t.TempDir(), "map.tsv")
	if err := os.WriteFile(m, []byte("# Update is Step\nUpdate\tStep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, said := compare(moved, m); code != 0 {
		t.Errorf("the rename map did not carry the moved function's keys: %s", said)
	}
}

// TestADiagnosticIsSplit holds the three forms the compiler prints.
func TestADiagnosticIsSplit(t *testing.T) {
	for _, c := range []struct {
		line        string
		file, msg   string
		lineNo, col int
		ok          bool
	}{
		{"modes/tty/router.go:463:9: out escapes to heap", "modes/tty/router.go", "out escapes to heap", 463, 9, true},
		{"<autogenerated>:1: inlining call to Dashboard.settled", "<autogenerated>", "inlining call to Dashboard.settled", 1, 0, true},
		{"# github.com/branden-thompson/watchpost/modes/tty", "", "", 0, 0, false},
	} {
		file, lineNo, col, msg, ok := splitDiagnostic(c.line)
		if file != c.file || lineNo != c.lineNo || col != c.col || msg != c.msg || ok != c.ok {
			t.Errorf("%q split as %q %d %d %q %v", c.line, file, lineNo, col, msg, ok)
		}
	}
}

// TestAnExternalFileIsKeyedWithoutItsVersion holds the keys for code from
// the module cache: the standard library as std/, a module without its
// version, so no key reads as an email address and an upgrade renames none.
func TestAnExternalFileIsKeyedWithoutItsVersion(t *testing.T) {
	for in, want := range map[string]string{
		"../../go/pkg/mod/golang.org/toolchain" + "@" + "v0.0.1-go1.27.4.darwin-arm64/src/slices/sort.go": "std/slices/sort.go",
		"../../go/pkg/mod/charm.land/bubbletea/v2@v2.0.3/commands.go":                                     "charm.land/bubbletea/v2/commands.go",
	} {
		if got := external(in); got != want || strings.Contains(got, "@") {
			t.Errorf("external(%q) = %q; want %q", in, got, want)
		}
	}
}

// TestTheJudgeHoldsEachRule holds the four rules on planted counts: a new
// escape key, a type escaping more across the trees, a lost inlining key, and
// a fall in the trees' inlined total.
func TestTheJudgeHoldsEachRule(t *testing.T) {
	baseEsc := counts{"modes/tty\tUpdate\t*Model": 1}
	baseInl := counts{"app\tRun\ttty.Model.Update": 2}
	same := &probe{escapes: counts{"modes/tty\tUpdate\t*Model": 1}, inlining: counts{"app\tRun\ttty.Model.Update": 2}}
	if got := judge(baseEsc, baseInl, same); len(got) != 0 {
		t.Errorf("the baseline itself failed: %q", got)
	}
	more := &probe{escapes: counts{"modes/tty\tUpdate\t*Model": 1, "modes/tty\tView\t*Model": 1}, inlining: baseInl}
	if got := strings.Join(judge(baseEsc, baseInl, more), "\n"); !strings.Contains(got, "escape modes/tty | View | *Model") || !strings.Contains(got, "type *Model escapes 2 times") {
		t.Errorf("a new escape: %q", got)
	}
	fewer := &probe{escapes: baseEsc, inlining: counts{"app\tRun\ttty.Model.Update": 1}}
	if got := strings.Join(judge(baseEsc, baseInl, fewer), "\n"); !strings.Contains(got, "inlining app | Run | tty.Model.Update: 1, against 2") || !strings.Contains(got, "inline 1 calls, against 2") {
		t.Errorf("a lost inlining: %q", got)
	}
}

// TestRenamesMergeKeys holds 0.19.6 A-26's union: two old functions mapped to one
// new name are counted together.
func TestRenamesMergeKeys(t *testing.T) {
	c := counts{"app\tbuildA\tx": 1, "app\tbuildB\tx": 2}
	got := renamed(c, [][2]string{{"buildA", "build"}, {"buildB", "build"}})
	if len(got) != 1 || got["app\tbuild\tx"] != 3 {
		t.Errorf("renamed %v", got)
	}
}

// TestTheTreesKeepWhatTheCompilerDecided holds the repository to the
// baselines recorded on e57afeaf: no new escape, no lost inlining (0.19.6
// B0.7, PB-2, PB-3). On another platform the comparison says so and passes,
// and under another Go it fails until the baselines are recorded again.
func TestTheTreesKeepWhatTheCompilerDecided(t *testing.T) {
	if testing.Short() {
		t.Skip("builds both trees")
	}
	var out, errs bytes.Buffer
	code := run([]string{"compare", "-dir", "../..", "-escapes", "../../06_docs/perf/escapes-e57afeaf.tsv", "-inlining", "../../06_docs/perf/inlining-e57afeaf.tsv"}, &out, &errs)
	t.Log(strings.TrimSpace(out.String()))
	if code != 0 {
		t.Errorf("the trees left the compiler's baselines (exit %d):\n%s%s", code, out.String(), errs.String())
	}
}
