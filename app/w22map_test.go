package app

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/history"
)

// NDFD'S RAIN AND SNOW TOTALS ARE KEPT (W22.2, D-226): NDFD serves no history
// of its forecasts, so each day it gave a box goes into the history.
func TestNDFDsTotalsAreKept(t *testing.T) {
	ask := tempAsk(true)
	store := history.Open(t.TempDir(), func() time.Time { return tempNow }, ndfdRainDays)
	withRainDays(context.Background(), tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), ask, tempNow,
		&rainRescue{ndfd: temperature.NewNDFD(&ndfdTotalsGet{}, ""), store: store})
	box := fieldBoxes(ask.Region, ask.View)[0].Name
	rec, ok := store.Get(ndfdRainDays.Name, history.Key{Source: "ndfd", Place: box}, dayStart(askAnchor(ask, tempNow), 0))
	if !ok || len(rec.Values["rain"]) == 0 || len(rec.Values["snow"]) == 0 {
		t.Fatalf("today's NDFD totals for %s were not kept: %+v, %v", box, rec, ok)
	}
}

// OPEN-METEO'S TEMPERATURE IS KEPT (W22.2, D-231): each hour it gave a box,
// up to the current one - as the source or as the fill - goes into the
// history; NDFD's own answer is the recorder's, not this dataset's.
func TestOpenMeteosTemperatureIsKept(t *testing.T) {
	ask := tempAsk(false)
	store := history.Open(t.TempDir(), func() time.Time { return tempNow }, omHourly)
	box := fieldBoxes(ask.Region, ask.View)[0].Name
	anchor := askAnchor(ask, tempNow)
	key := history.Key{Source: "openmeteo", Place: box}

	buildTemperature(context.Background(), &fakeTemp{name: "NDFD", now: tempNow}, nil, ask, tempNow, &fallback{store: store})
	if _, ok := store.Get(omHourly.Name, key, anchor); ok {
		t.Fatal("NDFD's answer was kept as Open-Meteo's")
	}
	buildTemperature(context.Background(), &fakeTemp{name: "Open-Meteo", now: tempNow}, nil, ask, tempNow, &fallback{store: store})
	rec, ok := store.Get(omHourly.Name, key, anchor)
	if !ok || rec.Values["temp"][0] != 10 || rec.Values["wind_from"][0] != 270 {
		t.Fatalf("Open-Meteo's current hour was not kept: %+v, %v", rec, ok)
	}
	if _, ok := store.Get(omHourly.Name, key, anchor.Add(time.Hour)); ok {
		t.Error("an hour ahead of the current one was kept: the forecast is not the record")
	}

	filled := history.Open(t.TempDir(), func() time.Time { return tempNow }, omHourly)
	buildTemperature(context.Background(), &fakeTemp{name: "NDFD", now: tempNow, failed: true}, &fakeTemp{name: "Open-Meteo", now: tempNow}, ask, tempNow, &fallback{store: filled})
	if _, ok := filled.Get(omHourly.Name, key, anchor); !ok {
		t.Error("Open-Meteo's fill was not kept")
	}
}

// OPEN-METEO MARINE'S WAVES ARE KEPT (W22.2, D-231): each hour it gave a box's
// points past NDFD's reach, up to the current one, goes into the history -
// the points it was not asked for missing, never zero.
func TestOpenMeteoMarinesWavesAreKept(t *testing.T) {
	ask := tempAsk(true)
	store := history.Open(t.TempDir(), func() time.Time { return tempNow }, omWaves)
	withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{metres: 1, max: 2, beyond: 2}, fakeWaves{metres: 3, max: 4}, ask, tempNow, waveKeep{land: &landPoints{}, store: store})
	box := fieldBoxes(ask.Region, ask.View)[0].Name
	rec, ok := store.Get(omWaves.Name, history.Key{Source: "openmeteo", Place: box}, askAnchor(ask, tempNow))
	if !ok {
		t.Fatal("Open-Meteo Marine's current hour was not kept")
	}
	if _, ok := store.Get(omWaves.Name, history.Key{Source: "openmeteo", Place: box}, askAnchor(ask, tempNow).Add(time.Hour)); ok {
		t.Error("an hour ahead of the current one was kept")
	}
	v := rec.Values["waves"]
	if len(v) < 3 || !math.IsNaN(v[0]) || v[2] != 3 {
		t.Errorf("the kept waves are %v; want NDFD's points missing and Open-Meteo's 3 m past its reach", v)
	}
}
