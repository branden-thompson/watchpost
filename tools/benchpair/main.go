// benchpair is 0.19.6 D-28's wall-time rule: every benchmark in the trees
// 0.19.6 splits, run back to back on a base commit and on the tree, twice,
// and compared by benchstat at its pinned version. A benchmark that both
// pairs show significantly slower (p < 0.05) by 2 % or more fails, and so does
// a base benchmark that finds no pair; the result goes to the HUM LEAD.
//
// WHY PAIRED. A laptop's wall time moves with its temperature and whatever
// else runs; a figure recorded once and compared later measures the machine.
// The two arms of a pair run minutes apart on the same machine, and a
// slowdown counts only when a second pair repeats it.
//
// WHY THE PACKAGE IS IGNORED. A move changes a benchmark's package and not its
// name, so the arms are paired by name (benchstat -ignore pkg). A name that
// two packages declare would pair the wrong benchmarks, so the judge refuses
// a run where one does.
//
// usage:
//
//	go run ./tools/benchpair run -base <commit> [-count 10] [-out <dir>] <packages>...
//	go run ./tools/benchpair judge <base1.txt> <new1.txt> <base2.txt> <new2.txt>
//
// run checks the base out into a worktree under -out, runs base, tree, base,
// tree, each `go test -run '^$' -bench . -benchmem -count N`, then judges.
// It is local and HUM LEAD-run on a quiet machine, never a gate in CI.
package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// benchstat is the pinned comparison (0.19.6 D-28): a version, not latest,
// so the statistics a batch is judged by do not move under it.
const benchstat = "golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e"

// slowdown is 0.19.6 D-28's threshold, in percent, and alpha its significance.
const (
	slowdown = 2.0
	alpha    = 0.05
)

func main() {
	os.Exit(run(os.Args[1:], &out{w: os.Stdout}))
}

// out is a writer that keeps its first error.
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

// run is main without the process.
func run(args []string, o *out) int {
	if len(args) == 0 {
		o.say("benchpair: say run or judge")
		return 2
	}
	code := 2
	switch args[0] {
	case "judge":
		code = judgeFiles(args[1:], o)
	case "run":
		code = runPairs(args[1:], o)
	default:
		o.say("benchpair: say run or judge")
	}
	if o.err != nil {
		return max(code, 1)
	}
	return code
}

// ---- running ------------------------------------------------------------------

// runPairs runs the two pairs and judges them.
func runPairs(args []string, o *out) int {
	fs := flag.NewFlagSet("benchpair run", flag.ContinueOnError)
	base := fs.String("base", "", "the commit to pair against")
	count := fs.Int("count", 10, "go test -count for each arm")
	dir := fs.String("out", "dist/benchpair", "where the worktree and the arms' files go")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	pkgs := fs.Args()
	if *base == "" || len(pkgs) == 0 || *count < 2 {
		o.say("benchpair: run needs -base <commit>, a -count of at least 2, and the packages")
		return 2
	}
	tree := filepath.Join(*dir, "base")
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		o.say("benchpair: %v", err)
		return 1
	}
	if b, err := command(".", "git", "worktree", "add", "--detach", "--force", tree, *base); err != nil {
		o.say("benchpair: git worktree add: %v: %s", err, b)
		return 1
	}
	defer func() {
		if b, err := command(".", "git", "worktree", "remove", "--force", tree); err != nil {
			o.say("benchpair: git worktree remove %s: %v: %s", tree, err, b)
		}
	}()
	var files []string
	for pair := 1; pair <= 2; pair++ { // 0.19.6 D-28's two pairs (P10-02)
		for _, arm := range []struct{ name, dir string }{{"base", tree}, {"new", "."}} { // the pair's two arms (P10-02)
			f := filepath.Join(*dir, fmt.Sprintf("%s%d.txt", arm.name, pair))
			o.say("benchpair: pair %d, %s arm, %s", pair, arm.name, f)
			b, err := command(arm.dir, "go", append([]string{"test", "-run", "^$", "-bench", ".", "-benchmem", "-count", strconv.Itoa(*count)}, pkgs...)...)
			if werr := os.WriteFile(f, b, 0o644); werr != nil {
				o.say("benchpair: %v", werr)
				return 1
			}
			if err != nil {
				o.say("benchpair: the %s arm failed: %v; its output is in %s", arm.name, err, f)
				return 1
			}
			files = append(files, f)
		}
	}
	return judgeFiles(files, o)
}

// command runs a program in dir and returns its combined output.
func command(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// ---- judging ------------------------------------------------------------------

// judgeFiles judges two pairs of arm files: base1 new1 base2 new2.
func judgeFiles(files []string, o *out) int {
	if len(files) != 4 {
		o.say("benchpair: judge needs four files: base1 new1 base2 new2")
		return 2
	}
	for _, f := range files { // bounded by the four arms (P10-02)
		if dup, err := collidingNames(f); err != nil || dup != "" {
			o.say("benchpair: %s: %v%s", f, err, nameNote(dup))
			return 1
		}
	}
	var pairs []map[string]row
	for i := 0; i < 4; i += 2 { // the two pairs (P10-02)
		csvText, err := command(".", "go", "run", benchstat, "-format", "csv", "-ignore", "pkg", files[i], files[i+1])
		if err != nil {
			o.say("benchpair: benchstat: %v: %.400s", err, csvText)
			return 1
		}
		rows, err := readTimes(csvText)
		if err != nil {
			o.say("benchpair: %v", err)
			return 1
		}
		pairs = append(pairs, rows)
	}
	failures := judge(pairs[0], pairs[1])
	for _, f := range failures { // bounded by the verdict (P10-02)
		o.say("benchpair: FAIL %s", f)
	}
	if len(failures) > 0 {
		o.say("benchpair: the result goes to the HUM LEAD (0.19.6 D-28)")
		return 1
	}
	o.say("benchpair: OK - %d benchmarks paired twice, none significantly slower by %.0f %% in both pairs", len(pairs[0]), slowdown)
	return 0
}

// nameNote says what a colliding name means.
func nameNote(dup string) string {
	if dup == "" {
		return ""
	}
	return fmt.Sprintf("%q is declared in two packages, and pairing by name would pair the wrong two", dup)
}

// benchLine is a benchmark's result line: its name, less the GOMAXPROCS
// suffix.
var benchLine = regexp.MustCompile(`^(Benchmark\S+?)(?:-\d+)?\s+\d+\s`)

// collidingNames is a benchmark name two packages in one arm's file declare,
// or "".
func collidingNames(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	owner := map[string]string{}
	pkg := ""
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for range maxLines {
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		if p, ok := strings.CutPrefix(line, "pkg: "); ok {
			pkg = p
			continue
		}
		if m := benchLine.FindStringSubmatch(line); m != nil {
			if was, seen := owner[m[1]]; seen && was != pkg {
				return m[1], nil
			}
			owner[m[1]] = pkg
		}
	}
	if err := invariant.Check(len(owner) > 0, path+" holds no benchmark result: the arm ran nothing"); err != nil {
		return "", err
	}
	return "", sc.Err()
}

// maxLines bounds an arm's file: ten counts of every benchmark is a few
// thousand lines.
const maxLines = 1 << 20

// row is one benchmark's time comparison in a pair: the change in percent
// and its p-value, or paired false when the new arm has no figure for it.
type row struct {
	delta, p float64
	paired   bool
	same     bool // benchstat's "~": no significant change
}

// readTimes reads benchstat's CSV, the sec/op sections: one row a benchmark.
func readTimes(raw []byte) (map[string]row, error) {
	rows := map[string]row{}
	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = -1
	inTime := false
	for range maxLines {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			continue // benchstat's header lines are not CSV records it means to be read
		}
		switch {
		case len(rec) > 1 && rec[0] == "":
			inTime = rec[1] == "sec/op"
		case inTime && len(rec) > 0 && rec[0] != "geomean" && rec[0] != "":
			rows[benchName(rec[0])] = parseRow(rec)
		}
	}
	if err := invariant.Check(len(rows) > 0, "benchstat compared no benchmark's time"); err != nil {
		return nil, err
	}
	return rows, nil
}

// benchName is a CSV row's benchmark, its GOMAXPROCS suffix dropped.
func benchName(s string) string {
	if i := strings.LastIndex(s, "-"); i > 0 {
		if _, err := strconv.Atoi(s[i+1:]); err == nil {
			return s[:i]
		}
	}
	return s
}

// parseRow reads a sec/op row: name, base, CI, new, CI, vs base, P.
func parseRow(rec []string) row {
	if len(rec) < 7 || rec[3] == "" {
		return row{}
	}
	p := 1.0
	if v, ok := strings.CutPrefix(rec[6], "p="); ok {
		p, _ = strconv.ParseFloat(strings.Fields(v)[0], 64)
	}
	if rec[5] == "~" {
		return row{p: p, paired: true, same: true}
	}
	d, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimPrefix(rec[5], "+"), "%"), 64)
	return row{delta: d, p: p, paired: true}
}

// judge is every benchmark both pairs show slower, and every base benchmark
// with no pair.
func judge(first, second map[string]row) []string {
	var out []string
	names := make([]string, 0, len(first))
	for n := range first { // bounded by the benchmarks (P10-02)
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names { // bounded by the benchmarks (P10-02)
		a, b := first[n], second[n]
		switch {
		case !a.paired || !b.paired:
			out = append(out, fmt.Sprintf("%s has no pair on the tree: renamed, moved without its name, or deleted (D-28)", n))
		case slower(a) && slower(b):
			out = append(out, fmt.Sprintf("%s is slower in both pairs: %+.2f %% (p=%.3f) and %+.2f %% (p=%.3f)", n, a.delta, a.p, b.delta, b.p))
		}
	}
	return out
}

// slower says whether a pair shows 0.19.6 D-28's slowdown.
func slower(r row) bool { return !r.same && r.p < alpha && r.delta >= slowdown }
