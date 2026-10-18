package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// TestARunThatIsNotOneIsRefused: phases out of time order, or cumulative CPU
// falling inside a phase (two processes' samples mixed), are not a run - the
// first is refused, the second leaves the phase unreported rather than
// reporting a negative CPU.
func TestARunThatIsNotOneIsRefused(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "phases.log"), []byte("2026-09-29T10:10:00Z B\n2026-09-29T10:00:00Z A\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	if code := run([]string{"-in", dir}, &bytes.Buffer{}, &errOut); code == 0 || !strings.Contains(errOut.String(), "out of time order") {
		t.Errorf("phases out of order were read as a run (exit %d): %s", code, errOut.String())
	}
	s := []sample{{at: time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC), cpuS: 50}, {at: time.Date(2026, 9, 29, 10, 5, 0, 0, time.UTC), cpuS: 10}}
	if _, ok := resourcesIn(s, s[0].at, s[1].at.Add(time.Second)); ok {
		t.Error("a phase whose cumulative CPU fell was reported")
	}
}

// TestARunThatLostItsMapFails is the workload's render check (W14): a map
// draw that fell under its floor is named by phase, and the summary exits
// non-zero - a run that did not render passes for nothing.
func TestARunThatLostItsMapFails(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("phases.log", "2026-09-29T10:00:00Z C map open\n2026-09-29T10:10:00Z quit ok\n")
	write("cold-1.json", `{"timings":[{"seq":1,"at":"2026-09-29T10:01:00Z","trigger":"open","event":"m5","ms":900}]}`)
	var out bytes.Buffer
	if code := run([]string{"-in", dir}, &out, &bytes.Buffer{}); code != 0 || !strings.Contains(out.String(), "the map was drawn in every phase") {
		t.Fatalf("a run that drew its map: exit %d\n%s", code, out.String())
	}
	write("cold-1.json", `{"timings":[{"seq":1,"at":"2026-09-29T10:01:00Z","trigger":"render","event":"below-floor","ms":0},{"seq":2,"at":"2026-09-29T10:02:00Z","trigger":"render","event":"below-floor","ms":0}]}`)
	out.Reset()
	if code := run([]string{"-in", dir}, &out, &bytes.Buffer{}); code == 0 || !strings.Contains(out.String(), "| C map open | 2 |") {
		t.Errorf("a run that lost its map twice in C: exit %d\n%s", code, out.String())
	}
}

// TestAG1RunIsTheDifferenceOfItsArms is 0.19.0 W10.2: each arm measured from
// its "G measured" phase, the use arm's added CPU and RSS over the idle
// arm's, its downloads from the counters' wire bytes, each against G1's
// target; a target missed is a failed run.
func TestAG1RunIsTheDifferenceOfItsArms(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const head = "utc,elapsed,rss_kb,footprint_kb,threads,pcpu,heap_alloc,heap_inuse,heap_objects,goroutines,fds,disk_files,disk_bytes,publishes_recent,cpu_time\n"
	arm := func(name string, rssKB [3]int, cpu [3]string, wire [3]int64) {
		write(name+"/phases.log", "2026-10-19T00:00:00Z L launched: first data\n2026-10-19T01:00:00Z G measured\n")
		// The sample before the measure is the open's: never counted.
		write(name+"/samples.csv", head+
			"2026-10-19T00:30:00Z,,999999,,20,,,,,50,,,,,0:01.00\n"+
			fmt.Sprintf("2026-10-19T01:00:00Z,,%d,,20,,,,,50,,,,,%s\n", rssKB[0], cpu[0])+
			fmt.Sprintf("2026-10-20T01:00:00Z,,%d,,20,,,,,50,,,,,%s\n", rssKB[1], cpu[1])+
			fmt.Sprintf("2026-10-21T01:00:00Z,,%d,,20,,,,,50,,,,,%s\n", rssKB[2], cpu[2]))
		var recs strings.Builder
		for i, at := range []string{"2026-10-19T00:30:00Z", "2026-10-19T01:00:00Z", "2026-10-21T01:00:00Z"} {
			fmt.Fprintf(&recs, "{\"at\":%q,\"propagation_wire_bytes\":%d}\n", at, wire[i])
		}
		write(name+"/counters.jsonl", recs.String())
	}
	sum := func(useCPU string, useWire int64) (string, int) {
		arm("use", [3]int{204800, 215040, 225280}, [3]string{"10:00.00", "40:00.00", useCPU}, [3]int64{5e6, 6e6, useWire})
		arm("idle", [3]int{102400, 112640, 122880}, [3]string{"10:00.00", "30:00.00", "50:00.00"}, [3]int64{0, 0, 0})
		var out, errOut bytes.Buffer
		code := run([]string{"-g1", dir}, &out, &errOut)
		if code == 2 {
			t.Fatalf("exit 2: %s", errOut.String())
		}
		return out.String(), code
	}
	// Over 48 h the use arm spends 62 minutes of CPU (2.15 %), the idle arm
	// 40 (1.39 %): +0.76; RSS medians 210 and 110 MB; 6 MB to 486 MB on the
	// wire, counted from the first record inside the run, is 10 MB an hour.
	out, code := sum("1:12:00", 486e6)
	for _, want := range []string{
		"| the mode in use | 48.0 | 3 | 2.15 | 210.0 / 220.0 |",
		"| the mode never opened | 48.0 | 3 | 1.39 | 110.0 / 120.0 |",
		"| added mean CPU, % of one core | +0.76 | at most 1 | met |",
		"| added RSS, MB (medians) | +100.0 | at most 25 | **missed** |",
		"| downloads on the wire while open, MB an hour | 10.00 | at most 19 | met |",
		"| downloads with the mode never opened, MB | 0.00 | none | met |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the summary lacks %q:\n%s", want, out)
		}
	}
	if code != 1 || strings.Contains(out, "a dry run") {
		t.Errorf("a 48-hour run missing a target exits %d", code)
	}
	if out, _ := sum("2:00:00", 1e9); !strings.Contains(out, "| 20.71 | at most 19 | **missed** |") || !strings.Contains(out, "| +2.43 | at most 1 | **missed** |") {
		t.Errorf("1 GB and 110 minutes of CPU over 48 hours are not misses:\n%s", out)
	}
	write("use/counters.jsonl", "{\"at\":\"2026-10-20T01:00:00Z\",\"propagation_wire_bytes\":5}\n{\"at\":\"2026-10-20T01:30:00Z\",\"propagation_wire_bytes\":9}\n")
	var u bytes.Buffer
	if run([]string{"-g1", dir}, &u, &bytes.Buffer{}); !strings.Contains(u.String(), "| unread | at most 19 | **missed** |") {
		t.Errorf("records half an hour apart are read as a rate:\n%s", u.String())
	}
	write("idle/counters.jsonl", "{\"at\":\"2026-10-20T01:00:00Z\",\"propagation_wire_bytes\":5}\n{\"at\":\"2026-10-21T01:00:00Z\",\"propagation_wire_bytes\":9}\n")
	var b bytes.Buffer
	if run([]string{"-g1", dir}, &b, &bytes.Buffer{}); !strings.Contains(b.String(), "| 0.00 | none | **missed** |") {
		t.Errorf("an idle arm that read the wire is not a miss:\n%s", b.String())
	}
	write("idle/phases.log", "2026-10-19T00:00:00Z L launched: first data\n")
	if code := run([]string{"-g1", dir}, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Errorf("an arm that never reached its measure exits %d", code)
	}
	if code := run([]string{"-g1", dir, "-in", dir}, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Errorf("both flags exit %d", code)
	}
}
