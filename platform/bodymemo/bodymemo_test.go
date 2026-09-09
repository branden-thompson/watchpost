package bodymemo

import (
	"errors"
	"strings"
	"testing"
)

// upper is a top-level parse function, which is also what both providers pass:
// a closure that captures anything would allocate on every call, hits included.
func upper(raw []byte) (string, error) {
	if strings.HasPrefix(string(raw), "bad") {
		return "", errors.New("bad body")
	}
	return strings.ToUpper(string(raw)), nil
}

// THE THREE PROPERTIES OQ-9 RULED, one test each.

// 1 — A HIT RETURNS WHAT A PARSE WOULD, and a changed body misses. This is the
// half that makes the memo safe rather than merely fast: it may never answer
// for a body it has not seen.
func TestAChangedBodyMisses(t *testing.T) {
	m := New[string, string](4)
	if got, _ := m.Parsed("k", []byte("one"), upper); got != "ONE" {
		t.Fatalf("first parse: %q", got)
	}
	if got, _ := m.Parsed("k", []byte("two"), upper); got != "TWO" {
		t.Errorf("the same key with a NEW body answered %q — the memo answered for a body it "+
			"had never parsed", got)
	}
	if _, parses := m.Stats(); parses != 2 {
		t.Errorf("a changed body must parse again: %d parses", parses)
	}
}

// 2 — A HIT DOES NOT PARSE.
func TestAnUnchangedBodyDoesNotParseAgain(t *testing.T) {
	m := New[string, string](4)
	for range 5 {
		if got, _ := m.Parsed("k", []byte("one"), upper); got != "ONE" {
			t.Fatalf("hit answered %q", got)
		}
	}
	if _, parses := m.Stats(); parses != 1 {
		t.Errorf("five reads of one body parsed %d times", parses)
	}
}

// 3 — IT IS BOUNDED, and what it drops is the least recently used.
func TestItIsBoundedAndDropsTheLeastRecentlyUsed(t *testing.T) {
	m := New[string, string](2)
	for _, k := range []string{"a", "b", "a", "c"} { // a is used again before c evicts b
		if _, err := m.Parsed(k, []byte(k), upper); err != nil {
			t.Fatal(err)
		}
	}
	if n, _ := m.Stats(); n != 2 {
		t.Fatalf("the memo holds at most 2, got %d", n)
	}
	before := parsesOf(m)
	if _, err := m.Parsed("a", []byte("a"), upper); err != nil {
		t.Fatal(err)
	}
	if parsesOf(m) != before {
		t.Error("a was evicted; the eviction took the most recently used")
	}
	if _, err := m.Parsed("b", []byte("b"), upper); err != nil {
		t.Fatal(err)
	}
	if parsesOf(m) == before {
		t.Error("b survived; nothing was evicted and the bound does not hold")
	}
}

// A FAILED PARSE IS NOT REMEMBERED. Caching an error would make one bad fetch
// permanent for as long as the body did not change.
func TestAFailedParseIsNotStored(t *testing.T) {
	m := New[string, string](4)
	if _, err := m.Parsed("k", []byte("bad body"), upper); err == nil {
		t.Fatal("the fixture parse did not fail")
	}
	if n, _ := m.Stats(); n != 0 {
		t.Errorf("a failed parse left %d entries behind", n)
	}
	if got, err := m.Parsed("k", []byte("good"), upper); err != nil || got != "GOOD" {
		t.Errorf("the key is poisoned after a failure: %q %v", got, err)
	}
}

func parsesOf[K comparable, V any](m *Memo[K, V]) int {
	_, p := m.Stats()
	return p
}

// 3b — AND BOUNDED BY THE CALLER'S OWN LIVENESS RULE, which is the other way a
// memo stays bounded: keys that stop mattering go, whether or not anything else
// arrived to push them out.
func TestPruneDropsWhatTheCallerNoLongerWants(t *testing.T) {
	m := New[string, string](100)
	for _, k := range []string{"a", "b", "c"} {
		if _, err := m.Parsed(k, []byte(k), upper); err != nil {
			t.Fatal(err)
		}
	}
	live := map[string]bool{"b": true}
	m.Prune(func(k string) bool { return live[k] })
	if n, _ := m.Stats(); n != 1 {
		t.Fatalf("one key is live and %d survived", n)
	}
	before := parsesOf(m)
	if _, err := m.Parsed("b", []byte("b"), upper); err != nil {
		t.Fatal(err)
	}
	if parsesOf(m) != before {
		t.Error("the live key was pruned")
	}
	if _, err := m.Parsed("a", []byte("a"), upper); err != nil {
		t.Fatal(err)
	}
	if parsesOf(m) == before {
		t.Error("a dead key survived the prune: the memo is bounded by nothing")
	}
}
