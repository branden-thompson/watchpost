package main

// gateoracle_test.go — the Makefile half of the gate layer, EXECUTED rather than
// parsed.
//
// WHY EXECUTION. A blind reviewer defeated the parsed model six ways in one
// sitting and named the cause: a semantic question ("can this gate fail?")
// decided by pattern on a fragment, in a language whose global constructs the
// parser never reads. No number of regexes closes that. So make and sh are the
// oracle: every project checker is a stub, every toolchain command on PATH is a
// stub, and `make <gate>` is RUN.
//
// WHY A POSITIVE CONTROL. A second reviewer found the class execution creates:
// the scratch tree is not the real tree, so a recipe can be red here for a
// reason unrelated to its check — `test -f go.mod || exit 1` — and green
// forever in the tree it judges, with `|| exit 0` sitting behind the preflight.
// One bit of exit status cannot tell "the check failed" from "something else
// did". So every gate is run GREEN first: with every stub exiting 0 it must exit
// 0, or it is UNJUDGEABLE and says so by name — never "sound" (FR-11.6: an
// instrument answers a known case before it is believed about an unknown one).
//
// WHY STUBS ANSWER BY ARGUMENT. A control is proved reached by painting it red
// and watching its carrier go red — but a carrier that also runs the checker
// went red for the checker's sake, and a neutered control passed behind it. So
// a stub exits with one status for `--self-test` and another for anything
// else, and the proof paints the CONTROL status alone.
//
// WHY THE PHONY AUDIT. Four required gates were not in .PHONY, so `touch
// lint-identity` made `make lint-identity` say "is up to date" and exit 0 in
// the real tree — nothing ran. A required make target must be phony, and make's
// own database says whether it is.
//
// IT RUNS IN A SCRATCH TREE, NEVER THE REPOSITORY, and requires the make CI runs.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// toolchain are the commands a gate recipe reaches on PATH. Each is a stub that
// exits with the CHECKER status; `go` additionally honours treelock's contract.
var toolchain = []string{"go", "gofmt", "a2dh", "python3", "expect", "golangci-lint", "govulncheck", "shasum"}

// minMake is the oldest GNU make the oracle will judge with. CI runs 4.x; macOS
// ships 3.81, which has no .SHELLFLAGS, so a gate silenced that way would read
// as sound here and silenced there. Two verdicts for one Makefile is not a gate.
const minMake = "3.82"

var makeVersionLine = regexp.MustCompile(`GNU Make (\d+)\.(\d+)`)

func requireMake(t reporter) {
	t.Helper()
	out, err := exec.Command("make", "--version").Output()
	if err != nil {
		t.Fatalf("COULD NOT RUN — `make --version` failed: %v", err)
	}
	m := makeVersionLine.FindStringSubmatch(string(out))
	if m == nil {
		t.Fatalf("COULD NOT RUN — `make` is not GNU make: %q", strings.SplitN(string(out), "\n", 2)[0])
	}
	var major, minor, wantMajor, wantMinor int
	fmt.Sscanf(m[1]+" "+m[2], "%d %d", &major, &minor)
	fmt.Sscanf(strings.ReplaceAll(minMake, ".", " "), "%d %d", &wantMajor, &wantMinor)
	if major < wantMajor || (major == wantMajor && minor < wantMinor) {
		t.Fatalf("COULD NOT RUN — GNU Make %s.%s is on PATH and the oracle needs %s or newer, the make CI "+
			"runs. On macOS: `brew install make`, then put /opt/homebrew/opt/make/libexec/gnubin at the "+
			"front of PATH so `make` is 4.x.", m[1], m[2], minMake)
	}
}

// oracleTree is a scratch directory holding one Makefile, the stubs it reaches,
// and make's own database of it.
type oracleTree struct {
	dir      string
	checkers []string // every scripts/… path the database's recipes name
	siblings []string // every scripts/…_test.sh — a control in its own script
	db       map[string]*target
}

func newOracleTree(t reporter, makefile, requiredList string, extra map[string]string) *oracleTree {
	t.Helper()
	requireMake(t)
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
	for name, body := range extra { // bounded by the extra-file map (P10-02)
		must(os.MkdirAll(filepath.Join(dir, filepath.Dir(name)), 0o755))
		must(os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	}
	must(os.MkdirAll(filepath.Join(dir, "bin"), 0o755))
	o := &oracleTree{dir: dir}
	o.paintToolchain(t, 3)
	o.db = o.database(t)
	sibling := regexp.MustCompile(`(scripts/[A-Za-z0-9_/.-]+_test\.sh)`)
	for _, tg := range o.db { // bounded by the database (P10-02)
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			for _, k := range c.checkers() { // bounded by the command (P10-02)
				if strings.HasPrefix(k, "scripts/") && !strings.HasSuffix(k, "_test.sh") && !contains(o.checkers, k) {
					o.checkers = append(o.checkers, k)
				}
			}
			for _, m := range sibling.FindAllStringSubmatch(c.text, -1) {
				if !contains(o.siblings, m[1]) {
					o.siblings = append(o.siblings, m[1])
				}
			}
		}
	}
	o.paintAll(t, 3, 3)
	return o
}

// ---- stubs ---------------------------------------------------------------------------

// checkerStub answers `--self-test` with ctl and anything else with chk (K1).
func checkerStub(chk, ctl int) string {
	return fmt.Sprintf("#!/bin/sh\ncase \"$1\" in --self-test|-self-test) exit %d;; *) exit %d;; esac\n", ctl, chk)
}

// goStub exits with the checker status — EXCEPT `go run ./tools/treelock … --
// CMD…`, which execs CMD, because that is treelock's contract and `verify`
// delegates to the gates through it. Without this, verify would be judged on
// one stub call and never reach the gates below it (H5).
func goStub(status int) string {
	return fmt.Sprintf(`#!/bin/sh
if [ "$1" = run ] && [ "$2" = ./tools/treelock ]; then
  while [ $# -gt 0 ]; do
    if [ "$1" = -- ]; then shift; exec "$@"; fi
    shift
  done
fi
exit %d
`, status)
}

func writeExec(path, body string) error { return os.WriteFile(path, []byte(body), 0o755) }

func (o *oracleTree) paintToolchain(t reporter, status int) {
	t.Helper()
	for _, tool := range toolchain { // bounded by the toolchain list (P10-02)
		body := fmt.Sprintf("#!/bin/sh\nexit %d\n", status)
		if tool == "go" {
			body = goStub(status)
		}
		if err := writeExec(filepath.Join(o.dir, "bin", tool), body); err != nil {
			t.Fatalf("COULD NOT RUN — painting %s: %v", tool, err)
		}
	}
}

// paintChecker sets one checker's two statuses.
func (o *oracleTree) paintChecker(t reporter, k string, chk, ctl int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(o.dir, filepath.Dir(k)), 0o755); err != nil {
		t.Fatalf("COULD NOT RUN — %v", err)
	}
	if err := writeExec(filepath.Join(o.dir, k), checkerStub(chk, ctl)); err != nil {
		t.Fatalf("COULD NOT RUN — painting %s: %v", k, err)
	}
}

// paintSibling sets a `_test.sh` control's status.
func (o *oracleTree) paintSibling(t reporter, sib string, status int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(o.dir, filepath.Dir(sib)), 0o755); err != nil {
		t.Fatalf("COULD NOT RUN — %v", err)
	}
	if err := writeExec(filepath.Join(o.dir, sib), fmt.Sprintf("#!/bin/sh\nexit %d\n", status)); err != nil {
		t.Fatalf("COULD NOT RUN — painting %s: %v", sib, err)
	}
}

// paintAll sets every checker (chk, ctl), every sibling (ctl) and the toolchain (chk).
func (o *oracleTree) paintAll(t reporter, chk, ctl int) {
	t.Helper()
	o.paintToolchain(t, chk)
	for _, k := range o.checkers { // bounded by the checker list (P10-02)
		o.paintChecker(t, k, chk, ctl)
	}
	for _, s := range o.siblings { // bounded by the sibling list (P10-02)
		o.paintSibling(t, s, ctl)
	}
}

// run executes `make <gate>` with the stub PATH in front and returns the status.
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

// ---- make's database ------------------------------------------------------------------

// database is `make -pn`'s view: every real target, its expanded recipe, and
// whether make calls it phony. A target named through `$(VAR):`, brought in by
// `include`, or chosen by `ifeq` appears here as make will run it; a name that
// is not a target is preceded by `# Not a target:`.
func (o *oracleTree) database(t reporter) map[string]*target {
	t.Helper()
	cmd := exec.Command("make", "-pn", "-f", "Makefile")
	cmd.Dir = o.dir
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(o.dir, "bin")+":"+os.Getenv("PATH"))
	out, _ := cmd.Output()
	if len(out) == 0 {
		t.Fatalf("COULD NOT RUN — `make -pn` printed nothing; the Makefile does not parse")
	}
	targets := map[string]*target{}
	targetLine := regexp.MustCompile(`^([A-Za-z0-9_./-][^:=#\s]*):(?:[^=]|$)`)
	var cur *target
	var notATarget bool
	for _, line := range strings.Split(string(out), "\n") { // bounded by the database (P10-02)
		switch {
		case line == "# Not a target:":
			notATarget, cur = true, nil
		case strings.HasPrefix(line, "\t"):
			if cur != nil {
				if body := strings.TrimSpace(line); body != "" && !strings.HasPrefix(body, "#") {
					cur.cmds = append(cur.cmds, newCommand(body))
				}
			}
		case strings.HasPrefix(line, "#  Phony target"):
			if cur != nil {
				cur.phony = true
			}
		case strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "":
		default:
			mm := targetLine.FindStringSubmatch(line)
			cur = nil
			if mm == nil || notATarget || strings.HasPrefix(mm[1], ".") {
				notATarget = false
				continue
			}
			cur = &target{name: mm[1]}
			targets[mm[1]] = cur
		}
	}
	return targets
}

// checkersInvoked is every scripts/… checker any REQUIRED gate's recipe names.
func (o *oracleTree) checkersInvoked(required []string) []string {
	var out []string
	for _, g := range required { // bounded by the gate list (P10-02)
		tg := o.db[g]
		if tg == nil {
			continue
		}
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			for _, k := range c.checkers() { // bounded by the command (P10-02)
				if strings.HasPrefix(k, "scripts/") && !strings.HasSuffix(k, "_test.sh") && !contains(out, k) {
					out = append(out, k)
				}
			}
		}
	}
	return out
}

// ---- the executed assertions ----------------------------------------------------------

// EVERY REQUIRED MAKE TARGET IS PHONY, AND EVERY REQUIRED GATE IS A MAKE TARGET
// OR IS DECLARED CI-ONLY (J1–J3).
//
// A non-phony gate is silenced by a file with its name: make reports it "up to
// date" and runs nothing. A required gate absent from make's database — a
// pattern rule, a .DEFAULT, a deleted rule — is not "CI-only by omission"; it
// is either declared so, with a reason, or it is a finding.
func assertEveryRequiredGateIsPhony(t reporter, o *oracleTree, required []string, ciOnlyRows map[string]string) {
	t.Helper()
	var checked int
	for _, g := range required { // bounded by the gate list (P10-02)
		tg := o.db[g]
		if tg == nil {
			if _, declared := ciOnlyRows[g]; !declared {
				t.Errorf("%s is a REQUIRED gate and make's database has no such target — a pattern rule, "+
					".DEFAULT, or a deleted rule may still 'run' it, and nothing declares it CI-only.\n"+
					"Give it a rule, or add a `ciOnly` row with the reason.", g)
			}
			continue
		}
		checked++
		if !tg.phony {
			t.Errorf("%s is a REQUIRED gate and is not in .PHONY. A file named `%s` in the tree makes "+
				"`make %s` say \"is up to date\" and exit 0 having run NOTHING — four gates sat that way.\n"+
				"Add it to .PHONY.", g, g, g)
		}
	}
	if checked < 5 {
		t.Fatalf("COULD NOT RUN — only %d required gate(s) are make targets", checked)
	}
}

// EVERY REQUIRED GATE GOES GREEN WHEN ITS CHECKS DO, AND RED WHEN THEY DO NOT.
//
// The green half is the positive control (FR-11.6): a gate red under green
// stubs is red for a reason other than its check — a preflight the scratch tree
// cannot satisfy, a prerequisite that is red here — and its red-under-red
// verdict would mean nothing. It is UNJUDGEABLE, reported by name, and never
// counted as sound. The red half is the property.
func assertEveryRequiredGateCanFail(t reporter, o *oracleTree, required []string, ciOnlyRows map[string]string) {
	t.Helper()
	var checked int
	for _, g := range required { // bounded by the gate list (P10-02)
		if o.db[g] == nil {
			continue // the phony audit owns that
		}
		checked++
		o.paintAll(t, 0, 0)
		if code := o.run(g); code != 0 {
			t.Errorf("UNJUDGEABLE — `make %s` exits %d with EVERY check stubbed GREEN. Something in its "+
				"recipe or a prerequisite is red for a reason that is not its check (a preflight the scratch "+
				"tree cannot satisfy, a real tool run on an empty tree), so a red-under-red verdict would prove "+
				"nothing. This is COULD-NOT-RUN, not a pass: make the recipe judgeable, or declare the gate.", g, code)
			continue
		}
		o.paintAll(t, 3, 3)
		if code := o.run(g); code == 0 {
			t.Errorf("%s is a REQUIRED gate and `make %s` exits 0 with every check it runs stubbed RED.\n"+
				"The gate cannot fail — its status is discarded somewhere, and make has already said so.", g, g)
		}
	}
	o.paintAll(t, 3, 3)
	if checked < 5 {
		t.Fatalf("COULD NOT RUN — only %d required gate(s) are make targets", checked)
	}
}

// `make verify` REACHES THE GATES AND GOES RED WHEN ONE DOES (H5).
//
// Green first: with everything green verify must exit 0, or the entry point is
// unjudgeable. Then ONE checker red and everything else green: verify must exit
// non-zero, or the entry point discards the failure — a `-` on the treelock
// line — and release.yml, which reads that exit, would ship a red release.
func assertVerifyCanFail(t reporter, o *oracleTree, required []string) {
	t.Helper()
	if o.db["verify"] == nil {
		t.Fatalf("COULD NOT RUN — no verify target")
	}
	invoked := o.checkersInvoked(required)
	if len(invoked) == 0 {
		t.Fatalf("COULD NOT RUN — no required gate invokes a checker; nothing to paint red")
	}
	o.paintAll(t, 0, 0)
	if code := o.run("verify"); code != 0 {
		t.Errorf("UNJUDGEABLE — `make verify` exits %d with every check green; the entry point is red for a "+
			"reason that is not a gate", code)
		return
	}
	o.paintChecker(t, invoked[0], 3, 0)
	if code := o.run("verify"); code == 0 {
		t.Errorf("`make verify` exits 0 with %s red and everything else green. The entry point discards "+
			"the failure — release.yml reads this exit, so a red release would ship.", invoked[0])
	}
	o.paintAll(t, 3, 3)
}

// EVERY CONTROL IS REACHED (K1–K3).
//
// For each checker a required gate invokes: everything green, then ONLY that
// checker's CONTROL red — the `--self-test` status for a flag, the sibling
// script for a `_test.sh` — and every carrier tried. If none goes red, the
// control does not run: commented out, behind `|| true`, inside a define, or
// in a target nothing invokes. The checker status stays green throughout, so a
// carrier cannot go red for the checker's sake.
func assertEveryControlIsReached(t reporter, o *oracleTree, required []string, uncontrolledRows map[string]string) {
	t.Helper()
	invoked := o.checkersInvoked(required)
	var checked int
	for _, k := range invoked { // bounded by the checker list (P10-02)
		if _, declared := uncontrolledRows[k]; declared {
			continue
		}
		checked++
		carriers, viaSibling := o.controlsOf(k, required)
		if len(carriers) == 0 {
			t.Errorf("%s is invoked by a required gate and no required gate's recipe runs its control — "+
				"neither `%s --self-test` nor a sibling `%s_test.sh`.\nAdd the control to gate-controls, or "+
				"add a row to `uncontrolled` saying why this checker is trusted without evidence.",
				k, k, strings.TrimSuffix(k, ".sh"))
			continue
		}
		o.paintAll(t, 0, 0)
		if viaSibling {
			o.paintSibling(t, strings.TrimSuffix(k, ".sh")+"_test.sh", 3)
		} else {
			o.paintChecker(t, k, 0, 3)
		}
		var reached bool
		for _, g := range carriers { // bounded by the carrier list (P10-02)
			if o.run(g) != 0 {
				reached = true
				break
			}
		}
		if !reached {
			t.Errorf("%s's control is named in %v and NOT REACHED: with only that control red, every carrier "+
				"stays green. It is commented out, behind `|| true`, inside a define, or otherwise never runs "+
				"— make has said so.", k, carriers)
		}
	}
	o.paintAll(t, 3, 3)
	if checked < 3 {
		t.Fatalf("COULD NOT RUN — only %d checker(s) invoked by required gates", checked)
	}
}

// controlsOf is every required gate whose recipe names a control for k, and
// whether that control is the sibling script. Text finds CANDIDATES; execution
// decides. Every carrier is returned (K2): none is preferred over another.
func (o *oracleTree) controlsOf(k string, required []string) (carriers []string, viaSibling bool) {
	sib := strings.TrimSuffix(k, ".sh") + "_test.sh"
	for _, g := range required { // bounded by the gate list (P10-02)
		tg := o.db[g]
		if tg == nil {
			continue
		}
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			if strings.Contains(c.text, sib) {
				carriers, viaSibling = append(carriers, g), true
				break
			}
			if strings.Contains(c.text, k) && strings.Contains(c.text, "-self-test") {
				carriers = append(carriers, g)
				break
			}
		}
	}
	return carriers, viaSibling
}

// ---- the production gates ---------------------------------------------------------------

func realOracle(t *testing.T) (*oracleTree, []string) {
	t.Helper()
	req := read(t, "../../06_docs/required-gates.txt")
	o := newOracleTree(t, read(t, "../../Makefile"), req, nil)
	t.Cleanup(func() { _ = os.RemoveAll(o.dir) })
	return o, parseRequired(req)
}

func TestEveryRequiredGateIsPhony(t *testing.T) {
	o, required := realOracle(t)
	assertEveryRequiredGateIsPhony(t, o, required, ciOnly)
}

func TestEveryRequiredGateCanFail(t *testing.T) {
	o, required := realOracle(t)
	assertEveryRequiredGateCanFail(t, o, required, ciOnly)
}

func TestVerifyCanFail(t *testing.T) {
	o, required := realOracle(t)
	assertVerifyCanFail(t, o, required)
}

func TestEveryControlIsReached(t *testing.T) {
	o, required := realOracle(t)
	assertEveryControlIsReached(t, o, required, uncontrolled)
}
