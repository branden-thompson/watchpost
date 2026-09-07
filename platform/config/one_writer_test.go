package config_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// one_writer_test.go — FR-1.1's enforcement, and it is a MECHANISM rather than
// a comment.
//
// savePreference called itself "the one path" in a doc comment while five
// siblings bypassed it. A comment cannot fail, so it did not. FR-1.3 in this
// same release demands the band's single-writer rule be "enforced by a mechanism
// rather than a comment"; this is that demand turned on config, where the
// original defect lives.
//
// WHY AN AST WALK AND NOT A GREP. A grep matches the word in a comment, in a
// string, and in this file. The parser matches a CALL.
func TestConfigSaveHasOneCaller(t *testing.T) {
	// allowed is the single owner, plus the package's own internals. A file
	// here is a decision; anything else is a bypass.
	allowed := map[string]string{
		"platform/config": "Save's own package — Mutate wraps it, and the tests exercise it directly",
	}

	root := "../.." // from platform/config to the repo root
	var callers []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.Contains(path, "third_party") {
			return nil
		}
		f, perr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if perr != nil {
			return nil // not ours to police
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Save" {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "config" {
				return true
			}
			callers = append(callers, filepath.ToSlash(path))
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// THE CORPUS IS COUNTED OVER EVERY FILE, TESTS INCLUDED, and only NON-TEST
	// files are policed.
	//
	// The first version skipped tests entirely, and the moment the last
	// production bypass was migrated it found zero calls and could no longer
	// tell "nobody bypasses the one path" from "the matcher is broken" — the
	// matcher looks for a `config.Save` SELECTOR, and inside this package Mutate
	// calls Save unqualified. Its own empty-set guard caught that, which is the
	// second time today that guard has earned its place.
	if len(callers) == 0 {
		t.Fatal("no config.Save calls found anywhere, tests included — the matcher is broken, " +
			"so a green result here would prove nothing")
	}
	for _, c := range callers {
		if strings.HasSuffix(c, "_test.go") {
			continue // the corpus proves the matcher lives; tests may call Save directly
		}
		dir := filepath.ToSlash(filepath.Dir(strings.TrimPrefix(c, "../../")))
		if _, ok := allowed[dir]; !ok {
			t.Errorf("%s calls config.Save directly. Persisted config has ONE write path "+
				"(FR-1.1): go through config.Mutate, or add this package to the allowed set "+
				"with a written reason", c)
		}
	}
}
