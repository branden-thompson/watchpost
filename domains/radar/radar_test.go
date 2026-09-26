package radar

// radar_test.go — 0.18.0 W8.2 to W8.5 and W8.3a over the recorded responses
// (FR-5.3 as D-84 amends it, FR-5.7, FR-8.6, D-47).

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// fakeGet answers every request with a fixture and records the address.
type fakeGet struct {
	body []byte
	asks []string
}

func (f *fakeGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	f.asks = append(f.asks, rawURL)
	return f.body, nil
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestEveryRecordedRadarFixtureIsPresent(t *testing.T) {
	var m struct {
		Captured string   `json:"captured"`
		Files    []string `json:"files"`
	}
	if err := json.Unmarshal(fixture(t, "manifest.json"), &m); err != nil || m.Captured == "" || len(m.Files) < 7 {
		t.Fatalf("the manifest is %+v (%v)", m, err)
	}
	for _, f := range m.Files {
		if len(fixture(t, f)) == 0 {
			t.Errorf("%s is empty", f)
		}
	}
}

func TestTheTimesAreTheSourcesOwn(t *testing.T) {
	iem := NewIEM(&fakeGet{body: fixture(t, "iem-times.json")}, "")
	times, err := iem.Times(context.Background(), geo.RegionContiguous)
	if err != nil || len(times) < 20 {
		t.Fatalf("IEM's times: %d (%v)", len(times), err)
	}
	for i := 1; i < len(times); i++ {
		if d := times[i].Sub(times[i-1]); d != 5*time.Minute {
			t.Errorf("IEM's scans %v apart at %d; its grid is five minutes", d, i)
		}
	}
	if _, err := iem.Times(context.Background(), geo.RegionAlaska); !errors.Is(err, ErrNotCovered) {
		t.Errorf("IEM answered for Alaska: %v", err)
	}
	mt, err := mrmsTimes(fixture(t, "mrms-capabilities.xml"))
	if err != nil || len(mt) < 40 || !mt[0].Before(mt[len(mt)-1]) {
		t.Errorf("MRMS's times: %d (%v)", len(mt), err)
	}
	mrms := NewMRMS(&fakeGet{}, "")
	for _, r := range []string{geo.RegionContiguous, geo.RegionAlaska, geo.RegionHawaii, geo.RegionCaribbean, geo.RegionMarianas} {
		if !mrms.Covers(r) {
			t.Errorf("MRMS does not cover %s", r)
		}
	}
	if mrms.Covers(geo.RegionSamoa) || iem.Covers(geo.RegionSamoa) {
		t.Error("a source claims American Samoa, which none covers")
	}
}

// TestAFrameIsAskedForOnlyAtAnAdvertisedTime is D-84's request half: an
// off-grid time, a zero time or any time not listed is refused before
// anything is sent; an advertised one is always sent with its time, and the
// box asked for is the radar box, never the view (D-47).
func TestAFrameIsAskedForOnlyAtAnAdvertisedTime(t *testing.T) {
	advertised := []time.Time{time.Date(2026, 9, 26, 15, 40, 0, 0, time.UTC), time.Date(2026, 9, 26, 15, 45, 0, 0, time.UTC)}
	box := wholeBoxes[geo.RegionContiguous]
	for _, s := range []Source{NewIEM(&fakeGet{}, ""), NewMRMS(&fakeGet{}, "")} {
		get := &fakeGet{body: fixture(t, "iem-frame.png")}
		switch v := s.(type) {
		case *IEM:
			v.get = get
		case *MRMS:
			v.get = get
		}
		for _, bad := range []time.Time{{}, time.Date(2026, 9, 26, 15, 41, 0, 0, time.UTC)} {
			if _, err := s.Frame(context.Background(), geo.RegionContiguous, bad, advertised, box); !errors.Is(err, ErrNotAdvertised) {
				t.Errorf("%s asked for %v: %v", s.Name(), bad, err)
			}
		}
		if len(get.asks) != 0 {
			t.Fatalf("%s sent a request for a time it does not hold: %v", s.Name(), get.asks)
		}
		if _, err := s.Frame(context.Background(), geo.RegionContiguous, advertised[0], advertised, box); err != nil {
			t.Fatal(err)
		}
		u, _ := url.Parse(get.asks[0])
		q := u.Query()
		timeParam := q.Get("TIME") + q.Get("time")
		if !strings.HasPrefix(timeParam, "2026-09-26T15:40:00") {
			t.Errorf("%s sent time %q", s.Name(), timeParam)
		}
		if bbox := q.Get("BBOX"); bbox != "-126,23,-65,51" {
			t.Errorf("%s asked for the box %q, not the radar box", s.Name(), bbox)
		}
		if !strings.HasPrefix(get.asks[0], "https://") {
			t.Errorf("%s asked over %s", s.Name(), get.asks[0])
		}
	}
}

// TestTheCheckTellsAFrameFromNothing is D-84's picture half over the
// recorded responses: a frame with echo paints; an off-grid or expired
// answer is empty (which a request never provokes); a picture declaring more
// pixels than a frame may have is refused before it decodes.
func TestTheCheckTellsAFrameFromNothing(t *testing.T) {
	for name, want := range map[string]bool{"iem-frame.png": false, "mrms-frame.png": false, "iem-offgrid.png": true, "mrms-expired.png": true} {
		empty, err := Check(fixture(t, name))
		if err != nil || empty != want {
			t.Errorf("%s: empty %v (%v), want %v", name, empty, err, want)
		}
	}
	var huge bytes.Buffer
	if err := png.Encode(&huge, image.NewAlpha(image.Rect(0, 0, 600, 500))); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(huge.Bytes()); !errors.Is(err, ErrTooLarge) {
		t.Errorf("a 600x500 picture, past the library's 250,000 pixels, was not refused: %v", err)
	}
}

// TestTheBoxesAreFixedAndWithinTheCap is W8.3a: every box is within the
// library's per-image cap; a wide view takes the region's one box; a closer
// one takes the grid's boxes it meets, one to four; a small move inside a box
// asks for nothing new; American Samoa has none.
func TestTheBoxesAreFixedAndWithinTheCap(t *testing.T) {
	var all []Box
	for _, b := range wholeBoxes {
		all = append(all, b)
	}
	all = append(all, grid(wholeBoxes[geo.RegionContiguous])...)
	for _, b := range all {
		if b.Cols*b.Rows > maxPixels || b.Cols <= 0 || b.Rows <= 0 {
			t.Errorf("%s is %dx%d, past the cap", b.Name, b.Cols, b.Rows)
		}
	}
	if got := BoxesFor(geo.RegionContiguous, geo.Box{W: -125, S: 24, E: -66, N: 50}); len(got) != 1 || got[0].Name != "us" {
		t.Errorf("the whole lower 48 takes %v", got)
	}
	socal := BoxesFor(geo.RegionContiguous, geo.Box{W: -119.6, S: 32.1, E: -115.0, N: 34.2})
	if len(socal) < 1 || len(socal) > 4 {
		t.Fatalf("a state view takes %d boxes", len(socal))
	}
	moved := BoxesFor(geo.RegionContiguous, geo.Box{W: -119.4, S: 32.2, E: -114.8, N: 34.3})
	if len(moved) != len(socal) || moved[0].Name != socal[0].Name {
		t.Errorf("a small move asked for new boxes: %v then %v", socal, moved)
	}
	if got := BoxesFor(geo.RegionSamoa, geo.Box{W: -171, S: -15, E: -168, N: -11}); got != nil {
		t.Errorf("American Samoa takes %v", got)
	}
	if got := BoxesFor(geo.RegionAlaska, geo.Box{W: -150, S: 60, E: -149, N: 61}); len(got) != 1 || got[0].Name != "ak" {
		t.Errorf("a close view in Alaska takes %v; want Alaska's one box, never the lower 48's grid", got)
	}
	if got := BoxesFor(geo.RegionHawaii, geo.Box{W: -158, S: 21, E: -157.5, N: 21.5}); len(got) != 1 || got[0].Name != "hi" {
		t.Errorf("a close view in Hawaii takes %v", got)
	}
	for _, b := range socal { // the southern row alone: the view is south of the grid's middle
		if b.S != wholeBoxes[geo.RegionContiguous].S {
			t.Errorf("southern California takes %s, a northern box", b.Name)
		}
	}
}

// TestTheRadarClientKeepsNothingOnDisk is W8.14 and W8.5's client: memory
// only (no cache directory, so no radar request reaches disk), the 1 MiB cap,
// public addresses only, https only.
func TestTheRadarClientKeepsNothingOnDisk(t *testing.T) {
	c := ClientConfig("watchpost/test")
	if c.CacheDir != "" || c.MaxBodyBytes != 1<<20 || !c.RefusePrivate || !c.HTTPSOnly {
		t.Errorf("the radar client is %+v", c)
	}
}

// TestARefreshFetchesOnlyTheNewFrames is W8.7 (FR-5.5): with a frame held, the
// same frame asked again is not fetched again; a new time is.
func TestARefreshFetchesOnlyTheNewFrames(t *testing.T) {
	var hits atomic.Int32
	png := fixture(t, "iem-frame.png")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(png)
	}))
	defer srv.Close()
	c, err := httpx.New(httpx.Config{UserAgent: "test"}) // the loopback server: the radar client itself refuses it
	if err != nil {
		t.Fatal(err)
	}
	s := NewIEM(c, srv.URL)
	times := []time.Time{time.Date(2026, 9, 26, 15, 40, 0, 0, time.UTC), time.Date(2026, 9, 26, 15, 45, 0, 0, time.UTC)}
	box := wholeBoxes[geo.RegionContiguous]
	for _, at := range []time.Time{times[0], times[0], times[1], times[0], times[1]} {
		if _, err := s.Frame(context.Background(), geo.RegionContiguous, at, times, box); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 2 {
		t.Errorf("five asks over two frames fetched %d times; want each frame once", hits.Load())
	}
}
