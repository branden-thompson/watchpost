package main

// 0.19.0 W0.4 (A-29, FR-10.4): checks that read the release's record. The
// trace the PLAN red team ran by hand - every test a requirement names is in
// a plan task - and every ruling row that names a Go file cites its line.

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// releaseRecord is the release in flight's feature folder.
var releaseRecord = filepath.Join("..", "..", "06_docs", "02_features", "propagation-overlays")

// requirementRow is a requirements table row: its ID first ("FR-4.10",
// "NFR-2", "IS-1").
var requirementRow = regexp.MustCompile(`^\| \*{0,2}[A-Z]+-[\d.]+`)

// testName is a test or fuzz target named in backticks, the library it is
// in before it when it is one of the paired libraries': "go-ionomaps
// `TestX`".
var testName = regexp.MustCompile("(go-ionomaps |go-tuiMaps )?`((?:Test|Fuzz)[A-Za-z0-9_]+)`")

// testFunc is a test or fuzz target's declaration.
var testFunc = regexp.MustCompile(`(?m)^func ((?:Test|Fuzz)[A-Za-z0-9_]+)\(`)

// goFileRef is a Go file named in backticks, its line or range after it if
// one is given.
var goFileRef = regexp.MustCompile("`([A-Za-z0-9_./-]+\\.go)(:\\d+(?:-\\d+)?)?`")

// rulingRow is a rulings table row: a D- or A- row.
var rulingRow = regexp.MustCompile(`^\| ([AD]-\d+) `)

// testsNamed are the tests the requirement rows name, sorted, once each.
func testsNamed(requirements string) []string {
	var out []string
	for _, line := range strings.Split(requirements, "\n") {
		if !requirementRow.MatchString(line) {
			continue
		}
		for _, m := range testName.FindAllStringSubmatch(line, -1) {
			out = append(out, m[1]+m[2])
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// notInPlan are the names the plan does not carry, a library's test with
// its library before it.
func notInPlan(names []string, plan string) []string {
	var out []string
	for _, n := range names {
		lib, name, found := strings.Cut(n, " ")
		want := "`" + n + "`"
		if found {
			want = lib + " `" + name + "`"
		}
		if !strings.Contains(plan, want) {
			out = append(out, n)
		}
	}
	return out
}

// notInCode are the names no test in the tree declares; a paired library's
// test is checked in its own repository.
func notInCode(names []string, declared map[string]bool) []string {
	var out []string
	for _, n := range names {
		if !strings.Contains(n, " ") && !declared[n] {
			out = append(out, n)
		}
	}
	return out
}

// declaredTests are every test and fuzz target the tree declares.
func declaredTests(t *testing.T, root string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".claude") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range testFunc.FindAllStringSubmatch(string(b), -1) {
			out[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// unlinedRefs are the ruling rows that name a Go file without its line,
// each with the first such file.
func unlinedRefs(rulings string) []string {
	var out []string
	for _, line := range strings.Split(rulings, "\n") {
		row := rulingRow.FindStringSubmatch(line)
		if row == nil {
			continue
		}
		for _, m := range goFileRef.FindAllStringSubmatch(line, -1) {
			if m[2] == "" {
				out = append(out, row[1]+": "+m[1])
				break
			}
		}
	}
	return out
}

// readRecord is one of the release's documents.
func readRecord(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(releaseRecord, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestEveryRequirementsTestIsInAPlanTask is W0.4 (A-29): every test the
// requirements name is carried by a plan task, so none is left unbuilt
// without the plan saying so.
func TestEveryRequirementsTestIsInAPlanTask(t *testing.T) {
	names := testsNamed(readRecord(t, "01-objectives/requirements.md"))
	if len(names) < 100 {
		t.Fatalf("the requirements name %d tests; the reading is broken, or the rows are", len(names))
	}
	if missing := notInPlan(names, readRecord(t, "04-development/implementation-plan.md")); len(missing) > 0 {
		t.Errorf("the requirements name %d tests no plan task carries: %v", len(missing), missing)
	}
}

// TestEveryRequirementsTestExists is the BUILD red team's B-F4 (D-F2): every
// test the requirements name is declared in the tree - a paired library's,
// written "go-ionomaps `TestX`", in its own - so no requirement leans on a
// test that was never written or has since been renamed.
func TestEveryRequirementsTestExists(t *testing.T) {
	names := testsNamed(readRecord(t, "01-objectives/requirements.md"))
	declared := declaredTests(t, filepath.Join("..", ".."))
	if len(declared) < 1000 {
		t.Fatalf("the tree declares %d tests; the walk is broken", len(declared))
	}
	if missing := notInCode(names, declared); len(missing) > 0 {
		t.Errorf("the requirements name %d tests the tree does not declare: %v", len(missing), missing)
	}
}

// TestEveryRulingNamingCodeCitesItsLine is FR-10.4 (REFLECT L3): a ruling
// row that names a Go file cites the line its claim is about.
func TestEveryRulingNamingCodeCitesItsLine(t *testing.T) {
	rulings := readRecord(t, "02-analysis/rulings.md")
	if n := len(goFileRef.FindAllString(rulings, -1)); n < 50 {
		t.Fatalf("the rulings name %d Go files; the reading is broken", n)
	}
	if bad := unlinedRefs(rulings); len(bad) > 0 {
		t.Errorf("ruling rows name a Go file without its line: %v", bad)
	}
}

// citedLine is a Go file's line or range cited in a normative document, the
// paired library before it when the file is one of theirs.
var citedLine = regexp.MustCompile("(go-ionomaps |go-tuiMaps )?`([A-Za-z0-9_./-]+\\.go):(\\d+)(?:-(\\d+))?`")

// unresolvedCitations are the citations in a document that name no file in
// the tree - a bare file name included - or a line past its end; a paired
// library's is checked in its own repository.
func unresolvedCitations(doc, root string) []string {
	var out []string
	for _, m := range citedLine.FindAllStringSubmatch(doc, -1) {
		if m[1] != "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, m[2]))
		last := m[3]
		if m[4] != "" {
			last = m[4]
		}
		n, _ := strconv.Atoi(last)
		if err != nil || n > strings.Count(string(b), "\n")+1 {
			out = append(out, m[2]+":"+last)
		}
	}
	return out
}

// TestEveryCodeCitationResolves is the BUILD red team's C-10 and D-F5: in
// the documents that state the release as it is, every Go file cited by
// line is a path in the tree and the line is in the file. A ruling's
// citation is its moment's, the tree it cited in git, so the rulings are
// held to citing a line at all (TestEveryRulingNamingCodeCitesItsLine).
func TestEveryCodeCitationResolves(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, doc := range []string{"00-REQUIRED-READING.md", "01-objectives/requirements.md", "03-architecture-design/architecture.md", "04-development/implementation-plan.md"} {
		if bad := unresolvedCitations(readRecord(t, doc), root); len(bad) > 0 {
			t.Errorf("%s cites code that is not there: %v", doc, bad)
		}
	}
	planted := "`cmd/watchpost/record_test.go:1` `cmd/watchpost/record_test.go:99999` `record_test.go:1` go-ionomaps `reach.go:9` `no/such.go:1`"
	if got := unresolvedCitations(planted, root); !slices.Equal(got, []string{"cmd/watchpost/record_test.go:99999", "record_test.go:1", "no/such.go:1"}) {
		t.Errorf("the planted citations gave %v", got)
	}
}

// TestTheRecordChecksFailOnAPlantedGap is W0.4's acceptance: each check
// reports a planted gap - a test the plan dropped, a ruling without its
// line - and passes the same record without it.
func TestTheRecordChecksFailOnAPlantedGap(t *testing.T) {
	req := "| FR-1.1 *(O1)* | words | D-1 | `TestA`, `TestB` |\n| **NFR-2** | more | D-2 | `FuzzC` |\nprose `TestNotARow`\n"
	if got := testsNamed(req); !slices.Equal(got, []string{"FuzzC", "TestA", "TestB"}) {
		t.Fatalf("the requirement rows name %v", got)
	}
	lib := testsNamed("| FR-2 | x | D-3 | go-ionomaps `TestL` |\n")
	if got := notInCode(append(testsNamed(req), lib...), map[string]bool{"TestA": true, "FuzzC": true}); !slices.Equal(got, []string{"TestB"}) {
		t.Errorf("a test never written: %v (a library's test is its own repository's)", got)
	}
	if got := notInPlan(lib, "| W1 | `TestL` |"); len(got) != 1 {
		t.Errorf("a library's test the plan names as the tree's: %v", got)
	}
	plan := "| W1 | x | `TestA`, `FuzzC` |"
	if got := notInPlan(testsNamed(req), plan); !slices.Equal(got, []string{"TestB"}) {
		t.Errorf("a test the plan dropped: %v", got)
	}
	if got := notInPlan(testsNamed(req), plan+" `TestB`"); len(got) != 0 {
		t.Errorf("the plan carrying every test: %v", got)
	}
	rulings := "| D-1 | q `app/app.go:190` | a |\n| A-2 | (`modes/tty/x.go`) |\n| A-3 | `a/b.go:3-9` |\nprose `c.go`\n"
	if got := unlinedRefs(rulings); !slices.Equal(got, []string{"A-2: modes/tty/x.go"}) {
		t.Errorf("a ruling without its line: %v", got)
	}
}

// hookRuns are the pre-push hook's runs: each package and its tests.
var hookRuns = regexp.MustCompile(`\{"(\./[a-z/]+)", \[\]string\{([^}]*)\}\}`)

// TestThePrePushHookRunsTheRecordsTests is W0.1 (FR-10.2, REFLECT L5): the
// hook is Go (tools/prepush), `make hooks` builds and installs it, and every
// test it runs is declared in the package it names - a renamed test would
// otherwise leave the hook running nothing and passing.
func TestThePrePushHookRunsTheRecordsTests(t *testing.T) {
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "tools", "prepush", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	runs := hookRuns.FindAllStringSubmatch(strings.Join(strings.Fields(string(raw)), " "), -1)
	if len(runs) < 3 {
		t.Fatalf("the hook runs %d packages; the record's, the as-built map's and the atlas's", len(runs))
	}
	for _, r := range runs {
		src := packageSource(t, filepath.Join(root, r[1]))
		for _, name := range regexp.MustCompile(`"(Test[A-Za-z0-9_]+)"`).FindAllStringSubmatch(r[2], -1) {
			if !strings.Contains(src, "func "+name[1]+"(t *testing.T)") {
				t.Errorf("the hook runs %s in %s, which declares no such test", name[1], r[1])
			}
		}
	}
	for _, must := range []string{"TestEveryRequirementsTestIsInAPlanTask", "TestEveryRulingNamingCodeCitesItsLine", "TestTheAsBuiltMapIsKeptInStep", "TestTheAtlasMatchesAFreshRender"} {
		if !strings.Contains(string(raw), must) {
			t.Errorf("the hook does not run %s", must)
		}
	}
	mk, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mk), "hooks:\n\t@go run ./tools/prepush -install") {
		t.Error("make hooks does not install through tools/prepush")
	}
	inst, err := os.ReadFile(filepath.Join(root, "tools", "prepush", "install.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, must := range []string{`"./tools/prehook"`, `"config", "core.hooksPath", dir`, "--path-format=absolute"} {
		if !strings.Contains(string(inst), must) {
			t.Errorf("make hooks's installer does not %q: the hook would not run tools/prepush from the tree, or not be found (H-9)", must)
		}
	}
	hook, err := os.ReadFile(filepath.Join(root, "tools", "prehook", "main.go"))
	if err != nil || !strings.Contains(string(hook), `exec.Command("go", "run", "./tools/prepush")`) {
		t.Errorf("the hook does not run tools/prepush from the tree (%v): a built copy of it goes stale (H-9)", err)
	}
}

// packageSource is every test file of a package, joined.
func packageSource(t *testing.T, dir string) string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no test files in %s: %v", dir, err)
	}
	var b strings.Builder
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(raw)
	}
	return b.String()
}

// TestTheSayLegIsBoundedAndDiagnosed is 0.19.0 FR-9.3 (F-188): make
// test-say has a bound of its own, and its test samples a say still running
// near that bound, so a hang is read, not retried blind.
func TestTheSayLegIsBoundedAndDiagnosed(t *testing.T) {
	root := filepath.Join("..", "..")
	mk, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	leg := regexp.MustCompile(`(?m)^test-say:\n\t(.+)$`).FindStringSubmatch(string(mk))
	if leg == nil || !regexp.MustCompile(`-timeout \d+m`).MatchString(leg[1]) {
		t.Errorf("make test-say has no bound of its own: %v", leg)
	}
	src, err := os.ReadFile(filepath.Join(root, "domains", "radio", "synth", "synth_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "defer diagnoseHang(t, ") {
		t.Error("the say test does not sample a hang near its bound")
	}
}
