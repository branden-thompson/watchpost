package tty

// console_fixtures_test.go — the console as the root's tests build it: through
// New, the messages the Router sends it and its pointer's ScrollQueue, never
// its fields, so a root test reaches the console as the Router does (0.19.6
// D-14, D-22; design CK-5).

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// consoleW and consoleH are the size the root's console fixtures draw at.
const consoleW, consoleH = 150, 74

// consoleOf is a console opened on o and sent a line-up, sized by the resize
// every surface is sent.
func consoleOf(t *testing.T, o Opening, l lineup.Lineup) Broadcaster {
	t.Helper()
	b := New(o)
	b, _ = b.Update(LineupMsg{Lineup: l})
	b, _ = b.Update(tea.WindowSizeMsg{Width: consoleW, Height: consoleH})
	return b
}

// mainTrackOf is a line-up holding cards on its main track, in order.
func mainTrackOf(t *testing.T, cards ...lineup.Card) lineup.Lineup {
	t.Helper()
	var l lineup.Lineup
	for _, c := range cards {
		next, err := l.Queue(lineup.MainTrack, c)
		if err != nil {
			t.Fatalf("seeding the lineup: %v", err)
		}
		l = next
	}
	return l
}

// loadedConsoleOf is ttytest's loaded station of n places sent to a console
// opened on o: the console with everything on it (D-120).
func loadedConsoleOf(t testing.TB, o Opening, n int) Broadcaster {
	t.Helper()
	s := ttytest.Loaded(t, n, time.Now())
	b := New(o)
	b, _ = b.Update(tea.WindowSizeMsg{Width: consoleW, Height: consoleH})
	b, _ = b.Update(LineupMsg{Lineup: s.Lineup})
	b, _ = b.Update(StationAreaMsg{Transmitter: s.Transmitter, RadiusMi: s.RadiusMi, Pool: s.Pool})
	b, _ = b.WithRecent(s.Recent)
	return b
}

// pointAt puts the console's pointer on row at of the two tables it walks, the
// running order and then the pool, as the arrows do: to the top, then down.
func pointAt(t *testing.T, b Broadcaster, at int) Broadcaster {
	t.Helper()
	b = b.ScrollQueue(-MainTrackSlots - len(b.Pool())).ScrollQueue(at)
	got := b.LineupSelection()
	if got < 0 {
		got = MainTrackSlots - ScheduledFrom + b.PoolSelection()
	}
	if got != at {
		t.Fatalf("the pointer stops at row %d, not %d", got, at)
	}
	return b
}

// drawn is the console's frame as the operator reads it, ANSI stripped.
func drawn(b Broadcaster) string { return ttytest.StripANSI(b.View().Content) }

// drawnRows is how many rows a frame has.
func drawnRows(frame string) int { return strings.Count(frame, "\n") + 1 }

// gainDrawn reads the level the console's gain control draws, the number after
// its bar.
var gainDrawn = regexp.MustCompile(`GAIN .*?(\d+)\s`)

// drawnGain is the level the console's gain control draws (0.19.6 D-22).
func drawnGain(t *testing.T, b Broadcaster) int {
	t.Helper()
	m := gainDrawn.FindStringSubmatch(drawn(b))
	if m == nil {
		t.Fatal("the console draws no gain control")
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatal(err)
	}
	return n
}
