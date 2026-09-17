package gateoracle

// specimens_test.go — 06_docs/gate-attack-list.md, executed against this package.
//
// SECTIONS E, H, J, K, M, N AND O RUN MAKE. Each specimen plants one edit in a
// Makefile and runs `make <gate>` in a scratch tree whose stubs record what ran;
// CAUGHT means the oracle reports the gate cannot fail, is silenced by a file, or
// does not reach its control. No spelling is enumerated by the oracle — make and
// sh decide what runs and the stubs say what ran — so a specimen here is not
// "does the regex see this" but "does make's exit status say what it must". That
// is the difference between this file and gateattacks_test.go, and the reason
// this one exists.
//

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type execSpecimen struct {
	name   string
	mk     func(string) string
	extra  map[string]string // an `include`d file, say
	assert func(Reporter, *Oracle, []string)
	caught bool
}

func canFail(t Reporter, o *Oracle, req []string) {
	AssertEveryRequiredGateCanFail(t, o, req)
}
func verifyFails(t Reporter, o *Oracle, req []string) {
	AssertVerifyCanFail(t, o, req, map[string]string{"release-matrix": "CI-only in the base fixture", "install-test": "CI-only in the base fixture"})
}
func silenced(t Reporter, o *Oracle, req []string) {
	AssertNoFileSilencesARequiredGate(t, o, req)
}
func controlsReached(t Reporter, o *Oracle, req []string) {
	AssertEveryControlIsReached(t, o, req, map[string]string{"scripts/install-test.sh": "no control in the base fixture, by design"})
}

// baseScripts are the checkers the base fixture's tree carries; a specimen that
// names another is red under green, which is the loud direction.
var baseScripts = []string{
	"scripts/lint-a.sh", "scripts/lint-a", "scripts/lint-a_test.sh", "scripts/lint-b.sh", "scripts/install-test.sh",
	"scripts/quality/mutant-anchors.sh", "scripts/quality/p10-unmatched_test.sh", "scripts/quality/validate-journey.expect",
}

func prepend(line string) func(string) string {
	return func(s string) string { return line + "\n" + s }
}

func TestTheGateAttackListExecuted(t *testing.T) {
	if _, err := os.Stat("/usr/bin/true"); err != nil {
		t.Skip("E3 needs /usr/bin/true") // the only skip here, and it names what it needs
	}
	specimens := []execSpecimen{
		// ---- the control: the base itself, every checker red ----------------
		{name: "E-ok every required gate goes red when its checks do", mk: Same, assert: canFail, caught: false},
		{name: "E-ok verify itself goes red", mk: Same, assert: verifyFails, caught: false},
		{name: "E-ok every control is reached", mk: Same, assert: controlsReached, caught: false},
		{name: "H-ok every required gate is phony and present", mk: Same, assert: silenced, caught: false},

		// ---- H. scratch-tree divergence: the green control -----------------------
		{name: "H1 a preflight red only in scratch, || exit 0 behind it",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@test -f go.mod || { echo run from the repo root; exit 1; }\n\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "H2 go version as the preflight",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@/usr/bin/false || { echo go toolchain missing; exit 1; }\n\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "H3 the dirty-tree refusal this project writes",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@git diff --quiet || { echo refusing on a dirty tree; exit 1; }\n\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "H4 a prerequisite red under green stubs",
			mk: Sub("release-matrix:\n\tgo build $(TRIMPATH) -o dist/x-linux ./cmd/watchpost\n", "release-matrix:\n\t@ls dist/nothing\n"), assert: canFail, caught: true},
		{name: "H5 verify with a preflight and -@ on treelock",
			mk: Sub("\t@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates",
				"\t@true\n\t-@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates"),
			assert: verifyFails, caught: true},

		// ---- J. discovery: the phony audit ----------------------------------------
		{name: "J1 a required gate not in .PHONY",
			mk: Sub(" lint-a lint-b ", " lint-b "), assert: silenced, caught: true},
		{name: "J2 two gates consolidated into a pattern rule",
			mk: Sub("test-tags:\n\tgo test -tags watchpost_debug -count=1 ./app\n", "%-tags:\n\tgo $* -tags watchpost_debug -count=1 ./app || exit 0\n"), assert: canFail, caught: true},
		{name: "J3 a rule deleted and .DEFAULT supplying a green one",
			mk: func(s string) string {
				s = Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "")(s)
				return ".DEFAULT:\n\t@echo \"$@: no rule, skipped\"\n" + s
			}, assert: canFail, caught: true},

		// ---- K. the control proof paints the control, not the checker ------------------
		{name: "K1 the carrier runs the checker and a neutered control",
			mk: func(s string) string {
				s = Sub("\t@./scripts/lint-a.sh --self-test\n", "")(s)
				return Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@./scripts/lint-a.sh\n\t@./scripts/lint-a.sh --self-test || exit 0\n")(s)
			}, assert: controlsReached, caught: true},
		{name: "K2 sibling || exit 0 in one gate, flag || exit 0 in another",
			mk: func(s string) string {
				s = Sub("\t@./scripts/lint-a.sh --self-test\n", "\t@./scripts/lint-a.sh --self-test || exit 0\n")(s)
				return Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@./scripts/lint-a.sh\n\t@./scripts/lint-a_test.sh || exit 0\n")(s)
			}, assert: controlsReached, caught: true},
		{name: "K3 a checker with no extension, control removed",
			mk: func(s string) string {
				s = Sub("\t@./scripts/lint-a.sh --self-test\n", "")(s)
				return Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@./scripts/lint-a\n")(s)
			}, assert: controlsReached, caught: true},
		// `;` DISCARDS THE FIRST CONTROL'S STATUS — the line exits with the last
		// command's. The reviewer's R4c called this CAUGHT and was right; an earlier
		// draft of this table called it PASSES and was wrong. `&&` is the passing shape.
		{name: "K4 two controls joined by ; — the first's failure is lost",
			mk: Sub("\t@./scripts/lint-a.sh --self-test\n\t@./scripts/lint-b.sh --self-test\n", "\t@./scripts/lint-a.sh --self-test; ./scripts/lint-b.sh --self-test\n"), assert: controlsReached, caught: true},
		{name: "K-ok two controls joined by && — both reached",
			mk: Sub("\t@./scripts/lint-a.sh --self-test\n\t@./scripts/lint-b.sh --self-test\n", "\t@./scripts/lint-a.sh --self-test && ./scripts/lint-b.sh --self-test\n"), assert: controlsReached, caught: false},
		{name: "A7 a control made unreachable: true || x --self-test (executed)",
			mk: Sub("\t@./scripts/lint-a.sh --self-test", "\t@true || ./scripts/lint-a.sh --self-test"), assert: controlsReached, caught: true},
		// ---- M. red for the wrong reason — per-key painting ------------------------
		{name: "M1 || exit 0 then go version — red-under-red from the wrong stub",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh || exit 0\n\t@go version\n"), assert: canFail, caught: true},
		{name: "M2 a prerequisite reds it, own line || exit 0",
			mk: Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: race\n\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "M3 a parse-time $(shell go version) preflight before || exit 0",
			mk: func(s string) string {
				s = Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@test \"$(GO_OK)\" = yes || exit 1\n\t@./scripts/lint-a.sh || exit 0\n")(s)
				return "GO_OK := $(shell go version >/dev/null 2>&1 && echo yes)\n" + s
			}, assert: canFail, caught: true},
		{name: "M4 the verdict of a gate with two go calls replaced by exit 0",
			mk: Sub("mutant-check:\n\t@go test -tags mutants -count=1 ./06_docs/mutants/\n",
				"mutant-check:\n\t@go test -tags mutants -count=1 ./06_docs/mutants/; rc=$$?; go clean -testcache || exit 1; exit 0\n"),
			assert: canFail, caught: true},
		{name: "M7 verify: lint-a as a prerequisite, -@ on treelock",
			mk: Sub("verify:\n\t@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates",
				"verify: lint-a\n\t-@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates"),
			assert: verifyFails, caught: true},
		{name: "M-ok a gate with two go calls whose verdict is kept",
			mk: Sub("mutant-check:\n\t@go test -tags mutants -count=1 ./06_docs/mutants/\n",
				"mutant-check:\n\t@go test -tags mutants -count=1 ./06_docs/mutants/; rc=$$?; go clean -testcache || exit 1; exit $$rc\n"),
			assert: canFail, caught: false},

		// ---- N. the escapes --------------------------------------------------------------
		{name: "N1 rule deleted, .DEFAULT green — no row excuses it",
			mk: func(s string) string {
				s = Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "")(s)
				s = Sub(" lint-a lint-b ", " lint-b ")(s)
				return ".DEFAULT:\n\t@echo \"$@: no rule here, skipped\"\n" + s
			}, assert: silenced, caught: true},
		{name: "N2 the recipe on a non-phony node the gate reaches",
			mk: Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: lint-a-run\nlint-a-run:\n\t@./scripts/lint-a.sh\n"), assert: silenced, caught: true},
		{name: "A10 a control deleted outright (executed)",
			mk: Sub("\t@./scripts/lint-a.sh --self-test\n", ""), assert: controlsReached, caught: true},

		// ---- O. reach that text cannot see — decided by what the stubs recorded ----------
		{name: "O1 two go run calls to one tool: the self-test live, the check || true",
			mk:     Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run ./tools/lintb -self-test\n\t@go run ./tools/lintb || true\n"),
			assert: canFail, caught: true},
		{name: "O1-ok two go run calls to one tool, both live",
			mk:     Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run ./tools/lintb -self-test\n\t@go run ./tools/lintb\n"),
			assert: canFail, caught: false},
		{name: "O2 a tool named through a variable, || true (the shipped p10 shape)",
			mk:     Sub("lint-b:\n\t@scripts/lint-b.sh\n", "A2DH ?= a2dh\nlint-b:\n\t@$(A2DH) p10 check --json > out.json || true\n"),
			assert: canFail, caught: true},
		{name: "O2-ok a tool named through a variable, live",
			mk:     Sub("lint-b:\n\t@scripts/lint-b.sh\n", "A2DH ?= a2dh\nlint-b:\n\t@$(A2DH) p10 check --json > out.json || { echo live findings; exit 1; }\n"),
			assert: canFail, caught: false},
		{name: "O3 a checker inside command substitution, || true",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@out=$$(./scripts/lint-a.sh 2>&1) || true\n"), assert: canFail, caught: true},
		{name: "O3-ok a checker inside command substitution, status read (the shipped fmt shape)",
			mk:     Sub("\t@./scripts/lint-a.sh\n", "\t@out=$$(./scripts/lint-a.sh 2>&1); rc=$$?; test $$rc -eq 0 || { echo \"$$out\"; exit 1; }\n"),
			assert: canFail, caught: false},
		{name: "O4 a checker by absolute path, || true",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@$(CURDIR)/scripts/lint-a.sh || true\n"), assert: canFail, caught: true},
		{name: "O4 a checker behind sh -c, || true",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@sh -c './scripts/lint-a.sh' || true\n"), assert: canFail, caught: true},
		{name: "O4-ok a checker by absolute path, live",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@$(CURDIR)/scripts/lint-a.sh\n"), assert: canFail, caught: false},
		{name: "O5 $(MAKE) -s into a neutered target",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@$(MAKE) -s lint-a-run\nlint-a-run:\n\t@./scripts/lint-a.sh || true\n"),
			assert: canFail, caught: true},
		{name: "O5-ok $(MAKE) -s into a live phony target",
			mk: func(s string) string {
				s = Sub(" lint-a lint-b ", " lint-a lint-a-run lint-b ")(s)
				return Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@$(MAKE) -s lint-a-run\nlint-a-run:\n\t@./scripts/lint-a.sh\n")(s)
			}, assert: canFail, caught: false},
		{name: "O6 a pattern-rule prerequisite supplying a neutered recipe",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: lint-a-run\n%-run:\n\t@./scripts/$*.sh || true\n"),
			assert: canFail, caught: true},
		{name: "O6 a pattern-rule prerequisite, live but silenced by a file of its name",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: lint-a-run\n%-run:\n\t@./scripts/$*.sh\n"),
			assert: silenced, caught: true},
		{name: "O-ok an order-only directory prerequisite is not a silencer",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: | out\n\t@./scripts/lint-a.sh\nout:\n\tmkdir -p out\n"),
			assert: silenced, caught: false},
		{name: "O-ok a python checker by absolute path",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@python3 $(CURDIR)/scripts/lint-a.sh\n"), assert: canFail, caught: false},
		{name: "O8 python3 -m: not a scripts/ argument, refused rather than unseen",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@python3 -m lint_a || true\n"), assert: canFail, caught: true},
		{name: "O9 a diagnostic under || true is refused, not tolerated",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@go version || true\n\t@./scripts/lint-a.sh\n"), assert: canFail, caught: true},

		// ---- P. the tree, the binary, the ordinal -------------------------------------
		{name: "P1 skip the checker on a clean tree",
			mk:     Sub("\t@./scripts/lint-a.sh\n", "\t@git diff --quiet HEAD -- '*.go' 2>/dev/null && { echo skipped; exit 0; }; ./scripts/lint-a.sh\n"),
			assert: canFail, caught: true},
		{name: "P1-ok a tree predicate that only narrates",
			mk:     Sub("\t@./scripts/lint-a.sh\n", "\t@git diff --quiet HEAD -- '*.go' || echo dirty; ./scripts/lint-a.sh\n"),
			assert: canFail, caught: false},
		{name: "P2 two go test calls, the first || echo",
			mk:     Sub("race:\n\tgo test -race -count=1 ./...\n", "race:\n\tgo test -race -count=1 ./... || echo flaky\n\tgo test -count=1 ./cmd/x -run TestVersion\n"),
			assert: canFail, caught: true},
		{name: "P2-ok two live go test calls",
			mk:     Sub("race:\n\tgo test -race -count=1 ./...\n", "race:\n\tgo test -race -count=1 ./...\n\tgo test -count=1 ./cmd/x -run TestVersion\n"),
			assert: canFail, caught: false},
		{name: "P3 a compiled checker, || true",
			mk:     Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go build -o out/lintb ./tools/lintb\n\t@out/lintb || true\n"),
			assert: canFail, caught: true},
		{name: "P3-ok a compiled checker, live",
			mk:     Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go build -o out/lintb ./tools/lintb\n\t@out/lintb\n"),
			assert: canFail, caught: false},
		{name: "P4 a non-phony gate depending on go.mod, which the tree has",
			mk: func(s string) string {
				s = Sub(" lint-a lint-b ", " lint-b ")(s)
				return Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "go.mod go.sum: ;\nlint-a: go.mod\n\t@./scripts/lint-a.sh\n")(s)
			}, assert: silenced, caught: true},
		{name: "P5 a non-phony node hidden from a dry-run walk",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "ifeq (,$(findstring n,$(MAKEFLAGS)))\nlint-a: lint-a-run\nendif\nlint-a-run:\n\t@./scripts/lint-a.sh\n"),
			assert: silenced, caught: true},
		{name: "P6 verify passes FAST=1 and lint-a skips under it",
			mk: func(s string) string {
				s = Sub("-- $(MAKE) --no-print-directory verify-gates", "-- $(MAKE) --no-print-directory verify-gates FAST=1")(s)
				return Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@test -n \"$(FAST)\" || ./scripts/lint-a.sh\n")(s)
			}, assert: verifyFails, caught: true},
		{name: "P7 cd scripts && ./x.sh || true",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@cd scripts && ./lint-a.sh || true\n"), assert: canFail, caught: true},
		{name: "P7-ok cd scripts && ./x.sh",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@cd scripts && ./lint-a.sh\n"), assert: canFail, caught: false},
		{name: "P8 go run ./tools/x with no control anywhere",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run ./tools/lintb\n"), assert: controlsReached, caught: true},
		{name: "P9 .ONESHELL:", mk: prepend(".ONESHELL:"), assert: canFail, caught: true},
		// THE REVIEWER'S "CORRECT" SPELLING IS THE F-152 DEFECT: `bad=$$(…); test -z`
		// throws away the substitution's status, so a gofmt that fails to RUN prints
		// nothing and passes. The oracle refused it and was right; `&&` keeps it.
		{name: "P10 find | xargs gofmt, the status thrown away by ;",
			mk:     Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@bad=$$(find . -name '*.go' | xargs gofmt -l); test -z \"$$bad\"\n"),
			assert: canFail, caught: true},
		{name: "P10-ok find | xargs gofmt on a tree with Go files, status kept",
			mk:     Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@bad=$$(find . -name '*.go' | xargs gofmt -l) && test -z \"$$bad\"\n"),
			assert: canFail, caught: false},
		{name: "P11 one script twice, the first || true",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh --check || true; ./scripts/lint-a.sh --help\n"), assert: canFail, caught: true},

		// ---- Q. the tree as CI has it; the joints ----------------------------------------
		{name: "Q1 skip when HEAD is detached (CI is)",
			mk:     Sub("\t@./scripts/lint-a.sh\n", "\t@test \"$$(git rev-parse --abbrev-ref HEAD)\" != HEAD || { echo detached, skipping; exit 0; }; ./scripts/lint-a.sh\n"),
			assert: canFail, caught: true},
		{name: "Q2 skip when there are no tags (a depth-1 checkout has none)",
			mk:     Sub("\t@./scripts/lint-a.sh\n", "\t@git describe --tags >/dev/null 2>&1 || exit 0; ./scripts/lint-a.sh\n"),
			assert: canFail, caught: true},
		{name: "Q3 skip when a git-ignored file is absent (CI never has it)",
			mk:     Sub("\t@./scripts/lint-a.sh\n", "\t@test -f AGENTS.md || exit 0; ./scripts/lint-a.sh\n"),
			assert: canFail, caught: true},
		{name: "Q4 verify passes FAST=1 and race skips ONE of two go tests under it",
			mk: func(s string) string {
				s = Sub("-- $(MAKE) --no-print-directory verify-gates", "-- $(MAKE) --no-print-directory verify-gates FAST=1")(s)
				return Sub("race:\n\tgo test -race -count=1 ./...\n", "race:\n\tgo test -race -count=1 ./...\n\t@test -n \"$(FAST)\" || go test -count=1 ./cmd/x -run TestVersion\n")(s)
			}, assert: verifyFails, caught: true},
		{name: "Q5 two scripts whose paths collide under tr / _",
			mk:     Sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/quality_lint.sh || true\n\t@./scripts/quality/lint.sh\n"),
			extra:  map[string]string{"scripts/quality_lint.sh": "#!/usr/bin/env sh\n", "scripts/quality/lint.sh": "#!/usr/bin/env sh\n"},
			assert: canFail, caught: true},
		{name: "Q6 go run -tags foo ./tools/x with no control",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run -tags foo ./tools/lintb\n"), assert: controlsReached, caught: true},
		{name: "Q6 go run <module>/tools/x with no control",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run fixture/tools/lintb\n"), assert: controlsReached, caught: true},
		{name: "Q6 a built checker with no control",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go build -o out/lintb ./tools/lintb\n\t@out/lintb\n"), assert: controlsReached, caught: true},
		{name: "Q6-ok go run -tags foo ./tools/x, controlled",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run -tags foo ./tools/lintb -self-test\n\t@go run -tags foo ./tools/lintb\n"), assert: controlsReached, caught: false},
		{name: "Q7 a script with a #!/usr/bin/env python3.12 shebang, || true",
			mk:     Sub("\t@./scripts/lint-a.sh\n", "\t@go version\n\t@./scripts/check.py || true\n"),
			extra:  map[string]string{"scripts/check.py": "#!/usr/bin/env python3.12\n"},
			assert: canFail, caught: true},
		{name: "Q8 a $(MAKE) hop with its output hidden, recipe on a non-phony node",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@$(MAKE) --no-print-directory lint-a-run >/dev/null\nlint-a-run:\n\t@./scripts/lint-a.sh\n"),
			assert: silenced, caught: true},
		{name: "Q8 a $(MAKE) hop under MAKEFLAGS=, recipe on a non-phony node",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@MAKEFLAGS= $(MAKE) --no-print-directory lint-a-run\nlint-a-run:\n\t@./scripts/lint-a.sh\n"),
			assert: silenced, caught: true},
		{name: "Q9 go build -o out/ (directory form), the binary || true",
			mk:     Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@mkdir -p out && go build -o out/ ./tools/lintb\n\t@out/lintb || true\n"),
			assert: canFail, caught: true},
		{name: "Q11-ok bash -c with pipefail around a live checker",
			mk:     Sub("\t@./scripts/lint-a.sh\n", "\t@bash -c 'set -o pipefail; ./scripts/lint-a.sh 2>&1 | tee out.log'\n"),
			assert: canFail, caught: false},
		{name: "Q13 two go tests in parallel, the first || true",
			mk:     Sub("race:\n\tgo test -race -count=1 ./...\n", "race:\n\t{ go test -race -count=1 ./... || true; } & go test -count=1 ./cmd/x; wait\n"),
			assert: canFail, caught: true},

		// ---- R. drift, round eight ------------------------------------------------------
		{name: "R1 bash -ec with the script inside the string, || echo — flags before -c",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@bash -ec './scripts/lint-a.sh || echo see lint-a_test.sh'\n"), assert: canFail, caught: true},
		{name: "R1-ok sh -ec running a live script",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@sh -ec './scripts/lint-a.sh'\n"), assert: canFail, caught: false},
		{name: "R2 go build -o=path then the binary || true",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go build -o=out/lintb ./tools/lintb\n\t@out/lintb || true\n"), assert: canFail, caught: true},
		{name: "R2 go test -c then the test binary || true",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go test -c -o out/lintb.test ./tools/lintb\n\t@out/lintb.test || true\n"), assert: canFail, caught: true},
		{name: "R3 a dot-named hidden node, recipe on it",
			mk: Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: .lint-a-run\n.lint-a-run:\n\t@./scripts/lint-a.sh\n"), assert: silenced, caught: true},
		{name: "R4 go run tools/x/main.go as the only check, no control",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run tools/lintb/main.go\n"), assert: controlsReached, caught: true},
		{name: "R4 go run ./cmd/x as the only check, no control",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run ./cmd/lintb\n"), assert: controlsReached, caught: true},
		{name: "R5-ok a parse-time $(shell go env) at the top of the Makefile",
			mk: prepend("GOBIN := $(shell go env GOPATH)/bin"), assert: canFail, caught: false},
		{name: "R6-ok a control spelled with a trailing slash",
			mk: Sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run ./tools/lintb/ -self-test\n\t@go run ./tools/lintb\n"), assert: controlsReached, caught: false},
		{name: "R8 a stamp the recipe refuses masks a hidden non-phony node",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "stamp:\n\t@touch stamp\nlint-a:\n\t@test ! -e stamp || exit 1\n\t@$(MAKE) --no-print-directory lint-a-run >/dev/null\nlint-a-run:\n\t@./scripts/lint-a.sh\n"),
			assert: silenced, caught: true},

		// ---- E. make semantics -------------------------------------------------
		{name: "E1 .IGNORE: at the top", mk: prepend(".IGNORE:"), assert: canFail, caught: true},
		{name: "E2 MAKEFLAGS += -i", mk: prepend("MAKEFLAGS += -i"), assert: canFail, caught: true},
		{name: "E3 SHELL := /usr/bin/true", mk: prepend("SHELL := /usr/bin/true"), assert: canFail, caught: true},
		// THE REVIEWER'S SPELLING WAS INERT. `.SHELLFLAGS := -c true; #` leaves the
		// recipe red on 3.81 (no .SHELLFLAGS) AND on 4.4.1 — probed directly. The
		// class is real: `-c :`, `-c "true ;"` and `-c true \#` all exit 0. A
		// specimen must be an attack that works, or it proves nothing.
		{name: "E4 .SHELLFLAGS := -c : (a spelling that works)", mk: prepend(".SHELLFLAGS := -c :"), assert: canFail, caught: true},
		{name: "E5 a - prefix on the check line",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t-@./scripts/lint-a.sh\n"), assert: canFail, caught: true},
		{name: "E6 || exit 0",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "E7 || echo skipped — any word that succeeds",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh || echo lint-a skipped\n"), assert: canFail, caught: true},
		{name: "E8 ; exit 0",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh; exit 0\n"), assert: canFail, caught: true},
		{name: "E9 | tee with no pipefail — the status is tee's",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh | tee /dev/null\n"), assert: canFail, caught: true},
		{name: "E10 the check replaced by an echo of its path",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@echo \"lint-a: disabled, see scripts/lint-a.sh\"\n"), assert: canFail, caught: true},
		{name: "E11 the target named through a variable, recipe neutered",
			mk: Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "GATE := lint-a\n$(GATE):\n\t-@./scripts/lint-a.sh\n"), assert: canFail, caught: true},
		{name: "E12 the gate defined in an included file, neutered there",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "include gates.mk\n"),
			extra:  map[string]string{"gates.mk": "lint-a:\n\t@./scripts/lint-a.sh || true\n"},
			assert: canFail, caught: true},
		{name: "E13 ifeq with a @true branch active",
			mk: Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "ifeq (1,1)\nlint-a:\n\t@true\nelse\nlint-a:\n\t@./scripts/lint-a.sh\nendif\n"), assert: canFail, caught: true},
		{name: "E14 a control inside define/endef is not a control",
			mk: func(s string) string {
				s = Sub("\t@./scripts/lint-a.sh --self-test\n", "")(s)
				return s + "\ndefine NOT_A_RULE\ngate-controls:\n\t@./scripts/lint-a.sh --self-test\nendef\n"
			}, assert: controlsReached, caught: true},
		{name: "E15 -@ on the verify entry point",
			mk: Sub("\t@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates",
				"\t-@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates"),
			assert: verifyFails, caught: true},
		// the round-one control attacks, now decided by execution
		{name: "A5 a control commented out (executed)",
			mk: Sub("\t@./scripts/lint-a.sh --self-test", "\t# ./scripts/lint-a.sh --self-test"), assert: controlsReached, caught: true},
		{name: "A6 a control behind || true (executed)",
			mk: Sub("\t@./scripts/lint-a.sh --self-test", "\t@./scripts/lint-a.sh --self-test || true"), assert: controlsReached, caught: true},
		{name: "A9 a control moved to a target nothing runs (executed)",
			mk: func(s string) string {
				s = Sub("\t@./scripts/lint-a.sh --self-test\n", "")(s)
				return Sub("journey: build\n", "journey: build\n\t@./scripts/lint-a.sh --self-test\n")(s)
			}, assert: controlsReached, caught: true},
		{name: "A18 a required gate emptied to @true (executed)",
			mk: Sub("\t@./scripts/lint-a.sh\n", "\t@true\n"), assert: canFail, caught: true},
		{name: "A37 a column-0 comment inside a recipe does not end it (executed)",
			mk:     Sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@echo starting\n# make ignores this\n\t@./scripts/lint-a.sh\n"),
			assert: canFail, caught: false},
	}
	if len(specimens) < 100 {
		t.Fatalf("%d execution specimens; sections E, H, J, K, M, N, O, P, Q and R plus the re-executed round-one attacks", len(specimens))
	}
	req := ParseRequired(BaseRequired)
	for _, sp := range specimens { // bounded by the specimen table (P10-02)
		t.Run(sp.name, func(t *testing.T) {
			t.Parallel() // each specimen owns its scratch tree
			var done func()
			mk := sp.mk(BaseMakefile)
			if anchor := MissingAnchor(mk); anchor != "" {
				t.Fatalf("the specimen's anchor is not in the base fixture: %q", anchor)
			}
			fired, said := VerdictOf(func(r Reporter) {
				var o *Oracle
				o, done = SpecimenOracle(r, mk, BaseRequired, baseScripts, sp.extra)
				sp.assert(r, o, req)
			})
			if done != nil {
				done()
			}
			switch {
			case sp.caught && !fired:
				t.Errorf("SURVIVED — the oracle did not object to this spelling, so a gate it silences would pass.")
			case !sp.caught && fired:
				t.Errorf("FALSE POSITIVE — a correct Makefile was refused:\n  %s", strings.Join(said, "\n  "))
			}
		})
	}
}

// N4, O7: A PARENT MAKE'S ENVIRONMENT DOES NOT REACH THE ORACLE. The oracle runs
// inside `make race`. With MAKEFLAGS=i or GNUMAKEFLAGS=-i every child make
// ignores errors and every gate reads as "cannot fail" — the loud direction. With
// MAKEFILES naming a file that sets `.SHELLFLAGS := -ec`, every `;` discard turns
// red and the oracle certifies a neutered gate — the QUIET direction, which is
// why E8 is run under it and must still be CAUGHT.
func TestAParentMakesEnvironmentDoesNotReachTheOracle(t *testing.T) {
	for _, flag := range []string{"MAKEFLAGS", "GNUMAKEFLAGS"} { // bounded by the two flag variables (P10-02)
		t.Run(flag+"=i", func(t *testing.T) {
			t.Setenv(flag, "-i")
			fired, _ := VerdictOf(func(r Reporter) {
				o, done := SpecimenOracle(r, BaseMakefile, BaseRequired, baseScripts, nil)
				defer done()
				AssertEveryRequiredGateCanFail(r, o, ParseRequired(BaseRequired))
			})
			if fired {
				t.Errorf("with %s=-i in the parent environment the oracle reported gates that cannot fail — the flag reached the child make", flag)
			}
		})
	}
	t.Run("MAKEFILES sets .SHELLFLAGS := -ec", func(t *testing.T) {
		pre := filepath.Join(t.TempDir(), "strict.mk")
		if err := os.WriteFile(pre, []byte(".SHELLFLAGS := -ec\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("MAKEFILES", pre)
		fired, _ := VerdictOf(func(r Reporter) {
			o, done := SpecimenOracle(r, Sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh; exit 0\n")(BaseMakefile), BaseRequired, baseScripts, nil)
			defer done()
			AssertEveryRequiredGateCanFail(r, o, ParseRequired(BaseRequired))
		})
		if !fired {
			t.Error("SURVIVED — with MAKEFILES setting .SHELLFLAGS := -ec in the parent environment, `x.sh; exit 0` read as a gate that can fail")
		}
	})
}
