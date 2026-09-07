package app

// seen_store.go — the ids the ticker has already announced.
//
// SPLIT OUT OF ticker.go (2026-09-06). It was one of five separable concerns in
// a 978-line file with no header comment at all: the Producer, the words a
// burst says, the marquee's rows, this store and the metro tie. A file that
// holds five things cannot say what it holds, and app/ticker.go is the file the
// role model (MVS-D-77, S-7) calls the PRODUCER — a role that explicitly does
// not own words or pacing.
//
// A PURE MOVE. Nothing here changed; TestDeclarationSetUnchanged is the guard
// that says so.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
)

// tickerSeenWindow is how long a seen event id is remembered across restarts so
// it is not re-announced (the 7-day USGS window; HUM LEAD).
const tickerSeenWindow = 7 * 24 * time.Hour

// maxSeenIDs bounds the seen-store on load (P10-03): 7 days × a few hundred
// ids/day, with room. Past it the OLDEST entries are dropped.
const maxSeenIDs = 20_000

// seenStore persists the ids the ticker has already announced (id -> first
// seen), pruned to a window, so a restart does not re-announce a still-active
// event. Bounded by the window and the feeds' own sizes (P10).
type seenStore struct {
	mu     sync.Mutex
	path   string
	window time.Duration
	ids    map[string]time.Time
}

func loadSeen(dir string, window time.Duration) *seenStore {
	s := &seenStore{path: filepath.Join(dir, "seen.json"), window: window, ids: map[string]time.Time{}}
	if b, err := os.ReadFile(s.path); err == nil {
		var raw map[string]time.Time
		if json.Unmarshal(b, &raw) == nil {
			cutoff := time.Now().Add(-window)
			for id, at := range raw {
				if at.After(cutoff) { // drop the stale on load
					s.ids[id] = at
				}
			}
			s.capOldest()
		}
	}
	return s
}

// empty reports whether nothing has ever been announced — a genuine first run,
// or a store whose every entry has aged past the window.
func (s *seenStore) empty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.ids) == 0
}

// capOldest keeps the newest maxSeenIDs entries (NFR-13).
func (s *seenStore) capOldest() {
	if len(s.ids) <= maxSeenIDs {
		return
	}
	type kv struct {
		id string
		at time.Time
	}
	all := make([]kv, 0, len(s.ids))
	for id, at := range s.ids {
		all = append(all, kv{id, at})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].at.After(all[j].at) })
	s.ids = make(map[string]time.Time, maxSeenIDs)
	for _, e := range all[:maxSeenIDs] {
		s.ids[e.id] = e.at
	}
}

// set is a snapshot of the seen ids as a presence set (for Merge).
func (s *seenStore) set() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.ids))
	for id := range s.ids {
		out[id] = true
	}
	return out
}

// has reports whether one id has already been announced. The presence question
// for ONE id, rather than set()'s whole-map copy — this is asked once per ref
// while a card is composed (I-7).
func (s *seenStore) has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.ids[id]
	return ok
}

// mark records events as seen at t and prunes the stale.
func (s *seenStore) mark(evs []globalfeed.Event, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range evs {
		if _, ok := s.ids[e.ID]; !ok {
			s.ids[e.ID] = t
		}
	}
	cutoff := t.Add(-s.window)
	for id, at := range s.ids {
		if at.Before(cutoff) {
			delete(s.ids, id)
		}
	}
	s.capOldest() // the bound holds as the store GROWS, not only when it is read
}

func (s *seenStore) save() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, err := json.Marshal(s.ids); err == nil {
		_ = os.MkdirAll(filepath.Dir(s.path), 0o700) // private, like the config store (NFR-13)
		_ = os.WriteFile(s.path, b, 0o600)
		// WriteFile applies the mode only on create and MkdirAll never re-modes:
		// a store left by 0.12.0 at 0644/0755 is tightened here (R3-D-01).
		_ = os.Chmod(s.path, 0o600)
		_ = os.Chmod(filepath.Dir(s.path), 0o700)
	}
}
