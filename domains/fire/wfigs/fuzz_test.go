package wfigs

// fuzz_test.go — the incident layer and the perimeters are read from the
// network, so both decoders are fuzzed from the recorded answers: no panic;
// every incident kept has a name and a finite point, every perimeter kept an
// area and finite vertices.

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// finite reports whether every number is neither NaN nor infinite.
func finite(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

func FuzzDecodeLayer(f *testing.F) {
	f.Add([]byte(geojson))
	f.Add([]byte(layerBody))
	f.Fuzz(func(t *testing.T, raw []byte) {
		got, err := decodeLayer(raw)
		if err != nil {
			return
		}
		for _, in := range got {
			if in.name == "" {
				t.Fatal("an incident with no name kept")
			}
			if !finite(in.lat, in.lon) || !onEarth(in.lat, in.lon) {
				t.Fatalf("%s kept at %v, %v", in.name, in.lat, in.lon)
			}
		}
	})
}

func FuzzDecodePerimeters(f *testing.F) {
	b, err := os.ReadFile(filepath.Join("testdata", "perimeters.json"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(b)
	f.Fuzz(func(t *testing.T, raw []byte) {
		got, err := decodePerimeters(raw)
		if err != nil {
			return
		}
		for _, p := range got {
			if len(p.Areas) == 0 {
				t.Fatalf("%s kept with no area", p.Name)
			}
			for _, area := range p.Areas {
				for _, ring := range area {
					for _, v := range ring {
						if !finite(v[0], v[1]) || !onEarth(v[1], v[0]) {
							t.Fatalf("%s kept with a vertex at %v", p.Name, v)
						}
					}
				}
			}
		}
	})
}

// TestAPlaceOffTheEarthIsLeftOut: an incident or a perimeter whose
// coordinates name no place on Earth is no fire to draw (REVIEW IS-M2).
func TestAPlaceOffTheEarthIsLeftOut(t *testing.T) {
	layer := `{"features":[{"geometry":{"coordinates":[-120,95]},"properties":{"IncidentName":"Off"}},` +
		`{"geometry":{"coordinates":[-120,38]},"properties":{"IncidentName":"On"}}]}`
	got, err := decodeLayer([]byte(layer))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].name != "On" {
		t.Errorf("decoded %+v; want only the incident on Earth", got)
	}
	perims := `{"features":[{"geometry":{"type":"Polygon","coordinates":[[[200,38],[-119,38],[-119,39],[200,38]]]},"properties":{"poly_IncidentName":"Off"}},` +
		`{"geometry":{"type":"Polygon","coordinates":[[[-120,38],[-119,38],[-119,39],[-120,38]]]},"properties":{"poly_IncidentName":"On"}}]}`
	ps, err := decodePerimeters([]byte(perims))
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 || ps[0].Name != "On" {
		t.Errorf("decoded %d perimeters; want only the one on Earth", len(ps))
	}
}
