package tty

// m6_measure_test.go — M6 MEASURED IN PROCESS WITH THE LOOP PLAYING (D-53,
// D-251; W8.13's "with the Observer's publishers replaying").
//
//	WATCHPOST_VALIDATE_M5=1 go test -run '^TestMeasureM6InProcess$' -count=1 -v -timeout 10m ./modes/tty
//
// It runs only under WATCHPOST_VALIDATE_M5=1, so neither `make verify` nor CI
// ever runs it: it takes a minute and more, and its numbers are read, never
// asserted.
//
// The station's radar client keeps nothing on disk and dials only public
// HTTPS hosts, so the built binary on recorded responses has no radar to
// play (cmd/watchpost/measure_test.go). Here the Router and its dashboard run
// in a real Bubble Tea program at 149x38, the renderer writing to nowhere,
// with the map built on the embedded basemap and the radar answered by
// radarFeed's twelve recorded IEM frames over the lower 48. The Observer is
// live as a publisher keeps it: the test snapshot is sent once a second, and
// the dashboard's own clocks run. The map opens with g, 1 shows the lower 48,
// and space plays the loop for WATCHPOST_VALIDATE_M6_SECS seconds (60 by
// default) with → and ← pressed in turn every ten seconds; then the tick
// lateness, the intervals between delivered ticks, the keys' time to their
// frames and the hundred-frame times are logged as distributions.

import (
	"context"
	"io"
	"math"
	"os"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// stampedTiming is one interval and when it was said.
type stampedTiming struct {
	Timing
	at time.Time
}

// timingTape keeps what the instrument says, from any goroutine.
type timingTape struct {
	mu  sync.Mutex
	got []stampedTiming
}

func (tp *timingTape) record(t Timing) {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.got = append(tp.got, stampedTiming{t, time.Now()})
}

// since is a copy of what was said from index i on.
func (tp *timingTape) since(i int) []stampedTiming {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	return slices.Clone(tp.got[min(i, len(tp.got)):])
}

func (tp *timingTape) len() int {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	return len(tp.got)
}

// waitFor waits up to bound for trigger's event, said after index from.
func (tp *timingTape) waitFor(t *testing.T, from int, trigger, event string, bound time.Duration) time.Duration {
	t.Helper()
	for deadline := time.Now().Add(bound); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		for _, s := range tp.since(from) {
			if s.Trigger == trigger && s.Event == event {
				return s.After
			}
		}
	}
	t.Fatalf("no %s:%s within %v", trigger, event, bound)
	return 0
}

func TestMeasureM6InProcess(t *testing.T) {
	if os.Getenv("WATCHPOST_VALIDATE_M5") != "1" {
		t.Skip("measures M6 with the loop playing for a minute; WATCHPOST_VALIDATE_M5=1 runs it")
	}
	secs := 60
	if v, err := strconv.Atoi(os.Getenv("WATCHPOST_VALIDATE_M6_SECS")); err == nil && v > 0 {
		secs = v
	}
	tape := &timingTape{}
	var radarMu sync.Mutex
	asked := new([]string)
	radar := radarFeed(t, "MRMS", asked)
	d, err := NewDashboard(Config{Version: "0.18.0-measure", NewMap: embeddedMap, Timed: tape.record,
		MapFeed: boxFeed(-117.6, -117.1, false),
		MapRadar: func(ctx context.Context, ask MapAsk) MapRadar {
			radarMu.Lock() // radarFeed's record of asks is not shared-safe; the window asks off its own goroutine
			defer radarMu.Unlock()
			return radar(ctx, ask)
		},
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	if err != nil {
		t.Fatal(err)
	}
	p := tea.NewProgram(NewRouter(d), tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithWindowSize(149, 38),
		tea.WithoutSignalHandler())
	ended := make(chan tea.Model, 1)
	go func() {
		m, err := p.Run()
		if err != nil {
			t.Errorf("the program ended with %v", err)
		}
		ended <- m
	}()
	t.Cleanup(func() {
		p.Quit()
		if r, ok := (<-ended).(Router); ok {
			r.CloseMap()
		}
	})

	live, stopLive := context.WithCancel(context.Background())
	defer stopLive()
	p.Send(SnapshotMsg{Snap: placedSnap()})
	go func() {
		for tk := time.NewTicker(time.Second); ; {
			select {
			case <-live.Done():
				tk.Stop()
				return
			case <-tk.C:
				p.Send(SnapshotMsg{Snap: placedSnap()})
			}
		}
	}()

	press := func(code rune, text string) { p.Send(tea.KeyPressMsg{Code: code, Text: text}) }
	from := tape.len()
	press('g', "g")
	t.Logf("g: the alerts answered after %v", tape.waitFor(t, from, "open", "answered:feed", time.Minute)) // at the place the embedded basemap is a stand-in, never complete
	from = tape.len()
	press('1', "1")
	t.Logf("1: the lower 48 settled after %v", tape.waitFor(t, from, string(mapRegionActs[0]), "settled", time.Minute))

	from = tape.len()
	press(' ', " ")
	start := time.Now()
	pans := []struct {
		code rune
		name string
	}{{tea.KeyRight, "→"}, {tea.KeyLeft, "←"}}
	keys := []string{"space"}
	for n, next := 0, start.Add(10*time.Second); time.Since(start) < time.Duration(secs)*time.Second; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(next) {
			press(pans[n%2].code, "")
			keys = append(keys, pans[n%2].name)
			n++
			next = next.Add(10 * time.Second)
		}
	}
	stopLive()

	samples := map[string][]float64{}
	var lastTick time.Time
	var between []string // what was said since the last tick
	for _, s := range tape.since(from) {
		if s.Event != "tick" {
			between = append(between, s.Trigger+":"+s.Event+"@"+s.at.Sub(start).Round(time.Millisecond).String())
		} else {
			if gap := s.at.Sub(lastTick); !lastTick.IsZero() && gap > 1500*time.Millisecond {
				t.Logf("a %v interval ending at %v; said between the two ticks: %v", gap.Round(time.Millisecond), s.at.Sub(start).Round(time.Millisecond), between)
			}
			between = nil
		}
		ms := float64(s.After.Microseconds()) / 1000
		switch s.Event {
		case "key", "frames100":
			samples[s.Event] = append(samples[s.Event], ms)
		case "tick":
			samples["tick-late"] = append(samples["tick-late"], ms)
			if !lastTick.IsZero() {
				samples["tick-interval"] = append(samples["tick-interval"], float64(s.at.Sub(lastTick).Microseconds())/1000)
			}
			lastTick = s.at
		}
	}
	radarMu.Lock()
	t.Logf("played %d s at 149x38; keys %v; radar asked %d times", secs, keys, len(*asked))
	radarMu.Unlock()
	for _, k := range []string{"tick-interval", "tick-late", "key", "frames100"} {
		s := m6Spread(samples[k])
		t.Logf("M6 %-13s n=%.0f p50=%.1f p90=%.1f p99=%.1f max=%.1f ms", k, s[0], s[1], s[2], s[3], s[4])
	}
	if path := os.Getenv("WATCHPOST_VALIDATE_M6_OUT"); path != "" {
		var b []byte
		for _, k := range []string{"tick-interval", "tick-late", "key", "frames100"} {
			b = append(b, k...)
			for _, v := range samples[k] {
				b = append(b, ' ')
				b = strconv.AppendFloat(b, v, 'f', 3, 64)
			}
			b = append(b, '\n')
		}
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// m6Spread is xs's count, p50, p90, p99 and max by nearest rank.
func m6Spread(xs []float64) [5]float64 {
	if len(xs) == 0 {
		return [5]float64{}
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	rank := func(p float64) float64 { return s[max(int(math.Ceil(p*float64(len(s))))-1, 0)] }
	return [5]float64{float64(len(s)), rank(0.50), rank(0.90), rank(0.99), s[len(s)-1]}
}
