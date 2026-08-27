package app

// Quality pass Q3 (L1-F21, L4-F5): the embedded geodata index (36 ms,
// 19 MB, 500k allocations to load) is loaded once on the dashboard path —
// the resolver and the seed list share it. This pin reads the source: one
// Load call in RunDashboard, one on the one-shot report path, none in the
// helpers that used to load their own copy.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeodataLoadsOncePerEntryPoint(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	callers := map[string]int{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(".", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(src), "geodata.Load("); n > 0 {
			callers[e.Name()] = n
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
