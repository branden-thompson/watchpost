package player

import (
	"bytes"
	"io"
	"testing"
)

// RS-16 (architecture §10.4/§10.9): the relay decode path takes bytes from
// the network. These fuzzers pin that hostile input can only produce an
// error, never a panic or unbounded memory — for the ICY metadata stripper,
// the preroll buffer and the resampler. `go test -fuzz=FuzzICYReader
// ./domains/radio/player` runs them open-ended; the seed corpus runs in
// every `go test`.

func FuzzICYReader(f *testing.F) {
	f.Add([]byte("audio\x00"), 5)
	f.Add([]byte("aaaa\x01StreamTitle='x';\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00bbbb"), 4)
	f.Add([]byte("\xff\xff\xff"), 1)
	f.Add([]byte{}, 16000)
	f.Fuzz(func(t *testing.T, data []byte, metaint int) {
		if metaint <= 0 || metaint > 1<<20 {
			metaint = 1 + (metaint&0xffff+0x10000)%0xffff // keep it positive and bounded
		}
		s := &Stream{}
		r := newIcyReader(bytes.NewReader(data), metaint, s)
		out, err := io.ReadAll(io.LimitReader(r, 1<<22)) // audio bytes ≤ input bytes
		if len(out) > len(data) {
			t.Fatalf("stripper produced more audio (%d) than input (%d)", len(out), len(data))
		}
		if title := s.Title(); len(title) > 4080 { // one metadata block is at most 16×255 bytes
			t.Fatalf("title exceeds a metadata block: %d", len(title))
		}
		_ = err // truncated metadata is an error, never a panic
	})
}

func FuzzPrerollReader(f *testing.F) {
	f.Add([]byte("0123456789"), 4)
	f.Add([]byte{}, 12*1024)
	f.Fuzz(func(t *testing.T, data []byte, want int) {
		if want < 0 || want > 1<<16 {
			want = 1024
		}
		r := &prerollReader{r: bytes.NewReader(data), want: want}
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("preroll must pass bytes through: %v", err)
		}
		if !bytes.Equal(out, data) {
			t.Fatalf("preroll must not alter the stream")
		}
	})
}

func FuzzResampler(f *testing.F) {
	f.Add([]byte{1, 0, 2, 0, 3, 0, 4, 0}, 22050)
	f.Add([]byte{0xff}, 8000)
	f.Add([]byte{}, 48000)
	f.Fuzz(func(t *testing.T, pcm []byte, rate int) {
		if rate <= 0 || rate > 192000 {
			rate = 22050
		}
		out, _ := io.ReadAll(io.LimitReader(newResampler(bytes.NewReader(pcm), rate), 1<<22))
		if len(out)%4 != 0 {
			t.Fatalf("output must be whole stereo frames, got %d bytes", len(out))
		}
		// Output frames scale with the rate ratio (+1 frame of slack), never explode.
		maxFrames := (len(pcm)/4)*OutputRate/rate + 2
		if len(out)/4 > maxFrames {
			t.Fatalf("resampler produced %d frames from %d input frames at %d Hz", len(out)/4, len(pcm)/4, rate)
		}
	})
}
