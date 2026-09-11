package app

// The main track's reader (F-91): who performs a card, what comes home when the
// words run out, and what a read must NOT tell the schedule.

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/cast"

	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// report is a main-track card as the Director emits it.
func report(id string, words ...string) lineup.Speak {
	var parts []lineup.Part
	for _, w := range words {
		parts = append(parts, lineup.Part{Kind: lineup.PartLine, Text: w})
	}
	return lineup.Speak{ID: id, Slot: lineup.LocationReport, Headline: "OCEANSIDE, CA",
		Track: lineup.MainTrack, Script: lineup.Script{Parts: parts}}
}

// THE DEFECT THIS CLOSES, AND IT IS WHAT THE HUM LEAD HEARD (UAT 2026-09-11):
// every location report was admitted, built, then DECLINED at Speak — "the
// arbiter reads the rail, and the rail only" — which failed the card, discarded
// it and benched its location for five minutes (D-67). With the pool at 25 and
// the console drawing ten slots, the whole pool benches in one pass and every
// slot reads "waiting for the line-up". No audio, and the schedule churning.
func TestTheProgrammeIsPerformedOnTheBroadcastEngineAndNotThroughTheArbiter(t *testing.T) {
	v := &scriptVoice{}
	b := newBench(t, v)

	out := b.x.run(context.Background(), report("r1", "Now, the weather for Oceanside."))

	if len(out) != 1 {
		t.Fatalf("a read that finished comes home with one event; got %v", out)
	}
	if _, ok := out[0].(lineup.Finished); !ok {
		t.Fatalf("got %T, want Finished: the card was read in full", out[0])
	}
	if got := b.readCalls(); len(got) != 1 || got[0].ID != "r1" {
		t.Fatalf("the programme must reach the broadcast engine; it was asked for %v", got)
	}
	// THE ARBITER MUST NOT HAVE SEEN IT (D-33). A programme read through the
	// narration path speaks OVER the bed instead of replacing it, which is the
	// design that was built once and ruled out.
	if v.got() != "" {
		t.Errorf("the programme reached the narration arbiter (%q); a chosen read replaces the bed", v.got())
	}
	if r := b.reported(); len(r) != 0 {
		t.Errorf("a performed effect was also reported as declined: %v", r)
	}
}

// DR-24, ON BOTH LANES. A read that stopped early must say so — the operator
// went to standby, the pump is stopping, a line would not render — because a
// Finished tells the schedule a read happened that did not, and the card would
// sit ON AIR for ever with the slot holding a callout for it.
func TestAProgrammeReadThatEndedEarlyFailsTheCardRatherThanFinishingIt(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	b.readOK = false

	out := b.x.run(context.Background(), report("r1", "Now, the weather for Oceanside."))

	f := onlyFailed(t, out)
	if f.ID != "r1" {
		t.Errorf("failed %q, want r1", f.ID)
	}
	// ROUTED (I-2). None of the ways a read ends early is the station going
	// quiet, and a relay-fault modal for any of them is the noise regression
	// fault.go exists to avoid.
	if !f.Routed {
		t.Error("a read that ended early is routed, not a fault: the station is fine")
	}
}

// A STATION WITH NO BROADCAST ENGINE DECLINES BY NAME, never silently. The
// pathless build has no deck, and a card held on the air waiting for words that
// cannot come is DR-21's exact defect.
func TestAStationWithNoBroadcastEngineDeclinesTheProgrammeByName(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	b.x.read = nil

	out := b.x.run(context.Background(), report("r1", "Now, the weather for Oceanside."))

	if f := onlyFailed(t, out); f.ID != "r1" {
		t.Errorf("failed %q, want r1", f.ID)
	}
	r := b.reported()
	if len(r) != 1 || !strings.Contains(r[0], "no reader for the main track") {
		t.Errorf("reported %v, want one line naming the missing reader", r)
	}
}

// THE LANE DECIDES WHO READS, NOT THE SLOT (F-91). A transition's slot says
// only that the Director minted it. A hand-back after a takeover is read on the
// RAIL, in the breaking correspondent's voice, with the duck still down; a
// stale notice standing in for a dropped report is the PROGRAMME. One reader
// for both would say the wrong thing in the wrong voice, and the slot cannot
// tell them apart.
func TestATransitionIsReadByTheLaneItBookends(t *testing.T) {
	for _, tc := range []struct {
		name    string
		track   lineup.Track
		arbiter bool // read through the narration arbiter
	}{
		{"a hand-back on the rail", lineup.AlertRail, true},
		{"a stale notice on the programme", lineup.MainTrack, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := &scriptVoice{}
			b := newBench(t, v)
			sp := lineup.Speak{ID: "t1", Slot: lineup.Transition, Headline: "Report out of date",
				Track: tc.track, Script: lineup.Say("we now return")}

			if out := b.x.run(context.Background(), sp); len(out) != 1 {
				t.Fatalf("a transition that was read comes home once; got %v", out)
			}
			spoke, engine := v.got() != "", len(b.readCalls()) == 1
			if spoke != tc.arbiter || engine == tc.arbiter {
				t.Errorf("arbiter spoke=%t engine played=%t, want arbiter=%t", spoke, engine, tc.arbiter)
			}
		})
	}
}

// `[M]` IS THE ALERT MUTE, AND IT IS NOT THE STATION'S OFF SWITCH.
//
// The rail declines under it because reading a burst inaudibly would MARK every
// alert read and swallow a tornado warning (MVS-D-78) — the card is consumed
// either way. A location report consumes nothing: declining it would silence a
// station the operator has deliberately put ON AIR, over a control belonging to
// the programme they are not listening to.
func TestTheListenersAlertMuteDoesNotSilenceTheProgramme(t *testing.T) {
	b := newBench(t, &scriptVoice{})
	b.muted = true

	out := b.x.run(context.Background(), report("r1", "Now, the weather for Oceanside."))

	if len(out) != 1 {
		t.Fatalf("got %v", out)
	}
	if _, ok := out[0].(lineup.Finished); !ok {
		t.Fatalf("got %T, want Finished: the mute is the rail's, not the station's", out[0])
	}
	if len(b.readCalls()) != 1 {
		t.Error("a muted listener stopped the station's own programme")
	}
}

// THE WORDS SURVIVE THE CARD, AND NOTHING ELSE DOES (G-7). A card carries Text,
// because lineup.Part has nowhere to put a role or a pause and DR-1 keeps the
// domain out of the schedule — so the reader plays the card's words in the
// station's own voice. What must never differ is WHAT IS SAID.
func TestTheCardsWordsAreWhatReachesTheEngine(t *testing.T) {
	segs := []synth.Segment{
		{Key: "lead", Text: "Now, the weather for Oceanside.", Role: cast.Weather, Pause: time.Second},
		{Key: "blank", Text: "   "},
		{Key: "tail", Text: "This is Watchpost Weather Radio."},
	}
	sc := scriptFromSegments(segs)
	back := segmentsFromScript("r1", sc)

	if len(back) != 2 {
		t.Fatalf("a blank part must be dropped both ways; got %d segments: %+v", len(back), back)
	}
	if back[0].Text != segs[0].Text || back[1].Text != segs[2].Text {
		t.Errorf("the words changed across the card: %q / %q", back[0].Text, back[1].Text)
	}
	// THE KEY NAMES THE CARD. The Source caches rendered PCM by (key, voice);
	// two cards for the same location sharing a key would play each other's
	// audio.
	for i, seg := range back {
		if !strings.HasPrefix(seg.Key, "card:r1:") {
			t.Errorf("segment %d keyed %q, want it scoped to the card", i, seg.Key)
		}
	}
	if back[0].Key == back[1].Key {
		t.Error("two parts of one card share a cache key")
	}
	// AND NOTHING IS INVENTED. A role the card never carried would be a second
	// answer to who reads this station.
	if back[0].Role != cast.All || back[0].Pause != 0 {
		t.Errorf("the reader invented a role or a pause: %+v", back[0])
	}
}

// A READ ENDS ONCE, AND ONLY THE SIGN-OFF IS SUCCESS.
//
// The engine reports several statuses for one stream — a title change, then the
// stop — so a session that closed its channel twice would panic on the audio
// goroutine. And a Stopped that is NOT the sign-off is a Halt: the operator
// went to standby, or a later read took the engine.
func TestAReadSessionEndsOnceAndDistinguishesTheSignOffFromAHalt(t *testing.T) {
	for _, tc := range []struct {
		name  string
		st    player.Status
		ended bool
		want  bool
	}{
		{"the words ran out", player.Status{State: player.Stopped, Title: player.EndedTitle}, true, true},
		{"halted mid-read", player.Status{State: player.Stopped}, false, false},
		{"the voice died", player.Status{State: player.Failed, Err: "no voice"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &readSession{done: make(chan struct{})}
			noteRead(r, tc.st, tc.ended)
			noteRead(r, tc.st, tc.ended) // a second status must not close a closed channel
			select {
			case <-r.done:
			default:
				t.Fatal("a terminal status must end the read; the executor waits for ever otherwise")
			}
			if r.ok != tc.want {
				t.Errorf("ok=%t, want %t", r.ok, tc.want)
			}
		})
	}
}

// A STREAM STILL PLAYING ENDS NOTHING. Statuses arrive for every title change,
// and a read that came home on the first of them would report Finished with the
// words still going out.
func TestAReadIsNotEndedByAStatusThatIsStillPlaying(t *testing.T) {
	r := &readSession{done: make(chan struct{})}
	noteRead(r, player.Status{State: player.Playing, Title: "Oceanside"}, false)
	select {
	case <-r.done:
		t.Fatal("a Playing status ended the read")
	default:
	}
	noteRead(nil, player.Status{State: player.Stopped}, true) // no session: inert, never a panic
}

// A MAIN-TRACK READ IS NOT THE BED MOVING (F-91) — and this is the churn the
// HUM LEAD saw, arrived at from the other side.
//
// The engine reports the same Playing and Stopped for a card as it does for the
// bed. Told about them, the Director would take a card's start as `Tuned` — the
// bed landed somewhere, start its dwell — and a card's end as `Ended`, which
// moves the monitor's rotation on. The schedule would be driving the listener's
// watchlist from the station's programme.
func TestAMainTrackReadTellsTheDirectorNothingAboutTheBed(t *testing.T) {
	for _, tc := range []struct {
		mode string
		want bool // the bed's story reaches the Director
	}{
		{"read", false},
		{"synth", true},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			var told []lineup.Event
			d := &radioDeck{}
			d.emit = func(ev lineup.Event) { told = append(told, ev) }
			d.mode = tc.mode
			d.read = &readSession{done: make(chan struct{})}

			d.onStatus(player.Status{State: player.Playing, Name: "x"})
			d.onStatus(player.Status{State: player.Stopped, Title: player.EndedTitle})

			var bed int
			for _, ev := range told {
				switch ev.(type) {
				case lineup.Tuned, lineup.Ended:
					bed++
				}
			}
			if (bed > 0) != tc.want {
				t.Fatalf("mode %q told the Director %v; bed events wanted: %t", tc.mode, told, tc.want)
			}
			// AND THE READ ITSELF ENDS, which is the other half: the executor
			// blocks on this channel until the words run out.
			ended := false
			select {
			case <-d.read.done:
				ended = true
			default:
			}
			if ended != (tc.mode == "read") {
				t.Errorf("mode %q: read ended=%t", tc.mode, ended)
			}
		})
	}
}

// A HAZARD IS NEVER READ AS THE PROGRAMME. `Track`'s zero value is MainTrack,
// so an effect built without one routes to the broadcast engine by default —
// and a takeover read down that path plays its words with no attention tone and
// no per-alert callouts. A tornado warning, delivered as the weather.
func TestARailCardThatReachesTheProgrammesReaderIsRefused(t *testing.T) {
	b := newBench(t, &scriptVoice{})

	out := b.x.run(context.Background(), lineup.Speak{ID: "a1", Slot: lineup.BreakingAlert,
		Script: lineup.Script{Tone: "breaking", Parts: []lineup.Part{{Kind: lineup.PartLine, Text: "a tornado warning is in effect"}}}})

	if f := onlyFailed(t, out); f.ID != "a1" {
		t.Errorf("failed %q, want a1", f.ID)
	}
	if len(b.readCalls()) != 0 {
		t.Error("a breaking alert was played as the programme; its tone and callouts are gone")
	}
	r := b.reported()
	if len(r) != 1 || !strings.Contains(r[0], "tone") {
		t.Errorf("reported %v, want one line naming what would be lost", r)
	}
}

// STANDBY STOPS THE WORDS, AND SO DOES HANDING THE AIR BACK (F-91).
//
// The Director takes the card off the air when the power drops — but a card
// already reading is a WORKER BLOCKING ON THE ENGINE, not something the
// schedule can recall. Without this the operator presses GO TO STANDBY, the
// console says the station is off, and the station keeps talking to the end of
// the report. The same hole the other way: moving back to Observer must leave
// it "like if Observer was first opened", which is silent.
func TestTheOperatorSilencingTheStationStopsAReadAlreadyGoingOut(t *testing.T) {
	for _, tc := range []struct {
		name    string
		act     func(*mastercontrol)
		silence bool
	}{
		{"go to standby", func(m *mastercontrol) { m.GoToStandby() }, true},
		{"the air goes back to the monitor", func(m *mastercontrol) { m.HandAir(lineup.AirMonitor) }, true},
		{"go on air", func(m *mastercontrol) { m.GoOnAir() }, false},
		{"the operator stops listening", func(m *mastercontrol) { m.StopMonitor() }, false},
		{"cut to the bed", func(m *mastercontrol) { m.CutBed(true) }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mc := newMastercontrol(nil, func(tea.Msg) {})
			if mc == nil {
				t.Fatal("newMastercontrol refused a well-formed set")
			}
			silenced := 0
			mc.mu.Lock()
			mc.silenceProgramme = func() { silenced++ }
			mc.mu.Unlock()

			tc.act(mc)

			if (silenced > 0) != tc.silence {
				t.Errorf("%s silenced the programme %d time(s); wanted silence: %t", tc.name, silenced, tc.silence)
			}
		})
	}
}

// THE READ IS CANCELLED, NEVER HALTED FROM THE CALLER'S GOROUTINE (D-79).
//
// `engine.Halt` waits for the audio goroutine, and that goroutine is calling
// back into the program — so a halt reached from `Router.Update` sends to a loop
// that cannot receive and the app freezes hard enough to need the terminal
// killed. That is not a hypothetical: it is what happened on 2026-09-11, twice.
// `stopRead` must return whether or not anything is playing, and the WORKER
// waiting on the read is what does the halting.
func TestStoppingAReadReturnsAtOnceAndEndsTheWait(t *testing.T) {
	d := &radioDeck{}
	d.stopRead() // nothing reading: inert, and above all not a panic

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rctx, stop := context.WithCancel(ctx)
	defer stop()
	d.mu.Lock()
	d.read = &readSession{done: make(chan struct{}), stop: stop}
	d.mu.Unlock()

	done := make(chan struct{})
	go func() { defer close(done); d.stopRead() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stopRead blocked; from the update loop that is the freeze the HUM LEAD had to kill his terminal for")
	}
	select {
	case <-rctx.Done():
	default:
		t.Error("stopRead did not end the read's context, so the words play on")
	}
}
