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
func (b Broadcaster) poolRows() []render.LocationRow {
	pool := b.area.Pool
	out := make([]render.LocationRow, 0, len(pool))
	for i, ref := range pool { // bounded by locations.PoolCap (P10-02)
		row := render.LocationRow{
			Index:      i + 1,
			Name:       ref.Label,
			Zip:        ref.Zip,
			Population: ref.Population,
			Loading:    true,
			Selected:   i == b.poolSelection(),
		}
		if mi := b.milesFromTower(ref); mi != nil {
			// THE TABLE'S OWN FORMATTER TAKES KILOMETRES, and `StationDistance` is
			// the one owner of how a distance reads in the operator's units.
			km := *mi / 0.621371
			row.StationKM = &km
		}
		if loc := b.snapshotFor(ref); loc != nil {
			row = fillPoolWeather(row, loc)
		}
		out = append(out, row)
	}
	return out
}

// snapshotFor is what the app has fetched for a pool location, or nil.
//
// BY KEY, NOT BY NAME: `snapshot.Key` is the identity the whole app matches on,
// and two centroids of one place are two locations.
func (b Broadcaster) snapshotFor(ref snapshot.LocationRef) *snapshot.Location {
	if b.pool == nil {
		return nil
	}
	want := snapshot.Key(ref)
	for i := range b.pool.Locations { // bounded by the snapshot (P10-02)
		l := &b.pool.Locations[i]
		if snapshot.Key(snapshot.LocationRef{Lat: l.Lat, Lon: l.Lon}) == want {
			return l
		}
	}
	return nil
}

// fillPoolWeather puts what the snapshot knows onto the row.
func fillPoolWeather(row render.LocationRow, l *snapshot.Location) render.LocationRow {
	// THE SAME FIELDS OBSERVER'S OWN ROW READS (body.go), through the same
	// harmonised seam — so a pool row and a watchlist row for one place cannot
	// disagree about the weather there.
	row.Loading = rowLoading(l)
	row.Conditions = l.Harmonized.Condition
	row.Now = l.Harmonized.Temp
	row.Station = l.Harmonized.Source.ModelOrStation
	row.HasAlert = len(l.Alerts) > 0
	row.AlertCount = len(l.Alerts)
	for _, al := range l.Alerts { // bounded by the location's alerts (P10-02)
		if render.AlertIsWarning(al.Event, al.Severity) {
			row.WarnAlert = true
		}
	}
	return row
}

// poolRoom is how many rows the pool takes off the frame before the running
// order is windowed.
//
// THE POOL IS NOT WHAT IS LEFT OVER (D-104). It used to be, and the arithmetic
// showed: the running order took every row it could and the pool drew twelve of
// twenty-five locations with nothing on the frame to say so — which is what the
// HUM LEAD reported ("I can only see 12 locations of the 24 location pool").
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

// poolSpan is the LOCATION POOL table and its heading, windowed on the pointer.
func (b Broadcaster) poolSpan(used int) scrollSpan {
	w := b.tableWidth()
	room := b.height - used - 2*bcInsetRows - 1
	if w <= 0 || room < 6 {
		return scrollSpan{} // no room to say anything useful (FR-7.3)
	}
	rows := b.poolRows()
	// NO CAPTION OF ITS OWN: the table's GROUP BAND already reads "L O C A T I O N
	// P O O L", and a centred heading above it said the same thing twice. The
	// scheduled table needs one because its groups name the three QUESTIONS a row
	// answers rather than the list itself.
	lines := append([]string{""}, strings.Split(b.opts().PoolTable(rows, w), "\n")...)

	// THE WINDOW FOLLOWS THE POINTER, the way the running order's does — and for
	// the same reason: only the frame knows how much room the table has.
	total, off := len(lines), 0
	if room < len(lines) {
		dataAt := len(lines) - len(rows)
		if sel := b.poolSelection(); sel >= 0 {
			if at := dataAt + sel; at >= room {
				off = at - room + 1
			}
		}
		off = max(0, min(off, len(lines)-room))
		lines = append([]string(nil), lines[off:off+room]...)
	}
	// THE FOOTER COUNTS WHAT IS ON SCREEN AGAINST WHAT EXISTS, which is the
	// reference's own row and the only thing that tells the operator the list is
	// longer than the window. IT COUNTS THE WINDOW IT IS UNDER, so a pool scrolled
	// to its end reads "Showing 16 - 25" rather than starting at one forever.
	lo := max(0, off-(total-len(rows)))
	shown := max(0, min(len(rows)-lo, len(lines)-max(0, (total-len(rows))-off)))
	return scrollSpan{
		lines: append(lines, b.poolFooter(lo, lo+shown, len(rows), w)),
		off:   off,
		total: total,
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
