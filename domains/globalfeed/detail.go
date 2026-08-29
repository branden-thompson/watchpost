package globalfeed

// detail.go — per-class detail carried on an Event (0.13.0, SAM-D-21): what
// the severe-events window renders beyond the tape's thin fields. Every string
// from a feed is bounded here (P4 F5 / NFR-5): short fields to maxFieldRunes,
// prose to maxProseRunes, numbers to their physical ranges, lists to maxListLen.

import "time"

// maxProseRunes bounds an NWS description/instruction: real products run 1–3 KB;
// the cap only stops a hostile feed from growing the window without bound.
const maxProseRunes = 4000

// maxListLen bounds a feed-supplied list (zones, references, parameters).
const maxListLen = 50

// QuakeDetail is a USGS significant-feed event's record (the render list in
// 02-analysis/data-shape.md §4 plus the Keep fields).
type QuakeDetail struct {
	Mag       *float64 // nil when the feed omits it (absent ≠ 0.0)
	MagType   string   // "mww"
	DepthKm   float64  // geometry[2]
	Title     string   // "M 5.8 - 55 km NW of Kodāri, Nepal"
	Alert     string   // PAGER: green | yellow | orange | red | ""
	Sig       int      // significance
	Felt      int      // DYFI responses
	CDI, MMI  *float64 // community / modelled intensity, nil when absent
	Status    string   // "reviewed" | "automatic"
	Tsunami   bool
	UpdatedAt time.Time // properties.updated
	URL       string    // Keep, never rendered as a link (S6)
	DetailURL string    // Keep, never fetched or rendered as a link
}

// TropicalDetail is an NHC current-storm record.
type TropicalDetail struct {
	Name          string // "Dolly"
	Basin         string // "the Atlantic"
	BinNumber     string // "AT4"
	WindKt        int
	PressureMb    int
	MoveDirDeg    int
	MoveSpeedKt   int
	LatText       string // "15.0N"
	LonText       string // "46.9W"
	AdvisoryNum   string // publicAdvisory.advNum
	AdvisoryAt    time.Time
	ForecastNum   string // forecastAdvisory.advNum
	DiscussionNum string
	AdvisoryURL   string // Keep, never rendered as a link
}

// SevereDetail is a CAP alert record from the national NWS query (the
// location path carries the same fields on snapshot.Alert).
type SevereDetail struct {
	Headline      string
	Description   string // prose, ≤ maxProseRunes
	Instruction   string // prose, ≤ maxProseRunes
	Severity      string // CAP: Extreme | Severe | Moderate | Minor | Unknown
	Certainty     string
	Urgency       string
	MessageType   string // Alert | Update | Cancel
	Category      string
	Response      string
	SenderName    string
	Sender        string
	Effective     time.Time
	Sent          time.Time
	Expires       time.Time
	Ends          time.Time // zero when absent
	Onset         time.Time // zero when absent
	AffectedZones []string
	References    []string
	MaxWindGust   string // parameters allowlist (S7)
	MaxHailSize   string
	EventMotion   string
	NWSHeadline   string
	VTEC          string
}

// clampProse bounds prose to maxProseRunes (rune-safe).
func clampProse(s string) string {
	r := []rune(s)
	if len(r) <= maxProseRunes {
		return s
	}
	return string(r[:maxProseRunes])
}

// clampNonNeg bounds a count or physical quantity to [0, hi].
func clampNonNeg(n, hi int) int {
	if n < 0 {
		return 0
	}
	if n > hi {
		return hi
	}
	return n
}

// clampSlice bounds a list to maxListLen entries, each to maxFieldRunes.
func clampSlice(s []string) []string {
	if len(s) > maxListLen {
		s = s[:maxListLen]
	}
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = clampField(v)
	}
	return out
}

// clampFloat bounds a physical quantity; NaN/Inf read as lo.
func clampFloat(f, lo, hi float64) float64 {
	if f != f || f < lo { // NaN compares false to itself
		return lo
	}
	if f > hi {
		return hi
	}
	return f
}
