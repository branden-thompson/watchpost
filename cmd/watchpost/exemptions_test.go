package main

// exemptions_test.go — ONE registry for every exemption table, and one check
// over all of them.
//
// AN EXEMPTION TABLE IS A COST, NOT A FEATURE. Nine of them had accumulated
// across three files, each individually justified, together an attack surface:
// eight near-identical "row outlived its subject" loops, two tables with no
// staleness check at all, and a meta-test whose own list of tables was
// hand-written — the enumerate-don't-discover shape this package condemns, inside
// the test written to close the tables' last hole. A blind reviewer named the
// consequence: "every table is a one-line escape, and the only thing standing
// behind it is a reviewer noticing a plausible sentence."
//
// SO EVERY TABLE REGISTERS ITSELF, and one test walks the registry. A table says
// what its subjects are and how to tell whether one still exists and whether the
// exemption is still NEEDED — and the walk asserts, for every row in every
// table: the reason is real, the subject exists, and the rule would still fire
// without the row. FR-11.2: no hand-written list of things-to-check.
//
// AND A TABLE THAT IS NOT REGISTERED IS FOUND. The package's own test files are
// parsed for package-level `map[string]string` variables; one that is not in
// the registry fails here. That is the case a listed enumeration can never
// notice — a NEW table simply never being added (A33).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// exemptionTable is one table's rows and the two questions every row must
// answer.
type exemptionTable struct {
	name string
	rows map[string]string
	// exists says whether the subject is still a thing in the tree. A row for a
	// subject that is gone has outlived it.
	exists func(t *testing.T, subject string) bool
	// stillNeeded says whether the rule would fire on the subject WITHOUT the
	// row. A row for a subject the rule already accepts is stale in the other
	// direction — it reads as a considered exception and is a no-op (A31).
	stillNeeded func(t *testing.T, subject string) bool
	// satisfied is a subject the rule already accepts. THE REGISTRY CANNOT SEE
	// INSIDE A FUNCTION, so a table registering `stillNeeded: func(…) bool {
	// return true }` would pass every row (F2); `stillNeeded(satisfied)` must be
	// false, or the function has never been shown to return false at all
	// (FR-11.6). The absent subject for `exists` is NOT the table's to choose —
	// the registry generates a nonce (L1) — because a subject the table names is
	// a subject it can special-case. `satisfied` has no such fix: only the table
	// knows what its rule accepts, and a table honest for that one subject and
	// dishonest elsewhere is a declared ceiling (L-ceiling).
	satisfied string
}

// absentNonce is a subject that exists nowhere, chosen by the registry.
const absentNonce = "__registry-absent-nonce-7f3a9c__/no/such/subject"

// registry is every table declared in this package. Tables append themselves
// at declaration, so the list is the set of declarations, not a copy of it.
var registry []*exemptionTable

// exempt registers a table and hands its rows back for use as the map it was.
func exempt(tbl *exemptionTable) map[string]string {
	registry = append(registry, tbl)
	return tbl.rows
}

// shrugs are reasons that are not reasons. The floor is deliberately low — a
// reason must be a sentence, not a judgement of its quality; judging quality is
// the reviewer's job, and this exists to put the row in front of one.
var shrugs = map[string]bool{"": true, "n/a": true, "na": true, "ok": true, "todo": true, "tbd": true, "fixme": true, "later": true, "see above": true}

const reasonFloor = 20

func TestEveryExemptionRowIsRealAndStillNeeded(t *testing.T) {
	if len(registry) < 5 {
		t.Fatalf("the registry holds %d table(s); tables have stopped registering and this check "+
			"has lost its subject", len(registry))
	}
	assertRegistry(t, registry)
	// THE SUBJECT FLOOR BELONGS HERE, over the real registry — a synthetic one-row
	// table handed to assertRegistry by the specimen table is not an emptied
	// registry, it is a specimen.
	var rows int
	for _, tbl := range registry { // bounded by the registry (P10-02)
		rows += len(tbl.rows)
	}
	if rows < 15 {
		t.Fatalf("checked %d row(s) across %d table(s); the registry has emptied and this check has "+
			"lost its subject", rows, len(registry))
	}
}

// assertRegistry is the property, separated so the specimen table can run it
// over a synthetic table.
func assertRegistry(t reporter, tables []*exemptionTable) {
	t.Helper()
	tt, _ := t.(*testing.T)      // the table functions take *testing.T; a nil one is accepted by every table here
	for _, tbl := range tables { // bounded by the registry (P10-02)
		if tbl.exists == nil || tbl.stillNeeded == nil {
			t.Errorf("table %s registered without both an `exists` and a `stillNeeded` check: a table "+
				"that cannot say whether its rows are stale is a list nobody can audit", tbl.name)
			continue
		}
		// THE NEGATIVE CONTROLS FIRST. A table whose functions cannot return false
		// passes every row and proves nothing; these two calls are the evidence
		// that they can.
		if tbl.exists(tt, absentNonce) {
			t.Errorf("table %s: exists(<a nonce that exists nowhere>) is TRUE — the function cannot return "+
				"false, so every row's existence check is worthless", tbl.name)
		}
		if tbl.satisfied == "" {
			t.Errorf("table %s registered without a `satisfied` control subject: nothing shows its "+
				"stillNeeded can ever return false (FR-11.6)", tbl.name)
		} else if tbl.stillNeeded(tt, tbl.satisfied) {
			t.Errorf("table %s: stillNeeded(%q) is TRUE for a subject declared satisfied — the function "+
				"cannot return false, so every row's staleness check is worthless", tbl.name, tbl.satisfied)
		}
		for subject, why := range tbl.rows { // bounded by the table (P10-02)
			w := strings.ToLower(strings.TrimSpace(why))
			if shrugs[w] || len(w) < reasonFloor {
				t.Errorf("%s[%q] is exempt with the reason %q.\n"+
					"An exemption with no reason costs nothing to add and reads exactly like one that was "+
					"argued for. Say what makes this row true, or delete it.", tbl.name, subject, why)
			}
			if !tbl.exists(tt, subject) {
				t.Errorf("%s[%q] is exempt (%q) and its subject no longer exists: the row outlived it.\n"+
					"Delete the row.", tbl.name, subject, why)
				continue
			}
			if !tbl.stillNeeded(tt, subject) {
				t.Errorf("%s[%q] is exempt (%q) and the rule would no longer fire on it: the row is a "+
					"no-op that reads as a considered exception.\nDelete the row.", tbl.name, subject, why)
			}
		}
	}
}

// EVERY `map[string]string` DECLARED IN THIS PACKAGE'S TESTS IS REGISTERED.
//
// A hand-written list of tables can notice a table being dropped only by
// counting rows, and cannot notice a new one at all. This parses the package's
// own test files for the declaration shape every exemption table has, and
// requires each one to be in the registry — the table list is DERIVED from the
// source (FR-11.2), so adding a table without registering it fails here rather
// than silently gaining a one-line escape nobody audits.
func TestEveryExemptionTableIsRegistered(t *testing.T) {
	registered := map[string]bool{}
	for _, tbl := range registry { // bounded by the registry (P10-02)
		registered[tbl.name] = true
	}
	declared := packageLevelStringMaps(t)
	if len(declared) < 5 {
		t.Fatalf("found %d package-level map[string]string declarations; the shape has changed and this "+
			"check has lost its subject", len(declared))
	}
	for _, name := range declared { // bounded by the declarations (P10-02)
		if contains(notATable, name) {
			continue
		}
		if !registered[name] {
			t.Errorf("%s is a package-level map[string]string and is not in the exemption registry.\n"+
				"Declare it as `var %s = exempt(&exemptionTable{name: %q, rows: map[string]string{…}, "+
				"exists: …, stillNeeded: …})`, or add it to `notATable` with the reason it is not one.",
				name, name, name)
		}
	}
	for name := range registered { // bounded by the registry (P10-02)
		if !contains(declared, name) {
			t.Errorf("the registry holds %q and no such package-level declaration exists: a table was "+
				"renamed or removed without its registration", name)
		}
	}
	// AND THE ESCAPE LIST IS ITSELF AUDITED. A name in notATable that no longer
	// exists is a row that outlived its subject, by the same rule as every table.
	for _, name := range notATable { // bounded by the escape list (P10-02)
		if !contains(declared, name) {
			t.Errorf("notATable names %q and no such declaration exists: the row outlived the subject", name)
		}
	}
}

// notATable names package-level map[string]string vars that are NOT exemption
// tables, with the reason. It is itself checked: a name here that no longer
// exists fails above by the same rule.
var notATable = []string{
	"shrugs", // reasons that are not reasons — a rule, not an exemption
}

// packageLevelStringMaps parses this package's test files and returns the names
// of every package-level `var x = map[string]string{…}` or
// `var x = exempt(&exemptionTable{…})`.
func packageLevelStringMaps(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	var out []string
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries { // bounded by the directory (P10-02)
		// EVERY .go FILE IN THE PACKAGE, not only the tests. A table declared in a
		// non-test file of package main is still a silencing table (F4).
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		for _, d := range f.Decls { // bounded by the file (P10-02)
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs { // bounded by the declaration (P10-02)
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 {
					continue
				}
				// `var x map[string]string` filled in init() has a TYPE and no value
				// (F3); `var x = map[string]string{…}` has a value. Both are tables.
				if vs.Type != nil && isStringMapType(vs.Type) {
					out = append(out, vs.Names[0].Name)
					continue
				}
				if len(vs.Values) == 1 && isStringMapOrExempt(vs.Values[0]) {
					out = append(out, vs.Names[0].Name)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// isStringMapOrExempt matches `map[string]string{…}`, `map[string]bool{…}` and
// `exempt(…)` — the three spellings a silencing table takes in this package.
func isStringMapOrExempt(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.CompositeLit:
		return isStringMapType(v.Type)
	case *ast.CallExpr:
		id, ok := v.Fun.(*ast.Ident)
		return ok && id.Name == "exempt"
	}
	return false
}

// isStringMapType is `map[string]string` or `map[string]bool` written out. A
// type ALIAS to one of those is not seen — no type-checker is loaded and one
// will not be taken for a test — and that ceiling is declared in the attack
// list (F-ceiling).
func isStringMapType(e ast.Expr) bool {
	mt, ok := e.(*ast.MapType)
	if !ok {
		return false
	}
	k, kok := mt.Key.(*ast.Ident)
	val, vok := mt.Value.(*ast.Ident)
	return kok && vok && k.Name == "string" && (val.Name == "string" || val.Name == "bool")
}

// fileExists is the `exists` check for tables whose subjects are paths.
func fileExists(t *testing.T, rel string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join("..", "..", rel))
	return err == nil
}
