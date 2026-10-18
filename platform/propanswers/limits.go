package propanswers

// limits.go — an answer's limits, its best frequency, and how a frequency
// held fares against them (D-192, D-193, D-197). Bands are read by the band
// each answer names (go-ionomaps `BandStatus.Band`), never by their place in
// a list (BUILD red team L-F8).

import (
	"strconv"
	"strings"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// Limits are the limits an answer's bands name (R-2.6): the upper limit,
// the LUF and whether the circuit is too weak there rather than absorbed
// (D-70), and D-RAP's 1 dB frequency; zero where none decides.
type Limits struct {
	Upper, LUF, DRAP float64
	Weak             bool
}

// LimitsOf reads an answer's limits from its bands.
func LimitsOf(bands []ionomaps.BandStatus) Limits {
	if len(bands) == 0 {
		return Limits{} // no answer: no limit decides
	}
	var l Limits
	for _, b := range bands {
		switch b.Limit {
		case ionomaps.LimitMUF, ionomaps.LimitFoF2:
			if l.Upper == 0 {
				l.Upper = float64(b.LimitMHz)
			}
		case ionomaps.LimitAbsorption, ionomaps.LimitCircuit:
			l.LUF, l.Weak = float64(b.LimitMHz), l.Weak || b.Limit == ionomaps.LimitCircuit
		case ionomaps.LimitDRAP:
			l.DRAP = float64(b.LimitMHz)
		}
	}
	return l
}

// Usable are the bands an answer can use: open or disturbed, by name.
func Usable(bands []ionomaps.BandStatus) []string {
	if len(bands) == 0 {
		return nil
	}
	var out []string
	for _, b := range bands {
		if b.Status == ionomaps.Open || b.Status == ionomaps.Disturbed {
			out = append(out, b.Band.String())
		}
	}
	return out
}

// BestFactor is the share of the upper limit the best frequency is taken at
// (D-197): ITU-R P.373's FOT, commonly about 85 % of the MUF on F2 paths - a
// rule of thumb, the MUF being the hour's estimate.
const BestFactor = 0.85

// Best is an answer's best frequency (D-197): 85 % of its upper limit, and
// the band whose lower edge fits under it and above the LUF; false where
// none does - no frequency serves - or no upper limit is known (zero mhz).
func Best(l Limits) (mhz float64, band string, ok bool) {
	if l.Upper <= 0 {
		return 0, "", false
	}
	if !(l.LUF >= 0) {
		return 0, "", false // a LUF negative or not a number decides nothing
	}
	mhz = BestFactor * l.Upper
	for b := ionomaps.Band10m; b >= ionomaps.Band160m; b-- { // ten bands, the highest first
		if low := b.LowMHz(); low <= mhz && low >= l.LUF {
			return mhz, b.String(), true
		}
	}
	return mhz, "", false
}

// BestWords say the best frequency (D-197): long for the centre box, "best
// ~15.0 MHz (20m), 85% of the MUF"; short for a chart row, "best ~15.0
// (20m)"; where none serves, so; empty with no upper limit.
func BestWords(l Limits, long bool) string {
	if !(l.Upper > 0) {
		return "" // no upper limit known: no best frequency to say
	}
	mhz, band, ok := Best(l)
	switch {
	case mhz == 0:
		return ""
	case !ok && long:
		return "no frequency serves on this path"
	case !ok:
		return "none serves"
	case long:
		return "best ~" + MHz(mhz) + " MHz (" + band + "), 85% of the MUF"
	}
	return "best ~" + MHz(mhz) + " (" + band + ")"
}

// LimitWords name an answer's limits (D-192): its upper limit, from a band
// it decided; the LUF when the lower limit closes any band; and D-RAP's 1 dB
// frequency when it disturbs any; each figure in MHz with the unit given
// after it (the chart's rows write none, D-198); empty with no data.
func LimitWords(bands []ionomaps.BandStatus, unit string) string {
	if len(bands) == 0 {
		return "" // no data: no limit to name
	}
	l := LimitsOf(bands)
	upper := "MUF"
	for _, b := range bands {
		if b.Limit == ionomaps.LimitFoF2 {
			upper = "foF2"
		}
	}
	var parts []string
	for _, w := range []struct {
		name string
		mhz  float64
	}{{upper, l.Upper}, {"LUF", l.LUF}, {"D-RAP", l.DRAP}} {
		if w.mhz > 0 {
			parts = append(parts, w.name+" "+MHz(w.mhz)+unit)
		}
	}
	return strings.Join(parts, " · ")
}

// RowWords are a chart row's limits and its best frequency, in MHz without
// the unit on each (D-197, D-198).
func RowWords(bands []ionomaps.BandStatus) string {
	if len(bands) == 0 {
		return ""
	}
	limits, best := LimitWords(bands, ""), BestWords(LimitsOf(bands), false)
	if limits == "" || best == "" {
		return limits + best
	}
	return limits + " · " + best
}

// FreqVerdict is a held frequency on a path (D-193): with the LUF at or
// over the MUF, no frequency serves - too weak at 100 W; above the MUF, come
// down under it; under the LUF, go up to it (absorbed, or too weak); under
// D-RAP's 1 dB frequency, disturbed; else open.
func FreqVerdict(f float64, label string, l Limits) string {
	if !(f > 0) {
		return label + ": not a frequency" // nothing held to judge, or not a number
	}
	held := label + ": "
	switch {
	case l.Upper > 0 && l.LUF >= l.Upper:
		return held + "too weak at 100 W on this path"
	case l.Upper > 0 && f >= l.Upper:
		return held + "above the MUF - come down under " + MHz(l.Upper)
	case l.LUF > 0 && f < l.LUF && l.Weak:
		return held + "too weak at 100 W - go up to " + MHz(l.LUF) + "+"
	case l.LUF > 0 && f < l.LUF:
		return held + "absorbed - go up to " + MHz(l.LUF) + "+"
	case l.DRAP > 0 && f < l.DRAP:
		return held + "disturbed (D-RAP to " + MHz(l.DRAP) + ")"
	}
	return held + "open"
}

// MHz is a frequency as the windows say it: to a tenth of a megahertz.
func MHz(v float64) string {
	if v != v {
		return "?" // never "NaN" in words a listener hears
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// HighestOpen is the highest band an answer calls open, by name; "none", or
// "no data" where it has none.
func HighestOpen(bands []ionomaps.BandStatus) string {
	if len(bands) == 0 {
		return "no data"
	}
	best, data := "none", false
	for _, b := range bands {
		data = data || b.Status != ionomaps.StatusNoData
		if b.Status == ionomaps.Open {
			best = b.Band.String()
		}
	}
	if !data {
		return "no data"
	}
	return best
}

// OpenAt reports whether a moment falls in one of the spans.
func OpenAt(spans []ionomaps.Span, at time.Time) bool {
	if len(spans) == 0 {
		return false // never typically open
	}
	for _, s := range spans {
		if !at.Before(s.From) && at.Before(s.To) {
			return true
		}
	}
	return false
}

// Differs reports whether any band's status now differs from its typical day
// at the hour (D-135); a band with no data now is no difference.
func Differs(bands []ionomaps.BandStatus, hour time.Time) bool {
	if len(bands) == 0 {
		return false // no answer differs from nothing
	}
	at := hour.UTC().Truncate(time.Hour)
	for _, b := range bands {
		if b.Status != ionomaps.StatusNoData && (b.Status == ionomaps.Open) != OpenAt(b.OpenHours, at) {
			return true
		}
	}
	return false
}
