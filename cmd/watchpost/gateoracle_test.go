package main

// gateoracle_test.go — the Makefile half of the gate layer, EXECUTED, one thing
// painted at a time.
//
// WHY EXECUTION. A blind reviewer defeated the parsed model six ways in one
// sitting and named the cause: a semantic question ("can this gate fail?")
// decided by pattern on a fragment, in a language whose global constructs the
// parser never reads. So make and sh are the oracle: every project checker and
// every toolchain command is a stub, and `make <gate>` is RUN.
//
// WHY ONE THING AT A TIME. Two more reviewers found the two things execution
// introduces. First, the scratch tree is not the real tree: a preflight red only
// here masks `|| exit 0` behind it — so every gate runs GREEN first, and red
// under green is UNJUDGEABLE by name (FR-11.6). Second, and the deeper one: red
// under RED proves only that the gate is red when EVERYTHING is red. `x.sh ||
// exit 0` followed by `go version` is green-under-green, red-under-red, and
// neutered — and in the real Makefile `mutant-check` with its verdict replaced by
// `exit 0`, and `lint-injector … || true`, both read as sound. So for each gate,
// for each stub its recipe REACHES — its checkers, each toolchain sub-command,
// its prerequisites, the targets it recurses into — that ONE is painted red and
// the gate must go red. The control proof already worked this way; now
// everything does.
//
// WHY STUBS ANSWER BY SUB-COMMAND. `go test` and `go clean` are the same `go`;
// if the stub had one status, painting "go" red would red `mutant-check` for its
// cache-clean and hide the discarded verdict. `go:test`, `go:clean`, `go:build`
// are separate keys, and `--self-test` is a separate key on every checker.
//
// WHY THE PHONY AUDIT WALKS. Four required gates were not in .PHONY, so `touch
// lint-identity` silenced the real tree. And `gate: gate-run` with the recipe on
// a non-phony `gate-run` is the same hole one hop along — so every node a gate
// reaches through prerequisites that has a recipe must be phony too.
//
// WHY ABSENCE IS ALWAYS AN ERROR. If a `ciOnly` row excused a required gate from
// the database, then a deleted rule, a green `.DEFAULT:`, and one row would skip
// every executed check for it. A `ciOnly` row exempts a gate from `verify`; it
// never exempts it from having a phony rule.
//
// IT RUNS IN A SCRATCH TREE, NEVER THE REPOSITORY, with MAKEFLAGS stripped so a
// parent make's -i or -n cannot reach it, and requires the make CI runs.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// tools are the toolchain commands a gate recipe reaches on PATH.
var tools = []string{"go", "gofmt", "a2dh", "python3", "expect", "golangci-lint", "govulncheck", "shasum"}

// goSubs are the `go` sub-commands with their own status key (`go:test`…).
var goSubs = []string{"test", "build", "run", "clean", "vet", "version", "generate", "install", "mod:tidy", "mod:verify", "mod:download"}

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

// oracleTree is a scratch directory: one Makefile, make's own database of it,
// and a status per stub KEY. Keys are `scripts/x.sh` (the checker), `ctl:scripts/x.sh`
// (its --self-test answer), `scripts/x_test.sh` (a sibling control), `go:test`
// and the other sub-commands, and the remaining tools by name.
type oracleTree struct {
	dir      string
	db       map[string]*target
	checkers []string // scripts/… the database's recipes name, minus siblings
	siblings []string // scripts/…_test.sh
	status   map[string]int
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
	o := &oracleTree{dir: dir, status: map[string]int{}}
	o.paintAll(t, 3, 3) // tools exist before make -pn runs $(shell …)
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

// ---- stubs, rendered from the status map ------------------------------------------------

func (o *oracleTree) st(key string, def int) int {
	if v, ok := o.status[key]; ok {
		return v
	}
	return def
}

// render writes every stub from the status map. The checker stub answers
// `--self-test` with its ctl key; the go stub answers by sub-command and honours
// treelock's `--` contract so `verify` is judged through to the gates.
func (o *oracleTree) render(t reporter) {
	t.Helper()
	w := func(path, body string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("COULD NOT RUN — %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatalf("COULD NOT RUN — writing %s: %v", path, err)
		}
	}
	for _, k := range o.checkers { // bounded by the checker list (P10-02)
		w(filepath.Join(o.dir, k), fmt.Sprintf("#!/bin/sh\ncase \"$1\" in --self-test|-self-test) exit %d;; *) exit %d;; esac\n",
			o.st("ctl:"+k, 3), o.st(k, 3)))
	}
	for _, s := range o.siblings { // bounded by the sibling list (P10-02)
		w(filepath.Join(o.dir, s), fmt.Sprintf("#!/bin/sh\nexit %d\n", o.st(s, 3)))
	}
	for _, tool := range tools { // bounded by the tool list (P10-02)
		if tool != "go" {
			// AN INTERPRETER RUNS ITS SCRIPT. `python3 scripts/x.py` and `expect
			// scripts/x.expect` never exec the script themselves, so painting the
			// script red would land nowhere; the stub interpreter execs a scripts/
			// argument, and the script's own stub answers.
			// A BROKEN INTERPRETER FAILS EVERYTHING: its own status is consulted
			// before it execs anything, so painting `python3` red is "python3 is
			// gone" and every gate that needs it must go red.
			w(filepath.Join(o.dir, "bin", tool), fmt.Sprintf(`#!/bin/sh
st=%d
[ "$st" -ne 0 ] && exit "$st"
case "$1" in scripts/*|./scripts/*) exec "$@";; esac
exit "$st"
`, o.st(tool, 3)))
			continue
		}
		var cases strings.Builder
		for _, sub := range goSubs { // bounded by the sub-command list (P10-02)
			cases.WriteString(fmt.Sprintf("  %s) exit %d;;\n", sub, o.st("go:"+sub, 3)))
		}
		w(filepath.Join(o.dir, "bin", "go"), fmt.Sprintf(`#!/bin/sh
key="$1"; [ "$1" = mod ] && key="mod:$2"
if [ "$1" = run ] && [ "$2" = ./tools/treelock ]; then
  while [ $# -gt 0 ]; do
    if [ "$1" = -- ]; then shift; exec "$@"; fi
    shift
  done
fi
case "$key" in
%s  *) exit %d;;
esac
`, cases.String(), o.st("go:*", 3)))
	}
}

// paintAll sets every checker to chk, every control and sibling to ctl, and every
// tool key to chk.
func (o *oracleTree) paintAll(t reporter, chk, ctl int) {
	t.Helper()
	o.status = map[string]int{}
	for _, k := range o.checkers { // bounded by the checker list (P10-02)
		o.status[k], o.status["ctl:"+k] = chk, ctl
	}
	for _, s := range o.siblings { // bounded by the sibling list (P10-02)
		o.status[s] = ctl
	}
	for _, tool := range tools { // bounded by the tool list (P10-02)
		o.status[tool] = chk
	}
	for _, sub := range goSubs { // bounded by the sub-command list (P10-02)
		o.status["go:"+sub] = chk
	}
	o.status["go:*"] = chk
	o.render(t)
}

// paintKey sets ONE key and re-renders.
func (o *oracleTree) paintKey(t reporter, key string, status int) {
	t.Helper()
	o.status[key] = status
	o.render(t)
}

// run executes `make <gate>` with the stub PATH in front, MAKEFLAGS stripped, and
// returns the status.
func (o *oracleTree) run(gate string) int {
	cmd := exec.Command("make", "-s", gate)
	cmd.Dir = o.dir
	cmd.Env = o.env()
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			return ex.ExitCode()
		}
		return -1
	}
	return 0
}

// env is the parent environment minus what a parent make would hand down — a
// `-i` or `-n` in MAKEFLAGS reaches every child make, and the oracle runs inside
// `make race` (N4).
func (o *oracleTree) env() []string {
	var out []string
	for _, kv := range os.Environ() { // bounded by the environment (P10-02)
		switch strings.SplitN(kv, "=", 2)[0] {
		case "MAKEFLAGS", "MAKELEVEL", "MFLAGS", "MAKE_TERMOUT", "MAKE_TERMERR", "PATH":
			continue
		}
		out = append(out, kv)
	}
	return append(out, "PATH="+filepath.Join(o.dir, "bin")+":"+os.Getenv("PATH"))
}

// ---- make's database --------------------------------------------------------------------

// database is `make -pn`'s view: every real target, its expanded recipe, and
// whether make calls it phony.
func (o *oracleTree) database(t reporter) map[string]*target {
	t.Helper()
	cmd := exec.Command("make", "-pn", "-f", "Makefile")
	cmd.Dir = o.dir
	cmd.Env = o.env()
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
			after := line[len(mm[1])+1:]
			for _, d := range strings.Fields(after) { // bounded by the rule (P10-02)
				if !strings.HasPrefix(d, "#") && !strings.HasPrefix(d, "|") {
					cur.deps = append(cur.deps, d)
				}
			}
			targets[mm[1]] = cur
		}
	}
	return targets
}

// ---- reach -----------------------------------------------------------------------------------

var (
	goSubCall = regexp.MustCompile(`(?:^|[\s@=;&|(])(?:go|\$\(GO\))\s+(mod\s+\w+|\w+)`)
	// NOT PRECEDED BY `/`: `go run golang.org/x/vuln/cmd/govulncheck` reaches `go:run`,
	// not a govulncheck binary that is never exec'd.
	toolCall   = regexp.MustCompile(`(?:^|[\s@=;&|(])(gofmt|a2dh|python3|expect|golangci-lint|govulncheck|shasum)\b`)
	makeRecurs = regexp.MustCompile(`\$\(MAKE\)(?:\s+--[a-z-]+)*\s+([a-z][a-z0-9-]*)`)
)

// reach is every stub KEY a gate can touch: the checkers and tool sub-commands in
// its own recipe, in its prerequisites' recipes, and in the targets it recurses
// into with `$(MAKE)`. Bounded by a visited set.
func (o *oracleTree) reach(gate string) []string {
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		if !contains(keys, k) {
			keys = append(keys, k)
		}
	}
	var walk func(name string)
	walk = func(name string) {
		tg := o.db[name]
		if tg == nil || seen[name] {
			return
		}
		seen[name] = true
		for _, d := range tg.deps { // bounded by the rule (P10-02)
			walk(d)
		}
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			for _, sg := range c.segs { // bounded by the command (P10-02)
				if sg.orchestration {
					continue
				}
				selfTest := strings.Contains(sg.text, "-self-test")
				for _, m := range checkerRef.FindAllStringSubmatch(sg.text, -1) {
					k := m[1]
					switch {
					case !strings.HasPrefix(k, "scripts/"):
					case strings.HasSuffix(k, "_test.sh"):
						add(k) // a sibling control is its own key
					case selfTest:
						add("ctl:" + k) // the control answer, not the check
					default:
						add(k)
					}
				}
			}
			for _, m := range goSubCall.FindAllStringSubmatch(c.text, -1) {
				sub := strings.Join(strings.Fields(m[1]), ":")
				if contains(goSubs, sub) {
					add("go:" + sub)
				} else {
					add("go:*")
				}
			}
			for _, m := range toolCall.FindAllStringSubmatch(c.text, -1) {
				add(m[1])
			}
			for _, m := range makeRecurs.FindAllStringSubmatch(c.text, -1) {
				walk(m[1])
			}
		}
	}
	walk(gate)
	sort.Strings(keys)
	return keys
}

// checkersInvoked is every checker any REQUIRED gate reaches.
func (o *oracleTree) checkersInvoked(required []string) []string {
	var out []string
	for _, g := range required { // bounded by the gate list (P10-02)
		for _, k := range o.reach(g) { // bounded by the reach (P10-02)
			if strings.HasPrefix(k, "scripts/") && !strings.HasSuffix(k, "_test.sh") && !contains(out, k) {
				out = append(out, k)
			}
		}
	}
	return out
}

// ---- the executed assertions -----------------------------------------------------------------

// EVERY REQUIRED GATE IS A PHONY MAKE TARGET, AND SO IS EVERY RECIPE-BEARING
// NODE IT REACHES (J1–J3, N1, N2).
func assertEveryRequiredGateIsPhony(t reporter, o *oracleTree, required []string) {
	t.Helper()
	var checked int
	for _, g := range required { // bounded by the gate list (P10-02)
		if o.db[g] == nil {
			t.Errorf("%s is a REQUIRED gate and make's database has no such target. A pattern rule, "+
				".DEFAULT, or a deleted rule may still 'run' it green. A ciOnly row exempts a gate from "+
				"`verify`; it never exempts it from having a phony rule. Give it one.", g)
			continue
		}
		checked++
		seen := map[string]bool{}
		var walk func(name string)
		walk = func(name string) {
			tg := o.db[name]
			if tg == nil || seen[name] {
				return
			}
			seen[name] = true
			if !tg.phony && len(tg.cmds) > 0 {
				t.Errorf("%s (reached from the required gate %s) has a recipe and is not in .PHONY. A file "+
					"named `%s` makes make say \"is up to date\" and run NOTHING.\nAdd it to .PHONY.", name, g, name)
			}
			for _, d := range tg.deps { // bounded by the rule (P10-02)
				walk(d)
			}
		}
		walk(g)
	}
	if checked < 5 {
		t.Fatalf("COULD NOT RUN — only %d required gate(s) are make targets", checked)
	}
}

// EVERY REQUIRED GATE GOES GREEN WHEN EVERYTHING IS, AND RED FOR EACH THING IT
// REACHES, PAINTED ALONE (H1–H4, M1–M7).
func assertEveryRequiredGateCanFail(t reporter, o *oracleTree, required []string) {
	t.Helper()
	var checked int
	for _, g := range required { // bounded by the gate list (P10-02)
		if o.db[g] == nil {
			continue // the phony audit owns that
		}
		checked++
		o.paintAll(t, 0, 0)
		if code := o.run(g); code != 0 {
			t.Errorf("UNJUDGEABLE — `make %s` exits %d with EVERY stub GREEN. Something it reaches is red for a "+
				"reason that is not a check (a preflight the scratch tree cannot satisfy, a real tool on an empty "+
				"tree). COULD-NOT-RUN, not a pass: make the recipe judgeable, or declare the gate.", g, code)
			continue
		}
		keys := o.reach(g)
		if len(keys) == 0 {
			t.Errorf("%s is a REQUIRED gate and reaches no checker and no toolchain command: nothing it runs "+
				"can fail.", g)
			continue
		}
		for _, k := range keys { // bounded by the reach (P10-02)
			o.paintAll(t, 0, 0)
			o.paintKey(t, k, 3)
			if code := o.run(g); code == 0 {
				t.Errorf("%s is a REQUIRED gate and `make %s` exits 0 with ONLY %s red.\nThe gate reaches %s and "+
					"does not fail when it does — its status is discarded somewhere on that path, and make has "+
					"already said so.", g, g, k, k)
			}
		}
	}
	o.paintAll(t, 3, 3)
	if checked < 5 {
		t.Fatalf("COULD NOT RUN — only %d required gate(s) are make targets", checked)
	}
}

// `make verify` REACHES THE GATES AND GOES RED FOR EACH CHECKER, PAINTED ALONE
// (H5, M7).
func assertVerifyCanFail(t reporter, o *oracleTree, required []string) {
	t.Helper()
	if o.db["verify"] == nil {
		t.Fatalf("COULD NOT RUN — no verify target")
	}
	o.paintAll(t, 0, 0)
	if code := o.run("verify"); code != 0 {
		t.Errorf("UNJUDGEABLE — `make verify` exits %d with every stub green; the entry point is red for a "+
			"reason that is not a gate", code)
		return
	}
	// THE ENTRY POINT MUST ACTUALLY REACH THE GATES. If verify is green with every
	// stub red, its delegation never ran them — a wrapper the go stub's treelock
	// contract does not match — and that is UNJUDGEABLE, not "discards the failure".
	o.paintAll(t, 3, 3)
	if code := o.run("verify"); code == 0 {
		t.Errorf("UNJUDGEABLE — `make verify` exits 0 with EVERY stub red: the entry point never reaches the " +
			"gates. The oracle's go stub delegates only for `go run ./tools/treelock … -- CMD`; a different " +
			"spelling or wrapper is not judged, and this is COULD-NOT-RUN.")
		return
	}
	// WHAT VERIFY REACHES, not what every required gate does: a CI-only gate's
	// checker is never run by verify and cannot make it red.
	var invoked []string
	for _, k := range o.reach("verify") { // bounded by the reach (P10-02)
		if strings.HasPrefix(k, "scripts/") || strings.HasPrefix(k, "ctl:") {
			invoked = append(invoked, k)
		}
	}
	if len(invoked) == 0 {
		t.Fatalf("COULD NOT RUN — verify reaches no checker; nothing to paint red")
	}
	for _, k := range invoked { // bounded by the checker list (P10-02)
		o.paintAll(t, 0, 0)
		o.paintKey(t, k, 3)
		if code := o.run("verify"); code == 0 {
			t.Errorf("`make verify` exits 0 with ONLY %s red. The entry point discards that gate's failure — "+
				"release.yml reads this exit, so a red release would ship.", k)
		}
	}
	o.paintAll(t, 3, 3)
}

// EVERY CONTROL IS REACHED (K1–K4): only the CONTROL red, every carrier tried.
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
			o.paintKey(t, strings.TrimSuffix(k, ".sh")+"_test.sh", 3)
		} else {
			o.paintKey(t, "ctl:"+k, 3)
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
// whether that control is the sibling script. Every carrier is returned.
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

// ---- the production gates ----------------------------------------------------------------------

func realOracle(t *testing.T) (*oracleTree, []string) {
	t.Helper()
	req := read(t, "../../06_docs/required-gates.txt")
	o := newOracleTree(t, read(t, "../../Makefile"), req, nil)
	t.Cleanup(func() { _ = os.RemoveAll(o.dir) })
	return o, parseRequired(req)
}

func TestEveryRequiredGateIsPhony(t *testing.T) {
	o, required := realOracle(t)
	assertEveryRequiredGateIsPhony(t, o, required)
}

func TestEveryRequiredGateCanFail(t *testing.T) {
	o, required := realOracle(t)
	assertEveryRequiredGateCanFail(t, o, required)
}

func TestVerifyCanFail(t *testing.T) {
	o, required := realOracle(t)
	assertVerifyCanFail(t, o, required)
}

func TestEveryControlIsReached(t *testing.T) {
	o, required := realOracle(t)
	assertEveryControlIsReached(t, o, required, uncontrolled)
}
