package coops

import (
	"testing"
	"time"
)

// Quality pass Q5 (Q5b-5): predictions are astronomical and the request
// window is keyed by the UTC date — cached to UTC midnight, not for an hour.
func TestPredictionsAreCachedToUTCMidnight(t *testing.T) {
	at := time.Date(2026, 8, 26, 20, 30, 0, 0, time.UTC)
	if got := untilUTCMidnight(at); got != 3*time.Hour+30*time.Minute {
		t.Fatalf("20:30Z caches for 3h30m, got %v", got)
	}
	if got := untilUTCMidnight(time.Date(2026, 8, 26, 23, 59, 50, 0, time.UTC)); got != time.Minute {
		t.Fatalf("just before midnight: the one-minute floor, got %v", got)
	}
	if got := untilUTCMidnight(time.Date(2026, 8, 26, 16, 0, 0, 0, time.FixedZone("PDT", -7*3600))); got != time.Hour {
		t.Fatalf("a local clock is read in UTC (16:00 PDT = 23:00Z): got %v", got)
	}
}
