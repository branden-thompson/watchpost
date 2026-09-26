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
	"math"
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
	var now []string
	if h.Temp != nil {
		temp, unit := *h.Temp, "°C"
		if d.units == render.UnitF {
			temp, unit = temp*9/5+32, "°F"
		}
		now = append(now, strconv.Itoa(int(math.Round(temp)))+unit)
	}
	if h.Condition != "" {
		now = append(now, strings.ReplaceAll(strings.ToLower(h.Condition), "_", " "))
	}
	if len(now) == 0 {
		return loc.Label + "."
	}
	return loc.Label + " - Currently: " + strings.Join(now, ", ") + "." // D-74: the place, then how it is now
}

// alertSentence is one alert against the place.
func (d Dashboard) alertSentence(loc snapshot.Location, pa tuimaps.PlaceAlert) string {
	own := d.alertOnRecord(loc, pa.Feature)
	event := pa.Label
	if own != nil && own.Event != "" {
		event = own.Event
	}
	var areas string
	if own != nil {
		areas = areasInWords(own.AreaDesc)
	}
	// D-74: WHERE IT IS IN EFFECT, IN THE WORDS A LISTENER KNOWS - this area,
	// or the Weather Service's own names for the areas, nearby or not - and
	// never a distance and a bearing. The relation underneath is M1's.
	var s string
	switch Relation(pa, d.mapPane.inMissing[pa.Feature]) {
	case "covers-missing":
		s = event + " in effect for this area; its outline could not be drawn"
	case "covers":
		s = event + " in effect for this area"
	case "stops short":
		if areas == "" {
			areas = "areas"
		}
		s = event + " in effect for nearby " + areas
	default:
		if areas == "" {
			areas = "the area it covers"
		}
		s = event + " in effect for " + areas
	}
	if own != nil && !own.Expires.IsZero() {
		s += " until " + d.untilWords(loc, own.Expires)
	}
	return s + "."
}

// alertOnRecord is the alert as watchpost holds it, by id: among the selected
// place's, the station's other places', or the national and in-view ones the
// feed drew - its name, its areas and its end time.
func (d Dashboard) alertOnRecord(loc snapshot.Location, id string) *snapshot.Alert {
	if a := alertByID(loc, id); a != nil {
		return a
	}
	if d.snap != nil {
		for _, l := range d.snap.Locations {
			if a := alertByID(l, id); a != nil {
				return a
			}
		}
	}
	return alertByID(snapshot.Location{Alerts: d.mapPane.national}, id)
}

// areaWords are the generic words of the Weather Service's area names, said
// in lower case: "San Diego County Coastal Areas" reads "San Diego County
// coastal areas".
var areaWords = map[string]bool{"Coastal": true, "Areas": true, "Area": true, "Beaches": true, "Waters": true,
	"Valleys": true, "Valley": true, "Mountains": true, "Inland": true, "Foothills": true, "Deserts": true, "Including": true}

// areasInWords is an alert's area description as the description says it
// (D-74): its first two places, joined by "and", their generic words in
// lower case.
func areasInWords(desc string) string {
	var places []string
	for _, p := range strings.Split(desc, ";") {
		if p = strings.TrimSpace(p); p != "" {
			places = append(places, p)
		}
		if len(places) == 2 {
			break
		}
	}
	for i, p := range places {
		words := strings.Fields(p)
		for j, w := range words {
			if areaWords[w] {
				words[j] = strings.ToLower(w)
			}
		}
		places[i] = strings.Join(words, " ")
	}
	return strings.Join(places, " and ")
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
