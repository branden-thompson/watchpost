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
// reason and — where it is a decision rather than a fact — who owes it.
//
// THIS IS A HOLE, NOT A DESIGN, and it is written down so it is visible rather
// than discovered. A PR can merge today with a survived mutant, because the
// strongest gate in this repository runs only where somebody remembers to run
// it.
var verifyOnly = map[string]string{
	"mutant-check": "COST DECISION, HUM LEAD, pending: 171 mutants take 261 s on twelve cores here, " +
		"which is roughly fifteen to twenty minutes on a four-core runner — per push. Running it " +
		"per-push is correct on the merits and expensive on the clock; running it nightly leaves a " +
		"window where a survived mutant is merged.",
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

// verifyTargets is the verify recipe's prerequisites, from the Makefile.
func verifyTargets(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("../../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
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
	b, err := os.ReadFile("../../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^\s*-\s*run:\s*make\s+([a-z-]+)`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
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
