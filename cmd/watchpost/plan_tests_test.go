package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// landedBatches are 0.19.6's batches whose commits have landed. A batch adds
// itself here in its closing commit, and from then on every test its plan
// section names must exist.
var landedBatches = []string{"B0", "B1", "B2", "B3"}

// planPath is 0.19.6's implementation plan.
const planPath = "06_docs/02_features/0.19.6-re-architecting/04-development/implementation-plan.md"

// namedTest is a test or fuzz target the plan names in code format, a file's
// path before it allowed: what record_test.go's declaredTests finds.
var namedTest = regexp.MustCompile("`(?:[\\w/.]+:)?((?:Test|Fuzz)\\w+)`")

// TestEveryTestALandedBatchsPlanNamesExists is 0.19.6 A-37: once a batch has
// landed, every test its section of the plan names is declared in the tree,
// so a plan that promises a test the batch did not write, or a rename that
// drops one, is seen. Seen failing on a planted section.
func TestEveryTestALandedBatchsPlanNamesExists(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("../..", planPath))
	if err != nil {
		t.Fatal(err)
	}
	declared := declaredTests(t, "../..")
	if got := missingFromSection("## B9: planted\nIt adds `TestTheDeckIsTheSameUnderRandomActs` and `TestNoSuchPlantedTest`.\n", "B9", declared); !slices.Equal(got, []string{"TestNoSuchPlantedTest"}) {
		t.Fatalf("COULD NOT RUN: a planted section's missing test was reported as %v", got)
	}
	for _, b := range landedBatches { // bounded by the landed batches (P10-02)
		for _, name := range missingFromSection(string(raw), b, declared) { // bounded by the section (P10-02)
			t.Errorf("%s's section of the plan names %s, and no test file declares it", b, name)
		}
	}
}

// missingFromSection is the tests a batch's section names that are not
// declared, in order; a section the plan lacks is reported as itself.
func missingFromSection(plan, batch string, declared map[string]bool) []string {
	start := strings.Index(plan, "## "+batch+":")
	if start < 0 {
		return []string{"(no section headed \"## " + batch + ":\")"}
	}
	section := plan[start:]
	if end := strings.Index(section[3:], "\n## "); end >= 0 {
		section = section[:end+3]
	}
	var out []string
	for _, m := range namedTest.FindAllStringSubmatch(section, -1) { // bounded by the section (P10-02)
		if !declared[m[1]] && !slices.Contains(out, m[1]) {
			out = append(out, m[1])
		}
	}
	slices.Sort(out)
	return out
}
