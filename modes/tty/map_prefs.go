package tty

// map_prefs.go — the Maps tab's other Settings: the scale the map opens at
// (W1.11, W4.3, FR-2.3), how near an alert's edge must be to be called near
// (W9.2, FR-7.4), the layers the app's registry names (W1.11, W1.13, R-9.2)
// and the warning when they would cost a lot (W1.14, FR-9.2).

import (
	ionomaps "github.com/branden-thompson/go-ionomaps"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty/internal/frame"
	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/units"
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

// alertCategoriesOff are the alert categories unchecked in the Overlays
// menu, as the feed's ask says them (D-160).
func (mm mapModel) alertCategoriesOff(e mapEnv) []string {
	var off []string
	for _, c := range AlertCategories() {
		if !mm.layerOn(AlertCategorySwitch(c.Key), e) {
			off = append(off, c.Key)
		}
	}
	return off
}

// AlertCategorySwitch is an alert category's key among the layer choices -
// "alert-warnings" - so the switches are saved with the layers', and a hyphen
// keeps the file's table flat. The app asks the same switch when it builds the
// feed (D-149).
func AlertCategorySwitch(key string) string { return AlertLayer + "-" + key }

// alertLayerOffText is what the description says with the alert areas off:
// that they are off, never that nothing is there.
const alertLayerOffText = "Alert areas are switched off in the Overlays menu (O), so none is drawn or described." // D-76: switched at the map

// alertsFailedText is said where the view's alerts could not be asked: never
// that none is there (D-266).
const alertsFailedText = "Alerts for the map could not be loaded."

// MapCost is what one refresh of the map's data would fetch (FR-9.2).
type MapCost struct {
	Bytes    int64
	Requests int
}

// The cost warning's thresholds (FR-9.2, D-43): more than either is said.
// COUNTED OVER THE OVERLAYS CHOSEN ALONE, at D-149's thresholds: counted over
// everything, radar - the mode, 37 requests - and the alerts' zones, whatever
// their categories, would warn with no overlay on, and a warning that always
// speaks is one that is ignored.
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

// refreshMapCost asks the app's estimate again, with the layers as chosen:
// on opening the map or Settings, on the map's new data, and on a layer's
// switch. The frame reads the answer; it never asks.
func (mm mapModel) refreshMapCost(e mapEnv) mapModel {
	if mm.mapHooks.MapCost == nil {
		return mm
	}
	s := layerSwitches{mm: mm, e: e}
	mm.mapCost = mm.mapHooks.MapCost(mm.mapAsk(e), s.on)
	return mm
}

// layerSwitches is the layers' switches as the estimate reads them, the map
// and what it is lent held together, so the hook is handed one method value
// as before (0.19.6 design MS-10: no new escape).
type layerSwitches struct {
	mm mapModel
	e  mapEnv
}

// on is whether layer key is drawn.
func (s layerSwitches) on(key string) bool { return s.mm.layerOn(key, s.e) }

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
func (mm mapModel) cycleMapScale(forward bool) (mapModel, mapWant) {
	mm.mapScale = cycleIn([]mapScaleMode{mapScaleRegion, mapScaleState, mapScaleCounty}, itself, mm.mapScale, forward)
	return mm.wantTouched()
}

// mapNearbyChoices are the nearby distances offered, in kilometres.
var mapNearbyChoices = []int{5, 10, 15, 25, 50}

// mapNearbyDefaultKm is M1's own rule (W0.2's answer key): an edge within it
// stops short of the place; farther, it lies to one side.
const mapNearbyDefaultKm = 15

// mapNearbyByKm reads the file's number; anything not offered is the default.
func mapNearbyByKm(km int) int {
	if slices.Contains(mapNearbyChoices, km) {
		return km
	}
	return mapNearbyDefaultKm
}

// cycleNearby moves the nearby picker through the choices, and round again.
func (mm mapModel) cycleNearby(forward bool) (mapModel, mapWant) {
	mm.mapNearbyKm = cycleIn(mapNearbyChoices, itself, mm.mapNearbyKm, forward)
	return mm.wantTouched()
}

// propRadiusChoices are the Propagation mode's near-vertical radii, km
// (D-100, A-18).
var propRadiusChoices = []int{200, 400, 600}

// propRadiusDefaultKm is the near-vertical radius by default (D-100).
const propRadiusDefaultKm = 400

// propRadiusByKm reads the file's number; anything not offered is the
// default.
func propRadiusByKm(km int) int {
	if slices.Contains(propRadiusChoices, km) {
		return km
	}
	return propRadiusDefaultKm
}

// cyclePropRadius moves the near-vertical radius picker, and round again.
func (mm mapModel) cyclePropRadius(forward bool) (mapModel, mapWant) {
	mm.mapPropRadiusKm = cycleIn(propRadiusChoices, itself, mm.mapPropRadiusKm, forward)
	return mm.wantTouched()
}

// propNoiseChoices are the reference circuit's noise environments, quietest
// last, the default first (D-187): P.372's man-made noise categories.
var propNoiseChoices = []ionomaps.Noise{ionomaps.NoiseResidential, ionomaps.NoiseCity, ionomaps.NoiseRural, ionomaps.NoiseQuietRural}

// propNoiseKeys are the file's names for them.
var propNoiseKeys = map[ionomaps.Noise]string{ionomaps.NoiseResidential: "residential", ionomaps.NoiseCity: "city", ionomaps.NoiseRural: "rural", ionomaps.NoiseQuietRural: "quiet-rural"}

// propNoiseByKey reads the file's name; anything not offered is residential.
func propNoiseByKey(key string) ionomaps.Noise {
	for n, k := range propNoiseKeys {
		if k == key {
			return n
		}
	}
	return ionomaps.NoiseResidential
}

// propNoiseKey is the file's name for an environment; residential, the
// default, is written empty.
func propNoiseKey(n ionomaps.Noise) string {
	if n == ionomaps.NoiseResidential {
		return ""
	}
	return propNoiseKeys[n]
}

// propNoiseLabel is an environment as the Settings and the chart say it.
func propNoiseLabel(n ionomaps.Noise) string {
	switch n {
	case ionomaps.NoiseCity:
		return "City"
	case ionomaps.NoiseRural:
		return "Rural"
	case ionomaps.NoiseQuietRural:
		return "Quiet rural"
	}
	return "Residential"
}

// cyclePropNoise moves the noise picker, and round again; the mode, if
// open, asks for its snapshot with the new noise, which go-ionomaps answers
// without a request (D-187).
func (mm mapModel) cyclePropNoise(forward bool) (mapModel, mapWant) {
	mm.mapPropNoise = cycleIn(propNoiseChoices, itself, mm.mapPropNoise, forward)
	return mm.wantTouched()
}

// propRadiusLabel is the radius in the units the description speaks.
func (mm mapModel) propRadiusLabel(e mapEnv) string { return mm.distanceLabel(mm.mapPropRadiusKm, e) }

// toggleRadarSource switches the lower 48's radar between MRMS and IEM (D-83).
func (mm mapModel) toggleRadarSource() (mapModel, mapWant) {
	mm.mapRadarIEM = !mm.mapRadarIEM
	return mm.wantTouched() // the next open asks for the source chosen
}

// radarAheadChoices are the radar loop's hours ahead offered (D-114).
var radarAheadChoices = []int{1, 3, 6, 12}

// radarAheadDefault is three hours (D-114).
const radarAheadDefault = 3

// radarAheadByHours reads the file's number; anything not offered is the
// default.
func radarAheadByHours(h int) int {
	if slices.Contains(radarAheadChoices, h) {
		return h
	}
	return radarAheadDefault
}

// radarAheadLabel is the row's words.
func radarAheadLabel(h int) string { return strconv.Itoa(h) + " hour" + frame.Plural(h) }

// quakeFeeds are the quakes the map draws (D-122), USGS's summary feeds by
// their own names, the default first: M2.5+ the past week or the past day,
// M1.0+ the same.
var quakeFeeds = []string{"2.5_week", "2.5_day", "1.0_week", "1.0_day"}

// quakeFeedDefault is M2.5+ over the past week (D-122).
const quakeFeedDefault = "2.5_week"

// quakeFeedByKey reads the file's word; anything not offered is the default.
func quakeFeedByKey(k string) string {
	if slices.Contains(quakeFeeds, k) {
		return k
	}
	return quakeFeedDefault
}

// quakeFeedLabel is the row's words: "M2.5+, past week".
func quakeFeedLabel(k string) string {
	mag, span, _ := strings.Cut(quakeFeedByKey(k), "_")
	return "M" + mag + "+, past " + span
}

// cycleQuakeFeed moves the quakes picker, and round again.
func (mm mapModel) cycleQuakeFeed(forward bool) (mapModel, mapWant) {
	mm.mapQuakeFeed = cycleIn(quakeFeeds, itself, mm.mapQuakeFeed, forward)
	return mm.wantTouched()
}

// uvCityChoices are the counts of cities UV asks EPA for (D-202).
var uvCityChoices = []int{8, 24, 48}

// uvCitiesDefault is 24 (D-202).
const uvCitiesDefault = 24

// UVCitiesByCount reads a count of UV's cities - the file's, or an ask's:
// anything not offered is the default (D-202).
func UVCitiesByCount(n int) int {
	if slices.Contains(uvCityChoices, n) {
		return n
	}
	return uvCitiesDefault
}

// uvCitiesNote is said under the row (D-23): what the count costs.
func uvCitiesNote(n int) string {
	return "About " + strconv.Itoa(n) + " keyless EPA asks an hour for a view; more cities show UV a little later."
}

// cycleUVCities moves the UV cities picker, and round again (D-202).
func (mm mapModel) cycleUVCities(forward bool) (mapModel, mapWant) {
	mm.mapUVCities = cycleIn(uvCityChoices, itself, mm.mapUVCities, forward)
	return mm.wantTouched()
}

// cycleRadarAhead moves the hours-ahead picker, and round again.
func (mm mapModel) cycleRadarAhead(forward bool) (mapModel, mapWant) {
	mm.mapRadarAhead = cycleIn(radarAheadChoices, itself, mm.mapRadarAhead, forward)
	return mm.wantTouched()
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
func (mm mapModel) radarSourceLabel() string {
	if mm.mapRadarIEM {
		return "IEM (lower 48)"
	}
	return "MRMS"
}

// nearbyLabel is the distance in the units the description speaks.
func (mm mapModel) nearbyLabel(e mapEnv) string { return mm.distanceLabel(mm.mapNearbyKm, e) }

// distanceLabel is a distance in kilometres as the description speaks it:
// kilometres, or miles with the kilometres beside them.
func (mm mapModel) distanceLabel(km int, e mapEnv) string {
	metric := strconv.Itoa(km) + " km"
	if e.Opts.Units != render.UnitF {
		return metric
	}
	return strconv.Itoa(int(math.Round(units.MilesOf(float64(km))))) + " miles (" + metric + ")"
}

// layerOn reports whether a layer is drawn: the listener's choice, or the
// builders' default. A key the registry does not name is drawn. In the
// Propagation mode no weather layer is (D-21, FR-1.4); the choices are kept
// for the weather modes.
func (mm mapModel) layerOn(key string, e mapEnv) bool {
	if mm.mapMode(e) == modePropagation {
		return false
	}
	if key == TemperatureLayer && mm.mapPane.tempAuto && mm.mapMode(e) == modeForecast {
		return true // Forecast mode turned it on (D-103), its alone (D-104)
	}
	return mm.chosen(key, e)
}

// chosen is a layer as the listener chose it: its tick, inside its group.
// mapMode reads the radar through it rather than through layerOn, which
// would ask mapMode again - a cycle P10 forbids (W14, S-3); the radar is
// never Forecast mode's own, so the answer is the same.
func (mm mapModel) chosen(key string, e mapEnv) bool {
	if g, ok := layerGroup[key]; ok && !mm.groupOn(g) {
		return false // its group disabled: drawn nowhere, its tick kept (D-143)
	}
	return mm.ticked(key, e)
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
func (mm mapModel) groupOn(group string) bool {
	on, ok := choiceOf(mm.mapLayerChoice, group)
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
func (mm mapModel) fireMode() string {
	for _, m := range fireModes[1:] {
		if chosenOn(mm.mapLayerChoice, "fire:"+m) {
			return m
		}
	}
	return FireAll
}

// ticked is a layer's own switch, its group aside: the listener's choice, or
// the layer's default.
func (mm mapModel) ticked(key string, e mapEnv) bool {
	if on, ok := choiceOf(mm.mapLayerChoice, key); ok {
		return on
	}
	for _, l := range e.Layers {
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
func (mm mapModel) layerChoices() map[string]bool { return choicesOf(mm.mapLayerChoice) }

// toggleLayer switches the layer under the row's cursor, and asks the
// estimate again.
func (mm mapModel) toggleLayer(layerAt int, e mapEnv) (mapModel, mapWant) {
	layers := e.Layers
	if len(layers) == 0 {
		return mm, mapWant{}
	}
	key := layers[min(layerAt, len(layers)-1)].Key
	choice := mm.layerChoices()
	if choice == nil {
		choice = map[string]bool{}
	}
	on := !mm.ticked(key, e) // its own switch, whatever its group's (D-143)
	choice[key] = on
	if on && slices.Contains(oneTint, key) {
		for _, other := range oneTint {
			if other != key {
				choice[other] = false // one tint at a time, here as in the menu (D-119, D-137, D-139)
			}
		}
	}
	mm.mapLayerChoice = layerChoiceKey(choice)
	return mm.refreshMapCost(e).wantTouched()
}

// stepLayer moves the layers row's cursor a layer up or down: false at
// either end, where ↑↓ leave the group.
func (mm mapModel) stepLayer(down bool, layerAt int, e mapEnv) (mapWant, bool) {
	n := len(e.Layers)
	at := min(layerAt, max(n-1, 0))
	var w mapWant
	switch {
	case down && at < n-1:
		w.Settings.LayerAt = at + 1
	case !down && at > 0:
		w.Settings.LayerAt = at - 1
	default:
		return w, false
	}
	w.Settings.LayerAtSet, w.Settings.Settled = true, true
	return w, true
}

// feedForLayers is the feed with the layers switched off taken out: their
// overlays, an alert category's when it is off, and - with the alert areas
// off - the notes and the missing zones that speak for them.
func (mm mapModel) feedForLayers(f MapFeed, e mapEnv) MapFeed {
	var out MapFeed
	for _, o := range f.Overlays {
		key, rest, _ := strings.Cut(o.ID, "/")
		if !mm.layerOn(key, e) {
			continue
		}
		if cat, _, ok := strings.Cut(rest, "/"); ok && key == AlertLayer && !mm.layerOn(AlertCategorySwitch(cat), e) {
			continue // its category is switched off (D-80)
		}
		out.Overlays = append(out.Overlays, o)
	}
	if mm.layerOn(AlertLayer, e) {
		out.Notes, out.Undrawn, out.InMissing, out.InView, out.AlertsFailed = f.Notes, f.Undrawn, f.InMissing, f.InView, f.AlertsFailed
	}
	return out // the feed's times are not passed on: its overlays are drawn on every frame of both modes (D-275)
}

// mapPrefArrow is ←→ on the scale, the nearby distance and the layers.
func (mm mapModel) mapPrefArrow(forward bool, focus setupRowID, layerAt int, e mapEnv) (mapModel, mapWant, bool) {
	var w mapWant
	switch focus {
	case rowMapScale:
		mm, w = mm.cycleMapScale(forward)
	case rowMapColours:
		mm, w = mm.cycleMapDepth(forward)
	case rowMapPlayback:
		mm, w = mm.cycleMapPlayback(forward)
	case rowMapNearby:
		mm, w = mm.cycleNearby(forward)
	case rowMapRadarSource:
		mm, w = mm.toggleRadarSource()
	case rowMapTempSource:
		mm, w = mm.toggleTempSource()
	case rowMapRainDetail:
		mm, w = mm.toggleRainDetail()
	case rowMapUVCities:
		mm, w = mm.cycleUVCities(forward)
	case rowMapPropRadius:
		mm, w = mm.cyclePropRadius(forward)
	case rowMapPropNoise:
		mm, w = mm.cyclePropNoise(forward)
	case rowMapPropRefresh:
		mm, w = mm.cyclePropRefresh(forward)
	case rowMapPropReadings:
		mm, w = mm.togglePropCorrect()
	case rowMapPropStations:
		mm, w = mm.togglePropStations()
	case rowMapRadarAhead:
		mm, w = mm.cycleRadarAhead(forward)
	case rowMapQuakes:
		mm, w = mm.cycleQuakeFeed(forward)
	case rowMapDetailLevel:
		mm, w = mm.cycleDetailLevel(forward, e).wantTouched()
	case rowMapLayers:
		mm, w = mm.toggleLayer(layerAt, e) // ←→ switch it, as every Enabled / Disabled picker does
	default:
		return mm.toggleDetailRow(focus, e)
	}
	return mm, w, true
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
	// RainFull is Open-Meteo's full density for the rain past NDFD's reach
	// (D-192); a quarter of a box's points otherwise.
	RainFull bool
	// UVCities is how many cities UV asks EPA for (D-202): 8, 24 or 48; any
	// other number is the default, 24.
	UVCities int
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
	// Part is which part of the feed the ask is for (D-268): the alerts,
	// delivered first and on their own, or the other layers after them;
	// FeedWhole, the zero value, asks for both in one answer.
	Part FeedPart
}

// FeedPart is a part of the feed an ask is for (D-268).
type FeedPart int

// The feed's parts: both at once, the alerts alone, the other layers alone.
const (
	FeedWhole FeedPart = iota
	FeedAlerts
	FeedLayers
)

// mapAsk is the ask as the window stands: the watchlist's places, and the
// selected place when it is not one of them - a RECENT or SEARCHED place
// holds its alerts in the recent snapshot, and without it they are not
// drawn (UAT-1 U1-14). The watchlist's snapshot is shared, so the selected
// place joins a copy.
func (mm mapModel) mapAsk(e mapEnv) MapAsk {
	place := mm.mapPlace(e)
	snap := e.Pick.Snap
	if place != nil && e.Pick.Selected >= e.Pick.priorityRows() {
		joined := snapshot.Snapshot{}
		if snap != nil {
			joined = *snap
		}
		joined.Locations = append(append([]snapshot.Location(nil), joined.Locations...), *place)
		snap = &joined
	}
	return MapAsk{Snap: snap, Place: place, View: mm.viewBox(mm.mapBodySize(e)), Region: mm.mapPane.region.Name, RadarIEM: mm.mapRadarIEM,
		Forecast: mm.mapMode(e) == modeForecast, TempNDFD: mm.mapTempNDFD, RainFull: mm.mapRainFull, UVCities: mm.mapUVCities, RadarAhead: mm.mapRadarAhead, QuakeFeed: mm.mapQuakeFeed, Clock: e.Opts.Clock, Buoys: mm.layerOn(BuoyLayer, e), Tides: mm.layerOn(TideLayer, e), Fire: mm.layerOn(FireLayer, e), Quakes: mm.layerOn(QuakeLayer, e), AlertsOff: !mm.layerOn(AlertLayer, e), AlertCategoriesOff: mm.alertCategoriesOff(e), UV: mm.layerOn(UVLayer, e), Air: mm.layerOn(AirLayer, e), FireMode: mm.fireMode(), Fahrenheit: e.Opts.Units == render.UnitF, Anchor: mm.tempAnchor(e)}
}

// detailRowLayer is the detail layer a Map detail row switches, and whether the
// row is one (UAT-1 U1-35: a row each, in mapDetailLayers' order).
func detailRowLayer(id setupRowID) (mapDetailLayer, bool) {
	if id < rowMapDetailBorders || id > rowMapDetailNight {
		return mapDetailLayer{}, false
	}
	return mapDetailLayers()[id-rowMapDetailBorders], true
}

// toggleDetailRow switches the focused detail row's layer: two states, so
// space and either arrow are "the other one", as every on/off row is.
func (mm mapModel) toggleDetailRow(id setupRowID, e mapEnv) (mapModel, mapWant, bool) {
	l, ok := detailRowLayer(id)
	if !ok {
		return mm, mapWant{}, false
	}
	mm, w := mm.setDetail(l.key, !mm.detailOn(l.key), e).wantTouched()
	return mm, w, true
}
