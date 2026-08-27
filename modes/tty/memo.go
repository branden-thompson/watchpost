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
	"sync"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// bodyKey is every input the two tables read. Adding an input to row(),
// recentSection() or the layout means adding a field here; the
// invalidation table in memo_test.go has one row per field.
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
	radioKey       snapshot.LocationKey // the ▶ row
	radioPlaying   bool                 // ▶ clears on stop while radioKey stays
	radioRepeat    RepeatMode           // the ∞ mark on every row
	theme          uint64               // render.ThemeGeneration: every Tok() tint in the cells
	fireBoldMW     float64              // the bold-◆ rule (Setup)
	shimmer        int                  // the loading frame while a row loads, else 0
}

// bodyMemo is the single slot. hits/misses are read by the tests and the
// diagnostic dump; they are not policy.
type bodyMemo struct {
	mu               sync.Mutex
	ok               bool
	key              bodyKey
	priority, recent string
	hits, misses     int
}

// bodyKeyFor derives the key from the model and this frame's layout.
func (d Dashboard) bodyKeyFor(fl frameLayout) bodyKey {
	k := bodyKey{
		width: fl.o.Width, height: d.height, compact: fl.compact, radioH: fl.radioH, alertH: fl.alertH,
		controlRows: fl.controlRows, window: fl.window, days: fl.days,
		snap: d.snap, recent: d.recent, selected: d.selected, recentOff: d.recentOff,
		units: d.units, ascii: fl.o.ASCII,
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

// memoCounts reports the slot's hit/miss counters (0, 0 without a slot).
func (d Dashboard) memoCounts() (hits, misses int) {
	if d.memo == nil {
		return 0, 0
	}
	d.memo.mu.Lock()
	defer d.memo.mu.Unlock()
	return d.memo.hits, d.memo.misses
}

// anyLoading reports whether any row still shows the loading shimmer —
// the tick's reason to keep running and the memo's reason to key on the
// frame (row(): a location loads until its observation and daily land).
func (d Dashboard) anyLoading() bool {
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
func rowLoading(loc *snapshot.Location) bool {
	return loc.Harmonized.Source.Provider == "" || len(loc.Daily) == 0
}
