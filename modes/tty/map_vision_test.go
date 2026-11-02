package tty

// map_vision_test.go — F-198 (FR-7.3, A-107 (6)): every ramp the map draws
// for watchpost passes the library's colour-vision check, as a matrix of
// ramp, ground and depth, in watchpost's own tests.

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// visionRamps are the overlays watchpost hands the map with a ramp, built as
// the app builds them: each of the library's presets it draws, in each unit
// it offers.
func visionRamps(t *testing.T) map[string]tuimaps.Overlay {
	t.Helper()
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	grid := tuimaps.Grid{West: -120, South: 30, East: -115, North: 35, Cols: 2, Rows: 2, Values: []float64{10, 20, 30, 40}}
	var pic bytes.Buffer
	if err := png.Encode(&pic, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	radar := func(p tuimaps.Provider) tuimaps.Image {
		return tuimaps.Image{PNG: pic.Bytes(), Provider: p, West: -120, South: 30, East: -115, North: 35, Projection: tuimaps.PlateCarree}
	}
	return map[string]tuimaps.Overlay{
		"temperature C": tuimaps.TemperatureGrid("t", grid, tuimaps.Celsius, at),
		"temperature F": tuimaps.TemperatureGrid("t", grid, tuimaps.Fahrenheit, at),
		"wind mph":      tuimaps.WindGrid("t", grid, []float64{0, 90, 180, 270}, tuimaps.MilesPerHour, at),
		"wind m/s":      tuimaps.WindGrid("t", grid, []float64{0, 90, 180, 270}, tuimaps.MetresPerSecond, at),
		"waves ft":      tuimaps.WaveGrid("t", grid, tuimaps.Feet, at),
		"waves m":       tuimaps.WaveGrid("t", grid, tuimaps.Metres, at),
		"uv":            tuimaps.UVGrid("t", grid, at),
		"air":           tuimaps.AirQualityGrid("t", grid, at),
		"rain":          tuimaps.QPFGrid("t", grid, at),
		"muf":           tuimaps.MUFGrid("t", grid, at),
		"fof2":          tuimaps.FoF2Grid("t", grid, at),
		"radar IEM":     tuimaps.RadarImage("t", radar(tuimaps.ProviderIEM), at),
		"radar MRMS":    tuimaps.RadarImage("t", radar(tuimaps.ProviderMRMS), at),
	}
}

// groundSGR is a cell's background in truecolor.
var groundSGR = regexp.MustCompile(`48;2;(\d+);(\d+);(\d+)`)

// paintedGround is the colour the map paints behind every cell when watchpost
// declares none: the commonest background of a frame with no overlay.
func paintedGround(t *testing.T, size tuimaps.Size) tuimaps.RGB {
	t.Helper()
	m, err := embeddedMap(size)
	if err != nil {
		t.Fatal(err)
	}
	m.ColourDepth(tuimaps.Truecolor)
	frame, err := m.Render(size, time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	best := ""
	for _, match := range groundSGR.FindAllStringSubmatch(strings.Join(frame.Lines, "\n"), -1) {
		key := match[1] + ";" + match[2] + ";" + match[3]
		if seen[key]++; seen[key] > seen[best] {
			best = key
		}
	}
	var c tuimaps.RGB
	parts := strings.Split(best, ";")
	if len(parts) != 3 {
		t.Fatalf("the frame paints no ground: %q", best)
	}
	for i, p := range []*uint8{&c.R, &c.G, &c.B} {
		var v int
		for _, r := range parts[i] {
			v = v*10 + int(r-'0')
		}
		*p = uint8(v)
	}
	return c
}

// luminance is a colour's relative luminance (sRGB).
func luminance(c tuimaps.RGB) float64 {
	lin := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// rampMidpoint is a diverging ramp's turning class - the one lightest or
// darkest inside its ends - or -1 for a ramp that runs one way.
func rampMidpoint(ramp []tuimaps.RGB) int {
	if len(ramp) < 3 {
		return -1
	}
	lo, hi := 0, 0
	for i, c := range ramp {
		if luminance(c) < luminance(ramp[lo]) {
			lo = i
		}
		if luminance(c) > luminance(ramp[hi]) {
			hi = i
		}
	}
	for _, i := range []int{hi, lo} {
		if i > 0 && i < len(ramp)-1 {
			return i
		}
	}
	return -1
}

// TestEveryRampTheMapDrawsPassesTheVisionCheck is F-198: each ramp the map
// draws for watchpost, at each colour depth watchpost can set that draws
// colour (Truecolor and 256; at 16 and none the library draws shades and
// digits, its check's own reach), passes CheckRamp on the ground it is drawn
// on. The theme axis is one ground: watchpost declares none, so the map
// paints its own under every theme - and a declared ground fails this test
// until the matrix is given it.
func TestEveryRampTheMapDrawsPassesTheVisionCheck(t *testing.T) {
	size := tuimaps.Size{Cols: 80, Rows: 24}
	ground := paintedGround(t, size)
	for name, o := range visionRamps(t) {
		for _, depth := range []tuimaps.Depth{tuimaps.Truecolor, tuimaps.Colours256} {
			m, err := embeddedMap(size)
			if err != nil {
				t.Fatal(err)
			}
			m.ColourDepth(depth)
			if _, err := m.Set(o); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if _, err := m.Render(size, time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)); err != nil {
				t.Fatal(err)
			}
			legend := m.Legend()
			if len(legend) == 0 || len(legend[0].Classes) < 3 {
				t.Fatalf("%s at %v: the legend carries no ramp: %+v", name, depth, legend)
			}
			var ramp []tuimaps.RGB
			for _, c := range legend[0].Classes {
				if c.Drawn { // a class under the ramp's floor is left unpainted, the ground showing
					ramp = append(ramp, c.Colour)
				}
			}
			how := tuimaps.RampCheck{Ground: ground, Depth: depth, Midpoint: rampMidpoint(ramp), Lines: strings.HasPrefix(name, "wind")}
			if found := tuimaps.CheckRamp(ramp, how); len(found) > 0 {
				t.Errorf("%s at %v on %+v breaks the vision check: %+v", name, depth, ground, found)
			}
		}
	}
	for _, glob := range []string{"*.go", filepath.Join("..", "..", "app", "*.go")} {
		files, _ := filepath.Glob(glob)
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			for _, call := range []string{".Ground(", ".SetPalette("} {
				if bytes.Contains(src, []byte(call)) {
					t.Errorf("%s calls %s: the matrix's ground or palette is no longer the library's alone; add it here", f, call)
				}
			}
		}
	}
}
