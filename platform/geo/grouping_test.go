package geo

import (
	"os"
	"testing"
)

// TestRingsStayGroupedIntoThePolygonsTheyBelongTo is RT-8, and it is the
// finding the BUILD-exit red team got wrong.
//
// It was filed as "two overlapping zones would punch a phantom hole". The
// defect is larger and needs no overlap: the reader flattened away the level
// that says **which rings form one polygon**, and the thing that draws reads
// the first ring of a shape as the outline and **every ring after it as a
// hole**. A zone that is many separate islands then draws as one islet with
// holes punched in it, and anything outside that islet's band is not drawn at
// all.
//
// Once flattened the information cannot be recovered - an island and a hole
// are the same list of positions - so this is checked here, at the only place
// that still knows.
func TestRingsStayGroupedIntoThePolygonsTheyBelongTo(t *testing.T) {
	// Two areas. The first has a hole in it; the second is separate land.
	// Flattened this is three rings and nothing says so; grouped it is two
	// polygons, and the hole belongs to exactly one of them.
	const body = `{"type":"MultiPolygon","coordinates":[
		[[[-85,41],[-83,41],[-83,43],[-85,43],[-85,41]],
		 [[-84.5,41.5],[-83.5,41.5],[-83.5,42.5],[-84.5,42.5],[-84.5,41.5]]],
		[[[-80,35],[-79,35],[-79,36],[-80,36],[-80,35]]]]}`
	got, err := ReadGeometry([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("read as %d areas; the document describes two", len(got))
	}
	if len(got[0]) != 2 {
		t.Errorf("the first area has %d rings; it is an outline and one hole", len(got[0]))
	}
	if len(got[1]) != 1 {
		t.Errorf("the second area has %d rings; it is an outline and nothing else", len(got[1]))
	}
	if got.Vertices() != 15 {
		t.Errorf("%d positions in all; counted by hand there are 15", got.Vertices())
	}
}

// TestASeparateIslandIsNotAHole is the same rule stated as the consequence a
// viewer would see, on the worst real zone we hold. Glacier Bay is
// **thirty-two separate landmasses**, not one landmass with thirty-one holes
// in it. Measured against the live service: the first of those is a
// twenty-six-position islet spanning about one per cent of the zone's height,
// so reading the rest as its holes loses essentially the whole zone.
func TestASeparateIslandIsNotAHole(t *testing.T) {
	raw, err := os.ReadFile("testdata/zone-glacier-bay.geojson")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadGeometry(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 32 {
		t.Fatalf("Glacier Bay read as %d areas; the service sends thirty-two", len(got))
	}
	for i, p := range got {
		if len(p) != 1 {
			t.Errorf("area %d has %d rings; each of this zone's islands is an outline alone", i, len(p))
		}
	}
}

// TestEachPointOfAMultiPointStandsAlone is RT-7, which is the same root cause
// wearing a different hat: a MultiPoint read as one ring joined unrelated
// places into a single outline. Nothing sends one today, which is why it was
// deferred - and why it would have been believed if it ever arrived.
func TestEachPointOfAMultiPointStandsAlone(t *testing.T) {
	got, err := ReadGeometry([]byte(`{"type":"MultiPoint","coordinates":[[-85.1,41.1],[-80.2,35.2],[-70.3,43.3]]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("three places read as %d areas", len(got))
	}
	for i, p := range got {
		if len(p) != 1 || len(p[0]) != 1 {
			t.Errorf("place %d is %d rings of %d positions; it is one position", i, len(p), len(p[0]))
		}
	}
}

// TestAnAreaWithNoRingsIsRefused closes the door RT-1 came through, on the new
// level this grouping adds. Rings with no positions were refused; **an area
// with no rings is the same emptiness one level up**, and it would have been
// accepted by the same code that was fixed for the level below it.
func TestAnAreaWithNoRingsIsRefused(t *testing.T) {
	for _, body := range []string{
		`{"type":"MultiPolygon","coordinates":[[],[]]}`,
		`{"type":"MultiPolygon","coordinates":[[[[-85,41],[-84,41],[-84,42],[-85,41]]],[]]}`,
	} {
		if _, err := ReadGeometry([]byte(body)); err == nil {
			t.Errorf("%.56q was accepted", body)
		}
	}
}
