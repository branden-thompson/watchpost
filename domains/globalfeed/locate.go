package globalfeed

import (
	"strings"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// tieRadiusKm: an event within this distance of a tracked location is tied to
// it — "did this happen near one of MY places". 150 km is a felt-region scale.
const tieRadiusKm = 150

// NearestCity names a major metro near a point, for the fuzzy "the <metro>
// area" fallback; "" when none. Injected so Locate is testable without loading
// the geodata index.
type NearestCity func(lat, lon float64) string

// Locate resolves the single representative location an event is tied to (D5,
// HUM LEAD): one event, one name — never the same quake repeated per tracked
// location. In order:
//  1. the HIGHEST watchlist location within tieRadiusKm of the event point;
//  2. else the feed's own named place, cleaned to the place after "… of";
//  3. else a fuzzy "the <metro> area" from the nearest major city.
//
// hasPoint says whether (lat, lon) is a real location: a zone-only alert has
// none, so the point-based steps (watchlist tie, nearest-metro) are skipped —
// they must not tie an event to a place near Null Island, or assert a US metro
// for a distant/coordinate-less event (red-team 0.12.0 P4 F8).
func Locate(hasPoint bool, lat, lon float64, place string, watch []snapshot.LocationRef, nearest NearestCity) string {
	if hasPoint {
		for _, w := range watch { // watchlist order: the highest applicable location wins
			if geo.HaversineKM(lat, lon, w.Lat, w.Lon) <= tieRadiusKm {
				return w.Label
			}
		}
	}
	if p := cleanPlace(place); p != "" {
		return p
	}
	if hasPoint && nearest != nil {
		if c := nearest(lat, lon); c != "" {
			return "the " + c + " area"
		}
	}
	return place
}

// cleanPlace turns a feed's raw place into a spoken location: a USGS place
// ("55 km NW of Kodari, Nepal") keeps the part after the last " of "; an NWS
// areaDesc ("Oklahoma County; Cleveland County") keeps the first area.
func cleanPlace(place string) string {
	place = strings.TrimSpace(place)
	if place == "" {
		return ""
	}
	// The index comes from the lower-cased copy but slices the original; guard
	// the bound in case case-folding changed the byte length (invalid UTF-8
	// expands to U+FFFD) so a hostile place can't slice out of range (P4 F13).
	if i := strings.LastIndex(strings.ToLower(place), " of "); i >= 0 && i+len(" of ") <= len(place) {
		place = strings.TrimSpace(place[i+len(" of "):])
	}
	if i := strings.Index(place, ";"); i >= 0 {
		place = strings.TrimSpace(place[:i])
	}
	return place
}
