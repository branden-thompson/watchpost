package config

import "testing"

// THE SERVICE RADIUS IS DEFAULTED AND CLAMPED, NEVER REFUSED (D-72).
//
//	"Service area needs to be a setting and editable: minimum distance is 2
//	 miles, max distance is 50mi."
func TestTheServiceRadiusIsClampedIntoItsRuledBounds(t *testing.T) {
	for _, c := range []struct {
		set, want float64
		why       string
	}{
		{0, DefaultServiceRadiusMi, "unset takes the default"},
		{25, 25, "a ruled value is kept"},
		{2, 2, "the floor is legal"},
		{50, 50, "so is the ceiling"},
		{1, MinServiceRadiusMi, "under the floor is clamped up"},
		{-40, MinServiceRadiusMi, "and so is nonsense"},
		{500, MaxServiceRadiusMi, "over the ceiling is clamped down"},
	} {
		if got := (Broadcaster{ServiceRadiusMi: c.set}).ServiceRadius(); got != c.want {
			t.Errorf("%s: %v -> %v, want %v", c.why, c.set, got, c.want)
		}
	}
}

// AND AN EXISTING STATION KEEPS TRANSMITTING WITHOUT BEING MIGRATED.
//
// Every install that exists has a default location and no transmitter. Without
// the fallback they would all come up with no epicentre, an empty pool and a
// console that shimmers for ever — a migration dressed as a feature.
func TestTheTransmitterFallsBackToTheDefaultLocation(t *testing.T) {
	home := Location{Label: "Bonsall, CA", Lat: 33.2881, Lon: -117.2256}
	c := Config{Locations: []Location{home, {Label: "Vista, CA", Lat: 33.2, Lon: -117.24}}}
	got, ok := c.Station()
	if !ok || got.Label != home.Label {
		t.Errorf("an unset transmitter is the listener's default location; got %+v (%v)", got, ok)
	}
	// AND A TRANSMITTER THAT IS SET WINS, which is the whole point of the split.
	c.Broadcaster.Transmitter = Location{Label: "Fallbrook, CA", Lat: 33.3764, Lon: -117.2511}
	if got, _ := c.Station(); got.Label != "Fallbrook, CA" {
		t.Errorf("the transmitter is the station's own setting; got %+v", got)
	}
	// AND A STATION WITH NOWHERE TO TRANSMIT FROM SAYS SO rather than guessing.
	if _, ok := (Config{}).Station(); ok {
		t.Error("no locations and no transmitter is not a station")
	}
}

// THE BED'S FENCE IS ITS OWN SETTING, NOT THE SERVICE RADIUS (D-77).
//
//	"Bed fence should be another Broadcaster specific setting … I think it
//	 should default to 100" — HUM LEAD, 2026-09-10
//
// THEY MOVE FOR DIFFERENT REASONS: one is who the station is FOR, the other is
// what hardware happens to exist nearby. Around Bonsall a 25-mile search reaches
// ONE transmitter and a 100-mile one reaches eight — so a bed fence derived from
// a 25-mile service radius would leave the selector with nothing to select.
func TestTheBedRadiusIsItsOwnClampedSetting(t *testing.T) {
	for _, c := range []struct {
		set, want float64
		why       string
	}{
		{0, DefaultBedRadiusMi, "unset takes the default"},
		{100, 100, "the ruled default is kept"},
		{25, 25, "the floor is legal"},
		{150, 150, "so is the ceiling"},
		{5, MinBedRadiusMi, "under the floor is clamped up"},
		{900, MaxBedRadiusMi, "over the ceiling is clamped down"},
	} {
		if got := (Broadcaster{BedRadiusMi: c.set}).BedRadius(); got != c.want {
			t.Errorf("%s: %v -> %v, want %v", c.why, c.set, got, c.want)
		}
	}
	// AND IT IS NOT THE SERVICE RADIUS. Setting one must never move the other.
	b := Broadcaster{ServiceRadiusMi: 25}
	if b.BedRadius() != DefaultBedRadiusMi {
		t.Errorf("the service radius moved the bed's fence to %v", b.BedRadius())
	}
	if got := (Broadcaster{BedRadiusMi: 150}).ServiceRadius(); got != DefaultServiceRadiusMi {
		t.Errorf("the bed's fence moved the service radius to %v", got)
	}
}
