package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTheIndexIsCurrent is 0.19.5 D-11's check: 06_docs/ids.md is what the
// generator writes from today's record, so a table row added, renumbered or
// removed without regenerating fails here, in the docs lane too.
func TestTheIndexIsCurrent(t *testing.T) {
	root := filepath.Join("..", "..")
	want, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "06_docs", "ids.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Error("06_docs/ids.md is stale; run go run ./tools/ids -write and commit it")
	}
}

// TestARowDefinesAnIDByItsFirstCell pins what the index reads: the first row
// whose first cell is an ID, emphasis aside, with its first cell that says
// something as the subject, and a heading that opens with an ID; a row or a
// heading that only mentions an ID, a second row for one, and a date are not.
func TestARowDefinesAnIDByItsFirstCell(t *testing.T) {
	src := "| ID | Date | Ruling |\n|---|---|---|\n" +
		"| **D-4** | 2026-11-05 | M2 targets: 80 / 90 / 140 KB |\n" +
		"| D-4 | 2026-11-06 | a later table referring back |\n" +
		"| see D-9 | 2026-11-06 | a mention, not a definition |\n" +
		"| FR-3.2 | the first view is placed at once |\n" +
		"# D-55 — THE INJECTOR SHIPS\n" +
		"## Why D-55 matters\n"
	got := rowsOf(src, "02_features/0.19.5-quality-pass/02-analysis/rulings.md")
	if len(got) != 3 {
		t.Fatalf("got %d rows, want D-4, FR-3.2 and the D-55 heading: %+v", len(got), got)
	}
	if r := got[2]; r.family != "D" || r.num != "55" || r.subject != "THE INJECTOR SHIPS" {
		t.Errorf("the heading D-55 read as %+v", r)
	}
	if r := got[0]; r.family != "D" || r.num != "4" || r.feature != "0.19.5-quality-pass" || r.subject != "M2 targets: 80 / 90 / 140 KB" {
		t.Errorf("D-4 read as %+v", r)
	}
	if r := got[1]; r.family != "FR" || r.num != "3" || r.suffix != ".2" {
		t.Errorf("FR-3.2 read as %+v", r)
	}
}
