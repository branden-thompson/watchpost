package app

import (
	"math"
	"testing"
)

// TestAMissingRowIsAllMissing is 0.19.5 Q5.6 (DR-12): the history's and the
// temperature map's one helper for an hour with no readings. Every value is
// NaN, the mark for missing, never 0, which would draw as a real 0 °C.
func TestAMissingRowIsAllMissing(t *testing.T) {
	row := missingRow(6)
	if len(row) != 6 {
		t.Fatalf("missingRow(6) has %d values", len(row))
	}
	for i, v := range row {
		if !math.IsNaN(v) {
			t.Errorf("value %d of a missing row is %v; want NaN", i, v)
		}
	}
}
