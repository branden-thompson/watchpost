package config_test

import (
	"go/ast"
	"testing"

	"github.com/branden-thompson/watchpost/platform/singleowner"
)

// one_writer_test.go — FR-1.1's enforcement.
//
// savePreference called itself "the one path" in a doc comment while five
// siblings bypassed it. A comment cannot fail, so it never did.
//
// TIGHTENED when this moved onto platform/singleowner: the matcher now accepts
// the UNQUALIFIED spelling too, so a new function inside platform/config that
// called Save directly would be caught. Before, it could not see them — which
// is how the gate came to match nothing at all and still report green.
func TestConfigSaveHasOneCaller(t *testing.T) {
	singleowner.Check(t, "config write path",
		map[string]string{
			"platform/config/config.go": "Mutate is the one write path; it wraps Save under the lock that " +
				"makes the read-modify-write atomic (FR-1.1)",
		},
		func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return false
			}
			switch fn := call.Fun.(type) {
			case *ast.Ident: // Save(cfg) — inside platform/config
				return fn.Name == "Save"
			case *ast.SelectorExpr: // config.Save(cfg) — everywhere else
				pkg, ok := fn.X.(*ast.Ident)
				return ok && pkg.Name == "config" && fn.Sel.Name == "Save"
			}
			return false
		})
}
