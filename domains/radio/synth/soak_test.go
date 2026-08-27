package synth

import (
	"context"
	"io"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestSeismicBroadcastSoak is the R6 soak (P4 gate): the synth broadcast for a
// location with seismic activity plays continuously, and the process stays
// flat — goroutines and heap do not grow. Skipped unless
// WATCHPOST_SEISMIC_SOAK=1; WATCHPOST_SOAK_MINUTES (default 60) sets the run.
//
//	WATCHPOST_SEISMIC_SOAK=1 WATCHPOST_SOAK_MINUTES=60 go test ./domains/radio/synth -run Soak -v -timeout 90m
func TestSeismicBroadcastSoak(t *testing.T) {
	if os.Getenv("WATCHPOST_SEISMIC_SOAK") == "" {
		t.Skip("set WATCHPOST_SEISMIC_SOAK=1 for the 1-hour radio soak")
	}
	mins := 60
	if s := os.Getenv("WATCHPOST_SOAK_MINUTES"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			mins = n
		}
	}
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	sr := SeismicReport{Known: true, Lat: 35.62, Lon: -117.67, State: snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{
		quake(5.1, 141, 15, "N", 72*time.Hour, now),
		quake(4.2, 30, 8, "NE", 2*time.Hour, now),
		quake(3.6, 20, 5, "SSW", 26*time.Hour, now),
		quake(2.8, 6, 3, "E", 90*time.Minute, now),
		quake(1.4, 2, 2, "W", 30*time.Minute, now),
	}}}
	loc := snapshot.Location{Label: "Ridgecrest, CA"}
	segs := Compose(loc, nil, now, true, "rec", Station{}, FireReport{}, sr)

	src, err := NewSource(&recVoice{}, func(context.Context) ([]Segment, error) { return segs, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	src.gap = 20 * time.Millisecond // fast cycle turnover to stress the render loop
	src.Loop(true)

	total := time.Duration(mins) * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), total)
	defer cancel()
	go func() { _, _ = io.Copy(io.Discard, src.Open(ctx)) }() // drain the broadcast continuously

	sample := func(label string) (int, uint64) {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		g := runtime.NumGoroutine()
		t.Logf("%-8s goroutines=%d heapInUse=%d KB", label, g, m.HeapInuse/1024)
		return g, m.HeapInuse
	}

	time.Sleep(3 * time.Second) // let playback reach steady state
	baseG, baseHeap := sample("t=0")
	step := time.Duration(mins) * time.Minute / 6
	if step < time.Minute {
		step = time.Minute
	}
	var lastG int
	var lastHeap uint64
	for elapsed := step; elapsed < total; elapsed += step {
		select {
		case <-ctx.Done():
		case <-time.After(step):
		}
		lastG, lastHeap = sample(elapsed.Round(time.Minute).String())
	}
	cancel()
	time.Sleep(time.Second)
	sample("final")

	// Flat: goroutines must not climb (a small tolerance for the reader/writer
	// pair), and the heap must not run away.
	if lastG > baseG+4 {
		t.Fatalf("goroutine leak: %d → %d over %s", baseG, lastG, total)
	}
	if baseHeap > 0 && lastHeap > baseHeap*3 {
		t.Fatalf("heap growth: %d → %d KB over %s", baseHeap/1024, lastHeap/1024, total)
	}
}
