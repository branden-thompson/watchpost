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
		alert:      func(string) (globalfeed.Event, bool) { return globalfeed.Event{}, false },
		mark:       func(string) {},
		readAloud:  func(string) bool { return false },
		report:     func(lineup.Effect, string) {},
		cutTo:      func(string) {},
		escalate:   func(string) {},
		compose:    func(ctx context.Context, ref string) ([]synth.Segment, error) { return segs, err },
		playReport: stubPlayReport,
		held:       newSegmentStore(),
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

// P3(d): the SPEAK half, and Shape B.
//
// THE ARBITER OWNS WHO SPEAKS; THE SOURCE STILL OWNS HOW A REPORT IS SPOKEN.
// A location report takes the air as `narrateRotation` — the lowest class, so
// the programme gives way to a severe read and to a takeover — and is then
// played by the deck's own synth source, which is what keeps the per-segment
// marquee, the cast, the correspondent handoffs, repeat-one, the player row and
// the engine's give-way rule (a rendered cycle HOLDS, a live relay DIPS).

func TestALocationReportIsPlayedByItsSourceUnderTheArbiter(t *testing.T) {
	v := &scriptVoice{}
	b := newBench(t, v)
	segs := []synth.Segment{{Key: "obs", Text: "Currently sixty-one degrees."}, {Key: "tail", Text: "This has been Watchpost."}}
	b.x.held.put("r1", segs)

	out := b.x.run(context.Background(), lineup.Speak{
		ID: "r1", Slot: lineup.LocationReport, Subject: "33.1959,-117.3795",
		Script: lineup.Say("Currently sixty-one degrees."),
	})

	if len(out) == 0 {
		t.Fatal("a spoken card comes home with an event")
	}
	if _, failed := out[0].(lineup.Failed); failed {
		t.Fatalf("a location report must be SPOKEN, not declined: %v", out[0])
	}
	if _, ok := out[0].(lineup.Finished); !ok {
		t.Fatalf("a report that reached its sign-off comes home Finished; got %T", out[0])
	}
	// THE COMPOSED REPORT IS WHAT WAS PLAYED, not a re-composition. The
	// segments carry the roles, the self-introductions and the pauses that the
	// card's script cannot, so re-composing at the air would be eleven more
	// requests AND a second answer to a question already asked.
	if len(b.reads) != 1 {
		t.Fatalf("the rotation is played once, by its source; got %d reads", len(b.reads))
	}
	if b.reads[0] != "33.1959,-117.3795:obs,tail" {
		t.Errorf("the reader must be handed THIS card's location and THIS card's composed segments; got %q", b.reads[0])
	}
	// AND THE AIR WAS HELD FOR IT. The duck is the arbiter taking the air; the
	// report plays inside it, and the restore comes after.
	got := v.got()
	if !strings.Contains(got, "duck") || !strings.Contains(got, "report:33.1959,-117.3795") {
		t.Errorf("the report must play INSIDE the arbiter's hold on the air; got %q", got)
	}
	// NOT AN ASIDE. An aside is a TAKEOVER's line, whose visualizer does not
	// follow it; the programme is the broadcast.
	if strings.Contains(got, "aside:") {
		t.Errorf("the rotation is the programme, not a takeover: %q", got)
	}
	// NOR IS IT SPOKEN LINE BY LINE. Reading the script through the narrator
	// would replace the player and take the give-way rule with it.
	if strings.Contains(got, "speak:") {
		t.Errorf("the report is played by its source, not narrated clip by clip: %q", got)
	}
	// THE REPORT IS TAKEN FROM THE STORE, not left behind for a later card
	// with the same id to speak — and the rotation recycles ids by design.
	if _, still := b.x.held.take("r1"); still {
		t.Error("the composed report must be taken exactly once")
	}
}

func TestAReportThatDidNotReachItsSignOffComesHomeFailed(t *testing.T) {
	v := &scriptVoice{}
	b := newBench(t, v)
	b.readFails = true
	b.x.held.put("r1", []synth.Segment{{Key: "obs", Text: "x"}})

	out := b.x.run(context.Background(), lineup.Speak{
		ID: "r1", Slot: lineup.LocationReport, Subject: "here", Script: lineup.Say("x"),
	})

	if len(out) != 1 {
		t.Fatalf("one event; got %d", len(out))
	}
	f, failed := out[0].(lineup.Failed)
	if !failed {
		t.Fatalf("a read that ended early says so, always (DR-24); got %T — a Finished would tell the "+
			"schedule a read happened that did not", out[0])
	}
	// ROUTED: a stop, a halt or a pre-emption is not the station failing, and a
	// fault window for one would be the noise regression fault.go avoids (I-2).
	if !f.Routed {
		t.Error("a report cut short is routed, not a station that has gone quiet")
	}
}

func TestACardWithNoComposedReportIsDeclinedNotAired(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	// Nothing put in the store: the card was built and then dropped, or its
	// report was evicted by the bound.
	out := b.x.run(context.Background(), lineup.Speak{
		ID: "ghost", Slot: lineup.LocationReport, Subject: "here", Script: lineup.Say("x"),
	})
	if len(out) != 1 {
		t.Fatalf("one event; got %d", len(out))
	}
	if _, failed := out[0].(lineup.Failed); !failed {
		t.Fatalf("a card with nothing to play must be declined, not aired silently; got %T", out[0])
	}
	if len(b.reads) != 0 {
		t.Errorf("and nothing may reach the source; got %v", b.reads)
	}
}

// MUTE IS THE RAIL'S RULE, AND ONLY THE RAIL'S (0.16.0 P3(d)).
//
// `[M]` means "do not read me hazards". It has never silenced the broadcast —
// the radio plays through it today — so applying it to a location report would
// have made the merge turn the mute key into a stop button.
func TestMutingHoldsHazardsAndDoesNotStopTheBroadcast(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	b.muted = true
	b.x.held.put("r1", []synth.Segment{{Key: "obs", Text: "x"}})

	out := b.x.run(context.Background(), lineup.Speak{
		ID: "r1", Slot: lineup.LocationReport, Subject: "here", Script: lineup.Say("x"),
	})
	if _, failed := out[0].(lineup.Failed); failed {
		t.Fatalf("the rotation is not a hazard: muting must not stop the broadcast; got %v", out[0])
	}
	if len(b.reads) != 1 {
		t.Fatalf("the report still plays; got %d reads", len(b.reads))
	}

	// And the rail's rule is untouched: a hazard read while muted would be
	// consumed in silence and never sounded (MVS-D-78).
	alertOut := b.x.run(context.Background(), lineup.Speak{
		ID: "a1", Slot: lineup.BreakingAlert, Script: lineup.Say("Tornado warning."),
	})
	if _, failed := alertOut[0].(lineup.Failed); !failed {
		t.Fatal("a takeover read while muted would mark every alert read and sound none of them")
	}
}

// THE HANDOFF: what the BUILD composed is what the AIR plays (0.16.0 P3(d)).
//
// FOUND BY A PLANT THAT SURVIVED. Deleting the store's write changed no
// assertion, because the build test looked only at the card's script and the
// speak test put the segments in itself. Nothing ran the two halves together,
// so the one thing that carries a report from where it is composed to where it
// is voiced was untested — and a station with that wire cut would show a
// full lineup and read none of it.
func TestWhatTheBuildComposedIsWhatTheAirPlays(t *testing.T) {
	v := &scriptVoice{}
	b := newBench(t, v)
	// The bench's composer is the one seam here: whatever IT returns must be
	// what reaches the reader, with no second composition in between.
	evs := b.x.build(context.Background(), lineup.BuildCard{
		ID: "r1", Slot: lineup.LocationReport, Subject: "33.1959,-117.3795",
	})
	if len(evs) != 1 {
		t.Fatalf("one event comes home from a build; got %d", len(evs))
	}
	built, ok := evs[0].(lineup.Built)
	if !ok {
		t.Fatalf("the card must come home BUILT; got %T", evs[0])
	}

	out := b.x.run(context.Background(), lineup.Speak{
		ID: "r1", Slot: lineup.LocationReport, Subject: "33.1959,-117.3795", Script: built.Script,
	})
	if _, failed := out[0].(lineup.Failed); failed {
		t.Fatalf("the card the build just composed must be playable: %v — a build that composes and "+
			"then loses its report leaves a full lineup and a silent station", out[0])
	}
	if len(b.reads) != 1 || b.reads[0] != "33.1959,-117.3795:obs" {
		t.Errorf("the reader must be handed the segments THIS build composed; got %v", b.reads)
	}
	// AND THE DISPLAY AND THE AUDIO COME FROM ONE COMPOSITION. The script is
	// what the console shows; the segments are what is voiced; a station whose
	// two halves were composed separately would show one report and read
	// another.
	if got := built.Script.Text(); !strings.Contains(got, "Currently sixty-one degrees.") {
		t.Errorf("the card's script must carry the composed words for the console; got %q", got)
	}
}
