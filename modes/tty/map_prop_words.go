package tty

// map_prop_words.go — the Propagation mode's words without the picture
// (0.19.0 W6.1): in D-136's order - the origin and its MUF and foF2, day or
// night there, a status block, the best bands one line per target, the
// frequency held, the centre - and written for speech (FR-3.2): megahertz,
// metres and kilometres in full, each figure with its unit.

import (
	"context"
	"errors"
	"fmt"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/propanswers"
	"regexp"
	"strings"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// propFocusLimit is FR-3.7's statement (D-133): Help carries it, and the
// words say it on opening.
const propFocusLimit = "A screen reader cannot yet follow focus in the Propagation mode; watchpost 0.19.2 fixes that."

// propTerms are the short names the words use, said in full on opening the
// mode, as the focus limit is (AX-8).
const propTerms = "The words' short names: MUF, the maximum usable frequency; foF2, the F2 layer's critical frequency; LUF, the lowest usable frequency; M(3000)F2, the factor from foF2 to MUF(3000); D-RAP, NOAA's forecast of the D layer's absorption; GloTEC, NOAA's map of the ionosphere's electrons."

// propLayerFull are the layers' full names, said once (FR-3.2).
var propLayerFull = map[int]string{
	propMUF:  "the maximum usable frequency over 3,000 kilometres",
	propFoF2: "the F2 layer's critical frequency",
}

// propWords are the Propagation mode's lines of the description.
func (d Dashboard) propWords(lat, lon float64) []string {
	sentences := d.propSentences(lat, lon)
	out := make([]string, len(sentences))
	for i, s := range sentences {
		out[i] = s.text
	}
	return out
}

// wordSentence is a sentence of the Propagation mode's words and what it is
// about, by kind - "status", a target's name - which a reader's place in
// the words is kept by (D-211).
type wordSentence struct {
	keys []string
	text string
}

// said is a sentence of one kind.
func said(key, text string) wordSentence { return wordSentence{keys: []string{key}, text: text} }

// propSentences are the Propagation mode's words, each with its kind.
func (d Dashboard) propSentences(lat, lon float64) []wordSentence {
	if d.mapPane.propSnap == nil {
		return []wordSentence{said("mode", propagationWords), said("status", "Status: "+spoken(strings.TrimPrefix(d.propStatusWords(), "Propagation · "))+"."), said("focus", propFocusLimit)}
	}
	out := []wordSentence{said("mode", "The Propagation mode, over the whole world: "+propLayerFull[d.mapPane.propLayer]+", "+propLayerName(d.mapPane.propLayer)+", is drawn, in megahertz.")}
	if a := d.propAheadWords(lat, lon); a != "" {
		out = append(out, said("ahead", spoken(a))) // an hour ahead: typical, before the place (FR-1.13, D-136)
	}
	if f := d.propFoundWords(); f != "" {
		out = append(out, said("found", spoken(f))) // the place found, said (FR-1.10)
	}
	if o := d.propOriginWords(); len(o) > 0 {
		out = append(out, said("origin", o[0]))
		if len(o) > 1 {
			out = append(out, said("day", o[1]))
		}
	}
	out = append(out, said("status", d.propStatusBlock()))
	if d.mapPane.propSess.applied < 2 {
		out = append(out, said("focus", propFocusLimit), said("terms", propTerms)) // on opening, not after a refresh (FR-3.7, D-136, AX-8)
	}
	out = append(out, d.propTargetWords()...)
	if d.mapPane.propFreqMHz != 0 {
		out = append(out, said("held", spoken("Your frequency: "+d.mapPane.propFreqLabel+" held; its reach: "+d.propReachWords())+".")) // FR-1.16, D-127
	}
	if c := d.propCentreWords(); c != "" {
		out = append(out, said("centre", spoken(c)))
	}
	return out
}

// propOriginWords are the origin's answer for the hour shown, and whether
// it is day or night there and when that turns (D-64, AX-E).
func (d Dashboard) propOriginWords() []string {
	origin := d.mapPane.propOrigin
	if !origin.ok {
		return nil
	}
	shown, ready := d.propShown()
	origin.name = geo.SpokenPlace(origin.name, origin.country) // a place abroad by its country (D-231)
	when := "now"
	if k := d.mapPane.propAhead; k > 0 {
		when = hoursWords(k) + " ahead"
	}
	if !ready && d.mapPane.propAhead > 0 {
		return []string{upperFirst(spoken(origin.name + ", " + when + ": being worked out."))} // never now's figures said as the hour's
	}
	fo, okF := cellValue(shown.Now.FoF2, origin.at.Lat, origin.at.Lon)
	muf, okM := cellValue(shown.Now.MUF3000, origin.at.Lat, origin.at.Lon)
	if !okF || !okM {
		return []string{upperFirst(spoken(origin.name + ", " + when + ": no data at the place."))}
	}
	answer := fmt.Sprintf("%s, %s: MUF(3000) %s MHz, foF2 %s MHz.", origin.name, when, propanswers.MHz(muf), propanswers.MHz(fo))
	out := []string{upperFirst(spoken(answer))}                    // a centre named by its coordinates, said as degrees (AX-1)
	if w := d.propDayWords(origin, d.propNightMoment()); w != "" { // the night's own moment: words and picture at one (R2C-8)
		out = append(out, spoken(w))
	}
	return out
}

// propDayWords say whether the sun is up at a place at the moment shown, and
// when it next rises or sets there, by the window's clock: go-tuiMaps' sun,
// the one the night is drawn with for that moment, so the words and the
// picture agree an hour ahead as now (D-258, D-263). Empty for no moment.
func (d Dashboard) propDayWords(p propPlace, at time.Time) string {
	day := tuimaps.DaylightAt(tuimaps.LonLat{Lon: p.at.Lon, Lat: p.at.Lat}, at)
	if !day.Known {
		return ""
	}
	side, turn := "night", "rises"
	if day.Day {
		side, turn = "day", "sets"
	}
	words := "It is " + side + " at " + p.name
	if day.Turns.IsZero() {
		return words + ", and the sun neither rises nor sets there today."
	}
	next := day.Turns.In(d.now().Location())
	return words + "; the sun " + turn + " there at " + d.clockFmt.Time(next) + " " + next.Format("MST") + "."
}

// propStatusBlock is D-136's status block, said once: the data's age, what
// the field was made from, the origin's nearest station, a disagreement with
// the typical day, when it was updated after a refresh, and a failed update
// with the key that asks again.
func (d Dashboard) propStatusBlock() string {
	snap := *d.mapPane.propSnap
	age, _, _ := d.propAge(snap)
	made := strings.TrimPrefix(snapshotWords(snap, d.mapPane.propLayer), "Propagation · "+propLayerName(d.mapPane.propLayer)+" · ")
	parts := []string{age, "computed " + made}
	if sw := propScalesWords(snap.Scales); sw != "" {
		parts = append(parts, sw) // FR-1.18
	}
	if d.propChartShown() {
		c := d.mapPane.propChart
		if w, once := d.propStationOnce(); once {
			parts = append(parts, "for the origin and every target, "+w) // said here, and on no row (D-210)
		} else {
			parts = append(parts, "at the origin, "+d.propStationWords(c.local.Nearest))
		}
		if propanswers.Differs(c.local.Bands, c.key.hour) {
			parts = append(parts, "now differs from a typical day there")
		}
	}
	if d.mapPane.propSess.applied > 1 {
		at := d.mapPane.propUpdatedAt.In(d.now().Location())
		parts = append(parts, "updated at "+d.clockFmt.Time(at)+" "+at.Format("MST"))
	}
	if w := d.propFailedWords(); w != "" {
		parts = append(parts, w)
	}
	return "Status: " + spoken(strings.Join(parts, "; ")) + "."
}

// propFailedWords say an update that failed over the field kept (FR-5.2),
// with the key that asks again (D-120); empty when none failed.
func (d Dashboard) propFailedWords() string {
	err := d.mapPane.propErr
	if err == nil || errors.Is(err, context.Canceled) {
		return ""
	}
	return "the last update failed: " + strings.TrimPrefix(err.Error(), "ionomaps: ") + "; " + d.firstKey(actMapRefresh) + " refreshes now"
}

// propTargetWords are the best bands now, the local area's line and then
// the targets', each its name first, the open bands before the closed, and
// its own nearest station (D-127, D-116) - unless every row's reads the
// same, when the status block says it once; targets with the same answer
// share one line, their names first (D-210).
func (d Dashboard) propTargetWords() []wordSentence {
	if d.propChartFailed() {
		return []wordSentence{said("targets", "The best bands are "+d.propNotAvailable()+".")} // AX-9
	}
	if !d.propChartShown() {
		return []wordSentence{said("targets", "The best bands are being worked out.")}
	}
	names, rows := d.propChartRows()
	_, once := d.propStationOnce()
	var order []string // each answer said, in the order first met
	named := map[string][]string{}
	for i, a := range rows {
		say := propanswers.BandsSaid(a.Bands)
		if w := propanswers.RowWords(a.Bands); w != "" {
			say += "; " + w
		}
		if !once {
			say += "; " + d.propStationWords(a.Nearest)
		}
		if pf := d.mapPane.propFound; pf != nil && names[i] == d.mapPane.propChart.key.found.Name {
			names[i] = geo.SpokenPlace(names[i], pf.Country) // a place found abroad by its country (D-231)
		}
		if strings.Contains(names[i], ",") {
			say = names[i] + "\x00" + say // a name with a comma - the local area's, a place found - keeps its own line: a list would blur it
		}
		if _, met := named[say]; !met {
			order = append(order, say)
		}
		named[say] = append(named[say], names[i])
	}
	out := make([]wordSentence, 0, len(order))
	for _, say := range order {
		keys := []string{"targets"} // a reader on the line that said them being worked out stays with them
		for _, n := range named[say] {
			keys = append(keys, "target:"+n)
		}
		_, line, cut := strings.Cut(say, "\x00")
		if !cut {
			line = say
		}
		out = append(out, wordSentence{keys: keys, text: spoken(propanswers.List(named[say])+": "+line) + "."})
		if len(out) == 1 {
			out = append(out, said("typical", spoken(propanswers.TypicalWords(rows[0].Bands, d.mapPane.propChart.key.hour)))) // the local area's typical day, after its line (FR-1.15, AX-4)
		}
	}
	return out
}

// propStationOnce is the nearest station said once (D-210): its words, and
// true when every row of the chart - the local area's and each target's -
// reads the same.
func (d Dashboard) propStationOnce() (string, bool) {
	_, rows := d.propChartRows()
	if len(rows) == 0 {
		return "", false
	}
	first := d.propStationWords(rows[0].Nearest)
	for _, a := range rows[1:] {
		if d.propStationWords(a.Nearest) != first {
			return "", false
		}
	}
	return first, true
}

// upperFirst is a sentence begun with a capital: a view's name may not be.
func upperFirst(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// propStationWords say an answer's nearest reporting station (D-116,
// D-198): its distance, and beyond the far distance, or with none, that the
// answer is about as good as typical here, or worse (D-129: the Pacific's
// hybrid did worse than the climatology).
func (d Dashboard) propStationWords(n ionomaps.Nearest) string {
	var w string
	switch {
	case !n.Known:
		w = "no reporting station, about as good as typical here, or worse"
	case n.Km >= propFarKm:
		w = "nearest station " + d.roundDistance(float64(n.Km)) + " away, about as good as typical here, or worse"
	default:
		w = "nearest station " + d.roundDistance(float64(n.Km)) + " away"
	}
	if n.Known && n.Unscored {
		w += ", unscored: verify on air"
	}
	return w
}

// spokenRules write the windows' short forms out for speech, in order.
var spokenRules = []struct {
	re *regexp.Regexp
	to string
}{
	{regexp.MustCompile(`(\d)°N`), "$1 degrees north"},
	{regexp.MustCompile(`(\d)°S`), "$1 degrees south"},
	{regexp.MustCompile(`(\d)°E`), "$1 degrees east"},
	{regexp.MustCompile(`(\d)°W`), "$1 degrees west"},
	{regexp.MustCompile(`(\d)°F`), "$1 degrees Fahrenheit"},
	{regexp.MustCompile(`(\d)°C`), "$1 degrees Celsius"},
	{regexp.MustCompile(`(\d+)m-(\d+)m\b`), "$1 to $2 metres"},
	{regexp.MustCompile(`\b(\d+)m\b`), "$1 metres"},
	{regexp.MustCompile(`\bMHz\b`), "megahertz"},
	{regexp.MustCompile(`\bkm\b`), "kilometres"},
	{regexp.MustCompile(`\b(\d+) mi\b`), "$1 miles"},
	{regexp.MustCompile(`\b(\d+) W\b`), "$1 watts"},
	{regexp.MustCompile(`~ ?`), "about "},
	{regexp.MustCompile(`(\d+)%`), "$1 percent"},
	{regexp.MustCompile(`\b1 min\b`), "1 minute"},
	{regexp.MustCompile(`\b(\d+) min\b`), "$1 minutes"},
	{regexp.MustCompile(`\b1 h\b`), "1 hour"},
	{regexp.MustCompile(`\b(\d+) h\b`), "$1 hours"},
	{regexp.MustCompile(` · `), ", "},
}

// spokenFrequency is a frequency's figure after the name the windows give
// it - a limit, "best about", "under", "up to" - and its unit if written;
// the windows write it bare after the first, so the words give it its unit.
// A figure no frequency's name leads is left as it is: coordinates, hours.
var spokenFrequency = regexp.MustCompile(`\b(MUF\(3000\)|MUF|foF2|LUF|D-RAP to|D-RAP|best about|under|up to) (\d+\.\d+)(\+?)( megahertz)?`)

// spokenCoordinates are a pair of coordinates as a typed place's label
// writes them: "33.2000, -117.3800".
var spokenCoordinates = regexp.MustCompile(`(-?\d{1,2}\.\d{4}), (-?\d{1,3}\.\d{4})`)

// spoken is a window's text written for speech (FR-3.2): units and band
// names in full, a typed place's coordinates as degrees, and a frequency's
// figure given its unit (AX-2).
func spoken(s string) string {
	s = spokenCoordinates.ReplaceAllStringFunc(s, func(m string) string {
		p := spokenCoordinates.FindStringSubmatch(m)
		return hemisphere(p[1], "north", "south") + " " + hemisphere(p[2], "east", "west")
	})
	for _, r := range spokenRules {
		s = r.re.ReplaceAllString(s, r.to)
	}
	return spokenFrequency.ReplaceAllStringFunc(s, func(m string) string {
		p := spokenFrequency.FindStringSubmatch(m)
		out := p[1] + " " + p[2] + " megahertz"
		if p[3] != "" {
			out += " or higher" // "go up to 3.1+"
		}
		return out
	})
}

// hemisphere is a signed coordinate said as degrees and its side.
func hemisphere(v, plus, minus string) string {
	if n, ok := strings.CutPrefix(v, "-"); ok {
		return n + " degrees " + minus
	}
	return v + " degrees " + plus
}

// spokenAll is each line written for speech.
func spokenAll(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = spoken(l)
	}
	return out
}
