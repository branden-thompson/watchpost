package tty_test

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/branden-thompson/watchpost/platform/singleowner"
)

// one_band_writer_test.go — FR-1.3 / F-22.
//
// D-1 and T2.3 hold that only mastercontrol writes the band: two files
// constructing the same takeover message drift apart.
//
// F-22 records why this cannot be a test of behaviour: sending the message
// directly instead of through mastercontrol produces the identical observable
// message, so a behavioural check cannot tell the two apart. This structural
// check is what guards the rule.
func TestOnlyMastercontrolWritesTheBand(t *testing.T) {
	banned := map[string]bool{"TickerBreakingMsg": true, "TickerBreakingDoneMsg": true}
	match := func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return false
		}
		switch tn := lit.Type.(type) {
		case *ast.Ident: // bare, inside modes/tty
			return banned[tn.Name]
		case *ast.SelectorExpr: // qualified, everywhere else
			// BY ANY QUALIFIER, not only `tty` (0.19.6 TG-6): once the band's
			// messages move to a package with another name, a stranger's
			// construction must still match, whatever its import is called.
			_, ok := tn.X.(*ast.Ident)
			return ok && banned[tn.Sel.Name]
		}
		return false
	}
	// THE PLANTED STRANGER, under a qualifier that is not `tty`.
	planted, err := parser.ParseExpr("deck.TickerBreakingMsg{}")
	if err != nil {
		t.Fatal(err)
	}
	if !match(planted) {
		t.Fatal("control: a band write under another package's name went unmatched; the guard is blind past one qualifier")
	}
	singleowner.Check(t, "band writer",
		map[string]string{
			"app/mastercontrol.go": "the single owner (D-1, T2.3): every band write goes through cue/clearBand",
		},
		match)
}
