package synth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// FireReport is what the broadcast's fire segment reads from (UAT 114,
// HUM LEAD script): the location's fire state, the rings it was built
// with, and the feeds that contributed. Known is false while no fire feed
// has answered yet.
type FireReport struct {
	Known            bool
	State            snapshot.FireState
	RadiusKm         float64  // the fire ring
	IncidentRadiusKm float64  // named incidents beyond the ring, up to here
	Sources          []string // spoken feed names, broadcast order
	Lat, Lon         float64  // the location, for bearings
}

// firePause separates the fire notice from the counts (UAT 114 script).
const firePause = 2 * time.Second

// FireSegments composes the Fire and Hotspot report: the notice (sources,
// delay/safety), a two-second pause, the count inside the fire ring, the
// named fires inside it, then the named fires beyond it worth noting.
// Phrases whose data the feeds did not give are left out, never guessed.
// With no fire data for the location the report is skipped entirely (HUM
// LEAD: straight to the tail like normal).
func (c Composer) FireSegments(location string, fr FireReport, imperial bool, now time.Time) []Segment {
	if !fr.Known {
		return nil
	}
	place := ExpandStates(location)
	notice := c.say("fire-report", "head", map[string]string{"Location": place, "Sources": joinAnd(fr.Sources)})
	segs := []Segment{{Key: "fire:notice:" + contentKey(notice), Text: notice, Pause: firePause}} // keyed by content: the cache must never replay yesterday's feeds (REVIEW C1)

	var body []string
	body = append(body, c.say("fire-report", "count", map[string]any{"Count": len(fr.State.Hotspots), "Ring": ringWords(fr.RadiusKm, imperial)})) // adjectival: "a 16 mile fire ring"
	if h := strongest(fr.State.Hotspots); h != nil {
		body = append(body, c.hotspotSentence(fr, *h, imperial, now))
	}
	var inside, outside []snapshot.Incident
	for _, in := range fr.State.Incidents {
		if in.Source.DistanceKm != nil && *in.Source.DistanceKm <= fr.RadiusKm {
			inside = append(inside, in)
		} else {
			outside = append(outside, in)
		}
	}
	for _, in := range inside {
		body = append(body, c.incidentSentence(fr, in, imperial, now, true))
	}
	if len(outside) > 0 {
		body = append(body, c.say("fire-report", "outside", nil))
		for _, in := range outside {
			body = append(body, c.incidentSentence(fr, in, imperial, now, false))
		}
	}
	for _, piece := range body {
		if piece == "" {
			continue // a phrase without a script is not spoken
		}
		segs = append(segs, Segment{Key: "fire:" + contentKey(piece), Text: piece}) // content-keyed: counts change between cycles under repeat (REVIEW C1)
	}
	return segs
}

// contentKey is a short stable digest of a sentence for the audio cache.
func contentKey(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:8])
}

// strongest is the hotspot with the highest fire radiative power (the
// nearest when none is measured); nil for none.
func strongest(hs []snapshot.Hotspot) *snapshot.Hotspot {
	if len(hs) == 0 {
		return nil
	}
	best := hs[0]
	for _, h := range hs[1:] {
		if h.FRPMW != nil && (best.FRPMW == nil || *h.FRPMW > *best.FRPMW) {
			best = h
		}
	}
	return &best
}

// hotspotSentence: "The strongest hotspot is 6 miles north of your
// location, with a fire radiative power of 62 megawatts, detected 2 hours
// ago by GOES-West."
func (c Composer) hotspotSentence(fr FireReport, h snapshot.Hotspot, imperial bool, now time.Time) string {
	data := map[string]string{"Where": "in your area", "FRP": "", "Detected": "", "Satellite": satelliteWords(h.Source.ModelOrStation)}
	if h.DistanceKm != nil {
		data["Where"] = distanceWords(*h.DistanceKm, imperial) + " " + bearingWords(fr.Lat, fr.Lon, h.Lat, h.Lon) + " of your location"
	}
	if h.FRPMW != nil {
		data["FRP"] = fmt.Sprintf("%.0f", *h.FRPMW)
	}
	if !h.DetectedAt.IsZero() {
		data["Detected"] = durationWords(now.Sub(h.DetectedAt))
	}
	return c.say("fire-report", "strongest", data)
}

// incidentSentence: "Timber is 12 miles east of your location, with a size
// of 12,915 acres, has been active for 3 days and 4 hours, and is 26
// percent contained." Outside the ring the sentence opens with the
// distance ("Timber, at a distance of 30 miles to the northeast, …"). The
// direction is said when the feed gave the incident's point (UAT 116).
func (c Composer) incidentSentence(fr FireReport, in snapshot.Incident, imperial bool, now time.Time, inside bool) string {
	data := map[string]any{"Name": ExpandStates(in.Name), "HasDistance": in.Source.DistanceKm != nil, "Inside": inside, "Distance": "", "Direction": "", "Facts": ""}
	if in.Lat != 0 || in.Lon != 0 {
		data["Direction"] = bearingWords(fr.Lat, fr.Lon, in.Lat, in.Lon)
	}
	if in.Source.DistanceKm != nil {
		data["Distance"] = distanceWords(*in.Source.DistanceKm, imperial)
	}
	var facts []string
	if in.Acres != nil {
		facts = append(facts, fmt.Sprintf("with a size of %s acres", render.Thousands(*in.Acres)))
	}
	if !in.Discovered.IsZero() {
		facts = append(facts, "has been active for "+durationWords(now.Sub(in.Discovered)))
	}
	if in.PercentContained != nil {
		facts = append(facts, fmt.Sprintf("is %.0f percent contained", *in.PercentContained))
	}
	data["Facts"] = joinAnd(facts)
	return c.say("fire-report", "incident", data)
}

// distanceWords: "15 miles" / "25 kilometers" (whole units; "1 mile").
func distanceWords(km float64, imperial bool) string {
	if imperial {
		mi := int(math.Round(km * 0.621371))
		if mi == 1 {
			return "1 mile"
		}
		return fmt.Sprintf("%d miles", mi)
	}
	k := int(math.Round(km))
	if k == 1 {
		return "1 kilometer"
	}
	return fmt.Sprintf("%d kilometers", k)
}

// ringWords is the ring's size as an adjective ("16 mile", "25 kilometer").
func ringWords(km float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%d mile", int(math.Round(km*0.621371)))
	}
	return fmt.Sprintf("%d kilometer", int(math.Round(km)))
}

// bearingWords names the direction from the location to a point in words
// ("north", "west-northwest") on the 16-point compass.
func bearingWords(lat1, lon1, lat2, lon2 float64) string {
	pts := []string{"north", "north-northeast", "northeast", "east-northeast", "east", "east-southeast", "southeast", "south-southeast",
		"south", "south-southwest", "southwest", "west-southwest", "west", "west-northwest", "northwest", "north-northwest"}
	return pts[geo.CompassIndex(geo.BearingDeg(lat1, lon1, lat2, lon2), 16)]
}

// durationWords: "5 hours and 35 minutes", "3 days and 4 hours", "12 minutes".
func durationWords(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	plural := func(n int, unit string) string {
		if n == 1 {
			return "1 " + unit
		}
		return fmt.Sprintf("%d %ss", n, unit)
	}
	switch {
	case days > 0 && hours > 0:
		return plural(days, "day") + " and " + plural(hours, "hour")
	case days > 0:
		return plural(days, "day")
	case hours > 0 && mins > 0:
		return plural(hours, "hour") + " and " + plural(mins, "minute")
	case hours > 0:
		return plural(hours, "hour")
	}
	return plural(mins, "minute")
}

// satelliteWords says a satellite the way the feeds name it ("GOES-West").
func satelliteWords(name string) string {
	name = strings.TrimSpace(name)
	switch strings.ToUpper(name) {
	case "GOES-EAST":
		return "GOES-East"
	case "GOES-WEST":
		return "GOES-West"
	}
	return name
}

// joinAnd joins a list for speech: "a", "a and b", "a, b, and c".
func joinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
}
