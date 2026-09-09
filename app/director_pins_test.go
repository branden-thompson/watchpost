package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// PHASE 0 OF THE STATION DIRECTOR BUILD — the regression net, written BEFORE
// anything moves (03-architecture-design/director-build-plan.md).
//
// These pin rules that are about to be absorbed by the Director. The reason
// they exist at all is on the record: the last time one of these rules moved,
// the exported Tune lifted the alert duck and the unexported tune did not, the
// watchlist advance called the wrong one, and a listener heard the next
// location's report come up at full volume over a breaking alert still being
// read. "A rule that lives in the case of an identifier is a rule waiting to be
// missed" (app/radio.go).
//
// Each pin is mutation-validated in T0.4: deleting the rule it guards must make
// it fail. A pin that has never failed protects nothing.

// offlineDeck is a radioDeck that can run tune() END TO END with no network: an
// httptest server that 404s everything, the NWS provider and BOTH relay
// directories pointed at it, and a fake audio output.
//
// The alternative was to pin only advanceQueue's guards — the negative cases —
// and that has the defect radio_stop_test.go names at its own positive half: a
// guarded advance and a broken one look identical from outside. The rule being
// pinned here is about to move into the Director, so the pin has to be able to
// see the advance actually happen.
func offlineDeck(t *testing.T) (*radioDeck, *heldOutput) {
	t.Helper()
	// A HELPER CALLED offlineDeck MUST BE OFFLINE. Without this the deck picks
	// its voice on the RUNNING platform, and on Linux that means rawVoice finds
	// no Piper voice and installs one — a real 63 MB download inside a unit test,
	// whose progress callback then dereferences this deck's nil program. That is
	// what failed the first four Linux CI runs of this release; on a Mac the same
	// call returns a SayVoice immediately and nothing was ever visible.
	//
	// These five tests are about the Director's advance and the duck. The host's
	// voice catalogue is not the subject, so it is pinned rather than inherited.
	asPlatform(t, "darwin")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "offline", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	client, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: 1000, CacheDir: t.TempDir()})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	res, err := stream.NewResolver(stream.NewDirectory(client, srv.URL, srv.URL))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	out := &heldOutput{}
	eng, err := player.New(out, "test", func(player.Status) {})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	t.Cleanup(eng.Halt)
	return &radioDeck{
		nws: nws.New(client, srv.URL), resolver: res, engine: eng,
		limiter: synth.NewLimiter(2, synth.ReservedSlots), voiceDir: t.TempDir(),
	}, out
}

func pinRef(label string, lat, lon float64) snapshot.LocationRef {
	return snapshot.LocationRef{Label: label, Lat: lat, Lon: lon, TZ: "America/Los_Angeles"}
}

// T0.1 — THE DECK REPORTS THE FACTS THE DIRECTOR DECIDES ON (T3.2b).
//
// T0.1a and T0.1b pinned armDwell and advanceQueue: when the dwell armed, and
// that an advance respected a stop. Both were DECISIONS and both moved into the
// Director, where they are pure functions of the bed, the settings and the clock
// — pinned there by TestTheBedHoldsWhenItShould, TestALiveRelayAdvancesWhenIts
// DwellElapses and TestAStoppedProgrammeDoesNotAdvanceTheBed.
//
// What is left on this side is the half only the deck can do: OBSERVING. A stop
// is a stop, and the rotation is what the listener set. If the deck reports these
// wrongly the Director decides correctly about a fiction, so this pins the FACTS
// rather than the rules.
func TestT01TheDeckReportsTheFactsTheDirectorDecidesOn(t *testing.T) {
	d, _ := offlineDeck(t)
	var got []lineup.Event
	d.emit = func(ev lineup.Event) { got = append(got, ev) }
	a := pinRef("A", 33.19, -117.37)

	// A STOP IS REPORTED, or the Director moves the bed on five minutes later
	// and starts the station up again by itself.
	d.mode = "synth"
	d.Stop()
	if len(got) != 1 {
		t.Fatalf("a stop told the Director %d things, want one: %v", len(got), got)
	}
	if p, ok := got[0].(lineup.Powered); !ok || p.To != lineup.Stopped {
		t.Errorf("a stop reported %#v, want Powered{Stopped}", got[0])
	}

	// THE ROTATION IS REPORTED ON EVERY CHANGE, playing or not — a setting the
	// Director never heard is a setting that does not apply. Repeat that is not
	// Watchlist reaches it as a ZERO dwell rather than as the deck's own enum.
	got = nil
	d.SetRepeat(tty.RepeatWatchlist, []snapshot.LocationRef{a})
	if len(got) != 1 {
		t.Fatalf("a repeat change told the Director %d things, want one: %v", len(got), got)
	}
	if pr, ok := got[0].(lineup.Programme); !ok || pr.Dwell != liveDwell || len(pr.Watchlist) != 1 {
		t.Errorf("Watchlist reported as %#v, want the rotation with liveDwell", got[0])
	}
	got = nil
	d.SetRepeat(tty.RepeatOff, []snapshot.LocationRef{a})
	if len(got) != 1 {
		t.Fatalf("repeat Off told the Director %d things, want one: %v", len(got), got)
	}
	if pr, ok := got[0].(lineup.Programme); !ok || pr.Dwell != 0 {
		t.Errorf("repeat Off reported %#v, want a zero dwell — that is how 'not Watchlist' travels", got[0])
	}
}

// T0.2 — AN AUTOMATIC ADVANCE DOES NOT TAKE AN ALERT OFF THE AIR.
//
// This is the regression, expressed as a test rather than a comment. The
// exported Tune once lifted the alert duck and the unexported tune did not, and
// the Watchlist advance called the wrong one: the next location's report came up
// at full volume over a breaking alert that was still reading. Both spellings
// leave the duck alone today, and the duck has exactly one owner — the director.
// That ownership is what moves into the Lineup.
func TestT02AnAutomaticAdvanceDoesNotTakeAnAlertOffTheAir(t *testing.T) {
	d, out := offlineDeck(t)
	a, b := pinRef("A", 33.19, -117.37), pinRef("B", 32.71, -117.16)
	d.repeat, d.queue, d.mode = tty.RepeatWatchlist, []snapshot.LocationRef{a, b}, "synth"

	d.engine.Suppress() // a takeover is reading an alert
	// THE PATH THE DIRECTOR NOW TAKES (T3.2b). The advance used to be
	// advanceQueue's; it is the Director's decision now and reaches the deck
	// through the executor's tune seam, which calls exactly this. The rule is
	// unchanged and so is what it is asserted against.
	d.tune(b)
	if snapshot.Key(d.ref) != snapshot.Key(b) {
		t.Fatalf("the advance did not happen (ref=%q) — this proves nothing about the duck", d.ref.Label)
	}

	// Halt the stream the advance just started before probing. It carries no
	// audio here (the offline provider 404s every product), so leaving it
	// running only races the probe for `latest()` — and a probe that measured
	// the deck's own silent source would report "held" for the wrong reason.
	// Halt deliberately does NOT lift the suppression, which is the property
	// under test: whether an alert is on the air is not the stop button's to
	// answer (app/radio.go, Stop).
	d.engine.Halt()

	// Whatever the listener hears next must still give way to the alert.
	// The probe's player is identified by POSITION, not by being last: the
	// advance's own source opens its player asynchronously, so "latest" can
	// hand back the deck's silent stream instead — which would report "held"
	// for entirely the wrong reason.
	before := out.count()
	d.engine.StartSource("probe", 44100, func(context.Context) io.Reader { return endlessSilence{} })
	p := awaitPlayerAfter(t, out, before)
	time.Sleep(120 * time.Millisecond) // two watch ticks
	if n := p.plays.Load(); n != 0 {
		t.Errorf("an automatic advance must not lift the alert duck; the stream played %d time(s) over it", n)
	}

	// AND THE POSITIVE HALF — without it this passes on an engine that never
	// plays anything, and a held stream and a broken one are indistinguishable.
	d.engine.Restore()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && p.plays.Load() == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if p.plays.Load() == 0 {
		t.Error("…and once the alert is off the air the held stream must play")
	}
}

// awaitPlayerAfter waits for the player opened AFTER n already existed, so a
// test can name the stream it started rather than whichever one happens to be
// last.
func awaitPlayerAfter(t *testing.T, out *heldOutput, n int) *heldPlayer {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		out.mu.Lock()
		var p *heldPlayer
		if len(out.players) > n {
			p = out.players[n]
		}
		out.mu.Unlock()
		if p != nil {
			return p
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the probe stream never opened a player — the instrument is measuring nothing")
	return nil
}

// cueLog records, in order, the two things whose ORDERING is the contract: the
// marquee cue sent to the dashboard, and the words handed to the voice.
//
// One log rather than two counters, because "both happened" is not the property
// under test — "the band was told before the words were spoken" is, and only an
// ordered record can show it.
type cueLog struct {
	mu    sync.Mutex
	seq   []string
	onCue func() // fired after a cue is recorded; lets a test interrupt mid-sequence
}

func (l *cueLog) add(s string) {
	l.mu.Lock()
	l.seq = append(l.seq, s)
	hook := l.onCue
	l.mu.Unlock()
	if strings.HasPrefix(s, "cue") && hook != nil {
		hook()
	}
}

func (l *cueLog) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.seq...)
}

// cueVoice is a narrationVoice that records when each line is SPOKEN — and
// reports a short duration, so the sequence's holds do not wait out a real read.
//
// IT RECORDS `play`, NOT `render`, AND THE DIFFERENCE IS THE WHOLE CONTRACT.
// DR-18 is that the band is told before the words are HEARD. It logged the
// render instead, which was a faithful stand-in only while rendering and playing
// were adjacent; the moment a line was rendered ahead of time — during the tone,
// or during the line before it — the render moved in front of the cue while not
// one word reached the listener any earlier. The pin failed on a silent
// operation, which is a pin measuring the wrong event rather than a defect.
// Renders are still logged, so a reader can see them, but they are not "say".
type cueVoice struct{ log *cueLog }

func (v *cueVoice) duck()                         {}
func (v *cueVoice) tone(cast.Class) time.Duration { return 0 }
func (v *cueVoice) render(_ context.Context, _ cast.Role, text string) (clip, bool) {
	v.log.add("render" + tag(text))
	return clip{text: text, dur: time.Millisecond}, true
}

// tag names WHICH event a log entry belongs to, so the ordering can be asserted
// per event rather than by position.
//
// Position was not enough, and a mutant proved it: the first version exempted
// the burst head with an index test, the head is only index 0, and so the first
// event's words slipped through — moving the cue after the words was SURVIVED
// by the pin (m49). Identity cannot drift the way an index can.
func tag(s string) string {
	switch {
	case strings.Contains(s, "Norfolk"):
		return ":norfolk"
	case strings.Contains(s, "Raleigh"):
		return ":raleigh"
	}
	return ":structural" // the burst head and the closing tail belong to no event
}
func (v *cueVoice) play(c clip) { v.log.add("say" + tag(c.text)) }

// fault is inert in this double: the seam exists so a read can report a
// tone with no words (FR-9.2); nothing here reads it.
func (v *cueVoice) fault(string) {}

func (v *cueVoice) pause()   {}
func (v *cueVoice) resume()  {}
func (v *cueVoice) stop()    {}
func (v *cueVoice) discard() {}
func (v *cueVoice) restore() {}

// cueDeck is a ticker deck wired to the log: no network, no audio, no disk
// beyond the seen store's temp dir.
func cueDeck(t *testing.T, log *cueLog) *tickerDeck {
	t.Helper()
	return cueDeckWith(t, log, &cueVoice{log: log})
}

// cueDeckWith is cueDeck over a chosen voice, so a test can make a render fail
// or end the sequence at a chosen moment.
func cueDeckWith(t *testing.T, log *cueLog, v narrationVoice) *tickerDeck {
	t.Helper()
	band := func(m tea.Msg) {
		switch msg := m.(type) {
		case tty.TickerBreakingMsg:
			log.add("cue" + tag(msg.Item.Head))
		case tty.TickerBreakingDoneMsg:
			log.add("release")
		}
	}
	nar := testDirector(v, band)
	// Holds are instant: this pins ORDER, not pacing, and a real hold would
	// spend the burst's full air time proving nothing about the ordering.
	//
	// BUT THE HOLDS ARE LOGGED, because without them a render moved from during
	// a line to after it is INVISIBLE. Both arrangements log the render in the
	// same place relative to the words — the difference is whether the hold for
	// those words has already been spent, and a log with no holds in it cannot
	// see that. A pin for the pre-build was written without this and passed
	// against the defect it was named for.
	nar.sleep = func(ctx context.Context, _ time.Duration) bool {
		log.add("wait")
		return ctx.Err() == nil
	}
	return &tickerDeck{
		send: band,
		// THE SAME EFFECTOR THE ARBITER HAS. The takeover cues through it, so a
		// deck holding a second one would observe no cues at all and this pin
		// would pass while proving nothing.
		mc:    nar.mc,
		muted: &atomic.Bool{},
		voice: nar,
		seen:  loadSeen(t.TempDir(), time.Hour),
	}
}

// twoBreaking is a two-event burst, so the PER-EVENT ordering is observable.
// One event could not distinguish "the cue leads every event" from "a cue
// happened once somewhere".
//
// IT NAMES A SOURCE, and that is load-bearing rather than decoration. `burstHead`
// returns "" for a burst whose events name no agency, so without it these pins
// ran a burst with NO HEAD — and the head is a boundary of its own, where the
// first alert's render sat in the head's pause while every pin stayed green. A
// fixture that skips a branch makes every pin over it vacuous on that branch.
func twoBreaking() []globalfeed.Event {
	// AND IT IS LIVE (red team 2026-09-05, I-8). A fixed past date meant both
	// alerts had expired, and eventsFor now declines to compose a card whose
	// alert is no longer active — so every pin over this fixture would have
	// run against a burst that never reached the reader. The fixture-validity
	// guards in these tests are what caught it, which is what they are for.
	at := time.Now()
	return []globalfeed.Event{
		{ID: "pin-1", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "Norfolk, VA",
			Severity: globalfeed.SevRed, At: at, Until: at.Add(time.Hour)},
		{ID: "pin-2", Source: "NWS", Class: globalfeed.ClassSevereWx, Type: "Severe Thunderstorm Warning", Location: "Raleigh, NC",
			Severity: globalfeed.SevOrange, At: at, Until: at.Add(time.Hour)},
	}
}

// T0.3 — THE BAND IS TOLD BEFORE THE WORDS ARE SPOKEN, FOR EVERY EVENT.
//
// This is today's "3 … 2 … 1" cue, which the charter describes as a deliberate
// lead and which the code arrives at by accident: readBreaking sends the
// marquee message, then calls s.line, which RENDERS before it plays. The lead
// is real, nobody specified it, and DR-18 turns it into a contract — so its
// current shape is recorded here first.
func TestT03TheCuePrecedesTheWordsForEveryEvent(t *testing.T) {
	log := &cueLog{}
	d := cueDeck(t, log)

	newStation(t, d).takeover(context.Background(), twoBreaking())

	seq := log.all()
	// THE FIXTURE IS ASSERTED VALID BEFORE THE BEHAVIOUR IS. A sequence that
	// never cued and never spoke would satisfy every ordering claim below by
	// vacuity — which is exactly how five fixtures passed while proving nothing
	// this release.
	for _, ev := range []string{":norfolk", ":raleigh"} {
		cue, say := indexOf(seq, "cue"+ev), indexOf(seq, "say"+ev)
		if cue < 0 {
			t.Fatalf("no cue for %s — the burst must cue the band once per event: %v", ev, seq)
		}
		if say < 0 {
			t.Fatalf("%s was never spoken — the instrument is measuring nothing: %v", ev, seq)
		}
		// THE PROPERTY, PER EVENT: the band was told about THIS event before
		// THIS event's words were spoken. Asserted by identity, not position.
		if cue > say {
			t.Errorf("%s was spoken before the band was cued for it (cue at %d, words at %d): %v", ev, cue, say, seq)
		}
	}
	if last := seq[len(seq)-1]; last != "release" {
		t.Errorf("a completed burst releases the band back to rotation; the sequence ended with %q: %v", last, seq)
	}
}

// T0.3b / DR-24 — A TAKEOVER CUT SHORT STILL RELEASES THE BAND.
//
// INVERTED AT T3.5, AND THE INVERSION IS THE PROOF. This test used to assert
// the defect: TickerBreakingDoneMsg was sent on ONE path — the last line of the
// takeover closure — with four early returns above it that sent nothing, while
// the audio side was released unconditionally by the arbiter. That asymmetry is
// why it went unnoticed for so long: the sound came back, so the station seemed
// fine, and only the band sat frozen on an alert nobody was reading.
//
// The release is a `defer` now rather than a fifth call site, because DR-24 is
// explicit that this must be a PROPERTY and not a discipline — and "remember to
// release before every return" is the discipline that had already failed four
// times in this one function.
//
// It was written as a characterisation test that names its own replacement, and
// that is what made the handover free: the day the behaviour changed, the test
// failed with instructions rather than a puzzle.
func TestT03bATakeoverCutShortStillReleasesTheBand(t *testing.T) {
	log := &cueLog{}
	ctx, cancel := context.WithCancel(context.Background())
	// CUT IT SHORT MID-SEQUENCE, not before it starts. A pre-cancelled context
	// never enters the takeover at all, so "no release" would be true of a
	// perfect implementation too — the assertion would pass while proving
	// nothing, which is this release's most repeated mistake.
	log.onCue = func() { cancel() }
	d := cueDeck(t, log)

	newStation(t, d).takeover(ctx, twoBreaking())

	seq := log.all()
	// THE FIXTURE IS VALID ONLY IF THE TAKEOVER ACTUALLY GOT ON THE AIR and was
	// then interrupted. Both halves are asserted before the finding is.
	if count(seq, "cue") == 0 {
		t.Fatalf("the takeover never cued the band — it was cut short before it began, so this proves nothing: %v", seq)
	}
	if ctx.Err() == nil {
		t.Fatal("the sequence was never actually interrupted — the instrument is measuring nothing")
	}
	if count(seq, "release") != 1 {
		t.Fatalf("a takeover cut short must release the band exactly once, or the ticker keeps a "+
			"callout for a read that has stopped: %v", seq)
	}
	if last := seq[len(seq)-1]; last != "release" {
		t.Errorf("the release is the LAST thing the takeover does; the sequence ended with %q: %v", last, seq)
	}
}

func count(seq []string, prefix string) int {
	n := 0
	for _, s := range seq {
		if strings.HasPrefix(s, prefix) {
			n++
		}
	}
	return n
}

func indexOf(seq []string, want string) int {
	for i, s := range seq {
		if s == want {
			return i
		}
	}
	return -1
}

// count is how many players the fake output has handed out so far.
func (o *heldOutput) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.players)
}

// NO RENDER SITS BETWEEN A CUE AND ITS WORDS (T3.3, the pre-build).
//
// A render costs about a second and it used to run inside the pause before the
// line it belongs to: cue the band, render, speak. Every gap in a burst carried
// it, so the listener heard tone → a second of nothing → header, and about two
// and a half seconds between alerts where the ruling says one. Heard on a real
// alert at UAT 2026-09-03: "the uniform 2-3s pause in between every sentence
// feels like something is broken".
//
// THE FIX IS ORDER, NOT DURATION, SO THE PIN IS ABOUT ORDER. Each line is
// rendered while the PREVIOUS sound is still playing, which puts its render
// before its own cue. Pinning the elapsed time instead would make this a
// stopwatch test that fails on a slow machine and passes on a fast one while
// proving nothing about the arrangement.
func TestNoRenderSitsBetweenACueAndItsWords(t *testing.T) {
	log := &cueLog{}
	d := cueDeck(t, log)

	newStation(t, d).takeover(context.Background(), twoBreaking())

	seq := log.all()
	for _, ev := range []string{":norfolk", ":raleigh"} {
		render, cue, say := indexOf(seq, "render"+ev), indexOf(seq, "cue"+ev), indexOf(seq, "say"+ev)
		// THE FIXTURE FIRST: all three must exist, or the ordering below holds
		// by vacuity.
		if render < 0 || cue < 0 || say < 0 {
			t.Fatalf("%s is missing a step (render %d, cue %d, say %d) — the instrument is measuring nothing: %v",
				ev, render, cue, say, seq)
		}
		if render > cue {
			t.Errorf("%s was rendered AFTER its cue (render at %d, cue at %d), so its render sits in the pause "+
				"before it and the listener waits it out: %v", ev, render, cue, seq)
		}
	}
	// THE HEAD IS A BOUNDARY TOO, and it is the one the first version missed:
	// the head's own render overlapped the tone, but the FIRST ALERT was then
	// rendered when the read loop started — after the head had been spoken and
	// its pause already held. Every per-event check above passed while the
	// listener waited a full render on that one boundary.
	head := indexOf(seq, "say:structural")
	first := indexOf(seq, "render:norfolk")
	if head < 0 {
		t.Fatalf("the fixture produced no burst head, so this boundary is untested: %v", seq)
	}
	if first < 0 {
		t.Fatalf("the first alert was never rendered: %v", seq)
	}
	// NO HOLD MAY SEPARATE THE HEAD'S WORDS FROM THE FIRST ALERT'S RENDER.
	// Position alone cannot see this defect — both arrangements log the render
	// just after the head's words. What distinguishes them is whether the head's
	// own hold has already been spent by then, so the holds are what this reads.
	for i := head + 1; i < first; i++ {
		if seq[i] == "wait" {
			t.Errorf("the first alert is rendered AFTER the head's hold (words at %d, hold at %d, render at %d), "+
				"so its render sits in the head's pause and the listener waits it out: %v", head, i, first, seq)
			break
		}
	}
}

// endingVoice ends the sequence during its first render, and sounds an INSTANT
// tone — the muted-class case, where the tone's duration is zero.
type endingVoice struct {
	log    *cueLog
	cancel func()
	once   sync.Once
}

func (v *endingVoice) duck()                         {}
func (v *endingVoice) tone(cast.Class) time.Duration { return 0 }
func (v *endingVoice) render(_ context.Context, _ cast.Role, text string) (clip, bool) {
	v.log.add("render" + tag(text))
	v.once.Do(func() { v.cancel() }) // the sequence ends while this render runs
	return clip{text: text, dur: time.Millisecond}, true
}
func (v *endingVoice) play(c clip) { v.log.add("say" + tag(c.text)) }

// fault is inert in this double: the seam exists so a read can report a
// tone with no words (FR-9.2); nothing here reads it.
func (v *endingVoice) fault(string) {}

func (v *endingVoice) pause()   {}
func (v *endingVoice) resume()  {}
func (v *endingVoice) stop()    {}
func (v *endingVoice) discard() {}
func (v *endingVoice) restore() {}

// A SEQUENCE THAT ENDED WHILE THE WORK OVERRAN THE SOUND CUES NOTHING.
//
// `holdRest` waits the REMAINDER of a sound after the render that overlapped it,
// and that remainder is routinely non-positive: a muted class sounds an instant
// tone, so the very first hold of the burst has nothing left to wait. `s.hold(0)`
// returns true WITHOUT reaching its own air check — so a takeover whose context
// died during that render carried on and put a callout on the band for a read
// that never happened. The band would then hold a breaking headline for an alert
// nobody was ever going to speak.
//
// It asks awaitAir explicitly when there is nothing left to wait. Nothing left
// to wait is not the same as nothing left to check.
func TestASequenceThatEndedDuringAnOverlappedRenderCuesNothing(t *testing.T) {
	log := &cueLog{}
	ctx, cancel := context.WithCancel(context.Background())
	d := cueDeckWith(t, log, &endingVoice{log: log, cancel: cancel})

	// A SINGLE EVENT, DELIBERATELY: one alert has no burst head, so nothing
	// stands between the overlapped render and the first cue. With a head, the
	// `burstHeadGap` hold catches the dead context first and the defect is
	// masked — a fixture that hides the case it is written for.
	newStation(t, d).takeover(ctx, twoBreaking()[:1])

	seq := log.all()
	// THE FIXTURE MUST HAVE REACHED A RENDER, or "no cue" is true by vacuity.
	if indexOf(seq, "render:norfolk") < 0 {
		t.Fatalf("nothing was rendered, so this proves nothing about ending mid-render: %v", seq)
	}
	for _, step := range seq {
		if strings.HasPrefix(step, "cue") {
			t.Errorf("the band was cued after the sequence had ended (%q): a callout went up for a "+
				"read that never happened: %v", step, seq)
		}
	}
}

// refuseThenEndVoice refuses its first render outright — a live render failure,
// which is the one case `readBreaking`'s retry exists for — and then ends the
// sequence during the retry.
type refuseThenEndVoice struct {
	log    *cueLog
	cancel func()
	n      int
	mu     sync.Mutex
}

func (v *refuseThenEndVoice) duck() {}

// fault is inert here: the seam exists so a read can report a tone with no
// words (FR-9.2).
func (v *refuseThenEndVoice) fault(string) {}

func (v *refuseThenEndVoice) tone(cast.Class) time.Duration { return 0 }
func (v *refuseThenEndVoice) render(_ context.Context, _ cast.Role, text string) (clip, bool) {
	v.mu.Lock()
	v.n++
	n := v.n
	v.mu.Unlock()
	v.log.add("render" + tag(text))
	if n == 1 {
		return clip{}, false // the render failed on its own; the sequence is still live
	}
	v.cancel() // the sequence ends while the RETRY runs
	return clip{text: text, dur: time.Millisecond}, true
}
func (v *refuseThenEndVoice) play(c clip) { v.log.add("say" + tag(c.text)) }
func (v *refuseThenEndVoice) pause()      {}
func (v *refuseThenEndVoice) resume()     {}
func (v *refuseThenEndVoice) stop()       {}
func (v *refuseThenEndVoice) discard()    {}
func (v *refuseThenEndVoice) restore()    {}

// THE RETRY PATH CUES NOTHING EITHER.
//
// `holdRest` asks the air when a hold has nothing left to wait, which covers the
// paths that reach a cue through a hold. The RETRY does not: when the first
// render fails on its own, `readBreaking` renders a second time and goes
// straight to the cue with no hold between them. A sequence that ended during
// that second render still put a breaking headline on the band for a read that
// never happened — the same defect as the one already pinned, through a door
// the pin did not cover. Found by review, on the fixed code.
func TestTheRetryPathAlsoCuesNothingOnceTheSequenceEnded(t *testing.T) {
	log := &cueLog{}
	ctx, cancel := context.WithCancel(context.Background())
	d := cueDeckWith(t, log, &refuseThenEndVoice{log: log, cancel: cancel})

	newStation(t, d).takeover(ctx, twoBreaking()[:1])

	seq := log.all()
	// THE FIXTURE MUST HAVE TAKEN THE RETRY, or this pins the wrong path.
	if n := strings.Count(strings.Join(seq, " "), "render:norfolk"); n < 2 {
		t.Fatalf("the retry never ran (%d renders), so this proves nothing about it: %v", n, seq)
	}
	for _, step := range seq {
		if strings.HasPrefix(step, "cue") {
			t.Errorf("the band was cued after the sequence ended during the retry (%q): %v", step, seq)
		}
	}
}

// THE DIRECTOR IS TOLD THE PROGRAMME IS RUNNING, NOT ONLY THAT IT STOPPED.
//
// It starts Stopped on purpose — a station comes up silent — and Stop was the
// only power it ever heard, so `advances(MainTrack)` stayed false for the life
// of the process and the bed never moved on. Watchlist looked like it simply did
// nothing, on both the relay path and the synth path.
//
// NO TEST CAUGHT IT BECAUSE EVERY FIXTURE SENT Powered{Running} ITSELF. The
// tests supplied what production had forgotten, which is the one thing a fixture
// must never do for a wiring seam.
//
// MOVED TO tune AT 0.16.0 P3, and the same defect came back through the door
// the first fix left open (red team 2026-09-09, finding 1). The report rode on
// setMode's transition edge, which made "the programme is running" a fact about
// the DECK's mode string; the merged station does not change the deck's mode at
// all, so it was never powered and never read anything. THIS TEST DROVE setMode
// DIRECTLY, so it passed throughout — a pin on the carrier rather than on the
// rule, which is why it could not see the carrier become the wrong one.
//
// It drives `tune` now: the thing the LISTENER does. A pin that names the
// listener's act survives the next time the audio path is rearranged.
func TestTheDeckReportsThatTheProgrammeIsRunning(t *testing.T) {
	d, _ := offlineDeck(t)
	var got []lineup.Event
	d.emit = func(ev lineup.Event) { got = append(got, ev) }

	d.tune(pinRef("A", 33.19, -117.37))
	d.engine.Halt()
	if len(got) == 0 {
		t.Fatalf("starting the programme told the Director nothing")
	}
	if p, ok := got[0].(lineup.Powered); !ok || p.To != lineup.Running {
		t.Errorf("starting reported %#v, want Powered{Running} FIRST — a need reported to a "+
			"stopped Director is a card refused", got[0])
	}

	// A MODE CHANGE IS NOT A POWER CHANGE. The deck moving from synth to a
	// relay tells the Director nothing, because nothing about whether the
	// programme is running has changed — which is the coupling this fix broke.
	got = nil
	d.setMode("live", "KEC62", "a relay")
	if len(got) != 0 {
		t.Errorf("a mode change reported %v; the mode is the deck's business, the power is the listener's", got)
	}

	// And a stop is still reported, so the pair is balanced.
	got = nil
	d.Stop()
	if len(got) != 1 {
		t.Fatalf("a stop told the Director %d things, want one: %v", len(got), got)
	}
	if p, ok := got[0].(lineup.Powered); !ok || p.To != lineup.Stopped {
		t.Errorf("a stop reported %#v, want Powered{Stopped}", got[0])
	}
}

// THE SETTINGS CHOICE REACHES THE DIRECTOR, AND REACHES IT NOW.
//
// Storing the override is only half of it. The Director HOLDS the dwell it was
// last told, so a listener who shortens the rotation mid-programme keeps the
// old one until something else happens to re-send it — which, on a station
// that is already rotating, may be five minutes away. That is indistinguishable
// from "the setting does not work", and it is the complaint that produced this
// setting in the first place.
func TestTheSettingsRotationReachesTheDirectorAtOnce(t *testing.T) {
	d, _ := offlineDeck(t)
	var got []lineup.Event
	d.emit = func(ev lineup.Event) { got = append(got, ev) }
	a := pinRef("A", 33.19, -117.37)

	// The station is rotating on the default when the listener opens Settings.
	d.SetRepeat(tty.RepeatWatchlist, []snapshot.LocationRef{a})
	got = nil

	lp := &livePipelines{deck: d}
	lp.setRelayDwell()(30 * time.Second)

	// SETTINGS BEATS THE ENVIRONMENT. The variable exists to test the rotation
	// without waiting five minutes; a listener who then sets it in the window
	// must not be overruled by a variable they cannot see.
	t.Setenv("WATCHPOST_WATCHLIST_DWELL", "2m")
	if got := d.watchlistDwell(); got != 30*time.Second {
		t.Errorf("the choice was not stored: watchlistDwell is %v", got)
	}
	if len(got) != 1 {
		t.Fatalf("changing the rotation told the Director %d things, want one: %v", len(got), got)
	}
	pr, ok := got[0].(lineup.Programme)
	if !ok {
		t.Fatalf("the Director heard %#v, want a Programme", got[0])
	}
	if pr.Dwell != 30*time.Second {
		t.Errorf("the Director was told %v, the listener chose 30s", pr.Dwell)
	}
	if len(pr.Watchlist) != 1 {
		t.Errorf("the rotation lost its watchlist: %#v", pr.Watchlist)
	}
}
