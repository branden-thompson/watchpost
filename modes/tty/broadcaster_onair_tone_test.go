package tty

// broadcaster_onair_tone_test.go — the station's own row, and the box beside it.
//
// HUM LEAD, 2026-09-15: "When Broadcaster is Actively broadcasting, let's make
// this string *** ON AIR · BROADCASTING *** BOLD WHITE … and we'll make the
// cell background color of the UP NEXT box the same 'blue' token color used as
// the bkg for 'DIRECTION' and 'TODAY' column."

import (
	"regexp"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func onAirRow(t *testing.T, b Broadcaster) string {
	t.Helper()
	for _, r := range b.stationLine() {
		if strings.Contains(stripANSITest(r), "ON AIR "+b.opts().Glyphs().Dot) {
			return r
		}
	}
	return ""
}

func TestTheOnAirStateIsBoldWhileBroadcasting(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	b := broadcasterWithOneCard(t)
	b.power = lineup.Running
	row := onAirRow(t, b)
	if row == "" {
		t.Fatal("the running station draws no ON AIR · BROADCASTING row at all")
	}
	if !strings.Contains(row, "\x1b[1m") {
		t.Errorf("the on-air state is not bold:\n  %q", row)
	}
	if !strings.Contains(row, render.Tint("", render.Tok(render.AlertModalText))[:len("\x1b[97m")]) {
		t.Errorf("the on-air state does not carry the section's white:\n  %q", row)
	}
}

// AND THE OTHER TWO STATES ARE NOT SHOUTED. Bolding every state would spend the
// emphasis on the one thing it is meant to single out — the station is LIVE.
func TestOnlyTheLiveStateIsBold(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	for _, p := range []lineup.Power{lineup.Stopped, lineup.OffAir} {
		b := broadcasterWithOneCard(t)
		b.power = p
		for _, r := range b.stationLine() {
			plain := stripANSITest(r)
			if !strings.HasPrefix(strings.TrimSpace(plain), "STATION AIR:") {
				continue
			}
			// The GAIN control on this row carries bold of its own; the STATE is
			// the part before it.
			state := strings.SplitN(r, "GAIN", 2)[0]
			if strings.Contains(state, "\x1b[1m") {
				t.Errorf("power %v: a station that is not on the air is drawn bold:\n  %q", p, state)
			}
		}
	}
}

var bgRe = regexp.MustCompile(`48;2;\d+;\d+;\d+`)

// THE BOX WEARS THE BANDS' BLUE — compared against the CONSOLE'S OWN
// `D I R E C T I O N` header rather than against a token name, because the
// ruling is that the two match. A test naming the token would still pass on the
// day the table moved to a different one.
func TestTheUpNextBoxWearsTheSameBlueAsTheDirectionBand(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	b := broadcasterWithOneCard(t)
	b.width, b.height = 200, 74

	var direction string
	for _, r := range strings.Split(b.View().Content, "\n") {
		if strings.Contains(stripANSITest(r), "D I R E C T I O N") {
			direction = r
			break
		}
	}
	if direction == "" {
		t.Skip("this fixture draws no DIRECTION band; the comparison needs one")
	}
	// THE GROUND IMMEDIATELY BEFORE THE WORD, not the first one on the row.
	// That row carries several bands side by side, so `FindString` returned the
	// LEFTMOST — a grey cell three bands away — and the test reported a
	// mismatch that was entirely its own. The escape that paints a run of text
	// is the last one opened before it.
	head := direction[:strings.Index(direction, "D I R E C T I O N")]
	all := bgRe.FindAllString(head, -1)
	if len(all) == 0 {
		t.Fatalf("the DIRECTION band carries no background at all:\n  %q", direction)
	}
	want := all[len(all)-1]
	// THE LABEL CELL, NOT THE BOX (D-136). The report beside it keeps the modal
	// tone; it was painting BOTH that the HUM LEAD corrected with a diagram.
	// The cell is the row carrying the caption, which is the only row whose
	// label segment has text in it.
	var cell string
	for _, r := range b.upNextBox() {
		if strings.Contains(stripANSITest(r), bcUpNextLabel) {
			cell = r
			break
		}
	}
	if cell == "" {
		t.Fatalf("the box drew no %q cell", bcUpNextLabel)
	}
	got := ""
	for _, bg := range bgRe.FindAllString(cell, -1) {
		if bg == want {
			got = bg
		}
	}
	if got != want {
		t.Errorf("the UP NEXT label cell carries %v; DIRECTION is painted %q — the ruling is that they are the SAME blue",
			bgRe.FindAllString(cell, -1), want)
	}
}
