package app

import (
	"context"
	"strings"
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

// P3(a3): the SPEAK half. A location report reads through the arbiter as the
// rotation class, so the programme gives way to a severe read and to a
// takeover — which is the whole reason the class exists.
//
// THE STAGE IS SET EXPLICITLY (0.16.0 P3, maintrack.go). Until the flip, a
// main-track card is declined at speak so the rotation keeps sole ownership of
// the air; this test is about the MERGED station, so it asks for it by name.
func TestALocationReportIsSpokenAsTheRotationClass(t *testing.T) {
	t.Setenv("WATCHPOST_MAINTRACK", "live")
	v := &scriptVoice{}
	b := newBench(t, v)
	out := b.x.run(context.Background(), lineup.Speak{
		ID: "r1", Slot: lineup.LocationReport, Script: lineup.Say("Currently sixty-one degrees."),
	})
	if len(out) == 0 {
		t.Fatal("a spoken card comes home with an event")
	}
	if _, failed := out[0].(lineup.Failed); failed {
		t.Fatalf("a location report must now be SPOKEN, not declined: %v — this is the second half of "+
			"the decline that named T3.2", out[0])
	}
	if got := v.got(); !strings.Contains(got, "speak:Currently sixty-one degrees.") {
		t.Errorf("the words must reach the voice; got %q", got)
	}
	// NOT AN ASIDE. An aside is a TAKEOVER's line, whose visualizer does not
	// follow it; the programme is ordinary speech.
	if strings.Contains(v.got(), "aside:") {
		t.Errorf("the rotation is the programme, not a takeover: %q", v.got())
	}
}

// THE DARK STAGE IS THE ONE THING BETWEEN A HALF-MERGED PRODUCER AND TWO
// SPEAKERS, so it is pinned from the speak side as well as from the switch.
//
// The card is still queued, still composed and still published — that is what
// makes dark an observation of the real producer — and it stops HERE, one call
// short of the voice.
func TestADarkMainTrackCardIsDeclinedAtTheAirAndNeverReachesTheVoice(t *testing.T) {
	// DARK ONLY. The `off` arm was here too and proved nothing about `off`
	// (red team 2026-09-09, finding 12): in that stage the deck never reports,
	// so no LocationReport card can exist to be declined, and the arm read as
	// coverage it was not. What `off` guarantees is pinned where it is true —
	// at the seam, by TestTheDefaultStageTellsTheDirectorNothing.
	for _, stage := range []string{"dark"} {
		t.Setenv("WATCHPOST_MAINTRACK", stage)
		v := &scriptVoice{}
		b := newBench(t, v)
		out := b.x.run(context.Background(), lineup.Speak{
			ID: "r1", Slot: lineup.LocationReport, Script: lineup.Say("Currently sixty-one degrees."),
		})
		if len(out) != 1 {
			t.Fatalf("stage %q: a declined card comes home with one event; got %d", stage, len(out))
		}
		f, failed := out[0].(lineup.Failed)
		if !failed {
			t.Fatalf("stage %q: the rotation still owns the air here, so the card must be declined; got %T", stage, out[0])
		}
		// ROUTED, or the station raises a RELAY FAULT window every rotation
		// turn while the merge is dark — the noise regression fault.go exists
		// to avoid (I-2).
		if !f.Routed {
			t.Errorf("stage %q: a staged decline is not the station going quiet", stage)
		}
		if got := v.got(); strings.Contains(got, "speak:") {
			t.Errorf("stage %q: nothing may reach the voice while the rotation owns the air; got %q", stage, got)
		}
	}
}

// A TAKEOVER IS NOT STAGED. The rail has read through the arbiter since 0.14.0
// and the merge must not touch it — a stage check written against the wrong
// question would silence hazards.
func TestTheAlertRailReadsWhateverTheMainTrackStageIs(t *testing.T) {
	for _, stage := range []string{"", "dark", "live"} {
		t.Setenv("WATCHPOST_MAINTRACK", stage)
		v := &scriptVoice{}
		b := newBench(t, v)
		out := b.x.run(context.Background(), lineup.Speak{
			ID: "a1", Slot: lineup.BreakingAlert, Script: lineup.Say("Tornado warning."),
		})
		if len(out) == 0 {
			t.Fatalf("stage %q: a spoken card comes home with an event", stage)
		}
		if _, failed := out[0].(lineup.Failed); failed {
			t.Fatalf("stage %q: the alert rail is not staged: %v", stage, out[0])
		}
		if got := v.got(); !strings.Contains(got, "Tornado warning.") {
			t.Errorf("stage %q: the hazard must reach the voice; got %q", stage, got)
		}
	}
}
