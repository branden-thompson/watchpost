package main

// 0.19.0 W0.4 (A-29, FR-10.4): checks that read the release's record. The
// trace the PLAN red team ran by hand - every test a requirement names is in
// a plan task - and every ruling row that names a Go file cites its line.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// releaseRecord is the release in flight's feature folder (0.19.6 B0.13).
var releaseRecord = filepath.Join("..", "..", "06_docs", "02_features", "0.19.6-re-architecting")

// record0190 is 0.19.0's feature folder. Its requirements stay held here
// (0.19.6 A-37): 0.19.6 has no requirements document and its brief's rows
// name instruments, not tests, while 0.19.0's requirements name over a
// hundred tests that every move of 0.19.6 must keep.
var record0190 = filepath.Join("..", "..", "06_docs", "02_features", "propagation-overlays")

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
// test is passed over: nothing here reads the library's tree, and neither
// library checks the names watchpost's documents give it (RC-8).
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

// wholeFileClaim is a ruling row's claim about a Go file as a whole: a count
// of what it holds, or the file as the unit a move carries or keeps, which no
// line could stand for (0.19.6 A-37).
type wholeFileClaim struct{ row, file string }

// wholeFileClaims are 0.19.6's, fixed at B0: a row added later that names a
// file without its line is held to the rule unless it is listed here.
var wholeFileClaims = []wholeFileClaim{
	{"D-16", "app/bedrelay.go"}, {"D-16", "app/mapwaves.go"}, {"D-16", "inject.go"},
	{"D-24", "app/cast.go"}, {"D-24", "app/voices.go"}, {"D-24", "app/hostfacts.go"}, {"D-24", "app/cast_test.go"},
	{"A-19", "app/bedrelay.go"}, {"A-19", "app/pool.go"}, {"A-19", "app/dashboard.go"},
	{"A-19", "app/history.go"}, {"A-19", "app/inject.go"}, {"A-19", "app/mapwaves.go"},
	{"A-23", "saychildren.go"}, {"A-23", "hangup.go"}, {"A-23", "lazymemo.go"}, {"A-23", "timing.go"},
	// 0.19.6 D-47 is about the files' names, the basenames app and modes/tty share
	{"D-47", "dashboard.go"}, {"D-47", "debug.go"}, {"D-47", "doc.go"}, {"D-47", "setup.go"},
	{"D-47", "severe.go"}, {"D-47", "ticker.go"}, {"D-47", "timing.go"},
	{"D-47", "app/dashboard.go"}, {"D-47", "modes/tty/dashboard.go"},
	// 0.19.6 D-61 holds one file to a density floor of its own
	{"D-61", "app/mastercontrol.go"},
	// 0.19.6 A-50 is about the files B2's prep cuts into, by name, and the files mutants left
	{"A-50", "modes/tty/kit_row.go"}, {"A-50", "kit_cursor.go"}, {"A-50", "kit_tick.go"}, {"A-50", "kit_memo.go"},
	{"A-50", "kit_text.go"}, {"A-50", "masthead.go"}, {"A-50", "frame_height_test.go"},
	{"A-50", "modes/tty/ttytest_perturb_test.go"}, {"A-50", "ttytest_helpers_test.go"}, {"A-50", "cursor.go"}, {"A-50", "body.go"},
	// 0.19.6 D-66 names the files whose P10-09 rows are package-wide
	{"D-66", "pkg/schema/schema.go"}, {"D-66", "platform/render/units.go"},
	// 0.19.6 A-54 is about the packages' doc.go files, each whole
	{"A-54", "doc.go"},
	// 0.19.6 A-21 names the root file B3's prep cuts the card window into, which exists from that commit
	{"A-21", "card_window.go"},
	// 0.19.6 A-20 names 2a's new files, which exist from B4.2's commit (D-78) and B4.3's
	{"A-20", "map_model.go"}, {"A-20", "map_timing.go"}, {"A-20", "observer_map.go"},
	// 0.19.6 A-66 records B4.3 as built: the files it created, split or gave a row, each whole
	{"A-66", "observer_map.go"}, {"A-66", "modes/tty/observer_map.go"}, {"A-66", "map_timing.go"}, {"A-66", "timing.go"}, {"A-66", "doc.go"}, {"A-66", "window_keys.go"},
}

// unlinedRefs are the ruling rows that name a Go file without its line or a
// symbol, each with the file. A bare suffix (`_test.go`) names no file, a
// file the tree does not hold is a planned name no line can be cited in, and
// a listed whole-file claim stands as it is.
func unlinedRefs(rulings string, exists func(string) bool) []string {
	var out []string
	for _, line := range strings.Split(rulings, "\n") {
		row := rulingRow.FindStringSubmatch(line)
		if row == nil {
			continue
		}
		for _, m := range goFileRef.FindAllStringSubmatch(line, -1) {
			if m[2] != "" || strings.HasPrefix(m[1], "_") || !exists(m[1]) || slices.Contains(wholeFileClaims, wholeFileClaim{row[1], m[1]}) {
				continue
			}
			out = append(out, row[1]+": "+m[1])
		}
	}
	return out
}

// rowNames reports whether a ruling row names a Go file in backticks.
func rowNames(rulings, row, file string) bool {
	for _, line := range strings.Split(rulings, "\n") { // bounded by the rulings (P10-02)
		if strings.HasPrefix(line, "| "+row+" ") {
			return strings.Contains(line, "`"+file+"`")
		}
	}
	return false
}

// inTree reports whether the tree holds a Go file, named by its path or, for
// a bare name, anywhere under app and modes/tty.
func inTree(root string) func(string) bool {
	return func(name string) bool {
		if strings.Contains(name, "/") {
			_, err := os.Stat(filepath.Join(root, name))
			return err == nil
		}
		for _, dir := range []string{"app", "modes/tty"} { // the two trees (P10-02)
			if matches, _ := filepath.Glob(filepath.Join(root, dir, name)); len(matches) > 0 {
				return true
			}
		}
		return false
	}
}

// readRecord is one of the release's documents.
func readRecord(t *testing.T, rel string) string {
	t.Helper()
	return readRecordIn(t, releaseRecord, rel)
}

// readRecordIn is one of a feature folder's documents.
func readRecordIn(t *testing.T, folder, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(folder, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestEveryRequirementsTestIsInAPlanTask is 0.19.0 W0.4 (A-29): every test
// the requirements name is carried by a plan task, so none is left unbuilt
// without the plan saying so. It reads 0.19.0's record (0.19.6 A-37).
func TestEveryRequirementsTestIsInAPlanTask(t *testing.T) {
	names := testsNamed(readRecordIn(t, record0190, "01-objectives/requirements.md"))
	if len(names) < 100 {
		t.Fatalf("the requirements name %d tests; the reading is broken, or the rows are", len(names))
	}
	if missing := notInPlan(names, readRecordIn(t, record0190, "04-development/implementation-plan.md")); len(missing) > 0 {
		t.Errorf("the requirements name %d tests no plan task carries: %v", len(missing), missing)
	}
}

// TestEveryRequirementsTestExists is the BUILD red team's B-F4 (D-F2): every
// test the requirements name is declared in the tree - a paired library's,
// written "go-ionomaps `TestX`", passed over, unchecked (RC-8) - so no
// requirement leans on a test of watchpost's that was never written or has
// since been renamed. It reads 0.19.0's record (0.19.6 A-37), so a test a
// 0.19.6 move renames or loses is caught here too.
func TestEveryRequirementsTestExists(t *testing.T) {
	names := testsNamed(readRecordIn(t, record0190, "01-objectives/requirements.md"))
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
	if bad := unlinedRefs(rulings, inTree(filepath.Join("..", ".."))); len(bad) > 0 {
		t.Errorf("ruling rows name a Go file without its line: %v", bad)
	}
	for _, c := range wholeFileClaims { // bounded by the list (P10-02)
		if !strings.Contains(rulings, "\n| "+c.row+" ") || !rowNames(rulings, c.row, c.file) {
			t.Errorf("wholeFileClaims lists %s naming %s, which the row does not: drop it", c.row, c.file)
		}
	}
}

// citedLine is a Go file's line or range cited in a normative document, the
// paired library before it when the file is one of theirs.
var citedLine = regexp.MustCompile("(go-ionomaps |go-tuiMaps )?`([A-Za-z0-9_./-]+\\.go):(\\d+)(?:-(\\d+))?`")

// citedSymbol is a Go file cited with the symbol its claim is about -
// "`path.go:Symbol`", the form the flow map uses (docs_test.go) - the
// paired library before it when the file is one of theirs.
var citedSymbol = regexp.MustCompile("(go-ionomaps |go-tuiMaps )?`([A-Za-z0-9_./-]+\\.go):([A-Za-z_][A-Za-z0-9_]*)`")

// declaresName reports whether a Go file declares a name at its top level -
// a function, a method, a type, a variable or a constant - or as a field of
// a struct type it declares.
func declaresName(path, name string) bool {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == name {
				return true
			}
		case *ast.GenDecl:
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == name || fieldNamed(s, name) {
						return true
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if n.Name == name {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// fieldNamed reports whether a type spec is a struct with a field of a name.
func fieldNamed(s *ast.TypeSpec, name string) bool {
	st, ok := s.Type.(*ast.StructType)
	if !ok {
		return false
	}
	for _, f := range st.Fields.List {
		for _, n := range f.Names {
			if n.Name == name {
				return true
			}
		}
	}
	return false
}

// unresolvedCitations are the citations in a living document that do not
// hold: a Go file cited by line - lines drift as the code moves, so a
// living document cites the symbol - or by a symbol the file does not
// declare, or a file not in the tree; a paired library's is passed over,
// unchecked here or there (RC-8).
func unresolvedCitations(doc, root string) []string {
	var out []string
	for _, m := range citedLine.FindAllStringSubmatch(doc, -1) {
		if m[1] == "" {
			out = append(out, m[2]+":"+m[3]+" (cite its symbol)")
		}
	}
	for _, m := range citedSymbol.FindAllStringSubmatch(doc, -1) {
		if m[1] == "" && !declaresName(filepath.Join(root, m[2]), m[3]) {
			out = append(out, m[2]+":"+m[3])
		}
	}
	return out
}

// TestEveryCodeCitationResolves is 0.19.0's C-10, D-F5, CN2-7 and DN2-8: the
// documents that state the release as it is
// cite code by the symbol their claim is about, and each file declares it -
// a line number moves with the code and only proved the line was there. A
// ruling's citation is its moment's, the tree it cited in git, so the
// rulings are held to citing a line at all
// (TestEveryRulingNamingCodeCitesItsLine).
func TestEveryCodeCitationResolves(t *testing.T) {
	root := filepath.Join("..", "..")
	cited := 0
	for _, doc := range []string{"00-REQUIRED-READING.md", "01-objectives/project-brief.md", "03-architecture-design/architecture.md", "04-development/implementation-plan.md"} {
		text := readRecord(t, doc)
		cited += len(citedSymbol.FindAllString(text, -1))
		if bad := unresolvedCitations(text, root); len(bad) > 0 {
			t.Errorf("%s cites code that does not hold: %v", doc, bad)
		}
	}
	if cited < 10 {
		t.Errorf("the living documents cite %d symbols; the reading is broken", cited)
	}
	planted := "`cmd/watchpost/record_test.go:1` `cmd/watchpost/record_test.go:unresolvedCitations` `cmd/watchpost/record_test.go:citedSymbol` `cmd/watchpost/record_test.go:noSuchName` go-ionomaps `reach.go:Reach` `no/such.go:X`"
	want := []string{"cmd/watchpost/record_test.go:1 (cite its symbol)", "cmd/watchpost/record_test.go:noSuchName", "no/such.go:X"}
	if got := unresolvedCitations(planted, root); !slices.Equal(got, want) {
		t.Errorf("the planted citations gave %v; want %v", got, want)
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
	rulings := "| D-1 | q `app/app.go:190` | a |\n| A-2 | (`modes/tty/x.go`) |\n| A-3 | `a/b.go:3-9` |\nprose `c.go`\n" +
		"| D-16 | `app/bedrelay.go` 9 |\n| D-17 | `app/bedrelay.go` 9, `_test.go` files, `planned.go` |\n"
	held := func(name string) bool { return name != "planned.go" }
	if got := unlinedRefs(rulings, held); !slices.Equal(got, []string{"A-2: modes/tty/x.go", "D-17: app/bedrelay.go"}) {
		t.Errorf("a ruling without its line: %v (a listed whole-file claim, a suffix and a planned name pass)", got)
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
	for _, must := range []string{"TestEveryRequirementsTestIsInAPlanTask", "TestEveryRulingNamingCodeCitesItsLine", "TestEveryRequirementsTestExists",
		"TestEveryCodeCitationResolves", "TestTheAsBuiltMapIsKeptInStep", "TestTheAtlasMatchesAFreshRender"} {
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
	if !strings.Contains(string(src), "stop := diagnoseHang(t, ") {
		t.Error("the say test does not sample a hang near its bound")
	}
	if !strings.Contains(string(src), `"-file", report`) {
		t.Error("the hang's sample report is not written where the test reads it into the log (F-219): sample writes its report to a file, never to its output")
	}
}

// fr7Row is a requirement row of FR-7's, by its number.
var fr7Row = regexp.MustCompile(`^\| FR-7\.(\d) `)

// fr7PlanRow is one of the plan's W8.1 rows naming the FR-7 row it builds.
var fr7PlanRow = regexp.MustCompile(`^\| W8\.1[a-z] \|.*\(FR-7\.(\d)`)

// fr7Gaps are, for each FR-7 row the plan builds, the tests its W8.1 rows
// name that the requirement row does not - and an FR-7 row of 1 to 4 that
// names no test at all.
func fr7Gaps(requirements, plan string) []string {
	named := map[string][]string{}
	for _, line := range strings.Split(requirements, "\n") {
		if m := fr7Row.FindStringSubmatch(line); m != nil {
			named[m[1]] = testsNamed(line)
		}
	}
	var out []string
	for _, n := range []string{"1", "2", "3", "4"} {
		if len(named[n]) == 0 {
			out = append(out, "FR-7."+n+" names no test")
		}
	}
	for _, line := range strings.Split(plan, "\n") {
		m := fr7PlanRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		for _, t := range testName.FindAllStringSubmatch(line, -1) {
			if !slices.Contains(named[m[1]], t[2]) {
				out = append(out, "FR-7."+m[1]+" lacks "+t[2])
			}
		}
	}
	return out
}

// TestEveryFR7RowNamesItsTest is W8.2, D-63's SHIP gate: each of FR-7.1 to
// FR-7.4 names every test its plan rows (W8.1a to W8.1m) name, so
// TestEveryRequirementsTestExists holds each to the tree and make verify runs
// it; a planted gap is reported.
func TestEveryFR7RowNamesItsTest(t *testing.T) {
	if gaps := fr7Gaps(readRecordIn(t, record0190, "01-objectives/requirements.md"), readRecordIn(t, record0190, "04-development/implementation-plan.md")); len(gaps) > 0 {
		t.Errorf("FR-7's rows do not name their tests: %v", gaps)
	}
	req := "| FR-7.1 *(0.19.3)* | a | D-1 | `TestA` |\n| FR-7.2 *(0.19.3)* | b | D-2 | `TestB` |\n| FR-7.3 *(0.19.3)* | c | D-3 | `TestC` |\n| FR-7.4 *(0.19.3)* | d | D-4 | as before |\n"
	plan := "| W8.1a | x (FR-7.1) | `TestA`, `TestZ` |\n| W8.1b | y (FR-7.2; D-9) | `TestB` |\n"
	want := []string{"FR-7.4 names no test", "FR-7.1 lacks TestZ"}
	if got := fr7Gaps(req, plan); !slices.Equal(got, want) {
		t.Errorf("the planted gaps gave %v; want %v", got, want)
	}
}
