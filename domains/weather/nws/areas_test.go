package nws

// areas_test.go — 0.18.0 D-66: the map's "Alerts in view" asks the service
// for the active alerts of the areas the view touches - state and marine
// area codes, never the view's rectangle - in one request.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAlertsInAreasAsksOnceByAreaCode(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"features":[
		  {"properties":{"id":"urn:oid:a1","event":"Beach Hazards Statement","severity":"Moderate","affectedZones":["https://api.weather.gov/zones/forecast/CAZ043"]},"geometry":null},
		  {"properties":{"id":"urn:oid:a2","event":"Flood Warning","severity":"Severe","affectedZones":[]},
		   "geometry":{"type":"Polygon","coordinates":[[[-117.5,33.0],[-117.3,33.0],[-117.3,33.2],[-117.5,33.0]]]}}]}`))
	}))
	defer srv.Close()
	p := newProvider(t, srv.URL)
	got, err := p.AlertsInAreas(context.Background(), []string{"PZ", "CA", "CA"})
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0] != "status=actual&area=CA,PZ" {
		t.Errorf("the service was asked %q; want one request for the sorted, distinct areas", asked)
	}
	if len(got) != 2 || got[0].ID != "urn:oid:a1" || got[0].AffectedZones[0] != "CAZ043" || got[1].Area.Empty() {
		t.Errorf("the alerts read back %+v", got)
	}
	if none, err := p.AlertsInAreas(context.Background(), nil); err != nil || none != nil || len(asked) != 1 {
		t.Errorf("no areas asked the service (%d requests) or answered %v, %v", len(asked), none, err)
	}
}
