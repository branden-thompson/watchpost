package app

import (
	"reflect"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/render"
)

// A SAVED SETTING IS THE SETTING AFTER A RESTART (UAT-2 U2-61): every display
// preference the window saves is written to the file and handed back to the
// window the next run - the other half of the tty's guard, which proves the
// window saves what is in effect.
//
// DERIVED, NOT LISTED: every field of UIPrefs carries a value here other than
// its zero, and each is compared by the name the window's Config gives it.
func TestASavedSettingIsTheSettingAfterARestart(t *testing.T) {
	want := tty.UIPrefs{Theme: render.ThemeNames()[len(render.ThemeNames())-1], Units: "metric", Clock: "mil", Maps: "off", MapDescription: "instead", MapScale: "county",
		MapNearbyKm: 25, MapRadarSource: "iem", MapTempSource: "open-meteo", MapRainDetail: "full", MapUVCities: 48,
		MapRadarAhead: 6, MapQuakeFeed: "1.0_day", MapLayers: map[string]bool{"radar": false},
		MapDetail: map[string]bool{"parks": true}, MapDetailLevel: "full"}
	v := reflect.ValueOf(want)
	for i := range v.NumField() {
		if f := v.Type().Field(i); v.Field(i).IsZero() {
			t.Fatalf("UIPrefs.%s has no value other than its default here: the guard cannot see it kept", f.Name)
		}
	}
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) }) // the save applies the theme, process-wide
	withConfigFile(t)
	if err := setUIHook(want); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{}
	got := lp.ttyConfig("t", Options{}, false, cfg, nil, nil, nil, nil, nil, nil)
	back := map[string]any{"Theme": cfg.Theme, "Units": got.Units, "Clock": got.Clock, "Maps": got.Maps, "MapDescription": got.MapDescription,
		"MapScale": got.MapScale, "MapNearbyKm": got.MapNearbyKm, "MapRadarSource": got.MapRadarSource,
		"MapTempSource": got.MapTempSource, "MapRainDetail": got.MapRainDetail, "MapUVCities": got.MapUVCities,
		"MapRadarAhead": got.MapRadarAhead, "MapQuakeFeed": got.MapQuakeFeed, "MapLayers": got.MapLayerChoice,
		"MapDetail": got.MapDetailChoice, "MapDetailLevel": got.MapDetailLevel}
	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		b, ok := back[name]
		if !ok {
			t.Errorf("UIPrefs.%s is not handed back to the window here: the guard cannot see it kept", name)
			continue
		}
		if !reflect.DeepEqual(b, v.Field(i).Interface()) {
			t.Errorf("%s came back as %v; it was saved as %v", name, b, v.Field(i).Interface())
		}
	}
}
