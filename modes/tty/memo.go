package tty

// memo.go — the body memo (quality pass Q3, plan §2.5, PF-8, L1-F1 option
// 2, L4-F9): the two location tables are ~42 % of a frame and change only
// when one of their inputs does, so they are rendered once per input
// change and reused on every tick, marquee and visualizer frame between.
//
// The key is complete by construction — one field per input the tables
// read (R2-4) — and the slot is a pointer allocated at construction
// because View() is a value receiver (never package state, P10-06).
// Bound: one entry, the last key.

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// bodyKey is every input the two tables read. Adding an input to row(),
// recentSection() or the layout means adding a field here.
//
// THE COMPLETENESS GUARD IS memo_completeness_test.go, NOT A TABLE. This said
// the invalidation table in memo_test.go "has one row per field"; it had 15
// rows for 22 fields, and had never had one per field (red team 2026-09-05,
// R-14). A hand-kept register that tells the next author it is complete is
// worse than no register: they add the field, skip the row, and believe the
// table caught it. The enumerated table is still useful for the cases it
// names; it is not the guarantee.
type bodyKey struct {
	width, height  int
	compact        bool
	radioH, alertH int
	controlRows    int
	window, days   int
	snap, recent   *snapshot.Snapshot
	selected       int
	recentOff      int
	units          render.Units
	ascii          bool
	thinBands      bool                 // the bands' height
	radioKey       snapshot.LocationKey // the ▶ row
	radioPlaying   bool                 // ▶ clears on stop while radioKey stays
	radioRepeat    RepeatMode           // the ∞ mark on every row
	theme          uint64               // render.ThemeGeneration: every Tok() tint in the cells
	fireBoldMW     float64              // the bold-◆ rule (Setup)
	shimmer        int                  // the loading frame while a row loads, else 0
	// lookup is the pending looked-up location, by key — the table draws a row
	// for it before any snapshot carries it, so the memo has to see it appear
	// and disappear or the row lands a keystroke late (0.14.0).
	lookup snapshot.LocationKey
}

// bodyMemo is the single slot. hits/misses are read by the tests and the
// diagnostic dump; they are not policy.
// memoStats is the counter half every memo slot carries. ONE OWNER (metric D,
// 2026-09-08): bodyMemo and modalMemo each had their own nil-check-lock-return
// accessor, and a third memo would have written a third. Embedding it means the
// next one inherits `counts()` instead of copying it.
type memoStats struct {
	mu           sync.Mutex
	hits, misses int
}

// counts reports the slot's hit/miss counters; the zero value of a nil slot is
// (0, 0), which is what both callers already returned.
func (m *memoStats) counts() (hits, misses int) {
	if m == nil {
		return 0, 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hits, m.misses
}

type bodyMemo struct {
	memoStats
	ok               bool
	key              bodyKey
	priority, recent string
}

// bodyKeyFor derives the key from the model and this frame's layout.
func (d Dashboard) bodyKeyFor(fl frameLayout) bodyKey {
	k := bodyKey{
		width: fl.o.Width, height: d.height, compact: fl.compact, radioH: fl.radioH, alertH: fl.alertH,
		controlRows: fl.controlRows, window: fl.window, days: fl.days,
		snap: d.snap, recent: d.recent, selected: d.selected, recentOff: d.recentOff, lookup: d.lookupKey(),
		units: d.units, ascii: fl.o.ASCII, thinBands: fl.o.ThinBands,
		radioKey: d.radioKey, radioPlaying: d.radioPlaying, radioRepeat: d.radioRepeat,
		theme: render.ThemeGeneration(), fireBoldMW: d.fireBoldMW(),
	}
	if d.anyLoading() {
		k.shimmer = ((d.frame % 4) + 4) % 4 // the LoadingDots phase (units.go)
	}
	return k
}

// tables renders the priority table and the RECENT section, or returns
// the memoised pair when nothing they read has changed.
func (d Dashboard) tables(fl frameLayout) (priority, recent string) {
	key := d.bodyKeyFor(fl)
	m := d.memo
	if m == nil {
		return d.priorityTable(fl), d.recentSection(fl)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ok && m.key == key {
		m.hits++
		return m.priority, m.recent
	}
	priority, recent = d.priorityTable(fl), d.recentSection(fl)
	m.ok, m.key, m.priority, m.recent = true, key, priority, recent
	m.misses++
	return priority, recent
}

// memoCounts reports the body slot's hit/miss counters (0, 0 without a slot).
func (d Dashboard) memoCounts() (hits, misses int) { return d.memo.stats().counts() }

// lookupKey is the pending lookup's identity, zero when none is waiting.
func (d Dashboard) lookupKey() snapshot.LocationKey {
	if d.lookupRef == nil || d.lookupIndex() >= 0 {
		return ""
	}
	return snapshot.Key(*d.lookupRef)
}

// anyLoading reports whether any row still shows the loading shimmer —
// the tick's reason to keep running and the memo's reason to key on the
// frame (row(): a location loads until its observation and daily land).
func (d Dashboard) anyLoading() bool {
	if d.lookupRef != nil && d.lookupIndex() < 0 {
		return true // the placeholder row a lookup draws before its data lands
	}
	for _, sn := range []*snapshot.Snapshot{d.snap, d.recent} {
		if sn == nil {
			continue
		}
		for i := range sn.Locations {
			if rowLoading(&sn.Locations[i]) {
				return true
			}
		}
	}
	return false
}

// rowLoading is the one definition of "this row is still loading"
// (UAT 18.2): observation or daily forecast still pending.
// rowLoading: shimmer while the data is still COMING, never after it has been
// asked for and not arrived (issue #13).
//
// It used to be the first clause alone, and an empty location is empty in
// exactly the same way whether the feed has not answered yet or has answered
// and had nothing for this place — so a location the API does not cover
// shimmered for ever, across restarts, reading as "still loading" until the
// listener removed the row themselves.
//
// WeatherAsOf is what makes the two distinguishable: the reference provider
// stamps it when a fetch COVERING this location completes. Once it is set, a
// missing value is a fact rather than a wait, and the row falls through to the
// honest "n/a" the post-load path already draws (UAT 18.2). Nothing new is
// rendered — the whole defect was a flag that could never go false.
func rowLoading(loc *snapshot.Location) bool {
	if !loc.WeatherAsOf.IsZero() {
		return false
	}
	return loc.Harmonized.Source.Provider == "" || len(loc.Daily) == 0
}

// --- the modal memo (0.13.0, FR-10) ---

// modalKey is every input any window reads (SAM-D-13): one field per input,
// with the per-window extras projected only while that window is open, so
// a Setup keystroke never invalidates the severe table and a status-modal
// second never invalidates Help.
//
// THE COMPLETENESS GUARD IS memo_completeness_test.go (F-30, mechanised): it
// derives the fields by reflection and asserts that two dashboards which RENDER
// differently cannot share a key. This said severe_test.go's table "has one row
// per field" — 27 rows for 34 fields, and the nine missing included faultFocus,
// faultLeft, debugFocus and severeReadPause, the exact four whose absence froze
// three windows in UAT this release (R-14).
type modalKey struct {
	modal         modal
	opts          render.Opts // width, units, ascii, bands — Frame zeroed (the shimmer keys separately)
	width, height int
	scroll        int
	selected      int
	alertIdx      int
	// THE PENDING LOOKUP, because `selected` cannot stand in for it. Every
	// lookup focuses the same index — modal_location.go sets
	// `d.selected = len(watch)`, the first RECENT row — so two lookups in a row
	// produce an IDENTICAL key while the location differs, and the memo replays
	// the previous location's Details until its data lands and something else
	// moves the key. That is issue #11: "Lake Henshaw shows up, then Miami
	// magically swaps in". bodyKey has carried this since the fix; modalKey
	// never did (red-team round 3: modalKey's invalidation table had 27 rows
	// for 34 fields).
	lookup                    snapshot.LocationKey
	snap, recent              *snapshot.Snapshot
	severeGen                 uint64
	severeTab                 SevereTab
	severeRow                 int
	severeDetail              bool
	breakingID                string // the ▶ mark on the event being read (while the window is open)
	readingKey                string // the ▶ on the event being read
	addMode, addQuery, addErr string
	radioVoice                string
	voiceIdx, nvoices         int
	// THE CARD WINDOW'S IDENTITY AND GENERATION (D-88). cardRows is a func and a
	// key must be comparable, so the generation is what carries what it draws —
	// the same stand-in setupGen makes for Setup's two maps.
	cardID   string
	cardGen  int
	setupGen uint64   // Setup's state, by generation while it is open
	stats    [32]byte // the [S] stats, fingerprinted while it is open
	darkBG   bool
	theme    uint64
	minute   int64 // Details\' "N min ago" labels, projected while Details is open (a label may lag its rollover ≤ 59 s)
	second   int64 // [S] ages, while it is open
	shimmer  int   // Details' LoadingDots while a row loads
	// faultFocus and faultLeft are the relay-fault window's cursor and clock,
	// BOTH OF WHICH THE FRAME SHOWS. Absent from this key the window rendered
	// once and the memo replayed that frame for the life of the window: the
	// arrows moved the cursor in the model and nothing on screen followed, and
	// the countdown counted down to a fall-through that fired while the display
	// still read <10>. See the case below.
	faultFocus, faultLeft int

	// faultHeld is the clock's hold (FR-6.4). The footer reads "Auto Close
	// held" instead of a number, so it is a third thing the frame shows that
	// moves — and F-30's guard caught its absence the moment it was added,
	// which is what that guard is for.
	faultHeld bool
	// debugFocus is the ctrl+d window's cursor — the same rule, and it had the
	// same hole: its arrows moved the model and the memo replayed the frame.
	debugFocus int
	// severeReadPause is the severe window's Pause/Play state, and the THIRD
	// instance of the same omission (red team 2026-09-05). The frame reads it
	// twice — the chip flips Pause<->Play, and every row's glyph — while pausing
	// leaves readingKey unchanged, so the key was identical and the memo
	// replayed the pre-pause frame. modalSevere is not in tickNeeded either, so
	// nothing else invalidated it.
	severeReadPause bool
}

// modalMemo is the single slot.
type modalMemo struct {
	memoStats
	ok  bool
	key modalKey
	out string
}

// modalKeyFor derives the key from the model.
func (d Dashboard) modalKeyFor(o render.Opts) modalKey {
	o.Frame = 0
	k := modalKey{
		modal: d.modal, opts: o, width: d.width, height: d.height, scroll: d.modalScroll, selected: d.selected, alertIdx: d.alertIdx,
		lookup: d.lookupKey(),
		snap:   d.snap, recent: d.recent,
		severeGen: d.severe.Gen, severeTab: d.severeTab, severeRow: d.severeRow, severeDetail: d.severeDetail,
		addMode: d.addMode, addQuery: d.addQuery, addErr: d.addErr,
		radioVoice: d.radioVoice,
		// THE CARD WINDOW'S IDENTITY AND ITS GENERATION (D-88). What the window
		// draws comes from a func, which a key cannot hold — so the generation
		// stands in for it; see Dashboard.cardGen.
		cardID: d.cardID, cardGen: d.cardGen,
		voiceIdx: d.voiceIdx, nvoices: len(d.voiceList),
		darkBG: d.darkBG, theme: render.ThemeGeneration(),
	}
	switch d.modal {
	case modalSetup:
		// The GENERATION, not the state. Formatting the whole struct — which
		// carries two maps — ran on every frame and grew with them; it was the
		// largest single thing this window cost. Every writer of the Setup
		// state bumps gen (setupState.touch, via settled/castTouched), which
		// is what the counter was added for.
		k.setupGen = d.setup.gen
	case modalStatus:
		k.second = d.now().Truncate(time.Second).Unix()
		if d.cfg.Stats != nil {
			k.stats = statsFingerprint(d.cfg.Stats())
		}
	case modalSevere:
		if d.breaking != nil {
			k.breakingID = d.breaking.ID
		}
		k.readingKey, k.severeReadPause = d.severeReading, d.severeReadPause
	case modalDetails:
		k.minute = d.now().Truncate(time.Minute).Unix() // the "N min ago" labels (projected here only — R3-B-09)
		if d.anyLoading() {
			k.shimmer = ((d.frame % 4) + 4) % 4
		}
	case modalDebug:
		// THE PICKER'S VALUE, not the question's index: the window has one
		// question and the value is what moves on the frame (HUM LEAD mock,
		// 2026-09-07). The confirmation is NOT here, and does not need to be —
		// it is a second layer composited outside this memo, and the window
		// underneath it is unchanged.
		k.debugFocus = d.debug.focus*1000 + d.debugPick()
	case modalRelayFault:
		// EVERYTHING THE FRAME SHOWS THAT MOVES. This window has two such things
		// and neither was here, so the memo replayed one frame while the model
		// underneath it worked perfectly — the arrows moved the cursor, the
		// clock ran down, the fall-through fired on time, and the DISPLAY never
		// changed once (UAT 2026-09-05, three rounds).
		k.faultFocus, k.faultLeft, k.faultHeld = d.relayFault.focus, d.relayFault.left, d.relayFault.held
	}
	return k
}

// statsFingerprint is the [S] window's inputs reduced to a comparable value.
//
// THE DURATIONS ARE ROUNDED TO THE SECOND FIRST. They are nanosecond counts, so
// fingerprinting the struct as it stands produces a different key on every
// frame and the memo can never hit — the window rebuilds three tables about
// three times a second for as long as it is open, which is the most expensive
// thing the app draws. The row shows whole seconds, so anything finer is not a
// change the reader can see.
func statsFingerprint(st Stats) [sha256.Size]byte {
	st.Uptime = st.Uptime.Truncate(time.Second)
	st.Requests.Uptime = st.Requests.Uptime.Truncate(time.Second)
	return sha256.Sum256([]byte(fmt.Sprintf("%+v", st)))
}

// modalView renders the open window through the memo: "" when none.
func (d Dashboard) modalView(o render.Opts) string {
	if d.modal == modalNone {
		return ""
	}
	m := d.mmemo
	if m == nil {
		return d.renderModal(o)
	}
	key := d.modalKeyFor(o)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ok && m.key == key {
		m.hits++
		return m.out
	}
	out := d.renderModal(o)
	m.ok, m.key, m.out = true, key, out
	m.misses++
	return out
}

// modalMemoCounts reports the modal slot's hit/miss counters.
func (d Dashboard) modalMemoCounts() (hits, misses int) { return d.mmemo.stats().counts() }

// stats returns the embedded counters, or nil for a nil slot — so counts() can
// answer (0, 0) without every caller repeating the nil check.
func (m *bodyMemo) stats() *memoStats {
	if m == nil {
		return nil
	}
	return &m.memoStats
}

func (m *modalMemo) stats() *memoStats {
	if m == nil {
		return nil
	}
	return &m.memoStats
}
