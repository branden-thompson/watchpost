package main

// report.go - the table, the inputs' checksums, the tool's revision, and
// the scenarios kept outside the tree.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// reportOf is what the table is written from.
type reportOf struct {
	flags     scoreFlags
	days      []time.Time
	set       []scenario
	counts    map[base]*[outcomes]int
	res       scored
	facts     [columns]map[time.Time]hourFacts
	grids     int
	stations  int
	spotFiles int
	sum       string
	rev       string
}

// targetPoints is D-99's margin: the shipped build's agreement at least the
// climatology's plus 5 points, or no worse than it above 90 %.
const (
	targetPoints  = 5.0
	targetCeiling = 90.0
)

// report writes the table: the run's provenance, each metric's agreement
// by band for every column with the base rate beside it (D-219), each kind
// of outcome's agreement apart (D-218), and what fed each column.
func report(w io.Writer, r reportOf) error {
	if w == nil || len(r.days) == 0 {
		return errUsage
	}
	if r.res.tallies == nil {
		return errUsage // nothing was scored
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# M1 and M4, %s to %s\n\n", r.days[0].Format(time.DateOnly), r.days[len(r.days)-1].Format(time.DateOnly))
	fmt.Fprintf(&b, "Tool: reachscore at %s; seed %d; at most %d of each outcome a stratum, the strata by %s.\n", r.rev, r.flags.seed, r.flags.k, r.flags.strata)
	fmt.Fprintf(&b, "Inputs: %d GloTEC grids, %d GIRO replies, the solar file, %d days' spots; sha256 of their checksums %s.\n", r.grids, r.stations, r.spotFiles, r.sum)
	fmt.Fprintf(&b, "Reach: pairs at least %.0f km apart; M1's places within %.0f km of the tower, answered for that service radius; M4's transmitters within %.0f km of a continent's reference point, the listener answered with a %.0f km near-vertical radius.\n",
		r.flags.r.SkyKm, r.flags.r.ServedKm, r.flags.r.RegionKm, r.flags.r.NearKm)
	for m := range metrics {
		writeMetric(&b, m, r)
	}
	writeFeeds(&b, r.facts)
	_, err := io.WriteString(w, b.String())
	return err
}

// pct is a share as a percentage to a tenth, or a dash for none.
func pct(agree, n int) string {
	if n == 0 {
		return "-"
	}
	if agree < 0 || agree > n {
		return "?" // more agreeing than asked: a tally is wrong, and no figure is printed for it
	}
	return fmt.Sprintf("%.1f", 100*float64(agree)/float64(n))
}

// writeMetric writes one metric's two tables and its line against the
// target.
func writeMetric(b *strings.Builder, m metric, r reportOf) {
	if b == nil || m >= metrics {
		return
	}
	if r.counts == nil {
		return
	}
	fmt.Fprintf(b, "\n## %s\n\n| Band | Scenarios (open / weak / unheard) | Open in all pairs | Climatology | Background | Shipped | Shipped - climatology |\n|---|---|---|---|---|---|---|\n", m)
	var all [columns]tally
	var allCounts [outcomes]int
	for _, wb := range wsprBands {
		k := base{m, wb.band}
		t, c := r.res.tallies[k], r.counts[k]
		if t == nil || c == nil {
			continue
		}
		for col := range columns {
			for o := range outcomes {
				all[col].n[o] += t[col].n[o]
				all[col].agree[o] += t[col].agree[o]
				all[col].noData[o] += t[col].noData[o]
			}
		}
		for o := range outcomes {
			allCounts[o] += c[o]
		}
		writeRow(b, wb.band.String(), *t, *c)
	}
	writeRow(b, "**All**", all, allCounts)
	fmt.Fprintf(b, "\nBy kind, the agreement on open / weak / unheard:\n\n| | Climatology | Background | Shipped |\n|---|---|---|---|\n| **All** |")
	for col := range columns {
		fmt.Fprintf(b, " %s / %s / %s |", pct(all[col].agree[open], all[col].n[open]), pct(all[col].agree[weak], all[col].n[weak]), pct(all[col].agree[unheard], all[col].n[unheard]))
	}
	fmt.Fprintln(b)
	n, ca := all[climatology].total()
	_, sa := all[shipped].total()
	var noData int
	for col := range columns {
		for o := range outcomes {
			noData += all[col].noData[o]
		}
	}
	fmt.Fprintf(b, "\nAgainst the target (D-99): %s.\n", verdict(ca, sa, n))
	fmt.Fprintf(b, "Answers with no field for the band: %d of %d, counted as not reaching.\n", noData, int(columns)*n)
}

// writeRow writes one band's row, or the metric's total.
func writeRow(b *strings.Builder, name string, t [columns]tally, c [outcomes]int) {
	if b == nil {
		return
	}
	n, _ := t[climatology].total()
	nb, _ := t[background].total()
	ns, _ := t[shipped].total()
	if nb != n || ns != n {
		fmt.Fprintf(b, "| %s | the columns answered different numbers of scenarios: no figure stands |\n", name)
		return
	}
	pairs := c[open] + c[weak] + c[unheard]
	fmt.Fprintf(b, "| %s | %d (%d / %d / %d) | %s |", name, n, t[climatology].n[open], t[climatology].n[weak], t[climatology].n[unheard], pct(c[open], pairs))
	for col := range columns {
		_, a := t[col].total()
		fmt.Fprintf(b, " %s |", pct(a, n))
	}
	_, ca := t[climatology].total()
	_, sa := t[shipped].total()
	if n == 0 {
		fmt.Fprintln(b, " - |")
		return
	}
	fmt.Fprintf(b, " %+.1f |\n", 100*float64(sa-ca)/float64(n))
}

// verdict is the shipped build's agreement against D-99's target.
func verdict(climAgree, shipAgree, n int) string {
	if n == 0 {
		return "no scenarios"
	}
	if climAgree > n || shipAgree > n {
		return "more agreeing than scenarios: a tally is wrong, no verdict"
	}
	clim, ship := 100*float64(climAgree)/float64(n), 100*float64(shipAgree)/float64(n)
	target, rule := clim+targetPoints, "the climatology's plus 5 points"
	if clim > targetCeiling {
		target, rule = clim, "no worse than the climatology, which is above 90 %"
	}
	met := "met"
	if ship < target {
		met = "missed"
	}
	return fmt.Sprintf("the shipped build %.1f %% against %.1f %% (%s): %s", ship, target, rule, met)
}

// writeFeeds writes what fed each column at the scored hours.
func writeFeeds(b *strings.Builder, facts [columns]map[time.Time]hourFacts) {
	if b == nil {
		return
	}
	fmt.Fprintf(b, "\n## The replay\n\n| Column | Scored hours | With a GloTEC grid | Stations assimilated, mean | F10.7 mean, range |\n|---|---|---|---|---|\n")
	for col := range columns {
		var grids, stations int
		lo, hi := float32(0), float32(0)
		for _, f := range facts[col] {
			if f.Grid {
				grids++
			}
			stations += f.Stations
			if lo == 0 || f.F107 < lo {
				lo = f.F107
			}
			hi = max(hi, f.F107)
		}
		n := len(facts[col])
		mean := 0.0
		if n > 0 {
			mean = float64(stations) / float64(n)
		}
		fmt.Fprintf(b, "| %s | %d | %d | %.1f | %.0f to %.0f |\n", col, n, grids, mean, lo, hi)
	}
}

// checksums writes each file's sha256 to a list and returns the list's
// own: one figure naming every input.
func checksums(files []string, list string) (string, error) {
	if len(files) == 0 || list == "" {
		return "", errUsage
	}
	var lines []string
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(body)
		lines = append(lines, hex.EncodeToString(sum[:])+"  "+filepath.Base(path))
	}
	slices.Sort(lines)
	text := strings.Join(lines, "\n") + "\n"
	if err := os.MkdirAll(filepath.Dir(list), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(list, []byte(text), 0o600); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:]), nil
}

// keepScenarios writes each scenario and its three answers, outside the
// tree.
func keepScenarios(path string, set []scenario, answers [][columns]bool) error {
	if len(answers) != len(set) {
		return errUsage
	}
	if path == "" {
		return errUsage
	}
	var b strings.Builder
	b.WriteString("metric\tclock\tband\torigin_lat\torigin_lon\ttarget_lat\ttarget_lon\tkm\toutcome\tclimatology\tbackground\tshipped\n")
	for i, s := range set {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%.3f\t%.3f\t%.3f\t%.3f\t%.0f\t%s\t%t\t%t\t%t\n", s.Metric, s.At.Format(time.RFC3339), s.Band,
			s.Origin.Lat, s.Origin.Lon, s.Target.Lat, s.Target.Lon, s.Km, s.Outcome, answers[i][climatology], answers[i][background], answers[i][shipped])
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

// revision is the tree's commit and whether it is clean: a figure is cited
// only from the committed tool (E-19), so a dirty tree says so.
func revision() string {
	head, err := exec.Command("git", "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return "an unknown revision"
	}
	status, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(strings.TrimSpace(string(status))) > 0 {
		return strings.TrimSpace(string(head)) + ", the tree DIRTY: these figures are not citable (E-19)"
	}
	return strings.TrimSpace(string(head)) + ", the tree clean"
}
