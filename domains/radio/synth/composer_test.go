package synth

import (
	"github.com/branden-thompson/watchpost/platform/render"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// std is the composer over the built-in scripts, for the tests.
var std = Composer{}

// --- 0.14.0 P1 Task 1.2: Reports ---

// TestComposeWithZeroReportsIsTodaysBroadcast is FR-2's anchor. Reports exists
// so the maritime report can join a cycle without a ninth positional parameter
// (RS-12); the property that makes that safe is that its ZERO VALUE composes
// exactly what 0.13.0 composed. Every later batch is measured against this.
func TestComposeWithZeroReportsIsTodaysBroadcast(t *testing.T) {
	loc := snapshot.Location{Label: "Oceanside, CA", Harmonized: snapshot.Conditions{
		Condition: "partly_cloudy", Temp: f64(22.8), HumidityPct: f64(66),
		Source: snapshot.SourceInfo{Provider: "nws"}},
		Alerts: []snapshot.Alert{{ID: "a1", Headline: "Heat Advisory", Description: "* WHAT...Hot."}}}
	now := time.Date(2026, 8, 24, 16, 5, 0, 0, time.UTC)
	products := []Product{{ID: "p1", Type: "ZFP", Text: ".TONIGHT...Mostly clear. Lows 66 to 69.\n\n$$"}}
	compose := func(r Reports) []Segment {
		return std.Compose(loc, products, now, true, "Samantha", Station{Callsign: "KEC62"}, r, render.Clock12)
	}

	// The zero value and an explicitly-zeroed struct are the same broadcast:
	// nothing about the field's presence changes the cycle.
	zero, explicit := compose(Reports{}), compose(Reports{Fire: FireReport{}, Seismic: SeismicReport{}})
	if len(zero) != len(explicit) {
		t.Fatalf("zero Reports composed %d segments, explicitly-zeroed composed %d", len(zero), len(explicit))
	}
	for i := range zero {
		if zero[i] != explicit[i] {
			t.Fatalf("segment %d differs:\n zero: %+v\n explicit: %+v", i, zero[i], explicit[i])
		}
	}

	// The 0.13.0 shape, pinned by key, order and pause: lead (2 s) · span ·
	// conditions · the alert · the product · the tail (1 s before it). No
	// report segments appear, because there is no report data.
	wantKeys := []string{"lead:Oceanside, CAKEC62", "lead-span:2026-08-24", "wx:", "alert:a1:0", "ZFP:p1:0", "tail:" + VoiceToken}
	if len(zero) != len(wantKeys) {
		got := make([]string, len(zero))
		for i, s := range zero {
			got[i] = s.Key
		}
		t.Fatalf("zero Reports composed %d segments, want %d: %v", len(zero), len(wantKeys), got)
	}
	for i, want := range wantKeys {
		if !strings.HasPrefix(zero[i].Key, want) {
			t.Errorf("segment %d key = %q, want prefix %q", i, zero[i].Key, want)
		}
	}
	if zero[0].Pause != leadPause {
		t.Errorf("the safety notice is followed by %v, want the lead pause %v (UAT 112.3)", zero[0].Pause, leadPause)
	}
	if last := zero[len(zero)-2]; last.Pause != tailPause {
		t.Errorf("the segment before the sign-off pauses %v, want the tail pause %v (UAT 115)", last.Pause, tailPause)
	}
}

// TestComposeStillPlacesTheReportsCarriedInReports proves the struct is wiring,
// not just shape: a fire report and a seismic report still land between the
// forecast and the sign-off, each opened by the two-second report pause.
func TestComposeStillPlacesTheReportsCarriedInReports(t *testing.T) {
	loc := snapshot.Location{Label: "Ridgecrest, CA"}
	now := time.Date(2026, 8, 24, 16, 5, 0, 0, time.UTC)
	sr := SeismicReport{Known: true, Lat: 35.62, Lon: -117.67, State: snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{
		quake(4.2, 30, 8, "NE", 2*time.Hour, now),
	}}}
	segs := std.Compose(loc, nil, now, true, "Samantha", Station{}, Reports{Seismic: sr}, render.Clock12)

	var seismicAt, tailAt = -1, -1
	for i, s := range segs {
		if strings.HasPrefix(s.Key, "seismic") && seismicAt < 0 {
			seismicAt = i
		}
		if strings.HasPrefix(s.Key, "tail:") {
			tailAt = i
		}
	}
	if seismicAt < 0 {
		t.Fatal("Reports{Seismic: …} composed no seismic segments — the struct is not wired through")
	}
	if seismicAt >= tailAt {
		t.Fatalf("the seismic report is at %d, the sign-off at %d — it must come before the sign-off", seismicAt, tailAt)
	}
	if got := segs[seismicAt-1].Pause; got != reportPause {
		t.Errorf("the segment before the seismic report pauses %v, want the report pause %v (UAT 115)", got, reportPause)
	}
}

// TestComposeOrdersMarineFireAndSeismic is 0.19.5 Q2.4 (RN-5): the ruled order
// (MVS-D-18) is products, maritime, fire, seismic, then the sign-off; each
// report opens after two seconds of air, and the marine report's first
// segment carries its manifest line.
func TestComposeOrdersMarineFireAndSeismic(t *testing.T) {
	now := time.Date(2026, 8, 30, 15, 0, 0, 0, time.UTC)
	mr := marineFixture(now)
	fr := FireReport{Known: true, HotspotsKnown: true, IncidentsKnown: true, RadiusKm: 25, IncidentRadiusKm: 50,
		Sources: []string{"NOAA's Hazard Mapping System"}, State: snapshot.FireState{AsOf: now, HotspotsAsOf: now, IncidentsAsOf: now}}
	sr := SeismicReport{Known: true, Lat: 33.2, Lon: -117.3, State: snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{
		quake(4.2, 30, 8, "NE", 2*time.Hour, now),
	}}}
	segs := std.Compose(snapshot.Location{Label: "Oceanside, CA"}, nil, now, true, "Samantha", Station{},
		Reports{Maritime: mr, Fire: fr, Seismic: sr}, render.Clock12)
	first := func(prefix string) int {
		for i, s := range segs {
			if strings.HasPrefix(s.Key, prefix) {
				return i
			}
		}
		t.Fatalf("no %q segment was composed", prefix)
		return -1
	}
	order := []int{first("maritime"), first("fire"), first("seismic"), first("tail:")}
	for i := 1; i < len(order); i++ {
		if order[i-1] >= order[i] {
			t.Fatalf("the reports are out of the ruled order (maritime, fire, seismic, sign-off): %v", order)
		}
	}
	for _, at := range order[:3] {
		if got := segs[at-1].Pause; got != reportPause {
			t.Errorf("the segment before %q pauses %v; want the report pause %v", segs[at].Key, got, reportPause)
		}
	}
	if m := segs[order[0]]; m.Source != marineSource || m.Detail != marineSpan(mr, now) {
		t.Errorf("the marine report opens with manifest %q / %q; want %q / %q", m.Source, m.Detail, marineSource, marineSpan(mr, now))
	}
}

// A FAILED ALERT FEED IS SPOKEN, NOT OMITTED (FR-8.10).
//
// THE TWO STATES ARE THE SAME VALUE AND OPPOSITE FACTS. `loc.Alerts` is empty on
// a quiet day and empty when the feed returned a 502, and from the air the
// listener cannot tell them apart: the report runs lead, conditions, forecast,
// sign-off, and sounds complete. FR-8.10 exists because "the worst day is severe
// weather with a failing feed", and its exit is that the spoken output says the
// data is unavailable or omits it — never presents it as current.
//
// THE CONTROL IS THE HALF THAT MATTERS. A test asserting only that the sentence
// appears would pass on a composer that said it on every read, which would train
// a listener to ignore it inside a day. So the quiet day is asserted beside it.
func TestAFailedAlertFeedIsSaidAloudAndAQuietDayIsNot(t *testing.T) {
	loc := snapshot.Location{Label: "Oceanside, CA"}
	at := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

	spoken := func(r Reports) string {
		var b strings.Builder
		for _, s := range std.Compose(loc, nil, at, true, "Alpha", Station{}, r, render.Clock12) {
			b.WriteString(s.Text + " ")
		}
		return b.String()
	}

	const says = "Hazard information is unavailable"
	if got := spoken(Reports{HazardsUnavailable: true}); !strings.Contains(got, says) {
		t.Errorf("the alert feed failed and the report does not say so — it sounds like a quiet day:\n%s", got)
	}
	// AND A QUIET DAY STAYS QUIET.
	if got := spoken(Reports{}); strings.Contains(got, says) {
		t.Errorf("no feed failed and the report claimed hazard data was unavailable:\n%s", got)
	}
}
