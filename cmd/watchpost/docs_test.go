package main

// Quality pass Q2 (JD-4, JD-5): docs/where-things-happen.md names symbols
// as `path/file.go:Func`; this test opens every one and checks the function
// (or method) is declared there, so the flow map cannot drift from the code.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var symbolRef = regexp.MustCompile("`([a-z0-9_/]+\\.go):([A-Za-z_][A-Za-z0-9_]*)`")

func TestWhereThingsHappenNamesRealSymbols(t *testing.T) {
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "docs", "where-things-happen.md"))
	if err != nil {
		t.Fatal(err)
	}
	refs := symbolRef.FindAllStringSubmatch(string(raw), -1)
	if len(refs) < 30 {
		t.Fatalf("the flow map should name at least 30 symbols, found %d", len(refs))
	}
	sources := map[string]string{}
	for _, m := range refs {
		file, sym := m[1], m[2]
		src, ok := sources[file]
		if !ok {
			b, err := os.ReadFile(filepath.Join(root, file))
			if err != nil {
				t.Errorf("%s: %v", file, err)
				continue
			}
			src = string(b)
			sources[file] = src
		}
		if !strings.Contains(src, "func "+sym+"(") && !regexp.MustCompile(`func \([^)]*\) `+sym+`\(`).MatchString(src) {
			t.Errorf("%s does not declare %s", file, sym)
		}
	}
}
