package tty

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"
)

// everySettingChanged is a Config with every Setting NewDashboard reads set
// away from its default: what TestEveryConfigFieldReachesTheModel builds the
// model from beside the zero Config.
func everySettingChanged() Config {
	return Config{
		Maps: "off", MapDescription: "instead", MapRadarSource: "iem", MapTempSource: tempSourceOpenMeteo,
		MapRainDetail: rainDetailFull, MapUVCities: 48, MapPropRadius: 600, MapPropNoise: "rural",
		MapPropRefresh: "on-demand", MapPropNoReadings: noReadingsCorrected, MapPropStations: noStationsKey,
		MapColourDepth: "16", MapPlaybackMs: 1000, ReduceMotion: true, MapRadarAhead: 6, MapQuakeFeed: "1.0_day",
		MapScale: "county", MapNearbyKm: 50, MapLayerChoice: map[string]bool{"radar": true},
		MapDetailChoice: map[string]bool{"roads": false}, MapDetailLevel: "standard",
		Units: "metric", Clock: "24h", Voice: "Samantha",
	}
}

// modelFields is the model's own fields as name=value lines, sorted: every
// field but the stored Config, the memos and the clock, reached through any
// embedded struct by its promoted name, so the lines do not depend on how the
// fields are grouped.
func modelFields(t *testing.T, d Dashboard) string {
	t.Helper()
	skip := map[string]bool{"cfg": true, "memo": true, "mmemo": true, "now": true, "tickerScrolls": true}
	var lines []string
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		for i := range v.NumField() { // bounded by the struct (P10-02)
			f := v.Type().Field(i)
			if skip[f.Name] {
				continue
			}
			fv := reflect.NewAt(f.Type, unsafe.Pointer(v.Field(i).UnsafeAddr())).Elem()
			if f.Anonymous && f.Type.Kind() == reflect.Struct {
				walk(fv)
				continue
			}
			if f.Type.Kind() == reflect.Func {
				lines = append(lines, fmt.Sprintf("%s=func:%v", f.Name, !fv.IsNil()))
				continue
			}
			lines = append(lines, fmt.Sprintf("%s=%+v", f.Name, fv.Interface()))
		}
	}
	walk(reflect.ValueOf(&d).Elem())
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

// TestEveryConfigFieldReachesTheModel is 0.19.5 Q6.4's pin (D-14, A-16),
// written before NewDashboard's Settings move into their grouped builders:
// the model NewDashboard makes from the zero Config, and from one with every
// Setting changed, field for field. Recorded on the shipped constructor, so a
// builder that drops or misreads a Setting changes a line. A field the model
// does not have, or one that moved into an embedded struct, reads the same.
func TestEveryConfigFieldReachesTheModel(t *testing.T) {
	var out strings.Builder
	for _, c := range []struct {
		name string
		cfg  Config
	}{{"the zero Config", Config{}}, {"every Setting changed", everySettingChanged()}} {
		d, err := NewDashboard(c.cfg)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		fmt.Fprintf(&out, "== %s\n%s", c.name, modelFields(t, d))
	}
	checkGolden(t, "model-from-config.golden", out.String())
	zero, _ := NewDashboard(Config{})
	full, _ := NewDashboard(everySettingChanged())
	if modelFields(t, zero) == modelFields(t, full) {
		t.Fatal("changing every Setting changed nothing in the model; the fixture is wrong")
	}
}

// TestThePrefsSavedAreTheSettingsInEffect is 0.19.5 Q6.4's other pin (A-16):
// the display preferences a window saves are the Settings the model was built
// with, for the zero Config and for every Setting changed. Recorded before
// uiForSave is laid out one field per line. The theme is the process's, so it
// is left out.
func TestThePrefsSavedAreTheSettingsInEffect(t *testing.T) {
	var out strings.Builder
	for _, cfg := range []Config{{}, everySettingChanged()} {
		d, err := NewDashboard(cfg)
		if err != nil {
			t.Fatal(err)
		}
		p := d.uiForSave()
		p.Theme = ""
		fmt.Fprintf(&out, "%+v\n", p)
	}
	checkGolden(t, "prefs-saved.golden", out.String())
}
