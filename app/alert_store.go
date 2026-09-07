package app

// alert_store.go — the producer's record of what an alert IS (MVS-D-77, T3.10b).
//
// THE CARD STAYS DOMAIN-FREE (DR-1). A takeover card carries the alert ids it
// reads, in read order, and nothing else about them: the Director owns the
// ORDER because it planned it, and whoever proposed the card owns what each id
// means. This is that half — the same shape the `[space]` read already uses,
// where the reader keeps its own row rather than putting the row on the card.
//
// It is written by the producer as it sends arrivals and read by the executors
// as they compose, cue and mark. Both run on their own goroutines, so it locks.

import (
	"sort"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
)

// maxAlertRecords bounds the store (P10-03). It is a diagnostic-sized number
// rather than a tuned one: the rail admits five alerts per burst and holds a
// handful of bursts, so two hundred is many cycles of arrivals with room to
// spare.
//
// EVICTION IS OLDEST-FIRST AND CAN, IN PRINCIPLE, DROP A QUEUED CARD'S RECORD —
// it would take two hundred newer alerts between a burst being queued and being
// read, which is an outbreak far past anything the rail would still be holding.
// The failure is safe either way: the executor declines the build by name and
// reports it, rather than reading a burst short.
const maxAlertRecords = 200

// alertStore is the producer's records, by the producer's own ids.
type alertStore struct {
	mu   sync.Mutex
	at   map[string]time.Time
	byID map[string]globalfeed.Event
}

func newAlertStore() *alertStore {
	return &alertStore{at: map[string]time.Time{}, byID: map[string]globalfeed.Event{}}
}

// note records the events of one burst, so the executors can ask what its
// card's refs mean. Re-noting an event refreshes it rather than duplicating it:
// the same alert can arrive on several cycles before it is read.
func (s *alertStore) note(evs []globalfeed.Event, now time.Time) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range evs { // bounded by the cycle's arrivals (P10-02)
		s.byID[e.ID] = e
		s.at[e.ID] = now
	}
	s.capOldest()
}

// get is the record for one id, if the store still holds it.
func (s *alertStore) get(id string) (globalfeed.Event, bool) {
	if s == nil {
		return globalfeed.Event{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byID[id]
	return e, ok
}

// capOldest keeps the newest maxAlertRecords entries. Called under the lock.
//
// SORT AND TRIM, the shape seenStore.capOldest already uses — one pass rather
// than a condition-only loop that evicts one entry per turn. The loop form was
// bounded in fact and not in shape, which is what P10-02 is about: a bound the
// reader has to derive is a bound the next edit can remove without noticing.
func (s *alertStore) capOldest() {
	if len(s.byID) <= maxAlertRecords {
		return
	}
	type kv struct {
		id string
		at time.Time
	}
	all := make([]kv, 0, len(s.byID))
	for id, at := range s.at { // bounded by the store (P10-02)
		all = append(all, kv{id, at})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].at.After(all[j].at) })
	for _, e := range all[maxAlertRecords:] { // bounded by the overage (P10-02)
		delete(s.byID, e.id)
		delete(s.at, e.id)
	}
}
