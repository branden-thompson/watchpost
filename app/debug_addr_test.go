package app

import "testing"

// S-2 — THE DEBUG SERVER IS LOOPBACK, AND THAT IS NOW ENFORCED.
//
// debugAddr returned the environment's value verbatim, so an address there bound
// every interface — three unauthenticated routes, one of which writes profile
// sets to disk on a GET — while the comment above it said "Loopback only".
func TestTheDebugServerCannotBeMovedOffLoopback(t *testing.T) {
	for _, tc := range []struct{ set, want string }{
		{"", "127.0.0.1:6060"},
		{"7070", "127.0.0.1:7070"},         // a port is the supported use: a soak beside a soak
		{":7070", "127.0.0.1:7070"},        // and the ":port" spelling of it
		{"0.0.0.0:6060", "127.0.0.1:6060"}, // every interface — refused
		{"[::]:6060", "127.0.0.1:6060"},    // the same, spelled for v6
		{"evil.example:80", "127.0.0.1:6060"},
		{"127.0.0.1:6061", "127.0.0.1:6061"}, // a loopback host with a port is fine
		{"localhost:6061", "127.0.0.1:6061"},
		{"192.168.1.9:6060", "127.0.0.1:6060"}, // a LAN address — refused
	} {
		t.Setenv("WATCHPOST_DEBUG_PPROF_ADDR", tc.set)
		if got := debugAddr(); got != tc.want {
			t.Errorf("WATCHPOST_DEBUG_PPROF_ADDR=%q → %q, want %q", tc.set, got, tc.want)
		}
	}
}
