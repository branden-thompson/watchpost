//go:build darwin || linux

package main

// measure_test.go — M5 AND M6 MEASURED BY THEIR PROTOCOL (D-46, D-53, D-251),
// on the real binary, the journey's recorded responses and its no-network
// guards (journey_test.go).
//
//	WATCHPOST_VALIDATE_M5=1 go test -run '^TestMeasureM5M6$' -count=1 -v -timeout 30m ./cmd/watchpost
//
// It runs only under WATCHPOST_VALIDATE_M5=1, so neither `make verify` nor CI
// ever runs it: it takes minutes, and its numbers are read, never asserted.
//
// M5: twenty cold opens at 149x38. Each open is a new process under a new
// HOME holding only the config and the recorded answers - the journey's, and
// the recorded alert again as the map's "Alerts in view" ask - so the zone
// and map caches are empty. Once the dashboard shows the recorded alert, g is
// pressed and the open's events are read from the loopback debug server's
// /debug/counters for up to m5Bound: m5 - the dashboard's own interval from g
// to the first complete frame holding every alert's area
// (modes/tty/timing.go) - and the others the open said on the way. The
// basemap's tiles at the place and the radar are fetched by clients that
// read nothing from the HTTP cache, so on recorded responses they are refused
// like everything else; an open whose basemap never completes has no m5, and
// is counted as missing with the events it did say.
//
// M6: after the last open, space is pressed with the Observer live for
// WATCHPOST_VALIDATE_M6_SECS seconds (60 by default), with → and ← pressed in
// turn every ten seconds; the map's ticks, the keys' time to their frames and
// the hundred-frame times are read from the counters, polled every five
// seconds and joined by sequence number. With no radar answered there is no
// loop to play here: modes/tty/m6_measure_test.go plays one in process.
//
// The results go to the test log and, when WATCHPOST_VALIDATE_OUT names a
// file, to that file as JSON: the machine, OS, Go version, the date, the load
// average before and after, every sample and the percentiles.

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	measureRows, measureCols = 38, 149 // M5's size (D-46)
	measureOpens             = 20      // D-46's n; WATCHPOST_VALIDATE_OPENS sets another for a trial run
	// m5Bound is the longest an open is waited on for its m5; an open that
	// never reaches it is reported as such, not measured.
	m5Bound = 30 * time.Second
)

// timingRow is one interval as /debug/counters says it (app/timing.go).
type timingRow struct {
	Seq     uint64    `json:"seq"`
	At      time.Time `json:"at"`
	Trigger string    `json:"trigger"`
	Event   string    `json:"event"`
	MS      float64   `json:"ms"`
}

// spread is a distribution's summary, in milliseconds.
type spread struct {
	N   int     `json:"n"`
	P50 float64 `json:"p50_ms"`
	P90 float64 `json:"p90_ms"`
	P99 float64 `json:"p99_ms"`
	Max float64 `json:"max_ms"`
}

// openRecord is one cold open: the first time, from g, of each event the
// open said (answered:feed, answered:radar, complete, m5, settled...), in
// milliseconds.
type openRecord struct {
	Events map[string]float64 `json:"events_ms"`
}

// measurement is what one run took and found.
type measurement struct {
	Date        string               `json:"date"`
	Machine     string               `json:"machine"`
	OS          string               `json:"os"`
	Go          string               `json:"go"`
	LoadBefore  string               `json:"load_before"`
	LoadAfter   string               `json:"load_after"`
	Size        string               `json:"size"`
	Opens       []openRecord         `json:"opens"`
	M5MS        []float64            `json:"m5_ms"`
	M5Missing   int                  `json:"m5_missing"`
	M5          spread               `json:"m5"`
	M6Secs      int                  `json:"m6_secs"`
	M6Keys      []string             `json:"m6_keys"`
	M6          map[string]spread    `json:"m6"`
	M6Samples   map[string][]float64 `json:"m6_samples"`
	ProxyAsked  map[string]int       `json:"proxy_asked"`
	MapProblems []string             `json:"map_problems"`
}

func TestMeasureM5M6(t *testing.T) {
	if os.Getenv("WATCHPOST_VALIDATE_M5") != "1" {
		t.Skip("measures M5 and M6 on the real binary for minutes; WATCHPOST_VALIDATE_M5=1 runs it")
	}
	m6Secs := 60
	if v, err := strconv.Atoi(os.Getenv("WATCHPOST_VALIDATE_M6_SECS")); err == nil && v > 0 {
		m6Secs = v
	}
	opens := measureOpens
	if v, err := strconv.Atoi(os.Getenv("WATCHPOST_VALIDATE_OPENS")); err == nil && v > 0 {
		opens = v
	}
	bin := buildJourneyBinary(t)
	proxy := refusingProxy(t)
	res := measurement{Date: time.Now().UTC().Format(time.RFC3339), Machine: machine(), OS: osVersion(), Go: runtime.Version(),
		LoadBefore: loadAverage(), Size: fmt.Sprintf("%dx%d", measureCols, measureRows), M6Secs: m6Secs}

	var last *measuredStation
	for i := 1; i <= opens; i++ {
		st := startMeasured(t, bin, proxy)
		rec := st.coldOpen()
		res.Opens = append(res.Opens, rec)
		if ms, ok := rec.Events["m5"]; ok {
			res.M5MS = append(res.M5MS, ms)
			t.Logf("open %2d: m5 %.0f ms; the open said %v", i, ms, rec.Events)
		} else {
			res.M5Missing++
			t.Logf("open %2d: no m5 within %v; the open said %v; the map's problems: %v", i, m5Bound, rec.Events, st.counters().MapProblems)
		}
		if i < opens {
			st.j.quit()
			continue
		}
		last = st
	}
	res.M5 = spreadOf(res.M5MS)

	res.M6Keys, res.M6Samples = last.playLoop(time.Duration(m6Secs) * time.Second)
	res.M6 = map[string]spread{}
	for k, v := range res.M6Samples {
		res.M6[k] = spreadOf(v)
	}
	res.MapProblems = last.counters().MapProblems
	last.j.quit()
	res.LoadAfter = loadAverage()
	res.ProxyAsked = proxy.byHost()

	out, err := json.MarshalIndent(res, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("M5 %+v (missing %d)", res.M5, res.M5Missing)
	for k, s := range res.M6 {
		t.Logf("M6 %s %+v", k, s)
	}
	t.Logf("load before %s, after %s; %s; %s; %s", res.LoadBefore, res.LoadAfter, res.Machine, res.OS, res.Go)
	if path := os.Getenv("WATCHPOST_VALIDATE_OUT"); path != "" {
		writeFile(t, path, out)
		t.Logf("wrote %s", path)
	}
}

// measuredStation is one station with the timing instrument and its debug
// server on, and the address the counters are read from.
type measuredStation struct {
	t        *testing.T
	j        *journey
	counters func() countersDoc
}

// countersDoc is the part of /debug/counters the measurement reads.
type countersDoc struct {
	MapProblems []string    `json:"map_problems"`
	Timings     []timingRow `json:"timings"`
}

// startMeasured starts a station under a new HOME holding the recorded
// answers, at 149x38, with the instrument and the debug server on a free
// loopback port.
func startMeasured(t *testing.T, bin string, proxy *proxyLog) *measuredStation {
	t.Helper()
	port := freeLoopbackPort(t)
	home := journeyHome(t)
	seedViewAlerts(t, httpCacheDir(home))
	j := startJourneySized(t, bin, home, proxy, measureRows, measureCols,
		"WATCHPOST_DEBUG_TIMING=1", "WATCHPOST_DEBUG_PPROF=1", "WATCHPOST_DEBUG_PPROF_ADDR="+port)
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: nil}}
	url := "http://127.0.0.1:" + port + "/debug/counters"
	st := &measuredStation{t: t, j: j}
	st.counters = func() countersDoc {
		var doc countersDoc
		resp, err := client.Get(url)
		if err != nil {
			return doc
		}
		defer func() { _ = resp.Body.Close() }()    // a read-only answer
		_ = json.NewDecoder(resp.Body).Decode(&doc) // a short answer reads as no timings, and the poll asks again
		return doc
	}
	return st
}

// viewAlertsURL is the map's "Alerts in view" ask at the place at 149x38:
// the states its view touches (app/mapinview.go), North Carolina and
// Tennessee. A view that touched others would ask another address, find no
// answer and never reach m5, which the run reports as a missing open.
const viewAlertsURL = "https://api.weather.gov/alerts/active?status=actual&area=NC,TN"

// seedViewAlerts writes the recorded alert (M1 scenario 01, made active now)
// as the answer to the map's ask for the alerts in view.
func seedViewAlerts(t *testing.T, dir string) {
	t.Helper()
	recorded, err := os.ReadFile(filepath.Join("..", "..", "app", "testdata", "maps", "m1", "01-covers-oak-ridge", "alerts.json"))
	if err != nil {
		t.Fatalf("reading M1's recorded Oak Ridge alert: %v", err)
	}
	seedCacheEntry(t, dir, viewAlertsURL, activeNow(t, recorded), time.Now().Add(24*time.Hour))
}

// coldOpen waits for the dashboard to show the recorded alert, presses g and
// waits up to m5Bound for the open's m5, keeping each event the open said.
func (st *measuredStation) coldOpen() openRecord {
	st.j.step("the dashboard shows the place and its recorded alert", "",
		has("WATCHPOST Observer"), has("FLASH FLOOD WARNING - Oak Ridge, TN"), lacks("Map · "))
	st.j.step("g opens the map on the place", "g", matches(`Map · [^·\n]+ · Oak Ridge, TN`))
	rec := openRecord{Events: map[string]float64{}}
	for deadline := time.Now().Add(m5Bound); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		for _, r := range st.counters().Timings {
			if _, said := rec.Events[r.Event]; r.Trigger == "open" && !said {
				rec.Events[r.Event] = r.MS
			}
		}
		if _, ok := rec.Events["m5"]; ok {
			break
		}
	}
	return rec
}

// playLoop presses space and keeps the loop playing for d, pressing → and ←
// in turn every ten seconds, then presses space again. It returns the keys
// pressed during play and the samples, by event: "tick-late" (how late each map
// tick landed against the moment the library asked for), "tick-interval"
// (the time between delivered ticks), "key" (a key to its frame) and
// "frames100" (a hundred frames' time).
func (st *measuredStation) playLoop(d time.Duration) ([]string, map[string][]float64) {
	seen := map[uint64]bool{}
	var rows []timingRow
	take := func() {
		for _, r := range st.counters().Timings {
			if !seen[r.Seq] {
				seen[r.Seq] = true
				rows = append(rows, r)
			}
		}
	}
	take()
	var firstSeq uint64
	for s := range seen {
		firstSeq = max(firstSeq, s)
	}
	keys := []string{" "}
	if _, err := st.j.pty.Write([]byte(" ")); err != nil {
		st.t.Fatalf("pressing space: %v", err)
	}
	start := time.Now()
	nextKey, poll := start.Add(10*time.Second), start.Add(5*time.Second)
	pans := []string{"\x1b[C", "\x1b[D"}
	for n := 0; time.Since(start) < d; {
		now := time.Now()
		if now.After(nextKey) {
			if _, err := st.j.pty.Write([]byte(pans[n%2])); err != nil {
				st.t.Fatalf("pressing a pan: %v", err)
			}
			keys = append(keys, []string{"→", "←"}[n%2])
			n++
			nextKey = nextKey.Add(10 * time.Second)
		}
		if now.After(poll) {
			take()
			poll = poll.Add(5 * time.Second)
		}
		time.Sleep(50 * time.Millisecond)
	}
	take()
	if _, err := st.j.pty.Write([]byte(" ")); err != nil {
		st.t.Fatalf("pressing space: %v", err)
	}
	keys = append(keys, " ")
	slices.SortFunc(rows, func(a, b timingRow) int { return int(a.Seq) - int(b.Seq) })
	out := map[string][]float64{}
	var lastTick time.Time
	for _, r := range rows {
		if r.Seq <= firstSeq {
			continue
		}
		switch r.Event {
		case "key", "frames100":
			out[r.Event] = append(out[r.Event], r.MS)
		case "tick":
			out["tick-late"] = append(out["tick-late"], r.MS)
			if !lastTick.IsZero() {
				out["tick-interval"] = append(out["tick-interval"], float64(r.At.Sub(lastTick).Microseconds())/1000)
			}
			lastTick = r.At
		}
	}
	return keys, out
}

// spreadOf is xs's p50, p90, p99 and max by nearest rank.
func spreadOf(xs []float64) spread {
	if len(xs) == 0 {
		return spread{}
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	rank := func(p float64) float64 { return s[max(int(math.Ceil(p*float64(len(s))))-1, 0)] }
	return spread{N: len(s), P50: rank(0.50), P90: rank(0.90), P99: rank(0.99), Max: s[len(s)-1]}
}

// byHost is how many requests the proxy refused, by the host each asked for.
func (pl *proxyLog) byHost() map[string]int {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	hosts := map[string]int{}
	for _, l := range pl.asked {
		if f := strings.Fields(l); len(f) >= 2 {
			hosts[f[1]]++
		}
	}
	return hosts
}

// freeLoopbackPort is a loopback port free at the moment of asking.
func freeLoopbackPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	_ = ln.Close() // the station binds it next
	return port
}

// machine, osVersion and loadAverage describe where the measurement ran.
func machine() string {
	if runtime.GOOS == "darwin" {
		return strings.TrimSpace(cmdOut("sysctl", "-n", "hw.model")) + ", " + strings.TrimSpace(cmdOut("sysctl", "-n", "machdep.cpu.brand_string")) +
			", " + strings.TrimSpace(cmdOut("sysctl", "-n", "hw.ncpu")) + " CPUs, " + memGiB(cmdOut("sysctl", "-n", "hw.memsize"))
	}
	return strings.TrimSpace(cmdOut("uname", "-m")) + ", " + strconv.Itoa(runtime.NumCPU()) + " CPUs"
}

func osVersion() string {
	if runtime.GOOS == "darwin" {
		return "macOS " + strings.TrimSpace(cmdOut("sw_vers", "-productVersion")) + " (" + strings.TrimSpace(cmdOut("uname", "-r")) + ")"
	}
	return strings.TrimSpace(cmdOut("uname", "-sr"))
}

func loadAverage() string {
	if runtime.GOOS == "darwin" {
		return strings.Trim(strings.TrimSpace(cmdOut("sysctl", "-n", "vm.loadavg")), "{ }")
	}
	b, _ := os.ReadFile("/proc/loadavg") // empty where it cannot be read
	return strings.Join(strings.Fields(string(b))[:min(3, len(strings.Fields(string(b))))], " ")
}

func memGiB(bytes string) string {
	n, err := strconv.ParseFloat(strings.TrimSpace(bytes), 64)
	if err != nil {
		return "memory unknown"
	}
	return strconv.FormatFloat(n/(1<<30), 'f', 0, 64) + " GiB"
}

// cmdOut is a command's output, or "" when it fails.
func cmdOut(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}
