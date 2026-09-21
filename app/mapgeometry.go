package app

import (
	"context"

	"github.com/branden-thompson/watchpost/domains/weather/nws/zones"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// resolveAlertAreas works out what each alert covers.
//
// **It lives here because this is the only place that may name both sides.**
// The zone store is a domain and the thing that will draw is under `platform/`,
// and neither `modes/` nor `platform/` may import a domain
// (`scripts/lint-imports.sh`). So the domain fetches, this wires, and whatever
// draws is handed plain shapes it can name (MG-3).
//
// An alert that carries its own polygon uses it. **Four alerts in five carry
// none** and name forecast zones instead, and those are resolved to the zones'
// own outlines - which is the whole reason this release exists, because a map
// that drew only the polygons would be blank through most of what it is for.
//
// An alert whose zones cannot be got comes back with nothing, and that is not
// an error here. Whether a partly-known area should be drawn at all is a
// question for whatever has a view to answer it with, and this has none
// (MG-10).
func resolveAlertAreas(ctx context.Context, store *zones.Store, snap *snapshot.Snapshot) map[string]geo.Shape {
	out := map[string]geo.Shape{}
	if snap == nil {
		return out
	}
	// Every zone any alert names, asked for once (the same zone is commonly
	// named by several alerts, and by several locations' copies of one alert).
	var wanted []string
	for _, loc := range snap.Locations {
		for _, a := range loc.Alerts {
			if !a.Area.Empty() {
				continue // it brought its own; nothing to fetch
			}
			wanted = append(wanted, a.AffectedZones...)
		}
	}
	held := map[string]zones.Zone{}
	if store != nil && len(wanted) > 0 {
		held, _ = store.Zones(ctx, wanted)
	}
	for _, loc := range snap.Locations {
		for _, a := range loc.Alerts {
			if _, done := out[a.ID]; done {
				continue // the same alert is attached to every location it covers
			}
			if !a.Area.Empty() {
				out[a.ID] = a.Area
				continue
			}
			var area geo.Shape
			for _, id := range a.AffectedZones {
				if z, ok := held[id]; ok {
					area = append(area, z.Area...)
				}
			}
			out[a.ID] = area
		}
	}
	return out
}

// seedZoneShapes takes the outlines of the watched places' own zones, in the
// background, before any alert needs them.
//
// **Fetching only on demand is coldest exactly when the map matters** - severe
// weather activates many zones at once - and a place has about two zones, so
// this is a second's work once (MG-7). It was approved, built, tested and then
// not called at all, which the BUILD-exit red team found (RT-2): a ruling that
// does not run is a ruling that was not kept.
//
// Nothing waits on it. A start-up with no network is an ordinary morning.
func (lp *livePipelines) seedZoneShapes(ctx context.Context, refs []snapshot.LocationRef) {
	if lp == nil || lp.zoneShapes == nil || lp.weather == nil || len(refs) == 0 {
		return
	}
	go func() {
		// **A background convenience must never take the program down.**
		// Seeding runs off the start-up path, so a panic here would end a
		// weather station because a map shortcut failed - found by a test
		// that handed it a provider with no client.
		defer func() { _ = recover() }()
		var ids []string
		for _, ref := range refs {
			ids = append(ids, lp.weather.ZonesFor(ctx, ref)...)
		}
		lp.zoneShapes.Seed(ctx, ids)
	}()
}
