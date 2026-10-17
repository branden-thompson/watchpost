package tty

import "time"

// propRefresh is how often an open Propagation mode asks again (FR-4.10,
// D-120): each new GloTEC grid by default (D-94), hourly, or only on the
// refresh-now key.
type propRefresh uint8

// The choices, the default first.
const (
	refreshEvery10  propRefresh = iota // each new GloTEC grid, at most every 10 minutes (D-94)
	refreshHourly                      // once an hour
	refreshOnDemand                    // only the refresh-now key (D-120)
)

// propRefreshChoices are the Setting's choices, in its order.
var propRefreshChoices = []propRefresh{refreshEvery10, refreshHourly, refreshOnDemand}

// propRefreshKeys are the file's names for them; the default is written
// empty.
var propRefreshKeys = map[propRefresh]string{refreshEvery10: "", refreshHourly: "hourly", refreshOnDemand: "on-demand"}

// propRefreshByKey reads the file's name; anything not offered is the
// default.
func propRefreshByKey(key string) propRefresh {
	for r, k := range propRefreshKeys {
		if k == key {
			return r
		}
	}
	return refreshEvery10
}

// propRefreshKey is the file's name for a choice.
func propRefreshKey(r propRefresh) string { return propRefreshKeys[r] }

// every is how long the refresh tick waits; zero on demand, when none is set.
func (r propRefresh) every() time.Duration {
	switch r {
	case refreshHourly:
		return time.Hour
	case refreshOnDemand:
		return 0
	}
	return 10 * time.Minute
}

// propRefreshLabel is the choice as the Settings say it, the refresh-now key
// named as bound.
func (d Dashboard) propRefreshLabel() string {
	switch d.mapPropRefresh {
	case refreshHourly:
		return "Hourly"
	case refreshOnDemand:
		return "On demand (" + d.firstKey(actMapRefresh) + ")"
	}
	return "Every 10 min"
}

// propRefreshNote is a choice's measured cost on the wire while the mode is
// open (D-120, D-141), said under the row while it is chosen.
func propRefreshNote(r propRefresh) string {
	switch r {
	case refreshHourly:
		return "up to 3.3 MB an hour while the Propagation mode is open"
	case refreshOnDemand:
		return "about 2.6 MB each time it is refreshed"
	}
	return "up to 19 MB an hour while the Propagation mode is open"
}

// cyclePropRefresh moves the refresh picker, and round again; an open mode
// sets its next tick by the new choice (armPropRefresh).
func (d Dashboard) cyclePropRefresh(forward bool) Dashboard {
	d.mapPropRefresh = cycleIn(propRefreshChoices, itself, d.mapPropRefresh, forward)
	return d.uiTouched()
}

// noReadingsKey is the file's word for the no-readings correction (D-105):
// empty for GloTEC as is, the default, or "corrected".
func noReadingsKey(corrected bool) string {
	if corrected {
		return noReadingsCorrected
	}
	return ""
}

// noReadingsCorrected is the file's word for the correction on.
const noReadingsCorrected = "corrected"

// noReadingsLabel is the row's words (D-105): what is drawn while no station
// readings are held.
func noReadingsLabel(corrected bool) string {
	if corrected {
		return "Typical offset"
	}
	return "GloTEC as is"
}

// togglePropCorrect switches the no-readings correction; an open mode asks
// for its snapshot with it, which go-ionomaps answers without a request
// (armPropOptions).
func (d Dashboard) togglePropCorrect() Dashboard {
	d.mapPropCorrect = !d.mapPropCorrect
	return d.uiTouched()
}

// noStationsKey is the file's word for no GIRO (D-214); empty asks GIRO, the
// default.
const noStationsKey = "none"

// stationsKey is the file's word for the stations choice.
func stationsKey(noGIRO bool) string {
	if noGIRO {
		return noStationsKey
	}
	return ""
}

// stationsLabel is the row's words (D-214): GIRO's stations asked, or none.
func stationsLabel(noGIRO bool) string {
	if noGIRO {
		return "None (NOAA alone)"
	}
	return "GIRO"
}

// stationsNote is what the choice means, said beside the row while it has
// the focus: GIRO's terms when GIRO is asked (D-214, FR-4.1), what the field
// is made of when it is not.
func stationsNote(noGIRO bool) string {
	if noGIRO {
		return "GIRO is not asked; the field is NOAA's GloTEC and the climatology alone"
	}
	return "GIRO's readings are offered for educational and non-commercial research use only"
}

// togglePropStations switches GIRO's stations; an open mode asks for its
// snapshot with the choice, which go-ionomaps answers without a request
// (armPropOptions).
func (d Dashboard) togglePropStations() Dashboard {
	d.mapPropNoGIRO = !d.mapPropNoGIRO
	return d.uiTouched()
}
