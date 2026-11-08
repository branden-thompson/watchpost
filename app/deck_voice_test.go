package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/modes/tty"
)

// The tests in this file are 0.19.5 Q2.1 (RN-1): the radio deck's voice and
// relay-fault side, run on a real engine over an audio output that plays every
// clip to its end, so what the listener would hear and what the visualiser
// would draw can both be read back.

// drainingOutput plays each clip to its end at once and remembers its players.
type drainingOutput struct {
	mu      sync.Mutex
	players []*drainPlayer
}

func (o *drainingOutput) NewPlayer(r io.Reader) (player.Player, error) {
	p := &drainPlayer{r: r, done: make(chan struct{})}
	o.mu.Lock()
	o.players = append(o.players, p)
	o.mu.Unlock()
	return p, nil
}

// played waits for every player so far to finish, and says how many there were.
func (o *drainingOutput) played(t *testing.T) int {
	t.Helper()
	o.mu.Lock()
	ps := append([]*drainPlayer(nil), o.players...)
	o.mu.Unlock()
	for _, p := range ps {
		select {
		case <-p.done:
		case <-time.After(10 * time.Second):
			t.Fatal("a clip never finished playing")
		}
	}
	return len(ps)
}

// voiceDeck is a deck on a real engine over a draining output, with every
// status the engine reports kept in order, and a program whose messages the
// test can read.
func voiceDeck(t *testing.T) (*radioDeck, *drainingOutput, *statusLog, *sent) {
	t.Helper()
	asPlatform(t, "darwin")
	out := &drainingOutput{}
	log := &statusLog{}
	eng, err := player.New(out, "test", log.add)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	t.Cleanup(eng.Halt)
	msgs := &sent{}
	d := &radioDeck{engine: eng, limiter: synth.NewLimiter(2, synth.ReservedSlots), voiceDir: t.TempDir()}
	d.program = msgs.program(t)
	return d, out, log, msgs
}

// statusLog is every status the engine reported, in order.
type statusLog struct {
	mu   sync.Mutex
	seen []player.Status
}

func (l *statusLog) add(st player.Status) {
	l.mu.Lock()
	l.seen = append(l.seen, st)
	l.mu.Unlock()
}

// mounts is the mounts the engine tried, in the order it tried them.
func (l *statusLog) mounts() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, st := range l.seen {
		if st.Mount != "" && (len(out) == 0 || out[len(out)-1] != st.Mount) {
			out = append(out, st.Mount)
		}
	}
	return out
}

// sent is the messages a deck handed its program.
type sent struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

// sendRecorder is a program's model that only keeps what it is sent.
type sendRecorder struct{ s *sent }

func (r sendRecorder) Init() tea.Cmd { return nil }
func (r sendRecorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	r.s.mu.Lock()
	r.s.msgs = append(r.s.msgs, msg)
	r.s.mu.Unlock()
	return r, nil
}
func (r sendRecorder) View() tea.View { return tea.NewView("") }

// program runs a headless program over the sendRecorder until the test ends.
func (s *sent) program(t *testing.T) *tea.Program {
	t.Helper()
	p := tea.NewProgram(sendRecorder{s}, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutSignalHandler())
	ended := make(chan struct{})
	go func() { _, _ = p.Run(); close(ended) }()
	t.Cleanup(func() { p.Quit(); <-ended })
	return p
}

// await waits for a message the predicate accepts.
func (s *sent) await(t *testing.T, what string, ok func(tea.Msg) bool) tea.Msg {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		s.mu.Lock()
		for _, m := range s.msgs {
			if ok(m) {
				s.mu.Unlock()
				return m
			}
		}
		s.mu.Unlock()
	}
	t.Fatalf("the deck never sent %s", what)
	return nil
}

// detailSaying accepts a status message whose detail line holds want.
func detailSaying(want string) func(tea.Msg) bool {
	return func(m tea.Msg) bool {
		st, ok := m.(tty.RadioStatusMsg)
		return ok && strings.Contains(st.Detail, want)
	}
}

// barsFed is how many samples the visualiser's tap holds.
func barsFed(d *radioDeck) int {
	return d.engine.Samples(make([]float64, 4096))
}

// TestAMutedClassSoundsNoToneAndAToneNeverDrivesTheBars: a muted class's
// alert reads with no tone (MVS-D-26), and an unmuted class's tone plays
// beside the broadcast, never through the visualiser.
func TestAMutedClassSoundsNoToneAndAToneNeverDrivesTheBars(t *testing.T) {
	d, out, _, _ := voiceDeck(t)
	d.cast.cfg.Tones = cast.Tones{Mode: cast.ModeTonesMute, Muted: []string{cast.ClassWarning.Key()}}
	if dur := d.tone(cast.ClassWarning); dur != 0 || out.played(t) != 0 {
		t.Fatalf("a muted class sounded a %v tone", dur)
	}
	d.cast.cfg.Tones = cast.Tones{}
	if dur := d.tone(cast.ClassWarning); dur <= 0 {
		t.Fatal("an unmuted class sounded no tone")
	}
	if out.played(t) != 1 {
		t.Fatal("the tone was not played")
	}
	if n := barsFed(d); n != 0 {
		t.Errorf("the tone fed the visualiser %d samples; it plays aside", n)
	}
}

// TestAVizClipDrivesTheBarsAndATakeoverDoesNot: an event the listener asked to
// hear is drawn by the bars; a takeover's line plays aside, so the bars keep
// to the broadcast it interrupts.
func TestAVizClipDrivesTheBarsAndATakeoverDoesNot(t *testing.T) {
	pcm := alertTonePCM(cast.ClassWarning)
	takeover, out, _, _ := voiceDeck(t)
	takeover.play(clip{pcm: pcm, rate: synth.ToneRate})
	out.played(t)
	if n := barsFed(takeover); n != 0 {
		t.Errorf("a takeover's line fed the visualiser %d samples", n)
	}
	viz, out2, _, _ := voiceDeck(t)
	viz.play(clip{pcm: pcm, rate: synth.ToneRate, viz: true})
	out2.played(t)
	if barsFed(viz) == 0 {
		t.Error("a line the bars follow fed the visualiser nothing")
	}
}

// TestALineThatCannotBeReadSaysWhy: a silent takeover is never unexplained; the
// player's detail line names the voice that could not read.
func TestALineThatCannotBeReadSaysWhy(t *testing.T) {
	d, _, _, msgs := voiceDeck(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := d.render(ctx, cast.Alerts, "A tornado warning."); ok {
		t.Fatal("a line rendered under a cancelled context")
	}
	msgs.await(t, "the reason a line could not be read", detailSaying("could not read"))
}

// TestASpentClipIsSaidOnlyWhenAReadWasOnTheAir (FR-9): a read the player
// abandoned is told to the listener only if they were hearing it.
func TestASpentClipIsSaidOnlyWhenAReadWasOnTheAir(t *testing.T) {
	const said = "a read did not finish and was ended"
	idle, _, _, quiet := voiceDeck(t)
	idle.abandonRead = func(string) bool { return false }
	idle.onClipSpent()
	reading, _, _, told := voiceDeck(t)
	reading.abandonRead = func(string) bool { return true }
	reading.onClipSpent()
	told.await(t, "the spent read's words", detailSaying(said))
	quiet.mu.Lock()
	defer quiet.mu.Unlock()
	for _, m := range quiet.msgs {
		if detailSaying(said)(m) {
			t.Error("a spent clip from a read already ended was put on the detail line")
		}
	}
}

// relayStation is a station with one mount at url.
func relayStation(callsign, url string) stream.Station {
	return stream.Station{Transmitter: &stream.Transmitter{Callsign: callsign, Site: "Somewhere", State: "CA", FreqMHz: "162.550"},
		Mounts: []stream.Mount{{Callsign: callsign, URL: url, Relay: "wxradio.org"}}}
}

// TestASilentRelayOffersTheOtherStationsAndAPickLeads (MVS-D-76): the window
// offers every other station on the tune list once, in the engine's order; a
// pick plays that station's mounts first and the rest after, so a dead pick
// still falls through.
func TestASilentRelayOffersTheOtherStationsAndAPickLeads(t *testing.T) {
	gone := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(gone.Close)
	d, _, log, msgs := voiceDeck(t)
	urls := []string{gone.URL + "/a", gone.URL + "/b", gone.URL + "/c"}
	d.mountURLs = urls
	d.mountOwner = map[string]stream.Station{
		urls[0]: relayStation("KEC49", urls[0]),
		urls[1]: relayStation("WXL98", urls[1]),
		urls[2]: relayStation("KIH62", urls[2]),
	}
	d.onSilence(urls[0], "")
	m := msgs.await(t, "the relay-fault window", func(m tea.Msg) bool { _, ok := m.(tty.RelaySilentMsg); return ok })
	var offered []string
	for _, c := range m.(tty.RelaySilentMsg).Candidates {
		offered = append(offered, c.Key)
	}
	if got := strings.Join(offered, ","); got != "WXL98,KIH62" {
		t.Errorf("the window offers %s; want the other stations in order, WXL98,KIH62", got)
	}

	d.tuneCallsign("KIH62")
	for deadline := time.Now().Add(10 * time.Second); len(log.mounts()) < 3 && time.Now().Before(deadline); {
		time.Sleep(5 * time.Millisecond)
	}
	if got, want := strings.Join(log.mounts(), ","), urls[2]+","+urls[0]+","+urls[1]; got != want {
		t.Errorf("the engine tried %s; want the pick first, then the rest: %s", got, want)
	}
}

// TestTheSourcePickIsAlwaysSaved (D-91): [m] saves the pick whatever holds the
// air; that the re-tune is refused while the console holds it is pinned by
// air_boundary_test.go, where Radio.SetMode is classified as the monitor's.
func TestTheSourcePickIsAlwaysSaved(t *testing.T) {
	d, _, _, _ := voiceDeck(t)
	var saved []tty.RadioMode
	d.persistMode = func(m tty.RadioMode) error { saved = append(saved, m); return nil }
	d.air = func() bool { return false } // the console holds the air
	d.SetMode(tty.ModeRelay)
	if len(saved) != 1 || saved[0] != tty.ModeRelay {
		t.Fatalf("the pick was not saved: %v", saved)
	}
	d.mu.Lock()
	pref := d.pref
	d.mu.Unlock()
	if pref != tty.ModeRelay {
		t.Errorf("the deck holds %v after [m]; want the relay", pref)
	}
}
