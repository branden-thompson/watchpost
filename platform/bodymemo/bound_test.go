package bodymemo

import (
	"fmt"
	"testing"
)

func parseCopy(b []byte) (string, error) { return string(b), nil }

// THE MEMO IS BOUNDED, AND THE BOUND IS NOW DRIVEN (OQ-9 rule 3).
//
// THE PACKAGE DOC STATES IT — "It is bounded. At most max entries,
// least-recently-used out" — and nothing exercised it. An invariant was added to
// `Parsed` to assert it, and a mutant proved the invariant could be DELETED with
// every test still green: the bound was written down twice and checked nowhere.
//
// WHY IT MATTERS THAT NOTHING ELSE CATCHES IT. Every observable this package has
// stays correct while the memo grows without limit: a hit still returns what a
// parse would, and `parses` still counts only misses. An unbounded cache in a
// long-running broadcast app is a slow leak with no symptom until the machine is
// out of memory — and a full machine corrupts every measurement taken on it.
func TestTheMemoNeverHoldsMoreThanItsCap(t *testing.T) {
	m := New[string, string](4)
	for i := range 40 {
		k := fmt.Sprintf("k%d", i)
		if _, err := m.Parsed(k, []byte(k), parseCopy); err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if n, _ := m.Stats(); n > 4 {
			t.Fatalf("after %d distinct keys the memo holds %d, cap is 4", i+1, n)
		}
	}
	if n, _ := m.Stats(); n != 4 {
		t.Errorf("the memo settled at %d entries, want its cap of 4", n)
	}
}

// AND AN EVICTION WITH NOTHING TO EVICT DOES NOT DELETE THE ZERO KEY.
//
// With an empty map the walk finds no victim, and `delete` on the zero value of K
// is a silent no-op — so an empty memo would "evict" for ever and the caller's
// bound would never be reached.
func TestAnEmptyMemoHasNothingToEvict(t *testing.T) {
	m := New[string, string](1)
	m.evictLocked() // the caller would hold mu; nothing is running concurrently here
	if n, _ := m.Stats(); n != 0 {
		t.Fatalf("an empty memo reports %d entries", n)
	}
	// AND IT STILL WORKS AFTERWARDS: the zero key was not left in the map.
	if _, err := m.Parsed("a", []byte("a"), parseCopy); err != nil {
		t.Fatalf("the memo stopped accepting entries: %v", err)
	}
	if n, _ := m.Stats(); n != 1 {
		t.Errorf("after one parse the memo holds %d, want 1", n)
	}
}

// AND A CAP BELOW ONE IS THE CALLER'S MISTAKE, CLAMPED BUT NOT HIDDEN.
//
// It clamps to 1 — returning nil would move a start-up mistake into a nil
// dereference at the first read — and the violation is named rather than
// absorbed, so the bound check below it is not measuring its own clamp.
func TestACapBelowOneStillGivesAUsableMemo(t *testing.T) {
	m := New[string, string](0)
	if _, err := m.Parsed("a", []byte("a"), parseCopy); err != nil {
		t.Fatalf("a clamped memo must still work: %v", err)
	}
	if n, _ := m.Stats(); n != 1 {
		t.Errorf("the clamped memo holds %d, want 1", n)
	}
}
