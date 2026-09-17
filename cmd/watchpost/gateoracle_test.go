package main

// gateoracle_test.go — the Makefile half of the gate layer, EXECUTED and OBSERVED,
// IN THE TREE.
//
// WHY EXECUTION. "Can this gate fail?" is a question about what make and sh DO,
// and a parser answers it only for the spellings its author imagined. So make and
// sh are the oracle: every project checker and every toolchain command is a stub,
// and `make <gate>` is RUN.
//
// WHY OBSERVATION. "What does this gate reach?" is the same kind of question, and
// a regex over recipe text answered it wrongly six ways at once. So the stubs
// RECORD every invocation, and the green run's record IS the reach: whatever make
// actually executed, through variables, substitutions, absolute paths, wrappers
// and recursion. Nothing in this file reads a recipe.
//
// WHY THE TREE. A scratch tree that is empty answers every question a recipe asks
// of the tree the opposite way the repository does: `git diff --quiet && exit 0`
// runs the checker in scratch and skips it in CI; `find | xargs gofmt` records
// gofmt on CI's xargs and nothing on BSD's; `gate: go.mod` is silenced by a file
// only where go.mod exists. So the scratch is a shared clone of the repository
// with the working tree copied over it, and the stubs interpose by PATH ALONE: a
// script is answered through its `#!/usr/bin/env sh` shebang, and not one byte
// of the tree is rewritten. The `go build` stub writes a recording stub at `-o`,
// so a compiled checker is a key like any other.
//
// WHY ONE INVOCATION AT A TIME. Red under RED proves only that the gate is red
// when everything is; red under one KEY proves only that the gate is red when
// every `go test` is. So every gate runs GREEN first — red under green is
// UNJUDGEABLE by name (FR-11.6) — and then each recorded INVOCATION (`go:test#1`,
// `go:test#2`, `scripts/x.sh#1`, `ctl:scripts/x.sh#1`) is painted red alone and
// the gate must go red.
//
// WHY A FILE IS CREATED FOR EVERY NODE. A required gate that is not phony is
// silenced by a file of its name, and so is a non-phony node one hop along. So
// for every target make reports considering during the REAL green run, a file of
// that name is created and the gate run again: if it recorded LESS, that file
// silences it. Then all of them at once.
//
// WHY VERIFY MUST COVER. `verify` can reach a gate and still not run its check —
// a variable passed down that the gate skips under — so verify's own record must
// contain everything each non-CI-only gate records on its own.
//
// THE CEILING, STATED ONCE (FR-11.5): a toolchain reached by absolute path, a
// recipe that erases its environment with `env -i`, and a `go build` with no `-o`
// escape the stubs and are not judged; a passing run says so.

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
// is real; adding a checker that is neither a script nor `go run` nor built by
// `go build -o` means adding it here.
var tools = []string{"go", "gofmt", "a2dh", "golangci-lint", "govulncheck", "shasum", "sha256sum"}

// interpreters answer for their `scripts/` argument — a script's own shebang
// brings it here through `env` — and refuse anything else, so `python3 -m x` is
// UNJUDGEABLE rather than unseen.
var interpreters = []string{"sh", "bash", "python3", "expect"}

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

// oracleTree is a scratch directory: `tree/` is a git repository the Makefile
// under judgment lives in, `bin/` the stubs, `status/` the painted keys, and
// `reached` the log every stub appends its invocation to.
type oracleTree struct {
	root    string
	tree    string
	def     int             // the status every unpainted key answers with
	targets map[string]bool // the rules make -pn knows
	greens  map[string]greenRun
}

type greenRun struct {
	code  int
	reach []string // invocations, `key#n`
	nodes []string // every target make reported considering
}

const (
	envLog     = "ORACLE_LOG"
	envStatus  = "ORACLE_STATUS"
	envDefault = "ORACLE_DEFAULT"
	envRoot    = "ORACLE_ROOT"
)

// answerStub records `key#n` — n counting this key's invocations so far in the
// run — and exits with the status painted for the invocation, else for the key,
// else the default. A `--self-test` before `--` makes the key a control.
const answerStub = `#!/bin/sh
key="$1"; shift
for a in "$@"; do
  case "$a" in --) break;; --self-test|-self-test) key="ctl:$key"; break;; esac
done
n=$(awk -v k="$key#" 'index($0,k)==1{c++} END{print c+0}' "$ORACLE_LOG")
inv="$key#$((n+1))"
printf '%s\n' "$inv" >> "$ORACLE_LOG"
for k in "$inv" "$key"; do
  f="$ORACLE_STATUS/$(printf '%s' "$k" | tr / _)"
  [ -f "$f" ] && exit "$(cat "$f")"
done
exit "$ORACLE_DEFAULT"
`

// goStub answers by sub-command — `go:test`, `go:mod:tidy`, `go:run:<package>`
// — honours treelock's `--` contract, and when it "builds" to `-o` writes a
// recording stub there, keyed `built:<path>`.
const goStub = `#!/bin/sh
sub="$1"; key="go:$sub"
case "$sub" in mod|run) key="go:$sub:$2";; esac
"%[1]s" "$key" "$@" || exit $?
case "$sub" in
run)
  while [ $# -gt 0 ]; do
    if [ "$1" = -- ]; then shift; exec "$@"; fi
    shift
  done;;
build)
  while [ $# -gt 0 ]; do
    if [ "$1" = -o ]; then
      out="$2"; mkdir -p "$(dirname "$out")"
      printf '#!/bin/sh\nexec "%[1]s" "built:%%s" "$@"\n' "$out" > "$out" && chmod +x "$out"
      break
    fi
    shift
  done;;
esac
exit 0
`

// interpreterStub finds its scripts/ argument, resolves it against the working
// directory so `cd scripts && ./x.sh` and `$(CURDIR)/scripts/x.sh` are the same
// key, and answers for it.
const interpreterStub = `#!/bin/sh
for a in "$@"; do
  case "$a" in -*) continue;; esac
  case "$a" in /*) p="$a";; *) p="$PWD/$a";; esac
  d=$(cd "$(dirname "$p")" 2>/dev/null && pwd -P) || continue
  p="$d/$(basename "$p")"
  case "$p" in "$ORACLE_ROOT"/scripts/*) exec "%s" "${p#"$ORACLE_ROOT"/}" "$@";; esac
done
echo "oracle: %s run without a scripts/ argument is not judged" >&2
exit 3
`

var envShebang = regexp.MustCompile(`^#!/usr/bin/env (sh|bash|python3|expect)\b`)

// newOracleTree judges the Makefile in `tree`, a git repository the caller laid
// out; the oracle owns everything beside it.
func newOracleTree(t reporter, tree string) *oracleTree {
	t.Helper()
	requireMake(t)
	root, err := filepath.EvalSymlinks(filepath.Dir(tree))
	if err != nil {
		t.Fatalf("COULD NOT RUN — %v", err)
	}
	o := &oracleTree{root: root, tree: filepath.Join(root, filepath.Base(tree)), greens: map[string]greenRun{}}
	w := func(name, body string) {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("COULD NOT RUN — laying out the oracle: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatalf("COULD NOT RUN — writing %s: %v", name, err)
		}
	}
	answer := filepath.Join(root, "bin", "oracle-answer")
	w("bin/oracle-answer", answerStub)
	w("bin/go", fmt.Sprintf(goStub, answer))
	for _, tool := range tools { // bounded by the tool list (P10-02)
		if tool != "go" {
			w("bin/"+tool, fmt.Sprintf("#!/bin/sh\nexec \"%s\" %s \"$@\"\n", answer, tool))
		}
	}
	for _, in := range interpreters { // bounded by the interpreter list (P10-02)
		w("bin/"+in, fmt.Sprintf(interpreterStub, answer, in))
	}
	// EVERY EXECUTABLE SCRIPT MUST REACH THE STUBS THROUGH ITS SHEBANG. `#!/bin/sh`
	// bypasses PATH: the script would run for real and record nothing, and that
	// is the quiet direction, so it is refused here. A file without an executable
	// bit is data.
	_ = filepath.WalkDir(filepath.Join(o.tree, "scripts"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, e := d.Info(); e != nil || info.Mode()&0o111 == 0 {
			return nil
		}
		head := make([]byte, 64)
		f, e := os.Open(path)
		if e != nil {
			return nil
		}
		n, _ := f.Read(head)
		f.Close()
		if !envShebang.Match(head[:n]) {
			t.Errorf("COULD NOT JUDGE %s — its first line is not `#!/usr/bin/env sh|bash|python3|expect`, so the "+
				"kernel runs the interpreter by absolute path, the oracle's stub is never reached, and the "+
				"script would run for real and record nothing. Use an env shebang.", strings.TrimPrefix(path, o.tree+"/"))
		}
		return nil
	})
	if err := os.MkdirAll(filepath.Join(root, "status"), 0o755); err != nil {
		t.Fatalf("COULD NOT RUN — %v", err)
	}
	o.paint(0)
	o.targets = o.rules(t)
	return o
}

// ---- painting and running ----------------------------------------------------------------

// paint sets the default every key answers with and paints the named keys —
// invocations (`go:test#2`) or whole keys (`go:test`) — red.
func (o *oracleTree) paint(def int, red ...string) {
	o.def = def
	status := filepath.Join(o.root, "status")
	_ = os.RemoveAll(status)
	_ = os.MkdirAll(status, 0o755)
	for _, k := range red { // bounded by the painted keys (P10-02)
		_ = os.WriteFile(filepath.Join(status, strings.ReplaceAll(k, "/", "_")), []byte("3\n"), 0o644)
	}
}

var considered = regexp.MustCompile("Considering target file [`'](.+)'\\.")

// run executes `make <gate>` in the tree with the stub PATH in front and returns
// the status, every invocation the stubs recorded, and every target make
// reported considering.
func (o *oracleTree) run(gate string) greenRun {
	log := filepath.Join(o.root, "reached")
	_ = os.WriteFile(log, nil, 0o644)
	cmd := exec.Command("make", "-s", "--debug=v", gate)
	cmd.Dir = o.tree
	cmd.Env = o.env()
	out, err := cmd.Output()
	g := greenRun{}
	if err != nil {
		g.code = -1
		if ex, ok := err.(*exec.ExitError); ok {
			g.code = ex.ExitCode()
		}
	}
	raw, _ := os.ReadFile(log)
	for _, k := range strings.Split(string(raw), "\n") { // bounded by the record (P10-02)
		if k != "" && !contains(g.reach, k) {
			g.reach = append(g.reach, k)
		}
	}
	sort.Strings(g.reach)
	for _, m := range considered.FindAllStringSubmatch(string(out), -1) { // bounded by the walk (P10-02)
		if !contains(g.nodes, m[1]) {
			g.nodes = append(g.nodes, m[1])
		}
	}
	return g
}

// green is the gate's run with every key green, cached, and reported ONCE as
// UNJUDGEABLE if it is red.
func (o *oracleTree) green(t reporter, gate string) greenRun {
	t.Helper()
	if g, ok := o.greens[gate]; ok {
		return g
	}
	o.paint(0)
	g := o.run(gate)
	o.greens[gate] = g
	if g.code != 0 {
		t.Errorf("UNJUDGEABLE — `make %s` exits %d with EVERY stub GREEN. Something it reaches is red for a "+
			"reason that is not a check (a real tool the tree cannot satisfy, an interpreter run without a "+
			"script, a binary nothing built). COULD-NOT-RUN, not a pass: make the recipe judgeable, or "+
			"declare the gate.", gate, g.code)
	}
	return g
}

// env is the parent environment minus everything a parent make hands down, plus
// the stub PATH and the stubs' own variables.
func (o *oracleTree) env() []string {
	var out []string
	for _, kv := range os.Environ() { // bounded by the environment (P10-02)
		switch strings.SplitN(kv, "=", 2)[0] {
		case "MAKEFLAGS", "GNUMAKEFLAGS", "MFLAGS", "MAKELEVEL", "MAKEFILES", "MAKE", "MAKEOVERRIDES",
			"MAKE_TERMOUT", "MAKE_TERMERR", "PATH", envLog, envStatus, envDefault, envRoot:
			continue
		}
		out = append(out, kv)
	}
	return append(out,
		"PATH="+filepath.Join(o.root, "bin")+":"+os.Getenv("PATH"),
		envLog+"="+filepath.Join(o.root, "reached"),
		envStatus+"="+filepath.Join(o.root, "status"),
		envRoot+"="+o.tree,
		fmt.Sprintf("%s=%d", envDefault, o.def))
}

// rules is the set of names `make -pn` lists as rules — only whether make knows
// the name.
func (o *oracleTree) rules(t reporter) map[string]bool {
	t.Helper()
	cmd := exec.Command("make", "-pn", "-f", "Makefile")
	cmd.Dir = o.tree
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

// keyOf strips the ordinal: `go:test#2` → `go:test`.
func keyOf(inv string) string {
	if i := strings.LastIndex(inv, "#"); i >= 0 {
		return inv[:i]
	}
	return inv
}

func keysOf(invs []string) []string {
	var out []string
	for _, inv := range invs { // bounded by the reach (P10-02)
		if k := keyOf(inv); !contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// ---- the executed assertions -----------------------------------------------------------------

// EVERY REQUIRED GATE IS A RULE MAKE KNOWS, AND NO FILE NAMED AFTER ANY NODE ON
// ITS WALK — ONE AT A TIME, THEN ALL AT ONCE — MAKES IT RECORD LESS (J1–J3, N1,
// N2, O6, P4, P5).
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
		var creatable []string
		for _, node := range base.nodes { // bounded by make's walk (P10-02)
			if _, err := os.Stat(filepath.Join(o.tree, node)); err != nil {
				creatable = append(creatable, node)
			}
		}
		sets := [][]string{}
		for _, node := range creatable { // bounded by the nodes (P10-02)
			sets = append(sets, []string{node})
		}
		if len(creatable) > 1 {
			sets = append(sets, creatable)
		}
		for _, set := range sets { // bounded by the node sets (P10-02)
			for _, node := range set { // bounded by the set (P10-02)
				path := filepath.Join(o.tree, node)
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				_ = os.WriteFile(path, nil, 0o644)
			}
			o.paint(0)
			g := o.run(g)
			for _, node := range set { // bounded by the set (P10-02)
				_ = os.Remove(filepath.Join(o.tree, node))
			}
			if g.code != 0 {
				continue // the file broke it, which is not silence
			}
			var lost []string
			for _, k := range base.reach { // bounded by the reach (P10-02)
				if !contains(g.reach, k) {
					lost = append(lost, k)
				}
			}
			if len(lost) > 0 {
				t.Errorf("a file named %v makes `make %s` exit 0 without running %v: make says \"is up to date\" "+
					"and skips it. Add it to .PHONY.", set, g, lost)
			}
		}
	}
	if checked == 0 {
		t.Fatalf("COULD NOT RUN — no required gate is a make rule")
	}
}

// EVERY REQUIRED GATE GOES GREEN WHEN EVERYTHING IS, AND RED FOR EACH INVOCATION
// IT RECORDED, PAINTED ALONE (H1–H4, M1–M7, O1–O9, P1–P3, P7, P11).
func assertEveryRequiredGateCanFail(t reporter, o *oracleTree, required []string) {
	t.Helper()
	var checked, judged int
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
			t.Errorf("%s is a REQUIRED gate and ran no checker and no toolchain command in this tree: nothing it "+
				"does can fail — or a predicate on the tree skipped its check, which is the same thing.", g)
			continue
		}
		for _, inv := range base.reach { // bounded by the reach (P10-02)
			judged++
			o.paint(0, inv)
			if o.run(g).code == 0 {
				t.Errorf("%s is a REQUIRED gate and `make %s` exits 0 with ONLY %s red.\nThe gate ran it and does "+
					"not fail when it does — its status is discarded somewhere on that path, and make has already "+
					"said so. A diagnostic under `|| true` is refused for the same reason: move it out of the gate "+
					"or let it fail.", g, g, inv)
			}
		}
	}
	o.paint(0)
	if checked == 0 {
		t.Fatalf("COULD NOT RUN — no required gate is a make rule")
	}
	t.Logf("judged %d gates, %d invocations. NOT judged: a toolchain reached by absolute path, a recipe under "+
		"`env -i`, a binary `go build` writes without `-o`, and any command not in %v", checked, judged, tools)
}

// `make verify` REACHES THE GATES, RECORDS EVERYTHING EACH NON-CI-ONLY GATE
// RECORDS ON ITS OWN, AND GOES RED FOR EACH INVOCATION PAINTED ALONE (H5, M7,
// E15, P6).
func assertVerifyCanFail(t reporter, o *oracleTree, required []string, ciOnlyRows map[string]string) {
	t.Helper()
	if !o.targets["verify"] {
		t.Fatalf("COULD NOT RUN — no verify rule")
	}
	base := o.green(t, "verify")
	if base.code != 0 {
		return
	}
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
	verifyKeys := keysOf(base.reach)
	for _, g := range required { // bounded by the gate list (P10-02)
		if _, ci := ciOnlyRows[g]; ci || !o.targets[g] {
			continue
		}
		own := o.green(t, g)
		if own.code != 0 {
			continue
		}
		for _, k := range keysOf(own.reach) { // bounded by the reach (P10-02)
			if !contains(verifyKeys, k) {
				t.Errorf("`make %s` runs %s and `make verify` never does: verify reaches the gate under a variable "+
					"or flag the gate skips its check under. What verify runs is what ships.", g, k)
			}
		}
	}
	for _, inv := range base.reach { // bounded by the reach (P10-02)
		o.paint(0, inv)
		if o.run("verify").code == 0 {
			t.Errorf("`make verify` exits 0 with ONLY %s red. The entry point discards that failure — "+
				"release.yml reads this exit, so a red release would ship.", inv)
		}
	}
	o.paint(0)
}

// EVERY CONTROL IS REACHED (K1–K4, A5–A10, P8): only the CONTROL red, every
// carrier tried. A checker is every `scripts/` or `go run ./tools/…` key a
// required gate recorded that is not itself a control; its control is `ctl:<key>`
// or, for a script, a sibling `<key>_test.sh`; its carriers are the required
// gates that recorded that control.
func assertEveryControlIsReached(t reporter, o *oracleTree, required []string, uncontrolledRows map[string]string) {
	t.Helper()
	var invoked []string
	for _, g := range required { // bounded by the gate list (P10-02)
		for _, k := range keysOf(o.green(t, g).reach) { // bounded by the reach (P10-02)
			isScript := strings.HasPrefix(k, "scripts/") && !strings.HasSuffix(k, "_test.sh")
			isTool := strings.HasPrefix(k, "go:run:./tools/") || strings.HasPrefix(k, "go:run:./scripts/")
			if (isScript || isTool) && !contains(invoked, k) {
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
		ctl := "ctl:" + k
		sib := ""
		if strings.HasPrefix(k, "scripts/") {
			sib = strings.TrimSuffix(k, ".sh") + "_test.sh"
		}
		var carriers []string
		for _, g := range required { // bounded by the gate list (P10-02)
			keys := keysOf(o.green(t, g).reach)
			if contains(keys, ctl) || (sib != "" && contains(keys, sib)) {
				carriers = append(carriers, g)
			}
		}
		if len(carriers) == 0 {
			t.Errorf("%s is run by a required gate and no required gate runs its control — `%s --self-test`"+
				"%s.\nAdd the control to gate-controls, or add a row to `uncontrolled` saying why this checker "+
				"is trusted without evidence.", k, k, map[bool]string{true: " or a sibling `" + sib + "`", false: ""}[sib != ""])
			continue
		}
		o.paint(0, ctl, sib)
		var reached bool
		for _, g := range carriers { // bounded by the carrier list (P10-02)
			if o.run(g).code != 0 {
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
	if checked == 0 {
		t.Fatalf("COULD NOT RUN — no checker is run by a required gate")
	}
}

// ---- the production gates ----------------------------------------------------------------------

// repoTree is a shared clone of the repository with the working tree copied over
// it: the tree as it is, with a real `.git`, so every predicate a recipe puts to
// the tree answers as it does here and in CI.
func repoTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "clone", "--shared", "--quiet", repo, tree).CombinedOutput(); err != nil {
		t.Fatalf("COULD NOT RUN — cloning the repository: %v\n%s", err, out)
	}
	sync := exec.Command("rsync", "-a", "--delete", "--exclude", ".git", "--exclude", "dist", "--exclude", ".local", repo+"/", tree+"/")
	if out, err := sync.CombinedOutput(); err != nil {
		t.Fatalf("COULD NOT RUN — copying the working tree: %v\n%s", err, out)
	}
	return tree
}

func realOracle(t *testing.T) (*oracleTree, []string) {
	t.Helper()
	return newOracleTree(t, repoTree(t)), parseRequired(read(t, "../../06_docs/required-gates.txt"))
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
	assertVerifyCanFail(t, o, required, ciOnly)
}

func TestEveryControlIsReached(t *testing.T) {
	o, required := realOracle(t)
	assertEveryControlIsReached(t, o, required, uncontrolled)
}
