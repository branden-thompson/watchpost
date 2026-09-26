package tty

// map_prefs.go — the Maps tab's other Settings (0.18.0 batch 9): the scale
// the map opens at (W1.11, W4.3, FR-2.3), how near an alert's edge must be to
// be called near (W9.2, FR-7.4), the layers the app's registry names (W1.11,
// W1.13, R-9.2) and the warning when they would cost a lot (W1.14, FR-9.2).

import (
	"math"
	"sort"
	"strconv"
	"strings"

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

// alertLayerOffText is what the description says with the alert areas off:
// that they are off, never that nothing is there.
const alertLayerOffText = "Alert areas are switched off in the Overlays menu (O), so none is drawn or described." // D-76: switched at the map

// MapCost is what one refresh of the map's data would fetch (FR-9.2).
type MapCost struct {
	Bytes    int64
	Requests int
}

// The cost warning's thresholds (FR-9.2, D-43): more than either is said.
const (
	mapCostBytes    = 2_000_000
	mapCostRequests = 40
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
			" Requests | Switch off layers or zoom in for a better experience."
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
	at := 0
	for i, c := range mapNearbyChoices {
		if c == d.mapNearbyKm {
			at = i
		}
	}
	step := 1
	if !forward {
		step = len(mapNearbyChoices) - 1
	}
	d.mapNearbyKm = mapNearbyChoices[(at+step)%len(mapNearbyChoices)]
	return d.uiTouched()
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
	choice[key] = !d.layerOn(key)
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
	return id == rowMapDesc || id == rowMapScale || id == rowMapNearby || id == rowMapDetailLevel
}

// mapPrefArrow is ←→ on the scale, the nearby distance and the layers.
func (d Dashboard) mapPrefArrow(forward bool) (Dashboard, bool) {
	switch d.setup.focus {
	case rowMapScale:
		return d.cycleMapScale(forward), true
	case rowMapNearby:
		return d.cycleNearby(forward), true
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
	return MapAsk{Snap: snap, Place: place, View: d.viewBox(d.mapBodySize())}
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
