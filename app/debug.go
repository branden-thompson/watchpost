package app

// debug.go — the opt-in loopback debug server (pprof, /debug/counters, /debug/dump) and the launch-timing report. Split from dashboard.go by the quality pass (Q2, pure move).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// startDebugProfiles serves Go's runtime profiles on 127.0.0.1:6060 when
// WATCHPOST_DEBUG_PPROF=1 (UAT 73/74): threadcreate, goroutine, heap —
// the way to read a live process rather than guess. Loopback only; off by
// default; never in release notes as a feature. Quality pass Q0 adds two
// routes the soak harness reads: /debug/counters (counters.json, live) and
// /debug/dump (write a dump set — the trigger on platforms without SIGUSR1).
func startDebugProfiles(d *dumper) {
	if os.Getenv("WATCHPOST_DEBUG_PPROF") != "1" {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/counters", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(d.record(time.Now()))
	})
	mux.HandleFunc("/debug/dump", func(w http.ResponseWriter, _ *http.Request) {
		dir, err := d.Dump(time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		_, _ = fmt.Fprintln(w, dir)
	})
	go func() { _ = http.ListenAndServe(debugAddr(), mux) }()
}

// debugAddr is the loopback address of the debug server: 127.0.0.1:6060,
// or WATCHPOST_DEBUG_PPROF_ADDR so a second instrumented instance on one
// machine (a soak beside a soak) can pick its own port.
func debugAddr() string {
	if addr := os.Getenv("WATCHPOST_DEBUG_PPROF_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:6060"
}

// reportTiming prints the M1 launch->full-view measurement when
// WATCHPOST_DEBUG_TIMING=1 (split from RunDashboard, P10-04).
func reportTiming(firstFull time.Duration) {
	if os.Getenv("WATCHPOST_DEBUG_TIMING") != "1" {
		return
	}
	if err := invariant.Check(firstFull > 0, "M1 timer never fired — no fully-populated snapshot"); err != nil {
		fmt.Fprintln(os.Stderr, "watchpost timing:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "watchpost timing: M1 launch->full view = %s (target warm<=3s cold<=8s)\n", firstFull.Round(10*time.Millisecond))
}
