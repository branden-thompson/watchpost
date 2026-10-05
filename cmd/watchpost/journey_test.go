//go:build darwin || linux

package main

// journey_test.go — THE PTY JOURNEY ON THE REAL BINARY (D-271, mitigating RK-1).
//
// RK-1 is "correct parts, wrongly connected": every unit passes and the station
// still does the wrong thing, because the key, the window and the data are each
// right and are not joined. Only the built binary on a real terminal joins all
// of them, so this test builds it, runs it on a pseudo-terminal, presses the
// keys a listener presses and reads the screen after each one:
//
//	the dashboard, holding a recorded alert -> g opens the map on the place ->
//	A shows the alert in force there, A hides it -> → pans east -> + zooms in
//	-> - zooms out -> 2 shows Alaska -> O opens the Overlays menu -> esc
//	closes it -> esc closes the map -> q quits, exit 0, no panic
//
// RECORDED RESPONSES, NO NETWORK. The station's HTTP cache serves a fresh entry
// without asking the network, and an entry on disk warms a launch; the journey
// writes its recorded answers there, in the cache's own file format
// (platform/httpx/cache.go: a JSON header line, then the body, named by the
// URL's SHA-256). The alert is M1's recorded Oak Ridge Flash Flood Warning
// (app/testdata/maps/m1), its times moved to now so it is active. The map draws
// from the basemap embedded in the binary. Everything else is refused: every
// proxy variable names a local proxy that answers 403 and logs what was asked,
// and on macOS the binary runs under sandbox-exec with outbound connections
// denied except to loopback, which a probe proves before the journey relies on
// it. HOME and the XDG directories are temporary, PATH is an empty directory
// (no `say`, so nothing is spoken), and the window is 133x44.
//
// The journey waits on the SCREEN, never on a clock: each step polls the
// emulated screen (screen_test.go) until its condition holds, bounded by
// stepBound, and fails with the screen it last saw.
//
// It makes its own pseudo-terminal, so it needs no terminal to run in, and it
// takes a few seconds once the build is cached: it runs with the package's
// other tests under `make race`, locally and in CI. -short skips it.

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

const (
	journeyRows, journeyCols = 44, 133
	// stepBound is the longest one step may take to show its screen. It is a
	// bound, never an expectation: a step passes the moment its screen is up.
	stepBound = 30 * time.Second
	// The place the journey watches: M1 scenario 01's Oak Ridge, TN.
	journeyLat, journeyLon = 36.0104, -84.2696
	// noNetworkProfile denies every outbound connection but loopback's, where
	// the refusing proxy listens; DNS is a connection too, so names do not resolve.
	noNetworkProfile = `(version 1)(allow default)(deny network-outbound)(allow network-outbound (remote ip "localhost:*"))`
)

func TestJourneyMapOnTheRealBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("the PTY journey builds and runs the binary; -short skips it")
	}
	bin := buildJourneyBinary(t)
	home := journeyHome(t)
	proxy := refusingProxy(t)
	j := startJourney(t, bin, home, proxy)

	j.step("the dashboard shows the place and its recorded alert", "",
		has("WATCHPOST Observer"), has("L O C A T I O N"), has("Oak Ridge, TN"), has("FLASH FLOOD WARNING - Oak Ridge, TN"), lacks("Map · "))

	j.step("g opens the map on the place", "g",
		matches(`Map · [^·\n]+ · Oak Ridge, TN`), has("MAPS:"), has("O  Overlays"), scaleShown)

	j.step("A opens Area Alerts: the recorded warning is in force at the place", "A",
		has("┌─ Area Alerts"), has("Flash Flood Warning in effect for this"))

	atPlace := j.step("A closes Area Alerts", "A",
		lacks("┌─ Area Alerts"), matches(`Map · [^·\n]+ · Oak Ridge, TN`), scaleShown)
	scale0 := mapScale(atPlace)

	j.step("→ pans the map east: the picture moves west", "\x1b[C",
		has("Map · "), shiftedWestFrom(atPlace))

	j.step("+ zooms in: the scale bar shortens", "+",
		has("Map · "), scaleBelow(scale0))

	j.step("- zooms out: the scale bar is back", "-",
		has("Map · "), scaleEquals(scale0))

	j.step("2 switches the map to Alaska", "2",
		has("Map · Alaska"), scaleAbove(scale0))

	j.step("O opens the Overlays menu", "O",
		has("Map · Alaska"), has("MAP DETAILS / OVERLAYS"))

	j.step("esc closes the menu and leaves the map open", "\x1b",
		has("Map · Alaska"), lacks("MAP DETAILS / OVERLAYS"))

	j.step("esc closes the map", "\x1b",
		lacks("Map · "), has("L O C A T I O N"), has("Oak Ridge, TN"))

	j.quit()
	proxy.assertDiverted(t)
}

// ---- the binary and its world ---------------------------------------------

// buildJourneyBinary builds ./cmd/watchpost as the Makefile's build does, with
// -trimpath, into the test's temporary directory.
func buildJourneyBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "watchpost")
	if out, err := exec.Command("go", "build", "-trimpath", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("building watchpost: %v\n%s", err, out)
	}
	return bin
}

// journeyHome is a temporary HOME holding the config (one watched place) and
// the HTTP cache seeded with the recorded answers.
func journeyHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	cfg := "update_check = false\n\n[[locations]]\nlabel = 'Oak Ridge, TN'\ntag = 'OAKRG'\nzip = '37830'\n" +
		"lat = " + strconv.FormatFloat(journeyLat, 'f', 4, 64) + "\nlon = " + strconv.FormatFloat(journeyLon, 'f', 4, 64) +
		"\ntz = 'America/New_York'\n"
	writeFile(t, filepath.Join(home, "config", "watchpost", "config.toml"), []byte(cfg))
	seedRecordedAnswers(t, httpCacheDir(home))
	return home
}

// httpCacheDir is where the binary's data client keeps its disk cache:
// os.UserCacheDir()/watchpost/http under the journey's HOME and XDG_CACHE_HOME.
func httpCacheDir(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches", "watchpost", "http")
	}
	return filepath.Join(home, "cache", "watchpost", "http")
}

// seedRecordedAnswers writes the NWS answers the journey's place needs. The
// alert is recorded (M1 scenario 01). The repository holds no recording of
// Oak Ridge's /points or its stations, so those two are written here with
// only what routes the recorded alert: the county zone it names, and one
// station so the point resolves.
func seedRecordedAnswers(t *testing.T, dir string) {
	t.Helper()
	const api = "https://api.weather.gov"
	point := strconv.FormatFloat(journeyLat, 'f', 4, 64) + "," + strconv.FormatFloat(journeyLon, 'f', 4, 64)
	points := `{"properties":{"forecast":"` + api + `/gridpoints/MRX/1,1/forecast","forecastHourly":"` + api +
		`/gridpoints/MRX/1,1/forecast/hourly","forecastGridData":"` + api + `/gridpoints/MRX/1,1","observationStations":"` + api +
		`/gridpoints/MRX/1,1/stations","county":"` + api + `/zones/county/TNC001","timeZone":"America/New_York"}}`
	stations := `{"features":[{"geometry":{"coordinates":[-84.2696,36.0104]},"properties":{"stationIdentifier":"KOQT"}}]}`
	recorded, err := os.ReadFile(filepath.Join("..", "..", "app", "testdata", "maps", "m1", "01-covers-oak-ridge", "alerts.json"))
	if err != nil {
		t.Fatalf("reading M1's recorded Oak Ridge alert: %v", err)
	}
	expires := time.Now().Add(24 * time.Hour)
	for url, body := range map[string][]byte{
		api + "/points/" + point:                         []byte(points),
		api + "/gridpoints/MRX/1,1/stations?limit=4":     []byte(stations), // the station list as nws asks for it, stationCandidates at a time
		api + "/alerts/active?status=actual&zone=TNC001": activeNow(t, recorded),
	} {
		seedCacheEntry(t, dir, url, body, expires)
	}
}

// activeNow moves a recorded alert's times so it is in force while the
// journey runs: sent and effective an hour ago, ending in two hours.
func activeNow(t *testing.T, recorded []byte) []byte {
	t.Helper()
	var doc struct {
		Type     string           `json:"type"`
		Features []map[string]any `json:"features"`
	}
	if err := json.Unmarshal(recorded, &doc); err != nil || len(doc.Features) == 0 {
		t.Fatalf("M1's recorded alert does not decode as a feature collection: %v", err)
	}
	past, future := time.Now().Add(-time.Hour).Format(time.RFC3339), time.Now().Add(2*time.Hour).Format(time.RFC3339)
	for _, f := range doc.Features {
		props, ok := f["properties"].(map[string]any)
		if !ok {
			t.Fatal("a recorded feature has no properties")
		}
		for _, k := range []string{"sent", "effective", "onset"} {
			props[k] = past
		}
		for _, k := range []string{"expires", "ends"} {
			props[k] = future
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// seedCacheEntry writes one entry as platform/httpx's disk tier writes it.
func seedCacheEntry(t *testing.T, dir, url string, body []byte, expires time.Time) {
	t.Helper()
	hdr, err := json.Marshal(struct {
		URL     string    `json:"url"`
		Expires time.Time `json:"expires"`
	}{url, expires})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(url))
	writeFile(t, filepath.Join(dir, hex.EncodeToString(sum[:])+".cache"), append(append(hdr, '\n'), body...))
}

func writeFile(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// journeyEnv is the binary's whole environment: nothing is inherited.
func journeyEnv(home, proxyURL, path string) []string {
	return []string{
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, "config"),
		"XDG_STATE_HOME=" + filepath.Join(home, "state"),
		"XDG_CACHE_HOME=" + filepath.Join(home, "cache"),
		"XDG_DATA_HOME=" + filepath.Join(home, "data"),
		"TERM=xterm-256color",
		"LANG=en_US.UTF-8",
		"PATH=" + path,
		"HTTPS_PROXY=" + proxyURL, "https_proxy=" + proxyURL,
		"HTTP_PROXY=" + proxyURL, "http_proxy=" + proxyURL,
		"ALL_PROXY=" + proxyURL, "all_proxy=" + proxyURL,
		"NO_PROXY=", "no_proxy=",
	}
}

// ---- the refusing proxy -----------------------------------------------------

// proxyLog is a local proxy that refuses every request with a 403 and records
// the request line, so the journey can say what the binary asked for.
type proxyLog struct {
	url   string
	mu    sync.Mutex
	asked []string
}

func refusingProxy(t *testing.T) *proxyLog {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() }) // the listener's close ends the accept loop
	pl := &proxyLog{url: "http://" + ln.Addr().String()}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go pl.refuse(c)
		}
	}()
	return pl
}

func (pl *proxyLog) refuse(c net.Conn) {
	defer func() { _ = c.Close() }() // a refused connection has nothing left to say
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	line, _ := bufio.NewReader(c).ReadString('\n') // a short or empty line is still a request to refuse
	pl.mu.Lock()
	pl.asked = append(pl.asked, strings.TrimSpace(line))
	pl.mu.Unlock()
	_, _ = c.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")) // the peer may already be gone
}

// assertDiverted holds that the binary's requests came here, where each was
// refused: a station that asked the proxy nothing sent its requests some
// other way. It logs what was asked, by host.
func (pl *proxyLog) assertDiverted(t *testing.T) {
	t.Helper()
	pl.mu.Lock()
	defer pl.mu.Unlock()
	if len(pl.asked) == 0 {
		t.Error("the station asked the refusing proxy nothing, so its requests did not go through it")
	}
	hosts := map[string]int{}
	for _, l := range pl.asked {
		if f := strings.Fields(l); len(f) >= 2 {
			hosts[f[1]]++
		}
	}
	t.Logf("the proxy refused %d requests: %v", len(pl.asked), hosts)
}

// ---- the sandbox ------------------------------------------------------------

// withNoNetwork wraps argv in sandbox-exec where macOS has it, after proving
// the profile refuses a dial. Where there is no sandbox-exec, or it cannot
// apply a profile here, the proxy is the only guard and the test says so; a
// sandbox that applies and lets the dial through fails the test.
func withNoNetwork(t *testing.T, argv []string) []string {
	t.Helper()
	sb, err := exec.LookPath("/usr/bin/sandbox-exec")
	if runtime.GOOS != "darwin" || err != nil {
		t.Log("no sandbox-exec here: the refusing proxy is the journey's only network guard")
		return argv
	}
	probe := exec.Command(sb, "-p", noNetworkProfile, os.Args[0], "-test.run=^TestJourneyDialProbe$", "-test.count=1", "-test.v")
	probe.Env = []string{"JOURNEY_DIAL_PROBE=1"}
	out, err := probe.CombinedOutput()
	switch {
	case strings.Contains(string(out), "dial refused:"):
		t.Log("the station runs under sandbox-exec: the probe's dial outside loopback was refused")
		return append([]string{sb, "-p", noNetworkProfile}, argv...)
	case strings.Contains(string(out), "--- FAIL: TestJourneyDialProbe"):
		t.Fatalf("the no-network sandbox let the probe's dial through, so it cannot be relied on:\n%s", out)
	}
	t.Logf("sandbox-exec could not apply its profile here (%v): the refusing proxy is the journey's only network guard\n%s", err, out)
	return argv
}

// TestJourneyDialProbe is the sandbox's positive control: run inside the
// journey's sandbox, it dials an address outside loopback and must be refused
// by the sandbox (EPERM), before any packet is sent. Outside the sandbox it
// skips.
func TestJourneyDialProbe(t *testing.T) {
	if os.Getenv("JOURNEY_DIAL_PROBE") == "" {
		t.Skip("run by the PTY journey inside its sandbox")
	}
	c, err := net.DialTimeout("tcp", "192.0.2.1:80", 2*time.Second) // TEST-NET-1: never a real host
	if err == nil {
		_ = c.Close()
		t.Fatal("dial allowed")
	}
	if !errors.Is(err, syscall.EPERM) {
		t.Fatalf("the dial failed but was not refused by the sandbox: %v", err)
	}
	t.Log("dial refused: " + err.Error())
}

// ---- the journey ------------------------------------------------------------

// journey is one running station on a pseudo-terminal: done carries its exit,
// and exited is set once quit has received it.
type journey struct {
	t       *testing.T
	pty     *os.File
	term    *ptyScreen
	done    chan error
	exited  bool
	stepNum int
}

func startJourney(t *testing.T, bin, home string, proxy *proxyLog) *journey {
	t.Helper()
	return startJourneySized(t, bin, home, proxy, journeyRows, journeyCols)
}

// startJourneySized starts the station on a rows x cols pseudo-terminal, its
// environment the journey's with extraEnv added.
func startJourneySized(t *testing.T, bin, home string, proxy *proxyLog, rows, cols int, extraEnv ...string) *journey {
	t.Helper()
	primary, secondary, err := openPTY()
	if err != nil {
		t.Fatalf("opening a pseudo-terminal: %v", err)
	}
	setWinsize(t, primary, rows, cols)
	argv := withNoNetwork(t, []string{bin})
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(journeyEnv(home, proxy.url, t.TempDir()), extraEnv...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = secondary, secondary, secondary
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting watchpost: %v", err)
	}
	_ = secondary.Close() // the child holds its own copy; ours would keep the primary from seeing the end
	j := &journey{t: t, pty: primary, term: &ptyScreen{scr: newScreen(rows, cols)}, done: make(chan error, 1)}
	go j.term.pump(primary)
	go func() { j.done <- cmd.Wait() }()
	t.Cleanup(func() {
		if !j.exited {
			_ = cmd.Process.Kill() // a journey that failed midway leaves no station running
			<-j.done
		}
		_ = primary.Close() // the pump ends on the closed primary
	})
	return j
}

// check is one condition on the screen; "" means it holds, else what is wrong.
type check func(screen string) string

func has(s string) check {
	return func(scr string) string {
		if strings.Contains(scr, s) {
			return ""
		}
		return "the screen does not show " + strconv.Quote(s)
	}
}

func lacks(s string) check {
	return func(scr string) string {
		if !strings.Contains(scr, s) {
			return ""
		}
		return "the screen still shows " + strconv.Quote(s)
	}
}

func matches(expr string) check {
	re := regexp.MustCompile(expr)
	return func(scr string) string {
		if re.MatchString(scr) {
			return ""
		}
		return "the screen has nothing matching " + strconv.Quote(expr)
	}
}

// step sends keys, then waits until every check holds on the screen, and
// returns that screen. A step that never settles fails the test with the last
// screen and the reasons.
func (j *journey) step(name, keys string, checks ...check) string {
	j.t.Helper()
	j.stepNum++
	if keys != "" {
		if _, err := j.pty.Write([]byte(keys)); err != nil {
			j.t.Fatalf("step %d (%s): sending %q: %v", j.stepNum, name, keys, err)
		}
	}
	deadline := time.Now().Add(stepBound)
	for {
		scr := j.term.snapshot()
		var why []string
		for _, c := range checks {
			if w := c(scr); w != "" {
				why = append(why, w)
			}
		}
		if len(why) == 0 {
			j.t.Logf("step %d passed: %s", j.stepNum, name)
			return scr
		}
		select {
		case err := <-j.done:
			j.done <- err
			j.t.Fatalf("step %d (%s): watchpost exited (%v) before the step's screen:\n- %s\n%s", j.stepNum, name, err, strings.Join(why, "\n- "), scr)
		default:
		}
		if time.Now().After(deadline) {
			j.t.Fatalf("step %d (%s) never reached its screen:\n- %s\n--- the last screen ---\n%s", j.stepNum, name, strings.Join(why, "\n- "), scr)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// quit presses q and asserts the process ends by itself, with code 0 and no
// panic in anything it wrote.
func (j *journey) quit() {
	j.t.Helper()
	j.stepNum++
	if _, err := j.pty.Write([]byte("q")); err != nil {
		j.t.Fatalf("step %d (q quits): %v", j.stepNum, err)
	}
	select {
	case err := <-j.done:
		j.exited = true
		if err != nil {
			j.t.Errorf("step %d (q quits): watchpost exited with %v, want code 0\n%s", j.stepNum, err, j.term.snapshot())
			return
		}
	case <-time.After(stepBound):
		j.t.Fatalf("step %d (q quits): watchpost was still running %v after q\n%s", j.stepNum, stepBound, j.term.snapshot())
	}
	if raw := j.term.rawText(); strings.Contains(raw, "panic:") || strings.Contains(raw, "goroutine ") {
		j.t.Errorf("step %d (q quits): watchpost wrote a panic:\n%s", j.stepNum, raw[max(strings.Index(raw, "panic:"), 0):])
		return
	}
	j.t.Logf("step %d passed: q quits, code 0, no panic", j.stepNum)
}

// ptyScreen feeds a screen from the PTY's primary end and keeps the raw
// stream for the panic check.
type ptyScreen struct {
	mu  sync.Mutex
	scr *screen
	raw []byte
}

func (ps *ptyScreen) pump(p *os.File) {
	buf := make([]byte, 64<<10)
	for {
		n, err := p.Read(buf)
		ps.mu.Lock()
		ps.scr.write(buf[:n])
		ps.raw = append(ps.raw, buf[:n]...)
		ps.mu.Unlock()
		if err != nil {
			return // the child closed its end (EIO on Linux, EOF on macOS) or the test closed ours
		}
	}
}

func (ps *ptyScreen) snapshot() string {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.scr.text()
}

func (ps *ptyScreen) rawText() string {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return string(ps.raw)
}

func setWinsize(t *testing.T, f *os.File, rows, cols int) {
	t.Helper()
	ws := struct{ Row, Col, X, Y uint16 }{uint16(rows), uint16(cols), 0, 0}
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws))); e != 0 {
		t.Fatalf("sizing the pseudo-terminal: %v", e)
	}
}

// failPTY closes the half-made pair's primary and reports why it failed.
func failPTY(p *os.File, err error) (*os.File, *os.File, error) {
	_ = p.Close() // the pair failed; err is the one to report
	return nil, nil, err
}

// ---- reading the map --------------------------------------------------------

// scaleRE is the map's scale bar: a ruler, then its length.
var scaleRE = regexp.MustCompile(`├─*┤ ([0-9][0-9,.]*) (km|mi|m|ft)\b`)

// mapScale is the scale bar's length in metres, or 0 when none is shown.
func mapScale(scr string) float64 {
	m := scaleRE.FindStringSubmatch(scr)
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
	if err != nil {
		return 0
	}
	return v * map[string]float64{"km": 1000, "mi": 1609.344, "m": 1, "ft": 0.3048}[m[2]]
}

func scaleShown(scr string) string {
	if mapScale(scr) > 0 {
		return ""
	}
	return "the map shows no scale bar"
}

func scaleBelow(ref float64) check {
	return func(scr string) string {
		if s := mapScale(scr); s > 0 && s < ref {
			return ""
		}
		return "the scale bar is not shorter than " + strconv.FormatFloat(ref, 'f', 0, 64) + " m"
	}
}

func scaleAbove(ref float64) check {
	return func(scr string) string {
		if mapScale(scr) > ref {
			return ""
		}
		return "the scale bar is not longer than " + strconv.FormatFloat(ref, 'f', 0, 64) + " m"
	}
}

func scaleEquals(ref float64) check {
	return func(scr string) string {
		if mapScale(scr) == ref {
			return ""
		}
		return "the scale bar is not back at " + strconv.FormatFloat(ref, 'f', 0, 64) + " m"
	}
}

// brailleRows are the map picture's rows: each screen row's span from its
// first braille cell to its last, in order.
func brailleRows(scr string) [][]rune {
	var rows [][]rune
	for _, line := range strings.Split(scr, "\n") {
		r := []rune(line)
		first, last := -1, -1
		for i, c := range r {
			if c >= 0x2800 && c <= 0x28ff {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		if first >= 0 && last-first > journeyCols/4 {
			rows = append(rows, r[first:last+1])
		}
	}
	return rows
}

// shiftedWestFrom holds when the picture is the earlier one moved west: for
// some offset k > 0, nine in ten of the drawn cells now at column c were at
// column c+k before.
func shiftedWestFrom(before string) check {
	was := brailleRows(before)
	return func(scr string) string {
		now := brailleRows(scr)
		if len(now) != len(was) || len(now) == 0 {
			return "the map's picture is not on screen as before"
		}
		width := len(now[0])
		for k := 1; k < width/2; k++ {
			match, drawn := 0, 0
			for r := range now {
				for c := 0; c+k < len(was[r]) && c < len(now[r]); c++ {
					if now[r][c] == '⠀' {
						continue
					}
					drawn++
					if now[r][c] == was[r][c+k] {
						match++
					}
				}
			}
			if drawn >= 20 && match*10 >= drawn*9 {
				return ""
			}
		}
		return "the map's picture has not moved west"
	}
}
