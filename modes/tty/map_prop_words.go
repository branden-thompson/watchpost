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
	"regexp"
	"strconv"
	"strings"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/astro"
)

// propFocusLimit is FR-3.7's statement (D-133): Help carries it, and the
// words say it on opening.
const propFocusLimit = "A screen reader cannot yet follow focus in the Propagation mode; watchpost 0.19.1 fixes that."

// propLayerFull are the layers' full names, said once (FR-3.2).
var propLayerFull = map[int]string{
	propMUF:  "the maximum usable frequency over 3,000 kilometres",
	propFoF2: "the F2 layer's critical frequency",
}

// propWords are the Propagation mode's lines of the description.
func (d Dashboard) propWords(lat, lon float64) []string {
	if d.mapPane.propSnap == nil {
		return []string{propagationWords, "Status: " + spoken(strings.TrimPrefix(d.propStatusWords(), "Propagation · ")) + ".", propFocusLimit}
	}
	out := []string{"The Propagation mode, over the whole world: " + propLayerFull[d.mapPane.propLayer] + ", " + propLayerName(d.mapPane.propLayer) + ", is drawn, in megahertz."}
	if a := d.propAheadWords(lat, lon); a != "" {
		out = append(out, spoken(a)) // an hour ahead: typical, before the place (FR-1.13, D-136)
	}
	if f := d.propFoundWords(); f != "" {
		out = append(out, spoken(f)) // the place found, said (FR-1.10)
	}
	out = append(out, d.propOriginWords()...)
	out = append(out, d.propStatusBlock())
	if d.mapPane.propApplied < 2 {
		out = append(out, propFocusLimit) // on opening, not after a refresh (FR-3.7, D-136)
	}
	out = append(out, d.propTargetWords()...)
	if d.mapPane.propFreqMHz != 0 {
		out = append(out, spoken("Your frequency: "+d.mapPane.propFreqLabel+" held; its reach: "+d.propReachWords())+".") // FR-1.16, D-127
	}
	if c := d.propCentreWords(); c != "" {
		out = append(out, spoken(c))
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
	shown := d.propOnScreen()
	when := "now"
	if k := d.mapPane.propAhead; k > 0 {
		when = spoken(strconv.Itoa(k)+" h") + " ahead"
	}
	fo, okF := cellValue(shown.Hours[0].FoF2, origin.at.Lat, origin.at.Lon)
	muf, okM := cellValue(shown.Hours[0].MUF3000, origin.at.Lat, origin.at.Lon)
	if !okF || !okM {
		return []string{upperFirst(origin.name + ", " + when + ": no data at the place.")}
	}
	answer := fmt.Sprintf("%s, %s: MUF(3000) %s megahertz, foF2 %s megahertz.", origin.name, when, mhzWords(muf), mhzWords(fo))
	return []string{upperFirst(answer), d.propDayWords(origin, shown.Hours[0].At)}
}

// propDayWords say whether the sun is up at a place at a moment, and when
// it next rises or sets there, by the window's clock.
func (d Dashboard) propDayWords(p propPlace, at time.Time) string {
	side, turn := "night", "rises"
	if astro.Up(p.at.Lat, p.at.Lon, at) {
		side, turn = "day", "sets"
	}
	words := "It is " + side + " at " + p.name
	next, ok := astro.NextTurn(p.at.Lat, p.at.Lon, at)
	if !ok {
		return words + ", and the sun neither rises nor sets there today."
	}
	next = next.In(d.now().Location())
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
		parts = append(parts, "at the origin, "+d.propStationWords(c.local.Nearest))
		if propDiffers(c.local.Bands, c.key.hour) {
			parts = append(parts, "now differs from a typical day there")
		}
	}
	if d.mapPane.propApplied > 1 {
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

// propTargetWords are the best bands now, one line per target, its name
// first, the open bands before the closed, and its own nearest station
// (D-127, D-116).
func (d Dashboard) propTargetWords() []string {
	if !d.propChartShown() {
		return []string{"The best bands are being worked out."}
	}
	names, answers := d.propChartRows()
	out := make([]string, 0, len(names))
	for i, a := range answers {
		line := names[i] + ": " + propBandsSaid(a.Bands)
		if w := propRowWords(a.Bands); w != "" {
			line += "; " + w
		}
		out = append(out, spoken(line+"; "+d.propStationWords(a.Nearest))+".")
	}
	return out
}

// propStatusSaid are the bands' states in the order said: the open first.
var propStatusSaid = []struct {
	status ionomaps.Status
	words  string
}{
	{ionomaps.Open, "open on "},
	{ionomaps.Disturbed, "disturbed on "},
	{ionomaps.Absorbed, "absorbed on "},
	{ionomaps.Weak, "too weak on "},
	{ionomaps.AboveUpper, "above the upper limit on "},
}

// propBandsSaid are a target's bands by state, the open before the closed:
// "open on 20m; above the upper limit on 17, 15, 12 and 10m".
func propBandsSaid(bands []ionomaps.BandStatus) string {
	var parts []string
	for _, s := range propStatusSaid {
		var nums []string
		for i, b := range bands {
			if b.Status == s.status && i < len(hfBands) {
				nums = append(nums, strings.TrimSuffix(hfBands[i].name, "m"))
			}
		}
		if len(nums) > 0 {
			parts = append(parts, s.words+listWords(nums)+"m")
		}
	}
	if len(parts) == 0 {
		return "no data"
	}
	if bandsOpen(bands) == 0 {
		parts = append([]string{"nothing open"}, parts...)
	}
	return strings.Join(parts, "; ")
}

// bandsOpen counts the bands open or disturbed.
func bandsOpen(bands []ionomaps.BandStatus) int {
	n := 0
	for _, b := range bands {
		if b.Status == ionomaps.Open || b.Status == ionomaps.Disturbed {
			n++
		}
	}
	return n
}

// upperFirst is a sentence begun with a capital: a view's name may not be.
func upperFirst(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// listWords joins words as a sentence lists them: "a", "a and b", "a, b
// and c".
func listWords(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// propStationWords say an answer's nearest reporting station (D-116,
// D-198): its distance, and beyond the far distance, or with none, that the
// answer is about as good as typical.
func (d Dashboard) propStationWords(n ionomaps.Nearest) string {
	var w string
	switch {
	case !n.Known:
		w = "no reporting station, about as good as typical"
	case n.Km >= propFarKm:
		w = "nearest station " + d.roundDistance(float64(n.Km)) + " away, about as good as typical"
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

// spokenFigure is a decimal figure and the word after it, if any.
var spokenFigure = regexp.MustCompile(`\d+\.\d+( [a-z]+)?`)

// spoken is a window's text written for speech (FR-3.2): units and band
// names in full, and a frequency's figure, which the windows write bare
// after its first, given its unit.
func spoken(s string) string {
	for _, r := range spokenRules {
		s = r.re.ReplaceAllString(s, r.to)
	}
	return spokenFigure.ReplaceAllStringFunc(s, func(m string) string {
		if strings.HasSuffix(m, " megahertz") || strings.HasSuffix(m, " degrees") {
			return m
		}
		num, rest, _ := strings.Cut(m, " ")
		if rest != "" {
			rest = " " + rest
		}
		return num + " megahertz" + rest
	})
}
