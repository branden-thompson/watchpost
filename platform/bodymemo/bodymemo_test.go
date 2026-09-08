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
	m.Parsed("a", []byte("a"), upper)
	m.Parsed("b", []byte("b"), upper)
	m.Parsed("a", []byte("a"), upper) // a is now the most recently used
	m.Parsed("c", []byte("c"), upper) // evicts b
	if n, _ := m.Stats(); n != 2 {
		t.Fatalf("the memo holds at most 2, got %d", n)
	}
	before := parsesOf(m)
	m.Parsed("a", []byte("a"), upper)
	if parsesOf(m) != before {
		t.Error("a was evicted; the eviction took the most recently used")
	}
	m.Parsed("b", []byte("b"), upper)
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
