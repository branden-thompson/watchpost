package tty

import (
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/modes/tty/internal/frame"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// traceMessages is the alphabet the router trace draws from, each message
// named in the trace by its index: the keys a listener and an operator press
// (navigation, the windows, the player, the swaps, the digits, typing), the
// map's keys (Propagation, its layer, the chart, the reach, the overlays, the
// playback, zoom), resizes, and the ticks that arrive between them. Widened
// once at 0.19.6 B0 (RX-8, A-12), before the golden was recorded; it is never
// reordered or extended, since adding a message changes every draw. An entry's
// spelling may change with a move (tickMsg{} becoming frame.TickMsg{}) and its
// index does not.
var traceMessages = []tea.Msg{
	tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyDown},
	tea.KeyPressMsg{Code: tea.KeyLeft}, tea.KeyPressMsg{Code: tea.KeyRight},
	tea.KeyPressMsg{Code: tea.KeyTab}, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},
	tea.KeyPressMsg{Code: tea.KeyEnter}, tea.KeyPressMsg{Code: tea.KeyEscape},
	tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, tea.KeyPressMsg{Code: tea.KeyBackspace},
	tea.KeyPressMsg{Code: '?', Text: "?"}, tea.KeyPressMsg{Code: 's', Text: "s"},
	tea.KeyPressMsg{Code: 'w', Text: "w"}, tea.KeyPressMsg{Code: 'r', Text: "r"},
	tea.KeyPressMsg{Code: 'v', Text: "v"}, tea.KeyPressMsg{Code: 'm', Text: "m"},
	tea.KeyPressMsg{Code: 'g', Text: "g"}, tea.KeyPressMsg{Code: 'a', Text: "a"},
	tea.KeyPressMsg{Code: 'O', Text: "O"}, tea.KeyPressMsg{Code: 'B', Text: "B"},
	tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}, tea.KeyPressMsg{Code: '2', Text: "2"},
	tea.KeyPressMsg{Code: '5', Text: "5"}, tea.KeyPressMsg{Code: 'x', Text: "x"},
	tea.WindowSizeMsg{Width: 133, Height: 44}, tea.WindowSizeMsg{Width: 80, Height: 24},
	tea.WindowSizeMsg{Width: 160, Height: 74},
	frame.TickMsg{}, vizTickMsg{}, TickerAdvanceMsg{},
	// 0.19.6 B0's widening (RX-8): the map, Propagation, Settings and the console.
	tea.KeyPressMsg{Code: 'P', Text: "P"}, tea.KeyPressMsg{Code: 'L', Text: "L"},
	tea.KeyPressMsg{Code: 'U', Text: "U"}, tea.KeyPressMsg{Code: 'F', Text: "F"},
	tea.KeyPressMsg{Code: 'R', Text: "R"}, tea.KeyPressMsg{Code: 'A', Text: "A"},
	tea.KeyPressMsg{Code: '[', Text: "["}, tea.KeyPressMsg{Code: ']', Text: "]"},
	tea.KeyPressMsg{Code: '+', Text: "+"}, tea.KeyPressMsg{Code: '-', Text: "-"},
	tea.KeyPressMsg{Code: '<', Text: "<"}, tea.KeyPressMsg{Code: 'n', Text: "n"},
	tea.KeyPressMsg{Code: tea.KeyPgUp}, tea.KeyPressMsg{Code: tea.KeyPgDown},
	tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift},
	tea.KeyPressMsg{Code: '1', Text: "1"}, tea.KeyPressMsg{Code: 't', Text: "t"},
	tea.KeyPressMsg{Code: 'S', Text: "S"}, tea.KeyPressMsg{Code: 'c', Text: "c"},
	tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl}, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl},
	tea.KeyPressMsg{Code: 'g', Text: "g"}, tea.KeyPressMsg{Code: 'P', Text: "P"},
}

// key is a key press as the alphabet spells it.
func key(code rune, text string) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Text: text} }

// traceWalk is one seed's walk: the messages it opens with, then the pool its
// random draws come from, nil for the whole alphabet. Uniform draws over the
// whole alphabet reach the map's modes too rarely to hold them, since Settings
// and the other windows, once open, take most keys; a walk with a pool spends
// its steps where it is meant to, and every message is still named by its
// index in the one alphabet.
type traceWalk struct {
	opening, pool []tea.Msg
}

// mapPool is the map window's own keys, with the ticks and resizes between
// them.
var mapPool = []tea.Msg{
	tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyDown},
	tea.KeyPressMsg{Code: tea.KeyLeft}, tea.KeyPressMsg{Code: tea.KeyRight},
	tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, tea.KeyPressMsg{Code: tea.KeyPgUp}, tea.KeyPressMsg{Code: tea.KeyPgDown},
	tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift},
	key('O', "O"), key('B', "B"), key('R', "R"), key('A', "A"), key('[', "["), key(']', "]"),
	key('+', "+"), key('-', "-"), key('<', "<"), key('n', "n"), key('1', "1"), key('2', "2"), key('5', "5"),
	key('L', "L"), key('U', "U"), key('F', "F"), key('P', "P"),
	tea.WindowSizeMsg{Width: 133, Height: 44}, tea.WindowSizeMsg{Width: 160, Height: 74},
	frame.TickMsg{}, vizTickMsg{}, TickerAdvanceMsg{},
}

// propPool is the Propagation mode's keys: the map's, without the ones that
// leave the mode for the weather.
var propPool = []tea.Msg{
	tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyDown},
	tea.KeyPressMsg{Code: tea.KeyLeft}, tea.KeyPressMsg{Code: tea.KeyRight},
	key('B', "B"), key('A', "A"), key('[', "["), key(']', "]"), key('+', "+"), key('-', "-"),
	key('L', "L"), key('U', "U"), key('F', "F"), key('1', "1"),
	tea.WindowSizeMsg{Width: 133, Height: 44}, tea.WindowSizeMsg{Width: 160, Height: 74},
	frame.TickMsg{}, TickerAdvanceMsg{},
}

// settingsPool is the Settings window's keys: its rows, its tabs, its
// controls and typing, with no esc, so a walk stays in the window.
var settingsPool = []tea.Msg{
	tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyDown},
	tea.KeyPressMsg{Code: tea.KeyLeft}, tea.KeyPressMsg{Code: tea.KeyRight},
	tea.KeyPressMsg{Code: tea.KeyTab}, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},
	tea.KeyPressMsg{Code: tea.KeyEnter}, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "},
	tea.KeyPressMsg{Code: tea.KeyBackspace}, key('1', "1"), key('2', "2"), key('5', "5"),
	frame.TickMsg{}, TickerAdvanceMsg{},
}

// traceWalks are the six seeds' walks: the whole alphabet from the
// dashboard, the map, the map in Propagation, the whole alphabet from the
// console, Settings from the dashboard with one tone class muted and enabled
// again, and Settings from the console, where the station's rows are. The last
// two are 0.19.6 D-67's: each walk keeps its seed, so a walk added at the end
// leaves every line before it as it was recorded.
var traceWalks = []traceWalk{
	{},
	{opening: []tea.Msg{key('g', "g")}, pool: mapPool},
	{opening: []tea.Msg{key('g', "g"), key('P', "P")}, pool: propPool},
	{opening: []tea.Msg{tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl}}},
	{opening: []tea.Msg{key('s', "s"), tea.KeyPressMsg{Code: tea.KeyTab}, tea.KeyPressMsg{Code: tea.KeyTab}, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}}, pool: settingsPool},
	{opening: []tea.Msg{tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl}, key('s', "s")}, pool: settingsPool},
}

// traceIndex is a message's index in traceMessages.
func traceIndex(t *testing.T, msg tea.Msg) int {
	t.Helper()
	for i, m := range traceMessages { // bounded by the alphabet (P10-02)
		if reflect.DeepEqual(m, msg) {
			return i
		}
	}
	t.Fatalf("%v is not in traceMessages", msg)
	return 0
}

// draw is a walk's next message index: its opening first, then a draw from
// its pool.
func (w traceWalk) draw(t *testing.T, rnd *rand.Rand, step int) int {
	t.Helper()
	if step < len(w.opening) {
		return traceIndex(t, w.opening[step])
	}
	if w.pool == nil {
		return rnd.IntN(len(traceMessages))
	}
	return traceIndex(t, w.pool[rnd.IntN(len(w.pool))])
}

// traceReplies is every message a command run by the trace returns, each
// named in the trace by its index, so a reply's type can move package or be
// renamed without the golden changing. A reply of a type not listed fails the
// trace: list it at the end.
var traceReplies = []tea.Msg{
	propUpdatedMsg{}, heldTick{}, mapFeedMsg{}, mapWorkedMsg{},
	uiSavedMsg{},
	committedMsg{},
	propAheadMsg{},
	propChartMsg{},
	resolvedMsg{},
}

// traceReplyName is a reply's index in traceReplies.
func traceReplyName(t *testing.T, msg tea.Msg) string {
	t.Helper()
	want := reflect.TypeOf(msg)
	for i, r := range traceReplies { // bounded by the list (P10-02)
		if reflect.TypeOf(r) == want {
			return fmt.Sprintf("r%02d", i)
		}
	}
	t.Fatalf("a command returned a %T, which traceReplies does not list: add it at the end", msg)
	return ""
}

// traceCmd describes a command without depending on where a function lives:
// "-" for none, "tick" and "every" for Bubble Tea's timers, which the trace
// does not wait out, batch(…) and seq(…) for Bubble Tea's combinators, whose
// closure only returns its list, and otherwise the reply it returns, which
// run is set to receive. A command run returns at once or fails the trace:
// the map's own ticks go through heldTicks, which answer at once.
func traceCmd(t *testing.T, cmd tea.Cmd, run func(tea.Msg), depth int) string {
	t.Helper()
	if cmd == nil {
		return "-"
	}
	if depth > 4 { // bounded nesting (P10-02)
		t.Fatal("a command nests batches more than four deep")
	}
	name := runtime.FuncForPC(reflect.ValueOf(cmd).Pointer()).Name()
	switch {
	case strings.HasPrefix(name, "charm.land/bubbletea/v2.Tick."):
		return "tick"
	case strings.HasPrefix(name, "charm.land/bubbletea/v2.Every."):
		return "every"
	case strings.HasPrefix(name, "charm.land/bubbletea/v2.compactCmds["):
		msg := cmd()
		kind := "seq"
		if _, ok := msg.(tea.BatchMsg); ok {
			kind = "batch"
		}
		list := reflect.ValueOf(msg)
		parts := make([]string, 0, list.Len())
		for i := range list.Len() { // bounded by the batch (P10-02)
			parts = append(parts, traceCmd(t, list.Index(i).Interface().(tea.Cmd), run, depth+1))
		}
		return kind + "(" + strings.Join(parts, ",") + ")"
	}
	if run == nil {
		return "fn"
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		if msg == nil {
			return "nil"
		}
		run(msg)
		return traceReplyName(t, msg)
	case <-time.After(10 * time.Second):
		t.Fatalf("a command did not return in 10 s (%s): the trace runs every command it is given", name)
	}
	return ""
}

// traceDash is the trace's Dashboard: the benchmark fixture with a real map
// (the embedded basemap and offline feeds), a Settings save that records, and
// a Propagation update that answers at once (0.19.6 A-29, RR-1). The map's
// ticks answer at once through heldTicks, unrun.
func traceDash(t *testing.T, at time.Time) Dashboard {
	t.Helper()
	d := benchDash(t, 133, 44).(Dashboard)
	d.now = func() time.Time { return at }
	d.cfg.NewMap, d.cfg.MapFeed = embeddedMap, boxFeed(-117.6, -117.1, false)
	d.cfg.SetUI = func(UIPrefs) error { return nil }
	s := &seam{answer: PropagationResult{Snapshot: stationsSnap(0, ionomaps.Fresh)}}
	d.cfg.PropagationUpdate, d.cfg.PropagationAck = s.update, propAckVersion
	d.cfg.SavePropagationAck = func(int) error { return nil }
	d.ticks = heldTicks
	return d
}

// traceReach is how many frames of a trace showed each place the alphabet
// is widened to reach, read from the frame alone so a move does not edit it.
type traceReach struct{ mapped, propagation, settings, console, tones, station int }

// TestRouterUpdateIsTheSameUnderRandomMessages is 0.19.5 Q3.1 (PF-1, RR-8) in
// 0.19.6 B0's form (design CK-6, A-12, A-29): random message sequences on
// fixed seeds run through NewRouter's Router over a Dashboard with a real map.
// Each command a message returns is described, and run when it is the app's
// own, its reply fed back through the Router; the replies' own commands are
// described, not run. After each step the trace records the message's index,
// the surface, the command and a digest of the frame, so it holds every batch
// of the split to the code of e57afeaf, which the golden records: every walk's
// lines come out the same on production code identical to it (0.19.6 D-68).
// Six walks on fixed seeds (traceWalks) start from the dashboard, the map, the
// map in Propagation, the console, and Settings on each surface (0.19.6 D-67);
// each place they reach is seen in a fixed number of frames at least, or the
// trace has stopped reaching it.
//
// About names the Go release the test binary was built with, so each frame is
// hashed with that name taken out: a toolchain update would otherwise change
// every frame About is open in, and the trace holds the router, not Go.
func TestRouterUpdateIsTheSameUnderRandomMessages(t *testing.T) {
	at := time.Date(2026, 8, 24, 1, 2, 0, 0, time.UTC)
	built := "GO " + strings.TrimPrefix(runtime.Version(), "go")
	var trace strings.Builder
	var reach traceReach
	for n, walk := range traceWalks { // the six walks (P10-02)
		seed := uint64(n + 1)
		r := NewRouter(traceDash(t, at)) // the console takes the Dashboard's clock (0.19.6 D-21)
		rnd := rand.New(rand.NewPCG(seed, 195))
		var m tea.Model = r
		for step := range 300 { // bounded (P10-02)
			i := walk.draw(t, rnd, step)
			var cmd tea.Cmd
			m, cmd = m.Update(traceMessages[i])
			var replies []tea.Msg
			desc := traceCmd(t, cmd, func(msg tea.Msg) { replies = append(replies, msg) }, 0)
			for _, reply := range replies { // bounded by the command's replies (P10-02)
				var again tea.Cmd
				m, again = m.Update(reply)
				desc += ">" + traceCmd(t, again, nil, 0)
			}
			frame := strings.ReplaceAll(m.View().Content, built, "GO")
			fmt.Fprintf(&trace, "%d.%03d m%02d surface=%v cmd=%s frame=%x\n", seed, step, i, m.(Router).active, desc, sha256.Sum256([]byte(frame)))
			reach.count(m.(Router).active, ttytest.StripANSI(frame))
		}
	}
	// THE FLOORS, the first four fixed at 0.19.6 B0 (CQ F-18) under the 343,
	// 114, 418 and 144 frames its recording showed; the tone rows' and the
	// station rows' at 0.19.6 D-68, under the 137 and 54 the six walks show.
	for _, f := range []struct {
		name     string
		got, min int
	}{{"the map", reach.mapped, 100}, {"Propagation", reach.propagation, 50}, {"Settings", reach.settings, 100}, {"the console", reach.console, 50}, {"the tone rows", reach.tones, 50}, {"the station's rows", reach.station, 20}} {
		if f.got < f.min {
			t.Errorf("the trace showed %s in %d frames, under its floor of %d; the walks no longer reach it", f.name, f.got, f.min)
		}
	}
	checkGolden(t, "router-trace.golden", trace.String())
}

// count adds one frame to the places it shows.
func (c *traceReach) count(surface Surface, frame string) {
	if surface == SurfaceBroadcaster {
		c.console++
	}
	if strings.IndexFunc(frame, func(r rune) bool { return r > 0x2800 && r <= 0x28ff }) >= 0 {
		c.mapped++
	}
	if strings.Contains(frame, "MUF(3000)") {
		c.propagation++
	}
	if strings.Contains(frame, "── Settings ──") {
		c.settings++
	}
	if strings.Contains(frame, "ALERTS - TONE") {
		c.tones++
	}
	if strings.Contains(frame, "Service radius:") {
		c.station++
	}
}
