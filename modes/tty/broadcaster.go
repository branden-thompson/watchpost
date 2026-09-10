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
		return b, nil
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		b.width, b.height = v.Width, v.Height
	case tea.BackgroundColorMsg:
		b.darkBG = v.IsDark()
	case LineupMsg:
		b.lineup = v.Lineup
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
	return render.Opts{Width: b.width, ASCII: b.ascii}
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
	c, r := b.minSize()
	return b.width < c || b.height < r
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
	lane := newCardLane(b.laneWidth(), g)
	out := strings.Split(b.header(b.opts()), "\n")
	out = append(out, b.stationLine()...)
	out = append(out, b.heldNotice()...)
	out = append(out, "")

	// THE PRIORITY TRACK IS DRAWN FIRST because it DRAINS first, in every
	// state. Drawing it below the rotation would put the lane that interrupts
	// everything under the lane it interrupts.
	out = append(out, "PRIORITY")
	rail := b.lineup.Cards(lineup.AlertRail)
	if len(rail) == 0 {
		out = append(out, "  (clear)")
	}
	for _, c := range rail {
		out = append(out, "  "+lane.render(c, "T", "PRIORITY"))
	}
	out = append(out, "")

	// THE BREAKPOINT SELECTS THE LAYOUT (FR-7.1, D-13). Not a call whose
	// result is discarded: the class decides what a lane row can afford, and
	// changing the class changes the frame.
	wide := term.BreakpointFor(b.width) >= term.BreakOptima
	if wide {
		out = append(out, "SCHEDULED LINE UP")
	} else {
		out = append(out, "LINE UP")
	}
	// THE LINE-UP, NOT THE SCHEDULE (D-44). The Director's own structural cards
	// are read on air and never shown: the operator did not ask for them, and a
	// slot number spent on one is a number they cannot address. The staleness
	// notice has been in the schedule since 0.14.0, so this is a live
	// difference, not a future one.
	main := b.lineup.Projection(lineup.MainTrack)
	// A ROLLING VIEW OF TEN (FR-3.1). An eleventh card exists in the schedule
	// and does not reach the frame; the console shows a window onto the
	// lineup, never a second copy of it.
	if len(main) > MainTrackSlots {
		main = main[:MainTrackSlots]
	}
	if len(main) == 0 {
		out = append(out, "  (nothing scheduled)")
	}
	for i, c := range main {
		out = append(out, "  "+lane.render(c, strconv.Itoa(i), "STANDARD"))
	}
	out = append(out, "")
	out = append(out, "BED   (no relay tuned)")
	return out
}

// bcGainCells is the gain bar's width, from the reference mock: thirty cells,
// which is what the level steps across at the tens.
const bcGainCells = 30

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
	// THE SEPARATOR COMES FROM THE GLYPH SET, not a literal. A middle dot
	// here passed --ascii only because that test's fixture leaves the station
	// STOPPED, whose line carries no separator — a coverage hole in my own
	// gate, closed by sweeping every power state.
	g := b.opts().Glyphs()
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
	// GAIN RIDES THE SECOND ROW (HUM LEAD's layout, 2026-09-10) — the row
	// Variant C left free when it absorbed the control line. It is Observer's
	// own bar under the station's word for it, so there is one level and one
	// place it is drawn.
	o := b.opts()
	gain := levelControl(o, "GAIN  ", b.gain,
		o.KeyCapIf("-", b.gain > 0), o.KeyCapIf("+", b.gain < 100), bcGainCells)
	// VARIANT C (D-21): a labelled field, the transition in parentheses. The
	// two are the ENDS of one line — the transition RIGHT-ANCHORED rather than
	// padded to a fixed column, which is what it was and which lands correctly
	// at exactly one terminal width.
	lane := b.laneWidth()
	hint := "( SHIFT + ENTER  " + g.Arrow + "  " + to + " )"
	// THE CONTROL SURVIVES AND THE PROSE YIELDS. Same rule as the card's
	// handle: the operator ACTS on the bar, and a level they cannot see is a
	// station they cannot set. The sentence explains something they can also
	// read in the state above it.
	room := lane - render.Width(gain) - 2
	return []string{
		render.PadBetween("STATION:  "+state, hint, lane),
		render.PadBetween(render.TruncateCells("          "+why, max(0, room)), gain, lane),
	}
}

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
		lane: lane, g: g,
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
	row.SetBadge(l.g.Bullet+badge+l.g.Bullet+"  [ "+handle+" ]", 2)
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
