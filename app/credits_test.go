package app

import (
	"strings"
	"testing"
)

// TestCreditsCoverEverySource is UAT 75 (OQ-15) with 0.18.0 D-148: every
// source the build reads is credited in About - the station's and the map's
// - each CC BY source with its licence, the map's basemap with the ODbL;
// then the relays' condition of use and the safety framing, last.
func TestCreditsCoverEverySource(t *testing.T) {
	station := strings.Join(credits(), "\n")
	for _, want := range []string{"National Weather Service", "Data Buoy Center", "Tides & Currents", "GeoNames", "Open-Meteo", "NWR transmitter", "wxradio.org"} {
		if !strings.Contains(station, want) {
			t.Errorf("the station's credits lack %q:\n%s", want, station)
		}
	}
	if strings.Count(station, "CC BY 4.0") != 2 {
		t.Errorf("both of the station's CC BY sources must carry their licence:\n%s", station)
	}
	maps := strings.Join(mapCredits(), "\n")
	for _, want := range []string{"OpenFreeMap", "OpenStreetMap contributors (ODbL)", "MRMS", "Iowa Environmental Mesonet", "HRRR", "NDFD",
		"Temperature, feels-like, wind and UV: Open-Meteo.com (CC BY 4.0)", "Rain and snow", "Waves", "Air quality", "AirNow", "WFIGS", "HMS", "USGS", "Data Buoy Center", "Tides & Currents"} {
		if !strings.Contains(maps, want) {
			t.Errorf("the map's credits lack %q:\n%s", want, maps)
		}
	}
	notes := aboutNotes()
	if len(notes) != 3 || !strings.Contains(notes[0], "not for life-safety") || notes[2] != SafetyNext {
		t.Errorf("About's closing lines are %q; want the relays' condition of use, then the safety framing", notes)
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
