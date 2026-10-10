package propagation

import (
	"context"
	"errors"
	"net/http"
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
	if rec.count() != 0 {
		t.Errorf("the client was asked %d times", rec.count())
	}
	rec.answer = func(string) (httpx.Answer, error) { return httpx.Answer{}, nil }
	if _, err := f.Fetch(context.Background(), "https://services.swpc.noaa.gov/text/x.txt", ionomaps.Validators{}); !errors.Is(err, errNoStatus) {
		t.Errorf("a reply with no status: %v", err)
	}
	rec.answer = nil
	if _, err := f.Fetch(context.Background(), "https://services.swpc.noaa.gov/text/x.txt", ionomaps.Validators{}); err != nil {
		t.Errorf("a listed host was refused: %v", err)
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

// TestAnUpdateIsJoinedNotDoubled is FR-4.8 (D-131): updates asked while one
// runs join it - one ask of NOAA, one result to each. Two updates not joined
// would also ask NOAA once, since the library answers the second "too soon";
// the same answer to both is what shows they were one.
func TestAnUpdateIsJoinedNotDoubled(t *testing.T) {
	rec := &recorder{hold: make(chan struct{})}
	s, err := New(rec, Options{Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	a, b := s.Ask(context.Background()), s.Ask(context.Background())
	close(rec.hold)
	ra, rb := <-a, <-b
	if ra.Snapshot.Early != rb.Snapshot.Early || ra.Snapshot.Early == ionomaps.TooSoon {
		t.Errorf("two joined updates were answered %v and %v: a second update ran, and was told too soon", ra.Snapshot.Early, rb.Snapshot.Early)
	}
	index := 0
	for _, u := range rec.asked {
		if strings.HasSuffix(u, "/geojson_2d_urt/") {
			index++
		}
	}
	if index != 1 {
		t.Errorf("two joined updates asked the index %d times: %v", index, rec.asked)
	}
}

// TestClosingTheModeCancelsTheUpdate is FR-4.2: the mode's context ends the
// update it started, and the result says so.
func TestClosingTheModeCancelsTheUpdate(t *testing.T) {
	rec := &recorder{hold: make(chan struct{})}
	s, err := New(rec, Options{Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	got := s.Ask(ctx)
	cancel()
	select {
	case r := <-got:
		if r.Err == nil && len(r.Snapshot.Hours) != 0 {
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
	s, err := New(rec, Options{Clock: fixedClock, Deadline: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.Ask(context.Background()):
	case <-time.After(5 * time.Second):
		t.Fatal("an update past its deadline never answered")
	}
	if _, err := New(rec, Options{}); err != nil {
		t.Errorf("the default options were refused: %v", err)
	}
	if _, err := New(nil, Options{}); !errors.Is(err, errNoClient) {
		t.Errorf("no client: %v", err)
	}
	if _, err := New(rec, Options{Deadline: -time.Second}); !errors.Is(err, errNoDeadline) {
		t.Errorf("a negative deadline: %v", err)
	}
	if r := <-(&Service{}).Ask(context.Background()); !errors.Is(r.Err, errNoService) {
		t.Errorf("a service not made with New: %v", r.Err)
	}
}

func fixedClock() time.Time { return time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC) }

// fetcherOf is the fetcher a service is made with.
func fetcherOf(get Getter, note func(string)) (*fetcher, error) {
	s, err := New(get, Options{Note: note})
	if err != nil {
		return nil, err
	}
	return s.fetch, nil
}
