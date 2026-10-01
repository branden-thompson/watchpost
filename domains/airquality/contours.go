package airquality

// contours.go — AirNow's current AQI as contours (W19.4, D-193): the keyless
// national file of category polygons AirNow redraws each hour, which the map
// tints the AQI scale with in place of a metered model (D-185).
//
// THE POLYGONS TILE THE COUNTRY WITH HOLES. One national Good polygon has
// every other contour as a hole, and contours nest - an Unhealthy one inside
// an Unhealthy-for-Sensitive-Groups one, a hole in it. A point is a
// polygon's when its outer ring holds it and none of its holes does; the
// highest category holding it is its own.

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// The AQI's categories, lowest first: a contour's style names one.
const (
	Good = iota
	Moderate
	UnhealthySG
	Unhealthy
	VeryUnhealthy
	Hazardous
)

// categoryStyles are the file's style names, by category.
var categoryStyles = map[string]int{"Good": Good, "Moderate": Moderate, "UnhealthySG": UnhealthySG,
	"Unhealthy": Unhealthy, "VeryUnhealthy": VeryUnhealthy, "Hazardous": Hazardous}

// CategoryAQI is an AQI inside each category, to draw it in the scale: the
// middle of its range, Hazardous's floor and a hundred.
func CategoryAQI(category int) float64 {
	mids := [...]float64{25, 75, 125, 175, 250, 400}
	return mids[min(max(category, 0), len(mids)-1)]
}

// Point is a contour's vertex.
type Point struct{ Lon, Lat float64 }

// Contour is one category's polygon: its outer ring first, then its holes,
// and the box its outer ring spans.
type Contour struct {
	Category   int
	Rings      [][]Point
	W, S, E, N float64
}

// newContour is a contour with its box worked out.
func newContour(category int, rings [][]Point) Contour {
	c := Contour{Category: category, Rings: rings, W: 180, S: 90, E: -180, N: -90}
	if len(rings) == 0 {
		return c
	}
	for _, p := range rings[0] { // the outer ring's vertices (P10-02)
		c.W, c.E, c.S, c.N = min(c.W, p.Lon), max(c.E, p.Lon), min(c.S, p.Lat), max(c.N, p.Lat)
	}
	return c
}

// Contours is the file: the hour it is for, in UTC, and its polygons.
type Contours struct {
	Hour  time.Time
	Areas []Contour
}

// contoursFile is AirNow's combined AQI contours, refreshed about hourly.
const contoursFile = "/airnow/today/cur_aqi_combined.kml"

// Contours reads AirNow's current AQI contours.
func (p *Provider) Contours(ctx context.Context, now time.Time) (Contours, error) {
	body, err := p.get.GetText(ctx, p.base+contoursFile, httpx.TTL(fileAge))
	if err != nil {
		return Contours{}, fmt.Errorf("AirNow contours: %w", err)
	}
	return parseContours(body, now)
}

// kml is the parts of the file read.
type kml struct {
	Folder struct {
		Name       string `xml:"name"`
		Placemarks []struct {
			Style   string `xml:"styleUrl"`
			Polygon []struct {
				Outer string   `xml:"outerBoundaryIs>LinearRing>coordinates"`
				Inner []string `xml:"innerBoundaryIs>LinearRing>coordinates"`
			} `xml:"Polygon"`
		} `xml:"Placemark"`
	} `xml:"Document>Folder"`
}

// parseContours reads the file's polygons by category; a style that is no
// category - Unavailable, Invisible - is left out.
func parseContours(body []byte, now time.Time) (Contours, error) {
	var doc kml
	if err := xml.Unmarshal(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")), &doc); err != nil {
		return Contours{}, fmt.Errorf("AirNow contours: %w", err)
	}
	if len(doc.Folder.Placemarks) == 0 {
		return Contours{}, errors.New("AirNow contours: the file holds none")
	}
	out := Contours{Hour: contourHour(doc.Folder.Name, now)}
	for _, pm := range doc.Folder.Placemarks { // the file's contours (P10-02)
		category, ok := categoryStyles[strings.TrimPrefix(strings.TrimSpace(pm.Style), "#")]
		if !ok {
			continue
		}
		for _, poly := range pm.Polygon {
			rings := [][]Point{ring(poly.Outer)}
			for _, h := range poly.Inner {
				rings = append(rings, ring(h))
			}
			if len(rings[0]) >= 3 {
				out.Areas = append(out.Areas, newContour(category, rings))
			}
		}
	}
	return out, nil
}

// contourHour is the hour the file's folder names - "..._2026100103", UTC -
// or now's hour where it names none, never later than now.
func contourHour(name string, now time.Time) time.Time {
	hour := now.UTC().Truncate(time.Hour)
	i := strings.LastIndex(name, "_")
	if i < 0 {
		return hour
	}
	t, err := time.Parse("2006010215", name[i+1:])
	if err != nil || t.After(hour) {
		return hour
	}
	return t
}

// ring is a coordinates list's vertices: "lon,lat,alt" each.
func ring(coords string) []Point {
	var out []Point
	for _, f := range strings.Fields(coords) { // a ring's vertices (P10-02)
		parts := strings.Split(f, ",")
		if len(parts) < 2 {
			continue
		}
		lon, err1 := strconv.ParseFloat(parts[0], 64)
		lat, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil {
			out = append(out, Point{Lon: lon, Lat: lat})
		}
	}
	return out
}

// CategoryAt is the highest category whose contour holds a point; false
// where none does.
func (c Contours) CategoryAt(lat, lon float64) (int, bool) {
	best, found := 0, false
	for _, a := range c.Areas { // the file's contours (P10-02)
		if lon < a.W || lon > a.E || lat < a.S || lat > a.N || (found && a.Category <= best) {
			continue
		}
		if a.holds(lat, lon) {
			best, found = a.Category, true
		}
	}
	return best, found
}

// Raster is a box's cells' categories, row by row from the north: each the
// highest category holding its middle, -1 where none does. A scanline - each
// row's crossings of a contour's rings, filled between pairs, holes and all
// by the even-odd rule - not a point test a cell: the national Good contour
// has some 26,000 vertices.
func (c Contours) Raster(w, s, e, n float64, cols, rows int) []int {
	out := make([]int, max(cols*rows, 0))
	for i := range out {
		out[i] = -1
	}
	if cols <= 0 || rows <= 0 || e <= w || n <= s {
		return out
	}
	cw, rh := (e-w)/float64(cols), (n-s)/float64(rows)
	for _, a := range c.Areas { // the file's contours (P10-02)
		if a.E < w || a.W > e || a.N < s || a.S > n {
			continue
		}
		for r := range rows {
			lat := n - (float64(r)+0.5)*rh
			if lat < a.S || lat > a.N {
				continue
			}
			xs := a.crossings(lat)
			for k := 0; k+1 < len(xs); k += 2 { // pairs: inside between them (P10-02)
				from := max(int(math.Ceil((xs[k]-w)/cw-0.5)), 0)
				to := min(int(math.Floor((xs[k+1]-w)/cw-0.5)), cols-1)
				for col := from; col <= to; col++ {
					if cell := r*cols + col; a.Category > out[cell] {
						out[cell] = a.Category
					}
				}
			}
		}
	}
	return out
}

// crossings are where a contour's rings cross a latitude, west to east: the
// even-odd rule's edges, at inRing's half-open test so the two agree.
func (a Contour) crossings(lat float64) []float64 {
	var xs []float64
	for _, r := range a.Rings { // its rings (P10-02)
		for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
			p, q := r[i], r[j]
			if (p.Lat > lat) != (q.Lat > lat) {
				xs = append(xs, (q.Lon-p.Lon)*(lat-p.Lat)/(q.Lat-p.Lat)+p.Lon)
			}
		}
	}
	sort.Float64s(xs)
	return xs
}

// holds reports whether a contour holds a point: inside its outer ring and
// none of its holes.
func (a Contour) holds(lat, lon float64) bool {
	if len(a.Rings) == 0 || !inRing(a.Rings[0], lat, lon) {
		return false
	}
	for _, h := range a.Rings[1:] { // its holes (P10-02)
		if inRing(h, lat, lon) {
			return false
		}
	}
	return true
}

// inRing is the even-odd rule: a ray east from the point crosses the ring an
// odd number of times where the ring holds it.
func inRing(r []Point, lat, lon float64) bool {
	in := false
	for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 { // a ring's edges (P10-02)
		a, b := r[i], r[j]
		if (a.Lat > lat) != (b.Lat > lat) && lon < (b.Lon-a.Lon)*(lat-a.Lat)/(b.Lat-a.Lat)+a.Lon {
			in = !in
		}
	}
	return in
}
