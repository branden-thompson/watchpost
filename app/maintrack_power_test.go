package app

// THE STATION MUST BE ABLE TO START (0.16.0 P3, red team finding 1).
//
// THE DEFECT THIS PINS, AND IT WAS FATAL: the Director begins Stopped, on
// purpose, so a station comes up silent. The ONLY thing that ever told it
// otherwise was setMode's transition edge — a side effect of the DECK changing
// mode — and on the synthesised path setMode is reached only from startSynth.
// The live stage does not call startSynth. So the first need arrived at a
// stopped Director, advances(MainTrack) refused the card, no audio started, no
// mode changed, and the Director was never powered: every subsequent need was
// refused identically. A permanently silent station with a permanently empty
// lineup and no fault, because nothing failed and nothing was ever admitted.
//
// THE ASYMMETRY WAS THE DEFECT. Stop is reported where the LISTENER acts
// (radio.go, Stop); start was reported where the AUDIO happened to begin. The
// deck now reports the programme running from the same place it reports it
// stopped — the moment it is asked to carry a location, whatever medium wins.
//
// DRIVEN THROUGH tune(), NOT PAST IT. The 0.15.0 build log records being burned
// twice by driving through the wrong seam, and the red team reproduced this at
// needsRead — one level below where the fix belongs, so a test written there
// would still fail with the defect fixed.

import (
	"go/ast"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/declset"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

func TestALiveStageStationCanActuallyStart(t *testing.T) {
	t.Setenv("WATCHPOST_MAINTRACK", "live")
	d, _ := offlineDeck(t)
	dir := lineup.New(lineup.Settings{Max: 5}, time.Now())
	d.emit = func(ev lineup.Event) {
		next, _ := dir.Step(ev)
		dir = next
	}
	// The DEFAULT preference, which is the whole point: Synth is the default
	// (UAT 78/97), so on a fresh install every tune reaches the read path.
	d.pref = tty.ModeSynth

	d.tune(pinRef("A", 33.19, -117.37))
	d.engine.Halt()

	if got := dir.Power(); got != lineup.Running {
		t.Fatalf("the deck was asked to carry a location and the Director is still %v — "+
			"every main-track card is refused by advances() and the station is permanently silent", got)
	}
	cards := dir.Lineup().Cards(lineup.MainTrack)
	if len(cards) != 1 {
		t.Fatalf("a live-stage tune must put ONE report on the main track; got %d", len(cards))
	}
	if cards[0].Slot != lineup.LocationReport {
		t.Errorf("it is a location report; got %v", cards[0].Slot)
	}
}

// THE PREFERENCE DOES NOT DECIDE WHETHER THE STATION IS ON.
//
// WHAT THIS CANNOT REACH, SAID PLAINLY (INST-5): the offline fixture's relay
// directory answers nothing, so a listener who asks for a relay still falls
// through to a read here. This proves the preference is not consulted before
// the report; it does NOT exercise a live relay, and a plant that moved the
// report into the synth branch would survive it. That plant is caught
// structurally instead, by TestThePowerReportIsNotInsideABranch below — the
// rule is "before the fork", and a position is what a walk can assert.
func TestARelayTuneAlsoPowersTheDirector(t *testing.T) {
	t.Setenv("WATCHPOST_MAINTRACK", "live")
	d, _ := offlineDeck(t)
	dir := lineup.New(lineup.Settings{Max: 5}, time.Now())
	d.emit = func(ev lineup.Event) {
		next, _ := dir.Step(ev)
		dir = next
	}
	d.pref = tty.ModeRelay // the offline directory answers nothing, so this still falls through to a read

	d.tune(pinRef("B", 32.71, -117.16))
	d.engine.Halt()

	if got := dir.Power(); got != lineup.Running {
		t.Fatalf("asking for a relay is also starting the station; got %v", got)
	}
}

// A stop, and a start after it, are pinned by
// TestTheDeckReportsThatTheProgrammeIsRunning, which owns that pair and carries
// the history of the first time this broke. Not repeated here.

// THE REPORT IS NOT INSIDE A BRANCH, and that IS the rule (red team finding 1).
//
// "The programme is running" must not depend on which medium wins, so the send
// sits in `tune`'s own body and not under any condition. A behavioural test
// cannot see the difference without a live relay, which no offline fixture has
// — so the position is asserted directly, and the walk finds the statement
// rather than trusting a line number.
func TestThePowerReportIsNotInsideABranch(t *testing.T) {
	fset, files, err := declset.Files(".")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "tune" || fn.Body == nil {
				continue
			}
			// Only the statements of the function's OWN body count. A send
			// nested in an if, a for or a select is a send some configuration
			// does not make.
			top := map[ast.Node]bool{}
			for _, st := range fn.Body.List {
				top[st] = true
			}
			var enclosing ast.Stmt
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if st, ok := n.(ast.Stmt); ok && top[st] {
					enclosing = st
				}
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				sel, ok := lit.Type.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Powered" {
					return true
				}
				found++
				if _, direct := enclosing.(*ast.ExprStmt); !direct {
					t.Errorf("%s: the power report sits inside %T — whether the station is on must not "+
						"depend on which medium wins, and a listener whose relay resolves would take a "+
						"different answer from one whose does not", fset.Position(lit.Pos()), enclosing)
				}
				return true
			})
		}
	}
	// SILENCE IS A DISTINCT VERDICT (INST-2): no send at all means the walk
	// broke, or the report has moved out of tune again — which is the defect.
	if found != 1 {
		t.Fatalf("want exactly one Powered report in tune; found %d", found)
	}
}
