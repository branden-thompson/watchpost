package synth

import (
	"context"
	"encoding/binary"
	"math"
	"time"
)

// The alert attention signal.
//
// 0.12.0 had one tone: three ~1 kHz pulses then a 2 s pause, the shape the HUM
// LEAD specified. 0.14.0 keeps that sound as Classic and makes it one of a
// family, because a listener away from the screen should hear WHICH KIND of
// alert is coming before any words arrive (FR-11).
//
// A preset is PARAMETERS, never a recorded sample: parameters are reviewable
// against `02-analysis/tones.md` §1, reproducible on any rate, and cost about a
// millisecond to render — which is why the alert path can sound one from
// constants without resolving, installing or waiting for a voice (FR-9).
// The preset names, matching domains/radio/cast's ToneName. They are declared
// here beside the presets so a name and its sound cannot drift apart; cast
// holds the same strings because it must not import this package.
const (
	PresetNameDualTone  = "dual-tone"
	PresetName1050      = "1050hz"
	PresetNameClassic   = "classic"
	PresetNameSoftChime = "soft-chime"
	PresetNameLowSweep  = "low-sweep"
)

const (
	// ToneRate is the sample rate every tone renders at. The tone is generated,
	// not spoken, so it does not follow a voice's rate — and it must not, or an
	// alert's sound would change with the correspondent reading it.
	ToneRate = 22050

	// alertTailDur is the pause between the signal and the first word. Every
	// preset carries it inside its own buffer, so a caller that holds for the
	// buffer's length holds for the pause too (app/ticker.go's s.hold).
	alertTailDur = 2 * time.Second

	// alertEnvDur is the attack/release ramp on a pulse, so it opens and closes
	// without a click.
	alertEnvDur = 8 * time.Millisecond
)

// Preset is one alert sound, as parameters.
//
// The zero value renders nothing: a preset must be named and given at least one
// frequency and a positive duration. That is deliberate — a silently-empty tone
// on an alert path is the failure this whole feature exists to prevent, so it
// is made impossible to reach by accident rather than merely unlikely.
type Preset struct {
	Name string

	// Freqs are summed and divided by their count, so a two-tone preset is as
	// loud as a single-tone one.
	Freqs []float64

	// Pulses is how many times the signal sounds; PulseDur is each one's
	// length and GapDur the silence between them. One pulse with no gap is a
	// sustained tone.
	Pulses   int
	PulseDur time.Duration
	GapDur   time.Duration

	// SweepTo, when non-zero, linearly sweeps from Freqs[0] to it across each
	// pulse — the horn-like shape.
	SweepTo float64

	// Decay, when non-zero, applies an exponential decay with this time
	// constant across each pulse — the chime shape. It replaces the release
	// ramp; the attack ramp still applies.
	Decay time.Duration

	// Amp is the peak amplitude as a fraction of full scale.
	Amp float64
}

// Classic is 0.12.0's tone, unchanged: three enveloped 1 kHz pulses, 200 ms
// each, 100 ms apart. It precedes Advisories, and it is the fallback for any
// preset name this build does not know — never silence.
//
// A function rather than a package var: a var would be mutable process-wide
// state, and every caller wants a fresh value anyway.
func Classic() Preset {
	return Preset{
		Name:     PresetNameClassic,
		Freqs:    []float64{1000},
		Pulses:   3,
		PulseDur: 200 * time.Millisecond,
		GapDur:   100 * time.Millisecond,
		Amp:      0.45, // present but not harsh under a ducked broadcast
	}
}

// PresetByName resolves a preset name — the strings domains/radio/cast's
// ToneName returns.
//
// An unknown name yields Classic, the LOUD default, and never nil. A class
// whose preset this build does not know must still be ANNOUNCED — falling back
// to silence would make an unrecognised alert the quietest one.
func PresetByName(name string) Preset {
	for _, p := range Presets() {
		if p.Name == name {
			return p
		}
	}
	return Classic()
}

// DualTone is the loudest signal, and the one two classes share: Significant
// Quakes & Disasters and Warnings (MVS-D-28). It follows the Emergency Alert
// System attention signal — two tones summed and sustained — because that is
// the sound a listener already reads as "stop what you are doing".
func DualTone() Preset {
	return Preset{
		Name:     PresetNameDualTone,
		Freqs:    []float64{853, 960},
		Pulses:   1,
		PulseDur: 2 * time.Second,
		Amp:      0.45,
	}
}

// Tone1050 precedes Watches. The style is NOAA Weather Radio's warning alarm
// tone; the HUM LEAD assigned it to Watches deliberately (MVS-D-11), so it is
// borrowed as a SOUND, not as a meaning.
func Tone1050() Preset {
	return Preset{
		Name:     PresetName1050,
		Freqs:    []float64{1050},
		Pulses:   1,
		PulseDur: 2 * time.Second,
		Amp:      0.45,
	}
}

// SoftChime precedes Special Weather Statements: a public-address chime, an
// octave pair with an exponential decay. Quieter and shorter on purpose — a
// statement is information, not an instruction.
func SoftChime() Preset {
	return Preset{
		Name:     PresetNameSoftChime,
		Freqs:    []float64{880, 1760},
		Pulses:   1,
		PulseDur: 1200 * time.Millisecond,
		Decay:    350 * time.Millisecond,
		Amp:      0.40,
	}
}

// LowSweep precedes the Tropical / Winter Storm class (the Setup label
// "Maritime"): a horn-like rising sweep, twice.
func LowSweep() Preset {
	return Preset{
		Name:     PresetNameLowSweep,
		Freqs:    []float64{330},
		SweepTo:  520,
		Pulses:   2,
		PulseDur: 700 * time.Millisecond,
		GapDur:   200 * time.Millisecond,
		Amp:      0.45,
	}
}

// Presets is every preset this build can sound, in the order tones.md §1 lists
// them.
func Presets() []Preset {
	return []Preset{DualTone(), Tone1050(), Classic(), SoftChime(), LowSweep()}
}

// AlertTone renders a preset as 16-bit little-endian STEREO PCM at rate,
// followed by the 2 s pause before narration.
//
// It validates its own edges and returns nil rather than panicking on any of
// them: rate, pulse count and pulse duration all arrive from data, and a tone
// is rendered on the path an alert is waiting on. nil means "no tone"; the
// words still read.
func AlertTone(p Preset, rate int) []byte {
	if rate <= 0 || p.Pulses <= 0 || p.PulseDur <= 0 || len(p.Freqs) == 0 || p.Amp <= 0 {
		return nil
	}
	samples := func(d time.Duration) int { return int(d.Seconds() * float64(rate)) }
	pulseN, gapN := samples(p.PulseDur), samples(p.GapDur)
	envN, tailN := samples(alertEnvDur), samples(alertTailDur)
	if pulseN <= 0 {
		return nil
	}

	// One allocation for the whole tone: the gaps and the tail are written into
	// the pre-sized buffer rather than appended from throwaway slices, so a
	// tone on the alert path allocates once regardless of its shape.
	total := (p.Pulses*pulseN + max(p.Pulses-1, 0)*gapN + tailN) * 4
	out := make([]byte, 0, total)
	for i := range p.Pulses {
		out = appendPulse(out, p, rate, pulseN, envN)
		if i < p.Pulses-1 {
			out = appendSilence(out, gapN)
		}
	}
	return appendSilence(out, tailN)
}

// appendPulse writes one pulse: the summed frequencies, the attack ramp, and
// either an exponential decay or a release ramp.
func appendPulse(out []byte, p Preset, rate, pulseN, envN int) []byte {
	for i := range pulseN {
		t := float64(i) / float64(rate)
		s := p.Amp * envelope(p, i, pulseN, envN, t) * wave(p, t, float64(i)/float64(pulseN))
		v := uint16(int16(s * math.MaxInt16))
		out = binary.LittleEndian.AppendUint16(out, v) // left
		out = binary.LittleEndian.AppendUint16(out, v) // right
	}
	return out
}

// wave is the instantaneous sample of the preset's frequencies at time t.
// progress (0..1) drives the sweep.
func wave(p Preset, t, progress float64) float64 {
	var sum float64
	for _, f := range p.Freqs {
		if p.SweepTo > 0 && len(p.Freqs) == 1 {
			f += (p.SweepTo - f) * progress
		}
		sum += math.Sin(2 * math.Pi * f * t)
	}
	return sum / float64(len(p.Freqs))
}

// envelope shapes one pulse: an attack ramp always, then either an exponential
// decay (a chime) or a release ramp (a pulse).
func envelope(p Preset, i, pulseN, envN int, t float64) float64 {
	if envN > 0 && i < envN {
		return float64(i) / float64(envN) // attack
	}
	if p.Decay > 0 {
		return math.Exp(-t / p.Decay.Seconds())
	}
	if envN > 0 && i >= pulseN-envN {
		return float64(pulseN-i) / float64(envN) // release
	}
	return 1
}

// appendSilence extends out by n silent stereo frames.
func appendSilence(out []byte, n int) []byte {
	if n <= 0 {
		return out
	}
	return append(out, make([]byte, n*4)...)
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
