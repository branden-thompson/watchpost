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
// deck now reports from the same place it reports a stop — the moment it is
// asked to carry a location, whatever medium wins.
//
// WHAT IT REPORTS CHANGED AT D-74, AND THE DEFECT DID NOT. It said
// `Powered{Running}` when the STATION's power and the OPERATOR'S MONITOR were
// one field; they are two now, and a tune starts the MONITOR. The hazard this
// pins is the same one in the new shape: a deck asked to carry a location while
// the Director believes nobody is listening will never rotate, and the station
// is permanently silent for exactly the old reason.
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

func TestAReportingStationCanActuallyStart(t *testing.T) {
	t.Setenv("WATCHPOST_MAINTRACK", "dark")
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

	if !dir.MonitorRunning() {
		t.Fatal("the deck was asked to carry a location and the Director still believes nobody is listening — " +
			"the rotation never advances and the station is permanently silent")
	}
	// AND THE STATION'S OWN POWER IS UNTOUCHED BY IT (D-74). A tune is the
	// OPERATOR listening; it must never put their station on the air, which is
	// the defect D-69 traced and this is the pin for it.
	if got := dir.Power(); got == lineup.Running {
		t.Error("a tune put the STATION on the air; listening is not broadcasting")
	}
	// AND THE NEED REACHES THE SCHEDULE, which is the other half of the report
	// and the half that is NOT about power at all.
	//
	// THE CONSOLE HAS TO PUT THE STATION ON THE AIR FIRST (D-74). It did not
	// used to: the tune declared the power itself, so the card was admitted by
	// the very event that reported it. Two declarations now, by two people — the
	// operator listening, and the operator broadcasting — and this is the second.
	dir, _ = dir.Step(lineup.Aired{To: lineup.AirProgramme})
	dir, _ = dir.Step(lineup.Powered{To: lineup.Running})
	d.tune(pinRef("A", 33.19, -117.37))
	d.engine.Halt()
	cards := dir.Lineup().Cards(lineup.MainTrack)
	if len(cards) != 1 {
		t.Fatalf("a reporting tune must put ONE report on the main track; got %d", len(cards))
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
	t.Setenv("WATCHPOST_MAINTRACK", "dark")
	d, _ := offlineDeck(t)
	dir := lineup.New(lineup.Settings{Max: 5}, time.Now())
	d.emit = func(ev lineup.Event) {
		next, _ := dir.Step(ev)
		dir = next
	}
	d.pref = tty.ModeRelay // the offline directory answers nothing, so this still falls through to a read

	d.tune(pinRef("B", 32.71, -117.16))
	d.engine.Halt()

	// THE MONITOR STARTS WHICHEVER MEDIUM WINS (D-74). It said "asking for a
	// relay is also starting the STATION" when the two powers were one field;
	// the rule it was protecting is the one that survives — the report must not
	// depend on which branch the tune takes.
	if !dir.MonitorRunning() {
		t.Fatal("asking for a relay is also starting the operator's own listening; it reported nothing")
	}
	if got := dir.Power(); got == lineup.Running {
		t.Error("and it must not put the STATION on the air")
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
				// `Monitored` SINCE D-74. The rule is unchanged — the report must
				// not sit inside a branch — and the event it walks for is the
				// one a tune now sends: the OPERATOR'S monitor, not the
				// STATION's power.
				if !ok || sel.Sel.Name != "Monitored" {
					return true
				}
				found++
				if _, direct := enclosing.(*ast.ExprStmt); !direct {
					t.Errorf("%s: the monitor report sits inside %T — whether the operator is listening "+
						"must not depend on which medium wins, and a relay that resolves would take a "+
						"different answer from one that does not", fset.Position(lit.Pos()), enclosing)
				}
				return true
			})
		}
	}
	// SILENCE IS A DISTINCT VERDICT (INST-2): no send at all means the walk
	// broke, or the report has moved out of tune again — which is the defect.
	if found != 1 {
		t.Fatalf("want exactly one Monitored report in tune; found %d", found)
	}
}
