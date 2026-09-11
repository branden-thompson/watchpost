package locations

// pool.go — the Broadcaster's candidate locations (D-72).
//
// THE WATCHLIST IS THE OBSERVER'S, AND THE POOL IS THE STATION'S. The HUM LEAD
// ruled the split on 2026-09-10: "the Observer watchlist is its lineup, and a
// different rolling window / stack / list needs to serve as Broadcaster Location
// Pool for producers to create the lineup." Until then the Producer offered the
// watchlist, so the schedule could never be deeper than the number of places the
// LISTENER happened to watch — three in his UAT, against ten drawn slots
// (F-81, F-82).
//
// THE TRANSMITTER IS NOT THE DEFAULT LOCATION EITHER. It is the station's
// epicentre, "from which the service radius fence radiates" — a setting of its
// own, not a borrowed one.
//
// THREE TIERS, AND THE THIRD IS THE POINT: home, then the region's cities
// nearest-first, then the hyper-local zip places behind them. "The big value
// here is the hyper local station reports" — and the zip table is the only one
// of the two that knows Bonsall's own 92003, or San Luis Rey, exists.

import (
	"strings"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// PoolCap is how many locations the Broadcaster's pool may hold.
//
// TWENTY-FIVE, RULED (HUM LEAD, 2026-09-10: "25 is approved"). It is DELIBERATELY
// lower than the listener's world: the station reads its own region, and a pool
// wider than the operator can hold in their head is a rotation they cannot
// predict. It is also comfortably deeper than the console's ten slots, so the
// Director always has more to choose from than it needs — which is what makes
// the cadence rule (D-47, D-48) a CHOICE rather than an inventory.
const PoolCap = 25

// Pool is the station's candidate locations inside its fence, in read-priority
// order, at most limit of them.
//
// AN EMPTY TRANSMITTER YIELDS AN EMPTY POOL, deliberately: a station with
// nowhere to transmit from has no region, and the safe reading of an unset
// epicentre is "nothing" rather than "the whole country". `lineup.Fence` makes
// the same choice for the same reason.
func Pool(idx *geodata.Index, transmitter snapshot.LocationRef, radiusMi float64, limit int) []snapshot.LocationRef {
	if idx == nil || limit <= 0 || (transmitter.Lat == 0 && transmitter.Lon == 0) {
		return nil
	}
	out := make([]snapshot.LocationRef, 0, limit)
	// SEEN BY IDENTITY AND BY NAME. `snapshot.Key` is what the SCHEDULE uses to
	// refuse a duplicate card, and the label is what a LISTENER would hear
	// repeated — Vista has four zip centroids inside twenty miles of Bonsall,
	// and all four have different keys.
	keys, names := map[snapshot.LocationKey]bool{}, map[string]bool{}
	add := func(r snapshot.LocationRef) bool {
		if r.Label == "" || len(out) >= limit {
			return false
		}
		if k := snapshot.Key(r); keys[k] || names[r.Label] {
			return false
		}
		keys[snapshot.Key(r)], names[r.Label] = true, true
		out = append(out, r)
		return true
	}

	// TIER ONE — HOME. A station reads where it transmits from first: it is the
	// most local report it has, and the one its listeners are standing in.
	add(transmitter)

	// TIER TWO — THE REGION'S CITIES, nearest first. The table is already
	// population-filtered, so distance order IS "major locations first" and no
	// ranking rule is needed to produce the HUM LEAD's own example.
	for _, c := range idx.Near(transmitter.Lat, transmitter.Lon, radiusMi, limit) { // bounded by the limit (P10-02)
		ref := cityToRefFast(idx, c)
		if ref.Zip == "" {
			// THE SLOW PATH ON A MISS, WHICH IS WHAT `Seeds` ALREADY DOES: the
			// centroid scan costs ~6 ms and this runs at startup, not per
			// keystroke. Without it "Rancho Penasquitos, CA" reached the pool
			// with no postal code — and the card the operator reads names one.
			ref = cityToRef(idx, c)
		}
		add(ref)
	}

	// TIER THREE — THE HYPER-LOCAL PLACES, nearest first, filling in behind.
	//
	// THE SCAN IS WIDER THAN THE CAP because most of what it returns is a
	// DUPLICATE PLACE: four Vistas, three Oceansides. Asking for `limit` rows
	// would leave the tier short of distinct places long before the pool is
	// full.
	for _, z := range idx.NearZips(transmitter.Lat, transmitter.Lon, radiusMi, limit*zipsPerPlace) { // bounded (P10-02)
		add(zipToRef(idx, z, transmitter.TZ))
	}
	return out
}

// zipsPerPlace is how many zip centroids the scan allows per place it hopes to
// keep. MEASURED, not guessed: twenty miles around Bonsall holds 49 zip rows
// over roughly 20 distinct places, so a place costs about two and a half rows —
// four is slack, and the scan is bounded either way.
const zipsPerPlace = 4

// zipToRef builds a ref from a zip row — the package-level twin of the
// Resolver's method, which needs a Resolver only for the same index this takes
// directly.
func zipToRef(idx *geodata.Index, z geodata.ZipRow, fallbackTZ string) snapshot.LocationRef {
	label := z.Place
	if z.State != "" {
		label = z.Place + ", " + z.State
	}
	return snapshot.LocationRef{Label: label, Zip: z.Zip, Lat: z.Lat, Lon: z.Lon, TZ: tzFor(idx, z, fallbackTZ)}
}

// tzFor backfills a zip row's timezone from the city table, which is where the
// zone lives — a zip row carries none, and a ref saved without one was a real
// defect once (the B2 PTY verification).
// AND THE TRANSMITTER'S ZONE IS THE FALLBACK, because the tier that needs this
// is exactly the tier the city table does not hold: San Luis Rey, Pala, Palomar
// Mountain — every hyper-local place around Bonsall came back with no zone at
// all, and a ref saved without one was a real defect once (the B2 PTY
// verification).
//
// THE ASSUMPTION IS STATED: a place within the station's own fence keeps the
// station's clock. At the ruled maximum of fifty miles that is true everywhere
// but astride a zone boundary, and the city table's answer is still preferred
// wherever there is one — this is the floor, not the rule.
func tzFor(idx *geodata.Index, z geodata.ZipRow, fallback string) string {
	for _, c := range idx.PrefixSearch(z.Place, 8) { // bounded by the limit (P10-02)
		if strings.EqualFold(c.State, z.State) && (strings.EqualFold(c.Name, z.Place) || strings.EqualFold(c.ASCII, z.Place)) {
			return c.TZ
		}
	}
	return fallback
}
