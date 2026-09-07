package tty

// nav.go — selection, sort and scroll: row navigation, modal scrolling, the RECENT viewport. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"sort"
	"strings"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// handleNav routes selection/paging actions (split from handleKey, P10-04).
// The focus index spans BOTH tables (UAT 4.4): 0..numPriority-1 walks the
// priority rows, then the recent rows, auto-scrolling the recent window.
func (d Dashboard) handleNav(act term.Action) Dashboard {
	switch d.modal {
	case modalHelp, modalDetails, modalAlerts, modalStatus, modalAbout: // the scrolling windows
		return d.handleModalNav(act)
	case modalSevere:
		return d.handleSevereNav(act) // 0.13.0: tabs and rows, or the record's scroll
	case modalRelayFault:
		return d.handleRelayFaultNav(act) // MVS-D-76: the three ways out of a dead relay
	case modalDebug:
		return d.handleDebugNav(act) // F-21
	}
	switch act {
	case "nav-up":
		if d.selected > 0 {
			d.selected--
			d.alertIdx = 0
		}
	case "nav-down":
		if d.selected < d.numPriority()+d.numRecent()-1 {
			d.selected++
			d.alertIdx = 0
		}
	case "alert-prev":
		if d.alertIdx > 0 {
			d.alertIdx--
		}
	case "alert-next":
		if loc := d.selectedLocation(); loc != nil && d.alertIdx < len(loc.Alerts)-1 {
			d.alertIdx++
		}
	}
	return d.syncRecentView()
}

// severityLevel orders NWS severities most-dangerous-first (UAT 16.2).
func severityLevel(sev string) int {
	switch strings.ToLower(sev) {
	case "extreme":
		return 0
	case "severe":
		return 1
	case "moderate":
		return 2
	case "minor":
		return 3
	}
	return 4
}

// sortAlerts orders every location's alerts most severe first — warnings
// outrank advisories within a tier — so index 0 (the module's default page,
// the name tint, the details view) is always the worst active alert. The
// snapshot is this consumer's own published copy; sorting in place is safe.
func sortAlerts(sn *snapshot.Snapshot) {
	if sn == nil {
		return
	}
	rank := func(a snapshot.Alert) int {
		r := severityLevel(a.Severity) * 2
		if !render.AlertIsWarning(a.Event, a.Severity) {
			r++ // advisory sorts after a warning of the same tier
		}
		return r
	}
	for i := range sn.Locations {
		sort.SliceStable(sn.Locations[i].Alerts, func(x, y int) bool {
			return rank(sn.Locations[i].Alerts[x]) < rank(sn.Locations[i].Alerts[y])
		})
	}
}

// handleModalNav owns navigation while a modal floats (split from
// handleNav, P10-04): up/down scroll the window (UAT 10.4); in the [A]
// modal, left/right page alerts without an esc round-trip (UAT 23.1).
func (d Dashboard) handleModalNav(act term.Action) Dashboard {
	switch act {
	case "nav-up":
		d.modalScroll = max(0, d.modalScroll-1)
	case "nav-down":
		d.modalScroll = min(d.modalScroll+1, max(0, len(d.modalLines())-d.modalMax()))
	case "alert-prev":
		if d.modal == modalAlerts && d.alertIdx > 0 {
			d.alertIdx--
			d.modalScroll = 0
		}
	case "alert-next":
		if loc := d.selectedLocation(); d.modal == modalAlerts && loc != nil && d.alertIdx < len(loc.Alerts)-1 {
			d.alertIdx++
			d.modalScroll = 0
		}
	}
	return d
}

func (d Dashboard) numPriority() int {
	if d.snap == nil {
		return 0
	}
	return len(d.snap.Locations)
}

// It counts the same list recentLocations draws WITHOUT BUILDING IT. The three
// callers are all on the frame path, and since 0.12.0's ticker the frame draws
// continuously — so copying fifty Location values to take a length ran forever,
// at 6.2 MB/min, a fifth of the app's whole allocation rate (perf pass,
// 2026-08-30). TestNumRecentAgreesWithTheDrawnList pins the two together.
func (d Dashboard) numRecent() int {
	n := 0
	if d.lookupRef != nil && d.lookupIndex() < 0 {
		n++
	}
	if d.recent != nil {
		n += len(d.recent.Locations)
	}
	return n
}

// recentLocations is the RECENT list AS THE TABLE DRAWS IT: the snapshot's
// locations, with the looked-up one PREPENDED while its data is still coming.
//
// A lookup used to put nothing in the table until the rebuilt snapshot arrived,
// so the row simply appeared some seconds later — which reads as the app having
// missed the keystroke. The placeholder carries no
// readings, so rowLoading marks it and the temperature cells shimmer, exactly as
// they do for a location still loading on first launch. The row is there from
// the first frame and fills in where it stands.
//
// ONE OWNER for the drawn list, because the focus arithmetic spans both tables:
// a table showing a row the navigation did not count would put the cursor off
// the end of it.
func (d Dashboard) recentLocations() []snapshot.Location {
	var out []snapshot.Location
	if d.lookupRef != nil && d.lookupIndex() < 0 {
		r := *d.lookupRef
		out = append(out, snapshot.Location{Label: r.Label, Tag: r.Tag, Zip: r.Zip, Lat: r.Lat, Lon: r.Lon, TZ: r.TZ})
	}
	if d.recent != nil {
		out = append(out, d.recent.Locations...)
	}
	return out
}

// selectedLocation resolves the focus index across both tables.
func (d Dashboard) selectedLocation() *snapshot.Location {
	np := d.numPriority()
	if d.selected < np {
		return &d.snap.Locations[d.selected]
	}
	// The drawn list, which already carries the pending lookup at its top — so
	// Details reads the same record the table is showing, rather than each
	// building its own idea of what the focus is on.
	rec := d.recentLocations()
	if i := d.selected - np; i >= 0 && i < len(rec) {
		return &rec[i]
	}
	return nil
}

// lookupIndex is the RECENT row of the looked-up location once the rebuilt
// list carries it (found by identity, wherever it landed — R5-C-02: an empty
// RECENT at lookup time put the focus past the tables); −1 while it waits.
func (d Dashboard) lookupIndex() int {
	if d.lookupRef == nil || d.recent == nil {
		return -1
	}
	for i, l := range d.recent.Locations {
		if sameLocation(refOf(l), *d.lookupRef) {
			return i
		}
	}
	return -1
}

// syncRecentView keeps the focused recent row inside the visible window.
func (d Dashboard) syncRecentView() Dashboard {
	np := d.numPriority()
	if d.selected < np {
		return d
	}
	idx := d.selected - np
	window := d.layout().window // once per key event (Q3, PR-5)
	if idx < d.recentOff {
		d.recentOff = idx
	}
	if idx >= d.recentOff+window {
		d.recentOff = idx - window + 1
	}
	return d
}
