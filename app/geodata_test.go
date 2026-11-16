package app

// Quality pass Q3 (L1-F21, L4-F5): the embedded geodata index (36 ms,
// 19 MB, 500k allocations to load) is loaded once on the dashboard path —
// the resolver and the seed list share it. This pin reads the source: one
// Load call in RunDashboard, one on the one-shot report path, and none in the
// helpers, which take the loaded index rather than loading a copy.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

func TestGeodataLoadsOncePerEntryPoint(t *testing.T) {
	// THE WHOLE app TREE (0.19.6 TG-2): a third load in a package below app
	// is the 36 ms and 19 MB this rule exists to stop.
	paths, err := declset.SourcePaths(".")
	if err != nil {
		t.Fatal(err)
	}
	callers := map[string]int{}
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(src), "geodata.Load("); n > 0 {
			callers[filepath.ToSlash(p)] = n
		}
	}
	want := map[string]int{"dashboard.go": 1, "app.go": 1}
	if len(callers) != len(want) {
		t.Fatalf("geodata.Load callers: %v, want exactly %v", callers, want)
	}
	for f, n := range want {
		if callers[f] != n {
			t.Fatalf("%s loads the index %d times, want %d (%v)", f, callers[f], n, callers)
		}
	}
}
