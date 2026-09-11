package lineup

import (
	"math"

	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

// kmPerMi converts the fence's radius, and is written out here rather than
// imported.
//
// THE ARITHMETIC IS DELIBERATELY IDENTICAL to globalfeed.WithinMiles
// (domains/globalfeed/stack.go:22) — the same haversine, the same constant, the
// same `<=`. platform/ cannot import domains/*, and extracting a shared helper
// would mean editing the hazard path to route through it, where a rounding
// difference at the boundary would put one hazard on the tape and not in the
// burst. scopeEvents already warns that "the two surfaces cannot disagree about
// one hazard"; matching the form is how this one keeps that promise, and this
// comment is the reason it is not extracted at its second caller.
const kmPerMi = 1.609344

// QuakeReachFrom is the magnitude at which a quake starts carrying reach of its
// own. It is the feed's own "strong quake" threshold (domains/globalfeed/
// usgs.go:171), so nothing below what the app already calls strong buys an
// exception to the listener's fence.
const QuakeReachFrom = 5.5

// quakeReachBase is how far a quake at QuakeReachFrom reaches, in miles. Reach
// then DOUBLES per magnitude point.
const quakeReachBase = 32.0

// quakeReachTo is the largest magnitude the scale is meaningful at: M9.5, the
// largest earthquake ever recorded (Chile, 1960).
//
// THE CEILING IS ON THE MAGNITUDE, NOT ON THE DISTANCE, because that is where
// the unreality is. USGS clamps its own magnitude field to 12
// (domains/globalfeed/usgs.go:111), which is a bound on bad data rather than a
// magnitude that can occur — and doubling per point, an M12 would reach 2,896
// miles and pull in Fairbanks, contradicting the very ruling this scale exists
// to encode. Clamping the input keeps the exception SCALED BY SIGNIFICANCE
// (G-7) rather than by whatever a malformed feed row happens to say.
const quakeReachTo = 9.5

// QuakeReachMi is how far a significant quake carries its own effects, in miles.
// Zero below QuakeReachFrom: an ordinary quake buys no exception at all.
//
// THE PRINCIPLE, in the HUM LEAD's words (BD-6, ratified 2026-09-02): the two
// cases illustrate "the correlation between distance and urgency / importance".
// A hazard matters less the further away it is — that is what the fence encodes
// — and a big enough hazard pushes the point at which it stops mattering further
// out. Reach is how far one hazard's own significance carries that.
//
// So the scale DOUBLES per magnitude point, which is the shape the principle
// asks for: magnitude is logarithmic, and a linear reach would either fence out
// the great quakes or drag in every moderate one.
//
//	M5.5 →  32 mi     M7.5 → 128 mi ← covers "an M7.5 in Los Angeles (~120 mi)"
//	M6.5 →  64 mi     M8.5 → 256 mi
//	M7.0 →  91 mi     M9.0 → 362 mi ← Fairbanks is ~2,530 mi; not admitted
//
// The ~120 is ILLUSTRATIVE, not a floor — it is what a person means by how far
// away Los Angeles is, and the ruling used it to show the correlation rather
// than to set a threshold. The scale is still asserted against it, because a
// scale that cannot reproduce the example the rule was explained with is not
// encoding the rule.
//
// It is SCALED BY SIGNIFICANCE, NOT UNBOUNDED (G-7 — Fairbanks is the
// counter-example), and the cascade needs no modelling: a derived hazard — the
// tsunami warning for San Diego behind that M9.0 — arrives as its own local
// alert inside the fence (DR-13).
func QuakeReachMi(mag float64) float64 {
	if mag < QuakeReachFrom {
		return 0
	}
	if mag > quakeReachTo {
		mag = quakeReachTo
	}
	reach := quakeReachBase * math.Pow(2, mag-QuakeReachFrom)
	// NOT `reach >= quakeReachBase`, which was the first form and was WRONG:
	// that is the threshold guard above said a second time, so deleting the
	// guard left this to return 0 in its place and the deletion changed nothing
	// observable. m83 survived on exactly that, and a check that can stand in
	// for the rule it is checking is not an invariant.
	if err := invariant.Check(!math.IsInf(reach, 0) && !math.IsNaN(reach), "a magnitude never buys unbounded reach"); err != nil {
		return 0
	}
	if err := invariant.Check(mag <= quakeReachTo, "the magnitude was clamped before it was raised"); err != nil {
		return 0
	}
	return reach
}

// Fence is the listener's service radius: a HARD boundary on what reaches the
// lineup at all (DR-13, G-7), not a sort key. What it keeps out is not read, not
// counted and not pointed at.
//
// The zero value is no fence — "All" — which is the DEFAULT PATH:
// config.TickerRadiusMi defaults to 0 (platform/config/config.go:260), so every
// fresh install is here.
type Fence struct {
	// RadiusMi is how far the listener's world extends. Zero is All.
	RadiusMi float64

	// Lat and Lon are the default location the radius is measured from, and
	// HasOrigin is whether one is set at all. A fence with nowhere to measure
	// from admits NOTHING rather than falling back to the global stack the UI
	// says is scoped away — today's rule (app/ticker.go:tickerDeck.scopeToRadius,
	// which returns nil with no watchlist), unchanged.
	Lat, Lon  float64
	HasOrigin bool
}

// InForce reports whether a radius is set.
//
// THE ONLY CARRIER OF THAT QUESTION. Admission asks it, and so does the ordering
// rule that makes Disasters outrank Warnings unconditionally (DR-12) — one
// question with two askers, never a boolean kept beside the radius that could
// disagree with it.
func (f Fence) InForce() bool {
	if err := invariant.Check(!math.IsNaN(f.RadiusMi), "the radius is a number"); err != nil {
		return false
	}
	return f.RadiusMi > 0
}

// Admits reports whether an arrival reaches the lineup at all.
func (f Fence) Admits(a Arrival) bool {
	if !f.InForce() {
		return true
	}
	if !f.HasOrigin {
		return false
	}
	if !a.HasPoint {
		// A zone-only alert has no point to measure. It reaches a scoped surface
		// only by being one the app is already tracking at a watched location —
		// today's rule (app/severe.go:scopeEvents), said the same way here so
		// the tape and the burst cannot disagree about one hazard.
		return a.Tracked
	}
	km := geo.HaversineKM(f.Lat, f.Lon, a.Lat, a.Lon)
	// A DISTANCE IS A NUMBER, and the failure if it is not is silent in the worst
	// direction: NaN compares false against everything, so a malformed point
	// fails the radius test AND the significance exception, and the hazard is
	// fenced out rather than admitted. A feed that parses a coordinate badly
	// would quietly empty the burst.
	if err := invariant.Check(!math.IsNaN(km), "the distance to an arrival is a number"); err != nil {
		return false
	}
	if km <= f.RadiusMi*kmPerMi {
		return true
	}
	// THE SIGNIFICANCE EXCEPTION, and it is for a DISASTER. Such a disaster has
	// proximal effects, so it carries its own reach; a warning a thousand miles
	// away is still a warning a thousand miles away, however severe.
	if a.Category != category.Disasters {
		return false
	}
	return km <= a.ReachMi*kmPerMi
}

// AdmitsAny reports whether a fence admits ANY of the arrivals a card was
// planned from (D-75).
//
// ANY, NOT ALL. A burst is one card carrying several hazards; if even one of
// them is inside the service area, the card is about something the operator
// needs to hear. Holding it because a companion alert was farther out would
// silence a hazard in their own town.
//
// A CARD PLANNED FROM NOTHING IS ADMITTED. The Director's own structural cards
// carry no arrivals, and a fence is a rule about where HAZARDS are — not a
// reason to hold a transition or a station credit.
func (f Fence) AdmitsAny(from []Arrival) bool {
	if len(from) == 0 {
		return true
	}
	for _, a := range from { // bounded by the burst's Max (P10-02)
		if f.Admits(a) {
			return true
		}
	}
	return false
}
