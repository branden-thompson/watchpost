package app

// debug_origin_test.go — IS-1 and IS-M5: the opt-in debug server answers only
// a request addressed to its own loopback name and port, never one a web page
// sent, never says where a dump was written, and says on stderr when it could
// not bind.

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// debugRequest is a request as the debug server's own client sends it.
func debugRequest(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.Host = debugAddrDefault
	return r
}

// TestTheDebugServerRefusesAForeignHost is IS-1's DNS rebinding: a page whose
// name now resolves to 127.0.0.1 sends its own name as Host. Every route
// refuses it; the server's own names, 127.0.0.1 and localhost on its port,
// are answered.
func TestTheDebugServerRefusesAForeignHost(t *testing.T) {
	mux := debugMux(testDumper(t, t.TempDir(), time.Now()), debugAddrDefault)
	for _, host := range []string{"rebind.attacker.example", "rebind.attacker.example:6060", "127.0.0.1:6061", "localhost", "127.0.0.1", "[::1]:6060", ""} {
		for _, path := range []string{"/debug/counters", "/debug/pprof/", "/debug/dump"} {
			r := httptest.NewRequest(http.MethodGet, path, nil)
			r.Host = host
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Errorf("Host %q, %s: answered %d, want 403", host, path, w.Code)
			}
		}
	}
	for _, host := range []string{"127.0.0.1:6060", "localhost:6060"} {
		r := httptest.NewRequest(http.MethodGet, "/debug/counters", nil)
		r.Host = host
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("Host %q: answered %d, want 200 - the server's own name", host, w.Code)
		}
	}
}

// TestTheDebugServerRefusesAPagesRequest is IS-1's cross-site POST: a
// text/plain POST from a page carries an Origin, and is refused before
// anything is written.
func TestTheDebugServerRefusesAPagesRequest(t *testing.T) {
	dir := t.TempDir()
	mux := debugMux(testDumper(t, dir, time.Now()), debugAddrDefault)
	for _, path := range []string{"/debug/dump", "/debug/counters"} {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader("x"))
		r.Host = debugAddrDefault
		r.Header.Set("Content-Type", "text/plain")
		r.Header.Set("Origin", "https://evil.example")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s with an Origin: answered %d, want 403", path, w.Code)
		}
	}
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Errorf("a refused request wrote %d entries", len(left))
	}
}

// TestTheDumpAnswerNamesNoPath is IS-1: the answer to a dump is that it was
// written, never where - the path carries the user's name.
func TestTheDumpAnswerNamesNoPath(t *testing.T) {
	dir := t.TempDir()
	mux := debugMux(testDumper(t, dir, time.Now()), debugAddrDefault)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, debugRequest(http.MethodPost, "/debug/dump"))
	if w.Code != http.StatusOK {
		t.Fatalf("POST /debug/dump answered %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), dir) || strings.Contains(w.Body.String(), string(os.PathSeparator)) {
		t.Errorf("the answer names a path: %q", w.Body.String())
	}
	if left, _ := os.ReadDir(dir); len(left) == 0 {
		t.Error("the dump was not written")
	}
}

// TestADebugServerThatCannotBindSaysSo is IS-M5: a port already taken is said
// on stderr, not swallowed.
func TestADebugServerThatCannotBindSaysSo(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = taken.Close() }()
	var stderr bytes.Buffer
	serveDebug(taken.Addr().String(), http.NewServeMux(), &stderr)
	if !strings.Contains(stderr.String(), taken.Addr().String()) {
		t.Errorf("a failed bind said %q, want the address it could not take", stderr.String())
	}
}
