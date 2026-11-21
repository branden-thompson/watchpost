package main

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

// pinnedMessages is every message type the two trees 0.19.6 splits declare,
// by name, at 0.19.6 B0 (R8): the split moves where a message is declared,
// never how many there are. B2 exports the kit's tickMsg as TickMsg (design
// CK-1), one name for one.
var pinnedMessages = []string{
	"BedMsg", "BedRelaysMsg", "LineupMsg", "RadioStatusMsg", "RecentSnapshotMsg", "RelaySilentMsg",
	"SevereMsg", "SevereReadingMsg", "SnapshotMsg", "StationAreaMsg", "StationControlMsg",
	"StationFaultMsg", "StationMsg", "TickMsg", "TickerAdvanceMsg", "TickerBreakingDoneMsg",
	"TickerBreakingMsg", "TickerMsg", "VoiceNoteMsg", "castSavedMsg", "committedMsg", "forecastTickMsg",
	"historyClearedMsg", "locatePauseMsg", "locateVerdictMsg", "mapClearedMsg", "mapFeedMsg",
	"mapPanicMsg", "mapRadarAgainMsg", "mapRadarMsg", "mapReleaseMsg", "mapTempMsg", "mapTickMsg",
	"mapViewSettledMsg", "mapWorkedMsg", "propAheadMsg", "propCentreMsg", "propChartMsg",
	"propFoundMsg", "propReachMsg", "propTickMsg", "propUpdatedMsg", "resolvedMsg", "uiSavedMsg",
	"vizTickMsg",
}

// messageTypes lists the Msg types the non-test code under the two trees
// declares, by name, sorted, a name declared twice listed twice.
func messageTypes(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, tree := range []string{"app", "modes/tty"} {
		_, files, err := declset.Tree(filepath.Join("..", "..", tree))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			for _, d := range f.Decls {
				g, ok := d.(*ast.GenDecl)
				if !ok || g.Tok != token.TYPE {
					continue
				}
				for _, sp := range g.Specs {
					if ts := sp.(*ast.TypeSpec); strings.HasSuffix(ts.Name.Name, "Msg") {
						out = append(out, ts.Name.Name)
					}
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// TestTheMessageTypesArePinned is 0.19.6 R8 and RR F7: a split must not turn a
// direct call into a message, a new hop on every update the frame path pays
// for and the router must route. The set of message types the two trees
// declare is pinned by name, so a new one fails here until it is added with
// the ruling that allows it, and one that vanishes fails too.
func TestTheMessageTypesArePinned(t *testing.T) {
	got := messageTypes(t)
	if len(got) < 45 { // the trees declare 45 at 0.19.6 B0
		t.Fatalf("the walk found %d message types; it is broken", len(got))
	}
	want := append([]string(nil), pinnedMessages...)
	sort.Strings(want)
	added, removed := declset.Diff(want, got)
	if len(added)+len(removed) > 0 {
		t.Errorf("the message types are not the pinned set: new %v, gone %v. A new message where the code calls directly today is R8's to refuse; a moved one keeps its name", added, removed)
	}
}
