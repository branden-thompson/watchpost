package history

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// header is what a document says of itself: its schema, dataset and version.
type header interface{ header() (int, string, int) }

func (doc *dayDoc) header() (int, string, int)    { return doc.Schema, doc.Dataset, doc.Version }
func (doc *rollupDoc) header() (int, string, int) { return doc.Schema, doc.Dataset, doc.Version }

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

// maxDocBytes bounds a document read, decompressed: a roll-up part is held to
// half of it (rollUpBudget).
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

// writeCounted writes v to path and keeps the store's size (PF-2): the file
// written, and the size of the one it replaced, -1 where there was none.
func (s *Store) writeCounted(path string, v any) (fs.FileInfo, int64, error) {
	had := int64(-1)
	if path == "" {
		return nil, had, errors.New("history: no path to write")
	}
	if info, err := os.Stat(path); err == nil {
		had = info.Size()
	}
	written, err := writeGz(path, v)
	if err != nil {
		return nil, had, err
	}
	s.grow(path, written.Size()-max(had, 0))
	return written, had, nil
}

// removeCounted removes a file and keeps the store's size: true when it was
// removed.
func (s *Store) removeCounted(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Lstat(path)
	if err != nil {
		return false // already gone
	}
	if os.Remove(path) != nil {
		return false
	}
	if info.Mode().IsRegular() {
		s.grow(path, -info.Size())
	}
	return true
}

// grow adds delta to the size of the dataset path lies in, once the store has
// been measured: before, the walk counts it.
func (s *Store) grow(path string, delta int64) {
	if s == nil || delta == 0 {
		return
	}
	rel, err := filepath.Rel(s.root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return
	}
	name, _, nested := strings.Cut(filepath.ToSlash(rel), "/")
	if !nested {
		name = "" // a file at the root
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sizes == nil {
		return
	}
	s.sizes[name] += delta
	// ANOTHER INSTANCE MAY HAVE REMOVED FILES THIS ONE COUNTED (several share a
	// store): a size below nothing is held at nothing until the next walk.
	held := s.sizes[name]
	if invariant.Check(held >= 0, "history: a dataset's size below zero") != nil {
		s.sizes[name] = 0
	}
}

// noteDay keeps the oldest day held, once the store has been measured: a
// record put for a day before it.
func (s *Store) noteDay(at time.Time) {
	if s == nil || at.IsZero() {
		return
	}
	day := at.UTC().Truncate(24 * time.Hour)
	s.mu.Lock()
	defer s.mu.Unlock()
	older := s.oldest.IsZero() || day.Before(s.oldest)
	if !s.measured.IsZero() && older {
		s.oldest = day
	}
}

// syncFile makes a written file durable before its rename: the file's own
// Sync (on macOS a full flush to the disk). A test may stand in for it.
var syncFile = (*os.File).Sync

// writeGz writes v compressed to path by temp file and rename, the temp
// file in path's own directory so the rename is atomic: the file written. A
// directory a prune removed as empty meanwhile is made again.
func writeGz(path string, v any) (fs.FileInfo, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(body); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	var f *os.File
	for try := 0; try < 2 && f == nil; try++ { // made again once (P10-02)
		if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		f, err = os.CreateTemp(filepath.Dir(path), ".tmp-*")
	}
	if err != nil {
		return nil, err
	}
	tmp := f.Name()
	_, werr := f.Write(buf.Bytes())
	serr := syncFile(f)
	info, ierr := f.Stat()
	cerr := f.Close()
	if err := errors.Join(werr, serr, ierr, cerr, os.Chmod(tmp, 0o600), os.Rename(tmp, path)); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	return info, nil
}
