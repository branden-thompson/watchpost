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
