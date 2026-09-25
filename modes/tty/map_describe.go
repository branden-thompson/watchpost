package tty

// map_describe.go — the map's text description (0.18.0 W1.4, FR-7.4; W9.2
// folded by D-60: it is built on the library's Report, not Describe).
//
// IT IS WRITTEN IN THE WORDS M1 ASKS ABOUT. For every alert on the map the
// description says whether it covers the selected place, stops short of it or
// lies to one side, how far its nearest edge is and which way, how severe it
// is and until when - so a listener who cannot see the picture answers the
// same question from the text alone (M1b). It names the place and never says
// "you" (D-29), and writes units and directions as words, because the voice
// will read it (FR-7.3).

import (
	"strconv"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Relation is M1's word for an alert against a place: it covers the place,
// stops short of it (its edge within the nearby distance), or lies to one
// side. An alert whose missing zone holds the place covers it by that zone,
// and the words say the zone could not be drawn.
func Relation(pa tuimaps.PlaceAlert, inMissing bool) string {
	switch {
	case inMissing:
		return "covers-missing"
	case pa.Where == tuimaps.Inside:
		return "covers"
	case pa.Where == tuimaps.Nearby:
		return "stops short"
	}
	return "lies to one side"
}

// describeLines is the description of the selected place: its conditions,
// then each alert on the map, joined to watchpost's own alert for its times.
func (d Dashboard) describeLines() []string {
	loc := d.selectedLocation()
	if loc == nil {
		return nil
	}
	out := []string{d.placeFacts(*loc)}
	if !d.layerOn(AlertLayer) {
		return append(out, alertLayerOffText) // off is said, never "nothing is there"
	}
	alerts := d.mapPane.report.Alerts
	if len(alerts) == 0 {
		return append(out, "No alert on the map covers or comes near "+loc.Label+".")
	}
	for _, pa := range alerts {
		out = append(out, d.alertSentence(*loc, pa))
	}
	return out
}

// placeFacts is the place's own weather, in words.
func (d Dashboard) placeFacts(loc snapshot.Location) string {
	h := loc.Harmonized
	facts := loc.Label
	if h.Temp != nil {
		temp, unit := *h.Temp, "Celsius"
		if d.units == render.UnitF {
			temp, unit = temp*9/5+32, "Fahrenheit"
		}
		facts += ": " + strconv.Itoa(int(temp+0.5)) + " degrees " + unit
		if h.Condition != "" {
			facts += " and " + strings.ToLower(h.Condition)
		}
	}
	return facts + "."
}

// alertSentence is one alert against the place.
func (d Dashboard) alertSentence(loc snapshot.Location, pa tuimaps.PlaceAlert) string {
	own := alertByID(loc, pa.Feature)
	if own == nil {
		own = alertByID(snapshot.Location{Alerts: d.mapPane.national}, pa.Feature) // a national event (W5.3)
	}
	event := pa.Label
	if own != nil && own.Event != "" {
		event = own.Event
	}
	s := event
	if word := strings.ToLower(pa.Severity.Word()); word != "" && word != "unknown" {
		s += ", " + word
	}
	switch Relation(pa, d.mapPane.inMissing[pa.Feature]) {
	case "covers-missing":
		s += ", covers " + loc.Label + " by a zone that could not be drawn."
	case "covers":
		s += ", covers " + loc.Label + "; its nearest edge is " + distanceWords(pa) + "."
	case "stops short":
		s += ", stops short of " + loc.Label + "; its nearest edge is " + distanceWords(pa) + "."
	default:
		s += ", lies to one side of " + loc.Label + "; its nearest edge is " + distanceWords(pa) + "."
	}
	if own != nil && !own.Expires.IsZero() {
		s += " It is in effect until " + d.untilWords(loc, own.Expires) + "."
	}
	return s
}

// distanceWords is how far and which way, in words: "7 kilometres to the
// north-east", with one decimal under ten.
func distanceWords(pa tuimaps.PlaceAlert) string {
	n := strconv.FormatFloat(pa.Distance, 'f', 0, 64)
	if pa.Distance < 10 {
		n = strconv.FormatFloat(pa.Distance, 'f', 1, 64)
	}
	s := n + " " + pa.Unit
	if pa.Compass != "" {
		s += " to the " + pa.Compass
	}
	return s
}

// untilWords is a time as the place keeps it, with its day.
func (d Dashboard) untilWords(loc snapshot.Location, t time.Time) string {
	if zone, err := time.LoadLocation(loc.TZ); err == nil && loc.TZ != "" {
		t = t.In(zone)
	}
	return d.clockFmt.Time(t) + " on " + t.Weekday().String()
}

// alertByID is the place's own alert with an id, if it has it.
func alertByID(loc snapshot.Location, id string) *snapshot.Alert {
	for i := range loc.Alerts {
		if loc.Alerts[i].ID == id {
			return &loc.Alerts[i]
		}
	}
	return nil
}
