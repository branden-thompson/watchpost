package tty

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"strings"
	"testing"
	"time"

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
		{ID: "tor", Category: CatWarning, Head: "Tornado Warning · the Oklahoma City area  declared 3:42 PM · expires 4:15 PM", Severity: TickerRed},
		{ID: "svr", Category: CatWarning, Head: "Severe Thunderstorm Warning · Cherry, NE  declared 3:50 PM · expires 4:30 PM", Severity: TickerOrange},
		{ID: "wat", Category: CatWatch, Head: "Tornado Watch · the Dallas area  declared 3:00 PM · expires 6:00 PM", Severity: TickerYellow},
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
	if !strings.Contains(raw, render.Tok(render.TickerWarningBG)) {
		t.Fatalf("the Warnings lane wears the orange band:\n%q", raw)
	}
}

func TestTickerLaneRotatesEvery90s(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	items := []TickerItem{
		{ID: "tor", Category: CatWarning, Head: "Tornado Warning · OKC  declared 3:42 PM"},
		{ID: "wat", Category: CatWatch, Head: "Tornado Watch · Dallas  declared 3:00 PM"},
	}
	d := tickerDash(t, items, false)
	// Starts on the Warnings lane (Orange).
	if raw := d.tickerMarquee(render.Opts{Width: 120}); !strings.Contains(raw, render.Tok(render.TickerWarningBG)) {
		t.Fatalf("starts on the Warnings lane (orange band):\n%q", raw)
	}
	d.advanceTickerCategory() // the 90s switch
	raw := d.tickerMarquee(render.Opts{Width: 120})
	// Rotates to the Watches lane (Yellow).
	if !strings.Contains(raw, render.Tok(render.TickerWatchBG)) {
		t.Fatalf("rotates to the Watches lane (yellow band):\n%q", raw)
	}
}

func TestTickerBreakingTakesOverCentredInItsLaneColour(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	// The normal tape is Watches (yellow); a breaking Warning takes it over.
	d := tickerDash(t, []TickerItem{{ID: "w", Category: CatWatch, Head: "Tornado Watch · Dallas"}}, false)
	item := TickerItem{ID: "tor", Category: CatWarning, Head: "Tornado Warning · the Oklahoma City area  declared 3:42 PM · expires 4:15 PM"}
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
	if !strings.Contains(raw, render.Tok(render.TickerWarningBG)) || strings.Contains(raw, render.Tok(render.TickerWatchBG)) {
		t.Fatalf("the band is the breaking lane's colour:\n%q", raw)
	}
	// Done resumes the normal (Watches) tape.
	d.breaking = nil
	if plain := stripANSITest(d.tickerMarquee(render.Opts{Width: 120})); !strings.Contains(plain, "Tornado Watch") {
		t.Fatalf("clearing breaking resumes normal rotation: %q", plain)
	}
}

func TestTickerSingleLaneDoesNotRotate(t *testing.T) {
	d := tickerDash(t, []TickerItem{{ID: "a", Category: CatWarning, Head: "one"}}, false)
	d.advanceTickerCategory()
	if d.tickerCatIdx != 0 {
		t.Fatalf("one lane present ⇒ no rotation, got idx=%d", d.tickerCatIdx)
	}
}

// [M] LEFT THE HEADER at 0.14.0. The six alert
// classes are separately mutable now, and one key cannot mean six things — so
// [M] opens Settings at the tone rows instead of flipping a single switch, and
// the header no longer carries a chip whose label claimed to know the state.
//
// The binding stays live, which is the half that matters to a listener who
// learnt the key.
func TestMuteDeepLinksRatherThanFlippingAHeaderChip(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	o := render.Opts{Width: 160}
	for _, muted := range []bool{false, true} {
		if got := stripANSITest(tickerDash(t, nil, muted).header(o)); strings.Contains(got, "[M] Mute") || strings.Contains(got, "[M] Unmute") {
			t.Errorf("muted=%v: the mute chip left the header for Settings:\n%q", muted, got)
		}
	}
	d := tickerDash(t, nil, false)
	m, _ := d.Update(tea.KeyPressMsg{Code: 'M', Text: "M"})
	nd := m.(Dashboard)
	if nd.modal != modalSetup {
		t.Fatalf("[M] opens Settings, got modal %v", nd.modal)
	}
	if setupTable()[nd.setup.focus].group != groupTone {
		t.Errorf("[M] lands on the tone rows, got focus %v", nd.setup.focus)
	}
}

func TestTickerTapeScrollsContinuouslyAndWraps(t *testing.T) {
	// A lane whose tape is longer than the window scrolls and loops.
	items := []TickerItem{
		{ID: "a", Category: CatWarning, Head: strings.Repeat("Tornado Warning · somewhere far away  declared 3:42 PM ", 3)},
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
		{ID: "w", Category: CatWarning, Head: "Tornado Warning · OKC"},
		{ID: "wa", Category: CatWatch, Head: "Tornado Watch · Dallas"},
	}, false)
	d.tickerCatIdx = 1 // showing the Watches lane
	// The next publish carries only Warnings (the watch expired and was dropped).
	d.setTicker([]TickerItem{{ID: "w", Category: CatWarning, Head: "Tornado Warning · OKC"}})
	if cats := d.tickerCategories(); len(cats) != 1 || cats[0] != CatWarning {
		t.Fatalf("the expired Watches lane drops out: %v", cats)
	}
	if d.tickerCatIdx != 0 {
		t.Fatalf("the lane index stays valid as lanes drop: idx=%d", d.tickerCatIdx)
	}
}

// THE TAPE REPAINTS THE MOMENT THE CLOCK CHANGES (HUM LEAD, UAT 2026-08-30:
// "there's a delay").
//
// It used to arrive from the app already formatted, so the times on it were
// written when the ticker last cycled — up to two minutes earlier. Changing
// Show Time in did nothing until the next cycle, and the band sat there in the
// old format with no way to tell it had been heard.
//
// The item carries the FACTS now and the tape is composed every frame, so the
// preference reaches it the way the theme does: immediately, because there is
// nothing older to repaint.
func TestTheTapeFollowsTheClockWithoutWaitingForACycle(t *testing.T) {
	at := time.Date(2026, 8, 30, 16, 50, 0, 0, time.Local)
	d := tickerDash(t, []TickerItem{
		{ID: "tor", Category: CatWarning, Head: "Tornado Warning · OKC", Verb: "declared", At: at},
	}, false)
	d.now = func() time.Time { return at.Add(time.Hour) } // same day: a bare time

	twelve := stripANSITest(d.tickerMarquee(d.opts()))
	if !strings.Contains(twelve, "4:50 PM") {
		t.Fatalf("the 12-hour tape reads the time: %q", twelve)
	}

	// The listener picks 24-hour. No new event, no new cycle — just the setting.
	d.clockFmt = render.Clock24
	if got := stripANSITest(d.tickerMarquee(d.opts())); !strings.Contains(got, "16:50") || strings.Contains(got, "PM") {
		t.Errorf("the tape follows the clock at once, got %q", got)
	}
	d.clockFmt = render.ClockMil
	if got := stripANSITest(d.tickerMarquee(d.opts())); !strings.Contains(got, "1650") {
		t.Errorf("military too, got %q", got)
	}
}

// THE SHOWING LANE IS AN IDENTITY, NOT A POSITION (HUM LEAD, UAT 2026-08-30:
// "it doesn't seem to be rotating between all the categories — I'm seeing
// Disasters, Marine a lot").
//
// tickerCatIdx indexes the PRESENT lanes, and the present set changes on every
// publish as alerts arrive and expire. setTicker used to keep the index valid
// with `idx %= len(cats)`, which silently teleports it whenever the set shrinks
// — and because Disasters and Marine come from the national feed they are
// almost always present AND first in the rotation order, so every shrink
// dragged the band back onto them.
//
// The rule: a lane that is still present keeps showing. Only a lane that has
// actually gone hands over, and it hands over FORWARD.
func TestTickerLaneSurvivesThePresentSetChanging(t *testing.T) {
	item := func(c TickerCategory, id string) TickerItem {
		return TickerItem{ID: id, Category: c, Head: "HEAD", Verb: "issued", At: time.Now()}
	}
	lane := func(d Dashboard) TickerCategory {
		cats := d.tickerCategories()
		if len(cats) == 0 {
			t.Fatal("no lanes present")
		}
		return cats[d.tickerCatIdx%len(cats)]
	}
	var d Dashboard
	d.setTicker([]TickerItem{item(CatDisasters, "q"), item(CatMarine, "m"), item(CatWarning, "w"), item(CatWatch, "x")})
	d.advanceTickerCategory()
	d.advanceTickerCategory()
	d.advanceTickerCategory()
	if got := lane(d); got != CatWatch {
		t.Fatalf("three rotations from Disasters should reach Watches, got %v", got)
	}
	// The Disasters lane empties. Watches is untouched and must keep showing —
	// this is the case the modulo got wrong, landing on Marine.
	d.setTicker([]TickerItem{item(CatMarine, "m"), item(CatWarning, "w"), item(CatWatch, "x")})
	if got := lane(d); got != CatWatch {
		t.Errorf("a lane that is still present must keep showing; showing %v", got)
	}
	// Now the showing lane itself empties: hand over to the next one in the
	// rotation order, not back to the front.
	d.setTicker([]TickerItem{item(CatMarine, "m"), item(CatWarning, "w"), item(CatAdvisory, "a")})
	if got := lane(d); got != CatAdvisory {
		t.Errorf("the lane after Watches is Advisories; showing %v", got)
	}
	// And past the end it wraps.
	d.setTicker([]TickerItem{item(CatDisasters, "q"), item(CatWarning, "w"), item(CatStatement, "s")})
	d.setTicker([]TickerItem{item(CatDisasters, "q"), item(CatWarning, "w")})
	if got := lane(d); got != CatDisasters {
		t.Errorf("past the last lane the rotation wraps to the first; showing %v", got)
	}
}

// Every lane gets a turn: rotating as many times as there are lanes visits each
// one exactly once, which is what "rotates through the non-empty lanes" means.
func TestTickerRotationVisitsEveryLaneOnce(t *testing.T) {
	var items []TickerItem
	for i, c := range tickerCatOrder() {
		items = append(items, TickerItem{ID: fmt.Sprint(i), Category: c, Head: "HEAD", Verb: "issued", At: time.Now()})
	}
	var d Dashboard
	d.setTicker(items)
	seen := map[TickerCategory]int{}
	for range tickerCatOrder() {
		cats := d.tickerCategories()
		seen[cats[d.tickerCatIdx%len(cats)]]++
		d.advanceTickerCategory()
	}
	for _, c := range tickerCatOrder() {
		if seen[c] != 1 {
			t.Errorf("lane %v shown %d times in a full rotation, want exactly 1", c, seen[c])
		}
	}
}

// THE BAND IS READABLE WITH THE COLOUR OFF (R-12a). Six lanes share one strip
// and only one is on screen at a time, so a reader who cannot tell the
// backgrounds apart — colourblind, a monochrome theme, NO_COLOR, a screenshot in
// black and white — has no reference to compare against and no way to know which
// hazard class they are looking at. The lane says its own name.
func TestEachTickerLaneNamesItselfWithoutColour(t *testing.T) {
	item := func(c TickerCategory) TickerItem {
		return TickerItem{ID: "x" + fmt.Sprint(c), Category: c, Head: "HEAD", Verb: "issued", At: time.Now()}
	}
	seen := map[string]TickerCategory{}
	for _, cat := range tickerCatOrder() {
		var d Dashboard
		d.width = 133
		d.setTicker([]TickerItem{item(cat)})
		band := stripANSITest(d.tickerMarquee(d.opts()))
		if label := cat.Label(); !strings.Contains(band, label) {
			t.Errorf("lane %v must name itself %q on the band:\n%s", cat, label, band)
		}
		if prev, dup := seen[cat.Label()]; dup {
			t.Errorf("lanes %v and %v share the label %q — they must be distinguishable", prev, cat, cat.Label())
		}
		seen[cat.Label()] = cat
	}
	if len(seen) != len(tickerCatOrder()) {
		t.Errorf("every lane needs its own name: %d labels for %d lanes", len(seen), len(tickerCatOrder()))
	}
}

// THE BAND'S LANES COME FROM THE REGISTRY, AND EACH IS DRESSED.
//
// The rotation used to be a list kept by hand beside the category enum, and a
// lane missing from it never reached the band however many alerts it held. It
// is derived now (F-21) — this checks the ticker's view of it holds up: every
// lane it rotates through has a name and a colour, and Forecasts is absent
// because the marquee is for what is happening (MVS-D-59).
func TestTheBandsLanesAreDressedAndExcludeForecasts(t *testing.T) {
	seen := map[TickerCategory]bool{}
	for _, c := range tickerCatOrder() {
		if seen[c] {
			t.Errorf("lane %v appears twice in the rotation", c)
		}
		seen[c] = true
		if c.Label() == "" {
			t.Errorf("lane %v has no name; colour alone is not a channel (R-12a)", c)
		}
		if tickerCatBG(c) == "" {
			t.Errorf("lane %v has no band colour", c)
		}
	}
	if seen[SevereForecasts] {
		t.Error("Forecasts must not rotate: an outlook is what might happen, not what is")
	}
	if len(seen) == 0 {
		t.Fatal("the rotation is empty")
	}
}

// THE BAND SAYS WHEN WHAT IT IS SHOWING IS FABRICATED (FR-4.4).
//
// A photograph of the marquee carrying a tornado warning is indistinguishable
// from a real one — the same sentence that keeps the injector out of a release
// build.
//
// PREPENDED AND POSTPENDED, ON THE ITEM (HUM LEAD, 2026-09-07). The tape
// scrolls, so a marker at one end only is off-window half the time; a marker at
// both ends means the item cannot be on screen without one of them. The earlier
// design put it in the band's top row as lane chrome, on the argument that an
// 18-cell prefix per item at the 80-column floor makes the marker the majority
// of the tape — the ruling overrides that argument, and it is the item that is
// fabricated rather than the lane.
func TestTheTapeMarksAFabricatedItemAtBothEnds(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	real := TickerItem{ID: "tor", Category: CatWarning, Head: "Tornado Warning · Olathe, KS", Severity: TickerRed}
	test := real
	test.ID, test.Test = "injected", true
	d := tickerDash(t, []TickerItem{test}, false)

	line := stripANSITest(d.tapeLine(render.Opts{Width: 80}, test))
	if !strings.HasPrefix(line, testEventMark) || !strings.HasSuffix(line, testEventMark) {
		t.Errorf("the tape reads %q; a fabricated item is marked at BOTH ends, so it cannot be on "+
			"screen without one of them", line)
	}
	if got := stripANSITest(d.tapeLine(render.Opts{Width: 80}, real)); strings.Contains(got, testEventMark) {
		t.Errorf("a REAL warning is marked as a test event: %q", got)
	}

	for _, width := range []int{80, 133} {
		if got := stripANSITest(d.tickerMarquee(render.Opts{Width: width})); !strings.Contains(got, testEventMark) {
			t.Errorf("%d cells: the band is showing a fabricated warning and says nothing:\n%s", width, got)
		}
		if got := stripANSITest(tickerDash(t, []TickerItem{real}, false).tickerMarquee(render.Opts{Width: width})); strings.Contains(got, testEventMark) {
			t.Errorf("%d cells: a REAL warning is marked as a test event:\n%s", width, got)
		}
	}
}

// AND SO DOES THE TAKEOVER, which is the surface that most looks like the real
// thing: one event, centred, across the whole band, in its lane's colour. It
// draws the same tape line, so it inherits the marking rather than repeating
// the rule.
func TestTheTakeoverMarksAFabricatedEvent(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	it := TickerItem{ID: "injected", Category: CatEmergency, Head: "Evacuation Immediate · Paradise, CA", Severity: TickerRed, Test: true}
	d := tickerDash(t, nil, false)
	d.breaking = &it
	if got := stripANSITest(d.tickerMarquee(render.Opts{Width: 80})); !strings.Contains(got, testEventMark) {
		t.Errorf("a fabricated evacuation order is taking over the band unmarked:\n%s", got)
	}
	real := it
	real.Test = false
	d.breaking = &real
	if got := stripANSITest(d.tickerMarquee(render.Opts{Width: 80})); strings.Contains(got, testEventMark) {
		t.Errorf("a REAL evacuation order is marked as a test event:\n%s", got)
	}
}
