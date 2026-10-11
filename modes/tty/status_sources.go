package tty

import (
	"fmt"
	"math/bits"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// StatusFeature is what a source serves, by which the Status window groups
// its hosts (0.19.0 D-170).
type StatusFeature uint8

// The features.
const (
	FeatureWeather StatusFeature = iota + 1 // forecasts, observations, marine, air quality, the geocoder
	FeatureAdverse                          // fire, quakes, tropical storms
	FeatureMap                              // what the map draws
)

// StatusSource is a source as the Status window groups and names it: what it
// serves and its badge, the chip the map and About show for it (D-170).
type StatusSource struct {
	Feature StatusFeature
	Badge   string
}

// statusGroup is one of the Status window's groups of hosts: a host two or
// more features use is common to them; one feature's alone is its group's.
type statusGroup struct {
	title   string
	feature StatusFeature // 0 for COMMON DATA and OTHER DATA, which the count decides
}

// statusGroups are the groups in the order drawn (D-170, A-43): what fails
// in COMMON DATA fails more than one feature, so it is first. OTHER DATA
// holds a host the window counts that no source claims.
var statusGroups = [...]statusGroup{
	{title: "COMMON DATA"},
	{title: "WEATHER DATA", feature: FeatureWeather},
	{title: "ADVERSE EVENTS", feature: FeatureAdverse},
	{title: "MAP DRAWING DATA", feature: FeatureMap},
	{title: "OTHER DATA"},
}

// The groups' places in statusGroups.
const (
	groupCommon = 0
	groupOther  = len(statusGroups) - 1
)

// groupOf is the place of a host's group, by the features that use it.
func groupOf(features uint8) int {
	switch n := bits.OnesCount8(features); {
	case n == 0:
		return groupOther
	case n > 1:
		return groupCommon
	}
	for i, g := range statusGroups {
		if g.feature != 0 && features == 1<<g.feature {
			return i
		}
	}
	return groupOther
}

// use records a source on a host's row: the feature it serves and its badge,
// once each.
func (r *endpointRow) use(s StatusSource, fallback string) {
	if s.Feature != 0 {
		r.features |= 1 << s.Feature
	}
	badge := s.Badge
	if badge == "" {
		badge = fallback
	}
	for _, b := range r.badges {
		if b == badge {
			return
		}
	}
	r.badges = append(r.badges, badge)
}

// sourceRows are the Status window's hosts, one row each (D-170): the
// station's providers on the hosts they use, with their health; the map's
// sources on theirs; the request counters the station's client and the map's
// clients keep, added; and a host they count that no source claims. A
// provider's health is its host's state; a host the map alone vouches for is
// OK while its last answer is its latest word, FAIL while a failure is, IDLE
// before it is asked.
func sourceRows(provs []snapshot.ProviderStatus, st Stats, maps []MapSource, now time.Time) []endpointRow {
	byHost := map[string]*endpointRow{}
	var order []string
	row := func(host string) *endpointRow {
		r, ok := byHost[host]
		if !ok {
			r = &endpointRow{endpoint: host}
			byHost[host], order = r, append(order, host)
		}
		return r
	}
	for _, p := range provs {
		hosts := st.Endpoints[p.ID]
		if len(hosts) == 0 {
			hosts = []string{p.ID} // unmapped: name it rather than drop it
		}
		for _, h := range hosts {
			r := row(h)
			r.use(st.ProviderSources[p.ID], strings.ToUpper(p.ID))
			r.reported = true
			// The WORST status on a shared host wins, and the OLDEST fetch: a
			// row that says ok because one of its two feeds is fine would hide
			// the one that is not.
			if r.status == "" || worseStatus(p.Status, r.status) {
				r.status, r.state = p.Status, roleState(p)
			}
			if r.fetchedAt.IsZero() || (!p.FetchedAt.IsZero() && p.FetchedAt.Before(r.fetchedAt)) {
				r.fetchedAt = p.FetchedAt
			}
		}
	}
	for _, s := range maps {
		r := row(s.Host)
		r.use(StatusSource{Feature: FeatureMap, Badge: s.Badge}, s.Layers) // a source with no badge yet is named by its layers
		r.mapped = true
	}
	for _, rs := range [2]httpx.RequestStats{st.Requests, st.MapRequests} {
		for _, h := range rs.Hosts {
			if _, ok := byHost[h.Host]; !ok {
				row(h.Host).use(st.HostSources[h.Host], "—") // A HOST NO PROVIDER CLAIMS still gets a row: dropping it would under-report the traffic this window accounts for
			}
		}
	}
	out := make([]endpointRow, 0, len(order))
	for _, host := range order {
		r := byHost[host]
		h, counted := hostTotal(st, host)
		r.count(h, counted, now)
		r.providers = badgeFaces(r.badges)
		out = append(out, *r)
	}
	return out
}

// hostTotal is a host's counters across the station's client and the map's
// clients: each counts its own asks, so they add.
func hostTotal(st Stats, host string) (httpx.HostStats, bool) {
	a, okA := hostIn(st.Requests, host)
	b, okB := hostIn(st.MapRequests, host)
	sum := httpx.HostStats{Host: host, Attempts: a.Attempts + b.Attempts, Net: a.Net + b.Net, Cache: a.Cache + b.Cache,
		Neg: a.Neg + b.Neg, BytesNet: a.BytesNet + b.BytesNet, LastOK: later(a.LastOK, b.LastOK), LastFail: later(a.LastFail, b.LastFail)}
	return sum, okA || okB
}

// later is the later of two moments.
func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// count puts a host's counters on its row and its state: a provider's where
// one reports on it; else, for a host the map uses, its counters' - OK while
// its last answer is its latest word, FAIL while a failure is, IDLE before
// it is asked; else none, its numbers real but nothing vouching for them.
func (r *endpointRow) count(h httpx.HostStats, counted bool, now time.Time) {
	if counted {
		r.seen = true
		r.tries, r.net, r.cache, r.neg, r.bytes = h.Attempts, h.Net, h.Cache, h.Neg, h.BytesNet
	}
	switch {
	case r.reported:
		r.claimed = true
	case r.mapped && counted && h.Attempts+h.Cache+h.Neg > 0:
		r.claimed, r.fetchedAt = true, h.LastOK
		r.state, r.status = "OK", snapshot.ProviderOK
		if h.LastFail.After(h.LastOK) {
			r.state, r.status = "FAIL", snapshot.ProviderDegraded
		}
	case r.mapped:
		r.claimed, r.state, r.status = false, "IDLE", snapshot.ProviderOff
	default:
		r.state = "not reported"
	}
	if !r.fetchedAt.IsZero() {
		r.age = now.Sub(r.fetchedAt)
	}
}

// badgeFaces are the badges as the table lays them out: each a chip's face,
// padded as chipFace pads it, so a column measured on it fits the chips
// drawn over it (chippedSources).
func badgeFaces(badges []string) string {
	faces := make([]string, len(badges))
	for i, b := range badges {
		faces[i] = "  " + b + "  "
	}
	return strings.Join(faces, " ")
}

// chippedSources draws each badge's face in a table's lines as its chip:
// the same width, so the columns stay where the table put them; a face the
// table cut short stays plain.
func chippedSources(lines []string, rows []endpointRow) []string {
	seen := map[string]bool{}
	var faces []string
	for _, r := range rows {
		for _, b := range r.badges {
			if !seen[b] {
				seen[b] = true
				faces = append(faces, b)
			}
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		for _, b := range faces {
			l = strings.ReplaceAll(l, "  "+b+"  ", chipFace(b))
		}
		out[i] = l
	}
	return out
}

// sourceLines are the Status window's hosts, by group (D-170): a headline
// counting them, the columns' header once, each group's title over its rows,
// then the notes about the map's data and what it sends (D-132, D-151).
func (d Dashboard) sourceLines(o render.Opts, rows []endpointRow, shape tableShape, fillTo int) []string {
	var groups [len(statusGroups)][]endpointRow
	for _, r := range rows {
		g := groupOf(r.features)
		groups[g] = append(groups[g], r)
	}
	used := 0
	for _, g := range groups {
		if len(g) > 0 {
			used++
		}
	}
	out := []string{statusHeader(fmt.Sprintf("DATA SOURCES  (%d host%s in %d group%s)", len(rows), plural(len(rows)), used, plural(used)))}
	if len(rows) == 0 {
		return append(out, "   awaiting first snapshot...")
	}
	headed := false
	for i, g := range groups {
		if len(g) == 0 {
			continue
		}
		table := chippedSources(providerTable(o, g, "SOURCES", shape, fillTo), g)
		if headed {
			out = append(out, "") // a blank row between the groups (D-171)
		} else {
			out, headed = append(out, table[0]), true // the columns, once over every group
		}
		out = append(out, statusHeader(d.groupTitle(i)))
		out = append(out, table[1:]...)
	}
	return append(out, d.mapWords(widest(out))...)
}

// groupTitle is a group's title; MAP DRAWING DATA's carries the tiles the
// map holds on disk.
func (d Dashboard) groupTitle(i int) string {
	title := statusGroups[i].title
	if statusGroups[i].feature != FeatureMap {
		return title
	}
	if m := d.mapPane.m; m != nil {
		if disk := m.CacheUse().Disk; disk.Limit > 0 {
			title += "  (tiles held " + render.HumanBytes(disk.Held) + " of " + render.HumanBytes(disk.Limit) + ")"
		}
	}
	return title
}

// mapWords are the notes about the map's data (D-132), its notes as they
// stand (D-159) and what it sends (D-151, D-261), wrapped to the tables -
// never widening the window - and muted.
func (d Dashboard) mapWords(tableW int) []string {
	if len(d.cfg.MapSources) == 0 {
		return nil
	}
	wrapAt := max(tableW-len(statusInset), 40)
	muted := render.Tok(render.TableMuted)
	var words []string
	for _, s := range d.cfg.MapSources {
		words = append(words, s.Notes...)
	}
	words = append(words, d.mapNotesNow()...)
	words = append(words, mapDisclosure(d.cfg.MapSources))
	var out []string
	said := map[string]bool{}
	for _, w := range words {
		if said[w] {
			continue // a note several hosts carry is said once (D-172)
		}
		said[w] = true
		for _, l := range render.WrapText(w, wrapAt) {
			out = append(out, statusInset+render.Tint(l, muted))
		}
	}
	return out
}
