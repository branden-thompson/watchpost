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

// alertLayerOffText is what the description says with the alert areas off:
// that they are off, never that nothing is there.
const alertLayerOffText = "Alert areas are switched off in Settings (s), on the Maps tab, so none is drawn or described."

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

// costWarning is the warning's words, or nothing at or under both thresholds.
func costWarning(c MapCost) string {
	if c.Bytes <= mapCostBytes && c.Requests <= mapCostRequests {
		return ""
	}
	return "The map's layers would fetch about " + strconv.FormatFloat(float64(c.Bytes)/1e6, 'f', 1, 64) + " MB in " +
		strconv.Itoa(c.Requests) + " requests a refresh, more than the " + strconv.Itoa(mapCostBytes/1_000_000) + " MB or " +
		strconv.Itoa(mapCostRequests) + " requests this station warns at. Switching a layer off, or drawing this station's alerts only, costs less."
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
	for _, part := range strings.Split(d.mapLayerChoice, ",") {
		if k, v, ok := strings.Cut(part, "="); ok && k == key {
			return v == "on"
		}
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
func (d Dashboard) layerChoices() map[string]bool {
	if d.mapLayerChoice == "" {
		return nil
	}
	out := map[string]bool{}
	for _, part := range strings.Split(d.mapLayerChoice, ",") {
		if k, v, ok := strings.Cut(part, "="); ok {
			out[k] = v == "on"
		}
	}
	return out
}

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
// overlays, and - with the alert areas off - the notes and the missing zones
// that speak for them.
func (d Dashboard) feedForLayers(f MapFeed) MapFeed {
	var out MapFeed
	for _, o := range f.Overlays {
		key, _, _ := strings.Cut(o.ID, "/")
		if d.layerOn(key) {
			out.Overlays = append(out.Overlays, o)
		}
	}
	if d.layerOn(AlertLayer) {
		out.Notes, out.InMissing, out.National = f.Notes, f.InMissing, f.National
	}
	return out
}

// mapPickerRow reports the map's pickers, which have nothing to preview.
func mapPickerRow(id setupRowID) bool {
	return id == rowMapDesc || id == rowMapScale || id == rowMapNearby || id == rowMapScope
}

// mapPrefArrow is ←→ on the scale, the nearby distance and the layers.
func (d Dashboard) mapPrefArrow(forward bool) (Dashboard, bool) {
	switch d.setup.focus {
	case rowMapScale:
		return d.cycleMapScale(forward), true
	case rowMapNearby:
		return d.cycleNearby(forward), true
	case rowMapScope:
		return d.cycleScope(), true
	case rowMapLayers:
		return d.stepLayer(forward), true
	}
	return d, false
}

// mapPrefLines are the scale, nearby and layers rows, with the cost warning
// under the layers when the estimate is over (FR-9.2).
func (d Dashboard) mapPrefLines(o render.Opts, lines []string, at int) ([]string, int) {
	focus, chips := d.setup.focus, newArrowChips(o)
	row := func(id setupRowID, label, cell string) {
		if focus == id {
			at = len(lines)
		}
		lines = append(lines, "  "+setupMark(o, focus == id)+settingLabel(label, focus == id)+"  "+cell)
	}
	row(rowMapScale, "Default scale -", pickerCellW(d.mapScale.Label(), chips, d.pickerFlashFor(rowMapScale), len("County")))
	row(rowMapNearby, "Nearby -", pickerCellW(d.nearbyLabel(), chips, d.pickerFlashFor(rowMapNearby), len("31 miles (50 km)")))
	row(rowMapScope, "Alerts -", pickerCellW(d.mapScope.Label(), chips, d.pickerFlashFor(rowMapScope), len(ScopeNational.Label())))
	row(rowMapLayers, "Layers -", d.layersCell(o, focus == rowMapLayers))
	for _, l := range render.WrapText(costWarning(d.mapCost), 56) {
		lines = append(lines, "    "+settingSupport(l))
	}
	return lines, at
}

// AlertScope is which alerts the map draws (FR-4.3, D-23): the station's own
// places' alerts, or those and the national severe events in the selected
// place's region - never wider than the region (D-28).
type AlertScope int

const (
	ScopeStation  AlertScope = iota // the default: the station's own places' alerts
	ScopeNational                   // and the national severe events in the region
)

// Key is the word the file keeps.
func (s AlertScope) Key() string {
	if s == ScopeNational {
		return "national"
	}
	return "station"
}

// Label is the picker's words.
func (s AlertScope) Label() string {
	if s == ScopeNational {
		return "Plus national severe events"
	}
	return "This station's places"
}

// alertScopeByKey reads the file's word; anything else is the default.
func alertScopeByKey(key string) AlertScope {
	if key == "national" {
		return ScopeNational
	}
	return ScopeStation
}

// cycleScope moves the scope's picker - two states, so either arrow is the
// other one - and asks the estimate again: the scope is what it most depends on.
func (d Dashboard) cycleScope() Dashboard {
	d.mapScope = 1 - d.mapScope
	return d.refreshMapCost().uiTouched()
}

// MapAsk is what the map's feed and its estimate are asked with (0.18.0): the
// station's data, the selected place - whose region bounds the national
// events drawn - and the scope.
type MapAsk struct {
	Snap  *snapshot.Snapshot
	Place *snapshot.Location
	Scope AlertScope
}

// mapAsk is the ask as the window stands.
func (d Dashboard) mapAsk() MapAsk {
	return MapAsk{Snap: d.snap, Place: d.selectedLocation(), Scope: d.mapScope}
}
