package propagation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// recorder is the propagation client in a test: every address asked, and an
// answer for each.
type recorder struct {
	mu     sync.Mutex
	asked  []string
	answer func(url string) (httpx.Answer, error)
	hold   chan struct{} // when set, every ask waits for it
}

func (r *recorder) Get(ctx context.Context, url, etag, lastModified string) (httpx.Answer, error) {
	r.mu.Lock()
	r.asked = append(r.asked, url)
	hold, answer := r.hold, r.answer
	r.mu.Unlock()
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return httpx.Answer{}, ctx.Err()
		}
	}
	if answer == nil {
		return httpx.Answer{Status: 404}, nil
	}
	return answer(url)
}

func (r *recorder) count() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.asked) }

// TestTheClientRefusesAnUnlistedHost is 0.19.0 FR-4.7 (D-113): an address on
// a host go-ionomaps does not export is refused before the client is asked.
func TestTheClientRefusesAnUnlistedHost(t *testing.T) {
	rec := &recorder{}
	f, err := fetcherOf(rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Fetch(context.Background(), "https://elsewhere.example/data", ionomaps.Validators{}); err == nil {
		t.Error("an unlisted host was asked")
	}
	if _, err := f.Fetch(context.Background(), "http://services.swpc.noaa.gov/text/x.txt", ionomaps.Validators{}); err == nil {
		t.Error("plain http was asked")
	}
	withCredentials := (&url.URL{Scheme: "https", User: url.UserPassword("u", "p"), Host: "services.swpc.noaa.gov", Path: "/text/x.txt"}).String()
	for _, u := range []string{"https://services.swpc.noaa.gov:8443/text/x.txt", withCredentials} {
		if _, err := f.Fetch(context.Background(), u, ionomaps.Validators{}); err == nil {
			t.Errorf("a listed host on another port, or with credentials, was asked (I-6): %s", u)
		}
	}
	if rec.count() != 0 {
		t.Errorf("the client was asked %d times", rec.count())
	}
	rec.answer = func(string) (httpx.Answer, error) { return httpx.Answer{}, nil }
	if _, err := f.Fetch(context.Background(), "https://services.swpc.noaa.gov/text/x.txt", ionomaps.Validators{}); !errors.Is(err, errNoStatus) {
		t.Errorf("a reply with no status: %v", err)
	}
	rec.answer = nil
	for _, u := range []string{"https://services.swpc.noaa.gov/text/x.txt", "https://services.swpc.noaa.gov:443/text/x.txt"} {
		if _, err := f.Fetch(context.Background(), u, ionomaps.Validators{}); err != nil {
			t.Errorf("a listed host was refused: %v", err)
		}
	}
}

// TestOnlyRateHeadersAreRead is FR-4.5 (D-83): of a reply's headers only the
// rate-related ones and the validators reach the library, and each rate
// header's first sighting is noted by host and name, never its value.
func TestOnlyRateHeadersAreRead(t *testing.T) {
	h := http.Header{}
	for k, v := range map[string]string{"Retry-After": "60", "X-RateLimit-Limit": "90", "X-RateLimit-Remaining": "3", "RateLimit-Reset": "30",
		"ETag": `"e"`, "Last-Modified": "Fri, 09 Oct 2026 12:00:00 GMT", "Set-Cookie": "secret", "X-Forwarded-For": "203.0.113.9"} {
		h.Set(k, v)
	}
	rec := &recorder{answer: func(string) (httpx.Answer, error) { return httpx.Answer{Status: 429, Header: h}, nil }}
	var notes []string
	f, err := fetcherOf(rec, func(s string) { notes = append(notes, s) })
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		r, err := f.Fetch(context.Background(), "https://lgdc.uml.edu/x", ionomaps.Validators{})
		if err != nil {
			t.Fatal(err)
		}
		want := ionomaps.RateHeaders{RetryAfter: "60", Limit: "90", Remaining: "3", Reset: "30"}
		if r.Status != 429 || r.Rate != want || r.Validators != (ionomaps.Validators{ETag: `"e"`, LastModified: "Fri, 09 Oct 2026 12:00:00 GMT"}) {
			t.Errorf("the library got %+v", r)
		}
	}
	text := strings.Join(notes, "\n")
	if len(notes) != 4 || !strings.Contains(text, "lgdc.uml.edu") || !strings.Contains(text, "Retry-After") {
		t.Errorf("notes %q; want each rate header's first sighting, by host and name", notes)
	}
	if strings.Contains(text, "60") || strings.Contains(text, "secret") || strings.Contains(text, "203.0.113.9") {
		t.Errorf("a note carries a value: %q", notes)
	}
}

// TestAnUnnamedStatusIsNotedOnce is D-272: a status D-247, D-260 and D-272
// do not name - a 418, a 451 - is noted in the diagnostics by host and
// status the first time it comes, never again; a named one - an answer, a
// 304, a 401, 403, 404, 410 or 429, a server's 5xx - never (R3L-5, R3C-3,
// R4D-3).
func TestAnUnnamedStatusIsNotedOnce(t *testing.T) {
	status := 418
	rec := &recorder{answer: func(string) (httpx.Answer, error) { return httpx.Answer{Status: status, Header: http.Header{}}, nil }}
	var notes []string
	f, err := fetcherOf(rec, func(s string) { notes = append(notes, s) })
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []int{418, 418, 451, 200, 304, 401, 403, 404, 410, 429, 500, 503, 451} {
		status = s
		if r, err := f.Fetch(context.Background(), "https://lgdc.uml.edu/x", ionomaps.Validators{}); err != nil || r.Status != s {
			t.Fatalf("status %d reached the library as %d, %v", s, r.Status, err)
		}
	}
	if len(notes) != 2 || !strings.Contains(notes[0], "lgdc.uml.edu answered 418") || !strings.Contains(notes[1], "answered 451") {
		t.Errorf("notes %q; want 418 and 451, each once, by host, and no named status", notes)
	}
}

// TestTwoUpdatesAskNOAAOnce is FR-4.8 (A-30): updates asked at once run one
// at a time in go-ionomaps - the second waits for the first and is answered
// from what it fetched, so no address of NOAA's is asked twice.
func TestTwoUpdatesAskNOAAOnce(t *testing.T) {
	rec := &recorder{hold: make(chan struct{})}
	s, err := NewService(rec, Options{Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	a, b := s.Ask(context.Background(), ionomaps.UpdateOptions{}), s.Ask(context.Background(), ionomaps.UpdateOptions{})
	close(rec.hold)
	<-a
	<-b
	asked := map[string]int{}
	for _, u := range rec.asked {
		asked[u]++
	}
	if len(asked) == 0 {
		t.Fatal("nothing was asked: this measures nothing")
	}
	for u, n := range asked {
		if n > 1 {
			t.Errorf("two updates asked at once asked %s %d times: %v", u, n, rec.asked)
		}
	}
}

// TestAReopenedModeGetsItsOwnUpdate is the BUILD red team's C-2: a mode left
// while its update is still ending, then opened again, is answered by an
// update of its own - never handed the left session's cancelled result,
// which the window would hide and not ask again until its next tick.
func TestAReopenedModeGetsItsOwnUpdate(t *testing.T) {
	slow := make(chan struct{})
	rec := &recorder{}
	first := true
	rec.answer = func(string) (httpx.Answer, error) {
		rec.mu.Lock()
		wait := first
		first = false
		rec.mu.Unlock()
		if wait {
			<-slow // the left session's update, still ending when the mode opens again
			return httpx.Answer{}, context.Canceled
		}
		return httpx.Answer{Status: 404}, nil
	}
	s, err := NewService(rec, Options{Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	left, leave := context.WithCancel(context.Background())
	gone := s.Ask(left, ionomaps.UpdateOptions{})
	leave()
	again := s.Ask(context.Background(), ionomaps.UpdateOptions{})
	close(slow)
	<-gone
	select {
	case r := <-again:
		if errors.Is(r.Err, context.Canceled) {
			t.Errorf("the reopened mode was handed the left session's cancelled update: %v", r.Err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the reopened mode's update never answered")
	}
}

// TestClosingTheModeCancelsTheUpdate is FR-4.2: the mode's context ends the
// update it started, and the result says so.
func TestClosingTheModeCancelsTheUpdate(t *testing.T) {
	rec := &recorder{hold: make(chan struct{})}
	s, err := NewService(rec, Options{Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	got := s.Ask(ctx, ionomaps.UpdateOptions{})
	cancel()
	select {
	case r := <-got:
		if r.Err == nil && len(r.Snapshot.Now.FoF2.Values) != 0 {
			t.Errorf("a cancelled update gave a field: %+v", r.Snapshot.Inputs)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a cancelled update never answered")
	}
}

// TestAnUpdateHasItsOwnDeadline is the architecture's 60 s: an update whose
// sources never answer ends at its deadline, whatever the caller's context.
func TestAnUpdateHasItsOwnDeadline(t *testing.T) {
	rec := &recorder{hold: make(chan struct{})}
	s, err := NewService(rec, Options{Clock: fixedClock, Deadline: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.Ask(context.Background(), ionomaps.UpdateOptions{}):
	case <-time.After(5 * time.Second):
		t.Fatal("an update past its deadline never answered")
	}
	if _, err := NewService(rec, Options{}); err != nil {
		t.Errorf("the default options were refused: %v", err)
	}
	if _, err := NewService(nil, Options{}); !errors.Is(err, errNoClient) {
		t.Errorf("no client: %v", err)
	}
	if _, err := NewService(rec, Options{Deadline: -time.Second}); !errors.Is(err, errNoDeadline) {
		t.Errorf("a negative deadline: %v", err)
	}
	if r := <-(&Service{}).Ask(context.Background(), ionomaps.UpdateOptions{}); !errors.Is(r.Err, errNoService) {
		t.Errorf("a service not made with New: %v", r.Err)
	}
}

func fixedClock() time.Time { return time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC) }

// fetcherOf is the fetcher a service is made with.
func fetcherOf(get Getter, note func(string)) (*fetcher, error) {
	s, err := NewService(get, Options{Note: note})
	if err != nil {
		return nil, err
	}
	return s.fetch, nil
}

// TestTheHostsOptionsReachTheLibrary is 0.19.0 D-187: the noise environment
// the window asks with is the update's, so the snapshot answers for it.
func TestTheHostsOptionsReachTheLibrary(t *testing.T) {
	var b strings.Builder
	for d := 30; d >= 1; d-- {
		day := fixedClock().AddDate(0, 0, -d)
		b.WriteString(day.Format("2006 01 02") + "  150  140  1  2  0  0  0  0  0  0  0\n")
	}
	rec := &recorder{answer: func(url string) (httpx.Answer, error) {
		if strings.HasSuffix(url, "daily-solar-indices.txt") {
			return httpx.Answer{Status: 200, Body: []byte(b.String())}, nil
		}
		return httpx.Answer{Status: 404}, nil
	}}
	s, err := NewService(rec, Options{Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	r := <-s.Ask(context.Background(), ionomaps.UpdateOptions{Noise: ionomaps.NoiseCity})
	if r.Err != nil || len(r.Snapshot.Now.FoF2.Values) == 0 || r.Snapshot.Noise != ionomaps.NoiseCity {
		t.Errorf("asked with city noise: %v, %d cells, noise %v", r.Err, len(r.Snapshot.Now.FoF2.Values), r.Snapshot.Noise)
	}
}

// TestNoRawSourceDataIsStored is FR-4.6 (IS-2, I-8): a GIRO reply carries the
// requester's address in its text; nothing of any reply - its text, or a
// rate header's value - reaches the diagnostics or an error, and the
// propagation client has no cache or directory to keep one in.
func TestNoRawSourceDataIsStored(t *testing.T) {
	const address = "203.0.113.7"
	body := []byte("# Requested by unknown guest from IP:" + address + "\nno reply this library reads\n")
	rec := &recorder{answer: func(string) (httpx.Answer, error) {
		return httpx.Answer{Status: 200, Body: body, Header: http.Header{"Retry-After": {address}}}, nil
	}}
	var notes []string
	s, err := NewService(rec, Options{Clock: fixedClock, Note: func(n string) { notes = append(notes, n) }})
	if err != nil {
		t.Fatal(err)
	}
	r := <-s.Ask(context.Background(), ionomaps.UpdateOptions{})
	if rec.count() == 0 || len(notes) == 0 {
		t.Fatalf("control: %d asks, %d notes; the check reads nothing", rec.count(), len(notes))
	}
	kept := append(notes, fmt.Sprint(r.Err))
	for _, k := range kept {
		if strings.Contains(k, address) || strings.Contains(k, "Requested by") {
			t.Errorf("a reply's text reached %q", k)
		}
	}
	plain := reflect.TypeOf(httpx.PlainConfig{})
	for i := range plain.NumField() {
		if name := strings.ToLower(plain.Field(i).Name); strings.Contains(name, "cache") || strings.Contains(name, "dir") {
			t.Errorf("the propagation client's config has %s: a reply could be kept", plain.Field(i).Name)
		}
	}
}

// TestTheThrottleStateOutlivesARestart is watchpost D-213: the service keeps
// go-ionomaps' throttle state in its file after each update, and a service
// made over that file - the next run - takes it; a file it cannot take is
// noted and the service starts as new, never failing to start.
func TestTheThrottleStateOutlivesARestart(t *testing.T) {
	var b strings.Builder
	for d := 30; d >= 1; d-- {
		b.WriteString(fixedClock().AddDate(0, 0, -d).Format("2006 01 02") + "  150  140  1  2  0  0  0  0  0  0  0\n")
	}
	rec := &recorder{answer: func(url string) (httpx.Answer, error) {
		if strings.HasSuffix(url, "daily-solar-indices.txt") {
			return httpx.Answer{Status: 200, Body: []byte(b.String())}, nil
		}
		return httpx.Answer{Status: 404}, nil
	}}
	file := filepath.Join(t.TempDir(), "state", "propagation-throttle.json")
	first, err := NewService(rec, Options{Clock: fixedClock, StateFile: file})
	if err != nil {
		t.Fatal(err)
	}
	if r := <-first.Ask(context.Background(), ionomaps.UpdateOptions{}); r.Err != nil {
		t.Fatal(r.Err)
	}
	kept, err := os.ReadFile(file)
	if err != nil || len(kept) == 0 {
		t.Fatalf("after an update the state file holds %d bytes (%v)", len(kept), err)
	}
	if left, _ := filepath.Glob(file + ".*.tmp"); len(left) != 0 {
		t.Errorf("the write's file beside the state was left behind: %v", left)
	}
	var notes []string
	next, err := NewService(rec, Options{Clock: fixedClock, StateFile: file, Note: func(n string) { notes = append(notes, n) }})
	if err != nil || len(notes) != 0 || string(next.lib.State()) != string(kept) {
		t.Errorf("the next run did not take the state kept (%v, notes %v)", err, notes)
	}
	if err := os.WriteFile(file, []byte("{held"), 0o600); err != nil {
		t.Fatal(err)
	}
	notes = nil
	if _, err := NewService(rec, Options{Clock: fixedClock, StateFile: file, Note: func(n string) { notes = append(notes, n) }}); err != nil || len(notes) != 1 || !strings.Contains(notes[0], "the kept state was not taken") {
		t.Errorf("a state that cannot be taken: %v, notes %v; want the service made and the refusal noted", err, notes)
	}
	// R3L-6: a part refused is named, the other said taken.
	edit := func(key string, v any) []byte {
		var m map[string]any
		if err := json.Unmarshal(kept, &m); err != nil {
			t.Fatal(err)
		}
		m[key] = v
		out, _ := json.Marshal(m)
		return out
	}
	for name, c := range map[string]struct {
		state []byte
		want  string
	}{
		"the mean refused": {edit("f107", 900), "the kept solar flux was not taken; GIRO's throttle state was"},
		"GIRO's refused":   {edit("burst", fixedClock().Add(time.Hour).Unix()), "GIRO's throttle state was not taken, so GIRO starts as new; the kept solar flux was taken"},
		"neither was taken": {func() []byte {
			var m map[string]any
			_ = json.Unmarshal(edit("f107", 900), &m)
			m["burst"] = fixedClock().Add(time.Hour).Unix()
			out, _ := json.Marshal(m)
			return out
		}(), "neither GIRO's throttle state nor the kept solar flux was taken"},
	} {
		if err := os.WriteFile(file, c.state, 0o600); err != nil {
			t.Fatal(err)
		}
		notes = nil
		if _, err := NewService(rec, Options{Clock: fixedClock, StateFile: file, Note: func(n string) { notes = append(notes, n) }}); err != nil || len(notes) != 1 || !strings.Contains(notes[0], c.want) || strings.Contains(notes[0], "\n") {
			t.Errorf("%s: %v, notes %q; want one line saying %q", name, err, notes, c.want)
		}
	}
}

// TestAnUpdateInFlightKeepsItsStateBeforeTheEnd is watchpost D-213 at the
// app's end: Settle waits for an update still running - a burst the quit
// cancelled - until it has kept the throttle state, within its bound.
func TestAnUpdateInFlightKeepsItsStateBeforeTheEnd(t *testing.T) {
	release := make(chan struct{})
	rec := &recorder{answer: func(url string) (httpx.Answer, error) {
		<-release
		return httpx.Answer{Status: 404}, nil
	}}
	file := filepath.Join(t.TempDir(), "propagation-throttle.json")
	s, err := NewService(rec, Options{Clock: fixedClock, StateFile: file})
	if err != nil {
		t.Fatal(err)
	}
	done := s.Ask(context.Background(), ionomaps.UpdateOptions{})
	if s.Settle(20 * time.Millisecond) {
		t.Fatal("Settle reported the update ended while it was still asking")
	}
	close(release)
	if !s.Settle(5 * time.Second) {
		t.Fatal("the update did not end within the bound")
	}
	if _, err := os.Stat(file); err != nil {
		t.Errorf("after Settle the state is not kept: %v", err)
	}
	<-done
}
