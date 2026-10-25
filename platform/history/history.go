// Package history is watchpost's local record of what its sources said, by
// time (W18.3; D-166, D-169 to D-173, D-175 to D-178): any dataset, recorded
// by hour, read back by range, kept for its own retention and rolled up into
// days past it. The map's fallback is its first reader; trend views a later
// one. It knows nothing of weather: a record is named values over a shape.
//
// ON DISK (D-170, D-171): <root>/<dataset>/v<version>/<source>/<place>/.
// A record is written to its own bucket file, <YYYY-MM>/<DD>/<HHMM>.json.gz,
// which no writer of another bucket touches; once its day is done the
// buckets are compacted into the day's one file, <YYYY-MM>/<DD>.json.gz, and
// past the hours' retention the day is rolled up into its month's parts,
// rollup/<YYYY-MM>-p<N>.json.gz, each within a budget below the read cap
// (a series' yearly roll-up, rollup/<YYYY>.json.gz, is read too). A year of a series is 365
// day files, not thousands, and `zcat` shows any of it. Every read merges a
// day's file with its buckets not yet compacted. A dataset past its byte
// bound loses its oldest files first (D-143).
//
// SEVERAL INSTANCES, ONE STORE (design section 4, 4b). Every write is a temp
// file renamed over its target, so a reader sees a whole document or the one
// before it. No record is ever read, merged and replaced by a writer - that
// loses another's record under contention - so each bucket is its own file. A
// day is compacted under a claim, and a bucket file is removed only once the
// day's
// file holds it at an issue at least as new. A bucket is fetched by one
// instance: Claim creates a claim file exclusively, holding its time; a claim
// older than claimStale with nothing recorded may be taken over. Nothing here
// locks; every removal tolerates what is already gone.
//
// NOTHING HERE IS SHOWN TO THE LISTENER (D-124). A disk that cannot be used,
// a file that does not parse, a version not ours: each reads as absent and is
// counted in Stats, for the diagnostics.
package history

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/agememo"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

// schema is the on-disk document's own version: a document of another is
// not read.
const schema = 1

// Field is one named value a dataset records: its name, the words a reader
// shows for it, its unit, and the decimals kept on disk - temp, "Temperature",
// °C, 1; muf, "Maximum usable frequency", MHz, 2.
type Field struct {
	Name, Label, Unit string
	Decimals          int
}

// Dataset is a kind of record: its name and version; its step - the width
// of one record's bucket, a minute to a day (an hour when zero): NDFD's
// hours, an ionosonde's MUF every 5 or 15 minutes, a tide gauge's 6; its
// numeric fields; how long its records are kept (Hours) and how long its
// days, rolled up, past that (Days; zero keeps no roll-up); and the most it
// holds on disk, every version (MaxBytes, D-143: past it, its oldest files go
// first, before its retention ends; zero, no bound). Roll-ups are of the
// numeric fields; a record's document is kept, not rolled up.
type Dataset struct {
	Name, Title, Description string // what it is, for a reader that did not register it (an Analyst's catalog)
	Version                  int
	Step                     time.Duration
	Fields                   []Field
	Hours                    time.Duration
	Days                     time.Duration
	MaxBytes                 int64
}

// step is the dataset's bucket width, an hour by default, held to a minute
// to a day.
func (d Dataset) step() time.Duration {
	if d.Step <= 0 {
		return time.Hour
	}
	if d.Step < time.Minute {
		return time.Minute
	}
	if d.Step > 24*time.Hour {
		return 24 * time.Hour
	}
	return d.Step
}

// Key is a series: whose (Source) and where (Place) - {ndfd, us-a}, or a
// place's own, {station, ksea}. Each is [a-z0-9._-], at most 64 bytes.
type Key struct{ Source, Place string }

// Shape is what a record's values cover: a lattice of Cols x Rows points
// over Box, or with one of each, a point.
type Shape struct {
	Box        geo.Box
	Cols, Rows int
}

// Record is one bucket of a series, in either or both of two payloads:
// Values, each numeric field over its shape - rows from the north, each west
// to east; a missing value NaN - for grids and points (temperatures, a
// station's MUF, a buoy's waves); and Doc, a JSON document of at most
// maxDocPayload bytes whose shape is the dataset's own (versioned with it) -
// for what is not a number over a shape: alerts, quakes, fire perimeters.
type Record struct {
	Key      Key
	At       time.Time // the bucket's start, UTC
	IssuedAt time.Time // when the source issued it: a newer record replaces an older
	Shape    Shape
	Values   map[string][]float64
	Doc      json.RawMessage
}

// maxDocPayload bounds a record's document: a region's alerts with their
// polygons, with room. Pictures are not recorded here: the HTTP cache and
// the map library keep those (a later payload kind would carry its own
// budget).
const maxDocPayload = 1 << 20

// Day is one day of a series rolled up: each field's minimum, maximum and
// mean at each point over the day's recorded hours, and how many there were.
type Day struct {
	Key   Key
	Date  time.Time // the day's start, UTC
	Shape Shape
	Hours int
	Min   map[string][]float64
	Max   map[string][]float64
	Mean  map[string][]float64
}

// Stats is what the store has done, for the diagnostics (D-124).
type Stats struct {
	Puts, PutFailures, Skipped                 int64
	Corrupt, VersionMismatch, Pruned, RolledUp int64
	// DayReads is how many times a day's hours were read from its files: a
	// day read again only when its files changed (W14 P-18).
	DayReads int64
}

// Store is a history on disk. The zero root is a store that does nothing.
type Store struct {
	root  string
	sets  map[string]Dataset
	now   func() time.Time
	mu    sync.Mutex
	stats Stats
	// days keeps each day's hours as read for the store's readers, by the
	// day's file, with what its files were when read: read again only when
	// they change (W14 P-18), held within dayRules by count and by weight
	// (D-242).
	days *agememo.Memo[string, heldDay]
	// index keeps each day's hours as recording reads them - each hour and
	// when it was issued, never its document - so recording holds no
	// document (D-242).
	index *agememo.Memo[string, heldIndex]
	// SIZE, KEPT AS IT GOES (PF-2): what each dataset's directory holds on
	// disk, by its name ("" for a file at the root), and the oldest day held -
	// walked once, at the first read or the first prune, then kept by every
	// write and removal, and walked again every remeasureEvery for what other
	// instances wrote. measured is when it was last walked; zero, never.
	sizes    map[string]int64
	oldest   time.Time
	measured time.Time
	// cursor is the series the last prune pass finished with - dataset,
	// source, place - so the next pass goes on past it (QA-4); zero, from the
	// start.
	cursor [3]string
	// bounded is each dataset its byte bound has cut short of its retention
	// (D-143), since the store opened, was cleared, or the retention changed.
	bounded map[string]bool
}

// DefaultRoot is $XDG_DATA_HOME/watchpost/weather/history, by default
// ~/.local/share/...; "" where neither resolves (the store then does
// nothing). A relative XDG_DATA_HOME is refused, as config.Path refuses one.
func DefaultRoot() string {
	base := os.Getenv("XDG_DATA_HOME")
	abs := filepath.IsAbs(base)
	if abs {
		return filepath.Join(base, "watchpost", "weather", "history")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share", "watchpost", "weather", "history")
}

// Open is a store at root holding the datasets given. It never fails: a root
// that cannot be made is a store whose writes fail and whose reads find
// nothing, each counted. Each dataset's manifest is written beside its
// records, so any reader - another instance, an Analyst - knows what it is.
func Open(root string, now func() time.Time, sets ...Dataset) *Store {
	s := &Store{root: root, now: now, sets: map[string]Dataset{}}
	if now == nil {
		s.now = time.Now
	}
	s.days = agememo.NewSized[string, heldDay](agememo.Options{Fresh: dayRules.Fresh, Max: dayRules.Max, MaxBytes: dayRules.MaxBytes, Now: s.now}, heldDay.weight)
	s.index = agememo.New[string, heldIndex](agememo.Options{Fresh: indexRules.Fresh, Max: indexRules.Max, Now: s.now})
	for _, d := range sets { // a handful (P10-02)
		s.sets[d.Name] = d
	}
	if root == "" {
		return s
	}
	err := os.MkdirAll(root, 0o700)
	if err != nil {
		return s // a failure shows as failed writes
	}
	for _, d := range sets { // P10-02
		s.writeManifest(d)
	}
	return s
}

// manifest is a dataset version's description on disk.
type manifest struct {
	Schema  int     `json:"schema"`
	Dataset Dataset `json:"dataset"`
}

// versionDir is a dataset version's directory, "" where the store has no
// root or the name is not a path segment.
func (s *Store) versionDir(name string, version int) string {
	if s.root == "" || version < 0 {
		return ""
	}
	if !validSegment(name) {
		return ""
	}
	return filepath.Join(s.root, name, "v"+strconv.Itoa(version))
}

// writeManifest writes a dataset's manifest when it is absent or differs.
func (s *Store) writeManifest(d Dataset) {
	vdir := s.versionDir(d.Name, d.Version)
	if vdir == "" {
		return
	}
	var have manifest
	want, _ := json.Marshal(d)
	got, _ := json.Marshal(have.Dataset)
	found := s.readDoc(filepath.Join(vdir, "manifest.json.gz"), &have)
	if found {
		got, _ = json.Marshal(have.Dataset)
	}
	if found && have.Schema == schema && bytes.Equal(want, got) {
		return
	}
	_, _, _ = s.writeCounted(filepath.Join(vdir, "manifest.json.gz"), manifest{Schema: schema, Dataset: d})
}

// maxCatalog bounds the catalog: datasets and their versions.
const maxCatalog = 256

// Catalog is every dataset version on disk, from their manifests - those this
// store registered and those another instance or version wrote - for a reader
// that browses what is recorded (the Analyst mode to come, D-169).
func (s *Store) Catalog() []Dataset {
	if s.root == "" {
		return nil
	}
	var out []Dataset
	visits := 0
	for _, name := range readDirs(s.root, &visits) { // bounded (P10-02)
		for _, v := range readDirs(filepath.Join(s.root, name), &visits) {
			if len(out) >= maxCatalog {
				return out
			}
			var m manifest
			if s.readDoc(filepath.Join(s.root, name, v, "manifest.json.gz"), &m) && m.Schema == schema {
				out = append(out, m.Dataset)
			}
		}
	}
	return out
}

// Series is a dataset's recorded series, at most max.
func (s *Store) Series(dataset string, max int) []Key {
	d, ok := s.dataset(dataset)
	if !ok || max <= 0 {
		return nil
	}
	vdir := s.versionDir(d.Name, d.Version)
	if vdir == "" {
		return nil
	}
	var out []Key
	visits := 0
	for _, src := range readDirs(vdir, &visits) { // bounded (P10-02)
		for _, place := range readDirs(filepath.Join(vdir, src), &visits) {
			if len(out) >= max {
				return out
			}
			out = append(out, Key{Source: src, Place: place})
		}
	}
	return out
}

// Extent is a series' first and last recorded day, from its day files, its
// buckets and its roll-ups: what a reader can ask of it.
func (s *Store) Extent(dataset string, k Key) (first, last time.Time, ok bool) {
	dir, _, found := s.seriesDir(dataset, k)
	if !found {
		return time.Time{}, time.Time{}, false
	}
	visits := 0
	for _, month := range readDirs(dir, &visits) { // bounded (P10-02)
		for _, day := range dayNames(filepath.Join(dir, month), month) {
			if first.IsZero() || day.Before(first) {
				first = day
			}
			if day.After(last) {
				last = day
			}
		}
	}
	if first.IsZero() {
		return first, last, false
	}
	return first, last, true
}

// dayNames are the days a month directory holds - its day files and its
// days' bucket directories - or the first days of a roll-up directory's
// files, its years and its months' parts.
func dayNames(dir, month string) []time.Time {
	if dir == "" || month == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []time.Time
	for _, e := range entries { // a month's days, or a roll-up directory's files (P10-02)
		name := trimExt(e.Name())
		if t, err := time.Parse("2006-01/02", month+"/"+name); err == nil && name[0] != '.' {
			out = append(out, t)
		} else if r, ok := parseRollup(name); ok && month == "rollup" {
			out = append(out, r.start)
		}
	}
	return out
}

// trimExt is a file's name without its .json.gz (or .json).
func trimExt(name string) string {
	if name == "" {
		return name
	}
	if name[0] == '.' {
		return name // a claim or a temp file: not a record's name
	}
	return strings.TrimSuffix(strings.TrimSuffix(name, ".gz"), ".json")
}

// seriesDir is a series' directory, or false for a dataset not held or a
// key that is not a path segment.
func (s *Store) seriesDir(dataset string, k Key) (string, Dataset, bool) {
	d, ok := s.dataset(dataset)
	if !ok {
		return "", Dataset{}, false
	}
	valid := validSegment(k.Source) && validSegment(k.Place)
	if !valid {
		return "", Dataset{}, false
	}
	vdir := s.versionDir(d.Name, d.Version)
	if vdir == "" {
		return "", Dataset{}, false
	}
	return filepath.Join(vdir, k.Source, k.Place), d, true
}

// validSegment reports whether v is a path segment the store writes: one to
// 64 bytes of [a-z0-9._-], never "." or "..".
func validSegment(v string) bool {
	if len(v) == 0 || len(v) > 64 {
		return false
	}
	if v == "." || v == ".." {
		return false
	}
	for i := 0; i < len(v); i++ { // at most 64 (P10-02)
		c := v[i]
		ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-'
		if !ok {
			return false
		}
	}
	return true
}

// seriesFiles are where a series' records for one time lie: the day's
// compacted file, its directory of buckets, the bucket's own file, and an
// older release's yearly roll-up - read while one is held, never written:
// roll-ups are a month's parts since 0.19.0 (#27, W1.2), and pruneRollups
// removes a yearly one once its days are past retention, so none outlives
// the longest retention, a year, after the release that stopped writing them.
type seriesFiles struct{ day, buckets, bucket, oldYear string }

// filesAt is a series' files for the bucket at; empty for no series.
func filesAt(dir string, at time.Time) seriesFiles {
	if dir == "" {
		return seriesFiles{}
	}
	if at.IsZero() {
		return seriesFiles{}
	}
	month, dd := filepath.Join(dir, at.Format("2006-01")), at.Format("02")
	return seriesFiles{
		day:     filepath.Join(month, dd+".json.gz"),
		buckets: filepath.Join(month, dd),
		bucket:  filepath.Join(month, dd, at.Format("1504")+".json.gz"),
		oldYear: filepath.Join(dir, "rollup", strconv.Itoa(at.Year())+".json.gz"),
	}
}

// dayDoc is a day's document on disk.
type dayDoc struct {
	Schema  int         `json:"schema"`
	Dataset string      `json:"dataset"`
	Version int         `json:"version"`
	Key     Key         `json:"key"`
	Date    string      `json:"date"`
	Hours   []hourEntry `json:"hours"`
}

type hourEntry struct {
	Hour   time.Time         `json:"at"`
	Issued time.Time         `json:"issued"`
	Shape  Shape             `json:"shape"`
	Values map[string]values `json:"values,omitempty"`
	Doc    json.RawMessage   `json:"doc,omitempty"`
}

// rollupDoc is roll-ups on disk: a month's part, or an older release's yearly
// roll-up, read alike.
type rollupDoc struct {
	Schema  int        `json:"schema"`
	Dataset string     `json:"dataset"`
	Version int        `json:"version"`
	Key     Key        `json:"key"`
	Days    []dayEntry `json:"days"`
}

type dayEntry struct {
	Date  string                       `json:"date"`
	Shape Shape                        `json:"shape"`
	Hours int                          `json:"hours"`
	Stats map[string]map[string]values `json:"stats"` // field -> min|max|mean -> values
}

// Record records one bucket of a series, in the bucket's own file. It is skipped
// when an equal or newer issue of that bucket is already recorded. False
// when it could not be written.
func (s *Store) Record(dataset string, r Record) bool {
	failed := func(st *Stats) { st.PutFailures++ }
	dir, d, ok := s.seriesDir(dataset, r.Key)
	if !ok {
		return s.note(failed, false)
	}
	valid := validRecord(d, r)
	if !valid {
		return s.note(failed, false)
	}
	at := r.At.UTC().Truncate(d.step())
	entry := toEntry(d, r, at)
	f := filesAt(dir, at)
	held := s.indexOf(f, d, at)
	issued, seen := held.issuedAt(at)
	stale := seen && !issued.Before(entry.Issued)
	if stale {
		return s.note(func(st *Stats) { st.Skipped++ }, true)
	}
	doc := dayDoc{Schema: schema, Dataset: d.Name, Version: d.Version, Key: r.Key, Date: at.Format("2006-01-02"), Hours: []hourEntry{entry}}
	written, had, err := s.writeCounted(f.bucket, doc)
	if err != nil {
		return s.note(failed, false)
	}
	s.noteDay(at)
	s.keepPut(f, held, entry, written, had)
	return s.note(func(st *Stats) { st.Puts++ }, true)
}

// validRecord reports whether a record is one d holds: a time; values only
// over a shape, each field's as many as the shape's points; a document of
// JSON within its bound.
func validRecord(d Dataset, r Record) bool {
	n := r.Shape.Cols * r.Shape.Rows
	if err := invariant.Check(r.Shape.Cols >= 0 && r.Shape.Rows >= 0, "a shape is not negative"); err != nil {
		return false
	}
	if n == 0 && len(r.Values) > 0 {
		return false // values over no shape
	}
	if len(r.Doc) > maxDocPayload {
		return false
	}
	if len(r.Doc) > 0 && !json.Valid(r.Doc) {
		return false
	}
	for _, f := range d.Fields { // a dataset's few (P10-02)
		v, ok := r.Values[f.Name]
		if ok && len(v) != n {
			return false
		}
	}
	return !r.At.IsZero()
}

// toEntry is a record as its bucket's entry on disk: each of d's fields
// rounded to its decimals, the document as it is.
func toEntry(d Dataset, r Record, hour time.Time) hourEntry {
	e := hourEntry{Hour: hour, Issued: r.IssuedAt.UTC(), Shape: r.Shape, Values: map[string]values{}, Doc: r.Doc}
	if len(r.Values) == 0 {
		return e
	}
	if len(d.Fields) == 0 {
		return e
	}
	for _, f := range d.Fields { // P10-02
		if v, ok := r.Values[f.Name]; ok {
			e.Values[f.Name] = encode(v, f.Decimals)
		}
	}
	return e
}

// findHour is the entry for a bucket among entries.
func findHour(hs []hourEntry, hour time.Time) (hourEntry, bool) {
	if len(hs) == 0 {
		return hourEntry{}, false
	}
	for _, h := range hs { // at most a day's buckets (P10-02)
		if h.Hour.Equal(hour) {
			return h, true
		}
	}
	return hourEntry{}, false
}

// Get is one bucket of a series - the one at holds - as recorded.
func (s *Store) Get(dataset string, k Key, at time.Time) (Record, bool) {
	dir, d, ok := s.seriesDir(dataset, k)
	if !ok {
		return Record{}, false
	}
	at = at.UTC().Truncate(d.step())
	h, ok := findHour(s.dayEntries(dir, d, at), at)
	if !ok {
		return Record{}, false
	}
	return fromEntry(k, h), true
}

// Range is a series' records from from to to, oldest first, at most max.
func (s *Store) Range(dataset string, k Key, from, to time.Time, max int) []Record {
	dir, d, ok := s.seriesDir(dataset, k)
	if !ok || max <= 0 {
		return nil
	}
	from, to = from.UTC().Truncate(d.step()), to.UTC()
	var out []Record
	days := int(to.Sub(from).Hours()/24) + 2
	for i := 0; i < days && len(out) < max; i++ { // bounded by the span (P10-02)
		day := time.Date(from.Year(), from.Month(), from.Day()+i, 0, 0, 0, 0, time.UTC)
		for _, h := range s.dayEntries(dir, d, day) { // a day's buckets (P10-02)
			if !h.Hour.Before(from) && !h.Hour.After(to) && len(out) < max {
				out = append(out, fromEntry(k, h))
			}
		}
	}
	return out
}

// Latest is a series' newest record at or before at, no older than maxAge.
func (s *Store) Latest(dataset string, k Key, at time.Time, maxAge time.Duration) (Record, bool) {
	if maxAge < 0 {
		return Record{}, false
	}
	recs := s.Range(dataset, k, at.UTC().Add(-maxAge), at.UTC(), maxLatestScan)
	if len(recs) == 0 {
		return Record{}, false
	}
	return recs[len(recs)-1], true
}

// maxLatestScan bounds Latest's read: three days of minutes.
const maxLatestScan = 3 * 1440

// Days is a series' rolled-up days from from to to, oldest first, at most max.
func (s *Store) Days(dataset string, k Key, from, to time.Time, max int) []Day {
	dir, d, ok := s.seriesDir(dataset, k)
	if !ok {
		return nil
	}
	if max <= 0 {
		return nil
	}
	from, to = from.UTC().Truncate(24*time.Hour), to.UTC()
	byDate := map[string]dayEntry{}
	for y := from.Year(); y <= to.Year(); y++ { // the yearly roll-ups, bounded by the span (P10-02)
		if doc, ok := readAs[rollupDoc](s, filesAt(dir, time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)).oldYear, d); ok {
			addDays(byDate, doc.Days)
		}
	}
	parts := rollupParts(dir)
	first := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	for m := first; !m.After(to); m = m.AddDate(0, 1, 0) { // the span's months (P10-02)
		for _, n := range parts[m.Format("2006-01")] { // a month's parts, a broken one passed over (P10-02)
			if doc, ok := readAs[rollupDoc](s, rollupPart(dir, m, n), d); ok {
				addDays(byDate, doc.Days) // a part's day stands over a yearly one
			}
		}
	}
	dates := make([]string, 0, len(byDate))
	for date := range byDate { // the days read (P10-02)
		dates = append(dates, date)
	}
	sort.Strings(dates)
	var out []Day
	for _, ds := range dates { // in date order (P10-02)
		date, err := time.Parse("2006-01-02", ds)
		if err == nil && !date.Before(from) && !date.After(to) && len(out) < max {
			out = append(out, fromDayEntry(k, date, byDate[ds]))
		}
	}
	return out
}

// addDays files days by their date, a later one standing over an earlier.
func addDays(byDate map[string]dayEntry, days []dayEntry) {
	for _, e := range days { // a file's days (P10-02)
		if e.Date != "" {
			byDate[e.Date] = e
		}
	}
}

// Stats is a copy of what the store has done.
func (s *Store) Stats() Stats {
	if s == nil {
		return Stats{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

// note counts what happened, for the diagnostics, and returns result - so a
// failure is one guard: `if bad { return s.note(tally, false) }`.
func (s *Store) note(tally func(*Stats), result bool) bool {
	if s == nil {
		return result
	}
	if tally == nil {
		return result
	}
	s.mu.Lock()
	tally(&s.stats)
	s.mu.Unlock()
	return result
}

// fromEntry is a bucket's entry as its record.
func fromEntry(k Key, h hourEntry) Record {
	r := Record{Key: k, At: h.Hour, IssuedAt: h.Issued, Shape: h.Shape, Values: map[string][]float64{}, Doc: h.Doc}
	if len(h.Values) == 0 {
		return r
	}
	for name, v := range h.Values { // a dataset's fields (P10-02)
		r.Values[name] = decode(v)
	}
	return r
}

// fromDayEntry is a rolled-up day as read.
func fromDayEntry(k Key, date time.Time, e dayEntry) Day {
	d := Day{Key: k, Date: date, Shape: e.Shape, Hours: e.Hours, Min: map[string][]float64{}, Max: map[string][]float64{}, Mean: map[string][]float64{}}
	if len(e.Stats) == 0 {
		return d
	}
	for name, st := range e.Stats { // P10-02
		d.Min[name], d.Max[name], d.Mean[name] = decode(st["min"]), decode(st["max"]), decode(st["mean"])
	}
	return d
}

// dataset is a held dataset, as its retention stands now.
func (s *Store) dataset(name string) (Dataset, bool) {
	if s == nil || name == "" {
		return Dataset{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.sets[name]
	return d, ok
}

// datasets are the held datasets, a copy.
func (s *Store) datasets() []Dataset {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sets) == 0 {
		return nil
	}
	out := make([]Dataset, 0, len(s.sets))
	for _, d := range s.sets { // a handful (P10-02)
		out = append(out, d)
	}
	return out
}

// Retain sets a held dataset's retention - its records' hours and its
// rolled-up days - while the store runs (D-175: the [ Data ] tab), its
// manifest rewritten to say so. False for a dataset not held.
func (s *Store) Retain(name string, hours, days time.Duration) bool {
	if s == nil || hours <= 0 || days < 0 {
		return false
	}
	s.mu.Lock()
	d, ok := s.sets[name]
	if ok {
		if d.Hours != hours || d.Days != days {
			delete(s.bounded, name) // said again only once the bound cuts the new retention short
		}
		d.Hours, d.Days = hours, days
		s.sets[name] = d
	}
	s.mu.Unlock()
	if !ok {
		return false
	}
	s.writeManifest(d)
	return true
}

// Clear removes every record, roll-up, claim and manifest the store holds -
// each entry of its root, never the root itself nor anything beside it (D-177:
// Clear history, behind the app's confirmation) - and writes the manifests
// again, so recording goes on.
func (s *Store) Clear() error {
	if s == nil || s.root == "" {
		return nil
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil // nothing recorded yet
	}
	var errs []error
	for _, e := range entries { // the datasets (P10-02)
		errs = append(errs, os.RemoveAll(filepath.Join(s.root, e.Name())))
	}
	s.mu.Lock()
	s.sizes, s.oldest, s.measured, s.bounded = map[string]int64{}, time.Time{}, s.now(), nil
	s.mu.Unlock()
	for _, d := range s.datasets() { // P10-02
		s.writeManifest(d)
	}
	err = errors.Join(errs...)
	if err != nil {
		s.measure() // what is left
	}
	return err
}

// ForgetDataset removes a held dataset's records and roll-ups, every version
// of them, and writes its manifest again, so recording goes on (D-269: Clear
// map data forgets EPA's UV readings by city): the record files removed. A
// dataset not held is not touched.
func (s *Store) ForgetDataset(name string) (int, error) {
	if s == nil || s.root == "" {
		return 0, nil
	}
	d, held := s.dataset(name)
	if !held || !validSegment(name) {
		return 0, nil
	}
	dir := filepath.Join(s.root, name)
	files := 0
	_ = filepath.WalkDir(dir, func(_ string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() && e.Name() != "manifest.json.gz" && e.Name()[0] != '.' {
			files++
		}
		return nil
	})
	err := os.RemoveAll(dir)
	s.mu.Lock()
	delete(s.bounded, name)
	s.mu.Unlock()
	s.writeManifest(d)
	s.measure()
	return files, err
}
