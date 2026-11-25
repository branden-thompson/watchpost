package main

import (
	"bytes"
	"maps"
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

// TestAToolchainOutsideTheModuleCacheIsRefused holds 0.19.6 D-75: the probe
// runs only under a go whose standard library external keys as std/, the
// module cache's toolchain, and refuses a system go of the same version, whose
// library it would key by base name. Seen failing as 154 keys new and lost
// under Homebrew's go 1.27.4 before the refusal.
func TestAToolchainOutsideTheModuleCacheIsRefused(t *testing.T) {
	if err := keyedRoot(filepath.Join("home", "go", "pkg", "mod", "golang.org", "toolchain"+"@"+"v0.0.1-go1.27.4.darwin-arm64")); err != nil {
		t.Errorf("the module cache's toolchain was refused: %v", err)
	}
	if err := keyedRoot(filepath.Join("opt", "homebrew", "Cellar", "go", "1.27.4", "libexec")); err == nil {
		t.Error("a system go's standard library was accepted; its keys would name it by base name")
	}
}

// TestTheJudgeHoldsEachRule holds the four rules on planted counts: a new
// escape key, a type escaping more across the trees, a lost inlining key, and
// a fall in the trees' inlined total.
func TestTheJudgeHoldsEachRule(t *testing.T) {
	baseEsc := counts{"modes/tty\tUpdate\t*Model": 1}
	baseInl := counts{"app\tRun\ttty.Model.Update": 2}
	same := &probe{escapes: counts{"modes/tty\tUpdate\t*Model": 1}, inlining: counts{"app\tRun\ttty.Model.Update": 2}}
	if got := judge(baseEsc, baseInl, same, nil); len(got) != 0 {
		t.Errorf("the baseline itself failed: %q", got)
	}
	more := &probe{escapes: counts{"modes/tty\tUpdate\t*Model": 1, "modes/tty\tView\t*Model": 1}, inlining: baseInl}
	if got := strings.Join(judge(baseEsc, baseInl, more, nil), "\n"); !strings.Contains(got, "escape modes/tty | View | *Model") || !strings.Contains(got, "type *Model escapes 2 times") {
		t.Errorf("a new escape: %q", got)
	}
	fewer := &probe{escapes: baseEsc, inlining: counts{"app\tRun\ttty.Model.Update": 1}}
	if got := strings.Join(judge(baseEsc, baseInl, fewer, nil), "\n"); !strings.Contains(got, "inlining app | Run | tty.Model.Update: 1, against 2") || !strings.Contains(got, "inline 1 calls, against 2") {
		t.Errorf("a lost inlining: %q", got)
	}
}

// TestAnExcuseCoversItsKeyAlone holds 0.19.6 D-82's list: an excused key may
// reach its count and no further, a key outside the list fails as before,
// and a line whose key is back within its baseline fails as stale.
func TestAnExcuseCoversItsKeyAlone(t *testing.T) {
	baseEsc := counts{"modes/tty\tUpdate\t*Model": 1}
	baseInl := counts{"modes/tty\trender\topts": 3}
	moved := &probe{escapes: counts{"modes/tty\tUpdate\t*Model": 1, "modes/tty\tcarved\ttea.Cmd": 2}, inlining: counts{"modes/tty\trender\topts": 0, "modes/tty\tcarved\tx": 3}}
	ex := []excuse{{"escape", "modes/tty\tcarved\ttea.Cmd", 2, "moved"}, {"type", "tea.Cmd", 2, "moved"}, {"inlining", "modes/tty\trender\topts", 0, "seam"}}
	if got := judge(baseEsc, baseInl, moved, ex); len(got) != 0 {
		t.Errorf("the excused keys failed: %q", got)
	}
	if got := strings.Join(judge(baseEsc, baseInl, moved, nil), "\n"); !strings.Contains(got, "escape modes/tty | carved | tea.Cmd") || !strings.Contains(got, "inlining modes/tty | render | opts") {
		t.Errorf("without the list the keys pass: %q", got)
	}
	further := &probe{escapes: counts{"modes/tty\tUpdate\t*Model": 1, "modes/tty\tcarved\ttea.Cmd": 3}, inlining: moved.inlining}
	if got := strings.Join(judge(baseEsc, baseInl, further, ex), "\n"); !strings.Contains(got, "escape modes/tty | carved | tea.Cmd: 3") {
		t.Errorf("an excused key past its count passed: %q", got)
	}
	other := &probe{escapes: counts{"modes/tty\tUpdate\t*Model": 1, "modes/tty\tcarved\ttea.Cmd": 2, "modes/tty\tView\t*Model": 1}, inlining: moved.inlining}
	if got := strings.Join(judge(baseEsc, baseInl, other, ex), "\n"); !strings.Contains(got, "escape modes/tty | View | *Model") {
		t.Errorf("a key outside the list passed: %q", got)
	}
	back := &probe{escapes: baseEsc, inlining: baseInl}
	if got := strings.Join(judge(baseEsc, baseInl, back, ex), "\n"); strings.Count(got, "needs no excuse") != 3 {
		t.Errorf("stale lines passed: %q", got)
	}
}

// TestRenamesMergeKeys holds 0.19.6 A-26's union: two old functions mapped to one
// new name are counted together.
func TestRenamesMergeKeys(t *testing.T) {
	c := counts{"app\tbuildA\tx": 1, "app\tbuildB\tx": 2}
	got := renamed(c, []rename{{old: "buildA", new: "build"}, {old: "buildB", new: "build"}})
	if len(got) != 1 || got["app\tbuild\tx"] != 3 {
		t.Errorf("renamed %v", got)
	}
}

// TestTheTreesKeepWhatTheCompilerDecided holds the repository to the
// baselines recorded on e57afeaf: no new escape, no lost inlining (0.19.6
// B0.7, PB-2, PB-3), through the release's renames, 06_docs/perf/renames.tsv
// (0.19.6 A-44), its package moves, moves.tsv (A-52), and the reviewed list
// of keys no rename carries, excused-B4.tsv (D-82). On another platform the comparison says so and passes,
// and under another Go it fails until the baselines are recorded again.
func TestTheTreesKeepWhatTheCompilerDecided(t *testing.T) {
	if testing.Short() {
		t.Skip("builds both trees")
	}
	var out, errs bytes.Buffer
	code := run([]string{"compare", "-dir", "../..", "-escapes", "../../06_docs/perf/escapes-e57afeaf.tsv", "-inlining", "../../06_docs/perf/inlining-e57afeaf.tsv", "-map", "../../06_docs/perf/renames.tsv", "-moves", "../../06_docs/perf/moves.tsv", "-excused", "../../06_docs/perf/excused-B4.tsv"}, &out, &errs)
	t.Log(strings.TrimSpace(out.String()))
	if code != 0 {
		t.Errorf("the trees left the compiler's baselines (exit %d):\n%s%s", code, out.String(), errs.String())
	}
}

// TestARenameReplacesOnlyAWholeName holds 0.19.6 A-49: a rename of a bare
// function name renames that function's keys and no name it is part of, so
// B2's `tick` becoming `Tick` leaves the ticker's keys and `sidesOnly` alone,
// and a line scoped to modes/tty leaves app's method `schedule.tick`. Seen
// failing under a plain text replace, which made `tickerRow` `TickerRow`, and
// with no scope, which made `schedule.tick` `schedule.Tick`.
func TestARenameReplacesOnlyAWholeName(t *testing.T) {
	c := counts{
		"modes/tty\ttick\ttickMsg":                 1,
		"modes/tty\ttick.func1\ttickMsg":           1,
		"modes/tty\ttickerRow\tstring":             1,
		"modes/tty\tDashboard.armTick\ttickMsg":    1,
		"modes/tty\tsidesOnly\tstrings.IndexByte":  1,
		"modes/tty\tcursorIn\tsides":               1,
		"app\t(*mastercontrol).dip\tsync.(*Mutex)": 1,
		"app\tschedule.tick\tlineup.Tick":          1,
	}
	got := renamed(c, []rename{{"tick", "Tick", "modes/tty"}, {"tickMsg", "TickMsg", "modes/tty"}, {"sides", "Sides", "modes/tty"}, {"(*mastercontrol).dip", "(*mastercontrol).duck", ""}})
	want := counts{
		"modes/tty\tTick\tTickMsg":                  1,
		"modes/tty\tTick.func1\tTickMsg":            1,
		"modes/tty\ttickerRow\tstring":              1,
		"modes/tty\tDashboard.armTick\tTickMsg":     1,
		"modes/tty\tsidesOnly\tstrings.IndexByte":   1,
		"modes/tty\tcursorIn\tSides":                1,
		"app\t(*mastercontrol).duck\tsync.(*Mutex)": 1,
		"app\tschedule.tick\tlineup.Tick":           1,
	}
	if !maps.Equal(got, want) {
		t.Errorf("renamed\n%v\nwant\n%v", got, want)
	}
}

// movedFixture is a tree where package a's Plural, Slot and Mark moved to
// a/b, and its move map.
func movedFixture(t *testing.T) (dir, file string, moves []move) {
	t.Helper()
	dir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, src := range map[string]string{
		"a.go":        "package a\n",
		"b/b.go":      "package b\n\nfunc Plural(n int) string { return \"\" }\n\ntype Slot struct{}\n\nfunc (s *Slot) Get() string { return \"\" }\n\nconst Mark = \"x\"\n",
		"b/b_test.go": "package b\n\nfunc Header() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, "a", name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	file = filepath.Join(dir, "moves.tsv")
	if err := os.WriteFile(file, []byte("# a comment\na\ta/b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	moves, err := readMoves(dir, file)
	if err != nil {
		t.Fatal(err)
	}
	return dir, file, moves
}

// TestAMoveCarriesItsNamesKeysToTheirNewPackage holds 0.19.6 A-52: a move
// line names two packages, and the names that moved are read from the
// destination's source. A moved function's and method's keys become the new
// package's, its old package's calls to it read qualified as the compiler
// writes them, a bare name after a dot or in another package is left alone,
// and a key no function owns is counted across both packages, on either side.
// A destination whose files name two packages is refused.
func TestAMoveCarriesItsNamesKeysToTheirNewPackage(t *testing.T) {
	dir, file, moves := movedFixture(t)
	base := counts{
		"a\tPlural\tstrconv.Itoa":            1,
		"a\tSlot.Get\tstring":                1,
		"a\t(*Slot).Get\tstring":             1,
		"a\tDashboard.header\t(*Slot).Get":   1,
		"a\tDashboard.view\t*Slot":           2,
		"a\tDashboard.sourceLines\tPlural":   1,
		"a\tDashboard.Plural\tstring":        1,
		"a\tHeader\tstring":                  1,
		"a\t<autogenerated>\t~&tea.Cursor{}": 1,
		"c\tother\tPlural":                   1,
	}
	want := counts{
		"a/b\tPlural\tstrconv.Itoa":          1,
		"a/b\tSlot.Get\tstring":              1,
		"a/b\t(*Slot).Get\tstring":           1,
		"a\tDashboard.header\tb.(*Slot).Get": 1,
		"a\tDashboard.view\t*b.Slot":         2,
		"a\tDashboard.sourceLines\tb.Plural": 1,
		"a\tDashboard.Plural\tstring":        1,
		"a\tHeader\tstring":                  1,
		"a\t<autogenerated>\t~&tea.Cursor{}": 1,
		"c\tother\tPlural":                   1,
	}
	if got := moved(base, moves, true); !maps.Equal(got, want) {
		t.Errorf("the baseline moved\n%v\nwant\n%v", got, want)
	}
	tree := counts{"a/b\t<autogenerated>\tPlural": 1, "a/b\tcharm.land/x/y.go\t~Mark": 1, "a/b\tPlural\tstring": 1, "a\tDashboard.view\t*b.Slot": 1}
	wantTree := counts{"a\t<autogenerated>\tb.Plural": 1, "a\tcharm.land/x/y.go\t~b.Mark": 1, "a/b\tPlural\tstring": 1, "a\tDashboard.view\t*b.Slot": 1}
	if got := moved(tree, moves, false); !maps.Equal(got, wantTree) {
		t.Errorf("the tree moved\n%v\nwant\n%v", got, wantTree)
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "b", "c.go"), []byte("package c\n\nconst N = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"a\n", "a\ta\n", "a\ta/none\n", "a\tb\tc\n", "a\ta/b\n"} {
		if err := os.WriteFile(file, []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := readMoves(dir, file); err == nil {
			t.Errorf("readMoves took %q", bad)
		}
	}
}

// TestAThirdPackageNamesAMovedNameByItsNewPackage: a key in a package that
// imports the moved code names it as the compiler does, through the package
// it is in. The baseline's c reads a.Plural and a.(*Slot).Get, the tree's
// b.Plural and b.(*Slot).Get, and a name that did not move keeps its package
// (0.19.6 A-60).
func TestAThirdPackageNamesAMovedNameByItsNewPackage(t *testing.T) {
	_, _, moves := movedFixture(t)
	base := counts{
		"c	other	a.Plural":                1,
		"c	other.view	*a.Slot":            1,
		"c	<autogenerated>	a.(*Slot).Get": 1,
		"c	other.keep	a.Dashboard":        1,
		"c	other.own	data.Plural":         1,
		"c	Plural	string":                 1,
	}
	want := counts{
		"c	other	b.Plural":                1,
		"c	other.view	*b.Slot":            1,
		"c	<autogenerated>	b.(*Slot).Get": 1,
		"c	other.keep	a.Dashboard":        1,
		"c	other.own	data.Plural":         1,
		"c	Plural	string":                 1,
	}
	if got := moved(base, moves, true); !maps.Equal(got, want) {
		t.Errorf("the baseline moved\n%v\nwant\n%v", got, want)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b", "b.go"), []byte("package b\n\nconst N = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "moves.tsv")
	if err := os.WriteFile(file, []byte("none\tb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readMoves(dir, file); err == nil {
		t.Error("readMoves took a move from a directory with no package to name")
	}
}

// TestAPackageScopeEscapeIsCountedAcrossTheMove: a package-level variable's
// initializer is a key no function owns, so which of from's package-scope
// escapes a move takes no name says; they are counted across both packages,
// as a compiler wrapper is (0.19.6 A-60).
func TestAPackageScopeEscapeIsCountedAcrossTheMove(t *testing.T) {
	_, _, moves := movedFixture(t)
	base := counts{"a\tpackage scope\t[]string": 8, "a\tpackage scope\tmap[Mark]bool": 1}
	tree := counts{"a\tpackage scope\t[]string": 2, "a/b\tpackage scope\t[]string": 6, "a/b\tpackage scope\tmap[Mark]bool": 1}
	if got, want := moved(base, moves, true), moved(tree, moves, false); !maps.Equal(got, want) {
		t.Errorf("the baseline moved\n%v\nthe tree moved\n%v", got, want)
	}
}
