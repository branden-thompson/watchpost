package globalfeed

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// TestRenderListCoverage is the COV metric (brief M2 v1.2.0): every field of
// the frozen render list (02-analysis/data-shape.md §4, column "Render",
// amended at BUILD exit to the ratified record shape — red-team R3-C-05) is
// populated by the parser from the committed probe samples. Fields the feed
// carries only sometimes (PAGER alert, felt/cdi/mmi, depth for a surface
// event, hail/gust parameters, an instruction) are Render-when-present and
// are asserted across the set, not per event.
func TestRenderListCoverage(t *testing.T) {
	quakes := fetchFixture(t, "usgs_significant_week.json", func(c *httpx.Client, base string) Source { return NewUSGS(c, base) })
	q := quakes[0].Quake
	covQuake := map[string]bool{"Mag": q.Mag != nil, "MagType": q.MagType != "", "Status": q.Status != "", "UpdatedAt": !q.UpdatedAt.IsZero(), "Sig": q.Sig != 0,
		"Tsunami": true, // a bool renders either way ("Tsunami no")
		"DepthKm": false, "Alert": false, "Felt": false, "CDI": false, "MMI": false}
	for _, e := range quakes {
		qq := e.Quake
		covQuake["DepthKm"] = covQuake["DepthKm"] || qq.DepthKm > 0
		covQuake["Alert"] = covQuake["Alert"] || qq.Alert != ""
		covQuake["Felt"] = covQuake["Felt"] || qq.Felt > 0
		covQuake["CDI"] = covQuake["CDI"] || qq.CDI != nil
		covQuake["MMI"] = covQuake["MMI"] || qq.MMI != nil
	}
	storms := fetchFixture(t, "nhc_currentstorms.json", func(c *httpx.Client, base string) Source { return NewNHC(c, base) })
	s := storms[0].Tropical
	covStorm := map[string]bool{"Name": s.Name != "", "BinNumber": s.BinNumber != "", "WindKt": s.WindKt != 0, "PressureMb": s.PressureMb != 0,
		"MoveDirDeg": s.MoveDirDeg != 0, "MoveSpeedKt": s.MoveSpeedKt != 0, "LatText": s.LatText != "", "LonText": s.LonText != "", "Basin": s.Basin != "",
		"AdvisoryNum": s.AdvisoryNum != "", "AdvisoryAt": !s.AdvisoryAt.IsZero(), "ForecastNum": false, "DiscussionNum": false}
	for _, e := range storms {
		covStorm["ForecastNum"] = covStorm["ForecastNum"] || e.Tropical.ForecastNum != ""
		covStorm["DiscussionNum"] = covStorm["DiscussionNum"] || e.Tropical.DiscussionNum != ""
	}
	warnings := fetchFixture(t, "nws_active_unfiltered_trimmed.json", func(c *httpx.Client, base string) Source { return NewNWS(c, base) })
	var w *SevereDetail
	for _, e := range warnings {
		if e.Type == "Extreme Heat Warning" {
			w = e.Severe
		}
	}
	if w == nil {
		t.Fatal("the sample carries an Extreme Heat Warning")
	}
	covWarn := map[string]bool{"Description": w.Description != "", "Severity": w.Severity != "", "Urgency": w.Urgency != "", "Certainty": w.Certainty != "",
		"SenderName": w.SenderName != "", "Sent": !w.Sent.IsZero(), "Expires": !w.Expires.IsZero(),
		"Instruction": false, "MaxWindGust": false, "MaxHailSize": false, "EventMotion": false}
	for _, e := range warnings {
		d := e.Severe
		covWarn["Instruction"] = covWarn["Instruction"] || d.Instruction != ""
		covWarn["MaxWindGust"] = covWarn["MaxWindGust"] || d.MaxWindGust != ""
		covWarn["MaxHailSize"] = covWarn["MaxHailSize"] || d.MaxHailSize != ""
		covWarn["EventMotion"] = covWarn["EventMotion"] || d.EventMotion != ""
	}
	for name, cov := range map[string]map[string]bool{"quake": covQuake, "storm": covStorm, "warning": covWarn} {
		missing := 0
		for field, ok := range cov {
			if !ok {
				missing++
				t.Errorf("%s: render-list field %s not populated by any sample", name, field)
			}
		}
		t.Logf("COV %s: %d/%d", name, len(cov)-missing, len(cov))
	}
}
