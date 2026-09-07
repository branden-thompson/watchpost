package player

// silence.go — a relay that is up and broadcasting nothing.
//
// AN OUTAGE DOES NOT LOOK LIKE AN OUTAGE. On 2026-09-04 weatherusa.net answered
// every check correctly — HTTP 200, Content-Type audio/mpeg, correct ICY
// headers, ~19 KB/s of well-formed MP3 that decoded without a single error — and
// broadcast digital silence on every mount. Reachability, content type, byte
// rate and decoder health were all green while the listener heard nothing, and
// the station reported PLAYING throughout. For a weather radio that is the worst
// available answer: confident and wrong.
//
// The only check that could tell the difference is one that looks at the SAMPLES.

import (
	"io"
	"math"
	"time"
)

// SilenceFloor is the RMS below which a window counts as silent.
//
// Speech on these relays reads in the high hundreds to low thousands (a working
// mount measured 1663-1775). A dead one reads 0.0-0.1. Ten is far above the
// noise a real broadcast never drops to and far below anything audible, so the
// gap between the two populations is three orders of magnitude wide — this
// threshold does not sit near either.
const SilenceFloor = 10.0

// silenceWindow is how much audio one reading covers. Fine enough to see a gap,
// coarse enough that a single quiet frame is not a finding.
const silenceWindow = 250 * time.Millisecond

// silenceReader watches decoded PCM go past and calls fire once the stream has
// been quiet for longer than after.
//
// IT COUNTS BYTES, NOT SECONDS. PCM leaves the resampler at a fixed rate, so the
// bytes ARE the clock — exactly, and without a timer. That is what lets the rule
// be tested by handing it a buffer instead of by waiting five seconds and hoping
// the machine was not busy.
type silenceReader struct {
	r     io.Reader
	fire  func()        // called at most once; MUST NOT BLOCK — this is the audio path
	after time.Duration // how long quiet must last
	win   int           // bytes per reading window
	buf   []byte        // the partial window carried between Reads
	quiet time.Duration // how long it has been quiet
	fired bool
}

// newSilenceReader wraps r, which must be 16-bit little-endian stereo at rate.
func newSilenceReader(r io.Reader, rate int, after time.Duration, fire func()) *silenceReader {
	win := int(float64(rate) * 4 * silenceWindow.Seconds()) // 4 bytes a frame
	if win < 4 {
		win = 4
	}
	return &silenceReader{r: r, fire: fire, after: after, win: win, buf: make([]byte, 0, win)}
}

// Read passes the audio through untouched and measures it on the way.
func (s *silenceReader) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if n > 0 {
		s.measure(p[:n])
	}
	return n, err
}

// measure accumulates whole windows and judges each one.
func (s *silenceReader) measure(b []byte) {
	if s.fired || s.fire == nil {
		return // one report per stream: the listener is told once, not every window
	}
	s.buf = append(s.buf, b...)
	// One pass per whole window the buffer holds, counted before the walk
	// begins: each pass consumes exactly one and nothing adds to the buffer
	// inside the loop, so the count IS the bound (P10-02).
	for range len(s.buf) / s.win {
		if s.judge(s.buf[:s.win]) {
			return
		}
		s.buf = s.buf[s.win:]
	}
	// Keep the remainder, and never let it grow past a window.
	if len(s.buf) > s.win {
		s.buf = s.buf[len(s.buf)-s.win:]
	}
}

// judge reads one window and reports whether it fired.
func (s *silenceReader) judge(win []byte) bool {
	var sumSq float64
	var n int
	for i := 0; i+1 < len(win); i += 2 { // bounded by the window (P10-02)
		v := float64(int16(uint16(win[i]) | uint16(win[i+1])<<8))
		sumSq += v * v
		n++
	}
	if n == 0 {
		return false
	}
	if math.Sqrt(sumSq/float64(n)) >= SilenceFloor {
		s.quiet = 0 // any audible window clears the run: this is a RUN, not a total
		return false
	}
	s.quiet += silenceWindow
	if s.quiet < s.after {
		return false
	}
	s.fired = true
	s.fire()
	return true
}
