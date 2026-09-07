package lineup

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// I-6 — THE CLOSED SET IS DERIVED, NOT REMEMBERED.
//
// TestTheEffectSetIsClosed enumerated eight effects and omitted Escalate, which
// is live and declared in fault.go. The one test whose whole job is to notice
// the set growing a member had already failed to notice one — the same shape as
// F-18's colour register and F-30's memo key, both of which were answered by
// deriving the set instead of listing it (red team 2026-09-05).
//
// The marker is the derivation: every effect embeds isEffect, so the package's
// own source is the authority on what the set contains. The runtime list below
// still exists — an AST cannot construct a value — but it can no longer be
// INCOMPLETE, which is the failure that mattered.
func effectNamesFromSource(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing the package: %v", err)
	}
	var out []string
	for _, p := range pkgs {
		for _, f := range p.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}
				for _, fld := range st.Fields.List {
					id, ok := fld.Type.(*ast.Ident)
					if ok && len(fld.Names) == 0 && id.Name == "isEffect" {
						out = append(out, ts.Name.Name)
					}
				}
				return true
			})
		}
	}
	sort.Strings(out)
	return out
}

// everyEffect is one value of each member. The test above proves it is complete.
func everyEffect() []Effect {
	return []Effect{
		BuildCard{ID: "x"}, Speak{ID: "x"}, CueTicker{ID: "x"}, ReleaseTicker{ID: "x"},
		Duck{}, Restore{}, Tune{Ref: "KEC62"}, Escalate{ID: "x"}, Publish{},
	}
}

func TestTheEffectSetIsDerivedFromTheSource(t *testing.T) {
	want := effectNamesFromSource(t)
	if len(want) == 0 {
		t.Fatal("the derivation found no effects at all; it is measuring nothing")
	}
	got := make([]string, 0, len(everyEffect()))
	for _, e := range everyEffect() {
		got = append(got, reflect.TypeOf(e).Name())
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("the effect set and the tests that walk it disagree.\n source: %v\n tested: %v\n"+
			"An effect missing here is an effect Describe, CardOf and Holds are never checked against.", want, got)
	}
}

// EVERY MEMBER IS DESCRIBED, PLACED AND DECLARED.
//
// Describe feeding the DR-23 trace, CardOf deciding which effects keep Step's
// order (the cue before the words, DR-18) and which card a panic fails (DR-22),
// and Holds deciding what may run alongside what. Missing from CardOf, an
// effect loses its ordering and fails no card on a panic; missing from Holds it
// races the band or the bed. Neither had any test at all.
func TestEveryEffectIsDescribedAndPlaced(t *testing.T) {
	for _, e := range everyEffect() {
		name := reflect.TypeOf(e).Name()
		if Describe(e) == "" {
			t.Errorf("%s describes as an empty line; the DR-23 trace would show a blank", name)
		}
		// CardOf and Holds are TOTAL, not universal: publishing, ducking and
		// tuning deliberately name no card. What is asserted is that each was
		// DECIDED — an effect about one card must claim it, or the pump gives
		// it its own run and the cue can overtake the words.
		id, ofCard := CardOf(e)
		if ofCard && id == "" {
			t.Errorf("%s claims a card and names none", name)
		}
		if got := Holds(e); ofCard && len(got) == 0 {
			t.Errorf("%s is about card %q and holds nothing, so nothing serialises it", name, id)
		}
	}
	// THE CARD-BEARING MEMBERS, stated so a silent removal from CardOf fails
	// here rather than showing up as an out-of-order read in someone's ear.
	wantCard := map[string]bool{"BuildCard": true, "Speak": true, "CueTicker": true, "ReleaseTicker": true}
	for _, e := range everyEffect() {
		name := reflect.TypeOf(e).Name()
		if _, ofCard := CardOf(e); ofCard != wantCard[name] {
			t.Errorf("%s: CardOf says %v, the ordering rule (DR-18) says %v", name, ofCard, wantCard[name])
		}
	}
}
