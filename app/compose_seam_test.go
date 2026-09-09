package app

// composeFor: how a main-track card gets its words (0.16.0 P3).
//
// THE RESOLUTION IS THE PART A TEST CAN OWN. Composing a real report is eleven
// network requests through the deck, so what is pinned here is the seam either
// side of it: the key a domain-free card carries becomes a place again, and a
// key that names no watched location produces a NAMED failure rather than an
// empty report — which the executor turns into a decline that says why.

import (
	"context"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// The package already has an oceanside(); these are the two the resolution
// tests compare, so the SECOND-entry case is a real one.
var (
	refOceanside = oceanside()
	refBonsall   = snapshot.LocationRef{Label: "Bonsall, CA", Zip: "92003", Lat: 33.2887, Lon: -117.2253, TZ: "America/Los_Angeles"}
)

func watchOf(refs ...snapshot.LocationRef) func() []snapshot.LocationRef {
	return func() []snapshot.LocationRef { return refs }
}

func TestACardsKeyResolvesBackToTheLocationItNames(t *testing.T) {
	watch := watchOf(refOceanside, refBonsall)
	for _, want := range []snapshot.LocationRef{refOceanside, refBonsall} {
		got, ok := refFor(watch, string(snapshot.Key(want)))
		if !ok {
			t.Fatalf("%s: the watchlist holds it, so its key must resolve", want.Label)
		}
		if got.Label != want.Label {
			t.Errorf("resolved to %q, want %q — the SECOND location is the one a single-entry "+
				"walk gets wrong", got.Label, want.Label)
		}
	}
}

func TestAKeyForNothingWatchedResolvesToNothing(t *testing.T) {
	if _, ok := refFor(watchOf(refOceanside), string(snapshot.Key(refBonsall))); ok {
		t.Error("a location the listener has since removed must not resolve to a different one")
	}
	if _, ok := refFor(nil, "0.0000,0.0000"); ok {
		t.Error("no watchlist resolves nothing; a station with no locations is a supported configuration")
	}
	if _, ok := refFor(watchOf(refOceanside), ""); ok {
		t.Error("an empty key names no location")
	}
}

func TestComposingForAnUnknownLocationFailsByName(t *testing.T) {
	compose := composeFor(&radioDeck{}, watchOf(refOceanside))
	segs, err := compose(context.Background(), string(snapshot.Key(refBonsall)))
	if err == nil {
		t.Fatal("a card for a location nobody watches must fail, not compose an empty report — " +
			"an empty report becomes a card on the air with nothing to say")
	}
	if len(segs) != 0 {
		t.Error("nothing is composed for a location that could not be resolved")
	}
}

func TestComposingWithNoDeckFailsRatherThanPanics(t *testing.T) {
	// A STATION WITH NO AUDIO IS A SUPPORTED CONFIGURATION, and the schedule
	// still runs on it.
	compose := composeFor(nil, watchOf(refOceanside))
	if _, err := compose(context.Background(), string(snapshot.Key(refOceanside))); err == nil {
		t.Error("no deck composes no words, and says so")
	}
}
