package history

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Since is the oldest day the store holds anything for, and whether it holds
// anything - what a longer retention's cost is measured from (D-231): kept as
// records are put, and walked for when the store is measured, at its first
// read or prune and every remeasureEvery after (PF-2) - so a day pruned
// meanwhile still counts, at most that long.
func (s *Store) Since() (time.Time, bool) {
	if s == nil || s.root == "" {
		return time.Time{}, false
	}
	s.measureOnce()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.oldest, !s.oldest.IsZero()
}

// walkSince is the oldest day a store's root holds anything for: read from
// its month folders' names and their days', walked at most maxSizeVisits deep.
func walkSince(root string) (time.Time, bool) {
	if root == "" {
		return time.Time{}, false
	}
	var oldest time.Time
	visits := 0
	_ = filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		visits++
		if err != nil || visits > maxSizeVisits {
			return fs.SkipAll
		}
		if !e.IsDir() {
			return nil
		}
		month, perr := time.Parse("2006-01", e.Name())
		if perr != nil {
			return nil
		}
		if day, ok := firstDay(path, month); ok && (oldest.IsZero() || day.Before(oldest)) {
			oldest = day
		}
		return fs.SkipDir // a month's days are read by firstDay
	})
	return oldest, !oldest.IsZero()
}

// firstDay is a month folder's earliest day: its day files and folders are
// named by the day of the month.
func firstDay(dir string, month time.Time) (time.Time, bool) {
	if err := invariant.Check(dir != "", "a month folder has a path"); err != nil {
		return time.Time{}, false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}, false
	}
	first := 0
	for _, e := range entries { // a month's days (P10-02)
		dd, err := strconv.Atoi(strings.TrimSuffix(e.Name(), ".json.gz"))
		if err == nil && dd >= 1 && dd <= 31 && (first == 0 || dd < first) {
			first = dd
		}
	}
	if first == 0 {
		return time.Time{}, false
	}
	return month.AddDate(0, 0, first-1), true
}

// BytesOf is what one dataset holds on disk, every version of it (D-231):
// kept as the store writes and removes (PF-2).
func (s *Store) BytesOf(dataset string) int64 {
	if s == nil || s.root == "" || dataset == "" || strings.ContainsAny(dataset, `/\`) || dataset == "." || dataset == ".." {
		return 0
	}
	s.measureOnce()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sizes[dataset]
}

// treeBytes is a folder's files' sizes, walked at most maxSizeVisits deep.
func treeBytes(root string) int64 {
	if err := invariant.Check(root != "", "a tree to measure has a root"); err != nil {
		return 0
	}
	var total int64
	visits := 0
	_ = filepath.WalkDir(root, func(_ string, e fs.DirEntry, err error) error {
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

// maxSizeVisits bounds a walk of the store: far past any store the retention
// allows.
const maxSizeVisits = 200_000

// Bytes is what the store holds on disk (D-175: the [ Data ] tab says it):
// kept as the store writes and removes, so reading it walks nothing (PF-2).
func (s *Store) Bytes() int64 {
	if s == nil || s.root == "" {
		return 0
	}
	s.measureOnce()
	s.mu.Lock()
	defer s.mu.Unlock()
	var total int64
	for _, n := range s.sizes { // the datasets (P10-02)
		total += n
	}
	return total
}

// measureOnce walks the store for its size when it has never been measured.
func (s *Store) measureOnce() {
	s.mu.Lock()
	never := s.measured.IsZero()
	s.mu.Unlock()
	if never {
		s.measure()
	}
}

// measure walks the store for what each dataset holds and the oldest day
// held, and keeps them.
func (s *Store) measure() {
	if s == nil || s.root == "" {
		return
	}
	sizes := map[string]int64{}
	entries, _ := os.ReadDir(s.root)
	for _, e := range entries { // the datasets (P10-02)
		if e.IsDir() {
			sizes[e.Name()] += treeBytes(filepath.Join(s.root, e.Name()))
		} else if info, err := e.Info(); err == nil && info.Mode().IsRegular() {
			sizes[""] += info.Size()
		}
	}
	oldest, recorded := walkSince(s.root)
	if !recorded {
		oldest = time.Time{} // nothing recorded yet: no oldest
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sizes, s.oldest, s.measured = sizes, oldest, now
}
