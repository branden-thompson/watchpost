package nws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestTwoUnidentifiedAlertsDoNotBecomeOne.
//
// A CAP alert should always carry an id. One that does not is still shown - it
// is never dropped silently (RS-10) - but the stand-in was the headline, and
// **two warnings of the same kind carry the same headline all the time**:
// "Tornado Warning issued" is what the service writes for every one of them.
// Two hazards then share an identity, and anything keyed by it - the area
// resolved for drawing, the read-once mark, the dedupe - keeps one and
// silently discards the other.
func TestTwoUnidentifiedAlertsDoNotBecomeOne(t *testing.T) {
	body := []byte(`{"features":[
	  {"properties":{"event":"Tornado Warning","headline":"Tornado Warning issued","areaDesc":"Johnson, KS",
	    "sent":"2026-09-21T10:00:00+00:00","senderName":"NWS Kansas City","affectedZones":["https://api.weather.gov/zones/forecast/KSZ103"]}},
	  {"properties":{"event":"Tornado Warning","headline":"Tornado Warning issued","areaDesc":"Custer, OK",
	    "sent":"2026-09-21T10:04:00+00:00","senderName":"NWS Norman","affectedZones":["https://api.weather.gov/zones/forecast/OKZ026"]}}
	]}`)
	alerts, err := decodeAlerts(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Fatalf("%d alerts decoded", len(alerts))
	}
	if alerts[0].ID == alerts[1].ID {
		t.Errorf("two tornado warnings four minutes and four hundred kilometres apart share the id %q", alerts[0].ID)
	}
	for _, a := range alerts {
		if a.ID == "" {
			t.Error("an alert with no id of its own was given no stand-in either")
		}
	}
}

// TestOneUnreachableLocationDoesNotLoseTheRest. Alerts are fetched for every
// watched place in one pass, and a place whose own lookup failed aborted the
// whole pass - so **one unreachable location left every other location with no
// alerts**, including places that answered perfectly. A station watching five
// cities went silent because of the one it could not resolve.
func TestOneUnreachableLocationDoesNotLoseTheRest(t *testing.T) {
	good := snapshot.LocationRef{Label: "Good", Lat: 41.08, Lon: -85.14, TZ: "America/New_York"}
	bad := snapshot.LocationRef{Label: "Bad", Lat: 1.0, Lon: 1.0, TZ: "UTC"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/points/1"): // the unreachable one
			w.WriteHeader(http.StatusInternalServerError)
		case strings.HasPrefix(r.URL.Path, "/points/"):
			w.Header().Set("Content-Type", "application/geo+json")
			_, _ = w.Write([]byte(`{"properties":{"gridId":"IWX","gridX":1,"gridY":2,` +
				`"forecast":"` + baseOf(r) + `/f","forecastHourly":"` + baseOf(r) + `/fh",` +
				`"forecastGridData":"` + baseOf(r) + `/g","observationStations":"` + baseOf(r) + `/s",` +
				`"forecastZone":"` + baseOf(r) + `/zones/forecast/INZ027",` +
				`"county":"` + baseOf(r) + `/zones/county/INC003"}}`))
		case r.URL.Path == "/s":
			w.Header().Set("Content-Type", "application/geo+json")
			_, _ = w.Write([]byte(`{"features":[{"properties":{"stationIdentifier":"KFWA","name":"Fort Wayne"}}]}`))
		case r.URL.Path == "/alerts/active":
			w.Header().Set("Content-Type", "application/geo+json")
			_, _ = w.Write([]byte(`{"features":[{"properties":{"id":"urn:oid:1","event":"Flood Warning",` +
				`"affectedZones":["` + baseOf(r) + `/zones/forecast/INZ027"]}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	p := newProvider(t, srv.URL)

	frag := snapshot.Fragment{PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	err := p.fetchAlerts(context.Background(), []snapshot.LocationRef{bad, good}, &frag)
	if err != nil {
		t.Fatalf("one unreachable location failed the whole pass: %v", err)
	}
	got := frag.PerLocation[snapshot.Key(good)].Alerts
	if len(got) != 1 {
		t.Errorf("the reachable location got %d alerts; its zone is under a Flood Warning", len(got))
	}
}

// baseOf is the server's own address, as the service writes absolute URLs.
func baseOf(r *http.Request) string { return "http://" + r.Host }

// TestAnAlertOverEightyZonesKeepsThemAll. The zone ids are what an alert's
// ground is resolved from, and they were bounded by the general list cap of
// fifty - a bound meant for provider prose. A Winter Storm Warning naming
// eighty zones kept fifty, so thirty pieces of its area went missing with
// nothing recording it, while the MATCHING ran over the full list. A listener
// in the sixtieth zone was correctly told the alert affected them and would
// have been shown an area that excluded them.
func TestAnAlertOverEightyZonesKeepsThemAll(t *testing.T) {
	var b []byte
	b = append(b, `{"features":[{"properties":{"id":"urn:oid:1.2.3","event":"Winter Storm Warning","affectedZones":[`...)
	for i := range 80 {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `"https://api.weather.gov/zones/forecast/MNZ`...)
		b = append(b, byte('0'+i/100%10), byte('0'+i/10%10), byte('0'+i%10))
		b = append(b, '"')
	}
	b = append(b, `]}}]}`...)

	alerts, err := decodeAlerts(b)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(alerts[0].AffectedZones); n != 80 {
		t.Errorf("an eighty-zone warning kept %d of its zones", n)
	}
}

// TestNoLocationResolvingIsStillAFailure is the other half of the rule above,
// and it needs saying on its own: tolerating ONE bad location must not become
// tolerating all of them. If nothing resolved there is nothing to report, and
// the caller has to hear about it - silence there would read as "no alerts
// anywhere", which is the one thing this program must never say by accident.
func TestNoLocationResolvingIsStillAFailure(t *testing.T) {
	// **The alerts endpoint answers perfectly**, and only the per-location
	// lookups fail. That is what makes this test able to tell the two apart: a
	// version with no guard would sail past the failures, ask for alerts over
	// an empty zone list, get a clean empty answer, and report success.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/alerts/active" {
			w.Header().Set("Content-Type", "application/geo+json")
			_, _ = w.Write([]byte(`{"features":[]}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	p := newProvider(t, srv.URL)

	frag := snapshot.Fragment{PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	err := p.fetchAlerts(context.Background(),
		[]snapshot.LocationRef{{Label: "A", Lat: 1, Lon: 1, TZ: "UTC"}, {Label: "B", Lat: 2, Lon: 2, TZ: "UTC"}}, &frag)
	if err == nil {
		t.Error("every location failed to resolve and the pass reported success")
	}
}
