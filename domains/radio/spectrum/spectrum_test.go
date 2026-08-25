package spectrum

import (
	"math"
	"math/cmplx"
	"testing"
)

func sine(hz float64, rate, n int, amp float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = amp * math.Sin(2*math.Pi*hz*float64(i)/float64(rate))
	}
	return out
}

func TestFFTMatchesNaiveDFT(t *testing.T) {
	// A radix-2 FFT of a small vector equals the textbook DFT.
	n := 16
	in := make([]complex128, n)
	for i := range in {
		in[i] = complex(math.Sin(float64(i))+0.3*float64(i%3), 0)
	}
	want := make([]complex128, n)
	for k := range want {
		for i := range in {
			want[k] += in[i] * cmplx.Exp(complex(0, -2*math.Pi*float64(k*i)/float64(n)))
		}
	}
	got := append([]complex128(nil), in...)
	fft(got, twiddles(n))
	for k := range got {
		if cmplx.Abs(got[k]-want[k]) > 1e-9 {
			t.Fatalf("bin %d: fft %v dft %v", k, got[k], want[k])
		}
	}
}

func TestBandsPeakWhereTheToneIs(t *testing.T) {
	// UAT 92: a 1 kHz tone lights the band that spans 900–1400 Hz above every
	// other; a full-scale tone saturates it; levels are clamped to 0..1.
	a, err := New(44100)
	if err != nil {
		t.Fatal(err)
	}
	bands := a.Bands(sine(1000, 44100, FFTSize, 0.5))
	if len(bands) != Bands {
		t.Fatalf("want %d bands, got %d", Bands, len(bands))
	}
	peak := 0
	for i, v := range bands {
		if v < 0 || v > 1 {
			t.Fatalf("band %d out of range: %v", i, v)
		}
		if v > bands[peak] {
			peak = i
		}
	}
	if want := bandFor(1000); peak != want {
		t.Fatalf("1 kHz must peak in band %d (%v–%v Hz), got %d: %v", want, edges()[want], edges()[want+1], peak, bands)
	}
	if bands[peak] < 0.5 {
		t.Fatalf("a half-scale tone reads well up the bar, got %.2f", bands[peak])
	}
}

func TestBandsAttackFastAndDecaySlowThenGateOnSilence(t *testing.T) {
	// The smoothing follows CLIAmp: fast attack (60 % new), slow decay
	// (25 % new); silence decays what is left by 20 % per frame without an FFT.
	a, err := New(44100)
	if err != nil {
		t.Fatal(err)
	}
	loud := sine(1000, 44100, FFTSize, 0.9)
	b := bandFor(1000)
	first := a.Bands(loud)[b]
	second := a.Bands(loud)[b]
	if !(second > first) {
		t.Fatalf("attack: the second frame of a sustained tone is higher (%.3f → %.3f)", first, second)
	}
	quiet := a.Bands(nil)[b]
	if math.Abs(quiet-second*0.8) > 1e-9 {
		t.Fatalf("silence gate: ×0.8 per frame, want %.4f got %.4f", second*0.8, quiet)
	}
	for i := 0; i < 200 && a.Bands(nil)[b] > 0.001; i++ { // bounded decay
	}
	if v := a.Bands(nil)[b]; v > 0.001 {
		t.Fatalf("bars settle to rest in silence, still %.4f", v)
	}
	if _, err := New(0); err == nil {
		t.Fatal("an analyzer needs a sample rate")
	}
}

func TestBandsReturnsAFreshSlice(t *testing.T) {
	a, _ := New(44100)
	x := a.Bands(sine(500, 44100, FFTSize, 0.5))
	y := a.Bands(sine(500, 44100, FFTSize, 0.5))
	x[0] = 42
	if y[0] == 42 {
		t.Fatal("the caller may keep a frame; the analyzer never aliases it")
	}
}

// BenchmarkBands pins the per-frame cost (the visualizer runs this on the
// update loop 20× a second — it must stay far below a millisecond).
func BenchmarkBands(b *testing.B) {
	a, _ := New(44100)
	in := sine(1000, 44100, FFTSize, 0.5)
	for b.Loop() {
		a.Bands(in)
	}
}

// bandFor locates the band whose edges span hz.
func bandFor(hz float64) int {
	e := edges()
	for i := 0; i < len(e)-1; i++ {
		if hz >= e[i] && hz < e[i+1] {
			return i
		}
	}
	return -1
}
