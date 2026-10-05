package radar

// fuzz_test.go — MRMS's capabilities and every source's frames are read from
// the network, so both readers are fuzzed from the recorded answers: no
// panic; the times a source advertises are some, oldest first (Source's
// Times); a frame passed is a PNG within the library's image cap, and one
// called empty paints nothing.

import (
	"bytes"
	"image/png"
	"testing"
)

func FuzzMRMSTimes(f *testing.F) {
	f.Add(fixture(f, "mrms-capabilities.xml"))
	f.Fuzz(func(t *testing.T, body []byte) {
		times, err := mrmsTimes(body)
		if err != nil {
			return
		}
		if len(times) == 0 {
			t.Fatal("no time and no error")
		}
		for i := 1; i < len(times); i++ {
			if times[i].Before(times[i-1]) {
				t.Fatalf("the times are not oldest first: %v then %v", times[i-1], times[i])
			}
		}
	})
}

func FuzzCheck(f *testing.F) {
	for _, name := range []string{"mrms-frame.png", "mrms-expired.png", "iem-frame.png", "iem-offgrid.png", "hrrr-frame.png"} {
		f.Add(fixture(f, name))
	}
	f.Fuzz(func(t *testing.T, frame []byte) {
		empty, err := Check(frame)
		if err != nil {
			if empty {
				t.Fatal("a refused frame is called empty")
			}
			return
		}
		cfg, cerr := png.DecodeConfig(bytes.NewReader(frame))
		if cerr != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixels {
			t.Fatalf("a frame of %dx%d (%v) passed the cap of %d pixels", cfg.Width, cfg.Height, cerr, maxPixels)
		}
		if !empty {
			return
		}
		img, derr := png.Decode(bytes.NewReader(frame))
		if derr != nil {
			t.Fatalf("an empty frame does not decode: %v", derr)
		}
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
					t.Fatalf("a frame called empty paints at %d,%d", x, y)
				}
			}
		}
	})
}
