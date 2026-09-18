package snapshot

import (
	"strings"
	"testing"
)

func TestReplaceKeysRewritesEveryPairAndNothingElse(t *testing.T) {
	line := "director:fx:tune(33.2887,-117.2253) cards=[read:33.1959,-117.3795:ADMITTED] spoken=1.5s at 12.5000 mi"
	got := ReplaceKeys(line, func(k LocationKey) string { return "<" + string(k) + ">" })
	if !strings.Contains(got, "tune(<33.2887,-117.2253>)") || !strings.Contains(got, "read:<33.1959,-117.3795>:ADMITTED") {
		t.Errorf("a pair was not rewritten: %q", got)
	}
	if !strings.Contains(got, "spoken=1.5s at 12.5000 mi") {
		t.Errorf("a number that is not a pair was touched: %q", got)
	}
	if HasKey(ReplaceKeys(line, Opaque)) {
		t.Errorf("an Opaque rewrite still carries a pair: %q", ReplaceKeys(line, Opaque))
	}
	// THE BOUND IS WIDER THAN KEY'S OWN SPELLING: a pair that reaches a writer
	// through free text — `%.6f,%.6f`, the `%v` of a LocationRef — is a pair.
	for _, free := range []string{"33.288700,-117.225300", "{Bonsall, CA  92003 33.2887 -117.2253  0}", "at 33.29, -117.23 now"} { // bounded by the probes (P10-02)
		if !HasKey(free) || HasKey(ReplaceKeys(free, Opaque)) {
			t.Errorf("a pair in free text escaped the bound: %q -> %q", free, ReplaceKeys(free, Opaque))
		}
	}
	for _, plain := range []string{"spoken=1.5s", "version 1.27.0", "12.5 mi at 14:30", "run=3 of 380"} { // bounded by the probes (P10-02)
		if HasKey(plain) {
			t.Errorf("text with no pair was taken for one: %q", plain)
		}
	}
}

func TestOpaqueIsStableDistinctAndNotAPosition(t *testing.T) {
	a, b := Opaque("33.2887,-117.2253"), Opaque("33.2888,-117.2253")
	if a != Opaque("33.2887,-117.2253") {
		t.Error("the same key named twice differently")
	}
	if a == b {
		t.Error("two places one metre apart share a name")
	}
	if HasKey(a) || strings.Contains(a, "33.28") || !strings.HasPrefix(a, "place:") || len(a) != len("place:")+8 {
		t.Errorf("the name is not opaque: %q", a)
	}
}
