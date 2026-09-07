package singleowner

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"
)

// fixture writes a tiny module tree and returns its root.
func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, src := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// matchWidget accepts a composite literal of type Widget, bare or qualified.
func matchWidget(n ast.Node) bool {
	lit, ok := n.(*ast.CompositeLit)
	if !ok {
		return false
	}
	switch tn := lit.Type.(type) {
	case *ast.Ident:
		return tn.Name == "Widget"
	case *ast.SelectorExpr:
		return tn.Sel.Name == "Widget"
	}
	return false
}

func TestCheckInFailsForANonOwner(t *testing.T) {
	root := fixture(t, map[string]string{
		"owner/owner.go":  "package owner\ntype Widget struct{}\nvar _ = Widget{}\n",
		"other/other.go":  "package other\nimport \"x/owner\"\nvar _ = owner.Widget{}\n",
		"owner/o_test.go": "package owner\nvar _ = Widget{}\n",
	})
	spy := &testing.T{}
	CheckIn(spy, root, "widget", map[string]string{"owner/owner.go": "the owner"}, matchWidget)
	if !spy.Failed() {
		t.Error("a non-owner constructing the type must fail")
	}
}

func TestCheckInPassesWhenOnlyTheOwnerAndTestsMatch(t *testing.T) {
	root := fixture(t, map[string]string{
		"owner/owner.go":  "package owner\ntype Widget struct{}\nvar _ = Widget{}\n",
		"other/o_test.go": "package other\nimport \"x/owner\"\nvar _ = owner.Widget{}\n",
	})
	spy := &testing.T{}
	CheckIn(spy, root, "widget", map[string]string{"owner/owner.go": "the owner"}, matchWidget)
	if spy.Failed() {
		t.Error("the owner plus a test-only use must pass")
	}
}

// TestABlindMatcherCannotPass is the reason this package exists. Both gates that
// preceded it went green at least once while matching nothing at all.
func TestABlindMatcherCannotPass(t *testing.T) {
	root := fixture(t, map[string]string{
		"owner/owner.go": "package owner\ntype Widget struct{}\nvar _ = Widget{}\n",
	})
	spy := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		CheckIn(spy, root, "nothing", nil, func(ast.Node) bool { return false })
	}()
	<-done
	if !spy.Failed() {
		t.Error("a matcher that finds nothing must FAIL: green would be indistinguishable from compliance")
	}
}

// TestTheQualifiedSpellingIsNotMissed pins the blindness the config gate found
// in itself: a type is bare inside its own package and qualified outside it.
func TestTheQualifiedSpellingIsNotMissed(t *testing.T) {
	root := fixture(t, map[string]string{
		"owner/owner.go": "package owner\ntype Widget struct{}\nvar _ = Widget{}\n",
		"far/far.go":     "package far\nimport \"x/owner\"\nvar _ = owner.Widget{}\n",
	})
	spy := &testing.T{}
	CheckIn(spy, root, "widget", map[string]string{"owner/owner.go": "the owner"}, matchWidget)
	if !spy.Failed() {
		t.Error("a QUALIFIED construction outside the owner must be caught")
	}
}
