package globalfeed

import (
	"github.com/branden-thompson/watchpost/platform/category"
	"sort"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// miPerKm converts the radius (miles, as the user sets it) to the kilometres
// the haversine returns.
const kmPerMi = 1.609344

// WithinMiles reports whether (lat2, lon2) is inside radiusMi of (lat1, lon1).
//
// The one distance measure every scoped surface shares, so the severe window
// can scope its TRACKED LOCATIONS by the same preference that scopes its feed
// events — the statements and advisories tabs have no feed half at all, so
// without it they were bounded by nothing but the watchlist.
func WithinMiles(lat1, lon1, lat2, lon2, radiusMi float64) bool {
	return geo.HaversineKM(lat1, lon1, lat2, lon2) <= radiusMi*kmPerMi
}

// Active drops events whose active window has closed (now past Until), so a
// severe alert leaves the marquee when it is no longer active (HUM LEAD
// 2026-08-27, #2). Events with no Until — a quake's instant, a storm the feed
// still lists — are kept; they age out when the feed stops returning them. The
// input is not mutated.
func Active(evs []Event, now time.Time) []Event {
	out := make([]Event, 0, len(evs))
	for _, e := range evs {
		if !e.Until.IsZero() && now.After(e.Until) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// MaxPerLane caps EACH LANE's share of the ticker stack (P10-03: a big outbreak
// cannot grow the marquee or the memo without bound).
//
// PER LANE, not per stack. The marquee rotates through the lanes and a lane with
// no events drops out of the rotation entirely, so one budget shared across
// lanes lets a busy lane empty a quiet one: a tornado outbreak issues more
// warnings than any whole-stack cap can hold, and every one of them is
// top-severity, so ordering the cut by severity does not rescue a live
// hurricane or a significant quake — only giving each lane its own floor does.
const MaxPerLane = 30

// Lane is the marquee lane an event belongs to, and the unit the cap is per.
//
// The feed produces four. Advisories and statements reach the marquee from the
// tracked locations rather than from here, and carry their own bound.
// Lane is a category, named for the job it does here: the unit the national
// stack's cap is per. NOT an enum of its own — it was, and a lane missing from
// the order list beside it would have had its events dropped from the stack
// with nothing to say so (F-21).
//
// The national feed can only produce these four; the rest of the registry's
// categories arrive through tracked locations.
type Lane = category.Category

const (
	// LaneEmergency is the Emergency Orders lane — ReadRank 1, and the only
	// lane the read ladder never demotes. It joined the national feed at the
	// BUILD-exit red team (C-2): the query now asks for the Evacuation
	// Immediate, so the lane can actually be occupied.
	LaneEmergency = category.Emergency
	LaneDisasters = category.Disasters
	LaneMarine    = category.Marine
	LaneWarning   = category.Warnings
	LaneWatch     = category.Watches
)

// LaneOf is the lane an event belongs to. One owner: the marquee asks the same
// question when it draws, and two answers would put an event in one lane and
// cap it in another.
func LaneOf(e Event) Lane {
	switch e.Class {
	case ClassQuake:
		return LaneDisasters
	case ClassTropical:
		return LaneMarine
	}
	// THE CIVIL-EMERGENCY FAMILY IS MATCHED BY NAME, BEFORE THE KEYWORD ARM.
	// None of those products names a warning or a watch, so without this an
	// evacuation order fell through to LaneWarning — the rung below the one the
	// ladder reserves for it, and indistinguishable on the band from a
	// thunderstorm warning (C-2). The table is shared with domains/severe so
	// the window and the marquee cannot disagree about one hazard.
	if c, ok := CivilEmergencyCategory(e.Type); ok {
		return c
	}
	if strings.Contains(e.Type, "Watch") {
		return LaneWatch
	}
	return LaneWarning
}

// laneOrder is the categories the national feed can produce, in a fixed order
// so a capped stack is deterministic.
//
// This IS the list of what the feed yields — the registry describes every
// category, and only these arrive from the national query. LaneOf can return
// nothing else, and the test walks every event class through it to prove the
// two agree.
//
// A LANE MISSING FROM HERE IS SILENTLY DROPPED, not merely uncapped: capPerLane
// walks this list and keeps nothing else. That is why Emergency had to be added
// here in the same change that taught LaneOf the family (C-2) — laning an
// evacuation order correctly and leaving this list at four would have deleted
// it from the marquee outright.
//
// EMERGENCY LEADS, matching the read ladder and the window's tab order: an
// evacuation order is the most serious thing the feed can carry.
func laneOrder() []Lane {
	return []Lane{LaneEmergency, LaneDisasters, LaneMarine, LaneWarning, LaneWatch}
}

// capPerLane keeps at most MaxPerLane of each lane, worst first within a lane.
func capPerLane(evs []Event) []Event {
	byLane := map[Lane][]Event{}
	for _, e := range evs {
		byLane[e.laneKey()] = append(byLane[e.laneKey()], e)
	}
	out := make([]Event, 0, len(evs))
	for _, lane := range laneOrder() {
		held := byLane[lane]
		keepWorst(held)
		if len(held) > MaxPerLane {
			held = held[:MaxPerLane]
		}
		out = append(out, held...)
	}
	return out
}

func (e Event) laneKey() Lane { return LaneOf(e) }

// Sort orders events for the marquee: most recent first, and most severe first
// among events of the same instant. The stack is a breaking-news order.
func Sort(evs []Event) {
	sort.SliceStable(evs, func(i, j int) bool {
		if !evs[i].At.Equal(evs[j].At) {
			return evs[i].At.After(evs[j].At)
		}
		return evs[i].Severity > evs[j].Severity
	})
}

// keepWorst orders by severity so a truncation keeps the worst events. Recency
// breaks the tie: among equally severe events the fresher one holds its place.
func keepWorst(evs []Event) {
	sort.SliceStable(evs, func(i, j int) bool {
		if evs[i].Severity != evs[j].Severity {
			return evs[i].Severity > evs[j].Severity
		}
		return evs[i].At.After(evs[j].At)
	})
}

// Merge folds a fresh fetch into the ticker: it dedups by source ID (one entry
// per event — a single quake felt by many locations is still one USGS id, so
// the global feed never repeats it, D5), sorts most-recent-first, caps at
// MaxPerLane, and reports which events are NEW (their ID not in seen). The
// caller persists the seen set and sounds the tone/narration for the new ones
// (and, on a cold start, seeds seen quietly instead — P3).
func Merge(fetched []Event, seen map[string]bool) (stack, fresh []Event) {
	byID := make(map[string]Event, len(fetched))
	order := make([]string, 0, len(fetched))
	for _, e := range fetched {
		if e.ID == "" {
			continue // an event without a stable id cannot be deduped or tracked — drop it
		}
		if _, ok := byID[e.ID]; !ok {
			order = append(order, e.ID)
		}
		byID[e.ID] = e
	}
	stack = make([]Event, 0, len(order))
	for _, id := range order {
		stack = append(stack, byID[id])
	}
	// FRESH IS TAKEN BEFORE THE CAP. It is what the app sounds a tone for and
	// reads aloud, and an event the cap drops after this point is still one the
	// listener has never been told about. Whether an alert is NEW and whether it
	// fits on the tape are two different questions.
	for _, e := range stack {
		if !seen[e.ID] {
			fresh = append(fresh, e)
		}
	}

	// WHICH events survive is a severity question, per lane; the order they are
	// SHOWN in is a recency one.
	stack = capPerLane(stack)
	Sort(stack)
	return stack, fresh
}
