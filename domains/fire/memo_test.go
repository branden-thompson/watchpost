package fire

import (
	"errors"
	"testing"
)

// Quality pass Q3: one parse per body change, errors memoised, bound one.
func TestMemoParsesOncePerBodyChange(t *testing.T) {
	var m Memo[int]
	calls := 0
	parse := func(b []byte) (int, error) {
		calls++
		if len(b) == 0 {
			return 0, errors.New("empty")
		}
		return len(b), nil
	}
	if v, err := m.Get([]byte("abc"), parse); v != 3 || err != nil || calls != 1 {
		t.Fatalf("first body parses once: v=%d err=%v calls=%d", v, err, calls)
	}
	if v, _ := m.Get([]byte("abc"), parse); v != 3 || calls != 1 {
		t.Fatalf("the same bytes are not re-parsed: calls=%d", calls)
	}
	if v, _ := m.Get([]byte("abcd"), parse); v != 4 || calls != 2 {
		t.Fatalf("changed bytes re-parse: v=%d calls=%d", v, calls)
	}
	if _, err := m.Get(nil, parse); err == nil || calls != 3 {
		t.Fatal("a parse error comes back")
	}
	if _, err := m.Get(nil, parse); err == nil || calls != 3 {
		t.Fatal("a parse error is memoised: the same bad body is not re-parsed")
	}
	if v, ok := m.Peek(); !ok || v != 0 {
		t.Fatalf("Peek returns the last value: %d %v", v, ok)
	}
	if m.Parses() != 3 {
		t.Fatalf("parses counted: %d", m.Parses())
	}
	var fresh Memo[int]
	if _, ok := fresh.Peek(); ok {
		t.Fatal("nothing memoised before the first Get")
	}
}
