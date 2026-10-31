package history

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// rollUpBudget bounds a roll-up part's decompressed size: half the read cap, so
// a part the store writes is always one it can read back (#27, 0.19.0 W1.2).
var rollUpBudget = maxDocBytes / 2

// rollupPart is the n-th part of a series' roll-ups for at's month:
// rollup/<YYYY-MM>-p<n>.json.gz.
func rollupPart(dir string, at time.Time, n int) string {
	return filepath.Join(dir, "rollup", at.Format("2006-01")+"-p"+strconv.Itoa(n)+".json.gz")
}

// rollupParts is a series' roll-up parts on disk, each month's numbers in
// order, by "YYYY-MM": read from the directory, so a part removed never hides
// the parts after it.
func rollupParts(dir string) map[string][]int {
	entries, err := os.ReadDir(filepath.Join(dir, "rollup"))
	if err != nil {
		return nil
	}
	out := map[string][]int{}
	for _, e := range entries { // a series' roll-ups (P10-02)
		if r, ok := parseRollup(trimExt(e.Name())); ok && r.part >= 1 {
			month := r.start.Format("2006-01")
			out[month] = append(out[month], r.part)
		}
	}
	for _, ns := range out { // the months (P10-02)
		sort.Ints(ns)
	}
	return out
}

// rollupName is a roll-up file's name, read: the days it holds, from start
// to end, and its part - 0 for a year's file (rollup/<YYYY>.json.gz), 1 on
// for a month's parts (rollup/<YYYY-MM>-p<n>.json.gz).
type rollupName struct {
	start, end time.Time
	part       int
}

// parseRollup reads a roll-up file's name, its extension trimmed: the one
// reading the store's readers and its pruning share, so a file one sees the
// others see; false for any other name.
func parseRollup(name string) (rollupName, bool) {
	if y, err := strconv.Atoi(name); err == nil {
		start := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
		return rollupName{start: start, end: start.AddDate(1, 0, 0)}, true
	}
	month, part, found := strings.Cut(name, "-p")
	if !found {
		return rollupName{}, false
	}
	n, err := strconv.Atoi(part)
	if err != nil || n < 1 {
		return rollupName{}, false
	}
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return rollupName{}, false
	}
	return rollupName{start: start, end: start.AddDate(0, 1, 0), part: n}, true
}

// rollUp writes a day's numeric fields - each point's minimum, maximum and
// mean over the day's records - into its month's roll-ups, under the series'
// roll-up claim for that month (a part is read, merged and replaced, so one
// writer at a time), and reports whether a part now holds that day: only then
// may its records go. A day of documents alone rolls up to nothing and is kept
// until the days' retention passes too.
func (s *Store) rollUp(d Dataset, dir string, date time.Time) bool {
	entries := s.dayEntriesOnce(dir, d, date)
	if len(entries) == 0 {
		return false
	}
	entry, ok := dayStats(d, dayDoc{Date: date.Format("2006-01-02"), Hours: entries})
	if !ok {
		return false
	}
	claim := filepath.Join(dir, "rollup", ".claim-"+date.Format("2006-01"))
	if !s.claim(claim) {
		return false // another instance is rolling up this month: next pass
	}
	defer func() { _ = os.Remove(claim) }() // the claim let go
	path, doc, ok := s.partFor(d, dir, date, entry)
	if !ok {
		return false // a part cannot be read (#27), or the day fits no part: the day waits
	}
	doc.Schema, doc.Dataset, doc.Version, doc.Key = schema, d.Name, d.Version, keyOf(dir)
	doc.Days = withDay(doc.Days, entry)
	if _, _, err := s.writeCounted(path, doc); err != nil {
		return false
	}
	if back, ok := readAs[rollupDoc](s, path, d); ok && hasDay(back.Days, entry.Date) {
		s.note(func(st *Stats) { st.RolledUp++ }, true)
		return true
	}
	return false
}

// maxMonthParts bounds a month's parts: each holds at least one day.
const maxMonthParts = 31

// partFor is the part a day's entry is written to, and what it holds now: the
// part already holding that date; else the month's last part, when the entry
// still fits its budget; else a new part. False when a part exists but cannot
// be read - it is never written over (#27) - or when the entry, alone or in
// its date's part, would pass the budget: a part the store writes is always
// one it can read back (W1.2).
func (s *Store) partFor(d Dataset, dir string, date time.Time, entry dayEntry) (string, rollupDoc, bool) {
	var lastPath string
	var last rollupDoc
	parts := rollupParts(dir)[date.Format("2006-01")]
	for _, n := range parts { // a month's parts, at most maxMonthParts written (P10-02)
		path := rollupPart(dir, date, n)
		doc, ok := readAs[rollupDoc](s, path, d)
		if !ok {
			return "", rollupDoc{}, false
		}
		if hasDay(doc.Days, entry.Date) {
			return path, doc, fitsBudget(withDay(doc.Days, entry))
		}
		lastPath, last = path, doc
	}
	if !fitsBudget([]dayEntry{entry}) {
		return "", rollupDoc{}, false // larger than a part on its own
	}
	if lastPath != "" && fitsBudget(withDay(last.Days, entry)) {
		return lastPath, last, true
	}
	if len(parts) >= maxMonthParts {
		return "", rollupDoc{}, false
	}
	next := 1
	if len(parts) > 0 {
		next = parts[len(parts)-1] + 1
	}
	return rollupPart(dir, date, next), rollupDoc{}, true
}

// fitsBudget reports whether days, written as a part, stay within the budget.
func fitsBudget(days []dayEntry) bool {
	raw, err := json.Marshal(rollupDoc{Days: days})
	return err == nil && len(raw) <= rollUpBudget
}

// dayStats is a day's records rolled up, field by field and point by point;
// false when no record has numeric values or the shapes disagree.
func dayStats(d Dataset, doc dayDoc) (dayEntry, bool) {
	if len(doc.Hours) == 0 || len(d.Fields) == 0 {
		return dayEntry{}, false
	}
	shape := doc.Hours[0].Shape
	n := shape.Cols * shape.Rows
	if n <= 0 {
		return dayEntry{}, false
	}
	e := dayEntry{Date: doc.Date, Shape: shape, Hours: len(doc.Hours), Stats: map[string]map[string]values{}}
	for _, f := range d.Fields { // a dataset's few (P10-02)
		lo, hi, sum, count := make([]float64, n), make([]float64, n), make([]float64, n), make([]int, n)
		for _, h := range doc.Hours { // a day's buckets (P10-02)
			if h.Shape != shape {
				return dayEntry{}, false
			}
			for i, x := range h.Values[f.Name] { // bounded by the shape (P10-02)
				if math.IsNaN(x) || i >= n {
					continue
				}
				if count[i] == 0 || x < lo[i] {
					lo[i] = x
				}
				if count[i] == 0 || x > hi[i] {
					hi[i] = x
				}
				sum[i] += x
				count[i]++
			}
		}
		e.Stats[f.Name] = map[string]values{"min": statsOf(lo, count, 1, f), "max": statsOf(hi, count, 1, f), "mean": statsOf(sum, count, 0, f)}
	}
	return e, true
}

// statsOf is one statistic's values on disk: missing where no record had a
// value there; a mean is the sum over its count (asIs 0), a bound as it is.
func statsOf(v []float64, count []int, asIs int, f Field) values {
	if len(v) == 0 {
		return nil
	}
	if len(count) != len(v) {
		return nil
	}
	out := make([]float64, len(v))
	for i := range v { // P10-02
		switch {
		case count[i] == 0:
			out[i] = math.NaN()
		case asIs == 1:
			out[i] = v[i]
		default:
			out[i] = v[i] / float64(count[i])
		}
	}
	return encode(out, f.Decimals)
}

// withDay is days with e in its date's place, in order.
func withDay(days []dayEntry, e dayEntry) []dayEntry {
	if e.Date == "" {
		return days
	}
	kept := days[:0:0]
	for _, x := range days { // at most 366 (P10-02)
		if x.Date != e.Date {
			kept = append(kept, x)
		}
	}
	kept = append(kept, e)
	sort.Slice(kept, func(i, j int) bool { return kept[i].Date < kept[j].Date })
	return kept
}

// hasDay reports whether days hold date.
func hasDay(days []dayEntry, date string) bool {
	if len(days) == 0 || date == "" {
		return false
	}
	for _, x := range days { // P10-02
		if x.Date == date {
			return true
		}
	}
	return false
}
