// Package severe is the unified severe-event index (0.13.0, SAM-D-26 AX-1 = A):
// the global feeds' events (domains/globalfeed) and the tracked locations'
// alerts (platform/snapshot) folded into one de-duplicated, classified,
// sorted, capped list of rows, plus the [A]-shaped record of any row. Pure —
// no goroutines, no UI types — so the TTY window, report mode and any future
// surface consume the same index.
package severe

import (
	"github.com/branden-thompson/watchpost/platform/category"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Tab is the window's category — names for the one registry
// (platform/category), never an ordering of its own. Anything that needs to
// know about a category asks the registry, so there is nothing for a second
// list to disagree with.
type Tab = category.Category

const (
	TabEmergency  = category.Emergency
	TabWarnings   = category.Warnings
	TabWatches    = category.Watches
	TabAdvisories = category.Advisories
	TabStatements = category.Statements
	TabDisasters  = category.Disasters
	TabMarine     = category.Marine
	TabForecasts  = category.Forecasts
	NumTabs       = category.Count
	TabNone       = category.None // Classify's "not shown" — never a valid index
)

// MaxRows caps the retained index (NFR-4 / SAM-D-22, P10-03): the most recent win.
const MaxRows = 500

// Row is one event in the index.
type Row struct {
	Key      string // normalised id — the de-dup key
	Tab      Tab
	Source   string                // "USGS" | "NHC" | "NWS" | "NWS-local"
	Product  string                // "Tornado Warning" · "Earthquake" · "Tropical Storm"
	Name     string                // storm name; "" otherwise
	Location string                // the tied label (globalfeed.Locate) or the tracked location's label
	Tied     *snapshot.LocationRef // the tracked location the row belongs to; nil for a global-only row (the zone rule, FR-9)
	Severity globalfeed.Severity
	At       time.Time // declared / recorded / reported
	Until    time.Time // expires; zero when none
	HasPoint bool
	Lat, Lon float64
	Sender   string
	Sent     time.Time
	Detail   Detail

	// Test marks a row the ctrl+d window fabricated (FR-4.4), carried from
	// globalfeed.Event.Fabricated so the window and its spoken report can say
	// so. Nothing in a release build sets it.
	Test bool
}

// Detail is the per-class record (SAM-D-21). Alert is the tracked-location
// CAP record and wins the rendering when present; Severe is the national
// feed's CAP record, kept beside it for the parameters only the national
// query carries (gusts, hail, VTEC).
type Detail struct {
	Quake    *globalfeed.QuakeDetail
	Tropical *globalfeed.TropicalDetail
	Severe   *globalfeed.SevereDetail
	Alert    *snapshot.Alert
}

// oidRE is the CAP alert identifier grammar api.weather.gov emits: a dotted
// OID with a hex fingerprint segment. Anything else keeps its raw id and never
// merges (red-team S4: a crafted "…/urn:oid:<real>" must not collide).
var oidRE = regexp.MustCompile(`^urn:oid:[0-9]+(\.[0-9a-f]+)+$`)

// nwsAlertPrefix is the one legitimate URL form of an NWS alert id.
const nwsAlertPrefix = "https://api.weather.gov/alerts/"

// NormalizeID joins the two forms an NWS alert id takes — the feature URL
// (https://api.weather.gov/alerts/urn:oid:…) on the ticker path and the bare
// urn:oid:… on the location path. nws reports whether the id validated as one.
func NormalizeID(id string) (key string, nws bool) {
	id = strings.TrimSpace(id)
	i := strings.Index(id, "urn:oid:")
	if i < 0 {
		return id, false
	}
	cand := id[i:]
	if !oidRE.MatchString(cand) {
		return id, false
	}
	if i > 0 && id[:i] != nwsAlertPrefix {
		return id, false // an OID under a foreign prefix is not the same alert
	}
	return cand, true
}

// Classify maps an event class + product name to its tab; ok is false (and
// the tab TabNone) for a product the window does not show (Air Quality Alert,
// Hydrologic Outlook…). English substring matching on NWS product
// names is an accepted limitation (objectives §5); "Statements" is Special
// Weather Statements by name (SAM-D-10).
func Classify(class globalfeed.Class, product string) (Tab, bool) {
	switch class {
	case globalfeed.ClassQuake:
		return TabDisasters, true
	case globalfeed.ClassTropical:
		return TabMarine, true
	}
	// The civil-emergency family is matched BY NAME, before the keyword arms.
	// None of these products names a warning, watch or advisory, so without
	// this they fall through to "not shown" — which is where the Weather
	// Service's highest-urgency products were sitting.
	//
	// THE TABLE MOVED TO globalfeed AND IS NO LONGER OURS (C-2, D-1). The
	// producer asks the same question when it lanes an arrival for the marquee
	// and the read, and this package is the one that imports globalfeed rather
	// than the reverse — so a copy here was a copy the marquee could not see,
	// and it laned an evacuation order as an ordinary warning.
	if tab, ok := globalfeed.CivilEmergencyCategory(product); ok {
		return tab, true
	}
	// Warning, Watch and Advisory are decided first: a product naming one of
	// them is that thing, whatever else its name contains.
	switch {
	case strings.Contains(product, "Warning"):
		return TabWarnings, true
	case strings.Contains(product, "Watch"):
		return TabWatches, true
	case strings.Contains(product, "Advisory"), product == "Air Quality Alert":
		// The Air Quality Alert is an advisory by name in everything but the
		// word (MVS-D-58). Named exactly rather than matching "Alert", which
		// would sweep in products from other programmes on a coincidence.
		return TabAdvisories, true
	case strings.Contains(product, "Marine"):
		// MARINE is the tab's name to the listener. Swept against the live
		// catalogue this arm matches exactly ONE product — Marine Weather
		// Statement — because every other marine product (Gale Warning,
		// Hazardous Seas Watch, Small Craft Advisory, Hurricane Force Wind
		// Warning) names warning, watch or advisory and is decided above.
		// So narrowing this to that one string changes nothing today, and no
		// test can tell the two apart without inventing a product the Weather
		// Service does not issue. Whether marine products belong here at all
		// rather than in their severity tab is an open ruling.
		return TabMarine, true
	case strings.Contains(product, "Outlook"), strings.Contains(product, "Forecast"):
		// FORECASTS AND OUTLOOKS (MVS-D-59): what MIGHT develop, days out, over
		// a whole forecast area — issued below a watch and written as
		// discussion. Decided after warning, watch and advisory, so a product
		// that names one of those is that thing however it is worded.
		return TabForecasts, true
	case strings.Contains(product, "Statement"):
		// EVERY remaining statement, not only the Special Weather Statement
		// (MVS-D-57). A Coastal Flood Statement used to fall through to "not
		// shown", so the office had told the listener something and the app
		// had quietly decided not to pass it on.
		return TabStatements, true
	}
	return TabNone, false
}

// Guard applies the guarded superseded rule to the tracked locations' alerts
// (NFR-12): the ids a same-sender, same-product, newer message replaces.
func Guard(locs []snapshot.Location) map[string]bool {
	var refs []globalfeed.Ref
	seen := map[string]bool{}
	for i := range locs {
		for _, a := range locs[i].Alerts {
			key, _ := NormalizeID(a.ID)
			if seen[key] {
				continue
			}
			seen[key] = true
			r := globalfeed.Ref{ID: key, Sender: a.SenderName, Product: a.Event, Sent: a.Sent}
			for _, x := range a.References {
				if k, _ := NormalizeID(x); k != key {
					r.Replaces = append(r.Replaces, k)
				}
			}
			refs = append(refs, r)
		}
	}
	return globalfeed.SupersededBy(refs)
}

// index accumulates rows by key, first-seen order preserved.
type index struct {
	rows  map[string]Row
	order []string
}

func (x *index) add(r Row) {
	if _, ok := x.rows[r.Key]; !ok {
		x.order = append(x.order, r.Key)
	}
	x.rows[r.Key] = r
}

// Union folds the feed events and the locations' alerts into one row per
// event: keyed on the normalised id; a tracked location's record wins over the
// feed's (it carries the prose), with the feed's point, tied label and CAP
// parameters merged in; superseded messages on either path are dropped; a
// multi-location alert is one row tied to its first location in watchlist
// order; now drops expired.
func Union(feed []globalfeed.Event, locs []snapshot.Location, now time.Time) []Row {
	x := &index{rows: make(map[string]Row, len(feed)), order: make([]string, 0, len(feed))}
	superseded := Guard(locs)
	x.addFeed(feed, superseded, now)
	x.addLocations(locs, superseded, now)
	out := make([]Row, 0, len(x.order))
	for _, k := range x.order {
		out = append(out, x.rows[k])
	}
	if err := invariant.Check(len(out) == len(x.rows), "union emits exactly one row per key"); err != nil {
		return nil
	}
	return out
}

// addFeed adds the global feed's events; a superseded event contributes its
// key to the superseded set instead (an alert the national feed saw replaced
// must not resurface via a tracked location whose zone query lagged).
func (x *index) addFeed(feed []globalfeed.Event, superseded map[string]bool, now time.Time) {
	for _, e := range feed {
		key, _ := NormalizeID(e.ID)
		if e.Superseded || superseded[key] { // the feed's own flag, or the Guard set from a tracked location's newer update (R3-A-04)
			superseded[key] = true
			continue
		}
		if e.ID == "" || (!e.Until.IsZero() && now.After(e.Until)) {
			continue
		}
		tab, ok := Classify(e.Class, e.Type)
		if !ok {
			continue
		}
		r := Row{Key: key, Tab: tab, Source: e.Source, Product: e.Type, Name: e.Name, Location: e.Location,
			Severity: e.Severity, At: e.At, Until: e.Until, HasPoint: e.HasPoint, Lat: e.Lat, Lon: e.Lon,
			Test:   e.Fabricated,
			Detail: Detail{Quake: e.Quake, Tropical: e.Tropical, Severe: e.Severe}}
		if e.Severe != nil {
			r.Sender, r.Sent = e.Severe.SenderName, e.Severe.Sent
		}
		x.add(r)
	}
}

// addLocations adds the tracked locations' alerts; a location's record wins
// over the feed's for the same key (it carries the prose), merging the feed's
// point and CAP parameters; the first (highest) location keeps a multi-location alert.
func (x *index) addLocations(locs []snapshot.Location, superseded map[string]bool, now time.Time) {
	for i := range locs {
		loc := locs[i]
		ref := snapshot.LocationRef{Label: loc.Label, Tag: loc.Tag, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, TZ: loc.TZ}
		for j := range loc.Alerts {
			r, ok := locationRow(loc.Alerts[j], &ref, superseded, now)
			if !ok {
				continue
			}
			if prev, seen := x.rows[r.Key]; seen {
				if prev.Tied != nil {
					continue // already tied to an earlier (higher) watchlist location — one row per alert
				}
				r.HasPoint, r.Lat, r.Lon = prev.HasPoint, prev.Lat, prev.Lon // the feed's point
				r.Detail.Severe = prev.Detail.Severe                         // the national record's CAP parameters, kept beside the location record
				r.Severity = max(r.Severity, prev.Severity)                  // both paths agree on a curated product (severityOf); the higher tier wins otherwise
			}
			x.add(r)
		}
	}
}

// locationRow builds one location-path row; ok is false when the alert is
// superseded, expired or a product the window does not show.
func locationRow(a snapshot.Alert, ref *snapshot.LocationRef, superseded map[string]bool, now time.Time) (Row, bool) {
	key, _ := NormalizeID(a.ID)
	if superseded[key] {
		return Row{}, false
	}
	until := a.Expires
	if a.Ends != nil {
		until = *a.Ends
	}
	if !until.IsZero() && now.After(until) {
		return Row{}, false
	}
	tab, ok := Classify(globalfeed.ClassSevereWx, a.Event)
	if !ok {
		return Row{}, false
	}
	at := a.Sent // declared = issued (UAT 2026-08-28); the onset is the record's "Starts"
	if at.IsZero() {
		at = a.Effective
	}
	if at.IsZero() && a.Onset != nil {
		at = *a.Onset
	}
	alert := a
	return Row{Key: key, Tab: tab, Source: "NWS-local", Product: a.Event, Location: ref.Label, Tied: ref,
		Severity: severityOf(a.Event, a.Severity), At: at, Until: until, Sender: a.SenderName, Sent: a.Sent,
		Detail: Detail{Alert: &alert}}, true
}

// severityOf is the row's tier: the curated national list first, so one
// product reads the same tier by every path (a Tornado Watch is yellow tied
// or untied — R3-A-05); then the CAP severity + product for the rest.
func severityOf(product, capSeverity string) globalfeed.Severity {
	if sev, known := globalfeed.CuratedSeverity(product); known {
		return sev
	}
	switch strings.ToLower(capSeverity) {
	case "extreme":
		return globalfeed.SevRed
	case "severe":
		return globalfeed.SevOrange
	}
	if strings.Contains(product, "Warning") {
		return globalfeed.SevOrange
	}
	return globalfeed.SevYellow
}

// Sorted is Sort returning its argument (composable in tests).
func Sorted(rows []Row) []Row { Sort(rows); return rows }

// Sort orders rows Declared DESC (SAM-D-8), then most severe, then key — stable.
func Sort(rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].At.Equal(rows[j].At) {
			return rows[i].At.After(rows[j].At)
		}
		if rows[i].Severity != rows[j].Severity {
			return rows[i].Severity > rows[j].Severity
		}
		return rows[i].Key < rows[j].Key
	})
}

// Cap keeps the first n rows of a sorted slice (most recent wins) and reports
// the total before capping, for "showing N of M".
func Cap(rows []Row, n int) (kept []Row, total int) {
	total = len(rows)
	if err := invariant.Check(n > 0, "the retained index needs a positive cap (P10-03)"); err != nil {
		n = MaxRows
	}
	if len(rows) > n {
		rows = rows[:n]
	}
	return rows, total
}

// ByTab splits rows into their tabs, order preserved.
func ByTab(rows []Row) [NumTabs][]Row {
	var out [NumTabs][]Row
	for _, r := range rows {
		if r.Tab >= 0 && r.Tab < NumTabs {
			out[r.Tab] = append(out[r.Tab], r)
		}
	}
	return out
}
