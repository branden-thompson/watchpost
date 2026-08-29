package synth

import (
	"fmt"
	"math"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// SeismicReport is what the broadcast's seismic segment reads from (P4, HUM
// LEAD script): the location's recent quakes and the point they are measured
// from. Known is false until the USGS feed has answered.
type SeismicReport struct {
	Known    bool
	State    snapshot.SeismicState
	Lat, Lon float64 // the location (bearings are precomputed on each Quake)
}

// seismicPause separates the seismic notice from the counts (script).
const seismicPause = 2 * time.Second

// spokenQuakeCap is how many quakes the broadcast reads — the strongest three
// (HUM LEAD); the rest are summarised and left to the location details view.
const spokenQuakeCap = 3

// SeismicSegments composes the Seismic Activity report: the notice (source,
// delay/safety), a two-second pause, the count, the strongest few quakes
// read largest-first, a summary of any beyond the cap, and the where-to-learn
// -more tail. With no quakes for the location the report is skipped entirely
// (HUM LEAD: the report plays only if there are seismic entries).
func (c Composer) SeismicSegments(location string, sr SeismicReport, imperial bool, now time.Time) []Segment {
	if !sr.Known || len(sr.State.Quakes) == 0 {
		return nil
	}
	place := ExpandStates(location)
	notice := c.say("seismic-report", "head", map[string]string{"Location": place})
	segs := []Segment{{Key: "seismic:notice:" + contentKey(notice), Text: notice, Pause: seismicPause}} // content-keyed: the cache must never replay a stale feed (REVIEW C1)

	n := len(sr.State.Quakes)
	body := []string{c.say("seismic-report", "count", map[string]any{"Count": n})}
	shown := min(n, spokenQuakeCap)
	for _, q := range sr.State.Quakes[:shown] { // already largest-first (the provider sorts)
		body = append(body, c.quakeSentence(q, imperial, now))
	}
	if rest := n - spokenQuakeCap; rest > 0 {
		noun := "quakes"
		if rest == 1 {
			noun = "quake"
		}
		body = append(body, c.say("seismic-report", "more", map[string]any{"Rest": rest, "Noun": noun, "Location": place}))
	}
	body = append(body, c.say("seismic-report", "link", nil)) // no trailing slash in the script: it would read an extra "slash" (HUM LEAD UAT)
	for _, piece := range body {
		if piece == "" {
			continue // a phrase without a script is not spoken
		}
		segs = append(segs, Segment{Key: "seismic:" + contentKey(piece), Text: piece}) // content-keyed: counts change between cycles under repeat (REVIEW C1)
	}
	return segs
}

// quakeSentence: "A magnitude 5.1 earthquake, 88 miles north of your location,
// at a depth of 9 kilometers, recorded 3 days ago. A quake of this magnitude
// has a strong likelihood of being felt when it occurs." Fields the feed did
// not give are left out, never guessed.
func (c Composer) quakeSentence(q snapshot.Quake, imperial bool, now time.Time) string {
	data := map[string]string{"Mag": fmt.Sprintf("%.1f", q.Mag), "Where": "", "Depth": "", "Ago": "", "Felt": c.say("seismic-report", "felt", map[string]string{"Likelihood": feltLikelihood(q.Mag)})}
	if q.DistanceKm > 0 && q.Bearing != "" {
		data["Where"] = distanceWords(q.DistanceKm, imperial) + " " + bearingLong(q.Bearing)
	}
	if q.DepthKm > 0 { // depth is always in kilometres (the seismology convention), even under imperial (HUM LEAD)
		data["Depth"] = depthWords(q.DepthKm)
	}
	if !q.At.IsZero() {
		data["Ago"] = durationWords(now.Sub(q.At))
	}
	return c.say("seismic-report", "quake", data)
}

// feltLikelihood is the magnitude's general felt-likelihood — keyed to the
// felt bands the detail section labels (not the glyph energy ramp, which tops
// out at "significant" ≥ 5.0): low below 3.5 ("below feeling"), moderate 3.5–4.5
// ("might feel it"), strong at 4.5 and up ("almost certainly felt" and
// "significant"). This keeps the spoken likelihood consistent with the screen's
// label for the same quake. It speaks of the magnitude in general ("when it
// occurs"), not of whether this event was felt at this distance.
func feltLikelihood(mag float64) string {
	switch {
	case mag >= 4.5:
		return "strong"
	case mag >= 3.5:
		return "moderate"
	default:
		return "low"
	}
}

// depthWords: "9 kilometers" / "1 kilometer" (whole units).
func depthWords(km float64) string {
	k := int(math.Round(km))
	if k == 1 {
		return "1 kilometer"
	}
	return fmt.Sprintf("%d kilometers", k)
}

// bearingLong turns a 16-point compass abbreviation (the form each Quake
// carries) into its spoken word ("NNW" → "north-northwest"). The table is a
// local (the codebase keeps compass tables per-call, not as globals — P10-06);
// this runs once per quake while composing the broadcast (cold path).
func bearingLong(short string) string {
	return map[string]string{
		"N": "north", "NNE": "north-northeast", "NE": "northeast", "ENE": "east-northeast",
		"E": "east", "ESE": "east-southeast", "SE": "southeast", "SSE": "south-southeast",
		"S": "south", "SSW": "south-southwest", "SW": "southwest", "WSW": "west-southwest",
		"W": "west", "WNW": "west-northwest", "NW": "northwest", "NNW": "north-northwest",
	}[short]
}
