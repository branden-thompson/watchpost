package tty

import (
	"slices"
	"testing"
)

// A LAYER'S NOTES SHOW WHILE THAT LAYER IS ON (UAT-2 U2-52, D-203): the map's
// notes were shown only while Temperature or Feels like was, and Air quality,
// UV, Rain and Waves each turn those off - so what they said was never seen.
// Each layer's notes now go with it; temperature's go with temperature.
func TestALayersNotesShowWhileItIsOn(t *testing.T) {
	d := mapDash(t, Config{MapLayers: []MapLayer{
		{Key: AirLayer, Label: "Air quality", On: true}, {Key: UVLayer, Label: "UV", On: false},
		{Key: TemperatureLayer, Label: "Temperature", On: false}, {Key: FeelsLayer, Label: "Feels like", On: false}, {Key: RadarLayer, Label: "Radar", On: true}}}) // Radar mode: temperature is the listener's alone (D-103)
	d.mapPane.temp.Notes = []string{"temperature's note"}
	d.mapPane.temp.LayerNotes = map[string][]string{AirLayer: {"air's note"}, UVLayer: {"uv's note"}}
	got := d.mapNotesNow()
	if !slices.Contains(got, "air's note") {
		t.Errorf("Air quality is on and its note is not shown: %v", got)
	}
	if slices.Contains(got, "uv's note") || slices.Contains(got, "temperature's note") {
		t.Errorf("a layer that is off has its note shown: %v", got)
	}
}
