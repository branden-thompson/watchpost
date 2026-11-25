package tty

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"

	ionomaps "github.com/branden-thompson/go-ionomaps"
	"github.com/branden-thompson/watchpost/platform/declset"
)

// postUpdateSteps is Dashboard.update's body in order: the dispatch, the
// resume, the map's nine arms, the tick (0.19.6 MS-5, GP-3, R4). The arms depend
// on order: each reads what the one before it settled.
var postUpdateSteps = []string{
	"dispatch", "resume",
	"mapPanicked", "armMapTick", "followNight", "armPropChart", "armPropReach",
	"armPropAhead", "armPropCentre", "armPropOptions", "armPropRefresh",
	"armTick",
}

// stepsIn lists, in source order, the calls in a Dashboard.update body whose
// method name is one of the listed steps.
func stepsIn(fn *ast.FuncDecl) []string {
	var got []string
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && slices.Contains(postUpdateSteps, sel.Sel.Name) {
			got = append(got, sel.Sel.Name)
		}
		return true
	})
	return got
}

// dashboardUpdate finds Dashboard.update among parsed files.
func dashboardUpdate(files []*ast.File) *ast.FuncDecl {
	for _, f := range files {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if ok && fn.Name.Name == "update" && fn.Recv != nil && len(fn.Recv.List) == 1 {
				if id, ok := fn.Recv.List[0].Type.(*ast.Ident); ok && id.Name == "Dashboard" {
					return fn
				}
			}
		}
	}
	return nil
}

// TestThePostUpdateStepsRunInTodaysOrder is 0.19.6 MS-5 and R4: the steps after
// every message stay in Dashboard.update, at the root, in the order they run
// today, each exactly once. The map's split moves what each arm does, never
// where or when it runs. The check is seen failing on a planted swapped pair;
// it reads the listed names only, so the map's Env built beside them in 2a
// does not edit it.
func TestThePostUpdateStepsRunInTodaysOrder(t *testing.T) {
	planted := "package tty\n\nfunc (d Dashboard) update(m int) (Dashboard, int) {\n\td.dispatch(m)\n\td.armMapTick()\n\td.mapPanicked()\n\treturn d, 0\n}\n"
	pf, err := parser.ParseFile(token.NewFileSet(), "planted.go", planted, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := stepsIn(dashboardUpdate([]*ast.File{pf}))
	if !slices.Equal(got, []string{"dispatch", "armMapTick", "mapPanicked"}) {
		t.Fatalf("control: the planted swapped pair read %v; the walk must keep source order, or a swap reads as today's", got)
	}
	_, files, err := declset.Tree(".")
	if err != nil {
		t.Fatal(err)
	}
	fn := dashboardUpdate(files)
	if fn == nil {
		t.Fatal("found no Dashboard.update in the modes/tty tree")
	}
	if got := stepsIn(fn); !slices.Equal(got, postUpdateSteps) {
		t.Errorf("Dashboard.update runs %v; want %v, in that order and each once", got, postUpdateSteps)
	}
}

// unrelatedMsg is a message no part of the Dashboard answers.
type unrelatedMsg struct{}

// TestANonMapMessageStillReasksAStaleSnapshot is 0.19.6 MS-5 (GP-3's silent
// regression): armPropOptions runs after every message, not only the map's, so
// a Propagation snapshot made with other options than the Settings' is asked
// again on whatever message comes next. Today's tests call the arm directly;
// this one goes through Dashboard.Update, which is what the split could
// quietly narrow.
func TestANonMapMessageStillReasksAStaleSnapshot(t *testing.T) {
	s := &seam{answer: PropagationResult{Snapshot: stationsSnap(0, ionomaps.Fresh)}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	d = runProp(t, d, cmd)
	asked := len(s.opts)
	if asked == 0 {
		t.Fatal("the mode's first update was never asked")
	}
	d = d.withMap(d.mapModel.togglePropCorrect())
	d.mapModel.mapPane.propSess.busy = false
	m, again := d.Update(unrelatedMsg{})
	if again == nil {
		t.Fatal("a message the map does not answer returned no command; a stale snapshot was not asked again")
	}
	runProp(t, m.(Dashboard), again)
	if len(s.opts) == asked || !s.opts[len(s.opts)-1].CorrectNoReadings {
		t.Errorf("after the Setting changed, a message the map does not answer asked %d update(s) (want one more, with the correction): %+v", len(s.opts)-asked, s.opts)
	}
}
