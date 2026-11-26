// graph is 0.19.6's committed graph tool (B0.12, A-12, D-30): the figures M3
// holds a batch to, counted with full type information over `modes/tty` and
// `app`, tests included, across packages the same way as within one.
//
// WHAT IT COUNTS, each as DISCOVER and PLAN defined it (0.19.6 D-41, D-43, D-44):
//   - the hub's fan-in: non-test files that use `Dashboard`, a member of it,
//     or declare a method on it, in any package;
//   - `Dashboard`'s methods and the files that declare them;
//   - each area's reach into the model: the distinct objects declared in the
//     model's files that the area's non-test files read, a field once as
//     itself, and every field of a carrier type (an `Env`, the hooks) the area
//     reads, so a pass-through counts what it passes;
//   - the map's reach by type: members of the model's types, declared outside
//     the map's files, that the map's files read;
//   - MS-7: the test functions that reach the map's unexported code under 2b,
//     their helpers' reach included, split into C3 (moves with the code) and
//     C4 (cannot be served) by the committed allowlist (0.19.6 D-43) and the
//     seam's own root names, each with the allowed names it stands for
//     (0.19.6 A-67);
//   - `app`'s unexported crossings: uses of an unexported object whose file
//     and the object's are planned for different packages (the planned map);
//   - the deck-only test files (design AP-7): the app objects each reaches,
//     through its helpers, that the planned map puts outside the deck.
//
// WHY IT EXISTS. The scratch tool PLAN used saw only uses within a package
// (0.19.6 CQ F-17): once the map is a package of its own, its reach into the model
// falls by construction. This one resolves every use to its object, wherever
// it is declared.
//
// usage:
//
//	go -C tools/graph run . record -dir <tree> [-inputs ../..] -out <figures.tsv> [-tests <ms7.tsv>] [-crossings <list.tsv>]
//	go -C tools/graph run . check -dir ../.. -baseline <figures.tsv>
//	go -C tools/graph run . derive-allow -dir <tree> [-inputs ../..] -classes <tests-fn-MS.tsv> -out <allow.tsv>
//
// -dir is the tree measured; -inputs the root the committed inputs are read
// from, so the e57afeaf baseline is recorded from a worktree of that commit,
// which predates the design data.
//
// check fails when a held figure rises above its baseline (M3's figures and
// the crossings may only fall), or a deck-only test file reaches outside the
// deck, and reports MS-7's counts beside theirs.
package main

import (
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], &out{w: os.Stdout}, &out{w: os.Stderr}))
}

// out is a writer that keeps its first error, so a run whose verdict could
// not be written is not taken for one that passed.
type out struct {
	w   io.Writer
	err error
}

// say writes one formatted line, once nothing before it failed.
func (o *out) say(format string, a ...any) {
	if o.err == nil {
		_, o.err = fmt.Fprintf(o.w, format+"\n", a...)
	}
}

// run is main without the process.
func run(args []string, stdout, stderr *out) int {
	code := dispatch(args, stdout, stderr)
	if stdout.err != nil || stderr.err != nil {
		return max(code, 1)
	}
	return code
}

// The committed inputs, relative to the repository's root.
const (
	crossingsMS = "06_docs/02_features/0.19.6-re-architecting/03-architecture-design/design-data/MS/crossings-MS.tsv"
	allowMS     = "06_docs/graph/ms7-allow.tsv"
	seamMS      = "06_docs/graph/ms7-seam-B4.tsv"
	plannedApp  = "app/testdata/sharedlock/planned.tsv"
)

// dispatch reads the subcommand and its flags.
func dispatch(args []string, stdout, stderr *out) int {
	if len(args) == 0 || !slices.Contains([]string{"record", "check", "derive-allow"}, args[0]) {
		stderr.say("graph: say record, check or derive-allow")
		return 2
	}
	fs := flag.NewFlagSet("graph "+args[0], flag.ContinueOnError)
	fs.SetOutput(stderr.w)
	dir := fs.String("dir", ".", "the tree to measure, a repository's root")
	inputs := fs.String("inputs", "", "the root the committed inputs are read from (the design's crossings, the allowlist, the planned map); -dir when empty")
	outFile := fs.String("out", "", "record, derive-allow: the file to write")
	tests := fs.String("tests", "", "record: also write MS-7's test functions and their classes here")
	crossings := fs.String("crossings", "", "record: also write app's unexported crossings here")
	baseline := fs.String("baseline", "", "check: the figures to hold the tree to")
	classes := fs.String("classes", "", "derive-allow: the design's tests-fn-MS.tsv")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	ix, err := load(*dir)
	if err != nil {
		stderr.say("graph: %v", err)
		return 1
	}
	ix.inputs = ix.dir
	if *inputs != "" {
		if ix.inputs, err = filepath.Abs(*inputs); err != nil {
			stderr.say("graph: %v", err)
			return 1
		}
	}
	switch args[0] {
	case "derive-allow":
		return deriveAllow(ix, *classes, *outFile, stdout, stderr)
	case "record":
		return record(ix, *outFile, *tests, *crossings, stdout, stderr)
	}
	return check(ix, *baseline, stdout, stderr)
}

// ---- recording and checking ---------------------------------------------------

// figure is one named count.
type figure struct {
	name  string
	value int
}

// measure is every figure over the index.
func measure(ix *index) ([]figure, *ms7, []string, error) {
	cross, err := readCrossings(filepath.Join(ix.inputs, crossingsMS))
	if err != nil {
		return nil, nil, nil, err
	}
	allow, err := readSet(filepath.Join(ix.inputs, allowMS))
	if err != nil {
		return nil, nil, nil, err
	}
	seam, err := readSet(filepath.Join(ix.inputs, seamMS))
	if err != nil {
		return nil, nil, nil, err
	}
	for name := range seam { // bounded by the file (P10-02)
		allow[name] = true // a seam name is the path to the model or what the Env carries, as the names it stands for were
	}
	planned, err := readPlanned(filepath.Join(ix.inputs, plannedApp))
	if err != nil {
		return nil, nil, nil, err
	}
	methods, files := ix.dashboardMethods()
	m := ix.ms7(cross, allow)
	crossed := ix.crossings(planned)
	figs := []figure{
		{"hub fan-in (files using Dashboard)", ix.fanIn()},
		{"Dashboard methods", methods},
		{"Dashboard method files", files},
	}
	for _, a := range areas { // bounded by the areas (P10-02)
		figs = append(figs, figure{a.name, len(ix.reach(a))})
	}
	figs = append(figs,
		figure{"map reach by type", len(ix.reachByType(mapByType))},
		figure{"MS-7 tests reaching the map's unexported code", len(m.tests)},
		figure{"MS-7 C3, moves with the code", m.count("C3")},
		figure{"MS-7 C4, cannot be served", m.count("C4")},
		figure{"app unexported crossings under the planned map", len(crossed)},
	)
	return figs, m, crossed, nil
}

// record writes the figures, and the lists behind two of them.
func record(ix *index, path, tests, crossings string, stdout, stderr *out) int {
	if path == "" {
		stderr.say("graph: record needs -out <file>")
		return 2
	}
	figs, m, crossed, err := measure(ix)
	if err != nil {
		stderr.say("graph: %v", err)
		return 1
	}
	var b strings.Builder
	b.WriteString("# graph: 0.19.6 M3's figures (B0.12), recorded by tools/graph; check holds each at or below\nfigure\tvalue\n")
	for _, f := range figs { // bounded by the figures (P10-02)
		fmt.Fprintf(&b, "%s\t%d\n", f.name, f.value)
	}
	writes := map[string]string{path: b.String()}
	if tests != "" {
		writes[tests] = m.table()
	}
	if crossings != "" {
		writes[crossings] = "# graph: app's unexported crossings under the planned map: user file, object, its file\n" + strings.Join(crossed, "\n") + "\n"
	}
	for _, p := range slices.Sorted(maps.Keys(writes)) { // bounded by the outputs (P10-02)
		if err := os.WriteFile(p, []byte(writes[p]), 0o644); err != nil {
			stderr.say("graph: %v", err)
			return 1
		}
	}
	for _, f := range figs { // bounded by the figures (P10-02)
		stdout.say("graph: %-52s %d", f.name, f.value)
	}
	return 0
}

// check holds the tree's figures to a baseline: none may rise, and every
// baseline figure must still be measured.
func check(ix *index, path string, stdout, stderr *out) int {
	base, err := readFigures(path)
	if err != nil {
		stderr.say("graph: %v", err)
		return 2
	}
	figs, _, _, err := measure(ix)
	if err != nil {
		stderr.say("graph: %v", err)
		return 1
	}
	failed := 0
	now := map[string]int{}
	for _, f := range figs { // bounded by the figures (P10-02)
		now[f.name] = f.value
		was, ok := base[f.name]
		switch {
		case !ok:
			stdout.say("graph: FAIL %q has no baseline: record it", f.name)
			failed++
		case reported(f.name):
			stdout.say("graph: %-52s %d (baseline %d; reported, re-measured at B4.4 under D-9)", f.name, f.value, was)
		case f.value > was:
			stdout.say("graph: FAIL %s rose: %d, against %d at baseline (M3's figures may only fall)", f.name, f.value, was)
			failed++
		default:
			stdout.say("graph: %-52s %d (baseline %d)", f.name, f.value, was)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(base)) { // bounded by the baseline (P10-02)
		if _, ok := now[name]; !ok {
			stdout.say("graph: FAIL %q is in the baseline and was not measured", name)
			failed++
		}
	}
	pl, err := readPlanned(filepath.Join(ix.inputs, plannedApp))
	if err != nil {
		stderr.say("graph: %v", err)
		return 1
	}
	for _, f := range deckOnlyTests { // bounded by the list (P10-02)
		outside := ix.outsideTheDeck(pl, f)
		for _, o := range outside { // bounded by the findings (P10-02)
			stdout.say("graph: FAIL %s reaches %s: it moves intact with the deck (design AP-7), so it may reach the deck's code and app's test support alone", f, o)
		}
		failed += len(outside)
		if len(outside) == 0 {
			stdout.say("graph: %-52s deck-only", f)
		}
	}
	if failed > 0 {
		return 1
	}
	stdout.say("graph: OK - %d figures, none above its baseline", len(figs))
	return 0
}

// reported says whether a figure is reported and not held: MS-7's counts
// grow with every test that reaches the map's internals, and 0.19.6 D-9 has them
// re-measured on 2a's code, where the HUM LEAD rules on them (B4.4). M3's
// figures and the crossings are held: they may only fall.
func reported(name string) bool { return strings.HasPrefix(name, "MS-7 ") }

// readFigures reads a figures file.
func readFigures(path string) (map[string]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the file (P10-02)
		name, v, ok := strings.Cut(line, "\t")
		if !ok || strings.HasPrefix(line, "#") || name == "figure" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %q: %v", path, line, err)
		}
		out[name] = n
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s holds no figure: it is not a graph baseline", path)
	}
	return out, nil
}

// readSet reads a file's first column, less comments and its header.
func readSet(path string) (map[string]bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the file (P10-02)
		first, _, _ := strings.Cut(line, "\t")
		if first != "" && !strings.HasPrefix(first, "#") {
			out[first] = true
		}
	}
	return out, nil
}
