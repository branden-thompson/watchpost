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

	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/report"
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
	segs, err := compose(context.Background(), string(snapshot.Key(refBonsall)), 0)
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
	if _, err := compose(context.Background(), string(snapshot.Key(refOceanside)), 0); err == nil {
		t.Error("no deck composes no words, and says so")
	}
}

// TestOnlyTheChosenKindsAreGathered.
//
// R2's whole claim: a report carries what was asked for and nothing else, and
// the sources nobody asked for are not even FETCHED. That second half is the
// point — skipping them at composition time would still pay for the network.
//
// DRIVEN THROUGH THE HOOKS THE DECK ACTUALLY USES. `d.fire`, `d.seismic` and
// `d.marine` are the seams `segments` calls; counting their calls is how this
// asks "did you go and get it" rather than "did you say it".
func TestOnlyTheChosenKindsAreGathered(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		want                  report.Set
		fire, seismic, marine bool
	}{
		{"everything", report.Everything(), true, true, true},
		{"fire alone", report.Set(0).Add(report.Fire), true, false, false},
		{"quake alone", report.Set(0).Add(report.Seismic), false, true, false},
		{"marine alone", report.Set(0).Add(report.Marine), false, false, true},
		{"NWS alone asks for none of the three", report.Set(0).Add(report.NWS), false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotFire, gotSeismic, gotMarine bool
			// A REAL PROVIDER OVER A CLIENT THAT ANSWERS NOTHING. `segments` asks
			// the NWS provider before it reaches the three hooks, and a nil
			// provider dereferences rather than declining — so the fixture needs
			// one even though the question here is about the OTHER three.
			client, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: 30, MaxRetries: 1, CacheDir: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			d := &radioDeck{
				nws:      nws.New(client, ""),
				products: synth.NewProducts(client, ""),
				fire:     func(snapshot.LocationRef) synth.FireReport { gotFire = true; return synth.FireReport{} },
				seismic:  func(snapshot.LocationRef) synth.SeismicReport { gotSeismic = true; return synth.SeismicReport{} },
				marine:   func(snapshot.LocationRef) synth.MarineReport { gotMarine = true; return synth.MarineReport{} },
			}
			_, _ = d.segments(context.Background(), refOceanside, synth.VoiceToken, tc.want)

			if gotFire != tc.fire {
				t.Errorf("fire fetched=%v, want %v", gotFire, tc.fire)
			}
			if gotSeismic != tc.seismic {
				t.Errorf("seismic fetched=%v, want %v", gotSeismic, tc.seismic)
			}
			if gotMarine != tc.marine {
				t.Errorf("marine fetched=%v, want %v", gotMarine, tc.marine)
			}
		})
	}
}

// TestTheForecastProductsAreNotPulledUnlessAsked.
//
// THE PRODUCTS ARE THE MOST EXPENSIVE OF THE FOUR — a forecast-office lookup and
// the UGC filtering after it — and the one most often not wanted: a FIRE-only
// card has no use for a zone forecast.
//
// MUTANT mAX2 MADE THEM UNCONDITIONAL AND SURVIVED, because the test above counts
// the deck's three HOOKS and the products do not go through one. They go over the
// wire, so this counts requests instead — which is the only way to ask "did you
// go and get it" of a source that has no seam of its own.
func TestTheForecastProductsAreNotPulledUnlessAsked(t *testing.T) {
	load := func(want report.Set) int64 {
		t.Helper()
		client, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: 30, MaxRetries: 1, CacheDir: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		d := &radioDeck{nws: nws.New(client, ""), products: synth.NewProducts(client, "")}
		_, _ = d.segments(context.Background(), refOceanside, synth.VoiceToken, want)
		r := totalRequests(client)
		return r.net + r.cache
	}
	// THE FRAME STILL FETCHES: the observation and the alerts are asked for
	// whatever was chosen, so this is a COMPARISON rather than an absolute — the
	// question is whether asking for NWS costs more than not asking for it.
	withNWS := load(report.Set(0).Add(report.NWS))
	without := load(report.Set(0).Add(report.Fire))
	if withNWS <= without {
		t.Errorf("asking for the NWS forecast made %d requests and not asking made %d; "+
			"the products are being pulled either way", withNWS, without)
	}
}

// TestACardThatNamesNoSetStillComposesAFullReport.
//
// EVERY CARD IN THE TREE IS IN THAT STATE. The Director's own cards carry no
// report set — only an operator's request does — so "no set" has to mean the
// whole report, not an empty one. Read the other way round, the station would
// compose a frame with nothing in it and the rotation would go quiet.
//
// MUTANT mAX3 MADE THE DEFAULT EMPTY AND SURVIVED: nothing drove `composeFor`'s
// default at all, because the one test that reaches it is about a location
// NOBODY WATCHES and fails before composing.
func TestACardThatNamesNoSetStillComposesAFullReport(t *testing.T) {
	var gotFire, gotSeismic, gotMarine bool
	client, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: 30, MaxRetries: 1, CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	d := &radioDeck{
		nws: nws.New(client, ""), products: synth.NewProducts(client, ""),
		fire:    func(snapshot.LocationRef) synth.FireReport { gotFire = true; return synth.FireReport{} },
		seismic: func(snapshot.LocationRef) synth.SeismicReport { gotSeismic = true; return synth.SeismicReport{} },
		marine:  func(snapshot.LocationRef) synth.MarineReport { gotMarine = true; return synth.MarineReport{} },
	}
	compose := composeFor(d, watchOf(refOceanside)) // nil wants: no card names a set yet (R2)
	// AN EMPTY SET, which is what every card the Director makes for itself
	// carries — and must mean the WHOLE report.
	_, _ = compose(context.Background(), string(snapshot.Key(refOceanside)), 0)

	if !gotFire || !gotSeismic || !gotMarine {
		t.Errorf("a card naming no report set gathered fire=%v seismic=%v marine=%v; "+
			"no set means the WHOLE report, or the rotation composes an empty one",
			gotFire, gotSeismic, gotMarine)
	}
}
