package stream

import "testing"

// bonsall is the HUM LEAD's own station, and the epicentre the bed's fence was
// ruled against (D-77).
const bonsallLat, bonsallLon = 33.2881, -117.2256

// THE BED'S FENCE HOLDS WHAT THE OPERATOR MAY CHOOSE, NEAREST FIRST.
//
// AND THE NUMBERS ARE WHY THE FENCE IS 100 MILES. Measured from Bonsall: 25
// holds ONE transmitter, 50 holds three, 100 holds eight. A selector with one
// choice in it is not a selector, which is what made the bed's fence a setting
// of its own rather than the service radius.
func TestWithinHoldsTheFencesTransmittersNearestFirst(t *testing.T) {
	table, err := LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	tight := table.Within(bonsallLat, bonsallLon, 25)
	if len(tight) != 1 {
		t.Errorf("a 25-mile bed fence around Bonsall holds one transmitter; got %d", len(tight))
	}
	wide := table.Within(bonsallLat, bonsallLon, 100)
	if len(wide) < 6 {
		t.Fatalf("a 100-mile fence is a real choice; got %d", len(wide))
	}
	last := -1.0
	for _, n := range wide {
		if n.KM/kmPerMile > 100.0001 {
			t.Errorf("%s is %.1f mi out and the fence is 100", n.Callsign, n.KM/kmPerMile)
		}
		if n.KM < last {
			t.Errorf("out of order: %s at %.1f km follows %.1f", n.Callsign, n.KM, last)
		}
		last = n.KM
	}
	// THE WIDER FENCE CONTAINS THE TIGHTER ONE — the property that makes
	// widening a way to get MORE choice rather than a different one.
	if wide[0].Callsign != tight[0].Callsign {
		t.Errorf("the nearest transmitter is the same at any radius; got %q and %q", wide[0].Callsign, tight[0].Callsign)
	}
}

// AND A FENCE WITH NO RADIUS HOLDS NOTHING, which is the rule every other fence
// in this app states: an unset radius is not "everywhere".
func TestABedFenceWithNoRadiusHoldsNothing(t *testing.T) {
	table, err := LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	if got := table.Within(bonsallLat, bonsallLon, 0); len(got) != 0 {
		t.Errorf("no radius admits nothing; got %d", len(got))
	}
	if got := (*Table)(nil).Within(bonsallLat, bonsallLon, 100); got != nil {
		t.Error("a nil table answers nothing rather than panicking")
	}
}

// AN OUT-OF-SERVICE TRANSMITTER IS NOT OFFERED, and this asserts it at a place
// where one actually exists.
//
// THE FIRST VERSION WAS VACUOUS AND A PLANT SAID SO: it swept the fence around
// Bonsall for a status it would never find, so deleting the filter changed
// nothing it could see. The fixture is asserted valid first — the table really
// does hold a dead transmitter here — so the exclusion is what the test measures
// rather than the absence of one.
func TestWithinNeverOffersATransmitterTheTunerWouldRefuse(t *testing.T) {
	table, err := LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	// KEC86 Pensacola, FL is OUT OF SERVICE in the embedded table.
	const lat, lon = 30.588611, -87.070556
	dead, ok := table.ByCallsign("KEC86")
	if !ok || dead.Status != statusOutOfService {
		t.Fatalf("the fixture's transmitter is not out of service; it pins nothing (%+v)", dead)
	}
	got := table.Within(lat, lon, 50)
	if len(got) == 0 {
		t.Fatal("the fence around it holds something, or the exclusion cannot be seen")
	}
	for _, n := range got {
		if n.Callsign == "KEC86" {
			t.Error("a transmitter the tuner refuses was offered as a choice")
		}
		if n.Status == statusOutOfService {
			t.Errorf("%s is out of service and was offered anyway", n.Callsign)
		}
	}
}
