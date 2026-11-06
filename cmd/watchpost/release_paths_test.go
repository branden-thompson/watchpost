package main

import (
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
