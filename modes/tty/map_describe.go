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
	"sort"
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

// describeLinesAll is the whole description: every alert in view. It is what
// the window shows in place of the picture, and what a screen reader reads.
func (d Dashboard) describeLinesAll() []string { return d.describeUpTo(0) }

// describeUpTo is the description of what is in view (D-78): while the
// selected place is in view, its conditions and the alerts that cover it or
// come near it, in M1's words; away from it, the view's name and nothing of
// the place - no information is better than information that does not match
// the map. Then every other alert in view, most severe first. With most over
// zero, at most that many alerts are said, and the last line says how many
// more there are.
func (d Dashboard) describeUpTo(most int) []string {
	loc := d.selectedLocation()
	if loc == nil {
		return nil
	}
	v := d.viewBox(d.mapBodySize())
	here := d.mapPane.m == nil || v.Contains(loc.Lat, loc.Lon)
	out := []string{d.viewHeading(*loc, here)}
	if !d.layerOn(AlertLayer) {
		return append(out, alertLayerOffText) // off is said, never "nothing is there"
	}
	var said []string
	told := map[string]bool{}
	if here {
		for _, pa := range d.mapPane.report.Alerts {
			if Relation(pa, d.mapPane.inMissing[pa.Feature]) == "lies to one side" {
				continue // said below with the view's others, if it is in view
			}
			said, told[pa.Feature] = append(said, d.alertSentence(*loc, pa)), true
		}
		if len(said) == 0 {
			said = append(said, "No alert on the map covers or comes near "+loc.Label+".")
		}
	}
	for _, a := range d.alertsInView(v) {
		if !told[a.id] {
			said = append(said, d.viewAlertSentence(*loc, a))
		}
	}
	if len(said) == 0 {
		said = append(said, "No alerts in view.")
	}
	if most > 0 && len(said) > most {
		said = append(said[:most-1], "And "+strconv.Itoa(len(said)-most+1)+" more in view.")
	}
	return append(out, said...)
}

// viewHeading is the description's first line: the place and how it is now
// while it is in view, and the view's name when it is not.
func (d Dashboard) viewHeading(loc snapshot.Location, here bool) string {
	if here {
		return d.placeFacts(loc)
	}
	name := d.viewName()
	return strings.ToUpper(name[:1]) + name[1:] + "."
}

// viewName is what the view is called: the namer's name for its centre and
// width, or the region it is in.
func (d Dashboard) viewName() string {
	size := d.mapBodySize()
	if d.cfg.MapAreaName != nil && d.mapPane.m != nil {
		centre, _ := d.mapPane.m.Centre()
		v := d.viewBox(size)
		widthKm := (v.E - v.W) * 111.32 * math.Cos(centre.Lat*math.Pi/180)
		if name := d.cfg.MapAreaName(centre, widthKm); name != "" {
			return name
		}
	}
	if d.mapPane.region.Name != "" {
		return d.mapPane.region.Name
	}
	return "the map"
}

// viewAlert is an alert drawn in view: its id, its name and severity as the
// map has them, and its end.
type viewAlert struct {
	id, label string
	sev       tuimaps.Severity
	expires   time.Time
}

// alertsInView is every alert area drawn that meets the view, once each,
// the most severe first.
func (d Dashboard) alertsInView(v MapView) []viewAlert {
	seen := map[string]bool{}
	var out []viewAlert
	for id, o := range d.mapPane.given {
		if !strings.HasPrefix(id, AlertLayer+"/") {
			continue
		}
		for _, f := range o.Features {
			if f.ID == "" || seen[f.ID] || !featureMeets(f, v) {
				continue
			}
			seen[f.ID] = true
			out = append(out, viewAlert{id: f.ID, label: f.Label, sev: f.Severity, expires: f.Expires})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].sev != out[j].sev {
			return out[i].sev > out[j].sev
		}
		if out[i].label != out[j].label {
			return out[i].label < out[j].label
		}
		return out[i].id < out[j].id
	})
	return out
}

// featureMeets reports whether a feature's outline's box meets the view.
func featureMeets(f tuimaps.Feature, v MapView) bool {
	w, s, e, n := 180.0, 90.0, -180.0, -90.0
	for _, ring := range f.Rings {
		for _, p := range ring {
			w, s, e, n = min(w, p.Lon), min(s, p.Lat), max(e, p.Lon), max(n, p.Lat)
		}
	}
	return w <= v.E && e >= v.W && s <= v.N && n >= v.S
}

// viewAlertSentence is an alert in view that neither covers the place nor
// comes near it: "<event> in effect for <areas> until <time>." (D-74).
func (d Dashboard) viewAlertSentence(loc snapshot.Location, a viewAlert) string {
	event, areas, until := a.label, "", a.expires
	if own := d.alertOnRecord(loc, a.id); own != nil {
		if own.Event != "" {
			event = own.Event
		}
		areas = areasInWords(own.AreaDesc)
		if !own.Expires.IsZero() {
			until = own.Expires
		}
	}
	if areas == "" {
		areas = "the area it covers"
	}
	s := event + " in effect for " + areas
	if !until.IsZero() {
		s += " until " + d.untilWords(loc, until)
	}
	return s + "."
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
// place's, the station's other places', or the in-view ones the feed drew - its name, its areas and its end time.
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
	return alertByID(snapshot.Location{Alerts: d.mapPane.inView}, id)
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
