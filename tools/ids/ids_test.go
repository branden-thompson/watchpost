package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// TestAnMVSRulingIsIndexedByItsBestMention pins 0.19.5 D-35's order: an
// ID's own heading, then a bold statement of what it ruled in the
// multi-voice docs, then one elsewhere, then any line in the multi-voice
// docs, then any line elsewhere. A bold ID inside a citation is a mention,
// and a bold ID in a table row with nothing after it takes the row's later
// cells.
func TestAnMVSRulingIsIndexedByItsBestMention(t *testing.T) {
	mvs := map[string]mention{}
	mentionsOf("see MVS-D-1 and MVS-D-2 here\n(as ruled, **MVS-D-3**) and so on\n", "02_features/0.16.0-broadcaster-ui/notes.md", mvs)
	mentionsOf("MVS-D-2 cited in the release\n**MVS-D-4** full maritime report · **MVS-D-5** (E-1)\n| AM-1 | **MVS-D-6** (E-2) | the question | the answer |\n", "02_features/multi-voice-support/brief.md", mvs)
	mentionsOf("# MVS-D-4 — the report's own file\n", "02_features/0.16.0-broadcaster-ui/rulings-d4.md", mvs)
	want := map[string]struct {
		rank    int
		file    string
		subject string
	}{
		"1": {rankCited, "notes.md", "see MVS-D-1 and MVS-D-2 here"},
		"2": {rankCitedHere, "brief.md", "MVS-D-2 cited in the release"},
		"3": {rankCited, "notes.md", "(as ruled, MVS-D-3) and so on"},
		"4": {rankHeading, "rulings-d4.md", "the report's own file"},
		"5": {rankCitedHere, "brief.md", "MVS-D-4 full maritime report · MVS-D-5 (E-1)"},
		"6": {rankRuledHere, "brief.md", "the question — the answer"},
	}
	for num, w := range want {
		got, ok := mvs[num]
		if !ok {
			t.Errorf("MVS-D-%s is not indexed", num)
			continue
		}
		if got.rank != w.rank || got.row.file != w.file || got.row.subject != w.subject {
			t.Errorf("MVS-D-%s = rank %d, %s, %q; want rank %d, %s, %q", num, got.rank, got.row.file, got.row.subject, w.rank, w.file, w.subject)
		}
	}
}

// TestEveryMVSRulingTheCodeCitesIsIndexed holds 0.19.5 D-35 to the code: an
// MVS-D ID in a Go comment that 06_docs/ids.md has no row for is a citation a
// junior cannot look up.
func TestEveryMVSRulingTheCodeCitesIsIndexed(t *testing.T) {
	root := filepath.Join("..", "..")
	index, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	cited := map[string]string{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "third_party" || d.Name() == "dist" || d.Name() == ".claude") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range mvsCite.FindAllStringSubmatch(string(b), -1) {
			cited[m[1]] = path
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cited) < 40 {
		t.Fatalf("found %d MVS-D IDs in the Go files; the walk is not reading the tree", len(cited))
	}
	for num, path := range cited {
		if !strings.Contains(index, "| MVS-D-"+num+" |") {
			t.Errorf("%s cites MVS-D-%s, which 06_docs/ids.md does not index", path, num)
		}
	}
}

// TestAnUntabledRulingIsIndexedByItsMention pins 0.19.5 D-37's rule for the
// release that never tabled its later rulings: a D-ID its tables do not
// define is indexed at its first bold statement in the release's folder, else
// its first mention; one its tables define is left to them, and one another
// release qualifies is not this release's.
func TestAnUntabledRulingIsIndexedByItsMention(t *testing.T) {
	rows := []row{{family: "D", num: "5", feature: untabledFeature, file: "rulings.md"}}
	files := map[string]string{
		"02_features/" + untabledFeature + "/a-log.md":  "we built D-7 first\nD-5 again\nper MVS-D-9 and 0.18.0 D-8\n",
		"02_features/" + untabledFeature + "/b-plan.md": "**D-7** the card has one window\n",
	}
	got := map[string]row{}
	for _, r := range untabledRows(rows, files) {
		got[r.num] = r
	}
	if len(got) != 1 {
		t.Fatalf("indexed %v; want only D-7 (D-5 is tabled, MVS-D-9 and 0.18.0 D-8 are another release's)", got)
	}
	if r := got["7"]; r.file != "b-plan.md" || r.line != 1 || r.subject != "Mentioned: D-7 the card has one window" {
		t.Errorf("D-7 = %s:%d %q; want its bold statement in b-plan.md:1", r.file, r.line, r.subject)
	}
}

// TestAPriorityStillDefinesAndAReferenceIsNoSubject pins two reading rules
// the blind M5 review found (0.19.5 D-37): a requirement whose first cell
// carries its priority ("FR-3.8 (P1)") still defines it, and a cell of
// references only is passed over for the first cell with a word in it.
func TestAPriorityStillDefinesAndAReferenceIsNoSubject(t *testing.T) {
	src := "| FR-3.8 *(P1)* | Tile sources are a closed list |\n| D-2 | W1.13, W3.7 | the plan's own task |\n"
	got := rowsOf(src, "02_features/observer-maps/01-objectives/requirements.md")
	if len(got) != 2 {
		t.Fatalf("got %d rows, want FR-3.8 and D-2: %+v", len(got), got)
	}
	if r := got[0]; r.family != "FR" || r.num != "3" || r.suffix != ".8" || r.subject != "Tile sources are a closed list" {
		t.Errorf("FR-3.8 read as %+v", r)
	}
	if r := got[1]; r.subject != "the plan's own task" {
		t.Errorf("D-2's subject is %q; want the cell with a word, not its references", r.subject)
	}
}
