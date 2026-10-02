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
// past the hours' retention the day is rolled up into rollup/<YYYY>.json.gz.
// A year of a series is 365 day files, not thousands, and `zcat` shows any
// of it. Every read merges a day's file with its buckets not yet compacted.
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
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"math"
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
// days, rolled up, past that (Days; zero keeps no roll-up). Roll-ups are of
// the numeric fields; a record's document is kept, not rolled up.
type Dataset struct {
	Name, Title, Description string // what it is, for a reader that did not register it (an Analyst's catalog)
	Version                  int
	Step                     time.Duration
	Fields                   []Field
	Hours                    time.Duration
	Days                     time.Duration
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
	// days keeps each day's hours as read, by the day's file, with what its
	// files were when read: read again only when they change (W14 P-18).
	days *agememo.Memo[string, heldDay]
}

// heldDay is a day's hours as read, and its files when they were read.
type heldDay struct {
	files   dayFiles
	entries []hourEntry
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
// decide, not the age - for the series and days a map replays.
var dayRules = agememo.Options{Fresh: 24 * time.Hour, Max: 256}

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
	s.days = agememo.New[string, heldDay](agememo.Options{Fresh: dayRules.Fresh, Max: dayRules.Max, Now: s.now})
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
	_ = writeGz(filepath.Join(vdir, "manifest.json.gz"), manifest{Schema: schema, Dataset: d})
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
// days' bucket directories - or a roll-up directory's years as their first
// days.
func dayNames(dir, month string) []time.Time {
	if dir == "" || month == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []time.Time
	for _, e := range entries { // a month's days, or the years (P10-02)
		name := trimExt(e.Name())
		if t, err := time.Parse("2006-01/02", month+"/"+name); err == nil && name[0] != '.' {
			out = append(out, t)
		} else if y, err := strconv.Atoi(name); err == nil && month == "rollup" {
			out = append(out, time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC))
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
// compacted file, its directory of buckets, the bucket's own file, and the
// year's roll-ups.
type seriesFiles struct{ day, buckets, bucket, year string }

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
		year:    filepath.Join(dir, "rollup", strconv.Itoa(at.Year())+".json.gz"),
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
	Hour   time.Time             `json:"at"`
	Issued time.Time             `json:"issued"`
	Shape  Shape                 `json:"shape"`
	Values map[string][]*float64 `json:"values,omitempty"`
	Doc    json.RawMessage       `json:"doc,omitempty"`
}

// yearDoc is a year's roll-ups on disk.
type yearDoc struct {
	Schema  int        `json:"schema"`
	Dataset string     `json:"dataset"`
	Version int        `json:"version"`
	Key     Key        `json:"key"`
	Days    []dayEntry `json:"days"`
}

type dayEntry struct {
	Date  string                           `json:"date"`
	Shape Shape                            `json:"shape"`
	Hours int                              `json:"hours"`
	Stats map[string]map[string][]*float64 `json:"stats"` // field -> min|max|mean -> values
}

// Put records one bucket of a series, in the bucket's own file. It is skipped
// when an equal or newer issue of that bucket is already recorded. False
// when it could not be written.
func (s *Store) Put(dataset string, r Record) bool {
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
	have, seen := findHour(s.dayEntries(dir, d, at), at)
	stale := seen && !have.Issued.Before(entry.Issued)
	if stale {
		return s.note(func(st *Stats) { st.Skipped++ }, true)
	}
	doc := dayDoc{Schema: schema, Dataset: d.Name, Version: d.Version, Key: r.Key, Date: at.Format("2006-01-02"), Hours: []hourEntry{entry}}
	err := writeGz(filesAt(dir, at).bucket, doc)
	if err != nil {
		return s.note(failed, false)
	}
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
	e := hourEntry{Hour: hour, Issued: r.IssuedAt.UTC(), Shape: r.Shape, Values: map[string][]*float64{}, Doc: r.Doc}
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

// maxDayBuckets bounds a day's bucket files read: a minute's step fills 1,440.
const maxDayBuckets = 1440

func (s *Store) dayEntries(dir string, d Dataset, day time.Time) []hourEntry {
	if dir == "" {
		return nil
	}
	f := filesAt(dir, day)
	if f.day == "" {
		return nil
	}
	entries, _ := os.ReadDir(f.buckets)
	// WHAT THE DAY'S FILES ARE NOW: its document's time and size, its bucket
	// directory's time and count, its buckets' newest time and bytes.
	var now dayFiles
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
	if s.days == nil {
		return s.readDay(f, d, day, entries)
	}
	// THE FILES DECIDE, NOT THE AGE: what was read stands while they do. (Last,
	// Forget and Do, not Get and Put: P10's call graph matches a call by its
	// bare name, and this store's own Get and Put reach here.)
	if held, _, ok := s.days.Last(f.day); ok && held.files == now {
		return held.entries
	}
	s.days.Forget(f.day)
	held, _ := s.days.Do(context.Background(), f.day, func() (heldDay, error) {
		return heldDay{files: now, entries: s.readDay(f, d, day, entries)}, nil
	})
	return held.entries
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
	var out []Day
	for y := from.Year(); y <= to.Year() && len(out) < max; y++ { // bounded by the span (P10-02)
		doc, ok := readAs[yearDoc](s, filesAt(dir, time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)).year, d)
		if !ok {
			continue
		}
		for _, e := range doc.Days { // at most 366 (P10-02)
			date, err := time.Parse("2006-01-02", e.Date)
			if err == nil && !date.Before(from) && !date.After(to) && len(out) < max {
				out = append(out, fromDayEntry(k, date, e))
			}
		}
	}
	return out
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

// header is what a document says of itself: its schema, dataset and version.
type header interface{ header() (int, string, int) }

func (doc *dayDoc) header() (int, string, int)  { return doc.Schema, doc.Dataset, doc.Version }
func (doc *yearDoc) header() (int, string, int) { return doc.Schema, doc.Dataset, doc.Version }

// readAs is a document of d's - a day's or a year's - or false where it is
// absent, does not parse, or is another schema or version, each but absence
// counted.
func readAs[T any, P interface {
	*T
	header
}](s *Store, path string, d Dataset) (T, bool) {
	var doc T
	if !s.readDoc(path, &doc) {
		return doc, false
	}
	sch, name, version := P(&doc).header()
	if sch != schema || name != d.Name || version != d.Version {
		var zero T
		return zero, s.note(func(st *Stats) { st.VersionMismatch++ }, false)
	}
	return doc, true
}

// maxDocBytes bounds a document read: a year's roll-ups of a large lattice
// with room to spare.
const maxDocBytes = 32 << 20

// readDoc reads a compressed JSON document into v: false where it is absent
// (not a fault), too large, or does not decompress or parse (each counted).
func (s *Store) readDoc(path string, v any) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false // absent: not a fault
	}
	if info.Size() > maxDocBytes {
		s.note(func(st *Stats) { st.Corrupt++ }, true)
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false // removed under us
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		s.note(func(st *Stats) { st.Corrupt++ }, true)
		return false
	}
	body, err := io.ReadAll(io.LimitReader(zr, maxDocBytes))
	if err != nil || json.Unmarshal(body, v) != nil {
		s.note(func(st *Stats) { st.Corrupt++ }, true)
		return false
	}
	return true
}

// writeGz writes v compressed to path by temp file and rename, the temp
// file in path's own directory so the rename is atomic.
func writeGz(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(body); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(buf.Bytes())
	serr := f.Sync()
	cerr := f.Close()
	if err := errors.Join(werr, serr, cerr, os.Chmod(tmp, 0o600), os.Rename(tmp, path)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// maxDecimals bounds a field's kept decimals: past it, rounding is noise.
const maxDecimals = 9

// encode is values as kept on disk: each rounded to decimals, a missing one
// null.
func encode(v []float64, decimals int) []*float64 {
	if len(v) == 0 {
		return nil
	}
	if err := invariant.Check(decimals >= 0 && decimals <= maxDecimals, "a field's decimals are 0 to 9"); err != nil {
		return nil
	}
	scale := math.Pow(10, float64(decimals))
	out := make([]*float64, len(v))
	for i, x := range v { // bounded by the shape (P10-02)
		if math.IsNaN(x) || math.IsInf(x, 0) {
			continue // missing: null on disk
		}
		r := math.Round(x*scale) / scale
		out[i] = &r
	}
	return out
}

// decode is values as read from disk: a null is NaN.
func decode(v []*float64) []float64 {
	if len(v) == 0 {
		return nil
	}
	out := make([]float64, len(v))
	for i, p := range v { // P10-02
		out[i] = math.NaN()
		if p != nil {
			out[i] = *p
		}
	}
	return out
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

// claimStale is how old a claim may be with nothing recorded before another
// instance may take it over: its claimant crashed or closed.
const claimStale = 10 * time.Minute

// Claim takes a series' bucket for this instance to fetch and record: true
// when it is this instance's to do - no other holds it and it is not already
// recorded, a record issued before its bucket began not counting. Of several
// instances, one wins (claim).
func (s *Store) Claim(dataset string, k Key, at time.Time) bool {
	dir, d, ok := s.seriesDir(dataset, k)
	if !ok {
		return false
	}
	at = at.UTC().Truncate(d.step())
	if had, done := s.Get(dataset, k, at); done && !had.IssuedAt.Before(at) {
		return false // recorded in its own time; a record kept ahead of it is not (D-188)
	}
	return s.claim(filepath.Join(filepath.Dir(filesAt(dir, at).buckets), ".claim-"+at.Format("02T1504")))
}

// claim creates path exclusively, holding this store's time: true when this
// instance holds it. A claim whose time is claimStale old is its claimant's
// no longer - it crashed or closed - and is taken over.
func (s *Store) claim(path string) bool {
	if path == "" {
		return false
	}
	err := os.MkdirAll(filepath.Dir(path), 0o700)
	if err != nil {
		return false
	}
	now := s.now().UTC()
	for try := 0; try < 2; try++ { // a stale claim is removed once, then claimed (P10-02)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, werr := f.WriteString(now.Format(time.RFC3339Nano))
			return f.Close() == nil && werr == nil
		}
		stale := s.claimStale(path, now)
		if !stale {
			return false // another instance holds it
		}
		_ = os.Remove(path)
	}
	return false
}

// claimStale reports whether a claim's time is claimStale before now: a
// claim that cannot be read is judged by its file's age on the real clock.
func (s *Store) claimStale(path string, now time.Time) bool {
	if path == "" {
		return false
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return false // gone, or not ours to judge: try again next time
	}
	at, err := time.Parse(time.RFC3339Nano, string(body))
	if err == nil {
		return now.Sub(at) > claimStale
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) > claimStale
}

// maxPruneVisits and maxPruneRemovals bound one pass of Prune: what is left
// waits for the next.
const (
	maxPruneVisits   = 256
	maxPruneRemovals = 64
)

// RollUpAndPrune keeps each series to its dataset's retention: a day past
// Hours is rolled up into its year (when the dataset keeps Days) before its
// hours are removed; a year past Days is removed; stale temp files and
// claims are swept. Bounded; any instance may run it; it tolerates what
// another has already removed.
func (s *Store) RollUpAndPrune() {
	if s.root == "" {
		return
	}
	now := s.now().UTC()
	visits, removals := 0, 0
	for _, d := range s.datasets() { // a handful (P10-02)
		vdir := filepath.Join(s.root, d.Name, "v"+strconv.Itoa(d.Version))
		for _, series := range listSeries(vdir, &visits) { // bounded by maxPruneVisits
			removals += s.pruneSeries(d, series, now, &visits, maxPruneRemovals-removals)
			if visits >= maxPruneVisits || removals >= maxPruneRemovals {
				return
			}
		}
	}
}

// listSeries is a dataset version's series directories, source then place.
func listSeries(vdir string, visits *int) []string {
	if vdir == "" || visits == nil {
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
// days past retention; the number of files removed, at most budget.
func (s *Store) pruneSeries(d Dataset, dir string, now time.Time, visits *int, budget int) int {
	if dir == "" || visits == nil {
		return 0
	}
	if budget <= 0 {
		return 0
	}
	removed := 0
	hourCut := now.Add(-d.Hours).Truncate(24 * time.Hour)
	for _, month := range readDirs(dir, visits) { // P10-02
		if month == "rollup" {
			removed += s.pruneYears(d, filepath.Join(dir, month), now, budget-removed)
			continue
		}
		entries, err := os.ReadDir(filepath.Join(dir, month))
		if err != nil {
			continue
		}
		for _, e := range entries { // a month's days and their buckets (P10-02)
			if removed >= budget {
				return removed
			}
			removed += s.pruneEntry(d, dir, month, e, now, hourCut)
		}
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
		_ = os.Remove(path) // a temp file: its writer is long gone
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
	entries := s.dayEntries(dir, d, date)
	if len(entries) == 0 {
		return 0
	}
	doc := dayDoc{Schema: schema, Dataset: d.Name, Version: d.Version, Key: keyOf(dir), Date: date.Format("2006-01-02"), Hours: entries}
	err := writeGz(f.day, doc)
	if err != nil {
		return 0
	}
	written, _ := readAs[dayDoc](s, f.day, d)
	removed := 0
	files, _ := os.ReadDir(f.buckets)
	for i, b := range files { // bounded by maxDayBuckets (P10-02)
		path := filepath.Join(f.buckets, b.Name())
		if i < maxDayBuckets && b.Name()[0] != '.' && s.heldBy(written.Hours, path, d) && os.Remove(path) == nil {
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
	if err := os.Remove(f.day); err == nil {
		removed++
	}
	files, _ := os.ReadDir(f.buckets)
	for i, b := range files { // bounded by maxDayBuckets (P10-02)
		if i < maxDayBuckets && os.Remove(filepath.Join(f.buckets, b.Name())) == nil {
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

// pruneYears removes years of roll-ups wholly past the days' retention.
func (s *Store) pruneYears(d Dataset, dir string, now time.Time, budget int) int {
	if dir == "" || budget <= 0 {
		return 0
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	removed := 0
	for _, e := range entries { // a file a year (P10-02)
		y, err := strconv.Atoi(trimExt(e.Name()))
		if err != nil || removed >= budget {
			continue
		}
		if now.Sub(time.Date(y+1, 1, 1, 0, 0, 0, 0, time.UTC)) > d.Hours+d.Days && os.Remove(filepath.Join(dir, e.Name())) == nil {
			removed++
			s.note(func(st *Stats) { st.Pruned++ }, true)
		}
	}
	return removed
}

// rollUp writes a day's numeric fields - each point's minimum, maximum and
// mean over the day's records - into its year's roll-up, under the series'
// roll-up claim (a year's file is read, merged and replaced, so one writer at
// a time), and reports whether the year now holds that day: only then may
// its records go. A day of documents alone rolls up to nothing and is kept
// until the days' retention passes too.
func (s *Store) rollUp(d Dataset, dir string, date time.Time) bool {
	entries := s.dayEntries(dir, d, date)
	if len(entries) == 0 {
		return false
	}
	entry, ok := dayStats(d, dayDoc{Date: date.Format("2006-01-02"), Hours: entries})
	if !ok {
		return false
	}
	claim := filepath.Join(dir, "rollup", ".claim-"+strconv.Itoa(date.Year()))
	if !s.claim(claim) {
		return false // another instance is rolling up this year: next pass
	}
	defer func() { _ = os.Remove(claim) }() // the claim let go
	yp := filesAt(dir, date).year
	year, _ := readAs[yearDoc](s, yp, d)
	year.Schema, year.Dataset, year.Version, year.Key = schema, d.Name, d.Version, keyOf(dir)
	year.Days = withDay(year.Days, entry)
	if writeGz(yp, year) != nil {
		return false
	}
	if back, ok := readAs[yearDoc](s, yp, d); ok && hasDay(back.Days, entry.Date) {
		s.note(func(st *Stats) { st.RolledUp++ }, true)
		return true
	}
	return false
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
	e := dayEntry{Date: doc.Date, Shape: shape, Hours: len(doc.Hours), Stats: map[string]map[string][]*float64{}}
	for _, f := range d.Fields { // a dataset's few (P10-02)
		lo, hi, sum, count := make([]float64, n), make([]float64, n), make([]float64, n), make([]int, n)
		for _, h := range doc.Hours { // a day's buckets (P10-02)
			if h.Shape != shape {
				return dayEntry{}, false
			}
			for i, p := range h.Values[f.Name] { // bounded by the shape (P10-02)
				if p == nil || i >= n {
					continue
				}
				if count[i] == 0 || *p < lo[i] {
					lo[i] = *p
				}
				if count[i] == 0 || *p > hi[i] {
					hi[i] = *p
				}
				sum[i] += *p
				count[i]++
			}
		}
		e.Stats[f.Name] = map[string][]*float64{"min": statsOf(lo, count, 1, f), "max": statsOf(hi, count, 1, f), "mean": statsOf(sum, count, 0, f)}
	}
	return e, true
}

// statsOf is one statistic's values on disk: missing where no record had a
// value there; a mean is the sum over its count (asIs 0), a bound as it is.
func statsOf(v []float64, count []int, asIs int, f Field) []*float64 {
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

// maxSizeVisits bounds Bytes' walk: far past any store the retention allows.
const maxSizeVisits = 200_000

// Bytes is what the store holds on disk: its files' sizes, walked at most
// maxSizeVisits deep (D-175: the [ Data ] tab says it).
func (s *Store) Bytes() int64 {
	if s == nil || s.root == "" {
		return 0
	}
	var total int64
	visits := 0
	_ = filepath.WalkDir(s.root, func(_ string, e fs.DirEntry, err error) error {
		if err != nil {
			return nil // gone under us, or unreadable: counted as nothing
		}
		visits++
		if visits > maxSizeVisits {
			return fs.SkipAll
		}
		if info, ierr := e.Info(); ierr == nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
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
	for _, d := range s.datasets() { // P10-02
		s.writeManifest(d)
	}
	return errors.Join(errs...)
}
