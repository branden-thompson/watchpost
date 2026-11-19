package main

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// streamOf is a `go test -json` stream: each test passes after logging its
// pins, given as "label=allocs,bytes".
func streamOf(t *testing.T, tests map[string][]string) string {
	t.Helper()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	put := func(e event) {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(tests)) { // bounded by the planted tests (P10-02)
		put(event{Action: "run", Package: "m/p", Test: name})
		for _, pin := range tests[name] { // bounded by the planted pins (P10-02)
			label, figures, _ := strings.Cut(pin, "=")
			allocs, bytes, _ := strings.Cut(figures, ",")
			put(event{Action: "output", Package: "m/p", Test: name, Output: "    x_test.go:9: pin " + label + ": " + allocs + " allocs, " + bytes + " bytes\n"})
		}
		put(event{Action: "pass", Package: "m/p", Test: name})
	}
	put(event{Action: "pass", Package: "m/p"})
	return b.String()
}

// testBaseline is a baseline of two pins on this platform and Go.
func testBaseline() *baseline {
	return &baseline{runs: 10, platform: "darwin/arm64", goVer: "go1.27.4", base: "e57afeaf", pins: map[string]span{
		"beat idle": {168, 168, 75_980, 75_990, false},
		"map hit":   {3_095, 3_096, 400_000, 400_100, false},
	}}
}

// judged is the verdict on a planted stream.
func judged(t *testing.T, stream string) verdict {
	t.Helper()
	s, err := readStream(strings.NewReader(stream))
	if err != nil {
		t.Fatal(err)
	}
	return judge(testBaseline(), s, "darwin/arm64", "go1.27.4")
}

// TestTheComparatorHoldsEachPinToItsBaseline is 0.19.6 C-8 and A-10 on planted
// streams: a measurement within the range holds, a rise in a count or in bytes
// past 0.1 % fails, a fall passes and is said, and every way a pin can go
// unmeasured fails.
func TestTheComparatorHoldsEachPinToItsBaseline(t *testing.T) {
	for _, c := range []struct {
		name  string
		tests map[string][]string
		fail  string // a failure that must be said, or "" for none
		fall  bool
	}{
		{"at the baseline", map[string][]string{"TestA": {"beat idle=168,75990"}, "TestB": {"map hit=3096,400100"}}, "", false},
		{"bytes inside 0.1 %", map[string][]string{"TestA": {"beat idle=168,76060"}, "TestB": {"map hit=3095,400000"}}, "", false},
		{"one allocation more", map[string][]string{"TestA": {"beat idle=169,75990"}, "TestB": {"map hit=3096,400100"}}, `"beat idle" rose`, false},
		{"bytes past 0.1 %", map[string][]string{"TestA": {"beat idle=168,76070"}, "TestB": {"map hit=3096,400100"}}, `"beat idle" rose`, false},
		{"a fall", map[string][]string{"TestA": {"beat idle=150,70000"}, "TestB": {"map hit=3096,400100"}}, "", true},
		{"a baseline pin unmeasured", map[string][]string{"TestA": {"beat idle=168,75990"}, "TestB": {"other=1,1"}}, `"map hit" is in the baseline and was not measured`, false},
		{"a measurement with no row", map[string][]string{"TestA": {"beat idle=168,75990", "new=1,1"}, "TestB": {"map hit=3096,400100"}}, `"new" was measured and has no baseline row`, false},
		{"a label reported twice", map[string][]string{"TestA": {"beat idle=168,75990"}, "TestB": {"map hit=3096,400100", "beat idle=168,75990"}}, `"beat idle" was reported twice`, false},
		{"a test that reported nothing", map[string][]string{"TestA": {"beat idle=168,75990"}, "TestB": {"map hit=3096,400100"}, "TestC": nil}, "m/p.TestC ran and reported no pin", false},
	} {
		v := judged(t, streamOf(t, c.tests))
		all := strings.Join(v.failures, "\n")
		switch {
		case c.fail == "" && len(v.failures) > 0:
			t.Errorf("%s: failed: %s", c.name, all)
		case c.fail != "" && !strings.Contains(all, c.fail):
			t.Errorf("%s: want a failure saying %q; got %q", c.name, c.fail, all)
		case c.fall != (len(v.falls) > 0):
			t.Errorf("%s: falls %q; want a fall %v", c.name, v.falls, c.fall)
		}
	}
}

// TestACountOnlyPinIsHeldByItsCount holds a pin whose bytes are not held
// (allocpin.ReportCount): its count against the baseline, its bytes never,
// and a change of form fails.
func TestACountOnlyPinIsHeldByItsCount(t *testing.T) {
	b := testBaseline()
	b.pins["tone"] = span{allocsMin: 185, allocsMax: 185, counted: true}
	line := func(allocs string) string {
		return `{"Action":"run","Package":"m/p","Test":"TestT"}` + "\n" +
			`{"Action":"output","Package":"m/p","Test":"TestT","Output":"    t_test.go:9: pin tone: ` + allocs + ` allocs (bytes not held, the clock)\n"}` + "\n" +
			`{"Action":"pass","Package":"m/p","Test":"TestT"}` + "\n"
	}
	rest := streamOf(t, map[string][]string{"TestA": {"beat idle=168,75990"}, "TestB": {"map hit=3096,400100"}})
	for _, c := range []struct{ name, stream, want string }{
		{"the count held", rest + line("185"), ""},
		{"the count rose", rest + line("186"), `"tone" rose`},
		{"the bytes reported again", rest + strings.ReplaceAll(line("185"), " allocs (bytes not held, the clock)", " allocs, 21680 bytes"), `"tone" changed form`},
	} {
		s, err := readStream(strings.NewReader(c.stream))
		if err != nil {
			t.Fatal(err)
		}
		v := judge(b, s, "darwin/arm64", "go1.27.4")
		all := strings.Join(v.failures, "\n")
		if (c.want == "") != (all == "") || !strings.Contains(all, c.want) {
			t.Errorf("%s: failures %q; want %q", c.name, all, c.want)
		}
	}
}

// TestTheComparatorFailsWhatGoTestFailed holds the stream's own failures: a
// failed test, a skipped one, a build that broke, an empty stream.
func TestTheComparatorFailsWhatGoTestFailed(t *testing.T) {
	ok := streamOf(t, map[string][]string{"TestA": {"beat idle=168,75990"}, "TestB": {"map hit=3096,400100"}})
	for _, c := range []struct{ name, stream, want string }{
		{"a failed test", strings.Replace(ok, `"Action":"pass","Package":"m/p","Test":"TestA"`, `"Action":"fail","Package":"m/p","Test":"TestA"`, 1), "m/p.TestA failed"},
		{"a skipped test", strings.Replace(ok, `"Action":"pass","Package":"m/p","Test":"TestA"`, `"Action":"skip","Package":"m/p","Test":"TestA"`, 1), "m/p.TestA skipped"},
		{"a broken build", ok + `{"Action":"build-fail","Package":"m/q"}` + "\n", "m/q failed to build"},
	} {
		v := judged(t, c.stream)
		if all := strings.Join(v.failures, "\n"); !strings.Contains(all, c.want) {
			t.Errorf("%s: want %q; got %q", c.name, c.want, all)
		}
	}
	if _, err := readStream(strings.NewReader("")); err == nil {
		t.Error("an empty stream was read as a run")
	}
}

// TestTheComparatorKnowsItsPlatformAndGo holds the two conditions the counts
// depend on: another platform checks the stream alone and says so; another Go
// fails until the baseline is recorded again.
func TestTheComparatorKnowsItsPlatformAndGo(t *testing.T) {
	stream := streamOf(t, map[string][]string{"TestA": {"beat idle=999,1"}, "TestB": {"map hit=3096,400100"}})
	s, err := readStream(strings.NewReader(stream))
	if err != nil {
		t.Fatal(err)
	}
	if v := judge(testBaseline(), s, "linux/amd64", "go1.27.4"); len(v.failures) > 0 || !strings.Contains(v.note, "linux/amd64") {
		t.Errorf("on another platform: failures %q, note %q; want none, and a note naming it", v.failures, v.note)
	}
	if v := judge(testBaseline(), s, "darwin/arm64", "go1.28.0"); len(v.failures) != 1 || !strings.Contains(v.failures[0], "record it again") {
		t.Errorf("under another Go: %q; want the one failure asking for a new record", v.failures)
	}
}

// TestARecordedBaselineIsReadBack records a baseline from planted runs and
// reads it back: each pin's range over the runs, and the header the judge
// needs. A run that is not clean, or reports other pins, is refused.
func TestARecordedBaselineIsReadBack(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, tests map[string][]string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(streamOf(t, tests)), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	r1 := write("r1.json", map[string][]string{"TestA": {"beat idle=168,75980"}})
	r2 := write("r2.json", map[string][]string{"TestA": {"beat idle=169,75990"}})
	out := filepath.Join(dir, "base.tsv")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-record", out, "-base", "e57afeaf", r1, r2}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("record exited %d: %s", code, stderr.String())
	}
	b, err := readBaseline(out)
	if err != nil {
		t.Fatal(err)
	}
	if b.runs != 2 || b.base != "e57afeaf" || b.platform != platform() || b.pins["beat idle"] != (span{168, 169, 75_980, 75_990, false}) {
		t.Errorf("read back %+v", b)
	}
	other := write("r3.json", map[string][]string{"TestA": {"map hit=1,1"}})
	if code := run([]string{"-record", out, "-base", "e57afeaf", r1, other}, nil, &stdout, &stderr); code == 0 {
		t.Error("a run reporting other pins was recorded")
	}
}
