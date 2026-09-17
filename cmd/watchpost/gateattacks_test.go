package main

// gateattacks_test.go — 06_docs/gate-attack-list.md, executable.
//
// EVERY ROW OF THE LIST IS A SPECIMEN HERE, and the list was written and
// committed BEFORE the model it holds to account. Three remediation rounds each
// verified a fix against the spellings its author happened to think of; the
// fourth wrote the attacks first. An attack thought of after the fact is added
// to both files and re-run — never verified ad hoc and declared closed.
//
// BOTH DIRECTIONS. A gate that flags everything passes every CAUGHT row and is
// useless, so every attack has a PASSES twin: the base fixture itself, which
// must be clean under every assertion, and the legitimate spellings that must
// stay green.

import (
	"strings"
	"testing"
)

// baseMakefile is a small Makefile that is CORRECT under every gate.
const baseMakefile = `
.PHONY: verify verify-gates race test-tags lint-a lint-b gate-controls alloc-budget build-check mutant-anchors mutant-check release-matrix install-test
verify:
	@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates

verify-gates: race test-tags lint-a lint-b gate-controls alloc-budget build-check mutant-anchors mutant-check

race:
	go test -race -count=1 ./...

test-tags:
	go test -tags watchpost_debug -count=1 ./app

test:
	go test ./...

lint-a:
	@./scripts/lint-a.sh

lint-b:
	@scripts/lint-b.sh

mutant-anchors:
	@./scripts/quality/mutant-anchors.sh

gate-controls:
	@./scripts/lint-a.sh --self-test
	@./scripts/lint-b.sh --self-test
	@./scripts/quality/mutant-anchors.sh --self-test
	# a real comment beside a live control is fine
	@./scripts/quality/p10-unmatched_test.sh

alloc-budget:
	go test -count=1 -run 'AllocBudget' ./...

build-check:
	go build $(TRIMPATH) -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY) ./cmd/watchpost

mutant-check:
	@go test -tags mutants -count=1 ./06_docs/mutants/

release-matrix:
	go build $(TRIMPATH) -o dist/x-linux ./cmd/watchpost

install-test: release-matrix
	@./scripts/install-test.sh

quality-bench:
	go test ./x -run '^$$' -bench . -count 10

journey: build
	@expect scripts/quality/validate-journey.expect
`

// baseWorkflow is a small ci.yml that is CORRECT under every gate.
const baseWorkflow = `
name: ci
on:
  push:
jobs:
  policy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "mutants=$(make -s mutant-policy)"
  verify:
    needs: policy
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - run: make race
      - run: make test-tags
      - run: make lint-a
      - run: make lint-b
      - run: make gate-controls
      - run: make alloc-budget
      - run: make build-check
      - run: make mutant-anchors
      - run: make release-matrix
      - run: make install-test
        if: runner.os == 'Linux'
      - name: mutant-check
        if: >-
          runner.os == 'Linux' && (
            needs.policy.outputs.mutants == 'push'
          )
        run: make mutant-check
`

const baseRequired = `
race
test-tags
lint-a
lint-b
gate-controls
alloc-budget
build-check
mutant-anchors
mutant-check
release-matrix
install-test
`

type specimen struct {
	name string
	mk   func(string) string // mutate the Makefile
	ci   func(string) string // mutate the workflow
	req  func(string) string // mutate the required list
	// assert is the property under attack. Every assertion is run against a
	// model whose sources may be empty, so mustRun's verdict is part of it.
	assert func(reporter, *buildModel)
	caught bool // true: the gate MUST fire; false: it MUST stay green
}

func same(s string) string { return s }

func sub(old, new string) func(string) string {
	return func(s string) string {
		if !strings.Contains(s, old) {
			panic("specimen anchor not found: " + old)
		}
		return strings.Replace(s, old, new, 1)
	}
}

func all(t reporter, m *buildModel) {
	m.mustRun(t)
	assertThreeListsAgree(t, m)
	assertGateShapedListed(t, m)
	assertNoSilencedCIStep(t, m)
	assertNoCachedGate(t, m)
	assertEveryBuildTrimmed(t, m)
}

func TestTheGateAttackList(t *testing.T) {
	specimens := []specimen{
		// ---- the control: the base itself ---------------------------------------
		{name: "base fixture is clean under every gate", mk: same, ci: same, req: same, assert: all, caught: false},

		// ---- A. the Makefile model ------------------------------------------------
		{name: "A1 a gate-shaped target on no list",
			mk: sub("quality-bench:", "sneaky:\n\t@./scripts/sneaky.sh\n\nquality-bench:"), ci: same, req: same,
			assert: assertGateShapedListed, caught: true},
		{name: "A2 a gate spelled scripts/x.sh with no ./",
			mk: sub("quality-bench:", "sneaky:\n\t@scripts/sneaky.sh\n\nquality-bench:"), ci: same, req: same,
			assert: assertGateShapedListed, caught: true},
		{name: "A3 a gate spelled python3 scripts/x.py",
			mk: sub("quality-bench:", "sneaky:\n\t@python3 scripts/sneaky.py\n\nquality-bench:"), ci: same, req: same,
			assert: assertGateShapedListed, caught: true},
		{name: "A4 a gate spelled bash scripts/x.sh",
			mk: sub("quality-bench:", "sneaky:\n\t@bash scripts/sneaky.sh\n\nquality-bench:"), ci: same, req: same,
			assert: assertGateShapedListed, caught: true},
		{name: "A11 a gate running go test without -count=1",
			mk: sub("go test -race -count=1 ./...", "go test -race ./..."), ci: same, req: same,
			assert: assertNoCachedGate, caught: true},
		{name: "A12 a build with no -o, dropping a binary in the working directory",
			mk: sub("quality-bench:", "sneaky-build:\n\tgo build ./cmd/watchpost\n\nquality-bench:"), ci: same, req: same,
			assert: assertEveryBuildTrimmed, caught: true},
		{name: "A13 a build with -o=path and no trimpath",
			mk: sub("quality-bench:", "sneaky-build:\n\tgo build -o=$(DIST)/x ./cmd/watchpost\n\nquality-bench:"), ci: same, req: same,
			assert: assertEveryBuildTrimmed, caught: true},
		{name: "A14 a build whose package is a variable",
			mk: sub("quality-bench:", "sneaky-build:\n\tgo build -o $(DIST)/x $(PKG)\n\nquality-bench:"), ci: same, req: same,
			assert: assertEveryBuildTrimmed, caught: true},
		{name: "A15 a build missing $(TRIMPATH)",
			mk: sub("go build $(TRIMPATH) -ldflags", "go build -ldflags"), ci: same, req: same,
			assert: assertEveryBuildTrimmed, caught: true},
		{name: "A17 a second verify-gates rule appending a gate CI never runs",
			mk: sub("quality-bench:", "verify-gates: extra-gate\n\nextra-gate:\n\t@./scripts/extra.sh\n\nquality-bench:"), ci: same, req: same,
			assert: assertThreeListsAgree, caught: true},
		{name: "A-ok3 a correct build line",
			mk: same, ci: same, req: same, assert: assertEveryBuildTrimmed, caught: false},

		// ---- B. the CI model ---------------------------------------------------------
		{name: "A19 if: on the line after - run:",
			mk: same, ci: sub("      - run: make lint-a\n", "      - run: make lint-a\n        if: false\n"), req: same,
			assert: assertNoSilencedCIStep, caught: true},
		{name: "A20 - if: false as the step's FIRST key",
			mk: same, ci: sub("      - run: make lint-a\n", "      - if: false\n        run: make lint-a\n"), req: same,
			assert: assertNoSilencedCIStep, caught: true},
		{name: "A21 - continue-on-error: true as the step's first key",
			mk: same, ci: sub("      - run: make lint-a\n", "      - continue-on-error: true\n        run: make lint-a\n"), req: same,
			assert: assertNoSilencedCIStep, caught: true},
		{name: "A22 job-level if: false",
			mk: same, ci: sub("  verify:\n    needs: policy\n", "  verify:\n    needs: policy\n    if: false\n"), req: same,
			assert: assertNoSilencedCIStep, caught: true},
		{name: "A23 job-level continue-on-error: true",
			mk: same, ci: sub("  verify:\n    needs: policy\n", "  verify:\n    needs: policy\n    continue-on-error: true\n"), req: same,
			assert: assertNoSilencedCIStep, caught: true},
		{name: "A24 continue-on-error in expression form",
			mk: same, ci: sub("      - run: make lint-a\n", "      - run: make lint-a\n        continue-on-error: ${{ true }}\n"), req: same,
			assert: assertNoSilencedCIStep, caught: true},
		{name: "A25 - run: make X || true, no condition at all",
			mk: same, ci: sub("      - run: make lint-a\n", "      - run: make lint-a || true\n"), req: same,
			assert: assertNoSilencedCIStep, caught: true},
		{name: "A26 a required gate removed from ci.yml only",
			mk: same, ci: sub("      - run: make lint-a\n", ""), req: same,
			assert: assertThreeListsAgree, caught: true},
		{name: "A27 a required gate removed from all three lists",
			mk: func(s string) string {
				// EXPLICIT ANCHORS: `.PHONY` also names lint-a now, and a bare " lint-a "
				// would hit it first and leave verify's list intact.
				s = sub("verify-gates: race test-tags lint-a lint-b", "verify-gates: race test-tags lint-b")(s)
				s = sub("gate-controls alloc-budget build-check mutant-anchors mutant-check release-matrix install-test\nverify:",
					"gate-controls alloc-budget build-check mutant-anchors mutant-check release-matrix install-test\nverify:")(s)
				return sub("lint-a:\n\t@./scripts/lint-a.sh\n", "")(s)
			},
			ci:  sub("      - run: make lint-a\n", ""),
			req: sub("lint-a\n", ""),
			// A gate gone from every list is invisible to a list comparison —
			// which is why the gate-shaped check exists. Deleting the TARGET too
			// (as here) is the one shape nothing can see, and is recorded as such.
			assert: assertThreeListsAgree, caught: false},
		{name: "B-ok1 the two legitimately conditional steps, declared",
			mk: same, ci: same, req: same, assert: assertNoSilencedCIStep, caught: false},
		{name: "B-ok2 a non-required step carrying if:",
			mk: same, ci: sub("      - run: make release-matrix\n", "      - run: make release-matrix\n      - run: make optional-thing\n        if: false\n"), req: same,
			assert: assertNoSilencedCIStep, caught: false},

		// ---- D. the model's own silence (FR-11.3) ---------------------------------------
		{name: "A34 a Makefile with zero targets is COULD-NOT-RUN",
			mk: func(string) string { return "# nothing here\n" }, ci: same, req: same, assert: all, caught: true},
		{name: "A35 a workflow with zero jobs is COULD-NOT-RUN",
			mk: same, ci: func(string) string { return "name: ci\n" }, req: same, assert: all, caught: true},
		{name: "A36 an empty required list is COULD-NOT-RUN",
			mk: same, ci: same, req: func(string) string { return "# none\n" }, assert: all, caught: true},
	}

	// THE FLOOR IS THE COUNT. Nine rows moved to the executed table when the
	// Makefile half stopped being parsed; a floor left at the old number would
	// let three more vanish unnoticed.
	if len(specimens) < 26 {
		t.Fatalf("%d specimens; the parsed table holds 26 and this one has lost some", len(specimens))
	}
	for _, sp := range specimens { // bounded by the specimen table (P10-02)
		t.Run(sp.name, func(t *testing.T) {
			m := newBuildModel(sp.mk(baseMakefile), sp.ci(baseWorkflow), sp.req(baseRequired))
			fired, said := verdictOf(func(r reporter) { sp.assert(r, m) })
			switch {
			case sp.caught && !fired:
				t.Errorf("SURVIVED — the attack was not caught. A gate that this spelling defeats is a " +
					"gate that can be silenced in one line.")
			case !sp.caught && fired:
				t.Errorf("FALSE POSITIVE — a correct spelling was refused:\n  %s\n"+
					"A gate that flags correct code trains authors to delete it.", strings.Join(said, "\n  "))
			}
		})
	}
}
