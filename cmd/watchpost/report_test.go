package main

// Quality pass Q2 (L3-F25): the report command's output paths — plain
// text, --json, --verbose, and the exit code for a degraded provider —
// against a fake fetch, with no network.

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func fakeReport(degraded bool) func(context.Context, string) (*snapshot.Snapshot, httpx.RequestStats, error) {
	return func(_ context.Context, query string) (*snapshot.Snapshot, httpx.RequestStats, error) {
		if query == "nowhere" {
			return nil, httpx.RequestStats{}, errors.New("no match for nowhere")
		}
		t := 22.8
		status := snapshot.ProviderOK
		if degraded {
			status = snapshot.ProviderDegraded
		}
		return &snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion, GeneratedAt: time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
				Locations: []snapshot.Location{{Label: "Oceanside, CA", Zip: "92057", Harmonized: snapshot.Conditions{Temp: &t, Condition: "clear",
					ObservedAt: time.Date(2026, 8, 26, 11, 50, 0, 0, time.UTC), Source: snapshot.SourceInfo{Provider: "nws", ModelOrStation: "KOKB"}}}},
				Providers: []snapshot.ProviderStatus{{ID: "nws", Status: status, Attribution: "NOAA/NWS"}}},
			httpx.RequestStats{Uptime: time.Second, Hosts: []httpx.HostStats{{Host: "api.weather.gov", Attempts: 3, Net: 3, BytesNet: 1234}}}, nil
	}
}

func runReport(t *testing.T, fake func(context.Context, string) (*snapshot.Snapshot, httpx.RequestStats, error), args ...string) (string, error) {
	t.Helper()
	root := newRootCmdWith(fake)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"report"}, args...))
	err := root.Execute()
	return out.String(), err
}

func TestReportPlainJSONVerboseAndExitCodes(t *testing.T) {
	out, err := runReport(t, fakeReport(false), "92057")
	if err != nil || !strings.Contains(out, "Oceanside, CA (92057)") || !strings.Contains(out, "temp 22.8 C") || strings.Contains(out, "\x1b[") {
		t.Fatalf("plain report must carry the location and values with no escapes: err=%v\n%s", err, out)
	}
	if strings.Contains(out, "requests ") {
		t.Fatal("request counters appear only with --verbose")
	}
	out, err = runReport(t, fakeReport(false), "92057", "--verbose")
	if err != nil || !strings.Contains(out, "requests api.weather.gov: attempts 3 net 3") {
		t.Fatalf("--verbose appends one counter line per host: err=%v\n%s", err, out)
	}
	out, err = runReport(t, fakeReport(false), "92057", "--json")
	if err != nil || !strings.Contains(out, `"schema_version"`) || !strings.Contains(out, `"92057"`) || strings.Contains(out, "requests ") {
		t.Fatalf("--json prints the envelope and nothing else: err=%v\n%s", err, out)
	}
	_, err = runReport(t, fakeReport(true), "92057")
	var ec exitCodeError
	if !errors.As(err, &ec) || ec.ExitCode() != 2 {
		t.Fatalf("a degraded provider still prints and exits 2, got %v", err)
	}
	if _, err := runReport(t, fakeReport(false), "nowhere"); err == nil || !strings.Contains(err.Error(), "no match") {
		t.Fatalf("a resolve failure is the command's error: %v", err)
	}
}
