package snapshot

import (
	"fmt"
	"testing"
	"time"
)

// hoursAsRead is a location's hours as a reader sees them, values and not
// pointers, so a later write through a shared pointer or slice shows.
func hoursAsRead(loc Location) string {
	out := ""
	for _, h := range loc.Hourly { // bounded by the hours (P10-02)
		out += fmt.Sprintf("%v %v %s|", h.Time.Unix(), *h.Temp, h.Condition)
	}
	for _, sec := range loc.ByProvider {
		for _, h := range sec.Hourly {
			out += fmt.Sprintf("%v %v|", h.Time.Unix(), *h.Temp)
		}
	}
	return out
}

// TestAPublishedForecastIsNeverWrittenAfterwards is 0.19.5 Q4.1 (PF-10): the
// snapshot shares each section's hours with the assembler instead of copying
// them, under the rule that Apply replaces a section's hours and never writes
// into them. A snapshot published before a new forecast arrives reads its own
// hours after it, and the next snapshot reads the new ones.
func TestAPublishedForecastIsNeverWrittenAfterwards(t *testing.T) {
	ref := LocationRef{Label: "Oceanside, CA", Zip: "92054", Lat: 33.1959, Lon: -117.3795}
	a := NewAssembler([]LocationRef{ref}, []string{"nws"})
	hour := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	forecast := func(temp float64) Fragment {
		var hrs []Hourly
		for i := range 3 { // bounded (P10-02)
			v := temp + float64(i)
			hrs = append(hrs, Hourly{Time: hour.Add(time.Duration(i) * time.Hour), Temp: &v, Condition: "clear"})
		}
		return Fragment{Provider: "nws", Kind: KindForecast, FetchedAt: hour, PerLocation: map[LocationKey]PartialData{Key(ref): {Hourly: hrs}}}
	}
	a.Apply(forecast(20), nil)
	first := a.Snapshot()
	read := hoursAsRead(first.Locations[0])
	if first.Locations[0].Hourly == nil {
		t.Fatal("the fixture published no hours; this checks nothing")
	}
	a.Apply(forecast(30), nil)
	second := a.Snapshot()
	if got := hoursAsRead(first.Locations[0]); got != read {
		t.Errorf("a snapshot's hours changed after it was published:\n was %s\n now %s", read, got)
	}
	if hoursAsRead(second.Locations[0]) == read {
		t.Error("the next snapshot still reads the old forecast")
	}
}
