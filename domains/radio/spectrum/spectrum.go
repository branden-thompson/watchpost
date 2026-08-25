// Package spectrum turns the player's latest PCM into visualizer band levels
// (UAT 92). The analysis follows CLIAmp's default spectrum: a Hann-windowed
// radix-2 FFT, log-spaced bands averaged over their bins, a dB-like scale
// into 0..1, fast-attack / slow-decay smoothing, and a silence gate that
// decays the bars without an FFT. The band edges are tuned to a voice
// broadcast (NWR relays are 32 kbps speech; Piper renders at 22.05 kHz):
// CLIAmp's music edges run to 20 kHz, where a weather broadcast has nothing,
// and would leave the top bars flat forever.
package spectrum

import (
	"math"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

const (
	// Bands is how many levels a frame carries.
	Bands = 10
	// FFTSize is the analysis window (2048 samples ≈ 46 ms at 44.1 kHz).
	FFTSize = 2048
)

// edges are the band boundaries in Hz — voice-weighted, log-ish spacing.
func edges() []float64 {
	return []float64{50, 100, 200, 350, 550, 900, 1400, 2200, 3500, 5500, 9000}
}

// Analyzer holds the FFT scratch and the smoothing state.
type Analyzer struct {
	rate   float64
	window []float64
	tw     []complex128
	buf    []complex128
	power  []float64
	prev   []float64
}

// New builds an analyzer for PCM at rate Hz.
func New(rate int) (*Analyzer, error) {
	if err := invariant.Check(rate > 0, "spectrum: sample rate must be positive"); err != nil {
		return nil, err
	}
	return &Analyzer{
		rate:   float64(rate),
		window: hann(FFTSize),
		tw:     twiddles(FFTSize),
		buf:    make([]complex128, FFTSize),
		power:  make([]float64, FFTSize/2),
		prev:   make([]float64, Bands),
	}, nil
}

// Bands analyses the latest samples (±1, up to FFTSize; fewer or none is
// silence) and returns a fresh frame of Bands levels in 0..1.
func (a *Analyzer) Bands(samples []float64) []float64 {
	out := make([]float64, Bands)
	if silent(samples) {
		for b := range a.prev {
			a.prev[b] *= 0.8 // the gate: bars settle without an FFT
			out[b] = a.prev[b]
		}
		return out
	}
	have := min(len(samples), FFTSize)
	for i := range FFTSize {
		if i < have {
			a.buf[i] = complex(samples[i]*a.window[i], 0)
		} else {
			a.buf[i] = 0
		}
	}
	fft(a.buf, a.tw)
	a.power[0] = 0
	for i := 1; i < len(a.power); i++ {
		re, im := real(a.buf[i]), imag(a.buf[i])
		a.power[i] = re*re + im*im
	}
	binHz := a.rate / FFTSize
	e := edges()
	for b := range Bands {
		level := 0.0
		if p := a.bandPower(e[b]/binHz, e[b+1]/binHz); p > 0 {
			level = (10*math.Log10(p) + 10) / 50 // 10·log10(power) == 20·log10(magnitude)
		}
		level = max(0, min(1, level))
		if level > a.prev[b] {
			level = level*0.6 + a.prev[b]*0.4 // fast attack
		} else {
			level = level*0.25 + a.prev[b]*0.75 // slow decay
		}
		a.prev[b] = level
		out[b] = level
	}
	return out
}

// bandPower averages the power bins in [lo, hi) (fractional bin positions;
// at least one bin).
func (a *Analyzer) bandPower(lo, hi float64) float64 {
	first := max(1, int(lo))
	last := max(first+1, min(len(a.power), int(hi)))
	sum := 0.0
	for i := first; i < last; i++ { // bounded by the half spectrum
		sum += a.power[i]
	}
	return sum / float64(last-first)
}

// silent is the quick max-abs scan that lets the gate skip the FFT.
func silent(samples []float64) bool {
	for _, s := range samples {
		if math.Abs(s) >= 1e-5 {
			return false
		}
	}
	return true
}

func hann(n int) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
	}
	return w
}

// twiddles are the first n/2 complex n-th roots of unity, w[k] = e^(-2πik/n).
func twiddles(n int) []complex128 {
	w := make([]complex128, n/2)
	for k := range w {
		angle := -2 * math.Pi * float64(k) / float64(n)
		w[k] = complex(math.Cos(angle), math.Sin(angle))
	}
	return w
}

// fft is an in-place radix-2 Cooley–Tukey transform; len(buf) is a power
// of two and w holds len(buf)/2 twiddles. Allocates nothing.
func fft(buf []complex128, w []complex128) {
	n := len(buf)
	if n < 2 {
		return
	}
	// Bit-reversal permutation.
	for i, j := 1, 0; i < n; i++ { // bounded by n
		bit := n >> 1
		for ; j&bit != 0 && bit > 0; bit >>= 1 { // bounded by log2(n)
			j ^= bit
		}
		j ^= bit
		if i < j {
			buf[i], buf[j] = buf[j], buf[i]
		}
	}
	// Butterflies; the stride into the shared table is n/size.
	for size := 2; size <= n; size <<= 1 { // bounded by log2(n) stages
		half, step := size>>1, n/size
		for start := 0; start < n; start += size {
			for k := range half {
				t := w[k*step] * buf[start+k+half]
				u := buf[start+k]
				buf[start+k] = u + t
				buf[start+k+half] = u - t
			}
		}
	}
}
