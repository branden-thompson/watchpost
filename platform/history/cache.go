package history

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/branden-thompson/watchpost/platform/agememo"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

// heldDay is a day's hours as read, and its files when they were read.
type heldDay struct {
	files   dayFiles
	entries []hourEntry
}

// weight is what a held day keeps in memory, near enough to bound by: each
// hour's document and values, and a little for each hour and field (D-242).
func (h heldDay) weight() int64 {
	w := int64(0)
	for _, e := range h.entries { // a day's buckets (P10-02)
		w += 96 + int64(len(e.Doc))
		for k, v := range e.Values { // a dataset's fields (P10-02)
			w += 48 + int64(len(k)) + 8*int64(len(v))
		}
	}
	return w
}

// heldIndex is a day's hours as recording reads them - each hour and when it
// was issued - and the day's files when they were read (D-242).
type heldIndex struct {
	files dayFiles
	hours []hourMark
}

// hourMark is one hour recorded and when it was issued.
type hourMark struct {
	Hour, Issued time.Time
}

// marksOf are a day's hours as recording reads them.
func marksOf(es []hourEntry) []hourMark {
	out := make([]hourMark, 0, len(es))
	for _, e := range es { // a day's buckets (P10-02)
		out = append(out, hourMark{Hour: e.Hour, Issued: e.Issued})
	}
	return out
}

// issuedAt is when the day's hour was issued, as recorded; false for an
// hour not recorded.
func (x heldIndex) issuedAt(hour time.Time) (time.Time, bool) {
	for _, m := range x.hours { // a day's buckets (P10-02)
		if m.Hour.Equal(hour) {
			return m.Issued, true
		}
	}
	return time.Time{}, false
}

// withMark is a day's marks with m in its hour's place, in order: a new
// slice, as the held one is shared.
func withMark(ms []hourMark, m hourMark) []hourMark {
	out := make([]hourMark, 0, len(ms)+1)
	placed := false
	for _, h := range ms { // a day's buckets (P10-02)
		if !placed && !h.Hour.Before(m.Hour) {
			out, placed = append(out, m), true
		}
		if !h.Hour.Equal(m.Hour) {
			out = append(out, h)
		}
	}
	if !placed {
		out = append(out, m)
	}
	return out
}

// dayFiles is what a day's files are: the day's own document's time and
// size, its bucket directory's time and how many it holds, and its buckets'
// newest time and their bytes. Any write - this instance's by a rename,
// another's, or one in place - moves one of them.
type dayFiles struct {
	dayMod       time.Time
	daySize      int64
	bucketsMod   time.Time
	buckets      int
	newestBucket time.Time
	bucketBytes  int64
}

// dayRules keep a day's hours for as long as its files stand - the files
// decide, not the age - for the series and days a map replays: 256 days at
// most, weighing 8 MB at most together (D-242), one AirNow day 2.5 MB.
var dayRules = agememo.Options{Fresh: 24 * time.Hour, Max: 256, MaxBytes: 8 << 20}

// indexRules keep a day's hours as recording reads them: an hour and its
// issue, about 1 KB a day, for the days being written.
var indexRules = agememo.Options{Fresh: 24 * time.Hour, Max: 512}

// keepPut adds a record just written to its day's index, so the next Put of
// the day reads none of its files again (PF-8) - when the day's files are now
// what they were with only that bucket written. Any other change - another
// instance's bucket, a temp file, a compaction - has the day read again. The
// day's documents, held for readers, are forgotten: they are no longer the
// day's files (D-242).
func (s *Store) keepPut(f seriesFiles, held heldIndex, e hourEntry, written fs.FileInfo, had int64) {
	if s.index == nil || written == nil {
		return
	}
	s.days.Forget(f.day)
	if had < -1 { // a size, or -1 for none: anything else is no size this store wrote
		s.index.Forget(f.day)
		return
	}
	newest := written.ModTime().After(held.files.newestBucket)
	if !newest {
		s.index.Forget(f.day)
		return
	}
	want := held.files
	want.newestBucket = written.ModTime()
	want.bucketBytes += written.Size() - max(had, 0)
	if had < 0 {
		want.buckets++
	}
	now, _ := statDay(f)
	want.bucketsMod = now.bucketsMod // a rename moves the directory's time
	s.index.Forget(f.day)
	if now != want {
		return
	}
	s.index.Put(f.day, heldIndex{files: now, hours: withMark(held.hours, hourMark{Hour: e.Hour, Issued: e.Issued})})
}

// maxDayBuckets bounds a day's bucket files read: a minute's step fills 1,440.
const maxDayBuckets = 1440

func (s *Store) dayEntries(dir string, d Dataset, day time.Time) []hourEntry {
	if dir == "" {
		return nil
	}
	return s.heldDayOf(filesAt(dir, day), d, day).entries
}

// indexOf is a day's hours as recording reads them: the index held while the
// day's files stand, else the day's documents held for readers, else the
// day's files read - the index alone then kept (D-242).
func (s *Store) indexOf(f seriesFiles, d Dataset, day time.Time) heldIndex {
	if f.day == "" {
		return heldIndex{}
	}
	now, entries := statDay(f)
	if s.index == nil {
		return heldIndex{files: now, hours: marksOf(s.readDay(f, d, day, entries))}
	}
	if held, _, ok := s.index.Last(f.day); ok && held.files == now {
		return held
	}
	x := heldIndex{files: now}
	if held, _, ok := s.days.Last(f.day); ok && held.files == now {
		x.hours = marksOf(held.entries)
	} else {
		x.hours = marksOf(s.readDay(f, d, day, entries))
	}
	s.index.Put(f.day, x)
	return x
}

// dayEntriesOnce is a day's hours for compaction and roll-up, which read a
// finished day once and rewrite it: the day held for readers if its files
// stand, else read and held by nothing, so a day being rewritten never
// pushes out the days a map replays (D-242).
func (s *Store) dayEntriesOnce(dir string, d Dataset, day time.Time) []hourEntry {
	if dir == "" {
		return nil
	}
	f := filesAt(dir, day)
	if f.day == "" {
		return nil
	}
	now, entries := statDay(f)
	if held, _, ok := s.days.Last(f.day); ok && held.files == now {
		return held.entries
	}
	return s.readDay(f, d, day, entries)
}

// heldDayOf is a day's hours and what its files were when they were read:
// read again only when the files have changed since.
func (s *Store) heldDayOf(f seriesFiles, d Dataset, day time.Time) heldDay {
	if f.day == "" {
		return heldDay{}
	}
	now, entries := statDay(f)
	if s.days == nil {
		return heldDay{files: now, entries: s.readDay(f, d, day, entries)}
	}
	// THE FILES DECIDE, NOT THE AGE: what was read stands while they do, so the
	// day is the last read whatever its age, read again when its files change.
	if held, _, ok := s.days.Last(f.day); ok && held.files == now {
		return held
	}
	held := heldDay{files: now, entries: s.readDay(f, d, day, entries)}
	s.days.Put(f.day, held)
	return held
}

// statDay is what a day's files are now - its document's time and size, its
// bucket directory's time and count, its buckets' newest time and bytes - and
// the bucket directory's entries.
func statDay(f seriesFiles) (dayFiles, []os.DirEntry) {
	var now dayFiles
	if f.day == "" {
		return now, nil
	}
	entries, _ := os.ReadDir(f.buckets)
	if fi, err := os.Stat(f.day); err == nil {
		now.dayMod, now.daySize = fi.ModTime(), fi.Size()
	}
	if fi, err := os.Stat(f.buckets); err == nil {
		now.bucketsMod = fi.ModTime()
	}
	now.buckets = len(entries)
	for i, e := range entries { // bounded by maxDayBuckets (P10-02)
		if fi, err := e.Info(); err == nil && i < maxDayBuckets {
			now.bucketBytes += fi.Size()
			if fi.ModTime().After(now.newestBucket) {
				now.newestBucket = fi.ModTime()
			}
		}
	}
	return now, entries
}

// readDay reads a day's hours from its files: the day's document, then its
// buckets, the newest issue of each hour standing. An hour that is not the
// day's is a misfiled or damaged document: not read, and counted.
func (s *Store) readDay(f seriesFiles, d Dataset, day time.Time, entries []os.DirEntry) []hourEntry {
	s.note(func(st *Stats) { st.DayReads++ }, true)
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.AddDate(0, 0, 1)
	corrupt := func(st *Stats) { st.Corrupt++ }
	byAt := map[time.Time]hourEntry{}
	if doc, ok := readAs[dayDoc](s, f.day, d); ok {
		for _, h := range doc.Hours { // a day's buckets (P10-02)
			inDay := !h.Hour.Before(start) && h.Hour.Before(end)
			if err := invariant.Check(inDay, "history: an hour filed under another day"); err != nil {
				s.note(corrupt, false)
				continue
			}
			byAt[h.Hour] = h
		}
	}
	for i, e := range entries { // bounded by maxDayBuckets (P10-02)
		if i >= maxDayBuckets || e.Name()[0] == '.' {
			continue
		}
		if doc, ok := readAs[dayDoc](s, filepath.Join(f.buckets, e.Name()), d); ok && len(doc.Hours) == 1 {
			h := doc.Hours[0]
			inDay := !h.Hour.Before(start) && h.Hour.Before(end)
			if err := invariant.Check(inDay, "history: an hour filed under another day"); err != nil {
				s.note(corrupt, false)
				continue
			}
			if have, ok := byAt[h.Hour]; !ok || have.Issued.Before(h.Issued) {
				byAt[h.Hour] = h
			}
		}
	}
	out := make([]hourEntry, 0, len(byAt))
	for _, h := range byAt { // P10-02
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hour.Before(out[j].Hour) })
	// A DAY HOLDS NO MORE HOURS THAN ITS STEP ALLOWS: more is a document whose
	// hours are off the step - damaged - and the day is read as absent.
	most := int(24 * time.Hour / d.step())
	if err := invariant.Check(len(out) <= most, "history: a day holds more hours than its step allows"); err != nil {
		s.note(corrupt, false)
		return nil
	}
	return out
}

// Memory is what the store holds in memory (D-242): the days held for
// readers and what they weigh, and the days indexed for recording.
func (s *Store) Memory() (days int, dayBytes int64, indexed int) {
	if s == nil || s.days == nil {
		return 0, 0, 0
	}
	days, dayBytes = s.days.Held()
	indexed, _ = s.index.Held()
	return days, dayBytes, indexed
}
