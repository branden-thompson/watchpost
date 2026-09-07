package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// execNow is the fixed clock every executor test composes against (DR-20).
var execNow = time.Date(2026, 9, 2, 15, 4, 0, 0, time.UTC)

// tornado is the alert the producer holds for card a1.
func tornado() globalfeed.Event {
	return globalfeed.Event{ID: "a1", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
		Location: "Bonsall, CA", At: execNow.Add(-3 * time.Minute), Until: execNow.Add(40 * time.Minute), Source: "NWS"}
}

// bench is one executor set over fakes: a scripted voice, a captured band, a
// producer that knows exactly the alerts it was given, and a fault sink.
type bench struct {
	x           *executors
	voice       *scriptVoice
	nar         *director
	mu          sync.Mutex
	msgs        []tea.Msg
	reports     []string
	tuned       []string        // the beds the Director asked for (T3.2b)
	escalations []string        // faults that reached a person (DR-21)
	asked       map[string]bool // producer records the Reader resolved for a cue
	holds       []time.Duration
	known       map[string]globalfeed.Event
	marked      []string // what the read recorded as spoken aloud
	muted       bool
	release     chan struct{} // lets a test hold a sequence on the air
}

func newBench(t *testing.T, v *scriptVoice) *bench {
	t.Helper()
	b := &bench{voice: v, known: map[string]globalfeed.Event{"a1": tornado()}, release: make(chan struct{})}
	// ONE BAND. The arbiter's effector is what the executors write through, so
	// the capture the test reads must be the one it was built with.
	band := func(m tea.Msg) { b.mu.Lock(); b.msgs = append(b.msgs, m); b.mu.Unlock() }
	if v == nil {
		b.nar = testDirector(nil, band) // no audio at all: the visuals still run
	} else {
		b.nar = testDirector(v, band)
	}
	b.nar.sleep = func(_ context.Context, d time.Duration) bool {
		b.mu.Lock()
		b.holds = append(b.holds, d)
		b.mu.Unlock()
		return true
	}
	b.x = newExecutors(executors{
		voice:   b.nar,
		mc:      b.nar.mc, // the ONE owner, shared with the arbiter
		clock:   func() render.Clock { return render.Clock12 },
		now:     func() time.Time { return execNow },
		audible: func() bool { return !b.muted },
		alert: func(id string) (globalfeed.Event, bool) {
			b.mu.Lock()
			if b.asked == nil {
				b.asked = map[string]bool{}
			}
			b.asked[id] = true
			b.mu.Unlock()
			e, ok := b.known[id]
			return e, ok
		},
		mark: func(id string) {
			b.mu.Lock()
			b.marked = append(b.marked, id)
			b.mu.Unlock()
		},
		muted: func() bool { b.mu.Lock(); defer b.mu.Unlock(); return b.muted },
		// The same store mark writes to, asked the other way (I-7).
		readAloud: func(id string) bool {
			b.mu.Lock()
			defer b.mu.Unlock()
			return slices.Contains(b.marked, id)
		},
		report: func(f lineup.Effect, why string) {
			b.mu.Lock()
			b.reports = append(b.reports, lineup.Describe(f)+": "+why)
			b.mu.Unlock()
		},
		cutTo:    func(ref string) { b.mu.Lock(); b.tuned = append(b.tuned, ref); b.mu.Unlock() },
		escalate: func(why string) { b.mu.Lock(); b.escalations = append(b.escalations, why); b.mu.Unlock() },
	})
	if b.x == nil {
		t.Fatal("newExecutors refused a well-formed set")
	}
	return b
}

func (b *bench) held() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	var sum time.Duration
	for _, d := range b.holds {
		sum += d
	}
	return sum
}

func (b *bench) reported() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.reports...)
}

func (b *bench) sent() []tea.Msg {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]tea.Msg(nil), b.msgs...)
}

// onlyBuilt is the one Built event in out, or a fatal.
func onlyBuilt(t *testing.T, out []lineup.Event) lineup.Built {
	t.Helper()
	if len(out) != 1 {
		t.Fatalf("got %d events %v, want exactly one", len(out), out)
	}
	built, ok := out[0].(lineup.Built)
	if !ok {
		t.Fatalf("got %#v, want a Built", out[0])
	}
	return built
}

// onlyFailed is the one Failed event in out, or a fatal.
func onlyFailed(t *testing.T, out []lineup.Event) lineup.Failed {
	t.Helper()
	if len(out) != 1 {
		t.Fatalf("got %d events %v, want exactly one", len(out), out)
	}
	failed, ok := out[0].(lineup.Failed)
	if !ok {
		t.Fatalf("got %#v, want a Failed", out[0])
	}
	return failed
}

// TestABreakingAlertIsBuiltFromTheProducersRecord — BuildCard for a takeover is
// the Composer over the events the producer holds, composed at standby against
// the injected clock (DR-7, DR-20).
//
// THE TWO FORMS ARE THE CARD'S NOW, NOT THE PRODUCER'S (MVS-D-77, T3.10b). One
// alert carries its own broadcast tail inside its line; several get a head, a
// line each and a closing tail. The choice used to be a flag the producer set
// beside each alert — which meant the producer decided the SHAPE of a burst it
// did not schedule. It follows from how many records the card reads, which is
// the one thing that cannot disagree with the burst the Director planned.
func TestABreakingAlertIsBuiltFromTheProducersRecord(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	lead := lineup.BurstID("a1")
	single := onlyBuilt(t, b.x.run(context.Background(), lineup.BuildCard{
		ID: lead, Slot: lineup.BreakingAlert, Subject: "a1", Refs: []string{"a1"}}))
	if single.ID != lead {
		t.Errorf("built %q, want %s", single.ID, lead)
	}
	if want := alertNarration(nil, tornado(), render.Clock12, execNow); single.Script.Text() != want {
		t.Errorf("a single alert built as\n  %q\nwant today's narration\n  %q", single.Script.Text(), want)
	}
	if !strings.Contains(single.Script.Text(), "Tornado Warning") {
		t.Errorf("the line %q does not name the hazard", single.Script.Text())
	}
	// A LONE ALERT GETS NO HEAD AND NO TAIL. An absent part is absent rather
	// than present and empty, so the Reader has nothing to skip and no gap to
	// pace around.
	if n := len(single.Script.Lines(lineup.PartHead)) + len(single.Script.Lines(lineup.PartTail)); n != 0 {
		t.Errorf("a single alert composed %d structural parts; it carries its own tail", n)
	}

	b.known["a2"] = globalfeed.Event{ID: "a2", Class: globalfeed.ClassSevereWx, Type: "Flash Flood Warning",
		Location: "Fallbrook, CA", At: execNow.Add(-2 * time.Minute), Until: execNow.Add(time.Hour), Source: "NWS"}
	under := onlyBuilt(t, b.x.run(context.Background(), lineup.BuildCard{
		ID: lead, Slot: lineup.BreakingAlert, Subject: "a1", Refs: []string{"a1", "a2"}}))
	lines := under.Script.Lines(lineup.PartLine)
	if len(lines) != 2 {
		t.Fatalf("a two-alert burst composed %d lines, want one per alert", len(lines))
	}
	if want := breakingLine(nil, tornado(), true, render.Clock12, execNow); lines[0].Text != want {
		t.Errorf("an alert under a head built as\n  %q\nwant today's burst line\n  %q", lines[0].Text, want)
	}
	if len(under.Script.Lines(lineup.PartHead)) != 1 || len(under.Script.Lines(lineup.PartTail)) != 1 {
		t.Errorf("a burst opens with a head and closes with a tail, got %v", under.Script.Parts)
	}
	if lines[0].Text == single.Script.Text() {
		t.Error("the two forms read the same; how many alerts the card reads reached nothing")
	}
	// AND EACH LINE NAMES ITS OWN ALERT, or the band cannot follow the read.
	if lines[0].Ref != "a1" || lines[1].Ref != "a2" {
		t.Errorf("the lines refer to %q and %q, want the producer's own ids", lines[0].Ref, lines[1].Ref)
	}
}

// TestABuildForACardTheProducerDoesNotKnowFailsTheCard. A card the producer
// cannot account for is not built from nothing: it is failed, so the schedule
// re-plans around it (DR-21) rather than waiting at standby for words that
// will never come.
func TestABuildForACardTheProducerDoesNotKnowFailsTheCard(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	failed := onlyFailed(t, b.x.run(context.Background(), lineup.BuildCard{ID: "zz", Slot: lineup.BreakingAlert, Subject: "zz"}))
	if failed.ID != "zz" || failed.Reason == "" {
		t.Errorf("got %#v, want zz failed with a reason", failed)
	}
	if r := b.reported(); len(r) != 1 || !strings.HasPrefix(r[0], "build(zz)") {
		t.Errorf("reported %v, want the build of zz, once", r)
	}
}

// TestAScriptThatRendersNothingFailsTheCardRatherThanBuildingSilence. Today an
// empty line is a silent hold. Under the lineup a Built with no words trips the
// Director's own invariant and the card would sit at standby for ever — so the
// executor fails it, loudly, instead of handing the schedule a blank.
func TestAScriptThatRendersNothingFailsTheCardRatherThanBuildingSilence(t *testing.T) {
	// The built-in tree always says something, so silence takes a broken
	// override — which is the one way Say renders "" (round 4, A-13).
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "breaking"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "breaking", "single.txt"), []byte("{{.Line"), 0o600); err != nil {
		t.Fatal(err)
	}
	broken := script.New(dir)
	if got := breakingLine(broken, tornado(), false, render.Clock12, execNow); got != "" {
		t.Fatalf("the fixture is not broken: the line reads %q", got)
	}
	b := newBench(t, &scriptVoice{})
	b.x.scripts = broken
	card := lineup.BurstID("a1")
	failed := onlyFailed(t, b.x.run(context.Background(), lineup.BuildCard{
		ID: card, Slot: lineup.BreakingAlert, Subject: "a1", Refs: []string{"a1"}}))
	if failed.ID != card || !strings.Contains(failed.Reason, "nothing") {
		t.Errorf("got %#v, want %s failed for having nothing to say", failed, card)
	}
}

// TestSpeakReadsThroughTheNarratorAndComesHomeFinished. Speak is the existing
// render and play, run through today's arbiter so the duck keeps its one owner
// across Phase 2 (RD-2), in the takeover's class — the bars keep to the
// broadcast, not the alert. It comes home as Finished, and only then.
func TestSpeakReadsThroughTheNarratorAndComesHomeFinished(t *testing.T) {
	for _, slot := range []lineup.Slot{lineup.BreakingAlert} { // the rail's one slot (T3.10 red team)
		t.Run(slot.String(), func(t *testing.T) {
			v := &scriptVoice{dur: 2 * time.Second}
			b := newBench(t, v)
			out := b.x.run(context.Background(), lineup.Speak{ID: "a1", Slot: slot, Script: lineup.Say("a tornado warning is in effect")})
			if len(out) != 1 {
				t.Fatalf("got %v, want one Finished", out)
			}
			if fin, ok := out[0].(lineup.Finished); !ok || fin.ID != "a1" {
				t.Errorf("got %#v, want Finished{a1}", out[0])
			}
			if got := v.got(); got != "duck,aside:a tornado warning is in effect,restore" {
				t.Errorf("the voice saw %q; want one duck, the line aside, one restore", got)
			}
			// THE REMAINDER, NOT THE WHOLE. The Reader holds what is LEFT of a
			// part's length given the work already done (holdRest) — that
			// subtraction is the overlap MVS-D-72 exists for, and it is why a
			// gap in a burst never contains a render. So the hold lands just
			// under the nominal length rather than exactly on it.
			if got := b.held(); got > 2*time.Second || got < 2*time.Second-50*time.Millisecond {
				t.Errorf("held %v, want the line's own length (less the work already done)", got)
			}
		})
	}
}

// TestSpeakWithNoVoiceStillHoldsSoTheCalloutCanBeRead is P4 F10 carried over:
// with no voice the line plays nothing, and the card still holds the air for
// the fixed time so the band's callout is readable — never blitted past.
func TestSpeakWithNoVoiceStillHoldsSoTheCalloutCanBeRead(t *testing.T) {
	b := newBench(t, nil)
	out := b.x.run(context.Background(), lineup.Speak{ID: "a1", Slot: lineup.BreakingAlert, Script: lineup.Say("words")})
	if len(out) != 1 {
		t.Fatalf("got %v, want one Finished", out)
	}
	if got := b.held(); got > breakingHold || got < breakingHold-50*time.Millisecond {
		t.Errorf("held %v with no voice, want the fixed %v less the work already done", got, breakingHold)
	}
}

// TestSpeakEndedByTheContextComesHomeFailedNotFinished.
//
// A Finished for words that were never finished would tell the schedule a read
// happened that did not — which is why this cannot come home Finished. It used
// to come home with NOTHING, and that was the other half of the same defect: a
// card that says nothing stays ON AIR for ever, and the band keeps a callout for
// a read that has stopped (DR-24). Failed says exactly what happened, and the
// release is paired with it.
func TestSpeakEndedByTheContextComesHomeFailedNotFinished(t *testing.T) {
	b := newBench(t, &scriptVoice{dur: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := b.x.run(ctx, lineup.Speak{ID: "a1", Slot: lineup.BreakingAlert, Script: lineup.Say("words")})
	failed := onlyFailed(t, out)
	if failed.ID != "a1" {
		t.Errorf("the failure names %q, want the card whose read was cut", failed.ID)
	}
	if failed.Reason == "" {
		t.Error("a read cut short with no reason is a card leaving the air for no stated cause")
	}
	for _, e := range out {
		if _, ok := e.(lineup.Finished); ok {
			t.Error("a cancelled read came home Finished; the words never finished")
		}
	}
}

// TestSpeakForACardWithNothingToSayIsFailedNotAired. Card.To(OnAir) already
// refuses this; the executor refuses it again because it is the last thing
// between the schedule and a silent hold with a callout promised (DR-18).
func TestSpeakForACardWithNothingToSayIsFailedNotAired(t *testing.T) {
	v := &scriptVoice{}
	b := newBench(t, v)
	failed := onlyFailed(t, b.x.run(context.Background(), lineup.Speak{ID: "a1", Slot: lineup.BreakingAlert}))
	if failed.ID != "a1" {
		t.Errorf("failed %q, want a1", failed.ID)
	}
	if v.got() != "" {
		t.Errorf("the voice saw %q for a card with no words", v.got())
	}
}

// TestTheCueReachesTheBandAndIsRecorded (DR-18). The cue is the existing
// takeover message for the alert the producer holds; it returns no event — the
// voice never waits on the band — and the record is what a test, and the
// diagnostic, can read afterwards.
func TestTheCueReachesTheBandAndIsRecorded(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	if out := b.x.run(context.Background(), lineup.CueTicker{ID: "a1", Headline: "Tornado Warning"}); len(out) != 0 {
		t.Errorf("a cue came home with %v; it is fire-and-trust", out)
	}
	msgs := b.sent()
	if len(msgs) != 1 {
		t.Fatalf("the band got %v, want one takeover message", msgs)
	}
	cue, ok := msgs[0].(tty.TickerBreakingMsg)
	if !ok || cue.Item.ID != "a1" || cue.Item.Head == "" {
		t.Errorf("the band got %#v, want the takeover for a1 with its head", msgs[0])
	}
	if !b.x.band.has("cue(a1)") {
		t.Errorf("the record reads %v; the cue is not in it", b.x.band.recent())
	}
}

// TestTheReleaseReachesTheBandAndIsRecorded — the cue's other half (DR-24).
func TestTheReleaseReachesTheBandAndIsRecorded(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	if out := b.x.run(context.Background(), lineup.ReleaseTicker{ID: "a1"}); len(out) != 0 {
		t.Errorf("a release came home with %v", out)
	}
	msgs := b.sent()
	if len(msgs) != 1 {
		t.Fatalf("the band got %v, want one done message", msgs)
	}
	if _, ok := msgs[0].(tty.TickerBreakingDoneMsg); !ok {
		t.Errorf("the band got %#v, want the takeover's end", msgs[0])
	}
	if !b.x.band.has("release(a1)") {
		t.Errorf("the record reads %v; the release is not in it", b.x.band.recent())
	}
}

// TestACueTheProducerCannotAccountForIsReportedAndFailsNothing. The words must
// still read (DR-18): a band that cannot be cued is a diagnostic, not a reason
// to take the card off the air.
func TestACueTheProducerCannotAccountForIsReportedAndFailsNothing(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	if out := b.x.run(context.Background(), lineup.CueTicker{ID: "zz"}); len(out) != 0 {
		t.Errorf("a cue that could not be built came home with %v; the read must go on", out)
	}
	if len(b.sent()) != 0 {
		t.Errorf("the band got %v for a card nobody can account for", b.sent())
	}
	if r := b.reported(); len(r) != 1 || !strings.HasPrefix(r[0], "cue(zz)") {
		t.Errorf("reported %v, want the cue of zz, once", r)
	}
	if b.x.band.has("cue(zz)") {
		t.Error("the record claims a cue the band never got")
	}
}

// TestAPublishIsCarriedAndDeclined — the effect is EMITTED and nobody reads it.
//
// The seam it used to reach was `func(lineup.Lineup) {}`, a no-op behind a
// nil-guard, dispatched once a second (red team 2026-09-05): that reads as a
// wired feature and is not one. The Broadcaster surface that consumes a
// published lineup is 0.15.0's.
//
// THE EFFECT STAYS EMITTED, and this pins why: the "readers are told last"
// ordering in settle depends on it existing, so it must be carried and declined
// rather than dropped from the set.
func TestAPublishIsCarriedAndDeclined(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	d := lineup.New(lineup.Settings{Max: 10}, execNow)
	d, fx := d.Step(lineup.Arrived{Arrivals: arrivals("a1")})
	// IT IS STILL THE LAST THING A STEP DESCRIBES. A publish nobody reads must
	// still be emitted in that position, or the ordering it guarantees is gone
	// the day a reader arrives.
	if len(fx) == 0 {
		t.Fatal("a step described nothing")
	}
	if _, ok := fx[len(fx)-1].(lineup.Publish); !ok {
		t.Errorf("the last effect is %T, want the publish", fx[len(fx)-1])
	}
	if out := b.x.run(context.Background(), lineup.Publish{Lineup: d.Lineup()}); len(out) != 0 {
		t.Errorf("a publish came home with %v", out)
	}
}

// TestEffectsNotYetEmittedAreDeclinedNotHalfDone — the closed set, walked. Every
// member is either performed or declined by name with the task that brings its
// executor; nothing falls through to the default, which is what catches the set
// growing a member the executors never learned.
//
// `Tune` LEFT THIS TABLE at T3.2b, which is the point of the table: an effect
// stops being declined when its executor arrives, and the row goes with it. It
// is asserted now by TestTheDirectorsTuneLeavesTheDuckAlone, which checks the
// thing that actually matters about it — that an automatic tune does not lift
// the alert duck.
func TestEffectsNotYetEmittedAreDeclinedNotHalfDone(t *testing.T) {
	for _, tc := range []struct {
		effect lineup.Effect
		failed string // the card that must be failed, or empty
		task   string // the task named in the reason
	}{
		{lineup.BuildCard{ID: "r1", Slot: lineup.LocationReport, Subject: "33.2887,-117.2179"}, "r1", "T3.2"},
		{lineup.BuildCard{ID: "s1", Slot: lineup.SevereRead, Subject: "s1"}, "s1", "T3.2"},
		{lineup.Speak{ID: "r1", Slot: lineup.LocationReport, Script: lineup.Say("the report")}, "r1", "T3.2"},
		{lineup.Speak{ID: "t1", Slot: lineup.Transition, Script: lineup.Say("we now return")}, "t1", "T3.2"},
		{lineup.BuildCard{ID: "h1", Slot: lineup.Transition, Subject: "h1"}, "h1", "proposal"},
	} {
		t.Run(lineup.Describe(tc.effect), func(t *testing.T) {
			v := &scriptVoice{}
			b := newBench(t, v)
			out := b.x.run(context.Background(), tc.effect)
			if tc.failed == "" {
				if len(out) != 0 {
					t.Errorf("got %v for an effect that names no card, want nothing", out)
				}
			} else if f := onlyFailed(t, out); f.ID != tc.failed {
				t.Errorf("failed %q, want %q", f.ID, tc.failed)
			}
			r := b.reported()
			if len(r) != 1 || !strings.Contains(r[0], tc.task) {
				t.Errorf("reported %v, want one line naming %s", r, tc.task)
			}
			if v.got() != "" || len(b.sent()) != 0 {
				t.Errorf("a declined effect reached the voice (%q) or the band (%v)", v.got(), b.sent())
			}
		})
	}
}

// TestARealStepsEffectsAreBuiltNotDeclined feeds the executors what Step
// actually emits, so the seam between the two is exercised and not assumed.
//
// IT CHANGED SHAPE AT T3.10b, exactly as its previous form said it would. Step
// queues ONE takeover per burst (MVS-D-77), so the build names a BURST and
// carries the alert ids it reads; the Composer turns those into the card's
// words. Between T3.10a and T3.10b this pinned the DECLINE, because there was
// no Composer on this side of the seam yet and the alternative — resolving the
// producer's record from the card's own id — would have read the lead alert
// alone and called the burst done.
func TestARealStepsEffectsAreBuiltNotDeclined(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	d := lineup.New(lineup.Settings{Max: 10}, execNow)
	_, fx := d.Step(lineup.Arrived{Arrivals: []lineup.Arrival{{ID: "a1", Category: category.Warnings,
		Headline: "Tornado Warning", Subject: "a1", Severity: 50, At: execNow.Add(-time.Minute)}}})
	var build lineup.BuildCard
	for _, f := range fx {
		if v, ok := f.(lineup.BuildCard); ok {
			build = v
		}
	}
	if build.ID == "" {
		t.Fatalf("the step emitted %v; no build to run", fx)
	}
	if build.ID != lineup.BurstID("a1") {
		t.Errorf("the build names %q, want the takeover the burst was queued as", build.ID)
	}
	if build.Slot != lineup.BreakingAlert {
		t.Errorf("the build carries slot %q, want the card's %q", build.Slot, lineup.BreakingAlert)
	}
	// THE ORDER RIDES ON THE EFFECT (BD-8). Without it the Composer would have
	// to re-plan or look the card up from a lineup the publish is concurrently
	// replacing, and either would be a second planner.
	if len(build.Refs) != 1 || build.Refs[0] != "a1" {
		t.Fatalf("the build carries refs %v, want the alert the burst reads", build.Refs)
	}

	built := onlyBuilt(t, b.x.run(context.Background(), build))
	if built.ID != build.ID {
		t.Errorf("the words came home for %q, want the card that was built", built.ID)
	}
	if built.Script.Empty() {
		t.Error("a real step's build came home with nothing to say")
	}
	if !strings.Contains(built.Script.Text(), "Tornado Warning") {
		t.Errorf("the card reads %q; it does not name the hazard", built.Script.Text())
	}
	// THE PRODUCER WAS ASKED BY THE ALERT'S ID, NOT THE CARD'S. The card is a
	// burst and names no alert; asking by it is the mistake that would read the
	// lead alone and call the burst done.
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.asked["a1"] {
		t.Errorf("the producer was asked for %v, not for the alert the card reads", b.asked)
	}
	if b.asked[lineup.BurstID("a1")] {
		t.Error("the producer was asked for the CARD's id; a burst is not one of its own records")
	}
}

// TestAMutedReadIsDeclinedAndNothingIsConsumed — MVS-D-78, and mU2 found this
// UNPINNED: the mutant deleted the mute check and NOTHING FAILED.
//
// The producer already refuses to send a burst while the listener is muted, so
// this is the second door: `[M]` can land between an arrival and its read. Read
// inaudibly, the card would cue the band, MARK EVERY ALERT READ and finish — a
// tornado warning consumed in silence and never sounded, which is the exact
// defect that ruling exists to prevent.
//
// DECLINED, NOT HELD. The alerts were never marked, so the producer offers them
// again on its next cycle.
func TestAMutedReadIsDeclinedAndNothingIsConsumed(t *testing.T) {
	card := lineup.BurstID("a1")
	// A script with a REF on its line, so the mark hook has something to record:
	// a script whose parts named no alert could not be consumed either way, and
	// this test would pass against the defect.
	script := lineup.Script{Tone: "warning", Parts: []lineup.Part{
		{Kind: lineup.PartLine, Text: "a tornado warning is in effect", Ref: "a1"},
	}}
	speak := lineup.Speak{ID: card, Slot: lineup.BreakingAlert, Script: script}

	b := newBench(t, &scriptVoice{})
	b.muted = true
	failed := onlyFailed(t, b.x.run(context.Background(), speak))
	if failed.ID != card {
		t.Errorf("the decline names %q, want the card that was not read", failed.ID)
	}
	if !strings.Contains(failed.Reason, "muted") {
		t.Errorf("the decline reads %q; it must say why, or the timeline cannot explain the silence", failed.Reason)
	}
	b.mu.Lock()
	marked, voiced := append([]string(nil), b.marked...), b.voice.got()
	b.mu.Unlock()
	if len(marked) != 0 {
		t.Errorf("a muted read marked %v as read aloud; those hazards are now consumed in silence", marked)
	}
	if voiced != "" {
		t.Errorf("a muted read spoke %q", voiced)
	}
	for _, m := range b.sent() {
		if _, ok := m.(tty.TickerBreakingMsg); ok {
			t.Error("a muted read cued the band for words it will not say")
		}
	}

	// THE CONTROL. Unmuted, the same speak reads and marks — without it this
	// passes against a speak that never works at all, which is the shape that
	// let five fixtures prove nothing this release.
	c := newBench(t, &scriptVoice{})
	if out := c.x.run(context.Background(), speak); len(out) != 1 {
		t.Fatalf("unmuted, the same read produced %v, want one Finished", out)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !equalStrings(c.marked, []string{"a1"}) {
		t.Errorf("unmuted, the read marked %v, want the alert it said", c.marked)
	}
}

// equalStrings is a small helper: two string slices, same order.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestTheBandRecordIsBounded. It is a diagnostic ring, not a log: it keeps the
// most recent lines and never grows with the day (P10-03).
func TestTheBandRecordIsBounded(t *testing.T) {
	r := &bandRecord{}
	for i := range bandRecordCap + 5 {
		r.note("cue(" + string(rune('a'+i%26)) + ")")
	}
	if got := len(r.recent()); got != bandRecordCap {
		t.Errorf("the record holds %d lines, want the cap %d", got, bandRecordCap)
	}
	if !r.has("cue(" + string(rune('a'+(bandRecordCap+4)%26)) + ")") {
		t.Error("the newest line was the one dropped")
	}
}

// TestExecutorsRefuseToBeBuiltWithoutTheirSeams. A nil seam would panic on the
// first effect, on a worker, where the panic is contained and read as a fault of
// the card rather than as the wiring bug it is.
func TestExecutorsRefuseToBeBuiltWithoutTheirSeams(t *testing.T) {
	whole := func() executors {
		return executors{
			voice: testDirector(nil, nil), clock: func() render.Clock { return render.Clock12 },
			now: func() time.Time { return execNow }, mc: newMastercontrol(nil, func(tea.Msg) {}),
			audible: func() bool { return true }, muted: func() bool { return false },
			alert:     func(string) (globalfeed.Event, bool) { return globalfeed.Event{}, false },
			mark:      func(string) {},
			readAloud: func(string) bool { return false },
			report:    func(lineup.Effect, string) {},
			cutTo:     func(string) {},
			escalate:  func(string) {},
		}
	}
	if newExecutors(whole()) == nil {
		t.Fatal("a whole set was refused")
	}
	// THE LIST IS EXPLICIT; ITS COMPLETENESS IS DERIVED.
	//
	// This was a hand-written map alone, and the `escalate` seam added at T3.4
	// was simply not in it — so the one guard between "a fault channel exists"
	// and "a fault channel reaches nobody" went unmeasured, and a mutant
	// deleting that guard SURVIVED. A list you must remember to extend goes
	// stale on the day it matters.
	//
	// reflect cannot SET an unexported field, so the strippers stay explicit.
	// What is derived is that they COVER the struct: add a seam and this fails
	// until it is listed or declared optional, which is the half that was
	// missing (D-8).
	strippers := map[string]func(*executors){
		"voice":     func(x *executors) { x.voice = nil },
		"clock":     func(x *executors) { x.clock = nil },
		"now":       func(x *executors) { x.now = nil },
		"mc":        func(x *executors) { x.mc = nil },
		"cutTo":     func(x *executors) { x.cutTo = nil },
		"audible":   func(x *executors) { x.audible = nil },
		"alert":     func(x *executors) { x.alert = nil },
		"mark":      func(x *executors) { x.mark = nil },
		"readAloud": func(x *executors) { x.readAloud = nil },
		"muted":     func(x *executors) { x.muted = nil },
		"report":    func(x *executors) { x.report = nil },
		"escalate":  func(x *executors) { x.escalate = nil },
	}
	optional := map[string]bool{
		"scripts": true, // nil means the built-in script tree
		"band":    true, // built by newExecutors itself, never passed in
	}
	v := reflect.ValueOf(whole())
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		switch v.Field(i).Kind() {
		case reflect.Func, reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
		default:
			continue // not a seam: it cannot be nil
		}
		if optional[f.Name] {
			continue
		}
		if _, ok := strippers[f.Name]; !ok {
			t.Errorf("seam %q is neither checked nor declared optional: a nil one would reach production unmeasured", f.Name)
		}
	}
	for name, strip := range strippers {
		x := whole()
		strip(&x)
		if newExecutors(x) != nil {
			t.Errorf("a set with no %s was built", name)
		}
	}
}

// TestTheDuckAndItsRestoreReachTheEffector — the half of T2.3 that MVS-D-67
// makes a prerequisite for the alert rail: the bed ducks ONCE for a whole rail
// drain and is restored only after the tail, so the effects that do it must
// exist and must go through the one owner.
func TestTheDuckAndItsRestoreReachTheEffector(t *testing.T) {
	v := &scriptVoice{}
	b := newBench(t, v)

	if out := b.x.run(context.Background(), lineup.Duck{}); len(out) != 0 {
		t.Errorf("a duck came home with %v; it is an instruction, not a question", out)
	}
	if got := v.got(); got != "duck" {
		t.Fatalf("the voice saw %q, want the bed given way to", got)
	}
	if !b.x.mc.givenWay() {
		t.Error("the effector does not report the bed as ducked")
	}

	// IDEMPOTENT AT THE OWNER (D-1). The narration arbiter ducks for its own
	// sequences too; between them the bed must dip once, not twice, or the
	// listener hears it step down again under a read already in progress.
	b.x.run(context.Background(), lineup.Duck{})
	if got := v.got(); got != "duck" {
		t.Errorf("the voice saw %q; a second duck dipped the bed again", got)
	}

	if out := b.x.run(context.Background(), lineup.Restore{}); len(out) != 0 {
		t.Errorf("a restore came home with %v", out)
	}
	if got := v.got(); got != "duck,restore" {
		t.Errorf("the voice saw %q, want the bed given back once", got)
	}
	if b.x.mc.givenWay() {
		t.Error("the effector still reports the bed as ducked after restoring it")
	}

	// And a restore with nothing ducked lifts nothing, rather than lifting a
	// broadcast nobody dipped.
	b.x.run(context.Background(), lineup.Restore{})
	if got := v.got(); got != "duck,restore" {
		t.Errorf("the voice saw %q; a restore with no duck outstanding still acted", got)
	}
}

// TestTheBandHasOneWriter — D-1, as a property rather than a promise.
//
// The takeover message was constructed in app/ticker.go for the live path and in
// app/executors.go for the Director's, so one rule had two carriers in two
// files. Both go through the effector now, and this asserts a cue produces the
// message the band's own item builder makes — so the two paths cannot drift into
// sending different things for the same event.
func TestTheBandHasOneWriter(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	b.x.run(context.Background(), lineup.CueTicker{ID: "a1", Headline: "Tornado Warning"})
	msgs := b.sent()
	if len(msgs) != 1 {
		t.Fatalf("the band got %v, want one message", msgs)
	}
	got, ok := msgs[0].(tty.TickerBreakingMsg)
	if !ok {
		t.Fatalf("the band got %#v, want a takeover", msgs[0])
	}
	if want := breakingItem(tornado()); got.Item != want {
		t.Errorf("the cue sent %#v; the band's own builder makes %#v", got.Item, want)
	}
}

// TestTheBedIsDippedOnceForAWholeDrain is MVS-D-67, and the reason T2.3 is a
// prerequisite for the rail.
//
// The narration arbiter gives way per SEQUENCE and takes the bed back the
// moment nothing is waiting, so a rail of two cards dipped, lifted, dipped and
// lifted between them — the listener hearing the broadcast surge back up
// between two alerts of one burst. Measured before the hold existed:
//
//	duck, aside:first, restore, duck, aside:second, restore
//
// Held by the Director for the whole drain it is one dip and one lift, and the
// per-sequence take-back in between is refused.
//
// THE UNBRACKETED CASE IS ASSERTED TOO, and it is the more important half: it
// is today's Observer behaviour, which this change must not alter. Nothing
// emits Duck yet, so every existing path must sound exactly as it did.
func TestTheBedIsDippedOnceForAWholeDrain(t *testing.T) {
	read := func(bracketed bool) string {
		v := &scriptVoice{}
		b := newBench(t, v)
		ctx := context.Background()
		if bracketed {
			b.x.run(ctx, lineup.Duck{})
		}
		b.x.run(ctx, lineup.Speak{ID: "a1", Slot: lineup.BreakingAlert, Script: lineup.Say("first")})
		b.x.run(ctx, lineup.Speak{ID: "a1", Slot: lineup.BreakingAlert, Script: lineup.Say("second")})
		if bracketed {
			b.x.run(ctx, lineup.Restore{})
		}
		return v.got()
	}

	if got, want := read(true), "duck,aside:first,aside:second,restore"; got != want {
		t.Errorf("held for the drain the bed heard %q, want %q — one dip, one lift", got, want)
	}
	// Today's behaviour, unchanged: the arbiter still dips and lifts per read.
	if got, want := read(false), "duck,aside:first,restore,duck,aside:second,restore"; got != want {
		t.Errorf("unheld the bed heard %q, want %q — Observer must sound exactly as it did", got, want)
	}
}

// TestTheHeldBedIsGivenBackEvenIfNothingElseSpeaks. A hold that nothing lifts is
// a broadcast that never comes back, which is worse than pumping.
func TestTheHeldBedIsGivenBackEvenIfNothingElseSpeaks(t *testing.T) {
	v := &scriptVoice{}
	b := newBench(t, v)
	ctx := context.Background()
	b.x.run(ctx, lineup.Duck{})
	if !b.x.mc.givenWay() {
		t.Fatal("the bed was not dipped")
	}
	b.x.run(ctx, lineup.Restore{})
	if b.x.mc.givenWay() {
		t.Error("the bed is still down after the rail released it; the broadcast never comes back")
	}
	if got := v.got(); got != "duck,restore" {
		t.Errorf("the bed heard %q, want one dip and one lift", got)
	}
}

// TestTheRailNeverLiftsTheBedOverALiveRead — the finding a fresh review found
// and its mutant proved unpinned.
//
// Releasing the hold used to ask the arbiter whether it was idle and then act on
// that answer, with the question answered outside the effector's lock: a job
// admitted between the two had the bed restored out from under it, and the
// listener heard the broadcast surge to full volume over a read in progress —
// the exact class the single-owner work exists to prevent.
//
// PINNED SEQUENTIALLY, so it does not depend on a scheduler. A read is put on
// the air and held there; releasing the rail's hold must leave the bed down,
// because something is speaking over it.
func TestTheRailNeverLiftsTheBedOverALiveRead(t *testing.T) {
	v := &scriptVoice{}
	b := newBench(t, v)
	ctx := context.Background()

	// The rail holds the bed for its drain.
	b.x.run(ctx, lineup.Duck{})
	if !b.x.mc.givenWay() {
		t.Fatal("the rail did not dip the bed")
	}

	// A read takes the air and stays on it: Run holds the job until its
	// sequence returns, so inside the sequence onAir is set.
	onAir := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		b.nar.Run(ctx, narrateRead, cast.SevereRead, true, func(context.Context, *speaker) {
			close(onAir)
			<-b.release
		})
	}()
	<-onAir

	// The rail's tail plays and it gives the bed back — but something is still
	// speaking, so the bed must stay down.
	b.x.run(ctx, lineup.Restore{})
	if !b.x.mc.givenWay() {
		t.Error("the bed was given back while a read was on the air; the broadcast surged over it")
	}

	close(b.release)
	<-done
	// And once nothing is speaking, the ordinary path lifts it.
	if b.x.mc.givenWay() {
		t.Error("the bed never came back after everything finished")
	}
}

// TestTheBedIsNeverLiftedBetweenTheDipAndTheClaim is F-D5, and it is the
// ACQUIRE side of the defect mE6 pinned on the release side.
//
// `hold` once dipped the bed under the lock, let the lock go, and took it again
// to set `held`. `hold` runs on the executor's goroutine and `takeBack` on the
// arbiter's, sharing only this mutex — so a take-back landing in that window saw
// held=false over a ducked bed and lifted it, and `held` was then set over a
// broadcast already back at full volume. The rail reads its whole drain against
// an undipped bed, which is the exact thing MVS-D-67 asks for and the opposite
// of what it gets.
//
// THE PROBE KEEPS A TAKE-BACK IN FLIGHT ACROSS THE WINDOW rather than hoping to
// land in it: a goroutine calls takeBack in a tight loop for the whole duration
// of the hold, so the dip is met by a lift within nanoseconds of becoming
// visible. A first attempt that waited to OBSERVE the dip before lifting passed
// against the defective code and was discarded — it never ran before hold had
// finished, and an instrument that cannot fail is not measuring.
//
// When the dip and the claim are one critical section, no takeBack can run
// between them and the bed stays down. When they are two, one gets in.
// probeRounds is sized from the MEASURED hit rate, not guessed. At 300 rounds
// the probe caught the defective code in four runs out of five — a test that
// misses a real defect one time in five is a coin toss, not a gate. The
// per-round rate that implies is ~0.5 %, so 5000 rounds miss with probability
// ~1e-12, and the whole loop still costs well under a second.
const probeRounds = 5000

func TestTheBedIsNeverLiftedBetweenTheDipAndTheClaim(t *testing.T) {
	// IT SETS ITS OWN PARALLELISM, and the spin below is deliberately TIGHT.
	// Both facts were measured rather than chosen. Given one P the probe cannot
	// measure at all: `hold` runs from the dip to the claim with no preemption
	// point, the window is never observable, and the probe reports the DEFECTIVE
	// code clean. Yielding inside the loop instead of fixing the parallelism
	// looks tidier and is worse — it drops the hit rate from five runs in five
	// to one in five, buying timeout-safety by rarely landing in the window at
	// all. An instrument whose answer depends on the core count of whatever runs
	// it is not an instrument.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(2))

	for i := 0; i < probeRounds; i++ {
		v := &probeVoice{onPlay: func(clip) {}}
		m := newMastercontrol(v, func(tea.Msg) {})

		stop, done, started := make(chan struct{}), make(chan struct{}), make(chan struct{})
		go func() {
			defer close(done)
			m.takeBack() // a no-op: nothing is ducked yet
			close(started)
			for {
				select {
				case <-stop:
					return
				default:
				}
				m.takeBack()
			}
		}()
		<-started // the lift is already running when the dip happens

		m.hold()
		close(stop)
		<-done

		m.mu.Lock()
		held, ducked := m.held, m.ducked
		m.mu.Unlock()
		if held && !ducked {
			t.Fatalf("iteration %d: the rail holds the bed but the bed is up — "+
				"a take-back slipped between the dip and the claim, and the whole "+
				"drain now reads against a broadcast at full volume", i)
		}
	}
}

// DR-21 — THE SELF-HEALING FALLBACK RAISES NOTHING.
//
// Named in the requirement's own verification column: "a test that today's
// relay→synth fallback raises no modal". A relay that dies is handled without
// anybody being told, because it fixes itself — the engine falls through its
// mount list and then to the synthesized report, and the listener keeps
// hearing a station. Escalating that would teach them to dismiss the window
// that matters, and on a safety surface a noise regression is a safety
// regression.
//
// The distinction the release turns on: a relay that DIES is routed around; a
// relay that is UP AND SILENT is not, because nothing detects it and nothing
// falls through (MVS-D-76). Same component, opposite grade.
func TestDR21ARoutedFaultNeverReachesAPerson(t *testing.T) {
	b := newBench(t, &scriptVoice{})

	// A fault the Director routed around never becomes an Escalate at all, so
	// the executors are asked to perform no escalation.
	for _, f := range []lineup.Effect{
		lineup.Duck{},
		lineup.Restore{},
		lineup.Tune{Ref: "oceanside-ca"},
	} {
		b.x.run(context.Background(), f)
	}
	b.mu.Lock()
	got := append([]string(nil), b.escalations...)
	b.mu.Unlock()
	if len(got) != 0 {
		t.Errorf("routine effects must reach nobody, got %v", got)
	}

	// And the one effect that IS an escalation reaches a person, with the words
	// the producer gave — the channel is wired, not merely quiet.
	b.x.run(context.Background(), lineup.Escalate{ID: "c1", Reason: "no voice could read it"})
	b.mu.Lock()
	got = append([]string(nil), b.escalations...)
	b.mu.Unlock()
	if len(got) != 1 || !strings.Contains(got[0], "no voice") {
		t.Errorf("a stopped schedule reaches a person with the reason, got %v", got)
	}
}

// THE DIRECTOR'S SPEAK READS THROUGH THE ONE READER (T3.8).
//
// This is what the change to a script-carrying Speak was FOR. The Director's
// executor and the live takeover now perform a card the same way — the tone,
// MVS-D-72's pauses, and the overlap that keeps a render out of every gap.
//
// Two implementations of a ruling the HUM LEAD found BY EAR would be two places
// for it to drift, and the one nobody edited would be the one that bites. The
// assertion is that a MULTI-PART script is performed as its parts: a Reader
// handed a flat string could not do this, which is precisely what Speak used to
// hand it.
func TestTheDirectorsSpeakPerformsAScriptsParts(t *testing.T) {
	v := &scriptVoice{}
	b := newBench(t, v)
	b.nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }

	out := b.x.run(context.Background(), lineup.Speak{
		ID: "a1", Slot: lineup.BreakingAlert,
		Script: lineup.Script{Parts: []lineup.Part{
			{Kind: lineup.PartHead, Text: "the following alerts have been declared"},
			{Kind: lineup.PartLine, Text: "a tornado warning is in effect", Ref: "a1"},
			{Kind: lineup.PartTail, Text: "for more information press W"},
		}},
	})
	if len(out) != 1 {
		t.Fatalf("got %v, want one Finished", out)
	}
	// THREE SEPARATE UTTERANCES, not one containing three phrases.
	//
	// The first version of this asserted that all three texts appeared in
	// order — and PASSED against a script flattened to one part, because
	// Script.Text() joins the parts with spaces and the joined string contains
	// them all in order. It measured nothing about parts. What only parts can
	// satisfy is that the voice is asked THREE times, with a pause between
	// each, which is the shape a listener hears.
	got := v.got()
	if n := strings.Count(got, "aside:"); n != 3 {
		t.Errorf("the reader speaks each part on its own: %d utterances in %q, want 3", n, got)
	}
	for _, want := range []string{"aside:the following alerts have been declared", "aside:a tornado warning is in effect", "aside:for more information press W"} {
		if !strings.Contains(got, want) {
			t.Errorf("the reader must speak %q on its own; it said %q", want, got)
		}
	}
	if i, j := strings.Index(got, "following"), strings.Index(got, "tornado"); i < 0 || j < 0 || i > j {
		t.Errorf("the head precedes the lines: %q", got)
	}
	if i, j := strings.Index(got, "tornado"), strings.Index(got, "press W"); i < 0 || j < 0 || i > j {
		t.Errorf("the tail follows the lines: %q", got)
	}

	// AND THE BAND IS CUED FROM THE PRODUCER'S RECORD, for the part that names
	// one. The cue is asked for, never constructed here — the band has one
	// owner and the producer owns what an alert IS (D-1).
	if !b.asked["a1"] {
		t.Error("the part's Ref must be resolved through the producer, or the band shows nothing")
	}
}

// TestTheDivertNoticeSaysTheCountAndTheDestination — DR-14, end to end.
//
// THE BURST IS A SNAPSHOT AND THE REMAINDER IS SPOKEN. Plan has computed a
// Divert count since T1.3 and until now NOTHING CARRIED IT TO THE LISTENER: the
// ticker's `diverted` field was written every cycle and read by nobody, and the
// DivertNotice slot was proposed by nothing. Both were retired as dead; this is
// the carrier they were standing in for.
//
// DRIVEN FROM Plan, so the spoken count is the planner's own arithmetic rather
// than a number this test chose — DR-14's second check is "the spoken count
// equals exactly what was not read", and a literal here could not show that.
func TestTheDivertNoticeSaysTheCountAndTheDestination(t *testing.T) {
	for _, tc := range []struct {
		name   string
		arrive int
		max    int
		want   string
		absent string
	}{
		{name: "more arrived than the burst reads", arrive: 9, max: 5, want: "4 other alerts"},
		{name: "exactly one was left out reads SINGULAR", arrive: 6, max: 5, want: "1 other alert"},
		{name: "nothing was left out says nothing about a count",
			arrive: 3, max: 5, want: "press W in Watchpost", absent: "other alert"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evs := make([]globalfeed.Event, 0, tc.arrive)
			for i := range tc.arrive {
				evs = append(evs, globalfeed.Event{ID: fmt.Sprintf("d%02d", i), Source: "NWS",
					Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
					Location: fmt.Sprintf("County %d", i), Severity: globalfeed.SevRed,
					At: execNow.Add(-time.Duration(i) * time.Minute), Until: execNow.Add(time.Hour)})
			}
			// THE PLANNER'S OWN COUNT. Asserting against a literal would pass
			// against a Composer that invented the number.
			b, err := lineup.Plan(arrivalsOf(evs), lineup.Settings{Max: tc.max}, execNow)
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			if want := tc.arrive - min(tc.arrive, tc.max); b.Divert != want {
				t.Fatalf("the fixture diverts %d, want %d; it is not exercising what it claims", b.Divert, want)
			}

			b2 := newBench(t, &scriptVoice{})
			for _, e := range evs {
				b2.known[e.ID] = e
			}
			built := onlyBuilt(t, b2.x.run(context.Background(), lineup.BuildCard{
				ID: b.Takeover.ID, Slot: lineup.BreakingAlert, Subject: b.Takeover.Subject,
				Refs: b.Takeover.Refs, Divert: b.Takeover.Divert}))

			tails := built.Script.Lines(lineup.PartTail)
			if len(tails) != 1 {
				t.Fatalf("a burst closes with ONE tail, got %d: %v", len(tails), built.Script.Parts)
			}
			tail := tails[0].Text
			if !strings.Contains(tail, tc.want) {
				t.Errorf("the tail reads %q, want it to say %q", tail, tc.want)
			}
			if tc.absent != "" && strings.Contains(tail, tc.absent) {
				t.Errorf("the tail reads %q; nothing was left out, so it must not mention %q", tail, tc.absent)
			}
			// THE DESTINATION IS ALWAYS THERE. It is the one edition-specific
			// value in the line, and a count with nowhere to go is worse than
			// no count at all.
			if !strings.Contains(tail, "Watchpost") {
				t.Errorf("the tail reads %q; it never says where the rest are", tail)
			}
			// AND THE COUNT IS NOT SPOKEN AS AN ALERT LINE. DR-15: structural
			// parts are the Director's arrangement, and Max counts alert reads.
			if got := len(built.Script.Lines(lineup.PartLine)); got != min(tc.arrive, tc.max) {
				t.Errorf("the burst reads %d alert lines, want %d — the tail spent the budget", got, min(tc.arrive, tc.max))
			}
		})
	}
}
