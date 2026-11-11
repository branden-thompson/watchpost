package tty

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// The goldens in this file are 0.19.5 Q2.9 and Q2.10 (RN-11): whole frames for
// the surfaces that had none, the console, Details, Help, About, Status and
// Diagnostics, and the radio playing, captured before the pass changes the
// code that draws them.

// goldenNow is the instant every golden here is drawn at.
var goldenNow = time.Date(2026, 8, 24, 1, 2, 0, 0, time.UTC)

// goldenConsole is the loaded console with an operator's card in its running
// order and a takeover on its rail, at a size, on standby or on the air.
func goldenConsole(t *testing.T, w, h int, onAir, colour bool) Broadcaster {
	t.Helper()
	rendering.SetColorEnabledForTest(colour)
	b := loadedConsoleAt(t, loadedPoolSize, goldenNow.Add(-10*time.Minute))
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
	b.now = func() time.Time { return goldenNow }
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
				checkGolden(t, name, goldenConsole(t, size[0], size[1], onAir, colour).View().Content)
			}
		}
	}
}

// goldenWindow is the Observer's golden fixture with one window open.
func goldenWindow(t *testing.T, w, h int, ascii bool, m modal) Dashboard {
	t.Helper()
	d := goldenDash(t, ascii)
	d.width, d.height = w, h
	return d.toggle(m)
}

// TestObserverWindowGoldens pins Details, Help, About, Status and Diagnostics
// at the mock's size and the floor, plain and under --ascii. The About pane
// names the Go it was built with, which is masked so a toolchain bump is not a
// change of frame.
func TestObserverWindowGoldens(t *testing.T) {
	for _, w := range []struct {
		name string
		m    modal
	}{
		{"details", modalDetails}, {"help", modalHelp}, {"about", modalAbout},
		{"status", modalStatus}, {"diagnostics", modalDebug},
	} {
		for _, size := range [][2]int{{133, 44}, {80, 24}} {
			for _, ascii := range []bool{false, true} {
				got := goldenWindow(t, size[0], size[1], ascii, w.m).View().Content
				got = strings.ReplaceAll(got, strings.TrimPrefix(runtime.Version(), "go"), "<go>")
				name := fmt.Sprintf("%s-%dx%d-%s.golden", w.name, size[0], size[1], map[bool]string{false: "plain", true: "ascii"}[ascii])
				checkGolden(t, name, got)
			}
		}
	}
}

// TestRadioPlayingGoldens pins the frame while the radio plays with the
// visualiser on, compact at 133x44 and full at 133x70, colour off and on.
func TestRadioPlayingGoldens(t *testing.T) {
	for _, h := range []int{44, 70} {
		for _, colour := range []bool{false, true} {
			d := playingDash(t, 133, h)
			rendering.SetColorEnabledForTest(colour)
			name := fmt.Sprintf("radio-133x%d-%s.golden", h, map[bool]string{false: "plain", true: "colour"}[colour])
			checkGolden(t, name, d.View().Content)
		}
	}
}

// playingDash is the bench dashboard with the radio playing a relay and the
// visualiser drawing fixed bars, at the golden instant.
func playingDash(t testing.TB, w, h int) Dashboard {
	t.Helper()
	d := benchDash(t, w, h).(Dashboard)
	bands := make([]float64, 32)
	for i := range bands {
		bands[i] = float64(i%7) / 7
	}
	d = d.WithSpectrum(func() []float64 { return bands })
	d.radioViz = true
	m, _ := d.Update(RadioStatusMsg{State: "playing", Station: "KEC49 Monterey CA 162.550 MHz · 78 mi (nearest relayed)",
		Detail: "The National Weather Service in San Diego has issued a heat advisory for the coastal valleys until 8 PM Friday",
		Volume: 70, Spoken: 20 * time.Second})
	d = m.(Dashboard)
	d.vizBands = bands
	d.now = func() time.Time { return goldenNow }
	return d
}
