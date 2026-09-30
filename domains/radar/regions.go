package radar

import (
	"github.com/branden-thompson/watchpost/platform/geo"
)

// Box is a radar box (W8.3a, D-47, D-54): a fixed piece of a region, fetched
// at a fixed size, so what a source is asked for never says where the
// listener is looking.
type Box struct {
	Name       string
	W, S, E, N float64
	Cols, Rows int
}

// maxPixels is the library's image cap at one byte a pixel (go-tuiMaps
// D-85, D-36: a host may lower it, never raise it); every box is within it.
// A terminal map shows a few tens of thousands of braille dots, so this is
// ample - found live, when a larger box was refused by the library.
const maxPixels = 250_000

// wholeBoxes are each region's one box, drawn when the view is wide. Alaska's
// stops at the antimeridian: a plain-degrees request cannot cross it, and the
// Aleutians west of 180° are left out.
var wholeBoxes = map[string]Box{
	geo.RegionContiguous: {Name: "us", W: -126, S: 23, E: -65, N: 51, Cols: 600, Rows: 275},
	// OUTSIDE THE LOWER 48, EACH BOX IS ITS MRMS PRODUCT'S WHOLE EXTENT, from
	// the services' capabilities (UAT-2 U2-12): a box cut to the old region
	// cut the radar off south of the Big Island, where the hurricane was.
	geo.RegionAlaska:    {Name: "ak", W: -176, S: 50, E: -126, N: 72, Cols: 600, Rows: 264},
	geo.RegionHawaii:    {Name: "hi", W: -164, S: 15, E: -151, N: 26, Cols: 480, Rows: 406},
	geo.RegionCaribbean: {Name: "pr", W: -90, S: 10, E: -60, N: 25, Cols: 600, Rows: 300},
	geo.RegionMarianas:  {Name: "gu", W: 140, S: 9, E: 150, N: 18, Cols: 400, Rows: 360},
}

// gridCols and gridRows split the lower 48 into its closer boxes.
const (
	gridCols = 4
	gridRows = 2
)

// wideDegrees is the view width past which the lower 48 is one box: past
// about three states across, the grid's boxes would all be asked for.
const wideDegrees = 16.0

// BoxesFor is the radar boxes a view takes in a region: the region's one box
// when the view is wide or the region small, otherwise the lower 48's grid
// boxes the view meets, one to four of them.
func BoxesFor(region string, view geo.Box) []Box {
	whole, ok := wholeBoxes[region]
	if !ok {
		return nil
	}
	if region != geo.RegionContiguous || view.E-view.W > wideDegrees {
		return []Box{whole}
	}
	var out []Box
	for _, b := range grid(whole) {
		if b.W <= view.E && b.E >= view.W && b.S <= view.N && b.N >= view.S {
			out = append(out, b)
		}
	}
	return out
}

// GridBoxes are a region's closer boxes, which a narrower view draws in
// place of the whole region's: the lower 48's grid, or none. The history
// records each (W18.3b), so any view can replay.
func GridBoxes(region string) []Box {
	whole, ok := wholeBoxes[region]
	if !ok || region != geo.RegionContiguous {
		return nil
	}
	return grid(whole)
}

// grid is the lower 48's closer boxes. SIZED TO THE LOOP, NOT THE CAP (UAT-2
// U2-5): a box's two-hour loop must fit the map's image budget, which
// charges a frame its PNG and a byte a pixel - boxes near the cap were
// refused whole. At 400x373 a county view still has twice a terminal's dots.
func grid(whole Box) []Box {
	w, h := (whole.E-whole.W)/gridCols, (whole.N-whole.S)/gridRows
	var out []Box
	for r := range gridRows {
		for c := range gridCols {
			out = append(out, Box{Name: "us-" + string(rune('a'+r*gridCols+c)),
				W: whole.W + float64(c)*w, E: whole.W + float64(c+1)*w, S: whole.S + float64(r)*h, N: whole.S + float64(r+1)*h,
				Cols: 400, Rows: 373})
		}
	}
	return out
}
