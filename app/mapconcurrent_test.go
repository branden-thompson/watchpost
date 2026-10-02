package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// slowGet is a Getter whose every answer takes a while.
type slowGet struct{ d time.Duration }

func (g slowGet) GetText(context.Context, string, ...httpx.Option) ([]byte, error) {
	time.Sleep(g.d)
	return nil, nil
}

// TestTheFeedsInputsAreAskedTogether is W14's P-5: the view's alerts, the
// fire, the quakes, the buoys, the tide stations and AirNow do not depend on
// one another, so they are asked together and the feed waits for the slowest,
// not the sum.
func TestTheFeedsInputsAreAskedTogether(t *testing.T) {
	const each = 300 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(each)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	lp := &livePipelines{idx: idx,
		areaAlerts: func(context.Context, []string) ([]snapshot.Alert, error) { time.Sleep(each); return nil, nil },
		fire:       []snapshot.Provider{wfigs.New(c, srv.URL+"/query", fire.DefaultRules())}, rules: fire.DefaultRules(),
		mapQuakes: newMapQuakes(c, srv.URL+"/"),
		marine:    []snapshot.Provider{ndbc.New(c, srv.URL), coops.New(c, srv.URL)},
		airnow:    airquality.New(slowGet{each}, ""),
	}
	ask := tty.MapAsk{Snap: &snapshot.Snapshot{}, View: tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1},
		Fire: true, Quakes: true, Buoys: true, Tides: true, Air: true}
	start := time.Now()
	_ = lp.mapInputsFetching(context.Background(), ask)
	if took := time.Since(start); took > 4*each {
		t.Errorf("six inputs each %v took %v: asked one after another (the sum is at least %v)", each, took, 6*each)
	}
}
