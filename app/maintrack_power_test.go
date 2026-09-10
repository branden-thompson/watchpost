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
	"context"
	"go/ast"
	"go/token"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/synth"

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

// A READ THE LISTENER HAS ALREADY MOVED ON FROM MUST NOT START (N-3, C-3).
//
// FOUND BY COMPARING THE FLIP AGAINST WHAT IT REPLACED. `startSynth` opened
// with an epoch check — "a stale fallback must not relabel anything" — and took
// `tuneMu` around the second one, whose own comment records the defect it
// closed: "without it a Stop landing between the check and engine.Start left
// audio playing". The flip carried neither across, which put a previously-fixed
// race back in on the path that now carries EVERY ordinary broadcast.
func TestAReadForAGenerationThatHasMovedOnStartsNothing(t *testing.T) {
	d, _ := offlineDeck(t)
	d.mu.Lock()
	d.gen++ // the listener pressed stop, or tuned elsewhere, while this read was queued
	d.mu.Unlock()

	ok := d.readReport(context.Background(), pinRef("A", 33.19, -117.37), gen(d)-1,
		[]synth.Segment{{Key: "obs", Text: "Currently sixty-one degrees."}})

	if ok {
		t.Error("a stale read must not report success: the schedule would mark the card read")
	}
	d.mu.Lock()
	mode, src := d.mode, d.source
	d.mu.Unlock()
	if mode != "" {
		t.Errorf("nor relabel the station it is no longer on; got mode %q", mode)
	}
	if src != nil {
		t.Error("nor start audio the listener has already stopped")
	}
}

// gen is the deck's current tune epoch, for a test that needs to name one.
func gen(d *radioDeck) uint64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.gen
}

// "CHECK THE EPOCH, THEN START THE ENGINE" IS ONE STEP, AND THE LOCK IS WHAT
// MAKES IT ONE (N-3).
//
// WHY THIS IS STRUCTURAL AND NOT BEHAVIOURAL. Both halves are windows between
// two statements, and no fixture can stand in one of them: the first needs a
// Stop to land between the epoch check and `engine.StartSource`, the second
// needs one to land while a report is playing. Two plants proved it — deleting
// the pre-start check and holding the lock across the wait both SURVIVED every
// behavioural test in this package.
//
// So the rule is asserted as what it is: an ORDER of statements. The walk finds
// them rather than trusting a line number.
//
// WHAT IT STILL CANNOT SEE, SAID PLAINLY (INST-5). A walk can find the check;
// it cannot judge whether the CONDITION is honest. Neutralising it — `if false
// && !d.epoch(gen)` — leaves the call where this test looks for it and
// SURVIVES, and tightening the gate to reject that particular shape would only
// move the goalposts to `gen == gen`. The window it guards is between two
// statements and no fixture can stand in it, so this is the strongest gate
// available and its limit is recorded rather than papered over.
//
// THE TWO FAILURES IT CATCHES, AND THEY ARE OPPOSITE. Unlocking too early (or
// never checking) lets a Stop land between the check and the start, and audio
// plays on after the listener silenced the station — the defect `tuneMu`'s own
// comment records. Unlocking too LATE — holding it across the report — blocks
// Stop for the report's whole length, which is the same
// silence-that-will-not-stop by a different route.
func TestTheEpochCheckAndTheEngineStartAreOneStep(t *testing.T) {
	fset, files, err := declset.Files(".")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range files { // bounded by the package (P10-02)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "readReport" || fn.Body == nil {
				continue
			}
			// THE RECEIVER IS PART OF THE SUBJECT. `livePipelines` has a
			// `readReport` of its own, about the dashboard's fetch cycle and
			// nothing to do with audio — and a walk that took it too failed
			// this test against a function that has no business holding a tune
			// lock. Which is how the collision was found.
			if recvName(fn) != "radioDeck" {
				continue
			}
			found = true
			var lockAt, unlockAt, epochAt, startAt, selectAt token.Pos
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.DeferStmt:
					if calls(v.Call, "tuneMu", "Unlock") {
						t.Errorf("%s: the tune lock is released by DEFER, which holds it for the report's "+
							"whole length — a stop that does nothing for minutes is the same silence that "+
							"will not stop, by another route", fset.Position(v.Pos()))
					}
				case *ast.SelectStmt:
					if !selectAt.IsValid() {
						selectAt = v.Pos()
					}
				case *ast.CallExpr:
					switch {
					case calls(v, "tuneMu", "Lock"):
						lockAt = v.Pos()
					case calls(v, "tuneMu", "Unlock"):
						unlockAt = v.Pos() // the LAST one, which is what must precede the wait
					case calls(v, "engine", "StartSource"):
						startAt = v.Pos()
					case sel(v) == "epoch" && lockAt.IsValid() && !epochAt.IsValid():
						epochAt = v.Pos()
					}
				}
				return true
			})
			switch {
			case !lockAt.IsValid():
				t.Error("readReport takes no tune lock: a Stop landing between the epoch check and the " +
					"engine start leaves audio playing after the listener silenced the station")
			case !epochAt.IsValid() || epochAt < lockAt || epochAt > unlockAt:
				t.Error("the epoch is not checked INSIDE the tune lock, so the check and the start are " +
					"two steps and a Stop can land between them")
			case !startAt.IsValid() || startAt < lockAt || startAt > unlockAt:
				t.Error("the engine is not started inside the tune lock, so the check it was paired with " +
					"guarantees nothing by the time it starts")
			case selectAt.IsValid() && selectAt < unlockAt:
				t.Error("the report is waited on while the tune lock is held, which blocks Stop for the " +
					"report's whole length")
			}
		}
	}
	// SILENCE IS A DISTINCT VERDICT (INST-2).
	if !found {
		t.Fatal("readReport was not found — the walk did not run, which is not the same as passing")
	}
}

// calls reports whether this is a call to <recv>.<name> on the deck.
func calls(c *ast.CallExpr, recv, name string) bool {
	s, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || s.Sel.Name != name {
		return false
	}
	inner, ok := s.X.(*ast.SelectorExpr)
	return ok && inner.Sel.Name == recv
}

// sel is the method name a call selects, or "".
func sel(c *ast.CallExpr) string {
	if s, ok := c.Fun.(*ast.SelectorExpr); ok {
		return s.Sel.Name
	}
	return ""
}

// recvName is the type a method is declared on, without the pointer.
func recvName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	t := fn.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
