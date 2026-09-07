package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/category"
)

// C-2 — AN EVACUATION ORDER IS SPOKEN, AND IT LEADS.
//
// The Weather Service's highest-urgency product is an INSTRUCTION rather than
// a description: leave, now. The person it exists for is, by its own premise,
// not looking at the screen — so a station that paints it on the marquee and
// says nothing has failed at the one thing it is for.
//
// THIS STARTS AT THE PRODUCER (D-12), not at lineup.Plan. The ~40 ordering pins
// in plan_test.go exercise rung 1 by building an Arrival with the category set
// by hand; every one of them stayed green while no arrival in the running app
// could carry it.
func TestAnEvacuationOrderLeadsTheReadAndIsSpokenAloud(t *testing.T) {
	nar := testDirector(&scriptVoice{}, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	seen := loadSeen(t.TempDir(), time.Hour)
	deck := &tickerDeck{send: func(tea.Msg) {}, muted: &atomic.Bool{}, voice: nar, seen: seen}

	at := time.Now().Add(-5 * time.Minute)
	evs := []globalfeed.Event{
		// THE TORNADO WARNING IS FIRST AND EQUALLY SEVERE. Leading the read can
		// then only come from the ladder's rung, not from input order and not
		// from a severity tie-break.
		{ID: "tor", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "the Oklahoma City area", Severity: globalfeed.SevRed, At: at},
		{ID: "evac", Class: globalfeed.ClassSevereWx, Type: "Evacuation Immediate", Location: "Paradise, CA", Severity: globalfeed.SevRed, At: at},
	}
	// THE PRODUCER'S OWN TRANSLATION, asserted before the read: this is the step
	// that could not carry the category.
	arr := arrivalsOf(evs)
	if len(arr) != 2 {
		t.Fatalf("the fixture lost an arrival: %d", len(arr))
	}
	var evac category.Category = category.Forecasts
	for _, a := range arr {
		if a.ID == "evac" {
			evac = a.Category
		}
	}
	if evac != category.Emergency {
		t.Errorf("the producer hands the evacuation order over as %s; the ladder reserves rung 1 for Emergency Orders", category.Of(evac).Bucket)
	}

	order, _ := planned(t, deck, evs)
	if len(order) != 2 || order[0].ID != "evac" {
		got := make([]string, 0, len(order))
		for _, e := range order {
			got = append(got, e.ID)
		}
		t.Fatalf("the evacuation order must LEAD the read (ReadRank 1), got %v", got)
	}

	newStation(t, deck).takeover(context.Background(), evs)
	if !seen.set()["evac"] {
		t.Error("the evacuation order was never read aloud: painted on the tape and never spoken is the failure this product exists to prevent")
	}
}
