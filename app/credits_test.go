package app

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty"
)

// creditText is every group and line as the window says them, one per line.
func creditText(groups []tty.CreditGroup) string {
	var b strings.Builder
	for _, g := range groups {
		b.WriteString(g.Name + "\n")
		for _, c := range g.Lines {
			b.WriteString(strings.Join([]string{c.Badge, c.What, c.Host, c.Note}, " | ") + "\n")
		}
	}
	return b.String()
}

// EVERY SOURCE IS CREDITED ONCE, WITH WHAT ITS TERMS ASK (W21, D-220, D-228):
// the About window's data sets, a group a provider as the mock draws them -
// each source the app reads once, never a station's list and a map's list
// both; CC BY 4.0 on Open-Meteo and GeoNames, Open-Meteo's change said
// ("interpolated") on every grid drawn from it, the ODbL's OpenStreetMap
// credit, AirNow's condition; the voices under the licence they are published
// under; GitHub, no data set, not credited; the mock's spellings corrected.
func TestEverySourceIsCreditedOnce(t *testing.T) {
	text := creditText(creditGroups())
	for _, want := range []string{"National Weather Service", "National Data Buoy Center", "Tides & Currents", "Wildfire Satellite Hotspots",
		"Tropical Storms", "Transmitter List", "MRMS", "HRRR", "Gridded Forecast Data", "Fire Hotspots, API Key Required", "WFIGS", "wxradio.org & weatherUSA",
		"AirNow", "Envirofacts", "Earthquake Hazards Program", "IEM", "Geocoding", "Basemap Tiles", "Cities & Postal Codes", "rhasspy/piper-voices"} {
		if n := strings.Count(text, want); n != 1 {
			t.Errorf("%q is credited %d times; want once:\n%s", want, n, text)
		}
	}
	for _, want := range []string{"OPEN-METEO (CC BY 4.0)", "GEONAMES (CC BY 4.0)", "© OpenStreetMap contributors (ODbL)", "Preliminary data, not fully verified", "LANCE FIRMS", "(MIT)"} {
		if !strings.Contains(text, want) {
			t.Errorf("the credits lack %q:\n%s", want, text)
		}
	}
	for _, g := range creditGroups() {
		for _, c := range g.Lines {
			if !tty.ChipKnown(c.Badge) || c.What == "" {
				t.Errorf("%q under %s has no chip of its own (D-235): every credit is a chip and a phrase", c.What, g.Name)
			}
			if strings.HasPrefix(g.Name, "OPEN-METEO") && (c.Badge != "O-METEO" || c.Host == "") {
				t.Errorf("Open-Meteo's %q is %q at %q; want its chip and its endpoint (D-235)", c.What, c.Badge, c.Host)
			}
			if strings.HasPrefix(g.Name, "OPEN-METEO") && c.What != "Geocoding" && c.Note != "Interpolated" {
				t.Errorf("Open-Meteo's %q does not say it is interpolated (CC BY 4.0 asks a change be said)", c.What)
			}
		}
	}
	for _, typo := range []string{"INTEDED", "AERONAUTICAL", "UNITED STATED", "MESSONET", "UVIndex", "GitHub", "github.com"} {
		if strings.Contains(text, typo) {
			t.Errorf("the credits say %q", typo)
		}
	}
	want := []string{"NOT INTENDED TO SUBSTITUTE OFFICIAL WARNING SOURCES, DEVICES, OR FOR LIFE SAFETY USE.",
		"FOR LIFE SAFETY, USE NOAA WEATHER RADIO AND COMPATIBLE DEVICES.", "WEATHER RELAYS MAY BE INCOMPLETE OR DELAYED."}
	if w := aboutWarnings(); strings.Join(w, "|") != strings.Join(want, "|") {
		t.Errorf("About's warnings are %q; want D-232's three, in order", w)
	}
}

// TestTheStatusWindowSaysNoCredit is D-148: the Status window is about the
// sources' state and what they are sent; no credit is said there.
func TestTheStatusWindowSaysNoCredit(t *testing.T) {
	for _, s := range mapSourceList() {
		for _, n := range s.Notes {
			if strings.Contains(n, "CC BY") || strings.Contains(n, "Attribution") || strings.Contains(n, "EPA AirNow") || strings.Contains(n, "(c)") || strings.Contains(n, "©") {
				t.Errorf("under %s the Status window credits: %q", s.Name, n)
			}
		}
	}
}
