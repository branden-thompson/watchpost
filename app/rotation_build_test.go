package app

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// P3: the executors can BUILD a location-report card.
//
// AT PARITY, AND NOTHING PRODUCES ONE YET. This is the T3.2a discipline that
// worked in 0.14.0: wire the capability first and claim "nothing changes",
// then switch the producer over. The two declines that said "read by the main
// track, which arrives with T3.2" are what this removes.

func buildDeps(t *testing.T, segs []synth.Segment, err error) *executors {
	t.Helper()
	x := newExecutors(executors{
		voice: testDirector(nil, nil), clock: func() render.Clock { return render.Clock12 },
		now: func() time.Time { return execNow }, mc: newMastercontrol(nil, func(tea.Msg) {}),
		audible: func() bool { return true }, muted: func() bool { return false },
		alert:     func(string) (globalfeed.Event, bool) { return globalfeed.Event{}, false },
		mark:      func(string) {},
		readAloud: func(string) bool { return false },
		report:    func(lineup.Effect, string) {},
		cutTo:     func(string) {},
		escalate:  func(string) {},
		compose:   func(ctx context.Context, ref string) ([]synth.Segment, error) { return segs, err },
	})
	if x == nil {
		t.Fatal("the executors refused to build with a composer")
	}
	return x
}

func TestALocationReportCardIsBuiltNotDeclined(t *testing.T) {
	x := buildDeps(t, []synth.Segment{{Key: "obs", Text: "Currently sixty-one degrees."}}, nil)
	evs := x.build(context.Background(), lineup.BuildCard{ID: "c1", Slot: lineup.LocationReport, Subject: "OCEANSIDE"})
	if len(evs) != 1 {
		t.Fatalf("one event comes home from a build; got %d", len(evs))
	}
	built, ok := evs[0].(lineup.Built)
	if !ok {
		t.Fatalf("a location report must now come home BUILT, not declined: got %T — "+
			"this is the decline that said 'read by the main track, which arrives with T3.2'", evs[0])
	}
	if built.Script.Empty() {
		t.Error("the card came home with no words; a card takes the air with its words already on it")
	}
	if got := built.Script.Text(); got == "" {
		t.Error("the composed words must survive onto the card")
	}
}

func TestALocationReportThatComposesNothingIsDeclinedNotAired(t *testing.T) {
	x := buildDeps(t, nil, nil)
	evs := x.build(context.Background(), lineup.BuildCard{ID: "c2", Slot: lineup.LocationReport, Subject: "NOWHERE"})
	if len(evs) != 1 {
		t.Fatalf("one event; got %d", len(evs))
	}
	if _, ok := evs[0].(lineup.Built); ok {
		t.Error("a card that composed nothing must be DECLINED, not built — a card on the air with " +
			"no words is silence under a callout the band has already promised")
	}
}

// D-33 RETIRED THE SPEAK HALF, AND THESE THREE TESTS WITH IT.
//
// P3(a3) taught `speak` to read a LocationReport as its own narration class,
// and P3's flip built on that. The HUM LEAD ruled on 2026-09-09 that a chosen
// read REPLACES the bed rather than speaking over it — so the programme is not
// a narration, does not belong on the narration path, and `speak` declines
// every slot that is not on the rail.
//
// The three tests removed here pinned: that a report is spoken as the rotation
// class, that a dark-stage card is declined at the air, and that the rail reads
// whatever the stage is. The first two assert a design that no longer exists;
// the third's subject — a stage that could hand the air over — went with it.
//
// WHAT REPLACES THEM is the single decline at the top of `speak`, and the two
// BUILD tests above, which still stand: the executors can compose a report, and
// a report that composes nothing is declined rather than aired.
