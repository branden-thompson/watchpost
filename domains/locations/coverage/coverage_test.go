package coverage

import (
	"strings"
	"testing"
)

func TestNWSCoversTheStatesAndTerritories(t *testing.T) {
	for _, cc := range []string{"US", "PR", "VI", "GU", "AS", "MP"} {
		if !NWS(cc) {
			t.Fatalf("%s is NWS territory", cc)
		}
	}
	for _, cc := range []string{"FR", "CA", "MX", "GB", ""} {
		if NWS(cc) {
			t.Fatalf("%s is not", cc)
		}
	}
	if msg := Outside("Paris, FR"); !strings.HasPrefix(msg, "Paris, FR is outside NWS coverage") || !strings.Contains(msg, "1.0") {
		t.Fatalf("the refusal says what, why and what is coming: %q", msg)
	}
}
