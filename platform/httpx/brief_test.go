package httpx

// brief_test.go — IS-M6: a failure said for the diagnostics names the host
// and the status, or the kind of failure, never the address asked.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"testing"
)

func TestBriefNamesTheHostAndStatusOnly(t *testing.T) {
	secret := "https://www.ndbc.noaa.gov/data/realtime2/46025.txt?key=k"
	for name, c := range map[string]struct {
		err  error
		want string
	}{
		"status":   {fmt.Errorf("ndbc: %w", &StatusError{URL: secret, Status: 503, Attempts: 2, Degraded: true}), "www.ndbc.noaa.gov HTTP 503"},
		"dns":      {&ReachError{URL: secret, Err: &net.DNSError{Name: "www.ndbc.noaa.gov", IsNotFound: true}}, "www.ndbc.noaa.gov name not resolved"},
		"refused":  {&ReachError{URL: secret, Err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}}, "www.ndbc.noaa.gov connection refused"},
		"policy":   {&ReachError{URL: secret, Err: &net.OpError{Op: "dial", Err: fmt.Errorf("refused to dial 10.0.0.1: %w", ErrNotPublic)}}, "www.ndbc.noaa.gov refused: not a public address"},
		"deadline": {fmt.Errorf("x: %w", context.DeadlineExceeded), "timed out"},
		"other":    {fmt.Errorf("uv: EPA has no forecast for %s: %w", "AUSTIN", errors.New("empty")), "failed (*errors.errorString)"},
	} {
		got := Brief(c.err)
		if got != c.want {
			t.Errorf("%s: Brief = %q, want %q", name, got, c.want)
		}
		if strings.Contains(got, "46025") || strings.Contains(got, "AUSTIN") {
			t.Errorf("%s: Brief kept the request: %q", name, got)
		}
	}
	if Brief(nil) != "" {
		t.Error("no error is said as something")
	}
}

func TestScrubURLsKeepsSchemeAndHost(t *testing.T) {
	in := `Get "https://data.epa.gov/efservice/CITY/AUSTIN/STATE/TX/JSON": EOF; then http://127.0.0.1:8080/zones/TXZ192?bbox=1,2 and https://x.example`
	want := `Get "https://data.epa.gov": EOF; then http://127.0.0.1:8080 and https://x.example`
	if got := ScrubURLs(in); got != want {
		t.Errorf("ScrubURLs =\n%q\nwant\n%q", got, want)
	}
}
