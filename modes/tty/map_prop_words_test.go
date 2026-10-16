package tty

// map_prop_words_test.go — 0.19.0 W6.1: the Propagation mode's words
// without the picture, in D-136's order, written for speech (FR-3.1, FR-3.2,
// FR-3.5 to FR-3.7).

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/render"
)

// propWordsFrom are the description's lines from the Propagation mode's
// first on: the place's own heading and the map's lines before it aside.
func propWordsFrom(t *testing.T, d Dashboard) []string {
	t.Helper()
	all := d.describeLinesAll()
	for i, l := range all {
		if strings.HasPrefix(l, "The Propagation mode") {
			return all[i:]
		}
	}
	t.Fatalf("the words have no Propagation mode line:\n%s", strings.Join(all, "\n"))
	return nil
}

// lineWith is the index of the first line holding the text, or -1.
func lineWith(lines []string, text string) int {
	for i, l := range lines {
		if strings.Contains(l, text) {
			return i
		}
	}
	return -1
}

// heldDash is stationDash with a frequency held and the centre answered.
func heldDash(t *testing.T) Dashboard {
	t.Helper()
	d := stationDash(t)
	d.mapPane.propFreqMHz, d.mapPane.propFreqLabel = 7.2, "7.2 MHz"
	return workCentre(t, d)
}

// TestTheWordsFollowTheRuledOrder is D-136: the origin and its MUF and
// foF2; day or night there; the status block; the best bands, one line per
// target; the frequency held; the centre.
func TestTheWordsFollowTheRuledOrder(t *testing.T) {
	words := propWordsFrom(t, heldDash(t))
	order := []string{
		"Oceanside, CA (no transmitter set), now: MUF(3000) 18.0 megahertz",
		"It is night at Oceanside, CA (no transmitter set); the sun rises there at ",
		"Status: ",
		"Local, about 400 kilometres",
		"North America: ",
		"Oceania: ",
		"7.2 megahertz held",
		"The map's centre",
	}
	at := -1
	for _, want := range order {
		i := lineWith(words, want)
		if i <= at {
			t.Fatalf("%q is at line %d, not after line %d:\n%s", want, i, at, strings.Join(words, "\n"))
		}
		at = i
	}
}

// TestThePropagationWordsAnswerWithoutThePicture is FR-3.1 (D-64, D-127):
// the words say the origin's answer wherever the view is, and each target's
// bands one line each, its name first, the open before the closed.
func TestThePropagationWordsAnswerWithoutThePicture(t *testing.T) {
	d := chartDash(t)
	d.mapDesc = mapDescInstead
	words := propWordsFrom(t, d)
	if lineWith(words, ", now: MUF(3000) 18.0 megahertz, foF2 6.0 megahertz.") < 0 {
		t.Errorf("the origin's answer is not said:\n%s", strings.Join(words, "\n"))
	}
	names, _ := d.propChartRows()
	for _, n := range names {
		i := lineWith(words, spoken(n)+": ")
		if i < 0 {
			t.Errorf("no line starts with the target %q:\n%s", n, strings.Join(words, "\n"))
			continue
		}
		line := words[i]
		if !strings.HasPrefix(line, spoken(n)+": ") {
			t.Errorf("the target's line does not start with its name: %q", line)
		}
		if open, closed := strings.Index(line, "open on "), strings.Index(line, "above the upper limit"); open >= 0 && closed >= 0 && open > closed {
			t.Errorf("a target's closed bands are said before its open: %q", line)
		}
	}
	if na := lineWith(words, "North America: open on "); na < 0 || !strings.Contains(words[na], "20 metres;") {
		t.Errorf("North America, open on 20 m in the fixture, is not said so:\n%s", strings.Join(words, "\n"))
	}
}

// TestPropagationWordsCarryUnitsForSpeech is FR-3.2 (D-34, AX-K): no
// "MHz", no "40m", no "~" or "km"; megahertz and metres written out, and
// the layer's full name said once.
func TestPropagationWordsCarryUnitsForSpeech(t *testing.T) {
	words := strings.ReplaceAll(strings.Join(propWordsFrom(t, heldDash(t)), "\n"), propFocusLimit, "") // its release number is no figure
	for _, bad := range []*regexp.Regexp{
		regexp.MustCompile(`MHz`), regexp.MustCompile(`\b\d+m\b`), regexp.MustCompile(`~`),
		regexp.MustCompile(`\bkm\b`), regexp.MustCompile(`%`), regexp.MustCompile(`°`),
		regexp.MustCompile(`\d\.\d(,|\.|;|$| [^dmkp])`), // a figure with no unit after it
	} {
		if m := bad.FindString(words); m != "" {
			t.Errorf("the words carry %q, not written for speech:\n%s", m, words)
		}
	}
	for _, want := range []string{"megahertz", " metres", "kilometres"} {
		if !strings.Contains(words, want) {
			t.Errorf("the words do not say %q:\n%s", want, words)
		}
	}
	if n := strings.Count(words, "the maximum usable frequency over 3,000 kilometres"); n != 1 {
		t.Errorf("the layer's full name is said %d times; once:\n%s", n, words)
	}
}

// TestSpokenWritesTheFiguresOut is FR-3.2: the windows' short forms, as the
// words say them.
func TestSpokenWritesTheFiguresOut(t *testing.T) {
	for in, want := range map[string]string{
		"usable 160m-60m · foF2 6.0 MHz":      "usable 160 to 60 metres, foF2 6.0 megahertz",
		"best ~5.1 MHz (80m), 85% of the MUF": "best about 5.1 megahertz (80 metres), 85 percent of the MUF",
		"33.0°N 117.5°W; MUF(3000) 18.0":      "33.0 degrees north 117.5 degrees west; MUF(3000) 18.0 megahertz",
		"Local, ~400 km":                      "Local, about 400 kilometres",
		"data 1 min old":                      "data 1 minute old",
		"STALE: data 2 h 5 min old":           "STALE: data 2 hours 5 minutes old",
		"a station 250 mi away":               "a station 250 miles away",
	} {
		if got := spoken(in); got != want {
			t.Errorf("spoken(%q) = %q; want %q", in, got, want)
		}
	}
}

// TestTheStatusIsSaidWithoutThePicture is FR-3.5 and FR-5.2: asking, a
// failure with the key that asks again, the age and the fallback are all
// said in the words; a failed refresh is said over the field it kept.
func TestTheStatusIsSaidWithoutThePicture(t *testing.T) {
	d := propDash(t, &seam{}, true)
	d.cfg.ASCII = true
	d = pressMap(t, d, "P")
	d.mapPane.propBusy, d.mapPane.propSnap = true, nil
	if words := strings.Join(propWordsFrom(t, d), "\n"); !strings.Contains(words, "asking NOAA for the newest grid") {
		t.Errorf("asking is not said:\n%s", words)
	}
	d.mapPane.propBusy, d.mapPane.propErr = false, errors.New("ionomaps: NOAA not reached")
	if words := strings.Join(propWordsFrom(t, d), "\n"); !strings.Contains(words, "NOAA not reached") || !strings.Contains(words, "U refreshes now") {
		t.Errorf("a failure is not said with its path:\n%s", words)
	}
	held, _ := drawnDash(t)
	held.mapPane.propErr = errors.New("ionomaps: NOAA not reached")
	words := propWordsFrom(t, held)
	status := words[lineWith(words, "Status: ")]
	for _, want := range []string{"data 30 minutes old", "the last update failed: NOAA not reached", "U refreshes now", "foF2 from GloTEC"} {
		if !strings.Contains(status, want) {
			t.Errorf("the status does not say %q: %q", want, status)
		}
	}
	held.mapPane.propErr = context.Canceled
	if w := strings.Join(propWordsFrom(t, held), "\n"); strings.Contains(w, "failed") {
		t.Errorf("an update ended by leaving the mode is said as failed:\n%s", w)
	}
	held.mapPane.propErr = errors.New("ionomaps: NOAA not reached")
	if !strings.Contains(stripANSITest(held.mapStatusLine()), "the last update failed") {
		t.Errorf("a failed refresh over a field kept is not said on the status line: %q", stripANSITest(held.mapStatusLine()))
	}
}

// refreshed is drawnDash with a second update applied, five minutes on.
func refreshed(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	later := drawnAt.Add(5 * time.Minute)
	d.now = func() time.Time { return later }
	m, _ := d.Update(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: propSnapAt(later, 18)}})
	return m.(Dashboard)
}

// TestARefreshKeepsTheReadersPlace is FR-3.6 (D-127): a refresh keeps the
// window's scroll and the menu's focus.
func TestARefreshKeepsTheReadersPlace(t *testing.T) {
	d, _ := drawnDash(t)
	d.mapDesc = mapDescInstead
	d.modalScroll, d.mapPane.menuAt = 3, 2
	d = refreshed(t, d)
	if d.modalScroll != 3 || d.mapPane.menuAt != 2 {
		t.Errorf("after a refresh the scroll is %d and the menu's focus %d; 3 and 2 kept", d.modalScroll, d.mapPane.menuAt)
	}
}

// TestARefreshIsSaidOnce is FR-3.6 (D-127): the words say once that they
// were updated, and when - not on the first answer, and in one line after a
// refresh; GIRO's stations following NOAA's field at once are no refresh.
func TestARefreshIsSaidOnce(t *testing.T) {
	d, _ := drawnDash(t)
	if w := strings.Join(propWordsFrom(t, d), "\n"); strings.Contains(w, "updated at") {
		t.Errorf("the first answer says it was updated:\n%s", w)
	}
	d = refreshed(t, d)
	w := strings.Join(propWordsFrom(t, d), "\n")
	if n := strings.Count(w, "updated at "+d.clockFmt.Time(drawnAt.Add(5*time.Minute))); n != 1 {
		t.Errorf("after a refresh \"updated at\" is said %d times; once:\n%s", n, w)
	}
	first := propDash(t, &seam{}, true)
	first.now = func() time.Time { return drawnAt }
	first = pressMap(t, first, "P")
	for _, snap := range []ionomaps.Snapshot{stationsSnap(0, ionomaps.StationsNext), stationsSnap(12, ionomaps.Fresh)} {
		m, _ := first.Update(propUpdatedMsg{gen: first.mapPane.propGen, res: PropagationResult{Snapshot: snap}})
		first = m.(Dashboard)
	}
	if w := strings.Join(propWordsFrom(t, first), "\n"); strings.Contains(w, "updated at") {
		t.Errorf("GIRO's stations following NOAA's field at once are said as a refresh:\n%s", w)
	}
}

// TestTheFocusLimitIsSaid is FR-3.7 (D-133): Help says a screen reader
// cannot yet follow focus in the Propagation mode, and that 0.19.1 fixes
// it; the words say it on opening, and not again after a refresh.
func TestTheFocusLimitIsSaid(t *testing.T) {
	d, _ := drawnDash(t)
	if lineWith(propWordsFrom(t, d), propFocusLimit) < 0 {
		t.Errorf("the words on opening do not say the focus limit")
	}
	for _, s := range []Surface{SurfaceObserver, SurfaceBroadcaster} {
		d.surface = s
		if help := strings.Join(d.helpLines(render.Opts{Width: 200}), "\n"); !strings.Contains(help, propFocusLimit) {
			t.Errorf("Help on surface %v does not say the focus limit", s)
		}
	}
	d.surface = SurfaceObserver
	d = refreshed(t, d)
	if lineWith(propWordsFrom(t, d), propFocusLimit) >= 0 {
		t.Errorf("the focus limit is said again after a refresh")
	}
	d, _ = keyCmd(t, d, "P") // left, and opened again
	d, cmd := keyCmd(t, d, "P")
	if d = runProp(t, d, cmd); lineWith(propWordsFrom(t, d), propFocusLimit) < 0 {
		t.Errorf("the focus limit is not said on opening the mode again")
	}
	if !strings.Contains(propFocusLimit, "0.19.1") {
		t.Errorf("the focus limit does not say what fixes it: %q", propFocusLimit)
	}
}

// TestCaveatsAreSaidOncePerOpen is D-136: the status block's caveats - the
// offset, a disagreement with the typical day, the origin's station - are
// said once, in it; a target's line carries only its own "about typical".
func TestCaveatsAreSaidOncePerOpen(t *testing.T) {
	d := chartDash(t)
	words := propWordsFrom(t, d)
	status := lineWith(words, "Status: ")
	if status < 0 {
		t.Fatalf("no status block:\n%s", strings.Join(words, "\n"))
	}
	for i, l := range words {
		if i == status {
			continue
		}
		for _, caveat := range []string{" old", "typical offset", "differs from a typical day", "computed at"} {
			if strings.Contains(l, caveat) {
				t.Errorf("line %d says the status block's %q again: %q", i, caveat, l)
			}
		}
	}
	names, answers := d.propChartRows()
	for i, n := range names {
		line := words[lineWith(words, spoken(n)+": ")]
		far := !answers[i].Nearest.Known || answers[i].Nearest.Km >= propFarKm
		if says := strings.Contains(line, "about as good as typical"); says != far {
			t.Errorf("%s's line says about typical %v; its station is far %v: %q", n, says, far, line)
		}
	}
}

// TestTheDayAndItsTurnAreSaid is FR-3.1 (AX-E): whether it is day or night
// at the origin and when the sun next rises or sets there; where it does
// neither, that is said.
func TestTheDayAndItsTurnAreSaid(t *testing.T) {
	d := chartDash(t)
	place := propPlace{at: ionomaps.LatLon{Lat: 33.2, Lon: -117.38}, name: "Oceanside", ok: true}
	if w := d.propDayWords(place, time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)); !strings.HasPrefix(w, "It is day at Oceanside; the sun sets there at ") {
		t.Errorf("Oceanside at its noon: %q", w)
	}
	pole := propPlace{at: ionomaps.LatLon{Lat: 89}, name: "the pole", ok: true}
	if w := d.propDayWords(pole, time.Date(2026, 12, 21, 12, 0, 0, 0, time.UTC)); w != "It is night at the pole, and the sun neither rises nor sets there today." {
		t.Errorf("the pole at the solstice: %q", w)
	}
}

// TestTheBandsAreSaidOpenFirst is D-127: a target's bands by state, the
// open first, then each closed state; with none open, that is said first.
func TestTheBandsAreSaidOpenFirst(t *testing.T) {
	bands := make([]ionomaps.BandStatus, len(hfBands))
	for i := range bands {
		bands[i].Status = ionomaps.AboveUpper
	}
	if got := propBandsSaid(bands); got != "nothing open; above the upper limit on 160, 80, 60, 40, 30, 20, 17, 15, 12 and 10m" {
		t.Errorf("every band above the upper limit: %q", got)
	}
	bands[0].Status, bands[2].Status, bands[5].Status = ionomaps.Absorbed, ionomaps.Disturbed, ionomaps.Open
	if got := propBandsSaid(bands); !strings.HasPrefix(got, "open on 20m; disturbed on 60m; absorbed on 160m; above the upper limit on 80, 40, 30") {
		t.Errorf("open, disturbed and closed bands: %q", got)
	}
	if got := listWords([]string{"a"}); got != "a" {
		t.Errorf("one item listed: %q", got)
	}
}

// TestEachTargetSaysItsStation is D-116 and D-198: each target's line says
// its nearest station's distance; with none, or beyond the far distance,
// that the answer is about as good as typical; an unscored one, to verify
// on air.
func TestEachTargetSaysItsStation(t *testing.T) {
	d := chartDash(t)
	for _, c := range []struct {
		n    ionomaps.Nearest
		want string
	}{
		{ionomaps.Nearest{}, "no reporting station, about as good as typical"},
		{ionomaps.Nearest{Known: true, Km: 300}, "nearest station " + d.roundDistance(300) + " away"},
		{ionomaps.Nearest{Known: true, Km: propFarKm}, "nearest station " + d.roundDistance(propFarKm) + " away, about as good as typical"},
		{ionomaps.Nearest{Known: true, Km: 300, Unscored: true}, "nearest station " + d.roundDistance(300) + " away, unscored: verify on air"},
	} {
		if got := d.propStationWords(c.n); got != c.want {
			t.Errorf("%+v: %q; want %q", c.n, got, c.want)
		}
	}
}
