package player

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"os"
	"testing"
	"time"

	"github.com/hajimehoshi/go-mp3"
)

// TestMountDiag answers the one question the network cannot: IS ANYTHING COMING
// OUT OF THIS MOUNT?
//
// A relay outage does not look like an outage. weatherusa.net answered every
// probe correctly on 2026-09-04 — HTTP 200, Content-Type audio/mpeg, correct ICY
// headers, ~19 KB/s of well-formed MP3 that decoded without a single error — and
// broadcast DIGITAL SILENCE on every mount. Reachability, content type, byte
// rate and decoder health were all green, and a listener heard nothing. The only
// instrument that could tell the difference was one that decoded the audio and
// looked at the samples.
//
// It opens the mount exactly as the engine does — the same Open, the same
// preroll, the same decoder — so what it reports is what a listener would get,
// not what a well-behaved HTTP client would.
//
// Live and opt-in; `make verify` never runs it:
//
//	WATCHPOST_MOUNT_DIAG=<url> go test ./domains/radio/player -run TestMountDiag -v
//
// RMS is the reading. Speech sits in the high hundreds to low thousands; a
// working relay measured 1775. Anything under ~10 is silence however healthy
// every other signal looks.
//
// It also reports the LONGEST QUIET RUN, which is the number a silence detector
// has to be set against. Measured on a healthy mount (KIH62 via wxradio.org,
// 2026-09-04): 406 windows over 102 seconds, ZERO of them quiet. NWR runs a
// continuous loop over a constant noise floor, so real silence on a working
// transmitter does not appear at all — the worry that a detector would demote a
// station idling between cycles is not borne out by the stream.
func TestMountDiag(t *testing.T) {
	url := os.Getenv("WATCHPOST_MOUNT_DIAG")
	if url == "" {
		t.Skip("set WATCHPOST_MOUNT_DIAG")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	s, err := Open(ctx, "watchpost-diag", url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	t.Logf("stream: name=%q type=%q bitrate=%d", s.Name, s.Type, s.Bitrate)

	dec, err := mp3.NewDecoder(&prerollReader{r: s, want: preroll})
	if err != nil {
		t.Fatalf("decoder: %v", err)
	}
	t.Logf("decoder: sampleRate=%d", dec.SampleRate())

	// 250 ms windows: fine enough to see a gap, coarse enough that one quiet
	// frame is not a finding.
	win := dec.SampleRate() // 250 ms of 16-bit stereo
	buf := make([]byte, win)
	var total, windows, quiet, run, longest int
	var sumSq float64
	var n int
	deadline := time.Now().Add(diagFor)
	for time.Now().Before(deadline) {
		k, err := io.ReadFull(dec, buf)
		total += k
		var wSq float64
		var wN int
		for i := 0; i+1 < k; i += 2 {
			v := float64(int16(binary.LittleEndian.Uint16(buf[i : i+2])))
			wSq += v * v
			wN++
		}
		if wN > 0 {
			sumSq, n = sumSq+wSq, n+wN
			windows++
			if math.Sqrt(wSq/float64(wN)) < silenceFloor {
				quiet++
				run++
				if run > longest {
					longest = run
				}
			} else {
				run = 0
			}
		}
		if err != nil {
			t.Logf("read stopped after %d bytes: %v", total, err)
			break
		}
	}
	rms := 0.0
	if n > 0 {
		rms = math.Sqrt(sumSq / float64(n))
	}
	t.Logf("RESULT: %.0fs of audio · RMS=%.1f (under %.0f is silence, >200 is speech) · %d/%d windows quiet · LONGEST QUIET RUN %.2fs",
		float64(windows)*0.25, rms, silenceFloor, quiet, windows, float64(longest)*0.25)
}

const (
	silenceFloor = 10.0 // RMS below this is silence; speech reads in the hundreds
	diagFor      = 30 * time.Second
)
