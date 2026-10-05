package lineup

import (
	"testing"
	"time"
)

// testmark_test.go — a fabricated alert stays fabricated all the way to the card
// (D-55, FR-4.4).
//
// EVERY SURFACE MARKS A FABRICATED EVENT: the ticker band puts `**TEST EVENT**`
// at BOTH ends, the severe window leads its EVENT column with it, and the audio
// says "this is only a test" four separate times. The card is a surface too, so
// `Arrival.Test` must reach it through `takeoverOf`, not stop at
// `selectBurst`'s ordering.
//
// A console drawing a fabricated takeover as an ordinary one is the exact
// hazard the injector's "never ship" rule was written against — "a screenshot
// of a fabricated tornado warning is indistinguishable from a real one".

func TestAFabricatedAlertsCardSaysSo(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	burst, err := Plan([]Arrival{{
		ID: "t1", Subject: "oceanside", Headline: "TORNADO WARNING", At: now, Test: true,
	}}, Settings{Max: 5}, now)
	if err != nil {
		t.Fatalf("planning the burst: %v", err)
	}
	if !burst.HasTakeover() {
		t.Fatal("a fabricated alert still takes the air; that is the point of injecting one")
	}
	if !burst.Takeover.Test {
		t.Error("the card must carry that it was fabricated, or no surface reading it can say so")
	}
}

func TestARealAlertsCardIsNotMarked(t *testing.T) {
	// THE OTHER HALF, and it is the one that matters most: a mark that appeared
	// on a real hazard would teach an operator to ignore it.
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	burst, err := Plan([]Arrival{{
		ID: "r1", Subject: "oceanside", Headline: "TORNADO WARNING", At: now,
	}}, Settings{Max: 5}, now)
	if err != nil {
		t.Fatalf("planning the burst: %v", err)
	}
	if burst.Takeover.Test {
		t.Error("a real hazard is never marked as a test")
	}
}

func TestABurstIsMarkedOnlyWhenEVERYAlertInItWasFabricated(t *testing.T) {
	// ONE CARD CARRIES THE WHOLE BURST (MVS-D-77), so the question is what the
	// card claims about the alerts inside it. A burst holding one REAL hazard is
	// not a test, and marking it would hide a live alert behind a label that
	// says to ignore it — the audio path already makes this exact call
	// (`allFabricated`), and the card must not disagree with what is spoken.
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	burst, err := Plan([]Arrival{
		{ID: "t1", Subject: "oceanside", Headline: "TORNADO WARNING", At: now, Test: true},
		{ID: "r1", Subject: "carlsbad", Headline: "FLASH FLOOD WARNING", At: now},
	}, Settings{Max: 5}, now)
	if err != nil {
		t.Fatalf("planning the burst: %v", err)
	}
	if burst.Takeover.Test {
		t.Error("a burst carrying a real hazard is not a test, whatever else is in it")
	}
}
