package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/declset"
)

// flatScan is a structural rule's walk of one directory: the package's own
// files by os.ReadDir, declset.Files or a glob with no directory in it, given
// directly or as the first of a list of directories or globs.
var flatScan = regexp.MustCompile(`os\.ReadDir\("\."\)|declset\.Files\("\."\)|filepath\.Glob\("[^"/]*"\)|\[\]string\{"\.",|\[\]string\{"[^"/]*\*[^"/]*",`)

// flatScansIn lists the lines of a test file's source that walk one directory.
func flatScansIn(src string) []string {
	var out []string
	for i, line := range strings.Split(src, "\n") {
		if flatScan.MatchString(line) && !strings.HasPrefix(strings.TrimSpace(line), "//") {
			out = append(out, strings.TrimSpace(line)+" (line "+strconv.Itoa(i+1)+")")
		}
	}
	return out
}

// splitTreeTests is the test files of the two trees 0.19.6 splits.
func splitTreeTests(t *testing.T) []string {
	t.Helper()
	var all []string
	for _, root := range []string{"app", filepath.Join("modes", "tty")} {
		paths, err := declset.TestPaths(filepath.Join("..", "..", root))
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, paths...)
	}
	sort.Strings(all)
	return all
}

// TestNoStructuralRuleScansOnlyItsOwnDirectory is 0.19.6 TG-2: a test that
// holds a rule about `app` or `modes/tty` by walking its own directory goes
// blind, not red, to code a split moves below it. Every such walk goes through
// declset.Tree or declset.TestPaths, rooted at the package the rule was written
// for, so its scope holds whatever moves. The detector is proved on a planted
// specimen, and the scan has a fixed floor so a renamed tree cannot pass it
// vacuously.
func TestNoStructuralRuleScansOnlyItsOwnDirectory(t *testing.T) {
	planted := "func TestX(t *testing.T) {\n\tentries, _ := os.ReadDir(\".\")\n\t_, files, _ := declset.Files(\".\")\n\tnames, _ := filepath.Glob(\"*.go\")\n\t// os.ReadDir(\".\") in a comment is not a walk\n\tok, _ := filepath.Glob(\"testdata/*.golden\")\n\tfor _, dir := range []string{\".\", \"../modes/tty\"} {}\n\tfor _, g := range []string{\"map_*.go\", \"x\"} {}\n}\n"
	if got := flatScansIn(planted); len(got) != 5 {
		t.Fatalf("the detector found %d flat walks in the planted specimen, %v; want 5", len(got), got)
	}
	files := splitTreeTests(t)
	const floor = 360 // the test files of app and modes/tty at 0.19.6 B0: 363
	if len(files) < floor {
		t.Fatalf("the scan read %d test files; want at least %d, so a moved tree cannot pass it empty", len(files), floor)
	}
	var found []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, hit := range flatScansIn(string(b)) {
			rel, _ := filepath.Rel(filepath.Join("..", ".."), f)
			found = append(found, filepath.ToSlash(rel)+": "+hit)
		}
	}
	if len(found) > 0 {
		t.Errorf("%d structural rule(s) walk only their own directory; walk declset.Tree or declset.TestPaths from the package the rule was written for:\n  %s", len(found), strings.Join(found, "\n  "))
	}
}

// propertyRun is one of `make property`'s invocations: the test it names and
// the tree it searches.
var propertyRun = regexp.MustCompile(`-run '\^(Test[A-Za-z0-9_]+)\$\$' \./(\S+)/\.\.\.`)

// TestThePropertyGatesTestsExist is 0.19.6 TG-2: `make property` names its
// tests and the tree each is run over, and a `-run` that matches nothing
// passes with "no tests to run". So a property test a split moved out of its
// tree would leave the gate green and empty; this fails instead, naming it.
// The recipe itself stays a plain `go test`, so the gate oracle can judge it.
func TestThePropertyGatesTestsExist(t *testing.T) {
	root := filepath.Join("..", "..")
	mk, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	_, recipe, ok := strings.Cut(string(mk), "\nproperty:\n")
	if !ok {
		t.Fatal("the Makefile has no property target")
	}
	recipe, _, _ = strings.Cut(recipe, "\n\n")
	runs := propertyRun.FindAllStringSubmatch(recipe, -1)
	if len(runs) < 2 { // the property gate's two tests at 0.19.6 B0
		t.Fatalf("make property names %d test(s) by tree; want the 2 it was fixed on", len(runs))
	}
	for _, r := range runs {
		name, tree := r[1], r[2]
		paths, err := declset.TestPaths(filepath.Join(root, tree))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, p := range paths {
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), "func "+name+"(") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("make property runs %s over ./%s/..., where no test file declares it: the gate would pass empty", name, tree)
		}
	}
}
