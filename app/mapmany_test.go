package app

// mapmany_test.go — 0.18.0 D-129 (UAT-2 U2-34): with buoys and tides on, one
// overlay a station - hundreds at a national view - passed the library's
// queue and starved the basemap. Every layer of many things is one overlay.

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestEveryLayerOfManyThingsIsOneOverlay is D-129's wiring: the feed, handed
// three hundred of each - quakes, buoys, tide stations, fire perimeters and
// incidents - gives the map one overlay a kind, and every thing is in it.
func TestEveryLayerOfManyThingsIsOneOverlay(t *testing.T) {
	const many = 300
	now := time.Now()
	view := tty.MapView{W: -125, S: 24, E: -66, N: 50}
	in := mapInputs{snap: &snapshot.Snapshot{}, region: geo.RegionContiguous, view: view, clock: render.ClockByKey("12h")}
	for i := range many {
		lon, lat := -120+float64(i)/10, 30+float64(i%100)/10
		id := strconv.Itoa(i)
		mag := 2.5
		in.quakes = append(in.quakes, globalfeed.Event{ID: id, Class: globalfeed.ClassQuake, Lat: lat, Lon: lon, HasPoint: true, At: now.Add(-time.Hour), Quake: &globalfeed.QuakeDetail{Mag: &mag}})
		wave := 1.0
		in.buoys = append(in.buoys, ndbc.Obs{ID: id, Lat: lat, Lon: lon, At: now.Add(-time.Minute), WaveM: &wave})
		in.tides = append(in.tides, tideMark{station: coops.Station{ID: id, Lat: lat, Lon: lon}})
		box := [][2]float64{{lon, lat}, {lon + 0.05, lat}, {lon + 0.05, lat + 0.05}, {lon, lat}}
		in.fire.perimeters = append(in.fire.perimeters, wfigs.Perimeter{Name: "p" + id, Areas: [][][][2]float64{{box}}})
		in.fire.incidents = append(in.fire.incidents, snapshot.Incident{Name: "i" + id, Lat: lat, Lon: lon})
	}
	feed := (&livePipelines{}).mapFeedWith(context.Background(), in, func(snapshot.Location) []string { return nil })
	overlays, things := map[string]int{}, map[string]int{}
	for _, o := range feed.Overlays {
		kind := o.ID[:strings.LastIndex(o.ID, "/")+1]
		if strings.HasPrefix(o.ID, tty.FireLayer+"/") {
			kind = o.ID // the perimeters and the incidents, each its own
		}
		overlays[kind]++
		things[kind] += len(o.Features)
	}
	if len(overlays) != 5 {
		t.Fatalf("the feed hands the map %d overlays of %d kinds; want five, one a kind: the quakes, buoys, tides, perimeters and incidents", len(feed.Overlays), len(overlays))
	}
	for kind, n := range overlays {
		if n != 1 || things[kind] != many {
			t.Errorf("%s is %d overlays holding %d things; want one holding all %d", kind, n, things[kind], many)
		}
	}
}
