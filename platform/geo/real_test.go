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
	if len(got) != 1 {
		t.Errorf("Dallas has %d rings; the service sends one", len(got))
	}
	if got.Vertices() != 80 {
		t.Errorf("Dallas has %d positions; counted independently there are 80", got.Vertices())
	}
	if !got[0].Closed() {
		t.Error("the zone's ring is not closed")
	}
}

// TestTheReaderHoldsTheWorstZoneMeasured is the tail, not the median. Glacier
// Bay is a marine zone of thirty-two rings and twelve thousand positions - a
// hundred and fifty times the size of a typical county - and it is the case any
// store built on this has to survive. Counted independently at 12,004.
func TestTheReaderHoldsTheWorstZoneMeasured(t *testing.T) {
	raw, err := os.ReadFile("testdata/zone-glacier-bay.geojson")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadGeometry(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 32 {
		t.Errorf("Glacier Bay has %d rings; counted independently there are 32", len(got))
	}
	if got.Vertices() != 12004 {
		t.Errorf("Glacier Bay has %d positions; counted independently there are 12,004", got.Vertices())
	}
	for i, r := range got {
		if !r.Closed() {
			t.Errorf("ring %d of the zone is not closed", i)
		}
	}
	// It is well inside what one hazard may hold, which is why nothing here
	// simplifies it (MG-6).
	if got.Vertices() > MaxVertices {
		t.Errorf("the largest zone measured is over the cap of %d", MaxVertices)
	}
}
