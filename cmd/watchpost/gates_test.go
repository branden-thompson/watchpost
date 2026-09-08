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
var verifyOnly = map[string]string{}

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
	for _, g := range required {
		if !has(verify, g) {
			t.Errorf("%s is required and `make verify` does not run it", g)
		}
		// mutant-check runs on a schedule CI decides (MUTANT_POLICY), so its
		// absence from a given workflow read is not a finding; every other
		// required gate must be on both machines.
		if g != "mutant-check" && !has(ci, g) {
			t.Errorf("%s is required and CI does not run it", g)
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
func verifyTargets(t *testing.T) []string {
	t.Helper()
	for _, line := range makeLines(t) {
		if after, ok := strings.CutPrefix(line, "verify:"); ok {
			return sorted(strings.Fields(after))
		}
	}
	t.Fatal("the Makefile has no verify target; this test measures nothing")
	return nil
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
