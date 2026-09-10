// wires — the producer/consumer completeness check over the closed sets.
//
// PROPOSED AT 0.14.0 AND UNBUILT UNTIL NOW. The 0.14.0 debrief called it "the
// single highest-value thing 0.15.0 could inherit", against a defect shape it
// had counted NINE times in one release:
//
//	"The wire is not pinned. A rule implemented and pinned in one layer, not
//	 carried by the layer that would deliver it."
//
// and round 3 wrote the design in one sentence: "Every Effect, every Event,
// every Arrival field and every Slot must name a production writer and a
// production reader, or carry an explicit 'unwired, and why' entry."
//
// It stayed unbuilt through 0.15.0 and 0.16.0 P3, and P3's audio merge was
// another instance: `Duck` and `Restore` are declared effects with executors
// and NO PRODUCER, and the batch that was supposed to give them one instead
// routed around them.
//
// WHAT IT CHECKS. For every member of every CLOSED SET in production code:
//
//	a WRITER   — somewhere constructs it, or assigns/passes the constant
//	a READER   — somewhere discriminates on it, or compares against it
//
// A member missing either is UNWIRED, and must carry a ratified row in
// 06_docs/wires-ratified.md saying so and why. Same discipline as metric D and
// the P10 exemptions: the reason is HUM LEAD's, never self-issued.
//
// HOW THE SUBJECT LIST IS DERIVED, NEVER ENUMERATED (INST-1). Two shapes, and
// the codebase declares both deliberately:
//
//	a SENTINEL ENUM — a const block whose last member is `num<Something>`,
//	  which this codebase uses precisely to say "the set is closed and this
//	  bounds it" (numTracks, numSlots, numPowers, numNarrationClasses, …)
//	a MARKER SET    — structs embedding `isEffect` or `isEvent`, which
//	  `platform/lineup` uses to keep the effect and event sets closed
//
// A hand-written list of sets would go stale on the day a thirteenth is added,
// which is the failure this tool exists to catch, one level up.
//
// WHAT IT CAUGHT ON ITS FIRST RUN, INCLUDING ONE I PREDICTED IT WOULD MISS.
// This header claimed a priority enum compared only ordinally would count as
// read, and therefore that `narrateRotation` — the class whose give-way was
// P3's first blocker — would slip past. IT DID NOT. The ordinal comparisons in
// the arbiter are `job.class > on.class`, variable against variable: the
// CONSTANT is never named on either side, so nothing discriminates on it and
// the member reports NO READER.
//
// So the check would have found two of P3's four blockers before a line was
// written: `narrateRotation` with no reader, and `Power.OffAir` with no writer,
// which is the unreachable STANDBY behind RS-3 and F-72.
//
// THE REAL LIMIT, corrected. A reader is any discrimination — a switch case, an
// equality, an ordered comparison against the constant, a registry key, or
// membership of a set written out as a literal. A member discriminated on for
// the WRONG reason still counts as read, and a member handled in three places
// and forgotten in a fourth looks whole. This finds the DEAD member and the
// WRITE-ONLY member. It does not find the incomplete switch.
//
// usage: go run ./tools/wires [-ledger path] [-json] [-self-test]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// member is one element of one closed set, and what production code does with it.
type member struct {
	Set     string `json:"set"`     // the type it belongs to
	Name    string `json:"name"`    // the constant or struct name
	Decl    string `json:"decl"`    // file:line where it is declared
	Writers int    `json:"writers"` // production sites that construct or assign it
	Readers int    `json:"readers"` // production sites that discriminate on it

	// WriteAt and ReadAt are WHERE, which is what makes a review of an unwired
	// member possible without grepping: the question is never only "is it
	// wired" but "by whom, and is that the right whom".
	WriteAt []string `json:"writeAt,omitempty"`
	ReadAt  []string `json:"readAt,omitempty"`

	// Zero marks the FIRST member of an iota enum — the type's zero value.
	//
	// EVERY ZERO CONSTRUCTION WRITES IT, and none of them names it: `Card{…}`
	// with no State field makes a PROPOSED card, and no walk of the syntax can
	// see that. Reporting such a member as having no writer is the tool being
	// wrong about the code, and a ratified exemption for it would record a
	// false thing as ratified.
	Zero bool `json:"zero,omitempty"`
}

// wrote and read record a site as well as counting it.
func (m *member) wrote(at string) { m.Writers++; m.WriteAt = append(m.WriteAt, at) }
func (m *member) read(at string)  { m.Readers++; m.ReadAt = append(m.ReadAt, at) }

// unwired reports whether this member is missing a writer or a reader.
//
// A ZERO VALUE IS NEVER MISSING A WRITER — see Zero. It can still be missing a
// READER, and that is a real finding: being the default does not mean anything
// discriminates on it.
func (m member) unwired() bool {
	if m.Readers == 0 {
		return true
	}
	return m.Writers == 0 && !m.Zero
}

// why says which half is missing, in the words the ledger uses.
func (m member) why() string {
	switch {
	case m.Zero && m.Readers == 0:
		return "NO READER — the type's zero value, so every zero construction makes one, and nothing discriminates on it"
	case m.Writers == 0 && m.Readers == 0:
		return "no writer and no reader — nothing makes it and nothing looks at it"
	case m.Writers == 0:
		return "NO WRITER — declared and handled, but nothing in production makes one"
	default:
		return "NO READER — made in production, and nothing discriminates on it"
	}
}

func main() {
	ledger := flag.String("ledger", "06_docs/wires-ratified.md", "the ratified-exemption ledger")
	asJSON := flag.Bool("json", false, "machine-readable output")
	sites := flag.Bool("sites", false, "print WHERE each unwired member is written and read")
	selfTest := flag.Bool("self-test", false, "prove the instrument can fail, then exit")
	root := flag.String("root", ".", "the tree to scan")
	flag.Parse()

	if *selfTest {
		os.Exit(runSelfTest())
	}
	members, err := scan(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wires:", err)
		os.Exit(2)
	}
	ratified, err := readLedger(filepath.Join(*root, *ledger))
	if err != nil {
		fmt.Fprintln(os.Stderr, "wires:", err)
		os.Exit(2)
	}
	report(members, ratified, *asJSON, *sites)
}

// report prints the verdict and exits non-zero if anything is unexplained or
// any ledger row has gone stale.
func report(members []member, ratified map[string]bool, asJSON, sites bool) {
	unexplained, exempt, stale := triage(members, ratified)
	if asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"members": len(members), "unexplained": unexplained, "exempt": len(exempt), "stale": stale,
		})
	} else {
		// SILENCE IS A DISTINCT VERDICT (INST-2). Scanning nothing and finding
		// nothing is a broken walk, not a clean tree, and it must not read as
		// a pass.
		if len(members) == 0 {
			fmt.Fprintln(os.Stderr, "wires: found no closed sets at all — the walk did not run")
			os.Exit(2)
		}
		fmt.Printf("wires: %d member(s) across the closed sets; %d ratified as unwired, %d NOT\n",
			len(members), len(exempt), len(unexplained))
		for _, k := range stale {
			fmt.Printf("  STALE EXEMPTION  %s\n      it is WIRED now — the ledger row was owed until this "+
				"happened, and its removal is what closes the obligation\n", k)
		}
		for _, m := range unexplained {
			fmt.Printf("  %-28s %s\n      %s\n", m.Set+"."+m.Name, m.Decl, m.why())
			if sites {
				for _, w := range m.WriteAt {
					fmt.Printf("        written  %s\n", w)
				}
				for _, r := range m.ReadAt {
					fmt.Printf("        read     %s\n", r)
				}
			}
		}
		fmt.Println("  scope: production code only. A reader is any discrimination, including an")
		fmt.Println("  ordered comparison — so a member nothing treats SPECIALLY still counts as read.")
	}
	if failing(unexplained, stale) {
		os.Exit(1)
	}
}

// failing is the exit decision, and BOTH halves fail the build.
//
// A STALE ROW MUST FAIL, NOT WARN. It is the whole enforcement: the ledger's
// rows are owed, and what makes "remove it once the wiring lands" true is that
// wiring the member breaks the build until the row goes. A stale row that only
// printed a line would be a reminder, and a reminder is what this project has
// repeatedly proved it does not act on.
func failing(unexplained []member, stale []string) bool {
	return len(unexplained) > 0 || len(stale) > 0
}

// triage splits the members against the ledger: what is unexplained, what is
// ratified and still earning its row, and which rows have gone STALE.
//
// THE LEDGER'S OWN EXPIRY. Every row is OWED, not accepted (HUM LEAD
// 2026-09-09: "they can join the ledger now, but need to be removed once wiring
// is in place"), and a promise to clean up later is the exact shape this
// project keeps paying for. So a row whose member is no longer unwired — or
// whose member no longer EXISTS — is stale and it fails. Wiring the member is
// what breaks the build until the row goes, which is the only version of
// "remember to remove it" that works.
func triage(members []member, ratified map[string]bool) (unexplained, exempt []member, stale []string) {
	left := map[string]bool{}
	for k := range ratified { // bounded by the ledger (P10-02)
		left[k] = true
	}
	for _, m := range members { // bounded by the closed sets (P10-02)
		key := m.Set + "." + m.Name
		if !m.unwired() {
			continue
		}
		delete(left, key) // still unwired: the row is still earning its place
		if ratified[key] {
			exempt = append(exempt, m)
			continue
		}
		unexplained = append(unexplained, m)
	}
	return unexplained, exempt, sortedKeys(left)
}

// scan walks the tree and returns every closed-set member with its counts.
func scan(root string) ([]member, error) {
	fset := token.NewFileSet()
	var files []*ast.File
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Vendored, hidden and build directories are not this tree's rules.
			base := filepath.Base(path)
			if base != "." && (strings.HasPrefix(base, ".") || base == "dist" || base == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}
		files = append(files, f)
		return nil
	})
	if err != nil {
		return nil, err
	}
	decls := declaredMembers(fset, files)
	countUses(fset, files, decls)
	out := make([]member, 0, len(decls))
	for _, m := range decls {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Set != out[j].Set {
			return out[i].Set < out[j].Set
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// declaredMembers finds every closed set and its members, by the two shapes the
// codebase uses to declare closure.
func declaredMembers(fset *token.FileSet, files []*ast.File) map[string]*member {
	out := map[string]*member{}
	for _, f := range files {
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			switch gen.Tok {
			case token.CONST:
				for _, m := range sentinelEnum(fset, gen) {
					out[m.Name] = m
				}
			case token.TYPE:
				for _, m := range markerStructs(fset, gen) {
					out[m.Name] = m
				}
			}
		}
	}
	return out
}

// sentinelEnum reads a const block whose LAST name is `num<Set>` — this
// codebase's way of saying the set is closed and that identifier bounds it.
// The sentinel is not itself a member.
func sentinelEnum(fset *token.FileSet, gen *ast.GenDecl) []*member {
	var names []*ast.Ident
	var typeName string
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if id, ok := vs.Type.(*ast.Ident); ok && typeName == "" {
			typeName = id.Name
		}
		names = append(names, vs.Names...)
	}
	if len(names) < 2 || typeName == "" {
		return nil
	}
	last := names[len(names)-1].Name
	if !strings.HasPrefix(last, "num") || last == "num" {
		return nil
	}
	out := make([]*member, 0, len(names)-1)
	for i, n := range names[:len(names)-1] {
		if n.Name == "_" {
			continue
		}
		// THE FIRST NAME IS THE ZERO VALUE. iota starts at 0 and this codebase
		// declares every closed enum that way; a block that did not would put
		// its own type on the first spec, which sentinelEnum already requires.
		out = append(out, &member{Set: typeName, Name: n.Name, Decl: pos(fset, n.Pos()), Zero: i == 0})
	}
	return out
}

// markerStructs reads a type block for structs embedding a marker that keeps a
// set closed — `isEffect` and `isEvent` in platform/lineup.
func markerStructs(fset *token.FileSet, gen *ast.GenDecl) []*member {
	var out []*member
	for _, spec := range gen.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			continue
		}
		for _, fld := range st.Fields.List {
			if len(fld.Names) != 0 {
				continue // an embedded field has no name of its own
			}
			id, ok := fld.Type.(*ast.Ident)
			if !ok {
				continue
			}
			set := map[string]string{"isEffect": "Effect", "isEvent": "Event"}[id.Name]
			if set == "" {
				continue
			}
			out = append(out, &member{Set: set, Name: ts.Name.Name, Decl: pos(fset, ts.Pos())})
		}
	}
	return out
}

// countUses walks every production file and attributes each use of a member to
// WRITING it or READING it.
//
// THE DECLARATION IS NOT A USE. A const block naming a member, and the struct
// type declaring it, are what created the question; counting them as answers
// would make every member look wired.
func countUses(fset *token.FileSet, files []*ast.File, decls map[string]*member) {
	for _, f := range files { // bounded by the package (P10-02)
		ast.Inspect(f, func(n ast.Node) bool {
			// ast.Inspect CALLS BACK WITH NIL on the way up, and the split
			// below dereferences the node to position it — so what used to be
			// a type switch that quietly matched nothing became a panic. The
			// self-test found it on the first run, which is the argument for
			// having one.
			if n == nil {
				return false
			}
			if gen, ok := n.(*ast.GenDecl); ok {
				// THE DECLARATION IS NOT A USE. A const block naming a member,
				// and the struct type declaring it, are what created the
				// question; counting them as answers would make every member
				// look wired.
				return gen.Tok != token.CONST && gen.Tok != token.TYPE
			}
			note(decls, n, pos(fset, n.Pos()))
			return true
		})
	}
}

// note attributes one node to writing a member, reading one, or neither.
//
// SPLIT FROM THE WALK, and each arm is its own sentence. As one switch it
// measured 26 against P10-04's bound of 15 — a function doing eight unrelated
// jobs because they happened to share a type switch.
func note(decls map[string]*member, n ast.Node, at string) {
	switch v := n.(type) {
	case *ast.CompositeLit:
		noteComposite(decls, v, at)
	case *ast.CaseClause:
		// `case Foo:` READS it — a type switch or a value switch.
		noteEach(decls, v.List, at, asReader)
	case *ast.BinaryExpr:
		// `x == Foo`, `x >= Foo` READS it.
		noteEach(decls, []ast.Expr{v.X, v.Y}, at, asReader)
	case *ast.KeyValueExpr:
		noteKeyValue(decls, v, at)
	case *ast.AssignStmt:
		// `x = Foo` and `x := Foo` MAKE one.
		noteEach(decls, v.Rhs, at, asWriter)
	case *ast.CallExpr:
		// `f(Foo)` passes one along, which is making it someone's input.
		noteEach(decls, v.Args, at, asWriter)
	case *ast.ReturnStmt:
		noteEach(decls, v.Results, at, asWriter)
	}
}

// noteComposite handles `Foo{…}` and the members listed inside a set literal.
func noteComposite(decls map[string]*member, v *ast.CompositeLit, at string) {
	if m := decls[typeNameOf(v.Type)]; m != nil {
		m.wrote(at)
	}
	// `[]Track{AlertRail, MainTrack}` — a set written out as a literal is the
	// walk this codebase uses for precedence, and listing a member there is
	// naming it as one of the things to consider. That is a read.
	noteEach(decls, v.Elts, at, asReader)
}

// noteKeyValue handles `Foo: value` and `Field: Foo`, which are opposite.
func noteKeyValue(decls map[string]*member, v *ast.KeyValueExpr, at string) {
	// A REGISTRY keyed by the member is a discrimination: the table is how this
	// codebase branches on a closed set without a switch (originNames, the slot
	// registry, bandNames), and it is the shape INST-1 asks for.
	if m := decls[typeNameOf(v.Key)]; m != nil {
		m.read(at)
	}
	// The member as the VALUE means something is being MADE with it. Missing
	// this called every Power unwritten while `Powered{To: Running}` sat in two
	// files.
	if m := decls[typeNameOf(v.Value)]; m != nil {
		m.wrote(at)
	}
}

// how is which half of the question a use answers.
//
// NAMED asWriter/asReader, NOT wrote/read. P10 resolves by NAME, so a free
// function `wrote` calling a method `wrote` reads as recursion — the EIGHTH
// collision of this shape in this release, and every one of them a function on
// one type against a function on another. The rename is the cheap half.
type how func(*member, string)

func asWriter(m *member, at string) { m.wrote(at) }
func asReader(m *member, at string) { m.read(at) }

// noteEach records every expression that names a member.
func noteEach(decls map[string]*member, es []ast.Expr, at string, h how) {
	for _, e := range es { // bounded by the expression list (P10-02)
		if m := decls[typeNameOf(e)]; m != nil {
			h(m, at)
		}
	}
}

// typeNameOf is the bare identifier an expression names, ignoring a package
// qualifier and a pointer. "" when it names none.
// A LOOP, NOT RECURSION (P10-01), and a COUNTED one (P10-02). Every iteration
// strips exactly one pointer; the bound says how many, where it can be seen,
// rather than resting on "an expression has finite depth" two files away.
func typeNameOf(e ast.Expr) string {
	for range maxPointerDepth { // an EXPLICIT bound, not a condition (P10-02)
		switch v := e.(type) {
		case *ast.Ident:
			return v.Name
		case *ast.SelectorExpr:
			return v.Sel.Name
		case *ast.StarExpr:
			e = v.X
		default:
			return ""
		}
	}
	return ""
}

// maxPointerDepth is how many pointers this will strip before giving up.
//
// EIGHT IS SLACK, NOT A GUESS: `**T` is already exotic in this tree and `***T`
// appears nowhere in it. Past the bound the answer is "names none", which is
// the safe direction — a member the walk cannot name is simply not counted,
// and an uncounted member reports as unwired rather than as wired.
const maxPointerDepth = 8

// sortedKeys is a stable order for a set, so two runs read the same.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m { // bounded by the map (P10-02)
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func pos(fset *token.FileSet, p token.Pos) string {
	pp := fset.Position(p)
	return fmt.Sprintf("%s:%d", pp.Filename, pp.Line)
}

// The fence that marks the exemption table. An exemption is only what sits
// between these lines.
const (
	ledgerOpen  = "<!-- wires:exemptions -->"
	ledgerClose = "<!-- /wires:exemptions -->"
)

// readLedger reads the ratified exemptions: table rows BETWEEN THE FENCE, whose
// first cell is `Set.Member` in backticks.
//
// THE FENCE IS NOT TIDINESS, IT IS A HAZARD FIX, and the tool found it in its
// own ledger on day two. Without it every table row starting with a backticked
// dotted name was an exemption — so a PROSE table listing rows that had been
// CLOSED silently re-created them, and two obligations that had just been
// discharged came back as exemptions with no ratification behind them.
//
// A ledger where writing about a member exempts it is worse than no ledger: it
// fails in the direction of silence, and silence here means an unwired member
// nobody is told about.
func readLedger(path string) (map[string]bool, error) {
	out := map[string]bool{}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return out, nil // no ledger yet is not an error; it means nothing is exempt
	}
	if err != nil {
		return nil, err
	}
	inside := false
	for _, line := range strings.Split(string(b), "\n") { // bounded by the file (P10-02)
		switch t := strings.TrimSpace(line); {
		case t == ledgerOpen:
			inside = true
			continue
		case t == ledgerClose:
			inside = false
			continue
		case !inside || !strings.HasPrefix(t, "|"):
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 3 {
			continue
		}
		key := strings.Trim(strings.TrimSpace(cells[1]), "`*")
		if strings.Contains(key, ".") {
			out[key] = true
		}
	}
	return out, nil
}
