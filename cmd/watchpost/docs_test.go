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

// symbolRef matches both forms the pages use: `path/file.go:Func` and the
// package-scoped `path/pkg:Func`.
//
// THE `.go` USED TO BE MANDATORY, so 12 of the flow map's 163 refs were
// invisible to this test — including both entries rewritten in 0.14.0 (red team
// 2026-09-05, R-13). The page's own header promises it "cannot drift silently",
// and a reader's stated reason to trust it was false for the newest rows.
var symbolRef = regexp.MustCompile("`([a-z0-9_/]+(?:\\.go)?):([A-Za-z_][A-Za-z0-9_]*)`")

// declares reports whether src declares sym as a function or a method.
func declares(src, sym string) bool {
	return strings.Contains(src, "func "+sym+"(") ||
		regexp.MustCompile(`func \([^)]*\) `+sym+`\(`).MatchString(src)
}

// sourcesFor is the Go source a ref points at: one file, or every file in the
// package when the ref names a package rather than a file.
func sourcesFor(root, ref string) ([]string, error) {
	if strings.HasSuffix(ref, ".go") {
		b, err := os.ReadFile(filepath.Join(root, ref))
		if err != nil {
			return nil, err
		}
		return []string{string(b)}, nil
	}
	entries, err := os.ReadDir(filepath.Join(root, ref))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, ref, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, string(b))
	}
	return out, nil
}

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
	sources := map[string][]string{}
	pkgRefs := 0
	for _, m := range refs {
		where, sym := m[1], m[2]
		if !strings.HasSuffix(where, ".go") {
			pkgRefs++
		}
		srcs, ok := sources[where]
		if !ok {
			var err error
			if srcs, err = sourcesFor(root, where); err != nil {
				t.Errorf("%s: %v", where, err)
				continue
			}
			sources[where] = srcs
		}
		found := false
		for _, src := range srcs {
			if declares(src, sym) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s does not declare %s", where, sym)
		}
	}
	// THE WIDENING IS ITSELF PINNED. If the package-scoped form stops being
	// matched, this test goes quietly back to checking only the file-scoped
	// refs — which is the state it was in, and nothing said so.
	if pkgRefs == 0 {
		t.Error("no package-scoped `pkg:Symbol` ref was matched; the regex has narrowed back " +
			"and part of the flow map is unchecked again")
	}
}

// docs/accepted-costs.md names its sites the same way the flow map does, and
// is checked the same way — a decision that points at a function nobody can
// find is a decision nobody can honour.
//
// The page exists so a measured, chosen cost is not re-derived as a bug and
// "fixed" in a later release. The test only pins the symbols; the reason each
// entry stands, and the trigger that would re-open it, are prose and stay prose.
func TestAcceptedCostsNamesRealSymbols(t *testing.T) {
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "docs", "accepted-costs.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	refs := symbolRef.FindAllStringSubmatch(body, -1)
	if len(refs) < 8 {
		t.Fatalf("the register should name at least 8 sites, found %d", len(refs))
	}
	for _, m := range refs {
		file, sym := m[1], m[2]
		src, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		if !strings.Contains(string(src), "func "+sym+"(") &&
			!regexp.MustCompile(`func \([^)]*\) `+sym+`\(`).MatchString(string(src)) {
			t.Errorf("%s does not declare %s", file, sym)
		}
	}
	// EVERY ENTRY MUST SAY WHAT WOULD RE-OPEN IT. An accepted cost with no
	// stated trigger is not a decision, it is a shrug — and the next reader has
	// no route back other than re-arguing it from scratch.
	entries := strings.Count(body, "\n## ") - strings.Count(body, "\n## Older accepted non-decisions")
	if triggers := strings.Count(body, "**What would re-open it.**"); triggers != entries {
		t.Errorf("%d entries but %d re-open triggers — every accepted cost states its own way back", entries, triggers)
	}
}
