package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCPUIsTheChangeInCumulativeTime: every ps time shape, and a phase's CPU
// % is its CPU seconds over its wall seconds - never pcpu's decaying average.
func TestCPUIsTheChangeInCumulativeTime(t *testing.T) {
	for in, want := range map[string]float64{"0:01.50": 1.5, "12:03.25": 723.25, "1:02:03": 3723, "2-00:00:01": 172801} {
		if got, ok := cpuSeconds(in); !ok || got != want {
			t.Errorf("cpuSeconds(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "12", "a:b", "1:2:3:4"} {
		if _, ok := cpuSeconds(bad); ok {
			t.Errorf("cpuSeconds(%q) read a time from nothing", bad)
		}
	}
}

// TestQuantilesAreObservedValues: nearest rank, so a p90 is a run that happened.
func TestQuantilesAreObservedValues(t *testing.T) {
	v := []float64{5, 1, 4, 2, 3, 10, 9, 8, 7, 6}
	if quantile(v, 0.5) != 5 || quantile(v, 0.9) != 9 || quantile([]float64{7}, 0.9) != 7 {
		t.Errorf("median %v p90 %v", quantile(v, 0.5), quantile(v, 0.9))
	}
}

// TestARunIsSummarisedByPhase: a run directory in the shape workload.sh
// writes - resources from the samples inside each phase, timings placed in
// the phase they were measured in, deduped across the poller's overlapping
// reads and the end-of-run read of the same process.
func TestARunIsSummarisedByPhase(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("run.txt", "workload-v1 session heavy commit=abc\n")
	write("phases.log", "2026-09-29T10:00:00Z A idle\n2026-09-29T10:10:00Z B radio\n2026-09-29T10:20:00Z quit ok\n")
	// Phases are half-open, [from, to): samples inside each, as soak.sh's
	// 20 s cadence puts them.
	write("samples.csv", "utc,elapsed,rss_kb,footprint_kb,threads,pcpu,heap_alloc,heap_inuse,heap_objects,goroutines,fds,disk_files,disk_bytes,publishes_recent,cpu_time\n"+
		"2026-09-29T10:00:00Z,,,102400,20,,10485760,,,50,,,,,0:10.00\n"+
		"2026-09-29T10:05:00Z,,,204800,24,,20971520,,,60,,,,,0:13.00\n"+ // A: 3 s CPU over 300 s = 1 %
		"2026-09-29T10:10:00Z,,,204800,24,,20971520,,,60,,,,,0:13.00\n"+
		"2026-09-29T10:15:00Z,,,204800,26,,20971520,,,70,,,,,0:43.00\n") // B: 30 s over 300 s = 10 %
	rec := `{"timings":[{"seq":1,"at":"2026-09-29T10:05:00Z","trigger":"open","event":"m5","ms":900},{"seq":2,"at":"2026-09-29T10:15:00Z","trigger":"open","event":"m5","ms":300}]}`
	write("counters.jsonl", rec+"\n"+rec+"\n")
	write("session-end.json", rec)
	var out, errOut bytes.Buffer
	if code := run([]string{"-in", dir}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	for _, want := range []string{
		"| A idle | 10.0 | 2 | 1.0 | 100.0 / 200.0 | 10.0 | 50 | 24 |", // nearest rank: two samples' median is the lower
		"| B radio | 10.0 | 2 | 10.0 | 200.0 / 200.0 | 20.0 | 60 | 26 |",
		"| A idle | open | m5 | 1 | 900 | 900 | 900 |",
		"| B radio | open | m5 | 1 | 300 | 300 | 300 |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the summary lacks %q:\n%s", want, got)
		}
	}
}

// TestColdRunsAreTheirOwnProcesses: each cold-N.json restarts seq at 1, and
// is not deduped against another.
func TestColdRunsAreTheirOwnProcesses(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "phases.log"), []byte("2026-09-29T10:00:00Z L launched\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for i, ms := range []string{"1000", "2000", "3000"} {
		if err := os.WriteFile(filepath.Join(dir, "cold-"+string(rune('1'+i))+".json"),
			[]byte(`{"timings":[{"seq":1,"at":"2026-09-29T10:00:01Z","trigger":"open","event":"m5","ms":`+ms+`}]}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	run([]string{"-in", dir}, &out, &bytes.Buffer{})
	if !strings.Contains(out.String(), "| open | m5 | 3 | 2000 | 3000 | 3000 |") {
		t.Errorf("three cold runs are three samples:\n%s", out.String())
	}
}
