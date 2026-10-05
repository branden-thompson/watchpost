package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// CLEAR MAP DATA FORGETS THE TIDES AND THE UV CITIES TOO (D-269): the CO-OPS
// client's cached tide predictions and the history's EPA UV readings by city
// go, each counted with the rest; the history's other records stay.
func TestClearingForgetsTheTidesAndTheUVCities(t *testing.T) {
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked.Add(1)
		_, _ = w.Write([]byte(`{"predictions":[{"t":"2026-10-05 16:01","v":"1.193","type":"H"}]}`))
	}))
	defer srv.Close()
	c, err := httpx.New(httpx.Config{UserAgent: UserAgent, CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = c.ForgetPrefix("settle:") }) // its disk writes settled before the directory goes
	tides := coops.New(c, srv.URL)
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	tides.Now = func() time.Time { return now }
	if _, err := tides.NextTide(context.Background(), "9410170", now); err != nil {
		t.Fatal(err)
	}
	store := history.Open(t.TempDir(), func() time.Time { return now }, epaUVCities, ndfdHourly)
	var known knownUVCities
	city := geodata.City{Name: "San Diego", ASCII: "San Diego", State: "CA", Lat: 32.7, Lon: -117.2, TZ: "America/Los_Angeles"}
	known.record(store, city, []uv.Reading{{At: now, Index: 7}, {At: now.Add(time.Hour), Index: 8}})
	kept := history.Key{Source: "ndfd", Place: "us-a"}
	store.Put(ndfdHourly.Name, history.Record{Key: kept, At: now, IssuedAt: now, Shape: pointShape(32.7, -117.2), Values: map[string][]float64{"temp": {20}}})

	lp := &livePipelines{tides: tides, history: store}
	got := lp.clearMapData()
	if got.Err != nil || got.Files != 3 { // a tide prediction and two UV hours
		t.Errorf("cleared %+v; want the tide prediction and the two UV hours counted", got)
	}
	if recs := store.Range(epaUVCities.Name, uvCityKey(city), now, now.Add(2*time.Hour), 10); len(recs) != 0 {
		t.Errorf("%d UV hours survived", len(recs))
	}
	if _, ok := store.Get(ndfdHourly.Name, kept, now); !ok {
		t.Error("a record of another dataset was forgotten too")
	}
	if _, err := tides.NextTide(context.Background(), "9410170", now); err != nil {
		t.Fatal(err)
	}
	if n := asked.Load(); n != 2 {
		t.Errorf("the tide prediction was asked %d times; want again once forgotten", n)
	}
}
