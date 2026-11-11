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

// awaitRuns waits for the publish count to REACH n. It is a failure bound: it
// costs nothing when the publish is prompt and only fires if it never happens.
//
// Waiting longer than necessary is safe here and that is the point — once the
// wave has been sent, no further trigger exists to publish, so the count cannot
// drift past n no matter how long we wait. That is what makes "exactly n"
// assertable without racing a clock.
func awaitRuns(t *testing.T, runs *atomic.Int32, n int32) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if runs.Load() >= n {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("publishes never reached %d (got %d)", n, runs.Load())
}

// TestPublisherWindowFoldsAWave is quality pass Q3 (plan §2.4, PF-9): the
// RECENT publisher's window folds a tier tick's wave of triggers into one
// publish; the priority window is the 50 ms one.
func TestPublisherWindowFoldsAWave(t *testing.T) {
	var runs atomic.Int32
	// THE WINDOW IS GENEROUS ON PURPOSE. Twenty triggers spaced a millisecond
	// apart very nearly fill a 40 ms window; on a loaded CI box the wave spills
	// past, a second window opens, and folding twice is the CORRECT behaviour for
	// that timing (F-51).
	//
	// So the wave is sent back-to-back — microseconds — into a half-second
	// window, and the assertions wait for a state instead of sleeping past one.
	pb := &publisher{window: 500 * time.Millisecond, run: func() *snapshot.Snapshot { runs.Add(1); return nil }}
	pb.Trigger() // the first publish is immediate (F-1); the wave comes after it
	awaitRuns(t, &runs, 1)
	for range 20 {
		pb.Trigger() // no spacing: all twenty are inside one window by construction
	}
	awaitRuns(t, &runs, 2)
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
	clock := &windowClock{at: time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)}
	pb := &publisher{window: 400 * time.Millisecond, launchWindow: 40 * time.Millisecond, launchUntil: clock.at.Add(300 * time.Millisecond),
		run: func() *snapshot.Snapshot { runs.Add(1); return nil }, now: clock.Now, after: clock.After}
	pb.Trigger()
	clock.Advance(0)
	if runs.Load() != 1 {
		t.Fatalf("the first publish is immediate: runs=%d", runs.Load())
	}
	pb.Trigger() // inside the launch phase: the short window
	clock.Advance(39 * time.Millisecond)
	if runs.Load() != 1 {
		t.Fatalf("a launch-phase trigger published before its window: runs=%d", runs.Load())
	}
	clock.Advance(time.Millisecond)
	if runs.Load() != 2 {
		t.Fatalf("a launch-phase trigger publishes when the launch window elapses: runs=%d", runs.Load())
	}
	clock.Advance(300 * time.Millisecond) // past launchUntil
	pb.Trigger()
	clock.Advance(399 * time.Millisecond)
	if runs.Load() != 2 {
		t.Fatalf("after the launch phase the steady-state window holds: runs=%d", runs.Load())
	}
	clock.Advance(time.Millisecond)
	if runs.Load() != 3 {
		t.Fatalf("…and publishes when it elapses: runs=%d", runs.Load())
	}
	if recentLaunchWindow >= time.Second || recentLaunchPhase < 30*time.Second {
		t.Fatalf("the RECENT launch window is sub-second for the whole launch burst: %v for %v", recentLaunchWindow, recentLaunchPhase)
	}
}

// windowClock is a clock and a timer that move only when the test moves them,
// so a publish window is measured exactly (0.19.5 Q2.11, F-51): a timer fires
// on the Advance that reaches it, on the test's goroutine.
type windowClock struct {
	at     time.Time
	timers []windowTimer
}

type windowTimer struct {
	due time.Time
	f   func()
}

func (c *windowClock) Now() time.Time { return c.at }

func (c *windowClock) After(d time.Duration, f func()) {
	c.timers = append(c.timers, windowTimer{due: c.at.Add(d), f: f})
}

// Advance moves the clock on and fires every timer it reaches, in due order.
func (c *windowClock) Advance(d time.Duration) {
	c.at = c.at.Add(d)
	for {
		next := -1
		for i, tm := range c.timers {
			if !tm.due.After(c.at) && (next < 0 || tm.due.Before(c.timers[next].due)) {
				next = i
			}
		}
		if next < 0 {
			return
		}
		f := c.timers[next].f
		c.timers = append(c.timers[:next], c.timers[next+1:]...)
		f()
	}
}
