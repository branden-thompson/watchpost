package config

// broadcaster.go — the station's own settings (D-72).
//
// THE TRANSMITTER IS NOT THE DEFAULT LOCATION. The HUM LEAD ruled the split on
// 2026-09-10: "Default Location no longer = Transmitter Location — this is
// Broadcaster epicenter from which the service radius fence radiates from."
// They were one field doing two jobs, and the jobs belong to different
// surfaces: the default location is where the LISTENER lives, and the
// transmitter is where the STATION broadcasts from.
//
// THEY MAY WELL BE THE SAME PLACE, and on an existing install they start that
// way — see `Station`. What changed is that they no longer have to be.

import "github.com/branden-thompson/watchpost/platform/invariant"

// Broadcaster is the station's settings: where it transmits from, and how far
// its service area reaches.
type Broadcaster struct {
	// Transmitter is the station's epicentre. Empty means "not set yet", and
	// `Station` falls back to the listener's default location for it.
	Transmitter Location `toml:"transmitter,omitempty"`

	// ServiceRadiusMi is how far the fence reaches, in statute miles. Zero
	// means the default; anything outside the ruled bounds is clamped into
	// them rather than refused, because a service area is a preference and a
	// station should still come up.
	ServiceRadiusMi float64 `toml:"service_radius_mi,omitempty"`

	// BedRadiusMi is how far the station looks for a RELAY to carry on its bed
	// (D-77) — a different question from how far it READS, and therefore a
	// different number.
	//
	// IT IS WIDER THAN THE SERVICE RADIUS BECAUSE RELAYS ARE SPARSE, and that is
	// measured rather than assumed. Around Bonsall: 25 miles holds ONE
	// transmitter, 50 holds three, 100 holds eight. A selector with one choice
	// in it is not a selector.
	//
	// AND IT IS NOT DERIVED FROM THE SERVICE RADIUS. The two move for different
	// reasons — one is who the station is FOR, the other is what hardware
	// happens to exist nearby — and deriving one from the other would couple two
	// numbers that have nothing to say to each other.
	BedRadiusMi float64 `toml:"bed_radius_mi,omitempty"`
}

const (
	// MinServiceRadiusMi and MaxServiceRadiusMi are the HUM LEAD's ruled bounds
	// (2026-09-10): "minimum distance is 2 miles, max distance is 50mi."
	//
	// TWO MILES IS LEGAL AND NEARLY EMPTY, and that is measured rather than
	// feared: around Bonsall a two-mile fence holds NO city and one zip place —
	// the station's own. The floor is the operator's to choose; what the console
	// owes them is to say honestly how few it holds (F-83).
	MinServiceRadiusMi = 2
	MaxServiceRadiusMi = 50

	// DefaultServiceRadiusMi is what a station that has not chosen gets.
	//
	// TWENTY-FIVE, MEASURED AGAINST THE CONSOLE'S TEN SLOTS. Around Bonsall a
	// 10-mile fence yields 7 candidates, 20 yields 18 and 25 fills the pool's
	// whole cap — so this is the narrowest radius that reliably gives the
	// Director more to choose from than it must schedule, which is what makes
	// the cadence rule a choice rather than an inventory.
	DefaultServiceRadiusMi = 25

	// MinBedRadiusMi, MaxBedRadiusMi and DefaultBedRadiusMi bound the relay
	// search (HUM LEAD, 2026-09-10: "I think it should default to 100").
	//
	// MEASURED AT FOUR PLACES, not chosen: at 100 miles Bonsall reaches 8
	// transmitters, Lone Pine 4, Chicago 16 and Minot 4 — a real choice
	// everywhere sampled. Below 75 some real stations reach NONE (Lone Pine has
	// no relay inside fifty miles), which is what the operator is warned about
	// rather than prevented from choosing.
	//
	// THE FLOOR IS 25 BECAUSE EVEN CHICAGO REACHES ONLY ONE THERE, and the
	// ceiling is 150 because past it the bed is carrying a forecast for a region
	// the station's listeners are not in — which is the whole objection that
	// made the bed the station's business in the first place.
	MinBedRadiusMi     = 25
	MaxBedRadiusMi     = 150
	DefaultBedRadiusMi = 100
)

// ServiceRadiusMi is the fence's radius, defaulted and clamped into the ruled
// bounds.
//
// CLAMPED, NOT REFUSED. A hand-edited config with `service_radius_mi = 500` is a
// person asking for a wider station, not a corrupt file; the answer is the
// widest station they may have, and a station that comes up.
func (b Broadcaster) ServiceRadius() float64 {
	if b.ServiceRadiusMi == 0 {
		return DefaultServiceRadiusMi
	}
	return min(max(b.ServiceRadiusMi, MinServiceRadiusMi), MaxServiceRadiusMi)
}

// BedRadius is how far the station looks for a relay, defaulted and clamped.
//
// CLAMPED, NOT REFUSED, for `ServiceRadius`'s reason: a hand-edited config
// asking for a wider search is a person asking for more choice, and the answer
// is the widest search they may have plus a station that comes up.
func (b Broadcaster) BedRadius() float64 {
	if b.BedRadiusMi == 0 {
		return DefaultBedRadiusMi
	}
	return min(max(b.BedRadiusMi, MinBedRadiusMi), MaxBedRadiusMi)
}

// Station is where the station transmits from, falling back to the listener's
// default location when no transmitter has been set.
//
// THE FALLBACK IS WHAT MAKES THE SPLIT FREE. Every install that exists today has
// a default location and no transmitter; without this they would all come up
// with no epicentre, an empty pool and a console that shimmers for ever — a
// migration dressed as a feature. With it, the split costs an existing station
// nothing and the operator moves the transmitter when they want to.
func (c Config) Station() (Location, bool) {
	if c.Broadcaster.Transmitter.Lat != 0 || c.Broadcaster.Transmitter.Lon != 0 {
		return c.Broadcaster.Transmitter, true
	}
	if err := invariant.Check(len(c.Locations) >= 0, "a configuration has a location list, empty or not"); err != nil {
		return Location{}, false
	}
	if len(c.Locations) == 0 {
		return Location{}, false
	}
	return c.Locations[0], true
}
