package tty

// map_colour_guard_test.go — F-197 (FR-7.3, A-107 (5)): the map's colours
// come from the library's roles and the app's theme tokens, never from a
// literal in the map's own files.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// colourLiteralExempt are the functions that may hold one, each with its
// reason: the radar-encoding table the library reads back as classes and
// never draws (go-tuiMaps D-45).
var colourLiteralExempt = map[string]string{
	"rateColour": "the rain model's encoding, read back as classes",
	"ratePNG":    "paints rateColour's table into the frame the library decodes",
}

// colourString is a string that is a colour: a hex colour, or an SGR
// colour sequence.
var colourString = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$|[34]8;[25];|\x1b\[`)

// colourLiterals are the colour literals in one file's source, each as
// "function: what", outside colourLiteralExempt.
func colourLiterals(name, src string) []string {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		return []string{name + ": " + err.Error()}
	}
	var out []string
	for _, decl := range f.Decls {
		owner := "package level"
		if fn, ok := decl.(*ast.FuncDecl); ok {
			owner = fn.Name.Name
		}
		if _, ok := colourLiteralExempt[owner]; ok {
			continue
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CompositeLit:
				if what := selectorName(n.Type); slices.Contains([]string{"tuimaps.RGB", "color.RGBA", "color.NRGBA"}, what) {
					out = append(out, owner+": "+what+"{...}")
				}
			case *ast.CallExpr:
				if selectorName(n.Fun) == "lipgloss.Color" && len(n.Args) == 1 {
					if _, lit := n.Args[0].(*ast.BasicLit); lit {
						out = append(out, owner+": lipgloss.Color of a literal")
					}
				}
			case *ast.BasicLit:
				if n.Kind == token.STRING {
					if s, err := strconv.Unquote(n.Value); err == nil && colourString.MatchString(s) {
						out = append(out, owner+": "+n.Value)
					}
				}
			}
			return true
		})
	}
	return out
}

// selectorName is "pkg.Name" for a selector expression, "" otherwise.
func selectorName(e ast.Expr) string {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return pkg.Name + "." + sel.Sel.Name
}

// TestTheMapsFilesHoldNoColourLiteral is F-197: no file of the map's, in
// this package or the app's, holds a colour literal; the check finds each
// kind planted, and passes the exempt table.
func TestTheMapsFilesHoldNoColourLiteral(t *testing.T) {
	var files []string
	for _, glob := range []string{"map_*.go", filepath.Join("..", "..", "app", "map*.go")} {
		found, err := filepath.Glob(glob)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, found...)
	}
	files = slices.DeleteFunc(files, func(f string) bool { return strings.HasSuffix(f, "_test.go") })
	if len(files) < 40 {
		t.Fatalf("the guard reads %d of the map's files; the glob is broken", len(files))
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if got := colourLiterals(f, string(src)); len(got) > 0 {
			t.Errorf("%s holds a colour literal: %v", f, got)
		}
	}
	planted := `package p
func a() { _ = tuimaps.RGB{R: 1}; _ = color.RGBA{}; _ = lipgloss.Color("212"); _ = "#ff8800"; _ = "\x1b[38;5;1m" }
func rateColour(k int) tuimaps.RGB { return tuimaps.RGB{R: uint8(k)} }
`
	want := []string{"a: tuimaps.RGB{...}", "a: color.RGBA{...}", "a: lipgloss.Color of a literal", `a: "#ff8800"`, `a: "\x1b[38;5;1m"`}
	if got := colourLiterals("planted.go", planted); !slices.Equal(got, want) {
		t.Errorf("the planted literals gave %q; want %q", got, want)
	}
}
