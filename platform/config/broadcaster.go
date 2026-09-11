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
