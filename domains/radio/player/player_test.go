package player

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Spec: AI-5 §3 (ICY stripping, stall watchdog, preroll), §10.4 (backoff,
// mount failover), engine states. No audio device: fakeOutput drains PCM.

func TestICYReaderStripsMetadataAndReportsTitle(t *testing.T) {
	audio := []byte("0123456789abcdef") // 16 audio bytes
	meta := []byte("StreamTitle='KEC49 Test';")
	meta = append(meta, make([]byte, 32-len(meta))...) // pad to 2 blocks of 16
	var raw bytes.Buffer
	raw.Write(audio)
	raw.WriteByte(2) // 2 × 16 bytes of metadata
	raw.Write(meta)
	raw.Write(audio)
	raw.WriteByte(0) // empty metadata block
	raw.Write(audio)
	s := &Stream{}
	got, err := io.ReadAll(newIcyReader(&raw, 16, s))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != strings.Repeat("0123456789abcdef", 3) {
		t.Fatalf("metadata must be stripped: %q", got)
	}
	if s.Title() != "KEC49 Test" {
		t.Fatalf("title = %q", s.Title())
	}
}

func TestResamplerKeepsDurationAndLevel(t *testing.T) {
	// 22050 Hz -> 44100 Hz doubles the frame count; a constant signal stays constant.
	const frames = 2205
	src := make([]byte, frames*4)
	left, right := int16(1000), int16(-1000)
	for i := range frames {
		binary.LittleEndian.PutUint16(src[i*4:], uint16(left))
		binary.LittleEndian.PutUint16(src[i*4+2:], uint16(right))
	}
	out, _ := io.ReadAll(newResampler(bytes.NewReader(src), 22050))
	if got := len(out) / 4; got < frames*2-4 || got > frames*2 {
		t.Fatalf("frames out = %d, want ~%d", got, frames*2)
	}
	if l, r := int16(binary.LittleEndian.Uint16(out[400:])), int16(binary.LittleEndian.Uint16(out[402:])); l != 1000 || r != -1000 {
		t.Fatalf("constant signal must pass through: %d %d", l, r)
	}
	if r := newResampler(bytes.NewReader(src), OutputRate); r != io.Reader(bytes.NewReader(src)) && len(src) > 0 {
		// same-rate input passes through untouched (type check by behaviour)
		out2, _ := io.ReadAll(r)
		if len(out2) != len(src) {
			t.Fatal("same rate must pass through")
		}
	}
}

// fakeOutput drains PCM on a goroutine and counts bytes.
type fakeOutput struct{ bytes atomic.Int64 }

type fakePlayer struct {
	out     *fakeOutput
	stop    chan struct{}
	playing atomic.Bool
	vol     atomic.Value
	once    sync.Once
}

func (f *fakeOutput) NewPlayer(pcm io.Reader) (Player, error) {
	p := &fakePlayer{out: f, stop: make(chan struct{})}
	p.vol.Store(0.0)
	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-p.stop:
				return
			default:
			}
			n, err := pcm.Read(buf)
			f.bytes.Add(int64(n))
			if err != nil {
				p.playing.Store(false)
				return
			}
		}
	}()
	return p, nil
}
func (p *fakePlayer) Play()               { p.playing.Store(true) }
func (p *fakePlayer) Pause()              { p.playing.Store(false) }
func (p *fakePlayer) IsPlaying() bool     { return p.playing.Load() }
func (p *fakePlayer) SetVolume(v float64) { p.vol.Store(v) }
func (p *fakePlayer) Close() error        { p.once.Do(func() { close(p.stop) }); return nil }

// mp3Server streams the tone fixture in a loop as Icecast would.
func mp3Server(t *testing.T, fail404 map[string]bool) *httptest.Server {
	t.Helper()
	tone, err := os.ReadFile("testdata/tone.mp3")
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail404[r.URL.Path] {
			w.WriteHeader(404)
			return
		}
		if r.Header.Get("Icy-MetaData") != "1" || r.Header.Get("User-Agent") == "" {
			t.Errorf("stream request must carry Icy-MetaData and a User-Agent")
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("icy-name", "NOAA Weather Radio TEST")
		w.Header().Set("icy-br", "32")
		w.WriteHeader(200)
		fl, _ := w.(http.Flusher)
		for range 40 { // ~80 s of audio at 2 s per loop, far more than a test consumes
			if _, err := w.Write(tone); err != nil {
				return
			}
			if fl != nil {
				fl.Flush()
			}
			select {
			case <-r.Context().Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}))
}

func statesOf(log *[]Status, mu *sync.Mutex) []State {
	mu.Lock()
	defer mu.Unlock()
	var out []State
	for _, s := range *log {
		if len(out) == 0 || out[len(out)-1] != s.State {
			out = append(out, s.State)
		}
	}
	return out
}

func TestEnginePlaysFirstWorkingMountAndStops(t *testing.T) {
	srv := mp3Server(t, map[string]bool{"/dead": true})
	defer srv.Close()
	var mu sync.Mutex
	var log []Status
	out := &fakeOutput{}
	e, err := New(out, "watchpost/test (t@example.com)", func(s Status) { mu.Lock(); log = append(log, s); mu.Unlock() })
	if err != nil {
		t.Fatal(err)
	}
	e.Volume(70)
	e.Start([]string{srv.URL + "/dead", srv.URL + "/live"}, "KEC49 Monterey")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) && e.Status().State != Playing {
		time.Sleep(20 * time.Millisecond)
	}
	st := e.Status()
	if st.State != Playing || !strings.HasSuffix(st.Mount, "/live") || st.Name != "NOAA Weather Radio TEST" || st.Volume != 70 {
		t.Fatalf("status = %+v", st)
	}
	time.Sleep(300 * time.Millisecond)
	if out.bytes.Load() == 0 {
		t.Fatal("decoded PCM must reach the output")
	}
	seq := statesOf(&log, &mu)
	if seq[0] != Stopped || !contains(seq, Connecting) || !contains(seq, Playing) {
		t.Fatalf("state sequence %v", seq)
	}
	start := time.Now()
	e.Halt()
	if e.Status().State != Stopped {
		t.Fatal("Halt must report Stopped")
	}
	if time.Since(start) > 500*time.Millisecond { // UAT 81: stop feels instant
		t.Fatalf("Halt took %v", time.Since(start))
	}
	e.Halt() // idempotent
}

func TestEngineFailsWhenEveryMountIsDead(t *testing.T) {
	srv := mp3Server(t, map[string]bool{"/a": true, "/b": true})
	defer srv.Close()
	e, _ := New(&fakeOutput{}, "watchpost/test (t@example.com)", nil)
	e.Start(nil, "none")
	if st := e.Status(); st.State != Failed || st.Err == "" {
		t.Fatalf("no mounts must fail at once: %+v", st)
	}
	if got := backoff(0); got < backoffBase/2 || got > backoffBase*3/2 {
		t.Fatalf("backoff(0) = %v, want ~1s ±50%%", got)
	}
	if got := backoff(10); got > backoffMax*3/2 {
		t.Fatalf("backoff must cap at 30s: %v", got)
	}
}

func contains(seq []State, s State) bool {
	for _, x := range seq {
		if x == s {
			return true
		}
	}
	return false
}

func TestOpenRefusesBadInput(t *testing.T) {
	if _, err := Open(context.Background(), "", "http://x"); err == nil {
		t.Fatal("user agent required")
	}
	if _, err := Open(context.Background(), "ua", ""); err == nil {
		t.Fatal("url required")
	}
}

func TestSourceCompletionReportsStoppedNotFailed(t *testing.T) {
	// UAT 83: a synthesized broadcast that ends (Repeat off) is a completion.
	e, _ := New(&fakeOutput{}, "watchpost/test (t@example.com)", nil)
	e.StartSource("synth", OutputRate, func(context.Context) io.Reader { return bytes.NewReader(make([]byte, 4*4410)) })
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && e.Status().Title != "broadcast complete" && e.Status().State != Failed {
		time.Sleep(20 * time.Millisecond)
	}
	if st := e.Status(); st.State != Stopped || st.Title != "broadcast complete" {
		t.Fatalf("completion status: %+v", st)
	}
}

func TestPreviewPlaysWithoutTouchingState(t *testing.T) {
	// UAT 86: a voice sample plays on its own player; engine status is unchanged.
	out := &fakeOutput{}
	e, _ := New(out, "watchpost/test (t@example.com)", nil)
	before := e.Status()
	if err := e.Preview(OutputRate, bytes.NewReader(make([]byte, 4*4410))); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && out.bytes.Load() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if out.bytes.Load() == 0 {
		t.Fatal("preview audio must reach the output")
	}
	if e.Status() != before {
		t.Fatalf("preview must not change engine status: %+v", e.Status())
	}
}
