package tty

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// D-1 / FR-1.4: Broadcaster must be in STANDBY before the operator may swap
// away to Observer.
//
// THIS IS THE RELEASE'S SAFETY GATE. A swap taken while the station is live
// leaves audio running with no surface owning it, and there is no graceful
// stop anywhere in the tree to fall back on.

func routerAt(p lineup.Power, active Surface) Router {
	b := NewBroadcaster()
	b, _ = b.Update(StationMsg{Power: p})
	return Router{observer: Dashboard{}, broadcaster: b, active: active}
}

func TestSwappingAwayIsRefusedWhileTheStationIsLive(t *testing.T) {
	ok, why := routerAt(lineup.Running, SurfaceBroadcaster).canSwap(SurfaceObserver)
	if ok {
		t.Error("D-1: a swap away from a RUNNING station must be refused — there is no graceful stop " +
			"anywhere in the tree, so the audio would be left with no owner")
	}
	if strings.TrimSpace(why) == "" {
		t.Error("a refusal the operator cannot read is indistinguishable from the control being broken")
	}
}

func TestSwappingAwayIsPermittedFromStandby(t *testing.T) {
	if ok, why := routerAt(lineup.OffAir, SurfaceBroadcaster).canSwap(SurfaceObserver); !ok {
		t.Errorf("STANDBY is exactly the state the ruling asks for; the swap was refused: %q", why)
	}
}

func TestSwappingAwayIsPermittedWhenStopped(t *testing.T) {
	if ok, why := routerAt(lineup.Stopped, SurfaceBroadcaster).canSwap(SurfaceObserver); !ok {
		t.Errorf("a STOPPED station is not broadcasting either; the swap was refused: %q", why)
	}
}

func TestSwappingTOTheConsoleIsAlwaysPermitted(t *testing.T) {
	// The ruling bounds leaving a live station, not arriving at one.
	for p := lineup.Power(0); p.String() != ""; p++ {
		if ok, why := routerAt(p, SurfaceObserver).canSwap(SurfaceBroadcaster); !ok {
			t.Errorf("power=%v: arriving at the console is not the hazard the ruling bounds; refused: %q", p, why)
		}
	}
}

func TestAnUndeclaredSurfaceIsRefused(t *testing.T) {
	if ok, _ := routerAt(lineup.OffAir, SurfaceObserver).canSwap(Surface(99)); ok {
		t.Error("a surface outside the declared set must be refused — fail closed on a corrupt value")
	}
}

// The gate must be the ONE place the precondition is checked. A second
// carrier of the same rule is the shape that produced the duck-lift bug, and
// the band and the config writer each have an AST guard for exactly this.
//
// DERIVED (INST-1): it walks the package's syntax tree for reads of the
// console's power, rather than asserting against a remembered list of files.
func TestThePowerPreconditionHasOneReaderInTheRouter(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	readers := map[string]int{}
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "power" {
					return true
				}
				inner, ok := sel.X.(*ast.SelectorExpr)
				if !ok || inner.Sel.Name != "broadcaster" {
					return true
				}
				readers[name]++
				return true
			})
		}
	}
	total := 0
	for _, n := range readers {
		total += n
	}
	// SILENCE IS A DISTINCT VERDICT (INST-2): zero reads means the walk broke,
	// not that the rule is honoured.
	if total == 0 {
		t.Fatal("found no read of the console's power in this package — the check did not run, " +
			"which is not the same as passing")
	}
	if total > 1 {
		t.Errorf("the D-1 precondition has %d readers %v — it must have exactly one, in canSwap. "+
			"Two carriers of one rule is the shape that produced the duck-lift bug", total, readers)
	}
	t.Logf("checked %d read(s); blind to any reader outside package tty", total)
}
