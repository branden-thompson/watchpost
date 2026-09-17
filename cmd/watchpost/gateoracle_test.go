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
// WHY THE TREE, AS CI HAS IT. A scratch tree that is empty answers every question
// a recipe asks of the tree the opposite way the repository does: `git diff
// --quiet && exit 0` runs the checker in scratch and skips it in CI; `find |
// xargs gofmt` records gofmt on CI's xargs and nothing on BSD's; `gate: go.mod`
// is silenced by a file only where go.mod exists. And a tree that is the
// developer's answers differently from CI's: HEAD attached where CI's is
// detached, tags where a depth-1 checkout has none, a git-ignored file CI never
// sees. So the scratch is a shared clone of the repository, HEAD detached at the
// commit, tags deleted, with the working tree's tracked and unignored files
// copied over it; and the stubs interpose by PATH ALONE — a script is answered
// through its `#!/usr/bin/env sh` shebang, and not one byte of the tree is
// rewritten. The `go build` stub writes a recording stub at `-o`, so a compiled
// checker is a key like any other.
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
	targets map[string]bool // the rules make -pn knows, and whether make calls each phony
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
until mkdir "$ORACLE_LOG.lock" 2>/dev/null; do sleep 0.01; done
n=$(awk -v k="$key#" 'index($0,k)==1{c++} END{print c+0}' "$ORACLE_LOG")
inv="$key#$((n+1))"
printf '%s\n' "$inv" >> "$ORACLE_LOG"
rmdir "$ORACLE_LOG.lock"
for k in "$inv" "$key"; do
  f="$ORACLE_STATUS/$(printf '%s' "$k" | sed 's|/|%2F|g')"
  [ -f "$f" ] && exit "$(cat "$f")"
done
exit "$ORACLE_DEFAULT"
`

// goStub answers by sub-command — `go:test`, `go:mod:tidy`, `go:run:<package>`
// where the package is the first argument shaped like one (`./x`, `../x`, `/x`,
// `x.go`, `.`, `<module>/x` normalised to `./x`, or `host.tld/x`), so neither
// `-tags foo` nor `-o out/` is the key — honours treelock's `--` contract, and
// when it "builds" to `-o` writes a recording stub there, keyed `built:<path>`;
// `-o dir/` lands the stub at `dir/<package basename>`.
const goStub = `#!/bin/sh
sub="$1"; key="go:$sub"; shift
pkg=""
mod=$(awk '/^module /{print $2; exit}' "$ORACLE_ROOT/go.mod" 2>/dev/null)
for a in "$@"; do
  case "$a" in
    --) break;;
    -*) continue;;
    .|./*|../*|/*|*.go) pkg="$a"; break;;
  esac
  [ -n "$mod" ] && case "$a" in "$mod"/*) pkg="./${a#"$mod"/}"; break;; esac
  case "${a%%/*}" in *.*) pkg="$a"; break;; esac
done
case "$sub" in mod) key="go:mod:$1";; run) key="go:run:$pkg";; esac
"%[1]s" "$key" "$sub" "$@" || exit $?
case "$sub" in
run)
  while [ $# -gt 0 ]; do
    if [ "$1" = -- ]; then shift; exec "$@"; fi
    shift
  done;;
build)
  while [ $# -gt 0 ]; do
    if [ "$1" = -o ]; then
      out="$2"
      case "$out" in */) out="$out$(basename "$pkg")";; esac
      [ -d "$out" ] && out="$out/$(basename "$pkg")"
      mkdir -p "$(dirname "$out")"
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
// key, and answers for it. `sh -c '…'` runs the REAL shell — the script inside
// the string records through its own shebang.
const interpreterStub = `#!/bin/sh
if [ "$1" = -c ] && [ -n "%[3]s" ]; then exec "%[3]s" "$@"; fi
for a in "$@"; do
  case "$a" in -*) continue;; esac
  case "$a" in /*) p="$a";; *) p="$PWD/$a";; esac
  d=$(cd "$(dirname "$p")" 2>/dev/null && pwd -P) || continue
  p="$d/$(basename "$p")"
  case "$p" in "$ORACLE_ROOT"/scripts/*) exec "%[1]s" "${p#"$ORACLE_ROOT"/}" "$@";; esac
done
echo "oracle: %[2]s run without a scripts/ argument is not judged" >&2
exit 3
`

// envShebang is DERIVED from the interpreter list and anchored: `python3.12` is
// a real interpreter the oracle does not stub.
var envShebang = regexp.MustCompile(`^#!/usr/bin/env (` + strings.Join(interpreters, "|") + `)[ \t\r]*$`)

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
		real := ""
		if in == "sh" || in == "bash" {
			real, _ = exec.LookPath(in) // resolved before bin/ is on PATH
		}
		w("bin/"+in, fmt.Sprintf(interpreterStub, answer, in, real))
	}
	// EVERY EXECUTABLE SCRIPT MUST REACH THE STUBS THROUGH ITS SHEBANG. `#!/bin/sh`
	// bypasses PATH: the script would run for real and record nothing, and that
	// is the quiet direction, so it is refused here. A file without an executable
	// bit is data.
	_ = filepath.WalkDir(filepath.Join(o.tree, "scripts"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, e := os.Stat(path); e != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			return nil // a directory (or a link to one) is not a script; a file without an executable bit is data
		}
		head := make([]byte, 128)
		f, e := os.Open(path)
		if e != nil {
			return nil
		}
		n, _ := f.Read(head)
		f.Close()
		first, _, _ := strings.Cut(string(head[:n]), "\n")
		if !envShebang.MatchString(first) {
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
		_ = os.WriteFile(filepath.Join(status, strings.ReplaceAll(k, "/", "%2F")), []byte("3\n"), 0o644)
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

// rules is every name `make -pn` lists as a rule, true where make reports
// `#  Phony target` for it.
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
	cur := ""
	for _, line := range strings.Split(string(out), "\n") { // bounded by the database (P10-02)
		switch {
		case line == "# Not a target:":
			notATarget = true
		case strings.HasPrefix(line, "#  Phony target"):
			if cur != "" {
				rules[cur] = true
			}
		case strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" || strings.HasPrefix(line, "\t"):
		default:
			cur = ""
			if m := ruleLine.FindStringSubmatch(line); m != nil && !notATarget && !strings.HasPrefix(m[1], ".") {
				cur = m[1]
				rules[cur] = false
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
		if _, known := o.targets[g]; !known {
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
		absent := func(node string) bool {
			_, err := os.Stat(filepath.Join(o.tree, node))
			return err != nil
		}
		var creatable []string
		for _, node := range base.nodes { // bounded by make's walk (P10-02)
			if absent(node) {
				creatable = append(creatable, node)
			}
		}
		sets := [][]string{}
		for _, node := range creatable { // bounded by the nodes (P10-02)
			sets = append(sets, []string{node})
		}
		// ALL AT ONCE, PLUS EVERY NON-PHONY RULE MAKE KNOWS: a `$(MAKE)` hop whose
		// output is hidden keeps its node out of the observed walk, and a file of
		// that name still silences it.
		all := append([]string{}, creatable...)
		for name, phony := range o.targets { // bounded by the database (P10-02)
			if !phony && absent(name) && !contains(all, name) {
				all = append(all, name)
			}
		}
		sort.Strings(all)
		if len(all) > 1 {
			sets = append(sets, all)
		}
		for _, set := range sets { // bounded by the node sets (P10-02)
			for _, node := range set { // bounded by the set (P10-02)
				path := filepath.Join(o.tree, node)
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				_ = os.WriteFile(path, nil, 0o644)
			}
			o.paint(0)
			touched := o.run(g)
			for _, node := range set { // bounded by the set (P10-02)
				_ = os.Remove(filepath.Join(o.tree, node))
			}
			if touched.code != 0 {
				continue // the file broke it, which is not silence
			}
			var lost []string
			for _, k := range base.reach { // bounded by the reach (P10-02)
				if !contains(touched.reach, k) {
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
	ceiling(t, fmt.Sprintf("audited %d gates for silence by a file", checked))
}

// EVERY REQUIRED GATE GOES GREEN WHEN EVERYTHING IS, AND RED FOR EACH INVOCATION
// IT RECORDED, PAINTED ALONE (H1–H4, M1–M7, O1–O9, P1–P3, P7, P11).
func assertEveryRequiredGateCanFail(t reporter, o *oracleTree, required []string) {
	t.Helper()
	var checked, judged int
	for _, g := range required { // bounded by the gate list (P10-02)
		if _, known := o.targets[g]; !known {
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
	ceiling(t, fmt.Sprintf("judged %d gates, %d invocations", checked, judged))
}

// ceiling is what every passing run says it did not judge (FR-11.5).
func ceiling(t reporter, did string) {
	t.Helper()
	t.Logf("%s. NOT judged: a toolchain reached by absolute path, a recipe under `env -i` or with ORACLE_* "+
		"reassigned, a discard inside a script, a binary `go build` writes without `-o`, and any command not "+
		"in %v. Git metadata is judged as CI has it: HEAD detached at the commit, no tags.", did, tools)
}

// `make verify` REACHES THE GATES, RECORDS EVERYTHING EACH NON-CI-ONLY GATE
// RECORDS ON ITS OWN, AND GOES RED FOR EACH INVOCATION PAINTED ALONE (H5, M7,
// E15, P6).
func assertVerifyCanFail(t reporter, o *oracleTree, required []string, ciOnlyRows map[string]string) {
	t.Helper()
	if _, known := o.targets["verify"]; !known {
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
	// BY COUNT, NOT BY KEY: a variable passed down that a gate skips ONE of two
	// `go test` calls under leaves the key present and the check gone.
	verifyCounts, need := countsOf(base.reach), map[string]int{}
	for _, g := range required { // bounded by the gate list (P10-02)
		if _, known := o.targets[g]; !known {
			continue
		}
		if _, ci := ciOnlyRows[g]; ci {
			continue
		}
		own := o.green(t, g)
		if own.code != 0 {
			continue
		}
		for k, n := range countsOf(own.reach) { // bounded by the reach (P10-02)
			need[k] += n
		}
	}
	for _, k := range sortedKeys(need) { // bounded by the key set (P10-02)
		if verifyCounts[k] < need[k] {
			t.Errorf("`make verify` runs %s %d time(s) and the required gates it carries run it %d on their own: "+
				"verify reaches a gate under a variable or flag the gate skips a check under. What verify runs "+
				"is what ships. (A prerequisite two gates share runs once under verify; give each its own.)",
				k, verifyCounts[k], need[k])
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
	ceiling(t, fmt.Sprintf("verify judged for %d invocations", len(base.reach)))
}

// countsOf is how many times each key was invoked.
func countsOf(invs []string) map[string]int {
	out := map[string]int{}
	for _, inv := range invs { // bounded by the reach (P10-02)
		out[keyOf(inv)]++
	}
	return out
}

func sortedKeys(m map[string]int) []string {
	var out []string
	for k := range m { // bounded by the map (P10-02)
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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
			isTool := strings.HasPrefix(k, "go:run:./tools/") || strings.HasPrefix(k, "go:run:./scripts/") || strings.HasPrefix(k, "built:")
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
				"%s. (`go test` of a tool is not a control: a tool's tests prove the tool, the self-test proves "+
				"it can FAIL in this tree.)\nAdd the control to gate-controls, or add a row to `uncontrolled` "+
				"saying why this checker is trusted without evidence.", k, k,
				map[bool]string{true: " or a sibling `" + sib + "`", false: ""}[sib != ""])
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
	ceiling(t, fmt.Sprintf("%d checkers' controls judged", checked))
}

// ---- the production gates ----------------------------------------------------------------------

// cloneForOracle is the tree AS CI HAS IT: a shared clone of source at its
// commit, HEAD detached, no tags, with source's tracked and unignored files laid
// over it — never a git-ignored file, never `dist/`. It lives under root/tree.
func cloneForOracle(t reporter, source, root string) string {
	t.Helper()
	tree := filepath.Join(root, "tree")
	git := func(dir string, args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("COULD NOT RUN — git %v in %s: %v\n%s", args, dir, err, out)
		}
		return string(out)
	}
	git(source, "clone", "--shared", "--quiet", source, tree)
	git(tree, "checkout", "--quiet", "--detach")
	for _, tag := range strings.Fields(git(tree, "tag", "--list")) { // bounded by the tag list (P10-02)
		git(tree, "tag", "-d", tag)
	}
	listed := git(source, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	for _, rel := range strings.Split(listed, "\x00") { // bounded by the file list (P10-02)
		if rel == "" || strings.HasPrefix(rel, "dist/") {
			continue
		}
		src, dst := filepath.Join(source, rel), filepath.Join(tree, rel)
		info, err := os.Lstat(src)
		if err != nil {
			_ = os.Remove(dst) // tracked here, deleted in the working tree
			continue
		}
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		if info.Mode()&os.ModeSymlink != 0 {
			link, _ := os.Readlink(src)
			_ = os.Remove(dst)
			_ = os.Symlink(link, dst)
			continue
		}
		body, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("COULD NOT RUN — copying %s: %v", rel, err)
		}
		_ = os.Remove(dst)
		if err := os.WriteFile(dst, body, info.Mode().Perm()); err != nil {
			t.Fatalf("COULD NOT RUN — writing %s: %v", rel, err)
		}
	}
	return tree
}

func realOracle(t *testing.T) (*oracleTree, []string) {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return newOracleTree(t, cloneForOracle(t, repo, t.TempDir())), parseRequired(read(t, "../../06_docs/required-gates.txt"))
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
