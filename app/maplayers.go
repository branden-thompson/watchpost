package app

// maplayers.go — the map's registry of layers and sources (0.18.0 W1.13,
// FR-9.3), and the cost estimate read from it (W1.14, FR-9.2).
//
// MEMBERS REGISTER THEMSELVES, FROM THEIR OWN FILES, AND ARE DISCOVERED: the
// Settings row, the window's switching and the estimate all walk the
// registry, so a new layer or source plugs in without editing the others
// ("Discover Consumers, Don't Enumerate Them"). A layer's key is the part of
// its overlays' ids before the slash, which is how the window switches them.
// Options are the Settings table's rows (D-62), which already take a new row
// without editing the others.

import (
	"context"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// mapInputs is what the feed and each layer's cost read: the station's data,
// the selected place, and the alerts of the view (D-66, D-76).
type mapInputs struct {
	snap   *snapshot.Snapshot
	place  *snapshot.Location
	inView []snapshot.Alert // the view's alerts, which the map draws whoever holds them
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
	in := mapInputs{snap: ask.Snap, place: ask.Place}
	if lp != nil {
		in.inView = inViewOnly(lp.viewAlerts(ctx, ask.View, fetch), ask.View)
	}
	return in
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
	cost       func(in mapInputs) (bytes int64, requests int)
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
	// mapSources is every registered source; the first is the basemap until
	// radar's sources join (W8).
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
		out = append(out, tty.MapLayer{Key: l.key, Label: l.label, On: l.on})
	}
	return out
}

// refreshCost is what one refresh of the map's data would fetch with the
// layers switched on, as if nothing were held (FR-9.2): each layer's own
// requests. The basemap is not a refresh's - it is fetched once per view and
// kept for its retention.
func refreshCost(in mapInputs, on func(key string) bool) tty.MapCost {
	var out tty.MapCost
	for _, l := range mapLayers {
		if l.cost == nil || !on(l.key) {
			continue
		}
		b, r := l.cost(in)
		out.Bytes += b
		out.Requests += r
	}
	return out
}
