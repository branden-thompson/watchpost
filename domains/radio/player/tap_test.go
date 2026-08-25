package player

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"math"
	"testing"
	"time"
)

// stereoPCM renders n frames of a full-scale mono value into 16-bit LE stereo.
func stereoPCM(vals []int16) []byte {
	out := make([]byte, 0, len(vals)*4)
	for _, v := range vals {
		out = binary.LittleEndian.AppendUint16(out, uint16(v))
		out = binary.LittleEndian.AppendUint16(out, uint16(v))
	}
	return out
}

func TestTapKeepsTheLatestFramesAsMono(t *testing.T) {
	// UAT 92: the tap copies what passes through into a ring; Samples yields
	// the most recent frames (oldest first) scaled to ±1, mixing L+R.
	tap, err := NewTap(4)
	if err != nil {
		t.Fatal(err)
	}
	src := stereoPCM([]int16{1000, 2000, 3000, 4000, 5000, 6000})
	r := tap.Wrap(bytes.NewReader(src))
	got, _ := io.ReadAll(r)
	if !bytes.Equal(got, src) {
		t.Fatal("the tap must pass PCM through untouched")
	}
	dst := make([]float64, 4)
	if n := tap.Samples(dst); n != 4 {
		t.Fatalf("ring of 4 after 6 frames yields 4, got %d", n)
	}
	for i, want := range []float64{3000, 4000, 5000, 6000} {
		if math.Abs(dst[i]-want/32768) > 1e-9 {
			t.Fatalf("sample %d: want %.4f got %.4f (latest frames, oldest first)", i, want/32768, dst[i])
		}
	}
	// Fewer frames than asked: only what exists.
	tap.Reset()
	if n := tap.Samples(dst); n != 0 {
		t.Fatalf("after Reset nothing is available, got %d", n)
	}
	_, _ = io.ReadAll(tap.Wrap(bytes.NewReader(stereoPCM([]int16{-32768, 32767}))))
	if n := tap.Samples(dst); n != 2 || dst[0] != -1 || math.Abs(dst[1]-32767.0/32768) > 1e-9 {
		t.Fatalf("2 frames → 2 samples at ±1: n=%d %v", n, dst[:2])
	}
	if _, err := NewTap(0); err == nil {
		t.Fatal("a tap needs room")
	}
}

func TestEngineTapsPlaybackAndClearsOnHalt(t *testing.T) {
	// UAT 92: whatever the engine plays (relay or synth) is visible to the
	// visualizer through Samples; Halt clears the ring so the bars decay.
	out := &fakeOutput{}
	e, err := New(out, "watchpost-test", nil)
	if err != nil {
		t.Fatal(err)
	}
	tone := make([]int16, OutputRate) // one second of a loud square-ish tone
	for i := range tone {
		if i%20 < 10 {
			tone[i] = 20000
		} else {
			tone[i] = -20000
		}
	}
	e.StartSource("tone", OutputRate, func(context.Context) io.Reader { return bytes.NewReader(stereoPCM(tone)) })
	dst := make([]float64, 512)
	deadline := time.Now().Add(2 * time.Second)
	// Bounded: polls until the tap has audio or the deadline passes.
	for i := 0; i < 200 && time.Now().Before(deadline); i++ {
		if n := e.Samples(dst); n == 512 && math.Abs(dst[0]) > 0.5 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if n := e.Samples(dst); n != 512 || math.Abs(dst[0]) < 0.5 {
		t.Fatalf("the tap must see the playing tone: n=%d first=%.3f", n, dst[0])
	}
	e.Halt()
	if n := e.Samples(dst); n != 0 {
		t.Fatalf("Halt clears the tap, got %d samples", n)
	}
}
