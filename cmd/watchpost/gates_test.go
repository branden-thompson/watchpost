package main

// gates_test.go — FR-8: `make verify` and CI run the same gates, every required
// gate runs somewhere, and no gate can be silenced without a written reason.
//
// EVERY GATE HERE IS AN ASSERTION OVER ONE MODEL. The parsing — what counts as a
// live command, a gate-shaped target, a silenced step, a build line — lives in
// gatemodel_test.go and nowhere else; the attacks it is held to are in
// 06_docs/gate-attack-list.md and run as specimens in gateattacks_test.go.
// Every exemption table registers with exemptions_test.go, which checks every
// row of every table in one place.
//
// THE HAZARD IS A GATE THAT EXISTS AND DOES NOT RUN. required-gates.txt makes a
// gate need three edits to LEAVE; TestEveryGateShapedTargetIsListedOrExempt
// catches one that never ARRIVES; the CI checks catch one that is present on
// every list and silenced in the workflow.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/tools/gateoracle"
)

// ---- the exemption tables --------------------------------------------------

// ciOnly are gates CI runs that a local `make verify` deliberately does not.
var ciOnly = exempt(&exemptionTable{
	name: "ciOnly", satisfied: "race",
	rows: map[string]string{
		"release-matrix": "it builds five platforms; a local verify would spend minutes producing artifacts nobody is about to publish",
		"install-test":   "it installs the artifacts release-matrix built, so it cannot run without them",
	},
	exists:      func(t *testing.T, g string) bool { _, ok := loadBuildModel(t).ciGates()[g]; return ok },
	stillNeeded: func(t *testing.T, g string) bool { return !contains(loadBuildModel(t).verifyGates(), g) },
})

// verifyOnly are gates a local `make verify` runs that CI does not.
// phaseExit are required gates that run at BUILD and REVIEW exit under `make
// quality`, by the HUM LEAD, and are recorded in the roster — not on every
// `make verify` and not in CI. The release workflow runs `make verify` on a
// clean clone, so a gate that needs something outside the public tree cannot
// live in verify without breaking the tag by construction (REVIEW red team,
// 2026-09-17, HUM LEAD ruling 1).
var phaseExit = exempt(&exemptionTable{
	name: "phaseExit", satisfied: "race",
	rows: map[string]string{
		"p10": "the P10 harness CLI and its exemptions ledger live OUTSIDE the public tree, so CI and the release runner have no `a2dh`; it fails loud rather than skips, which is right for a gate and wrong for a release runner, so it runs under `make quality` at phase exit and its 0/0/0 is recorded in the roster",
	},
	exists: func(t *testing.T, g string) bool {
		return contains(loadBuildModel(t).required, g) && loadBuildModel(t).hasTarget(g)
	},
	stillNeeded: func(t *testing.T, g string) bool {
		m := loadBuildModel(t)
		_, ci := m.ciGates()[g]
		return !contains(m.verifyGates(), g) && !ci
	},
})

// unlisted are gate-shaped Makefile targets deliberately on NO gate list.
var unlisted = exempt(&exemptionTable{
	name: "unlisted", satisfied: "race",
	rows: map[string]string{
		"quality-bench":     "a measurement, not a gate — it reports numbers and has no pass condition (INST-5)",
		"pty-severe":        "it drives a real pty, which CI has no terminal for",
		"journey":           "the VALIDATE journey against LIVE feeds; its exit code is a FAIL count, run deliberately at release",
		"test":              "`go test ./...` without the race detector; `race` supersedes it on every gate path, and running both would double the suite for no extra property",
		"test-platforms":    "it re-runs the app suite under WATCHPOST_TEST_GOOS for two other platforms; release-matrix is the cross-platform gate and this is the manual probe behind it",
		"mutant-verdicts":   "the corpus SWEEP — hours, and its verdicts are promoted into the release record by hand; mutant-anchors and mutant-check are the per-push halves",
		"tree-free":         "it ASKS whether a gate run is in flight and reports; it asserts nothing about the code",
		"lint-update":       "it REWRITES the golangci baseline; it is the opposite of a gate, and running it on a gate path would erase the ratchet",
		"verify-docs-gates": "the docs lane (go-tuiMaps v0.2.0 D-15, watchpost 0.18.0 D-38): a local SUBSET of verify for a change that is Markdown alone, refused by tools/docslane for any other file; CI and every other change run verify, which runs every gate this does",
	},
	exists: func(t *testing.T, g string) bool { return loadBuildModel(t).targets[g] != nil },
	stillNeeded: func(t *testing.T, g string) bool {
		m := loadBuildModel(t)
		return contains(m.gateShaped(), g) && !contains(m.required, g)
	},
})

// cacheableGate are gate recipes whose `go test` deliberately omits -count=1.
var cacheableGate = exempt(&exemptionTable{
	name: "cacheableGate", satisfied: "race",
	rows: map[string]string{
		"test":          "not a gate — it is the plain suite, declared `unlisted`; `race` is what runs on every gate path and it carries the flag",
		"quality-bench": "a benchmark with `-count 10`, which is the sample size rather than a cache defence; benchmarks are not cached",
	},
	exists: func(t *testing.T, g string) bool { return loadBuildModel(t).targets[g] != nil },
	stillNeeded: func(t *testing.T, g string) bool {
		for _, c := range loadBuildModel(t).targets[g].cmds { // bounded by the recipe (P10-02)
			if c.isGoTest() && !c.countOne() {
				return true
			}
		}
		return false
	},
})

// conditionalStep are required gates whose CI step legitimately carries `if:`.
var conditionalStep = exempt(&exemptionTable{
	name: "conditionalStep", satisfied: "race",
	rows: map[string]string{
		"install-test": "the matrix runs three operating systems and this installs what release-matrix built; doing it once, on Linux, is the test",
		"test-say":     "it drives the real macOS `say`, so on Linux the test can only skip and the step would be green with nothing run; on macOS a missing `say` fails it",
		"mutant-check": "MUTANT_POLICY decides its schedule (push / nightly / label) and all three conditions are written out, so switching between them is a word in the Makefile rather than an edit here",
	},
	exists: func(t *testing.T, g string) bool {
		m := loadBuildModel(t)
		_, ok := m.ciGates()[g]
		return ok && contains(m.required, g)
	},
	stillNeeded: func(t *testing.T, g string) bool { return loadBuildModel(t).ciGates()[g].conditional },
})

// uncontrolled are checkers a required gate invokes that have no positive
// control. EVERY ROW IS A GATE TRUSTED WITHOUT EVIDENCE; the list is meant to
// shrink.
var uncontrolled = exempt(&exemptionTable{
	name: "uncontrolled", satisfied: "scripts/lint-imports.sh",
	rows: map[string]string{
		"scripts/lint.sh":                      "F-118 — it discards golangci-lint's exit code with `|| true` and treats non-empty JSON as liveness, so it does not fail closed. A control would pin the behaviour we intend to CHANGE; it is slated for conversion to Go",
		"scripts/install-test.sh":              "it installs and runs a built artifact, so a control would be a second installation on a machine that has just done one; it is CI-only and runs on a clean runner",
		"scripts/quality/p10-ledger-mirror.py": "a GENERATOR, and its control is the linter that runs immediately after it: `lint-ledger.sh` reads the file this writes, in the same recipe, and that linter has its own self-test",
	},
	exists: func(t *testing.T, k string) bool {
		m := loadBuildModel(t)
		for _, g := range m.required { // bounded by the gate list (P10-02)
			if contains(m.checkersOf(g), k) {
				return true
			}
		}
		return false
	},
	stillNeeded: func(t *testing.T, k string) bool { return !loadBuildModel(t).controlled()[k] },
})

// ---- the gates ----------------------------------------------------------------

// mutantModes are the schedules the mutant corpus can be put on. ALL THREE ARE
// WIRED AT ONCE, whichever is chosen (HUM LEAD, 2026-09-08): the workflow carries
// every mode's condition and the Makefile carries one word.
var mutantModes = []string{"push", "nightly", "label"}

func TestTheMutantPolicyIsASwitchAndNotARewrite(t *testing.T) {
	mk := read(t, "../../Makefile")
	ci := read(t, "../../.github/workflows/ci.yml")

	m := regexp.MustCompile(`(?m)^MUTANT_POLICY \?= (\w+)`).FindStringSubmatch(mk)
	if m == nil {
		t.Fatal("the Makefile declares no MUTANT_POLICY: the decision has gone back into the workflow")
	}
	if !contains(mutantModes, m[1]) {
		t.Errorf("MUTANT_POLICY is %q, which is not one of %v", m[1], mutantModes)
	}
	for _, mode := range mutantModes { // bounded by the mode list (P10-02)
		if !strings.Contains(ci, "'"+mode+"'") {
			t.Errorf("the workflow does not mention the %q mode: switching to it would be a "+
				"workflow change, which is the thing this arrangement exists to avoid", mode)
		}
	}
	for _, need := range []string{"schedule:", "cron:", "labeled"} { // bounded by the list (P10-02)
		if !strings.Contains(ci, need) {
			t.Errorf("the workflow has no %q, so at least one mode cannot fire", need)
		}
	}
	if !strings.Contains(ci, "make -s mutant-policy") {
		t.Error("the workflow does not read the policy from the Makefile: two copies of a decision " +
			"is one that can disagree with itself")
	}
}

// THE THREE LISTS AGREE. verify and CI run the same gates, and every required
// gate is on at least one of them — with every difference declared.
func TestTheThreeGateListsAgree(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	assertThreeListsAgree(t, m)
}

// assertThreeListsAgree is the property, separated so the specimen table can run
// it over a synthetic model.
func assertThreeListsAgree(t reporter, m *buildModel) {
	t.Helper()
	verify, ci := m.verifyGates(), m.ciGates()
	if len(verify) < 5 {
		t.Fatalf("COULD NOT RUN — verify resolves to %d gate(s); the delegation shape has changed and "+
			"this check has lost its subject", len(verify))
	}
	for _, g := range verify { // bounded by the gate list (P10-02)
		if _, ok := ci[g]; ok {
			continue
		}
		t.Errorf("`make verify` runs %s and CI does not, and nothing says why: a gate that only "+
			"runs on one machine is a gate that passes on the other by not being asked", g)
	}
	for g := range ci { // bounded by the workflow (P10-02)
		if contains(verify, g) {
			continue
		}
		if _, declared := ciOnly[g]; !declared {
			t.Errorf("CI runs %s and `make verify` does not, and nothing says why", g)
		}
	}
	// A REQUIRED GATE RUNS SOMEWHERE. CI-only is a legitimate somewhere, and so
	// is phase exit under `make quality` when the gate is a real target; running
	// NOWHERE is not, and neither is running only on one machine undeclared.
	for _, g := range m.required { // bounded by the gate list (P10-02)
		_, onCI := ci[g]
		onVerify := contains(verify, g)
		_, isCIOnly := ciOnly[g]
		_, isPhaseExit := phaseExit[g]
		switch {
		case isPhaseExit && (onVerify || onCI):
			t.Errorf("%s is declared a phase-exit gate and also runs in verify or CI; one of the two is stale", g)
		case isPhaseExit && !m.hasTarget(g):
			t.Errorf("%s is declared a phase-exit gate and is not a make target `make quality` can run", g)
		case isPhaseExit:
		case !onVerify && !onCI:
			t.Errorf("%s is required and runs NOWHERE — not in verify, not in CI, not declared phase-exit", g)
		case !onVerify && !isCIOnly:
			t.Errorf("%s is required and `make verify` does not run it, and nothing declares it CI-only", g)
		case !onCI:
			t.Errorf("%s is required and CI does not run it, and nothing declares it local-only", g)
		}
	}
}

// EVERY GATE-SHAPED TARGET IS ON A LIST, OR SAYS WHY NOT.
//
// required-gates.txt makes a gate need three edits to LEAVE; it has no answer to
// a gate that never ARRIVES. A target on zero lists is invisible to a
// comparison of two lists it is on neither of, however loudly it is declared
// to fail. This asks the Makefile.
func TestEveryGateShapedTargetIsListedOrExempt(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	assertGateShapedListed(t, m)
}

func assertGateShapedListed(t reporter, m *buildModel) {
	t.Helper()
	shaped := m.gateShaped()
	if len(shaped) < 5 {
		t.Fatalf("COULD NOT RUN — found %d gate-shaped target(s); the Makefile shape has changed and "+
			"this check has lost its subject", len(shaped))
	}
	for _, g := range shaped { // bounded by the Makefile (P10-02)
		if contains(m.required, g) {
			continue
		}
		if _, declared := unlisted[g]; declared {
			continue
		}
		t.Errorf("%s runs a gate and is on no list.\n"+
			"A gate nothing invokes is a gate that cannot fail — `make p10` sat that way, red, "+
			"while verify reported ALL GATES GREEN.\n"+
			"Add it to 06_docs/required-gates.txt and to the verify/CI paths, or add a row to "+
			"`unlisted` saying why it runs nowhere.", g)
	}
}

// NO REQUIRED GATE'S CI STEP CAN BE SILENCED WITHOUT A REASON.
//
// A condition on the step (`if:`), a condition on the JOB (which silences every
// step at once, one line higher and cheaper), `continue-on-error` in any spelling
// but the literal false, or `|| true` on the run line — each leaves all three
// lists agreeing while the gate never runs or never fails. Conditions are not
// banned; UNDECLARED ones are. A gate that cannot fail has no legitimate reason.
func TestNoRequiredGatesCIStepIsSilenced(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	assertNoSilencedCIStep(t, m)
}

func assertNoSilencedCIStep(t reporter, m *buildModel) {
	t.Helper()
	ci := m.ciGates()
	if len(ci) < 5 {
		t.Fatalf("COULD NOT RUN — the workflow resolves to %d gate step(s); its shape has changed and "+
			"this check has lost its subject", len(ci))
	}
	for _, g := range m.required { // bounded by the gate list (P10-02)
		st, ok := ci[g]
		if !ok {
			continue // absence is TestTheThreeGateListsAgree's question
		}
		_, declared := conditionalStep[g]
		switch {
		case st.jobSilenced:
			t.Errorf("%s is a REQUIRED gate inside a CI job that carries a job-level `if:` or "+
				"continue-on-error, which silences EVERY gate in it at once while all three lists still "+
				"agree.\nPut the condition on the individual steps that need it and declare them in "+
				"`conditionalStep`.", g)
		case st.tolerant:
			t.Errorf("%s is a REQUIRED gate and its CI step sets continue-on-error (in any form but the "+
				"literal false). A gate that cannot fail is not a gate.", g)
		case st.cannotFail:
			t.Errorf("%s is a REQUIRED gate and its run line discards the exit status (`|| true` or the "+
				"like). A gate that cannot fail is not a gate.", g)
		case st.conditional && !declared:
			t.Errorf("%s is a REQUIRED gate and its CI step carries an `if:` that nothing declares.\n"+
				"Add a row to `conditionalStep` with the reason, or remove the condition.", g)
		}
	}
}

// NO GATE IS ANSWERABLE FROM CACHE (AP-STALE-01).
//
// A cached PASS answers the question about a tree that may be several edits
// old, and looks exactly like a green run.
func TestNoGateIsAnswerableFromCache(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	assertNoCachedGate(t, m)
}

func assertNoCachedGate(t reporter, m *buildModel) {
	t.Helper()
	var checked int
	for name, tg := range m.targets { // bounded by the Makefile (P10-02)
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			if !c.isGoTest() {
				continue
			}
			checked++
			if c.countOne() {
				continue
			}
			if _, declared := cacheableGate[name]; declared {
				continue
			}
			t.Errorf("%s runs `go test` without -count=1.\nA cached PASS answers the question about "+
				"a tree that may be several edits old, and looks exactly like a green run.\n"+
				"Add -count=1, or add a row to `cacheableGate` saying why this one may be cached.", name)
		}
	}
	if checked < 5 {
		t.Fatalf("COULD NOT RUN — found %d `go test` recipes; the Makefile shape has changed and "+
			"this check has lost its subject", checked)
	}
}

// THE BUILD PATH IS NOT SHIPPED (FR-7.5, HUM LEAD 2026-09-08).
//
// Without -trimpath every binary embeds the directory it was compiled from,
// which names the person who built it. Every line that produces a binary —
// whatever names the package, and whether or not it names an output — must
// carry the flag. A bare `go build ./cmd/x` with no `-o` drops an untrimmed
// binary in the working directory, where `git add -A` sweeps it into a
// commit.
func TestEveryBuildTargetTrimsThePath(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	assertEveryBuildTrimmed(t, m)
}

func assertEveryBuildTrimmed(t reporter, m *buildModel) {
	t.Helper()
	builds := m.builds()
	if len(builds) == 0 {
		t.Fatalf("COULD NOT RUN — no build line found in the Makefile: this gate is looking at the wrong file")
	}
	for _, c := range builds { // bounded by the build lines (P10-02)
		if !c.trimsPath() {
			t.Errorf("a build target ships the path it was compiled from:\n  %s", c.text)
		}
	}
}

// THE ALLOC PINS ARE ALL REACHED. TestEveryRunSelectorSelectsATest fails a
// selector that reaches nothing; alloc-budget's pins are a set, so this also
// counts them, and a rename that drops some of the nine fails here.
func TestTheAllocBudgetSelectsItsPins(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	pattern := selectorOf(t, m, "alloc-budget")
	re := regexp.MustCompile(pattern)
	var pins int
	for _, f := range testFilesUnder(t, "../..") { // bounded by the tree (P10-02)
		for _, name := range regexp.MustCompile(`(?m)^func (Test\w+)\(`).FindAllStringSubmatch(f, -1) {
			if re.MatchString(name[1]) {
				pins++
			}
		}
	}
	if pins < 9 {
		t.Errorf("alloc-budget's pattern %q selects %d pin(s); there were nine. A selector that "+
			"reaches nothing exits 0 and reports a budget nobody measured.", pattern, pins)
	}
}

// testFilesUnder is the text of every _test.go under root, less third_party.
func testFilesUnder(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	var walk func(dir string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries { // bounded by the directory (P10-02)
			p := dir + "/" + e.Name()
			switch {
			case e.IsDir() && e.Name() != "third_party" && e.Name() != ".git":
				walk(p)
			case strings.HasSuffix(e.Name(), "_test.go"):
				b, err := os.ReadFile(p)
				if err == nil {
					out = append(out, string(b))
				}
			}
		}
	}
	walk(root)
	if len(out) < 50 {
		t.Fatalf("COULD NOT RUN — found %d test files under %s", len(out), root)
	}
	return out
}

// ---- helpers --------------------------------------------------------------------

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func sorted(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func contains(hay []string, want string) bool {
	for _, h := range hay { // bounded by the slice (P10-02)
		if h == want {
			return true
		}
	}
	return false
}

// ---- the executed half: the oracle over THIS tree ---------------------------------------

func realOracle(t *testing.T) (*gateoracle.Oracle, []string) {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	o := gateoracle.New(t, gateoracle.CloneForOracle(t, repo, t.TempDir()))
	return o, parseRequired(read(t, "../../06_docs/required-gates.txt"))
}

func TestNoFileSilencesARequiredGate(t *testing.T) {
	o, required := realOracle(t)
	gateoracle.AssertNoFileSilencesARequiredGate(t, o, required)
}

func TestEveryRequiredGateCanFail(t *testing.T) {
	o, required := realOracle(t)
	gateoracle.AssertEveryRequiredGateCanFail(t, o, required)
}

func TestVerifyCanFail(t *testing.T) {
	o, required := realOracle(t)
	gateoracle.AssertVerifyCanFail(t, o, required, notUnderVerify())
}

func TestEveryControlIsReached(t *testing.T) {
	o, required := realOracle(t)
	gateoracle.AssertEveryControlIsReached(t, o, required, uncontrolled)
}

// notUnderVerify is every required gate `make verify` is declared not to run —
// CI-only and phase-exit — so the oracle's coverage check asks verify only for
// the gates verify carries.
func notUnderVerify() map[string]string {
	out := map[string]string{}
	for _, tbl := range []map[string]string{ciOnly, phaseExit} { // bounded by the two tables (P10-02)
		for g, why := range tbl { // bounded by the table (P10-02)
			out[g] = why
		}
	}
	return out
}

// THE IDENTITY GATE SELECTS EVERY TEST THAT GUARDS THE CLASS.
// TestEveryRunSelectorSelectsATest fails a selector that reaches nothing; this
// asks for the whole set.
func TestTheIdentityGateSelectsItsTest(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	re := regexp.MustCompile(selectorOf(t, m, "lint-identity"))

	// IT MUST SELECT EVERY TEST THAT GUARDS THE CLASS, not just one. A
	// recipe naming a single test leaves any other guard running only under the
	// full race sweep, which is not the gate anyone invokes.
	//
	// THE SET IS DISCOVERED BY A MARKER, NOT BY A FILE NAME. A guard lives
	// wherever its author puts it, so a test carrying the marker below, in any
	// test file of this package, is part of the gate; keying on one file name is
	// the same hardcoding one layer out. The marker is written once, in the call
	// below — spelling it in prose would count as a marker attached to nothing,
	// which is exactly what markedTests refuses.
	marked := markedTests(t, ".", "identity-gate:")
	if len(marked) == 0 {
		t.Fatal("COULD NOT RUN — no test carries the identity-gate marker")
	}
	for _, name := range marked { // bounded by the package's test declarations (P10-02)
		if !re.MatchString(name) {
			t.Errorf("lint-identity's -run pattern %q does not select %s, which is marked as guarding the identity class", re, name)
		}
	}
}

// markedTests returns the tests in dir whose doc comment carries marker.
//
// IT READS THE SYNTAX TREE, NOT THE TEXT. Splitting a file on blank lines loses
// a marker separated from its function by one, and misattributes a marker
// sitting between two functions — both silently, which for a gate that decides
// what the gate runs is the wrong failure. `fn.Doc` is exact.
func markedTests(t *testing.T, dir, marker string) []string {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("COULD NOT RUN — %s: %v", dir, err)
	}
	var out []string
	for _, e := range entries { // bounded by the directory (P10-02)
		if !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("COULD NOT RUN — parsing %s: %v", e.Name(), err)
		}
		{
			var attached, written int
			for _, d := range f.Decls { // bounded by the file's declarations (P10-02)
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Doc == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
					continue
				}
				if strings.Contains(fn.Doc.Text(), marker) {
					out = append(out, fn.Name.Name)
					attached++
				}
			}
			// EVERY MARKER WRITTEN MUST BE A MARKER ATTACHED. A blank line
			// between the marker and its function detaches the doc comment, and
			// the test then drops out of the gate with nobody told — which for
			// the check that decides what the gate runs is the wrong silence.
			// Markers in comments are counted; one written as a string, like
			// the argument below, is code and is not one.
			for _, g := range f.Comments { // bounded by the file's comments (P10-02)
				if strings.Contains(g.Text(), marker) {
					written++
				}
			}
			if written != attached {
				t.Errorf("%s: %d %q marker(s) written, %d attached to a test — a marker separated from its "+
					"function by a blank line silently leaves the gate", fset.Position(f.Pos()).Filename, written, marker, attached)
			}
		}
	}
	return out
}

// runSelector is one `go test -run` in a recipe: the target it belongs to, the
// pattern as the shell hands it to go, and the package directories it names,
// relative to the repository root ("..." suffixed for a subtree).
type runSelector struct {
	target, pattern string
	pkgs            []string
	bench           bool
}

// runFlag is `-run` with its pattern quoted either way or bare.
var runFlag = regexp.MustCompile(`-run[= ]+(?:'([^']*)'|"([^"]*)"|(\S+))`)

// selectorOf is the -run pattern of target's recipe.
func selectorOf(t *testing.T, m *buildModel, target string) string {
	t.Helper()
	for _, sel := range m.runSelectors() { // bounded by the selectors (P10-02)
		if sel.target == target {
			return sel.pattern
		}
	}
	t.Fatalf("COULD NOT RUN — %s names no -run pattern", target)
	return ""
}

// runSelectors is every `go test -run` in the Makefile, in target order.
func (m *buildModel) runSelectors() []runSelector {
	var out []runSelector
	for _, name := range sorted(keys(m.targets)) { // bounded by the Makefile (P10-02)
		for _, c := range m.targets[name].cmds { // bounded by the recipe (P10-02)
			if !c.isGoTest() {
				continue
			}
			mm := runFlag.FindStringSubmatch(c.text)
			if mm == nil {
				continue
			}
			sel := runSelector{
				target:  name,
				pattern: strings.ReplaceAll(mm[1]+mm[2]+mm[3], "$$", "$"), // make's $$ is the shell's $
				bench:   strings.Contains(c.text, "-bench"),
			}
			for _, f := range strings.Fields(strings.SplitN(c.text, "|", 2)[0]) { // bounded by the command (P10-02)
				if strings.HasPrefix(f, "./") {
					sel.pkgs = append(sel.pkgs, strings.TrimSuffix(strings.TrimPrefix(f, "./"), "/"))
				}
			}
			out = append(out, sel)
		}
	}
	return out
}

func keys(m map[string]*target) []string {
	out := make([]string, 0, len(m))
	for k := range m { // bounded by the map (P10-02)
		out = append(out, k)
	}
	return out
}

// inPackages says whether dir is one of pkgs, a subtree ("x/...") counting
// every directory under x and "..." every directory.
func inPackages(dir string, pkgs []string) bool {
	for _, p := range pkgs { // bounded by the recipe's packages (P10-02)
		switch {
		case p == "...":
			return true
		case strings.HasSuffix(p, "/..."):
			root := strings.TrimSuffix(p, "/...")
			if dir == root || strings.HasPrefix(dir, root+"/") {
				return true
			}
		case dir == p:
			return true
		}
	}
	return false
}

// assertEveryRunSelectsATest: every `-run` in a recipe selects at least one
// test in the packages that recipe names. tests maps a package directory to
// the test functions declared in it.
//
// A -run THAT SELECTS NOTHING EXITS 0. Rename the test, or move it to another
// package, and the gate runs zero tests and reports green (FR-11.3). Only the
// top level of the pattern is matched, as go matches it against a test name;
// `^$` beside -bench selects no test by design and is not a gate's selector.
func assertEveryRunSelectsATest(t reporter, m *buildModel, tests map[string][]string) {
	t.Helper()
	sels := m.runSelectors()
	if len(sels) == 0 {
		t.Fatalf("COULD NOT RUN — no recipe runs `go test -run`; the Makefile shape has changed and " +
			"this check has lost its subject")
	}
	for _, sel := range sels { // bounded by the selectors (P10-02)
		top := strings.SplitN(sel.pattern, "/", 2)[0]
		if sel.bench && top == "^$" {
			continue
		}
		re, err := regexp.Compile(top)
		if err != nil {
			t.Errorf("%s: -run %q does not compile: %v", sel.target, sel.pattern, err)
			continue
		}
		if len(sel.pkgs) == 0 {
			t.Errorf("%s: -run %q names no package this check can read", sel.target, sel.pattern)
			continue
		}
		var matched int
		for dir, names := range tests { // bounded by the tree (P10-02)
			if !inPackages(dir, sel.pkgs) {
				continue
			}
			for _, n := range names { // bounded by the package (P10-02)
				if re.MatchString(n) {
					matched++
				}
			}
		}
		if matched == 0 {
			t.Errorf("%s: -run %q selects no test in %v. A selector that reaches nothing runs zero "+
				"tests and exits 0, so the gate passes by asking nothing.", sel.target, sel.pattern, sel.pkgs)
		}
	}
}

// EVERY -run IN THE MAKEFILE SELECTS A TEST, in the packages its recipe names.
func TestEveryRunSelectorSelectsATest(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	assertEveryRunSelectsATest(t, m, testsByDir(t, "../.."))
}

// testsByDir maps each package directory under root, relative and
// slash-separated, to the test functions its _test.go files declare. Build
// tags are not consulted: a tagged test is still the test its gate selects.
func testsByDir(t *testing.T, root string) map[string][]string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)
	out := map[string][]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == "third_party" || n == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, filepath.Dir(p))
		if err != nil {
			return err
		}
		dir := filepath.ToSlash(rel)
		for _, mm := range re.FindAllStringSubmatch(string(src), -1) { // bounded by the file (P10-02)
			out[dir] = append(out[dir], mm[1])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 10 {
		t.Fatalf("COULD NOT RUN — found tests in %d package(s) under %s", len(out), root)
	}
	return out
}

// THE PROPERTY CATCHES A SELECTOR THAT REACHES NOTHING, and passes one that
// reaches its tests. The base fixture's alloc-budget selects `AllocBudget`
// across ./...; quality-bench's `^$` beside -bench is a benchmark run that
// selects no test by design.
func TestTheRunSelectorGuardCatchesAnEmptySelection(t *testing.T) {
	tests := map[string][]string{"x": {"TestAllocBudgetOne"}, "y": {"TestOther"}}
	cases := []struct {
		name   string
		mk     func(string) string
		caught bool
	}{
		{"the base fixture", gateoracle.Same, false},
		{"a pattern that matches no test",
			gateoracle.Sub("-run 'AllocBudget' ./...", "-run 'AllocBugdet' ./..."), true},
		{"a pattern matching a test only in a package the recipe does not name",
			gateoracle.Sub("-run 'AllocBudget' ./...", "-run 'AllocBudget' ./y"), true},
		{"an anchored pattern with make's doubled dollar",
			gateoracle.Sub("-run 'AllocBudget' ./...", "-run '^TestAllocBudgetOne$$' ./x"), false},
	}
	for _, c := range cases { // bounded by the case table (P10-02)
		t.Run(c.name, func(t *testing.T) {
			mk := c.mk(gateoracle.BaseMakefile)
			if anchor := gateoracle.MissingAnchor(mk); anchor != "" {
				t.Fatalf("the case's anchor is not in the base fixture: %q", anchor)
			}
			m := newBuildModel(mk, baseWorkflow, gateoracle.BaseRequired)
			fired, said := gateoracle.VerdictOf(func(r reporter) { assertEveryRunSelectsATest(r, m, tests) })
			switch {
			case c.caught && !fired:
				t.Errorf("SURVIVED — a -run that selects nothing passed the guard")
			case !c.caught && fired:
				t.Errorf("FALSE POSITIVE:\n  %s", strings.Join(said, "\n  "))
			}
		})
	}
}
