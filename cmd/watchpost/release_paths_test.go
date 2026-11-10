package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/tools/gateoracle"
)

// TestTheReleasePathsAreDerivedFromTheLastTag is 0.19.5 D-16: the comparator
// hygiene keeps and the file a mutant sweep is promoted to are both named from
// the newest release tag, so no release has to move them by hand and no sweep
// overwrites a shipped release's record. The tag is injected, so the test does
// not depend on the clone having tags.
func TestTheReleasePathsAreDerivedFromTheLastTag(t *testing.T) {
	gateoracle.RequireMake(t)
	show := exec.Command("make", "--no-print-directory", "-s", "-f", "Makefile", "-f", "-",
		"LAST_TAG=9.9.9", "show-release-paths")
	show.Dir = filepath.Join("..", "..")
	show.Stdin = strings.NewReader("show-release-paths:\n\t@echo '$(HYGIENE_KEEP)|$(VERDICTS_RECORD)'\n")
	out, err := show.Output()
	if err != nil {
		t.Fatalf("make could not print the release paths: %v", err)
	}
	keep, record, _ := strings.Cut(strings.TrimSpace(string(out)), "|")
	if !strings.Contains(" "+keep+" ", " watchpost-9.9.9 ") {
		t.Errorf("hygiene keeps %q; want the newest tag's binary, watchpost-9.9.9", keep)
	}
	if record != "06_docs/mutants/verdicts/since-v9.9.9.log" {
		t.Errorf("a sweep is promoted to %q; want 06_docs/mutants/verdicts/since-v9.9.9.log", record)
	}
}

// TestPromoteRefusesASweepThatDidNotFinish is 0.19.5 A-26: `promote-verdicts`
// copies a sweep's log over the release's record, and the next sweep reads its
// killers from that record, so a log that is not a whole sweep must leave the
// record as it was. A whole sweep's log is promoted, named by the commit it
// judged.
func TestPromoteRefusesASweepThatDidNotFinish(t *testing.T) {
	gateoracle.RequireMake(t)
	root := filepath.Join("..", "..")
	head, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(head))
	tree, err := exec.Command("git", "-C", root, "ls-tree", "--name-only", commit, "06_docs/mutants/").Output()
	if err != nil {
		t.Fatal(err)
	}
	all := 0
	for _, f := range strings.Fields(string(tree)) {
		if strings.HasPrefix(filepath.Base(f), "m") && strings.HasSuffix(f, ".py") {
			all++
		}
	}
	header := func(n int) string {
		return fmt.Sprintf("mutant-verdicts: %d mutant(s), commit %s, 4 workers\n\nbaseline ./... ... green\n", n, commit)
	}
	summary := func(n int) string {
		return fmt.Sprintf("\nmutant-verdicts: %d mutant(s) — %d CAUGHT, 0 SURVIVED, 0 NO EVIDENCE\n", n, n)
	}
	for _, c := range []struct {
		name, log string
		promoted  bool
	}{
		{"an empty log", "", false},
		{"an interrupted sweep", header(all), false},
		{"a red baseline", fmt.Sprintf("mutant-verdicts: %d mutant(s), commit %s, 4 workers\n\nbaseline ./... ... RED — the unmutated tree is not green; fix that first\n", all, commit), false},
		{"a MUTANT_ONLY run", header(1) + "[1/1] CAUGHT   06_docs/mutants/m1.py — TestX (0.00s)\n" + summary(1), false},
		{"a whole sweep", header(all) + summary(all), true},
	} {
		dir := t.TempDir()
		record := filepath.Join(dir, "record.log")
		const before = "the last real record\n"
		if err := os.WriteFile(record, []byte(before), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "mutant-verdicts.log"), []byte(c.log), 0o644); err != nil {
			t.Fatal(err)
		}
		promote := exec.Command("make", "--no-print-directory", "-s", "promote-verdicts",
			"LAST_TAG=9.9.9", "DIST="+dir, "VERDICTS_RECORD="+record)
		promote.Dir = root
		out, err := promote.CombinedOutput()
		got, _ := os.ReadFile(record)
		switch {
		case c.promoted && (err != nil || !strings.HasPrefix(string(got), "commit: "+commit)):
			t.Errorf("%s was not promoted (%v):\n%s", c.name, err, out)
		case !c.promoted && (err == nil || string(got) != before):
			t.Errorf("%s was promoted over the record (%v):\n%s", c.name, err, out)
		}
	}
}
