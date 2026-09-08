package app

// debug_method_test.go — F-9 / FR-4.7.
//
// /debug/dump WRITES A PROFILE SET TO DISK, and it did it on a GET. A GET is
// what any page a developer has open can issue at 127.0.0.1:6060 without
// reading the response — the browser does not need permission to make the
// request, only to see the answer. The route is opt-in and loopback-only, which
// is why this is a hardening item rather than a live incident; FR-4 is what
// changes the calculus, because it documents the debug surface for users.

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTheDumpRouteRefusesAGet(t *testing.T) {
	dir := t.TempDir()
	mux := debugMux(testDumper(t, dir, time.Now()))

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, "/debug/dump", nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /debug/dump answered %d, want 405: a route that writes to disk must not "+
				"be reachable by a request a page can make without being read", method, w.Code)
		}
		if allow := w.Header().Get("Allow"); allow != http.MethodPost {
			t.Errorf("%s /debug/dump answered 405 with Allow %q, want POST — a refusal that does "+
				"not say what would work sends the reader to the source", method, allow)
		}
	}

	// AND POST STILL WORKS, or this is a gate that broke the tool it guards.
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/debug/dump", nil))
	if w.Code == http.StatusMethodNotAllowed {
		t.Errorf("POST /debug/dump answered 405: the dump trigger no longer works at all")
	}
}

// AND /debug/counters IS A READ, so it keeps its GET.
func TestTheCountersRouteStillAnswersAGet(t *testing.T) {
	mux := debugMux(testDumper(t, t.TempDir(), time.Now()))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/debug/counters", nil))
	if w.Code != http.StatusOK {
		t.Errorf("GET /debug/counters answered %d: it reads counters and changes nothing", w.Code)
	}
}
