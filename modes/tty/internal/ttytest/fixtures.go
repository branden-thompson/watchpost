package ttytest

// fixtures.go — the data both surfaces' tests build their models from: the
// card window's composed cards, a tinted card, a pool candidate and the loaded
// station. Each package keeps the few lines that send it to its own model, so
// nothing here names a model and the console's tests import this package with
// no cycle (design CK-4).

import (
	"fmt"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// DetailAt is a fixed moment, so the card window's age line is a fact rather
// than a race.
func DetailAt() time.Time { return time.Date(2026, 9, 11, 17, 45, 0, 0, time.UTC) }

// ComposedCard is a card the Composer has finished with: words, a manifest and
// a stamp — and the ADMITTED card it grew from, because the lineup takes only
// those.
//
// THE REAL CHAIN, NOT A STRUCT LITERAL. `Director.onBuilt` queues an admitted
// card and `Set`s the composed one over it; a fixture that assembled the end
// state by hand would keep passing on the day an invariant along that chain
// started refusing it.
func ComposedCard(t *testing.T, id, subject string, contents []lineup.Content, at time.Time) (lineup.Card, lineup.Card) {
	t.Helper()
	base := Card(t, id, subject)
	c, err := base.To(lineup.Standby)
	if err != nil {
		t.Fatalf("standby %s: %v", id, err)
	}
	c, err = c.WithScript(
		lineup.Script{Parts: []lineup.Part{
			{Kind: lineup.PartLine, Text: "The forecast for " + subject + " calls for sun."},
			{Kind: lineup.PartLine, Text: "Seas are two to three feet."},
		}},
		contents, at,
	)
	if err != nil {
		t.Fatalf("composing %s: %v", id, err)
	}
	return base, c
}

// FiveSources is a manifest one row LONGER than the card can draw.
//
// THE CARD READS FOUR, so the fifth row exists only in the window — which is
// the claim the window is for, and a fixture of four could not tell the two
// apart.
func FiveSources() []lineup.Content {
	return []lineup.Content{
		{Name: "NWS Weather Forecast", Detail: "09/11 - 09/17"},
		{Name: "Watchpost Marine Report", Detail: "09/11 - 09/13"},
		{Name: "Watchpost Fire Report", Detail: "3 hotspots / 10 incidents"},
		{Name: "Watchpost Seismic Report", Detail: "2 quakes"},
		{Name: "Watchpost Air Quality Report", Detail: "AQI 41"},
	}
}

// Seed is a lineup holding composed cards, queued and Set exactly as the
// Director does it.
func Seed(t *testing.T, pairs ...[2]lineup.Card) lineup.Lineup {
	t.Helper()
	var l lineup.Lineup
	for _, p := range pairs {
		next, err := l.Queue(lineup.MainTrack, p[0])
		if err != nil {
			t.Fatalf("queueing %s: %v", p[0].ID, err)
		}
		if next, err = next.Set(p[1]); err != nil {
			t.Fatalf("setting %s: %v", p[1].ID, err)
		}
		l = next
	}
	return l
}

// OneCard is the card window tests' fixture: one composed card with five
// sources.
func OneCard(t *testing.T) [2]lineup.Card {
	t.Helper()
	a, c := ComposedCard(t, "a", "Oceanside, CA", FiveSources(), DetailAt())
	return [2]lineup.Card{a, c}
}

// Toned is a card of a slot and an origin, carrying an arrival of each
// category given.
func Toned(id string, slot lineup.Slot, origin lineup.Origin, cats ...category.Category) lineup.Card {
	c := lineup.Card{ID: id, Slot: slot, Origin: origin, Subject: id, Headline: id}
	for _, k := range cats {
		c.From = append(c.From, lineup.Arrival{ID: id, Category: k})
	}
	return c
}

// Admitted puts a card through the states the schedule requires before it may
// be queued — the lineup holds admitted cards only, and a fixture that skips
// this trips the invariant rather than testing anything.
func Admitted(t *testing.T, c lineup.Card) lineup.Card {
	t.Helper()
	c.State = lineup.Proposed
	out, err := lineup.Propose(c)
	if err != nil {
		t.Fatalf("proposing %s: %v", c.ID, err)
	}
	if out, err = out.To(lineup.Admitted); err != nil {
		t.Fatalf("admitting %s: %v", c.ID, err)
	}
	return out
}

// PoolLoc is a candidate with everything a pool row can show.
func PoolLoc() *snapshot.Location {
	now, hi, lo, thi, tlo, stnKM := 73.0, 88.0, 55.0, 91.0, 62.0, 4.2
	return &snapshot.Location{
		Label: "Fallbrook, CA", Zip: "92028", Lat: 33.37, Lon: -117.25,
		WeatherAsOf: time.Now(),
		Harmonized: snapshot.Conditions{
			Condition: "CLEAR", Temp: &now,
			Source: snapshot.SourceInfo{Provider: "nws", ModelOrStation: "KOKB", DistanceKm: &stnKM},
		},
		Daily: []snapshot.Daily{
			{Date: "2026-09-12", Condition: "CLEAR", TempMax: &hi, TempMin: &lo},
			{Date: "2026-09-13", Condition: "RAIN", TempMax: &thi, TempMin: &tlo},
		},
	}
}

// LoadedStation is a station with a full pool, a full running order, and
// real weather behind every one of them: what a console is sent to be loaded.
type LoadedStation struct {
	Transmitter snapshot.LocationRef
	RadiusMi    float64
	Pool        []snapshot.LocationRef
	Recent      *snapshot.Snapshot
	Lineup      lineup.Lineup
}

// Loaded is the loaded station of n candidates, its weather observed at a
// given time, so a golden frame of it is the same on every run.
//
// SEVEN DAYS OF FORECAST EACH, because that is what the providers return and it
// is what makes the day-cell work real: a fixture with two days would measure a
// path the app never takes.
func Loaded(t testing.TB, n int, observed time.Time) LoadedStation {
	t.Helper()
	pool := make([]snapshot.LocationRef, 0, n)
	locs := make([]snapshot.Location, 0, n)
	now, hi, lo := 20.0, 31.0, 12.0
	for i := range n { // bounded by the ask (P10-02)
		ref := snapshot.LocationRef{Label: fmt.Sprintf("Place %02d, CA", i), Zip: fmt.Sprintf("920%02d", i),
			Lat: 33.2 + float64(i)/50, Lon: -117.2, Population: 1000 * (i + 1)}
		pool = append(pool, ref)
		daily := make([]snapshot.Daily, 0, 7)
		for d := range 7 { // bounded by the forecast (P10-02)
			daily = append(daily, snapshot.Daily{Date: fmt.Sprintf("2026-09-%02d", 13+d),
				Condition: "CLEAR", TempMax: &hi, TempMin: &lo})
		}
		locs = append(locs, snapshot.Location{Label: ref.Label, Zip: ref.Zip, Lat: ref.Lat, Lon: ref.Lon,
			WeatherAsOf: observed,
			Harmonized: snapshot.Conditions{Condition: "CLEAR", Temp: &now,
				Source: snapshot.SourceInfo{Provider: "nws", ModelOrStation: "KOKB"}},
			Daily: daily})
	}
	var l lineup.Lineup
	for i := range lineup.MainTrackCap { // bounded by the track (P10-02)
		c, err := lineup.Propose(lineup.Card{ID: fmt.Sprint("c", i), Slot: lineup.LocationReport,
			Subject: string(snapshot.Key(pool[i%n])), Headline: pool[i%n].Label, State: lineup.Proposed})
		if err != nil {
			t.Fatalf("proposing c%d: %v", i, err)
		}
		if c, err = c.To(lineup.Admitted); err != nil {
			t.Fatalf("admitting c%d: %v", i, err)
		}
		if l, err = l.Queue(lineup.MainTrack, c); err != nil {
			t.Fatalf("queueing c%d: %v", i, err)
		}
	}
	return LoadedStation{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    100, Pool: pool, Recent: &snapshot.Snapshot{Locations: locs}, Lineup: l,
	}
}
