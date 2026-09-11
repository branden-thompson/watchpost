package app

// airscope.go — what the alert rail is scoped to, and who decides (D-73).
//
// THE ACCEPTANCE TEST IS THE HUM LEAD'S OWN (2026-09-10): "as long as when I
// switch to Broadcaster I don't hear alerts outside my service radius, then
// that's fine — 'it just works' — and when I switch back to Observer, that
// alert track has to re-adapt to whatever my filter settings dictate."
//
// SO THE FENCE FOLLOWS THE SURFACE, NOT THE SETTING. Both `fence()` and
// `scopeToRadius` read the LISTENER's alert radius around the LISTENER's default
// location, and nothing else could be true while one surface existed. With two,
// the same rail has to answer two different questions — and the honest way to
// hold that is one rail asking ONE function what it is scoped to, rather than
// two rails each sure of its own answer.
//
// ONE RAIL, NOT TWO. Giving each surface a schedule of its own would give each a
// hazard rail of its own, and a burst fed to both would be read TWICE. The rail
// is the safety path; it stays single, and what MOVES is the fence around it.

import (
	"sync/atomic"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// airScope is an origin and a radius — what a surface considers near.
//
// `set` FALSE IS "ALL", which is the rule both readers already state: a deck
// with no radius is unfenced rather than empty. It is a field rather than a
// zero-radius convention because "no fence" and "a fence of zero miles" are
// different answers and one of them must never be mistaken for the other.
type airScope struct {
	lat, lon float64
	radiusMi float64
	set      bool
}

// airOwner is which surface the operator is looking at.
//
// AN ATOMIC, BECAUSE THE ASKERS ARE THE TICKER'S OWN GOROUTINE and the setter is
// the program's. It carries a Surface rather than a bool so a third surface is a
// value rather than a rewrite.
type airOwner struct{ v atomic.Int32 }

func (o *airOwner) set(s tty.Surface) { o.v.Store(int32(s)) }

func (o *airOwner) get() tty.Surface {
	if o == nil {
		return tty.SurfaceObserver
	}
	return tty.Surface(o.v.Load())
}

// scopeFor is the ONE answer to "what is in scope right now", asked by both the
// rail's fence and the feed's filter.
//
// THE LISTENER'S FILTER IS THE DEFAULT, in every sense: it is what Observer
// shows, it is what an unset owner gets, and it is what the rail falls back to
// when the station has no epicentre. A console with no transmitter must not
// silently widen the listener's alerts to the whole country.
func scopeFor(owner *airOwner, listener func() airScope, st func() stationArea) airScope {
	if owner.get() != tty.SurfaceBroadcaster || st == nil {
		return listener()
	}
	s := st()
	if s.transmitter.Lat == 0 && s.transmitter.Lon == 0 {
		return listener() // no epicentre: the station has no region of its own to be scoped to
	}
	return airScope{lat: s.transmitter.Lat, lon: s.transmitter.Lon, radiusMi: s.radiusMi, set: s.radiusMi > 0}
}

// listenerScope is the alert radius the listener set, around the location they
// set it from — the answer Observer has always given.
func listenerScope(radius *atomic.Int64, watch func() []snapshot.LocationRef) airScope {
	if radius == nil || watch == nil {
		return airScope{} // a deck built for one narrow question: unfenced
	}
	r := float64(radius.Load())
	if r <= 0 {
		return airScope{} // All
	}
	w := watch()
	if len(w) == 0 {
		// FILTERED WITH NO DEFAULT LOCATION SHOWS NOTHING, which is the rule
		// `scopeToRadius` already states — and a radius with no origin is how
		// that is said here.
		return airScope{radiusMi: r, set: true}
	}
	return airScope{lat: w[0].Lat, lon: w[0].Lon, radiusMi: r, set: true}
}

// hasOrigin reports whether the scope has somewhere to measure from. A fence
// with a radius and no origin admits NOTHING, which is the filtered-with-no-
// default rule; a scope with neither is "All".
func (s airScope) hasOrigin() bool { return s.lat != 0 || s.lon != 0 }

// takeTheAir records which surface the operator moved to and asks the rail to
// re-scope AT ONCE.
//
// THE NUDGE IS HALF THE FEATURE. The ticker cycles every two minutes; without
// it the tape would go on showing the other surface's alerts for up to that
// long after a swap, which is the opposite of "it just works" in both
// directions.
func (lp *livePipelines) takeTheAir(s tty.Surface) {
	lp.owner.set(s)
	lp.ticker.nudgeRescope()
}
