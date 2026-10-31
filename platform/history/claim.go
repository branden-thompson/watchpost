package history

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

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
		if errors.Is(err, fs.ErrNotExist) {
			_ = os.MkdirAll(filepath.Dir(path), 0o700) // a prune removed it as empty meanwhile
			continue
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
