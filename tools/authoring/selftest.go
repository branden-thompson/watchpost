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
	return append(append(histSpecimens(), deadSpecimens()...), docSpecimens()...)
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
		{"an unexported function needs no doc", `package p

// Alpha does the first thing.
func Alpha() {}

func beta() {}`, nil},
	}
}

func runSelfTest() int {
	bad := 0
	for _, s := range specimens() { // bounded by the table (P10-02)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, s.name+".go", s.src, parser.ParseComments)
		if err != nil {
			fmt.Printf("  BROKEN SPECIMEN  %s: %v\n", s.name, err)
			bad++
			continue
		}
		var got []string
		seen := map[string]bool{}
		for _, fd := range append(append(
			checkHistory(fset, f, s.name),
			checkBlankKeepAlive(fset, f, s.name)...),
			checkDocAttached(fset, f, s.name)...) {
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
