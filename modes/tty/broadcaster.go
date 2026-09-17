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
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/category"
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

// StationFaultMsg is the Director's escalation reaching the console: Run cards
// in a row the station could not perform, and the last reason. Run 0 clears
// it — a read that finished, or STANDBY (F-150, REVIEW 2026-09-17).
type StationFaultMsg struct {
	Run    int
	Reason string
}

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

// BedRelaysMsg is how many relays actually STREAM within the station's reach
// (D-117).  Zero disables the bed.
//
// HUM LEAD, 2026-09-13: "If none exist in that area - we should probably tell the
// broadcaster there is no valid relays for their area and disable the BED option
// so the Operator cannot choose something that will broadcast dead air."
//
// A COUNT RATHER THAN A BOOL, because the row has something to say with it: "no
// relays reach this station" is a different sentence from "(no relay tuned)", and
// only the count can tell them apart.
//
// ITS OWN MESSAGE, AND D-125 IS WHY. It was a field of `BedMsg`, which has THREE
// publishers — the resolver, the selector and the deck's state — of which exactly
// one set it. The other two left it at zero and silently retracted the resolver's
// answer, so the console disabled a bed that was carrying. One fact, one message,
// ONE WRITER: `setBedStations`, on the path that resolves them.
type BedRelaysMsg struct{ Count int }

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

	// Pool is the station's candidate locations, nearest first, as the Producer
	// itself sees them (D-93).
	//
	// IT TRAVELS WITH THE AREA BECAUSE IT IS THE SAME DERIVATION. The pool is a
	// pure function of the transmitter and the service radius (pool.go), so a
	// console told them separately could hold a pool that belongs to an area it
	// is no longer showing — which is D-59's torn pair, one fact along.
	//
	// THE CONSOLE READS IT, IT DOES NOT DERIVE IT. `locations.Pool` needs the
	// geodata index and `modes/tty` may not import a domain; more to the point,
	// a second derivation would be a second answer to what the station may read.
	Pool []snapshot.LocationRef
}

// MainTrackSlots is how many slots the console draws for the running order.
//
// THE SCHEDULE'S NUMBER, READ RATHER THAN RESTATED (R3). It was `16` here and
// nowhere else, because the console was the first thing that needed it — and
// then `Insert` needed a cap to shed the last card past, which made it a
// SCHEDULE rule with the console holding the only copy.
//
// `modes/` MAY IMPORT `platform/`, so this is a direct read rather than a value
// handed through `Config` the way the service radius is (D-124): that one went
// the long way round because `platform/config` is STORAGE and nothing under
// `modes/` reads storage. The running order's length is not storage.
const MainTrackSlots = lineup.MainTrackCap

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

	// fault is the last StationFaultMsg, shown in the band while ON AIR until
	// a read finishes or the station goes to STANDBY.
	fault StationFaultMsg

	// listenerMuted is the Observer's [M], handed across by the Router: while it
	// holds, every hazard read on this station is declined, and the console
	// says so while ON AIR (REVIEW 2026-09-17, ruling 6-ii).
	listenerMuted bool

	// frame is the shimmer's animation phase, and tickArmed keeps exactly one
	// tick in flight (D-64). ITS OWN, NOT THE DASHBOARD'S: Observer arms its
	// tick only while IT needs one, and the console needs one whenever a slot
	// is still waiting on the Director — two different predicates, so a shared
	// arm would leave whichever surface asked second without an animation.
	frame     int
	tickArmed bool

	// lineupGen and areaGen count the WHOLESALE assignments of the two table
	// inputs that are not `==` types (a Lineup holds an array of slices, a
	// StationAreaMsg holds one). The app replaces both from a message and never
	// edits either in place, so a counter bumped where the assignment happens is
	// an exact identity for the value — the memo's reason for them.
	lineupGen, areaGen uint64

	// memo is the two tables' slot. A POINTER, because View() is a value
	// receiver and every Update copies the console (P10-06) — the cache has to
	// outlive the copy that filled it. Nil is legal and means no memo: a console
	// built as a bare literal renders every frame.
	memo *consoleMemo

	// fireBoldMW is the operator's [fire] bold_frp_mw, inherited RESOLVED from
	// the Dashboard (NewRouter) so the default lives in one place. Zero means a
	// console built without a Router; `fireBold` falls back for it.
	fireBoldMW float64

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

	// THE POINTER MOVES AND THE WINDOW FOLLOWS IT at render time, where the room
	// is known (scheduledLines, D-101). An offset held on the MODEL cannot know
	// how many rows fit, which is why the window is derived rather than stored
	// and never came back.

	// selected is the focused row, indexed across BOTH tables — the running order
	// first, the pool after it (D-101).
	//
	// ONE POINTER OVER TWO TABLES, which is Observer's own shape: `d.selected`
	// spans the watchlist and RECENT through `numPriority`, and the operator
	// walks from one into the other without noticing a boundary. The HUM LEAD
	// asked for the same here — "pointer state is shared across the two go-studs
	// tables … just like it is in the Observer tables" — and it is what `enter`
	// acts on, so a surface without it has a key with no target.
	selected int

	// bed is what the broadcast is riding on (F-79). Published, never guessed:
	// the console draws it and holds no opinion of its own, the same rule the
	// power follows.
	bed BedMsg
	// bedRelays is how many relays actually STREAM near the station, and
	// bedRelaysTold is whether the Producer has answered yet (D-117).
	//
	// ITS OWN FACT, ARRIVING ON ITS OWN MESSAGE (D-125). Carried on `BedMsg` it
	// would have three publishers of which exactly ONE sets it, so the selector's
	// message and the deck's state message would each leave it at zero — silently
	// retracting the resolver's answer, and the console
	// disabled a bed that was carrying. D-117 exists so the operator cannot pick
	// something that broadcasts dead air; that defect told them there was nothing
	// to pick while a relay was streaming.
	//
	// UNTOLD IS NOT ZERO. Before the Producer has answered, the bed is OFFERED —
	// refusing it on the strength of an answer nobody has given yet would hide
	// the control for the whole of start-up.
	bedRelays     int
	bedRelaysTold bool

	// pool is the recent pipeline's snapshot — where the LOCATION POOL's weather
	// comes from (D-99). Never the masthead's: that stamp is the priority
	// pipeline's, and one field for both would report a freshness the console does
	// not have.
	pool *snapshot.Snapshot

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
func NewBroadcaster() Broadcaster { return Broadcaster{memo: &consoleMemo{}} }

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
	// HAZARDS, NOT CARDS — AND ONLY READABLE ONES (D-142).
	//
	// TWO DEFECTS IN ONE LINE, on the console's loudest safety surface. It
	// counted CARDS: a burst is ONE card carrying many arrivals (MVS-D-77), so
	// five hazards held read "1 HAZARD(S) HELD", and the escalation ladder keys
	// off that number. And it read `Cards()` where every other rail reader
	// reads `Projection()`, which drops out-of-fence cards precisely because
	// they are NOT READ — so a rail the fence excluded still said "Go ON AIR to
	// read them", an instruction that would read nothing, escalating to "may be
	// dropped unread".
	//
	// `Projection` IS THE ONE OWNER OF "WHAT THIS RAIL CAN ACTUALLY SAY", and
	// counting its arrivals is the same question the takeover box answers — so
	// the band and the box can no longer disagree about whether anything is
	// being withheld.
	var held int
	for _, c := range b.lineup.Projection(lineup.AlertRail) { // bounded by the rail (P10-02)
		if n := len(c.From); n > 0 {
			held += n
			continue
		}
		held++ // a card with no arrivals recorded is still one hazard held
	}
	if held == 0 {
		return nil
	}
	for _, s := range heldEscalation() { // bounded by the ladder (P10-02)
		if b.standbySince.IsZero() || b.clock().Sub(b.standbySince) >= s.after {
			return b.heldBand(held, s.mark, s.say)
		}
	}
	return nil
}

// heldBand draws the notice as a BAND rather than a line (D-138).
//
// HUM LEAD, 2026-09-15: "Let's make that look like the ticker I almost missed
// this: 3 lines, message in the center, 3 line bkg should be the darker yellow
// (not orange, not red) use the same tint as the 'LOCAL ALERT Advisory BKG'."
//
// ALMOST MISSED IS THE WHOLE FINDING. The notice was one unpainted row among
// painted regions, which is the least visible thing a frame can contain — and
// this is the one row that says a hazard is being held off the air. Its words
// were right and nobody's eye stopped on them.
//
// `AlertModalAdvBG` IS THE TINT NAMED: the LOCAL ALERT window's advisory tile,
// "muted yellow" in its own comment — not `TickerAdvisoryBG`, which is the
// burnt orange of the tape's advisory lane and the colour the ruling excludes.
// `AlertModalText` comes with it: `aaPairs` already registers that pair, so the
// band inherits a contrast answer measured in every theme.
//
// BUILT THE WAY THE STATION SECTION IS BUILT — padded to the frame less the
// inset, the inset added, then painted in one Block call. Two adjacent bands
// assembled two different ways is how a three-cell disagreement gets in.
func (b Broadcaster) heldBand(held int, mark, say string) []string {
	count := strconv.Itoa(held) + " HAZARD(S) HELD"
	return b.noticeBand(count, mark+"  "+count+" — the station is in STANDBY and nothing is going to air. "+say)
}

// faultNotice is the band an ON AIR station shows when it could not perform a
// run of cards (F-150): the count shouts, the reason says what to check.
func (b Broadcaster) faultNotice() []string {
	if b.power != lineup.Running || b.fault == (StationFaultMsg{}) { // the zero message is the clear
		return nil
	}
	if b.fault.Run == 0 {
		// AN ESCALATION WITH NO RUN — a bed that could not be tuned, a schedule
		// emptied on standby — is shown by its reason alone (F-163, HUM LEAD
		// 2026-09-17); a band keyed on the count dropped it.
		return b.noticeBand("STATION FAULT", "!!!  STATION FAULT — "+b.fault.Reason)
	}
	count := strconv.Itoa(b.fault.Run) + " CARD(S) FAILED"
	return b.noticeBand(count, "!!!  "+count+" — the station could not perform them: "+b.fault.Reason)
}

// muteNotice is the band an ON AIR station shows while the listener's [M]
// holds: every hazard read is declined while it does, and a console that said
// nothing let a station broadcast silence (REVIEW 2026-09-17, ruling 6-ii).
func (b Broadcaster) muteNotice() []string {
	if b.power != lineup.Running || !b.listenerMuted {
		return nil
	}
	const count = "LISTENER MUTED"
	return b.noticeBand(count, "!!!  "+count+" — [M] in the Observer is on, so no hazard is read aloud on this station until it is lifted.")
}

// noticeBand draws one band under the station section — the COUNT in bold on
// the band's own white, the prose plain, wrapped before it is styled — for the
// held notice and the fault notice alike.
func (b Broadcaster) noticeBand(count, plain string) []string {
	o := b.opts()
	// THE COUNT SHOUTS AND THE PROSE DOES NOT (the ruling, line by line):
	// "'1 HAZARD(S) HELD' - BOLD WHITE / -- the station is in STANDBY and
	// nothing is going to air. <- normal text color / 'ON AIR' - BOLD".
	//
	// BOLD ON THE BAND'S OWN WHITE, NOT A SECOND WHITE. `AlertModalText` is
	// white already, so weight is the only thing added — and weight is not
	// contrast, so no AA answer moves (the same argument D-134 makes).
	// WRAPPED BEFORE IT IS STYLED, WHICH IS D-129a's RULE AND THE REASON THIS
	// IS NOT ONE `centerText` CALL. Centring alone CLIPS: the !!! rung's
	// sentence is ~149 cells and the band is 127 at the HUM LEAD's width, so
	// the most severe message on the console lost its ending — "may be drop".
	// The previous single line wrapped in the terminal instead, so clipping
	// would have been a regression introduced by making it prettier.
	//
	// AND THE EMPHASIS IS APPLIED PER LINE, AFTER the wrap, for the same reason
	// the caveat's tint is: styling spans do not survive being cut in half.
	//
	// THREE LINES IS THE COMMON CASE, not a promise the band breaks to keep. At
	// any width where the sentence fits — which is every rung but the last on a
	// wide terminal — this is exactly the blank, message, blank the ruling asks
	// for; where it does not fit, the band grows rather than the message
	// shrinking, because a truncated hazard notice is the one outcome this
	// notice exists to prevent.
	body := make([]string, 0, 3)
	for _, l := range render.WrapText(plain, b.bandWidth()) { // bounded by the text (P10-02)
		body = append(body, centerText(emphasiseHeld(l, count), b.bandWidth()))
	}

	// ONE BLANK ABOVE AND BELOW, which is what makes it a band instead of a
	// coloured row.
	rows := make([]string, 0, len(body)+2)
	for _, r := range append(append([]string{""}, body...), "") {
		rows = append(rows, render.PadTo(bcSectionInset+r, b.frameWidth()-len(bcSectionInset))+bcSectionInset)
	}
	band := o.Block(strings.Join(rows, "\n"), render.Tok(render.AlertModalText), render.Tok(render.AlertModalAdvBG))
	// AND A BARE ROW ABOVE IT, unpainted, so the band does not fuse with the
	// station section it follows — the same air the frame keeps between every
	// pair of regions.
	return append([]string{""}, strings.Split(band, "\n")...)
}

// emphasiseHeld picks out the two things the ruling names on ONE wrapped line:
// the count, and the action the operator must take.
//
// PER LINE, AND ONLY WHAT THAT LINE HOLDS. A phrase split across a wrap is left
// plain rather than half-bolded — the alternative is an escape opened on one
// row and closed on the next, which is the defect D-129a was filed for.
func emphasiseHeld(line, count string) string {
	for _, phrase := range []string{count, "ON AIR"} {
		line = strings.ReplaceAll(line, phrase, render.Bold(phrase))
	}
	return line
}

// heldEscalation is the ladder, LONGEST FIRST so the walk returns the most
// severe rung that has been reached.
//
// The rungs are a PARAMETER, written down the way a threshold is meant to be
// (INST-1 scopes its rule to the SET being iterated, not to the numbers).
// heldEscalation is the notice's ladder: how long a hazard has been held, the
// mark that rung wears, and what it says.
//
// A FUNCTION RATHER THAN A PACKAGE VARIABLE (P10-06), which is the convention
// `report.all` states in as many words. The reason a map here would stay a var
// does not apply: this is a fixed-size array of small values returned BY VALUE,
// so there is nothing to rebuild on the heap and the table cannot be reassigned
// by anything in the package — which for an escalation ladder on the console's
// loudest safety surface is the point.
func heldEscalation() [3]struct {
	after time.Duration
	mark  string
	say   string
} {
	return [3]struct {
		after time.Duration
		mark  string
		say   string
	}{
		{15 * time.Minute, "!!!", "They have been held past the staleness bound and may be dropped unread."},
		{5 * time.Minute, "!!", "Go ON AIR or stand the station down."},
		{0, "!", "Go ON AIR to read them."},
	}
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
	// AND THE RECENT ONE, WHICH IS WHERE THE POOL'S WEATHER LIVES (D-99). Kept
	// apart from `snap` deliberately: the masthead's stamp and the API summary are
	// the PRIORITY pipeline's, and merging the two would make the console report a
	// freshness it does not have.
	if v, ok := msg.(RecentSnapshotMsg); ok && v.Snap != nil {
		b.pool = v.Snap
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
		// THE COUNTER MOVES WITH THE VALUE, on the same line that replaces it.
		// The memo's key carries the generation because a Lineup cannot be
		// compared; a bump left behind here is a table that stops redrawing.
		b.lineup, b.lineupGen = v.Lineup, b.lineupGen+1
	case StationAreaMsg:
		b.area, b.areaGen = v, b.areaGen+1
	case BedMsg:
		b.bed = v
	case BedRelaysMsg:
		// THE COUNT ARRIVES ON ITS OWN (D-125). On `BedMsg` it would have three
		// publishers of which one sets it, so the selector's message and the
		// deck's state message would each zero what the resolver established.
		b.bedRelays, b.bedRelaysTold = v.Count, true
	case StationFaultMsg:
		b.fault = v
	case StationMsg:
		b = b.powered(v.Power)
	}
	return b, nil
}

// powered takes the station's power state, and starts the standby clock on
// the TRANSITION, not on every message: a station that has been silent an hour
// must not look freshly quiet because another message arrived.
func (b Broadcaster) powered(to lineup.Power) Broadcaster {
	if to != b.power {
		b.standbySince = time.Time{}
		if to == lineup.OffAir {
			b.standbySince = b.clock()
			b.fault = StationFaultMsg{} // standby is the operator acting on it
		}
	}
	b.power = to
	return b
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
	return render.Opts{Width: b.frameWidth(), ASCII: b.ascii, Frame: b.frame}
}

// frameWidth is how much room the console's own content has: the terminal, less
// the margin on BOTH sides (D-96, D-100).
//
// THREE COLUMNS EACH SIDE. The first cut took only the left, and the HUM LEAD saw
// the result immediately: "3 col right inset not respected by the tables" — every
// table ran to the terminal's edge while the boxes above stopped short of it. A
// margin on one side is not a margin, it is a shift.
//
// THE LAYOUT IS BUILT AT THIS AND THE MARGIN IS ADDED ONCE. Building at the
// terminal's width and insetting afterwards would push three columns of every
// row off the right edge.
func (b Broadcaster) frameWidth() int {
	return max(0, b.width-len(bcLeftInset)-bcRightInset)
}

// tableWidth is what a RAILED table gets: the frame, less the scroll control's
// own column and the blank one in front of it.
//
// HUM LEAD, UAT 2026-09-12: "Vertical scroll control on the right hand side needs
// to match the visual pattern of Observer; Table runs right up to the control,
// the control is immediately to left of the 2 col right global inset."
//
// OBSERVER'S OWN ARITHMETIC, and it is the reason the right margin is TWO and not
// three: `rail := o.TableRowLen(days) + 2` with the comment "UAT 9.2: one blank
// col between the last cell and the rail". Table, blank, control, two columns of
// air. The console was drawing THREE blanks and putting the control a cell short
// of Observer's, and the pool table was skipping the arithmetic entirely — which
// is why the two tables ended in different columns.
func (b Broadcaster) tableWidth() int { return max(0, b.frameWidth()-2) }

const (
	// bcLeftInset is Observer's own left margin, matched (HUM LEAD, 2026-09-12:
	// "we need the global 3 col left inset as well").
	bcLeftInset = "   "
	// bcRightInset is Observer's own right margin, which is NOT the left one.
	// The scroll control sits immediately inside it (HUM LEAD, 2026-09-12).
	bcRightInset = 2
)

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

// laneWidth IS GONE (D-80), and its seam is not. It was "the ONE place that
// number is decided" when the frame was one lane; the layout has three owners
// now, each of which knows what it is measuring — `cardBoxWidth` for a card
// (which carries D-51's right-rail seam), `orderWidth` for a row of the running
// order, `bandWidth` for the station's painted text. A fourth number that agreed
// with all three by coincidence is what put the band three cells too wide.

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
	// THE BOUND IS IN THE SHAPE (P10-02). The frame pads to the viewport's
	// height, and that count is knowable before the loop rather than re-asked
	// on every pass.
	if b.height > 0 {
		for range max(0, b.height-len(lines)) {
			lines = append(lines, "")
		}
	}
	// AND CUT TO IT, WHICH THE WIDTH SIDE HAS ALWAYS DONE (D-102). The rule below
	// is already stated — "the frame IS the viewport, in both dimensions" — and
	// only one dimension enforced it: height padded and never truncated, so a
	// region that mis-budgeted by a row drew past the bottom of the terminal.
	//
	// IT IS A BACKSTOP, NOT THE BUDGET. Each region still windows itself
	// (scheduledLines, poolLines), and this is what makes a mistake there a
	// missing row rather than a broken frame.
	if b.height > 0 && len(lines) > b.height {
		lines = lines[:b.height]
	}
	if b.width > 0 {
		for i, l := range lines {
			// TRUNCATED *AND* PADDED: the frame IS the viewport, in both
			// dimensions. Truncating alone leaves a ragged frame whose widest
			// line is whatever the longest lane happens to be — and
			// `render.Overlay` composites against that, so a window centred on
			// the TERMINAL landed past the frame's right edge and the composite
			// grew sideways instead of stacking (UAT, 2026-09-10).
			// AND INSET FROM THE LEFT, HERE AND NOWHERE ELSE (D-96). Observer
			// insets EVERY row by three — masthead, radio panel, ticker, controls
			// and table alike — and the console now matches it, so the operator's
			// eye finds the same left edge on both surfaces.
			//
			// ONE APPLICATION POINT, because an inset applied per region is an
			// inset each new region has to remember. The layout is built at
			// `frameWidth` and the frame adds the margin once.
			lines[i] = render.PadTo(bcLeftInset+render.TruncateCells(l, b.frameWidth()), b.width)
		}
	}
	// AND THE THEME'S FOREGROUND IS ARMED FOR THE WHOLE FRAME (D-108).
	//
	// HUM LEAD, UAT 2026-09-12: "Some of the Broadcaster UIs are not using
	// themeable token values for colors - particularly in the 'UP NEXT' section -
	// Watchpost Light has white lines and text when it should be inverted
	// appropriately."
	//
	// NOTHING WAS PAINTING THEM WHITE. They were painted by NOBODY — every
	// character this console draws without an explicit tint took the TERMINAL's
	// default foreground, which on a dark terminal is white whatever theme the app
	// is wearing. On the dark themes that happens to look right, so the whole
	// surface has been reading the terminal's palette and calling it the theme's.
	//
	// OBSERVER HAS ALWAYS DONE THIS, in `frameText`: it arms `TextBase` at the top
	// of the frame and re-arms it after every inner reset, so tinted spans keep
	// their colours and everything else is the theme's. This is that rule, applied
	// where the console finishes its frame — one place, like the inset above it.
	//
	// THROUGH `FgSGR`, NOT `TintDefault`. That helper hard-codes a `38;5;` prefix
	// and the Light theme's `TextBase` is TRUECOLOR — the one theme this finding
	// was reported against would have come out as a palette index.
	out := strings.Join(lines, "\n")
	if render.ColorOn() {
		out = render.TintKeeping(out, render.FgSGR(render.Tok(render.TextBase)))
	}
	return out
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
	out = append(out, b.faultNotice()...)
	out = append(out, b.muteNotice()...)
	// THE AIR BOX IS INSIDE THE STATION SECTION NOW (D-107), so nothing is drawn
	// here: `stationSection` carries it, painted with the section's own ground.
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

	// UP NEXT AND THE TAKEOVER, LEVEL WITH EACH OTHER (D-97). The region machinery
	// and the vertical rail retire here: LIVE went to the air box (D-95), the
	// SCHEDULED slots went to the table (D-94), and what is left is two boxes side
	// by side that the reference draws at the same height.
	out = append(out, b.chrome(b.readPair(), false, 0)...)
	// AND THE POOL BELOW IT (D-98) — the candidates the operator promotes FROM,
	// with enough weather to decide on them.
	//
	// ONE SCROLL CONTROL, AND IT BELONGS TO WHAT SCROLLS (D-106, HUM LEAD
	// 2026-09-12: "Location Pool Scrolls, Line-up doesnt"). D-104 spanned it over
	// both tables on the strength of the shared pointer; the running order does
	// not actually move, so the control was claiming a scroll that never happens
	// and the pool's headers were inside the window it drew. ▲ on the pool's
	// column titles, ▼ on its "Showing" line — Observer's own shape.
	// THE WEATHER IS INDEXED ONCE FOR THE WHOLE FRAME (D-120). Both tables join
	// against it — forty lookups that were forty linear scans.
	sched, pool := b.spans(len(out))
	// THE FIRST REGION THAT SCROLLS OWNS THE CONTROL, and each region is asked
	// rather than assumed. Reading only the pool's answer made the running
	// order's `from` a field nothing consulted — so a region could claim a scroll
	// and the frame would not draw it, which a mutant found by claiming one and
	// changing nothing.
	from := -1
	switch {
	case sched.from >= 0:
		from = sched.from
	case pool.from >= 0:
		from = len(sched.lines) + pool.from
	}
	out = append(out, b.chromeAt(joinSpans(sched, pool), from, pool.off, pool.total)...)
	// AND THE FRAME ENDS WHERE THE RUNNING ORDER DOES, as the reference draws it:
	// the frame closes under the scroll rail's ▼ and the rest of the screen is
	// empty, rather than carrying walled blank rows to the bottom of the
	// terminal. `clamp` still pads the view to the terminal's height, so
	// D-63's rule holds: the frame is the viewport, and `render.Overlay` still
	// composites against a full-height base.
	return append(out, b.inset()...)
}

// scrollQueue moves the queue's window by `by` rows and hands the console back.
//
// CLAMPED AT THE ENDS, NEVER WRAPPED. A list that jumped from its last row to
// its first would lose the operator's place — and on a running order, "where am
// I" is the question the numbers exist to answer.
//
// THE BOTTOM IS WHERE THE LAST CARD IS FULLY DRAWN, not where the rows run out:
// scrolling past it would leave the operator looking at air with the rail saying
// there is more.
func (b Broadcaster) scrollQueue(by int) Broadcaster {
	// THE ARROWS MOVE THE POINTER AND THE WINDOW FOLLOWS (D-101), which is what
	// the reference's own footer says they do — "[↑↓] Navigate" — and what
	// Observer does. Moving the WINDOW alone would let the operator scroll a list
	// they have no position in.
	n := b.rowCount()
	if n == 0 {
		return b
	}
	b.selected = max(0, min(n-1, b.selected+by))
	// THE WINDOW IS NOT MOVED HERE. It follows the pointer at RENDER time, where
	// the room is known (scheduledLines) — a offset chosen at the keystroke cannot
	// know how many rows fit, and the first attempt scrolled one way and never
	// came back.
	return b
}

// rowCount is how many rows the two tables hold between them — the space the
// pointer walks.
func (b Broadcaster) rowCount() int {
	return max(0, MainTrackSlots-bcScheduledFrom) + len(b.area.Pool)
}

// lineupSelection is the focused row WITHIN the running order, or -1.
func (b Broadcaster) lineupSelection() int {
	if b.selected < MainTrackSlots-bcScheduledFrom {
		return b.selected
	}
	return -1
}

// poolSelection is the focused row within the pool, or -1.
//
// THE POOL BEGINS WHERE THE RUNNING ORDER ENDS, which is the one place that
// arithmetic is written — `numPriority` is Observer's twin of it.
func (b Broadcaster) poolSelection() int {
	if at := b.selected - (MainTrackSlots - bcScheduledFrom); at >= 0 {
		return at
	}
	return -1
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

// THE LANE CAPTION RETIRED AT D-97, and with it `bcLaneLabel`, `laneLabelRow`
// and `cardGap`. "MAIN SCHEDULE (ROLLING WINDOW)" named a column of cards that no
// longer exists: LIVE went to the air box (D-95), the ordered slots to the table
// (D-94), and UP NEXT is one of a pair. What the reference captions now is the
// TABLE, and `bcScheduledHeading` is that caption.
//
// THE WORD IT OWED IS STILL OWED. FR-3.1 makes the slots a window onto a deeper
// schedule and nothing on the surface says "rolling" any more — the scroll rail
// implies it, which is weaker. Recorded here rather than lost.

// THE REGIONS RETIRED WITH THE CARD COLUMN (D-110). `bcRegion` and `bcRegions`
// divided the running order into named bands of cards; LIVE went to the air box
// (D-95), the ordered slots to the table (D-94), and UP NEXT is one of a pair
// (D-97). One card is not a region, and a table's regions are its group bands.

// LIVE IS NOT A CARD REGION EITHER (D-95). What is on the air is one row of the
// AIR BOX above the running order, beside the bed it is mutually exclusive with —
// so the exclusivity is drawn rather than described. D-89's standby box retires
// with it: the wording it carried is now the row's own empty state.

// SCHEDULED IS NO LONGER A CARD REGION (D-94). Slots 2 and up are a TABLE now —
// Observer's table, through Observer's own machinery — so the region that drew
// them as flat cards, and the vertical rail that named them, are gone. What the
// rail said in letters down the side, the table says in a heading above it.

// burstBody is what a takeover box lists: the hazards it would read, as a table
// (D-103).
//
// IT WAS PROSE UNTIL THE HUM LEAD SAW IT (UAT 2026-09-12): the Composer's header
// sentence, then each alert wrapped over two lines. That reads as a paragraph, and
// what the operator is doing is SCANNING a list to decide whether to let it
// interrupt the programme. The reference draws two columns and ten numbered rows.
//
// TEN ROWS WHETHER OR NOT THERE ARE TEN, which is the reference's own idiom and
// the same rule the running order follows: a slot is an ADDRESS, and the box does
// not change height because a hazard arrived.
func (b Broadcaster) burstBody(c lineup.Card, w, list int) []string {
	room := w - 2 - 2*len(bcCardInset)
	if room < 8 {
		return nil
	}
	if list < 1 {
		return nil
	}
	rows := make([]render.AlertRow, 0, list)
	for i := range list { // bounded by the box (P10-02)
		r := render.AlertRow{Num: fmt.Sprintf("%02d.", i+1)}
		if i < len(c.From) {
			// THE ARRIVAL'S OWN WORDS, ONE FACT PER COLUMN.
			r.Kind = hazardOf(c.From[i])
			r.Location = plaintext.Text(c.From[i].Subject)
		}
		rows = append(rows, r)
	}
	out := []string{""}
	for _, l := range strings.Split(b.opts().AlertTable(rows, room), "\n") { // bounded (P10-02)
		out = append(out, bcCardInset+l)
	}
	// AND THE WAY IN, AT THE BOTTOM, where the reference puts it and where the
	// UP NEXT card already puts its own.
	return append(out, "", bcCardInset+" "+b.opts().KeyCap("A")+"  Details / Full Read / Manage")
}

// hazardOf is the hazard an arrival names, WITHOUT the place it names after it.
//
// `Arrival.Headline` IS THE TAPE'S LINE, not the hazard's name: `tapeHead`
// composes it as `<title> · <location>` because the ticker is one line and has to
// carry both. This table has a LOCATION COLUMN, so the composition arrives back
// as the place said twice — which is what the HUM LEAD saw, "SEVERE THUNDERSTORM
// WARN…Harper, KS", the hazard cut short to make room for a repeat.
//
// IT TAKES THE TITLE BACK RATHER THAN CHANGING WHAT `Headline` MEANS. That field
// reaches the card, the ticker cue and the SPOKEN script; a hazard read on air
// without its location would be a far worse defect than a crowded column. The
// separator is the app's own constant, so this is a split on a known join.
func hazardOf(a lineup.Arrival) string {
	head := plaintext.Text(a.Headline)
	if at := strings.Index(head, render.HeadlineJoin); at >= 0 {
		head = head[:at]
	}
	return strings.ToUpper(head)
}

// bcAlertChrome is what the takeover box spends around its list: the two borders,
// the air above the table, the table's own header, the air below, and the control
// row. The TITLE ROW left at D-110 — the box names itself in its rule.
//
// THE LIST IS WHAT IS LEFT, so the box is exactly as tall as the card beside it
// (D-103). The reference draws ten rows because its UP NEXT card is that tall; a
// constant here would have made the two boxes disagree about their own height and
// cut whichever lost.
const bcAlertChrome = 6

// THE HAZARD'S TIMES RETIRED WITH THE PROSE (D-103). `burstWhen` drew
// "<LOCATION> • 09/12 16:02 - 09/12 18:00" on every alert line; the reference's
// table has three columns and none of them is a span. The times are still on the
// arrival and still reachable through the card.

// AND `priorityColumn` RETIRED WITH THE OVERLAY (D-97). It drew the takeover as a
// column of its own, as tall as the burst; the v3 pair draws it beside UP NEXT at
// UP NEXT's height. Two drawers of one box is exactly the shape the `dupes` gate
// is for, and only its tests were still calling this one.

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
	// THE SAME INSET ON BOTH SIDES (HUM LEAD, UAT 2026-09-11): "since it's a
	// colored bkg, let's ensure we have a consistent inset for content — 1 line
	// top/bottom, 3 col left/right." It was FOUR on the left and NONE on the
	// right, which reads as a band the text is sliding out of.
	//
	// PADDED TO THE WIDTH LESS THE INSET, THEN THE INSET ADDED — not padded to
	// the full width and trimmed, which would put the right-hand air inside the
	// paint and leave the row a different length from every other.
	// THE AIR BOX IS PART OF THIS SECTION (D-107).
	//
	// HUM LEAD, UAT 2026-09-12: "the LIVE NOW / BED Table is also supposed to be
	// INSIDE the station playing section … this data is also tied DIRECTLY to the
	// ON AIR state - so it should all be in 1 section."
	//
	// AND THAT IS WHAT ENDS THE DUPLICATION. The bed had a row here AND a row in
	// the box; one section means one place for it. What the box says — which of
	// the two is carrying — is the same question the STATION AIR row above asks,
	// so the region answers it once, in the order the reference draws.
	body := append(append([]string{""}, b.stationLine()...), b.airBox()...)
	for _, r := range append(body, "") {
		rows = append(rows, render.PadTo(bcSectionInset+r, b.frameWidth()-len(bcSectionInset))+bcSectionInset)
	}
	return o.Block(strings.Join(rows, "\n"), fg, bg)
}

// bandWidth is how much room the station band's TEXT has: the terminal, less
// the inset it keeps on each side.
//
// ONE OWNER, because the rows are built to it and the band is painted to it, and
// the two disagreeing by three cells is exactly what put the text hard against
// the right edge of a coloured band.
func (b Broadcaster) bandWidth() int {
	w := b.frameWidth() - 2*len(bcSectionInset)
	if w < 1 {
		return 0
	}
	return w
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

// bcAirBoundaries is FR-5.5 in the words its own line has room for, longest form
// first.
//
// A LADDER, BECAUSE ONE WORDING CANNOT SERVE EVERY WIDTH. The rungs are 97, 61
// and 22 cells and each is taken only if it fits the band ENTIRE. Measured on the
// real frame: the full sentence renders from 120 columns up, the middle form at
// 100, and 99 is below the console's floor — where the frame is a refusal rather
// than a console. So every DRAWABLE width states the boundary.
//
// THE MEASUREMENTS THIS PARAGRAPH ONCE QUOTED WERE THE OTHER PLACEMENT'S (D-160).
// Ruling C put the boundary beside the state and the gain control, where the
// spare space is 51 cells at 160 and 35 at 144 — so the shortened sentence
// rendered only from 160 up, leaving FR-5.5 dead at the console's own reference
// width: the D-153 defect reimplemented inside its own fix. The HUM LEAD then
// ruled it a line of its own, which is what removed the constraint the rung
// lengths were sized against. They are kept because 97 cells still does not fit
// every width — but not for the reason the old text gave.
//
// EVERY RUNG NEGATES THE SAME INFERENCE. The danger is an operator reading a
// confident ON AIR and concluding their antenna is radiating, so the half that
// can never be dropped is the one about the transmitter. "audio only" says what
// Watchpost does produce and is the first thing to go.
//
// AND IT MUST NOT READ AS CONFIGURATION STATE, which the first wording did. The
// BROADCASTING FROM row two lines down already says "(no transmitter set)" when
// the operator has not chosen one — so "no transmitter observed" beside it reads
// as the same complaint, and the operator concludes that SETTING a transmitter
// will clear it. That inverts FR-5.5: the boundary is not a setup step they have
// missed, it is a permanent property of the product. "unverified" is true of
// every station, configured or not.
// A FUNCTION RATHER THAN A PACKAGE VARIABLE (P10-06), which is the convention
// `report.all` already states for the same reason: an immutable table with no
// run-time writer, and Go has no const slice. A package-level var here would be
// a rung of a SAFETY sentence that anything in the package could reassign.
func bcAirBoundaries() []string {
	return []string{
		"ON AIR means audio is leaving this program; Watchpost cannot verify a transmitter is carrying it.",
		"Watchpost cannot verify a transmitter is carrying this audio.",
		"transmitter unverified",
	}
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
	state, to := "STOPPED", "ON AIR"
	switch b.power {
	case lineup.Running:
		// BOLD WHITE ON THE SECTION'S RED (HUM LEAD, 2026-09-15): "When
		// Broadcaster is Actively broadcasting, let's make this string *** ON
		// AIR · BROADCASTING *** BOLD WHITE."
		//
		// THE SECTION'S OWN TEXT TOKEN, NOT A NEW WHITE. `stationTone` already
		// paints this band `AlertModalText` on `TickerEmergencyBG`, and
		// AlertModalText IS white — so this states the tone the row already
		// wears and adds the WEIGHT. Registering TextBright against the
		// emergency ground instead would have lifted TextBright everywhere it
		// is painted, which `aaPairs` warns about in as many words: "widening a
		// shared token's ground set changes it everywhere".
		//
		// WEIGHT IS NOT CONTRAST, so this changes no AA answer: the pair is
		// already in production and is unchanged.
		state = render.Bold(render.Tint("*** ON AIR "+g.Dot+" BROADCASTING ***", render.Tok(render.AlertModalText)))
		to = "STANDBY"
	case lineup.OffAir:
		state = "STANDBY (DEAD AIR)"
	}
	// A NOTICE IS THE ONLY PROSE THIS ROW CARRIES. A refusal describes something
	// the operator JUST DID, and the row they are looking at has to answer the key
	// they just pressed; a sentence describing the state instead says what the row
	// above it already names.
	// THE BAND'S TEXT COLUMN: the terminal, less the inset on BOTH sides (D-80).
	// It read `sectionWidth()` — the width of a region inside the frame's walls,
	// which this band no longer has since colour became its edge (D-70) — so
	// every row came out three cells too wide and the right-hand inset had
	// nowhere to go.
	lane := b.bandWidth()
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
	rows := []string{
		// THE GAIN RIDES THE STATE'S OWN ROW, where the reference draws it: how
		// loud the station is and whether it is on the air are one question asked
		// twice, and the operator checks them together.
		render.PadBetween(render.TruncateCells(label("STATION AIR:")+state, max(0, room)), gain, lane),
		// THE TRANSMITTER'S IDENTITY MOVED HERE FROM THE MASTHEAD (D-71). It is
		// a fact about the STATION; the masthead is what both surfaces share.
		render.PadBetween(label("BROADCASTING FROM:")+b.transmitterRow(max(0, lane-bcLabelCells-render.Width(hint)-2)), hint, lane),
	}
	// FR-5.5'S BOUNDARY GETS A LINE OF ITS OWN (F-109, HUM LEAD 2026-09-16:
	// "Give F109C its own line in the section area then if it creates new
	// defects").
	//
	// RULING C PUT IT ON THE STATION AIR ROW AND THAT CREATED TWO. Measured on
	// the real frame, the row's spare space beside the state and the gain control
	// is 35 cells at 144 — the console's own reference width — so the shortened
	// sentence rendered only at 160 and above. FR-5.5 would have been dead at
	// every width the console is actually used at: D-153's defect reimplemented
	// inside its own fix. And the wording that DID fit sat two rows above
	// "(no transmitter set)", so the operator reads a permanent property of the
	// product as a setup step they missed, and concludes that choosing a
	// transmitter clears it — the requirement inverted.
	//
	// A LINE OF ITS OWN HAS ROOM TO SAY IT PROPERLY, which is the whole gain:
	// the full sentence names what IS happening (audio is leaving this program)
	// before what cannot be known (whether a transmitter carries it), and a
	// boundary that only ever half-appeared now appears whole at every width.
	//
	// ONLY WHILE RUNNING, because that is the claim that can mislead. STOPPED and
	// STANDBY assert nothing about a transmitter, and a standing line there would
	// be the prose row D-107 deliberately removed — "it explained a state the row
	// above already names". The band is one row taller ON AIR, which is the state
	// that already changes the band's colour entire.
	if b.power == lineup.Running {
		// WHOLE OR NOT AT ALL, RUNG BY RUNG. A safety sentence cut mid-word is a
		// different claim, not a shorter one: on the old placement the cut
		// rendered as "· audio on" at 120 cells — complete, reassuring, and the
		// opposite of the sentence it came from. Each rung is taken only if it
		// fits ENTIRE, longest first, and the shortest fits any width this
		// console draws at.
		for _, words := range bcAirBoundaries() { // bounded by the ladder (P10-02)
			if render.Width(words) <= lane {
				rows = append(rows, render.PadTo(render.Tint(words, render.Tok(render.AlertModalText)), lane))
				break
			}
		}
	}
	// THE BED'S OWN ROW IS GONE (D-107). It was the SECOND place this console
	// drew the bed — the selector and its state here, and the same selector and
	// the same state in the LIVE NOW / RELAY BED box below — which the HUM LEAD
	// called out exactly: "the BED is now duplicated in the UI - which is
	// confusing - this data is also tied DIRECTLY to the ON AIR state - so it
	// should all be in 1 section". The box is that one place.

	// AND THE STANDING PROSE WITH IT. "the programme is stopped; hazards still
	// read" explained a state the row above already names, and the reference has
	// no line for it. A NOTICE STILL GETS ONE, because a refusal answers a key the
	// operator just pressed and has to be somewhere they are looking.
	// FR-5.5's BOUNDARY WAS ASSIGNED AND NEVER RENDERED, AND IT IS NOW BOTH
	// (D-153 found it, F-109 ruled it, D-160 corrected this paragraph).
	//
	// WHAT D-153 FOUND: `why` carried a sentence per state and this row is drawn
	// ONLY when a `statusNote` exists — by which point the note had already
	// replaced it. So every per-state sentence was dead, including the one FR-5.5
	// exists for. D-107 had removed the standing prose row and taken the boundary
	// with it, while `stationLine`'s own comment still claimed the boundary is
	// stated "HERE, where they read it — not only in a design document".
	//
	// WHAT CHANGED: the HUM LEAD ruled the placement (F-109), the boundary now
	// renders on a line of its own above, and the dead `why` values were deleted.
	// The two layout gates this paragraph once cited as reasons NOT to fix it —
	// `TestTheBedRidesInTheStationSection` and `TestTheStationBandIsEvenlyInset` —
	// were updated to the ruled shape rather than worked around.
	//
	if b.statusNote != "" {
		rows = append(rows, render.TruncateCells(label("")+b.statusNote, max(0, lane)))
	}
	return rows
}

// bedAvailable is whether the station has any relay it could actually carry
// (D-117).
//
// THE COUNT IS THE PRODUCER'S ANSWER, not the console's guess: it is how many
// transmitters within the station's reach the directories actually stream. The
// console holds no region and no directory, so it is told.
//
// UNTOLD IS AVAILABLE. `BedMsg` arrives once the resolve lands, and a console
// that greyed the control out until then would refuse a key that is about to
// work — which reads as a broken button rather than as a pending answer.
func (b Broadcaster) bedAvailable() bool { return !b.bedRelaysTold || b.bedRelays > 0 }

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
	relay := b.bed.Relay
	if relay == "" {
		relay = bcNoRelay
	}
	// AND A STATION WITH NOTHING TO TUNE OFFERS NO SELECTOR (D-117). Arrows over
	// an empty list are a control that cannot act, and the row has something
	// truer to say with the cells.
	if !b.bedAvailable() {
		return bcNoRelaysHere
	}
	// SHIFTED (D-111). The bare arrows belong to the card's PRESENTER now, and a
	// chip here that read `←` would name a key that steps a different control —
	// which is worse than no chip, because the operator would try it.
	return o.KeyCap("⇧←") + "  " + relay + "  " + o.KeyCap("⇧→")
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
	// "BROADCASTING FROM:" IS THE LONGEST OF THEM (D-107), at eighteen.
	bcLabelCells = 21

	// bcNoRelay is what the bed's row says before a relay is tuned. The relay's
	// own description — its call sign, frequency and distance — arrives with the
	// bed's state, which the schedule does not publish yet (F-79).
	bcNoRelay = "(no relay tuned)"

	// bcNoRelaysHere is what the row says when nothing STREAMS within the
	// station's reach (D-117).
	//
	// HUM LEAD, 2026-09-13: "If none exist in that area - we should probably tell
	// the broadcaster there is no valid relays for their area and disable the BED
	// option so the Operator cannot choose something that will broadcast dead
	// air."
	//
	// IT IS A DIFFERENT SENTENCE FROM `bcNoRelay`, and the difference is the
	// point: one says "you have not chosen yet" and the other says "there is
	// nothing to choose". Told apart, the second is actionable — widen the bed's
	// fence, or accept that this station has no relay.
	bcNoRelaysHere = "no relays reach this station"
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
		Name: "headline",
		Fill: true,
		// LEFT, FROM THE v2 REFERENCE (D-87). A centred title floated between
		// the badge and the border while every other row of the card began at
		// the same inset — so the one line naming the card was the one line that
		// did not line up with it.
		Alignment:         "left",
		Truncatable:       true,
		TruncatedMinWidth: 8,
		TruncationTail:    g.Ellipsis,
	}
	// THE CARD'S OWN INSET, AS A COLUMN (D-87). Every row of a card's interior
	// begins three cells in, and until now the title row began at zero — so the
	// one line naming the card was the one line that did not line up with it.
	//
	// A COLUMN RATHER THAN A PREFIX, because the badge is right-anchored to the
	// ROW: prefixing would have pushed the row three cells wide and taken the
	// handle with it.
	inset := components.ColumnDefinition{Name: "inset", Width: len(bcCardInset)}
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
		row:    components.NewDataTableRow(lane, []components.ColumnDefinition{inset, headline}),
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
	headline := kindFirst(cardTitle(c, l.g), l.lane, l.g)
	// A TAKEOVER NAMES ITSELF IN ITS BORDER (D-87), so repeating the headline
	// inside it would say the same thing twice on a box whose whole job is to be
	// read at a glance. What the row still carries is the badge and the handle.
	if c.Slot == lineup.BreakingAlert {
		// AND ITS HANDLE IS AT THE BOTTOM (D-103), where the reference draws it:
		// "A  Details / Full Read / Manage". A chip in the title row as well would
		// be the same key offered twice on one box.
		headline = ""
		handle = ""
	}
	data := map[string]string{"headline": headline}
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
	// AND A CARD WITH NO HANDLE WEARS NO CHIP. An empty `KeyCap` paints an empty
	// chip — "[]" without colour — which is a control offering no key: worse than
	// the absent one it stands for. The takeover's way in is the row at the BOTTOM
	// of its box (D-103), so its title row has no handle to show.
	chip := l.g.Bullet + badge + l.g.Bullet
	if handle != "" {
		chip += "  " + l.o.KeyCap(handle)
	}
	row.SetBadge(chip+" ", 2)
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

// `box` RETIRED WITH THE CARD COLUMN (D-110). It drew a card with no interior —
// the flat, ordered slots of the pre-table running order — and those became rows
// of `LineupTable` at D-94. Its one remaining caller was itself.

// boxOf is the card, with whatever the region puts INSIDE it below the first
// row (D-68).
//
// THE READ CARDS ARE TALLER THAN THE SCHEDULED ONES, which is the reference and
// was the HUM LEAD's UAT (2026-09-10): "the LIVE CARD should be bigger to
// support showing at least most the script being played … UP NEXT should also be
// bigger." A card the operator READS FROM needs the words on it; a card they are
// merely deciding the ORDER of needs its name and its handle.
//
// boxOf is a card as a BOX: the borders name it, and the body is whatever the
// caller puts inside (D-110).
//
// THERE IS NO TITLE ROW (D-110). The reference puts the headline and the badge in
// the RULE and the handle at the BOTTOM beside the presenter, so a row carrying
// all three would say what the frame and the footer say, and cost
// the manifest a line to do it.
//
// ONE DRAWER FOR BOTH, because everything except the interior is the same card.
// A second box function would be a second place for a corner or a tint to drift,
// which is the D-56 shape this file has already paid for once.

// inner is the box's interior width — everything between the two rails.
//
// ONE OWNER, because two things measure it now: the box that draws the borders
// and the script window that has to fit inside them. `bandWidth` was the same
// lesson at D-80 — a fourth number agreeing with three others by coincidence is
// what put the station band three cells over its frame.
func (l cardLane) inner() int { return l.lane - 2 }

func (l cardLane) boxOf(c lineup.Card, badge string, body []string) []string {
	return l.shell(cardBoxTitle(c, l.g), l.badgeOf(badge), body, cardTone(c))
}

// badgeOf is how a card's grade reads in the box's own rule: `• STANDARD •`.
func (l cardLane) badgeOf(badge string) string {
	if badge == "" {
		return ""
	}
	return l.g.Bullet + " " + badge + " " + l.g.Bullet
}

// shell is a box of the card's shape, around whatever is put in it, painted on
// one ground.
//
// EXTRACTED AT THE SECOND CALLER (D-89), which is the standing modularity rule.
// The LIVE slot's empty state is a box with no card in it — no title row, no
// badge, no handle — and drawing it through a second copy of these six lines
// would be two places for a corner, a rule or a tint to drift. What differs
// between a card and an empty slot is the CONTENTS, and that is now the only
// thing that differs.
func (l cardLane) shell(boxTitle, boxBadge string, body []string, ground string) []string {
	if l.lane < 4 {
		return nil
	}
	inner := l.inner()
	// THE MASTHEAD'S BOX (D-85, HUM LEAD 2026-09-11): "Remove the rounded
	// corners -> straight corners … All Main track cards should have BOLD lines
	// (like the masthead)." That is `render.HeavyBox`, which the masthead has
	// drawn since 0.13.0 — shared rather than copied, so the console and the
	// header cannot come to disagree about what a border looks like.
	bx := render.HeavyBox(l.o.ASCII)
	rows := []string{bx.TL + boxRule(bx.Rule, boxTitle, boxBadge, inner) + bx.TR}
	for _, r := range body { // bounded by the card's own height (P10-02)
		rows = append(rows, bx.Rail+render.PadTo(render.TruncateCells(r, inner), inner)+bx.Rail)
	}
	rows = append(rows, bx.BL+strings.Repeat(bx.Rule, inner)+bx.BR)
	// THE WHOLE BOX IS PAINTED, BORDERS INCLUDED (D-86). A card is one object;
	// a ground that stopped at the border would draw a coloured window inside a
	// colourless frame, which reads as a fill rather than as a card.
	for i, r := range rows { // bounded by the card's own height (P10-02)
		rows[i] = render.TintKeeping(r, ground)
	}
	return rows
}

// bcStandbyNotice is the HUM LEAD's wording, verbatim (D-89): "empty state needs
// to be a grey box with a centered text of: NO REPORTS READ OR ACTIVE IN STANDBY
// MODE". The BOX retired at D-95; the SENTENCE is what the LIVE row says.
const bcStandbyNotice = "NO REPORTS READ OR ACTIVE IN STANDBY MODE"

// `standbyBox` RETIRED WITH THE LIVE CARD (D-110). D-89 drew the empty LIVE slot
// as a grey box — "NO REPORTS READ OR ACTIVE IN STANDBY MODE" — and D-95 made
// LIVE one ROW of the air box, where `liveLine` says the same sentence in the
// space a row has. The words survived; the box around them did not.

func cardTitle(c lineup.Card, g render.Glyphs) string {
	head := plaintext.Text(c.Headline)
	kind := strings.ToUpper(c.Slot.String())
	switch {
	case kind == "":
		return head
	case head == "":
		return kind
	}
	// THE SEPARATOR COMES FROM THE GLYPH SET, never a literal — the parity gate
	// found the first draft's bullet, which is the same catch it made on the
	// station line and on this batch's em-dash.
	return kind + " " + g.Bullet + " " + head
}

// cardRuleTitle is what a box's rule NAMES: the card, with a fabricated event
// saying so before anything else (D-110).
//
// THE MARK IS IN THE RULE, WITH THE TITLE (D-110), and it is non-truncatable so
// the headline can never eat it. Drawn anywhere the title is not, the mark would
// simply be absent, which is the screenshot hazard FR-4.4 exists
// to prevent: a test event that looks exactly like a real one.
//
// FIRST, AND BEFORE THE HEADLINE CAN GIVE WAY. `boxRule` drops the whole title
// when the box is too narrow for it, so a rule that fits ANYTHING fits the mark —
// and a box too narrow for even that draws no title at all rather than a title
// with the mark cut off it, which is D-55's own direction.
func cardRuleTitle(c lineup.Card, g render.Glyphs) string {
	title := cardTitle(c, g)
	if !c.Test {
		return title
	}
	if title == "" {
		return testEventMark
	}
	return testEventMark + " " + title
}

// cardBoxTitle is what a card carries in its TOP RULE, or nothing.
//
// ONLY THE TAKEOVER HAS ONE (D-87), and the reference is emphatic about it:
// `┏━━━ ! ALERT ! - TAKEOVER ━━━┓`. A hazard interrupting the programme
// announces itself in the frame around it, not in a row inside it — which is
// what lets the box be read as an interruption at a glance, from the shape
// rather than from the words.
func cardBoxTitle(c lineup.Card, g render.Glyphs) string {
	if c.Slot == lineup.BreakingAlert {
		// A TAKEOVER NAMES WHAT IT IS, NOT WHICH ONE. It interrupts the programme,
		// and what the operator needs off the frame is that something has — the
		// hazards themselves are the list inside it (D-103).
		if c.Test {
			return testEventMark + " " + bcTakeoverTitle
		}
		return bcTakeoverTitle
	}
	// EVERY OTHER CARD NAMES ITSELF IN THE RULE (D-110): the reference puts
	// `LOCATION REPORT • Oceanside, CA 92057` in the border.
	return cardRuleTitle(c, g)
}

// bcTakeoverTitle is the takeover box's own name, from the v2 reference.
const bcTakeoverTitle = "! ALERT ! - TAKEOVER"

// boxRule is a top border carrying a title at its left, in the masthead's shape.
//
// THE CORNERS ALWAYS LAND. A title wider than the rule is dropped rather than
// pushing a corner off the row — the same rule `render.BoxTitled` states, which
// is where this shape comes from.
func boxRule(mark, title, badge string, inner int) string {
	if title == "" || inner < render.Width(title)+6 {
		return strings.Repeat(mark, inner)
	}
	// THE TITLE READS AS A MODAL'S DOES (D-118, HUM LEAD 2026-09-13: "Let's make
	// the Title 'LOCATION REPORT * <location>' Bold and White like the Modals").
	//
	// `ModalTitle` IS THAT TREATMENT, and it is the panel's own: `PanelColored`
	// tints every window title with it, so a box that named itself in the ground's
	// base grey was the one titled thing on the frame not doing so. Now that the
	// box wears the modal's tile (D-114) the difference was the only thing left
	// telling them apart.
	//
	// THE RULE'S MARKS KEEP THE GROUND'S TONE. What is being picked out is the
	// NAME, not the border it sits in.
	head := strings.Repeat(mark, 3) + " " + render.Tint(title, render.Tok(render.ModalTitle)) + " "
	// AND THE BADGE RIDES THE SAME RULE, at the right (D-110). The reference
	// draws both boxes that way — `┏━━ LOCATION REPORT • Oceanside, CA 92057 ━━━
	// • STANDARD • ━━━┓` — so what the card IS and how it is GRADED are read off
	// the frame, and the manifest has the row instead.
	//
	// IT GIVES WAY FIRST, because the title says WHICH card and the badge only
	// says what kind: a box too narrow for both keeps the one that identifies it.
	tail := ""
	if badge != "" && inner-render.Width(head) >= render.Width(badge)+6 {
		tail = " " + badge + " " + strings.Repeat(mark, 3)
	}
	return head + strings.Repeat(mark, inner-render.Width(head)-render.Width(tail)) + tail
}

// cardTone is the ground a card is painted on, and the empty string for a slot
// that holds no card (D-86).
//
// AN EMPTY SLOT IS NOT A CARD. The waiting placeholder and the LIVE slot on a
// station at rest carry no identity, and painting them would give the operator a
// coloured card that is not there.
//
// A HAZARD WEARS ITS CATEGORY, and it is the [w] window's own tint rather than a
// second palette: "Alert cards should be color coded to match the most severe
// alert based on the [w] category bkgs in Observer (they should match)" (HUM
// LEAD, 2026-09-11). `category.Of(...).Tint` IS that background, so the two
// match by construction and cannot drift.
//
// EVERYTHING ELSE WEARS ITS ORIGIN, narrow by ruling: the station's own
// proposals on one ground and the operator's requests on another.
func cardTone(c lineup.Card) string {
	if c.ID == "" {
		return ""
	}
	return render.Tok(render.CardText) + ";" + render.Tok(cardGroundToken(c))
}

// cardGroundToken is the ground a card is painted on — the ONE owner of that
// decision, because the card and the WINDOW it opens must not disagree.
//
// HUM LEAD, UAT 2026-09-13: "if the Card on the layout is the [w] orange - when I
// press <shift+a> that modal should MATCH the tone, not be the blue that is
// currently is." The card wore its category and the window it opened wore the
// standard modal ground, so the same hazard was two colours one keypress apart.
func cardGroundToken(c lineup.Card) render.Token {
	if c.Slot == lineup.BreakingAlert {
		if worst, ok := worstCategory(c.From); ok {
			return category.Of(worst).Tint
		}
		// A BURST WITH NO ARRIVAL TO READ A CATEGORY FROM keeps the ordinary
		// card ground rather than guessing at a severity. Guessing paints a
		// hazard the wrong colour, which is worse than painting it no colour.
		return render.CardBG
	}
	if c.Origin == lineup.FromOperator {
		return render.CardOperatorBG
	}
	return render.CardBG
}

// cardWindowGround is the ground the card's WINDOW floats on, or "" for the
// standard modal tone.
//
// ONLY A HAZARD CARRIES ITS GROUND INTO THE WINDOW, and that is the narrow
// reading of the ruling on purpose: a burst is the card whose colour MEANS
// something — it is the severity the operator is deciding about. An ordinary
// report's ground is `CardBG`, which is not a severity and not a signal, and
// carrying it in would restyle every other window to say nothing new.
func cardWindowGround(c lineup.Card) string {
	if c.Slot != lineup.BreakingAlert {
		return ""
	}
	if _, ok := worstCategory(c.From); !ok {
		return ""
	}
	return render.Tok(cardGroundToken(c))
}

// worstCategory is the most severe category among a card's alerts.
//
// BY READ RANK, which is the registry's own severity order — the same ladder
// the Producer plans a burst with. A second notion of "most severe" here would
// paint a card one colour and read it in another order.
func worstCategory(from []lineup.Arrival) (category.Category, bool) {
	worst, found := category.Category(0), false
	for _, a := range from { // bounded by the burst (P10-02)
		spec := category.Of(a.Category)
		if spec.ReadRank == 0 {
			continue // not a category the rail reads
		}
		if !found || spec.ReadRank < category.Of(worst).ReadRank {
			worst, found = a.Category, true
		}
	}
	return worst, found
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
	sep := " " + g.Bullet + " "
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
