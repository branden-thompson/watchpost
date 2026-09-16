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
		// THE CONSOLE REFUSES WHAT IT CANNOT REACH, and it refuses it HERE
		// rather than by asking and discarding the answer. The chip is already
		// drawn unavailable; a key that acted anyway would contradict the
		// window's own sentence — HUM LEAD, UAT 2026-09-14: "<enter> opens
		// location details modal for Lone Pine, CA".
		if d.lookupIsScoped() {
			// A DEFINITE NO IS REFUSED; "not yet known" IS NOT (D-130). The
			// check can take 300ms plus a geocoder round trip, and a key that
			// went inert while the field was still thinking would be the same
			// dead control this rule exists to prevent — so enter is refused
			// only once there IS an answer and the answer is no.
			// A DEFINITE NO IS REFUSED; "COULD NOT ASK" IS NOT (D-151). A
			// timeout answers nothing, so refusing the key on it would leave
			// the operator with a real location they cannot request and no way
			// to retry — the dead control again, arrived at from the other side.
			if d.addLocate.settled() && d.addLocate.asked && !d.addLocate.reachable() {
				return d, nil
			}
			if d.addLocate.couldNotAsk() {
				d.addLocate.submitted = true
				return d, d.locateCmd(locateLookup, d.addLocate.gate.Seq(), d.addLocate.query)
			}
			if d.addLocate.reachable() {
				// AND THE ANSWER ALREADY HELD IS THE ANSWER. Re-asking would be
				// a second round trip to re-learn it, and a second authority
				// that can disagree with the first.
				ref := *d.addLocate.ref
				return d, func() tea.Msg { return resolvedMsg{mode: d.addMode, ref: ref} }
			}
			// NOT YET KNOWN: ASK THE SCOPED HOOK NOW (D-141).
			//
			// THIS FELL THROUGH TO `cfg.Resolve` — the UNSCOPED geocoder — so
			// the console's own scope was escapable by pressing enter inside the
			// 300 ms pause, which is the defect D-129 was filed for, still
			// reachable. The press is held on the field and honoured when the
			// verdict lands, so the key is neither inert nor a way out.
			d.addLocate.submitted = true
			return d, d.locateCmd(locateLookup, d.addLocate.gate.Seq(), d.addLocate.query)
		}
		if q := strings.TrimSpace(d.addQuery); q != "" {
			return d, d.resolveCmd(q, d.addMode)
		}
		return d, nil
	case "esc", "ctrl+a":
		d = d.close()
		d.addQuery, d.addErr, d.addLocate = "", "", locateState{}
	case "backspace":
		if r := []rune(d.addQuery); len(r) > 0 {
			d.addQuery = string(r[:len(r)-1])
		}
		return d.afterLookupEdit()
	default:
		if key.Text != "" {
			d.addQuery += key.Text
			return d.afterLookupEdit()
		}
	}
	return d, nil
}

// afterLookupEdit restarts the pause after a change to the search box.
//
// EVERY EDIT RESETS IT, which is the rule the HUM LEAD asked for: "Everytime
// the human user presses an alpha-numeric key in the text field, we can infer
// they're still typing, and we can reset that timer."
func (d Dashboard) afterLookupEdit() (tea.Model, tea.Cmd) {
	if !d.lookupIsScoped() {
		return d, nil
	}
	var cmd tea.Cmd
	d.addLocate, cmd = d.addLocate.edit(locateLookup, d.addQuery)
	return d, cmd
}

// lookupIsScoped: the search window is serving the CONSOLE (D-129).
//
// D-56 — ONE KEY, ONE MEANING PER SURFACE — IS WHAT MAKES THIS CORRECT RATHER
// THAN INCONSISTENT. `[l]` means "find me a place" on both surfaces; what
// differs is what a place IS. The listener may look anywhere, which is what
// Observer is for. The station can only broadcast about what it can reach, so
// on the console a location outside the service radius is not a location —
// HUM LEAD: "either what they type is a valid location within the service
// radius or not."
func (d Dashboard) lookupIsScoped() bool {
	return d.addMode == "lookup" && d.surface == SurfaceBroadcaster && d.cfg.LocateInRadius != nil
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
		d.setup.ref, d.setup.focus, d.setup.err = &v.ref, rowFIRMSKey, ""
		d = d.settled()
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
		ref := v.ref
		ref.Label, ref.Tag, ref.Zip = render.PlainLine(ref.Label), render.PlainLine(ref.Tag), render.PlainLine(ref.Zip) // the placeholder is drawn before the assembler cleans it (R5-C-05)
		d.lookupRef = &ref                                                                                              // ...which read as the looked-up location from the first frame, blank until its data lands
	}
	return d, d.commitCmd(watch, recent, v.mode)
}

// showLocation opens Details for a location the CONSOLE pointed at (D-113).
//
// HUM LEAD, UAT 2026-09-12: "make it so <enter> on a location pool row opens the
// location details (like observer)."
//
// THROUGH THE LOOKUP'S OWN SEAM, not a second one. A looked-up location is
// already a location shown in Details that may not be in either list yet — the
// exact shape a pool candidate is — so `lookupRef` carries it until the recent
// pipeline's data lands, and `selected` takes over once it has. Building a second
// way to show a location's Details would be a second answer to which location the
// modal is about, on a modal that reads five sections off that one fact.
//
// THE LABEL IS CLEANED HERE for the same reason the lookup cleans it: the
// placeholder is drawn before the assembler has been near it (R5-C-05).
func (d Dashboard) showLocation(ref snapshot.LocationRef) Dashboard {
	r := ref
	r.Label, r.Tag, r.Zip = render.PlainLine(r.Label), render.PlainLine(r.Tag), render.PlainLine(r.Zip)
	d.lookupRef = &r
	// IF IT IS ALREADY ON THE RECENT LIST, point at it there and drop the
	// placeholder: the row has real data and the modal should read it, which is
	// what `applyRecent` does when a lookup's own data arrives.
	if i := d.lookupIndex(); i >= 0 {
		d.selected, d.lookupRef = d.numPriority()+i, nil
	}
	return d.open(modalDetails)
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
	lines = append(lines, "  Search: "+d.addQuery+o.Glyphs().Cursor, "")
	if d.addErr != "" {
		lines = append(lines, "  "+o.Glyphs().Alert+" "+d.addErr, "")
	}
	// WHAT THE POOL SAYS ABOUT IT, IN THE REQUEST WINDOW'S OWN WORDS (D-129).
	// Shown as it is typed rather than on enter, because a refusal that waits
	// for the key it is going to refuse teaches the operator nothing.
	if fact, aside := d.addLocate.locateNote(); d.lookupIsScoped() && fact != "" {
		lines = append(lines, poolNoteLines(o, fact, aside, modalHelperWidth(d.modalWidth()))...)
		lines = append(lines, "")
	} else if d.lookupIsScoped() && d.addLocate.reachable() {
		// AND IT SAYS WHAT IT MATCHED. A prefix match means "Vis" is already a
		// hit, so without the name the operator cannot tell WHICH pooled place
		// enter is about to open — the request window names it for the same
		// reason.
		lines = append(lines, "    "+d.addLocate.ref.Label, "")
	}
	lines = append(lines, "  Type a city name or ZIP code.", "")
	verb := "Add"
	enabled := d.addMode != "add" || !d.watchlistFull()
	if d.addMode == "lookup" {
		verb = "Lookup"
	}
	// AND THE CHIP SAYS SO. CtlIf draws an unavailable control as unavailable,
	// which is the only honest state for a key the window will refuse.
	if d.lookupIsScoped() {
		// UNKNOWN READS AS AVAILABLE. The field is only briefly unsettled, and
		// greying the key while it thinks would flicker the control on every
		// keystroke.
		enabled = !d.addLocate.settled() || d.addLocate.reachable() || d.addLocate.couldNotAsk()
	}
	return append(lines, "  "+o.Controls("   ", render.CtlIf("enter", verb, enabled), render.Ctl("esc", "Cancel")))
}
