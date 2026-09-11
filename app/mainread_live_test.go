package app

// mainread_live_test.go — the reader END TO END (F-95).
//
// EVERY OTHER TEST OF THE READER DRIVES A FAKE `read` SEAM. That is right for
// what they assert — which lane performs a card, what comes home — and it is why
// the REAL path shipped a defect that killed every card one millisecond after it
// was asked for: arm the session, start the engine, come home Finished was
// covered by nothing. P-1, for the third time in one session.
//
// IT RUNS ON ANY HOST. The voice is a silent one installed through the deck's
// own seam, and the audio output drains rather than plays, so this needs neither
// `say` nor a sound card — a test that skipped on hosts without them would be a
// gate that does not run.

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// quietVoice renders a fixed, short burst of silence for any line: enough PCM
// for the engine to have something to play, and no dependency on the host.
type quietVoice struct{}

func (quietVoice) Name() string { return "Quiet" }
func (quietVoice) Rate() int    { return player.OutputRate }
func (quietVoice) Say(context.Context, string) ([]byte, error) {
	return make([]byte, 2*player.OutputRate/10), nil // 0.1 s of mono silence
}

// drainOutput consumes the PCM, so a source reaches its end the way a real sound
// card makes it. A player that never drains leaves the stream open for ever,
// which is a different test.
type drainOutput struct{}

type drainPlayer struct {
	r    io.Reader
	once sync.Once
	done chan struct{}
}

func (drainOutput) NewPlayer(r io.Reader) (player.Player, error) {
	return &drainPlayer{r: r, done: make(chan struct{})}, nil
}

func (p *drainPlayer) Play() {
	p.once.Do(func() {
		go func() { _, _ = io.Copy(io.Discard, p.r); close(p.done) }()
	})
}
func (p *drainPlayer) Pause() {}
func (p *drainPlayer) IsPlaying() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}
func (p *drainPlayer) SetVolume(float64) {}
func (p *drainPlayer) Close() error      { return nil }

// liveDeck is a deck whose reader can actually be run.
func liveDeck(t *testing.T) *radioDeck {
	t.Helper()
	d := &radioDeck{voiceFor: func() (synth.Voice, error) { return quietVoice{}, nil }}
	eng, err := player.New(drainOutput{}, "test", d.onStatus)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	d.engine = eng
	t.Cleanup(eng.Halt)
	return d
}

func aReport(id string, lines ...string) lineup.Speak {
	parts := make([]lineup.Part, 0, len(lines))
	for _, l := range lines {
		parts = append(parts, lineup.Part{Kind: lineup.PartLine, Text: l})
	}
	return lineup.Speak{ID: id, Slot: lineup.LocationReport, Headline: "OCEANSIDE, CA",
		Track: lineup.MainTrack, Script: lineup.Script{Parts: parts}}
}

// THE DEFECT, AND IT IS THE ONE THE HUM LEAD REPORTED TWICE (F-95).
//
// `StartSource` HALTS WHATEVER IT IS REPLACING FIRST, and `halt` ends with
// `set(Status{State: Stopped})`. The session is armed before that call — it has
// to be, or a status could land with nothing listening — so the displaced
// source's Stopped came home as THIS read's ending, one millisecond after it was
// asked for and before a word was spoken.
//
// Every card was then Failed{Routed}, discarded, and its location benched for
// five minutes (D-67). At twenty-five pool entries the console read "waiting for
// the line-up" in every slot: "cards churn until it gets to 'waiting for
// line-up'."
//
// MEASURED: the first run of this probe returned false in ONE MILLISECOND, with
// the engine reporting a Stopped whose Name was empty — a status for a source
// that was not this one.
func TestAReadComesHomeFinishedOverARealEngine(t *testing.T) {
	d := liveDeck(t)

	done := make(chan bool, 1)
	start := time.Now()
	go func() {
		done <- readerFor(d)(context.Background(), aReport("r1",
			"Now, the weather for Oceanside.", "Currently sixty one degrees and fair."))
	}()

	select {
	case ok := <-done:
		if !ok {
			if d.source != nil {
				t.Logf("TRACE source err = %v", d.source.Err())
			}
			t.Logf("TRACE engine = %+v", d.engine.Status())
			t.Fatalf("the read came home FAILED after %s; every card of the line-up would be "+
				"discarded and its location benched for five minutes", time.Since(start).Round(time.Millisecond))
		}
		// AND IT TOOK TIME, which is the half that says the words were actually
		// played rather than the session being closed by somebody else's status.
		// The defect returned in about a millisecond.
		if el := time.Since(start); el < 50*time.Millisecond {
			t.Errorf("the read came home in %s; nothing can have been spoken in that time", el)
		}
	case <-time.After(30 * time.Second):
		t.Fatalf("the read never came home; the engine reports %+v", d.engine.Status())
	}
}

// AND THE WORDS THAT WENT OUT ARE THE CARD'S. The reader plays the card's script
// and never recomposes: what the operator read in the slot is what the listener
// hears (D-81).
func TestTheReadPlaysTheCardsOwnWords(t *testing.T) {
	d := liveDeck(t)
	var said []string
	var mu sync.Mutex
	d.voiceFor = func() (synth.Voice, error) {
		return recordingVoice{say: func(text string) { mu.Lock(); said = append(said, text); mu.Unlock() }}, nil
	}

	if !readerFor(d)(context.Background(), aReport("r1", "first line", "second line")) {
		t.Fatal("the read failed")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, want := range []string{"first line", "second line"} {
		found := false
		for _, s := range said {
			if s == want {
				found = true
			}
		}
		if !found {
			t.Errorf("the card's %q never reached the voice; it said %v", want, said)
		}
	}
}

type recordingVoice struct{ say func(string) }

func (recordingVoice) Name() string { return "Recording" }
func (recordingVoice) Rate() int    { return player.OutputRate }
func (v recordingVoice) Say(_ context.Context, text string) ([]byte, error) {
	v.say(text)
	return bytes.Repeat([]byte{0}, 2*player.OutputRate/10), nil
}

// THE HALT OF THE SOURCE THIS READ REPLACED IS NOT THIS READ ENDING (F-95).
//
// The unit form of the defect above, pinned at the exact sequence the engine
// produces: `StartSource` halts first and `halt` ends with a Stopped, and only
// then does this read's own audio report Connecting.
func TestAReadIsNotEndedByTheHaltOfTheSourceItReplaced(t *testing.T) {
	r := &readSession{done: make(chan struct{})}

	// StartSource halts whatever was playing. This Stopped is not ours.
	noteRead(r, player.Status{State: player.Stopped}, false)
	select {
	case <-r.done:
		t.Fatal("the read ended on a status for the source it displaced, before a word was spoken — " +
			"every card of the line-up is then discarded and its location benched for five minutes")
	default:
	}

	// Then this read's own audio starts, plays, and runs to its sign-off.
	noteRead(r, player.Status{State: player.Connecting, Name: "Watchpost Programme"}, false)
	noteRead(r, player.Status{State: player.Playing, Name: "Watchpost Programme"}, false)
	noteRead(r, player.Status{State: player.Stopped, Title: player.EndedTitle}, true)

	select {
	case <-r.done:
	default:
		t.Fatal("the read never came home")
	}
	if !r.ok {
		t.Error("a read that played to its sign-off came home failed")
	}
}

// AND A READ THAT IS HALTED AFTER IT HAS STARTED STILL SAYS SO (DR-24). The
// guard is "has this read begun", not "ignore the first Stopped".
func TestAReadHaltedAfterItStartedComesHomeFailed(t *testing.T) {
	r := &readSession{done: make(chan struct{})}
	noteRead(r, player.Status{State: player.Playing}, false)
	noteRead(r, player.Status{State: player.Stopped}, false)

	select {
	case <-r.done:
	default:
		t.Fatal("a read halted mid-sentence never came home; the schedule waits on it for ever")
	}
	if r.ok {
		t.Error("a read cut short came home finished; the card leaves the line-up unread")
	}
}
