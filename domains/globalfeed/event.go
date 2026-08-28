// Package globalfeed is the global event ticker's data layer (0.12.0): the
// largest active hazard events in the world — significant earthquakes (USGS),
// tropical cyclones (NHC) and US severe/tornado warnings (NWS) — mapped to one
// Event model, deduped and stacked most-recent-first. It is a SEPARATE pipeline
// from the per-location snapshot: these events belong to no tracked location.
package globalfeed

import (
	"strings"
	"time"
)

// Class is which hazard family an event belongs to (the marquee rotates
// through the classes; the narration verb follows from it).
type Class int

const (
	ClassQuake    Class = iota // an earthquake / seismic event — "recorded"
	ClassTropical              // a tropical cyclone — "reported"
	ClassSevereWx              // a US severe/tornado warning — "declared"
)

// Severity is the background-colour tier (§plan): the ordering the stack and
// the marquee use, most severe first among equally-recent events.
type Severity int

const (
	SevYellow Severity = iota // watches / minor — yellow
	SevOrange                 // severe — orange
	SevRed                    // most severe (tornado, hurricane, great quake, tsunami) — red
)

// Event is one global hazard event, unified across the three feeds.
type Event struct {
	ID         string    // stable source id — the dedup key (USGS id, NHC id, NWS alert id)
	Class      Class     // hazard family
	Severity   Severity  // the bg tier
	Type       string    // the spoken "<severe alert type>": "Earthquake", "Hurricane", "Tornado Warning"
	Place      string    // the feed's raw place text, before D5 tying (locate.go resolves the spoken Location)
	Location   string    // the tied representative location (D5) — set by Locate
	Lat, Lon   float64   // the event point, for the watchlist tie and the nearest-city fuzzy area
	HasPoint   bool      // whether Lat/Lon is a real location (a zone-only NWS alert has none) — the radius filter excludes point-less alerts
	Superseded bool      // this alert was updated/replaced by a newer one (NWS references) — kept only so the ticker can seen-mark it, never displayed/announced
	At         time.Time // event / issue time — stack recency, the "declared/recorded" time
	Until      time.Time // active-window end (NWS ends/expires); zero = no expiry — a quake's instant, a storm the feed still lists (Active keeps it until the feed drops it)
	Source     string    // "USGS" | "NHC" | "NWS"
}

// Verb is how the narration says the event happened, by class (HUM LEAD script):
// a warning is declared, a quake recorded, a storm reported.
func (e Event) Verb() string {
	switch e.Class {
	case ClassQuake:
		return "recorded"
	case ClassTropical:
		return "reported"
	default:
		return "declared"
	}
}

// maxFieldRunes bounds a feed-supplied Type/Place so a hostile or compromised
// feed cannot make the marquee or the TTS narration render an unbounded string
// (red-team 0.12.0 P4 F5). The broadcast path caps a spoken segment near this;
// real event names and place descriptions are far shorter.
const maxFieldRunes = 120

// clampField truncates a feed field to maxFieldRunes (rune-safe).
func clampField(s string) string {
	r := []rune(s)
	if len(r) <= maxFieldRunes {
		return s
	}
	return string(r[:maxFieldRunes])
}

// Sentence is the event's lead line, shared by the marquee and the spoken
// narration (single owner): "A(n) <Type> has been <verb> for <Location>". The
// radio narration appends its own tail; the marquee shows this as-is.
func (e Event) Sentence() string {
	return e.Article() + " " + e.Type + " has been " + e.Verb() + " for " + e.Location
}

// Article is "A" or "An" agreeing with the Type's initial sound (a simple
// vowel test covers the real types: Earthquake→An, Tornado/Hurricane→A).
func (e Event) Article() string {
	if e.Type == "" {
		return "A"
	}
	switch strings.ToLower(e.Type[:1]) {
	case "a", "e", "i", "o", "u":
		return "An"
	default:
		return "A"
	}
}
