package tty

import (
	"github.com/branden-thompson/watchpost/platform/propanswers"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// chartDash is the map in the Propagation mode with a snapshot drawn and the
// chart's answers worked out, as the command the Update armed works them.
func chartDash(t *testing.T) Dashboard {
	t.Helper()
	d, _ := drawnDash(t)
	return workChart(t, d)
}

// settled is the dashboard once its view has stood still: the settle tick
// the harness dropped, delivered (P-1: an answer from or to the centre waits
// for it).
func settled(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	if d.mapPane.viewGen == d.mapPane.viewAsked || d.modal != modalMap {
		return d
	}
	m, _ := d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen})
	return m.(Dashboard)
}

// workChart runs the chart's command and applies it: the one the last
// Update armed, which the harness that fed it dropped, asked again.
func workChart(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	d = settled(t, d)
	d.mapPane.propChartAsked = propChartKey{}
	d, cmd := d.armPropChart()
	if cmd == nil {
		return d
	}
	m, _ := d.Update(cmd())
	return m.(Dashboard)
}

// TestTheContinentsAreTheDefaultTargets is FR-1.9 (D-25, A-45): with no
// place found, the chart answers for each continent's reference point, every
// band with a status that names the path's MUF; the block under the map
// names each continent's highest open band, and the key to the chart.
func TestTheContinentsAreTheDefaultTargets(t *testing.T) {
	d := chartDash(t)
	c := d.mapPane.propChart
	if c.err != nil || len(c.targets) != len(propanswers.Continents) {
		t.Fatalf("the chart holds %d targets (%v); want the %d continents", len(c.targets), c.err, len(propanswers.Continents))
	}
	for i, a := range c.targets {
		bands := a.Bands
		if len(bands) != len(hfBands) {
			t.Fatalf("%s has %d bands; want %d", propanswers.Continents[i].Name, len(bands), len(hfBands))
		}
		if l := bands[0].Limit; l != ionomaps.LimitMUF && l != ionomaps.LimitAbsorption && l != ionomaps.LimitCircuit && !propanswers.Continents[i].Near(c.key.origin, c.key.radiusKm) {
			t.Errorf("%s's 160m names limit %v; a path's is its MUF, or the lower limit's (D-70)", propanswers.Continents[i].Name, l)
		}
	}
	block := stripANSITest(strings.Join(d.propBlock(d.scrubW()), "\n"))
	for _, want := range []string{"Up to:", "NA ", "SA ", "EU ", "AF ", "AS ", "OC ", "B chart"} {
		if !strings.Contains(block, want) {
			t.Errorf("the block under the map does not say %q:\n%s", want, block)
		}
	}
}

// TestBestBandsNameTheirLimit is FR-1.15 (D-73, D-187): the chart window
// says the reference circuit and its noise, the near-vertical area's limit is
// foF2 and each target's the path's MUF, each with its value.
func TestBestBandsNameTheirLimit(t *testing.T) {
	d := pressMap(t, chartDash(t), "B")
	if d.modal != modalPropChart {
		t.Fatalf("B opened window %v; want the chart (D-176)", d.modal)
	}
	text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n"))
	for _, want := range []string{"100 W, 0 dBi, +13 dB · residential noise", "Local, ~400 km", "foF2 6.0", "North America", "Oceania", "MUF ", "figures in MHz"} {
		if !strings.Contains(text, want) {
			t.Errorf("the chart does not say %q:\n%s", want, text)
		}
	}
	m, _ := d.Update(mapKeyMsg(t, "esc"))
	if back := m.(Dashboard); back.modal != modalMap {
		t.Errorf("Esc left window %v; the chart closes back to the map", back.modal)
	}
}

// TestTheDaysOpenHoursAreShown is FR-1.15 (D-135): the selected target's
// bands by the UTC day's hours, each hour open in its typical day drawn
// open; the arrows move the target the day is drawn for.
func TestTheDaysOpenHoursAreShown(t *testing.T) {
	d := pressMap(t, chartDash(t), "B")
	day := d.propDayRows(d.mapPane.propChart.local.Bands)
	if len(day) != len(hfBands) {
		t.Fatalf("the typical day has %d rows; want one a band", len(day))
	}
	for i, b := range d.mapPane.propChart.local.Bands {
		cells := []rune(stripANSITest(day[i]))
		cells = cells[len(cells)-47:] // 24 hours, a space between each
		for h := range 24 {
			at := d.mapPane.propChart.key.hour.UTC().Truncate(24 * time.Hour).Add(time.Duration(h) * time.Hour)
			if want := propanswers.OpenAt(b.OpenHours, at); (cells[2*h] == propOpenGlyph) != want {
				t.Errorf("%s at %02d UTC is drawn %q; typically open: %v", hfBands[i].name, h, cells[2*h], want)
			}
		}
	}
	m, _ := d.Update(mapKeyMsg(t, "down"))
	if got := m.(Dashboard).mapPane.propChartRow; got != 1 {
		t.Errorf("down moved the chart's target to %d; want 1, the first continent", got)
	}
}

// TestTheChartSaysItsMeasuredAgreement is D-330: the chart says how often its
// answers agreed with what WSPR heard, the climatology's beside them, in
// either glyph set and within the window's width.
func TestTheChartSaysItsMeasuredAgreement(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		d := pressMap(t, chartDash(t), "B")
		d.cfg.ASCII = ascii
		text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n"))
		for _, want := range []string{"agrees with WSPR reports", "Local 54%", "continents 68%", "typical alone 52%, 64%"} {
			if !strings.Contains(text, want) {
				t.Errorf("ascii %v: the chart does not say %q:\n%s", ascii, want, text)
			}
		}
	}
	if w, legend := utf8.RuneCountInString(propAgreementLine), utf8.RuneCountInString(propShadeLegend(false)); w > legend {
		t.Errorf("the agreement line is %d columns, wider than the shade legend's %d, the widest line the window already holds", w, legend)
	}
}

// TestTheTypicalDayIsLabelled is D-135: the hours come from the climatology,
// and the chart says so.
func TestTheTypicalDayIsLabelled(t *testing.T) {
	d := pressMap(t, chartDash(t), "B")
	if text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n")); !strings.Contains(text, "TYPICAL DAY") {
		t.Errorf("the chart's hours are not labelled a typical day:\n%s", text)
	}
}

// TestADisagreementWithTheTypicalDayIsSaid is D-135: where a band's status
// now differs from its typical day at this hour, the chart says "now differs
// from a typical day" for that target; where they agree it does not.
func TestADisagreementWithTheTypicalDayIsSaid(t *testing.T) {
	d := pressMap(t, chartDash(t), "B")
	c := &d.mapPane.propChart
	hour := c.key.hour.UTC().Truncate(time.Hour)
	agree := make([]ionomaps.BandStatus, len(hfBands))
	for i := range agree {
		agree[i] = ionomaps.BandStatus{Band: ionomaps.AllBands()[i], Status: ionomaps.AboveUpper, Limit: ionomaps.LimitFoF2}
	}
	c.local = ionomaps.Answer{Bands: agree}
	if text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n")); strings.Contains(text, "now differs") {
		t.Errorf("every band closed now and in its typical day, yet a difference is said:\n%s", text)
	}
	differ := append([]ionomaps.BandStatus(nil), agree...)
	differ[0].OpenHours = []ionomaps.Span{{From: hour, To: hour.Add(time.Hour)}} // typically open this hour, closed now
	c.local = ionomaps.Answer{Bands: differ}
	if text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n")); !strings.Contains(text, "now differs from a typical day") {
		t.Errorf("160m is typically open this hour and closed now, yet no difference is said:\n%s", text)
	}
}

// TestTheOriginFollowsTheUIMode is FR-1.8 as D-193 amends it: on Observer
// the chart answers from the map's centre, to half a degree, named for the
// view, and a pan asks again; on the Broadcaster from its transmitter, or,
// with none set, the selected place, said; a UI mode added later must rule
// its own (TestEveryUIModeHasAPropagationRole).
func TestTheOriginFollowsTheUIMode(t *testing.T) {
	d := chartDash(t)
	centre, _ := d.propCentreAt()
	if o := d.mapPane.propChart.key.origin; o != centre || d.mapPane.propOrigin.name != d.viewName() {
		t.Fatalf("Observer's chart rows from %v (%q); the centre is %v (%q)", o, d.mapPane.propOrigin.name, centre, d.viewName())
	}
	moved := pressMap(t, d, "right")
	moved = settled(t, moved.renderMap()) // asked once the view stands still (P-1)
	if _, cmd := moved.armPropChart(); cmd == nil && moved.mapPane.propChartAsked == d.mapPane.propChart.key {
		t.Error("the map panned and Observer's chart was not asked again")
	}
	if _, asked := workCentre(t, d).propCentreWanted(); asked || strings.Contains(stripANSITest(strings.Join(workCentre(t, d).mapBodyLines(), "\n")), "┌─ Centre") {
		t.Error("Observer's centre is its origin, yet it is asked and drawn as a target")
	}
	st := stationDash(t)
	loc := st.selectedLocation()
	if o := st.mapPane.propOrigin; o.at.Lat != loc.Lat || o.at.Lon != loc.Lon || !strings.HasSuffix(o.name, bcNoTransmitter) {
		t.Errorf("with no transmitter the station's origin is %+v; the selected place, said", o)
	}
	st.cfg.Transmitter = &snapshot.LocationRef{Label: "KXYZ tower", Lat: 40.1, Lon: -105.2}
	st = st.renderMap()
	if o := st.mapPane.propOrigin; o.at != (ionomaps.LatLon{Lat: 40.1, Lon: -105.2}) || o.name != "KXYZ tower" {
		t.Errorf("with a transmitter the station's origin is %+v; the transmitter", o)
	}
}

// TestEveryUIModeHasAMapRole is D-193's and D-195's table walked: every UI
// mode rules whose place its map is on and where its Propagation answers come
// from, so a mode added later (Analyst, D-27) cannot inherit them silently;
// an unruled one is Observer's.
func TestEveryUIModeHasAMapRole(t *testing.T) {
	for s := Surface(0); s < numSurfaces; s++ {
		if r := mapRoles[s]; r.place == mapPlaceUnruled || r.prop.origin == propOriginUnruled {
			t.Errorf("UI mode %d has no map role: rule its place and Propagation origin in mapRoles (D-193, D-195)", s)
		}
	}
	if (Dashboard{surface: numSurfaces}).mapRole() != mapRoles[SurfaceObserver] {
		t.Error("an unruled UI mode does not fall back to Observer's role")
	}
	if (Dashboard{surface: numSurfaces}).propRole().origin != propFromCentre {
		t.Error("an unruled UI mode does not explore from the centre")
	}
}

// TestAnswersNeverBlockTheUI is FR-1.17 (A-27): the chart's answers are
// worked out in a command, never in Update: a new snapshot leaves the chart
// as it was until the command's result arrives.
func TestAnswersNeverBlockTheUI(t *testing.T) {
	d, _ := drawnDash(t)
	if len(d.mapPane.propChart.targets) != 0 {
		t.Fatal("the chart was worked out inside Update")
	}
	if _, ok := d.propChartWanted(); !ok || d.mapPane.propChartAsked == (propChartKey{}) {
		t.Fatal("the Update that kept the snapshot did not ask for the chart")
	}
	d.mapPane.propChartAsked = propChartKey{} // the harness dropped that command: ask again
	d, cmd := d.armPropChart()
	if cmd == nil {
		t.Fatal("no command works the chart out")
	}
	if _, again := d.armPropChart(); again != nil {
		t.Error("the chart was asked twice for the same snapshot and place")
	}
	older := propChartMsg{chart: propChart{key: propChartKey{radiusKm: 1}, targets: make([]ionomaps.Answer, len(propanswers.Continents))}}
	if m, _ := d.Update(older); len(m.(Dashboard).mapPane.propChart.targets) != 0 {
		t.Error("an answer to an older ask was kept over the one asked for")
	}
}

// TestTheNightIsDrawnOnThePropagationMap is go-tuiMaps P4 in the mode
// (L-4.1 to L-4.3, D-174): the terminator and the night side are handed in
// with the field, for the moment shown, whatever the Day / night switch
// (D-199).
func TestTheNightIsDrawnOnThePropagationMap(t *testing.T) {
	d, _ := drawnDash(t)
	if d.detailOn(nightDetailKey) {
		t.Fatal("control: Day / night is on, so this proves nothing about the mode drawing it always")
	}
	var line, night bool
	for _, o := range d.mapPane.nightGiven {
		for _, f := range o.Features {
			line = line || f.Role == terminatorRole
			night = night || f.Role == nightRole
		}
	}
	if !line || !night {
		t.Errorf("the Propagation map hands in the terminator %v and the night %v; both, for day and night (D-174)", line, night)
	}
}

// TestTheNearVerticalRadiusIsASetting is FR-1.15 (D-100, A-18, D-121): the
// Maps tab offers 200, 400 or 600 km, 400 by default, the Observer's alone;
// the block, the chart and the answers follow it, and it is saved.
func TestTheNearVerticalRadiusIsASetting(t *testing.T) {
	if r := setupTable()[rowMapPropRadius]; r.group != groupMap || r.scope != scopeObserver || r.kind != rowPicker {
		t.Fatalf("the radius row is %+v; want a Maps picker, the Observer's (D-121)", r)
	}
	if propRadiusByKm(0) != 400 || propRadiusByKm(250) != 400 || propRadiusByKm(600) != 600 {
		t.Error("the file's radius is read wrongly: 400 by default, any value not offered the default")
	}
	d := chartDash(t)
	seen := map[int]bool{}
	for range propRadiusChoices {
		d = d.cyclePropRadius(true)
		seen[d.mapPropRadiusKm] = true
	}
	if len(seen) != 3 || !seen[200] || !seen[400] || !seen[600] {
		t.Errorf("the picker goes round %v; want 200, 400 and 600 km", seen)
	}
	for d.mapPropRadiusKm != 600 {
		d = d.cyclePropRadius(true)
	}
	d = workChart(t, d)
	if got := d.mapPane.propChart.key.radiusKm; got != 600 {
		t.Errorf("the chart rows for %v km; the Setting says 600", got)
	}
	block := stripANSITest(strings.Join(d.propBlock(d.scrubW()), "\n"))
	if !strings.Contains(block, "Local, to ~600 km (NVIS)") {
		t.Errorf("the block does not say the Setting's radius:\n%s", block)
	}
	if p := d.uiForSave(); p.MapPropRadius != 600 {
		t.Errorf("the radius saved is %d; want 600", p.MapPropRadius)
	}
}

// TestTheChartIsReadableInASCII is BUILD red team AX-4: under --ascii the
// chart draws letters a screen reader says - O, D, A, W, a dash, > and v - and
// none of the circles, pointer or arrow a reader names by their Unicode
// names; the words say the local area's typical day, its open hours in UTC.
func TestTheChartIsReadableInASCII(t *testing.T) {
	d := chartDash(t)
	d.cfg.ASCII = true
	o := d.opts()
	chart := stripANSITest(strings.Join(d.propChartLines(o), "\n"))
	for _, g := range []string{"●", "◐", "○", "◌", "·", "▸", "▼"} {
		if strings.Contains(chart, g) {
			t.Errorf("under --ascii the chart draws %q:\n%s", g, chart)
		}
	}
	for _, want := range []string{"O open", "D disturbed (D-RAP)", "A absorbed", "W weak", "- above the upper limit", o.Glyphs().Pointer + " "} {
		if !strings.Contains(chart, want) {
			t.Errorf("under --ascii the chart does not draw %q:\n%s", want, chart)
		}
	}
	if !strings.Contains(chart, "Row") || !strings.Contains(chart, "Close") {
		t.Errorf("the chart window has no controls line (AX-6):\n%s", chart)
	}
	words := propWordsFrom(t, d)
	typical := lineWith(words, "On a typical day")
	if typical < 0 || typical != lineWith(words, "Local, about")+1 {
		t.Fatalf("the local area's typical day is not said after its line:\n%s", strings.Join(words, "\n"))
	}
	if !regexp.MustCompile(`\d+ metres \d\d:\d\d to \d\d:\d\d`).MatchString(words[typical]) {
		t.Errorf("the typical day does not say a band's open hours: %q", words[typical])
	}
}

// TestARebuiltSnapshotIsAnsweredAgain is the BUILD red team's L-F3: the
// no-readings correction or the stations switched rebuild the snapshot at
// the same moment, and the chart, the reach and the centre are worked out
// again for it - never shown from the snapshot they were worked out for.
func TestARebuiltSnapshotIsAnsweredAgain(t *testing.T) {
	d := chartDash(t)
	d.mapPane.propFreqMHz, d.mapPane.propFreqLabel = 7.2, "7.2 MHz"
	before := d.mapPane.propChart.key
	rebuilt := propSnapAt(d.mapPane.propSnap.Computed, 25) // the same moment, other values
	m, _ := d.Update(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: rebuilt}, correct: !d.mapPane.propSnapCorrect})
	d = m.(Dashboard)
	if !d.mapPane.propSnap.Computed.Equal(before.made.computed) {
		t.Fatal("control: the rebuilt snapshot is not of the same moment")
	}
	if d.propChartShown() {
		t.Error("the chart worked out for the snapshot before is shown for the rebuilt one")
	}
	if key, ok := d.propReachWanted(); ok && key == d.mapPane.propReach.key && d.mapPane.propReach.key != (propReachKey{}) {
		t.Error("the reach worked out for the snapshot before is the one wanted for the rebuilt one")
	}
	d = workChart(t, d)
	if !d.propChartShown() || d.mapPane.propChart.key == before {
		t.Errorf("the rebuilt snapshot's chart is not worked out: %+v", d.mapPane.propChart.key)
	}
}

// TestAPanAsksOnceItSettles is the BUILD red team's P-1: on the Observer the
// chart answers for the map's centre, so a pan held down would ask it again
// at every half degree; it is asked once the view has stood still.
func TestAPanAsksOnceItSettles(t *testing.T) {
	d := chartDash(t)
	for range 4 {
		m, _ := d.Update(mapKeyMsg(t, "right"))
		d = m.(Dashboard)
	}
	d.mapPane.propChartAsked = propChartKey{} // what the harness's dropped commands had asked
	if _, cmd := d.armPropChart(); cmd != nil {
		t.Error("the chart was asked while the view was still moving")
	}
	m, _ := d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen})
	d = m.(Dashboard)
	d.mapPane.propChartAsked = propChartKey{}
	if _, cmd := d.armPropChart(); cmd == nil {
		t.Error("the chart was not asked once the view stood still")
	}
}

// TestARefreshAnsweredTooSoonKeepsItsAnswers is the BUILD red team's P-4: a
// refresh answered too soon hands back the snapshot held - its moment and its
// options - and its answers and hours ahead stand, never worked out again; a
// snapshot rebuilt with other options is another.
func TestARefreshAnsweredTooSoonKeepsItsAnswers(t *testing.T) {
	d := stepAhead(t, chartDash(t), "shift+right")
	if len(d.mapPane.propAheadSnaps) == 0 {
		t.Fatal("control: no hour ahead is worked out")
	}
	built := d.mapPane.propBuilt
	again := *d.mapPane.propSnap
	again.Reason = ionomaps.TooSoon
	m, _ := d.Update(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: again}, correct: d.mapPane.propSnapCorrect})
	d = m.(Dashboard)
	if d.mapPane.propBuilt != built || len(d.mapPane.propAheadSnaps) == 0 {
		t.Errorf("a refresh answered too soon counted as another snapshot (built %d, was %d) or dropped the hours ahead (%d)", d.mapPane.propBuilt, built, len(d.mapPane.propAheadSnaps))
	}
	m, _ = d.Update(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: again}, correct: !d.mapPane.propSnapCorrect})
	if d = m.(Dashboard); d.mapPane.propBuilt == built || len(d.mapPane.propAheadSnaps) != 0 {
		t.Error("a snapshot rebuilt with the correction switched is kept as the one held")
	}
}

// TestAPanSettledUnderAWindowIsAnsweredOnReturn is P-1's settle without its
// trap: a pan, then a window over the map before the view settles - the
// frequency entry here - leaves the settle tick waiting, and back at the
// map the view is answered, never left "being worked out" until the next
// pan.
func TestAPanSettledUnderAWindowIsAnsweredOnReturn(t *testing.T) {
	d := pressMap(t, chartDash(t), "right")
	gen := d.mapPane.viewGen
	d = pressMap(t, d, "F")
	if d.modal != modalPropFreq {
		t.Fatalf("control: F opened window %v", d.modal)
	}
	m, cmd := d.Update(mapViewSettledMsg{gen: gen})
	d = m.(Dashboard)
	if cmd == nil || d.mapPane.viewAsked == gen {
		t.Fatal("a settle tick under the window was dropped, or asked for a view the map does not show")
	}
	m, _ = d.Update(mapKeyMsg(t, "esc"))
	d = m.(Dashboard)
	if d.modal != modalMap {
		t.Fatalf("control: esc left window %v", d.modal)
	}
	m, _ = d.Update(mapViewSettledMsg{gen: gen}) // the tick it waited with
	d = m.(Dashboard)
	d.mapPane.propChartAsked = propChartKey{}
	if _, ask := d.armPropChart(); d.mapPane.viewAsked != gen || ask == nil {
		t.Errorf("back at the map the view is not answered: asked %d of %d, chart asked %v", d.mapPane.viewAsked, gen, ask != nil)
	}
}

// TestEachBandIsFoundByItsName is the BUILD red team's L-F8: a row's glyphs
// sit under the bands they name, whatever order the answer gives them in,
// and a band the answer lacks is blank; the typical day's rows are named by
// their bands; the bands and their edges are go-ionomaps' own.
func TestEachBandIsFoundByItsName(t *testing.T) {
	var d Dashboard
	cells := stripANSITest(d.propBandCells([]ionomaps.BandStatus{{Band: ionomaps.Band20m, Status: ionomaps.Open}}))
	if len([]rune(cells)) != 3*len(hfBands) {
		t.Fatalf("the row is %d cells for %d bands: %q", len([]rune(cells)), len(hfBands), cells)
	}
	for i, hb := range hfBands {
		cell := strings.TrimSpace(string([]rune(cells)[3*i : 3*i+3]))
		if (hb.name == "20m") != (cell != "") {
			t.Errorf("under %s the row has %q; only 20m answered", hb.name, cell)
		}
	}
	if len(hfBands) != len(ionomaps.AllBands()) || hfBands[6].mhz != "18.07" || hfBands[0].name != ionomaps.Band160m.String() {
		t.Errorf("the bands are not go-ionomaps' own: %+v", hfBands)
	}
	rows := d.propDayRows([]ionomaps.BandStatus{{Band: ionomaps.Band40m}, {Band: ionomaps.Band10m}})
	if !strings.HasPrefix(rows[0], "40m") || !strings.HasPrefix(rows[1], "10m") {
		t.Errorf("the typical day's rows are not named by their bands: %q", rows)
	}
}

// TestTheChartsTargetKeysAreRebindable is FR-1.14 (D-256): moving between
// the chart's targets is a keymap action of the chart window's own scope,
// up and down by default, rebound in [keys] like every other control in
// the mode, and Help lists the keys as bound.
func TestTheChartsTargetKeysAreRebindable(t *testing.T) {
	d := pressMap(t, chartDash(t), "B")
	if d.modal != modalPropChart {
		t.Fatalf("B opened window %v; want the chart", d.modal)
	}
	keys, err := chartKeysFrom(term.KeyMap{actChartNext: {Keys: []string{"j"}}, actChartPrev: {Keys: []string{"k"}}, actMapPanUp: {Keys: []string{"w"}}})
	if err != nil {
		t.Fatal(err)
	}
	d.chartKeys = keys
	press := func(key string) int {
		m, _ := d.Update(mapKeyMsg(t, key))
		d = m.(Dashboard)
		return d.mapPane.propChartRow
	}
	if row := press("j"); row != 1 {
		t.Errorf("j, rebound to the next target, left row %d; want 1", row)
	}
	if row := press("down"); row != 1 {
		t.Errorf("down, no longer the chart's, moved the row to %d", row)
	}
	if row := press("k"); row != 0 {
		t.Errorf("k, rebound to the previous target, left row %d; want 0", row)
	}
	if got := panName(keys); got != "Pan (Bands: k/j target)" {
		t.Errorf("Help says %q; want the chart's rebound keys beside the pan row, Pan (Bands: k/j target)", got)
	}
}
