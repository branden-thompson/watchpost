package gateoracle

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ParseRequired is the gate list in 06_docs/required-gates.txt: one name per
// line, `#` comments and blanks ignored.
func ParseRequired(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") { // bounded by the file (P10-02)
		if l := strings.TrimSpace(line); l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}

// AssertNoFileSilencesARequiredGate: EVERY REQUIRED GATE IS A RULE MAKE KNOWS,
// AND NO FILE NAMED AFTER ANY NODE — one at a time, then all that do not break
// the run at once — MAKES IT RECORD LESS. Nodes are what make reported
// considering in the real green run, plus every non-phony rule the database
// lists (a `$(MAKE)` hop with its output hidden keeps its node out of the walk).
func AssertNoFileSilencesARequiredGate(t Reporter, o *Oracle, required []string) {
	t.Helper()
	var checked int
	for _, g := range required { // bounded by the gate list (P10-02)
		if !o.known(g) {
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
		o.auditSilence(t, g, base)
	}
	if checked == 0 {
		t.Fatalf("COULD NOT RUN — no required gate is a make rule")
	}
	ceiling(t, fmt.Sprintf("audited %d gates for silence by a file", checked))
}

func (o *Oracle) auditSilence(t Reporter, g string, base run) {
	t.Helper()
	o.reset() // absence is judged against the commit, not against what the green run left behind
	var nodes []string
	for name, phony := range o.rules { // bounded by the database (P10-02)
		if !phony && o.absent(name) {
			nodes = append(nodes, name)
		}
	}
	for _, node := range base.nodes { // bounded by make's walk (P10-02)
		if o.absent(node) && !contains(nodes, node) {
			nodes = append(nodes, node)
		}
	}
	sort.Strings(nodes)
	var harmless []string        // nodes a file of whose name does not break the run
	for _, node := range nodes { // bounded by the nodes (P10-02)
		if broke := o.silencedBy(t, g, base, []string{node}); !broke {
			harmless = append(harmless, node)
		}
	}
	if len(harmless) > 1 {
		if broke := o.silencedBy(t, g, base, harmless); broke {
			t.Errorf("UNJUDGEABLE — `make %s` is red with files named %v all present, though each alone is "+
				"harmless. The recipe refuses a combination the audit cannot reduce; make it judgeable.", g, harmless)
		}
	}
}

// silencedBy creates the files, runs the gate green, and reports a shrunken
// record; it returns whether the files broke the run instead.
func (o *Oracle) silencedBy(t Reporter, g string, base run, set []string) (broke bool) {
	t.Helper()
	o.reset()
	for _, node := range set { // bounded by the set (P10-02)
		path := filepath.Join(o.tree, node)
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		_ = os.WriteFile(path, nil, 0o644)
	}
	o.paint()
	touched := o.runGateAsIs(g, o.env())
	o.reset()
	if touched.code != 0 {
		return true
	}
	var lost []string
	for _, k := range base.reach { // bounded by the reach (P10-02)
		if !contains(touched.reach, k) {
			lost = append(lost, k)
		}
	}
	if len(lost) > 0 {
		t.Errorf("a file named %v makes `make %s` exit 0 without running %v: make says \"is up to date\" and "+
			"skips it. Add it to .PHONY.", set, g, lost)
	}
	return false
}

// AssertEveryRequiredGateCanFail: EVERY REQUIRED GATE GOES GREEN WHEN EVERYTHING
// IS, AND RED FOR EACH INVOCATION IT RECORDED, PAINTED ALONE.
func AssertEveryRequiredGateCanFail(t Reporter, o *Oracle, required []string) {
	t.Helper()
	var checked, judged int
	for _, g := range required { // bounded by the gate list (P10-02)
		if !o.known(g) {
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
			o.paint(inv)
			if o.runGate(g).code == 0 {
				t.Errorf("%s is a REQUIRED gate and `make %s` exits 0 with ONLY %s red.\nThe gate ran it and does "+
					"not fail when it does — its status is discarded somewhere on that path, and make has already "+
					"said so. A diagnostic under `|| true` is refused for the same reason: move it out of the gate "+
					"or let it fail.", g, g, inv)
			}
		}
		o.paint()
		for _, tool := range absentable(keysOf(base.reach)) { // bounded by the reach (P10-02)
			judged++
			without := o.runGateEnv(g, o.withoutTool(t, tool))
			if without.code == 0 && fallbackOf(base, without) == "" {
				t.Errorf("%s is a REQUIRED gate and `make %s` exits 0 with %s ABSENT from PATH and nothing taking its "+
					"place.\nThe gate ran %s when it was there and is green when it is not: a `command -v … || exit 0` "+
					"or `which … || true` skips the check on any machine without the tool. Absence must be loud — "+
					"exit 1 — or a fallback must run.", g, g, tool, tool)
			}
		}
	}
	o.paint()
	if checked == 0 {
		t.Fatalf("COULD NOT RUN — no required gate is a make rule")
	}
	ceiling(t, fmt.Sprintf("judged %d gates, %d invocations", checked, judged))
}

// AssertVerifyCanFail: `make verify` REACHES THE GATES, RECORDS AT LEAST AS MANY
// INVOCATIONS OF EVERY KEY AS THE NON-CI-ONLY GATES RECORD ON THEIR OWN, AND
// GOES RED FOR EACH INVOCATION PAINTED ALONE.
func AssertVerifyCanFail(t Reporter, o *Oracle, required []string, ciOnlyRows map[string]string) {
	t.Helper()
	if !o.known("verify") {
		t.Fatalf("COULD NOT RUN — no verify rule")
	}
	base := o.green(t, "verify")
	if base.code != 0 {
		return
	}
	if !reachesACheck(base.reach) {
		t.Errorf("UNJUDGEABLE — `make verify` exits 0 and ran no checker: the entry point never reaches the " +
			"gates. The oracle's go stub delegates only for `go run ./tools/treelock … -- CMD`; a different " +
			"spelling or wrapper is not judged, and this is COULD-NOT-RUN.")
		return
	}
	o.verifyCoversEachGate(t, base, required, ciOnlyRows)
	for _, inv := range base.reach { // bounded by the reach (P10-02)
		o.paint(inv)
		if o.runGate("verify").code == 0 {
			t.Errorf("`make verify` exits 0 with ONLY %s red. The entry point discards that failure — "+
				"release.yml reads this exit, so a red release would ship.", inv)
		}
	}
	o.paint()
	ceiling(t, fmt.Sprintf("verify judged for %d invocations", len(base.reach)))
}

func reachesACheck(reach []string) bool {
	for _, k := range reach { // bounded by the reach (P10-02)
		if strings.HasPrefix(k, "scripts/") || strings.HasPrefix(k, "ctl:") {
			return true
		}
	}
	return false
}

// verifyCoversEachGate: BY COUNT, NOT BY KEY. A variable passed down that a gate
// skips ONE of two `go test` calls under leaves the key present and the check
// gone.
func (o *Oracle) verifyCoversEachGate(t Reporter, base run, required []string, ciOnlyRows map[string]string) {
	t.Helper()
	verifyCounts, need := countsOf(base.reach), map[string]int{}
	for _, g := range required { // bounded by the gate list (P10-02)
		if _, ci := ciOnlyRows[g]; ci || !o.known(g) {
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
}

// fallbackOf is a tool key the run without a tool recorded that the green run
// did not — `shasum` standing in for an absent `sha256sum` — or "".
func fallbackOf(base, without run) string {
	had := keysOf(base.reach)
	for _, k := range keysOf(without.reach) { // bounded by the reach (P10-02)
		if contains(tools(), k) && !contains(had, k) {
			return k
		}
	}
	return ""
}

// absentable is every stubbed tool a reach names whose absence PATH can
// simulate: not `go` (nothing runs without it), not what the OS ships.
func absentable(keys []string) []string {
	var out []string
	for _, k := range keys { // bounded by the reach (P10-02)
		if k != "go" && contains(tools(), k) && !contains(osShipped(), k) && !contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// isChecker: a project-written checker owes a control. Scripts (not `_test.sh`
// siblings), local `go run ./…` packages, and binaries `go build -o` wrote. A
// third-party `go run host.tld/…` tool carries no `-self-test` contract of ours.
func isChecker(key string) bool {
	switch {
	case strings.HasPrefix(key, "scripts/"):
		return !strings.HasSuffix(key, "_test.sh")
	case key == "go:run:.", strings.HasPrefix(key, "go:run:./"), strings.HasPrefix(key, "built:"):
		return true
	}
	return false
}

// AssertEveryControlIsReached: EVERY CONTROL IS REACHED — only the CONTROL red,
// every carrier tried. A checker's control is `ctl:<key>` or, for a script, a
// sibling `<key>_test.sh`; its carriers are the required gates that recorded it.
func AssertEveryControlIsReached(t Reporter, o *Oracle, required []string, uncontrolledRows map[string]string) {
	t.Helper()
	var checked int
	for _, k := range o.checkersInvoked(t, required) { // bounded by the checker list (P10-02)
		if _, declared := uncontrolledRows[k]; declared {
			continue
		}
		checked++
		ctl, sib := controlsFor(k)
		carriers := o.carriersOf(t, required, ctl, sib)
		if len(carriers) == 0 {
			orSibling := ""
			if sib != "" {
				orSibling = " or a sibling `" + sib + "`"
			}
			t.Errorf("%s is run by a required gate and no required gate runs its control — `%s --self-test`%s. "+
				"(`go test` of a tool is not a control: a tool's tests prove the tool, the self-test proves it can "+
				"FAIL in this tree.)\nAdd the control to gate-controls, or add a row to `uncontrolled` saying why "+
				"this checker is trusted without evidence.", k, k, orSibling)
			continue
		}
		o.paint(ctl, sib)
		if !o.anyRed(carriers) {
			t.Errorf("%s's control ran in %v and its failure is DISCARDED: with only that control red, every "+
				"carrier stays green — `|| true`, `;`, a `-` prefix — and make has said so.", k, carriers)
		}
	}
	o.paint()
	if checked == 0 {
		t.Fatalf("COULD NOT RUN — no checker is run by a required gate")
	}
	ceiling(t, fmt.Sprintf("%d checkers' controls judged", checked))
}

// checkersInvoked is every checker key any required gate recorded, sorted.
func (o *Oracle) checkersInvoked(t Reporter, required []string) []string {
	t.Helper()
	var invoked []string
	for _, g := range required { // bounded by the gate list (P10-02)
		for _, k := range keysOf(o.green(t, g).reach) { // bounded by the reach (P10-02)
			if isChecker(k) && !contains(invoked, k) {
				invoked = append(invoked, k)
			}
		}
	}
	sort.Strings(invoked)
	return invoked
}

// controlsFor is a checker's control key and, for a script, its sibling.
func controlsFor(k string) (ctl, sib string) {
	ctl = "ctl:" + k
	if strings.HasPrefix(k, "scripts/") {
		sib = strings.TrimSuffix(k, ".sh") + "_test.sh"
	}
	return ctl, sib
}

// carriersOf is every required gate whose green run recorded ctl or sib.
func (o *Oracle) carriersOf(t Reporter, required []string, ctl, sib string) []string {
	t.Helper()
	var carriers []string
	for _, g := range required { // bounded by the gate list (P10-02)
		keys := keysOf(o.green(t, g).reach)
		if contains(keys, ctl) || (sib != "" && contains(keys, sib)) {
			carriers = append(carriers, g)
		}
	}
	return carriers
}

// anyRed runs each gate as painted and says whether one went red.
func (o *Oracle) anyRed(gates []string) bool {
	for _, g := range gates { // bounded by the gate list (P10-02)
		if o.runGate(g).code != 0 {
			return true
		}
	}
	return false
}
