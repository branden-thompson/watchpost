package tty

// status.go — the [S] API Status modal: providers, pipelines, issues, requests, dumps. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// statusLines is the [S] API diagnostics body (UAT 24.2/31.1): aligned
// provider rows with a fixed-width freshness age, pipeline snapshot ages,
// and warnings AGGREGATED by code+provider (count, locations hit, latest
// message) so a busy pipeline reads as a handful of issues, not a flood.
// True request latency needs httpx instrumentation - queued with B5.
func (d Dashboard) statusLines() []string {
	o := d.opts()
	lines := []string{"", "  PROVIDERS"}
	if d.snap == nil {
		lines = append(lines, "    awaiting first snapshot...")
	}
	for _, p := range providersOf(d.snap) {
		age := "    n/a"
		if !p.FetchedAt.IsZero() {
			age = fixedAge(time.Since(p.FetchedAt))
		}
		lines = append(lines, fmt.Sprintf("    %s %-9s %-4s fetched %s ago",
			render.PadTo(o.HealthGlyph(strings.ToUpper(p.ID), p.Status), 13), p.Role, p.Status, age))
	}
	var st Stats
	if d.cfg.Stats != nil {
		st = d.cfg.Stats()
	}
	lines = append(lines, "", "  PIPELINES")
	lines = append(lines, pipelineLine("Priority", d.snap, st.Pipelines[0]), pipelineLine("Recent  ", d.recent, st.Pipelines[1]))
	lines = append(lines, "", "  ISSUES")
	lines = append(lines, aggregateWarnings(d.snap, d.recent)...)
	if d.cfg.Stats != nil {
		lines = append(lines, requestLines(st)...)
	}
	return append(lines, "", "  "+o.KeyCap("esc")+" Close   "+o.KeyCap("↑↓")+" Scroll")
}

// requestLines is the [S] REQUESTS and DUMPS body (quality pass Q0):
// one row per host, busiest first, counters since launch; then the
// diagnostic dump's last outcome and the trigger for this platform.
func requestLines(st Stats) []string {
	lines := []string{"", fmt.Sprintf("  REQUESTS (since launch, %s)", fixedAge(st.Requests.Uptime))}
	if len(st.Requests.Hosts) == 0 {
		lines = append(lines, "    none yet")
	} else {
		lines = append(lines, fmt.Sprintf("    %-29s %6s %5s %5s %3s %6s", "HOST", "TRIES", "NET", "CACHE", "NEG", "BYTES"))
	}
	for _, h := range st.Requests.Hosts { // 63 cells: fits the 68-wide modal without wrapping (UAT 25: never truncate)
		lines = append(lines, fmt.Sprintf("    %-29s %6d %5d %5d %3d %6s", h.Host, h.Attempts, h.Net, h.Cache, h.Neg, render.HumanBytes(h.BytesNet)))
	}
	lines = append(lines, "", "  DUMPS")
	if st.LastDump == "" {
		lines = append(lines, "    none yet")
	} else {
		lines = append(lines, "    last: "+st.LastDump)
	}
	if st.DumpHint != "" {
		lines = append(lines, "    trigger: "+st.DumpHint)
	}
	return lines
}

// fixedAge formats a duration in a 7-cell right-aligned slot so ages line
// up: "59m 59s", " 1m  5s", "    55s", " 2h 05m".
func fixedAge(dur time.Duration) string {
	dur = dur.Round(time.Second)
	h, m, sec := int(dur.Hours()), int(dur.Minutes())%60, int(dur.Seconds())%60
	switch {
	case h > 0:
		return fmt.Sprintf("%2dh %02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%2dm %2ds", m, sec)
	}
	return fmt.Sprintf("    %2ds", sec)
}

// issue is one aggregated warning class.
type issue struct {
	code, provider, latest string
	count, locations       int
}

// aggregateWarnings folds both snapshots' warnings into issue classes and
// renders them (UAT 31.1). Split fold/render for P10-04.
func aggregateWarnings(snaps ...*snapshot.Snapshot) []string {
	issues := foldWarnings(snaps)
	if len(issues) == 0 {
		return []string{"    none"}
	}
	return renderIssues(issues)
}

// foldWarnings groups by code+provider: count, distinct locations, latest
// message; provider errors sort first, then by count.
func foldWarnings(snaps []*snapshot.Snapshot) []*issue {
	byKey := map[string]*issue{}
	seenLoc := map[string]map[string]bool{}
	for _, sn := range snaps {
		if sn == nil {
			continue
		}
		for _, w := range sn.Warnings {
			k := w.Code + "|" + w.Provider
			it := byKey[k]
			if it == nil {
				it = &issue{code: w.Code, provider: w.Provider}
				byKey[k] = it
				seenLoc[k] = map[string]bool{}
			}
			it.count++
			it.latest = w.Message
			if w.Location != "" && !seenLoc[k][w.Location] {
				seenLoc[k][w.Location] = true
				it.locations++
			}
		}
	}
	issues := make([]*issue, 0, len(byKey))
	for _, it := range byKey {
		issues = append(issues, it)
	}
	sort.Slice(issues, func(i, j int) bool {
		pi, pj := issues[i].code == snapshot.WarnProviderError, issues[j].code == snapshot.WarnProviderError
		if pi != pj {
			return pi
		}
		return issues[i].count > issues[j].count
	})
	return issues
}

// renderIssues formats issue rows, capped at 8 with an overflow note.
func renderIssues(issues []*issue) []string {
	out := []string{}
	for i, it := range issues {
		if i == 8 {
			return append(out, fmt.Sprintf("    ... and %d more issue classes", len(issues)-8))
		}
		glyph := "⚠"
		if it.code == snapshot.WarnProviderError {
			glyph = "✘"
		}
		prov := it.provider
		if prov == "" {
			prov = "-"
		}
		scope := fmt.Sprintf("×%d", it.count)
		if it.locations > 0 {
			scope += fmt.Sprintf(" (%d locations)", it.locations)
		}
		out = append(out, fmt.Sprintf("    %s %-11s %-22s %s", glyph, strings.ToUpper(prov), it.code, scope))
		if it.latest != "" {
			out = append(out, "        latest: "+it.latest)
		}
	}
	return out
}

// providersOf guards the nil snapshot for the status view.
func providersOf(sn *snapshot.Snapshot) []snapshot.ProviderStatus {
	if sn == nil {
		return nil
	}
	return sn.Providers
}

// pipelineLine summarizes one pipeline's snapshot for the status view.
func pipelineLine(name string, sn *snapshot.Snapshot, ps PipelineStats) string {
	if sn == nil {
		return "    " + name + "  awaiting first snapshot..."
	}
	line := fmt.Sprintf("    %s  snapshot %s · %d locations", name,
		dataAsOf(sn).Local().Format("15:04:05"), len(sn.Locations))
	if ps.Publishes > 0 {
		line += fmt.Sprintf(" · %d publishes (%d folded)", ps.Publishes, ps.Folded)
	}
	return line
}
