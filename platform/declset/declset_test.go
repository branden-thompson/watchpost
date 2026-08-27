package declset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetListsEveryTopLevelDeclarationSorted(t *testing.T) {
	dir := t.TempDir()
	src := "package p\n\nconst A, B = 1, 2\n\nvar v int\n\ntype T struct{}\n\nfunc (t *T) M() {}\n\nfunc F() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "p.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "p_test.go"), []byte("package p\n\nfunc G() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Set(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := "const A\nconst B\nfunc (*T).M\nfunc F\ntype T\nvar v"
	if strings.Join(got, "\n") != want {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), want)
	}
	if n, err := Write(dir); err != nil || n != 6 {
		t.Fatalf("write: %d %v", n, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "q.go"), []byte("package p\n\nfunc H() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	added, removed, ok, err := Compare(dir)
	if err != nil || !ok || len(added) != 1 || added[0] != "func H" || len(removed) != 0 {
		t.Fatalf("compare must report the new declaration: %v %v %v %v", added, removed, ok, err)
	}
}
