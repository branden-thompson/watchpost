package synth

import (
	"context"
	"encoding/binary"
	"strings"
	"testing"
)

func TestAlertToneIsThreePulsesThenAPause(t *testing.T) {
	const rate = 22050
	pcm := AlertTone(rate)
	if len(pcm)%4 != 0 {
		t.Fatalf("not 16-bit stereo PCM: %d bytes (want a multiple of 4)", len(pcm))
	}

	// Duration ≈ 3×200 ms pulses + 2×100 ms gaps + a 2 s pause = 2800 ms.
	frames := len(pcm) / 4
	gotMs := float64(frames) / float64(rate) * 1000
	if gotMs < 2790 || gotMs > 2810 {
		t.Fatalf("tone is %.0f ms, want ~2800 ms (3 pulses + gaps + a 2 s pause)", gotMs)
	}

	// Stereo: the two channels are identical per frame.
	for i := 0; i+4 <= len(pcm); i += 4 {
		l := int16(binary.LittleEndian.Uint16(pcm[i:]))
		r := int16(binary.LittleEndian.Uint16(pcm[i+2:]))
		if l != r {
			t.Fatalf("channels differ at frame %d: L=%d R=%d", i/4, l, r)
		}
	}

	// The pulses are audible in the first 800 ms.
	head := pcm[:int(0.8*float64(rate))*4]
	var peak int16
	for i := 0; i+2 <= len(head); i += 2 {
		if v := int16(binary.LittleEndian.Uint16(head[i:])); v > peak {
			peak = v
		}
	}
	if peak < 8000 {
		t.Fatalf("pulses too quiet: peak amplitude %d", peak)
	}

	// The final 2 s (the pause before narration) is silent.
	tail := pcm[len(pcm)-int(2.0*float64(rate))*4:]
	for i := range tail {
		if tail[i] != 0 {
			t.Fatalf("the 2 s pause must be silence; byte %d is %d", i, tail[i])
		}
	}
}

func TestAlertToneRejectsNonPositiveRate(t *testing.T) {
	if AlertTone(0) != nil || AlertTone(-1) != nil {
		t.Fatal("a non-positive rate yields no tone")
	}
}

// stubVoice returns fixed mono PCM regardless of text (the composition, not
// the synthesizer, is under test).
type stubVoice struct{ mono []byte }

func (stubVoice) Name() string                                      { return "Stub" }
func (stubVoice) Rate() int                                         { return 8000 }
func (v stubVoice) Say(_ context.Context, _ string) ([]byte, error) { return v.mono, nil }

// recordingVoice captures the text it is handed (after ExpandStates/Pronounce).
type recordingVoice struct{ got string }

func (recordingVoice) Name() string { return "Rec" }
func (recordingVoice) Rate() int    { return 8000 }
func (v *recordingVoice) Say(_ context.Context, text string) ([]byte, error) {
	v.got = text
	return []byte{0, 0}, nil
}

func TestAlertNarrationExpandsStatesForTheVoice(t *testing.T) {
	v := &recordingVoice{}
	if _, err := AlertNarration(context.Background(), v, "A Tornado Warning has been declared for Norfolk, VA at 3:42 PM"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(v.got, "Virginia") || strings.Contains(v.got, " VA ") {
		t.Fatalf("the voice reads the state in full: %q", v.got)
	}
}

func TestAlertNarrationDoublesMonoToStereo(t *testing.T) {
	// two mono samples: 1 and 2.
	v := stubVoice{mono: []byte{0x01, 0x00, 0x02, 0x00}}
	pcm, err := AlertNarration(context.Background(), v, "a tornado warning")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x01, 0x00, 0x01, 0x00, 0x02, 0x00, 0x02, 0x00} // each sample in both channels
	if len(pcm) != len(want) {
		t.Fatalf("stereo PCM is %d bytes, want %d", len(pcm), len(want))
	}
	for i := range want {
		if pcm[i] != want[i] {
			t.Fatalf("byte %d = %d, want %d", i, pcm[i], want[i])
		}
	}
}
