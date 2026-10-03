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
			b.WriteString(strings.Join([]string{c.Abbr, c.What, c.Host, c.Note}, " | ") + "\n")
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
		"Tropical Storms", "Transmitter List", "MRMS", "HRRR", "National Digital Forecast Database", "Fire Hotspots, API Key Required", "WFIGS", "wxradio.org & weatherUSA",
		"AirNow", "Envirofacts", "Earthquake Hazards Program", "IEM", "Geocoding", "Basemap Tiles", "Cities & Postal Codes", "rhasspy/piper-voices"} {
		if n := strings.Count(text, want); n != 1 {
			t.Errorf("%q is credited %d times; want once:\n%s", want, n, text)
		}
	}
	for _, want := range []string{"OPEN-METEO (CC BY 4.0)", "GEONAMES (CC BY 4.0)", "© OpenStreetMap contributors (ODbL)", "preliminary data, not fully verified", "LANCE FIRMS", "(MIT)"} {
		if !strings.Contains(text, want) {
			t.Errorf("the credits lack %q:\n%s", want, text)
		}
	}
	for _, g := range creditGroups() {
		if !strings.HasPrefix(g.Name, "OPEN-METEO") {
			continue
		}
		for _, c := range g.Lines {
			if c.What != "Geocoding" && !strings.HasSuffix(c.What, "Interpolated") {
				t.Errorf("Open-Meteo's %q does not say it is interpolated (CC BY 4.0 asks a change be said)", c.What)
			}
		}
	}
	for _, typo := range []string{"INTEDED", "AERONAUTICAL", "UNITED STATED", "MESSONET", "UVIndex", "GitHub", "github.com"} {
		if strings.Contains(text, typo) {
			t.Errorf("the credits say %q", typo)
		}
	}
	w := aboutWarnings()
	if len(w) != 4 || !strings.Contains(w[0], "NOT INTENDED AS A SUBSTITUTE") || !strings.Contains(w[3], "NOAA WEATHER RADIO") {
		t.Errorf("About's warnings are %q; want the mock's three and the life-safety pointer last (D-229)", w)
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
