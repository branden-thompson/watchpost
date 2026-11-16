package main

import (
	"os"
	"strings"
	"testing"
)

// TestOutcomesReadEveryTestAndSubtest is 0.19.6 D-36: the inventory takes a
// test's and a subtest's last outcome from `go test -json`, and ignores the
// package-level lines and the raw text a build error prints.
func TestOutcomesReadEveryTestAndSubtest(t *testing.T) {
	stream := strings.Join([]string{
		`{"Action":"run","Package":"github.com/branden-thompson/watchpost/app","Test":"TestA"}`,
		`{"Action":"pass","Package":"github.com/branden-thompson/watchpost/app","Test":"TestA"}`,
		`{"Action":"skip","Package":"github.com/branden-thompson/watchpost/app","Test":"TestB/case_one"}`,
		`{"Action":"pass","Package":"github.com/branden-thompson/watchpost/app","Test":"TestB"}`,
		`{"Action":"pass","Package":"github.com/branden-thompson/watchpost/app"}`,
		`# github.com/branden-thompson/watchpost/x [build failed]`,
	}, "\n")
	got, err := outcomes(strings.NewReader(stream))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"app\tTestA": "PASS", "app\tTestB/case_one": "SKIP", "app\tTestB": "PASS"}
	if len(got) != len(want) {
		t.Fatalf("the inventory read %v; want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%q reads %q; want %q", k, got[k], v)
		}
	}
}

// TestCompareFailsALostOrChangedTest is 0.19.6 A-30 and RR F5: a test that is
// gone, one that now skips, and one that fails each fail the comparison; a
// test that moved with its package, through the move map, or that a reviewed
// renaming names, is kept; a new test is reported.
func TestCompareFailsALostOrChangedTest(t *testing.T) {
	base := map[string]string{
		"modes/tty\tTestMoved":   "PASS",
		"modes/tty\tTestStays":   "PASS",
		"modes/tty\tTestSkips":   "PASS",
		"modes/tty\tTestGone":    "PASS",
		"app\tTestSplit":         "PASS",
		"modes/tty\tTestFailing": "PASS",
	}
	now := map[string]string{
		"modes/tty/console\tTestMoved": "PASS",
		"modes/tty\tTestStays":         "PASS",
		"modes/tty\tTestSkips":         "SKIP",
		"app/internal/deck\tTestSplit": "PASS",
		"modes/tty\tTestFailing":       "FAIL",
		"modes/tty\tTestNew":           "PASS",
	}
	accepted := map[string]string{"app\tTestSplit": "app/internal/deck\tTestSplit"}
	probs, added := compare(base, now, map[string]string{}, accepted)
	got := strings.Join(probs, "; ")
	for _, want := range []string{"TestMoved: was PASS, and is gone", "TestSkips: was PASS, now SKIP", "TestGone: was PASS, and is gone", "TestFailing: was PASS, now FAIL"} {
		if !strings.Contains(got, want) {
			t.Errorf("with no move map the comparison says %q; want it to say %q", got, want)
		}
	}
	if strings.Contains(got, "TestSplit") || strings.Contains(got, "TestStays") {
		t.Errorf("a kept test or an accepted renaming was reported: %q", got)
	}
	if len(added) == 0 || !strings.Contains(strings.Join(added, " "), "TestNew") {
		t.Errorf("the new test was not reported: %v", added)
	}
	probs, _ = compare(base, now, map[string]string{"modes/tty": "modes/tty/console"}, accepted)
	if strings.Contains(strings.Join(probs, "; "), "TestMoved") {
		t.Errorf("a test that moved with its package, through the map, was reported: %v", probs)
	}
}

// TestCompareFailsTwoTestsMergedOntoOne is 0.19.6 A-30: a move map that puts
// two packages' same-named tests onto one key would read the second as kept,
// so a lost test passed. The merge is reported instead.
func TestCompareFailsTwoTestsMergedOntoOne(t *testing.T) {
	base := map[string]string{"modes/tty\tTestDraws": "PASS", "app\tTestDraws": "PASS"}
	now := map[string]string{"app/internal/deck\tTestDraws": "PASS"}
	paths := map[string]string{"modes/tty": "app/internal/deck", "app": "app/internal/deck"}
	probs, _ := compare(base, now, paths, map[string]string{})
	if !strings.Contains(strings.Join(probs, "; "), "another baseline test maps onto too") {
		t.Errorf("two baseline tests merged onto one key were not reported: %v", probs)
	}
}

// TestReadRefusesAMalformedInventory is 0.19.6 D-36: an outcome that is not
// one the run writes, and a key listed twice, are refused, not read.
func TestReadRefusesAMalformedInventory(t *testing.T) {
	for name, body := range map[string]string{
		"an unknown outcome": "app\tTestA\tPASSED\n",
		"a key twice":        "app\tTestA\tPASS\napp\tTestA\tFAIL\n",
	} {
		p := t.TempDir() + "/inv.tsv"
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := read(p); err == nil {
			t.Errorf("%s was read as an inventory", name)
		}
	}
}
