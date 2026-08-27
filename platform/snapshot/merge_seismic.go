package snapshot

import "maps"

// merge_seismic.go — seismic has one provider (USGS), so there is no
// cross-provider merge like fire's; the assembler keeps the latest state per
// location and this deep-copies it into the published snapshot so nothing
// aliases assembler state (the Snapshot isolation contract).

// cloneSeismic returns a deep copy of a SeismicState so the published value
// shares no backing memory with the assembler: the Quakes slice, each quake's
// Felt pointer, and the reference fields inside its Source (DistanceKm pointer,
// FillFrom map) are all copied. stateFor currently leaves the Source reference
// fields nil, but the copy is complete regardless, so populating them later
// cannot silently alias assembler state (REVIEW P5 F2).
func cloneSeismic(ss *SeismicState) *SeismicState {
	out := &SeismicState{AsOf: ss.AsOf, Quakes: append([]Quake(nil), ss.Quakes...)}
	for i := range out.Quakes {
		if f := out.Quakes[i].Felt; f != nil {
			v := *f
			out.Quakes[i].Felt = &v
		}
		out.Quakes[i].Source = cloneSource(out.Quakes[i].Source)
	}
	return out
}

// cloneSource deep-copies the reference fields of a SourceInfo (value fields
// ride along in the struct copy).
func cloneSource(s SourceInfo) SourceInfo {
	if s.DistanceKm != nil {
		v := *s.DistanceKm
		s.DistanceKm = &v
	}
	if s.FillFrom != nil {
		m := make(map[string]string, len(s.FillFrom))
		maps.Copy(m, s.FillFrom)
		s.FillFrom = m
	}
	return s
}
