package app

import (
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// marineFor is the maritime report's data, read the narrow way — the same shape
// as fireFor and seismicFor: ask each assembler for one location's merged
// coastal block rather than cloning a snapshot per cycle (REVIEW C2).
//
// An inland location returns a zero report, and MarineSegments then composes
// nothing. That is deliberate and different from "no data available": the sea
// is not missing for Denver, it is not applicable.
func (lp *livePipelines) marineFor(ref snapshot.LocationRef) synth.MarineReport {
	var asms []*snapshot.Assembler
	if lp.priority != nil {
		asms = append(asms, lp.priority.asm)
	}
	if lp.recent != nil {
		asms = append(asms, lp.recent.asm)
	}
	for _, asm := range asms {
		if m, tzName, lat, lon, ok := asm.MarineFor(ref); ok {
			return marineReportOf(m, tzName, lat, lon)
		}
	}
	return synth.MarineReport{}
}

// marineReportOf turns the merged block into the report the composer reads.
func marineReportOf(m *snapshot.Marine, tzName string, lat, lon float64) synth.MarineReport {
	if m == nil {
		return synth.MarineReport{}
	}
	r := synth.MarineReport{Known: true, State: *m, Lat: lat, Lon: lon}
	// Tide times are read in the LOCATION's zone, not the listener's
	// (MVS-D-21): a tide happens where the water is.
	if tzName != "" {
		if z, err := time.LoadLocation(tzName); err == nil {
			r.TZ = z
		}
	}
	return r
}
