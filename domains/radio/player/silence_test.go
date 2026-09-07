package player

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"
	"time"
)

// pcm builds d worth of 16-bit stereo at rate: silent when amp is 0, a tone
// otherwise. The tone is what a real broadcast looks like to the detector —
// amplitude well above the floor.
func pcm(rate int, d time.Duration, amp float64) []byte {
	frames := int(float64(rate) * d.Seconds())
	b := make([]byte, 0, frames*4)
	for i := 0; i < frames; i++ {
		v := int16(amp * math.Sin(2*math.Pi*440*float64(i)/float64(rate)))
		var s [2]byte
		binary.LittleEndian.PutUint16(s[:], uint16(v))
		b = append(b, s[0], s[1], s[0], s[1]) // both channels
	}
	return b
}

// A DEAD RELAY IS REPORTED, AND ONLY AFTER THE FULL WINDOW.
//
// The numbers are the HUM LEAD's ruling (MVS-D-76): five seconds of silence.
// Counting bytes rather than seconds is what lets this be exact — four seconds
// of silence must NOT fire, and the test does not have to wait four seconds to
// say so.
func TestSilenceIsReportedOnlyAfterTheRuledWindow(t *testing.T) {
	const rate = 44100
	for _, tc := range []struct {
		name string
		d    time.Duration
		want bool
	}{
		{"four seconds is not yet a fault", 4 * time.Second, false},
		{"five seconds is", 5 * time.Second, true},
		{"six seconds certainly is", 6 * time.Second, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fired := 0
			r := newSilenceReader(bytes.NewReader(pcm(rate, tc.d, 0)), rate, 5*time.Second, func() { fired++ })
			if _, err := io.Copy(io.Discard, r); err != nil {
				t.Fatal(err)
			}
			if got := fired > 0; got != tc.want {
				t.Errorf("silence for %v fired=%v, want %v", tc.d, got, tc.want)
			}
		})
	}
}

// A WORKING BROADCAST NEVER FIRES.
//
// Measured on a healthy mount (KIH62 via wxradio.org, 2026-09-04): 406 windows
// over 102 seconds, ZERO of them under the floor. NWR runs a continuous loop
// over a constant noise floor, so this is not a close call — but a detector on
// the safety path has to be shown not to fire on the good case, not assumed not
// to.
func TestAudibleAudioNeverReportsSilence(t *testing.T) {
	const rate = 44100
	fired := 0
	r := newSilenceReader(bytes.NewReader(pcm(rate, 30*time.Second, 3000)), rate, 5*time.Second, func() { fired++ })
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Fatal(err)
	}
	if fired != 0 {
		t.Errorf("a 30 s broadcast reported silence %d times", fired)
	}
}

// IT IS A RUN, NOT A TOTAL.
//
// A station with pauses in it accumulates plenty of quiet windows without ever
// being dead. Summing them instead of measuring the unbroken run would take a
// perfectly good transmitter off the air after enough ordinary gaps.
func TestScatteredQuietIsNotAFault(t *testing.T) {
	const rate = 44100
	var b []byte
	for i := 0; i < 8; i++ { // 8 x (4 s quiet + 1 s audible) = 32 s, 32 s of it quiet
		b = append(b, pcm(rate, 4*time.Second, 0)...)
		b = append(b, pcm(rate, time.Second, 3000)...)
	}
	fired := 0
	r := newSilenceReader(bytes.NewReader(b), rate, 5*time.Second, func() { fired++ })
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Fatal(err)
	}
	if fired != 0 {
		t.Errorf("32 s of scattered quiet, longest run 4 s, fired %d times", fired)
	}
}

// ONE REPORT PER STREAM. The listener is told once and offered a choice; a
// detector that fired every window would raise the modal over and over.
func TestADeadRelayIsReportedOnce(t *testing.T) {
	const rate = 44100
	fired := 0
	r := newSilenceReader(bytes.NewReader(pcm(rate, 60*time.Second, 0)), rate, 5*time.Second, func() { fired++ })
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Fatal(err)
	}
	if fired != 1 {
		t.Errorf("a minute of silence reported %d times, want exactly 1", fired)
	}
}

// The audio is passed through byte for byte. A detector that altered the stream
// it measures would be a new source of the very defect it looks for.
func TestTheAudioPassesThroughUntouched(t *testing.T) {
	const rate = 8000
	in := pcm(rate, 2*time.Second, 3000)
	var out bytes.Buffer
	r := newSilenceReader(bytes.NewReader(in), rate, 5*time.Second, func() {})
	if _, err := io.Copy(&out, r); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(in, out.Bytes()) {
		t.Errorf("the stream changed: %d bytes in, %d out", len(in), out.Len())
	}
}
