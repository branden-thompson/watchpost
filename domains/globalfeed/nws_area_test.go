package globalfeed

// nws_area_test.go — 0.18.0 W5.3 (FR-4.3): the national feed keeps each
// alert's own polygon, as the station's own alerts do, so the map can draw a
// national severe event in view.

import (
	"context"
	"testing"
)

// TestTheNationalFeedKeepsTheAlertsPolygon: a warning with a polygon keeps
// it, and its point still comes from the first vertex; a zone-only watch
// keeps none and names its zones; a geometry that cannot be read is no shape,
// and the alert still stands.
func TestTheNationalFeedKeepsTheAlertsPolygon(t *testing.T) {
	body := `{"features":[
	  {"id":"urn:oid:tor1","geometry":{"type":"Polygon","coordinates":[[[-97.5,35.4],[-97.3,35.4],[-97.3,35.6],[-97.5,35.4]]]},
	   "properties":{"event":"Tornado Warning","areaDesc":"Oklahoma County, OK","affectedZones":["https://api.weather.gov/zones/county/OKC109"]}},
	  {"id":"urn:oid:tw1","geometry":null,
	   "properties":{"event":"Tornado Watch","areaDesc":"Chaves, NM","affectedZones":["https://api.weather.gov/zones/forecast/NMZ238"]}},
	  {"id":"urn:oid:bb1","geometry":{"bbox":[-90,30,-89,31],"type":"Polygon","coordinates":[[[-89.5,30.5],[-89.4,30.5],[-89.4,30.6],[-89.5,30.5]]]},
	   "properties":{"event":"Flash Flood Warning","areaDesc":"Harrison, MS"}},
	  {"id":"urn:oid:bad1","geometry":{"type":"Polygon","coordinates":"nonsense"},
	   "properties":{"event":"Severe Thunderstorm Warning","areaDesc":"Dallas County, TX"}}
	]}`
	evs, err := NewNWS(client(t), serve(t, body)).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 4 {
		t.Fatalf("%d events, want 4", len(evs))
	}
	tor, watch, boxed, bad := evs[0], evs[1], evs[2], evs[3]
	if !boxed.HasPoint || boxed.Lat != 30.5 || boxed.Lon != -89.5 {
		t.Errorf("a geometry with a bounding box first is placed at %v %v, want its first vertex, not the box", boxed.Lat, boxed.Lon)
	}
	if tor.Severe == nil || tor.Severe.Area.Empty() || tor.Severe.Area.Vertices() != 4 {
		t.Errorf("the warning's polygon was not kept: %+v", tor.Severe)
	}
	if !tor.HasPoint || tor.Lat != 35.4 || tor.Lon != -97.5 {
		t.Errorf("the warning's point is %v %v %v, want its first vertex", tor.HasPoint, tor.Lat, tor.Lon)
	}
	if watch.Severe == nil || !watch.Severe.Area.Empty() || watch.HasPoint || len(watch.Severe.AffectedZones) != 1 {
		t.Errorf("the zone-only watch reads %+v", watch.Severe)
	}
	if bad.Severe == nil || !bad.Severe.Area.Empty() {
		t.Errorf("an unreadable geometry became a shape: %+v", bad.Severe)
	}
}
