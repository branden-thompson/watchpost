package tty

// map_viewbox_test.go — viewBox works out the view's box from the library's
// published scale; this holds it to the library's own idea of the view.

import (
	"fmt"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// TestTheViewBoxIsTheLibrarysView places small alerts just inside viewBox's
// corners and edges and just outside each edge, at two zooms, and asks the
// library's Report which are in view: the inside ones all, the outside ones
// none. A change in the library's geometry - its tile size, its cell, its
// projection - fails here instead of "Alerts in view" quietly asking for the
// wrong box.
func TestTheViewBoxIsTheLibrarysView(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	m := d.mapPane.m
	if m == nil {
		t.Fatal("no map: this measures nothing")
	}
	size := d.mapBodySize()
	for _, zoom := range []float64{6, 9} {
		if err := m.Zoom(zoom); err != nil {
			t.Fatal(err)
		}
		v := d.viewBox(size)
		dx, dy := (v.E-v.W)*0.02, (v.N-v.S)*0.02
		square := func(id string, lon, lat float64) tuimaps.Overlay {
			ring := []tuimaps.LonLat{{Lon: lon - dx/4, Lat: lat - dy/4}, {Lon: lon + dx/4, Lat: lat - dy/4},
				{Lon: lon + dx/4, Lat: lat + dy/4}, {Lon: lon - dx/4, Lat: lat + dy/4}, {Lon: lon - dx/4, Lat: lat - dy/4}}
			return tuimaps.Overlay{ID: "alert/" + id, Valid: time.Now(), Keeps: time.Hour, Features: []tuimaps.Feature{{
				Kind: tuimaps.Polygon, Rings: [][]tuimaps.LonLat{ring}, Role: tuimaps.AlertSevere, Label: id, ID: id}}}
		}
		midLon, midLat := (v.W+v.E)/2, (v.N+v.S)/2
		inside := map[string]bool{}
		for i, c := range [][2]float64{ // the corners and each edge's middle, just inside; then the same just outside, one axis at a time
			{v.W + dx, v.N - dy}, {v.E - dx, v.N - dy}, {v.W + dx, v.S + dy}, {v.E - dx, v.S + dy},
			{midLon, v.N - dy}, {midLon, v.S + dy}, {v.W + dx, midLat}, {v.E - dx, midLat},
			{midLon, v.N + dy}, {midLon, v.S - dy}, {v.W - dx, midLat}, {v.E + dx, midLat}} {
			id := fmt.Sprintf("z%v-%d", zoom, i)
			inside[id] = i < 8
			if _, err := m.Set(square(id, c[0], c[1])); err != nil {
				t.Fatal(err)
			}
		}
		r, err := m.Report(nil)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, a := range r.Alerts {
			seen[a.Feature] = true
		}
		for id, in := range inside {
			if seen[id] != in {
				t.Errorf("zoom %v: %s in view %v by the library; viewBox %+v says %v", zoom, id, seen[id], v, in)
			}
		}
	}
}
