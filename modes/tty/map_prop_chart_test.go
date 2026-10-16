package tty

import (
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// chartDash is the map in the Propagation mode with a snapshot drawn and the
// chart's answers worked out, as the command the Update armed works them.
func chartDash(t *testing.T) Dashboard {
	t.Helper()
	d, _ := drawnDash(t)
	return workChart(t, d)
}

// workChart runs the chart's command and applies it: the one the last
// Update armed, which the harness that fed it dropped, asked again.
func workChart(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
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
	if c.err != nil || len(c.targets) != len(propContinents) {
		t.Fatalf("the chart holds %d targets (%v); want the %d continents", len(c.targets), c.err, len(propContinents))
	}
	for i, a := range c.targets {
		bands := a.Bands
		if len(bands) != len(hfBands) {
			t.Fatalf("%s has %d bands; want %d", propContinents[i].name, len(bands), len(hfBands))
		}
		if l := bands[0].Limit; l != ionomaps.LimitMUF && l != ionomaps.LimitAbsorption && l != ionomaps.LimitCircuit && !propContinents[i].near(c.key.origin, c.key.radiusKm) {
			t.Errorf("%s's 160m names limit %v; a path's is its MUF, or the lower limit's (D-70)", propContinents[i].name, l)
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
			if want := openAt(b.OpenHours, at); (cells[2*h] == propOpenGlyph) != want {
				t.Errorf("%s at %02d UTC is drawn %q; typically open: %v", hfBands[i].name, h, cells[2*h], want)
			}
		}
	}
	m, _ := d.Update(mapKeyMsg(t, "down"))
	if got := m.(Dashboard).mapPane.propChartRow; got != 1 {
		t.Errorf("down moved the chart's target to %d; want 1, the first continent", got)
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
		t.Fatalf("Observer's chart answers from %v (%q); the centre is %v (%q)", o, d.mapPane.propOrigin.name, centre, d.viewName())
	}
	moved := pressMap(t, d, "right")
	moved = moved.renderMap()
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
	older := propChartMsg{chart: propChart{key: propChartKey{radiusKm: 1}, targets: make([]ionomaps.Answer, len(propContinents))}}
	if m, _ := d.Update(older); len(m.(Dashboard).mapPane.propChart.targets) != 0 {
		t.Error("an answer to an older ask was kept over the one asked for")
	}
}

// TestTheNightFollowsTheClock is L-4.3 in the mode: as the clock moves a
// step, the night handed in moves with it, with nothing else changing.
func TestTheNightFollowsTheClock(t *testing.T) {
	d, _ := drawnDash(t)
	later := d.now().Add(2 * propNightStep)
	d.now = func() time.Time { return later }
	m, _ := d.Update(struct{}{})
	night, ok := m.(Dashboard).mapPane.propGiven["propagation/night"]
	if want := later.UTC().Truncate(propNightStep); !ok || !night.Valid.Equal(want) {
		t.Errorf("the night handed in is for %v (%v); the clock says %v", night.Valid, ok, want)
	}
}

// TestTheNightIsDrawnOnThePropagationMap is go-tuiMaps P4 in the mode
// (L-4.1 to L-4.3, D-174): the terminator and the night side are handed in
// with the field, for the moment shown.
func TestTheNightIsDrawnOnThePropagationMap(t *testing.T) {
	d, _ := drawnDash(t)
	var line, night bool
	for _, o := range d.propOverlays() {
		for _, f := range o.Features {
			line = line || f.Role == propTerminatorRole
			night = night || f.Role == propNightRole
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
		t.Errorf("the chart answers for %v km; the Setting says 600", got)
	}
	block := stripANSITest(strings.Join(d.propBlock(d.scrubW()), "\n"))
	if !strings.Contains(block, "Local, to ~600 km (NVIS)") {
		t.Errorf("the block does not say the Setting's radius:\n%s", block)
	}
	if p := d.uiForSave(); p.MapPropRadius != 600 {
		t.Errorf("the radius saved is %d; want 600", p.MapPropRadius)
	}
}
