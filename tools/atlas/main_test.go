package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheAtlasMatchesAFreshRender is 0.19.0 W0.4 (H-3.1): the committed
// atlas is the page the documents render today, so a diagram changed
// without `go run ./tools/atlas` stops the gate.
func TestTheAtlasMatchesAFreshRender(t *testing.T) {
	repo := filepath.Join("..", "..")
	fresh, n, err := render(repo)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile(filepath.Join(repo, "06_docs", "architecture-atlas.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(committed) != fresh {
		t.Errorf("06_docs/architecture-atlas.html is not what the documents render (%d diagrams): run go run ./tools/atlas", n)
	}
}

// TestAChangedDiagramChangesTheAtlas is W0.4's planted gap: a diagram
// edited in a document renders another page, so the comparison above sees
// it.
func TestAChangedDiagramChangesTheAtlas(t *testing.T) {
	repo := t.TempDir()
	doc := filepath.Join(repo, "06_docs", "02_features", "x", "as-built.md")
	for _, dir := range []string{filepath.Dir(doc), filepath.Join(repo, "docs")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(src string) string {
		t.Helper()
		if err := os.WriteFile(doc, []byte("# A map\n\nIts lead.\n\n```mermaid\n"+src+"\n```\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, n, err := render(repo)
		if err != nil || n != 1 {
			t.Fatalf("rendered %d diagrams: %v", n, err)
		}
		return out
	}
	before, after := write("flowchart LR\n  a --> b"), write("flowchart LR\n  a --> c")
	if before == after || !strings.Contains(after, "a --&gt; c") {
		t.Error("an edited diagram renders the same page")
	}
}

// TestOnlyTheAsBuiltDiagramsAreDrawn is 0.19.5 D-15: a plan's diagram stays in
// its own document, and the atlas draws docs/ and the as-built maps alone.
func TestOnlyTheAsBuiltDiagramsAreDrawn(t *testing.T) {
	repo := t.TempDir()
	diagram := "# A map\n\nIts lead.\n\n```mermaid\nflowchart LR\n  a --> b\n```\n"
	for _, p := range []string{
		filepath.Join("06_docs", "02_features", "x", "03-architecture-design", "as-built-x.md"),
		filepath.Join("06_docs", "02_features", "x", "04-development", "plan.md"),
		filepath.Join("docs", "architecture.md"),
	} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, p), []byte(diagram), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	found, err := collect(repo)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, d := range found {
		files = append(files, filepath.ToSlash(d.file))
	}
	if len(found) != 2 || strings.Contains(strings.Join(files, " "), "plan.md") {
		t.Errorf("the atlas drew %v; want the as-built map and docs/ alone", files)
	}
}
