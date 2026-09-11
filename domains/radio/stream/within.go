package stream

// within.go — the transmitters inside a station's bed fence (D-77).
//
// THE BED IS THE STATION'S, NOT THE LISTENER'S. The HUM LEAD's case, and it is
// the one that settles it: *"I may, in Observer, choose 10 locations COMPLETELY
// OUTSIDE the Transmitter service area … Lone Pine's 'Fresno' relay is
// completely inappropriate for being the relay."* What the station carries
// between its reads has to come from where the station IS.
//
// THIS IS THE OFFLINE HALF, AND THE DISTINCTION MATTERS. The embedded table says
// a transmitter EXISTS; whether anyone STREAMS it comes from the relay directory,
// over the network (`Resolver.ResolveWithStatus`). So this answers "how many
// transmitters does my fence contain" — a structural fact about the fence that
// never lies and needs no network — and the bed's own row answers "which of them
// can I actually tune", at tune time, where the app is already making that call.
//
// A SETTINGS PREVIEW THAT ASKED THE DIRECTORY would make a network request per
// keystroke, and would read ZERO offline — which is a different lie.

import (
	"sort"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// Within lists the transmitters inside radiusMi of a point, nearest first.
//
// OUT OF SERVICE IS EXCLUDED, because the question this answers is what the
// operator may CHOOSE — and `Resolver.order` already refuses those, so a count
// that included them would promise a choice the tuner then refuses.
func (t *Table) Within(lat, lon, radiusMi float64) []Near {
	if t == nil || radiusMi <= 0 {
		return nil
	}
	out := make([]Near, 0, 8)
	for _, tx := range t.all { // bounded by the table (P10-02)
		if tx.Status == statusOutOfService {
			continue
		}
		km := geo.HaversineKM(lat, lon, tx.Lat, tx.Lon)
		if km > radiusMi*kmPerMile {
			continue
		}
		out = append(out, Near{tx, km})
	}
	// TIES BROKEN BY CALLSIGN, over a TOTAL order — `Nearest`'s own rule, and for
	// its reason: co-located transmitters share coordinates to six decimals, and
	// an unstable sort gave Vista the Spanish feed once (HUM LEAD, UAT
	// 2026-09-04).
	sort.Slice(out, func(i, j int) bool {
		if out[i].KM != out[j].KM {
			return out[i].KM < out[j].KM
		}
		return out[i].Callsign < out[j].Callsign
	})
	return out
}

const (
	// statusOutOfService is the table's own word for a transmitter that is not
	// carrying. Spelled once, so the count and the tuner cannot disagree.
	statusOutOfService = "OUT OF SERVICE"

	// kmPerMile converts the operator's miles to the table's kilometres.
	kmPerMile = 1.609344
)
