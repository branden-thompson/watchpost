package tty

// router_lookup_test.go — the search window's ANSWER has to reach the window.
//
// HUM LEAD, UAT 2026-09-14: location search answers every query, valid or
// invalid, with a suggestion or an error, and <enter> acts on it.

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
// quiet from the console.
//
// SO THE GUARD READS THE PACKAGE. Every unexported `*Msg` the package declares
// is either an ANSWER a window is owed — in which case the Router must carry it
// to Observer — or it is excused BY NAME with the reason. An unexcused new one
// fails here, at the moment it is written, rather than in a UAT.
//
// EVERY EXPORTED ONE IS JUDGED TOO, AND EVERY NAME HERE MUST STILL EXIST
// (0.19.6 TG-6). A split exports what crosses a package line, so a reply
// exported for a subpackage to send would otherwise leave the guard with a
// green run; an exported message is one of the app's inputs, named below with
// the surface that reads it, or a cadence excused with its reason, as the kit's
// TickMsg is (0.19.6 B2). And an excuse for a type that has moved or gone is an
// excuse for nothing, which fails rather than lingers.
func TestEveryWindowReplyIsRoutedBackToTheWindow(t *testing.T) {
	// The two TICK messages are Observer's own animation cadence, not an answer
	// owed to an open window. They are excused with their reason, not omitted.
	excused := map[string]string{
		"TickMsg":    "Observer's own 300ms cadence — nothing asked for it, so nothing is owed an answer",
		"vizTickMsg": "the visualizer's 50ms frame cadence, for the same reason",
	}
	// inputs are the messages the app sends in, with the routing that delivers each.
	inputs := map[string]string{
		"SnapshotMsg":           "programScoped: the priority pipeline's publish, to both surfaces (D-59)",
		"RecentSnapshotMsg":     "programScoped: the recent pipeline's publish, to both surfaces (D-99)",
		"LineupMsg":             "Scoped: the running order",
		"StationMsg":            "Scoped: the station's power",
		"StationAreaMsg":        "Scoped: the station's area and pool",
		"BedMsg":                "Scoped: whether the bed carries",
		"BedRelaysMsg":          "the Router's default: the active surface",
		"StationFaultMsg":       "the Router's default: the active surface",
		"StationControlMsg":     "scopedMessage: the Router keeps the control it hands the console",
		"RadioStatusMsg":        "the Router's default: the active surface",
		"RelaySilentMsg":        "the Router's default: the active surface",
		"VoiceNoteMsg":          "the Router's default: the active surface",
		"SevereMsg":             "the Router's default: the active surface",
		"SevereReadingMsg":      "the Router's default: the active surface",
		"TickerMsg":             "the Router's default: the active surface",
		"TickerAdvanceMsg":      "the Router's default: the active surface",
		"TickerBreakingMsg":     "the Router's default: the active surface",
		"TickerBreakingDoneMsg": "the Router's default: the active surface",
	}
	seen := map[string]bool{}

	_, files, err := declset.Tree(".")
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
				if !ok || !strings.HasSuffix(ts.Name.Name, "Msg") {
					continue
				}
				name := ts.Name.Name
				seen[name] = true
				if why, skip := excused[name]; skip && ast.IsExported(name) {
					t.Logf("excused %s: %s", name, why)
					continue
				}
				if ast.IsExported(name) {
					if _, ok := inputs[name]; !ok {
						t.Errorf("%s is an exported message this guard has not judged: name it in inputs with the surface that reads it, "+
							"or, if it is a reply a window is owed, carry it in observerScoped", name)
					}
					continue
				}
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
	for _, names := range []map[string]string{excused, inputs} {
		for name := range names {
			if !seen[name] {
				t.Errorf("%s is named here but declared nowhere in the tree: an excuse for a type that moved or went is an excuse for nothing", name)
			}
		}
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
		"mapRadarAgainMsg":  mapRadarAgainMsg{}, // D-204: the hours ahead owed are asked again by the map window
		"mapTempMsg":        mapTempMsg{},       // W10
		"forecastTickMsg":   forecastTickMsg{},  // D-94: Forecast mode's playback is owed to the map window
		"mapClearedMsg":     mapClearedMsg{},
		"historyClearedMsg": historyClearedMsg{},
		"mapReleaseMsg":     mapReleaseMsg{},  // D-221: the map's release is owed to its window, whichever surface is shown
		"propUpdatedMsg":    propUpdatedMsg{}, // 0.19.0 W4.1: the Propagation mode's update is owed to the map window
		"propTickMsg":       propTickMsg{},    // and its refresh tick
		"propChartMsg":      propChartMsg{},   // W5.1: the chart's answers, owed to the map and its chart window
		"propReachMsg":      propReachMsg{},   // W5.2: the frequency's reach, owed to the map
		"propFoundMsg":      propFoundMsg{},   // FR-1.10: a find's answer, owed to the map and its finder
		"propAheadMsg":      propAheadMsg{},   // W5.4: the hour ahead, owed to the map
		"propCentreMsg":     propCentreMsg{},  // W5.4: the centre's answer, owed to the map
		"mapPanicMsg":       mapPanicMsg{},    // QA-11: a map command stopped a panic, and its window is marked failed
	}
	v, ok := known[name]
	if !ok {
		t.Fatalf("%s is declared but this guard cannot build one: add it here and decide its routing", name)
	}
	return reflect.TypeOf(v)
}
