package history

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// maxPruneVisits and maxPruneRemovals bound one pass of Prune: what is left
// waits for the next.
const (
	maxPruneVisits   = 256
	maxPruneRemovals = 64
)

// remeasureEvery is how often a prune pass walks the store for its size
// again: what other instances wrote is counted only by a walk.
const remeasureEvery = 6 * time.Hour

// RollUpAndPrune keeps each series to its dataset's retention: a day past
// Hours is rolled up into its year (when the dataset keeps Days) before its
// hours are removed; a year past Days is removed; stale temp files and
// claims are swept; a month, series or source directory emptied is removed.
// Bounded: a pass goes on from the series the last one finished with, the
// datasets by name and their series in order, and starts again from the
// first once it has passed the last (QA-4). Any instance may run it; it
// tolerates what another has already removed. Every remeasureEvery it walks
// the store for its size.
func (s *Store) RollUpAndPrune() {
	if s.root == "" {
		return
	}
	now := s.now().UTC()
	s.prunePass(now)
	s.mu.Lock()
	stale := s.measured.IsZero() || now.Sub(s.measured) >= remeasureEvery
	s.mu.Unlock()
	if stale {
		s.measure()
	}
}

// prunePass is one bounded pass of RollUpAndPrune, from the cursor on.
func (s *Store) prunePass(now time.Time) {
	visits, removals := 0, 0
	s.mu.Lock()
	from := s.cursor
	s.mu.Unlock()
	sets := s.datasets()
	sort.Slice(sets, func(i, j int) bool { return sets[i].Name < sets[j].Name })
	// The bounds' own visits: a dataset that cannot get under its bound never
	// spends the walk's.
	bounds := 0
	for _, d := range sets { // a handful (P10-02)
		removals += s.holdToBound(d, now, &bounds, maxPruneRemovals-removals)
	}
	for _, d := range sets { // a handful (P10-02)
		if d.Name < from[0] {
			continue // passed already
		}
		vdir := filepath.Join(s.root, d.Name, "v"+strconv.Itoa(d.Version))
		for _, series := range listSeries(vdir, &visits) { // bounded by maxPruneVisits
			k := keyOf(series)
			at := [3]string{d.Name, k.Source, k.Place}
			if !cursorAfter(at, from) {
				continue // passed already
			}
			removals += s.pruneSeries(d, series, now, &visits, maxPruneRemovals-removals)
			if visits >= maxPruneVisits || removals >= maxPruneRemovals {
				return // this series may be unfinished: the next pass begins with it
			}
			s.mu.Lock()
			s.cursor = at
			s.mu.Unlock()
		}
	}
	s.mu.Lock()
	s.cursor = [3]string{} // the last series passed: the next pass starts again
	s.mu.Unlock()
}

// holdToBound removes a dataset's oldest files while it holds more than its
// byte bound (D-143): its roll-ups, oldest first, then its oldest days, each
// day with its buckets; never the current day. Every version counts, so every
// version's series are looked at: each one's roll-ups and oldest month in a
// pass, a month emptied leaving the next for the following pass. The files
// removed, at most budget.
func (s *Store) holdToBound(d Dataset, now time.Time, visits *int, budget int) int {
	if d.MaxBytes <= 0 || budget <= 0 {
		return 0
	}
	if s.BytesOf(d.Name) <= d.MaxBytes {
		return 0
	}
	if s.versionDir(d.Name, d.Version) == "" {
		return 0
	}
	var series []string
	for _, v := range readDirs(filepath.Join(s.root, d.Name), visits) { // its versions (P10-02)
		series = append(series, listSeries(filepath.Join(s.root, d.Name, v), visits)...)
	}
	today := now.Truncate(24 * time.Hour)
	removed := 0
	for _, f := range oldestFiles(series, visits) { // bounded by maxPruneVisits (P10-02)
		if removed >= budget || s.BytesOf(d.Name) <= d.MaxBytes {
			break
		}
		if !f.at.Before(today) {
			continue // the current day is being recorded
		}
		removed += s.removeFiles(f)
	}
	if removed > 0 {
		s.mu.Lock()
		if s.bounded == nil {
			s.bounded = map[string]bool{}
		}
		s.bounded[d.Name] = true
		s.mu.Unlock()
		s.note(func(st *Stats) { st.Pruned++ }, true)
	}
	return removed
}

// oldFile is a file a byte bound may remove: a roll-up, or a day's file with
// its buckets' directory, and the first day it holds.
type oldFile struct {
	at            time.Time
	path, buckets string
}

// oldestFiles is what a bound removes first across series, oldest first, a
// roll-up before a day of the same date: each series' roll-ups and the days
// of its oldest month.
func oldestFiles(series []string, visits *int) []oldFile {
	var out []oldFile
	for _, dir := range series { // bounded by maxPruneVisits (P10-02)
		rdir := filepath.Join(dir, "rollup")
		for _, e := range readEntries(rdir, visits) { // a series' roll-ups (P10-02)
			if r, ok := parseRollup(trimExt(e.Name())); ok && !e.IsDir() {
				out = append(out, oldFile{at: r.start, path: filepath.Join(rdir, e.Name())})
			}
		}
		months := readDirs(dir, visits)
		sort.Strings(months)
		for _, month := range months { // the oldest month only: the first that is one (P10-02)
			if _, err := time.Parse("2006-01", month); err != nil {
				continue
			}
			out = append(out, monthDays(filepath.Join(dir, month), month, visits)...)
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].at.Equal(out[j].at) {
			return out[i].at.Before(out[j].at)
		}
		return out[i].buckets == "" && out[j].buckets != ""
	})
	return out
}

// monthDays is a month directory's days, each its file and its buckets'
// directory, whichever stand.
func monthDays(mdir, month string, visits *int) []oldFile {
	byDate := map[string]oldFile{}
	for _, e := range readEntries(mdir, visits) { // a month's days (P10-02)
		dd := trimExt(e.Name())
		date, err := time.Parse("2006-01/02", month+"/"+dd)
		if err != nil || e.Name()[0] == '.' {
			continue
		}
		byDate[dd] = oldFile{at: date, path: filepath.Join(mdir, dd+".json.gz"), buckets: filepath.Join(mdir, dd)}
	}
	out := make([]oldFile, 0, len(byDate))
	for _, f := range byDate { // P10-02
		out = append(out, f)
	}
	return out
}

// readEntries is a directory's entries, counted as a visit; none past the
// pass's visits or where it cannot be read.
func readEntries(dir string, visits *int) []os.DirEntry {
	if *visits >= maxPruneVisits {
		return nil
	}
	*visits++
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	return entries
}

// removeFiles removes an old file and, for a day, its buckets: the files
// removed.
func (s *Store) removeFiles(f oldFile) int {
	removed := 0
	if s.removeCounted(f.path) {
		removed++
	}
	if f.buckets == "" {
		return removed
	}
	files, _ := os.ReadDir(f.buckets)
	for i, b := range files { // bounded by maxDayBuckets (P10-02)
		if i < maxDayBuckets && s.removeCounted(filepath.Join(f.buckets, b.Name())) {
			removed++
		}
	}
	_ = os.Remove(f.buckets) // only once empty
	return removed
}

// Bounded reports whether a dataset's byte bound has cut it short of its
// retention (D-143) since the store opened, was cleared, or the dataset's
// retention changed: the Data tab says so.
func (s *Store) Bounded(dataset string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bounded[dataset]
}

// cursorAfter reports whether series a comes after b, dataset then source
// then place.
func cursorAfter(a, b [3]string) bool {
	for i := range a { // three (P10-02)
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// listSeries is a dataset version's series directories, source then place.
func listSeries(vdir string, visits *int) []string {
	if vdir == "" {
		return nil
	}
	if *visits >= maxPruneVisits {
		return nil
	}
	var out []string
	for _, src := range readDirs(vdir, visits) { // bounded by maxPruneVisits (P10-02)
		for _, place := range readDirs(filepath.Join(vdir, src), visits) {
			out = append(out, filepath.Join(vdir, src, place))
		}
	}
	return out
}

func readDirs(dir string, visits *int) []string {
	if *visits >= maxPruneVisits {
		return nil
	}
	*visits++
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // absent or removed under us
	}
	var out []string
	for _, e := range entries { // P10-02
		if e.IsDir() && e.Name()[0] != '.' {
			out = append(out, e.Name())
		}
	}
	return out
}

// pruneSeries compacts a series' finished days, and rolls up and removes its
// days past retention; the number of files removed, at most budget. A month
// directory it empties, or finds empty, is removed, and the series' own and
// its source's once they hold nothing.
func (s *Store) pruneSeries(d Dataset, dir string, now time.Time, visits *int, budget int) int {
	if dir == "" {
		return 0
	}
	if budget <= 0 {
		return 0
	}
	removed := 0
	hourCut := now.Add(-d.Hours).Truncate(24 * time.Hour)
	months := readDirs(dir, visits)
	left := len(months)
	for _, month := range months { // P10-02
		before, mdir := removed, filepath.Join(dir, month)
		empty := false
		if month == "rollup" {
			removed += s.pruneRollups(d, mdir, now, budget-removed)
		} else if entries, err := os.ReadDir(mdir); err == nil {
			for _, e := range entries { // a month's days and their buckets (P10-02)
				if removed >= budget {
					return removed
				}
				removed += s.pruneEntry(d, dir, month, e, now, hourCut)
			}
			empty = len(entries) == 0
		}
		if (empty || removed > before) && os.Remove(mdir) == nil { // only once empty
			left--
		}
	}
	if left == 0 && os.Remove(dir) == nil { // only once empty
		_ = os.Remove(filepath.Dir(dir)) // the source, only once empty
	}
	return removed
}

// pruneEntry handles one entry of a month: a stale temp file or claim swept;
// a finished day's buckets compacted into its file; a day past the hours'
// retention rolled up and removed. The files removed.
func (s *Store) pruneEntry(d Dataset, dir, month string, e fs.DirEntry, now, hourCut time.Time) int {
	if e == nil || dir == "" {
		return 0
	}
	name := e.Name()
	path := filepath.Join(dir, month, name)
	if name[0] == '.' {
		s.sweep(path, e, now)
		return 0
	}
	date, err := time.Parse("2006-01/02", month+"/"+trimExt(name))
	if err != nil {
		return 0
	}
	expired := date.Before(hourCut)
	if expired {
		return s.expireDay(d, dir, date, now)
	}
	done := e.IsDir() && now.Sub(date) >= 24*time.Hour+d.step()
	if done {
		return s.compact(d, dir, date)
	}
	return 0
}

// sweep removes a temp file a crashed writer left, or a stale claim.
func (s *Store) sweep(path string, e fs.DirEntry, now time.Time) {
	if path == "" || e == nil {
		return
	}
	claim := strings.HasPrefix(e.Name(), ".claim-") || strings.HasPrefix(e.Name(), ".compact-")
	if claim {
		if s.claimStale(path, now) {
			_ = os.Remove(path)
		}
		return
	}
	info, err := e.Info()
	if err != nil {
		return
	}
	if time.Since(info.ModTime()) > claimStale {
		s.removeCounted(path) // a temp file: its writer is long gone
	}
}

// compact writes a finished day's records into its one file, under the
// day's claim, then removes each bucket file the day's file now holds at an
// issue at least as new. A bucket written meanwhile is left for the next
// pass. The bucket files removed.
func (s *Store) compact(d Dataset, dir string, date time.Time) int {
	f := filesAt(dir, date)
	if f.day == "" {
		return 0
	}
	claim := filepath.Join(filepath.Dir(f.buckets), ".compact-"+date.Format("02"))
	held := s.claim(claim)
	if !held {
		return 0
	}
	defer func() { _ = os.Remove(claim) }() // the claim let go
	entries := s.dayEntriesOnce(dir, d, date)
	if len(entries) == 0 {
		return 0
	}
	doc := dayDoc{Schema: schema, Dataset: d.Name, Version: d.Version, Key: keyOf(dir), Date: date.Format("2006-01-02"), Hours: entries}
	_, _, err := s.writeCounted(f.day, doc)
	if err != nil {
		return 0
	}
	written, ok := readAs[dayDoc](s, f.day, d)
	if !ok {
		return 0 // the day written cannot be read back: no hour's bucket goes
	}
	removed := 0
	files, _ := os.ReadDir(f.buckets)
	for i, b := range files { // bounded by maxDayBuckets (P10-02)
		path := filepath.Join(f.buckets, b.Name())
		if i < maxDayBuckets && b.Name()[0] != '.' && s.heldBy(written.Hours, path, d) && s.removeCounted(path) {
			removed++
		}
	}
	_ = os.Remove(f.buckets) // only once empty
	return removed
}

// heldBy reports whether a bucket file's record is held by entries at an
// issue at least as new: only then may the file go.
func (s *Store) heldBy(entries []hourEntry, path string, d Dataset) bool {
	if len(entries) == 0 || path == "" {
		return false
	}
	doc, ok := readAs[dayDoc](s, path, d)
	if !ok || len(doc.Hours) != 1 {
		return false
	}
	have, ok := findHour(entries, doc.Hours[0].Hour)
	return ok && !have.Issued.Before(doc.Hours[0].Issued)
}

// keyOf is a series directory's key: its source and place.
func keyOf(dir string) Key {
	if dir == "" {
		return Key{}
	}
	return Key{Source: filepath.Base(filepath.Dir(dir)), Place: filepath.Base(dir)}
}

// expireDay rolls a day past the hours' retention into its year, where the
// dataset keeps days, and only then removes its file and buckets. The files
// removed.
func (s *Store) expireDay(d Dataset, dir string, date time.Time, now time.Time) int {
	f := filesAt(dir, date)
	if f.day == "" {
		return 0
	}
	keep := d.Days > 0 && now.Sub(date) < d.Hours+d.Days
	if keep && !s.rollUp(d, dir, date) {
		return 0 // not rolled up: its hours stay until it is
	}
	removed := 0
	if s.removeCounted(f.day) {
		removed++
	}
	files, _ := os.ReadDir(f.buckets)
	for i, b := range files { // bounded by maxDayBuckets (P10-02)
		if i < maxDayBuckets && s.removeCounted(filepath.Join(f.buckets, b.Name())) {
			removed++
		}
	}
	_ = os.Remove(f.buckets)
	if removed == 0 {
		return 0
	}
	s.note(func(st *Stats) { st.Pruned++ }, true)
	return removed
}

// pruneRollups removes roll-ups wholly past the days' retention: the months'
// parts, and an older release's yearly roll-ups.
func (s *Store) pruneRollups(d Dataset, dir string, now time.Time, budget int) int {
	if dir == "" || budget <= 0 {
		return 0
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	removed := 0
	for _, e := range entries { // a file a year, or a part a month (P10-02)
		r, ok := parseRollup(trimExt(e.Name()))
		if !ok || removed >= budget {
			continue
		}
		if now.Sub(r.end) > d.Hours+d.Days && s.removeCounted(filepath.Join(dir, e.Name())) {
			removed++
			s.note(func(st *Stats) { st.Pruned++ }, true)
		}
	}
	return removed
}
