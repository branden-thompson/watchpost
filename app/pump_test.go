package app

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// pumpNow is the fixed clock the pump's Director plans against.
var pumpNow = time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

// recorder is a fake executor: it notes what it was asked to do, in the order it
// was asked, and lets a test hold one effect open for as long as it needs.
type recorder struct {
	mu   sync.Mutex
	seen []string

	hold map[string]chan struct{} // effect description -> released when closed
	// stubborn holds ignore cancellation, the way a write already under way
	// does. A hold that honours the context is not in flight after a cancel,
	// so it cannot exercise the drain.
	stubborn map[string]bool
	panics   map[string]bool
	emit     map[string][]lineup.Event
}

func newRecorder() *recorder {
	return &recorder{hold: map[string]chan struct{}{}, stubborn: map[string]bool{},
		panics: map[string]bool{}, emit: map[string][]lineup.Event{}}
}

func (r *recorder) run(ctx context.Context, f lineup.Effect) []lineup.Event {
	desc := lineup.Describe(f)
	r.mu.Lock()
	hold, held := r.hold[desc]
	boom, stubborn := r.panics[desc], r.stubborn[desc]
	r.mu.Unlock()
	if held && stubborn {
		<-hold // mid-write: the context cannot take this back
	}
	if held && !stubborn {
		select {
		case <-hold:
		case <-ctx.Done():
			return nil
		}
	}
	r.mu.Lock()
	r.seen = append(r.seen, desc)
	out := r.emit[desc]
	r.mu.Unlock()
	if boom {
		panic("the executor came apart")
	}
	return out
}

func (r *recorder) log() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.seen...)
}

// awaitContains waits for the executor to have been asked for one named effect.
// It is the honest wait condition when a test cares about a PARTICULAR effect —
// waiting for a COUNT and then asserting on the snapshot asserts against
// whatever happened to have arrived, which is a race dressed as a test.
func awaitContains(t *testing.T, r *recorder, want string) []string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got := r.log()
		if containsExact(got, want) {
			return got
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("waited for %q; the pump saw %v", want, r.log())
	return nil
}

// awaitLog waits for the executor to have seen n effects. It polls a mutex
// rather than sleeping a fixed time: a fixed sleep is either flaky or slow, and
// under -race it is reliably both.
func awaitLog(t *testing.T, r *recorder, n int, why string) []string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if got := r.log(); len(got) >= n {
			return got
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("waited for %d effects (%s); saw %v", n, why, r.log())
	return nil
}

// arrivals is one burst of breaking alerts, named, for the pump's Director to
// plan. It is ONE card on the rail however many alerts are in it (MVS-D-77), so
// a test that wants several cards sends several Arriveds — see rail.
func arrivals(ids ...string) []lineup.Arrival {
	out := make([]lineup.Arrival, 0, len(ids))
	for i, id := range ids {
		out = append(out, lineup.Arrival{ID: id, Category: category.Warnings,
			Headline: id + " headline", Subject: "Bonsall, CA", Severity: 50,
			At: pumpNow.Add(-time.Duration(i) * time.Minute)})
	}
	return out
}

// fx is how Describe renders an effect about the takeover whose lead alert is
// lead — the strings these tests hold the recorder open by.
func fx(kind, lead string) string { return kind + "(" + lineup.BurstID(lead) + ")" }

// rail sends one burst per lead, so the schedule holds one takeover per lead,
// and returns the card ids the rail now carries in order.
func rail(ctx context.Context, p *pump, leads ...string) []string {
	out := make([]string, 0, len(leads))
	for _, lead := range leads {
		p.send(ctx, lineup.Arrived{Arrivals: arrivals(lead)})
		out = append(out, lineup.BurstID(lead))
	}
	return out
}

// startPump runs a pump against a recorder and stops it when the test ends.
func startPump(t *testing.T, r *recorder, faults chan<- string) (*pump, context.Context) {
	t.Helper()
	onFault := func(f lineup.Effect, v any) {
		if faults != nil {
			faults <- lineup.Describe(f)
		}
	}
	p := newPump(lineup.New(lineup.Settings{Max: 10}, pumpNow), r.run, onFault)
	if p == nil {
		t.Fatal("newPump refused a well-formed pump")
	}
	ctx, cancel := context.WithCancel(context.Background())
	go p.loop(ctx)
	t.Cleanup(func() { cancel(); p.stop() })
	return p, ctx
}

// TestALongRunningEffectDoesNotDelayTheNextEvent is PL-1, and it is the finding
// this whole file exists to answer.
//
// The first draft of the loop ran each effect inline, which would have paid the
// 1.03 s card build ON THE PUMP and reintroduced the blocking class inside the
// approach chosen to avoid it. Here one effect is held open indefinitely and the
// schedule must carry on around it.
func TestALongRunningEffectDoesNotDelayTheNextEvent(t *testing.T) {
	r := newRecorder()
	stuck := make(chan struct{})
	r.hold[fx("build", "a1")] = stuck // never released until the test says so
	p, ctx := startPump(t, r, nil)

	rail(ctx, p, "a1")
	// The publish for that same step must land while the build is still stuck.
	seen := awaitLog(t, r, 1, "the publish, while the build is held open")
	if !containsPrefix(seen, "publish(") {
		t.Fatalf("the pump saw %v; the publish waited behind a held build", seen)
	}

	// And the next event is still processed.
	p.send(ctx, lineup.Tick{Now: pumpNow.Add(time.Minute)})
	after := awaitLog(t, r, 2, "a second publish from the tick")
	if countPrefix(after, "publish(") < 2 {
		t.Errorf("the pump saw %v; the tick waited behind a held build", after)
	}
	close(stuck)
}

// TestTheWordsCannotStartUntilTheCueReturns. Step guarantees the cue before the
// words as a LIST ORDER (DR-18); concurrent dispatch would throw that away, and
// the band would promise a callout after the read had started.
//
// IT IS ASSERTED BY BLOCKING, NOT BY READING BACK AN ORDER. An earlier version
// let both effects run and compared their positions in the log — which catches a
// broken grouping only when the race happens to go the wrong way. Under mB1 it
// passed at -count=1 and failed at -count=20: a pin that reports CAUGHT or
// SURVIVED depending on the scheduler is not a pin, and on the hazard path it is
// the worst kind, because it will be green on the run that matters.
//
// Holding the cue open makes it deterministic. The two effects are one serial
// run, so the words CANNOT be asked for while the cue is still going; if they
// ever are, the grouping is broken however the scheduler behaves that day.
func TestTheWordsCannotStartUntilTheCueReturns(t *testing.T) {
	r := newRecorder()
	held := make(chan struct{})
	r.hold[fx("cue", "a1")] = held
	p, ctx := startPump(t, r, nil)
	a1 := rail(ctx, p, "a1")[0]
	awaitLog(t, r, 2, "the build and the publish")
	p.send(ctx, lineup.Built{ID: a1, Script: lineup.Say("a severe thunderstorm warning is in effect")})

	// Long enough that a separately-dispatched read would have been asked for.
	// It is a bound on how long a wrong pump gets to look right, not a wait for
	// something expected to happen — the assertion is the ABSENCE below.
	time.Sleep(50 * time.Millisecond)
	if seen := r.log(); containsExact(seen, fx("speak", "a1")) {
		t.Fatalf("the pump saw %v; the words were asked for while the cue was still going", seen)
	}

	close(held)
	seen := awaitContains(t, r, fx("speak", "a1"))
	cue, speak := indexIn(seen, fx("cue", "a1")), indexIn(seen, fx("speak", "a1"))
	if cue < 0 || cue > speak {
		t.Errorf("the pump saw %v; the cue did not precede the words", seen)
	}
}

// TestEffectsAboutDifferentCardsRunAtOnce is the control for the test above.
//
// Without it, "the cue precedes the words" would also be satisfied by a pump
// that ran EVERYTHING in sequence — which would pass that assertion and fail the
// one that matters, because the next card's 1.03 s build would then wait behind
// the whole of this card's read.
func TestEffectsAboutDifferentCardsRunAtOnce(t *testing.T) {
	r := newRecorder()
	held := make(chan struct{})
	r.hold[fx("speak", "a1")] = held // the read, held open the way a real one lasts
	p, ctx := startPump(t, r, nil)

	// TWO BURSTS, because a burst is one card (MVS-D-77): "different cards" is a
	// rule about the rail, and two alerts of one burst are one card's content.
	a1 := rail(ctx, p, "a1", "b1")[0]
	awaitLog(t, r, 3, "the first build and both publishes")
	p.send(ctx, lineup.Built{ID: a1, Script: lineup.Say("words")})

	// a1 is speaking (and stuck). b1's build must go ahead regardless.
	seen := awaitContains(t, r, fx("build", "b1"))
	if !containsExact(seen, fx("build", "b1")) {
		t.Errorf("the pump saw %v; the next card's build waited behind a read", seen)
	}
	if containsExact(seen, fx("speak", "a1")) {
		t.Fatal("the held read completed; the test is not exercising what it claims")
	}
	close(held)
}

// TestAPanickingExecutorDoesNotKillThePump is DR-22.
//
// Under Approach C the pump is the single owner, so an unguarded panic would
// take both tracks and the bed with it — a strictly larger blast radius than
// today, where startTakeover guards a panic for ONE surface. The card is failed
// so the schedule re-plans around it, rather than the station going silent
// waiting for a completion that will never come.
func TestAPanickingExecutorDoesNotKillThePump(t *testing.T) {
	r := newRecorder()
	r.panics[fx("speak", "a1")] = true
	faults := make(chan string, 4)
	p, ctx := startPump(t, r, faults)

	a1 := rail(ctx, p, "a1", "b1")[0]
	awaitLog(t, r, 3, "the first build and both publishes")
	p.send(ctx, lineup.Built{ID: a1, Script: lineup.Say("words")})

	select {
	case got := <-faults:
		if got != fx("speak", "a1") {
			t.Errorf("the fault reported %q, want %s", got, fx("speak", "a1"))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the panic was never reported")
	}

	// THE SCHEDULE CONTINUES. The failed card leaves the air and the next one is
	// built — which is the whole claim, and it is about what happens AFTER the
	// panic, not merely that nothing crashed.
	seen := awaitContains(t, r, fx("release", "a1"))
	if !containsExact(seen, fx("build", "b1")) {
		t.Errorf("the pump saw %v; the schedule stopped at the panic", seen)
	}

	// And the pump still answers.
	before := len(seen)
	p.send(ctx, lineup.Tick{Now: pumpNow.Add(time.Minute)})
	awaitLog(t, r, before+1, "a publish from a tick after the panic")
}

// TestAPanicInAnEffectThatNamesNoCardFailsNothing. A publish that comes apart is
// a broken executor and must be recorded — but there is no card to fail, and
// inventing one would take a read off the air for a fault that never touched it.
func TestAPanicInAnEffectThatNamesNoCardFailsNothing(t *testing.T) {
	r := newRecorder()
	r.panics["publish(rail=["+lineup.BurstID("a1")+"] main=[])"] = true
	faults := make(chan string, 4)
	p, ctx := startPump(t, r, faults)

	rail(ctx, p, "a1")
	select {
	case got := <-faults:
		if !strings.HasPrefix(got, "publish(") {
			t.Errorf("the fault reported %q, want the publish", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the panic was never reported")
	}
	// The build for the same step still ran, and the schedule is untouched.
	seen := awaitLog(t, r, 2, "the build alongside the failed publish")
	if !containsExact(seen, fx("build", "a1")) {
		t.Errorf("the pump saw %v; a failed publish took the build with it", seen)
	}
	p.send(ctx, lineup.Built{ID: lineup.BurstID("a1"), Script: lineup.Say("words")})
	after := awaitLog(t, r, 4, "the cue and the words after a failed publish")
	if !containsExact(after, fx("speak", "a1")) {
		t.Errorf("the pump saw %v; a card was failed by a fault that never touched it", after)
	}
}

// TestAContainedPanicEmitsNothingForAnEffectThatNamesNoCard pins the pump's own
// contract, directly.
//
// THE END-TO-END TEST ABOVE CANNOT SEE THIS, and mB5 is the proof: remove the
// guard and the pump emits Failed{ID: ""}, which the Director then ignores
// because find("") holds nothing — so the schedule is unharmed either way and
// the mutant SURVIVED. The rule is enforced twice, in two layers, and only the
// far one was observable.
//
// The guard is kept rather than deleted, and the reason is not symmetry: an
// event naming no card is garbage on the wire. It would appear in the DR-23
// timeline as a failure of nothing, and it depends for its harmlessness on a
// detail of the Director's lookup that nothing here controls. This test is what
// stops that dependency being invisible.
func TestAContainedPanicEmitsNothingForAnEffectThatNamesNoCard(t *testing.T) {
	r := newRecorder()
	r.panics["publish(rail=[] main=[])"] = true
	seen := 0
	p := newPump(lineup.New(lineup.Settings{Max: 10}, pumpNow), r.run,
		func(lineup.Effect, any) { seen++ })

	out := p.contained(context.Background(), lineup.Publish{})
	if len(out) != 0 {
		t.Errorf("a card-less panic produced %v, want no events at all", out)
	}
	if seen != 1 {
		t.Errorf("the fault was reported %d times, want once — it is the only sign the executor broke", seen)
	}

	// The control: the same containment for an effect that DOES name a card
	// produces exactly one failure, so the emptiness above is the rule and not a
	// recover that swallows everything.
	r.panics["speak(a1)"] = true
	out = p.contained(context.Background(), lineup.Speak{ID: "a1", Script: lineup.Say("words")})
	if len(out) != 1 {
		t.Fatalf("a panic on a card produced %v, want one failure", out)
	}
	failed, ok := out[0].(lineup.Failed)
	if !ok || failed.ID != "a1" {
		t.Errorf("the pump emitted %#v, want a failure for a1", out[0])
	}
	if !strings.Contains(failed.Reason, "speak(a1)") {
		t.Errorf("the reason is %q; it does not say which effect came apart", failed.Reason)
	}
}

// TestRunsOfGroupsOnlyWhatMustStayInOrder — the grouping rule as a table, so a
// change to it is deliberate rather than emergent.
//
// Grouping keeps ONE CARD's effects in order. The band's own ordering is not
// here: it is the lane's, because the effects that must not overtake each other
// on the band routinely fall in different steps and no grouping can see that.
func TestRunsOfGroupsOnlyWhatMustStayInOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []lineup.Effect
		want [][]string
	}{
		{"a cue and its words are one run", []lineup.Effect{
			lineup.CueTicker{ID: "a1"}, lineup.Speak{ID: "a1"},
		}, [][]string{{"cue(a1)", "speak(a1)"}}},
		{"different cards are separate runs", []lineup.Effect{
			lineup.CueTicker{ID: "a1"}, lineup.Speak{ID: "a1"}, lineup.BuildCard{ID: "b1"},
		}, [][]string{{"cue(a1)", "speak(a1)"}, {"build(b1)"}}},
		// CONNECTED, NOT ADJACENT. An effect naming no card must not split a
		// card's cue from its words: they would land on separate workers and the
		// band could promise a callout after the read had started (DR-18). Duck
		// and Restore name no card and arrive with T2.3, so this is the shape
		// that is about to become ordinary.
		{"an effect naming no card never splits a card's run", []lineup.Effect{
			lineup.CueTicker{ID: "a1"}, lineup.Publish{}, lineup.Speak{ID: "a1"},
		}, [][]string{{"cue(a1)", "speak(a1)"}, {"publish(rail=[] main=[])"}}},
		{"effects naming no card each run alone", []lineup.Effect{
			lineup.Publish{}, lineup.Duck{}, lineup.Restore{},
		}, [][]string{{"publish(rail=[] main=[])"}, {"duck()"}, {"restore()"}}},
		// They are separate RUNS, but the duck and the restore ride the lane
		// with the read they bracket, so their ORDER is the lane's — see
		// TestTheDuckIsNotOvertakenByTheReadItBrackets.
		{"a release and the next card's cue are separate runs; the LANE orders them", []lineup.Effect{
			lineup.ReleaseTicker{ID: "a1"}, lineup.CueTicker{ID: "b1"}, lineup.Speak{ID: "b1"}, lineup.BuildCard{ID: "c1"},
		}, [][]string{{"release(a1)"}, {"cue(b1)", "speak(b1)"}, {"build(c1)"}}},
		{"nothing at all", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := [][]string{}
			for _, g := range runsOf(tc.in) {
				var run []string
				for _, f := range g {
					run = append(run, lineup.Describe(f))
				}
				got = append(got, run)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("runsOf made %v, want %v", got, tc.want)
			}
			for i := range got {
				if strings.Join(got[i], "|") != strings.Join(tc.want[i], "|") {
					t.Errorf("run %d is %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestOnlySharedOutputWorkRidesTheLane — the routing rule, so a run that must keep its
// place in the band's order cannot quietly stop doing so, and a build cannot
// quietly start waiting behind a read.
func TestOnlySharedOutputWorkRidesTheLane(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []lineup.Effect
		want bool
	}{
		{"a cue", []lineup.Effect{lineup.CueTicker{ID: "a1"}}, true},
		{"a release", []lineup.Effect{lineup.ReleaseTicker{ID: "a1"}}, true},
		{"a cue and its words", []lineup.Effect{lineup.CueTicker{ID: "a1"}, lineup.Speak{ID: "a1"}}, true},
		{"a build", []lineup.Effect{lineup.BuildCard{ID: "a1"}}, false},
		{"a read with no cue beside it", []lineup.Effect{lineup.Speak{ID: "a1"}}, true},
		{"a publish", []lineup.Effect{lineup.Publish{}}, false},
		{"a duck", []lineup.Effect{lineup.Duck{}}, true},
		{"a restore", []lineup.Effect{lineup.Restore{}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sharesAnOutput(tc.in); got != tc.want {
				t.Errorf("sharesAnOutput(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestStopDrainsEveryDispatchedEffect. A job in flight ends with the context and
// is waited for, never abandoned (R5-B-07) — without it a headless run's
// temp-directory cleanup races an executor still writing.
func TestStopDrainsEveryDispatchedEffect(t *testing.T) {
	r := newRecorder()
	held := make(chan struct{})
	r.hold[fx("build", "a1")] = held
	r.stubborn[fx("build", "a1")] = true // in flight, and cancellation cannot recall it
	onFault := func(lineup.Effect, any) {}
	p := newPump(lineup.New(lineup.Settings{Max: 10}, pumpNow), r.run, onFault)
	ctx, cancel := context.WithCancel(context.Background())
	go p.loop(ctx)
	rail(ctx, p, "a1")
	awaitLog(t, r, 1, "the publish")

	stopped := make(chan struct{})
	go func() { cancel(); p.stop(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("stop returned while an effect was still in flight")
	case <-time.After(50 * time.Millisecond):
	}
	close(held)
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("stop never returned after the effect finished")
	}
}

// TestAPumpWithNothingToRunIsRefused. A nil runner or a nil fault sink would
// panic on the first effect, on a goroutine, where the panic is the failure
// rather than the diagnosis.
func TestAPumpWithNothingToRunIsRefused(t *testing.T) {
	d := lineup.New(lineup.Settings{Max: 10}, pumpNow)
	if newPump(d, nil, func(lineup.Effect, any) {}) != nil {
		t.Error("a pump with no runner was built")
	}
	if newPump(d, func(context.Context, lineup.Effect) []lineup.Event { return nil }, nil) != nil {
		t.Error("a pump with nowhere to report a fault was built")
	}
}

func containsExact(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}
	return false
}

func containsPrefix(got []string, prefix string) bool {
	for _, g := range got {
		if strings.HasPrefix(g, prefix) {
			return true
		}
	}
	return false
}

func countPrefix(got []string, prefix string) int {
	n := 0
	for _, g := range got {
		if strings.HasPrefix(g, prefix) {
			n++
		}
	}
	return n
}

func indexIn(got []string, want string) int {
	for i, g := range got {
		if g == want {
			return i
		}
	}
	return -1
}

// TestTheBandIsReleasedBeforeTheNextCardIsCued — F-D2, and the reason the
// grouping is by RESOURCE rather than by card.
//
// `leave` emits [release(N), cue(N+1), speak(N+1), …]: the release belongs to
// the card going off the air and the cue to the card coming on, so a grouping
// keyed on the card alone puts them on separate workers. They race, and when
// the release lands second it CLEARS THE CALLOUT THE CUE JUST PUT UP — the next
// card reads with the band already back on rotation, which is exactly the stale
// -callout failure DR-24 exists to prevent, wearing the opposite sign.
//
// ASSERTED BY BLOCKING, like the cue/words pin (mB1): holding the release open
// makes the claim deterministic whatever the scheduler does that day.
func TestTheBandIsReleasedBeforeTheNextCardIsCued(t *testing.T) {
	r := newRecorder()
	held := make(chan struct{})
	r.hold[fx("release", "a1")] = held
	p, ctx := startPump(t, r, nil)

	// TWO BURSTS: "the next card" is the next TAKEOVER (MVS-D-77).
	ids := rail(ctx, p, "a1", "b1")
	a1, b1 := ids[0], ids[1]
	awaitContains(t, r, fx("build", "a1"))
	p.send(ctx, lineup.Built{ID: a1, Script: lineup.Say("the first alert")})
	awaitContains(t, r, fx("build", "b1"))
	p.send(ctx, lineup.Built{ID: b1, Script: lineup.Say("the second alert")})
	awaitContains(t, r, fx("speak", "a1"))

	// a1 leaves the air: the band is released, then b1 is cued and read.
	p.send(ctx, lineup.Finished{ID: a1})

	// Long enough that a separately-dispatched cue would have been asked for.
	// The assertion is the ABSENCE: b1's callout must not reach the band while
	// the release of a1's is still in flight.
	time.Sleep(50 * time.Millisecond)
	if seen := r.log(); containsExact(seen, fx("cue", "b1")) {
		t.Fatalf("the pump saw %v; the next card was cued while the band was still being released", seen)
	}
	close(held)
	seen := awaitContains(t, r, fx("cue", "b1"))
	if rel, cue := indexIn(seen, fx("release", "a1")), indexIn(seen, fx("cue", "b1")); rel < 0 || rel > cue {
		t.Errorf("the pump saw %v; the release did not precede the next cue", seen)
	}
}

// TestTheBandKeepsItsOrderAcrossSteps — the half F-D2's first fix did not close,
// found by an adversarial review and reproduced here before it was believed.
//
// Grouping orders effects WITHIN one step. On the ordinary slow path the two
// band effects fall in DIFFERENT steps: card a1 finishes while b1's 1.03 s
// build is still out, so `leave` emits only [release(a1), publish] — there is
// no cue to chain to, because b1 has no words yet. b1's cue arrives a step
// later, on a fresh goroutine, and can reach the band BEFORE the release does.
// When the release then lands it clears b1's callout, and b1 reads with the
// band already back on rotation: the same DR-24 failure, one step further out.
func TestTheBandKeepsItsOrderAcrossSteps(t *testing.T) {
	r := newRecorder()
	held := make(chan struct{})
	r.hold[fx("release", "a1")] = held
	p, ctx := startPump(t, r, nil)

	ids := rail(ctx, p, "a1", "b1")
	a1, b1 := ids[0], ids[1]
	awaitContains(t, r, fx("build", "a1"))
	p.send(ctx, lineup.Built{ID: a1, Script: lineup.Say("the first alert")})
	awaitContains(t, r, fx("speak", "a1"))

	// a1 finishes while b1 is still being built: the release goes out alone.
	p.send(ctx, lineup.Finished{ID: a1})
	// b1's words arrive a step later and it takes the air.
	p.send(ctx, lineup.Built{ID: b1, Script: lineup.Say("the second alert")})

	// Long enough that a cue dispatched on its own goroutine would have landed.
	time.Sleep(50 * time.Millisecond)
	if seen := r.log(); containsExact(seen, fx("cue", "b1")) {
		t.Fatalf("the pump saw %v; b1 was cued while a1's release was still in flight — the release will clear that callout", seen)
	}
	close(held)
	seen := awaitContains(t, r, fx("cue", "b1"))
	if rel, cue := indexIn(seen, fx("release", "a1")), indexIn(seen, fx("cue", "b1")); rel < 0 || rel > cue {
		t.Errorf("the pump saw %v; the band was written out of order across steps", seen)
	}
}

// TestStoppingTwiceIsSafe. Shutdown is exactly where a second stop arrives — a
// defer beside an explicit call — and closing the lane twice would panic in the
// one place a panic is least welcome.
func TestStoppingTwiceIsSafe(t *testing.T) {
	r := newRecorder()
	onFault := func(lineup.Effect, any) {}
	p := newPump(lineup.New(lineup.Settings{Max: 10}, pumpNow), r.run, onFault)
	ctx, cancel := context.WithCancel(context.Background())
	go p.loop(ctx)
	rail(ctx, p, "a1")
	awaitContains(t, r, fx("build", "a1"))
	cancel()
	p.stop()
	p.stop() // the second one must be a no-op, not a panic
}

// TestTheDuckIsNotOvertakenByTheReadItBrackets. The bed is one broadcast: the
// duck must reach it before the words are spoken over it, and the restore must
// not overtake a correspondent still speaking.
//
// Neither names a card, so grouping cannot order them against the read — they
// were dispatched to their own goroutines, concurrent with it, and a test in
// this file ASSERTED that concurrency as correct. A duck landing after the read
// has started is the duck-lift bug in a new costume (RD-2), and MVS-D-67 makes
// these effects binding before the rail goes live.
//
// Asserted by blocking, so the claim does not depend on the scheduler.
func TestTheDuckIsNotOvertakenByTheReadItBrackets(t *testing.T) {
	r := newRecorder()
	held := make(chan struct{})
	r.hold["duck()"] = held
	p, ctx := startPump(t, r, nil)

	p.dispatch(ctx, []lineup.Effect{
		lineup.Duck{},
		lineup.CueTicker{ID: "a1", Headline: "h"},
		lineup.Speak{ID: "a1", Script: lineup.Say("words")},
		lineup.Restore{},
	})

	// Long enough that a separately-dispatched read would have been asked for.
	time.Sleep(50 * time.Millisecond)
	if seen := r.log(); containsExact(seen, "speak(a1)") || containsExact(seen, "restore()") {
		t.Fatalf("the pump saw %v; the read began over a bed that was still being ducked", seen)
	}
	close(held)
	seen := awaitContains(t, r, "restore()")
	duck, speak, restore := indexIn(seen, "duck()"), indexIn(seen, "speak(a1)"), indexIn(seen, "restore()")
	if duck < 0 || duck > speak || speak > restore {
		t.Errorf("the pump saw %v; want the duck, then the words, then the restore", seen)
	}
}

// TestTheLaneRetiresWithTheLoop. Cancelling the context retired the pump and
// left the lane goroutine running until someone called stop; before the lane
// existed, cancelling retired every pump goroutine.
//
// ASSERTED ON THE CHANNEL, NOT ON A STACK DUMP. A receive from a closed channel
// is always ready and reports it closed, while a receive from an open empty one
// blocks — so the default arm tells them apart with no timing in the assertion
// at all. The leak was proved once with a throwaway probe and the probe was
// then deleted, which left the rule with no pin: its mutant survived.
func TestTheLaneRetiresWithTheLoop(t *testing.T) {
	r := newRecorder()
	p := newPump(lineup.New(lineup.Settings{Max: 10}, pumpNow), r.run, func(lineup.Effect, any) {})
	ctx, cancel := context.WithCancel(context.Background())
	go p.loop(ctx)
	// A build touches no shared output, so nothing is left queued on the lane.
	rail(ctx, p, "a1")
	awaitContains(t, r, fx("build", "a1"))

	cancel()
	<-p.done
	select {
	case _, open := <-p.lane:
		if open {
			t.Error("the lane handed out a run after the loop had returned")
		}
	default:
		t.Error("the lane is still open after the loop returned; cancelling leaks its goroutine")
	}
	p.stop()
}
