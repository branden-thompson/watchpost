package tty

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// otherThan is the first choice that is not the default.
func otherThan[T comparable](choices []T, def T) T {
	for _, c := range choices {
		if c != def {
			return c
		}
	}
	return def
}

// keptPrefs is every display preference at a value other than its default:
// what a listener who has chosen everything has on file.
func keptPrefs() UIPrefs {
	return UIPrefs{Units: "metric", Clock: "mil", Maps: "off", MapDescription: "instead", MapScale: "county",
		MapNearbyKm: otherThan(mapNearbyChoices, mapNearbyDefaultKm), MapRadarSource: "iem", MapTempSource: "open-meteo",
		MapRainDetail: "full", MapUVCities: 48, MapRadarAhead: otherThan(radarAheadChoices, radarAheadDefault),
		MapQuakeFeed: otherThan(quakeFeeds, quakeFeedDefault), MapLayers: map[string]bool{RadarLayer: false},
		MapDetail: map[string]bool{"parks": true}, MapDetailLevel: "full"}
}

// A SETTING IS SAVED AS IT IS, WHATEVER SAVES IT (UAT-2 U2-61): every display
// preference on file comes back out of the save the map's Overlays menu
// makes - in a session where Settings was never opened - unchanged. The
// clock and the units were read from the Settings window's own copy, filled
// only when it opens: a layer switched before it had been opened wrote 12-hour
// and Fahrenheit over the listener's choice, and the theme the same way.
//
// DERIVED, NOT LISTED: every field of UIPrefs must carry a value here other
// than its zero, so a preference added later is guarded by this test the day
// it is added.
func TestASettingIsSavedAsItIsWhateverSavesIt(t *testing.T) {
	want := keptPrefs()
	was := render.ThemeName()
	t.Cleanup(func() { render.SetTheme(was) })
	want.Theme = otherThan(render.ThemeNames(), render.ThemeNames()[0]) // the window's copy starts at the first
	render.SetTheme(want.Theme)
	v := reflect.ValueOf(want)
	for i := range v.NumField() {
		if f := v.Type().Field(i); v.Field(i).IsZero() {
			t.Fatalf("UIPrefs.%s has no value other than its default here: the guard cannot see it kept", f.Name)
		}
	}
	cfg := Config{Units: want.Units, Clock: want.Clock, Maps: want.Maps, MapDescription: want.MapDescription, MapScale: want.MapScale,
		MapNearbyKm: want.MapNearbyKm, MapRadarSource: want.MapRadarSource, MapTempSource: want.MapTempSource,
		MapRainDetail: want.MapRainDetail, MapUVCities: want.MapUVCities, MapRadarAhead: want.MapRadarAhead,
		MapQuakeFeed: want.MapQuakeFeed, MapLayers: []MapLayer{{Key: RadarLayer, Label: "Radar", On: true}},
		MapLayerChoice: want.MapLayers, MapDetailChoice: want.MapDetail, MapDetailLevel: want.MapDetailLevel}
	var saved *UIPrefs
	cfg.SetUI = func(p UIPrefs) error { saved = &p; return nil }
	d, err := NewDashboard(cfg)
	if err != nil {
		t.Fatal(err)
	}
	d.setup.uiDirty = true // a layer switched in the map's Overlays menu: Settings never opened
	cmd := d.uiApplyCmd()
	if cmd == nil {
		t.Fatal("nothing was saved")
	}
	cmd()
	if saved == nil {
		t.Fatal("the save never reached the file")
	}
	got := *saved
	if !reflect.DeepEqual(got, want) {
		gv := reflect.ValueOf(got)
		for i := range v.NumField() {
			if !reflect.DeepEqual(gv.Field(i).Interface(), v.Field(i).Interface()) {
				t.Errorf("%s was saved as %v; on file it is %v", v.Type().Field(i).Name, gv.Field(i).Interface(), v.Field(i).Interface())
			}
		}
	}
}

// [F] AND [C] KEEP THE UNITS THEY CHOOSE (U2-61): the key's units are saved as
// Settings' are, not left to whatever saves the group next.
func TestTheUnitsKeysKeepTheirChoice(t *testing.T) {
	var saved *UIPrefs
	d, err := NewDashboard(Config{SetUI: func(p UIPrefs) error { saved = &p; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if cmd == nil {
		t.Fatal("[c] saved nothing")
	}
	for _, msg := range msgsOf(t, cmd) {
		_, _ = m.(Dashboard).Update(msg)
	}
	if saved == nil || saved.Units != "metric" {
		t.Errorf("[c] saved %+v; want the units metric", saved)
	}
}
