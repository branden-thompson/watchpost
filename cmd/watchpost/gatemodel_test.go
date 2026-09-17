package main

// gatemodel_test.go — ONE model of the build system, parsed once.
//
// THE CLASS THIS REPLACES. Fourteen gates each re-parsed the Makefile and the
// workflow with their own regex, and the regexes disagreed: one accepted
// `scripts/x.sh`, its neighbour demanded `./scripts/x.sh`; one saw a step's `if:`
// on the line after `- run:` and not as the step's first key; one stripped
// column-0 comments and kept tab-indented ones. Six of thirteen open findings
// against this layer were that one defect wearing six hats — two regexes for one
// idea, free to disagree, in the file whose purpose is stopping two lists from
// disagreeing.
//
// SO THE SEMANTICS ARE DECIDED HERE AND NOWHERE ELSE. What counts as a live
// command, a gate-shaped target, a silenced step or a build line is answered by
// this file, and the gates are assertions over the answer. A parsing defect is
// found once, and 06_docs/gate-attack-list.md is the list it is found against.
//
// IT PARSES STRINGS, NOT PATHS. Production gates hand it the real files; the
// specimen table in gateattacks_test.go hands it synthetic ones. That is the
// shape tools/authoring's self-test uses, and the reason that detector has never
// been defeated: every attack is a row, and a row is re-run forever.
//
// AND SILENCE IS A VERDICT (FR-11.3). A source the model finds nothing in does
// not yield an empty model that every assertion passes over; it yields
// `couldNotRun`, and the gate that consults it fails with the reason.

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// ---- reporting -----------------------------------------------------------------

// reporter is the slice of testing.T the assertions use, so the specimen table
// can hand them a recorder and read the verdict instead of failing the parent.
type reporter interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// recorder captures what an assertion would have reported. Fatalf aborts the
// assertion the way testing.T would, by unwinding to the harness.
type recorder struct{ errs []string }

type fatalSentinel struct{}

func (r *recorder) Helper()                   {}
func (r *recorder) Errorf(f string, a ...any) { r.errs = append(r.errs, fmt.Sprintf(f, a...)) }
func (r *recorder) Fatalf(f string, a ...any) {
	r.errs = append(r.errs, fmt.Sprintf(f, a...))
	panic(fatalSentinel{})
}

// verdictOf runs an assertion against a recorder and says whether it fired.
func verdictOf(fn func(reporter)) (fired bool, said []string) {
	r := &recorder{}
	func() {
		defer func() {
			if x := recover(); x != nil {
				if _, ok := x.(fatalSentinel); !ok {
					panic(x)
				}
			}
		}()
		fn(r)
	}()
	return len(r.errs) > 0, r.errs
}

// ---- the model -------------------------------------------------------------------

// command is one live recipe command, judged PER SEGMENT.
//
// A recipe line is often several shell commands joined by `&&`, `||` and `;`,
// and the facts a gate asks about belong to a segment, not the line: `grep x ||
// true` beside `test $rc -eq 0 || exit 1` does not make the whole recipe
// unfailable, and `$(MAKE) promote-verdicts` beside `./scripts/x.sh` does not
// make the script orchestration. A model that judged the joined line got both
// wrong in the real Makefile.
type command struct {
	text string
	// ignoreExit is a leading `-`: make discards the whole line's status (A8).
	ignoreExit bool
	segs       []segment
}

// segment is one shell command inside a recipe line.
type segment struct {
	text string
	// cannotFail: this segment's failure is discarded — the line is `-`
	// prefixed, or the segment is followed by `|| true` / `|| :`, or preceded by
	// `true ||` (which makes it unreachable), or trailed by `; true` (A6, A7, A25).
	cannotFail bool
	// orchestration: a recursive `$(MAKE)`, which delegates rather than checks.
	orchestration bool
}

// target is one Makefile rule, with prerequisites ACCUMULATED across every rule
// that names it — make merges them, so a parser that reads the first rule reads
// a list make does not run (A17).
type target struct {
	name string
	deps []string
	cmds []command
}

// ciStep is one workflow step. Keys are the step's own keys wherever they sit —
// on the dash line or under it — because `- if: false` and a following
// `if: false` are the same step (A20, A21).
type ciStep struct {
	keys map[string]string
}

// ciJob is one workflow job: its own keys (a job-level `if:` silences every
// step at once, A22) and its steps.
type ciJob struct {
	keys  map[string]string
	steps []ciStep
}

// buildModel is the parsed build system.
type buildModel struct {
	targets  map[string]*target
	jobs     map[string]*ciJob
	required []string
	// couldNotRun names a source the model found NOTHING in. A gate that
	// consults a model in this state must fail with the reason rather than pass
	// over an empty set (FR-11.3, A34–A36).
	couldNotRun []string
}

// loadBuildModel is the production entry: the real files, from the repo root.
func loadBuildModel(t *testing.T) *buildModel {
	t.Helper()
	return newBuildModel(
		read(t, "../../Makefile"),
		read(t, "../../.github/workflows/ci.yml"),
		read(t, "../../06_docs/required-gates.txt"),
	)
}

// newBuildModel parses the three sources. Nothing else in the package parses
// them.
func newBuildModel(makefile, workflow, requiredList string) *buildModel {
	m := &buildModel{
		targets:  parseMakefile(makefile),
		jobs:     parseWorkflow(workflow),
		required: parseRequired(requiredList),
	}
	if len(m.targets) == 0 {
		m.couldNotRun = append(m.couldNotRun, "the Makefile parsed to zero targets")
	}
	if len(m.jobs) == 0 {
		m.couldNotRun = append(m.couldNotRun, "the workflow parsed to zero jobs")
	}
	if len(m.required) == 0 {
		m.couldNotRun = append(m.couldNotRun, "required-gates.txt lists nothing")
	}
	return m
}

// mustRun fails the calling gate if the model could not run. Every gate calls
// it first: a model with nothing in it must never let an assertion pass by
// having nothing to check.
func (m *buildModel) mustRun(t reporter) {
	t.Helper()
	for _, why := range m.couldNotRun { // bounded by the source list (P10-02)
		t.Fatalf("COULD NOT RUN — %s. A gate over an empty model would pass by asking nothing; "+
			"that is the silence FR-11.3 forbids, reported instead of swallowed.", why)
	}
}

// ---- the Makefile ------------------------------------------------------------------

var makeTargetLine = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9._-]*)\s*:(?:[^=]|$)`)

// parseMakefile joins backslash continuations, then walks rules.
//
// A RECIPE LINE IS A COMMAND ONLY IF MAKE WOULD RUN IT. A line whose first
// non-tab character is `#` is a comment to make and was a live command to the
// parser this replaces (A5). A leading `@` is silence, not semantics, and is
// stripped; a leading `-` is "ignore the exit status" and is kept as a fact.
func parseMakefile(src string) map[string]*target {
	out := map[string]*target{}
	var cur *target
	for _, raw := range joinContinuations(src) { // bounded by the file (P10-02)
		if strings.HasPrefix(raw, "\t") {
			if cur == nil {
				continue
			}
			body := strings.TrimSpace(raw)
			if body == "" || strings.HasPrefix(body, "#") {
				continue // a comment, whatever its indentation
			}
			cur.cmds = append(cur.cmds, newCommand(body))
			continue
		}
		// A COLUMN-0 COMMENT DOES NOT END A RECIPE. make ignores it and the next
		// tab line still belongs to the same target — `mutant-check` carries four
		// paragraphs between its `mkdir` and its `go test`, and a parser that reset
		// on the comment saw a required gate whose recipe ran nothing (A37).
		if strings.HasPrefix(raw, "#") {
			continue
		}
		cur = nil
		if strings.HasPrefix(raw, ".") {
			continue // a directive like .PHONY
		}
		mt := makeTargetLine.FindStringSubmatch(raw)
		if mt == nil {
			continue
		}
		name := mt[1]
		if out[name] == nil {
			out[name] = &target{name: name}
		}
		cur = out[name]
		after := raw[strings.Index(raw, ":")+1:]
		if i := strings.Index(after, "#"); i >= 0 {
			after = after[:i]
		}
		for _, d := range strings.Fields(after) { // bounded by the rule (P10-02)
			if !contains(cur.deps, d) {
				cur.deps = append(cur.deps, d) // ACCUMULATED across rules (A17)
			}
		}
	}
	return out
}

// joinContinuations is the Makefile with backslash-continued lines made one.
// A recipe split across lines is one command, and reading it as two loses
// whichever half carries the thing being looked for.
func joinContinuations(src string) []string {
	var out []string
	var cur string
	for _, l := range strings.Split(src, "\n") { // bounded by the file (P10-02)
		if strings.HasSuffix(l, "\\") {
			cur += strings.TrimSuffix(l, "\\") + " "
			continue
		}
		out = append(out, cur+l)
		cur = ""
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// shellJoiner splits on the shell's joiners so a check is judged on its own
// segment: `a.sh && b.sh --self-test` is a control for b.sh alone.
var shellJoiner = regexp.MustCompile(`&&|\|\||;`)

// newCommand splits a recipe line into segments and decides each one's facts.
func newCommand(body string) command {
	c := command{}
	body = strings.TrimLeft(body, "@")
	if strings.HasPrefix(body, "-") {
		c.ignoreExit = true
		body = strings.TrimLeft(body, "-@ ")
	}
	c.text = body
	parts := shellJoiner.Split(body, -1)
	joins := shellJoiner.FindAllString(body, -1)
	isTrue := func(s string) bool { s = strings.TrimSpace(s); return s == "true" || s == ":" }
	for i, raw := range parts { // bounded by the command (P10-02)
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}
		sg := segment{text: t, cannotFail: c.ignoreExit, orchestration: strings.Contains(t, "$(MAKE)")}
		if i < len(joins) && i+1 < len(parts) {
			next := parts[i+1]
			if (joins[i] == "||" && isTrue(next)) || (joins[i] == ";" && isTrue(next) && i+2 == len(parts)) {
				sg.cannotFail = true
			}
		}
		if i > 0 && joins[i-1] == "||" && isTrue(parts[i-1]) {
			sg.cannotFail = true // `true || x`: x is unreachable
		}
		c.segs = append(c.segs, sg)
	}
	return c
}

// checkerRef finds a project-written checker: ANY path under scripts/ — no
// extension required, because `scripts/lint-x` and `scripts/lint-x.bash` are
// checkers the extension list missed (K3) — with or without `./`, with or
// without an interpreter in front (A2–A4), or a tool run with `go run ./tools/<x>`.
var checkerRef = regexp.MustCompile(`(?:^|[\s@=])(?:(?:python3|bash|sh|expect)\s+)?(?:\./)?(scripts/[A-Za-z0-9_/.-]+|tools/[a-z][a-z0-9-]*)`)

// isCheck says whether one segment is the shape of a gate: a project checker, a
// `go test`, or a self-test — and is live.
func (sg segment) isCheck() bool {
	if sg.cannotFail || sg.orchestration {
		return false
	}
	return checkerRef.MatchString(sg.text) ||
		strings.Contains(sg.text, "go test") ||
		strings.Contains(sg.text, "-self-test") || strings.Contains(sg.text, "--self-test")
}

// runsACheck says whether any segment of the command is a live check.
func (c command) runsACheck() bool {
	for _, sg := range c.segs { // bounded by the command (P10-02)
		if sg.isCheck() {
			return true
		}
	}
	return false
}

// checkCannotFail says whether the command CARRIES a check whose failure is
// discarded — the thing a required gate must never do.
func (c command) checkCannotFail() bool {
	for _, sg := range c.segs { // bounded by the command (P10-02)
		if !sg.orchestration && sg.cannotFail && (checkerRef.MatchString(sg.text) || strings.Contains(sg.text, "go test")) {
			return true
		}
	}
	return false
}

// checkers is every project-written checker the command invokes, one per
// segment, minus sibling `_test.sh` controls (those are controls, not checkers)
// and minus orchestration.
func (c command) checkers() []string {
	var out []string
	for _, sg := range c.segs { // bounded by the command (P10-02)
		if sg.orchestration {
			continue
		}
		for _, m := range checkerRef.FindAllStringSubmatch(sg.text, -1) {
			if strings.HasSuffix(m[1], "_test.sh") || contains(out, m[1]) {
				continue
			}
			out = append(out, m[1])
		}
	}
	return out
}

// controls is every checker the command EXERCISES with a self-test — on the
// same segment as the flag, in a segment whose failure is not discarded.
func (c command) controls() []string {
	var out []string
	for _, sg := range c.segs { // bounded by the command (P10-02)
		if sg.cannotFail || sg.orchestration {
			continue
		}
		selfTest := strings.Contains(sg.text, "-self-test")
		for _, m := range checkerRef.FindAllStringSubmatch(sg.text, -1) {
			switch {
			case strings.HasSuffix(m[1], "_test.sh"):
				out = append(out, strings.TrimSuffix(m[1], "_test.sh")+".sh")
			case selfTest:
				out = append(out, m[1])
			}
		}
	}
	return out
}

// goBuild is the build VERB — `go build` or `$(GO) build` — and not the word
// inside `build-diag` in an echo. Any output form counts: `-o path`, `-o=path`,
// or none at all, which drops the binary in the working directory (A12–A14).
var goBuild = regexp.MustCompile(`(?:^|\s)(?:go|\$\(GO\))\s+build\b`)

func (c command) isBuild() bool {
	return goBuild.MatchString(c.text) && !strings.Contains(c.text, "-o /dev/null")
}

func (c command) trimsPath() bool {
	return strings.Contains(c.text, "$(TRIMPATH)") || strings.Contains(c.text, "-trimpath")
}

func (c command) isGoTest() bool { return strings.Contains(c.text, "go test") }

// countOne is `-count=1` or `-count 1` exactly — `-count 10` is a sample size,
// not a cache defence, and a substring match called it one.
var countOneFlag = regexp.MustCompile(`-count[= ]1\b`)

func (c command) countOne() bool { return countOneFlag.MatchString(c.text) }

// ---- derived views ------------------------------------------------------------------

// gateShaped is every target whose recipe runs a live check. Aggregators —
// targets that only name other targets, or only delegate to $(MAKE) — have no
// live check of their own and are not gate-shaped by construction.
func (m *buildModel) gateShaped() []string {
	var out []string
	for name, tg := range m.targets { // bounded by the Makefile (P10-02)
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			if c.runsACheck() {
				out = append(out, name)
				break
			}
		}
	}
	return sorted(out)
}

// verifyGates follows ONE hop of `$(MAKE) <target>` delegation from `verify`,
// because verify takes the tree lock and hands the list to verify-gates. One
// hop, not a chain — a chain is a place for a gate to hide.
func (m *buildModel) verifyGates() []string {
	v := m.targets["verify"]
	if v == nil {
		return nil
	}
	if len(v.deps) > 0 {
		return sorted(v.deps)
	}
	makeCall := regexp.MustCompile(`\$\(MAKE\)(?:\s+--[a-z-]+)*\s+([a-z][a-z0-9-]*)`)
	for _, c := range v.cmds { // bounded by the recipe (P10-02)
		if mm := makeCall.FindStringSubmatch(c.text); mm != nil {
			if inner := m.targets[mm[1]]; inner != nil {
				return sorted(inner.deps)
			}
		}
	}
	return nil
}

// ciGate is what the workflow does with one `make <gate>` step.
type ciGate struct {
	conditional bool // an `if:` on the step
	tolerant    bool // continue-on-error on the step, in any form but the literal false
	cannotFail  bool // `|| true` on the run line
	jobSilenced bool // the enclosing job carries an `if:` or continue-on-error
}

// A GATE STEP STARTS WITH `make`. `echo "$(make -s mutant-policy)"` reads a
// value in a subshell; it runs no gate, and matching it would report a target CI
// "runs" that verify never could.
var (
	ciMakeStep    = regexp.MustCompile(`^\s*make\s+([a-z][a-z0-9-]*)`)
	ciRunUnfailed = regexp.MustCompile(`\|\|\s*(true|:)\b`)
)

// ciGates is every `make <target>` the workflow runs, with how it is run.
func (m *buildModel) ciGates() map[string]ciGate {
	out := map[string]ciGate{}
	for _, job := range m.jobs { // bounded by the workflow (P10-02)
		jobSilenced := job.keys["if"] != "" || tolerant(job.keys["continue-on-error"])
		for _, st := range job.steps { // bounded by the job (P10-02)
			mm := ciMakeStep.FindStringSubmatch(st.keys["run"])
			if mm == nil {
				continue
			}
			out[mm[1]] = ciGate{
				conditional: st.keys["if"] != "",
				tolerant:    tolerant(st.keys["continue-on-error"]),
				cannotFail:  ciRunUnfailed.MatchString(st.keys["run"]),
				jobSilenced: jobSilenced,
			}
		}
	}
	return out
}

// tolerant reads continue-on-error by PRESENCE, not by literal value: `true`,
// `${{ true }}` and `${{ anything }}` all discard the failure (A24). Only the
// literal `false` does not.
func tolerant(v string) bool {
	v = strings.TrimSpace(v)
	return v != "" && v != "false"
}

// checkersOf is every project-written checker one target's live recipe invokes.
func (m *buildModel) checkersOf(name string) []string {
	tg := m.targets[name]
	if tg == nil {
		return nil
	}
	var out []string
	for _, c := range tg.cmds { // bounded by the recipe (P10-02)
		for _, k := range c.checkers() { // bounded by the command (P10-02)
			if !contains(out, k) {
				out = append(out, k)
			}
		}
	}
	return out
}

// controlled is every checker exercised with a self-test BY A GATE THAT RUNS —
// the recipes of required gates, and nowhere else (A9), on a live segment
// (A5, A6, A7).
func (m *buildModel) controlled() map[string]bool {
	out := map[string]bool{}
	for _, g := range m.required { // bounded by the gate list (P10-02)
		tg := m.targets[g]
		if tg == nil {
			continue
		}
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			for _, k := range c.controls() { // bounded by the command (P10-02)
				out[k] = true
			}
		}
	}
	return out
}

// builds is every live command that produces a binary, anywhere in the file.
func (m *buildModel) builds() []command {
	var out []command
	for _, tg := range m.targets { // bounded by the Makefile (P10-02)
		for _, c := range tg.cmds { // bounded by the recipe (P10-02)
			if c.isBuild() {
				out = append(out, c)
			}
		}
	}
	return out
}

// ---- the workflow -------------------------------------------------------------------

// parseWorkflow reads jobs, their own keys, and their steps, by indentation.
//
// THIS IS A RATCHET ON SPELLINGS, NOT A PROOF, and the gates over it say so.
// Nothing here executes GitHub Actions, so there is no oracle: a required job
// made to `needs:` a job that never runs, an `on.push.branches` that matches
// nothing, or an empty matrix all silence every gate and are invisible here
// (G2–G4, declared in the attack list). The guard for CI is that CI runs and its
// one-step-per-gate results are READ. What this catches is the cheap edits that
// would otherwise pass review as unremarkable.
//
// STDLIB ONLY, ON PURPOSE: a YAML dependency taken for one test is one the whole
// build carries. `jobs:` is at column 0, a job name at 2, its keys at 4, a step's
// dash at 6 and its keys at 8 — and a key on the DASH LINE belongs to that step
// (A20, A21). The shape is asserted by the caller's subject check.
func parseWorkflow(src string) map[string]*ciJob {
	out := map[string]*ciJob{}
	jobName := regexp.MustCompile(`^  ([A-Za-z][A-Za-z0-9_-]*):\s*$`)
	// `if : false` — a space before the colon — is the key `if` to YAML (G1).
	jobKey := regexp.MustCompile(`^    ([a-z][a-z-]*)\s*:\s*(.*)$`)
	stepDash := regexp.MustCompile(`^      -\s*(.*)$`)
	stepKey := regexp.MustCompile(`^        ([a-z][a-z-]*)\s*:\s*(.*)$`)
	keyValue := regexp.MustCompile(`^([a-z][a-z-]*)\s*:\s*(.*)$`)

	var cur *ciJob
	var inJobs bool
	var blockKey string                             // an `if: >-` block scalar continues on deeper lines
	for _, line := range strings.Split(src, "\n") { // bounded by the file (P10-02)
		if strings.HasPrefix(line, "jobs:") {
			inJobs = true
			continue
		}
		if !inJobs {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			inJobs = false // a new top-level key ends the jobs block
			continue
		}
		if mm := jobName.FindStringSubmatch(line); mm != nil {
			cur = &ciJob{keys: map[string]string{}}
			out[mm[1]] = cur
			blockKey = ""
			continue
		}
		if cur == nil {
			continue
		}
		if mm := jobKey.FindStringSubmatch(line); mm != nil {
			cur.keys[mm[1]] = strings.TrimSpace(mm[2])
			blockKey = ""
			continue
		}
		if mm := stepDash.FindStringSubmatch(line); mm != nil {
			st := ciStep{keys: map[string]string{}}
			if kv := keyValue.FindStringSubmatch(strings.TrimSpace(mm[1])); kv != nil {
				st.keys[kv[1]] = strings.TrimSpace(kv[2])
				blockKey = kv[1]
			}
			cur.steps = append(cur.steps, st)
			continue
		}
		if mm := stepKey.FindStringSubmatch(line); mm != nil && len(cur.steps) > 0 {
			cur.steps[len(cur.steps)-1].keys[mm[1]] = strings.TrimSpace(mm[2])
			blockKey = mm[1]
			continue
		}
		// A deeper line continues a block scalar (`if: >-`), so a multi-line
		// condition still registers as present.
		if blockKey != "" && len(cur.steps) > 0 && len(line)-len(strings.TrimLeft(line, " ")) > 8 {
			st := &cur.steps[len(cur.steps)-1]
			st.keys[blockKey] = strings.TrimSpace(st.keys[blockKey] + " " + trimmed)
		}
	}
	return out
}

// ---- the required list ------------------------------------------------------------------

func parseRequired(src string) []string {
	var out []string
	for _, l := range strings.Split(src, "\n") { // bounded by the file (P10-02)
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}
