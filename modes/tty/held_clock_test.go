package tty

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAnAgeIsReadAgainstAHeldClock is the batch 28 lesson (BUILD red team,
// hosted CI): a test that builds a propagation snapshot from the wall clock
// and reads an age in its words holds the dashboard's clock, or a minute
// turning over mid-test changes the age it reads ("43 min" against 42). One
// planted without the held clock is caught.
func TestAnAgeIsReadAgainstAHeldClock(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, fn := range unheldAges(t, name, src) {
			t.Errorf("%s: %s reads an age from a wall-clock snapshot with the dashboard's clock not held (set d.now)", name, fn)
		}
	}
	planted := []byte("package tty\n\nfunc TestPlanted(t *testing.T) {\n\ts := propSnapAt(time." + "Now(), 18)\n\t_ = \"(data 42 min old)\"\n\t_ = s\n}\n") // split, so this file is not itself the catch
	if got := unheldAges(t, "planted_test.go", planted); len(got) != 1 {
		t.Errorf("a planted unheld age was not caught: %v", got)
	}
}

// unheldAges are the test functions in a file that read the wall clock and
// an age, and never hold the dashboard's clock.
func unheldAges(t *testing.T, name string, src []byte) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), name, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		body := string(src[fn.Body.Pos()-1 : fn.Body.End()-1])
		if strings.Contains(body, "time.Now()") && strings.Contains(body, "min old") && !strings.Contains(body, ".now = func") {
			out = append(out, fn.Name.Name)
		}
	}
	return out
}
