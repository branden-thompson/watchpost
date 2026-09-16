package main

// gates_test.go — FR-8: `make verify` and CI run the same gates, and the test
// says so in BOTH directions.
//
// THE HAZARD IS A GATE THAT EXISTS AND DOES NOT RUN. Before this, `verify`
// carried vet-tags, test-tags and mutant-check that CI never ran, while CI
// carried alloc-budget, release-matrix and install-test that `verify` never ran
// — so "the gates passed" meant a different set depending on who said it, and
// nothing anywhere compared them.
//
// A NAIVE CONVERGENCE WOULD HAVE DELETED THE ALLOCATION GATE. Making CI run
// only what `verify` runs was the obvious fix and the wrong one: alloc-budget
// was CI-only, so the "same set" it converged on was the smaller one. The
// direction that matters is the union, with the exceptions named.

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ciOnly are gates CI runs that a local `make verify` deliberately does not,
// with the reason. A reason is not a silencer: a gate that starts running in
// both fails here, so the row cannot outlive its truth.
var ciOnly = map[string]string{
	"release-matrix": "it builds five platforms; a local verify would spend minutes producing artifacts nobody is about to publish",
	"install-test":   "it installs the artifacts release-matrix built, so it cannot run without them",
}

// verifyOnly are gates a local `make verify` runs that CI does not, with the
// reason. Empty today: mutant-check was the last one, and it runs per push.
//
// A reason is not a silencer — a row for a gate that runs in both fails, and so
// does one for a gate that runs in neither.
var verifyOnly = map[string]string{
	"p10": "the P10 harness CLI and its exemptions ledger live OUTSIDE the public tree (the ledger is .gitignore'd, red-team R2-2), so CI has no `a2dh` and no file to read; it is a local gate that must fail loud rather than skip",
}

// mutantModes are the schedules the mutant corpus can be put on.
//
// ALL THREE ARE WIRED AT ONCE, whichever is chosen (HUM LEAD, 2026-09-08):
// "I want any choice to be non-destructive... vs. simply flipping a switch."
// So the workflow carries every mode's condition and the Makefile carries one
// word, and this test fails if a mode's path is ever removed to make room for
// another — which is what would turn the next switch back into a rewrite.
var mutantModes = []string{"push", "nightly", "label"}

// THE SWITCH IS A WORD, AND STAYS ONE.
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
	// EVERY MODE'S CONDITION IS PRESENT, not only the chosen one.
	for _, mode := range mutantModes {
		if !strings.Contains(ci, "'"+mode+"'") {
			t.Errorf("the workflow does not mention the %q mode: switching to it would be a "+
				"workflow change, which is the thing this arrangement exists to avoid", mode)
		}
	}
	// AND THE TRIGGERS EVERY MODE NEEDS. A schedule with no cron cannot run
	// nightly; a pull_request that does not fire on `labeled` cannot run on a
	// label, however carefully the condition is written.
	for _, need := range []string{"schedule:", "cron:", "labeled"} {
		if !strings.Contains(ci, need) {
			t.Errorf("the workflow has no %q, so at least one mode cannot fire", need)
		}
	}
	// AND CI READS THE DECISION rather than carrying a second copy of it.
	if !strings.Contains(ci, "make -s mutant-policy") {
		t.Error("the workflow does not read the policy from the Makefile: two copies of a decision " +
			"is one that can disagree with itself")
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCIAndVerifyRunTheSameGates(t *testing.T) {
	verify := verifyTargets(t)
	ci := ciTargets(t)

	for _, g := range verify {
		if contains(ci, g) {
			continue
		}
		if why, ok := verifyOnly[g]; ok {
			if why == "" {
				t.Errorf("%s is declared verify-only with no reason", g)
			}
			continue
		}
		t.Errorf("`make verify` runs %s and CI does not, and nothing says why: a gate that only "+
			"runs on one machine is a gate that passes on the other by not being asked", g)
	}
	for _, g := range ci {
		if contains(verify, g) {
			continue
		}
		if why, ok := ciOnly[g]; ok {
			if why == "" {
				t.Errorf("%s is declared CI-only with no reason", g)
			}
			continue
		}
		t.Errorf("CI runs %s and `make verify` does not, and nothing says why", g)
	}
	for g := range ciOnly {
		if !contains(ci, g) {
			t.Errorf("%s is declared CI-only and CI does not run it: the row outlived the gate", g)
		}
		if contains(verify, g) {
			t.Errorf("%s is declared CI-only and `make verify` runs it too: delete the row, or the "+
				"next gate to go missing here will look expected", g)
		}
	}
	for g := range verifyOnly {
		if !contains(verify, g) {
			t.Errorf("%s is declared verify-only and verify does not run it: the row outlived the gate", g)
		}
		if contains(ci, g) {
			t.Errorf("%s is declared verify-only and CI runs it now: delete the row", g)
		}
	}
}

// THE REQUIRED SET IS A THIRD LIST, and without it this file proved only that
// two lists agree (red team, 2026-09-08). Deleting alloc-budget from `verify`
// AND ci.yml passed — the repository's only allocation gate gone from both
// machines, which is precisely the loss TestCIAndVerifyRunTheSameGates was
// written to prevent. Two lists can agree by both being wrong.
func TestEveryRequiredGateIsStillRun(t *testing.T) {
	var required []string
	for _, l := range strings.Split(read(t, "../../06_docs/required-gates.txt"), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			required = append(required, l)
		}
	}
	if len(required) == 0 {
		t.Fatal("the required-gates list is empty; this test measures nothing")
	}
	verify, ci := verifyTargets(t), ciTargets(t)
	has := func(set []string, want string) bool {
		for _, g := range set {
			if g == want {
				return true
			}
		}
		return false
	}
	// A REQUIRED GATE RUNS SOMEWHERE, and CI-only is a legitimate somewhere:
	// release-matrix inspects the published artifacts, which do not exist when
	// verify runs. What is not legitimate is running NOWHERE.
	ciOnlyOK := map[string]bool{"release-matrix": true, "install-test": true}
	for _, g := range required {
		if !has(verify, g) && !ciOnlyOK[g] {
			t.Errorf("%s is required and `make verify` does not run it", g)
		}
		if ciOnlyOK[g] && !has(ci, g) {
			t.Errorf("%s is required and CI-only, and CI does not run it either — it now runs NOWHERE", g)
		}
		// TWO GATES ARE LEGITIMATELY ABSENT FROM CI, AND BOTH SAY SO WHERE THE
		// REASON LIVES. `verifyOnly` carries the declared local-only gates;
		// mutant-check runs on a schedule CI decides (MUTANT_POLICY), so its
		// absence from a given workflow read is not a finding. Every other
		// required gate must be on both machines.
		//
		// IT READS `verifyOnly` RATHER THAN NAMING THE GATES AGAIN. A condition
		// listing them here would be a second copy of the map above, free to
		// disagree with it — which is the defect this whole file exists to stop,
		// committed inside the file that stops it.
		if _, localOnly := verifyOnly[g]; !localOnly && g != "mutant-check" && !has(ci, g) {
			t.Errorf("%s is required and CI does not run it, and nothing declares it local-only: "+
				"add a row to `verifyOnly` with the reason, or wire it into ci.yml", g)
		}
	}
}

// makeLines is the Makefile with BACKSLASH CONTINUATIONS JOINED.
//
// A recipe split across lines is one command, and reading it as two loses
// whichever half carries the thing you are looking for: build-diag puts
// `go build` on one line and `./cmd/watchpost` on the next, so a scan keyed on
// both silently dropped a shipping target — and verify's prerequisite list has
// the same shape waiting for the day it grows a continuation.
func makeLines(t *testing.T) []string {
	t.Helper()
	var out []string
	var cur string
	for _, l := range strings.Split(read(t, "../../Makefile"), "\n") {
		if strings.HasSuffix(l, "\\") {
			cur += strings.TrimSuffix(l, "\\") + " "
			continue
		}
		out = append(out, cur+l)
		cur = ""
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// verifyTargets is the verify recipe's prerequisites, from the Makefile.
// verifyTargets is every gate `make verify` reaches.
//
// IT FOLLOWS THE DELEGATION. `verify` holds the tree lock and hands the gate
// list to `verify-gates`, so reading `verify:`'s own prerequisites would find a
// list of one and report every CI gate as unrun. A target whose prerequisites
// are a single other target IS that target for this test's purposes — and the
// substitution is bounded to one hop, because a chain is a place for a gate to
// hide.
func verifyTargets(t *testing.T) []string {
	t.Helper()
	deps, ok := targetDeps(t, "verify")
	if !ok {
		t.Fatal("the Makefile has no verify target; this test measures nothing")
	}
	if len(deps) == 0 {
		inner, found := targetDeps(t, delegate(t, "verify"))
		if !found || len(inner) == 0 {
			t.Fatal("verify has neither gates nor a target to delegate them to; this test measures nothing")
		}
		return sorted(inner)
	}
	return sorted(deps)
}

// targetDeps is one target's prerequisites, and whether it has a rule at all.
func targetDeps(t *testing.T, name string) ([]string, bool) {
	t.Helper()
	if name == "" {
		return nil, false
	}
	for _, line := range makeLines(t) {
		if after, ok := strings.CutPrefix(line, name+":"); ok {
			return strings.Fields(after), true
		}
	}
	return nil, false
}

// delegate is the target a prerequisite-less rule hands its work to.
//
// ONE HOP, AND ONLY FROM A RULE WITH NO PREREQUISITES. A chain of delegations
// is a place for a gate to hide, and following an arbitrary depth would make
// this test agree with a Makefile no reader could follow.
func delegate(t *testing.T, name string) string {
	t.Helper()
	re := regexp.MustCompile(`\$\(MAKE\)(?:\s+--[a-z-]+)*\s+([a-z][a-z0-9-]*)`)
	lines := makeLines(t)
	for i, line := range lines {
		if !strings.HasPrefix(line, name+":") {
			continue
		}
		for _, r := range lines[i+1:] { // bounded by the file (P10-02)
			if !strings.HasPrefix(r, "\t") {
				return "" // the recipe ended without delegating
			}
			if m := re.FindStringSubmatch(r); m != nil {
				return m[1]
			}
		}
	}
	return ""
}

// ciTargets is every `make <target>` ci.yml runs.
func ciTargets(t *testing.T) []string {
	t.Helper()
	// `- run: make x` and `- name: x` + `run: make x` both count: a step with a
	// name is still a step that runs the gate.
	re := regexp.MustCompile(`(?m)^\s*(?:-\s*)?run:\s*make\s+([a-z-]+)`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(read(t, "../../.github/workflows/ci.yml"), -1) {
		out = append(out, m[1])
	}
	if len(out) == 0 {
		t.Fatal("ci.yml runs no make targets; this test measures nothing")
	}
	return sorted(out)
}

func sorted(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func contains(hay []string, want string) bool {
	for _, h := range hay {
		if h == want {
			return true
		}
	}
	return false
}

// THE BUILD PATH IS NOT SHIPPED (FR-7.5, HUM LEAD 2026-09-08).
//
// Without -trimpath every binary embeds the absolute directory it was compiled
// from, which names the person who built it: 473 occurrences in
// watchpost-darwin-arm64 and 485 in linux-amd64 when this was measured. The
// exposure scan found it; a documentation survey never would, because it never
// opens a binary.
//
// THIS ASKS THE MAKEFILE, NOT A BINARY. `go test` builds its own test binary
// with its own flags, so inspecting the running one would measure the test
// harness. Every target that produces a shippable artifact must carry the flag,
// and a new target that forgets it fails here rather than at a release.
func TestEveryBuildTargetTrimsThePath(t *testing.T) {
	lines := makeLines(t)
	var built []string
	for _, l := range lines {
		// A RECIPE LINE, NOT PROSE. The Makefile's comments discuss `go build`
		// — one explains why build-diag exists rather than "a bare go build" —
		// and the first version of this gate failed on that sentence. A gate
		// that reads documentation as configuration reports a defect in a
		// paragraph.
		// ANY RECIPE LINE THAT BUILDS A SHIPPED ARTIFACT, not the literal
		// "go build". One Make variable — $(GO) build — hid a whole target
		// from this scan (red team, 2026-09-08), and a target invisible to
		// the gate is a target that ships the path it was compiled from.
		if strings.HasPrefix(strings.TrimSpace(l), "#") || !strings.Contains(l, "build ") || !strings.Contains(l, "./cmd/") {
			continue
		}
		built = append(built, strings.TrimSpace(l))
	}
	if len(built) == 0 {
		t.Fatal("no `go build` line found in the Makefile: this gate is looking at the wrong file")
	}
	for _, l := range built {
		if !strings.Contains(l, "$(TRIMPATH)") && !strings.Contains(l, "-trimpath") {
			t.Errorf("a build target ships the path it was compiled from:\n  %s", l)
		}
	}
	t.Logf("%d build targets, all trimmed", len(built))
}

// unlisted are gate-shaped Makefile targets deliberately on NO gate list, with
// the reason. A reason is not a silencer: a row for a target that starts being
// listed fails here, and so does one for a target that no longer exists.
var unlisted = map[string]string{
	"quality-bench":   "a measurement, not a gate — it reports numbers and has no pass condition (INST-5)",
	"pty-severe":      "it drives a real pty, which CI has no terminal for",
	"journey":         "the VALIDATE journey against LIVE feeds; its exit code is a FAIL count, run deliberately at release",
	"test":            "`go test ./...` without the race detector; `race` supersedes it on every gate path, and running both would double the suite for no extra property",
	"test-platforms":  "it re-runs the app suite under WATCHPOST_TEST_GOOS for two other platforms; release-matrix is the cross-platform gate and this is the manual probe behind it",
	"mutant-verdicts": "the corpus SWEEP — hours, and its verdicts are promoted into the release record by hand; mutant-anchors and mutant-check are the per-push halves",
	"tree-free":       "it ASKS whether a gate run is in flight and reports; it asserts nothing about the code",
}

// EVERY GATE-SHAPED TARGET IS ON A LIST, OR SAYS WHY NOT.
//
// THE THREE-LIST RULE HAS A HOLE AND THIS IS IT. `required-gates.txt` makes a
// gate need three edits to LEAVE; it has no answer to a gate that never
// ARRIVES. `make p10` is declared "must fail loud, never skip", was RED on a
// clean tip, and appeared in verify, ci.yml and required-gates.txt zero times —
// so TestGateListsAgree could not see it, because that test compares verify
// against CI and a gate on neither list is invisible to a comparison of the two.
//
// SO THIS ASKS THE MAKEFILE, not the lists. A target whose recipe runs a gate
// script or a `go test` must be named by `required-gates.txt` or carry a row
// above saying why it is not. The cost of a gate that runs nowhere becomes a
// line a reviewer reads, which is the whole of the fix.
func TestEveryGateShapedTargetIsListedOrExempt(t *testing.T) {
	required := requiredGates(t)
	for _, target := range gateShapedTargets(t) { // bounded by the Makefile (P10-02)
		_, listed := required[target]
		why, exempt := unlisted[target]
		switch {
		case listed && exempt:
			t.Errorf("%s is on required-gates.txt AND declared unlisted (%q): delete the row, or the "+
				"reason is describing a state that is not true", target, why)
		case !listed && !exempt:
			t.Errorf("%s runs a gate and is on no list.\n"+
				"A gate nothing invokes is a gate that cannot fail — `make p10` sat that way, red, "+
				"while verify reported ALL GATES GREEN.\n"+
				"Add it to 06_docs/required-gates.txt and to the verify/CI paths, or add a row to "+
				"`unlisted` in this file saying why it runs nowhere.", target)
		}
	}
	for target, why := range unlisted { // bounded by the exemption table (P10-02)
		if _, ok := targetDeps(t, target); !ok {
			t.Errorf("%s is declared unlisted (%q) and the Makefile has no such target: the row "+
				"outlived the gate", target, why)
		}
	}
}

// requiredGates is 06_docs/required-gates.txt as a set, comments dropped.
func requiredGates(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, line := range strings.Split(read(t, "../../06_docs/required-gates.txt"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out[line] = true
	}
	if len(out) == 0 {
		t.Fatal("required-gates.txt lists nothing; this check measures nothing")
	}
	return out
}

// gateShapedTargets are the Makefile targets whose recipe RUNS a check.
//
// THE SHAPE, NOT A LIST — a list here would be the same enumeration that let
// `p10` hide. A target qualifies when its recipe invokes a script under
// scripts/, a `go test`, or a tool's own self-test, since those are the three
// forms every gate in this repository takes. Aggregators are excluded: they
// name other targets rather than running a check themselves.
func gateShapedTargets(t *testing.T) []string {
	t.Helper()
	aggregate := map[string]bool{"verify": true, "verify-gates": true, "gate-controls": true, "p10-mirror": true}
	runsACheck := regexp.MustCompile(`(\./scripts/|go test|go run \./tools/|-self-test|--self-test)`)
	target := regexp.MustCompile(`^([a-z][a-z0-9-]*):`)

	var out []string
	var cur string
	for _, line := range makeLines(t) { // bounded by the Makefile (P10-02)
		if m := target.FindStringSubmatch(line); m != nil {
			cur = m[1]
			continue
		}
		if cur == "" || !strings.HasPrefix(line, "\t") {
			if !strings.HasPrefix(line, "\t") {
				cur = ""
			}
			continue
		}
		if aggregate[cur] || !runsACheck.MatchString(line) {
			continue
		}
		if !contains(out, cur) {
			out = append(out, cur)
		}
	}
	if len(out) < 5 {
		t.Fatalf("found %d gate-shaped targets; the Makefile shape has changed and this check has "+
			"lost its subject", len(out))
	}
	return sorted(out)
}

// cacheableGate are gate recipes whose `go test` deliberately omits -count=1.
//
// A reason is not a silencer: a row for a recipe that gains the flag fails here,
// and so does one for a target that no longer exists.
var cacheableGate = map[string]string{
	"test":          "not a gate — it is the plain suite, declared `unlisted`; `race` is what runs on every gate path and it carries the flag",
	"quality-bench": "a benchmark with `-count 10`, which is the sample size rather than a cache defence; benchmarks are not cached",
}

// NO GATE IS ANSWERABLE FROM CACHE (AP-STALE-01).
//
// `go test` caches a package's result and replays it when nothing it depends on
// has changed. That is right for a developer loop and wrong for a gate: a gate
// exists to answer a question about the tree as it stands NOW, and a replayed
// PASS answers it about a tree that may be several edits old. The failure is
// silent and in the worst direction — the gate reports success, quickly, which
// is exactly what a green run looks like.
//
// THE PROJECT ALREADY KNOWS THIS, which is why almost every recipe carries
// `-count=1` already. This makes the convention mechanical, so the next recipe
// cannot quietly omit it.
func TestNoGateIsAnswerableFromCache(t *testing.T) {
	goTest := regexp.MustCompile(`go test\b`)
	target := regexp.MustCompile(`^([a-z][a-z0-9-]*):`)

	var cur string
	var checked int
	for _, line := range makeLines(t) { // bounded by the Makefile (P10-02)
		if m := target.FindStringSubmatch(line); m != nil {
			cur = m[1]
			continue
		}
		if !strings.HasPrefix(line, "\t") {
			cur = ""
			continue
		}
		if cur == "" || !goTest.MatchString(line) {
			continue
		}
		checked++
		if strings.Contains(line, "-count=1") || strings.Contains(line, "-count 1") {
			continue
		}
		if why, ok := cacheableGate[cur]; ok {
			_ = why
			continue
		}
		t.Errorf("%s runs `go test` without -count=1.\n"+
			"A cached PASS answers the question about a tree that may be several edits old, and it "+
			"looks exactly like a green run.\n"+
			"Add -count=1, or add a row to `cacheableGate` in this file saying why this one may be "+
			"answered from cache.", cur)
	}
	if checked < 5 {
		t.Fatalf("found %d `go test` recipes; the Makefile shape has changed and this check has lost "+
			"its subject", checked)
	}
	for target, why := range cacheableGate { // bounded by the exemption table (P10-02)
		if _, ok := targetDeps(t, target); !ok {
			t.Errorf("%s is declared cacheable (%q) and the Makefile has no such target: the row "+
				"outlived the recipe", target, why)
		}
	}
}

// conditionalStep are CI steps for a REQUIRED gate that legitimately carry a
// condition, with the reason. A reason is not a silencer: a row for a step that
// loses its condition fails here, and so does one for a gate CI no longer runs.
var conditionalStep = map[string]string{
	"install-test": "the matrix runs three operating systems and this installs what release-matrix built; doing it once, on Linux, is the test",
	"mutant-check": "MUTANT_POLICY decides its schedule (push / nightly / label) and all three conditions are written out, so switching between them is a word in the Makefile rather than an edit here",
}

// A REQUIRED GATE'S CI STEP MAY NOT BE SILENCED WITHOUT A REASON.
//
// THE CHEAPEST ATTACK ON THE THREE-LIST RULE, and the one it could not see.
// `ciTargets` matches `run: make <target>` and nothing else, so adding
// `if: false` — or `continue-on-error: true` — leaves the Makefile, ci.yml and
// required-gates.txt in perfect agreement while the gate never runs, or never
// fails. Conditional steps already exist in this workflow for good reasons, so
// one more would read as unremarkable in review: the shape is camouflage.
//
// IT DOES NOT BAN CONDITIONS, it bans UNDECLARED ones. `continue-on-error` is
// refused outright for a required gate, because a gate that cannot fail is not a
// gate whatever the reason.
//
// PARSED BY INDENTATION RATHER THAN BY A YAML LIBRARY. The project's tooling is
// stdlib-only on purpose, and a dependency taken for one test is a dependency the
// whole build carries. The shape this reads — a step opening with `- ` and its
// keys indented under it — is the shape of every step in the file, and the
// subject check below fails if that stops being true.
func TestNoRequiredGatesCIStepIsSilentlyConditional(t *testing.T) {
	required := requiredGates(t)
	steps := ciSteps(t)
	if len(steps) < 10 {
		t.Fatalf("parsed %d CI steps; the workflow's shape has changed and this check has lost its subject", len(steps))
	}

	seen := map[string]bool{}
	makeTarget := regexp.MustCompile(`make\s+([a-z][a-z0-9-]*)`)
	for _, step := range steps { // bounded by the workflow (P10-02)
		m := makeTarget.FindStringSubmatch(step)
		if m == nil || !required[m[1]] {
			continue
		}
		gate := m[1]
		seen[gate] = true
		conditional := regexp.MustCompile(`(?m)^\s+if:`).MatchString(step)
		tolerant := regexp.MustCompile(`(?m)^\s+continue-on-error:\s*true`).MatchString(step)
		why, declared := conditionalStep[gate]

		if tolerant {
			t.Errorf("%s is a REQUIRED gate and its CI step sets continue-on-error: true.\n"+
				"A gate that cannot fail is not a gate. Remove it, or remove the gate from "+
				"06_docs/required-gates.txt and say so.", gate)
		}
		switch {
		case conditional && !declared:
			t.Errorf("%s is a REQUIRED gate and its CI step carries an `if:` that nothing declares.\n"+
				"All three gate lists agree while the step may never run — which is the cheapest way "+
				"to retire a gate without deleting it.\n"+
				"Add a row to `conditionalStep` in this file with the reason, or remove the condition.", gate)
		case !conditional && declared:
			t.Errorf("%s is declared conditional (%q) and its CI step carries no condition: the row "+
				"outlived the reason", gate, why)
		}
	}
	for gate := range conditionalStep { // bounded by the exemption table (P10-02)
		if !seen[gate] {
			t.Errorf("%s is declared conditional and CI runs no such required gate: the row outlived the step", gate)
		}
	}
}

// ciSteps splits the workflow into one string per step, keys included.
//
// A step opens with `- ` and owns every following line indented deeper than its
// own dash — which covers a block scalar (`if: >-`) without understanding one.
func ciSteps(t *testing.T) []string {
	t.Helper()
	dash := regexp.MustCompile(`^(\s*)-\s`)
	var out []string
	var cur strings.Builder
	indent := -1
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, line := range strings.Split(read(t, "../../.github/workflows/ci.yml"), "\n") {
		if m := dash.FindStringSubmatch(line); m != nil {
			flush()
			indent = len(m[1])
			cur.WriteString(line + "\n")
			continue
		}
		if indent < 0 {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line)-len(strings.TrimLeft(line, " ")) > indent {
			cur.WriteString(line + "\n")
			continue
		}
		flush()
		indent = -1
	}
	flush()
	return out
}

// uncontrolled are custom checkers a required gate invokes that have no positive
// control, with the reason. A reason is not a silencer: a row for a checker that
// gains a control fails here, and so does one no required gate invokes any more.
//
// EVERY ROW HERE IS A GATE WE ARE TRUSTING WITHOUT EVIDENCE. The list is meant
// to be short and to shrink; it is not a place to park work.
var uncontrolled = map[string]string{
	"scripts/lint.sh":                      "F-118 — it discards golangci-lint's exit code with `|| true` and treats non-empty JSON as liveness, so it does not fail closed. A control would pin the behaviour we intend to CHANGE; it is slated for conversion to Go",
	"scripts/install-test.sh":              "it installs and runs a built artifact, so a control would be a second installation on a machine that has just done one; it is CI-only and runs on a clean runner",
	"scripts/quality/p10-ledger-mirror.py": "a GENERATOR, and its control is the linter that runs immediately after it: `lint-ledger.sh` reads the file this writes, in the same recipe, and that linter has its own self-test",
}

// EVERY REQUIRED GATE'S CHECKER HAS A POSITIVE CONTROL, AND THE CONTROL RUNS.
//
// THE PROJECT'S OWN CALIBRATION SAYS SO ("Guard Tests Require Positive Controls")
// and `gate-controls` is a HAND-WRITTEN LIST, which is the enumerate-don't-discover
// shape this repository warns against elsewhere. Two controls were found this
// round that existed and were invoked by nothing — `lint-injector`'s, the only
// thing between a hazard-fabricating build and a release, and `dupes-selftest`.
// Both were reachable by reading; neither was reachable by any gate.
//
// SO THIS DISCOVERS THE CHECKERS rather than listing them: every script under
// `scripts/` and every `go run ./tools/<x>` that a REQUIRED gate invokes must be
// exercised somewhere with a self-test flag, or by a sibling `_test.sh`.
//
// THE STANDARD TOOLCHAIN NEEDS NO CONTROL. `gofmt`, `go vet`, `go test`,
// `go mod` and `govulncheck` are not ours, and a positive control for `go vet`
// would be a test of Go. Only the checkers this project wrote are in scope.
func TestEveryRequiredGatesCheckerHasAControl(t *testing.T) {
	required := requiredGates(t)
	exercised := controlledCheckers(t)

	var checked int
	for gate := range required { // bounded by the gate list (P10-02)
		for _, checker := range checkersInvokedBy(t, gate) { // bounded by the recipe (P10-02)
			checked++
			if exercised[checker] {
				continue
			}
			if why, ok := uncontrolled[checker]; ok {
				_ = why
				continue
			}
			t.Errorf("%s is invoked by the required gate %s and nothing exercises it with a self-test.\n"+
				"A control that does not run is the same as no control, with the paperwork of one — "+
				"two were found that way this release.\n"+
				"Add `<checker> --self-test` to gate-controls, or add a row to `uncontrolled` in this "+
				"file saying why this checker is trusted without evidence.", checker, gate)
		}
	}
	if checked < 5 {
		t.Fatalf("resolved %d checkers across the required gates; the Makefile shape has changed and "+
			"this check has lost its subject", checked)
	}
	for checker, why := range uncontrolled { // bounded by the exemption table (P10-02)
		if exercised[checker] {
			t.Errorf("%s is declared uncontrolled (%q) and IS exercised: delete the row", checker, why)
		}
	}
}

// checkerRef finds a project-written checker in a recipe line.
var checkerRef = regexp.MustCompile(`(?:\./)?(scripts/[A-Za-z0-9_/.-]+\.(?:sh|py)|tools/[a-z][a-z0-9]*)`)

// checkersInvokedBy is every project-written checker one target's recipe runs.
func checkersInvokedBy(t *testing.T, target string) []string {
	t.Helper()
	var out []string
	for _, line := range recipeOf(t, target) { // bounded by the recipe (P10-02)
		for _, m := range checkerRef.FindAllStringSubmatch(line, -1) {
			if strings.HasSuffix(m[1], "_test.sh") { // a control is not a checker
				continue
			}
			if !contains(out, m[1]) {
				out = append(out, m[1])
			}
		}
	}
	return out
}

// controlledCheckers is every checker exercised with a self-test anywhere in the
// Makefile — by gate-controls, or by its own gate line as lint-authoring does.
//
// A SIBLING `_test.sh` COUNTS. `p10-unmatched.sh`'s control is a separate script
// rather than a flag, which is a shape and not an omission.
func controlledCheckers(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	selfTest := regexp.MustCompile(`--?self-test`)
	for _, line := range makeLines(t) { // bounded by the Makefile (P10-02)
		if !strings.HasPrefix(line, "\t") {
			continue
		}
		for _, m := range checkerRef.FindAllStringSubmatch(line, -1) {
			switch {
			case selfTest.MatchString(line):
				out[m[1]] = true
			case strings.HasSuffix(m[1], "_test.sh"):
				out[strings.TrimSuffix(m[1], "_test.sh")+".sh"] = true
			}
		}
	}
	return out
}

// recipeOf is one target's recipe lines.
func recipeOf(t *testing.T, target string) []string {
	t.Helper()
	var out []string
	var in bool
	for _, line := range makeLines(t) { // bounded by the Makefile (P10-02)
		if strings.HasPrefix(line, target+":") {
			in = true
			continue
		}
		if !in {
			continue
		}
		if strings.HasPrefix(line, "\t") {
			out = append(out, line)
			continue
		}
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		break
	}
	return out
}
