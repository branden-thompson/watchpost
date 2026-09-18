package gateoracle

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

var makeVersionLine = regexp.MustCompile(`GNU Make (\d+)\.(\d+)`)

// requireMake refuses, by name, a make older than the one CI runs.
func requireMake(t Reporter) {
	t.Helper()
	out, err := exec.Command("make", "--version").Output()
	if err != nil {
		t.Fatalf("COULD NOT RUN — `make --version` failed: %v", err)
	}
	m := makeVersionLine.FindStringSubmatch(string(out))
	if m == nil {
		t.Fatalf("COULD NOT RUN — `make` is not GNU make: %q", strings.SplitN(string(out), "\n", 2)[0])
	}
	major, _ := strconv.Atoi(m[1]) // the regex admits digits only
	minor, _ := strconv.Atoi(m[2])
	wantMajor, wantMinor := 3, 82
	if major < wantMajor || (major == wantMajor && minor < wantMinor) {
		t.Fatalf("COULD NOT RUN — GNU Make %s.%s is on PATH and the oracle needs %d.%d or newer, the make CI "+
			"runs. On macOS: `brew install make`, then put /opt/homebrew/opt/make/libexec/gnubin at the "+
			"front of PATH so `make` is 4.x.", m[1], m[2], wantMajor, wantMinor)
	}
}

// Oracle is one scratch: `tree/` the repository as CI has it, `bin/` the stubs
// (symlinks to this test binary), `status/` the painted keys, `reached` the
// record, and the recorded green run of every gate asked about.
type Oracle struct {
	root        string
	tree        string
	rules       map[string]bool // every rule make -pn lists, true where make calls it phony
	parseCounts map[string]int  // invocations the Makefile itself makes at parse time, per key
	greens      map[string]run
}

// run is what one `make <gate>` did.
type run struct {
	code  int
	reach []string // invocations recorded, sorted
	nodes []string // every target make reported considering, in order
	reads int      // how many times a makefile was read — each read re-runs $(shell)
}

// envShebang is DERIVED from the interpreter list and anchored: `python3.12` is
// a real interpreter the oracle does not stub.
func envShebang() *regexp.Regexp {
	return regexp.MustCompile(`^#!/usr/bin/env (` + strings.Join(interpreters(), "|") + `)[ \t\r]*$`)
}

// New judges the Makefile in tree, a directory CloneForOracle laid out; the
// oracle owns everything beside it.
func New(t Reporter, tree string) *Oracle {
	t.Helper()
	requireMake(t)
	root, err := filepath.EvalSymlinks(filepath.Dir(tree))
	if err != nil {
		t.Fatalf("COULD NOT RUN — %v", err)
	}
	o := &Oracle{root: root, tree: filepath.Join(root, filepath.Base(tree)), greens: map[string]run{}}
	for _, dir := range []string{"bin", "status"} { // bounded by the layout (P10-02)
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("COULD NOT RUN — %v", err)
		}
	}
	exe := buildStub(t, filepath.Join(root, "bin", "oraclestub"))
	for _, name := range append(tools(), interpreters()...) { // bounded by the stub list (P10-02)
		if err := os.Symlink(exe, filepath.Join(root, "bin", name)); err != nil {
			t.Fatalf("COULD NOT RUN — linking the %s stub: %v", name, err)
		}
	}
	o.refuseScriptsThatBypassPath(t)
	o.paint()
	o.rules, o.parseCounts = o.database(t)
	return o
}

// tracked is every file git lists in the tree.
func (o *Oracle) tracked(t Reporter) []string {
	t.Helper()
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = o.tree
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("COULD NOT RUN — git ls-files in the scratch: %v", err)
	}
	var files []string
	for _, rel := range strings.Split(string(out), "\x00") { // bounded by the file list (P10-02)
		if rel != "" {
			files = append(files, rel)
		}
	}
	return files
}

// refuseScriptsThatBypassPath: every tracked EXECUTABLE must reach the stubs
// through an env shebang. `#!/bin/sh` bypasses PATH — the script would run for
// real and record nothing, the quiet direction — so it is refused here. A file
// without an executable bit is data; a directory is not a script.
func (o *Oracle) refuseScriptsThatBypassPath(t Reporter) {
	t.Helper()
	for _, rel := range o.tracked(t) { // bounded by the file list (P10-02)
		path := filepath.Join(o.tree, rel)
		if info, e := os.Stat(path); e != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		body, e := os.ReadFile(path)
		if e != nil {
			continue
		}
		first, _, _ := strings.Cut(string(body), "\n")
		if !strings.HasPrefix(first, "#!") {
			continue // a binary, or a file whose executable bit means nothing to make
		}
		if !envShebang().MatchString(first) {
			t.Errorf("COULD NOT JUDGE %s — its first line is not `#!/usr/bin/env <%s>`, so the kernel runs the "+
				"interpreter by absolute path, the oracle's stub is never reached, and the script would run for "+
				"real and record nothing. Use an env shebang.", rel, strings.Join(interpreters(), "|"))
		}
	}
}

// paint makes every key green except the named ones — invocations (`go:test#2`)
// or whole keys (`go:test`) — which answer red.
func (o *Oracle) paint(red ...string) {
	status := filepath.Join(o.root, "status")
	_ = os.RemoveAll(status)
	_ = os.MkdirAll(status, 0o755)
	for _, k := range red { // bounded by the painted keys (P10-02)
		_ = os.WriteFile(filepath.Join(status, encodeKey(k)), []byte("3\n"), 0o644)
	}
}

var (
	considered = regexp.MustCompile("Considering target file [`'](.+)'\\.")
	reading    = regexp.MustCompile("Reading makefile [`'][^'`]+'")
)

// reset returns the tree to its commit: a stamp a gate touched, a binary it
// built, a file the silence audit created — none of it reaches the next run.
func (o *Oracle) reset() {
	for _, args := range [][]string{{"reset", "-q", "--hard"}, {"clean", "-fdxq"}} { // bounded by the two commands (P10-02)
		cmd := exec.Command("git", args...)
		cmd.Dir = o.tree
		_ = cmd.Run()
	}
}

// runMake executes make with args in the tree as it stands and returns its
// status, the invocations recorded, the targets it reported considering, and
// how many times it read a makefile. Callers reset the tree first, except the
// silence audit, which resets and THEN creates the files it is asking about.
func (o *Oracle) runMake(env []string, args ...string) run {
	log := filepath.Join(o.root, "reached")
	_ = os.WriteFile(log, nil, 0o644)
	_ = os.WriteFile(filepath.Join(o.root, "built"), nil, 0o644)
	cmd := exec.Command("make", args...)
	cmd.Dir = o.tree
	cmd.Env = env
	out, err := cmd.Output()
	r := run{}
	if err != nil {
		r.code = -1
		if ex, ok := err.(*exec.ExitError); ok {
			r.code = ex.ExitCode()
		}
	}
	r.reach = readLog(log)
	for _, m := range considered.FindAllStringSubmatch(string(out), -1) { // bounded by the walk (P10-02)
		if !contains(r.nodes, m[1]) {
			r.nodes = append(r.nodes, m[1])
		}
	}
	r.reads = len(reading.FindAllString(string(out), -1))
	return r
}

// runGate is `make -s --debug=v <gate>` with the Makefile's own parse-time
// invocations subtracted — once per makefile read the run reported, because a
// `$(MAKE)` hop re-parses and a `$(shell go env …)` at the top runs again.
func (o *Oracle) runGate(gate string) run {
	return o.runGateEnv(gate, o.env())
}

func (o *Oracle) runGateEnv(gate string, env []string) run {
	o.reset()
	return o.runGateAsIs(gate, env)
}

// runGateAsIs runs the gate on the tree exactly as it stands.
func (o *Oracle) runGateAsIs(gate string, env []string) run {
	r := o.runMake(env, "-s", "--debug=v", gate)
	reads := r.reads
	if reads < 1 {
		reads = 1
	}
	remaining := map[string]int{}
	for k, n := range o.parseCounts { // bounded by the parse-time record (P10-02)
		remaining[k] = n * reads
	}
	var reach []string
	for _, inv := range r.reach { // bounded by the reach (P10-02)
		if k := keyOf(inv); remaining[k] > 0 {
			remaining[k]--
			continue
		}
		reach = append(reach, inv)
	}
	r.reach = reach
	return r
}

// green is the gate's run with every key green, cached, and reported ONCE as
// UNJUDGEABLE if it is red or if a stub refused something on its path.
func (o *Oracle) green(t Reporter, gate string) run {
	t.Helper()
	if g, ok := o.greens[gate]; ok {
		return g
	}
	o.paint()
	g := o.runGate(gate)
	o.greens[gate] = g
	for _, inv := range g.reach { // bounded by the reach (P10-02)
		if role, ok := strings.CutPrefix(keyOf(inv), "unjudged:"); ok {
			t.Errorf("UNJUDGEABLE — `make %s` ran %s in a way the oracle does not judge (an interpreter with no "+
				"scripts/ file, a role the stub does not play), and the recipe went on. COULD-NOT-RUN, not a "+
				"pass: run a scripts/ file or a `go run` package, or declare the gate.", gate, role)
			g.code = 3
			o.greens[gate] = g
			return g
		}
	}
	if g.code != 0 {
		t.Errorf("UNJUDGEABLE — `make %s` exits %d with EVERY stub GREEN. Something it reaches is red for a "+
			"reason that is not a check (a real tool the tree cannot satisfy, an interpreter run without a "+
			"script, a binary nothing built). COULD-NOT-RUN, not a pass: make the recipe judgeable, or "+
			"declare the gate.", gate, g.code)
	}
	return g
}

// env is the parent environment minus everything a parent make hands down, plus
// the stub PATH, the C locale (the walk reads make's English), and the stubs'
// own variables.
func (o *Oracle) env() []string {
	return o.envWith(filepath.Join(o.root, "bin") + ":" + os.Getenv("PATH"))
}

func (o *Oracle) envWith(path string) []string {
	var out []string
	for _, kv := range os.Environ() { // bounded by the environment (P10-02)
		switch strings.SplitN(kv, "=", 2)[0] {
		case "MAKEFLAGS", "GNUMAKEFLAGS", "MFLAGS", "MAKELEVEL", "MAKEFILES", "MAKE", "MAKEOVERRIDES",
			"MAKE_TERMOUT", "MAKE_TERMERR", "PATH", "LC_ALL", "LANG", "LC_MESSAGES", EnvLog, EnvStatus, EnvRoot, EnvBin, EnvBuilt:
			continue
		}
		out = append(out, kv)
	}
	bin := filepath.Join(o.root, "bin")
	return append(out,
		"PATH="+path,
		"LC_ALL=C",
		EnvLog+"="+filepath.Join(o.root, "reached"),
		EnvStatus+"="+filepath.Join(o.root, "status"),
		EnvRoot+"="+o.tree,
		EnvBin+"="+bin,
		EnvBuilt+"="+filepath.Join(o.root, "built"))
}

// database is `make -pn`'s view: every rule by name, true where make reports
// `#  Phony target`, and the invocations the Makefile makes at parse time.
func (o *Oracle) database(t Reporter) (map[string]bool, map[string]int) {
	t.Helper()
	log := filepath.Join(o.root, "reached")
	_ = os.WriteFile(log, nil, 0o644)
	// A GOAL THAT DOES NOT EXIST: `make -pn` prints the database and then makes
	// the default goal under -n, and a recipe line carrying `$(MAKE)` runs even
	// under -n — through the whole tree if the default goal is verify — and
	// what it recorded would be subtracted from every gate as "parse time".
	cmd := exec.Command("make", "-pn", "-f", "Makefile", "oracle-database-only")
	cmd.Dir = o.tree
	cmd.Env = o.env()
	out, _ := cmd.Output()
	if len(out) == 0 {
		t.Fatalf("COULD NOT RUN — `make -pn` printed nothing; the Makefile does not parse")
	}
	return parseDatabase(string(out)), countsOf(readLog(log))
}

// parseDatabase reads `make -p`'s rules: a name at column 0 followed by `:`
// that is not under `# Not a target:` and not one of make's own targets; true
// where the next comment says `#  Phony target`.
func parseDatabase(out string) map[string]bool {
	rules := map[string]bool{}
	ruleLine := regexp.MustCompile(`^([A-Za-z0-9_./-][^:=#\s]*):(?:[^=]|$)`)
	var notATarget bool
	cur := ""
	for _, line := range strings.Split(out, "\n") { // bounded by the database (P10-02)
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
			if m := ruleLine.FindStringSubmatch(line); m != nil && !notATarget && !specialTarget(m[1]) {
				cur = m[1]
				rules[cur] = false
			}
			notATarget = false
		}
	}
	return rules
}

// withoutTool is an environment whose PATH has every stub but this one and only
// the OS's own directories after it, so `command -v <tool>` fails as it does on
// a machine without the tool.
func (o *Oracle) withoutTool(t Reporter, tool string) []string {
	t.Helper()
	dir := filepath.Join(o.root, "bin-without-"+tool)
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("COULD NOT RUN — %v", err)
		}
		exe := filepath.Join(o.root, "bin", "oraclestub")
		for _, name := range append(tools(), interpreters()...) { // bounded by the stub list (P10-02)
			if name != tool {
				_ = os.Symlink(exe, filepath.Join(dir, name))
			}
		}
	}
	return o.envWith(dir + ":/usr/bin:/bin")
}

// specialTarget: GNU make's own targets, which the database lists as rules and
// which are not nodes a file can silence. Any other dot-name is a node.
func specialTarget(name string) bool {
	return contains([]string{".PHONY", ".SUFFIXES", ".DEFAULT", ".PRECIOUS", ".INTERMEDIATE", ".NOTINTERMEDIATE",
		".SECONDARY", ".SECONDEXPANSION", ".DELETE_ON_ERROR", ".IGNORE", ".LOW_RESOLUTION_TIME", ".SILENT",
		".EXPORT_ALL_VARIABLES", ".NOTPARALLEL", ".ONESHELL", ".POSIX", ".WAIT"}, name)
}

// known says whether make lists a rule by that name.
func (o *Oracle) known(name string) bool {
	_, ok := o.rules[name]
	return ok
}

// absent says whether nothing of that name exists in the tree.
func (o *Oracle) absent(node string) bool {
	_, err := os.Lstat(filepath.Join(o.tree, node))
	return err != nil
}

// buildStub builds tools/gateoracle/stub at path. It is built rather than being
// this test binary re-exec'd because under the race detector every exec of an
// instrumented binary costs ten times more, and make execs the stub thousands
// of times per run; the build cache makes the second build nearly free.
func buildStub(t Reporter, path string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("COULD NOT RUN — no caller information for the stub's source")
	}
	build := exec.Command("go", "build", "-o", path, filepath.Join(filepath.Dir(thisFile), "stub"))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("COULD NOT RUN — building the stub: %v\n%s", err, out)
	}
	return path
}
