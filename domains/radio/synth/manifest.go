package synth

// manifest.go — what a read CONTAINS, for the operator who has to decide
// whether to put it on the air (D-87).
//
// HUM LEAD, 2026-09-11: "What DOES MAKE SENSE for the operator is to get a
// SUMMARY of what the report contains *before* it goes on air … a table that
// catalogs which scripts are included in the reads, and for certain reports, how
// many items are in there."
//
// THE NAMES ARE THE SOURCES', NOT OURS, and that is the point of naming them at
// all: an operator deciding whether a report is worth airing is deciding whether
// they trust what is in it, and "NWS" and "USGS" are what they trust. The one
// exception is the fire report, which is assembled from several feeds and is
// genuinely Watchpost's own work — `FireReport.Sources` names them inside it.

import (
	"fmt"
	"strings"
	"time"
)

// The source names the console lists, from the v2 reference.
const (
	forecastSource = "NWS Weather Forecast"
	marineSource   = "NDBC Marine Report"
	fireSource     = "Watchpost Fire Report"
	seismicSource  = "USGS 7-Day Seismic Report"
)

// productSpan is the range a forecast covers: "09/11 - 09/17".
//
// SEVEN DAYS FROM THE DATA'S OWN DAY, not from now. A product fetched at
// midnight is still the day's forecast, and dating it from the clock would shift
// the span across midnight while the words stayed the same.
func productSpan(p Product, now time.Time) string {
	from := p.Issued
	if from.IsZero() {
		from = now
	}
	return from.Format("01/02") + " - " + from.AddDate(0, 0, forecastDays-1).Format("01/02")
}

// marineSpan is the day a marine report speaks for.
//
// A DAY, NOT A SPAN. The coastal forecast and the tides are today's; giving it a
// week's range would promise an outlook it does not carry.
func marineSpan(m MarineReport, now time.Time) string {
	at := m.State.ObservedAt
	if at.IsZero() {
		at = now
	}
	if m.TZ != nil {
		at = at.In(m.TZ)
	}
	return at.Format("01/02/2006")
}

// fireCounts is how much fire there is: "3 Hotspots / 10 incidents".
//
// EACH HALF IS REPORTED ONLY IF ITS OWN FEED ANSWERED, which is the rule
// `FireReport` already states — "a zero count is only a fact when its own feed
// did". A manifest that read "0 Hotspots" because a provider was down would tell
// the operator the opposite of the truth, on the one line they use to decide.
func fireCounts(f FireReport) string {
	var parts []string
	if f.HotspotsKnown {
		parts = append(parts, fmt.Sprintf("%d %s", len(f.State.Hotspots), countNoun(len(f.State.Hotspots), "Hotspot")))
	}
	if f.IncidentsKnown {
		parts = append(parts, fmt.Sprintf("%d %s", len(f.State.Incidents), countNoun(len(f.State.Incidents), "incident")))
	}
	return strings.Join(parts, " / ")
}

// seismicCounts is how many quakes the week holds: "5 Quakes".
func seismicCounts(s SeismicReport) string {
	if !s.Known {
		return ""
	}
	return fmt.Sprintf("%d %s", len(s.State.Quakes), countNoun(len(s.State.Quakes), "Quake"))
}

// countNoun is the noun for a count, and it exists because "1 Hotspots" on a
// safety surface reads as a rendering fault rather than as a number.
func countNoun(n int, noun string) string {
	if n == 1 {
		return noun
	}
	return noun + "s"
}
