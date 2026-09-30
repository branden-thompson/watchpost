package tty

// map_prefs.go — the Maps tab's other Settings (0.18.0 batch 9): the scale
// the map opens at (W1.11, W4.3, FR-2.3), how near an alert's edge must be to
// be called near (W9.2, FR-7.4), the layers the app's registry names (W1.11,
// W1.13, R-9.2) and the warning when they would cost a lot (W1.14, FR-9.2).

import (
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// MapLayer is one layer the listener can switch, as the app's registry names
// it: its key - the part of its overlays' ids before the slash - its words,
// and the builders' default.
type MapLayer struct {
	Key, Label string
	On         bool
	// Chips are the sources the layer's badge names when they never change
	// (D-133): FIRE [NIFC]/[HMS]. A layer whose sources vary has none; its
	// answer names them.
	Chips []string
}

// AlertLayer is the alert areas' key: the app registers the layer under it,
// and the window's description and notes speak for it.
const AlertLayer = "alert"

// AlertCategory is one of the [w] window's categories the map's alerts are
// switched by (D-80): its key, which an alert overlay's id carries after the
// layer's ("alert/warnings/<id>"), and [w]'s own words and category.
type AlertCategory struct {
	Key, Label string
	Cat        category.Category
}

// AlertCategories are the map's alert categories, in [w]'s order: every tab
// but Disasters - earthquakes are their own layer - and Forecasts, which
// belong to the forecast overlays to come (D-80, F-182).
func AlertCategories() []AlertCategory {
	keys := map[category.Category]string{category.Emergency: "emergency", category.Warnings: "warnings", category.Watches: "watches",
		category.Advisories: "advisories", category.Statements: "statements", category.Marine: "marine"}
	var out []AlertCategory
	for _, c := range category.All() {
		if key, ok := keys[c]; ok {
			out = append(out, AlertCategory{Key: key, Label: category.Of(c).TabLabel, Cat: c})
		}
	}
	return out
}

// AlertCategoryKey is a category's key, and false for one the map does not
// draw.
func AlertCategoryKey(c category.Category) (string, bool) {
	for _, ac := range AlertCategories() {
		if ac.Cat == c {
			return ac.Key, true
		}
	}
	return "", false
}

// categoryChoice is a category's key among the layer choices - "alert-warnings"
// - so the switches are saved with the layers', and a hyphen keeps the file's
// table flat.
func categoryChoice(key string) string { return AlertLayer + "-" + key }

// alertCategoriesOff are the alert categories unchecked in the Overlays
// menu, as the feed's ask says them (D-160).
func (d Dashboard) alertCategoriesOff() []string {
	var off []string
	for _, c := range AlertCategories() {
		if !d.layerOn(categoryChoice(c.Key)) {
			off = append(off, c.Key)
		}
	}
	return off
}

// AlertCategorySwitch is an alert category's switch, as the app asks it
// (D-149): the key its checkbox is kept under.
func AlertCategorySwitch(key string) string { return categoryChoice(key) }

// alertLayerOffText is what the description says with the alert areas off:
// that they are off, never that nothing is there.
const alertLayerOffText = "Alert areas are switched off in the Overlays menu (O), so none is drawn or described." // D-76: switched at the map

// MapCost is what one refresh of the map's data would fetch (FR-9.2).
type MapCost struct {
	Bytes    int64
	Requests int
}

// The cost warning's thresholds (FR-9.2, D-43): more than either is said.
// RAISED BY D-149 AND COUNTED OVER THE OVERLAYS CHOSEN ALONE: at D-43's
// 2 MB and 40 over everything, radar - the mode, 37 requests - and the
// alerts' zones, whatever their categories, warned with no overlay on, and
// a warning that always speaks is one that is ignored.
const (
	mapCostBytes    = 3_000_000
	mapCostRequests = 25
)

// costWarningParts is the warning in the HUM LEAD's words (D-82): the
// sentence said in bold, and the estimate with what to do about it; nothing
// at or under both thresholds.
func costWarningParts(c MapCost) (head, detail string) {
	if c.Bytes <= mapCostBytes && c.Requests <= mapCostRequests {
		return "", ""
	}
	return "Map may experience performance issues at this zoom level.",
		"Est. " + strconv.FormatFloat(float64(c.Bytes)/1e6, 'f', 1, 64) + "MB / " + strconv.Itoa(c.Requests) +
			" Requests | " + costAdvice
}

// costAdvice is the warning's second sentence, said beside the first under
// the map, in the HUM LEAD's words (D-133).
const costAdvice = "Adjust layers/zoom to improve experience."

// costEstimate is the estimate alone, which the status line carries after the
// radar's (D-89); nothing at or under both thresholds.
func costEstimate(c MapCost) string {
	if head, _ := costWarningParts(c); head == "" {
		return ""
	}
	return "Est. " + strconv.FormatFloat(float64(c.Bytes)/1e6, 'f', 1, 64) + "MB / " + strconv.Itoa(c.Requests) + " Requests"
}

// mapCostLine is the warning under the map, on one line (D-89): its first
// sentence in bold yellow - the list pointer's, AA-checked on the window's
// ground in every theme - then the advice.
func mapCostLine(c MapCost, width int) []string {
	head, _ := costWarningParts(c)
	if head == "" {
		return nil
	}
	return render.WrapLines([]string{render.Tint(head, render.Tok(render.ListPointer)) + " " + costAdvice}, width)
}

// costWarningLines is the warning wrapped to a width, its first sentence in
// bold (D-82).
func costWarningLines(c MapCost, width int) []string {
	head, detail := costWarningParts(c)
	if head == "" {
		return nil
	}
	var out []string
	for _, l := range render.WrapText(head, width) {
		out = append(out, render.Bold(l))
	}
	return append(out, render.WrapText(detail, width)...)
}

// refreshMapCost asks the app's estimate again, with the layers as chosen:
// on opening the map or Settings, on the map's new data, and on a layer's
// switch. The frame reads the answer; it never asks.
func (d Dashboard) refreshMapCost() Dashboard {
	if d.cfg.MapCost == nil {
		return d
	}
	d.mapCost = d.cfg.MapCost(d.mapAsk(), d.layerOn)
	return d
}

// mapScaleMode is the scale the map opens at (FR-2.3). The region's bound is
// not a Setting: every scale is held inside it (D-28).
type mapScaleMode int

const (
	mapScaleState  mapScaleMode = iota // the default: about a state (D-54)
	mapScaleCounty                     // about a county
	mapScaleRegion                     // the whole region the place is in
)

// Key is the word the file keeps.
func (s mapScaleMode) Key() string {
	switch s {
	case mapScaleCounty:
		return "county"
	case mapScaleRegion:
		return "region"
	}
	return "state"
}

// Label is the picker's words.
func (s mapScaleMode) Label() string {
	switch s {
	case mapScaleCounty:
		return "County"
	case mapScaleRegion:
		return "Region"
	}
	return "State"
}

// zoom is the scale's zoom. The region's is none the bound allows, which the
// bound raises to the least that keeps the view inside the region.
func (s mapScaleMode) zoom() float64 {
	switch s {
	case mapScaleCounty:
		return 9
	case mapScaleRegion:
		return 0
	}
	return 6
}

// mapScaleByKey reads the file's word; anything else is the default.
func mapScaleByKey(key string) mapScaleMode {
	switch key {
	case "county":
		return mapScaleCounty
	case "region":
		return mapScaleRegion
	}
	return mapScaleState
}

// cycleMapScale moves the scale's picker: region, state, county, in order of
// closeness, and round again.
func (d Dashboard) cycleMapScale(forward bool) Dashboard {
	order := []mapScaleMode{mapScaleRegion, mapScaleState, mapScaleCounty}
	at := 0
	for i, s := range order {
		if s == d.mapScale {
			at = i
		}
	}
	step := 1
	if !forward {
		step = len(order) - 1
	}
	d.mapScale = order[(at+step)%len(order)]
	return d.uiTouched()
}

// mapNearbyChoices are the nearby distances offered, in kilometres.
var mapNearbyChoices = []int{5, 10, 15, 25, 50}

// mapNearbyDefaultKm is M1's own rule (W0.2's answer key): an edge within it
// stops short of the place; farther, it lies to one side.
const mapNearbyDefaultKm = 15

// mapNearbyByKm reads the file's number; anything not offered is the default.
func mapNearbyByKm(km int) int {
	for _, c := range mapNearbyChoices {
		if c == km {
			return km
		}
	}
	return mapNearbyDefaultKm
}

// cycleNearby moves the nearby picker through the choices, and round again.
func (d Dashboard) cycleNearby(forward bool) Dashboard {
	d.mapNearbyKm = nextChoice(mapNearbyChoices, d.mapNearbyKm, forward)
	return d.uiTouched()
}

// nextChoice is the choice after, or before, the one held, and round again:
// a picker's step through its numbers.
func nextChoice[T comparable](choices []T, held T, forward bool) T {
	at := 0
	for i, c := range choices {
		if c == held {
			at = i
		}
	}
	step := 1
	if !forward {
		step = len(choices) - 1
	}
	return choices[(at+step)%len(choices)]
}

// toggleRadarSource switches the lower 48's radar between MRMS and IEM (D-83).
func (d Dashboard) toggleRadarSource() Dashboard {
	d.mapRadarIEM = !d.mapRadarIEM
	return d.uiTouched() // the next open asks for the source chosen
}

// radarAheadChoices are the radar loop's hours ahead offered (D-114).
var radarAheadChoices = []int{1, 3, 6, 12}

// radarAheadDefault is three hours (D-114).
const radarAheadDefault = 3

// radarAheadByHours reads the file's number; anything not offered is the
// default.
func radarAheadByHours(h int) int {
	for _, c := range radarAheadChoices {
		if c == h {
			return h
		}
	}
	return radarAheadDefault
}

// radarAheadLabel is the row's words.
func radarAheadLabel(h int) string {
	if h == 1 {
		return "1 hour"
	}
	return strconv.Itoa(h) + " hours"
}

// quakeFeeds are the quakes the map draws (D-122), USGS's summary feeds by
// their own names, the default first: M2.5+ the past week or the past day,
// M1.0+ the same.
var quakeFeeds = []string{"2.5_week", "2.5_day", "1.0_week", "1.0_day"}

// quakeFeedDefault is M2.5+ over the past week (D-122).
const quakeFeedDefault = "2.5_week"

// quakeFeedByKey reads the file's word; anything not offered is the default.
func quakeFeedByKey(k string) string {
	for _, f := range quakeFeeds {
		if f == k {
			return k
		}
	}
	return quakeFeedDefault
}

// quakeFeedLabel is the row's words: "M2.5+, past week".
func quakeFeedLabel(k string) string {
	mag, span, _ := strings.Cut(quakeFeedByKey(k), "_")
	return "M" + mag + "+, past " + span
}

// cycleQuakeFeed moves the quakes picker, and round again.
func (d Dashboard) cycleQuakeFeed(forward bool) Dashboard {
	d.mapQuakeFeed = nextChoice(quakeFeeds, d.mapQuakeFeed, forward)
	return d.uiTouched()
}

// cycleRadarAhead moves the hours-ahead picker, and round again.
func (d Dashboard) cycleRadarAhead(forward bool) Dashboard {
	d.mapRadarAhead = nextChoice(radarAheadChoices, d.mapRadarAhead, forward)
	return d.uiTouched()
}

// radarSourceKey is the file's word: "iem", or empty for MRMS, the default.
func radarSourceKey(iem bool) string {
	if iem {
		return "iem"
	}
	return ""
}

// radarSourceLabel is the row's words: MRMS by default, or IEM, which covers
// the lower 48 alone.
func (d Dashboard) radarSourceLabel() string {
	if d.mapRadarIEM {
		return "IEM (lower 48)"
	}
	return "MRMS"
}

// nearbyLabel is the distance in the units the description speaks.
func (d Dashboard) nearbyLabel() string {
	km := strconv.Itoa(d.mapNearbyKm) + " km"
	if d.units != render.UnitF {
		return km
	}
	return strconv.Itoa(int(math.Round(float64(d.mapNearbyKm)*0.621371))) + " miles (" + km + ")"
}

// layerOn reports whether a layer is drawn: the listener's choice, or the
// builders' default. A key the registry does not name is drawn.
func (d Dashboard) layerOn(key string) bool {
	if key == TemperatureLayer && d.mapPane.tempAuto && !d.radarMode() {
		return true // Forecast mode turned it on (D-103), its alone (D-104)
	}
	return d.chosen(key)
}

// chosen is a layer as the listener chose it: its tick, inside its group.
// radarMode reads the radar through it rather than through layerOn, which
// would ask radarMode again - a cycle P10 forbids (W14, S-3); the radar is
// never Forecast mode's own, so the answer is the same.
func (d Dashboard) chosen(key string) bool {
	if g, ok := layerGroup[key]; ok && !d.groupOn(g) {
		return false // its group disabled: drawn nowhere, its tick kept (D-143)
	}
	return d.ticked(key)
}

// The menu's checkbox groups (D-141, D-143): each a switch of its own, which
// hides its members without touching their ticks. Alert Areas is the alert
// layer's own switch.
const (
	groupPoints  = "group:points"
	groupHazards = "group:hazards"
)

// QuakeLayer is the earthquakes' key, as the app registers it (D-80).
const QuakeLayer = "quake"

// layerGroup is each grouped layer's group.
var layerGroup = map[string]string{WindLayer: groupPoints, WaveLayer: groupPoints, BuoyLayer: groupPoints, TideLayer: groupPoints,
	RainLayer: groupPoints, FireLayer: groupHazards, QuakeLayer: groupHazards}

// groupOn reports whether a group is enabled: on until switched off.
func (d Dashboard) groupOn(group string) bool {
	on, ok := choiceOf(d.mapLayerChoice, group)
	return on || !ok
}

// The fire's choices (D-145): everything, the named fires alone - their
// perimeters and incidents - or the satellite hotspots alone.
const (
	FireAll      = "all"
	FireNamed    = "named"
	FireHotspots = "hotspots"
)

// fireModes are the Fire row's choices in order.
var fireModes = []string{FireAll, FireNamed, FireHotspots}

// fireMode is the Fire row's choice, All until another is chosen.
func (d Dashboard) fireMode() string {
	for _, m := range fireModes[1:] {
		if on, _ := choiceOf(d.mapLayerChoice, "fire:"+m); on {
			return m
		}
	}
	return FireAll
}

// ticked is a layer's own switch, its group aside: the listener's choice, or
// the layer's default.
func (d Dashboard) ticked(key string) bool {
	if on, ok := choiceOf(d.mapLayerChoice, key); ok {
		return on
	}
	for _, l := range d.cfg.MapLayers {
		if l.Key == key {
			return l.On
		}
	}
	return true
}

// layerChoiceKey is the listener's layer choices as one comparable word -
// "alert=off,quake=on", sorted - so a copied Dashboard never shares a map.
func layerChoiceKey(choice map[string]bool) string {
	parts := make([]string, 0, len(choice))
	for k, on := range choice {
		v := "off"
		if on {
			v = "on"
		}
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// layerChoices are the choices as the file keeps them.
func (d Dashboard) layerChoices() map[string]bool { return choicesOf(d.mapLayerChoice) }

// toggleLayer switches the layer under the row's cursor, and asks the
// estimate again.
func (d Dashboard) toggleLayer() Dashboard {
	layers := d.cfg.MapLayers
	if len(layers) == 0 {
		return d
	}
	key := layers[min(d.setup.layerAt, len(layers)-1)].Key
	choice := d.layerChoices()
	if choice == nil {
		choice = map[string]bool{}
	}
	on := !d.ticked(key) // its own switch, whatever its group's (D-143)
	choice[key] = on
	if on && slices.Contains(oneTint, key) {
		for _, other := range oneTint {
			if other != key {
				choice[other] = false // one tint at a time, here as in the menu (D-119, D-137, D-139)
			}
		}
	}
	d.mapLayerChoice = layerChoiceKey(choice)
	return d.refreshMapCost().uiTouched()
}

// stepLayer moves the layers row's cursor, and round again.
func (d Dashboard) stepLayer(forward bool) Dashboard {
	n := len(d.cfg.MapLayers)
	if n == 0 {
		return d
	}
	step := 1
	if !forward {
		step = n - 1
	}
	d.setup.layerAt = (d.setup.layerAt + step) % n
	return d.settled()
}

// layersCell is the layers row's value: a box per layer, the one under the
// cursor marked while the row has the focus.
func (d Dashboard) layersCell(o render.Opts, focused bool) string {
	var cells []string
	for i, l := range d.cfg.MapLayers {
		cell := checkMark(o, d.layerOn(l.Key)) + " " + l.Label
		if focused && len(d.cfg.MapLayers) > 1 && i == d.setup.layerAt {
			cell = render.ListLabel(cell, true)
		}
		cells = append(cells, cell)
	}
	return strings.Join(cells, "   ")
}

// feedForLayers is the feed with the layers switched off taken out: their
// overlays, an alert category's when it is off, and - with the alert areas
// off - the notes and the missing zones that speak for them.
func (d Dashboard) feedForLayers(f MapFeed) MapFeed {
	var out MapFeed
	for _, o := range f.Overlays {
		key, rest, _ := strings.Cut(o.ID, "/")
		if !d.layerOn(key) {
			continue
		}
		if cat, _, ok := strings.Cut(rest, "/"); ok && key == AlertLayer && !d.layerOn(categoryChoice(cat)) {
			continue // its category is switched off (D-80)
		}
		out.Overlays = append(out.Overlays, o)
	}
	if d.layerOn(AlertLayer) {
		out.Notes, out.InMissing, out.InView = f.Notes, f.InMissing, f.InView
	}
	return out
}

// mapPickerRow reports the map's pickers, which have nothing to preview.
func mapPickerRow(id setupRowID) bool {
	return id == rowMapDesc || id == rowMapScale || id == rowMapNearby || id == rowMapRadarSource || id == rowMapTempSource || id == rowMapRadarAhead || id == rowMapQuakes || id == rowMapDetailLevel
}

// mapPrefArrow is ←→ on the scale, the nearby distance and the layers.
func (d Dashboard) mapPrefArrow(forward bool) (Dashboard, bool) {
	switch d.setup.focus {
	case rowMapScale:
		return d.cycleMapScale(forward), true
	case rowMapNearby:
		return d.cycleNearby(forward), true
	case rowMapRadarSource:
		return d.toggleRadarSource(), true
	case rowMapTempSource:
		return d.toggleTempSource(), true
	case rowMapRadarAhead:
		return d.cycleRadarAhead(forward), true
	case rowMapQuakes:
		return d.cycleQuakeFeed(forward), true
	case rowMapDetailLevel:
		return d.cycleDetailLevel(forward).uiTouched(), true
	case rowMapLayers:
		return d.stepLayer(forward), true
	}
	return d.toggleDetailRow(d.setup.focus)
}

// MapAsk is what the map's feed and its estimate are asked with (0.18.0): the
// station's data, the selected place, and the view - the map draws every
// alert in it (D-66, D-76).
type MapAsk struct {
	Snap  *snapshot.Snapshot
	Place *snapshot.Location
	View  MapView // the map in view: its areas are asked for their alerts (D-66)
	// Region is the map region the view is bound to (D-28), whose radar the
	// radar is; RadarIEM is the listener's choice of IEM for the lower 48
	// (D-83; MRMS otherwise).
	Region   string
	RadarIEM bool
	// The temperature's (W10): Forecast mode (D-94); NDFD chosen for it
	// (D-93, D-101: Open-Meteo is the default); the listener's unit; and the start of the listener's hour, which
	// Forecast mode's steps are counted from.
	Forecast, TempNDFD, Fahrenheit bool
	// RadarAhead is the loop's hours ahead (D-114).
	RadarAhead int
	// QuakeFeed is the quakes the map draws, USGS's feed by its name
	// (D-122); Clock the listener's, for their times (D-123).
	QuakeFeed string
	Clock     render.Clock
	// Buoys and Tides say their rows are on: they are asked only then
	// (D-127, D-128) - the tides are a request a station.
	Buoys, Tides bool
	// Fire and Quakes say those rows are on (W14, P-3): each fetched only
	// then - D-149's "nothing of it is fetched" for what is off.
	Fire, Quakes bool
	// AlertsOff says the Alert areas layer is off, and AlertCategoriesOff
	// the categories unchecked (D-80): their alerts are gone from the map -
	// not drawn, described, noted nor fetched (D-160). Said as what is OFF,
	// so an ask that says nothing asks for every alert.
	AlertsOff          bool
	AlertCategoriesOff []string
	// UV and Air say those rows are on (D-137, D-139): each asked only then,
	// UV where temperature's source is not Open-Meteo's already.
	UV, Air bool
	// FireMode is the Fire row's choice (D-145): FireAll, FireNamed or
	// FireHotspots.
	FireMode string
	Anchor   time.Time
}

// mapAsk is the ask as the window stands: the watchlist's places, and the
// selected place when it is not one of them - a RECENT or SEARCHED place
// holds its alerts in the recent snapshot, and without it they were never
// drawn (UAT-1 U1-14). The watchlist's snapshot is shared, so the selected
// place joins a copy.
func (d Dashboard) mapAsk() MapAsk {
	place := d.selectedLocation()
	snap := d.snap
	if place != nil && d.selected >= d.numPriority() {
		joined := snapshot.Snapshot{}
		if snap != nil {
			joined = *snap
		}
		joined.Locations = append(append([]snapshot.Location(nil), joined.Locations...), *place)
		snap = &joined
	}
	return MapAsk{Snap: snap, Place: place, View: d.viewBox(d.mapBodySize()), Region: d.mapPane.region.Name, RadarIEM: d.mapRadarIEM,
		Forecast: !d.radarMode(), TempNDFD: d.mapTempNDFD, RadarAhead: d.mapRadarAhead, QuakeFeed: d.mapQuakeFeed, Clock: d.clockFmt, Buoys: d.layerOn(BuoyLayer), Tides: d.layerOn(TideLayer), Fire: d.layerOn(FireLayer), Quakes: d.layerOn(QuakeLayer), AlertsOff: !d.layerOn(AlertLayer), AlertCategoriesOff: d.alertCategoriesOff(), UV: d.layerOn(UVLayer), Air: d.layerOn(AirLayer), FireMode: d.fireMode(), Fahrenheit: d.units == render.UnitF, Anchor: d.tempAnchor()}
}

// detailRowLayer is the detail layer a Map detail row switches, and whether the
// row is one (UAT-1 U1-35: a row each, in mapDetailLayers' order).
func detailRowLayer(id setupRowID) (mapDetailLayer, bool) {
	if id < rowMapDetailBorders || id > rowMapDetailParks {
		return mapDetailLayer{}, false
	}
	return mapDetailLayers()[id-rowMapDetailBorders], true
}

// toggleDetailRow switches the focused detail row's layer: two states, so
// space and either arrow are "the other one", as every on/off row is.
func (d Dashboard) toggleDetailRow(id setupRowID) (Dashboard, bool) {
	l, ok := detailRowLayer(id)
	if !ok {
		return d, false
	}
	return d.setDetail(l.key, !d.detailOn(l.key)).uiTouched(), true
}
