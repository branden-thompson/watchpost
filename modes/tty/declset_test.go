package tty

// Quality pass Q2 (plan §3 Q2, red-team R2-22): a pure file move must not
// add, drop or rename a top-level declaration. The golden was captured
// before the first move; `-update-declset` re-captures after an
// intentional change (a batch that adds code).

import (
	"flag"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

var updateDeclset = flag.Bool("update-declset", false, "re-capture testdata/declset.txt")

func TestDeclarationSetUnchanged(t *testing.T) {
	if *updateDeclset {
		n, err := declset.Write(".")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d declarations)", declset.Golden("."), n)
		return
	}
	added, removed, ok, err := declset.Compare(".")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("no golden yet: run `go test ./%s -run DeclarationSet -update-declset` before the first move", "modes/tty")
	}
	if len(added)+len(removed) > 0 {
		t.Fatalf("top-level declarations changed (a pure move must not):\n  added:   %v\n  removed: %v", added, removed)
	}
}
