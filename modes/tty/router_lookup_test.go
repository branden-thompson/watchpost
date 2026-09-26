package tty

// router_lookup_test.go — the search window's ANSWER has to reach the window.
//
// HUM LEAD, UAT 2026-09-14: "now location search doesn't work at all - no
// suggestion or error for invalid location; pressing <enter> does nothing …
// tried multiple valid/invalid locations multiple times - same (lack) of
// behavior."

import (
	"errors"
	"go/ast"
	"go/token"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/declset"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// typeInto drives real key presses through the Router, which is the seam the
// KEY takes (P-1). Driving handleAddKey directly would pass against a build
// whose routing drops the message — which is the defect this file is about.
func typeInto(t *testing.T, r Router, text string) Router {
	t.Helper()
	for _, ch := range text {
		m, _ := r.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		out, ok := m.(Router)
		if !ok {
			t.Fatal("the router must stay the program's model")
		}
		r = out
	}
	return r
}

// submit presses enter AND RUNS WHAT COMES BACK, then feeds the reply to the
// Router the way bubbletea does. A test that stopped at the key press would
// measure that a command was returned, not that anybody received its answer.

// enterKey is the press both lookup files send; named so neither builds its own.
func enterKey() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }

func submit(t *testing.T, r Router) Router {
	t.Helper()
	m, cmd := r.Update(enterKey())
	out, ok := m.(Router)
	if !ok {
		t.Fatal("the router must stay the program's model")
	}
	if cmd == nil {
		t.Fatal("enter in the search window returned no command: nothing was ever asked")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("the search command produced no message")
	}
	m2, _ := out.Update(msg)
	out2, ok := m2.(Router)
	if !ok {
		t.Fatal("the router must stay the program's model")
	}
	return out2
}

// openSearchOnTheConsole puts the operator on the console and opens the search
// window with `[l]`, the key the operator actually presses.
func openSearchOnTheConsole(t *testing.T, d Dashboard) Router {
	t.Helper()
	r := consoleWith(t, d)
	r = pressAction(t, r, actLookup)
	if r.observer.modal != modalAdd {
		t.Fatalf("[l] on the console must open the search window; modal is %v", r.observer.modal)
	}
	return r
}

// A LOCATION THAT DOES NOT RESOLVE MUST SAY SO. The operator gets one binary
// answer out of this window — the place is findable or it is not — and a
// window that answers neither is indistinguishable from a broken key.
func TestAnInvalidLocationSearchedFromTheConsoleSaysSo(t *testing.T) {
	d := goldenDash(t, false)
	asked := ""
	d.cfg.Resolve = func(q string) (snapshot.LocationRef, error) {
		asked = q
		return snapshot.LocationRef{}, errors.New("no match for \"zzzz\" in the offline index")
	}

	r := openSearchOnTheConsole(t, d)
	r = typeInto(t, r, "zzzz")
	r = submit(t, r)

	if asked != "zzzz" {
		t.Fatalf("the typed query never reached the resolver: asked %q", asked)
	}
	if got := r.View().Content; !strings.Contains(got, "no match") {
		t.Errorf("the operator must be able to READ the refusal on the console; frame was:\n%s", got)
	}
}

// AND ONE THAT DOES RESOLVE MUST LAND. The same routing carries both answers,
// so pinning only the failure would leave the success unmeasured.
func TestAValidLocationSearchedFromTheConsoleLands(t *testing.T) {
	d := goldenDash(t, false)
	d.cfg.Resolve = func(string) (snapshot.LocationRef, error) {
		return snapshot.LocationRef{Label: "Lone Pine, CA", Tag: "LONEPINE", Zip: "93545", Lat: 36.6, Lon: -118.06}, nil
	}

	r := openSearchOnTheConsole(t, d)
	r = typeInto(t, r, "lone pine")
	r = submit(t, r)

	if r.observer.modal == modalAdd {
		t.Error("a resolved location must close the search window; it stayed open with no answer")
	}
	if got := r.View().Content; !strings.Contains(got, "Lone Pine") {
		t.Errorf("the resolved location must reach the frame; frame was:\n%s", got)
	}
}

// THE LIST IS DERIVED, NOT KEPT BY HAND. `observerScoped` names four reply
// types, and a hand-kept list of types is the F-30 failure: the fifth one is
// added to the Dashboard, nobody remembers this switch, and that window goes
// quiet from the console exactly the way the search window did.
//
// SO THE GUARD READS THE PACKAGE. Every unexported `*Msg` the package declares
// is either an ANSWER a window is owed — in which case the Router must carry it
// to Observer — or it is excused BY NAME with the reason. An unexcused new one
// fails here, at the moment it is written, rather than in a UAT.
func TestEveryWindowReplyIsRoutedBackToTheWindow(t *testing.T) {
	// The two TICK messages are Observer's own animation cadence, not an answer
	// owed to an open window. They are excused with their reason, not omitted.
	excused := map[string]string{
		"tickMsg":    "Observer's own 300ms cadence — nothing asked for it, so nothing is owed an answer",
		"vizTickMsg": "the visualizer's 50ms frame cadence, for the same reason",
	}

	_, files, err := declset.Files(".")
	if err != nil {
		t.Fatal(err)
	}
	var found int
	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, sp := range gd.Specs {
				ts, ok := sp.(*ast.TypeSpec)
				if !ok || !strings.HasSuffix(ts.Name.Name, "Msg") || ast.IsExported(ts.Name.Name) {
					continue
				}
				name := ts.Name.Name
				found++
				if why, skip := excused[name]; skip {
					t.Logf("excused %s: %s", name, why)
					continue
				}
				if !observerScoped(reflect.New(typeOfMsg(t, name)).Elem().Interface()) {
					t.Errorf("%s is a Dashboard reply the Router does not carry back to Observer: "+
						"issued from the console it would be delivered to the console, which cannot read it. "+
						"Add it to observerScoped, or excuse it here with the reason it is owed to nobody.", name)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("the guard found no unexported Msg types at all: it is measuring nothing")
	}
}

// typeOfMsg maps a declared name to its reflect.Type. Listing the constructors
// is unavoidable — Go cannot build a value from a name — but the LIST OF NAMES
// is derived above, so a type missing from here fails loudly rather than
// silently shrinking what the guard covers.
func typeOfMsg(t *testing.T, name string) reflect.Type {
	t.Helper()
	known := map[string]any{
		"resolvedMsg":       resolvedMsg{},
		"committedMsg":      committedMsg{},
		"castSavedMsg":      castSavedMsg{},
		"uiSavedMsg":        uiSavedMsg{},
		"locatePauseMsg":    locatePauseMsg{},
		"locateVerdictMsg":  locateVerdictMsg{},
		"mapWorkedMsg":      mapWorkedMsg{},
		"mapTickMsg":        mapTickMsg{},        // 0.18.0 W2.2: the map's clock is owed to its window, not a cadence
		"mapViewSettledMsg": mapViewSettledMsg{}, // 0.18.0 D-66: the view's settling asks its window's alerts
		"mapFeedMsg":        mapFeedMsg{},
		"mapRadarMsg":       mapRadarMsg{},
		"mapClearedMsg":     mapClearedMsg{},
	}
	v, ok := known[name]
	if !ok {
		t.Fatalf("%s is declared but this guard cannot build one: add it here and decide its routing", name)
	}
	return reflect.TypeOf(v)
}
