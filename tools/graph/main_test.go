package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// planted is a module with the repository's layout and the graph's inputs:
// the model in modes/tty, a map file and its test there, the map's package
// reading the model through a carrier, and app using the hub and crossing
// its planned packages.
var planted = map[string]string{
	"go.mod": "module github.com/branden-thompson/watchpost\n\ngo 1.26\n",
	"modes/tty/dashboard.go": `package tty

// Dashboard is the planted hub.
type Dashboard struct{ Cfg int; width int }

// Config is a second model type with a field named as one of the hub's.
type Config struct{ Cfg int }

func (d Dashboard) frame() int { return d.width }
`,
	"modes/tty/map_draw.go": `package tty

func (d Dashboard) mapLine() int { return d.width + d.frame() + mapInner() }

func mapInner() int { return 1 }
`,
	"modes/tty/map_draw_test.go": `package tty

import "testing"

func mapHelper(d Dashboard) int { return mapInner() + d.frame() }

func TestTheMapMovesWithItsCode(t *testing.T) { _ = mapHelper(Dashboard{}) }

func TestTheMapReadsTheFrame(t *testing.T) { var d Dashboard; _ = mapInner() + d.width }
`,
	"modes/tty/mapview/map_env.go": `package mapview

import "github.com/branden-thompson/watchpost/modes/tty"

// Env is the planted carrier: what the map is passed in place of the model.
type Env struct{ Width, Height int }

// Draw reads the model through the carrier and through the hub itself.
func Draw(e Env, d *tty.Dashboard, c tty.Config) int { return e.Width + d.Cfg + c.Cfg }
`,
	"app/app.go": `package app

import "github.com/branden-thompson/watchpost/modes/tty"

// Run uses the hub from another package.
func Run() tty.Dashboard { return tty.Dashboard{Cfg: helper()} }
`,
	"app/helper.go": "package app\n\nfunc helper() int { return 1 }\n",
	// the two deck-only test files: the first reaches the deck alone; the
	// second, through a helper in another test file, reaches the root's Run
	"app/cutback_hazard_test.go": "package app\n\nimport \"testing\"\n\nfunc TestDeckOnly(t *testing.T) { _ = helper() }\n",
	"app/deck_trace_test.go":     "package app\n\nimport \"testing\"\n\nfunc TestReachesTheRoot(t *testing.T) { _ = viaTheRoot() }\n",
	"app/support_test.go":        "package app\n\nfunc viaTheRoot() int { return Run().Cfg + helper() }\n",
	crossingsMS:                  "direction\tsymbol\tkind\tdeclaring_file\tdisposition\tnote\ntty->mapview\tmapInner\tfunc\tmodes/tty/map_draw.go\tMOVES\tplanted\n",
	allowMS:                      "# planted\nDashboard\nDashboard.frame\n",
	plannedApp:                   "# planted\nfile\tapp/app.go\tapp\nfile\tapp/helper.go\tdeck\n",
}

// plantedTree writes the planted module and returns its root.
func plantedTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range planted { // bounded by the planted files (P10-02)
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

// TestAPassThroughCountsWhatItPasses is 0.19.6 A-12 and CQ F-17 on a planted
// module: the map's package reads the model through a carrier and through the
// hub, across packages, and each field reached counts. Without the carrier on
// the area's row, the carried field is not counted: the pass-through would
// read as less reach than the model it passes.
func TestAPassThroughCountsWhatItPasses(t *testing.T) {
	ix, err := load(plantedTree(t))
	if err != nil {
		t.Fatal(err)
	}
	a := area{name: "planted map", readers: []string{"modes/tty/mapview/map_"}, model: []string{"modes/tty/dashboard.go"}, carriers: []string{module + "/modes/tty/mapview.Env"}}
	got := ix.reach(a)
	for _, want := range []string{".Width", ".Cfg", "modes/tty/dashboard.go:Dashboard", "modes/tty/dashboard.go:Config"} { // the planted reach (P10-02)
		if !got[want] {
			t.Errorf("the map's package's reach lacks %q: %v", want, got)
		}
	}
	if len(got) != 4 {
		t.Errorf("reach %v; want the 4 planted: Dashboard.Cfg and Config.Cfg are one field by name (D-44)", got)
	}
	a.carriers = nil
	if got := ix.reach(a); got[".Width"] || len(got) != 3 {
		t.Errorf("with no carrier the carried field still counts: %v", got)
	}
}

// TestTheFiguresOnAPlantedTree holds the other figures on the planted
// module: the fan-in across packages, the methods, MS-7's classes through a
// helper, and a crossing under the planned map.
func TestTheFiguresOnAPlantedTree(t *testing.T) {
	ix, err := load(plantedTree(t))
	if err != nil {
		t.Fatal(err)
	}
	ix.inputs = ix.dir
	figs, m, crossed, err := measure(ix)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, f := range figs { // bounded by the figures (P10-02)
		got[f.name] = f.value
	}
	for name, want := range map[string]int{
		"hub fan-in (files using Dashboard)":             4, // dashboard.go, map_draw.go, mapview/map_env.go, app/app.go
		"Dashboard methods":                              2,
		"Dashboard method files":                         2,
		"MS-7 tests reaching the map's unexported code":  2,
		"MS-7 C3, moves with the code":                   1,
		"MS-7 C4, cannot be served":                      1,
		"app unexported crossings under the planned map": 1,
		"map reach (symbols of the model's files)":       3, // Dashboard, .width and frame, from map_draw.go
	} {
		if got[name] != want {
			t.Errorf("%s = %d; want %d", name, got[name], want)
		}
	}
	for _, row := range m.tests { // bounded by the tests (P10-02)
		if row.name == "TestTheMapReadsTheFrame" && (row.class != "C4" || strings.Join(row.rootOut, " ") != "Dashboard.width") {
			t.Errorf("the test reading a root field outside the allowlist is %s, root out %v", row.class, row.rootOut)
		}
	}
	if len(crossed) != 1 || !strings.Contains(crossed[0], "helper") {
		t.Errorf("crossings %v; want app.go's use of helper, planned for another package", crossed)
	}
}

// TestCheckFailsARise records the planted tree, lowers one held figure and
// one reported figure in the baseline, and holds check to failing on the
// first alone.
func TestCheckFailsARise(t *testing.T) {
	dir := plantedTree(t)
	base := filepath.Join(t.TempDir(), "figures.tsv")
	var o, e bytes.Buffer
	if code := run([]string{"record", "-dir", dir, "-out", base}, &out{w: &o}, &out{w: &e}); code != 0 {
		t.Fatalf("record exited %d: %s", code, e.String())
	}
	raw, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	lowered := strings.Replace(string(raw), "Dashboard methods\t2", "Dashboard methods\t1", 1)
	lowered = strings.Replace(lowered, "MS-7 C4, cannot be served\t1", "MS-7 C4, cannot be served\t0", 1)
	if lowered == string(raw) {
		t.Fatal("COULD NOT RUN: the planted baseline did not hold the figures this test lowers")
	}
	if err := os.WriteFile(base, []byte(lowered), 0o644); err != nil {
		t.Fatal(err)
	}
	o.Reset()
	code := run([]string{"check", "-dir", dir, "-baseline", base}, &out{w: &o}, &out{w: &e})
	if code == 0 || !strings.Contains(o.String(), "FAIL Dashboard methods rose: 2, against 1") || strings.Contains(o.String(), "FAIL MS-7") {
		t.Errorf("check exited %d:\n%s", code, o.String())
	}
}

// TestTheRepositoryKeepsItsFigures holds the repository to the committed
// baseline recorded on e57afeaf (0.19.6 B0.12): no held figure above it.
func TestTheRepositoryKeepsItsFigures(t *testing.T) {
	if testing.Short() {
		t.Skip("type-checks both trees")
	}
	var o, e bytes.Buffer
	code := run([]string{"check", "-dir", "../..", "-baseline", "../../06_docs/graph/figures-e57afeaf.tsv"}, &out{w: &o}, &out{w: &e})
	t.Log(strings.TrimSpace(o.String()))
	if code != 0 {
		t.Errorf("the tree left its figures (exit %d):\n%s%s", code, o.String(), e.String())
	}
}

// TestADeckOnlyTestReachesTheDeckAlone is 0.19.6 A-45 on the planted module: a
// deck-only test file that reaches the deck's code passes, and one that
// reaches the root through a helper in another test file is reported with
// what it reaches, and check fails on it.
func TestADeckOnlyTestReachesTheDeckAlone(t *testing.T) {
	dir := plantedTree(t)
	ix, err := load(dir)
	if err != nil {
		t.Fatal(err)
	}
	pl, err := readPlanned(filepath.Join(dir, plannedApp))
	if err != nil {
		t.Fatal(err)
	}
	if got := ix.outsideTheDeck(pl, "app/cutback_hazard_test.go"); len(got) != 0 {
		t.Errorf("the planted deck-only file reaches %v; want nothing outside the deck", got)
	}
	if got := ix.outsideTheDeck(pl, "app/deck_trace_test.go"); len(got) != 1 || !strings.HasPrefix(got[0], "Run (app/app.go, planned app)") {
		t.Errorf("the planted file reaching the root through a helper reports %v; want Run alone", got)
	}
	if got := ix.outsideTheDeck(pl, "app/no_such_test.go"); len(got) != 1 || !strings.Contains(got[0], "declares nothing") {
		t.Errorf("a missing file reports %v; want it reported as itself", got)
	}
	held := deckOnlyTests
	deckOnlyTests = []string{"app/cutback_hazard_test.go", "app/deck_trace_test.go"}
	t.Cleanup(func() { deckOnlyTests = held })
	base := filepath.Join(t.TempDir(), "figures.tsv")
	var o, e bytes.Buffer
	if code := run([]string{"record", "-dir", dir, "-out", base}, &out{w: &o}, &out{w: &e}); code != 0 {
		t.Fatalf("record exited %d: %s", code, e.String())
	}
	o.Reset()
	code := run([]string{"check", "-dir", dir, "-baseline", base}, &out{w: &o}, &out{w: &e})
	if code == 0 || !strings.Contains(o.String(), "FAIL app/deck_trace_test.go reaches Run") || strings.Contains(o.String(), "FAIL app/cutback_hazard_test.go") {
		t.Errorf("check exited %d:\n%s", code, o.String())
	}
}
