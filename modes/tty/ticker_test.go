package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func tickerDash(t *testing.T, items []TickerItem, muted bool) Dashboard {
	d := dash(t).(Dashboard)
	d.ticker, d.tickerMuted = items, muted
	return d
}

func TestTickerEmptyIsAPersistentMutedBand(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	d := tickerDash(t, nil, false)
	band := d.tickerMarquee(render.Opts{Width: 100})
	rows := strings.Split(stripANSITest(band), "\n")
	if len(rows) != 3 {
		t.Fatalf("the ticker is a persistent 3-row band, got %d rows", len(rows))
	}
	if !strings.Contains(rows[1], "no active severe events") {
		t.Fatalf("the content row states the empty state:\n%q", rows[1])
	}
	for i, r := range rows { // every row spans the full width (no jitter)
		if render.Width(r) != 100 {
			t.Fatalf("row %d spans %d cells, want 100", i, render.Width(r))
		}
	}
}

func TestTickerTapesTheCurrentLaneWithCategoryColour(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	items := []TickerItem{
		{ID: "tor", Category: CatWarning, Text: "Tornado Warning · the Oklahoma City area  declared 3:42 PM · expires 4:15 PM", Severity: TickerRed},
		{ID: "svr", Category: CatWarning, Text: "Severe Thunderstorm Warning · Cherry, NE  declared 3:50 PM · expires 4:30 PM", Severity: TickerOrange},
		{ID: "wat", Category: CatWatch, Text: "Tornado Watch · the Dallas area  declared 3:00 PM · expires 6:00 PM", Severity: TickerYellow},
	}
	d := tickerDash(t, items, false)
	raw := d.tickerMarquee(render.Opts{Width: 200})
	plain := stripANSITest(raw)
	// Lane 0 in rotation order (Quake, Tropical, Warning, Watch) is Warnings.
	// The left indicator is [count] [glyph]; the lane is read by its colour.
	if !strings.Contains(plain, "2 ⚠") {
		t.Fatalf("the left indicator is the lane's count then the glyph: %q", plain)
	}
	if !strings.Contains(plain, "Tornado Warning") || !strings.Contains(plain, "•") {
		t.Fatalf("the lane's alerts ticker-tape, •-separated: %q", plain)
	}
	if strings.Contains(plain, "Tornado Watch") {
		t.Fatalf("the Watch (a different lane) must not appear in the Warnings tape: %q", plain)
	}
	// Warnings lane = ORANGE background (fixed per category, HUM LEAD colour pass).
	if !strings.Contains(raw, render.Tok(render.TickerOrangeBG)) {
		t.Fatalf("the Warnings lane wears the orange band:\n%q", raw)
	}
}

func TestTickerLaneRotatesEvery90s(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	items := []TickerItem{
		{ID: "tor", Category: CatWarning, Text: "Tornado Warning · OKC  declared 3:42 PM"},
		{ID: "wat", Category: CatWatch, Text: "Tornado Watch · Dallas  declared 3:00 PM"},
	}
	d := tickerDash(t, items, false)
	// Starts on the Warnings lane (Orange).
	if raw := d.tickerMarquee(render.Opts{Width: 120}); !strings.Contains(raw, render.Tok(render.TickerOrangeBG)) {
		t.Fatalf("starts on the Warnings lane (orange band):\n%q", raw)
	}
	d.advanceTickerCategory() // the 90s switch
	raw := d.tickerMarquee(render.Opts{Width: 120})
	// Rotates to the Watches lane (Yellow).
	if !strings.Contains(raw, render.Tok(render.TickerYellowBG)) {
		t.Fatalf("rotates to the Watches lane (yellow band):\n%q", raw)
	}
}

func TestTickerBreakingTakesOverCentredInItsLaneColour(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	// The normal tape is Watches (yellow); a breaking Warning takes it over.
	d := tickerDash(t, []TickerItem{{ID: "w", Category: CatWatch, Text: "Tornado Watch · Dallas"}}, false)
	item := TickerItem{ID: "tor", Category: CatWarning, Text: "Tornado Warning · the Oklahoma City area  declared 3:42 PM · expires 4:15 PM"}
	d.breaking = &item
	raw := d.tickerMarquee(render.Opts{Width: 120})
	plain := stripANSITest(raw)
	if !strings.Contains(plain, "Tornado Warning") || strings.Contains(plain, "Tornado Watch") {
		t.Fatalf("the breaking event alone takes the band: %q", plain)
	}
	// Centred: leading whitespace before the text (not the left [count] indicator).
	rows := strings.Split(plain, "\n")
	if strings.HasPrefix(strings.TrimRight(rows[1], " "), "Tornado") {
		t.Fatalf("the breaking event is centred, not left-aligned: %q", rows[1])
	}
	// The band wears the breaking event's lane colour (Warnings = orange), not the Watch yellow.
	if !strings.Contains(raw, render.Tok(render.TickerOrangeBG)) || strings.Contains(raw, render.Tok(render.TickerYellowBG)) {
		t.Fatalf("the band is the breaking lane's colour:\n%q", raw)
	}
	// Done resumes the normal (Watches) tape.
	d.breaking = nil
	if plain := stripANSITest(d.tickerMarquee(render.Opts{Width: 120})); !strings.Contains(plain, "Tornado Watch") {
		t.Fatalf("clearing breaking resumes normal rotation: %q", plain)
	}
}

func TestTickerSingleLaneDoesNotRotate(t *testing.T) {
	d := tickerDash(t, []TickerItem{{ID: "a", Category: CatWarning, Text: "one"}}, false)
	d.advanceTickerCategory()
	if d.tickerCatIdx != 0 {
		t.Fatalf("one lane present ⇒ no rotation, got idx=%d", d.tickerCatIdx)
	}
}

func TestMuteControlIsInTheHeaderAndFlips(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	o := render.Opts{Width: 160}
	if got := stripANSITest(tickerDash(t, nil, false).header(o)); !strings.Contains(got, "[M] Mute Severe Alerts") {
		t.Fatalf("the header controls carry [M] Mute Severe Alerts after [t] Theme:\n%q", got)
	}
	if got := stripANSITest(tickerDash(t, nil, true).header(o)); !strings.Contains(got, "[M] Unmute Severe Alerts") {
		t.Fatalf("muted flips the header label to Unmute:\n%q", got)
	}
}

func TestTickerTapeScrollsContinuouslyAndWraps(t *testing.T) {
	// A lane whose tape is longer than the window scrolls and loops.
	items := []TickerItem{
		{ID: "a", Category: CatWarning, Text: strings.Repeat("Tornado Warning · somewhere far away  declared 3:42 PM ", 3)},
	}
	d := tickerDash(t, items, false)
	d.tickerScroll = 0
	loop := d.tickerLoopLen()
	if loop == 0 {
		t.Fatal("a non-empty lane has a loop length")
	}
	for i := 0; i < loop; i++ {
		d.advanceTicker()
	}
	if d.tickerScroll != 0 {
		t.Fatalf("the tape wraps after one full loop: scroll=%d (loop=%d)", d.tickerScroll, loop)
	}
}

func TestExpiredLaneDropsFromRotation(t *testing.T) {
	d := tickerDash(t, []TickerItem{
		{ID: "w", Category: CatWarning, Text: "Tornado Warning · OKC"},
		{ID: "wa", Category: CatWatch, Text: "Tornado Watch · Dallas"},
	}, false)
	d.tickerCatIdx = 1 // showing the Watches lane
	// The next publish carries only Warnings (the watch expired and was dropped).
	d.setTicker([]TickerItem{{ID: "w", Category: CatWarning, Text: "Tornado Warning · OKC"}})
	if cats := d.tickerCategories(); len(cats) != 1 || cats[0] != CatWarning {
		t.Fatalf("the expired Watches lane drops out: %v", cats)
	}
	if d.tickerCatIdx != 0 {
		t.Fatalf("the lane index stays valid as lanes drop: idx=%d", d.tickerCatIdx)
	}
}
