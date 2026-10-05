package uv

// fuzz_test.go — EPA's answer is read from the network, so its reader is
// fuzzed from the recorded answer: no panic; a forecast read is some hours,
// each index a UV index - zero to maxIndex - and every hour within a day of
// the first.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func FuzzEPAHourly(f *testing.F) {
	b, err := os.ReadFile(filepath.Join("testdata", "epa-vista.json"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(b)
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		got, err := NewEPA(&bodyGet{body: string(body)}, "").Hourly(context.Background(), "Vista", "CA", la)
		if err != nil {
			return
		}
		if len(got) == 0 {
			t.Fatal("no hour and no error")
		}
		for _, r := range got {
			if r.Index < 0 || r.Index > maxIndex {
				t.Fatalf("a UV index of %v kept", r.Index)
			}
			if d := r.At.Sub(got[0].At); d >= 24*time.Hour || d <= -24*time.Hour {
				t.Fatalf("the hour %v is a day or more from the first, %v", r.At, got[0].At)
			}
			if v, ok := At(got, r.At); !ok || v < 0 || v > maxIndex {
				t.Fatalf("the reading at %v is %v (%v)", r.At, v, ok)
			}
		}
	})
}
