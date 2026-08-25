package tz

import (
	"testing"
	"time"
)

func TestLoadMemoizesZonesAndErrors(t *testing.T) {
	a, err := Location("America/Los_Angeles")
	if err != nil || a == nil {
		t.Fatal(err)
	}
	b, _ := Location("America/Los_Angeles")
	if a != b {
		t.Fatal("the same zone must be the same *Location (memoized)")
	}
	if _, err := Location("Nowhere/Invalid"); err == nil {
		t.Fatal("bad names error")
	}
	if v, ok := cache.Load("Nowhere/Invalid"); !ok || v.(result).err == nil {
		t.Fatal("errors are memoized too")
	}
	if _, err := Location(""); err == nil {
		t.Fatal("empty name is refused (callers pick their fallback)")
	}
	if got := time.Date(2026, 8, 24, 12, 0, 0, 0, a).Format("MST"); got != "PDT" {
		t.Fatalf("loaded zone must be real: %s", got)
	}
}
