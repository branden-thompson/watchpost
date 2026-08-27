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

func (d Dashboard) numRecent() int {
	if d.recent == nil {
		return 0
	}
	return len(d.recent.Locations)
}

// selectedLocation resolves the focus index across both tables.
func (d Dashboard) selectedLocation() *snapshot.Location {
	np := d.numPriority()
	if d.selected < np {
		return &d.snap.Locations[d.selected]
	}
	if i := d.selected - np; i < d.numRecent() {
		return &d.recent.Locations[i]
	}
	return nil
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
