package main

// The self-test: prove this instrument can FAIL before anyone quotes a number
// from it.
//
// THE STANDING RULE IS "VALIDATE THE INSTRUMENT", and a comment checker is
// unusually prone to breaking it. If the phrase table stopped matching, the tool
// would print "no findings" on any tree — and that zero is exactly what a reader
// would take for good news. So every specimen below has a KNOWN answer and the
// check is that the tool gives that answer and no other.
//
// BOTH DIRECTIONS ARE TESTED, which is the half that is usually missing. A
// detector that flags everything passes a "does it catch the bad one" test
// perfectly, so the corrected forms are specimens too: a comment that explains
// the code as it stands must NOT be reported, or the tool trains people to
// delete their reasoning.
//
// THE ANTI-SPECIMENS ARE THE CATALOGUE'S OWN, taken verbatim from
// `code-authoring-standards.md` where it quotes them, so this tool and the
// written rule cannot drift apart.

import (
	"fmt"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// specimen is one synthetic file and the rules that must come back from it.
type specimen struct {
	name string
	src  string
	want []string // rule IDs, sorted; empty means "this must be clean"
}

func specimens() []specimen {
	return append(append(append(append(histSpecimens(), deadSpecimens()...), docSpecimens()...), shellSpecimens()...), okSpecimens()...)
}

// okSpecimens covers AP-OK-01 in both directions; each declares its callee, as
// the module would.
func okSpecimens() []specimen {
	const decl = "\nfunc read(p string) (int, bool) { return 0, false }\n"
	return []specimen{
		{"ok-discarded-define", "package p\n\nfunc f() int { v, _ := read(\"x\"); return v }" + decl, []string{"AP-OK-01"}},
		{"ok-discarded-assign-method", "package p\n\ntype s struct{}\n\nfunc (s) get() (string, bool) { return \"\", false }\n\nfunc f(x s) (v string) { v, _ = x.get(); return }\n", []string{"AP-OK-01"}},
		{"ok-discarded-generic", "package p\n\nfunc readAs[T any](p string) (T, bool) { var z T; return z, false }\n\nfunc f() int { v, _ := readAs[int](\"x\"); return v }\n", []string{"AP-OK-01"}},
		{"ok-ok-handled", "package p\n\nfunc f() int { if v, ok := read(\"x\"); ok { return v }; return -1 }" + decl, nil},
		{"ok-ok-error-discard-is-errchecks", "package p\n\nfunc open(p string) (int, error) { return 0, nil }\n\nfunc f() int { v, _ := open(\"x\"); return v }\n", nil},
		{"ok-ok-map-lookup", "package p\n\nfunc f(m map[string]int) int { v, _ := m[\"x\"]; return v }\n", nil},
		{"ok-ok-ambiguous-name", "package p\n\nfunc read(p string) (int, bool) { return 0, false }\n\ntype t struct{}\n\nfunc (t) read() int { return 0 }\n\nfunc f() int { v, _ := read(\"x\"); return v }\n", nil},
	}
}

// shellSpecimens covers AP-SHELL-01 in both directions.
func shellSpecimens() []specimen {
	return []specimen{
		{"shell-stub-in-string", "package p\n\nconst stub = `#!/bin/sh\nkey=\"$1\"; exit 3\n`\n", []string{"AP-SHELL-01"}},
		{"shell-env-shebang-with-body", "package p\n\nconst stub = \"#!/usr/bin/env bash\\nset -e\\n\"\n", []string{"AP-SHELL-01"}},
		{"shell-exec-sh-c", "package p\n\nimport \"os/exec\"\n\nvar c = exec.Command(\"sh\", \"-c\", \"./x.sh\")\n", []string{"AP-SHELL-01"}},
		{"shell-exec-bash-flags-then-c", "package p\n\nimport \"os/exec\"\n\nvar c = exec.Command(\"bash\", \"-e\", \"-c\", \"x\")\n", []string{"AP-SHELL-01"}},
		{"shell-ok-shebang-alone-is-a-header", "package p\n\nconst header = \"#!/usr/bin/env sh\\n\"\n", nil},
		{"shell-ok-shebang-regex", "package p\n\nconst re = `^#!/usr/bin/env (sh|bash)$`\n", nil},
		{"shell-ok-exec-a-program", "package p\n\nimport \"os/exec\"\n\nvar c = exec.Command(\"git\", \"-c\", \"user.name=x\", \"commit\")\n", nil},
		{"shell-ok-mentions-shell", "package p\n\nconst why = \"a stub is the test binary, never sh -c\"\n", nil},
	}
}

// histSpecimens covers AP-HIST-01 in both directions.
func histSpecimens() []specimen {
	return []specimen{
		{"catalogue: used-to on a trailing comment", `package p
func f() { g() // used to throw away extra output
}`, []string{"AP-HIST-01"}},
		{"catalogue: legacy compatibility", `package p
// Legacy compatibility - maintains the old API for existing code
func NewThing() int { return 0 }`, []string{"AP-HIST-01"}},
		{"catalogue: backward compatibility", `package p
func f() {
	// Try to find by header name for backward compatibility
	_ = 1
}`, []string{"AP-HIST-01"}},
		{"remediation narrating itself", `package p
// Corrected at D-160: this paragraph said the opposite for a fortnight.
func g() {}`, []string{"AP-HIST-01"}},
		{"remediation citing its reviewer", `package p
// Found by red team round 3, on the row that ratifies the import.
func h() {}`, []string{"AP-HIST-01"}},

		{"a fix narrating the defect it fixed", `package p
// Land beside an empty point was blank under the nearest rule.
func w() {}`, []string{"AP-HIST-01"}},
		{"a fix narrating what was never seen", `package p
// Each of these turns temperature off, so under Notes they were never seen.
func x() {}`, []string{"AP-HIST-01"}},
		{"a fix narrating a batch", `package p
// Batch 97 made NDFD's lattice denser; its asks took half a minute.
func y() {}`, []string{"AP-HIST-01"}},

		// THE CORRECTED FORMS MUST PASS. If these are flagged the tool is telling
		// authors to delete the reasoning, which is worse than the defect.
		{"corrected form: states current intent", `package p
func f() { g() // discard extra log output during tests
}`, nil},
		{"rationale is not history", `package p
// The boundary errs towards telling the listener: an alert with no expiry never
// expires, and one expiring exactly now is KEPT.
func k() {}`, nil},
		{"a ruling cited as authority is not narration", `package p
// One key, one meaning per surface (D-56).
func m() {}`, nil},
		{"Deprecated is the language's own directive", `package p
// Deprecated: use NewThing instead; this wraps the old API.
func OldThing() {}`, nil},
		{"the phrase inside a string literal is not a comment", `package p
func f() string { return "this used to be the old API" }`, nil},
		{"the present tense of the same facts passes", `package p
// A cell is blank only where all four of its points are; NDFD's lattice is
// denser than Open-Meteo's, and its asks go four at a time.
func z() {}`, nil},
		{"a word ENDING in used is not the phrase", `package p
// A director that refused to be built is not a schedule.
func n() {}`, nil},
	}
}

// deadSpecimens covers AP-DEAD-01.
func deadSpecimens() []specimen {
	return []specimen{
		{"blank keep-alive on a closure", `package p
func f() {
	mark := func() int { return 1 }
	_ = mark
}`, []string{"AP-DEAD-01"}},
		// A DISCARDED CALL IS A DIFFERENT THING. `_ = f()` runs something and
		// throws the answer away, which is often deliberate; `_ = x` names a
		// thing and does nothing with it at all.
		{"a discarded call is not a keep-alive", `package p
func g() error { return nil }
func f() { _ = g() }`, nil},
	}
}

// docSpecimens covers SN-02's fused-doc case.
func docSpecimens() []specimen {
	return []specimen{
		// THE REAL SHAPE: the parent's prose stacked above the helper's opening
		// sentence, which is how all four instances in this repository looked.
		// The parent's block rarely opens with "Name is"; it is several
		// paragraphs of reasoning, and only the helper names itself.
		{"a parent's doc fused above a helper's opener", `package p

// PerLocation cannot answer this: a provider that returned nothing looks exactly
// like one nobody asked about, and telling those apart is the whole of it.
// Beta copies one location's payload into the snapshot under construction.
func Beta() {}`, []string{"SN-02"}},
		{"a multi-line doc that opens with its own name is fine", `package p

// Alpha does the first thing.
//
// AND IT SAYS WHY, at length, across several lines that do not name it again.
func Alpha() {}`, nil},
		{"separated blocks are fine", `package p

// Alpha does the first thing.
func Alpha() {}

// Beta does the second thing.
func Beta() {}`, nil},
		// THE SAME FUSION ON A TYPE, A VALUE AND A FIELD, with a verb no list
		// would name.
		{"a type carrying a deleted function's doc", `package p

// render draws the frame.
// Frame keeps one screen's rows.
type Frame struct{}`, []string{"SN-02"}},
		{"a constant carrying a type's doc", `package p

// Provider implements the feed.
// maxGrids caps the memo.
const maxGrids = 240`, []string{"SN-02"}},
		{"a field documented twice", `package p

type T struct {
	// read returns whether the words ran out.
	// read installs the card and returns its error.
	read func() error
}`, []string{"SN-02"}},
		{"a block that opens with another name, mid-sentence", `package p

// TestTheAir is D-139: the model's AQI from
// contours answers the fixture.
type contours struct{}`, []string{"SN-02"}},
		{"a wrapped line that starts with the name is prose", `package p

// frameChecks keeps each frame's check. A nil
// frameChecks checks every frame.
type frameChecks struct{}`, nil},
		{"a new paragraph may open with the name", `package p

// Role is a node of the registry.
//
// Role is a plain int and crosses package boundaries.
type Role int`, nil},
		{"an unexported function needs no doc", `package p

// Alpha does the first thing.
func Alpha() {}

func beta() {}`, nil},
	}
}

// packageDocControls proves SN-03 in both directions: two package comments
// in one package are both reported, and a file header detached by a blank
// line, or a test file's package comment, is not. It returns the number wrong.
func packageDocControls() int {
	files := map[string]string{
		"p/a.go":      "// Package p does one thing.\npackage p\n",
		"p/b.go":      "// B's fragment.\npackage p\n",
		"q/a.go":      "// Package q does one thing.\npackage q\n",
		"q/b.go":      "// A file header.\n\npackage q\n",
		"q/a_test.go": "// Package q's tests.\npackage q\n",
	}
	docs := map[string][]packageDoc{}
	for name, src := range files { // bounded by the table (P10-02)
		f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ParseComments|parser.PackageClauseOnly)
		if err != nil {
			fmt.Printf("  BROKEN SPECIMEN  %s: %v\n", name, err)
			return 1
		}
		if d, ok := packageDocOf(f, name); ok {
			docs[d.pkg] = append(docs[d.pkg], d)
		}
	}
	got := map[string]bool{}
	for _, fd := range checkPackageDocs(docs) { // bounded by the findings (P10-02)
		got[fd.File] = true
	}
	if len(got) != 2 || !got["p/a.go"] || !got["p/b.go"] {
		fmt.Printf("  %-15s two package comments in one package, and none elsewhere\n      got %v\n", "MISSED", got)
		return 1
	}
	fmt.Printf("  ok              two package comments in one package, and none elsewhere\n")
	return 0
}

// hashControls proves AP-HIST-01 over `#` comments in both directions: a
// narrating comment in a Makefile is reported, and a present-tense one, a
// recipe line and a script's shebang are not. It returns the number wrong.
func hashControls() int {
	bad := checkHashHistory("# The schema used to be written here by hand.\nschema:\n\t@go run ./tools/schema\n", "Makefile")
	good := checkHashHistory("#!/usr/bin/env sh\n# The version is read from the one place that defines it.\nschema:\n\techo used to\n", "Makefile")
	if len(bad) != 1 || len(good) != 0 || !hashCommented("scripts/x.sh") || hashCommented("x.go") {
		fmt.Printf("  %-15s a narrating # comment, and none in its present-tense form\n      got %d and %d\n", "MISSED", len(bad), len(good))
		return 1
	}
	fmt.Printf("  ok              a narrating # comment, and none in its present-tense form\n")
	return 0
}

// citeControls proves AP-CITE-01 in both directions: an ID the package did
// not carry at the base is reported, and one it carried, one after a release
// number and one with a release's prefix are not. It returns the number wrong.
func citeControls() int {
	src := "package p\n\n// a new rule (D-9), an old one (D-4), 0.19.5 D-12 and MVS-D-80.\nfunc f() {}\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p/f.go", src, parser.ParseComments)
	if err != nil {
		fmt.Printf("  BROKEN SPECIMEN  citations: %v\n", err)
		return 1
	}
	got := checkCitations(fset, f, "p/f.go", citedAt{"p": {"D-4": true}})
	if len(got) != 1 || !strings.HasPrefix(got[0].Text, "D-9 ") {
		fmt.Printf("  %-15s a new bare citation, and none that is old or qualified\n      got %v\n", "MISSED", got)
		return 1
	}
	fmt.Printf("  ok              a new bare citation, and none that is old or qualified\n")
	return 0
}

// capsControls proves AP-CAPS-01 in both directions: a paragraph opening
// with an all-caps AND in a changed declaration warns, and the same paragraph
// in an unchanged one, a sentence-case "And" and a caps headline that is no
// continuation do not. It returns the number wrong.
func capsControls() int {
	src := "package p\n\n// f is changed.\nfunc f() {\n\t// AND THE RAIL IS TOLD.\n\t// And the units too.\n\t//\n\t// IT BLOCKS UNTIL THE READ ENDS.\n}\n\n// g is not.\nfunc g() {\n\t// AND THE RAIL IS TOLD.\n}\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p/f.go", src, parser.ParseComments)
	if err != nil {
		fmt.Printf("  BROKEN SPECIMEN  caps: %v\n", err)
		return 1
	}
	got := checkCapsContinuation(fset, f, "p/f.go", changedLines{"p/f.go": {4: true}})
	if len(got) != 1 || got[0].Line != 5 {
		fmt.Printf("  %-15s a caps AND opening a paragraph in a changed declaration, and none elsewhere\n      got %v\n", "MISSED", got)
		return 1
	}
	fmt.Printf("  ok              a caps AND opening a paragraph in a changed declaration, and none elsewhere\n")
	return 0
}

func runSelfTest() int {
	bad := 0
	for _, s := range specimens() { // bounded by the table (P10-02)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, s.name+".go", s.src, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			fmt.Printf("  BROKEN SPECIMEN  %s: %v\n", s.name, err)
			bad++
			continue
		}
		var got []string
		seen := map[string]bool{}
		fns := newOKFuncs()
		fns.add(f)
		for _, fd := range checkFile(fset, f, s.name, fns) {
			if !seen[fd.Rule] {
				seen[fd.Rule] = true
				got = append(got, fd.Rule)
			}
		}
		sort.Strings(got)
		want := append([]string(nil), s.want...)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			verdict := "MISSED"
			if len(got) > len(want) {
				verdict = "FALSE POSITIVE"
			}
			fmt.Printf("  %-15s %s\n      want %v, got %v\n", verdict, s.name, want, got)
			bad++
			continue
		}
		fmt.Printf("  ok              %s\n", s.name)
	}
	bad += packageDocControls()
	bad += hashControls()
	bad += citeControls()
	bad += capsControls()
	if bad > 0 {
		fmt.Printf("\nauthoring self-test: %d specimen(s) wrong — the instrument does not discriminate\n", bad)
		return 1
	}
	fmt.Printf("\nauthoring self-test: %d specimen(s) passed; the instrument discriminates\n", len(specimens()))
	fmt.Println("  Both directions: the catalogue's anti-specimens are caught, and the corrected")
	fmt.Println("  forms beside them are not — a detector that flagged everything would pass")
	fmt.Println("  half of this table and teach authors to delete their reasoning.")
	return 0
}
