package main

// g1.go — the G1 summary (0.19.0 W10.2): watchpost's added cost with the
// Propagation mode in use against not, over a run of at least 48 hours
// (problem-statement G1). workload.sh g1 writes the run as three arms, each a
// session directory (phases.log, samples.csv, counters.jsonl, g1-end.json),
// run side by side on the same machine and the same network: <run>/use, the
// mode in use; <run>/map, the map open in Forecast mode; <run>/idle, the map
// never opened. The use arm is compared with each; -against names the
// comparison G1 is judged on (the map open, D-236), and the other is
// reported.
//
// Each arm is measured from its "G measured" phase - the use arm's once the
// mode is open - to its last sample:
//   - CPU % is the change in cumulative CPU time over that span;
//   - RSS is ps's resident set, median and max (the target is judged on the
//     median difference, D-131);
//   - downloads are the propagation client's wire bytes (DN-12), in decimal
//     MB, over the time between the first and last counters records inside
//     the span, the driver's end-of-run record among them.

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// G1's targets (0.19.0 problem-statement "Targets"; D-131, D-141). A run
// is at the 10-minute default refresh unless its workload is the hourly
// variant: that run is its own (A-79), judged on its downloads alone, at
// the hourly setting's target (A-83).
const (
	g1AddedCPUPct  = 1.0  // added mean CPU, % of one core
	g1AddedRSSMB   = 25.0 // added RSS, median over the run
	g1WireMBHour   = 19.0 // downloads on the wire while open, decimal MB an hour
	g1WireMBHourly = 3.3  // the same on the hourly setting
	g1Measured     = "G measured"
	g1Hourly       = "hourly" // the workload variant run.txt names
)

// g1Arm is one arm's measure over its span.
type g1Arm struct {
	res       resources
	hours     float64 // from the measure to the last sample
	wireMB    float64 // decimal MB read off the wire between the counters' first and last records in the span
	wireHours float64 // the time between those records
	wireRead  bool    // two counters records inside the span, an hour or more apart
	warned    int     // the driver's cycles it could not confirm drawn, or its phases that failed (RC-1)
}

// g1Bases are the arms the use arm is compared with (A-79, A-80): the key
// -against takes, the arm's directory, and its words.
var g1Bases = []struct{ key, words string }{
	{"idle", "the map never opened"},
	{"map", "the map open in Forecast mode"},
}

// summariseG1 renders a G1 run and returns how many of the judged
// comparison's targets it missed.
func summariseG1(dir, against string) (string, int, error) {
	judged := false
	for _, base := range g1Bases {
		judged = judged || base.key == against
	}
	if !judged {
		return "", 0, fmt.Errorf("-against is idle or map, not %q", against)
	}
	use, err := readG1Arm(filepath.Join(dir, "use"))
	if err != nil {
		return "", 0, fmt.Errorf("use: %w", err)
	}
	type based struct {
		key, words string
		arm        g1Arm
	}
	var bases []based
	for _, base := range g1Bases {
		arm, err := readG1Arm(filepath.Join(dir, base.key))
		if errors.Is(err, os.ErrNotExist) && base.key != against {
			continue // a run from before the arm was added: the judged arm alone is required
		}
		if err != nil {
			return "", 0, fmt.Errorf("%s: %w", base.key, err)
		}
		bases = append(bases, based{base.key, base.words, arm})
	}
	var b strings.Builder
	head, _ := os.ReadFile(filepath.Join(dir, "run.txt"))
	fmt.Fprintf(&b, "# %s\n\n", strings.TrimSpace(string(head)))
	hourly := runVariant(head) == g1Hourly
	wireTarget := g1WireMBHour
	if hourly {
		wireTarget = g1WireMBHourly
	}
	b.WriteString("## The arms\n\n| Arm | Hours | Samples | CPU % | RSS MB (median / max) | Footprint MB (median) | Heap MB (median) | Wire MB an hour | Wire MB a day |\n|---|---|---|---|---|---|---|---|---|\n")
	row := func(name string, a g1Arm) {
		r := a.res
		fmt.Fprintf(&b, "| %s | %.1f | %d | %.2f | %.1f / %.1f | %s | %s | %s | %s |\n", name, a.hours, r.samples, r.cpuPct, r.rssMedMB, r.rssMaxMB, gauge(r.footMedMB), gauge(r.heapMedMB), a.perHour(1), a.perHour(24))
	}
	row("the mode in use", use)
	for _, base := range bases {
		row(base.words, base.arm)
	}
	missed := 0
	verdict := func(ok, counts bool) string {
		switch {
		case ok:
			return "met"
		case counts:
			missed++
		}
		return "**missed**"
	}
	short := use.hours < 48
	b.WriteString("\n## The run itself\n\n| Measure | Measured | Target | Verdict |\n|---|---|---|---|\n")
	fmt.Fprintf(&b, "| the driver's cycles not confirmed drawn, or its steps failed, in the mode in use | %d | none | %s |\n", use.warned, verdict(use.warned == 0, true))
	b.WriteString("\n## The mode's downloads\n\n| Measure | Measured | Target | Verdict |\n|---|---|---|---|\n")
	fmt.Fprintf(&b, "| downloads on the wire while open, MB an hour | %s | at most %g | %s |\n", use.perHour(1), wireTarget, verdict(use.wireRead && use.wireMB/use.wireHours <= wireTarget, true))
	for _, base := range bases {
		counts := base.key == against && !hourly
		how := "reported, not judged"
		if counts {
			how = "**judged** (-against " + base.key + ")"
		}
		cpu, rss := use.res.cpuPct-base.arm.res.cpuPct, use.res.rssMedMB-base.arm.res.rssMedMB
		fmt.Fprintf(&b, "\n## The mode in use against %s: %s\n\n| Measure | Measured | Target | Verdict |\n|---|---|---|---|\n", base.words, how)
		fmt.Fprintf(&b, "| added mean CPU, %% of one core | %+.2f | at most %.0f | %s |\n", cpu, g1AddedCPUPct, verdict(cpu <= g1AddedCPUPct, counts))
		fmt.Fprintf(&b, "| added RSS, MB (medians) | %+.1f | at most %.0f | %s |\n", rss, g1AddedRSSMB, verdict(rss <= g1AddedRSSMB, counts))
		fmt.Fprintf(&b, "| downloads with the Propagation mode never opened, MB | %s | none | %s |\n", base.arm.total(), verdict(base.arm.wireRead && base.arm.wireMB == 0, counts))
		fmt.Fprintf(&b, "| its driver's cycles not confirmed drawn, or its steps failed | %d | none | %s |\n", base.arm.warned, verdict(base.arm.warned == 0, counts))
		short = short || base.arm.hours < 48
	}
	switch {
	case hourly:
		b.WriteString("\nThe hourly setting's own run (A-79, A-83): its downloads alone are judged; CPU and RSS are judged over G1's 48 hours at the default refresh.\n")
	case short:
		b.WriteString("\nThe run is shorter than G1's 48 hours: a dry run, not the measure, and it exits as one missed (RC-1).\n")
		missed++
	}
	return b.String(), missed, nil
}

// runVariant is the workload variant run.txt's first line names
// ("workload-v1 <mode> <variant> ..."), or "" when it names none.
func runVariant(head []byte) string {
	f := strings.Fields(string(head))
	if len(f) < 3 {
		return ""
	}
	return f[2]
}

// gauge is a median in MB, or "unread" where no sample's gauge read.
func gauge(mb float64) string {
	if math.IsNaN(mb) {
		return "unread"
	}
	return fmt.Sprintf("%.1f", mb)
}

// perHour is the arm's downloads over hours of its span, or "unread".
func (a g1Arm) perHour(hours float64) string {
	if !a.wireRead {
		return "unread"
	}
	return fmt.Sprintf("%.2f", a.wireMB/a.wireHours*hours)
}

// total is the arm's downloads over its span, or "unread".
func (a g1Arm) total() string {
	if !a.wireRead {
		return "unread"
	}
	return fmt.Sprintf("%.2f", a.wireMB)
}

// readG1Arm measures one arm from its "G measured" phase.
func readG1Arm(dir string) (g1Arm, error) {
	phases, err := readPhases(filepath.Join(dir, "phases.log"))
	if err != nil {
		return g1Arm{}, err
	}
	var from time.Time
	for _, p := range phases {
		if strings.HasPrefix(p.name, g1Measured) {
			from = p.from
			break
		}
	}
	if from.IsZero() {
		return g1Arm{}, fmt.Errorf("phases.log: no %q phase - the driver never reached its measure", g1Measured)
	}
	samples, err := readSamples(filepath.Join(dir, "samples.csv"))
	if err != nil {
		return g1Arm{}, err
	}
	for _, s := range samples {
		if !s.at.Before(from) && math.IsNaN(s.rssKB) {
			return g1Arm{}, fmt.Errorf("samples.csv: a sample at %s has no RSS that reads: G1 is judged on RSS, so no figure from this arm stands (RC-1)", s.at.Format(time.RFC3339))
		}
	}
	res, ok := resourcesIn(samples, from, time.Unix(1<<40, 0))
	if !ok {
		return g1Arm{}, fmt.Errorf("samples.csv: fewer than two samples after %s", from.Format(time.RFC3339))
	}
	a := g1Arm{res: res}
	for _, p := range phases {
		if strings.Contains(p.name, "WARN") || strings.Contains(p.name, "FAILED") {
			a.warned++ // a cycle the driver could not confirm, or a step that failed: the run did not do what it says
		}
	}
	for _, s := range samples {
		if !s.at.Before(from) {
			a.hours = s.at.Sub(from).Hours()
		}
	}
	first, last, err := wireSpan(dir, from)
	if err != nil {
		return g1Arm{}, err
	}
	// A CLIENT'S WIRE COUNT NEVER FALLS: a fall is two processes' records
	// mixed, and the downloads would come out negative.
	if err := invariant.Check(last.Wire >= first.Wire, "counters.jsonl: the wire bytes fell inside the run"); err != nil {
		return g1Arm{}, err
	}
	a.wireMB, a.wireHours = float64(last.Wire-first.Wire)/1e6, last.At.Sub(first.At).Hours()
	a.wireRead = a.wireHours >= 1
	return a, nil
}

// wireRecord is the part of a counters record G1 reads.
type wireRecord struct {
	At   time.Time `json:"at"`
	Wire int64     `json:"propagation_wire_bytes"`
}

// wireSpan is the first and last counters records at or after from: the
// polled counters.jsonl, then the driver's end-of-run g1-end.json when it
// wrote one.
func wireSpan(dir string, from time.Time) (first, last wireRecord, err error) {
	raw, err := os.ReadFile(filepath.Join(dir, "counters.jsonl"))
	if err != nil {
		return first, last, err
	}
	if end, err := os.ReadFile(filepath.Join(dir, "g1-end.json")); err == nil {
		raw = append(append(raw, '\n'), end...)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	for i := 0; dec.More(); i++ {
		if i == maxRows {
			return first, last, fmt.Errorf("counters: more than %d records", maxRows)
		}
		var r wireRecord
		if err := dec.Decode(&r); err != nil {
			return first, last, fmt.Errorf("counters: %w", err)
		}
		if r.At.Before(from) {
			continue
		}
		if first.At.IsZero() {
			first = r
		}
		last = r
	}
	return first, last, nil
}
