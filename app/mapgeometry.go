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
func resolveAlertAreas(ctx context.Context, store *zones.Store, snap *snapshot.Snapshot) map[string]geo.Area {
	out := map[string]geo.Area{}
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
	gone := map[string]bool{}
	if store != nil && len(wanted) > 0 {
		var missing []string
		held, missing = store.Zones(ctx, wanted)
		// **The store reports what it could not get, and that is kept.** It
		// was thrown away here, which made a shape built from two of nine
		// parts indistinguishable from a whole one (MG-10 defers the DECISION
		// to whatever draws; it never licensed dropping the evidence).
		for _, id := range missing {
			gone[id] = true
		}
	}
	for _, loc := range snap.Locations {
		for _, a := range loc.Alerts {
			if _, done := out[a.ID]; done {
				continue // the same alert is attached to every location it covers
			}
			if !a.Area.Empty() {
				// It brought its own ground; nothing about it is unknown.
				out[a.ID] = geo.Area{Shape: a.Area}
				continue
			}
			// **Each zone's areas are kept apart, not merged into one
			// outline** (RT-8). What draws reads an area's first ring as its
			// outline and the rest as holes, so pouring several zones' rings
			// into one list would make the second zone a hole in the first,
			// and two zones that overlap would leave the overlap - the part
			// the alert is most certainly about - unfilled. Appending whole
			// areas is what keeps them separate fills.
			var area geo.Area
			for _, id := range a.AffectedZones {
				z, ok := held[id]
				if !ok {
					if gone[id] {
						area.Missing = append(area.Missing, id)
					}
					continue
				}
				area.Shape = append(area.Shape, z.Area...)
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
