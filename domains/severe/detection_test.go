package severe

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestDetectionReadsTheSourceThenTheCertainty(t *testing.T) {
	cases := []struct {
		name string
		row  Row
		want string
	}{
		{"radar indicated", Row{Detail: Detail{Severe: &globalfeed.SevereDetail{Certainty: "Observed", Description: "At 845 AM CDT, a severe thunderstorm was located near Olathe. HAZARD: Tornado. SOURCE: Radar indicated rotation. IMPACT: …"}}}, "Radar Indicated"},
		{"radar confirmed", Row{Detail: Detail{Severe: &globalfeed.SevereDetail{Certainty: "Observed", Description: "SOURCE: Radar confirmed tornado."}}}, "Radar Confirmed"},
		{"spotters, location path", Row{Detail: Detail{Alert: &snapshot.Alert{Certainty: "observed", Description: "SOURCE: Trained weather spotters reported a tornado."}}}, "Spotter Reported"},
		{"emergency management", Row{Detail: Detail{Alert: &snapshot.Alert{Certainty: "likely", Description: "SOURCE: Emergency management."}}}, "Emergency Mgmt"},
		{"no source line: the certainty", Row{Detail: Detail{Severe: &globalfeed.SevereDetail{Certainty: "Likely", Description: "A heat advisory is in effect."}}}, "Likely"},
		{"lower-case certainty", Row{Detail: Detail{Alert: &snapshot.Alert{Certainty: "possible"}}}, "Possible"},
		{"unknown certainty reads blank", Row{Detail: Detail{Alert: &snapshot.Alert{Certainty: "Unknown"}}}, ""},
		{"an unrecognised source falls back to the certainty", Row{Detail: Detail{Severe: &globalfeed.SevereDetail{Certainty: "Observed", Description: "SOURCE: A passing airline pilot."}}}, "Observed"},
		{"quake: review status", Row{Detail: Detail{Quake: &globalfeed.QuakeDetail{Status: "reviewed"}}}, "Reviewed"},
		{"tropical: nothing discernible", Row{Detail: Detail{Tropical: &globalfeed.TropicalDetail{}}}, ""},
		{"escapes never leak", Row{Detail: Detail{Severe: &globalfeed.SevereDetail{Certainty: "Obs\x1b[31merved"}}}, "Observed"},
	}
	for _, c := range cases {
		if got := Detection(c.row); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// Upper-casing can lengthen bytes (ȿ → Ȿ is 2 → 3): the SOURCE search and
// the slice use the same string, or the index walks off the end (red-team
// round 4, A-05: a panic from feed prose on the publish goroutine).
func TestDetectionSurvivesCaseLengtheningRunes(t *testing.T) {
	row := Row{}
	row.Detail.Alert = &snapshot.Alert{Certainty: "Observed", Description: strings.Repeat("ȿ", 8) + "SOURCE: spotter reported."}
	if got := Detection(row); got != "Spotter Reported" {
		t.Fatalf("got %q", got)
	}
	row.Detail.Alert.Description = strings.Repeat("ȿ", 8) + "SOURCE:"
	if got := Detection(row); got != "Observed" {
		t.Fatalf("an empty source falls back to the certainty: %q", got)
	}
}
