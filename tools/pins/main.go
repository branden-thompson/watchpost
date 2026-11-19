// pins is the allocation pins' comparator (0.19.6 B0.7, C-8, A-10, D-28): it
// holds every pin `make alloc-budget` runs to its baseline, the maximum over
// 10 runs on code identical to `e57afeaf`, so a batch that moves code keeps
// each path's cost where it was.
//
// WHY THE PINS' OWN BUDGETS ARE NOT ENOUGH. Each pin's test holds its path to
// the measurement times 1.05, the headroom a feature spends. 0.19.6 changes no
// behaviour, so it has no feature to spend it on, and a move that added 3.8 KB
// to the idle beat would pass the budget (PB-1). The comparator holds each
// count at or below the baseline's maximum and its bytes within 0.1 % of it.
//
// It reads the pins from the lines internal/allocpin.Report writes, keyed by
// label, so a test that moves package keeps its pins. It fails on a rise, on a
// baseline pin with no measurement (skipped, renamed or moved out of the
// run), on a measurement with no baseline row, on a label reported twice, on a
// selected test that reported no pin, and on any test or build that failed. A
// fall passes and is printed, for the build log to explain (0.19.6 D-28).
//
// usage:
//
//	go test -count=1 -json -run '<pins>' ./... | go run ./tools/pins -baseline <file>
//	go run ./tools/pins -record <file> -base <commit> <run.json>...
//	go run ./tools/pins -self-test -baseline <file>
//
// The baseline holds the platform and the Go it was recorded under. On another
// platform the comparator checks the stream alone and says so: the counts are
// the runtime's and the standard library's, which differ by system, and the
// pins' own budgets hold there. Under another Go it fails, since a toolchain
// moves the counts: the baseline is recorded again on the commit that moves
// it, with -record over 10 runs.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is main without the process: the arguments, the stream and the two
// outputs, and the exit code. A verdict that could not be written fails.
func run(args []string, stdin io.Reader, stdoutW, stderrW io.Writer) int {
	stdout, stderr := &out{w: stdoutW}, &out{w: stderrW}
	code := judgeRun(args, stdin, stdout, stderr)
	if stdout.err != nil || stderr.err != nil {
		return max(code, 1)
	}
	return code
}

// out is a writer that keeps its first error, so a run whose verdict could
// not be written is not taken for one that passed.
type out struct {
	w   io.Writer
	err error
}

// say writes one formatted line, once nothing before it failed.
func (o *out) say(format string, a ...any) {
	if o.err == nil {
		_, o.err = fmt.Fprintf(o.w, format+"\n", a...)
	}
}

// judgeRun is run's work, writing through the two outs.
func judgeRun(args []string, stdin io.Reader, stdout, stderr *out) int {
	fs := flag.NewFlagSet("pins", flag.ContinueOnError)
	fs.SetOutput(stderr.w)
	baseline := fs.String("baseline", "", "the baseline to hold the stream on stdin to")
	record := fs.String("record", "", "write a baseline from the run files named after the flags")
	base := fs.String("base", "", "with -record: the commit the runs measured, or the one their product code equals")
	selfTest := fs.Bool("self-test", false, "with -baseline: prove the comparator fails against that baseline, then exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *record != "" {
		return recordBaseline(*record, *base, fs.Args(), stdout, stderr)
	}
	if *baseline == "" {
		stderr.say("pins: say -baseline <file>, or -record <file> -base <commit> <run.json>...")
		return 2
	}
	b, err := readBaseline(*baseline)
	if err != nil {
		stderr.say("pins: %v", err)
		return 2
	}
	if *selfTest {
		return selfTestOn(b, stdout)
	}
	s, err := readStream(stdin)
	if err != nil {
		stderr.say("pins: %v", err)
		return 1
	}
	v := judge(b, s, platform(), runtime.Version())
	v.print(stdout)
	if len(v.failures) > 0 {
		return 1
	}
	return 0
}

// platform is the system the comparator runs on, as the baseline names one.
func platform() string { return runtime.GOOS + "/" + runtime.GOARCH }

// ---- the stream ---------------------------------------------------------------

// event is the part of a `go test -json` line the comparator reads.
type event struct {
	Action, Package, Test, Output string
}

// stream is one alloc-budget run: each pin's measurement, the tests that ran
// with how each ended, what each failed test printed, and every failure that
// was not a test's (a build, a package).
type stream struct {
	pins     map[string]cost
	twice    []string            // labels reported more than once
	ended    map[string]string   // top-level test, package-qualified -> pass, fail or skip
	reported map[string]int      // top-level test -> pins it reported
	output   map[string][]string // top-level test -> its lines, for a failure
	broken   []string            // a failed build or package
	events   int
}

// cost is one pin's measurement; counted is a pin whose bytes are not held
// (allocpin.ReportCount).
type cost struct {
	allocs, bytes uint64
	counted       bool
}

// pinLine is internal/allocpin.Report's line, and ReportCount's, as t.Logf
// prints them.
var pinLine = regexp.MustCompile(`^\s*\S+\.go:\d+: pin ([^:\n]+): (\d+) allocs(?:, (\d+) bytes| \(bytes not held, [^)]*\))\s*$`)

// maxEvents bounds a stream: one alloc-budget run is a few thousand events.
const maxEvents = 1 << 22

// readStream reads a `go test -json` stream.
func readStream(r io.Reader) (*stream, error) {
	s := &stream{pins: map[string]cost{}, ended: map[string]string{}, reported: map[string]int{}, output: map[string][]string{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for range maxEvents {
		if !sc.Scan() {
			break
		}
		var e event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("the stream is not go test -json: %v: %.80s", err, sc.Text())
		}
		s.events++
		s.take(e)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if s.events == maxEvents {
		return nil, fmt.Errorf("the stream holds over %d events: it is not one alloc-budget run", maxEvents)
	}
	if err := invariant.Check(s.events > 0, "the stream is empty: go test ran nothing, or did not run"); err != nil {
		return nil, err
	}
	return s, nil
}

// take folds one event into the stream.
func (s *stream) take(e event) {
	top, _, _ := strings.Cut(e.Test, "/")
	key := e.Package + "." + top
	switch {
	case e.Action == "build-fail" || (e.Action == "fail" && e.Test == ""):
		s.broken = append(s.broken, e.Package)
	case e.Action == "output" && top != "":
		s.output[key] = append(s.output[key], e.Output)
		m := pinLine.FindStringSubmatch(e.Output)
		if m == nil {
			return
		}
		allocs, _ := strconv.ParseUint(m[2], 10, 64)
		bytes, _ := strconv.ParseUint(m[3], 10, 64)
		if _, seen := s.pins[m[1]]; seen {
			s.twice = append(s.twice, m[1])
		}
		s.pins[m[1]] = cost{allocs: allocs, bytes: bytes, counted: m[3] == ""}
		s.reported[key]++
	case (e.Action == "pass" || e.Action == "fail" || e.Action == "skip") && top != "" && top == e.Test:
		s.ended[key] = e.Action
	}
}

// ---- the baseline -------------------------------------------------------------

// baseline is the recorded range of every pin, and what it was recorded on.
type baseline struct {
	runs     int
	platform string
	goVer    string
	base     string
	pins     map[string]span
}

// span is a pin's range over the baseline's runs; counted is a pin whose
// bytes are not held, its byte columns "-".
type span struct {
	allocsMin, allocsMax, bytesMin, bytesMax uint64
	counted                                  bool
}

// readBaseline reads a baseline file: "# key<TAB>value" header lines, then a
// header row and one row a pin, label first.
func readBaseline(path string) (*baseline, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	b := &baseline{pins: map[string]span{}}
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the file (P10-02)
		f := strings.Split(line, "\t")
		switch {
		case len(f) == 2 && strings.HasPrefix(f[0], "# "):
			b.meta(strings.TrimPrefix(f[0], "# "), f[1])
		case len(f) == 5 && f[0] != "label":
			counted := f[3] == "-" && f[4] == "-"
			var n [4]uint64
			for i := range n { // the four figures (P10-02)
				if counted && i >= 2 {
					continue
				}
				if n[i], err = strconv.ParseUint(f[i+1], 10, 64); err != nil {
					return nil, fmt.Errorf("%s: %q: %v", path, line, err)
				}
			}
			b.pins[f[0]] = span{n[0], n[1], n[2], n[3], counted}
		}
	}
	complete := b.runs > 0 && b.platform != "" && b.goVer != "" && len(b.pins) > 0
	if err := invariant.Check(complete, path+" names no runs, platform, go or pin: it is not a pins baseline"); err != nil {
		return nil, err
	}
	return b, nil
}

// meta reads one header line.
func (b *baseline) meta(key, value string) {
	switch key {
	case "runs":
		b.runs, _ = strconv.Atoi(value)
	case "platform":
		b.platform = value
	case "go":
		b.goVer = value
	case "base":
		b.base = value
	}
}

// ---- the verdict --------------------------------------------------------------

// verdict is what the comparator found.
type verdict struct {
	held     int
	note     string
	falls    []string
	failures []string
}

// judge holds the stream to the baseline on the platform and Go the
// comparator runs on.
func judge(b *baseline, s *stream, plat, goVer string) verdict {
	var v verdict
	v.failures = append(v.failures, s.structural()...)
	if plat != b.platform {
		v.note = fmt.Sprintf("the baseline is %s's; on %s the stream is checked and the pins' own budgets hold alone", b.platform, plat)
		return v
	}
	if goVer != b.goVer {
		v.failures = append(v.failures, fmt.Sprintf("the baseline was recorded under %s and this is %s: a toolchain moves the counts, so record it again (-record, 10 runs) on the commit that moves it", b.goVer, goVer))
		return v
	}
	for _, label := range slices.Sorted(maps.Keys(b.pins)) { // bounded by the baseline (P10-02)
		bs, c := b.pins[label], s.pins[label]
		if _, ok := s.pins[label]; !ok {
			v.failures = append(v.failures, fmt.Sprintf("%q is in the baseline and was not measured: its test skipped, was renamed, or left the run", label))
			continue
		}
		v.held++
		if c.counted != bs.counted {
			v.failures = append(v.failures, fmt.Sprintf("%q changed form: the baseline holds its bytes %v and the run reports them %v", label, !bs.counted, !c.counted))
			continue
		}
		if bs.counted {
			c.bytes, bs.bytesMin, bs.bytesMax = 0, 0, 0
		}
		if c.allocs > bs.allocsMax || c.bytes*1000 > bs.bytesMax*1001 {
			v.failures = append(v.failures, fmt.Sprintf("%q rose: %d allocs and %d bytes, against at most %d and %d (+0.1 %%) over %d runs on %s", label, c.allocs, c.bytes, bs.allocsMax, bs.bytesMax, b.runs, b.base))
		} else if c.allocs < bs.allocsMin || c.bytes*1000 < bs.bytesMin*999 {
			v.falls = append(v.falls, fmt.Sprintf("%q fell: %d allocs and %d bytes, against at least %d and %d", label, c.allocs, c.bytes, bs.allocsMin, bs.bytesMin))
		}
	}
	for _, label := range slices.Sorted(maps.Keys(s.pins)) { // bounded by the stream (P10-02)
		if _, ok := b.pins[label]; !ok {
			v.failures = append(v.failures, fmt.Sprintf("%q was measured and has no baseline row: record it with the rest", label))
		}
	}
	return v
}

// structural is what fails the stream on any platform: a broken build or
// package, a failed or skipped test, a test that reported no pin, a label
// reported twice.
func (s *stream) structural() []string {
	var out []string
	for _, p := range s.broken { // bounded by the stream (P10-02)
		if !s.aTestFailedIn(p) {
			out = append(out, fmt.Sprintf("%s failed to build or failed outside a test", p))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(s.ended)) { // bounded by the tests run (P10-02)
		switch {
		case s.ended[key] == "fail":
			out = append(out, fmt.Sprintf("%s failed:\n%s", key, strings.Join(s.output[key], "")))
		case s.ended[key] == "skip":
			out = append(out, fmt.Sprintf("%s skipped: a pin that skips measures nothing", key))
		case s.reported[key] == 0:
			out = append(out, fmt.Sprintf("%s ran and reported no pin: report each measurement with allocpin.Report", key))
		}
	}
	for _, label := range s.twice { // bounded by the stream (P10-02)
		out = append(out, fmt.Sprintf("%q was reported twice: a label names one pin across the tree", label))
	}
	if len(s.ended) == 0 {
		out = append(out, "no test ran: the selector reaches nothing")
	}
	return out
}

// aTestFailedIn reports whether a test of the package failed, which is then
// the failure to show, not the package's own.
func (s *stream) aTestFailedIn(pkg string) bool {
	for key, how := range s.ended { // bounded by the tests run (P10-02)
		if how == "fail" && strings.HasPrefix(key, pkg+".") {
			return true
		}
	}
	return false
}

// print writes the verdict.
func (v verdict) print(w *out) {
	for _, f := range v.failures { // bounded by the verdict (P10-02)
		w.say("pins: FAIL %s", f)
	}
	for _, f := range v.falls { // bounded by the verdict (P10-02)
		w.say("pins: %s (a fall is explained in the build log, D-28)", f)
	}
	if v.note != "" {
		w.say("pins: %s", v.note)
	}
	if len(v.failures) == 0 {
		w.say("pins: OK - %d pin(s) held to the baseline, %d fell", v.held, len(v.falls))
	}
}

// ---- the self-test ------------------------------------------------------------

// selfTestOn proves the comparator can fail against the committed baseline:
// a stream of every pin at its maximum passes, and each planted fault in a
// copy of it fails (gate-controls: a gate's tool shows it can fail in this
// tree, not only in its own tests).
func selfTestOn(b *baseline, stdout *out) int {
	labels := slices.Sorted(maps.Keys(b.pins))
	first, bytesPin := labels[0], ""
	for _, l := range labels { // bounded by the baseline (P10-02)
		if !b.pins[l].counted && bytesPin == "" {
			bytesPin = l
		}
	}
	if err := invariant.Check(bytesPin != "", "the baseline holds no pin's bytes: the byte rule has nothing to plant on"); err != nil {
		stdout.say("pins self-test FAILED: %v", err)
		return 1
	}
	bump := func(p map[string]string, l string, allocs, bytes uint64) {
		p[l] = pinText(l, allocs, bytes, b.pins[l].counted)
	}
	controls := []struct {
		name  string
		plant func(pins map[string]string, extra *[]string)
	}{
		{"a count one above", func(p map[string]string, _ *[]string) {
			bump(p, first, b.pins[first].allocsMax+1, b.pins[first].bytesMax)
		}},
		{"bytes 0.2 % above", func(p map[string]string, _ *[]string) {
			bump(p, bytesPin, b.pins[bytesPin].allocsMax, b.pins[bytesPin].bytesMax+b.pins[bytesPin].bytesMax/500+1)
		}},
		{"a pin unmeasured", func(p map[string]string, _ *[]string) { delete(p, first) }},
		{"a pin with no row", func(p map[string]string, _ *[]string) { p["planted"] = pinText("planted", 1, 1, false) }},
		{"a label twice", func(p map[string]string, _ *[]string) { p["~another test's line"] = p[first] }},
		{"a test with no pin", func(_ map[string]string, x *[]string) { *x = append(*x, "TestPlantedReportsNothing") }},
	}
	if v := judge(b, plantedStream(b, nil), b.platform, b.goVer); len(v.failures) > 0 {
		stdout.say("pins self-test FAILED: the baseline's own maxima did not pass: %s", strings.Join(v.failures, "; "))
		return 1
	}
	for _, c := range controls { // bounded by the controls (P10-02)
		if v := judge(b, plantedStream(b, c.plant), b.platform, b.goVer); len(v.failures) == 0 {
			stdout.say("pins self-test FAILED: %s passed", c.name)
			return 1
		}
	}
	stdout.say("pins self-test: the baseline's maxima pass and all %d planted faults fail (OK)", len(controls))
	return 0
}

// pinText is a pin's line as allocpin writes it.
func pinText(label string, allocs, bytes uint64, counted bool) string {
	if counted {
		return fmt.Sprintf("    x_test.go:1: pin %s: %d allocs (bytes not held, planted)\n", label, allocs)
	}
	return fmt.Sprintf("    x_test.go:1: pin %s: %d allocs, %d bytes\n", label, allocs, bytes)
}

// plantedStream is a stream of every baseline pin at its maximum, each in a
// test of its own, after plant has changed the pins' lines or added tests
// that report nothing.
func plantedStream(b *baseline, plant func(pins map[string]string, extra *[]string)) *stream {
	pins := map[string]string{}
	for l, sp := range b.pins { // bounded by the baseline (P10-02)
		pins[l] = pinText(l, sp.allocsMax, sp.bytesMax, sp.counted)
	}
	var extra []string
	if plant != nil {
		plant(pins, &extra)
	}
	s := &stream{pins: map[string]cost{}, ended: map[string]string{}, reported: map[string]int{}, output: map[string][]string{}}
	for i, l := range slices.Sorted(maps.Keys(pins)) { // bounded by the pins (P10-02)
		test := fmt.Sprintf("TestPlanted%d", i)
		s.take(event{Action: "output", Package: "planted", Test: test, Output: pins[l]})
		s.take(event{Action: "pass", Package: "planted", Test: test})
	}
	for _, test := range extra { // bounded by the plant (P10-02)
		s.take(event{Action: "pass", Package: "planted", Test: test})
	}
	return s
}

// ---- recording ----------------------------------------------------------------

// recordBaseline writes a baseline from run files, each one alloc-budget
// run's `go test -json` stream: each pin's range over the runs. Every run
// must be clean and report the same pins.
func recordBaseline(path, base string, files []string, stdout, stderr *out) int {
	if base == "" || len(files) < 2 {
		stderr.say("pins: -record needs -base <commit> and at least two run files")
		return 2
	}
	pins, err := spans(files)
	if err != nil {
		stderr.say("pins: %v", err)
		return 1
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# pins: the allocation pins' baseline (0.19.6 B0.7, A-10): each pin's range over the runs, read by tools/pins\n")
	fmt.Fprintf(&b, "# runs\t%d\n# platform\t%s\n# go\t%s\n# base\t%s\n", len(files), platform(), runtime.Version(), base)
	b.WriteString("label\tallocs_min\tallocs_max\tbytes_min\tbytes_max\n")
	for _, label := range slices.Sorted(maps.Keys(pins)) { // bounded by the pins (P10-02)
		p := pins[label]
		if p.counted {
			fmt.Fprintf(&b, "%s\t%d\t%d\t-\t-\n", label, p.allocsMin, p.allocsMax)
			continue
		}
		fmt.Fprintf(&b, "%s\t%d\t%d\t%d\t%d\n", label, p.allocsMin, p.allocsMax, p.bytesMin, p.bytesMax)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		stderr.say("pins: %v", err)
		return 1
	}
	stdout.say("pins: recorded %d pin(s) over %d runs on %s to %s", len(pins), len(files), base, path)
	return 0
}

// spans is each pin's range over the run files.
func spans(files []string) (map[string]span, error) {
	pins := map[string]span{}
	for i, f := range files { // bounded by the runs (P10-02)
		r, err := os.Open(f)
		if err != nil {
			return nil, err
		}
		s, err := readStream(r)
		if cerr := r.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %v", f, err)
		}
		if bad := s.structural(); len(bad) > 0 {
			return nil, fmt.Errorf("%s is not a clean run: %s", f, strings.Join(bad, "; "))
		}
		if i > 0 && len(s.pins) != len(pins) {
			return nil, errors.New(f + " reports a different set of pins from the runs before it")
		}
		for label, c := range s.pins { // bounded by the run's pins (P10-02)
			p, seen := pins[label]
			if !seen && i > 0 {
				return nil, fmt.Errorf("%s reports %q, which the runs before it did not", f, label)
			}
			if !seen {
				p = span{c.allocs, c.allocs, c.bytes, c.bytes, c.counted}
			}
			if p.counted != c.counted {
				return nil, fmt.Errorf("%s reports %q in another form from the runs before it", f, label)
			}
			pins[label] = span{min(p.allocsMin, c.allocs), max(p.allocsMax, c.allocs), min(p.bytesMin, c.bytes), max(p.bytesMax, c.bytes), c.counted}
		}
	}
	return pins, nil
}
