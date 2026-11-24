package console

// broadcaster_golden_test.go — the console's whole frames, pinned (0.19.5 Q2.9).

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// goldenConsole is the loaded console with an operator's card in its running
// order and a takeover on its rail, at a size, on standby or on the air.
func goldenConsole(t *testing.T, w, h int, onAir, colour bool) Broadcaster {
	t.Helper()
	rendering.SetColorEnabledForTest(colour)
	b := loadedConsoleAt(t, ttytest.LoadedPoolSize, ttytest.GoldenNow().Add(-10*time.Minute))
	op, err := lineup.Propose(lineup.Card{ID: "operator", Slot: lineup.LocationReport, Origin: lineup.FromOperator,
		Subject: "Fallbrook, CA", Headline: "Fallbrook, CA", State: lineup.Proposed})
	if err != nil {
		t.Fatal(err)
	}
	if op, err = op.To(lineup.Admitted); err != nil {
		t.Fatal(err)
	}
	l, err := b.lineup.Insert(lineup.MainTrack, op, 2)
	if err != nil {
		t.Fatalf("placing the operator's card: %v", err)
	}
	b, _ = b.Update(LineupMsg{Lineup: l})
	b = withBurst(t, b, "Tornado Warning", "Flash Flood Warning")
	b.width, b.height, b.ascii = w, h, false
	b.now = func() time.Time { return ttytest.GoldenNow() }
	if onAir {
		b.power = lineup.Running
	}
	return b
}

// TestConsoleGoldens pins the console on standby and on the air, at its floor,
// the mock's size and the beat pin's, with colour off and on.
func TestConsoleGoldens(t *testing.T) {
	for _, size := range [][2]int{{100, 44}, {133, 44}, {160, 74}} {
		for _, onAir := range []bool{false, true} {
			for _, colour := range []bool{false, true} {
				name := fmt.Sprintf("console-%dx%d-%s-%s.golden", size[0], size[1],
					map[bool]string{false: "standby", true: "onair"}[onAir], map[bool]string{false: "plain", true: "colour"}[colour])
				ttytest.CheckGolden(t, filepath.Join("testdata", name), goldenConsole(t, size[0], size[1], onAir, colour).View().Content, *updateGolden)
			}
		}
	}
}

// TestTheConsoleGoldenDrawsADataPull pins the UP NEXT box over a composed card,
// so its DATA PULL line carries the card's stamp and age, which the twelve
// goldens above draw empty (0.19.6 A-33). At the beat pin's size, on standby,
// with colour off and on.
func TestTheConsoleGoldenDrawsADataPull(t *testing.T) {
	for _, colour := range []bool{false, true} {
		rendering.SetColorEnabledForTest(colour)
		b := New(Opening{Now: func() time.Time { return ttytest.DetailAt().Add(2 * time.Minute) }})
		b, _ = b.Update(LineupMsg{Lineup: ttytest.Seed(t, ttytest.OneCard(t))})
		b.width, b.height = 160, 74
		got := b.View().Content
		// THE PREMISE: the line is drawn, and drawn with a stamp.
		if !strings.Contains(ttytest.StripANSI(got), "DATA PULL: 09/11/26") {
			t.Fatal("the UP NEXT box draws no stamped DATA PULL line; this golden would pin the empty one again")
		}
		name := "console-160x74-standby-datapull-" + map[bool]string{false: "plain", true: "colour"}[colour] + ".golden"
		ttytest.CheckGolden(t, filepath.Join("testdata", name), got, *updateGolden)
	}
	rendering.SetColorEnabledForTest(false)
}
