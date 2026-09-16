package tty

// broadcaster_pool.go — the station's candidate locations, with enough weather to
// decide on them (D-98).
//
// HUM LEAD, 2026-09-12: "Location Pool table should fill with our 25 location cap,
// and should function just like Observer's weather table — RATIONALE: gives the
// human operator some basic weather info to determine if they want to have that
// location prioritized in the schedule line-up."
//
// IT IS OBSERVER'S TABLE, NOT A COPY OF IT (render.PoolTable): the same marks, the
// same bands, the same temperatures, one column more. Being the same CODE is the
// only way to be sure of "the same labelling / colorscheme / behavior approach".

import (
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// poolRows is the published pool, as table rows.
//
// THE WEATHER IS WHATEVER THE SNAPSHOT HAS, and for now that is nothing: the pool
// is not fetched yet (its cadence is ruled at StaleAfter and the wiring follows).
// A row with no observation draws its temperatures as LOADING rather than as
// "n/a", which is the same answer Observer gives while its own data is in flight
// (UAT 18.2) — and the honest one, because the data is coming.
func (b Broadcaster) poolRows(idx locIndex) []render.LocationRow {
	pool := b.area.Pool
	out := make([]render.LocationRow, 0, len(pool))
	for i, ref := range pool { // bounded by locations.PoolCap (P10-02)
		// THE ONE CONVERTER (D-112). `fillPoolWeather` was a second, hand-written
		// one, and it copied six of sixteen fields — HI, LOW, all of TOMORROW, the
		// trend arrow and the fire and seismic marks simply never reached the
		// table. `weatherRow` is what Observer's own rows go through.
		row := render.LocationRow{Name: ref.Label, Zip: ref.Zip, Loading: true}
		if loc := idx.at(ref); loc != nil {
			row = weatherRow(loc, b.fireBold(), 0)
		}
		// AND WHAT THE POOL KNOWS THAT THE SNAPSHOT DOES NOT: which row this is,
		// how many people the place serves, and whether the pointer is on it.
		row.Index, row.Population, row.Selected = i+1, ref.Population, i == b.poolSelection()
		// THE DISTANCE IS FROM THE TRANSMITTER, NOT FROM THE OBSERVING STATION.
		// `weatherRow` fills `StationKM` with how far the WX STN is from the place;
		// this column answers a different question — how far the PLACE is from the
		// tower — which is what an operator sizing up a candidate is asking.
		if mi := b.milesFromTower(ref); mi != nil {
			// THE TABLE'S OWN FORMATTER TAKES KILOMETRES, and `StationDistance` is
			// the one owner of how a distance reads in the operator's units.
			km := *mi / 0.621371
			row.StationKM = &km
		}
		out = append(out, row)
	}
	return out
}

// locIndex is the published weather BY KEY, built once per frame (D-120).
//
// IT WAS A LINEAR SCAN, ASKED ONCE PER ROW. `snapshotFor` walked the whole
// snapshot for every pool row and — since D-116 — for every line-up row too:
// forty lookups over the snapshot, on every frame, including ticks that changed
// nothing.
//
// AND THE SAVING IS SMALLER THAN IT LOOKS, which is worth writing down next to
// the change rather than leaving for the next person to re-derive. At the pool's
// cap of twenty-five the scan is a few hundred short string compares against
// half a millisecond of rendering; the map costs an allocation and twenty-five
// hashes to build. MEASURED, both ways, on the loaded fixture below — see the
// budget's own note. It is kept because it is O(1) per row where the scan is
// O(n), and the pool's cap is the only thing keeping n small.
//
// BY KEY, NOT BY NAME: `snapshot.Key` is the identity the whole app matches on,
// and two centroids of one place are two locations.
type locIndex map[snapshot.LocationKey]*snapshot.Location

// locIndex builds it from the recent snapshot.
func (b Broadcaster) locIndex() locIndex {
	if b.pool == nil {
		return nil
	}
	out := make(locIndex, len(b.pool.Locations))
	for i := range b.pool.Locations { // bounded by the snapshot (P10-02)
		l := &b.pool.Locations[i]
		out[snapshot.Key(snapshot.LocationRef{Lat: l.Lat, Lon: l.Lon})] = l
	}
	return out
}

// at is what the app has fetched for a location, or nil.
func (x locIndex) at(ref snapshot.LocationRef) *snapshot.Location {
	if x == nil {
		return nil
	}
	return x[snapshot.Key(ref)]
}

// `fillPoolWeather` RETIRED AT D-112. It was the second converter from a
// snapshot to a table row, and it had drifted from the first the day it was
// written: `weatherRow` is now the only one, and both surfaces go through it.

// fireBold is the console's threshold for a hotspot that reads emphasized.
//
// IT WAS A CONSTANT (`bcFireBoldMW = 50`) — Observer's default, stated here
// because the console had no Config to read the operator's override from. That
// made it a SECOND CARRIER of one fact, and it disagreed: an operator who set
// `bold_frp_mw` in [fire] saw it honoured on the watchlist and ignored here, so
// one location could read bold on one surface and plain on the other.
//
// RESOLVED BY THE DASHBOARD AND COPIED IN (NewRouter), the way `ascii` and
// `version` are. `cfg` is written once at construction and never reassigned, so
// there is nothing later to follow. The zero fallback is for a console built
// without a Router — the older tests — and it reads THE SAME CONSTANT Observer
// falls back to, so the two cannot drift apart again.
func (b Broadcaster) fireBold() float64 {
	if b.fireBoldMW > 0 {
		return b.fireBoldMW
	}
	return fireBoldDefaultMW
}

// poolRoom is how many rows the pool takes off the frame before the running
// order is windowed.
//
// THE POOL IS NOT WHAT IS LEFT OVER (D-104). Left over, the arithmetic shows:
// the running order takes every row it can and the pool draws twelve of
// twenty-five locations with nothing on the frame to say so — which is how the
// HUM LEAD read it ("I can only see 12 locations of the 24 location pool").
//
// TEN ROWS OF LOCATIONS, plus the band, the column header, the air above and the
// footer. It is a FLOOR, not a share: on a short terminal the running order gives
// way first, because the pool is where the operator SHOPS and the running order
// is what they are running.
func (b Broadcaster) poolRoom() int {
	if b.height-2*bcInsetRows < bcMinRows {
		return 0
	}
	// IT NEVER RESERVES ROOM FOR LOCATIONS THAT DO NOT EXIST. A station with two
	// candidates would otherwise hold ten rows of air out of the running order's
	// reach, which is the opposite of what a floor is for.
	return min(bcPoolRows, len(b.area.Pool)) + bcPoolChrome
}

const (
	// bcPoolRows is how many locations the pool shows before it scrolls.
	bcPoolRows = 10
	// bcPoolChrome is what the pool spends around them: the air above, the three
	// band rows, the column header, and the "Showing" footer.
	bcPoolChrome = 6
)

// poolSpan is the LOCATION POOL table, its heading, and where its window sits.
//
// THE HEADINGS DO NOT SCROLL (D-106). Windowing the whole block, band and column
// titles included, loses the operator the names of the columns the moment they
// move down the list — "the top of the vertical scroll aligns with the headers of
// the table (so they dont disappear when I scroll down)".
//
// `head` IS FIXED AND `data` IS THE WINDOW, which is Observer's own division: the
// band and the column titles are chrome, and the rows are the list.
func (b Broadcaster) poolSpan(used int, idx locIndex) scrollSpan {
	w := b.tableWidth()
	// THE CLOSING INSET ONLY (D-107). `used` is the rows already drawn, and those
	// INCLUDE the opening inset — subtracting both left the pool two rows short of
	// the frame and the operator two locations short of what fitted.
	room := b.height - used - bcInsetRows
	if w <= 0 || room < 6 {
		return scrollSpan{} // no room to say anything useful (FR-7.3)
	}
	rows := b.poolRows(idx)
	// NO CAPTION OF ITS OWN: the table's GROUP BAND already reads "L O C A T I O N
	// P O O L", and a centred heading above it said the same thing twice. The
	// scheduled table needs one because its groups name the three QUESTIONS a row
	// answers rather than the list itself.
	table := strings.Split(b.opts().PoolTable(rows, w), "\n")
	// THE SPLIT IS BY COUNT, taken from the table itself: everything that is not
	// a row is chrome. Asking the renderer how many rows it drew is the only way
	// that cannot drift from what it actually drew.
	headN := len(table) - len(rows)
	head := append([]string{""}, table[:headN]...)
	data := table[headN:]

	window := max(0, room-len(head)-1) // the footer is part of the budget
	lo := 0
	if window < len(data) {
		if sel := b.poolSelection(); sel >= window {
			lo = sel - window + 1
		}
		lo = max(0, min(lo, len(data)-window))
		data = data[lo : lo+window]
	}
	return scrollSpan{
		lines: append(append(head, data...), b.poolFooter(lo, lo+len(data), len(rows), w)),
		// THE CONTROL STARTS ON THE LAST FIXED ROW — the column titles — so ▲
		// marks where the list begins rather than where the section does.
		from:  len(head) - 1,
		off:   lo,
		total: len(rows),
	}
}

// poolFooter is the reference's "Showing 1 - n of N" line.
func (b Broadcaster) poolFooter(lo, hi, total, w int) string {
	if total == 0 {
		return ""
	}
	s := "Showing " + strconv.Itoa(lo+1) + " - " + strconv.Itoa(hi) + " of " + strconv.Itoa(total) + " Location Pool Locations"
	return render.PadTo(strings.Repeat(" ", max(0, w-len(s)))+s, w)
}
