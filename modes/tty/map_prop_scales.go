package tty

// map_prop_scales.go — NOAA's space-weather scales named (0.19.0 W5.5,
// FR-1.18, D-88): a scale above 0 by its level and NOAA's word, NOAA's
// outlook of a storm to come, and a missing feed said, never level 0.

import (
	"strconv"
	"strings"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// scaleWords are NOAA's words for a level, 1 to 5.
var scaleWords = []string{"minor", "moderate", "strong", "severe", "extreme"}

// propScales are the scales, each with what it measures, said plural or as
// one storm.
var propScales = []struct {
	letter, what string
	storm        bool
	level        func(ionomaps.Scales) int
}{
	{"R", "radio blackouts", false, func(s ionomaps.Scales) int { return s.R }},
	{"S", "solar radiation storm", true, func(s ionomaps.Scales) int { return s.S }},
	{"G", "geomagnetic storm", true, func(s ionomaps.Scales) int { return s.G }},
}

// scaleNotRead is said when no scales were read (D-88).
const scaleNotRead = "NOAA's scales not read"

// raised is whether a level is 1 to 5: unset or 0 is not.
func raised(level int) bool { return level >= 1 && level <= len(scaleWords) }

// propScaleParts are the scales above 0, in the chart's short form ("R2
// moderate radio blackouts") or the words' ("R2, moderate radio
// blackouts").
func propScaleParts(s ionomaps.Scales, spoken bool) []string {
	var parts []string
	for _, sc := range propScales {
		if l := sc.level(s); raised(l) {
			parts = append(parts, scaleName(sc.letter, sc.what, sc.storm, l, spoken))
		}
	}
	return parts
}

// scaleName is one scale at a level.
func scaleName(letter, what string, storm bool, level int, spoken bool) string {
	word := scaleWords[level-1]
	if !spoken {
		return letter + strconv.Itoa(level) + " " + word + " " + what
	}
	if storm {
		word = article(word) + " " + word
	}
	return letter + strconv.Itoa(level) + ", " + word + " " + what
}

// article is "a" or "an" before a word.
func article(word string) string {
	if strings.ContainsRune("aeiou", rune(word[0])) {
		return "an"
	}
	return "a"
}

// propOutlookParts are NOAA's outlook days that expect a geomagnetic
// storm, by day.
func propOutlookParts(s ionomaps.Scales, spoken bool) []string {
	var parts []string
	for _, day := range s.Outlook {
		if !raised(day.G) {
			continue
		}
		if spoken {
			parts = append(parts, scaleName("G", "geomagnetic storm", true, day.G, true)+", "+day.Day.Format("Monday 2 January"))
		} else {
			parts = append(parts, "G"+strconv.Itoa(day.G)+" "+day.Day.Format("Mon 2 Jan"))
		}
	}
	return parts
}

// propScalesLine is the chart's line: the scales raised and the outlook,
// or that none were read; empty while all are at 0 and none is expected.
func propScalesLine(s ionomaps.Scales) string {
	if s.Valid.IsZero() {
		return "SPACE WEATHER: " + scaleNotRead
	}
	parts := propScaleParts(s, false)
	if o := propOutlookParts(s, false); len(o) > 0 {
		parts = append(parts, "outlook "+strings.Join(o, ", "))
	}
	if len(parts) == 0 {
		return ""
	}
	return "SPACE WEATHER (NOAA): " + strings.Join(parts, " · ")
}

// propScalesWords are the status block's clause: the same, in words.
func propScalesWords(s ionomaps.Scales) string {
	if s.Valid.IsZero() {
		return "space weather: " + scaleNotRead
	}
	parts := propScaleParts(s, true)
	if o := propOutlookParts(s, true); len(o) > 0 {
		parts = append(parts, "NOAA expects "+listWords(o))
	}
	if len(parts) == 0 {
		return ""
	}
	return "space weather: " + strings.Join(parts, "; ")
}

// propStorm is whether a geomagnetic storm is in progress: G 1 or more
// (D-109).
func propStorm(s ionomaps.Scales) bool { return raised(s.G) }
