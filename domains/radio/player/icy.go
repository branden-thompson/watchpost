// Package player plays NOAA Weather Radio relays (B4, architecture §5/§10.4,
// AI-5): Icecast HTTP → ICY metadata strip → go-mp3 → resample → oto. Pure
// Go, no C toolchain; one oto context per process, created lazily on the
// first tune-in and never closed. Everything blocking runs on the engine's
// goroutines — never on the Bubble Tea update loop.
package player

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// StallTimeout is the longest gap between bytes before a stream counts as
// stalled (§10.4: 15 s — NWR never goes intentionally silent).
const StallTimeout = 15 * time.Second

// Stream is one open Icecast connection.
type Stream struct {
	Name    string // icy-name
	Type    string // Content-Type
	Bitrate int    // icy-br

	body   io.ReadCloser
	r      io.Reader
	cancel context.CancelFunc

	mu    sync.Mutex
	title string
	stall *time.Timer
}

// Open connects to a mount with ICY metadata requested. The context bounds
// the whole connection; a stall watchdog cancels it when no bytes arrive
// for StallTimeout.
func Open(ctx context.Context, userAgent, url string) (*Stream, error) {
	if userAgent == "" || url == "" {
		return nil, errors.New("player: user agent and url are required")
	}
	ctx, cancel := context.WithCancel(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Icy-MetaData", "1")
	resp, err := newStreamClient().Do(req)
	if err != nil {
		cancel()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		cancel()
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("mount %s: HTTP %d: %w", url, resp.StatusCode, ErrMountRefused)
		}
		return nil, fmt.Errorf("mount %s: HTTP %d", url, resp.StatusCode)
	}
	s := &Stream{Name: plain(resp.Header.Get("icy-name")), Type: resp.Header.Get("Content-Type"), body: resp.Body, cancel: cancel}
	s.Bitrate, _ = strconv.Atoi(resp.Header.Get("icy-br"))
	s.stall = time.AfterFunc(StallTimeout, cancel)
	var r io.Reader = &watchdogReader{r: resp.Body, s: s}
	if n, _ := strconv.Atoi(resp.Header.Get("icy-metaint")); n > 0 {
		r = &icyReader{r: bufio.NewReader(r), metaint: n, left: n, s: s}
	}
	s.r = r
	return s, nil
}

// newStreamClient has no overall timeout: Icecast bodies are infinite.
// Stalls are handled by the per-read watchdog. One per stream (a stream is
// long-lived; no shared global).
// Redirects follow the app-wide policy (quality pass Q1, IS-3): same
// scheme and host, three hops — a plain-HTTP relay must never be able to
// send the player to another host.
func newStreamClient() *http.Client {
	tr := httpx.NewTransport() // the app-wide policy (Q5), tuned for a long-lived stream
	tr.ResponseHeaderTimeout, tr.DisableCompression, tr.MaxConnsPerHost, tr.MaxIdleConnsPerHost = 15*time.Second, true, 2, 2
	return &http.Client{CheckRedirect: httpx.SameOriginRedirect, Transport: tr}
}

// Read yields audio bytes (metadata stripped).
func (s *Stream) Read(p []byte) (int, error) { return s.r.Read(p) }

// Close tears the connection down.
func (s *Stream) Close() error {
	s.stall.Stop()
	s.cancel()
	return s.body.Close()
}

// Title is the latest in-band StreamTitle ("" when the relay sends none).
func (s *Stream) Title() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.title
}

func (s *Stream) setTitle(t string) {
	s.mu.Lock()
	s.title = plain(t)
	s.mu.Unlock()
}

// plain drops control characters (ESC included) from relay-sent text —
// names and in-band titles reach the screen and the marquee (red-team
// 0.9.0 S-F6): a relay must not be able to address the terminal.
func plain(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return -1
		}
		return r
	}, s)
}

// watchdogReader re-arms the stall timer on every successful read.
type watchdogReader struct {
	r io.Reader
	s *Stream
}

func (w *watchdogReader) Read(p []byte) (int, error) {
	n, err := w.r.Read(p)
	if n > 0 {
		w.s.stall.Reset(StallTimeout)
	}
	return n, err
}

// icyReader strips Icecast in-band metadata: after every metaint audio
// bytes, one length byte (×16) then "StreamTitle='…';" padded with NULs.
type icyReader struct {
	r       *bufio.Reader
	metaint int
	left    int // audio bytes until the next metadata block
	s       *Stream
}

func (c *icyReader) Read(p []byte) (int, error) {
	if c.left == 0 {
		if err := c.readMeta(); err != nil {
			return 0, err
		}
		c.left = c.metaint
	}
	if len(p) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= n
	return n, err
}

func (c *icyReader) readMeta() error {
	lb, err := c.r.ReadByte()
	if err != nil {
		return err
	}
	if lb == 0 {
		return nil
	}
	buf := make([]byte, int(lb)*16)
	if _, err := io.ReadFull(c.r, buf); err != nil {
		return err
	}
	meta := strings.TrimRight(string(buf), "\x00")
	if i := strings.Index(meta, "StreamTitle='"); i >= 0 {
		rest := meta[i+len("StreamTitle='"):]
		if j := strings.Index(rest, "';"); j >= 0 {
			c.s.setTitle(rest[:j])
		}
	}
	return nil
}

// newIcyReader is the test seam for the stripper (no HTTP).
func newIcyReader(r io.Reader, metaint int, s *Stream) io.Reader {
	return &icyReader{r: bufio.NewReader(r), metaint: metaint, left: metaint, s: s}
}
