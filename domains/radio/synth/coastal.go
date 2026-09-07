package synth

import "strings"

// The coastal-waters forecast, for the maritime report (MVS-D-14).
//
// **Ruled (B) only for 0.14.0 (MVS-D-38, E-8).** The zone is the FIRST NEARSHORE
// BLOCK after the office's synopsis — no geometry. Option (A), the nearest zone
// by polygon centroid, is not built: MVS-D-14 ruled the forecast came "for
// free" through the existing products endpoint, and (A) is a new NWS endpoint
// with a fan-out to cap and a call budget to hold. It returns in a later
// release only if UAT shows a listener being read the wrong stretch of water.
//
// So there is deliberately no MarineZoneFor, no centroid memo and no fan-out
// cap in this release. If that changes, the seam is here.

// CoastalForecastType is the NWS product type carrying the Coastal Waters
// Forecast. The office issues one per marine area.
const CoastalForecastType = "CWF"

// SpokenPeriodsCap is how many forecast periods the broadcast reads (RAT-6,
// ratified MVS-D-37).
//
// A CWF runs five days. Read whole it would be the longest thing in the cycle
// by several minutes, and a listener wants tonight and tomorrow — the rest is
// on the screen.
const SpokenPeriodsCap = 3

// CoastalForecast is the coastal-waters text this location should hear: the
// office's synopsis plus the first nearshore block, cut to SpokenPeriodsCap
// periods. Empty when the office issued no CWF (an inland office, or an
// outage) — the maritime report then reads the buoy and the tides alone.
func CoastalForecast(products []Product, zone string) string {
	raw := ""
	for _, p := range products {
		if strings.EqualFold(p.Type, CoastalForecastType) {
			raw = p.Text
			break
		}
	}
	if raw == "" {
		return ""
	}
	// The period cut happens on the RAW text, before normalisation rewrites
	// the period tags: Normalize turns ".TONIGHT..." into prose, and the tags
	// are what marks a period boundary.
	return cutPeriods(nearshoreBlock(raw, zone), SpokenPeriodsCap)
}

// nearshoreBlock is the synopsis plus the first UGC block of the product.
//
// The listener's own marine zone, if the products endpoint happens to carry it,
// wins; otherwise the FIRST block after the synopsis is read. That is (B): a
// coastal office's first nearshore zone is adjacent to the one a listener sits
// on far more often than not, and being one zone out is a far smaller error
// than reading five days of every zone the office covers.
func nearshoreBlock(raw, zone string) string {
	if zone != "" {
		if only := FilterUGC(raw, zone, ""); strings.TrimSpace(only) != "" && only != raw {
			return only
		}
	}
	blocks := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "$$")
	var out []string
	for i, block := range blocks {
		pre, _, body, ok := splitUGC(block)
		if !ok {
			if i == 0 {
				out = append(out, block) // the synopsis, which carries no UGC line
			}
			continue
		}
		if i == 0 && strings.TrimSpace(pre) != "" {
			out = append(out, pre)
		}
		out = append(out, body)
		break // the FIRST nearshore block only
	}
	return strings.Join(out, "\n")
}

// cutPeriods keeps the first n period blocks. NWS period tags start a line with
// a dot and end with three dots (".TONIGHT...", ".WED...").
func cutPeriods(text string, n int) string {
	if n <= 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	var out []string
	seen := 0
	for _, line := range lines {
		if isPeriodTag(line) {
			seen++
			if seen > n {
				break
			}
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// isPeriodTag reports whether a line opens a forecast PERIOD.
//
// .SYNOPSIS... looks identical but is not a period — it is the office's
// overview of the whole area, and counting it would spend one of the three
// periods on something that is not a forecast at all.
func isPeriodTag(line string) bool {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, ".") || !strings.Contains(t, "...") {
		return false
	}
	return !strings.HasPrefix(strings.ToUpper(t), ".SYNOPSIS")
}
