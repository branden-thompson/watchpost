package main

// gateattacks_exec_test.go — sections E and F of 06_docs/gate-attack-list.md,
// executed.
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
// SECTION F RUNS THE REGISTRY over synthetic tables, so the CAUGHT direction of
// every registry rule is proved rather than asserted by prose (I2 from the
// second review).

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
	assert func(reporter, *oracleTree, []string)
	caught bool
}

func canFail(t reporter, o *oracleTree, req []string) {
	assertEveryRequiredGateCanFail(t, o, req)
}
func verifyFails(t reporter, o *oracleTree, req []string) { assertVerifyCanFail(t, o, req) }
func silenced(t reporter, o *oracleTree, req []string) {
	assertNoFileSilencesARequiredGate(t, o, req)
}
func controlsReached(t reporter, o *oracleTree, req []string) {
	assertEveryControlIsReached(t, o, req, map[string]string{"scripts/install-test.sh": "no control in the base fixture, by design"})
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
		{name: "E-ok every required gate goes red when its checks do", mk: same, assert: canFail, caught: false},
		{name: "E-ok verify itself goes red", mk: same, assert: verifyFails, caught: false},
		{name: "E-ok every control is reached", mk: same, assert: controlsReached, caught: false},
		{name: "H-ok every required gate is phony and present", mk: same, assert: silenced, caught: false},

		// ---- H. scratch-tree divergence: the green control -----------------------
		{name: "H1 a preflight red only in scratch, || exit 0 behind it",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@test -f go.mod || { echo run from the repo root; exit 1; }\n\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "H2 go version as the preflight",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@/usr/bin/false || { echo go toolchain missing; exit 1; }\n\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "H3 the dirty-tree refusal this project writes",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@git diff --quiet || { echo refusing on a dirty tree; exit 1; }\n\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "H4 a prerequisite red under green stubs",
			mk: sub("release-matrix:\n\tgo build $(TRIMPATH) -o dist/x-linux ./cmd/watchpost\n", "release-matrix:\n\t@ls dist/nothing\n"), assert: canFail, caught: true},
		{name: "H5 verify with a preflight and -@ on treelock",
			mk: sub("\t@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates",
				"\t@true\n\t-@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates"),
			assert: verifyFails, caught: true},

		// ---- J. discovery: the phony audit ----------------------------------------
		{name: "J1 a required gate not in .PHONY",
			mk: sub(" lint-a lint-b ", " lint-b "), assert: silenced, caught: true},
		{name: "J2 two gates consolidated into a pattern rule",
			mk: sub("test-tags:\n\tgo test -tags watchpost_debug -count=1 ./app\n", "%-tags:\n\tgo $* -tags watchpost_debug -count=1 ./app || exit 0\n"), assert: canFail, caught: true},
		{name: "J3 a rule deleted and .DEFAULT supplying a green one",
			mk: func(s string) string {
				s = sub("lint-a:\n\t@./scripts/lint-a.sh\n", "")(s)
				return ".DEFAULT:\n\t@echo \"$@: no rule, skipped\"\n" + s
			}, assert: canFail, caught: true},

		// ---- K. the control proof paints the control, not the checker ------------------
		{name: "K1 the carrier runs the checker and a neutered control",
			mk: func(s string) string {
				s = sub("\t@./scripts/lint-a.sh --self-test\n", "")(s)
				return sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@./scripts/lint-a.sh\n\t@./scripts/lint-a.sh --self-test || exit 0\n")(s)
			}, assert: controlsReached, caught: true},
		{name: "K2 sibling || exit 0 in one gate, flag || exit 0 in another",
			mk: func(s string) string {
				s = sub("\t@./scripts/lint-a.sh --self-test\n", "\t@./scripts/lint-a.sh --self-test || exit 0\n")(s)
				return sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@./scripts/lint-a.sh\n\t@./scripts/lint-a_test.sh || exit 0\n")(s)
			}, assert: controlsReached, caught: true},
		{name: "K3 a checker with no extension, control removed",
			mk: func(s string) string {
				s = sub("\t@./scripts/lint-a.sh --self-test\n", "")(s)
				return sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@./scripts/lint-a\n")(s)
			}, assert: controlsReached, caught: true},
		// `;` DISCARDS THE FIRST CONTROL'S STATUS — the line exits with the last
		// command's. The reviewer's R4c called this CAUGHT and was right; an earlier
		// draft of this table called it PASSES and was wrong. `&&` is the passing shape.
		{name: "K4 two controls joined by ; — the first's failure is lost",
			mk: sub("\t@./scripts/lint-a.sh --self-test\n\t@./scripts/lint-b.sh --self-test\n", "\t@./scripts/lint-a.sh --self-test; ./scripts/lint-b.sh --self-test\n"), assert: controlsReached, caught: true},
		{name: "K-ok two controls joined by && — both reached",
			mk: sub("\t@./scripts/lint-a.sh --self-test\n\t@./scripts/lint-b.sh --self-test\n", "\t@./scripts/lint-a.sh --self-test && ./scripts/lint-b.sh --self-test\n"), assert: controlsReached, caught: false},
		{name: "A7 a control made unreachable: true || x --self-test (executed)",
			mk: sub("\t@./scripts/lint-a.sh --self-test", "\t@true || ./scripts/lint-a.sh --self-test"), assert: controlsReached, caught: true},
		// ---- M. red for the wrong reason — per-key painting ------------------------
		{name: "M1 || exit 0 then go version — red-under-red from the wrong stub",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh || exit 0\n\t@go version\n"), assert: canFail, caught: true},
		{name: "M2 a prerequisite reds it, own line || exit 0",
			mk: sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: race\n\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "M3 a parse-time $(shell go version) preflight before || exit 0",
			mk: func(s string) string {
				s = sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@test \"$(GO_OK)\" = yes || exit 1\n\t@./scripts/lint-a.sh || exit 0\n")(s)
				return "GO_OK := $(shell go version >/dev/null 2>&1 && echo yes)\n" + s
			}, assert: canFail, caught: true},
		{name: "M4 the verdict of a gate with two go calls replaced by exit 0",
			mk: sub("mutant-check:\n\t@go test -tags mutants -count=1 ./06_docs/mutants/\n",
				"mutant-check:\n\t@go test -tags mutants -count=1 ./06_docs/mutants/; rc=$$?; go clean -testcache || exit 1; exit 0\n"),
			assert: canFail, caught: true},
		{name: "M7 verify: lint-a as a prerequisite, -@ on treelock",
			mk: sub("verify:\n\t@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates",
				"verify: lint-a\n\t-@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates"),
			assert: verifyFails, caught: true},
		{name: "M-ok a gate with two go calls whose verdict is kept",
			mk: sub("mutant-check:\n\t@go test -tags mutants -count=1 ./06_docs/mutants/\n",
				"mutant-check:\n\t@go test -tags mutants -count=1 ./06_docs/mutants/; rc=$$?; go clean -testcache || exit 1; exit $$rc\n"),
			assert: canFail, caught: false},

		// ---- N. the escapes --------------------------------------------------------------
		{name: "N1 rule deleted, .DEFAULT green — no row excuses it",
			mk: func(s string) string {
				s = sub("lint-a:\n\t@./scripts/lint-a.sh\n", "")(s)
				s = sub(" lint-a lint-b ", " lint-b ")(s)
				return ".DEFAULT:\n\t@echo \"$@: no rule here, skipped\"\n" + s
			}, assert: silenced, caught: true},
		{name: "N2 the recipe on a non-phony node the gate reaches",
			mk: sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: lint-a-run\nlint-a-run:\n\t@./scripts/lint-a.sh\n"), assert: silenced, caught: true},
		{name: "A10 a control deleted outright (executed)",
			mk: sub("\t@./scripts/lint-a.sh --self-test\n", ""), assert: controlsReached, caught: true},

		// ---- O. reach that text cannot see — decided by what the stubs recorded ----------
		{name: "O1 two go run calls to one tool: the self-test live, the check || true",
			mk:     sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run ./tools/lintb -self-test\n\t@go run ./tools/lintb || true\n"),
			assert: canFail, caught: true},
		{name: "O1-ok two go run calls to one tool, both live",
			mk:     sub("lint-b:\n\t@scripts/lint-b.sh\n", "lint-b:\n\t@go run ./tools/lintb -self-test\n\t@go run ./tools/lintb\n"),
			assert: canFail, caught: false},
		{name: "O2 a tool named through a variable, || true (the shipped p10 shape)",
			mk:     sub("lint-b:\n\t@scripts/lint-b.sh\n", "A2DH ?= a2dh\nlint-b:\n\t@$(A2DH) p10 check --json > out.json || true\n"),
			assert: canFail, caught: true},
		{name: "O2-ok a tool named through a variable, live",
			mk:     sub("lint-b:\n\t@scripts/lint-b.sh\n", "A2DH ?= a2dh\nlint-b:\n\t@$(A2DH) p10 check --json > out.json || { echo live findings; exit 1; }\n"),
			assert: canFail, caught: false},
		{name: "O3 a checker inside command substitution, || true",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@out=$$(./scripts/lint-a.sh 2>&1) || true\n"), assert: canFail, caught: true},
		{name: "O3-ok a checker inside command substitution, status read (the shipped fmt shape)",
			mk:     sub("\t@./scripts/lint-a.sh\n", "\t@out=$$(./scripts/lint-a.sh 2>&1); rc=$$?; test $$rc -eq 0 || { echo \"$$out\"; exit 1; }\n"),
			assert: canFail, caught: false},
		{name: "O4 a checker by absolute path, || true",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@$(CURDIR)/scripts/lint-a.sh || true\n"), assert: canFail, caught: true},
		{name: "O4 a checker behind sh -c, || true",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@sh -c './scripts/lint-a.sh' || true\n"), assert: canFail, caught: true},
		{name: "O4-ok a checker by absolute path, live",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@$(CURDIR)/scripts/lint-a.sh\n"), assert: canFail, caught: false},
		{name: "O5 $(MAKE) -s into a neutered target",
			mk:     sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@$(MAKE) -s lint-a-run\nlint-a-run:\n\t@./scripts/lint-a.sh || true\n"),
			assert: canFail, caught: true},
		{name: "O5-ok $(MAKE) -s into a live phony target",
			mk: func(s string) string {
				s = sub(" lint-a lint-b ", " lint-a lint-a-run lint-b ")(s)
				return sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@$(MAKE) -s lint-a-run\nlint-a-run:\n\t@./scripts/lint-a.sh\n")(s)
			}, assert: canFail, caught: false},
		{name: "O6 a pattern-rule prerequisite supplying a neutered recipe",
			mk:     sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: lint-a-run\n%-run:\n\t@./scripts/$*.sh || true\n"),
			assert: canFail, caught: true},
		{name: "O6 a pattern-rule prerequisite, live but silenced by a file of its name",
			mk:     sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: lint-a-run\n%-run:\n\t@./scripts/$*.sh\n"),
			assert: silenced, caught: true},
		{name: "O-ok an order-only directory prerequisite is not a silencer",
			mk:     sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a: | out\n\t@./scripts/lint-a.sh\nout:\n\tmkdir -p out\n"),
			assert: silenced, caught: false},
		{name: "O-ok a python checker by absolute path",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@python3 $(CURDIR)/scripts/lint-a.sh\n"), assert: canFail, caught: false},
		{name: "O8 python3 -m: not a scripts/ argument, refused rather than unseen",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@python3 -m lint_a || true\n"), assert: canFail, caught: true},
		{name: "O9 a diagnostic under || true is refused, not tolerated",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@go version || true\n\t@./scripts/lint-a.sh\n"), assert: canFail, caught: true},

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
			mk: sub("\t@./scripts/lint-a.sh\n", "\t-@./scripts/lint-a.sh\n"), assert: canFail, caught: true},
		{name: "E6 || exit 0",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh || exit 0\n"), assert: canFail, caught: true},
		{name: "E7 || echo skipped — any word that succeeds",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh || echo lint-a skipped\n"), assert: canFail, caught: true},
		{name: "E8 ; exit 0",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh; exit 0\n"), assert: canFail, caught: true},
		{name: "E9 | tee with no pipefail — the status is tee's",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh | tee /dev/null\n"), assert: canFail, caught: true},
		{name: "E10 the check replaced by an echo of its path",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@echo \"lint-a: disabled, see scripts/lint-a.sh\"\n"), assert: canFail, caught: true},
		{name: "E11 the target named through a variable, recipe neutered",
			mk: sub("lint-a:\n\t@./scripts/lint-a.sh\n", "GATE := lint-a\n$(GATE):\n\t-@./scripts/lint-a.sh\n"), assert: canFail, caught: true},
		{name: "E12 the gate defined in an included file, neutered there",
			mk:     sub("lint-a:\n\t@./scripts/lint-a.sh\n", "include gates.mk\n"),
			extra:  map[string]string{"gates.mk": "lint-a:\n\t@./scripts/lint-a.sh || true\n"},
			assert: canFail, caught: true},
		{name: "E13 ifeq with a @true branch active",
			mk: sub("lint-a:\n\t@./scripts/lint-a.sh\n", "ifeq (1,1)\nlint-a:\n\t@true\nelse\nlint-a:\n\t@./scripts/lint-a.sh\nendif\n"), assert: canFail, caught: true},
		{name: "E14 a control inside define/endef is not a control",
			mk: func(s string) string {
				s = sub("\t@./scripts/lint-a.sh --self-test\n", "")(s)
				return s + "\ndefine NOT_A_RULE\ngate-controls:\n\t@./scripts/lint-a.sh --self-test\nendef\n"
			}, assert: controlsReached, caught: true},
		{name: "E15 -@ on the verify entry point",
			mk: sub("\t@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates",
				"\t-@go run ./tools/treelock -name verify -- $(MAKE) --no-print-directory verify-gates"),
			assert: verifyFails, caught: true},
		// the round-one control attacks, now decided by execution
		{name: "A5 a control commented out (executed)",
			mk: sub("\t@./scripts/lint-a.sh --self-test", "\t# ./scripts/lint-a.sh --self-test"), assert: controlsReached, caught: true},
		{name: "A6 a control behind || true (executed)",
			mk: sub("\t@./scripts/lint-a.sh --self-test", "\t@./scripts/lint-a.sh --self-test || true"), assert: controlsReached, caught: true},
		{name: "A9 a control moved to a target nothing runs (executed)",
			mk: func(s string) string {
				s = sub("\t@./scripts/lint-a.sh --self-test\n", "")(s)
				return sub("journey: build\n", "journey: build\n\t@./scripts/lint-a.sh --self-test\n")(s)
			}, assert: controlsReached, caught: true},
		{name: "A18 a required gate emptied to @true (executed)",
			mk: sub("\t@./scripts/lint-a.sh\n", "\t@true\n"), assert: canFail, caught: true},
		{name: "A37 a column-0 comment inside a recipe does not end it (executed)",
			mk:     sub("lint-a:\n\t@./scripts/lint-a.sh\n", "lint-a:\n\t@echo starting\n# make ignores this\n\t@./scripts/lint-a.sh\n"),
			assert: canFail, caught: false},
	}
	if len(specimens) < 60 {
		t.Fatalf("%d execution specimens; sections E, H, J, K, M, N and O plus the re-executed round-one attacks", len(specimens))
	}
	req := parseRequired(baseRequired)
	for _, sp := range specimens { // bounded by the specimen table (P10-02)
		t.Run(sp.name, func(t *testing.T) {
			t.Parallel() // each specimen owns its scratch tree
			var o *oracleTree
			fired, said := verdictOf(func(r reporter) {
				o = newOracleTree(r, sp.mk(baseMakefile), baseRequired, baseScripts, sp.extra)
				sp.assert(r, o, req)
			})
			if o != nil {
				_ = os.RemoveAll(o.dir)
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
			fired, _ := verdictOf(func(r reporter) {
				o := newOracleTree(r, baseMakefile, baseRequired, baseScripts, nil)
				defer os.RemoveAll(o.dir)
				assertEveryRequiredGateCanFail(r, o, parseRequired(baseRequired))
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
		fired, _ := verdictOf(func(r reporter) {
			o := newOracleTree(r, sub("\t@./scripts/lint-a.sh\n", "\t@./scripts/lint-a.sh; exit 0\n")(baseMakefile), baseRequired, baseScripts, nil)
			defer os.RemoveAll(o.dir)
			assertEveryRequiredGateCanFail(r, o, parseRequired(baseRequired))
		})
		if !fired {
			t.Error("SURVIVED — with MAKEFILES setting .SHELLFLAGS := -ec in the parent environment, `x.sh; exit 0` read as a gate that can fail")
		}
	})
}

// ---- F. the registry, over synthetic tables ------------------------------------------

func TestTheRegistryAttackList(t *testing.T) {
	yes := func(*testing.T, string) bool { return true }
	// honest answers a known-absent subject and a known-satisfied one correctly.
	honestExists := func(_ *testing.T, s string) bool {
		return s != "absent-thing" && !strings.HasSuffix(s, "/no/such/subject")
	}
	honestNeeded := func(_ *testing.T, s string) bool { return s != "satisfied-thing" }
	good := func(rows map[string]string) *exemptionTable {
		return &exemptionTable{name: "specimen", rows: rows, exists: honestExists, stillNeeded: honestNeeded,
			satisfied: "satisfied-thing"}
	}
	realReason := "a real sentence explaining why this row is true today"

	specimens := []struct {
		name   string
		table  *exemptionTable
		caught bool
	}{
		{"C-ok1 a well-formed table with an honest row", good(map[string]string{"thing": realReason}), false},
		{"A28 an empty reason", good(map[string]string{"thing": ""}), true},
		{"A29 a shrug for a reason", good(map[string]string{"thing": "n/a"}), true},
		{"A29 a reason under the floor", good(map[string]string{"thing": "because"}), true},
		{"A30 a subject that no longer exists", good(map[string]string{"absent-thing": realReason}), true},
		{"A31 a subject the rule already accepts", good(map[string]string{"satisfied-thing": realReason}), true},
		{"F1 exists that cannot return false", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: yes, stillNeeded: honestNeeded, satisfied: "satisfied-thing"}, true},
		{"L1 exists honest only for a subject the table would have chosen", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: func(_ *testing.T, s string) bool { return s != "absent-thing" }, stillNeeded: honestNeeded, satisfied: "satisfied-thing"}, true},
		{"N3 exists that matches the nonce by SHAPE (a fixed prefix)", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: func(_ *testing.T, s string) bool { return !strings.HasPrefix(s, "__registry") }, stillNeeded: honestNeeded, satisfied: "satisfied-thing"}, true},
		{"F2 stillNeeded that cannot return false", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: honestExists, stillNeeded: yes, satisfied: "satisfied-thing"}, true},
		{"F1 a table with no satisfied subject declared", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: honestExists, stillNeeded: honestNeeded}, true},
		{"a table with no functions at all", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason}}, true},
		{"the no functions never called on a good table", good(map[string]string{"thing": realReason}), false},
	}
	for _, sp := range specimens { // bounded by the specimen table (P10-02)
		t.Run(sp.name, func(t *testing.T) {
			fired, said := verdictOf(func(r reporter) { assertRegistry(r, []*exemptionTable{sp.table}) })
			switch {
			case sp.caught && !fired:
				t.Errorf("SURVIVED — the registry accepted a row it must refuse")
			case !sp.caught && fired:
				t.Errorf("FALSE POSITIVE — a correct table was refused:\n  %s", strings.Join(said, "\n  "))
			}
		})
	}
}
