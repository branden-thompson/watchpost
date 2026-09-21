package nws

import (
	"strconv"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/plaintext"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// 0.13.0 (NFR-5, red-team S2): the location path was unbounded while the
// ticker path was not; every CAP field now has a bound, and the issuing
// office rides along for the superseded guard (NFR-12).
func TestMapAlertBoundsEveryFieldAndKeepsSender(t *testing.T) {
	long := strings.Repeat("y", 10_000)
	pr := alertProps{ID: "urn:oid:1", Event: long, Headline: long, Description: long, Instruction: long, SenderName: "NWS Test", AreaDesc: long}
	for i := 0; i < 200; i++ {
		pr.AffectedZones = append(pr.AffectedZones, "https://api.weather.gov/zones/forecast/Z"+strconv.Itoa(i))
		pr.References = append(pr.References, struct {
			ID string `json:"@id"`
		}{ID: "urn:oid:r" + strconv.Itoa(i)})
	}
	perKey := map[snapshot.LocationKey][]snapshot.Alert{}
	mapAlert(pr, nil, map[string][]snapshot.LocationKey{"Z1": {"k"}}, perKey)
	if len(perKey["k"]) != 1 {
		t.Fatalf("expected one alert on k, got %d", len(perKey["k"]))
	}
	a := perKey["k"][0]
	if len([]rune(a.Event)) > plaintext.MaxFieldRunes || len([]rune(a.Description)) > maxProseRunes || len(a.AffectedZones) > maxZones || len(a.References) > plaintext.MaxListLen {
		t.Fatalf("unbounded: event %d desc %d zones %d refs %d", len(a.Event), len(a.Description), len(a.AffectedZones), len(a.References))
	}
	if a.SenderName != "NWS Test" {
		t.Fatalf("SenderName lost: %q", a.SenderName)
	}
}

// A real CAP alert can affect more zones than the record keeps (the fixture's
// Hydrologic Outlook has 81; Winter Storm Warnings span 50–100): the match
// runs over the full list, the bound applies to the retained copy only
// (0.13.0 red-team R3-A-01 — the earlier test used zone index 1 and passed
// for the wrong reason).
func TestMapAlertAttachesWhenTheZoneIsBeyondTheListCap(t *testing.T) {
	// **The bound is maxZones, and it sits above every alert measured** - the
	// largest live was forty-two zones, a Winter Storm Warning can name eighty.
	// That matters because the zone ids are what an alert's ground is resolved
	// from, so a bound inside the real range is a piece of the map missing. This
	// drives deliberately past maxZones to show a bound still exists, and that
	// the match finds a zone lying beyond it.
	pr := alertProps{ID: "urn:oid:1", Event: "Winter Storm Warning"}
	for i := 0; i < maxZones+40; i++ {
		pr.AffectedZones = append(pr.AffectedZones, "https://api.weather.gov/zones/forecast/Z"+strconv.Itoa(i))
	}
	perKey := map[snapshot.LocationKey][]snapshot.Alert{}
	// The tracked zone is past the bound: the match must still find it.
	mapAlert(pr, nil, map[string][]snapshot.LocationKey{"Z" + strconv.Itoa(maxZones+10): {"olathe"}}, perKey)
	if len(perKey["olathe"]) != 1 {
		t.Fatalf("alert affecting the tracked zone (past the retained bound) was dropped: %d attached", len(perKey["olathe"]))
	}
	if got := len(perKey["olathe"][0].AffectedZones); got != maxZones {
		t.Fatalf("the retained zone list is bounded to %d, got %d", maxZones, got)
	}
}
