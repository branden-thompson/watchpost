package tty

// broadcaster.go — the operator console (P1, FR-2).
//
// THE SECOND SURFACE. Observer is built for a person watching the weather;
// Broadcaster is built for a person putting it on the air. It reads the
// schedule the Director PUBLISHES and never keeps a second copy — a console
// that painted its own idea of the running order would be asserting an order
// the schedule does not follow, which is the release's UI-integrity risk.
//
// P1 IS THE SHELL. The lanes, the breakpoints and the size notice arrive here;
// the station state is P2 and operator control is P4.

import (
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/plaintext"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
	"github.com/branden-thompson/watchpost/third_party/go-studs/components"
)

// LineupMsg carries the schedule the Director PUBLISHED.
//
// IT CARRIES THE LINEUP BY VALUE, as the effect that produces it does, and for
// the same reason: the pump dispatches asynchronously, so a console that
// fetched the current lineup when the message arrived would get whatever it
// had become by then, not what was published.
type LineupMsg struct{ Lineup lineup.Lineup }

// StationMsg carries the station's power as the DIRECTOR holds it (FR-5.1).
//
// THE CONSOLE KEEPS NO OPINION OF ITS OWN. A local "am I on air" flag could
// drift from the audio path, and the swap gate depends on this value — so a
// second carrier would be a safety bug rather than a display one.
type StationMsg struct{ Power lineup.Power }

// BedMsg carries what the broadcast is riding on (F-79, closed at D-78).
//
// THE CONSOLE DREW A CONSTANT BEFORE THIS. `(no relay tuned)` and `○ INACTIVE`
// were the true things it could say while the schedule published the lineup and
// the power and nothing about the bed — so the one region D-62 built to answer
// "what is on the air" was answering two thirds of it.
type BedMsg struct {
	// Relay is how the bed's source reads on the row — a call sign, its
	// frequency and how far out it is. Empty is "nothing is tuned".
	Relay string

	// Carrying is whether the bed holds the programme right now.
	Carrying bool
}

// StationAreaMsg carries WHERE the station transmits from and how far it reaches
// (D-72).
//
// SEPARATE FROM THE LISTENER'S DEFAULT LOCATION, which is the ruling: "Default
// Location no longer = Transmitter Location — this is Broadcaster epicenter from
// which the service radius fence radiates from." The console drew both as
// placeholder constants until now.
//
// A MESSAGE, NOT A CONFIG FIELD, because it MOVES: a station borrowing the
// listener's default location follows it, and the radius is an editable setting.
// A value read once at construction would be right until the first change.
type StationAreaMsg struct {
	Transmitter snapshot.LocationRef
	RadiusMi    float64
}

// MainTrackSlots is how many cards the rolling main-track view shows (FR-3.1).
//
// EXPORTED SO IT IS ONE NUMBER, NOT TWO (D-40). The Director fills the line-up
// to its `Settings.Depth` and the console draws this many; if the two ever
// disagree the station either holds cards nobody can address, or leaves slots
// empty for ever. `app` sets the depth from here rather than from a second
// constant that agrees with it today.
const MainTrackSlots = 10

// Broadcaster is the operator console's model.
type Broadcaster struct {
	width, height int
	darkBG        bool

	// ascii is the --ascii mode: box-drawing and symbol glyphs give way to
	// forms a terminal without them can draw.
	ascii bool

	// power is the station's state as the Director holds it. Never set from
	// a keypress directly — the console asks for a change and reads back what
	// the Director decided.
	power lineup.Power

	// now is the console's clock, injectable so a test can state the schedule
	// instead of waiting for it.
	now func() time.Time

	// standbySince is when the station last went silent. Zero while it is not.
	standbySince time.Time

	// frame is the shimmer's animation phase, and tickArmed keeps exactly one
	// tick in flight (D-64). ITS OWN, NOT THE DASHBOARD'S: Observer arms its
	// tick only while IT needs one, and the console needs one whenever a slot
	// is still waiting on the Director — two different predicates, so a shared
	// arm would leave whichever surface asked second without an animation.
	frame     int
	tickArmed bool

	// version is the build's, inherited from the Dashboard the Router was
	// built over — the same reason `ascii` is (NewRouter): two surfaces
	// disagreeing about which build this is would be one fact with two
	// carriers.
	version string

	// gain is the station's output level, 0-100 — Observer's VOL under the
	// station's own word for it (HUM LEAD, 2026-09-10). The Dashboard OWNS it;
	// the Router mirrors it here each update, so the two surfaces cannot
	// disagree about how loud the station is.
	gain int

	// bed is what the broadcast is riding on (F-79). Published, never guessed:
	// the console draws it and holds no opinion of its own, the same rule the
	// power follows.
	bed BedMsg

	// area is where the station transmits from and how far it reaches (D-72),
	// published by the schedule's own owner rather than guessed at here.
	area StationAreaMsg

	// statusNote is what the STATION SECTION says on its third row instead of the
	// power's own words — today, why a swap was refused.
	//
	// MIRRORED FROM THE ROUTER, NEVER OWNED HERE, for `gain`'s reason: the
	// Router is where a swap is decided, so the Router is the one thing that
	// knows a swap was refused. The console DRAWS it, because the console is
	// what the operator is looking at when the refusal happens.
	//
	// THE ROW THE REFERENCE RESERVES: "<this then becomes a status message of
	// something related to the broadcast bar>" (HUM LEAD's mock). That is
	// exactly what this is.
	statusNote string

	// snap is the last published snapshot, for the masthead's `Updated:` stamp
	// and its API summary.
	//
	// THE CONSOLE IS TOLD EVEN WHILE IT IS NOT ON SCREEN, which is the rule
	// `consoleScoped` already states for the schedule: "a surface that only
	// learns things while on screen is stale the instant it is swapped to."
	snap *snapshot.Snapshot

	// lineup is the last PUBLISHED schedule. It is never mutated here — the
	// console names an intent and the Director owns the order (D-23).
	lineup lineup.Lineup
}

// NewBroadcaster builds the console.
func NewBroadcaster() Broadcaster { return Broadcaster{} }

// clock is the console's time, real unless a test injected one.
func (b Broadcaster) clock() time.Time {
	if b.now != nil {
		return b.now()
	}
	return time.Now()
}

// heldNotice is NFR-7's bound: a silent station holding a hazard says so, and
// says it louder the longer it holds.
//
// WHY THIS EXISTS AT ALL. STANDBY holds EVERY track including the alert rail,
// and that is correct — it is what stops a station on standby putting a
// tornado warning to air. The cost is that the hazard is neither broadcast
// nor visible, and the operator is the only person who can end that state.
//
// IT IS SILENT WHEN THE RAIL IS EMPTY, deliberately. A silent station holding
// nothing is a station at rest; warning about it would train the operator to
// ignore the notice that matters.
func (b Broadcaster) heldNotice() []string {
	if b.power != lineup.OffAir {
		return nil
	}
	held := len(b.lineup.Cards(lineup.AlertRail))
	if held == 0 {
		return nil
	}
	for _, s := range heldEscalation {
		if b.standbySince.IsZero() || b.clock().Sub(b.standbySince) >= s.after {
			return []string{"", s.mark + "  " + strconv.Itoa(held) + " HAZARD(S) HELD — the station is in STANDBY and nothing is going to air. " + s.say}
		}
	}
	return nil
}

// heldEscalation is the ladder, LONGEST FIRST so the walk returns the most
// severe rung that has been reached.
//
// The rungs are a PARAMETER, written down the way a threshold is meant to be
// (INST-1 scopes its rule to the SET being iterated, not to the numbers).
var heldEscalation = []struct {
	after time.Duration
	mark  string
	say   string
}{
	{15 * time.Minute, "!!!", "They have been held past the staleness bound and may be dropped unread."},
	{5 * time.Minute, "!!", "Go ON AIR or stand the station down."},
	{0, "!", "Go ON AIR to read them."},
}

func (b Broadcaster) Init() tea.Cmd { return nil }

// Update takes the surface's own messages. The program-scoped ones arrive
// through the Router's fan-out, which is why they are handled here as well as
// in Observer: BOTH surfaces must know the size, including while inactive.
func (b Broadcaster) Update(msg tea.Msg) (Broadcaster, tea.Cmd) {
	// THE SNAPSHOT REACHES THE CONSOLE TOO (D-59). The masthead's `Updated:`
	// stamp and its API summary come from it, and both surfaces draw the same
	// masthead — so both are told, whichever one is on screen.
	if v, ok := msg.(SnapshotMsg); ok && v.Snap != nil {
		b.snap = v.Snap
		return b.armTick(nil)
	}
	if _, ok := msg.(tickMsg); ok {
		b.tickArmed = false
		b.frame++
		return b.armTick(nil)
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		b.width, b.height = v.Width, v.Height
	case tea.BackgroundColorMsg:
		b.darkBG = v.IsDark()
	case LineupMsg:
		b.lineup = v.Lineup
	case StationAreaMsg:
		b.area = v
	case BedMsg:
		b.bed = v
	case StationMsg:
		// THE CLOCK STARTS ON THE TRANSITION, not on every message: a station
		// that has been silent an hour must not look freshly quiet because
		// another message arrived.
		if v.Power != b.power {
			b.standbySince = time.Time{}
			if v.Power == lineup.OffAir {
				b.standbySince = b.clock()
			}
		}
		b.power = v.Power
	}
	return b, nil
}

// View renders the console: the priority track, the main track, the bed.
//
// STRUCTURE, NOT YET THE MOCK'S PIXELS. P1 is the shell — it renders the three
// lanes in the mock's own vocabulary (the handles, the badges, the lane names)
// from the PUBLISHED schedule. Byte-exact fidelity to
// `01-objectives/mock-broadcaster-v1.txt` is NOT claimed here and is not
// pretended: the mock is 150x74 with a column budget this does not yet honour,
// and layout is the HUM LEAD's pass.
func (b Broadcaster) View() tea.View {
	lines := b.lanes()
	if b.tooSmall() {
		lines = b.notice()
	}
	v := tea.NewView(b.clamp(lines))
	v.AltScreen = true
	v.BackgroundColor = render.WindowBG(b.darkBG)
	return v
}

// opts is the console's render options — one owner, so a glyph decision is
// made in one place rather than at every call site.
func (b Broadcaster) opts() render.Opts {
	return render.Opts{Width: b.width, ASCII: b.ascii, Frame: b.frame}
}

// minSize is the floor below which the console refuses to draw (FR-7.3).
//
// 44 LINES IS MEASURED, NOT CHOSEN: the fixed chrome plus one readable card,
// counted off the mock at DISCOVER wave 1. The mock draws 74, so 30 of its
// lines exist to show MORE cards, not to work at all.
func (b Broadcaster) minSize() (cols, rows int) { return bcMinCols, bcMinRows }

const (
	// bcMinRows is MEASURED: fixed chrome plus one readable card, counted off
	// the mock at wave 1. The mock draws 74, so 30 of its lines show MORE
	// cards rather than making it work.
	//
	// KEPT AT 44 AGAINST THE RULING'S EXAMPLE, WHICH SAID "100 x 25". The 100
	// is the ruled breakpoint and is applied above; the 25 came inside an
	// example of the MESSAGE ("or something like this"), and 44 is the measured
	// number. Lowering it would let the console draw a frame the terminal
	// cannot hold and clamp the remainder away — which is the F-55 defect this
	// floor exists to prevent, arriving through the notice meant to prevent it.
	// Flagged to the HUM LEAD rather than silently chosen.
	bcMinRows = 44

	// bcMinCols is the start of the platform's COMPACT class (D-50). Below it
	// the console has no honest layout — and D-13 chose the platform vocabulary
	// precisely so this number is a boundary someone already reasoned about
	// rather than one invented here.
	//
	// RAISED FROM 80 BY THE HUM LEAD'S RULING: "< 100 col : Not supported — we
	// adopt a 'btop' style". It is `term.BreakUnsupported`'s own boundary, so
	// the console and the classifier cannot drift apart.
	bcMinCols = 100
)

// sectionWidth is how much room a SECTION'S TEXT has: the lane, less the frame's
// two walls and the inset the reference leaves inside them.
//
// COUNTED BEFORE THE ROWS ARE BUILT, not after. A first version padded the rows
// to the full lane and THEN added the inset, so every row ran three cells long
// and the clamp ate the right-hand wall — the frame lost an edge on exactly the
// section that is supposed to be a painted region.
func (b Broadcaster) sectionWidth() int {
	w := b.laneWidth() - len(bcSectionInset)
	if w < 1 {
		return 0
	}
	return w
}

// laneWidth is how much room a card actually has, and it is the ONE place that
// number is decided (D-51).
//
// THAT IS THE RIGHT-RAIL SEAM, and it is the whole of what was owed now: above
// 150 columns a right rail is a LATER RELEASE, and when it arrives it must be a
// SMALLER NUMBER HANDED TO THE SAME CARD ROW rather than a second renderer. A
// card that derived its own width from `b.width` would have to be rewritten for
// every lane it ever appears in — which is exactly how the v1 mock came to be
// half a card.
func (b Broadcaster) laneWidth() int {
	const gutter = 2 // the "  " every lane row is indented by
	if b.width <= gutter {
		return 0
	}
	return b.width - gutter
}

// tooSmall reports whether the terminal is below the floor.
func (b Broadcaster) tooSmall() bool {
	// THE CLASSIFIER OWNS THE WIDTH BOUNDARY (D-50), and this asks it rather
	// than comparing against a second copy of the same number. `bcMinCols` is
	// what the NOTICE names; `BreakUnsupported` is what DECIDES — and the two
	// cannot drift, because a test asserts the floor is the first drawable
	// width.
	//
	// It also keeps the classifier in production: the layout became fully fluid
	// when the regions arrived, so nothing else branches on the class any more,
	// and a vocabulary with no caller is the state F-68 was filed about.
	if term.BreakpointFor(b.width) == term.BreakUnsupported {
		return true
	}
	_, r := b.minSize()
	return b.height < r
}

// clamp cuts the frame to the terminal, in BOTH directions.
//
// SILENT OVERFLOW IS A DEFECT, NOT A DEGRADATION (FR-7.3). F-55 measured the
// other surface rendering 57 cells into a 20-cell terminal with no clamp and
// no notice; this one does not inherit that. The clamp is applied to the
// NOTICE as well as to the lanes — a notice that overflows would let a sweep
// report zero overflows, which is the anti-solution the red team drove
// through M5.
func (b Broadcaster) clamp(lines []string) string {
	if b.height > 0 && len(lines) > b.height {
		lines = lines[:b.height]
	}
	// AND IT FILLS THE TERMINAL IT WAS GIVEN. A frame shorter than the screen
	// looks identical in the alt-screen — the rest is simply blank — which is
	// why this went unnoticed until something was COMPOSITED over it.
	//
	// `render.Overlay` centres vertically on the BASE's height, so a ten-line
	// frame in a seventy-four-line terminal pinned every window to the top rail
	// (UAT, 2026-09-10). The frame is the viewport, and it has to say so.
	for b.height > 0 && len(lines) < b.height {
		lines = append(lines, "")
	}
	if b.width > 0 {
		for i, l := range lines {
			// TRUNCATED *AND* PADDED: the frame IS the viewport, in both
			// dimensions. Truncating alone leaves a ragged frame whose widest
			// line is whatever the longest lane happens to be — and
			// `render.Overlay` composites against that, so a window centred on
			// the TERMINAL landed past the frame's right edge and the composite
			// grew sideways instead of stacking (UAT, 2026-09-10).
			lines[i] = render.PadTo(render.TruncateCells(l, b.width), b.width)
		}
	}
	return strings.Join(lines, "\n")
}

// notice is what the operator sees below the floor.
func (b Broadcaster) notice() []string {
	c, r := b.minSize()
	return []string{
		"TERMINAL TOO SMALL",
		"",
		"Broadcaster needs " + strconv.Itoa(c) + "x" + strconv.Itoa(r) + ".",
		"This terminal is " + strconv.Itoa(b.width) + "x" + strconv.Itoa(b.height) + ".",
	}
}

// lanes builds the three lanes from the last published schedule.
func (b Broadcaster) lanes() []string {
	g := b.opts().Glyphs()
	// ONE RENDERER FOR THE WHOLE FRAME (see cardLane): the lane width does not
	// change between the cards in it.
	lane := newCardLane(b.cardBoxWidth(), g)
	// THE FRAME OPENS WITH TWO BLANK ROWS, AS OBSERVER'S DOES (D-68). The HUM
	// LEAD, annotating his own mock: "Universal 2 line inset like Observer."
	// The console had its masthead hard against the top of the terminal, which
	// is the one place in the app that does not breathe.
	out := b.inset()
	out = append(out, strings.Split(b.header(b.opts()), "\n")...)
	// THE STATION BAR IS A SECTION (see stationSection): one region, painted by
	// one call, so the colour pass is a token rather than a sweep.
	fg, bg := b.stationTone()
	out = append(out, strings.Split(b.stationSection(b.opts(), fg, bg), "\n")...)
	out = append(out, b.heldNotice()...)
	// A BARE BLANK ROW SEPARATES THE STATION SECTION FROM THE RUNNING ORDER, and
	// the HUM LEAD annotated it twice: "Notice the blank line and how it
	// separates the rail — this is intentional." It carries NO walls, because
	// the station section is one closed box and the running order is another;
	// the air between them belongs to neither.
	out = append(out, "")
	// THE LANE NAMES ITSELF ON A BARE ROW (HUM LEAD, UAT 2026-09-10): "this line
	// … should have no pipes / lines." It is a caption over the running order,
	// not a row of it — so it is written before the frame's own columns are
	// added rather than inside them.
	out = append(out, b.laneLabel("STANDARD"))
	// NO SEPARATOR HERE. THE SECTION OWNS ITS OWN SPACING — `stationSection`
	// carries a breathing row above and below, and a second blank appended out
	// here made a DOUBLE gap that read as a rendering fault (HUM LEAD, UAT
	// 2026-09-10). One owner for the air around a region, like everything else.

	// THE PRIORITY TRACK IS INVISIBLE UNTIL IT HAS SOMETHING (D-61, HUM LEAD
	// 2026-09-10): "the PRIORITY rail label ONLY shows up when a priority card
	// sits on top of the main rail — this gives the operator more space to
	// view/manage the main rail during normal operation."
	//
	// A "(clear)" ROW IS NOT NOTHING. It spent two rows of the running order
	// saying that a hazard is not happening, which is the state the station is
	// in almost all of the time — and the track's own design is that it is
	// "normally INVISIBLE to the operator, so the main track takes the full
	// width of the UI."
	//
	// IT IS STILL DRAWN FIRST when it has something, because it DRAINS first in
	// every state: putting the lane that interrupts everything below the lane it
	// interrupts would say the wrong thing about which is which.

	// THE MAIN TRACK IS DRAWN AS NAMED REGIONS (D-60), which is what the
	// reference's left rail names: the card on the air, the one after it, the
	// ones scheduled behind that, and the rest of the line-up.
	//
	// THE RAIL IS WHY A CARD CARRIES NO STATE OF ITS OWN. An earlier card mock
	// had a strip saying whether a card was live or scheduled and the HUM LEAD
	// cut it — the rail already says it, once per region instead of once per
	// card.
	//
	// THE LINE-UP, NOT THE SCHEDULE (D-44). The Director's own structural cards
	// are read on air and never shown: the operator did not ask for them, and a
	// slot number spent on one is a number they cannot address.
	main := b.lineup.Projection(lineup.MainTrack)
	// A ROLLING VIEW OF TEN (FR-3.1). An eleventh card exists in the schedule
	// and does not reach the frame; the console shows a window onto the lineup,
	// never a second copy of it.
	if len(main) > MainTrackSlots {
		main = main[:MainTrackSlots]
	}
	// THE RUNNING ORDER IS IN TWO ZONES, and the boundary is where the scroll
	// rail starts (D-68). The HUM LEAD, annotating the reference beside the
	// SCHEDULED region: "Notice the top of the scroll is here, and the left rail
	// is separated." The cards the operator READS FROM do not scroll — there are
	// two of them and they are always the same two — so the gutter beside them
	// is empty and the rail begins below.
	// THE LANE NAMES ITSELF ABOVE ITS CARDS, and it is part of the BODY rather
	// than of the chrome above it — so it gets the frame's own right-hand
	// columns like every other row of the running order. Built here and not in
	// `out`, which is where the first version put it and why it came out with
	// no right wall at all.
	order, reads, after := b.laneHeader(), 0, (*bcRegion)(nil)
	for r := range bcRegions {
		reg := bcRegions[r]
		rows := b.region(reg, main, lane)
		if len(rows) == 0 {
			continue
		}
		// A BREAK FOLLOWS A READ REGION, AND ONLY A READ REGION (HUM LEAD, UAT
		// 2026-09-10): "there should be no line break here … this is one area
		// they should be continuous." SCHEDULED and LINE UP are one stack of
		// cards that the rail happens to name in two halves; LIVE and UP NEXT
		// are each a thing on its own, and the air is what says so.
		if after != nil && after.reads {
			order = append(order, b.regionGap())
		}
		order = append(order, rows...)
		after = &bcRegions[r]
		if reg.reads {
			reads = len(order)
		}
	}
	// AND THE RUNNING ORDER CLOSES ON A BLANK ROW, so the scroll rail's ▼ has
	// one of its own — which is where the reference draws it, under the last
	// card rather than across its border. Its ▲ already has one: the gap between
	// UP NEXT and SCHEDULED.
	if after != nil && reads < len(order) {
		order = append(order, b.regionGap())
	}

	// AND THE PRIORITY TRACK IS COMPOSITED ON TOP OF IT (D-61).
	//
	// "the priority track visually sits ON TOP of the main track — that's
	// because it's not supposed to always be on, and it signals that it is
	// TAKING OVER while there are alerts in that line."
	//
	// The running order is drawn at its FULL width underneath, which is what
	// the reference shows: its cards run 9..140 while the overlay covers 6..72,
	// so the operator keeps the right-hand half of every card it hides — the
	// half carrying the badge and the HANDLE they type.
	//
	// THE RAIL LABEL COMES WITH IT, which is the whole of the HUM LEAD's
	// ruling: "the PRIORITY rail label ONLY shows up when a priority card sits
	// on top of the main rail." While it is up, those rows are the priority
	// track's and say so.
	// THE FRAME'S RIGHT-HAND CHROME GOES ON LAST, over the assembled order and
	// whatever the priority track composited onto it — the scroll rail belongs
	// to the running order as a whole, not to any one region of it.
	body := b.withPriority(order, len(b.laneHeader()))
	if reads > len(body) {
		reads = len(body)
	}
	// AND THE FRAME RUNS THE FULL HEIGHT OF THE TERMINAL, LESS ITS CLOSING
	// INSET. It stopped at the last drawn row, so a station with a short line-up
	// showed a fragment floating in black — the reference carries its walls to
	// the bottom, and a frame that ends where its content does is not a frame.
	// THE SCROLL RAIL ENDS WITH THE CARDS, NOT WITH THE FRAME. Its ▼ sits on the
	// last row of the running order, which is what the reference draws — filler
	// below it is the frame reaching the bottom of the terminal, and a rail run
	// through that would say the line-up continues into empty space.
	out = append(out, b.chrome(body[:reads], false, 0, 0)...)
	out = append(out, b.chrome(body[reads:], true,
		MainTrackSlots, len(b.lineup.Projection(lineup.MainTrack)))...)
	// AND THE FRAME ENDS WHERE THE RUNNING ORDER DOES. It used to carry walled
	// blank rows to the bottom of the terminal, which is what the reference does
	// NOT do — its frame closes under the scroll rail's ▼ and the rest of the
	// screen is empty. `clamp` still pads the view to the terminal's height, so
	// D-63's rule holds: the frame is the viewport, and `render.Overlay` still
	// composites against a full-height base.
	return append(out, b.inset()...)
}

// inset is the blank air above and below the whole frame — Observer's own, which
// the console did not have.
func (b Broadcaster) inset() []string {
	return make([]string, bcInsetRows)
}

// bcInsetRows is how many blank rows open and close the frame.
//
// TWO, AND "UNIVERSAL" IS THE HUM LEAD'S OWN WORD FOR IT: "Universal 2 line
// inset like Observer." It is the app's air, not this surface's, which is why
// the number is stated once here rather than being folded into a caller.
const bcInsetRows = 2

// laneHeader names the lane the cards below it belong to, centred over them.
//
// THE REFERENCE DRAWS IT AND THE CONSOLE DID NOT (HUM LEAD, 2026-09-10: "Notice
// the header line; this should be centered"). It is the STANDARD lane; the
// priority lane names itself the same way when it has something, which is D-61's
// rule one row up from the rail label.
//
// A BARE BLANK ROW ABOVE IT, deliberately: "Notice the blank line and how it
// separates the rail — this is intentional." The station section is a closed box
// and the running order is another; the air between them belongs to neither, so
// it carries no walls.
func (b Broadcaster) laneLabel(label string) string {
	if b.cardBoxWidth() < 1 {
		return ""
	}
	// CENTRED OVER THE CARDS, not over the row: the caption names the lane, and
	// the lane is the card column. Measured in cells rather than bytes —
	// `render.Width` is the one measure (D-66).
	lead := bcRailWidth + bcRailGap + max(0, (b.cardBoxWidth()-render.Width(label))/2)
	return render.PadTo(strings.Repeat(" ", lead)+label, b.width)
}

// laneHeader is the air the running order opens with, under the lane's caption.
func (b Broadcaster) laneHeader() []string { return []string{b.railSpacer()} }

// orderWidth is how wide a row of the running order is BEFORE the frame's
// right-hand columns: the rail, the air beside it, and the card.
//
// ONE OWNER, because three things build such a row — a region, the gap between
// two regions, and the lane's header — and the first version of the header used
// the LANE's width instead. It came out seven cells long, so the chrome's own
// columns were pushed past the terminal's edge and clamped away, and that row
// alone lost its walls.
func (b Broadcaster) orderWidth() int { return bcRailWidth + bcRailGap + b.cardBoxWidth() }

// bcRegion is one named part of the running order, and the slots it holds.
type bcRegion struct {
	label      string
	from, upto int // half-open, in LINE-UP positions
	// reads is whether the operator READS FROM this region's cards rather than
	// merely ordering them — the LIVE card and the one after it (D-68). Those
	// draw the tall box that carries the script; the rest draw the flat one.
	reads bool
}

// bcRegions is the reference's own division of the main track, which is what
// its left rail spells out. The BED sits between UP NEXT and SCHEDULED and is
// not part of this table: it is not a card slot, and the schedule cannot put
// one there (Track's own comment refuses exactly that).
var bcRegions = []bcRegion{
	{"LIVE", 0, 1, true},
	{"UP NEXT", 1, 2, true},
	{"SCHEDULED", 2, 5, false},
	{"LINE UP", 5, MainTrackSlots, false},
}

// withPriority composites the priority track over the running order, or hands
// the order back untouched when the rail is clear.
//
// AN EMPTY RAIL COMPOSITES NOTHING, and the box being empty is the ONE thing
// that says so: a `len(rail) > 0` guard stood in the old drawing and its mutant
// SURVIVED, because a section built from no rows already returned nothing.
func (b Broadcaster) withPriority(order []string, from int) []string {
	lane := newCardLane(b.priorityWidth(), b.opts().Glyphs())
	rows := []string{}
	for _, c := range b.lineup.Cards(lineup.AlertRail) { // bounded by the rail (P10-02)
		rows = append(rows, lane.box(c, "T", "PRIORITY")...)
	}
	if len(rows) == 0 {
		return order
	}
	// IT COVERS THE CARDS, NOT THE LANE'S OWN HEADER. The running order now
	// opens with the header naming the lane and the spacer under it (D-68), and
	// an overlay spliced from row zero began on those — so the takeover's title
	// landed on a spacer and the card it is supposed to sit ON was one row down.
	if from < 0 || from > len(order) {
		from = 0
	}
	head := append([]string(nil), order[:from]...)
	order = append([]string(nil), order[from:]...)
	// THE OVERLAY CANNOT BE TALLER THAN WHAT IT COVERS. A takeover with more
	// rows than the running order would otherwise draw past the bottom of the
	// frame, which is the overflow FR-7.3 calls a defect rather than a
	// degradation.
	for len(order) < len(rows) {
		order = append(order, b.section("", []string{strings.Repeat(" ", b.cardBoxWidth())})...)
	}
	// A BLANK COLUMN AFTER THE BOX, so the overlay reads as sitting ON the
	// running order rather than merging with it. Without it the box's right
	// border butts straight into the card's own rule and the two draw as one
	// wide box — which says the opposite of what the overlay means.
	for i, r := range rows { // bounded by the overlay (P10-02)
		rows[i] = r + " "
	}
	out := spliceAt(order, railColumn("PRIORITY", len(rows), b.opts().Glyphs()), 0)
	return append(head, spliceAt(out, rows, bcPriorityCol)...)
}

// region draws one of them — EVERY slot it holds, decided or waiting (D-64).
//
// IT NO LONGER STOPS AT THE LAST DECIDED CARD. A region that drew nothing until
// the Director had chosen showed an empty frame on first launch, which is what
// the HUM LEAD found: "if I was a user who came upon this, I would not expect
// this to be working."
func (b Broadcaster) region(r bcRegion, cards []lineup.Card, lane cardLane) []string {
	return b.section(r.label, b.slotRows(r, cards, lane))
}

// bcGainCells is the gain bar's width, from the reference mock: thirty cells,
// which is what the level steps across at the tens.
const bcGainCells = 30

// stationSection is the station's state as a PAINTABLE REGION (HUM LEAD,
// 2026-09-10) — Variant C's two rows with breathing room above and below,
// rendered through `render.Opts.Block` so ONE call paints the whole thing.
//
//	"that station bar should be treated as a section/box — we're going to apply
//	a background color to it based on its state (RED for ON-AIR / GREY for
//	STANDBY) — so ensuring these are sectioned is important so we're not
//	painting color row by row / col by col manually."
//
// BLOCK IS THE RIGHT OWNER AND NOT JUST A CONVENIENCE. It pads every line to
// the width, and it RE-ARMS the tone at inner SGR resets — which is what stops
// a background tearing where a tinted run sits mid-line. This section contains
// three of those: the gain bar's filled cells, its chips and its level.
//
// THE PALETTE IS NOT CHOSEN HERE. Colour is the HUM LEAD's own pass, so the
// tone arrives as arguments and production passes what `stationTone` says —
// which today is nothing at all. What is built now is that the pass will be a
// token, not a sweep through every row.
func (b Broadcaster) stationSection(o render.Opts, fg, bg string) string {
	// WALLED LIKE EVERY OTHER ROW OF THE FRAME. The masthead draws its own box
	// and the running order carries the rail's; without these the station
	// section was the one region with no edges, and the frame read as broken
	// between them.
	rows := []string{}
	// NO WALLS. THE COLOUR IS THE EDGE (HUM LEAD, UAT 2026-09-10): "we can
	// remove the lines from the playing section, since we'll use color for the
	// differentiation." A painted band already has a boundary — drawing one as
	// well says the region is bordered AND filled, which is two answers to
	// where it begins.
	//
	// INSET FROM THE FRAME, as the reference draws it: the section's text begins
	// four cells in, not hard against the edge.
	for _, r := range append(append([]string{""}, b.stationLine()...), "") {
		rows = append(rows, render.PadTo(" "+bcSectionInset+r, b.width))
	}
	return o.Block(strings.Join(rows, "\n"), fg, bg)
}

// stationTone is the section's colour, by state.
//
// EMPTY UNTIL THE COLOUR PASS. The HUM LEAD has named the intent — red on air,
// grey on standby — and naming the TOKENS is their pass, not mine. `Block`
// treats an empty pair as "no tone of its own: the frame's base tone paints it",
// so the section is correct today and coloured by one edit here.
func (b Broadcaster) stationTone() (fg, bg string) {
	// THE HUM LEAD NAMED BOTH TONES BY THE THING THEY ALREADY EXIST ON (UAT
	// 2026-09-10): "the same grey taken as the Recent/Searched Locations on
	// STANDBY, and ALERT RED on ON AIR."
	//
	// SO THEY ARE THE SAME TOKENS, NOT NEW ONES. `GroupSectionBG` IS the
	// RECENT/SEARCHED band, and `TickerEmergencyBG` is what MVS-D-62 calls "THE
	// red" — a second red mixed here would be a second answer to what red means
	// in this app.
	if b.power == lineup.Running {
		return render.Tok(render.AlertModalText), render.Tok(render.TickerEmergencyBG)
	}
	return render.Tok(render.TextBase), render.Tok(render.GroupSectionBG)
}

// stationLine is the station's state, Variant C (D-21): a labelled field with
// the transition named in parentheses.
//
// THE WORDS CARRY THE STATE, NOT THE COLOUR (FR-5.3). A background treatment
// makes it legible at a glance and is the HUM LEAD's own pass — but a colour
// alone fails --ascii and any terminal without one, so a reader who sees no
// colour still reads the state.
//
// AND IT SAYS WHAT "ON AIR" MEANS (FR-5.5). Watchpost has no radio path: it
// produces audio, and a transmitter it cannot observe puts that over the air.
// An operator who reads a confident ON AIR and infers their antenna is
// radiating has been misled by us, so the boundary is stated HERE, where they
// read it — not only in a design document.
func (b Broadcaster) stationLine() []string {
	// THE SEPARATOR COMES FROM THE GLYPH SET, not a literal. A middle dot here
	// passed --ascii only because that test's fixture leaves the station
	// STOPPED, whose line carries no separator — a coverage hole in my own gate,
	// closed by sweeping every power state.
	o := b.opts()
	g := o.Glyphs()
	state, why, to := "STOPPED", "the programme is stopped; hazards still read", "ON AIR"
	switch b.power {
	case lineup.Running:
		state = "*** ON AIR " + g.Dot + " BROADCASTING ***"
		why = "audio out of this program; Watchpost does not observe a transmitter"
		to = "STANDBY"
	case lineup.OffAir:
		state = "STANDBY (DEAD AIR)"
		why = "nothing is broadcast, hazards included; the schedule holds what it has not said"
	}
	// A NOTICE DISPLACES THE PROSE. The state's own words describe a station at
	// rest; a refusal describes something the operator JUST DID, and the row
	// they are looking at has to answer the key they just pressed.
	if b.statusNote != "" {
		why = b.statusNote
	}
	lane := b.sectionWidth()
	// THE LABELS SHARE A VALUE COLUMN (D-62). "STATION:" and the bed's label are
	// different lengths, and a section whose two values began in different
	// columns would read as two unrelated rows rather than as one region saying
	// one thing.
	//
	// PADDED THROUGH `render.PadTo`, WHICH IS ALREADY ANSI-AWARE — "right-pads
	// a line to exactly width DISPLAY CELLS". That is how Observer handles the
	// same problem, and a hand-rolled version stood here briefly: a verbatim
	// reimplementation of PadTo, which is the second copy D-56 exists to
	// prevent. The bed's label opens with a CHIP, so its escape codes are
	// bytes that are not cells, and the ONE function that knows that should be
	// the only one that has to.
	label := func(s string) string { return render.PadTo(s, bcLabelCells) }
	hint := "( SHIFT + ENTER  " + g.Arrow + "  " + to + " )"
	gain := levelControl(o, "GAIN  ", b.gain,
		o.KeyCapIf("-", b.gain > 0), o.KeyCapIf("+", b.gain < 100), bcGainCells)
	// THE CONTROL SURVIVES AND THE PROSE YIELDS, the same rule as the card's
	// handle: the operator ACTS on the bar, and a level they cannot see is a
	// station they cannot set.
	room := lane - render.Width(gain) - 2
	// THE BED'S STATE READS AS A SENTENCE AND SITS AT THE RIGHT, where the
	// station's own transition hint sits — the two facts an operator checks
	// without reading the row are both in the same column (D-71).
	// THE BED SAYS WHAT IT IS ACTUALLY DOING (F-79). It said INACTIVE
	// unconditionally, because nothing published the answer.
	bed := g.Idle + "  BED IS INACTIVE"
	if b.bed.Carrying {
		bed = g.Live + "  BED IS ACTIVE"
	}
	return []string{
		render.PadBetween(label("STATION:")+state, hint, lane),
		// THE TRANSMITTER'S IDENTITY MOVED HERE FROM THE MASTHEAD (D-71). It is
		// a fact about the STATION; the masthead is what both surfaces share.
		render.PadTo(label("TRANSMITTER:")+b.transmitterRow(max(0, lane-bcLabelCells)), lane),
		render.PadBetween(label(o.KeyCap("b")+" BED:")+b.bedSelector(o), bed, lane),
		render.PadBetween(render.TruncateCells(label("")+why, max(0, room)), gain, lane),
	}
}

// bedRow is the bed's selector and its own state (D-62).
//
// THE BED IS THE THIRD CARRIER OF THE AIR STATE, and consolidating the three is
// the whole ruling: "the bed was yet another conveyer of the AIR STATE and we
// wanted to consolidate those … this way the ON AIR / STANDBY is all in one
// section, and the user doesn't have to look to different parts of the UI to
// determine what is and is not ON AIR."
//
// THE SELECTOR LIVES HERE, which is what F-77 was open about: it went missing
// when Variant C absorbed the control row, and the ruling put it on this row
// rather than restoring a separate card at the bottom of the frame.
func (b Broadcaster) bedSelector(o render.Opts) string {
	g := o.Glyphs()
	// THE ARROWS GO THROUGH KeyCap, which is the one owner that already names
	// them in WORDS under --ascii (`asciiKey`) — a literal here would print a
	// glyph a terminal without them cannot draw, in the row that says whether
	// the station is on the air.
	// THE RELAY IS THE PUBLISHED ONE (F-79, closed at D-78). It was the constant
	// `(no relay tuned)` while the schedule carried the lineup and the power and
	// nothing about the bed.
	//
	// THE STATE LEFT THIS ROW AT D-71 and sits at the right of the section with
	// the station's own transition hint; what stays here is the SELECTOR.
	_ = g
	relay := b.bed.Relay
	if relay == "" {
		relay = bcNoRelay
	}
	return o.KeyCap("←") + "  " + relay + "  " + o.KeyCap("→")
}

const (
	// bcSectionInset is the air between the frame's edge and a section's text,
	// from the reference: the wall at column 0 and "STATION:" at 4.
	bcSectionInset = "   "

	// bcLabelCells is the section's label column: wide enough for the longest of
	// them plus air, so every value starts in the same place.
	//
	// COUNTED IN DISPLAY CELLS, not bytes — the bed's label is a CHIP followed
	// by a word, and a chip carries SGR a byte count would charge it for.
	bcLabelCells = 16

	// bcNoRelay is what the bed's row says before a relay is tuned. The relay's
	// own description — its call sign, frequency and distance — arrives with the
	// bed's state, which the schedule does not publish yet (F-79).
	bcNoRelay = "(no relay tuned)"
)

// cardRow is one lane row: what it is, and the handle that addresses it.
// cardLane renders every card in one lane, at that lane's width (D-52).
//
// IT IS BUILT ONCE PER FRAME, NOT ONCE PER CARD, and that is not a
// micro-optimisation: a row constructed per card put the console frame at 70
// allocations against a budget of 14, which the alloc gate caught on the commit
// that introduced it. The lane's width does not change between the cards in it,
// so neither should the thing that measures it.
//
// IT TAKES THE LANE, NOT A CLASS. The previous version took `wide bool` and
// picked 60 or 30 columns from it, which is the hard-coded geometry the HUM LEAD
// ruled out: "this layout can dynamically resize in between our breakpoints".
// The class still decides the FRAME; it does not decide the card's arithmetic.
//
// THE ANCHORING RULE, and it is the whole of the mock:
//
//	title    centred, and truncates KIND-FIRST so the SUBJECT survives — the
//	         kind repeats down the whole lane, the subject is what tells one
//	         card from another
//	badge    right-anchored, inboard of the handle
//	handle   right-most and fixed. IT IS THE ADDRESS THE OPERATOR TYPES, so it
//	         is the one thing that never truncates and never moves
//
// A GO-STUDS ROW, per the standing rule, and it earns its place rather than
// merely satisfying it: the fill column is sized with the badge's width ALREADY
// RESERVED, so a centred title cannot run into the badge. The hand-rolled draft
// written while generating the mock tested whether the title fit BY LENGTH and
// produced "…(COASTAL)D•" — a centred title can fit by length and still collide
// by POSITION. Here that is structural rather than policed.
type cardLane struct {
	// o is the render options the lane draws with. It carries the CHIP
	// renderer, which the handle needs: `[ 6 ]` in the reference is a chip, not
	// text wearing brackets (HUM LEAD, 2026-09-10).
	o render.Opts

	row *components.DataTableRow
	// marked is the same row with a leading, NON-TRUNCATABLE column for the
	// fabricated-event mark (D-55). TWO ROWS RATHER THAN ONE, because a fixed
	// column present on every card would reserve its width on every card — the
	// real hazards would all sit 14 cells off-centre to make room for a label
	// they never carry.
	//
	// IT IS A COLUMN AND NOT `SetPrefix` BECAUSE THE COMPONENT DOES NOT RENDER
	// ONE ON DATA ROWS. `SetPrefix` is accounted for in the width calculation
	// (`prefixWidth`, data_table_row.go:521) and written only by `RenderHeader`
	// — so a prefix set here reserved its space and printed nothing, which the
	// test caught. Recorded as an M6 upstream candidate rather than patched
	// locally: the dependency is not re-implemented, and the gap goes home.
	marked *components.DataTableRow
	lane   int
	g      render.Glyphs
}

// newCardLane prepares the renderer for a lane of the given width.
func newCardLane(lane int, g render.Glyphs) cardLane {
	if lane <= 0 {
		return cardLane{lane: 0, g: g}
	}
	headline := components.ColumnDefinition{
		Name:              "headline",
		Fill:              true,
		Alignment:         "center",
		Truncatable:       true,
		TruncatedMinWidth: 8,
		TruncationTail:    g.Ellipsis,
	}
	mark := components.ColumnDefinition{
		Name:  "mark",
		Width: render.Width(testEventMark),
		// NEVER TRUNCATABLE. A half-eaten mark is worse than none: "**TEST EV…"
		// beside a tornado warning reads as damage rather than as a label.
		//
		// FALSE IS THE ZERO VALUE, so this line is documentation with syntax and
		// its mutant SURVIVES BY DESIGN — the component never needs to shrink
		// this column at any width the row can be built at, because the fill
		// column absorbs the squeeze first. It is stated because the guarantee
		// belongs to a dependency whose sizing this package does not own, and
		// the backstop below is what actually enforces the rule.
		Truncatable: false,
	}
	return cardLane{
		lane: lane, g: g, o: render.Opts{ASCII: g.Rule == "-"},
		row:    components.NewDataTableRow(lane, []components.ColumnDefinition{headline}),
		marked: components.NewDataTableRow(lane, []components.ColumnDefinition{mark, headline}),
	}
}

// render is one card as the operator reads it.
func (l cardLane) render(c lineup.Card, handle, badge string) string {
	// Both rows are built together, so one guard answers for both.
	if l.row == nil || l.marked == nil {
		return ""
	}
	// THE HEADLINE, NOT THE SUBJECT. The headline is what the card is ABOUT in
	// the words a person reads; the subject is its key.
	data := map[string]string{"headline": kindFirst(plaintext.Text(c.Headline), l.lane, l.g)}
	row := l.row
	if c.Test {
		// A FABRICATED TAKEOVER SAYS SO, AND SAYS IT FIRST (FR-4.4, D-55).
		//
		// THE SAME MARK THE BAND AND THE SEVERE WINDOW USE — one constant, so
		// the three surfaces cannot come to say different things about one
		// event. It is its own NON-TRUNCATABLE column (see newCardLane), so the
		// headline's truncation can never eat it.
		row, data["mark"] = l.marked, testEventMark
	}
	// THE BADGE AND THE HANDLE TRAVEL AS ONE right-anchored unit, which is how
	// the reference draws them: two cells apart, the handle outermost. Every
	// badge in the set is the same width, so the space the row reserves for one
	// is the space it reserves for all — the cards stay aligned down the lane,
	// marked and unmarked alike.
	// THE HANDLE IS A CHIP, NOT TEXT WEARING BRACKETS (HUM LEAD, 2026-09-10:
	// "[ 1 ] is a chip in the card(s)"). `KeyCap` paints " 6 " with the chip
	// background in colour and falls back to "[6]" without it — so the
	// reference's five cells ARE the chip, and hand-writing the brackets drew
	// its costume while losing everything it is: the palette, --ascii's word
	// forms, and every future state a control can show.
	//
	// THE TRAILING SPACE IS THE REFERENCE'S: the mock leaves ONE cell between
	// the handle and the card's right border, and the component right-aligns a
	// badge flush.
	row.SetBadge(l.g.Bullet+badge+l.g.Bullet+"  "+l.o.KeyCap(handle)+" ", 2)
	out := render.PadTo(render.TruncateCells(row.RenderRow(data), l.lane), l.lane)
	// A CARD THAT CANNOT BE LABELLED HONESTLY IS NOT DRAWN AT ALL (D-55).
	//
	// THE CLAMP ABOVE RUNS AT EVERY WIDTH, INCLUDING BELOW THE FLOOR, and a
	// test caught it chopping the mark into "**TEST E" — the one output that is
	// worse than no mark, because a reader takes a broken label for rendering
	// damage and the warning beside it for real. Dropping the row is the safe
	// direction: a fabricated takeover drawn WITHOUT its mark is the screenshot
	// hazard itself, so the only remaining choice is to draw nothing.
	//
	// It is unreachable in a supported configuration — below 100 columns the
	// console renders its notice and never reaches a card (D-50) — which is
	// exactly why it is a check and not a layout.
	if c.Test && !strings.Contains(out, testEventMark) {
		return ""
	}
	return out
}

// bcCardRows is a card's height: a top border, the title, a body row and a
// bottom border — four, from the reference.
//
// EVERY CARD IS THE SAME HEIGHT, fabricated or not. The body row is blank on a
// real card and carries D-57's corner marks on a test one, rather than a test
// card growing a fifth row: a lane whose rows moved when an alert was injected
// would renumber every slot below it, and the slot number is the address the
// operator types.
const bcCardRows = 4

// box draws one card as the reference draws it: four rows, walled on every edge.
//
// THE BORDERS ARE WHAT D-57's CORNERS NEEDED. The console drew flat rows until
// now, so a fabricated card had exactly one place to be marked — and the
// priority track sits ON TOP of the main track, which makes a takeover the card
// most likely to be partly occluded and one mark on it the weakest possible
// placement.
func (l cardLane) box(c lineup.Card, handle, badge string) []string {
	return l.boxOf(c, handle, badge, nil)
}

// boxOf is the card, with whatever the region puts INSIDE it below the first
// row (D-68).
//
// THE READ CARDS ARE TALLER THAN THE SCHEDULED ONES, which is the reference and
// was the HUM LEAD's UAT (2026-09-10): "the LIVE CARD should be bigger to
// support showing at least most the script being played … UP NEXT should also be
// bigger." A card the operator READS FROM needs the words on it; a card they are
// merely deciding the ORDER of needs its name and its handle.
//
// ONE DRAWER FOR BOTH, because everything except the interior is the same
// card — the borders, the centred title, the badge and the handle's chip. A
// second box function would be a second place for the handle to drift, which is
// the D-56 shape this file has already paid for once.
func (l cardLane) boxOf(c lineup.Card, handle, badge string, body []string) []string {
	if l.lane < 4 {
		return nil
	}
	inner := l.lane - 2
	g := l.g
	rule := strings.Repeat(g.Rule, inner)
	// The title row is the SAME renderer the flat row used, one width in: the
	// box does not get to move the handle or re-centre the title.
	title := newCardLane(inner, g)
	rows := []string{
		g.CornerTL + rule + g.CornerTR,
		g.Rail + title.render(c, handle, badge) + g.Rail,
		// THE FIRST INTERIOR ROW IS ALWAYS THE CORNERS' ROW, tall or flat: it is
		// where the fabricated-event marks live (D-57), and a mark that moved
		// with the card's height would be in a different place on every card.
		g.Rail + l.corners(inner, c.Test) + g.Rail,
	}
	for _, r := range body { // bounded by the card's own height (P10-02)
		rows = append(rows, g.Rail+render.PadTo(render.TruncateCells(r, inner), inner)+g.Rail)
	}
	return append(rows, g.CornerBL+rule+g.CornerBR)
}

// corners is the card's body row: blank, or D-57's marks at both ends.
//
// BOTH ENDS, which is the ticker's own reasoning one surface along — it marks a
// fabricated item at BOTH ends of the tape "because the tape scrolls: a marker
// at one end only is off-window half the time." A card can be occluded from
// either side by the priority overlay, and two corners cannot both be covered by
// a box that starts at the left.
// PASSED, NOT CARRIED. A first version set a `test` field on the lane inside
// `render` — which takes a VALUE receiver, so the flag died with the copy and
// the corners never drew. Hidden state across two methods of a value type is
// how that happens; the fact travels as an argument now.
func (l cardLane) corners(inner int, test bool) string {
	if !test || inner < 2*len(bcCornerMark)+4 {
		return strings.Repeat(" ", inner)
	}
	gap := inner - 2*len(bcCornerMark) - 4
	return "  " + bcCornerMark + strings.Repeat(" ", gap) + bcCornerMark + "  "
}

// bcCornerMark is the short form of the test mark, for the corners. The title
// row carries the full `**TEST EVENT**`; the corners repeat the fact in the
// space a corner has.
const bcCornerMark = "TEST"

// kindFirst drops the card's KIND before it touches the subject.
//
// "LOCATION REPORT • OCEANSIDE, CA 92057" becomes "… • OCEANSIDE, CA 92057"
// before it becomes "LOCATION REPORT • OCEANSID…". The kind is repeated on every
// card in the lane and carries almost no information there; the subject is the
// only part that says which card this is.
//
// Anything that still does not fit is handed to the row's own truncation, which
// owns the arithmetic.
func kindFirst(headline string, lane int, g render.Glyphs) string {
	const sep = " • "
	head, sub, split := strings.Cut(headline, sep)
	if !split || head == "" {
		return headline
	}
	// Only when the lane is genuinely too tight for the whole thing: a wide lane
	// keeps the kind, which is what the reference draws.
	if render.Width(headline) <= lane-bcRowTail {
		return headline
	}
	return g.Ellipsis + sep + sub
}

// bcRowTail is what the badge and handle reserve at the right of a card row: the
// badge, its two delimiters, the gutter and the five-cell handle. It is an
// ESTIMATE, used only to decide when the kind is dropped — the row itself owns
// the real arithmetic, and over-estimating here costs a kind, never a subject.
const bcRowTail = 24

// withSize is a console sized for a test, so a fixture reads as one expression.
func (b Broadcaster) withSize(w, h int) Broadcaster {
	b.width, b.height, b.ascii = w, h, true
	return b
}
