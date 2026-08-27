package tty

// modal_location.go — the add / remove modals: search, type-ahead, remove-confirm, and the watchlist ref helpers. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// handleAddKey owns keys while the add-location modal is open: printable
// keys build the query, so table/global bindings never fire mid-typing.
func (d Dashboard) handleAddKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		if d.addMode == "add" && d.watchlistFull() {
			return d, nil // chip is muted; the press is inert (UAT 26.3)
		}
		if q := strings.TrimSpace(d.addQuery); q != "" {
			return d, d.resolveCmd(q, d.addMode)
		}
		return d, nil
	case "esc", "ctrl+a":
		d = d.close()
		d.addQuery, d.addErr = "", ""
	case "backspace":
		if r := []rune(d.addQuery); len(r) > 0 {
			d.addQuery = string(r[:len(r)-1])
		}
	default:
		if key.Text != "" {
			d.addQuery += key.Text
		}
	}
	return d, nil
}

// The two list caps (UAT 48: 10 favourites + 50 most-recent = 60 tracked
// locations) — exported so the app, which builds the lists, reads the tty's
// numbers instead of its own copies (Q6, L3-F11).
const (
	WatchCap  = 10
	RecentCap = 50
)

// watchlistFull reports the priority cap (UAT 26.3).
func (d Dashboard) watchlistFull() bool { return d.numPriority() >= WatchCap }

// resolveCmd asks the app hook to turn the typed query into a ref.
func (d Dashboard) resolveCmd(query, mode string) tea.Cmd {
	res := d.cfg.Resolve
	return func() tea.Msg {
		if res == nil {
			return resolvedMsg{mode: mode, err: fmt.Errorf("search is not wired in this build")}
		}
		ref, err := res(query)
		return resolvedMsg{mode: mode, ref: ref, err: err}
	}
}

// handleResolved routes a resolved location into its flow (UAT 26.3/26.4).
func (d Dashboard) handleResolved(v resolvedMsg) (tea.Model, tea.Cmd) {
	if v.err != nil {
		d.addErr = v.err.Error()
		return d, nil
	}
	if v.mode == "setup" { // the Setup window's location question, answered by a full resolve (no hint matched)
		d.setup.ref, d.setup.focus, d.setup.err = &v.ref, focusKey, ""
		return d, nil
	}
	watch, recent := refsOf(d.snap), refsOf(d.recent)
	switch v.mode {
	case "add":
		if len(watch) >= WatchCap {
			d.addErr = fmt.Sprintf("the priority list is full (%d locations)", WatchCap)
			return d, nil
		}
		for _, r := range watch { // F4: a duplicate would leave the assembler with nothing to publish
			if sameLocation(r, v.ref) { // the lists' identity: zip first (UAT 106)
				d.addErr = v.ref.Label + " is already on the watchlist"
				return d, nil
			}
		}
		watch = append(watch, v.ref)       // bottom of the watchlist (UAT 26.3)
		recent = withoutRef(recent, v.ref) // UAT 106: a promoted location leaves RECENT — never on screen twice
	}
	d = d.close() // the search modal is done
	d.addQuery, d.addErr = "", ""
	if v.mode == "lookup" {
		recent = prependRef(recent, v.ref) // top of recent/searched (UAT 26.4)
		d.selected = len(watch)            // focus the looked-up location...
		d = d.open(modalDetails)           // ...and open its details
	}
	return d, d.commitCmd(watch, recent, v.mode)
}

// commitCmd hands the new ref sets to the app hook (persist + rebuild).
func (d Dashboard) commitCmd(watch, recent []snapshot.LocationRef, what string) tea.Cmd {
	commit := d.cfg.Commit
	return func() tea.Msg {
		if commit == nil {
			return committedMsg{err: fmt.Errorf("watchlist changes are not wired in this build"), what: what}
		}
		return committedMsg{err: commit(watch, recent), what: what}
	}
}

// handleRemoveKey owns the confirmation modal (UAT 26.2).
func (d Dashboard) handleRemoveKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		d = d.close()
		watch, recent := refsOf(d.snap), refsOf(d.recent)
		if d.selected >= len(watch) {
			return d, nil
		}
		removed := watch[d.selected]
		watch = append(watch[:d.selected:d.selected], watch[d.selected+1:]...)
		recent = prependRef(recent, removed) // top of recent (UAT 26.2)
		if d.selected >= len(watch) && d.selected > 0 {
			d.selected--
		}
		return d, d.commitCmd(watch, recent, "remove")
	case "esc":
		d = d.close()
	}
	return d, nil
}

// refsOf rebuilds the pipeline ref set from a snapshot's locations.
func refsOf(sn *snapshot.Snapshot) []snapshot.LocationRef {
	if sn == nil {
		return nil
	}
	refs := make([]snapshot.LocationRef, 0, len(sn.Locations))
	for _, l := range sn.Locations {
		refs = append(refs, refOf(l))
	}
	return refs
}

// refOf is a location's request identity (the one builder — three callers).
func refOf(l snapshot.Location) snapshot.LocationRef {
	return snapshot.LocationRef{Label: l.Label, Tag: l.Tag, Zip: l.Zip, Lat: l.Lat, Lon: l.Lon, TZ: l.TZ}
}

// withoutRef drops ref (by location key) from refs, order kept — the
// RECENT / SEARCHED list when a location is promoted to the watchlist
// (UAT 106): later entries move up one, the list shrinks by one.
func withoutRef(refs []snapshot.LocationRef, ref snapshot.LocationRef) []snapshot.LocationRef {
	out := make([]snapshot.LocationRef, 0, len(refs))
	for _, r := range refs {
		if !sameLocation(r, ref) {
			out = append(out, r)
		}
	}
	return out
}

// sameLocation: by zip when either side has one (the identity the lists
// dedupe on), else by location key.
func sameLocation(a, b snapshot.LocationRef) bool {
	if a.Zip != "" || b.Zip != "" {
		return a.Zip == b.Zip
	}
	return snapshot.Key(a) == snapshot.Key(b)
}

// prependRef puts ref at the head, deduped by zip, capped at RecentCap.
func prependRef(refs []snapshot.LocationRef, ref snapshot.LocationRef) []snapshot.LocationRef {
	out := []snapshot.LocationRef{ref}
	for _, r := range refs {
		if r.Zip == ref.Zip || len(out) == RecentCap {
			continue
		}
		out = append(out, r)
	}
	return out
}

// removeLines is the confirmation modal body (UAT 26.2).
func (d Dashboard) removeLines(o render.Opts) []string {
	label := "this location"
	if sel := d.selectedLocation(); sel != nil {
		label = sel.Label
	}
	return []string{
		"",
		"  Remove " + label + " from the watchlist?",
		"",
		"  It will move to the top of the RECENT / SEARCHED list.",
		"",
		"  " + o.Controls("   ", render.Ctl("enter", "Confirm"), render.Ctl("esc", "Cancel")),
	}
}

// addLines is the add-location modal body (UAT 16.3). The live type-ahead
// results need a search hook wired from app (modes cannot import domains -
// import lint); the hook lands with the M-V3 flow.
func (d Dashboard) addLines(o render.Opts) []string {
	lines := []string{""}
	if d.addMode == "add" && d.watchlistFull() {
		// UAT 26.3: cap note leads the modal when the watchlist is full.
		lines = append(lines, "  Only 10 locations are allowed in the priority list", "  for performance reasons.", "")
	}
	lines = append(lines, "  Search: "+d.addQuery+"▌", "")
	if d.addErr != "" {
		lines = append(lines, "  ⚠ "+d.addErr, "")
	}
	lines = append(lines, "  Type a city name or ZIP code.", "")
	verb := "Add"
	enabled := d.addMode != "add" || !d.watchlistFull()
	if d.addMode == "lookup" {
		verb = "Lookup"
	}
	return append(lines, "  "+o.Controls("   ", render.CtlIf("enter", verb, enabled), render.Ctl("esc", "Cancel")))
}
