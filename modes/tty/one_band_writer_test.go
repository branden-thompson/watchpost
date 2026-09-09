package tty_test

import (
	"go/ast"
	"testing"

	"github.com/branden-thompson/watchpost/platform/singleowner"
)

// one_band_writer_test.go — FR-1.3 / F-22.
//
// D-1 and T2.3 hold that only mastercontrol writes the band. It was written
// from app/director.go AND app/ticker.go before, two files constructing the
// same takeover message, and the pair drifted.
//
// F-22 records why this cannot be a test of behaviour: mutant m50 claimed to
// guard the rule, and re-anchored to send the message directly instead of
// through mastercontrol it SURVIVED — both forms produce the identical
// observable message. Its earlier CAUGHT verdicts came from an unrelated early
// return. It was retired rather than left as a green line measuring nothing,
// and this replaces it.
func TestOnlyMastercontrolWritesTheBand(t *testing.T) {
	banned := map[string]bool{"TickerBreakingMsg": true, "TickerBreakingDoneMsg": true}
	singleowner.Check(t, "band writer",
		map[string]string{
			"app/mastercontrol.go": "the single owner (D-1, T2.3): every band write goes through cue/clearBand",
		},
		func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return false
			}
			switch tn := lit.Type.(type) {
			case *ast.Ident: // bare, inside modes/tty
				return banned[tn.Name]
			case *ast.SelectorExpr: // tty-qualified, everywhere else
				pkg, ok := tn.X.(*ast.Ident)
				return ok && pkg.Name == "tty" && banned[tn.Sel.Name]
			}
			return false
		})
}
