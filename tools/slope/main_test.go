package main

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// synthetic writes a 72 h soak at 5-minute samples: a plateau (MB) with
// an AR(1) sawtooth of amplitude amp (MB) and a drift of slopeMBPerDay,
// plus spikes every 15 min (the HMS-parse shape) that the per-hour
// minimum must ignore.
func synthetic(t *testing.T, plateau, amp, slopeMBPerDay float64, seed int64) string {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	var b strings.Builder
	b.WriteString("utc,heap_alloc,goroutines\n")
	start := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	ar := 0.0
	for i := range 72 * 12 {
		at := start.Add(time.Duration(i) * 5 * time.Minute)
		days := float64(i) * 5 / (60 * 24)
		ar = 0.8*ar + rng.NormFloat64()*amp*0.6
		v := plateau + slopeMBPerDay*days + ar
		if i%3 == 0 {
			v += 20 // parse transient
		}
		fmt.Fprintf(&b, "%s,%d,%d\n", at.Format(time.RFC3339), int64(v*mb), 280)
	}
	path := filepath.Join(t.TempDir(), "soak.csv")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runSlope(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestQuietFlatHeapPasses(t *testing.T) {
	path := synthetic(t, 50, 0.1, 0, 1)
	code, out, errOut := runSlope(t, "-in", path)
	if code != 0 || !strings.Contains(out, "PASS") {
		t.Fatalf("a flat 50 MB heap with a quiet hourly minimum must pass: code %d\n%s%s", code, out, errOut)
	}
	if !strings.Contains(out, "n=66 buckets") {
		t.Fatalf("72 h at 5 min after a 6 h warm-up is 66 hourly buckets:\n%s", out)
	}
	if !strings.Contains(out, "floor=") || !strings.Contains(out, "bar=2.") {
		t.Fatalf("the report must state the detection floor and the bar (5%% of the plateau):\n%s", out)
	}
}

func TestNoisyFlatHeapIsUncertifiable(t *testing.T) {
	// σ ≈ 1.7 MB over 66 hourly buckets puts the 30-day floor near 20 MB:
	// the run cannot resolve a 2.5 MB bar either way and must say so
	// (red-team R2-1) — never "FAIL" a flat heap, never "PASS" blind.
	path := synthetic(t, 50, 2, 0, 1)
	code, out, _ := runSlope(t, "-in", path)
	if code != 3 || !strings.Contains(out, "UNCERTIFIABLE") {
		t.Fatalf("a noisy flat run must report UNCERTIFIABLE (exit 3): code %d\n%s", code, out)
	}
}

func TestOneMegabytePerDayIsGrowth(t *testing.T) {
	path := synthetic(t, 50, 0.3, 1, 2)
	code, out, _ := runSlope(t, "-in", path)
	if code != 1 || !strings.Contains(out, "GROWTH") {
		t.Fatalf("1 MB/day on a 50 MB plateau projects 30 MB — far above the 2.5 MB bar: code %d\n%s", code, out)
	}
	if !strings.Contains(out, "slope=+") {
		t.Fatalf("the slope must be reported positive:\n%s", out)
	}
}

func TestBucketMinimumIgnoresTransients(t *testing.T) {
	// Two buckets, each with a spike: the series is the minima.
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var s []sample
	for i, v := range []float64{50, 70, 51, 52, 90, 51} {
		s = append(s, sample{at: start.Add(time.Duration(i) * 20 * time.Minute), bytes: v * mb})
	}
	xs, ys := bucketMinima(s, start, time.Hour)
	if len(xs) != 2 || ys[0] != 50 || ys[1] != 51 {
		t.Fatalf("expected the per-hour minima [50 51], got %v", ys)
	}
	if math.Abs(xs[1]-1.0/24) > 1e-9 {
		t.Fatalf("x is days since start, got %v", xs)
	}
}

func TestInsufficientDataIsSaidOutright(t *testing.T) {
	path := filepath.Join(t.TempDir(), "short.csv")
	_ = os.WriteFile(path, []byte("utc,heap_alloc\n2026-01-01T00:00:00Z,1000\n2026-01-01T07:00:00Z,1000\n"), 0o600)
	code, _, errOut := runSlope(t, "-in", path)
	if code != 2 || !strings.Contains(errOut, "need at least 8") {
		t.Fatalf("too few buckets must exit 2 with the reason, got %d %q", code, errOut)
	}
	if code, _, errOut := runSlope(t, "-in", path, "-col", "nope"); code != 2 || !strings.Contains(errOut, "must both exist") {
		t.Fatalf("a missing column must be named, got %d %q", code, errOut)
	}
}

func TestTQuantileTable(t *testing.T) {
	for df, want := range map[int]float64{1: 12.706, 10: 2.228, 64: 1.998, 200: 1.960, 1000: 1.960} {
		if got := tQuantile975(df); math.Abs(got-want) > 0.01 {
			t.Fatalf("t(0.975, %d) = %.3f, want ≈ %.3f", df, got, want)
		}
	}
}

func TestOLSRecoversALine(t *testing.T) {
	xs := []float64{0, 1, 2, 3, 4}
	ys := []float64{1, 3, 5, 7, 9}
	if s, i := ols(xs, ys); math.Abs(s-2) > 1e-12 || math.Abs(i-1) > 1e-12 {
		t.Fatalf("slope %v intercept %v", s, i)
	}
}
