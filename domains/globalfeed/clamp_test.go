package globalfeed

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestAnIDIsBoundedInRunes is 0.19.5 Q5.6's pin (DR-8), written before the id
// bound uses the platform's clamp: an id is cut at maxIDRunes runes, never
// mid-rune, and a short one is left as it is.
func TestAnIDIsBoundedInRunes(t *testing.T) {
	long := strings.Repeat("é", maxIDRunes+40)
	if got := clampID(long); utf8.RuneCountInString(got) != maxIDRunes || !utf8.ValidString(got) {
		t.Errorf("a %d-rune id is cut to %d runes (valid %v); want %d", maxIDRunes+40, utf8.RuneCountInString(got), utf8.ValidString(got), maxIDRunes)
	}
	if got := clampID("us7000abcd"); got != "us7000abcd" {
		t.Errorf("a short id became %q", got)
	}
}
