package tty

// map_prop_honesty_test.go — 0.19.0 batch 25: the best frequency (D-197),
// each answer's confidence by its nearest station (FR-5.5, D-198), the
// offset compared and the no-readings Setting (FR-5.4, D-105), the refresh
// Setting (FR-4.10, D-120), the shared-device statement (FR-2.2, D-85) and
// the Settings both UI modes share (FR-2.3, D-121).

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// TestTheBestFrequencyIsNamed is D-197: 85 % of the upper limit and the band
// whose lower edge fits under it and above the LUF; where none fits, none
// serves; no upper limit, nothing said. The chart's rows say it short, the
// Broadcaster's target box long.
func TestTheBestFrequencyIsNamed(t *testing.T) {
	for _, c := range []struct {
		l           propLimits
		short, long string
	}{
		{propLimits{upper: 17.6, luf: 6.2}, "best ~15.0 (20m)", "best ~15.0 MHz (20m), 85% of the MUF"},
		{propLimits{upper: 6}, "best ~5.1 (80m)", "best ~5.1 MHz (80m), 85% of the MUF"},
		{propLimits{upper: 18.5, luf: 18.5}, "none serves", "no frequency serves on this path"},
		{propLimits{upper: 6, luf: 4.2}, "none serves", "no frequency serves on this path"},
		{propLimits{}, "", ""},
	} {
		if got := propBestWords(c.l, false); got != c.short {
			t.Errorf("%+v: the row says %q; want %q", c.l, got, c.short)
		}
		if got := propBestWords(c.l, true); got != c.long {
			t.Errorf("%+v: the box says %q; want %q", c.l, got, c.long)
		}
	}
	text := stripANSITest(strings.Join(pressMap(t, chartDash(t), "B").propChartLines(Dashboard{}.opts()), "\n"))
	if !strings.Contains(text, "· best ~") && !strings.Contains(text, "· none serves") {
		t.Errorf("no chart row names a best frequency:\n%s", text)
	}
}

// TestEachRowShowsItsNearestStation is FR-5.5 (D-116, D-198): each chart row
// starts with its nearest reporting station's shade - within 500 km, 500 to
// 2000 km, farther or none, ? unscored - plain under --ascii, the key under
// the band legend; the words say each distance, far ones about as good as
// typical.
func TestEachRowShowsItsNearestStation(t *testing.T) {
	for _, c := range []struct {
		n            ionomaps.Nearest
		shade, ascii string
	}{
		{ionomaps.Nearest{Known: true, Km: 300}, "▓", "#"},
		{ionomaps.Nearest{Known: true, Km: 1200}, "▒", "+"},
		{ionomaps.Nearest{Known: true, Km: 2500}, "░", "."},
		{ionomaps.Nearest{}, "░", "."},
		{ionomaps.Nearest{Known: true, Km: 300, Unscored: true}, "?", "?"},
	} {
		if got, plain := propShade(c.n, false), propShade(c.n, true); got != c.shade || plain != c.ascii {
			t.Errorf("%+v: shade %q (ascii %q); want %q (%q)", c.n, got, plain, c.shade, c.ascii)
		}
	}
	d := pressMap(t, chartDash(t), "B")
	c := &d.mapPane.propChart
	c.targets[0].Nearest = ionomaps.Nearest{Known: true, Km: 350}
	c.targets[1].Nearest = ionomaps.Nearest{Known: true, Km: 2900}
	text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n"))
	if !strings.Contains(text, "▓ "+propContinents[0].name) || !strings.Contains(text, "░ "+propContinents[1].name) || !strings.Contains(text, "nearest station: ▓ <500 km") {
		t.Errorf("the rows' shades or their key are not drawn:\n%s", text)
	}
	words := d.propNearestSummary()
	if !strings.Contains(words, propContinents[0].name+": a station ") || !strings.Contains(words, propContinents[1].name+": a station ") || !strings.Contains(words, "away, about as good as typical") {
		t.Errorf("the words do not say each row's nearest station: %q", words)
	}
	if !strings.Contains(strings.Join(d.close().describeLinesAll(), " "), "Nearest reporting station - ") {
		t.Error("the map's words do not carry the nearest stations")
	}
}

// offsetSnap is a snapshot with an offset.
func offsetSnap(o ionomaps.Offset) ionomaps.Snapshot {
	s := stationsSnap(6, ionomaps.Fresh)
	s.Offset = o
	return s
}

// TestStatusShowsTheLiveAndTypicalOffset is FR-5.4 (D-105): the Status
// window says the live offset beside its typical value and spread, and the
// diagnostics record each live offset; with none live, why.
func TestStatusShowsTheLiveAndTypicalOffset(t *testing.T) {
	var noted []string
	s := &seam{answer: PropagationResult{Snapshot: offsetSnap(ionomaps.Offset{LiveMHz: -0.62, TypicalMHz: -0.51, SpreadMHz: 0.23, Measured: true})}}
	d := propDash(t, s, true)
	d.cfg.MapProblem = func(p string) { noted = append(noted, p) }
	d.cfg.MapSources = []MapSource{{Name: "GIRO", Host: "lgdc.uml.edu", Badge: "GIRO"}}
	d, cmd := keyCmd(t, d, "P")
	d = runProp(t, d, cmd)
	status := strings.Join(d.mapWords(200), " ")
	if !strings.Contains(status, "live -0.62 MHz; typical -0.51 MHz, spread 0.23") {
		t.Errorf("the Status window does not say the live offset beside its typical: %q", status)
	}
	if !strings.Contains(strings.Join(noted, "\n"), "live -0.62 MHz") {
		t.Errorf("the diagnostics have no record of the live offset: %q", noted)
	}
	if w := propOffsetWords(ionomaps.Offset{TypicalMHz: -0.51, SpreadMHz: 0.23, Corrected: true}); !strings.Contains(w, "no live offset") || !strings.Contains(w, "typical offset is applied") {
		t.Errorf("with no live offset: %q", w)
	}
}

// TestAnUnusualOffsetIsSaid is FR-5.4 (D-105): a live offset more than two
// spreads from the typical is said in the chart's status label, the rows
// under the map and the words; one within is not; the correction in use is
// named.
func TestAnUnusualOffsetIsSaid(t *testing.T) {
	d := pressMap(t, chartDash(t), "B")
	with := func(o ionomaps.Offset) { // the snapshot shown, its offset this one
		s := *d.mapPane.propSnap
		s.Offset = o
		d.mapPane.propSnap = &s
	}
	with(ionomaps.Offset{LiveMHz: -1.4, TypicalMHz: -0.51, SpreadMHz: 0.23, Measured: true})
	const unusual = "GloTEC unusually far from the stations now"
	if chart := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n")); !strings.Contains(chart, unusual) {
		t.Errorf("the chart's status label does not say it:\n%s", chart)
	}
	if block := stripANSITest(strings.Join(d.propBlock(d.scrubW()), "\n")); !strings.Contains(block, unusual) {
		t.Errorf("the rows under the map do not say it:\n%s", block)
	}
	if !strings.Contains(d.propStatusWords(), unusual) {
		t.Errorf("the words do not say it: %q", d.propStatusWords())
	}
	with(ionomaps.Offset{LiveMHz: -0.6, TypicalMHz: -0.51, SpreadMHz: 0.23, Measured: true})
	if strings.Contains(d.propStatusWords(), unusual) {
		t.Error("an offset within two spreads is said unusual")
	}
	with(ionomaps.Offset{TypicalMHz: -0.51, SpreadMHz: 0.23, Corrected: true})
	if block := stripANSITest(strings.Join(d.propBlock(d.scrubW()), "\n")); !strings.Contains(block, "corrected by the typical offset") {
		t.Errorf("the correction in use is not named:\n%s", block)
	}
}

// TestTheNoReadingsCorrectionIsASetting is FR-5.4 (D-105): the Maps tab's
// picker, GloTEC as is by default or the typical offset; the update is asked
// with it, and a snapshot asked without it is asked again; it round-trips
// the file.
func TestTheNoReadingsCorrectionIsASetting(t *testing.T) {
	s := &seam{answer: PropagationResult{Snapshot: stationsSnap(0, ionomaps.Fresh)}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	d = runProp(t, d, cmd)
	if s.opts[0].CorrectNoReadings || noReadingsLabel(d.mapPropCorrect) != "GloTEC as is" {
		t.Fatalf("the first update was asked with %+v; GloTEC as is by default", s.opts[0])
	}
	d = d.togglePropCorrect()
	d, again := d.armPropOptions()
	if again == nil {
		t.Fatal("the Setting changed and the snapshot, asked without the correction, was not asked again")
	}
	if d = runProp(t, d, again); !s.opts[len(s.opts)-1].CorrectNoReadings {
		t.Errorf("the update after choosing the correction was asked with %+v", s.opts[len(s.opts)-1])
	}
	d.mapPane.propBusy = false // the ask the Update loop set off has landed: only the kept correction decides
	if _, more := d.armPropOptions(); more != nil || !d.mapPane.propSnapCorrect {
		t.Errorf("a snapshot asked with the Setting's correction was asked again, or its correction not kept (%v)", d.mapPane.propSnapCorrect)
	}
	if noReadingsKey(true) != "corrected" || noReadingsKey(false) != "" || noReadingsLabel(true) != "Typical offset" {
		t.Error("the file's words or the row's do not round-trip")
	}
}

// TestTheRefreshSettingStatesItsCost is FR-4.10 (D-120, D-141): the Maps
// tab's picker, every 10 minutes by default, hourly, or on demand on the
// refresh-now key, each with its measured cost; the open mode's tick follows
// it, none on demand.
func TestTheRefreshSettingStatesItsCost(t *testing.T) {
	s := &seam{answer: PropagationResult{Snapshot: stationsSnap(0, ionomaps.Fresh)}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	d = runProp(t, d, cmd)
	if d.propRefreshLabel() != "Every 10 min" || d.mapPane.propTickEvery != 10*time.Minute {
		t.Fatalf("the Setting starts at %q, its tick %v", d.propRefreshLabel(), d.mapPane.propTickEvery)
	}
	for _, want := range []struct {
		label, note string
		every       time.Duration
	}{
		{"Hourly", "up to 3.3 MB an hour", time.Hour},
		{"On demand (U)", "about 2.6 MB each time", 0},
		{"Every 10 min", "up to 19 MB an hour", 10 * time.Minute},
	} {
		d = d.cyclePropRefresh(true)
		var tick bool
		if d, cmd = d.armPropRefresh(); cmd != nil {
			tick = true
		}
		if d.propRefreshLabel() != want.label || !strings.Contains(propRefreshNote(d.mapPropRefresh), want.note) || d.mapPane.propTickEvery != want.every || tick != (want.every > 0) {
			t.Errorf("%s: the row says %q, costs %q, ticks every %v (a tick set: %v)", want.label, d.propRefreshLabel(), propRefreshNote(d.mapPropRefresh), d.mapPane.propTickEvery, tick)
		}
	}
	if propRefreshByKey("hourly") != refreshHourly || propRefreshByKey("nonsense") != refreshEvery10 || propRefreshKey(refreshOnDemand) != "on-demand" || propRefreshKey(refreshEvery10) != "" {
		t.Error("the file's names do not round-trip, or every 10 minutes is not the default")
	}
}

// TestTheSharedDeviceIsSaid is FR-2.2 (D-80, D-85): on the Broadcaster,
// Help and the Settings window's Broadcaster tab say the broadcast and a
// screen reader share the default sound output; Observer's Help does not.
func TestTheSharedDeviceIsSaid(t *testing.T) {
	d := mapDash(t, Config{})
	if strings.Contains(strings.Join(d.helpLines(d.opts()), " "), sharedDeviceNote) {
		t.Error("Observer's Help speaks of the broadcast's sound output")
	}
	d.surface = SurfaceBroadcaster
	if help := strings.Join(d.helpLines(d.opts()), " "); !strings.Contains(strings.Join(strings.Fields(help), " "), "the same one a screen reader speaks on") {
		t.Errorf("the console's Help does not say the device is shared:\n%s", help)
	}
	d = d.openSetup()
	for range setupTabs() { // at most once round the tabs
		if d.setupTab() == tabBroadcaster {
			break
		}
		m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		d = m.(Dashboard)
	}
	if d.setupTab() != tabBroadcaster {
		t.Fatal("Tab never reached the Broadcaster tab")
	}
	said := false
	for _, s := range d.tabNoticeSlots() {
		said = said || s.notice.text == sharedDeviceNote
	}
	if !said {
		t.Error("the Broadcaster tab does not say the device is shared")
	}
}

// TestTheConsolePropagationModeHonoursTheSharedSettings is FR-2.3 (D-121):
// "Instead of the map", the refresh and the no-readings correction are one
// Setting each across both UI modes, the near-vertical radius Observer's
// alone; on the console the update is asked with the correction and ticks by
// the refresh.
func TestTheConsolePropagationModeHonoursTheSharedSettings(t *testing.T) {
	for _, id := range []setupRowID{rowMapDesc, rowMapPropRefresh, rowMapPropReadings} {
		if setupTable()[id].scope != scopeShared {
			t.Errorf("row %d is not one Setting for both UI modes (D-121)", id)
		}
	}
	if setupTable()[rowMapPropRadius].scope != scopeObserver {
		t.Error("the near-vertical radius is not Observer's alone: the Broadcaster keeps its service radius")
	}
	s := &seam{answer: PropagationResult{Snapshot: stationsSnap(0, ionomaps.Fresh)}}
	d := propDash(t, s, true)
	d.surface, d.mapPropCorrect, d.mapPropRefresh = SurfaceBroadcaster, true, refreshHourly
	d, cmd := keyCmd(t, d, "P")
	d = runProp(t, d, cmd)
	if !s.opts[0].CorrectNoReadings || d.mapPane.propTickEvery != time.Hour {
		t.Errorf("on the console the update was asked with %+v and ticks every %v", s.opts[0], d.mapPane.propTickEvery)
	}
}

// TestInsteadOfTheMapIsShared is FR-2.3 (D-121): the description's Setting
// is drawn and applied on the console as on Observer: "Instead of the map"
// chosen there gives the console's map the words in its place.
func TestInsteadOfTheMapIsShared(t *testing.T) {
	d := stationMap(t, Config{MapDescription: "instead"})
	if d.mapDesc != mapDescInstead {
		t.Fatalf("the console's map does not hold the shared Setting: %v", d.mapDesc)
	}
	if d.mapPicture() {
		t.Error("with \"Instead of the map\" chosen the console's map still draws its picture")
	}
	if !d.shownOnSurface(rowMapDesc) {
		t.Error("the Setting is not drawn on the console")
	}
}

// ageSnap is a snapshot computed at now, its GloTEC grid valid gridAgo
// earlier, and its oldest GIRO reading readingAgo earlier (none for zero).
func ageSnap(now time.Time, gridAgo, readingAgo time.Duration) ionomaps.Snapshot {
	s := propSnapAt(now, 18)
	s.Inputs.GloTECValid = now.Add(-gridAgo)
	if readingAgo > 0 {
		s.Inputs.Stations, s.Inputs.OldestReading = 4, now.Add(-readingAgo)
	}
	return s
}

// TestTheAgeIsTheOldestInputs is FR-5.1 (A-30): the age said is the oldest
// input's - GloTEC's grid, or the oldest GIRO reading assimilated where that
// is older - never the time the field was computed; with the climatology
// alone, that is said in its place.
func TestTheAgeIsTheOldestInputs(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	d := Dashboard{now: func() time.Time { return now }}
	for _, c := range []struct {
		snap ionomaps.Snapshot
		says string
	}{
		{ageSnap(now, 25*time.Minute, 0), "data 25 min old"},
		{ageSnap(now, 25*time.Minute, 50*time.Minute), "data 50 min old"},
		{ageSnap(now, 40*time.Minute, 10*time.Minute), "data 40 min old"},
		{ageSnap(now, 20*time.Second, 0), "data just in"},
	} {
		if got, _, _ := d.propAge(c.snap); got != c.says {
			t.Errorf("%v grid, %v reading: %q; want %q", now.Sub(c.snap.Inputs.GloTECValid), now.Sub(c.snap.Inputs.OldestReading), got, c.says)
		}
	}
	clim := ageSnap(now, 25*time.Minute, 0)
	clim.Background.FoF2 = ionomaps.Climatology
	if got, _, _ := d.propAge(clim); got != "climatology" {
		t.Errorf("the climatology alone is said %q", got)
	}
}

// TestTheFieldsAgeIsAlwaysSaid is FR-5.1: on screen, in the mode line on its
// freshness tone, and in the words.
func TestTheFieldsAgeIsAlwaysSaid(t *testing.T) {
	d, _ := drawnDash(t)
	words, age, _ := d.propAge(*d.mapPane.propSnap)
	if block := strings.Join(d.propBlock(d.scrubW()), "\n"); !strings.Contains(block, render.Tint("("+words+")", render.Tok(render.FreshnessTone(age)))) {
		t.Errorf("the mode line does not say %q on its tone:\n%s", words, stripANSITest(block))
	}
	if !strings.Contains(d.propStatusWords(), words) {
		t.Errorf("the words do not say %q: %q", words, d.propStatusWords())
	}
}

// TestAStaleFieldIsSaidStale is FR-5.2 (A-56): past an hour - go-ionomaps'
// own currency for a grid and a reading - the field is said stale, in the
// alert's tone, on screen and in the words.
func TestAStaleFieldIsSaidStale(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	d, _ := drawnDash(t)
	old := ageSnap(d.now(), 2*time.Hour+5*time.Minute, 0)
	d.mapPane.propSnap = &old
	words, _, stale := d.propAge(old)
	if !stale || words != "STALE: data 2 h 5 min old" {
		t.Fatalf("a field two hours old is said %q (stale %v)", words, stale)
	}
	if block := strings.Join(d.propBlock(d.scrubW()), "\n"); !strings.Contains(block, render.Tint("("+words+")", render.Tok(render.AlertLabel))) {
		t.Errorf("the mode line does not say it stale in the alert's tone:\n%s", stripANSITest(block))
	}
	if !strings.Contains(d.propStatusWords(), "STALE") {
		t.Errorf("the words do not say it stale: %q", d.propStatusWords())
	}
}

// TestAFailedFetchIsSaidWithItsPath is FR-5.2 (D-120, 0.18.0 D-124): an
// update that failed is said with the key that asks again.
func TestAFailedFetchIsSaidWithItsPath(t *testing.T) {
	d := propDash(t, &seam{}, true)
	d.mapPane.propErr = errors.New("ionomaps: NOAA not reached")
	if w := d.propStatusWords(); !strings.Contains(w, "NOAA not reached") || !strings.Contains(w, "U refreshes now") {
		t.Errorf("a failed update is said %q; with the key that asks again", w)
	}
}

// TestTheFallbackIsNamed is FR-5.2: with no GloTEC grid the climatology's
// foF2 is named in the words, its source's badge is dimmed, and with no
// measured input at all the mode line says "climatology" where an age would
// be.
func TestTheFallbackIsNamed(t *testing.T) {
	d, _ := drawnDash(t)
	s := ageSnap(d.now(), 0, 10*time.Minute)
	s.Background.FoF2, s.Inputs.GloTECValid = ionomaps.Climatology, time.Time{}
	d.mapPane.propSnap = &s
	if w := d.propStatusWords(); !strings.Contains(w, "foF2 from the climatology: no GloTEC grid") {
		t.Errorf("the climatology's foF2, with stations, is said %q", w)
	}
	if got := strings.Join(d.propBadges(false), " "); strings.Contains(got, "SWPC") {
		t.Errorf("no GloTEC grid fed the field and the badges are %q", got)
	}
	s.Inputs.Stations, s.Inputs.OldestReading = 0, time.Time{}
	if block := stripANSITest(strings.Join(d.propBlock(d.scrubW()), "\n")); !strings.Contains(block, "(climatology)") {
		t.Errorf("the climatology alone is not said on the mode line:\n%s", block)
	}
}
