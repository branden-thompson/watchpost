package tty

import (
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// traceMessages is the alphabet the router trace draws from: the keys a
// listener and an operator press (navigation, the windows, the player, the
// swaps, the digits, typing), resizes, and the ticks that arrive between them.
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
	tickMsg{}, vizTickMsg{}, TickerAdvanceMsg{},
}

// TestRouterUpdateIsTheSameUnderRandomMessages is 0.19.5 Q3.1 (PF-1, RR-8):
// the Router and the Observer's handlers return their own types instead of a
// tea.Model, and route every message as they did. Random message sequences on
// fixed seeds run through the router, and after each message the trace records
// the surface on screen, whether a command came back, and a digest of the
// frame. The trace was recorded on the code before the change
// (testdata/router-trace.golden), so the test holds the new code to the old.
func TestRouterUpdateIsTheSameUnderRandomMessages(t *testing.T) {
	at := time.Date(2026, 8, 24, 1, 2, 0, 0, time.UTC)
	var trace strings.Builder
	for seed := uint64(1); seed <= 3; seed++ {
		d := benchDash(t, 133, 44).(Dashboard)
		d.now = func() time.Time { return at }
		r := NewRouter(d)
		r.broadcaster.now = func() time.Time { return at }
		rnd := rand.New(rand.NewPCG(seed, 195))
		var m tea.Model = r
		for step := range 300 { // bounded (P10-02)
			msg := traceMessages[rnd.IntN(len(traceMessages))]
			next, cmd := m.Update(msg)
			m = next
			fmt.Fprintf(&trace, "%d.%03d %T%v surface=%v cmd=%v frame=%x\n", seed, step, msg, msg,
				next.(Router).active, cmd != nil, sha256.Sum256([]byte(next.View().Content)))
		}
	}
	checkGolden(t, "router-trace.golden", trace.String())
}
