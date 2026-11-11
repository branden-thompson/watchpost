package tty

import (
	"testing"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// TestTheTitleAndTheDescriptionNameOneArea is 0.19.5 Q5.6 (DR-10): the map's
// title and its spoken description ask the area namer about the same view, so
// they hand it the same width and name the same area.
func TestTheTitleAndTheDescriptionNameOneArea(t *testing.T) {
	var widths []float64
	namer := func(_ tuimaps.LonLat, widthKm float64) string {
		widths = append(widths, widthKm)
		return "Southern California"
	}
	d := openMap(t, Config{MapAreaName: namer}, 133, 44)
	widths = nil
	_ = d.mapTitleAt(d.mapBodySize())
	_ = d.viewName()
	if len(widths) != 2 {
		t.Fatalf("the namer was asked %d time(s); want once by the title and once by the description", len(widths))
	}
	if widths[0] <= 0 || widths[0] != widths[1] {
		t.Errorf("the title asks about a view %.1f km wide and the description one %.1f km wide", widths[0], widths[1])
	}
}
