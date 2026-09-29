// Command perfsum summarises one run of W14's standard workload (0.18.0
// D-154; scripts/quality/workload.sh) into the numbers a before-and-after
// comparison reads: each phase's resources, and each timing's spread.
//
//	perfsum -in <run dir>
//
// Resources, per phase of phases.log, from samples.csv (soak.sh):
//   - CPU % is the change in cumulative CPU time over the phase's span, from
//     its first sample to its last: soak.sh's pcpu is a decaying average,
//     which a short phase cannot be read from;
//   - physical footprint (macOS) or Pss (Linux) median and max, post-GC heap
//     median, goroutines median, threads max.
//
// Timings, from every counters record in the run (counters.jsonl, and any
// *.json a driver wrote), per phase, trigger and event: n, median, p90, max.
// Records are deduped by (file set, seq): one process's seq rises, and each
// cold-N.json is its own process.
//
// Stdlib only, like tools/slope.
package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// maxRows bounds every input read (P10-02): a run is hours of 20 s samples
// and a few hundred phases or counter reads - far below this. A file past it
// is not a run's, and is refused rather than read without end.
const maxRows = 1 << 20

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("perfsum", flag.ContinueOnError)
	fs.SetOutput(errOut)
	in := fs.String("in", "", "the run's directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *in == "" {
		_, _ = fmt.Fprintln(errOut, "perfsum: -in is required")
		return 2
	}
	s, lost, err := summarise(*in)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, "perfsum:", err)
		return 2
	}
	if _, err := fmt.Fprint(out, s); err != nil {
		return 2
	}
	if lost > 0 {
		return 1 // the render check failed: a run that lost its map passes for nothing
	}
	return 0
}

// phase is one line of phases.log and the span to the next.
type phase struct {
	name       string
	from, to   time.Time
	resource   resources
	hasSamples bool
}

type resources struct {
	cpuPct                          float64
	footMedMB, footMaxMB, heapMedMB float64
	goroutinesMed                   float64
	threadsMax                      float64
	samples                         int
}

// timing is one interval from the counters.
type timing struct {
	Seq     uint64    `json:"seq"`
	At      time.Time `json:"at"`
	Trigger string    `json:"trigger"`
	Event   string    `json:"event"`
	MS      float64   `json:"ms"`
}

func summarise(dir string) (string, int, error) {
	phases, err := readPhases(filepath.Join(dir, "phases.log"))
	if err != nil {
		return "", 0, err
	}
	if samples, err := readSamples(filepath.Join(dir, "samples.csv")); err == nil {
		for i := range phases {
			phases[i].resource, phases[i].hasSamples = resourcesIn(samples, phases[i].from, phases[i].to)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", 0, err
	}
	timings, err := readTimings(dir)
	if err != nil {
		return "", 0, err
	}
	var b strings.Builder
	head, _ := os.ReadFile(filepath.Join(dir, "run.txt"))
	fmt.Fprintf(&b, "# %s\n\n", strings.TrimSpace(string(head)))
	b.WriteString("## Resources by phase\n\n| Phase | Minutes | Samples | CPU % | Footprint MB (median / max) | Heap MB (median) | Goroutines | Threads (max) |\n|---|---|---|---|---|---|---|---|\n")
	for _, p := range phases {
		if !p.hasSamples {
			continue
		}
		r := p.resource
		fmt.Fprintf(&b, "| %s | %.1f | %d | %.1f | %.1f / %.1f | %.1f | %.0f | %.0f |\n", p.name, p.to.Sub(p.from).Minutes(), r.samples, r.cpuPct, r.footMedMB, r.footMaxMB, r.heapMedMB, r.goroutinesMed, r.threadsMax)
	}
	lost := renderCheck(&b, phases, timings)
	b.WriteString("\n## Timings (ms)\n\n| Phase | Trigger | Event | n | Median | p90 | Max |\n|---|---|---|---|---|---|---|\n")
	for _, row := range timingRows(phases, timings) {
		if !strings.Contains(row, "| render | below-floor |") {
			b.WriteString(row)
		}
	}
	return b.String(), lost, nil
}

// renderCheck writes the render check - every draw that fell under the map's
// floor, by phase (the instrument's "below-floor") - and returns how many.
// The HUM LEAD's rule for W14: everything that is supposed to render,
// actually renders.
func renderCheck(b *strings.Builder, phases []phase, ts []timing) int {
	byPhase, lost := map[string]int{}, 0
	var order []string
	for _, t := range ts {
		if t.Trigger != "render" || t.Event != "below-floor" {
			continue
		}
		name := "(before the first phase)"
		for i := len(phases) - 1; i >= 0; i-- {
			if !t.At.Before(phases[i].from) {
				name = phases[i].name
				break
			}
		}
		if byPhase[name] == 0 {
			order = append(order, name)
		}
		byPhase[name]++
		lost++
	}
	b.WriteString("\n## Render check\n\n")
	if lost == 0 {
		b.WriteString("PASS: the map was drawn in every phase - no draw fell under its floor.\n")
		return 0
	}
	b.WriteString("**FAIL: the map was not drawn** - a draw fell under its floor and showed the notice in its place.\n\n| Phase | Draws not drawn |\n|---|---|\n")
	for _, name := range order {
		fmt.Fprintf(b, "| %s | %d |\n", name, byPhase[name])
	}
	return lost
}

// readPhases reads "<RFC3339> <name...>" lines; each phase runs to the next.
func readPhases(path string) ([]phase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // read-only: nothing to lose on close
	var out []phase
	sc := bufio.NewScanner(f)
	for range maxRows {
		if !sc.Scan() {
			return out, sc.Err()
		}
		ts, name, ok := strings.Cut(sc.Text(), " ")
		if !ok {
			continue
		}
		at, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("phases.log: %q: %w", sc.Text(), err)
		}
		if n := len(out); n > 0 {
			// A LOG OUT OF TIME ORDER IS NOT A RUN'S: every phase would get a
			// negative span and every sample the wrong phase.
			if err := invariant.Check(!at.Before(out[n-1].from), "phases.log: phases out of time order"); err != nil {
				return nil, err
			}
			out[n-1].to = at
		}
		out = append(out, phase{name: name, from: at, to: at})
	}
	return nil, fmt.Errorf("phases.log: more than %d lines", maxRows)
}

// sample is one soak.sh row, the columns perfsum reads.
type sample struct {
	at                           time.Time
	cpuS, footKB, heap, gor, thr float64
}

func readSamples(path string) ([]sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // read-only: nothing to lose on close
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("samples.csv: no header: %w", err)
	}
	col := map[string]int{}
	for i, h := range head {
		col[h] = i
	}
	for _, need := range []string{"utc", "cpu_time", "footprint_kb", "heap_alloc", "goroutines", "threads"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("samples.csv: no %q column (soak.sh from before W14?)", need)
		}
	}
	var out []sample
	for range maxRows {
		rec, err := r.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		at, err := time.Parse(time.RFC3339, rec[col["utc"]])
		if err != nil {
			continue
		}
		cpu, ok := cpuSeconds(rec[col["cpu_time"]])
		if !ok {
			continue
		}
		num := func(k string) float64 { v, _ := strconv.ParseFloat(rec[col[k]], 64); return v }
		out = append(out, sample{at: at, cpuS: cpu, footKB: num("footprint_kb"), heap: num("heap_alloc"), gor: num("goroutines"), thr: num("threads")})
	}
	return nil, fmt.Errorf("samples.csv: more than %d rows", maxRows)
}

// cpuSeconds reads ps's cumulative CPU time: [[dd-]hh:]mm:ss[.ff].
func cpuSeconds(v string) (float64, bool) {
	days := 0.0
	if d, rest, ok := strings.Cut(v, "-"); ok {
		n, err := strconv.ParseFloat(d, 64)
		if err != nil {
			return 0, false
		}
		days, v = n, rest
	}
	parts := strings.Split(v, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	total := 0.0
	for _, p := range parts {
		n, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return 0, false
		}
		total = total*60 + n
	}
	return days*86400 + total, true
}

// resourcesIn is a phase's resources from the samples inside its span.
func resourcesIn(samples []sample, from, to time.Time) (resources, bool) {
	var in []sample
	for _, s := range samples {
		if !s.at.Before(from) && s.at.Before(to) {
			in = append(in, s)
		}
	}
	if len(in) < 2 {
		return resources{}, false
	}
	first, last := in[0], in[len(in)-1]
	var foot, heap, gor, thr []float64
	for _, s := range in {
		foot, heap, gor, thr = append(foot, s.footKB/1024), append(heap, s.heap/(1<<20)), append(gor, s.gor), append(thr, s.thr)
	}
	span := last.at.Sub(first.at).Seconds()
	// CUMULATIVE CPU NEVER FALLS within one process: a fall is two processes'
	// samples mixed, and the phase's CPU would come out negative.
	if invariant.Check(last.cpuS >= first.cpuS, "samples.csv: cumulative CPU time fell inside a phase") != nil {
		return resources{}, false
	}
	r := resources{samples: len(in), footMedMB: quantile(foot, 0.5), footMaxMB: maxOf(foot), heapMedMB: quantile(heap, 0.5), goroutinesMed: quantile(gor, 0.5), threadsMax: maxOf(thr)}
	if span > 0 {
		r.cpuPct = (last.cpuS - first.cpuS) / span * 100
	}
	return r, true
}

// readTimings reads every counters record in dir: counters.jsonl (one
// process, polled) with session-end.json, and each cold-N.json (a process of
// its own). Deduped by (process, seq).
func readTimings(dir string) ([]timing, error) {
	type key struct {
		proc string
		seq  uint64
	}
	seen := map[key]bool{}
	var out []timing
	add := func(proc string, ts []timing) {
		for _, t := range ts {
			if k := (key{proc, t.Seq}); !seen[k] {
				seen[k] = true
				out = append(out, t)
			}
		}
	}
	type record struct {
		Timings []timing `json:"timings"`
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "counters.jsonl")); err == nil {
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		for i := 0; dec.More(); i++ {
			if i == maxRows {
				return nil, fmt.Errorf("counters.jsonl: more than %d records", maxRows)
			}
			var r record
			if err := dec.Decode(&r); err != nil {
				return nil, fmt.Errorf("counters.jsonl: %w", err)
			}
			add("session", r.Timings)
		}
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	sort.Strings(files)
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var r record
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		proc := strings.TrimSuffix(filepath.Base(f), ".json")
		if proc == "session-end" {
			proc = "session"
		}
		add(proc, r.Timings)
	}
	return out, nil
}

// timingRows groups the timings by the phase they were measured in, then
// trigger and event, in phase order.
func timingRows(phases []phase, ts []timing) []string {
	type key struct{ phase, trigger, event string }
	groups := map[key][]float64{}
	var order []key
	phaseOf := func(at time.Time) (int, string) {
		for i := len(phases) - 1; i >= 0; i-- {
			if !at.Before(phases[i].from) {
				return i, phases[i].name
			}
		}
		return -1, "(before the first phase)"
	}
	idx := map[key]int{}
	for _, t := range ts {
		i, name := phaseOf(t.At)
		k := key{name, t.Trigger, t.Event}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
			idx[k] = i
		}
		groups[k] = append(groups[k], t.MS)
	}
	sort.SliceStable(order, func(a, b int) bool {
		if idx[order[a]] != idx[order[b]] {
			return idx[order[a]] < idx[order[b]]
		}
		if order[a].trigger != order[b].trigger {
			return order[a].trigger < order[b].trigger
		}
		return order[a].event < order[b].event
	})
	var out []string
	for _, k := range order {
		v := groups[k]
		out = append(out, fmt.Sprintf("| %s | %s | %s | %d | %.0f | %.0f | %.0f |\n", k.phase, k.trigger, k.event, len(v), quantile(v, 0.5), quantile(v, 0.9), maxOf(v)))
	}
	return out
}

// quantile is the nearest-rank quantile: an observed value, never an
// interpolation between two runs that did not happen.
func quantile(v []float64, q float64) float64 {
	if len(v) == 0 || invariant.Check(q >= 0 && q <= 1, "quantile outside [0, 1]") != nil {
		return math.NaN()
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(math.Ceil(q*float64(len(s)))) - 1
	return s[max(0, min(i, len(s)-1))]
}

func maxOf(v []float64) float64 {
	m := math.Inf(-1)
	for _, x := range v {
		m = math.Max(m, x)
	}
	return m
}
