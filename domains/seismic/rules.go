// Package seismic holds the earthquake rules and the USGS provider (0.11.0).
// A quake near a tracked location shows in that location's Details only if
// it passes the magnitude-graduated distance rule (HUM LEAD 2026-08-27): a
// small quake must be very close, a large one shows from far away — because
// a person feels a M3 only nearby but a M6 across a region.
package seismic

import (
	"math"
	"sort"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// MileKm converts miles to kilometres (the rule is authored in miles — the
// config's and the UI's unit — and compared against haversine kilometres).
// Exported so the provider builds query radii in km from the same constant.
const MileKm = 1.609344

// mileKm is the internal alias (the rule reads in the local unit).
const mileKm = MileKm

// Band is one step of the graduated rule: a quake whose magnitude is below
// UpperMag is shown within RadiusMi miles of the location.
type Band struct {
	UpperMag float64
	RadiusMi float64
}

// Rules are the earthquake rules: the ascending step function (Bands),
// the lookback window and the USGS event types shown.
type Rules struct {
	Bands        []Band   // ascending by UpperMag; the first whose UpperMag > mag gives the radius
	LookbackDays int      // "recent" reaches this far back
	Types        []string // USGS event types shown (default ["earthquake"] — extendable, D4)
}

// DefaultRules are the ratified defaults (objectives §4).
func DefaultRules() Rules {
	return Rules{
		Bands: []Band{
			{1.0, 3}, {2.5, 10}, {3.5, 20}, {4.0, 40}, {4.5, 100},
			{5.0, 150}, {6.0, 400}, {7.0, 500}, {99, 1000},
		},
		LookbackDays: 7,
		Types:        []string{"earthquake"},
	}
}

// Valid reports whether the rules can be applied: at least one band, bands
// ascending by magnitude with positive radii, and a positive window.
func (r Rules) Valid() error {
	if err := invariant.Check(len(r.Bands) > 0 && r.LookbackDays > 0, "seismic rules: need at least one band and a positive lookback"); err != nil {
		return err
	}
	for i, b := range r.Bands {
		// Bands ascend by magnitude with positive radii, and radius is
		// non-decreasing (a larger quake is felt at least as far) — the
		// property the concentric QueryPlan's near/regional split relies on.
		ascending := i == 0 || (b.UpperMag > r.Bands[i-1].UpperMag && b.RadiusMi >= r.Bands[i-1].RadiusMi)
		if err := invariant.Check(b.RadiusMi > 0 && ascending, "seismic rules: bands must ascend by magnitude with non-decreasing, positive radii"); err != nil {
			return err
		}
	}
	return nil
}

// RadiusMiFor is how far a quake of magnitude mag is shown from — the miles
// of the first band whose UpperMag exceeds mag; the last band's radius for
// anything at or above the top (the top band's UpperMag is a sentinel).
func (r Rules) RadiusMiFor(mag float64) float64 {
	for _, b := range r.Bands {
		if mag < b.UpperMag {
			return b.RadiusMi
		}
	}
	if len(r.Bands) == 0 {
		return 0
	}
	return r.Bands[len(r.Bands)-1].RadiusMi
}

// FloorMag is the near query's magnitude floor. The bands cover all
// magnitudes — the sub-1.0 band shows a quake within 3 mi, and USGS ml runs
// slightly negative — so the fetch must reach below zero too, or a Keep-visible
// negative-magnitude quake underfoot would be dropped (REVIEW P5 F2). It sits
// below any recorded quake; RadiusMiFor / Keep do the real gating.
func (r Rules) FloorMag() float64 { return -9 }

// MaxRadiusMi is the widest reach of any band, in miles.
func (r Rules) MaxRadiusMi() float64 {
	m := 0.0
	for _, b := range r.Bands {
		m = math.Max(m, b.RadiusMi)
	}
	return m
}

// MaxRadiusKm is the widest reach of any band, in kilometres — the radius
// the USGS box request must cover so the rule can filter it locally.
func (r Rules) MaxRadiusKm() float64 { return r.MaxRadiusMi() * mileKm }

// nearFieldCapMi bounds the near-field query's radius. Bands whose reach is
// within it collapse into one low-magnitude near query ("did it shake right
// here"); bigger bands need a wide radius but only at higher magnitude, so
// they form a separate, sparse regional query. The knee sits at 20 mi / the
// M3.5 "might feel it" boundary — measured live 2026-08-27: a single wide
// minmagnitude-0 query pulled ~1 MB (≈1,450 events, nearly all discarded by
// Keep), while the concentric split pulls 4–31 KB with identical results.
const nearFieldCapMi = 20

// BandQuery is one concentric FDSN query the provider issues: fetch events in
// the magnitude window [MinMag, MaxMag) within RadiusMi of the location.
// MaxMag == 0 means no upper bound (the open regional query).
type BandQuery struct {
	MinMag   float64
	MaxMag   float64 // 0 ⇒ no upper bound
	RadiusMi float64
}

// QueryPlan is the minimal set of concentric queries that — unioned and then
// Keep-filtered — yields exactly the visible set (the equivalence property).
// Because the graduated rule is concentric, so is the fetch: one near-field
// query (low magnitude, tight radius) plus one regional query (≥ the pivot
// magnitude, the widest radius). Derived from the bands, so it stays correct
// if they are reconfigured. Fetching the whole widest circle at magnitude 0
// would pull the entire low-magnitude field the rule then throws away.
func (r Rules) QueryPlan() []BandQuery {
	if len(r.Bands) == 0 {
		return nil
	}
	var plan []BandQuery
	// Near group: the leading bands whose reach is within the near cap (the
	// bands ascend by radius, so once one exceeds the cap all the rest do).
	near, nearRadius := 0, 0.0
	for i, b := range r.Bands {
		if b.RadiusMi > nearFieldCapMi {
			break
		}
		near, nearRadius = i+1, b.RadiusMi
	}
	if near > 0 {
		plan = append(plan, BandQuery{MinMag: r.FloorMag(), MaxMag: r.Bands[near-1].UpperMag, RadiusMi: nearRadius})
	}
	// Regional group: everything above the near cap, one wide query. Its floor
	// is the pivot magnitude (the near group's top, or 0 with no near group);
	// its radius is the rule's widest reach.
	if near < len(r.Bands) {
		pivot := 0.0
		if near > 0 {
			pivot = r.Bands[near-1].UpperMag
		}
		plan = append(plan, BandQuery{MinMag: pivot, MaxMag: 0, RadiusMi: r.MaxRadiusMi()})
	}
	return plan
}

// Keep reports whether a quake at distanceKm from the location, of the given
// magnitude, is within its band's radius — the graduated rule.
func (r Rules) Keep(mag, distanceKm float64) bool {
	return distanceKm <= r.RadiusMiFor(mag)*mileKm
}

// wants reports whether the event type is one the rules show (D4).
func (r Rules) Wants(eventType string) bool {
	for _, t := range r.Types {
		if t == eventType {
			return true
		}
	}
	return false
}

// Sort orders quakes for display (HUM LEAD 2026-08-27): largest magnitude
// first (most impactful), then nearest first among equal magnitudes.
func Sort(qs []snapshot.Quake) {
	sort.SliceStable(qs, func(i, j int) bool {
		if qs[i].Mag != qs[j].Mag {
			return qs[i].Mag > qs[j].Mag
		}
		return qs[i].DistanceKm < qs[j].DistanceKm
	})
}
