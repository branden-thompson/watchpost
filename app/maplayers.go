package app

// maplayers.go — the map's registry of layers and sources (0.18.0 W1.13,
// FR-9.3), and the cost estimate read from it (W1.14, FR-9.2).
//
// MEMBERS REGISTER THEMSELVES AND ARE DISCOVERED: each layer from its own
// file, the closed list's one basemap where the map is built (maps.go). The
// Settings row, the window's switching and the estimate all walk the
// registry, so a new layer or source plugs in without editing the others
// ("Discover Consumers, Don't Enumerate Them"); a second basemap is a ruling
// of its own (D-31). A layer's key is the part of
// its overlays' ids before the slash, which is how the window switches them.
// Options are the Settings table's rows (D-62), which already take a new row
// without editing the others.

import (
	"context"
	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// mapInputs is what the feed and each layer's cost read: the station's data,
// the selected place, and the alerts of the view (D-66, D-76).
type mapInputs struct {
	snap   *snapshot.Snapshot
	place  *snapshot.Location
	inView []snapshot.Alert   // the view's alerts, which the map draws whoever holds them
	quakes []globalfeed.Event // the ticker's earthquakes in view (D-80), fetched by the ticker, never here
	region string             // the region the view is bound to, and the view: the radar's estimate reads both (W8.15)
	view   tty.MapView
	ahead  int // the radar's hours ahead (D-114), which its estimate counts
	// forecast is Forecast mode (D-94): its rain and snow cost a request a
	// box, which Radar mode never makes (D-117).
	forecast bool
	fire     fireInView // the fire in view (D-121), fetched for the feed alone
	// quakeFeed is the quakes chosen (D-122), which their estimate counts;
	// clock the listener's, for their times (D-123).
	quakeFeed string
	clock     render.Clock
	// buoys and tides are the sea's stations in view (D-127, D-128), asked
	// only while their rows are on.
	buoys []ndbc.Obs
	tides []tideMark
	// airnow is AirNow's reporting areas (D-138), asked while Air quality is
	// on; anchor the hour Forecast mode's steps count from.
	airnow []airquality.Area
	anchor time.Time
	// fireMode is the Fire row's choice (D-145).
	fireMode string
	// alertsOff and alertCategoriesOff are what the listener switched off:
	// their alerts gone from the map, their zones never asked (D-160).
	alertsOff          bool
	alertCategoriesOff []string
	// part is which part of the feed is wanted (D-268); inViewFailed says
	// the view's alerts could not be asked (D-266).
	part         tty.FeedPart
	inViewFailed bool
	// switchedOn is the window's switches, for an estimate that counts only
	// what is on - an alert category among them (D-149); nil counts all.
	switchedOn func(key string) bool
	imperial   bool
}

// mapInputs are the inputs for the estimate, on the UI goroutine: the view's
// alerts as remembered, never fetched.
func (lp *livePipelines) mapInputs(ask tty.MapAsk) mapInputs {
	return lp.inputsFor(context.Background(), ask, false)
}

// mapInputsFetching is mapInputs for the feed, off the UI goroutine: it may
// ask the service for the view's areas (D-66).
func (lp *livePipelines) mapInputsFetching(ctx context.Context, ask tty.MapAsk) mapInputs {
	return lp.inputsFor(ctx, ask, true)
}

// inputsFor is the inputs for an ask, fetching the view's areas only when
// fetch is on. THE MAP DRAWS WHAT IS REAL IN VIEW (D-76): there is no scope
// to choose.
func (lp *livePipelines) inputsFor(ctx context.Context, ask tty.MapAsk, fetch bool) mapInputs {
	in := mapInputs{snap: ask.Snap, place: ask.Place, region: ask.Region, view: ask.View, ahead: ask.RadarAhead, forecast: ask.Forecast,
		quakeFeed: ask.QuakeFeed, clock: ask.Clock, imperial: ask.Fahrenheit, anchor: ask.Anchor, fireMode: ask.FireMode,
		alertsOff: ask.AlertsOff, alertCategoriesOff: ask.AlertCategoriesOff, part: ask.Part}
	if lp == nil {
		return in
	}
	if !fetch {
		held, _ := lp.viewAlerts(ctx, ask.View, false)
		in.inView = inViewOnly(held, ask.View)
		in.tides = lp.tidesHeld(ask) // the estimate counts the stations held in view, asking nothing (W14, C-2)
		return in
	}
	lp.fetchInputs(ctx, ask, &in)
	return in
}

// layerInputLimit bounds each input of the feed's other layers (D-268): ten
// seconds, a third of the map's work limit, so the alerts' own ask is never
// waiting behind one.
const layerInputLimit = 10 * time.Second

// fetchInputs asks for the feed's inputs TOGETHER (W14, P-5): the view's
// alerts, the fire, the quakes, the buoys, the tide stations and AirNow do not
// depend on one another, and one after another the feed waited out their sum.
// Each goroutine writes its own field alone, and all are joined before the
// feed reads any; each request still goes through its lane's pacing, so no
// host is asked faster. Six at most, one an input - never one a thing.
func (lp *livePipelines) fetchInputs(ctx context.Context, ask tty.MapAsk, in *mapInputs) {
	var wg sync.WaitGroup
	run := func(name string, f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer lp.timings.stage("input:"+name, time.Now()) // each input's own time (W14)
			f()
		}()
	}
	if ask.Part != tty.FeedLayers {
		run("alerts", func() {
			alerts, err := lp.viewAlerts(ctx, ask.View, true)
			in.inView, in.inViewFailed = inViewOnly(alerts, ask.View), err != nil
		})
	}
	if ask.Part != tty.FeedAlerts {
		// EACH OTHER LAYER HAS ITS OWN BOUND (D-268): a hung host delays its
		// layer alone, never the alerts, which are asked apart.
		lctx, cancel := context.WithTimeout(ctx, layerInputLimit)
		defer cancel()
		if ask.Fire { // fetched only while on (W14, P-3; D-149)
			run("fire", func() { in.fire = lp.fireIn(lctx, ask) })
		}
		if ask.Quakes {
			run("quakes", func() { in.quakes = quakesIn(lp.mapQuakes.fetch(lctx, ask.QuakeFeed), ask.View) }) // D-122: the feed chosen, not the ticker's
		}
		run("buoys", func() { in.buoys = lp.buoysIn(lctx, ask) })             // D-127: while its row is on
		run("tides", func() { in.tides = lp.tidesIn(lctx, ask, time.Now()) }) // D-128: while its row is on
		run("airnow", func() { in.airnow = lp.airnowIn(lctx, ask) })          // D-138: while Air quality is on
	}
	wg.Wait()
}

// withInView is the snapshot the feed and the estimate walk: the station's,
// and the view's alerts as one more location's, on a copy - the station's
// snapshot is shared and never written.
func (in mapInputs) withInView() *snapshot.Snapshot {
	if len(in.inView) == 0 {
		return in.snap
	}
	out := snapshot.Snapshot{}
	if in.snap != nil {
		out = *in.snap
	}
	out.Locations = append(append([]snapshot.Location(nil), out.Locations...), snapshot.Location{Label: "alerts in view", Alerts: in.inView})
	return &out
}

// mapCost is the window's estimate: the registry's, over the ask's inputs.
func (lp *livePipelines) mapCost(ask tty.MapAsk, on func(key string) bool) tty.MapCost {
	return refreshCost(lp.mapInputs(ask), on)
}

// mapLayer is one layer the listener can switch (R-9.2): its key, the words
// Settings shows, the builders' default, and what one refresh of it fetches.
type mapLayer struct {
	key, label string
	on         bool
	cost       func(in mapInputs) (bytes int64, requests int) // nil: nothing fetched of its own
	chips      []string                                       // the sources its badge names, where they never change (D-133)
}

// mapSource is one entry of FR-3.8's closed list: a name and the exact
// address the library is pointed at. No path takes an address from anywhere
// else.
type mapSource struct {
	name, address string
}

var (
	// mapLayers is every registered layer, in registration order.
	mapLayers []mapLayer
	// mapSources is every registered source; the first is the basemap.
	mapSources []mapSource
)

// registerMapLayer adds a layer. A key registered twice is a programming
// error: the window switches overlays by it.
func registerMapLayer(l mapLayer) {
	for _, have := range mapLayers {
		if have.key == l.key {
			panic("map layer " + l.key + " registered twice")
		}
	}
	mapLayers = append(mapLayers, l)
}

// registerMapSource adds a source to the closed list.
func registerMapSource(s mapSource) { mapSources = append(mapSources, s) }

// windowLayers are the registered layers as the window's Settings shows them.
func windowLayers() []tty.MapLayer {
	out := make([]tty.MapLayer, 0, len(mapLayers))
	for _, l := range mapLayers {
		out = append(out, tty.MapLayer{Key: l.key, Label: l.label, On: l.on, Chips: l.chips})
	}
	return out
}

// refreshCost is what one refresh of the map's data would fetch with the
// layers switched on, as if nothing were held (FR-9.2): each layer's own
// requests. The basemap is not a refresh's - it is fetched once per view and
// kept for its retention.
func refreshCost(in mapInputs, on func(key string) bool) tty.MapCost {
	var out tty.MapCost
	in.switchedOn = on // the alerts' categories, each its own switch (D-149)
	for _, l := range mapLayers {
		if l.cost == nil || !on(l.key) {
			continue // a layer riding another's requests, or radar - the mode, not an overlay chosen (D-149)
		}
		b, r := l.cost(in)
		out.Bytes += b
		out.Requests += r
	}
	return out
}
