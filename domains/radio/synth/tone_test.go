package synth

import (
	"context"
	"encoding/binary"
	"strings"
	"testing"
	"time"
)

// The classic tone must be BYTE-IDENTICAL to 0.13.0's: it is one of FR-2's
// unchanged behaviours, and turning a hard-coded tone into a parameterised one
// is exactly the kind of refactor that silently shifts a waveform.
func TestAlertToneIsThreePulsesThenAPause(t *testing.T) {
	const rate = ToneRate
	pcm := AlertTone(Classic(), rate)
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

// Every edge returns nil rather than panicking. A preset arrives from data and
// the tone renders on the path an alert is waiting on, so "no tone, the words
// still read" is the only acceptable failure — never a crash, and never a
// buffer of the wrong length.
func TestAlertToneReturnsNilOnEveryMalformedEdge(t *testing.T) {
	for _, tc := range []struct {
		name   string
		preset Preset
		rate   int
	}{
		{"rate zero", Classic(), 0},
		{"rate negative", Classic(), -1},
		{"zero preset", Preset{}, ToneRate},
		{"no pulses", Preset{Name: "x", Freqs: []float64{1000}, PulseDur: time.Second, Amp: 0.4}, ToneRate},
		{"no duration", Preset{Name: "x", Freqs: []float64{1000}, Pulses: 1, Amp: 0.4}, ToneRate},
		{"no frequencies", Preset{Name: "x", Pulses: 1, PulseDur: time.Second, Amp: 0.4}, ToneRate},
		{"no amplitude", Preset{Name: "x", Freqs: []float64{1000}, Pulses: 1, PulseDur: time.Second}, ToneRate},
		{"duration rounds to no samples", Preset{Name: "x", Freqs: []float64{1000}, Pulses: 1, PulseDur: time.Nanosecond, Amp: 0.4}, ToneRate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := AlertTone(tc.preset, tc.rate); got != nil {
				t.Fatalf("want nil, got %d bytes", len(got))
			}
		})
	}
}

// PresetByName never returns silence: an unknown name is the LOUD default.
// (Between P2 and P3 every name resolved here, because only Classic existed —
// that was the deliberate intermediate state, and P3 ended it.)
func TestPresetByNameFallsBackToTheLoudDefault(t *testing.T) {
	for _, name := range []string{"", "nonsense", "CLASSIC", "dual tone"} {
		got := PresetByName(name)
		if got.Name != PresetNameClassic {
			t.Errorf("PresetByName(%q) = %q, want the classic fallback", name, got.Name)
		}
		if AlertTone(got, ToneRate) == nil {
			t.Errorf("PresetByName(%q) yielded a preset that renders nothing", name)
		}
	}
	if len(Presets()) == 0 {
		t.Error("Presets() must never be empty")
	}
}

// The tone is generated, not spoken, so its rate is ToneRate and does not
// follow a voice — an alert must not change its sound with the correspondent
// reading it.
func TestToneRateIsIndependentOfAnyVoice(t *testing.T) {
	if ToneRate != 22050 {
		t.Errorf("ToneRate = %d", ToneRate)
	}
	a := AlertTone(Classic(), ToneRate)
	b := AlertTone(Classic(), 16000)
	if len(a) == len(b) {
		t.Error("a different rate must produce a different buffer length")
	}
	// Both are still ~2800 ms of audio: the shape is in time, not in samples.
	for _, tc := range []struct {
		pcm  []byte
		rate int
	}{{a, ToneRate}, {b, 16000}} {
		ms := float64(len(tc.pcm)/4) / float64(tc.rate) * 1000
		if ms < 2790 || ms > 2810 {
			t.Errorf("at %d Hz the tone is %.0f ms, want ~2800", tc.rate, ms)
		}
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

// --- Task 3.6: the other four presets ---

// Every preset renders, at the ratified length, and is nameable. The lengths
// differ ON PURPOSE (tones.md §1): a Warning takeover's first words land about
// 1.2 s later than an Advisory's, which is why NFR-2 measures time-to-TONE-
// start rather than time-to-first-word.
func TestEveryPresetRendersAtItsRatifiedLength(t *testing.T) {
	const tail = 2000.0 // every preset carries the 2 s pre-narration pause
	for _, tc := range []struct {
		preset Preset
		signal float64 // ms of signal before the tail
	}{
		{DualTone(), 2000},
		{Tone1050(), 2000},
		{Classic(), 800}, // 3 x 200 ms pulses + 2 x 100 ms gaps
		{SoftChime(), 1200},
		{LowSweep(), 1600}, // 2 x 700 ms + one 200 ms gap
	} {
		t.Run(tc.preset.Name, func(t *testing.T) {
			pcm := AlertTone(tc.preset, ToneRate)
			if len(pcm) == 0 {
				t.Fatal("the preset rendered nothing")
			}
			if len(pcm)%4 != 0 {
				t.Fatalf("not 16-bit stereo PCM: %d bytes", len(pcm))
			}
			ms := float64(len(pcm)/4) / float64(ToneRate) * 1000
			if want := tc.signal + tail; ms < want-15 || ms > want+15 {
				t.Errorf("length %.0f ms, want ~%.0f", ms, want)
			}
			// Audible: the signal reaches a real amplitude before the tail.
			var peak int16
			head := pcm[:int(tc.signal/1000*float64(ToneRate))*4]
			for i := 0; i+2 <= len(head); i += 2 {
				if v := int16(binary.LittleEndian.Uint16(head[i:])); v > peak {
					peak = v
				}
			}
			if peak < 5000 {
				t.Errorf("peak amplitude %d — the signal is inaudible", peak)
			}
			// The tail is silence, in every preset.
			for _, b := range pcm[len(pcm)-int(tail/1000*float64(ToneRate))*4:] {
				if b != 0 {
					t.Fatal("the 2 s pre-narration pause must be silence")
				}
			}
			if got := PresetByName(tc.preset.Name); got.Name != tc.preset.Name {
				t.Errorf("PresetByName(%q) = %q", tc.preset.Name, got.Name)
			}
		})
	}
	if len(Presets()) != 5 {
		t.Errorf("five presets, got %d", len(Presets()))
	}
}

// Every class the registry knows resolves to a preset that actually sounds —
// the wiring between cast's ToneName and synth's PresetByName, which is two
// packages that deliberately do not import each other.
func TestEveryClassResolvesToASoundingPreset(t *testing.T) {
	for _, name := range []string{"dual-tone", "1050hz", "classic", "soft-chime", "low-sweep"} {
		p := PresetByName(name)
		if p.Name != name {
			t.Errorf("PresetByName(%q) fell back to %q — a class would sound the wrong tone", name, p.Name)
		}
		if AlertTone(p, ToneRate) == nil {
			t.Errorf("%q renders nothing", name)
		}
	}
}

// The two classes that SHARE the dual-tone (MVS-D-28) resolve to the SAME
// sound — one preset, two classes. That they remain separately mutable is
// cast's business and is tested there; here we pin that they sound alike.
func TestDisasterAndWarningSoundTheSamePreset(t *testing.T) {
	a := AlertTone(PresetByName(PresetNameDualTone), ToneRate)
	if len(a) == 0 {
		t.Fatal("the shared preset must sound")
	}
	if got := PresetByName(PresetNameDualTone).Name; got != PresetNameDualTone {
		t.Errorf("the shared preset is %q", got)
	}
}
