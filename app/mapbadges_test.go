package app

// mapbadges_test.go — 0.18.0 D-131 to D-133 (UAT-2 U2-36): each layer names
// its sources as chips for its badge; the full credits, and what never
// changes about a source, are the Status window's.

import (
	"slices"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty"
)

// TestEveryFixedLayerNamesItsSources is D-133: the layers whose sources never
// change say them as they register - the true ones.
func TestEveryFixedLayerNamesItsSources(t *testing.T) {
	want := map[string]string{tty.AlertLayer: "NWS", tty.FireLayer: "NIFC/HMS", quakeLayerKey: "USGS", tty.BuoyLayer: "NDBC", tty.TideLayer: "CO-OPS"}
	got := map[string]string{}
	for _, l := range windowLayers() {
		if len(l.Chips) > 0 {
			got[l.Key] = strings.Join(l.Chips, "/")
		}
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s names %q; want %q", k, got[k], w)
		}
	}
	for _, k := range []string{tty.TemperatureLayer, tty.WaveLayer, tty.RainLayer, tty.RadarLayer} {
		if got[k] != "" {
			t.Errorf("%s names %q at registration; its sources vary, and its answer names them", k, got[k])
		}
	}
}

// TestTheTemperaturesChipsAreItsSources is D-133 for the layers whose source
// varies: Open-Meteo alone [O-METEO]; NDFD with Open-Meteo filling [NDFD]
// and [O-METEO]; NDFD alone [NDFD] - the same for feels-like and wind, which
// ride its requests.
func TestTheTemperaturesChipsAreItsSources(t *testing.T) {
	for _, c := range []struct {
		source string
		filled bool
		want   string
	}{{"Open-Meteo", false, "O-METEO"}, {"Open-Meteo", true, "O-METEO"}, {"NDFD", true, "NDFD/O-METEO"}, {"NDFD", false, "NDFD"}} {
		chips := tempChips(c.source, c.filled)
		for _, k := range []string{tty.TemperatureLayer, tty.FeelsLayer, tty.WindLayer} {
			if got := strings.Join(chips[k], "/"); got != c.want {
				t.Errorf("%s, filled %v: %s names %q; want %q", c.source, c.filled, k, got, c.want)
			}
		}
	}
}

// TestTheStatusWindowHoldsTheFullCredits is D-132 as D-148 leaves it: the
// Status window says what never changes about a source's data - under
// Open-Meteo that each radar frame draws its own hour, under MRMS that its
// colours are approximate - and the credits are About's.
func TestTheStatusWindowHoldsTheFullCredits(t *testing.T) {
	notes := map[string][]string{}
	for _, s := range mapSourceList() {
		notes[s.Name] = append(notes[s.Name], s.Notes...)
	}
	for name, want := range map[string][]string{
		"Open-Meteo":         {tempFrameNote}, // what never changes about the data; the credits are About's (D-148)
		"NOAA / NCEP (MRMS)": {mrmsNote},
	} {
		for _, w := range want {
			if !slices.Contains(notes[name], w) {
				t.Errorf("under %s the Status window says %q; want %q", name, notes[name], w)
			}
		}
	}
}

// TestEverySourceNamedHasAChipOfItsOwn is D-134: every source a layer names,
// fixed or as drawn, has its own chip in the window - none drawn plain.
func TestEverySourceNamedHasAChipOfItsOwn(t *testing.T) {
	names := map[string]bool{"O-METEO": true, "NDFD": true, "MRMS": true, "IEM": true, "HRRR": true} // the temperature's, waves', rain's and radar's as drawn
	for _, l := range windowLayers() {
		for _, c := range l.Chips {
			names[c] = true
		}
	}
	for _, source := range []string{"Open-Meteo", "NDFD"} {
		for _, chips := range tempChips(source, true) {
			for _, c := range chips {
				names[c] = true
			}
		}
	}
	for n := range names {
		if !tty.ChipKnown(n) {
			t.Errorf("%s has no chip of its own", n)
		}
	}
}
