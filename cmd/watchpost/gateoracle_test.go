package main

// gateoracle_test.go — the Makefile half of the gate layer, EXECUTED and OBSERVED.
//
// WHY EXECUTION. "Can this gate fail?" is a question about what make and sh DO,
// and a parser answers it only for the spellings its author imagined. So make and
// sh are the oracle: every project checker and every toolchain command is a stub,
// and `make <gate>` is RUN.
//
// WHY OBSERVATION. "What does this gate reach?" is the same kind of question, and
// a regex over recipe text answered it wrongly six ways at once — `$(A2DH)`,
// `$$(./scripts/x.sh)`, `$(CURDIR)/scripts/x.sh`, `$(MAKE) -s`, a pattern rule, two
// `go run` calls collapsed to one key. So the stubs RECORD their invocation, and
// the green run's record IS the reach: whatever make actually executed, through
// variables, substitutions, absolute paths, wrappers and recursion. Nothing in
// this file reads a recipe.
//
// WHY ONE THING AT A TIME. Red under RED proves only that the gate is red when
// everything is. `x.sh || exit 0` followed by a live `go version` is red-under-red
// and neutered. So every gate runs GREEN first — red under green is UNJUDGEABLE
// by name (FR-11.6) — and then, for each key it recorded, that ONE is painted red
// and the gate must go red.
//
// WHY STUBS ANSWER BY KEY. `go test` and `go clean` are the same binary; if the
// stub had one status, painting "go" red would red `mutant-check` for its
// cache-clean and hide the discarded verdict. The key is `go:test`, `go:clean`,
// `go:run:<package>`, and `ctl:` in front of anything run with `--self-test`.
//
// WHY A FILE IS CREATED FOR EVERY NODE. A required gate that is not phony is
// silenced by a file of its name — `touch lint-identity` once silenced the real
// tree — and so is a non-phony node one hop along. So for every target make says
// it considers on the way to a gate, a file of that name is created and the gate
// run green again: if it recorded LESS than before, that file silences it.
//
// WHY ABSENCE IS ALWAYS AN ERROR. A `ciOnly` row exempts a gate from `verify`;
// it never exempts it from being a rule make knows.
//
// IT RUNS IN A SCRATCH TREE, NEVER THE REPOSITORY, with everything a parent make
// hands down stripped from the environment, and requires the make CI runs.

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// tools are the toolchain commands the oracle stubs on PATH. A command NOT here
// is real, and a real tool on an empty tree is red under green — loud. Adding a
// checker that is neither a script nor `go run` means adding it here.
var tools = []string{"go", "gofmt", "a2dh", "python3", "expect", "golangci-lint", "govulncheck", "shasum", "sha256sum"}

// interpreters exec their `scripts/` argument so the script's own stub answers;
// without one they refuse, so `python3 -m x` is UNJUDGEABLE rather than unseen.
var interpreters = []string{"python3", "expect"}

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

// oracleTree is a scratch directory: one Makefile, stubs that record every
// invocation to a log and answer from a status directory, and the recorded reach
// of each gate's green run.
type oracleTree struct {
	dir     string
	def     int             // the status every unpainted key answers with
	targets map[string]bool // the rules make -pn knows
	greens  map[string]greenRun
}

type greenRun struct {
	code  int
	reach []string
}

// The stubs read these from the environment; the tree sets them per run.
const (
	envLog     = "ORACLE_LOG"
	envStatus  = "ORACLE_STATUS"
	envDefault = "ORACLE_DEFAULT"
)

// answerStub records its key and exits with the painted status. A `--self-test`
// among the arguments before `--` makes the key a control (`ctl:`).
const answerStub = `#!/bin/sh
key="$1"; shift
for a in "$@"; do
  case "$a" in --) break;; --self-test|-self-test) key="ctl:$key"; break;; esac
done
printf '%s\n' "$key" >> "$ORACLE_LOG"
f="$ORACLE_STATUS/$(printf '%s' "$key" | tr / _)"
[ -f "$f" ] && exit "$(cat "$f")"
exit "$ORACLE_DEFAULT"
`

// goStub answers by sub-command — `go:test`, `go:mod:tidy`, `go:run:<package>` —
// and honours treelock's contract: `go run ./tools/treelock … -- CMD` runs CMD
// once its own status is green, so `verify` is judged through to the gates.
const goStub = `#!/bin/sh
sub="$1"; key="go:$sub"
case "$sub" in mod|run) key="go:$sub:$2";; esac
"%s" "$key" "$@" || exit $?
if [ "$sub" = run ]; then
  while [ $# -gt 0 ]; do
    if [ "$1" = -- ]; then shift; exec "$@"; fi
    shift
  done
fi
exit 0
`

// interpreterStub consults its own status first — a broken python3 fails
// everything that needs it — then execs its scripts/ argument.
const interpreterStub = `#!/bin/sh
"%s" %s "$@" || exit $?
while [ $# -gt 0 ]; do
  case "$1" in scripts/*|./scripts/*|*/scripts/*) exec "$@";; esac
  shift
done
echo "oracle: %s run without a scripts/ argument is not judged" >&2
exit 3
`

func newOracleTree(t reporter, makefile, requiredList string, scripts []string, extra map[string]string) *oracleTree {
	t.Helper()
	requireMake(t)
	dir, err := os.MkdirTemp("", "gate-oracle-*")
	if err != nil {
		t.Fatalf("COULD NOT RUN — no scratch directory: %v", err)
	}
	o := &oracleTree{dir: dir, greens: map[string]greenRun{}}
	w := func(name, body string, mode fs.FileMode) {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("COULD NOT RUN — laying out the oracle tree: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatalf("COULD NOT RUN — writing %s: %v", name, err)
		}
	}
	w("Makefile", makefile, 0o644)
	w("06_docs/required-gates.txt", requiredList, 0o644)
	for name, body := range extra { // bounded by the extra-file map (P10-02)
		w(name, body, 0o644)
	}
	answer := filepath.Join(dir, "bin", "oracle-answer")
	w("bin/oracle-answer", answerStub, 0o755)
	w("bin/go", fmt.Sprintf(goStub, answer), 0o755)
	for _, tool := range tools { // bounded by the tool list (P10-02)
		switch {
		case tool == "go":
		case contains(interpreters, tool):
			w("bin/"+tool, fmt.Sprintf(interpreterStub, answer, tool, tool), 0o755)
		default:
			w("bin/"+tool, fmt.Sprintf("#!/bin/sh\nexec \"%s\" %s \"$@\"\n", answer, tool), 0o755)
		}
	}
	for _, s := range scripts { // bounded by the script list (P10-02)
		w(s, fmt.Sprintf("#!/bin/sh\nexec \"%s\" %s \"$@\"\n", answer, s), 0o755)
	}
	if err := os.MkdirAll(filepath.Join(dir, "status"), 0o755); err != nil {
		t.Fatalf("COULD NOT RUN — %v", err)
	}
	o.paint(0)
	o.targets = o.rules(t)
	return o
}

// ---- painting and running ----------------------------------------------------------------

// paint sets the default every key answers with and paints the named keys red.
func (o *oracleTree) paint(def int, red ...string) {
	o.def = def
	status := filepath.Join(o.dir, "status")
	_ = os.RemoveAll(status)
	_ = os.MkdirAll(status, 0o755)
	for _, k := range red { // bounded by the painted keys (P10-02)
		_ = os.WriteFile(filepath.Join(status, strings.ReplaceAll(k, "/", "_")), []byte("3\n"), 0o644)
	}
}

// run executes `make <gate>` with the stub PATH in front and returns the status
// and every key the stubs recorded, sorted and deduplicated.
func (o *oracleTree) run(gate string) (int, []string) {
	log := filepath.Join(o.dir, "reached")
	_ = os.WriteFile(log, nil, 0o644)
	cmd := exec.Command("make", "-s", gate)
	cmd.Dir = o.dir
	cmd.Env = o.env()
	code := 0
	if err := cmd.Run(); err != nil {
		code = -1
		if ex, ok := err.(*exec.ExitError); ok {
			code = ex.ExitCode()
		}
	}
	raw, _ := os.ReadFile(log)
	seen := map[string]bool{}
	var reach []string
	for _, k := range strings.Split(string(raw), "\n") { // bounded by the record (P10-02)
		if k != "" && !seen[k] {
			seen[k] = true
			reach = append(reach, k)
		}
	}
	sort.Strings(reach)
	return code, reach
}

// green is the gate's run with every key green, cached, and reported ONCE as
// UNJUDGEABLE if it is red: something it reaches is red for a reason that is not
// a check, and no verdict about its checks can be read from it.
func (o *oracleTree) green(t reporter, gate string) greenRun {
	t.Helper()
	if g, ok := o.greens[gate]; ok {
		return g
	}
	o.paint(0)
	code, reach := o.run(gate)
	g := greenRun{code: code, reach: reach}
	o.greens[gate] = g
	if code != 0 {
		t.Errorf("UNJUDGEABLE — `make %s` exits %d with EVERY stub GREEN. Something it reaches is red for a "+
			"reason that is not a check (a preflight the scratch tree cannot satisfy, a real tool on an empty "+
			"tree, an interpreter run without a script). COULD-NOT-RUN, not a pass: make the recipe judgeable, "+
			"or declare the gate.", gate, code)
	}
	return g
}

// env is the parent environment minus everything a parent make hands down — a
// `-i` in MAKEFLAGS or GNUMAKEFLAGS, a MAKEFILES that sets .SHELLFLAGS — plus the
// stub PATH and the stubs' own variables.
func (o *oracleTree) env() []string {
	var out []string
	for _, kv := range os.Environ() { // bounded by the environment (P10-02)
		switch strings.SplitN(kv, "=", 2)[0] {
		case "MAKEFLAGS", "GNUMAKEFLAGS", "MFLAGS", "MAKELEVEL", "MAKEFILES", "MAKE", "MAKEOVERRIDES",
			"MAKE_TERMOUT", "MAKE_TERMERR", "PATH", envLog, envStatus, envDefault:
			continue
		}
		out = append(out, kv)
	}
	return append(out,
		"PATH="+filepath.Join(o.dir, "bin")+":"+os.Getenv("PATH"),
		envLog+"="+filepath.Join(o.dir, "reached"),
		envStatus+"="+filepath.Join(o.dir, "status"),
		fmt.Sprintf("%s=%d", envDefault, o.def))
}

// ---- what make itself reports ---------------------------------------------------------------

// rules is the set of names `make -pn` lists as rules — not the recipes, not
// the prerequisites; only whether make knows the name.
func (o *oracleTree) rules(t reporter) map[string]bool {
	t.Helper()
	cmd := exec.Command("make", "-pn", "-f", "Makefile")
	cmd.Dir = o.dir
	cmd.Env = o.env()
	out, _ := cmd.Output()
	if len(out) == 0 {
		t.Fatalf("COULD NOT RUN — `make -pn` printed nothing; the Makefile does not parse")
	}
	rules := map[string]bool{}
	ruleLine := regexp.MustCompile(`^([A-Za-z0-9_./-][^:=#\s]*):(?:[^=]|$)`)
	var notATarget bool
	for _, line := range strings.Split(string(out), "\n") { // bounded by the database (P10-02)
		switch {
		case line == "# Not a target:":
			notATarget = true
		case strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" || strings.HasPrefix(line, "\t"):
		default:
			if m := ruleLine.FindStringSubmatch(line); m != nil && !notATarget && !strings.HasPrefix(m[1], ".") {
				rules[m[1]] = true
			}
			notATarget = false
		}
	}
	return rules
}

var considered = regexp.MustCompile("Considering target file [`'](.+)'\\.")

// nodes is every target make reports considering on the way to gate — its own
// walk, expanded, through pattern rules and `$(MAKE)` recursion alike.
func (o *oracleTree) nodes(gate string) []string {
	cmd := exec.Command("make", "-n", "--debug=v", gate)
	cmd.Dir = o.dir
	cmd.Env = o.env()
	out, _ := cmd.Output()
	var names []string
	for _, m := range considered.FindAllStringSubmatch(string(out), -1) { // bounded by the walk (P10-02)
		if !contains(names, m[1]) {
			names = append(names, m[1])
		}
	}
	return names
}

// ---- the executed assertions -----------------------------------------------------------------

// EVERY REQUIRED GATE IS A RULE MAKE KNOWS, AND NO FILE NAMED AFTER ANY NODE ON
// ITS WALK MAKES IT RUN LESS (J1–J3, N1, N2, O6).
func assertNoFileSilencesARequiredGate(t reporter, o *oracleTree, required []string) {
	t.Helper()
	var checked int
	for _, g := range required { // bounded by the gate list (P10-02)
		if !o.targets[g] {
			t.Errorf("%s is a REQUIRED gate and make's database has no such rule. A pattern rule, .DEFAULT, "+
				"or a deleted rule may still 'run' it green. A ciOnly row exempts a gate from `verify`; it "+
				"never exempts it from having a rule. Give it one.", g)
			continue
		}
		checked++
		base := o.green(t, g)
		if base.code != 0 {
			continue
		}
		for _, node := range o.nodes(g) { // bounded by make's walk (P10-02)
			path := filepath.Join(o.dir, node)
			if _, err := os.Stat(path); err == nil {
				continue // a real file or directory is legitimately there
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				continue
			}
			if err := os.WriteFile(path, nil, 0o644); err != nil {
				continue
			}
			o.paint(0)
			code, reach := o.run(g)
			_ = os.Remove(path)
			if code != 0 {
				continue // the file broke it, which is not silence
			}
			var lost []string
			for _, k := range base.reach { // bounded by the reach (P10-02)
				if !contains(reach, k) {
					lost = append(lost, k)
				}
			}
			if len(lost) > 0 {
				t.Errorf("a file named `%s` makes `make %s` exit 0 without running %v: make says \"is up to date\" "+
					"and skips it. Add %s to .PHONY.", node, g, lost, node)
			}
		}
	}
	if checked < 5 {
		t.Fatalf("COULD NOT RUN — only %d required gate(s) are make rules", checked)
	}
}

// EVERY REQUIRED GATE GOES GREEN WHEN EVERYTHING IS, AND RED FOR EACH THING IT
// RECORDED, PAINTED ALONE (H1–H4, M1–M7, O1–O6).
func assertEveryRequiredGateCanFail(t reporter, o *oracleTree, required []string) {
	t.Helper()
	var checked int
	for _, g := range required { // bounded by the gate list (P10-02)
		if !o.targets[g] {
			continue // the silence audit owns that
		}
		checked++
		base := o.green(t, g)
		if base.code != 0 {
			continue
		}
		if len(base.reach) == 0 {
			t.Errorf("%s is a REQUIRED gate and runs no checker and no toolchain command: nothing it does can "+
				"fail.", g)
			continue
		}
		for _, k := range base.reach { // bounded by the reach (P10-02)
			o.paint(0, k)
			if code, _ := o.run(g); code == 0 {
				t.Errorf("%s is a REQUIRED gate and `make %s` exits 0 with ONLY %s red.\nThe gate ran %s and does "+
					"not fail when it does — its status is discarded somewhere on that path, and make has "+
					"already said so. A diagnostic under `|| true` is refused for the same reason: move it out "+
					"of the gate or let it fail.", g, g, k, k)
			}
		}
	}
	o.paint(0)
	if checked < 5 {
		t.Fatalf("COULD NOT RUN — only %d required gate(s) are make rules", checked)
	}
}

// `make verify` REACHES THE GATES AND GOES RED FOR EACH THING IT RECORDED,
// PAINTED ALONE (H5, M7, E15).
func assertVerifyCanFail(t reporter, o *oracleTree, required []string) {
	t.Helper()
	if !o.targets["verify"] {
		t.Fatalf("COULD NOT RUN — no verify rule")
	}
	base := o.green(t, "verify")
	if base.code != 0 {
		return
	}
	// THE ENTRY POINT MUST ACTUALLY REACH THE GATES. If its green run recorded no
	// checker, its delegation never ran them — a wrapper the go stub's treelock
	// contract does not match — and that is UNJUDGEABLE, not "discards the failure".
	var checkers int
	for _, k := range base.reach { // bounded by the reach (P10-02)
		if strings.HasPrefix(k, "scripts/") || strings.HasPrefix(k, "ctl:") {
			checkers++
		}
	}
	if checkers == 0 {
		t.Errorf("UNJUDGEABLE — `make verify` exits 0 and ran no checker: the entry point never reaches the " +
			"gates. The oracle's go stub delegates only for `go run ./tools/treelock … -- CMD`; a different " +
			"spelling or wrapper is not judged, and this is COULD-NOT-RUN.")
		return
	}
	for _, k := range base.reach { // bounded by the reach (P10-02)
		o.paint(0, k)
		if code, _ := o.run("verify"); code == 0 {
			t.Errorf("`make verify` exits 0 with ONLY %s red. The entry point discards that failure — "+
				"release.yml reads this exit, so a red release would ship.", k)
		}
	}
	o.paint(0)
}

// EVERY CONTROL IS REACHED (K1–K4, A5–A10): only the CONTROL red, every carrier
// tried. A checker is every `scripts/` key a required gate recorded that is not
// itself a control; its control is `ctl:<key>` or a sibling `<key>_test.sh`, and
// its carriers are the required gates that recorded that control.
func assertEveryControlIsReached(t reporter, o *oracleTree, required []string, uncontrolledRows map[string]string) {
	t.Helper()
	var invoked []string
	for _, g := range required { // bounded by the gate list (P10-02)
		for _, k := range o.green(t, g).reach { // bounded by the reach (P10-02)
			if strings.HasPrefix(k, "scripts/") && !strings.HasSuffix(k, "_test.sh") && !contains(invoked, k) {
				invoked = append(invoked, k)
			}
		}
	}
	sort.Strings(invoked)
	var checked int
	for _, k := range invoked { // bounded by the checker list (P10-02)
		if _, declared := uncontrolledRows[k]; declared {
			continue
		}
		checked++
		ctl, sib := "ctl:"+k, strings.TrimSuffix(k, ".sh")+"_test.sh"
		var carriers []string
		for _, g := range required { // bounded by the gate list (P10-02)
			if reach := o.green(t, g).reach; contains(reach, ctl) || contains(reach, sib) {
				carriers = append(carriers, g)
			}
		}
		if len(carriers) == 0 {
			t.Errorf("%s is run by a required gate and no required gate runs its control — neither `%s "+
				"--self-test` nor a sibling `%s`.\nAdd the control to gate-controls, or add a row to "+
				"`uncontrolled` saying why this checker is trusted without evidence.", k, k, sib)
			continue
		}
		o.paint(0, ctl, sib)
		var reached bool
		for _, g := range carriers { // bounded by the carrier list (P10-02)
			if code, _ := o.run(g); code != 0 {
				reached = true
				break
			}
		}
		if !reached {
			t.Errorf("%s's control ran in %v and its failure is DISCARDED: with only that control red, every "+
				"carrier stays green — `|| true`, `;`, a `-` prefix — and make has said so.", k, carriers)
		}
	}
	o.paint(0)
	if checked < 3 {
		t.Fatalf("COULD NOT RUN — only %d checker(s) run by required gates", checked)
	}
}

// ---- the production gates ----------------------------------------------------------------------

// repoScripts is every file under scripts/ — the checkers the real Makefile can
// name — derived from the tree, not from the Makefile.
func repoScripts(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir("../../scripts", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, strings.TrimPrefix(filepath.ToSlash(path), "../../"))
		}
		return err
	})
	if err != nil || len(out) == 0 {
		t.Fatalf("COULD NOT RUN — no scripts/ tree to stub: %v", err)
	}
	return out
}

func realOracle(t *testing.T) (*oracleTree, []string) {
	t.Helper()
	req := read(t, "../../06_docs/required-gates.txt")
	o := newOracleTree(t, read(t, "../../Makefile"), req, repoScripts(t), nil)
	t.Cleanup(func() { _ = os.RemoveAll(o.dir) })
	return o, parseRequired(req)
}

func TestNoFileSilencesARequiredGate(t *testing.T) {
	o, required := realOracle(t)
	assertNoFileSilencesARequiredGate(t, o, required)
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
