package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
)

// The tests in this file are 0.19.5 Q2.3 (RN-3): the two stores' caps keep the
// newest entries. Both are reached here by their cap, not by their window, so a
// cap that kept the oldest, or kept nothing, fails.

// TestTheSeenStoreKeepsTheNewestIDsAtItsCap: the ticker forgets the oldest
// announcements first, so a still-active event is not read again.
func TestTheSeenStoreKeepsTheNewestIDsAtItsCap(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	ids := make(map[string]time.Time, maxSeenIDs+10)
	for i := range maxSeenIDs + 10 { // every id inside the window: only the cap can trim
		ids[fmt.Sprintf("id%d", i)] = now.Add(-time.Duration(i) * time.Second)
	}
	b, err := json.Marshal(ids)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "seen.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	kept := loadSeen(dir, tickerSeenWindow).set()
	if len(kept) != maxSeenIDs {
		t.Fatalf("the store kept %d ids; want its cap, %d", len(kept), maxSeenIDs)
	}
	if !kept["id0"] || !kept[fmt.Sprintf("id%d", maxSeenIDs-1)] {
		t.Error("the cap dropped one of the newest ids")
	}
	if kept[fmt.Sprintf("id%d", maxSeenIDs)] || kept[fmt.Sprintf("id%d", maxSeenIDs+9)] {
		t.Error("the cap kept one of the oldest ids")
	}
}

// TestTheAlertStoreKeepsTheNewestRecordsAtItsCap: the records a queued burst
// will be read from are the newest, so the cap evicts the oldest.
func TestTheAlertStoreKeepsTheNewestRecordsAtItsCap(t *testing.T) {
	s := newAlertStore()
	base := time.Now()
	for i := range maxAlertRecords + 1 {
		s.note([]globalfeed.Event{{ID: fmt.Sprintf("a%d", i)}}, base.Add(time.Duration(i)*time.Second))
	}
	if _, ok := s.get("a0"); ok {
		t.Error("the oldest record survived a store past its cap")
	}
	for _, id := range []string{"a1", fmt.Sprintf("a%d", maxAlertRecords)} {
		if _, ok := s.get(id); !ok {
			t.Errorf("record %s, among the newest %d, was evicted", id, maxAlertRecords)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.byID) != maxAlertRecords || len(s.at) != maxAlertRecords {
		t.Errorf("the store holds %d records and %d times; want %d of each", len(s.byID), len(s.at), maxAlertRecords)
	}
}
