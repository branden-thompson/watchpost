package geo

import (
	"os"
	"testing"
)

// TestTheReaderAgreesWithTheServiceItself reads a real forecast zone as the
// service sends it. The counts are what an independent tool made of the same
// document, so this says the reader agrees with something other than itself.
func TestTheReaderAgreesWithTheServiceItself(t *testing.T) {
	raw, err := os.ReadFile("testdata/zone-dallas.geojson")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadGeometry(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0]) != 1 {
		t.Errorf("Dallas has %d areas of %d rings; the service sends one of one", len(got), got.Rings())
	}
	if got.Vertices() != 80 {
		t.Errorf("Dallas has %d positions; counted independently there are 80", got.Vertices())
	}
	if !got[0][0].Closed() {
		t.Error("the zone's ring is not closed")
	}
}

// TestTheReaderHoldsTheWorstZoneMeasured is the tail, not the median. Glacier
// Bay is a marine zone of thirty-two separate areas and twelve thousand
// positions - a hundred and fifty times the size of a typical county - and it
// is the case any store built on this has to survive. Counted independently at
// 12,004. That those areas stay separate is TestASeparateIslandIsNotAHole;
// this is only that all of them arrive.
func TestTheReaderHoldsTheWorstZoneMeasured(t *testing.T) {
	raw, err := os.ReadFile("testdata/zone-glacier-bay.geojson")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadGeometry(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Rings() != 32 {
		t.Errorf("Glacier Bay has %d rings; counted independently there are 32", got.Rings())
	}
	if got.Vertices() != 12004 {
		t.Errorf("Glacier Bay has %d positions; counted independently there are 12,004", got.Vertices())
	}
	for i, p := range got {
		for j, r := range p {
			if !r.Closed() {
				t.Errorf("ring %d of area %d is not closed", j, i)
			}
		}
	}
	// It is well inside what one hazard may hold, which is why nothing here
	// simplifies it (MG-6).
	if got.Vertices() > MaxVertices {
		t.Errorf("the largest zone measured is over the cap of %d", MaxVertices)
	}
}
