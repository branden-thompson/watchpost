package globalfeed

import (
	"sort"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// miPerKm converts the radius (miles, as the user sets it) to the kilometres
// the haversine returns.
const kmPerMi = 1.609344

// Within keeps only events within radiusMi of (lat, lon) — the alert-radius
// filter (HUM LEAD 2026-08-27: "Filtered to N Mi of my location" scopes the
// whole ticker). A non-positive radius keeps everything (the global "All"
// ticker). An event with no point (a zone-only NWS alert) is excluded when a
// radius is set — its distance cannot be known.
func Within(evs []Event, lat, lon, radiusMi float64) []Event {
	if radiusMi <= 0 {
		return evs
	}
	radiusKm := radiusMi * kmPerMi
	out := make([]Event, 0, len(evs))
	for _, e := range evs {
		if !e.HasPoint {
			continue
		}
		if geo.HaversineKM(lat, lon, e.Lat, e.Lon) <= radiusKm {
			out = append(out, e)
		}
	}
	return out
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

// MaxEvents caps the ticker stack (P10-03: a big outbreak cannot grow the
// marquee or the memo without bound — the stack shows the most recent/severe).
const MaxEvents = 30

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

// Merge folds a fresh fetch into the ticker: it dedups by source ID (one entry
// per event — a single quake felt by many locations is still one USGS id, so
// the global feed never repeats it, D5), sorts most-recent-first, caps at
// MaxEvents, and reports which events are NEW (their ID not in seen). The
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
	Sort(stack)
	if len(stack) > MaxEvents {
		stack = stack[:MaxEvents]
	}
	for _, e := range stack {
		if !seen[e.ID] {
			fresh = append(fresh, e)
		}
	}
	return stack, fresh
}
