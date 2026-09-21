package nws

import (
	"testing"
)

// TestAnAlertKeepsItsOwnPolygon. The response carries a geometry beside the
// properties, and the struct this package decoded into declared only the
// properties - so `encoding/json` threw the shape away, silently, with nothing
// anywhere recording that it did. One alert in five carries one.
func TestAnAlertKeepsItsOwnPolygon(t *testing.T) {
	body := []byte(`{"features":[
      {"properties":{"id":"a1","event":"Flood Warning","severity":"Severe","affectedZones":["https://api.weather.gov/zones/forecast/INZ018"]},
       "geometry":{"type":"Polygon","coordinates":[[[-85.2,41.0],[-85.0,41.0],[-85.0,41.2],[-85.2,41.0]]]}},
      {"properties":{"id":"a2","event":"Heat Advisory","severity":"Minor","affectedZones":["https://api.weather.gov/zones/forecast/INZ019"]}}
    ]}`)
	alerts, err := decodeAlerts(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Fatalf("%d alerts decoded", len(alerts))
	}
	if got := alerts[0].Area.Vertices(); got != 4 {
		t.Errorf("the alert with a polygon kept %d positions; the response has 4", got)
	}
	if !alerts[0].Area[0][0].Closed() {
		t.Error("the kept ring is not closed")
	}
	// **The ordinary case**: four alerts in five name zones and carry no shape
	// of their own, and that is not a failure.
	if !alerts[1].Area.Empty() {
		t.Errorf("a zone-only alert invented %d positions", alerts[1].Area.Vertices())
	}
}
