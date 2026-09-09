package app

// debug.go — the opt-in loopback debug server (pprof, /debug/counters, /debug/dump) and the launch-timing report. Split from dashboard.go by the quality pass (Q2, pure move).

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"strings"
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
	go func() { _ = http.ListenAndServe(debugAddr(), debugMux(d)) }()
}

// debugMux is the routes, apart from the listener, so they can be exercised
// without binding a port.
func debugMux(d *dumper) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/counters", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(d.record(time.Now()))
	})
	mux.HandleFunc("/debug/dump", func(w http.ResponseWriter, r *http.Request) {
		// POST, BECAUSE IT WRITES (F-9). A GET is what any page a developer has
		// open can issue at 127.0.0.1:6060 without reading the answer — the
		// browser needs no permission to make the request, only to see the
		// reply — and this route writes a profile set to disk. The route is
		// opt-in and loopback-only, which is why this was hardening rather than
		// an incident; FR-4 is what changes that, by documenting the debug
		// surface for users.
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "use POST: this writes a dump set to disk", http.StatusMethodNotAllowed)
			return
		}
		dir, err := d.Dump(time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		_, _ = fmt.Fprintln(w, dir)
	})
	return mux
}

// debugAddrDefault is where the debug server lives when nothing overrides it.
const debugAddrDefault = "127.0.0.1:6060"

// debugAddr is the loopback address of the debug server: 127.0.0.1:6060,
// or WATCHPOST_DEBUG_PPROF_ADDR so a second instrumented instance on one
// machine (a soak beside a soak) can pick its own port.
// A PORT, NOT AN ADDRESS (red team 2026-09-05, S-2). This returned whatever the
// environment said, verbatim, so WATCHPOST_DEBUG_PPROF_ADDR=0.0.0.0:6060 bound
// every interface and published three unauthenticated routes — one of which
// writes profile sets to disk on a GET — while the comment three lines up said
// "Loopback only". The variable exists so a second instrumented instance can
// pick its own PORT; that is all it may now do.
func debugAddr() string {
	v := os.Getenv("WATCHPOST_DEBUG_PPROF_ADDR")
	if v == "" {
		return debugAddrDefault
	}
	// A BARE PORT, or a host:port whose host is loopback. Anything else falls
	// back to the default rather than binding where it was told to.
	host, port := "", strings.TrimPrefix(v, ":")
	if h, p, err := net.SplitHostPort(v); err == nil {
		host, port = h, p
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return debugAddrDefault
	}
	if host != "" && host != "localhost" {
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			return debugAddrDefault
		}
	}
	return "127.0.0.1:" + port
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
