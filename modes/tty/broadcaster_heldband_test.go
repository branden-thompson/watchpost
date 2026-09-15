package tty

// broadcaster_heldband_test.go — the held-hazard notice as a BAND.
//
// HUM LEAD, 2026-09-15: "Let's make that look like the ticker I almost missed
// this: 3 lines, message in the center, 3 line bkg should be the darker yellow
// (not orange, not red) use the same tint as the 'LOCAL ALERT Advisory BKG'.
// '1 HAZARD(S) HELD' - BOLD WHITE … 'ON AIR' - BOLD.  Keep the freshness rules
// (it disappeared after a few minutes) the same."

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func heldBandRows(t *testing.T, after time.Duration) []string {
	t.Helper()
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
	return bcStandby(t, true, after).heldNotice()
}

// THREE PAINTED LINES, and the bare row above them that keeps the band from
// fusing with the station section it follows.
func TestTheHeldNoticeIsAThreeLineBand(t *testing.T) {
	// THREE LINES WHEN THE MESSAGE FITS — blank, message, blank — plus the bare
	// separating row above. A rung whose sentence does not fit grows the band
	// instead of losing its ending; TestTheSeverestRungIsNeverTruncated pins
	// that, and this fixture is a rung that fits.
	rows := heldBandRows(t, 90*time.Second)
	if len(rows) != 4 {
		t.Fatalf("want a bare row plus a three-line band; got %d rows:\n%q", len(rows), rows)
	}
	bg := render.Tok(render.AlertModalAdvBG)
	if strings.Contains(rows[0], bg) {
		t.Error("the separating row is painted; it is the air between two regions and belongs to neither")
	}
	for i, r := range rows[1:] {
		if !strings.Contains(r, bg) {
			t.Errorf("band line %d is not painted on the advisory tile:\n  %q", i, r)
		}
	}
}

// THE TINT IS THE ONE NAMED — the LOCAL ALERT window's advisory tile, and
// explicitly NOT the tape's advisory lane, which is the burnt orange the ruling
// rules out ("not orange, not red").
func TestTheBandWearsTheAlertWindowsAdvisoryTileAndNotTheTapes(t *testing.T) {
	rows := heldBandRows(t, 90*time.Second)
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, render.Tok(render.AlertModalAdvBG)) {
		t.Errorf("the band does not carry %q", render.Tok(render.AlertModalAdvBG))
	}
	for _, wrong := range []render.Token{render.TickerAdvisoryBG, render.TickerEmergencyBG, render.TickerWarningBG} {
		if v := render.Tok(wrong); v != "" && strings.Contains(joined, v) {
			t.Errorf("the band is painted %s (%q), which the ruling excludes", wrong, v)
		}
	}
}

// THE MESSAGE IS CENTRED, measured against the band's own width rather than
// eyeballed from a screenshot.
func TestTheHeldMessageIsCentredInTheBand(t *testing.T) {
	rows := heldBandRows(t, 90*time.Second) // COLOUR ON: Block trims trailing
	// spaces when colour is off, which would leave every row looking
	// right-aligned and the measurement meaningless.
	var line string
	for _, r := range rows {
		if strings.Contains(stripANSITest(r), "HAZARD(S) HELD") {
			line = stripANSITest(r)
		}
	}
	if line == "" {
		t.Fatal("no row carries the message")
	}
	left := len(line) - len(strings.TrimLeft(line, " "))
	right := len(line) - len(strings.TrimRight(line, " "))
	if d := left - right; d < -2 || d > 2 {
		t.Errorf("the message is not centred: %d cells of air on the left, %d on the right", left, right)
	}
}

// THE COUNT SHOUTS, THE PROSE DOES NOT, AND THE ACTION DOES.
func TestTheBandBoldsTheCountAndTheAction(t *testing.T) {
	rows := heldBandRows(t, 6*time.Minute) // the rung whose sentence says ON AIR
	joined := strings.Join(rows, "\n")

	// NOT `Contains(render.Bold(text))`. Painting the band rewrites every inner
	// RESET to the band's own ground, so a pre-rendered `\x1b[1m…\x1b[0m` literal
	// cannot survive into the output — the first draft of this test asserted
	// exactly that and failed against a perfectly bold band. What is checked is
	// the WEIGHT IN FORCE where the text is: the nearest escape opened before it.
	if !boldedAt(joined, "1 HAZARD(S) HELD") {
		t.Errorf("the count is not bold:\n%q", joined)
	}
	if !boldedAt(joined, "ON AIR") {
		t.Errorf("the action the operator must take is not bold:\n%q", joined)
	}
	// AND THE PROSE BETWEEN THEM IS NOT. Bolding the sentence would spend the
	// emphasis the count and the action are carrying.
	if boldedAt(joined, "the station is in STANDBY") {
		t.Error("the explanatory prose is bold; the emphasis is meant to pick out two things, not the row")
	}
}

// boldedAt reports whether `text` is drawn bold: the last escape opened before
// it turns weight on, and nothing since has reset it.
func boldedAt(s, text string) bool {
	at := strings.Index(s, text)
	if at < 0 {
		return false
	}
	head := s[:at]
	open := strings.LastIndex(head, "\x1b[")
	if open < 0 {
		return false
	}
	end := strings.Index(head[open:], "m")
	if end < 0 {
		return false
	}
	params := head[open+2 : open+end]
	for _, p := range strings.Split(params, ";") {
		if p == "1" {
			return true
		}
	}
	return false
}

// THE MESSAGE IS NEVER CLIPPED, and the band grows instead.
//
// CENTRING ALONE CLIPS, which is how the first draft shipped: the !!! rung's
// sentence is ~149 cells against a 127-cell band at the HUM LEAD's width, and
// the most severe notice on the console lost its ending — "may be drop". The
// single line it replaced wrapped in the terminal, so this would have been a
// REGRESSION introduced by making the notice prettier.
func TestTheSeverestRungIsNeverTruncated(t *testing.T) {
	for _, w := range []int{100, 133, 150, 200} {
		b := bcStandby(t, true, 20*time.Minute)
		b.width = w
		// Collapsed, for the reason clip_ansi_test.go's sweep records: a wrap
		// falling between the last two words separates them with centring
		// padding, which is not truncation.
		got := strings.Join(strings.Fields(stripANSITest(strings.Join(b.heldNotice(), " "))), " ")
		if !strings.Contains(got, "dropped unread") {
			t.Errorf("width %d: the severest rung is cut short:\n%q", w, got)
		}
	}
}

// AND THE FRESHNESS RULES ARE UNTOUCHED — the ruling said so in as many words.
// These repeat what the standby tests already pin, THROUGH THE BAND, because
// the band is what changed.
func TestTheBandKeepsTheFreshnessRules(t *testing.T) {
	if rows := bcStandby(t, false, 30*time.Minute).heldNotice(); rows != nil {
		t.Errorf("an empty rail raised a band: %q", rows)
	}
	early := stripANSITest(strings.Join(bcStandby(t, true, 30*time.Second).heldNotice(), "\n"))
	late := stripANSITest(strings.Join(bcStandby(t, true, 20*time.Minute).heldNotice(), "\n"))
	if early == late {
		t.Error("the band no longer escalates with time")
	}
}
