package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
)

// TestTheSeenStoreWritesOnlyWhatChanged is 0.19.5 Q4.6 (PF's minor list): the
// seen store is saved every two minutes, and rewrites its file only when its
// ids have changed since the last write. The first save of a session always
// writes, since that save is the one that tightens the file's mode (R3-D-01).
func TestTheSeenStoreWritesOnlyWhatChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seen.json")
	now := time.Now()
	s := loadSeen(dir, time.Hour)
	s.mark([]globalfeed.Event{{ID: "us7000abcd"}}, now)
	s.save()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the first save wrote nothing: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil { // as a store an older build left behind
		t.Fatal(err)
	}
	reloaded := loadSeen(dir, time.Hour)
	reloaded.save()
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("the first save of a session did not write the store at 0600: %v %v", info, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	reloaded.mark([]globalfeed.Event{{ID: "us7000abcd"}}, now) // already seen: nothing changes
	reloaded.save()
	if _, err := os.Stat(path); err == nil {
		t.Error("an unchanged store was written again")
	}
	reloaded.mark([]globalfeed.Event{{ID: "us7000efgh"}}, now)
	reloaded.save()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("a newly seen id was not written: %v", err)
	}
}
