package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// TestTheDaysAreReadInclusiveAndBounded is the span a run covers: both days
// included, never backwards, never past maxDays.
func TestTheDaysAreReadInclusiveAndBounded(t *testing.T) {
	ds, err := days("2026-09-29", "2026-10-05")
	if err != nil || len(ds) != 7 || ds[6].Format(time.DateOnly) != "2026-10-05" {
		t.Fatalf("PLAN's week: %v, %v", ds, err)
	}
	for _, c := range [][2]string{{"2026-10-05", "2026-09-29"}, {"2026-09-01", "2026-11-01"}, {"", "2026-10-05"}, {"29/09/2026", "2026-10-05"}} {
		if _, err := days(c[0], c[1]); !errors.Is(err, errDays) {
			t.Errorf("%q to %q: %v; want errDays", c[0], c[1], err)
		}
	}
	if err := run(nil, &bytes.Buffer{}); !errors.Is(err, errUsage) {
		t.Errorf("no command: %v", err)
	}
	if err := run([]string{"grade"}, &bytes.Buffer{}); !errors.Is(err, errUsage) {
		t.Errorf("an unknown command: %v", err)
	}
}

// TestTheDaysQueryIsOneFilteredAggregate is `m1-m4-instrument.md`'s "What
// the instrument would send": one aggregate query a day, filtered to the
// mainland US's receivers and the transmitters the metrics draw from, the
// spots' fields only - locators, never callsigns - its day reaching back to
// its first window's hour.
func TestTheDaysQueryIsOneFilteredAggregate(t *testing.T) {
	q := spotsQuery(time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), defaultRegionKm)
	for _, want := range []string{
		"time >= '2026-09-28 23:00:00' AND time < '2026-09-29 23:00:00'",
		"rx_lat BETWEEN 24 AND 50 AND rx_lon BETWEEN -125 AND -66",
		"band IN (1,3,5,7,10,14,18,21,24,28)",
		"max(snr + (50 - power)) AS best", "count() AS n", "GROUP BY tx_loc",
		"greatCircleDistance(tx_lon, tx_lat, 8.68, 50.11) < 1000000", // Frankfurt
		"FORMAT TSV",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("the query lacks %q:\n%s", want, q)
		}
	}
	for _, never := range []string{"tx_sign", "rx_sign", "SELECT *"} {
		if strings.Contains(q, never) {
			t.Errorf("the query asks %q:\n%s", never, q)
		}
	}
}

// TestTheFetchBacksOffAndKeepsOnlyTheRateHeaders is the courtesy: a refusal
// waits 60 s before the ask again, a day kept is never asked again, the
// instrument names itself, and the log keeps the rate headers alone (D-83).
func TestTheFetchBacksOffAndKeepsOnlyTheRateHeaders(t *testing.T) {
	asks := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asks++
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("asked as %q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Retry-After", "60")
		w.Header().Set("X-Served-By", "cache-node-7")
		if asks == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("row\n"))
	}))
	defer srv.Close()
	dir := t.TempDir()
	var waits []time.Duration
	var log bytes.Buffer
	f := fetcher{base: srv.URL + "/", client: srv.Client(), sleep: func(d time.Duration) { waits = append(waits, d) }, log: &log}
	day := []time.Time{time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)}
	if err := f.fetchDays(day, dir, defaultRegionKm, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if len(waits) != 1 || waits[0] != firstBackOff || asks != 2 {
		t.Errorf("after one refusal: %d asks, waits %v; want 2 asks after one %v", asks, waits, firstBackOff)
	}
	if body, err := os.ReadFile(spotsFile(dir, day[0])); err != nil || string(body) != "row\n" {
		t.Errorf("the day's spots kept: %q, %v", body, err)
	}
	if !strings.Contains(log.String(), "Retry-After=60") || strings.Contains(log.String(), "cache-node") {
		t.Errorf("the log keeps more than the rate headers, or not them:\n%s", log.String())
	}
	if err := f.fetchDays(day, dir, defaultRegionKm, &bytes.Buffer{}); err != nil || asks != 2 {
		t.Errorf("a day kept was asked again: %d asks, %v", asks, err)
	}
}

// TestTheFetchGivesUpAfterTheLongestPause is the back-off's end: doubling
// from 60 s, never a pause past 15 minutes, then the day given up; and a
// client error is never asked again.
func TestTheFetchGivesUpAfterTheLongestPause(t *testing.T) {
	status := http.StatusServiceUnavailable
	asks := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asks++
		w.WriteHeader(status)
	}))
	defer srv.Close()
	var waits []time.Duration
	f := fetcher{base: srv.URL + "/", client: srv.Client(), sleep: func(d time.Duration) { waits = append(waits, d) }, log: &bytes.Buffer{}}
	day := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	if _, err := f.fetchDay(day, defaultRegionKm); !errors.Is(err, errGivenUp) {
		t.Fatalf("always unavailable: %v", err)
	}
	want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute}
	if len(waits) != len(want) || waits[3] != want[3] || waits[0] != want[0] {
		t.Errorf("the pauses: %v; want %v", waits, want)
	}
	status, asks, waits = http.StatusBadRequest, 0, nil
	if _, err := f.fetchDay(day, defaultRegionKm); err == nil || asks != 1 || len(waits) != 0 {
		t.Errorf("a client error: %d asks, waits %v, %v; want one ask, no wait, an error", asks, waits, err)
	}
}

// TestASpotRowIsReadOrRefused is the rows read back: the query's shape and
// bounds, or an error naming the line.
func TestASpotRowIsReadOrRefused(t *testing.T) {
	good := "EM12\t32.5\t-97\tEM13ab\t33.04\t-96.96\t7\t2026-09-29 00:00:00\t12\t14.5"
	got, err := readSpots(strings.NewReader(good + "\n"))
	if err != nil || len(got) != 1 {
		t.Fatalf("a good row: %v, %v", got, err)
	}
	s := got[0]
	if s.Tx != "EM12" || s.Band != ionomaps.Band40m || s.N != 12 || s.Best != 14.5 || s.RxAt.Lat != 33.04 || s.Hour.Hour() != 0 {
		t.Errorf("the row read: %+v", s)
	}
	for _, bad := range []string{
		strings.Replace(good, "\t7\t", "\t50\t", 1), // 6 m: not answered
		strings.Replace(good, "32.5", "95", 1),      // off the globe
		strings.Replace(good, "\t12\t", "\t0\t", 1), // no spot
		strings.Replace(good, "\t14.5", "", 1),      // a field short
		strings.Replace(good, "EM12\t", "\t", 1),    // no locator
	} {
		if _, err := readSpots(strings.NewReader(good + "\n" + bad + "\n")); err == nil || !strings.Contains(err.Error(), "line 2") {
			t.Errorf("%q read without an error naming its line: %v", bad, err)
		}
	}
}
