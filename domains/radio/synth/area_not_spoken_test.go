package synth

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestAnAlertsAreaIsNeverSpoken is a guard, not a discovery. Today the composer
// reads an alert's headline and description by name, so the polygon this
// release adds cannot reach the air whatever it contains. **This test exists so
// that stays true**: a later change that walked an alert's fields, or a script
// template that reached for the area, would put raw coordinates on a broadcast.
//
// It is the same rule the text path already keeps, where LAT...LON blocks are
// stripped before narration (UAT 81).
func TestAnAlertsAreaIsNeverSpoken(t *testing.T) {
	// Coordinates chosen so their digits appear nowhere else in the cycle.
	area := geo.Shape{{
		{Lon: -85.7431, Lat: 41.6829}, {Lon: -85.7432, Lat: 41.6830},
		{Lon: -85.7433, Lat: 41.6831}, {Lon: -85.7431, Lat: 41.6829},
	}}
	loc := snapshot.Location{
		Label: "Fort Wayne", Lat: 41.08, Lon: -85.14, TZ: "America/New_York",
		Alerts: []snapshot.Alert{{
			ID: "a1", Event: "Flood Warning", Severity: "severe",
			Headline:    "Flood Warning issued for Allen county",
			Description: "Low lying areas will flood.",
			Area:        area,
		}},
	}
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	segs := std.Compose(loc, nil, now, true, "Samantha", Station{Callsign: "KEC62"}, Reports{}, render.Clock12)

	var spoken strings.Builder
	for _, s := range segs {
		spoken.WriteString(s.Text)
		spoken.WriteByte(' ')
	}
	said := spoken.String()
	if !strings.Contains(said, "Flood Warning") {
		t.Fatalf("the alert was not narrated at all, so this guard proves nothing:\n%s", said)
	}
	for _, digits := range []string{"85.74", "41.68", "7431", "6829"} {
		if strings.Contains(said, digits) {
			t.Errorf("an alert's coordinates reached the air (%q):\n%s", digits, said)
		}
	}
}
