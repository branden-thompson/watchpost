package app

// debug.go — the opt-in loopback debug server (pprof, /debug/counters, /debug/dump) and the launch-timing report.

import (
	"encoding/json"
	"fmt"
	"io"
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
// default; never in release notes as a feature. Two more routes serve the
// soak harness: /debug/counters (counters.json, live) and
// /debug/dump (write a dump set — the trigger on platforms without SIGUSR1).
func startDebugProfiles(d *dumper) {
	if os.Getenv("WATCHPOST_DEBUG_PPROF") != "1" {
		return
	}
	addr := debugAddr()
	go serveDebug(addr, debugMux(d, addr), os.Stderr)
}

// serveDebug serves h on addr until the listener fails, and says on stderr
// why it stopped - a port already taken included (IS-M5).
func serveDebug(addr string, h http.Handler, stderr io.Writer) {
	err := http.ListenAndServe(addr, h)
	_, _ = fmt.Fprintf(stderr, "watchpost debug: not serving on %s: %v\n", addr, err)
}

// debugMux is the routes, apart from the listener, so they can be exercised
// without binding a port. addr is the loopback address the server listens
// on; every route answers only a request addressed to it.
func debugMux(d *dumper, addr string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile) // a CPU profile: not a named profile, so Index cannot serve it (W14, CPU-1)
	mux.HandleFunc("/debug/counters", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(d.record(time.Now()))
	})
	mux.HandleFunc("/debug/dump", func(w http.ResponseWriter, r *http.Request) {
		// POST ONLY (F-9): the route writes a profile set to disk, and a
		// state change does not answer a GET. What keeps a web page out is
		// the guard around every route (IS-1), not the method.
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "use POST: this writes a dump set to disk", http.StatusMethodNotAllowed)
			return
		}
		// The answer names no path: a dump's directory carries the user's
		// name. The Status window's LAST DUMP says where, and why one failed.
		if _, err := d.Dump(time.Now()); err != nil {
			http.Error(w, "dump not written; the Status window's LAST DUMP says why", http.StatusTooManyRequests)
			return
		}
		_, _ = fmt.Fprintln(w, "dump written; see the debug directory")
	})
	return loopbackOnly(addr, mux)
}

// loopbackOnly answers a request only when it is addressed to the server's
// own loopback name and port - 127.0.0.1:<port> or localhost:<port> - and
// carries no Origin header (IS-1). A browser sends Origin with a page's
// cross-site POST, and a page whose name was rebound to 127.0.0.1 sends its
// own name as Host; both are refused with 403 before any route runs. The
// soak harness and curl send neither.
func loopbackOnly(addr string, next http.Handler) http.Handler {
	_, port, _ := net.SplitHostPort(addr)
	own := map[string]bool{"127.0.0.1:" + port: true, "localhost:" + port: true}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !own[strings.ToLower(r.Host)] || r.Header.Get("Origin") != "" {
			http.Error(w, "refused: the debug server answers only 127.0.0.1 or localhost on its own port, with no Origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// debugAddrDefault is where the debug server lives when nothing overrides it.
const debugAddrDefault = "127.0.0.1:6060"

// debugAddr is the loopback address of the debug server: 127.0.0.1:6060,
// or WATCHPOST_DEBUG_PPROF_ADDR so a second instrumented instance on one
// machine (a soak beside a soak) can pick its own port.
// A PORT, NOT AN ADDRESS (red team 2026-09-05, S-2). Taking the environment
// verbatim would let WATCHPOST_DEBUG_PPROF_ADDR=0.0.0.0:6060 bind every
// interface and publish three unauthenticated routes, one of which writes
// profile sets to disk. The variable exists so a second instrumented instance
// can pick its own PORT; that is all it may do.
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
// WATCHPOST_DEBUG_TIMING=1 (P10-04).
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
