package main

import (
	"fmt"
	"go/types"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
)

// crossingsPlan is the 2b plan's symbol-level overrides (design MS,
// crossings-MS.tsv): what goes to the kit instead of the map, and what moves
// into the map from the root.
type crossingsPlan struct {
	kit, mapSide map[string]bool
}

// readCrossings reads the design's crossings: a root-to-map row names a
// symbol on the map's side of 2b unless it goes to the kit; a map-to-root row
// that MOVES names a root symbol the map takes.
func readCrossings(p string) (*crossingsPlan, error) {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	c := &crossingsPlan{kit: map[string]bool{}, mapSide: map[string]bool{}}
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the file (P10-02)
		f := strings.Split(line, "\t")
		if len(f) < 5 {
			continue
		}
		switch {
		case f[0] == "tty->mapview" && f[4] == "KIT":
			c.kit[f[1]] = true
		case f[0] == "tty->mapview", f[0] == "mapview->tty" && f[4] == "MOVES":
			c.mapSide[f[1]] = true
		}
	}
	if len(c.mapSide) == 0 {
		return nil, fmt.Errorf("%s names no symbol on the map's side: it is not the design's crossings", p)
	}
	return c, nil
}

// ms7 is MS-7's measure: each test function that reaches the map's
// unexported code, with its class.
type ms7 struct {
	tests []ms7Test
}

// ms7Test is one test function's row.
type ms7Test struct {
	file, name, class string
	mapRefs           []string // the map's unexported symbols it reaches
	rootOut           []string // root symbols it reaches outside the allowlist
}

// count is the tests of one class.
func (m *ms7) count(class string) int {
	n := 0
	for _, t := range m.tests { // bounded by the tests (P10-02)
		if t.class == class {
			n++
		}
	}
	return n
}

// table is the rows as the tool records them.
func (m *ms7) table() string {
	var b strings.Builder
	b.WriteString("# graph: MS-7's test functions reaching the map's unexported code under 2b, and their classes (D-43)\ntest_file\ttest\tclass\tmap_unexported\troot_out\n")
	for _, t := range m.tests { // bounded by the tests (P10-02)
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n", t.file, t.name, t.class, strings.Join(t.mapRefs, " "), strings.Join(t.rootOut, " "))
	}
	return b.String()
}

// ttyDir is the root package's directory.
const ttyDir = "modes/tty"

// onMapSide says whether an object of package tty is on the map's side of
// 2b: declared in a non-test map file and not going to the kit, or named by
// the plan as the map's.
func (ix *index) onMapSide(c *crossingsPlan, o types.Object, decl string) bool {
	name := ix.symName(o)
	if c.mapSide[name] {
		return true
	}
	base := path.Base(decl)
	return path.Dir(decl) == ttyDir && strings.HasPrefix(base, "map_") && !strings.HasSuffix(base, "_test.go") && !c.kit[name]
}

// ms7 classifies package tty's test functions. A test's reach is its own
// body's and every helper's it calls, transitively. It is in MS-7 when it
// reaches an unexported symbol on the map's side; it is C3 when every root
// symbol it reaches is on the allowlist, C4 otherwise (0.19.6 D-43).
func (ix *index) ms7(c *crossingsPlan, allow map[string]bool) *ms7 {
	decl := map[types.Object]string{}
	for _, u := range ix.uses { // bounded by the uses (P10-02)
		decl[u.obj] = u.decl
	}
	m := &ms7{}
	for _, t := range ix.tests { // bounded by the tests (P10-02)
		reached := ix.reachOf(t)
		mapRefs, roots := map[string]bool{}, map[string]bool{}
		for o := range reached { // bounded by the test's reach (P10-02)
			d := decl[o]
			switch {
			case ix.onMapSide(c, o, d) && !o.Exported():
				mapRefs[ix.symName(o)] = true
			case !ix.onMapSide(c, o, d) && path.Dir(d) == ttyDir && !strings.HasSuffix(d, "_test.go"):
				roots[ix.symName(o)] = true
			}
		}
		if len(mapRefs) == 0 {
			continue
		}
		row := ms7Test{file: path.Base(t.file), name: t.name, class: "C3", mapRefs: slices.Sorted(maps.Keys(mapRefs))}
		for _, r := range slices.Sorted(maps.Keys(roots)) { // bounded by the test's root reach (P10-02)
			if !allow[r] {
				row.rootOut = append(row.rootOut, r)
			}
		}
		if len(row.rootOut) > 0 {
			row.class = "C4"
		}
		m.tests = append(m.tests, row)
	}
	return m
}

// reachOf is every object a test function's body names, through the helpers
// it calls, transitively; a helper is itself not counted.
func (ix *index) reachOf(t *testFunc) map[types.Object]bool {
	out := map[types.Object]bool{}
	seen := map[*testFunc]bool{}
	stack := []*testFunc{t}
	for len(stack) > 0 { // bounded by the test package's functions: each is visited once (P10-02)
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[f] {
			continue
		}
		seen[f] = true
		for _, o := range f.uses { // bounded by the function's body (P10-02)
			if h, ok := ix.testFuncs[o]; ok {
				stack = append(stack, h)
				continue
			}
			out[o] = true
		}
	}
	return out
}

// deriveAllow writes MS-7's allowlist from the design's classes (0.19.6 D-43): the
// root symbols the design's C3 tests reach between them. It refuses a set
// under which a design C4 test would read as C3, since then the design's
// boundary is not one an allowlist can hold.
func deriveAllow(ix *index, classesPath, outPath string, stdout, stderr *out) int {
	if classesPath == "" || outPath == "" {
		stderr.say("graph: derive-allow needs -classes <tests-fn-MS.tsv> and -out <file>")
		return 2
	}
	raw, err := os.ReadFile(classesPath)
	if err != nil {
		stderr.say("graph: %v", err)
		return 1
	}
	class := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the file (P10-02)
		if f := strings.Split(line, "\t"); len(f) > 2 && f[0] != "test_file" {
			class[f[0]+"\t"+f[1]] = f[2]
		}
	}
	c, err := readCrossings(ix.inputs + "/" + crossingsMS)
	if err != nil {
		stderr.say("graph: %v", err)
		return 1
	}
	everything := ix.ms7(c, map[string]bool{}) // nothing allowed: rootOut is every root symbol reached
	allow := map[string]bool{}
	for _, t := range everything.tests { // bounded by the tests (P10-02)
		if class[t.file+"\t"+t.name] == "C3" {
			for _, r := range t.rootOut { // bounded by the test's root reach (P10-02)
				allow[r] = true
			}
		}
	}
	held := ix.ms7(c, allow)
	for _, t := range held.tests { // bounded by the tests (P10-02)
		if want := class[t.file+"\t"+t.name]; want != t.class {
			stderr.say("graph: %s %s is %s by the derived list and %s in the design: the boundary does not hold", t.file, t.name, t.class, want)
			return 1
		}
	}
	var b strings.Builder
	b.WriteString("# graph: MS-7's allowlist (0.19.6 D-43): the root symbols the design's C3 tests reach between them,\n# derived from design-data/MS/tests-fn-MS.tsv by `graph derive-allow`; a test reaching a root symbol outside it is C4\n")
	for _, r := range slices.Sorted(maps.Keys(allow)) { // bounded by the list (P10-02)
		b.WriteString(r + "\n")
	}
	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		stderr.say("graph: %v", err)
		return 1
	}
	stdout.say("graph: derived %d allowed root symbols; the design's %d tests keep their classes under them", len(allow), len(held.tests))
	return 0
}
