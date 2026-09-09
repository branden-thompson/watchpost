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
		p := parts[id]
		if p.AsOf.After(out.AsOf) {
			out.AsOf = p.AsOf // the freshest answer from any fire feed
		}
		// AND WHICH HALF IT ANSWERED. Each fire provider stamps the half it
		// serves — HMS and FIRMS the hotspots, WFIGS the incidents — so a feed
		// that answered with NOTHING still records that it answered. Reading it
		// from the counts instead would make "nobody looked" and "nothing found"
		// the same value, which is the defect this exists to prevent.
		if p.HotspotsAsOf.After(out.HotspotsAsOf) {
			out.HotspotsAsOf = p.HotspotsAsOf
		}
		if p.IncidentsAsOf.After(out.IncidentsAsOf) {
			out.IncidentsAsOf = p.IncidentsAsOf
		}
	}
	for _, id := range ids {
		for _, h := range parts[id].Hotspots {
			k := spotKey{int(math.Round(h.Lat / 0.003)), int(math.Round(h.Lon / 0.003)), h.DetectedAt.UTC().Format("2006-01-02")}
			if i, ok := spots[k]; ok {
				if h.FRPOrMissing() > out.Hotspots[i].FRPOrMissing() {
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
	sort.SliceStable(out.Hotspots, func(i, j int) bool { return out.Hotspots[i].KmOrFar() < out.Hotspots[j].KmOrFar() })
	sort.SliceStable(out.Incidents, func(i, j int) bool { return out.Incidents[i].AcresOrMissing() > out.Incidents[j].AcresOrMissing() })
	return out
}
