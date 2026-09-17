package main

// gateattacks_exec_test.go — sections E and F of 06_docs/gate-attack-list.md,
// executed.
//
// SECTION E RUNS MAKE. Each specimen plants one edit in a Makefile, stubs every
// checker red, and runs `make <gate>` in a scratch tree; CAUGHT means the oracle
// reports the gate cannot fail. No spelling is enumerated by the oracle — make
// and sh decide — so a specimen here is not "does the regex see this" but "does
// make's exit status say what it must". That is the difference between this
// file and gateattacks_test.go, and the reason this one exists.
//
// SECTION F RUNS THE REGISTRY over synthetic tables, so the CAUGHT direction of
// every registry rule is proved rather than asserted by prose (I2 from the
// second review).

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

type execSpecimen struct {
	name   string
	mk     func(string) string
	extra  map[string]string // an `include`d file, say
	assert func(reporter, *oracleTree, []string)
	caught bool
	// needsMake is the make version the attack first exists in. THE ORACLE'S
	// VERDICT IS ONLY AS GOOD AS THE MAKE IT RUNS UNDER: `.SHELLFLAGS` is a 3.82
	// feature, macOS ships 3.81, and an attack the local make cannot express is
	// not-applicable here — a distinct verdict (FR-11.3), skipped by name, and
	// run for real on CI's 4.x.
	needsMake string
}

// makeVersion is the local GNU make's major.minor, or "" if it cannot be read.
func makeVersion() string {
	out, err := exec.Command("make", "--version").Output()
	if err != nil {
		return ""
	}
	m := regexp.MustCompile(`GNU Make (\d+\.\d+)`).FindStringSubmatch(string(out))
	if m == nil {
		return ""
	}
	return m[1]
}

// olderThan says whether version a is older than b, on major.minor.
func olderThan(a, b string) bool {
	var am, an, bm, bn int
	fmt.Sscanf(a, "%d.%d", &am, &an)
	fmt.Sscanf(b, "%d.%d", &bm, &bn)
	return am < bm || (am == bm && an < bn)
}

func canFail(t reporter, o *oracleTree, req []string)   { assertEveryRequiredGateCanFail(t, o, req) }
func verifyFails(t reporter, o *oracleTree, _ []string) { assertVerifyCanFail(t, o) }
func controlsReached(t reporter, o *oracleTree, req []string) {
	o.checkers = o.checkersInvoked(req)
	assertEveryControlIsReached(t, o, req, map[string]string{})
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

		// ---- E. make semantics -------------------------------------------------
		{name: "E1 .IGNORE: at the top", mk: prepend(".IGNORE:"), assert: canFail, caught: true},
		{name: "E2 MAKEFLAGS += -i", mk: prepend("MAKEFLAGS += -i"), assert: canFail, caught: true},
		{name: "E3 SHELL := /usr/bin/true", mk: prepend("SHELL := /usr/bin/true"), assert: canFail, caught: true},
		{name: "E4 .SHELLFLAGS := -c true; #", mk: prepend(".SHELLFLAGS := -c true; #"), assert: canFail, caught: true, needsMake: "3.82"},
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
	if len(specimens) < 20 {
		t.Fatalf("%d execution specimens; section E has 16 rows plus the re-executed round-one attacks", len(specimens))
	}
	req := parseRequired(baseRequired)
	local := makeVersion()
	for _, sp := range specimens { // bounded by the specimen table (P10-02)
		t.Run(sp.name, func(t *testing.T) {
			if sp.needsMake != "" && (local == "" || olderThan(local, sp.needsMake)) {
				t.Skipf("NOT APPLICABLE on GNU Make %s — this attack needs %s; it runs for real on CI's make", local, sp.needsMake)
			}
			var o *oracleTree
			fired, said := verdictOf(func(r reporter) {
				o = newOracleTree(r, sp.mk(baseMakefile), baseRequired, sp.extra)
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

// ---- F. the registry, over synthetic tables ------------------------------------------

func TestTheRegistryAttackList(t *testing.T) {
	yes := func(*testing.T, string) bool { return true }
	no := func(*testing.T, string) bool { return false }
	// honest answers a known-absent subject and a known-satisfied one correctly.
	honestExists := func(_ *testing.T, s string) bool { return s != "absent-thing" }
	honestNeeded := func(_ *testing.T, s string) bool { return s != "satisfied-thing" }
	good := func(rows map[string]string) *exemptionTable {
		return &exemptionTable{name: "specimen", rows: rows, exists: honestExists, stillNeeded: honestNeeded,
			absent: "absent-thing", satisfied: "satisfied-thing"}
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
			exists: yes, stillNeeded: honestNeeded, absent: "absent-thing", satisfied: "satisfied-thing"}, true},
		{"F2 stillNeeded that cannot return false", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: honestExists, stillNeeded: yes, absent: "absent-thing", satisfied: "satisfied-thing"}, true},
		{"F1 a table with no control subjects declared", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: honestExists, stillNeeded: honestNeeded}, true},
		{"a table with no functions at all", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason}}, true},
		{"the no functions never called on a good table", good(map[string]string{"thing": realReason}), false},
	}
	_ = no
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
