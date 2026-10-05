package app

// mapfeedparts_test.go — D-268, D-266, D-274: the feed's alerts asked and
// delivered apart from its other layers, each other layer under its own
// bound, a failed ask for the view's alerts said, and an unknown place's
// zones said as unknown.

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// hangGet never answers until its ask is cancelled, and keeps the deadline
// it was asked under.
type hangGet struct {
	asked    atomic.Int64
	deadline atomic.Value
}

func (g *hangGet) GetText(ctx context.Context, _ string, _ ...httpx.Option) ([]byte, error) {
	g.asked.Add(1)
	if d, ok := ctx.Deadline(); ok {
		g.deadline.Store(d)
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestTheAlertsAreNotHeldBehindAHungLayer(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	hang := &hangGet{}
	lp := &livePipelines{idx: idx, airnow: airquality.New(hang, ""),
		areaAlerts: func(context.Context, []string) ([]snapshot.Alert, error) { return nil, nil }}
	ask := tty.MapAsk{Snap: &snapshot.Snapshot{}, View: tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}, Air: true, Part: tty.FeedAlerts}
	done := make(chan struct{})
	go func() { _ = lp.mapInputsFetching(context.Background(), ask); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the alerts' ask waited on a hung layer")
	}
	if hang.asked.Load() != 0 {
		t.Error("the alerts' ask asked the other layers")
	}

	ask.Part = tty.FeedLayers
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	go func() { _ = lp.mapInputsFetching(ctx, ask) }()
	for i := 0; i < 200 && hang.asked.Load() == 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	d, ok := hang.deadline.Load().(time.Time)
	if !ok || time.Until(d) > layerInputLimit {
		t.Errorf("a layer was asked under %v (%v): want its own bound of %v at most", d, ok, layerInputLimit)
	}
}

func TestAFailedAskForTheViewsAlertsIsSaid(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{idx: idx, areaAlerts: func(context.Context, []string) ([]snapshot.Alert, error) {
		return nil, &httpx.StatusError{Status: 503}
	}}
	ask := tty.MapAsk{Snap: &snapshot.Snapshot{}, View: tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}, Part: tty.FeedAlerts}
	in := lp.mapInputsFetching(context.Background(), ask)
	feed := lp.mapFeedWith(context.Background(), in, func(snapshot.Location) []string { return nil })
	if !feed.AlertsFailed {
		t.Error("the view's alerts did not answer, and the feed does not say so")
	}
	said := strings.Join(lp.problems.held, "\n")
	if !strings.Contains(said, "Alerts in view") || !strings.Contains(said, "HTTP 503") || strings.Contains(said, "http") {
		t.Errorf("the diagnostics were told:\n%s\nwant the failure by its kind, no address", said)
	}
	if errKind(context.Canceled) != "cancelled" || errKind(errors.New("x")) != "no answer" {
		t.Error("errKind names a failure wrongly")
	}
}

func TestAnUnknownPlacesZonesAreSaidUnknown(t *testing.T) {
	a := snapshot.Alert{Event: "Flood Watch", AreaDesc: "A; B", AffectedZones: []string{"NMZ001", "NMZ002"}}
	area := geo.Area{Missing: []string{"NMZ002"}}
	if got := partialNote(a, area, "Roswell, NM", nil); !strings.Contains(got, "Whether Roswell, NM lies in it is unknown.") {
		t.Errorf("unknown zones read %q; want unknown, never that it does not lie in it (D-274)", got)
	}
	if got := partialNote(a, area, "Roswell, NM", map[string]bool{"NMZ001": true}); !strings.Contains(got, "does not lie in it") {
		t.Errorf("known zones outside read %q", got)
	}
}

// TestAViewsAreasAreWorkedOutOnce is REVIEW PF-7: the same view asked again
// is answered from the last, a moved view worked out anew.
func TestAViewsAreasAreWorkedOutOnce(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	var l lastViewAreas
	v := tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1}
	first := l.of(idx, v)
	if len(first) == 0 {
		t.Fatal("the view has no areas: this measures nothing")
	}
	if again := l.of(idx, v); &again[0] != &first[0] {
		t.Error("the same view was worked out again")
	}
	moved := tty.MapView{W: -98, S: 30, E: -96, N: 32}
	if got := l.of(idx, moved); strings.Join(got, ",") == strings.Join(first, ",") {
		t.Errorf("a moved view kept the last view's areas: %v", got)
	}
}
