package synth

import (
	"context"
	"encoding/binary"
	"math"
	"time"
)

// The severe-alert attention signal (0.12.0 global ticker): three short
// ~1 kHz pulses with a gap between them, then a 2 s pause before the spoken
// narration — the shape HUM LEAD specified ("tone tone tone <2 s> …"). The
// pulses carry a short attack/release envelope so they open and close without
// a click.
const (
	alertToneHz   = 1000.0
	alertPulses   = 3
	alertPulseDur = 200 * time.Millisecond
	alertGapDur   = 100 * time.Millisecond
	alertTailDur  = 2 * time.Second
	alertToneAmp  = 0.45 // of full scale — present but not harsh under a ducked broadcast
	alertEnvDur   = 8 * time.Millisecond
)

// AlertTone renders the attention signal — three enveloped ~1 kHz pulses then
// a 2 s pause — as 16-bit little-endian STEREO PCM at rate. It is the lead-in
// the ticker plays before the narration (0.12.0). A non-positive rate yields
// no tone.
func AlertTone(rate int) []byte {
	if rate <= 0 {
		return nil
	}
	samples := func(d time.Duration) int { return int(d.Seconds() * float64(rate)) }
	pulseN, gapN, envN, tailN := samples(alertPulseDur), samples(alertGapDur), samples(alertEnvDur), samples(alertTailDur)

	out := make([]byte, 0, (alertPulses*pulseN+(alertPulses-1)*gapN+tailN)*4)
	for p := 0; p < alertPulses; p++ {
		for i := 0; i < pulseN; i++ {
			env := 1.0
			if envN > 0 {
				if i < envN {
					env = float64(i) / float64(envN) // attack
				} else if i >= pulseN-envN {
					env = float64(pulseN-i) / float64(envN) // release
				}
			}
			s := alertToneAmp * env * math.Sin(2*math.Pi*alertToneHz*float64(i)/float64(rate))
			v := uint16(int16(s * math.MaxInt16))
			out = binary.LittleEndian.AppendUint16(out, v) // left
			out = binary.LittleEndian.AppendUint16(out, v) // right
		}
		if p < alertPulses-1 {
			out = append(out, make([]byte, gapN*4)...) // the inter-pulse gap
		}
	}
	return append(out, make([]byte, tailN*4)...) // the 2 s pause before narration
}

// AlertNarration renders alert narration text in a voice as 16-bit LE stereo
// PCM at the voice's rate — the ticker's spoken line after AlertTone. Product
// text is untrusted; ExpandStates reads "VA" as "Virginia" and Pronounce
// applies the voice-only spellings (the same order the broadcast uses), and the
// Voice adapters keep the text out of argv (§10.5).
func AlertNarration(ctx context.Context, v Voice, text string) ([]byte, error) {
	mono, err := v.Say(ctx, Pronounce(ExpandStates(text)))
	if err != nil {
		return nil, err
	}
	return monoToStereo(mono), nil
}
