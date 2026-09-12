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

// poolLines is the LOCATION POOL table and its heading, in whatever height the
// frame has left.
func (b Broadcaster) poolLines(used int) []string {
	w := b.frameWidth()
	room := b.height - used - 2*bcInsetRows
	if w <= 0 || room < 6 {
		return nil // no room to say anything useful (FR-7.3)
	}
	rows := b.poolRows()
	// NO CAPTION OF ITS OWN: the table's GROUP BAND already reads "L O C A T I O N
	// P O O L", and a centred heading above it said the same thing twice. The
	// scheduled table needs one because its groups name the three QUESTIONS a row
	// answers rather than the list itself.
	lines := append([]string{""}, strings.Split(b.opts().PoolTable(rows, w), "\n")...)

	// THE FOOTER COUNTS WHAT IS ON SCREEN AGAINST WHAT EXISTS, which is the
	// reference's own row and the only thing that tells the operator the list is
	// longer than the window.
	shown := max(0, min(len(rows), room-len(lines)+len(rows)))
	if len(lines) > room {
		lines = lines[:room]
	}
	// NO RAIL OF ITS OWN, YET. The reference draws ONE rail down the right of the
	// whole running order — the scheduled table AND the pool under it, with a
	// single ▲ at the top and ▼ at the bottom. Two rails put two ▲/▼ pairs on the
	// frame and interleaved them, which a test caught immediately: "the up cap is
	// above the down cap". One rail over two scrolling regions is its own piece of
	// work; until then the scheduled table keeps the rail and the pool does not
	// pretend to have one.
	return append(lines, b.poolFooter(shown, len(rows), w))
}

// poolFooter is the reference's "Showing 1 - n of N" line.
func (b Broadcaster) poolFooter(shown, total, w int) string {
	if total == 0 {
		return ""
	}
	s := "Showing 1 - " + strconv.Itoa(shown) + " of " + strconv.Itoa(total) + " Location Pool Locations"
	return render.PadTo(strings.Repeat(" ", max(0, w-len(s)-2))+s, w)
}
