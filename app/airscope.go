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
	tea "charm.land/bubbletea/v2"

	"sync/atomic"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/lineup"
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
// IT RETURNS A COMMAND (D-79). Silencing the monitor HALTS THE PLAYER, and the
// player calls back into the program — so run inline, from inside `Router.Update`,
// it sends to a loop that cannot receive and the app freezes hard. Observer has
// always stopped the radio this way: `withCmd(func() tea.Msg { radio.Stop();
// return nil })`. This is the same canonical way, reached from the swap.
//
// EVERYTHING ELSE STAYS INLINE, deliberately. Recording the owner, declaring the
// air and nudging the rail are all non-blocking and must be TRUE by the time the
// next frame draws — deferring them would let one frame render with the old
// surface's fence.
func (lp *livePipelines) takeTheAir(s tty.Surface) tea.Cmd {
	lp.owner.set(s)
	// THE AIR MOVES WITH THE SURFACE, AND MASTERCONTROL IS WHAT SAYS SO (D-74).
	//
	// "IF I'M IN THE BROADCASTER UI, THEN THE OBSERVER MODE MUST BE SILENT"
	// (HUM LEAD, 2026-09-10). Hearing the operator's own listening while looking
	// at a console that says STOPPED is the confusion this removes — so moving to
	// the console takes the air AND stops the monitor, audio and all.
	//
	// GOING BACK DOES NOT RESUME IT, which he ruled in the same breath: the
	// monitor comes back "like if Observer was first opened". The operator
	// presses play. That is also what keeps a rapid ctrl+b / ctrl+o flip from
	// re-resolving a relay and re-fetching its products on every swap.
	// THE FENCE GOES WITH THE AIR (D-75), and it is read AFTER the owner moves
	// so it is the fence of the surface being taken TO.
	if mc := lp.masterControl(); mc != nil {
		if s == tty.SurfaceBroadcaster {
			mc.HandAir(lineup.AirProgramme)
			mc.StopMonitor()
		} else {
			mc.HandAir(lineup.AirMonitor)
		}
	}
	lp.ticker.nudgeRescope()
	if s != tty.SurfaceBroadcaster {
		return nil
	}
	return func() tea.Msg {
		lp.silenceMonitor()
		return nil
	}
}

// masterControl is the effector, or nil in the modes that have no audio.
func (lp *livePipelines) masterControl() *mastercontrol {
	if lp == nil || lp.director == nil {
		return nil
	}
	return lp.director.mc
}

// silenceMonitor stops the audio the OPERATOR was listening to, without
// touching the station.
//
// THE DECK IS TOLD DIRECTLY, because the Director's `Monitored` event governs
// the ROTATION and not the player: a rotation that will not advance still leaves
// whatever is already playing on the air. Both halves are the same instruction
// and both are needed.
// IT RUNS ON A COMMAND'S GOROUTINE, NEVER ON THE UPDATE LOOP — see takeTheAir.
func (lp *livePipelines) silenceMonitor() {
	if lp == nil || lp.deck == nil {
		return
	}
	lp.deck.Stop()
}
