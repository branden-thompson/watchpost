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
// every list and silenced in the workflow. Each was a real finding before it was
// a test.

import (
	"github.com/branden-thompson/watchpost/tools/gateoracle"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
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
		"quality-bench":   "a measurement, not a gate — it reports numbers and has no pass condition (INST-5)",
		"pty-severe":      "it drives a real pty, which CI has no terminal for",
		"journey":         "the VALIDATE journey against LIVE feeds; its exit code is a FAIL count, run deliberately at release",
		"test":            "`go test ./...` without the race detector; `race` supersedes it on every gate path, and running both would double the suite for no extra property",
		"test-platforms":  "it re-runs the app suite under WATCHPOST_TEST_GOOS for two other platforms; release-matrix is the cross-platform gate and this is the manual probe behind it",
		"mutant-verdicts": "the corpus SWEEP — hours, and its verdicts are promoted into the release record by hand; mutant-anchors and mutant-check are the per-push halves",
		"tree-free":       "it ASKS whether a gate run is in flight and reports; it asserts nothing about the code",
		"build-diag":      "it builds the diagnostics binary and asserts the injector is IN it — a build with a check, run by hand before a diagnostics session; release-matrix is the gated form and runs lint-injector against every shipped artifact",
		"lint-update":     "it REWRITES the golangci baseline; it is the opposite of a gate, and running it on a gate path would erase the ratchet",
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
// a gate that never ARRIVES. `make p10` was declared "must fail loud, never
// skip", was RED on a clean tip, and was on zero lists — invisible to a
// comparison of two lists it was on neither of. This asks the Makefile.
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
// binary in the working directory, which is where a 4.7 MB one was committed
// from.
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

// THE ALLOC PINS ARE REACHED, AND A SELECTOR THAT REACHES NOTHING FAILS.
//
// `alloc-budget` selects its pins by a name pattern, which is the one gate whose
// subject can vanish without a compile error: rename the pins and the gate runs
// zero tests and exits 0. FR-11.3 says that silence is a verdict. This counts.
func TestTheAllocBudgetSelectsItsPins(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	tg := m.targets["alloc-budget"]
	if tg == nil {
		t.Fatal("COULD NOT RUN — no alloc-budget target")
	}
	pattern := ""
	for _, c := range tg.cmds { // bounded by the recipe (P10-02)
		if mm := regexp.MustCompile(`-run\s+'([^']+)'`).FindStringSubmatch(c.text); mm != nil {
			pattern = mm[1]
		}
	}
	if pattern == "" {
		t.Fatal("COULD NOT RUN — alloc-budget names no -run pattern")
	}
	re := regexp.MustCompile(pattern)
	var pins int
	for _, f := range testFilesUnder(t, "../..") { // bounded by the tree (P10-02)
		for _, name := range regexp.MustCompile(`(?m)^func (Test\w+)\(`).FindAllStringSubmatch(f, -1) {
			if re.MatchString(name[1]) {
				pins++
			}
		}
	}
	if pins < 8 {
		t.Errorf("alloc-budget's pattern %q selects %d pin(s); there were eight. A selector that "+
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

// THE IDENTITY GATE SELECTS ITS TEST (REVIEW 2026-09-17, Code Quality). The
// recipe is `go test ./cmd/watchpost -run PublishedTreeNames`; rename the test
// and zero tests run, exit 0 — the gate passes by asking nothing. The alloc
// budget got this guard; this is the same guard for the same shape.
func TestTheIdentityGateSelectsItsTest(t *testing.T) {
	m := loadBuildModel(t)
	m.mustRun(t)
	tg := m.targets["lint-identity"]
	if tg == nil {
		t.Fatal("COULD NOT RUN — no lint-identity target")
	}
	pattern := ""
	for _, c := range tg.cmds { // bounded by the recipe (P10-02)
		if mm := regexp.MustCompile(`-run\s+'?([^'\s]+)'?`).FindStringSubmatch(c.text); mm != nil {
			pattern = mm[1]
		}
	}
	if pattern == "" {
		t.Fatal("COULD NOT RUN — lint-identity names no -run pattern")
	}
	re := regexp.MustCompile(pattern)
	var matched int
	for name := range testNames(t, "../..") { // bounded by the tree (P10-02)
		if re.MatchString(name) {
			matched++
		}
	}
	if matched == 0 {
		t.Errorf("lint-identity's -run pattern %q selects no test: the gate would run zero tests and exit 0", pattern)
	}
}
