package app

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/history"
)

// A BOX THE FILL DRAWS KEEPS THE FILL'S OWN PAST: where NDFD refuses a box
// and Open-Meteo draws it, Open-Meteo answers the hours before the current
// one itself, so nothing recorded is replayed under it and the chips carry no
// RECORDED - however much the history holds for that box.
func TestABoxTheFillDrawsReplaysNothingRecorded(t *testing.T) {
	now := tempNow
	anchor := now.Truncate(time.Hour)
	store := history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly)
	box := fieldBoxes(tempAsk(false).Region, tempAsk(false).View)[0]
	lat := temperature.NDFDLatticeFor(box.Name, box.Box)
	f := &fakeHour{}
	for _, back := range []time.Duration{2 * time.Hour, time.Hour} {
		s, _ := f.hour(context.Background(), lat, anchor.Add(-back))
		rec, ok := ndfdRecord(s, history.Key{Source: "ndfd", Place: box.Name}, anchor.Add(-back))
		if !ok || !store.Put(ndfdHourly.Name, rec) {
			t.Fatal("could not seed the history")
		}
	}
	ndfd := &fakeTemp{name: "NDFD", now: now, failed: true}
	om := &fakeTemp{name: "Open-Meteo", now: now}
	got := buildTemperature(context.Background(), ndfd, om, tempAsk(false), now, &fallback{past: recordedHour(store)})
	if len(got.Overlays) == 0 {
		t.Fatal("the fill drew nothing; this test measures nothing")
	}
	if chips := got.Chips[tty.TemperatureLayer]; slices.Contains(chips, recordedChip) {
		t.Errorf("the chips are %v; a box Open-Meteo drew replays nothing recorded", chips)
	}
}
