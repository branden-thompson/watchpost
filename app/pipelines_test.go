package app

// Quality pass Q2 (L3-F22): the seed list, the tides client, the pipeline
// container's nil-safety, the launch-timing report — the composition root's
// small pure and offline paths.

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestSeedRecentSkipsFavouritesAndTagsEverySeed(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	prio := []snapshot.LocationRef{{Label: "New York, NY", Zip: "10001"}}
	seeds := seedRecent(idx, prio, 8)
	if len(seeds) != 8 {
		t.Fatalf("the seed list fills to n from the embedded index, got %d", len(seeds))
	}
	seen := map[string]bool{}
	for _, s := range seeds {
		if s.Zip == "10001" {
			t.Fatal("a favourite's zip never seeds the RECENT list")
		}
		if s.Tag == "" || s.Zip == "" || s.Label == "" {
			t.Fatalf("every seed carries label, zip and a derived tag: %+v", s)
		}
		if seen[s.Zip] {
			t.Fatalf("seeds are unique by zip: %s", s.Zip)
		}
		seen[s.Zip] = true
	}
	if got := seedRecent(idx, nil, 0); len(got) != 0 {
		t.Fatal("n = 0 seeds nothing")
	}
	if got := seedRecent(nil, nil, 8); got != nil {
		t.Fatal("no index (the embedded data failed to load): no seeds, no panic — the list is a nicety")
	}
}

func TestNewCoopsIsPacedOnItsOwnClient(t *testing.T) {
	tides, client, err := newCoops()
	if err != nil || tides == nil || client == nil {
		t.Fatalf("the tides provider and its client both come back: %v", err)
	}
	if tides.ID() != "coops" {
		t.Fatalf("provider id: %s", tides.ID())
	}
}

func TestLivePipelinesAreNilSafeBeforeWiring(t *testing.T) {
	lp := &livePipelines{recent: &recentPipeline{}}
	if got := lp.providers(); len(got) != 1 || got[0] != nil {
		t.Fatalf("the reference provider's slot is always first; nothing else before wiring: %v", got)
	}
	lp.stopAll()                                 // a recent pipeline with no schedulers stops cleanly
	lp.markFIRMS()                               // no FIRMS provider: nothing to mark
	lp.wireDeckWarnings()                        // no deck, no priority pipeline: nothing to wire
	lp.hydrate(snapshot.LocationRef{Label: "x"}) // no assembler: a no-op
	if st := lp.ttyStats(); len(st.Requests.Hosts) != 0 || st.Pipelines[0].Publishes != 0 {
		t.Fatalf("no clients, no counters: %+v", st)
	}
}

func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	f()
	_ = w.Close()
	os.Stderr = old
	var b bytes.Buffer
	_, _ = io.Copy(&b, r)
	return b.String()
}

func TestReportTimingIsOptInAndNamesTheMeasurement(t *testing.T) {
	t.Setenv("WATCHPOST_DEBUG_TIMING", "")
	if out := captureStderr(t, func() { reportTiming(2 * time.Second) }); out != "" {
		t.Fatalf("silent unless asked, got %q", out)
	}
	t.Setenv("WATCHPOST_DEBUG_TIMING", "1")
	if out := captureStderr(t, func() { reportTiming(1500 * time.Millisecond) }); !strings.Contains(out, "M1 launch->full view = 1.5s") {
		t.Fatalf("the measurement is printed with its target, got %q", out)
	}
	if out := captureStderr(t, func() { reportTiming(0) }); !strings.Contains(out, "never fired") {
		t.Fatalf("a timer that never fired says so instead of printing 0, got %q", out)
	}
}

// Quality pass Q3 (plan §2.4, PF-9): the RECENT publisher's window folds a
// tier tick's wave of triggers into one publish; the priority window is
// the 50 ms one.
func TestPublisherWindowFoldsAWave(t *testing.T) {
	var runs atomic.Int32
	pb := &publisher{window: 40 * time.Millisecond, run: func() *snapshot.Snapshot { runs.Add(1); return nil }}
	pb.Trigger() // the first publish is immediate (F-1); the wave comes after it
	time.Sleep(20 * time.Millisecond)
	for range 20 {
		pb.Trigger()
		time.Sleep(time.Millisecond)
	}
	time.Sleep(120 * time.Millisecond)
	if got := runs.Load(); got != 2 {
		t.Fatalf("twenty triggers inside one window publish once, got %d", got-1)
	}
	if st := pb.stats(); st.Publishes != 2 || st.Folded != 19 {
		t.Fatalf("the counters say so: %+v", st)
	}
	if recentPublishCoalesce < time.Second || publishCoalesce != 50*time.Millisecond {
		t.Fatalf("RECENT folds a wave (seconds); priority stays at 50 ms: %v / %v", recentPublishCoalesce, publishCoalesce)
	}
}

// Follow-up F-1 (HUM LEAD, 2026-08-27): the seed rows must be on screen at
// once — the first publish never waits for a window — and while the launch
// burst lands the rows fill progressively (the launch window), so the
// table rehydrates cell by cell under the loading shimmer instead of
// appearing all at once after the steady-state window.
func TestFirstPublishIsImmediateAndLaunchFillsProgressively(t *testing.T) {
	var runs atomic.Int32
	pb := &publisher{window: 400 * time.Millisecond, launchWindow: 40 * time.Millisecond, launchUntil: time.Now().Add(300 * time.Millisecond),
		run: func() *snapshot.Snapshot { runs.Add(1); return nil }}
	start := time.Now()
	pb.Trigger()
	deadline := time.Now().Add(100 * time.Millisecond)
	for runs.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if runs.Load() != 1 || time.Since(start) > 100*time.Millisecond {
		t.Fatalf("the first publish is immediate: runs=%d after %v", runs.Load(), time.Since(start))
	}
	pb.Trigger() // inside the launch phase: the short window
	time.Sleep(120 * time.Millisecond)
	if runs.Load() != 2 {
		t.Fatalf("a launch-phase trigger publishes within the launch window: runs=%d", runs.Load())
	}
	time.Sleep(250 * time.Millisecond) // past launchUntil
	pb.Trigger()
	time.Sleep(120 * time.Millisecond)
	if runs.Load() != 2 {
		t.Fatalf("after the launch phase the steady-state window holds: runs=%d", runs.Load())
	}
	time.Sleep(400 * time.Millisecond)
	if runs.Load() != 3 {
		t.Fatalf("…and publishes when it elapses: runs=%d", runs.Load())
	}
	if recentLaunchWindow >= time.Second || recentLaunchPhase < 30*time.Second {
		t.Fatalf("the RECENT launch window is sub-second for the whole launch burst: %v for %v", recentLaunchWindow, recentLaunchPhase)
	}
}
