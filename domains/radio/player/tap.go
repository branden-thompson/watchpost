package player

import (
	"encoding/binary"
	"io"
	"sync"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Tap copies the PCM it passes through (16-bit LE stereo at OutputRate)
// into a ring of mono samples so the visualizer can read the latest frames
// (UAT 92, CLIAmp's tap). It sits before the player's volume, so the bars
// show the broadcast's own dynamics, not the volume setting. One writer
// (the audio player's reads) and one reader (the UI tick) share a mutex —
// a few thousand int16 copies per call either way.
type Tap struct {
	mu      sync.Mutex
	ring    []int16
	pos     int   // next write index
	written int64 // frames written since Reset
}

// NewTap makes a tap holding the latest frames frames.
func NewTap(frames int) (*Tap, error) {
	if err := invariant.Check(frames > 0, "player: a tap needs room for at least one frame"); err != nil {
		return nil, err
	}
	return &Tap{ring: make([]int16, frames)}, nil
}

// Wrap returns r with every frame that passes through copied into the tap.
func (t *Tap) Wrap(r io.Reader) io.Reader { return &tapReader{t: t, r: r} }

// Reset forgets everything (Halt: the bars decay to rest).
func (t *Tap) Reset() {
	t.mu.Lock()
	t.pos, t.written = 0, 0
	t.mu.Unlock()
}

// Samples fills dst with the most recent frames (oldest first) scaled to
// ±1 and returns how many there were — fewer than len(dst) right after a
// start or Reset, never more than the ring holds.
func (t *Tap) Samples(dst []float64) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := min(len(dst), len(t.ring), int(min(t.written, int64(len(t.ring)))))
	start := (t.pos - n + len(t.ring)) % len(t.ring)
	for i := 0; i < n; i++ { // bounded by n ≤ len(ring)
		dst[i] = float64(t.ring[(start+i)%len(t.ring)]) / 32768
	}
	return n
}

// push mixes stereo frames down to mono and appends them to the ring.
func (t *Tap) push(pcm []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := 0; i+4 <= len(pcm); i += 4 { // bounded by len(pcm)
		l := int16(binary.LittleEndian.Uint16(pcm[i:]))
		r := int16(binary.LittleEndian.Uint16(pcm[i+2:]))
		t.ring[t.pos] = int16((int32(l) + int32(r)) / 2)
		t.pos = (t.pos + 1) % len(t.ring)
		t.written++
	}
}

// tapReader is the pass-through.
type tapReader struct {
	t *Tap
	r io.Reader
}

func (tr *tapReader) Read(p []byte) (int, error) {
	n, err := tr.r.Read(p)
	if n > 0 {
		tr.t.push(p[:n])
	}
	return n, err
}
