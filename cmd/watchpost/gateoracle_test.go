package main

// gateoracle_test.go — the Makefile half of the gate layer, EXECUTED rather than
// parsed.
//
// WHY EXECUTION. A blind reviewer defeated the parsed model six ways in one
// sitting — `.IGNORE:`, `MAKEFLAGS += -i`, `SHELL := true`, `|| exit 0`, a
// checker's path inside an echo, a target named through a variable — and named
// the cause: a semantic question ("can this gate fail?") decided by pattern on a
// fragment, in a language whose global constructs the parser never reads. The
// attack list enumerated spellings; the parser enumerated the same spellings
// back. No number of regexes closes that, because make and sh have more
// spellings than any list.
//
// SO MAKE AND SH ARE THE ORACLE. Every project checker is replaced by a stub that
// exits non-zero, every toolchain command on PATH likewise, and `make <gate>` is
// RUN. If make exits zero, the gate cannot fail — whatever the spelling, because
// the shell decided, not a regex. A control is proved live the same way: stub
// ONLY that checker red and run the gate that carries the control; if the gate
// stays green, the control is not reached. Discovery comes from `make -pn`,
// make's own expanded database, so a target named through a variable, an
// include or a conditional is found because make found it.
//
// IT RUNS IN A SCRATCH TREE, NEVER THE REPOSITORY. The Makefile text is written
// to a temp directory beside stub scripts and a stub bin/; nothing here touches
// the tree it is judging, and the specimen table hands it synthetic Makefiles by
// the same path.

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// toolchain are the commands a gate recipe reaches on PATH that must be red in
// the oracle: if any of them succeeded for real, a gate could pass by running
// the real tool against an empty scratch tree.
var toolchain = []string{"go", "gofmt", "a2dh", "python3", "expect", "golangci-lint", "govulncheck"}

// oracleTree is a scratch directory holding one Makefile, the stubs it reaches,
// and a required-gates list.
type oracleTree struct {
	dir string
	// checkers is every scripts/… path the Makefile's recipes name, as make's
	// database expanded them.
	checkers []string
	// siblings is every `<checker>_test.sh` the recipes name — a control in its
	// own script rather than a flag, stubbed like a checker.
	siblings []string
	// db is make's own view of the Makefile: every real target and its recipe.
	db map[string]*target
}

// newOracleTree lays out the scratch tree and asks make for its database.
//
// STUBS ARE WRITTEN FOR EVERY SCRIPT THE DATABASE NAMES, red by default. A script
// the Makefile references but the tree lacks would make `make` fail for the
// wrong reason — "not found" rather than "the check failed" — and both exit
// non-zero, which is the only fact the oracle reads. That is acceptable: a
// missing checker is not a gate that cannot fail.
func newOracleTree(t reporter, makefile, requiredList string, extra map[string]string) *oracleTree {
	t.Helper()
	dir, err := os.MkdirTemp("", "gate-oracle-*")
	if err != nil {
		t.Fatalf("COULD NOT RUN — no scratch directory: %v", err)
	}
	must := func(e error) {
		if e != nil {
			t.Fatalf("COULD NOT RUN — laying out the oracle tree: %v", e)
		}
	}
	must(os.WriteFile(filepath.Join(dir, "Makefile"), []byte(makefile), 0o644))
	must(os.MkdirAll(filepath.Join(dir, "06_docs"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "06_docs", "required-gates.txt"), []byte(requiredList), 0o644))
	for name, body := range extra { // bounded by the extra-file map (P10-02): an `include`d makefile, say
		must(os.MkdirAll(filepath.Join(dir, filepath.Dir(name)), 0o755))
		must(os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	}
	must(os.MkdirAll(filepath.Join(dir, "bin"), 0o755))
	for _, tool := range toolchain { // bounded by the toolchain list (P10-02)
		must(writeStub(filepath.Join(dir, "bin", tool), 3))
	}
	o := &oracleTree{dir: dir}
	o.db = o.database(t)
	sibling := regexp.MustCompile(`(scripts/[A-Za-z0-9_/.-]+_test\.sh)`)
	for _, tg := range o.db { // bounded by the database (P10-02)
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			for _, k := range c.checkers() { // bounded by the command (P10-02)
				if strings.HasPrefix(k, "scripts/") && !contains(o.checkers, k) {
					o.checkers = append(o.checkers, k)
				}
			}
			for _, m := range sibling.FindAllStringSubmatch(c.text, -1) { // a `_test.sh` is a CONTROL, stubbed too
				if !contains(o.siblings, m[1]) {
					o.siblings = append(o.siblings, m[1])
				}
			}
		}
	}
	for _, k := range append(append([]string{}, o.checkers...), o.siblings...) { // bounded by both lists (P10-02)
		must(os.MkdirAll(filepath.Join(dir, filepath.Dir(k)), 0o755))
		must(writeStub(filepath.Join(dir, k), 3))
	}
	return o
}

func writeStub(path string, exit int) error {
	return os.WriteFile(path, []byte("#!/bin/sh\nexit "+string(rune('0'+exit))+"\n"), 0o755)
}

// paint sets one checker's stub to the given exit status.
func (o *oracleTree) paint(t reporter, checker string, exit int) {
	t.Helper()
	if err := writeStub(filepath.Join(o.dir, checker), exit); err != nil {
		t.Fatalf("COULD NOT RUN — repainting %s: %v", checker, err)
	}
}

// paintAll sets every checker stub to the given exit status.
func (o *oracleTree) paintAll(t reporter, exit int) {
	t.Helper()
	for _, k := range append(append([]string{}, o.checkers...), o.siblings...) { // bounded by both lists (P10-02)
		o.paint(t, k, exit)
	}
}

// run executes `make <gate>` in the scratch tree with the red PATH in front and
// returns make's exit status. -1 means make itself could not be started.
func (o *oracleTree) run(gate string) int {
	cmd := exec.Command("make", "-s", gate)
	cmd.Dir = o.dir
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(o.dir, "bin")+":"+os.Getenv("PATH"))
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			return ex.ExitCode()
		}
		return -1
	}
	return 0
}

// database is `make -pn`'s view of the Makefile: every real target with its
// recipe AFTER expansion. A target named through `$(VAR):`, brought in by
// `include`, or chosen by `ifeq` appears here as make will run it.
//
// THE PARSE IS OF MAKE'S OUTPUT, NOT OF THE MAKEFILE. Make prints a target as
// `name:` followed by `#  …` notes and tab-indented recipe lines; a name that is
// not a target is preceded by `# Not a target:`. That format is make's own and is
// the thing this replaces a hand parser with.
func (o *oracleTree) database(t reporter) map[string]*target {
	t.Helper()
	cmd := exec.Command("make", "-pn", "-f", "Makefile")
	cmd.Dir = o.dir
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(o.dir, "bin")+":"+os.Getenv("PATH"))
	out, _ := cmd.Output() // -pn exits non-zero on a missing default goal; the database is still printed
	if len(out) == 0 {
		t.Fatalf("COULD NOT RUN — `make -pn` printed nothing; make is missing or the Makefile does not parse")
	}
	targets := map[string]*target{}
	targetLine := regexp.MustCompile(`^([A-Za-z0-9_./-][^:=#\s]*):(?:[^=]|$)`)
	var cur *target
	var notATarget bool
	for _, line := range strings.Split(string(out), "\n") { // bounded by the database (P10-02)
		switch {
		case line == "# Not a target:":
			notATarget = true
			cur = nil
		case strings.HasPrefix(line, "\t"):
			if cur != nil {
				body := strings.TrimSpace(line)
				if body != "" && !strings.HasPrefix(body, "#") {
					cur.cmds = append(cur.cmds, newCommand(body))
				}
			}
		case strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "":
			// a note about the current target, or a blank between them
		default:
			mm := targetLine.FindStringSubmatch(line)
			cur = nil
			if mm == nil {
				notATarget = false
				continue
			}
			if notATarget || strings.HasPrefix(mm[1], ".") {
				notATarget = false
				continue // a non-target, or a special target like .PHONY
			}
			cur = &target{name: mm[1]}
			targets[mm[1]] = cur
		}
	}
	return targets
}

// ---- the executed assertions ------------------------------------------------------

// EVERY REQUIRED GATE GOES RED WHEN ITS CHECKS DO.
//
// This is the whole of "a gate can fail", decided by make. With every checker
// and every toolchain command stubbed red, `make <gate>` must exit non-zero. A
// zero exit means the status was discarded somewhere — `.IGNORE:`, `-`,
// `|| exit 0`, `SHELL := true`, a path inside an echo — and it does not matter
// which, because the shell has already answered.
func assertEveryRequiredGateCanFail(t reporter, o *oracleTree, required []string) {
	t.Helper()
	o.paintAll(t, 3)
	var checked int
	for _, g := range required { // bounded by the gate list (P10-02)
		if o.db[g] == nil {
			continue // not a make target — a CI-only step; the list check owns that
		}
		checked++
		if code := o.run(g); code == 0 {
			t.Errorf("%s is a REQUIRED gate and `make %s` exits 0 with every check it runs stubbed RED.\n"+
				"The gate cannot fail. Somewhere its status is discarded — a `-` prefix, `|| true`, "+
				"`.IGNORE:`, a SHELL override, a check that is only echoed — and make has already "+
				"said so; this is not a spelling to argue with.", g, g)
		}
	}
	if checked < 5 {
		t.Fatalf("COULD NOT RUN — only %d required gate(s) are make targets in this tree", checked)
	}
}

// EVERY CONTROL IS REACHED.
//
// For each checker a required gate invokes, stub every checker GREEN except that
// one, and run every required gate whose recipe names it with a self-test flag.
// If none goes red, the control does not run — it is commented out, behind
// `|| true`, inside a `define`, or in a target nothing invokes — and again the
// shell decided.
func assertEveryControlIsReached(t reporter, o *oracleTree, required []string, uncontrolledRows map[string]string) {
	t.Helper()
	var checked int
	for _, k := range o.checkers { // bounded by the checker list (P10-02)
		if _, declared := uncontrolledRows[k]; declared {
			continue
		}
		carriers, control := o.controlOf(k, required)
		checked++
		if len(carriers) == 0 {
			t.Errorf("%s is invoked by a required gate and no required gate's recipe runs its control — "+
				"neither `%s --self-test` nor a sibling `%s_test.sh`.\n"+
				"A checker with no control is trusted without evidence. Add the control to gate-controls, "+
				"or add a row to `uncontrolled` saying why.", k, k, strings.TrimSuffix(k, ".sh"))
			continue
		}
		// PAINT THE CONTROL ITSELF RED, not the checker. A `--self-test` flag runs
		// the checker, so painting the checker reaches it; a sibling `_test.sh` is
		// its own script, so that is what must go red for the carrier to notice.
		o.paintAll(t, 0)
		o.paint(t, control, 3)
		var reached bool
		for _, g := range carriers { // bounded by the carrier list (P10-02)
			if o.run(g) != 0 {
				reached = true
				break
			}
		}
		if !reached {
			t.Errorf("%s's self-test is named in %v and NOT REACHED: with only that checker red, every "+
				"carrier stays green.\nThe control is commented out, behind `|| true`, inside a define, or "+
				"otherwise never runs — make has said so.", k, carriers)
		}
	}
	o.paintAll(t, 3)
	if checked < 3 {
		t.Fatalf("COULD NOT RUN — only %d checker(s) invoked by required gates", checked)
	}
}

// controlOf is every required gate whose recipe text names a control for `k`,
// and the path to paint red to exercise it: `k` itself for a `--self-test` flag,
// or the sibling `k_test.sh`. Text finds CANDIDATES only; execution decides.
func (o *oracleTree) controlOf(k string, required []string) (carriers []string, control string) {
	sib := strings.TrimSuffix(k, ".sh") + "_test.sh"
	for _, g := range required { // bounded by the gate list (P10-02)
		tg := o.db[g]
		if tg == nil {
			continue
		}
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			switch {
			case strings.Contains(c.text, k) && strings.Contains(c.text, "-self-test"):
				carriers, control = append(carriers, g), k
			case strings.Contains(c.text, sib):
				carriers, control = append(carriers, g), sib
			default:
				continue
			}
			break
		}
	}
	return carriers, control
}

// `make verify` ITSELF GOES RED ON A RED GATE (E15).
//
// verify takes the tree lock and delegates; a `-` on that one line makes the
// entry point exit 0 over any failure below it, and release.yml reads that
// exit. So the entry point is executed too.
func assertVerifyCanFail(t reporter, o *oracleTree) {
	t.Helper()
	if o.db["verify"] == nil {
		t.Fatalf("COULD NOT RUN — no verify target")
	}
	o.paintAll(t, 3)
	if code := o.run("verify"); code == 0 {
		t.Errorf("`make verify` exits 0 with every gate below it red. The entry point discards the " +
			"failure — release.yml reads this exit status, so a red release would ship.")
	}
}

// checkersInvoked is every scripts/… checker any REQUIRED gate's expanded recipe
// names — from make's database, so a target hidden behind a variable or an
// include is included.
func (o *oracleTree) checkersInvoked(required []string) []string {
	var out []string
	for _, g := range required { // bounded by the gate list (P10-02)
		tg := o.db[g]
		if tg == nil {
			continue
		}
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			for _, k := range c.checkers() { // bounded by the command (P10-02)
				if strings.HasPrefix(k, "scripts/") && !contains(out, k) {
					out = append(out, k)
				}
			}
		}
	}
	return out
}

// ---- the production gates ------------------------------------------------------------

func realOracle(t *testing.T) (*oracleTree, []string) {
	t.Helper()
	mk := read(t, "../../Makefile")
	req := read(t, "../../06_docs/required-gates.txt")
	o := newOracleTree(t, mk, req, nil)
	t.Cleanup(func() { _ = os.RemoveAll(o.dir) })
	return o, parseRequired(req)
}

func TestEveryRequiredGateCanFail(t *testing.T) {
	o, required := realOracle(t)
	assertEveryRequiredGateCanFail(t, o, required)
}

func TestVerifyCanFail(t *testing.T) {
	o, _ := realOracle(t)
	assertVerifyCanFail(t, o)
}

func TestEveryControlIsReached(t *testing.T) {
	o, required := realOracle(t)
	// Only checkers a required gate actually invokes need a control; the oracle
	// stubs every script the whole Makefile names, which is wider.
	o.checkers = o.checkersInvoked(required)
	assertEveryControlIsReached(t, o, required, uncontrolled)
}
