package app

// metro.go — the fuzzy "the <metro> area" tie for an event with a point but no
// useful place name.
//
// SPLIT OUT OF ticker.go (2026-09-06), a pure move. See seen_store.go's header.

import (
	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/geo"
)

// metroCapKm bounds the fuzzy tie: past this the nearest US metro is not a
// meaningful description, so an event out at sea or overseas isn't asserted to
// be near a US city (red-team 0.12.0 P4 F15).
const metroCapKm = 400

// nearestMetro resolves the nearest major US metro to a point (for the fuzzy
// "the <metro> area" tie); it searches the top cities so the scan is bounded.
func nearestMetro(idx *geodata.Index) globalfeed.NearestCity {
	if idx == nil {
		return nil
	}
	top := idx.TopUS(300)
	return func(lat, lon float64) string {
		best, bestKm := "", 0.0
		for _, c := range top {
			km := geo.HaversineKM(lat, lon, c.Lat, c.Lon)
			if best == "" || km < bestKm {
				best, bestKm = c.Name, km
			}
		}
		if bestKm > metroCapKm {
			return "" // nothing close enough to name
		}
		return best
	}
}
