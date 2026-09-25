package tty

// map_status_test.go — 0.18.0 W3.3 (FR-3.4) and W1.15's loading indicator:
// the window's last line says what the picture is while it is not whole.

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"
	"github.com/branden-thompson/go-tuimaps/assets"
)

// unreachable is a source that never answers: the station is offline.
type unreachable struct{}

func (unreachable) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline")
}

// answering is a source that answers everything, in memory: a TileJSON, and
// real tile bytes for every tile.
type answering struct{}

func (answering) RoundTrip(r *http.Request) (*http.Response, error) {
	body := []byte(`{"tilejson":"3.0.0","tiles":["https://tiles.example.test/planet/{z}/{x}/{y}.pbf"],"minzoom":0,"maxzoom":14}`)
	if strings.HasSuffix(r.URL.Path, ".pbf") {
		body, _ = assets.Tile(2, 1, 1)
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Header: http.Header{}, Request: r}, nil
}

// answeredMap names a source that answers every tile.
func answeredMap(size tuimaps.Size) (*tuimaps.Map, error) {
	m, err := embeddedMap(size)
	if err != nil {
		return nil, err
	}
	if err := m.SetFetchOptions(tuimaps.FetchOptions{Transport: answering{}}); err != nil {
		return nil, err
	}
	return m, m.Source("https://tiles.example.test/planet")
}

// offlineMap names a source that never answers, with the embedded tiles.
func offlineMap(size tuimaps.Size) (*tuimaps.Map, error) {
	m, err := embeddedMap(size)
	if err != nil {
		return nil, err
	}
	if err := m.SetFetchOptions(tuimaps.FetchOptions{Transport: unreachable{}}); err != nil {
		return nil, err
	}
	return m, m.Source("https://tiles.example.test/planet")
}

func statusLine(d Dashboard) string {
	lines := d.mapBodyLines()
	return stripANSITest(lines[len(lines)-1])
}

// TestTheWindowSaysWhatThePictureIs is W3.3 (FR-3.4) with W1.15's indicator:
// while the view is sharpening the window says it is loading; when the
// source cannot be reached it says the picture is what the map already holds;
// a whole picture carries no note.
func TestTheWindowSaysWhatThePictureIs(t *testing.T) {
	d := mapDash(t, Config{NewMap: offlineMap})
	d, _ = pressKey(d, "g")
	if got := statusLine(d); !strings.Contains(got, mapLoadingText) {
		t.Errorf("a view still sharpening says %q, want the loading line", got)
	}
	d = settleMap(t, d)
	if got := statusLine(d); !strings.Contains(got, mapOfflineText) {
		t.Errorf("a view whose source cannot be reached says %q, want the offline line", got)
	}
	whole := mapDash(t, Config{NewMap: answeredMap})
	whole, _ = pressKey(whole, "g")
	whole = settleMap(t, whole)
	if got := strings.TrimSpace(statusLine(whole)); got != "" {
		t.Errorf("a whole picture carries a note: %q", got)
	}
	coarse := mapDash(t, Config{}) // the embedded tiles alone stop at zoom 3
	coarse, _ = pressKey(coarse, "g")
	coarse = settleMap(t, coarse)
	if got := statusLine(coarse); !strings.Contains(got, mapCoarseText) {
		t.Errorf("a view the tiles cannot sharpen, with nothing pending, says %q, want the coarser line", got)
	}
	if len(whole.mapBodyLines()) != whole.modalMax() {
		t.Errorf("the window's body is %d lines, want the window's %d", len(whole.mapBodyLines()), whole.modalMax())
	}
}

// flaky is a source that is offline until it is told otherwise.
type flaky struct{ online *atomic.Bool }

func (f flaky) RoundTrip(r *http.Request) (*http.Response, error) {
	if !f.online.Load() {
		return nil, errors.New("offline")
	}
	return answering{}.RoundTrip(r)
}

// TestTheOfflineNoteClearsWhenTheSourceReturns is W3.3 (FR-3.4): the note is
// the truth about the picture, so it goes when the source answers again and
// the picture is whole.
func TestTheOfflineNoteClearsWhenTheSourceReturns(t *testing.T) {
	online := &atomic.Bool{}
	d := mapDash(t, Config{NewMap: func(size tuimaps.Size) (*tuimaps.Map, error) {
		m, err := embeddedMap(size)
		if err != nil {
			return nil, err
		}
		if err := m.SetFetchOptions(tuimaps.FetchOptions{Transport: flaky{online: online}}); err != nil {
			return nil, err
		}
		return m, m.Source("https://tiles.example.test/planet")
	}})
	start := d.now()
	d, _ = pressKey(d, "g")
	d = settleMap(t, d)
	if got := statusLine(d); !strings.Contains(got, mapOfflineText) {
		t.Fatalf("offline, the window says %q", got)
	}
	online.Store(true)
	d.now = func() time.Time { return start.Add(11 * time.Minute) } // past the library's longest wait before a retry
	m, _ := d.Update(tea.WindowSizeMsg{Width: d.width, Height: d.height})
	d = settleMap(t, m.(Dashboard))
	if got := strings.TrimSpace(statusLine(d)); got != "" {
		t.Errorf("back online with the picture whole, the window still says %q", got)
	}
}
