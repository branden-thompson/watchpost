package snapshot

// merge_fire.go — the fire merge: several providers' FireState into one per location. Split from assembler.go by the quality pass (Q2, pure move).

import (
	"math"
	"sort"
)

// mergeFire folds every provider's fire contribution into one FireState
// (B5): hotspots deduped across feeds (the same fire seen by HMS and FIRMS
// — ~300 m, same UTC day — keeps the strongest reading) and sorted nearest
// first; incidents deduped by name, largest first. Providers merge in name
// order so the result is stable.
func mergeFire(parts map[string]*FireState) FireState {
	ids := make([]string, 0, len(parts))
	for id := range parts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	type spotKey struct {
		lat, lon int
		day      string
	}
	spots := map[spotKey]int{}
	names := map[string]int{}
	out := FireState{Hotspots: []Hotspot{}, Incidents: []Incident{}}
	for _, id := range ids {
		if t := parts[id].AsOf; t.After(out.AsOf) {
			out.AsOf = t // the freshest answer from any fire feed
		}
	}
	for _, id := range ids {
		for _, h := range parts[id].Hotspots {
			k := spotKey{int(math.Round(h.Lat / 0.003)), int(math.Round(h.Lon / 0.003)), h.DetectedAt.UTC().Format("2006-01-02")}
			if i, ok := spots[k]; ok {
				if frpOf(h) > frpOf(out.Hotspots[i]) {
					out.Hotspots[i] = h
				}
				continue
			}
			spots[k] = len(out.Hotspots)
			out.Hotspots = append(out.Hotspots, h)
		}
		for _, in := range parts[id].Incidents {
			if i, ok := names[in.Name]; ok {
				if in.Source.IssuedAt.After(out.Incidents[i].Source.IssuedAt) {
					out.Incidents[i] = in
				}
				continue
			}
			names[in.Name] = len(out.Incidents)
			out.Incidents = append(out.Incidents, in)
		}
	}
	sort.SliceStable(out.Hotspots, func(i, j int) bool { return kmOf(out.Hotspots[i]) < kmOf(out.Hotspots[j]) })
	sort.SliceStable(out.Incidents, func(i, j int) bool { return acresOf(out.Incidents[i]) > acresOf(out.Incidents[j]) })
	return out
}

func frpOf(h Hotspot) float64 {
	if h.FRPMW == nil {
		return -1
	}
	return *h.FRPMW
}

func kmOf(h Hotspot) float64 {
	if h.DistanceKm == nil {
		return math.MaxFloat64
	}
	return *h.DistanceKm
}

func acresOf(in Incident) float64 {
	if in.Acres == nil {
		return -1
	}
	return *in.Acres
}
