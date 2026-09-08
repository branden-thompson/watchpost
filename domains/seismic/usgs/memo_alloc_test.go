package usgs

// memo_alloc_test.go — B4's instrument, the seismic half.
//
// THE TWIN OF domains/fire/firms/memo_alloc_test.go, and deliberately so: the
// two memos are the pair F-53 names as byte-identical, and the batch that
// consolidates them has to be able to see BOTH sides move. A pin on one of two
// identical implementations measures half a change.
//
// See the firms file for why the pin is here rather than in modes/tty: this
// memo is on the provider FETCH path, which no frame draws, so a frame pin
// returns a green false pass for a change the frame cannot see.

import (
	"testing"
	"time"
)

// ZERO, by the house rule (measurement × 1.05). A hit hashes the body, finds
// the entry and returns the parsed features; nothing about that needs the heap.
const boxMemoHitAllocBudget float64 = 0

func TestBoxMemoHitAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	m := newBoxMemo()
	raw := []byte(`{"type":"FeatureCollection","features":[` +
		quake{id: "m4", mag: 4.2, lat: 34.4, lon: -118.4, depth: 10, ago: time.Hour, typ: "earthquake"}.json() + `]}`)
	const url = "https://example.invalid/box"
	if _, err := m.features(url, raw); err != nil {
		t.Fatal(err)
	}
	// THE FIXTURE MUST HIT. Measured on a miss this pins the DECODER, which is
	// a different thing and is allowed to allocate.
	_, before := m.stats()
	got := testing.AllocsPerRun(50, func() {
		if _, err := m.features(url, raw); err != nil {
			t.Fatal(err)
		}
	})
	if _, after := m.stats(); after != before {
		t.Fatalf("the fixture decoded %d more times: it is missing, not hitting", after-before)
	}
	t.Logf("boxMemo hit: %.0f allocs (budget %.0f)", got, boxMemoHitAllocBudget)
	if got > boxMemoHitAllocBudget {
		t.Errorf("a box-memo HIT allocates %.0f, budget %.0f — the memo exists to make this "+
			"cheaper than the decode it replaces", got, boxMemoHitAllocBudget)
	}
}
