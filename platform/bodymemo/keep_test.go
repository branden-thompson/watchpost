package bodymemo

import (
	"errors"
	"testing"
)

// errSoft is a parse that answers with a value and says what is wrong with
// it - a truncated archive whose points are still good.
var errSoft = errors.New("soft")

// softLen is a parse whose empty body fails and whose short body answers
// softly.
func softLen(raw []byte) (int, error) {
	switch {
	case len(raw) == 0:
		return 0, errors.New("empty")
	case len(raw) < 3:
		return len(raw), errSoft
	}
	return len(raw), nil
}

// A MEMO THAT KEEPS ERRORS answers a body it has parsed with what the parse
// gave, the value and the error together: the same bad body is not parsed
// again for the rest of its cache life (the fire archives' rule), and a soft
// error's value comes back with it.
func TestAMemoThatKeepsErrorsAnswersWithThem(t *testing.T) {
	m := NewKeepingErrors[struct{}, int](1)
	for range 2 {
		if v, err := m.Parsed(struct{}{}, []byte("ab"), softLen); v != 2 || !errors.Is(err, errSoft) {
			t.Fatalf("a soft error's answer: %d %v; want 2 and the error", v, err)
		}
	}
	for range 2 {
		if _, err := m.Parsed(struct{}{}, nil, softLen); err == nil {
			t.Fatal("a failed parse came back without its error")
		}
	}
	if _, parses := m.Stats(); parses != 2 {
		t.Errorf("%d parses for two bodies each asked twice; want each parsed once", parses)
	}
	if v, err := m.Parsed(struct{}{}, []byte("abcd"), softLen); v != 4 || err != nil {
		t.Errorf("a good body after a bad one: %d %v", v, err)
	}
}

// LAST IS THE LAST PARSE KEPT FOR A KEY, whatever body comes next: what a
// caller reads when the body is known unchanged without hashing it, or reads
// what it holds without fetching.
func TestLastIsTheLastParseKept(t *testing.T) {
	m := New[string, string](2)
	if _, ok := m.Last("k"); ok {
		t.Fatal("a key never parsed has a last parse")
	}
	_, _ = m.Parsed("k", []byte("one"), upper)
	_, _ = m.Parsed("k", []byte("bad"), upper) // not kept: this memo keeps no error
	if got, ok := m.Last("k"); !ok || got != "ONE" {
		t.Errorf("the last parse kept is %q (%v); want ONE", got, ok)
	}
	if _, parses := m.Stats(); parses != 1 {
		t.Errorf("Last parsed: %d parses", parses)
	}
}
