package app

// Live diagnostic (follow-up F-2): WATCHPOST_LIVE_DIAG="lat,lon,SAME" walks
// the relay resolution for a location and probes every candidate mount —
// status, first audio byte latency — so a "nothing after 90 seconds" can
// be attributed to a mount, a directory or the engine's policy.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

func TestLiveRelayDiagnoseLocation(t *testing.T) {
	spec := os.Getenv("WATCHPOST_LIVE_DIAG")
	if spec == "" {
		t.Skip("set WATCHPOST_LIVE_DIAG=lat,lon,SAME (e.g. 33.158,-117.351,006073) to diagnose one location's relay resolution")
	}
	parts := strings.Split(spec, ",")
	lat, _ := strconv.ParseFloat(parts[0], 64)
	lon, _ := strconv.ParseFloat(parts[1], 64)
	same := ""
	if len(parts) > 2 {
		same = parts[2]
	}
	c, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: 5, MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}
	r, err := stream.NewResolver(stream.NewDirectory(c, "", ""))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	t0 := time.Now()
	stations, statuses := r.ResolveWithStatus(ctx, lat, lon, same)
	t.Logf("resolve took %v; directories: %+v", time.Since(t0).Round(time.Millisecond), statuses)
	for i, st := range stations {
		t.Logf("candidate %d: %s %s (%.0f km, covering=%v, status %q) mounts=%d", i, st.Callsign, st.Site, st.KM, st.Covering, st.Status, len(st.Mounts))
	}
	client := &http.Client{Transport: httpx.NewTransport(), CheckRedirect: httpx.SameOriginRedirect}
	for _, st := range stations {
		for _, m := range st.Mounts {
			pctx, pcancel := context.WithTimeout(ctx, 8*time.Second)
			req, _ := http.NewRequestWithContext(pctx, http.MethodGet, m.URL, nil)
			req.Header.Set("User-Agent", UserAgent)
			req.Header.Set("Icy-MetaData", "1")
			start := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				t.Logf("  %s %s: connect error after %v: %v", st.Callsign, m.URL, time.Since(start).Round(time.Millisecond), err)
				pcancel()
				continue
			}
			buf := make([]byte, 4096)
			n, rerr := io.ReadFull(resp.Body, buf)
			_ = resp.Body.Close()
			pcancel()
			t.Logf("  %s %s [%s]: HTTP %d %s; %d bytes in %v (%v)", st.Callsign, m.URL, m.Relay, resp.StatusCode, resp.Header.Get("Content-Type"), n, time.Since(start).Round(time.Millisecond), errStr(rerr))
		}
	}
}

func errStr(err error) string {
	if err == nil {
		return "ok"
	}
	return fmt.Sprint(err)
}
