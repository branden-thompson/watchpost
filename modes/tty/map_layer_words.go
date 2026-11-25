package tty

// map_layer_words.go — what each layer drawn says of the place (F-205 AX-4,
// D-363): after the alerts, one sentence a layer, in the Overlays menu's
// order, worded from the library's answer at the place - a field's value
// there, an image's class, or the nearest feature with its distance and way -
// and the radar's motion when the heavier rain moves. Only what the moment
// shown draws is said: the library answers for every overlay it holds, a
// Forecast mode's other days among them.

import (
	"github.com/branden-thompson/watchpost/platform/units"
	"math"
	"strconv"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// layerAnswerWords are the layers' sentences about a place in view.
func (mm mapModel) layerAnswerWords(loc snapshot.Location, e mapEnv) []string {
	if mm.mapMode(e) == modePropagation {
		return nil // the mode says its own (D-210)
	}
	best := map[string]tuimaps.Answer{}
	for _, a := range mm.mapPane.report.Answers {
		key, ok := mm.answerLayer(a, e)
		if !ok {
			continue
		}
		if cur, seen := best[key]; !seen || betterAnswer(a, cur) {
			best[key] = a
		}
	}
	var out []string
	for _, l := range e.Layers {
		a, ok := best[l.Key]
		if !ok {
			continue
		}
		out = append(out, mm.answerSentence(l.Label, loc.Label, a))
		if l.Key == RadarLayer {
			out = append(out, mm.motionWords(loc.Label, e)...)
		}
	}
	return out
}

// answerLayer is the layer an answer is about, when that layer is on and
// the moment shown draws the overlay: alerts are said on their own, and the
// radar's frames are the radar's only in Radar mode.
func (mm mapModel) answerLayer(a tuimaps.Answer, e mapEnv) (string, bool) {
	key, _, _ := strings.Cut(a.Overlay, "/")
	switch {
	case key == AlertLayer:
		return "", false
	case key == RadarLayer:
		return key, mm.mapMode(e) == modeRadar
	case !mm.layerOn(key, e):
		return "", false
	}
	o, held := mm.handedIn(a.Overlay)
	if !held {
		return "", false
	}
	from, to := mm.momentShown(e)
	return key, o.During.Meets(from, to)
}

// handedIn is an overlay as watchpost handed it to the map.
func (mm mapModel) handedIn(id string) (tuimaps.Overlay, bool) {
	for _, given := range []map[string]tuimaps.Overlay{mm.mapPane.given, mm.mapPane.tempGiven, mm.mapPane.radarGiven} {
		if o, ok := given[id]; ok {
			return o, true
		}
	}
	return tuimaps.Overlay{}, false
}

// momentShown is the moment the map draws: Forecast mode's step, else now.
func (mm mapModel) momentShown(e mapEnv) (time.Time, time.Time) {
	switch mm.mapMode(e) {
	case modeForecast:
		steps := mm.forecastSteps(e)
		sp := steps[min(mm.mapPane.fcStep, len(steps)-1)].Span
		return sp.From, sp.Until
	case modeRadar, modePropagation:
	}
	now := e.Now()
	return now, now
}

// betterAnswer prefers an answer with data, and of two, the nearer.
func betterAnswer(a, than tuimaps.Answer) bool {
	if a.NoData != than.NoData {
		return !a.NoData
	}
	return a.Km < than.Km
}

// The shapes an answer takes, in the library's own words for them (its
// Form): a point or a line is the nearest one's.
const (
	answerField = "field"
	answerImage = "image"
	answerArea  = "area"
)

// answerSentence is one layer's answer as a sentence, in the words of its
// shape.
func (mm mapModel) answerSentence(layer, place string, a tuimaps.Answer) string {
	switch a.Form {
	case answerField:
		return fieldWords(layer, place, a)
	case answerImage:
		return mm.imageWords(layer, place, a)
	case answerArea:
		return areaAnswerWords(layer, place, a)
	}
	return nearWords(layer, a)
}

// fieldWords: "Temperature at Oceanside, CA: 63 degrees Fahrenheit, rising
// to the north." A wind says where it blows from.
func fieldWords(layer, place string, a tuimaps.Answer) string {
	if a.NoData {
		return layer + ": no reading at " + place + "."
	}
	s := layer + " at " + place + ": " + strings.TrimSpace(roundWords(a.Value)+" "+a.ValueUnit)
	if a.From != "" {
		s += " from the " + a.From
	}
	if a.Rises != "" {
		s += ", higher to the " + a.Rises
	}
	return s + "."
}

// imageWords: the radar's class at the place as the legend words it, and
// the nearest heavier echo's distance and way.
func (mm mapModel) imageWords(layer, place string, a tuimaps.Answer) string {
	s := layer + ": no echo at " + place
	if words := mm.classWords(a.Overlay, a.Class); !a.NoData && a.Class > 0 && words != "" {
		s = layer + " at " + place + ": " + words
	}
	if a.Heavier != "" {
		s += "; the nearest heavier echo is " + distanceWords(a.HeavierAt, a.Unit) + " to the " + a.Heavier
	}
	return s + "."
}

// classWords are an overlay's class as its legend says it, with the unit.
func (mm mapModel) classWords(id string, class int) string {
	for _, e := range mm.mapPane.legend {
		if e.ID == id && class >= 0 && class < len(e.Classes) {
			return strings.TrimSpace(e.Classes[class].Label + " " + e.Unit)
		}
	}
	return ""
}

// areaAnswerWords: inside an area that is no alert, or its nearest edge.
func areaAnswerWords(layer, place string, a tuimaps.Answer) string {
	switch {
	case a.NoData:
		return layer + ": none in view of " + place + "."
	case a.Relation == tuimaps.Inside:
		return place + " lies inside the " + strings.ToLower(layer) + "."
	}
	return layer + ": the nearest edge is " + distanceWords(a.Distance, a.Unit) + " to the " + a.Compass + "."
}

// nearWords: the nearest point or line, named, with its distance and way.
func nearWords(layer string, a tuimaps.Answer) string {
	if a.NoData {
		return layer + ": none in view."
	}
	s := layer + ": the nearest"
	if a.Label != "" {
		s += ", " + a.Label + ","
	}
	return s + " is " + distanceWords(a.Distance, a.Unit) + " to the " + a.Compass + "."
}

// motionWords are the radar's observed motion near the place, when the
// heavier rain moves.
func (mm mapModel) motionWords(place string, e mapEnv) []string {
	var out []string
	for _, m := range mm.mapPane.motion {
		if !m.Moving || m.Missing != 0 {
			continue
		}
		s := "Heavier rain " + distanceWords(m.To.Distance, m.To.Unit) + " to the " + m.To.Compass + " of " + place +
			" moves towards the " + m.HeadingCompass + " at " + mm.speedWords(m.SpeedKmh, e)
		switch m.Trend {
		case tuimaps.Closer:
			s += ", coming closer"
		case tuimaps.Away:
			s += ", moving away"
		}
		out = append(out, s+".")
	}
	return out
}

// speedWords is a speed in the station's units.
func (mm mapModel) speedWords(kmh float64, e mapEnv) string {
	if e.Opts.Units == render.UnitF {
		return roundWords(units.MilesOf(kmh)) + " miles an hour" // km to miles, the same per hour
	}
	return roundWords(kmh) + " kilometres an hour"
}

// distanceWords is a distance rounded for speech, with its unit.
func distanceWords(v float64, unit string) string {
	if v < 1 {
		return "under one " + strings.TrimSuffix(unit, "s")
	}
	return strings.TrimSpace(roundWords(v) + " " + unit)
}

// roundWords is a number rounded to a whole, for speech.
func roundWords(v float64) string { return strconv.Itoa(int(math.Round(v))) }
