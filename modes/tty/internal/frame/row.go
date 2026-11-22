package frame

// row.go — the location row both surfaces draw: its day cells, its trend,
// its loading state and its marks, with the fire threshold's default and the
// seismic level the marks read (0.19.6 A-21).

import (
	"fmt"
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// WeatherRow is everything a SNAPSHOT knows about a location, as a table row.
//
// THE ONE CONVERTER (0.16.0 D-112), for the Broadcaster's pool as for the
// watchlist. A hand-built second copy carries only the fields it had on the day
// it was written, and every column it lacks — HI, LOW, TOMORROW, the trend
// arrow, the fire and seismic marks — draws `n/a` on a table whose whole
// purpose the HUM LEAD stated as "gives the human operator some basic weather
// info to determine if they want to have that location prioritized".
//
// Two carriers of one conversion drift silently. The pool table and the
// watchlist table are the SAME renderer already (`render.PoolTable` is
// `tableFor` with one flag); this makes the data behind them the same too.
//
// WHAT IT DOES NOT SET is what belongs to a SURFACE rather than to the weather:
// the row's number, the operator's pointer, what the radio is playing, and — on
// the pool — how far the place is from the TRANSMITTER rather than from the
// station that observed it.
func WeatherRow(loc *snapshot.Location, fireBoldMW float64, extDays int) render.LocationRow {
	row := render.LocationRow{
		Name:       loc.Label,
		Zip:        loc.Zip,
		Station:    loc.Harmonized.Source.ModelOrStation, // WX STN / DIST (UAT 60)
		StationKM:  loc.Harmonized.Source.DistanceKm,
		Conditions: loc.Harmonized.Condition, // display mapping in the seam (P.CLOUDY etc)
		Now:        loc.Harmonized.Temp,
		Trend:      Trend(*loc),
		HasAlert:   len(loc.Alerts) > 0,
		AlertCount: len(loc.Alerts),                        // ⚠ badge (UAT 20.2)
		Fire:       fireCount(loc.Fire),                    // B5 / UAT 110: n◆
		FireHot:    fireHot(loc.Fire.Hotspots, fireBoldMW), // B5
		Seismic:    seismicRowLevel(loc.Seismic),           // the strongest quake's felt-band glyph
		// UAT 18.2: shimmer until this location's data lands (obs or daily
		// still pending); post-load nils stay honest "n/a".
		Loading: RowLoading(loc),
	}
	for _, al := range loc.Alerts { // bounded by the location's alerts (P10-02)
		if render.AlertIsWarning(al.Event, al.Severity) {
			row.WarnAlert = true // warning-grade outranks advisory (UAT 14.1)
			break
		}
	}
	if len(loc.Daily) > 0 {
		row.Hi, row.Lo = loc.Daily[0].TempMax, loc.Daily[0].TempMin
	}
	if len(loc.Daily) > 1 {
		row.TomorrowConditions = loc.Daily[1].Condition
		row.TomorrowHi, row.TomorrowLo = loc.Daily[1].TempMax, loc.Daily[1].TempMin
	}
	// THE EXTENDED DAYS ARE THE CALLER'S ASK (0.16.0 D-120). Observer's ultra-wide
	// columns draw them; the console's two tables do not, and building five day
	// cells a row for a table that never reads them is work spent on nothing.
	for _, day := range loc.Daily[min(2, len(loc.Daily)):] { // bounded by the forecast (P10-02)
		if len(row.Extended) >= extDays {
			break
		}
		row.Extended = append(row.Extended, render.DayCell{Date: mmdd(day.Date), Hi: day.TempMax, Lo: day.TempMin})
	}
	return row
}

// mmdd shortens an ISO date to the mock's mm/dd column header.
func mmdd(iso string) string {
	if len(iso) < 10 {
		return iso
	}
	return iso[5:7] + "/" + iso[8:10]
}

// Trend derives the NOW arrow from the next forecast hour vs current temp
// (mock: 888°F↗; ±0.3°C deadband so noise doesn't flicker the arrow).
func Trend(loc snapshot.Location) string {
	if loc.Harmonized.Temp == nil || len(loc.Hourly) == 0 || loc.Hourly[0].Temp == nil {
		return ""
	}
	delta := *loc.Hourly[0].Temp - *loc.Harmonized.Temp
	switch {
	case delta > 0.3:
		return "up"
	case delta < -0.3:
		return "down"
	}
	return ""
}

// RowLoading is the one definition of "this row is still loading"
// (UAT 18.2): observation or daily forecast still pending.
// RowLoading: shimmer while the data is still COMING, never after it has been
// asked for and not arrived (issue #13).
//
// THE SECOND CLAUSE IS WHAT ENDS THE SHIMMER. An empty location is empty in
// exactly the same way whether the feed has not answered yet or has answered and
// had nothing for this place, so on the first clause alone a location the API
// does not cover shimmers for ever, across restarts, reading as "still loading"
// until the listener removes the row themselves.
//
// WeatherAsOf is what makes the two distinguishable: the reference provider
// stamps it when a fetch COVERING this location completes. Once it is set, a
// missing value is a fact rather than a wait, and the row falls through to the
// honest "n/a" the post-load path already draws (UAT 18.2). Nothing new is
// rendered — without the second clause the flag could never go false.
func RowLoading(loc *snapshot.Location) bool {
	if !loc.WeatherAsOf.IsZero() {
		return false
	}
	return loc.Harmonized.Source.Provider == "" || len(loc.Daily) == 0
}

// fireCount is the row badge's number (UAT 110): the named incidents
// nearby when there are any, else 1 for unnamed hotspots, 0 for no fire.
func fireCount(fs snapshot.FireState) int {
	switch {
	case len(fs.Incidents) > 0:
		return len(fs.Incidents)
	case len(fs.Hotspots) > 0:
		return 1
	}
	return 0
}

// fireHot reports whether any hotspot reads emphasized.
func fireHot(hs []snapshot.Hotspot, boldMW float64) bool {
	for _, h := range hs {
		if h.FRPMW != nil && *h.FRPMW >= boldMW {
			return true
		}
	}
	return false
}

// FireBoldDefaultMW is the emphasis threshold when the operator has set none.
//
// ONE OWNER, TWO READERS — this Dashboard and the console (Broadcaster.fireBold).
// A console constant of its own could not see the override, so one location
// would read bold on the watchlist and plain on the console.
const FireBoldDefaultMW = 50

// seismicRowLevel is the felt-band level the main-table row mark wears
// (0.11.0, HUM LEAD): the strongest recent quake's level, or 0 for none. The
// quakes are sorted largest-first, so the strongest is the first.
func seismicRowLevel(ss *snapshot.SeismicState) int {
	if ss == nil || len(ss.Quakes) == 0 {
		return 0
	}
	return render.SeismicLevel(ss.Quakes[0].Mag)
}

// LevelControl renders "<LABEL> [-]█████░░░░░[+] 55" — the ONE bar
// (0.16.0 D-56).
//
// THE CONSOLE CALLS IT "GAIN" AND OBSERVER CALLS IT "VOL", and that is the whole
// difference (HUM LEAD, 2026-09-10: "gain is the VOL control in Observer"). A
// second bar would be a second place the level is drawn, and two bars
// disagreeing about how loud the station is would be worse than either.
//
// The bar only steps at the tens, which is why the chips blink: the blink is the
// per-press feedback (UAT 41). The level always occupies 3 cells (0-100) so the
// row never jitters (UAT 42.1).
func LevelControl(o render.Opts, label string, level int, minus, plus string, width int) string {
	filled := width * level / 100
	return render.TintRaw(label, "1;97") + minus +
		render.Tint(strings.Repeat(o.Glyphs().RailCar, filled), render.Tok(render.RadioAccent)) +
		strings.Repeat(o.Glyphs().Fill, width-filled) +
		plus + " " + render.Tint(fmt.Sprintf("%3d", level), render.Tok(render.TextBright))
}
