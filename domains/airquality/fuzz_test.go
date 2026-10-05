package airquality

// fuzz_test.go — AirNow's files are read from the network, so both readers
// are fuzzed from the recorded files: no panic, every place on Earth, every
// AQI missing or a finite number not below zero, every category one of the
// six, the contours' hour never after now's.

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// recorded is a recorded file, whole, and its first lines alone: a seed the
// fuzzer can mutate quickly.
func recorded(f *testing.F, name string, lines int) (whole, head []byte) {
	f.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		f.Fatal(err)
	}
	cut := b
	for i := 0; i < lines; i++ {
		j := bytes.IndexByte(cut, '\n')
		if j < 0 {
			return b, b
		}
		cut = cut[j+1:]
	}
	return b, b[:len(b)-len(cut)]
}

// checkReading fails unless an AQI is missing or a finite number, not below
// zero.
func checkReading(t *testing.T, r Reading) {
	t.Helper()
	if !math.IsNaN(r.AQI) && (math.IsInf(r.AQI, 0) || r.AQI < 0) {
		t.Fatalf("an AQI of %v kept", r.AQI)
	}
	if v := r.Value(); math.IsInf(v, 0) || v < 0 {
		t.Fatalf("a reading's value is %v", v)
	}
}

func FuzzParse(f *testing.F) {
	whole, head := recorded(f, "reportingarea.dat", 40)
	f.Add(string(whole))
	f.Add(string(head))
	f.Fuzz(func(t *testing.T, body string) {
		for _, a := range parse(body, captured) {
			if !onEarth(a.Lat, a.Lon) {
				t.Fatalf("%s, %s kept at %v, %v: not on Earth", a.Name, a.State, a.Lat, a.Lon)
			}
			if a.Now != nil {
				checkReading(t, *a.Now)
			}
			for day, r := range a.Forecast {
				if day < 0 {
					t.Fatalf("%s's forecast for %d days ago kept", a.Name, -day)
				}
				checkReading(t, r)
			}
		}
	})
}

func FuzzParseContours(f *testing.F) {
	whole, head := recorded(f, "cur_aqi_combined.kml", 60)
	f.Add(whole)
	f.Add(append(head, []byte("</coordinates></LinearRing></outerBoundaryIs></Polygon></Placemark></Folder></Document></kml>")...))
	f.Fuzz(func(t *testing.T, body []byte) {
		c, err := parseContours(body, captured)
		if err != nil {
			return
		}
		if c.Hour.After(captured) {
			t.Fatalf("the contours' hour %v is after now, %v", c.Hour, captured)
		}
		for _, a := range c.Areas {
			if a.Category < Good || a.Category > Hazardous {
				t.Fatalf("a contour of category %d", a.Category)
			}
			if len(a.Rings) == 0 || len(a.Rings[0]) < 3 {
				t.Fatalf("a contour whose outline has %d rings", len(a.Rings))
			}
			for _, r := range a.Rings {
				for _, p := range r {
					if !onEarth(p.Lat, p.Lon) {
						t.Fatalf("a vertex at %v, %v: not on Earth", p.Lat, p.Lon)
					}
				}
			}
		}
		_, _ = c.CategoryAt(34, -118)
		cells := c.Raster(-125, 24, -66, 50, 8, 4)
		if len(cells) != 8*4 {
			t.Fatalf("a raster of %d cells for 8 by 4", len(cells))
		}
		for _, cell := range cells {
			if cell < -1 || cell > Hazardous {
				t.Fatalf("a raster cell of category %d", cell)
			}
		}
	})
}
