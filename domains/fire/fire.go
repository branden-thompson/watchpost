// Package fire holds what the wildfire providers share (B5): the proximity
// rules (from `[fire]` in config; AI-3 defaults), the confidence scale, the
// distance test, and hotspot clustering. The providers — hms (keyless
// analyst-curated satellite detections), wfigs (named incidents), firms
// (NASA VIIRS detections, keyed) — each turn their feed into the snapshot's
// FireState per location; this package is the one place the rules live.
package fire

import (
	"math"
	"sort"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Rules are the proximity and strength thresholds every provider applies.
type Rules struct {
	RadiusKm         float64 // hotspots this close count
	IncidentRadiusKm float64 // named incidents this close are listed
	MinFRPMW         float64 // weaker detections are ignored (0 = keep unknown FRP)
	BoldFRPMW        float64 // this strong reads emphasized
	MinConfidence    string  // low | nominal | high
}

// DefaultRules are the AI-3 values (25 km ring, nominal+, FRP ≥ 5 MW, 50 MW bold).
func DefaultRules() Rules {
	return Rules{RadiusKm: 25, IncidentRadiusKm: 50, MinFRPMW: 5, BoldFRPMW: 50, MinConfidence: "nominal"}
}

// Valid reports whether the rules can be applied.
func (r Rules) Valid() error {
	return invariant.Check(r.RadiusKm > 0 && r.IncidentRadiusKm > 0 && confidenceRank(r.MinConfidence) >= 0,
		"fire rules: radii must be positive and min_confidence one of low | nominal | high")
}

// confidenceRank orders the scale; -1 for an unknown label. HMS points are
// analyst-curated — the most trusted of all — so they outrank "high" and
// always pass min_confidence (red-team B5 docs lens: a keyless user who
// tightens the knob must not lose every hotspot).
func confidenceRank(c string) int {
	switch c {
	case "low":
		return 0
	case "nominal":
		return 1
	case "high":
		return 2
	case "analyst":
		return 3
	}
	return -1
}

// Keep reports whether a detection passes the rules: confident enough and
// strong enough. An unknown FRP passes (the feed did not measure it — HMS
// GOES points often carry none).
func (r Rules) Keep(confidence string, frpMW *float64) bool {
	if confidenceRank(confidence) < confidenceRank(r.MinConfidence) {
		return false
	}
	return frpMW == nil || *frpMW >= r.MinFRPMW
}

// Bold reports whether a hotspot reads emphasized.
func (r Rules) Bold(h snapshot.Hotspot) bool { return h.FRPMW != nil && *h.FRPMW >= r.BoldFRPMW }

// Near returns the distance from ref when the point is inside radius km.
func Near(ref snapshot.LocationRef, lat, lon, radiusKm float64) (float64, bool) {
	km := geo.HaversineKM(ref.Lat, ref.Lon, lat, lon)
	return km, km <= radiusKm
}

// Cluster merges detections of the same fire seen by several satellites or
// passes (AI-3: ~375 m, the VIIRS pixel, same UTC day): the strongest FRP
// survives, the count of passes is kept in the returned counts. The result
// is sorted nearest first.
func Cluster(hs []snapshot.Hotspot) []snapshot.Hotspot {
	type key struct {
		lat, lon int
		day      string
	}
	best := map[key]int{}
	var out []snapshot.Hotspot
	for _, h := range hs {
		k := key{int(math.Round(h.Lat / 0.003)), int(math.Round(h.Lon / 0.003)), h.DetectedAt.UTC().Format("2006-01-02")}
		if i, ok := best[k]; ok {
			if frp(h) > frp(out[i]) || (frp(h) == frp(out[i]) && h.DetectedAt.After(out[i].DetectedAt)) {
				out[i] = h
			}
			continue
		}
		best[k] = len(out)
		out = append(out, h)
	}
	sort.SliceStable(out, func(i, j int) bool { return dist(out[i]) < dist(out[j]) })
	if len(out) > snapshot.MaxHotspots {
		out = out[:snapshot.MaxHotspots] // nearest first, so the cap keeps what matters
	}
	return out
}

func frp(h snapshot.Hotspot) float64 {
	if h.FRPMW == nil {
		return -1
	}
	return *h.FRPMW
}

func dist(h snapshot.Hotspot) float64 {
	if h.DistanceKm == nil {
		return math.MaxFloat64
	}
	return *h.DistanceKm
}

// Age is a hotspot's age at now, for the views ("2 h ago").
func Age(h snapshot.Hotspot, now time.Time) time.Duration { return now.Sub(h.DetectedAt) }

// Bounds is the bounding box radiusKm around a point, in degrees — the
// shape the FIRMS area API and the incident search take.
func Bounds(lat, lon, radiusKm float64) (west, south, east, north float64) {
	dLat := radiusKm / 111.0
	dLon := radiusKm / (111.0 * math.Max(0.1, math.Cos(lat*math.Pi/180)))
	return lon - dLon, lat - dLat, lon + dLon, lat + dLat
}
