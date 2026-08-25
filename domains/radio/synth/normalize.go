package synth

import (
	"regexp"
	"strings"
)

// Normalizer turns NWS product text into spoken English (§5, RS-17):
// de-wraps the fixed-width lines, drops the WMO/AWIPS header and the "$$"
// footers, turns "..." into sentence breaks, expands the abbreviations the
// directives allow, and reads numbers the way the broadcast does. Unknown
// abbreviations pass through verbatim — a mispronunciation is not a failure.

var (
	// WMO/AWIPS header lines, UGC zone lines ("CAZ552-251015-", "CAZ043>048-251015-"), timestamps.
	headerLine = regexp.MustCompile(`^(\d{3}|[A-Z]{4}\d{2} [A-Z]{4} \d{6}|[A-Z]{6}|[A-Z]{2}[CZ]\d{3}[-\d>A-Z]*-\d{6}-)$`)
	timeStamp  = regexp.MustCompile(`^\d{3,4} [AP]M [A-Z]{3,4} [A-Za-z]{3} [A-Za-z]{3} \d{1,2} \d{4}$`)
	periodTag  = regexp.MustCompile(`^\.([A-Z][A-Z ]+?)\.\.\.`) // ".TONIGHT..." starts a period
	bulletTag  = regexp.MustCompile(`^([A-Z][A-Z ]+?)\.\.\.`)   // "WHAT..." labels a CAP bullet (UAT 95)
	// webAddress is a host with an alphabetic top-level label, optional
	// scheme/www and path — "www.weather.gov/sandiego"; never "3.5" or "U.S".
	webAddress = regexp.MustCompile(`(?i)^(https?://)?((?:[a-z0-9-]+\.)+[a-z]{2,})(/[^\s]*)?$`)
	ellipsis   = regexp.MustCompile(`\.(\s*\.)+`) // "..." and ". ." runs -> one sentence break
	multiSpace = regexp.MustCompile(`[ \t]+`)
	allCaps    = regexp.MustCompile(`^[A-Z][A-Z]+$`)
	clockTime  = regexp.MustCompile(`\b(\d{1,2})(\d{2}) ?([AP]M)\b`) // "442 PM" -> "4:42 PM" (read four forty-two)
	polygonKey = regexp.MustCompile(`^(LAT\.\.\.LON|TIME\.\.\.MOT\.\.\.LOC)`)
	numbersRow = regexp.MustCompile(`^[\d\s.]+$`)
)

// maxSegmentChars bounds one narrated segment (UAT 81): the marquee moves
// per segment and rendering stays short, so long zone forecasts are split
// at sentence ends into ~280-character pieces.
const maxSegmentChars = 280

// abbreviations is seeded from the NWS directives' contraction list; the
// full ~150-entry table is a checked-in TSV follow-on (§10.6).
func abbreviations() map[string]string {
	return map[string]string{
		"MPH": "miles per hour", "mph": "miles per hour", "KT": "knots", "kt": "knots", "KTS": "knots", "kts": "knots",
		"FT": "feet", "ft": "feet", "NM": "nautical miles", "nm": "nautical miles",
		"PDT": "Pacific Daylight Time", "PST": "Pacific Standard Time", "MDT": "Mountain Daylight Time", "MST": "Mountain Standard Time",
		"CDT": "Central Daylight Time", "CST": "Central Standard Time", "EDT": "Eastern Daylight Time", "EST": "Eastern Standard Time",
		"NWS": "National Weather Service", "NE": "northeast", "NW": "northwest", "SE": "southeast", "SW": "southwest",
		"SSW": "south southwest", "SSE": "south southeast", "NNW": "north northwest", "NNE": "north northeast",
		"WSW": "west southwest", "WNW": "west northwest", "ESE": "east southeast", "ENE": "east northeast",
		"TSTMS": "thunderstorms", "TSTM": "thunderstorm", "PCPN": "precipitation", "TEMPS": "temperatures", "HWY": "highway",
		"MTNS": "mountains", "VLYS": "valleys", "SFC": "surface", "PRES": "pressure", "VSBY": "visibility", "WX": "weather",
	}
}

// stateNames is the postal-abbreviation table (UAT 81: "San Diego CA" is
// read "San Diego California").
func stateNames() map[string]string {
	return map[string]string{
		"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas", "CA": "California", "CO": "Colorado", "CT": "Connecticut",
		"DE": "Delaware", "DC": "District of Columbia", "FL": "Florida", "GA": "Georgia", "HI": "Hawaii", "ID": "Idaho", "IL": "Illinois",
		"IN": "Indiana", "IA": "Iowa", "KS": "Kansas", "KY": "Kentucky", "LA": "Louisiana", "ME": "Maine", "MD": "Maryland",
		"MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota", "MS": "Mississippi", "MO": "Missouri", "MT": "Montana",
		"NE": "Nebraska", "NV": "Nevada", "NH": "New Hampshire", "NJ": "New Jersey", "NM": "New Mexico", "NY": "New York",
		"NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio", "OK": "Oklahoma", "OR": "Oregon", "PA": "Pennsylvania",
		"RI": "Rhode Island", "SC": "South Carolina", "SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah",
		"VT": "Vermont", "VA": "Virginia", "WA": "Washington", "WV": "West Virginia", "WI": "Wisconsin", "WY": "Wyoming",
		"PR": "Puerto Rico", "GU": "Guam", "VI": "Virgin Islands", "AS": "American Samoa", "MP": "Northern Mariana Islands",
	}
}

// ambiguousStates are abbreviations that are also words; they expand only
// in a clear place context (after a comma or beside a hyphen).
func ambiguousStates() map[string]bool {
	m := map[string]bool{}
	for _, w := range []string{"IN", "OR", "ME", "HI", "OK", "ID", "LA", "MA", "PA", "MT", "MO", "MS", "DE", "CO", "AL", "AS"} {
		m[w] = true
	}
	return m
}

// ExpandStates replaces postal state abbreviations where they name a
// place: after a comma ("Oceanside, CA"), beside a hyphen ("CA-San Diego
// County"), or after a Title-case word ("San Diego CA"). Words that are
// also English ("IN EFFECT", "OR") expand only after a comma or hyphen.
func ExpandStates(s string) string {
	names, ambiguous := stateNames(), ambiguousStates()
	runes := []rune(s)
	var out strings.Builder
	for i := 0; i < len(runes); i++ {
		if i+1 < len(runes) && isUpper(runes[i]) && isUpper(runes[i+1]) && (i+2 >= len(runes) || !isWordRune(runes[i+2])) && (i == 0 || !isWordRune(runes[i-1])) {
			tok := string(runes[i : i+2])
			if full, ok := names[tok]; ok && stateContext(runes, i, ambiguous[tok]) {
				out.WriteString(full)
				i++
				continue
			}
		}
		out.WriteRune(runes[i])
	}
	return out.String()
}

// stateContext decides whether the two letters at i name a place.
func stateContext(runes []rune, i int, ambiguous bool) bool {
	afterComma := i >= 2 && runes[i-1] == ' ' && runes[i-2] == ','
	hyphen := (i >= 1 && runes[i-1] == '-') || (i+2 < len(runes) && runes[i+2] == '-')
	if afterComma || hyphen {
		return true
	}
	if ambiguous || i < 2 {
		return false // nothing can precede a code at the start (round 2 N-1: "CA" at index 0 sliced runes[-1:-1]); "CA-…" was handled above
	}
	// Preceded by a Title-case word: "San Diego CA".
	j := i - 2                                                  // skip the space before the token
	for k := 0; k < 64 && j >= 0 && isWordRune(runes[j]); k++ { // bounded per P10-02: a word is < 64 runes
		j--
	}
	word := runes[j+1 : max(j+1, i-1)]
	return len(word) >= 2 && isUpper(word[0]) && !isUpper(word[1])
}

func isUpper(r rune) bool    { return r >= 'A' && r <= 'Z' }
func isWordRune(r rune) bool { return isUpper(r) || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') }

// NormalizeLine applies the word rules to a single line that is already a
// sentence (CAP headlines, UAT 82): clock times, abbreviations ("NWS" ->
// "National Weather Service"), state names.
func NormalizeLine(line string) string {
	s := clockTime.ReplaceAllString(strings.TrimSpace(line), "$1:$2 $3")
	s = ExpandStates(expandAbbreviations(s))
	return multiSpace.ReplaceAllString(s, " ")
}

// Normalize renders one product's text as narration paragraphs.
func Normalize(text string) []string {
	var paras []string
	for _, block := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
		para := normalizeBlock(block)
		if para != "" {
			paras = append(paras, para)
		}
	}
	return paras
}

// normalizeBlock de-wraps one blank-line-separated block into sentences.
func normalizeBlock(block string) string {
	var kept []string
	for _, line := range strings.Split(block, "\n") {
		l := strings.TrimSpace(line)
		if l == "" || l == "$$" || l == "&&" || headerLine.MatchString(l) || timeStamp.MatchString(l) {
			continue
		}
		if polygonKey.MatchString(l) || (numbersRow.MatchString(l) && len(kept) > 0 && strings.HasPrefix(kept[len(kept)-1], "LAT")) {
			kept = append(kept, "LAT") // polygon coordinates are not narrated (UAT 81); mark so wrapped rows drop too
			continue
		}
		kept = append(kept, labelLine(l))
	}
	kept = dropMarked(kept)
	if len(kept) == 0 {
		return ""
	}
	s := strings.Join(kept, " ")
	s = ellipsis.ReplaceAllString(s, ". ")        // "..." -> sentence break
	s = clockTime.ReplaceAllString(s, "$1:$2 $3") // "442 PM" -> "4:42 PM"
	s = ExpandStates(expandAbbreviations(s))
	s = multiSpace.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, " .", ".")
	s = strings.Trim(strings.TrimSpace(s), "-. ")
	if s == "" {
		return ""
	}
	return capitalizeSentences(s) + "."
}

// labelLine turns a product's labelled lines into spoken labels: a period
// tag ".TONIGHT...Mostly clear" -> "Tonight. Mostly clear"; a CAP bullet
// "* WHAT...Hot" -> "What: Hot" — a statement, never "asterisk", never
// "What?" (UAT 95). Split from normalizeBlock (P10-04).
func labelLine(l string) string {
	if m := periodTag.FindStringSubmatch(l); m != nil {
		return m[1] + ". " + strings.TrimSpace(l[len(m[0]):])
	}
	if strings.HasPrefix(l, "* ") {
		l = strings.TrimPrefix(l, "* ")
		if m := bulletTag.FindStringSubmatch(l); m != nil {
			return m[1] + ": " + strings.TrimSpace(l[len(m[0]):])
		}
	}
	return l
}

// properNouns stay capitalized when the product shouted them.
func properNouns() map[string]bool {
	m := map[string]bool{}
	for _, w := range []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday",
		"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"} {
		m[strings.ToUpper(w)] = true
	}
	return m
}

// capitalizeSentences reads shouted words quietly: all-caps words become
// lower case (a synthesizer would spell them) — proper nouns keep their
// capital — then every sentence starts with a capital.
func capitalizeSentences(s string) string {
	proper := properNouns()
	words := strings.Fields(s)
	for i, w := range words {
		core := strings.Trim(w, ".,;:")
		if allCaps.MatchString(core) && core != "AM" && core != "PM" { // AM/PM stay as read ("4:42 PM")
			lower := strings.ToLower(core)
			if proper[core] {
				lower = strings.ToUpper(lower[:1]) + lower[1:]
			}
			words[i] = strings.Replace(w, core, lower, 1)
		}
	}
	out := strings.Join(words, " ")
	runes := []rune(out)
	start := true
	for i, r := range runes {
		switch {
		case start && r >= 'a' && r <= 'z':
			runes[i] = r - 'a' + 'A'
			start = false
		case r == '.' || r == '!' || r == '?':
			start = true
		case r != ' ':
			start = false
		}
	}
	return string(runes)
}

func dropMarked(lines []string) []string {
	out := lines[:0]
	for _, l := range lines {
		if l != "LAT" {
			out = append(out, l)
		}
	}
	return out
}

// Segments splits narration paragraphs into pieces of at most
// maxSegmentChars at sentence ends (UAT 81).
func Segments(paras []string) []string {
	var out []string
	for _, para := range paras {
		for n := 0; n < 1000 && len(para) > maxSegmentChars; n++ { // bounded per P10-02: each pass shortens para
			cut := strings.LastIndex(para[:maxSegmentChars], ". ")
			if cut < maxSegmentChars/3 {
				cut = strings.LastIndex(para[:maxSegmentChars], " ")
				if cut <= 0 {
					cut = maxSegmentChars - 1
				}
				out = append(out, strings.TrimSpace(para[:cut+1]))
				para = strings.TrimSpace(para[cut+1:])
				continue
			}
			out = append(out, strings.TrimSpace(para[:cut+1]))
			para = strings.TrimSpace(para[cut+2:])
		}
		if para != "" {
			out = append(out, para)
		}
	}
	return out
}

// expandAbbreviations replaces whole-word entries; words are matched
// case-sensitively (products are upper-case where they abbreviate).
func expandAbbreviations(s string) string {
	dict := abbreviations()
	words := strings.Fields(s)
	for i, w := range words {
		core := strings.Trim(w, ".,;:")
		if full, ok := dict[core]; ok {
			words[i] = strings.Replace(w, core, full, 1)
		}
	}
	return strings.Join(words, " ")
}

// pronunciations are voice-only substitutions (UAT 83): what the screen
// shows stays as written; what the synthesizer hears is spelled the way
// it should be said. Whole words, case-sensitive.
func pronunciations() map[string]string {
	return map[string]string{
		"CLI":  "C L I", // never "klee"
		"NOAA": "Noah",  // the agency says it as a word (UAT 113.2: "Noah", one syllable flow — "NO-AH" clipped mid-word), never "N O double A"
	}
}

// Pronounce applies the voice-only substitutions to a segment's text:
// the word table, and web addresses spelled the way they are read out
// (UAT 95).
func Pronounce(text string) string {
	table := pronunciations()
	words := strings.Fields(text)
	for i, w := range words {
		core := strings.Trim(w, ".,;:")
		if say, ok := table[core]; ok {
			words[i] = strings.Replace(w, core, say, 1)
			continue
		}
		if base, poss := strings.CutSuffix(core, "'s"); poss { // "NOAA's" (UAT 115): the possessive keeps the word's sound
			if say, ok := table[base]; ok {
				words[i] = strings.Replace(w, base, say, 1)
				continue
			}
		}
		if m := webAddress.FindStringSubmatch(core); m != nil {
			words[i] = strings.Replace(w, core, spellAddress(m[2]+m[3]), 1)
			continue
		}
		if callsign.MatchString(core) { // UAT 112: "KEC62" is said "K E C six two", never "keck sixty-two"
			words[i] = strings.Replace(w, core, spellCallsign(core), 1)
			continue
		}
		// UAT 112.2: a frequency is read digit by digit — "162.400 MHz" is
		// "one six two dot four zero zero mega hertz" (only before MHz: "3.5
		// inches" stays a number).
		if i+1 < len(words) && strings.EqualFold(strings.Trim(words[i+1], ".,;:"), "MHz") && frequency.MatchString(core) {
			words[i] = strings.Replace(w, core, spellDigits(core), 1)
			words[i+1] = strings.Replace(words[i+1], strings.Trim(words[i+1], ".,;:"), "mega hertz", 1)
		}
	}
	return strings.Join(words, " ")
}

// frequency is a decimal megahertz figure ("162.400", "165.2024").
var frequency = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)

// spellDigits reads digits one by one with "dot" for the point.
func spellDigits(s string) string {
	out := make([]string, 0, len(s))
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			out = append(out, digitWords()[r-'0'])
		case r == '.':
			out = append(out, "dot")
		}
	}
	return strings.Join(out, " ")
}

// digitWords names the digits for the voice (a function, not a global — P10-06).
func digitWords() []string {
	return []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}
}

// callsign is an NWR transmitter callsign: K or W, two letters, two or
// three digits (KEC62, WXJ98, KZZ100).
var callsign = regexp.MustCompile(`^[KW][A-Z]{2}[0-9]{2,3}$`)

// spellCallsign reads a callsign letter by letter and digit by digit.
func spellCallsign(c string) string {
	out := make([]string, 0, len(c))
	for _, r := range c {
		if r >= '0' && r <= '9' {
			out = append(out, digitWords()[r-'0'])
		} else {
			out = append(out, string(r))
		}
	}
	return strings.Join(out, " ")
}

// spellAddress reads an address aloud: "www" letter by letter, dots,
// slashes, dashes and underscores by name; the scheme is not said.
func spellAddress(addr string) string {
	r := strings.NewReplacer(".", " dot ", "/", " slash ", "-", " dash ", "_", " underscore ")
	s := r.Replace(addr)
	words := strings.Fields(s)
	for i, w := range words {
		if strings.EqualFold(w, "www") {
			words[i] = "w w w"
		}
	}
	return strings.Join(words, " ")
}
