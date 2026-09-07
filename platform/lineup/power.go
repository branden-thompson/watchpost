package lineup

import "github.com/branden-thompson/watchpost/platform/invariant"

// Power is whether the Director is putting programme to air of its own accord.
//
// ONE FIELD, ONE RULE (PD-1). Today the same rule is `d.mode != ""` plus a tune
// epoch — one rule carried in two places, which is the shape that produced the
// duck-lift bug (RD-2): a rule living in two carriers is a rule that can
// disagree with itself, and the one nobody edited is the one that bites.
//
// It is an ENUM RATHER THAN A BOOL, and that is the seam. ON AIR / STANDBY is a
// second reason to be off, so a Broadcaster station adds a value here rather
// than a mechanism anywhere. That is a seam preserved by the thing that exists
// being honestly named — not by a stub, which would be dead code (AP-DEAD-01).
type Power int

const (
	// Stopped is the ZERO VALUE because that is how a station starts: nothing
	// plays until the listener asks for it, which is today's empty mode
	// (app/radio.go:radioDeck.setMode). A director that came up running would put a report to
	// air that nobody asked for.
	Stopped Power = iota
	Running

	// OffAir is DEAD AIR: nothing is broadcast, including hazards, and the
	// schedule HOLDS what it has not yet said (MVS-D-78).
	//
	// IT IS NOT A LOUDER "STOPPED", and the difference is the alert rail.
	// Stopped stops the PROGRAMME and lets hazards through — mute is the control
	// for "do not speak to me", stop for "do not play me a programme". Standby
	// is the listener saying the first about everything, so the rail is held
	// too, and a card the rail never reached stays new rather than being
	// consumed in silence.
	//
	// NAMED OffAir BECAUSE STANDBY IS ALREADY TAKEN, and by the opposite
	// meaning. A CARD on standby is READY TO AIR; a STATION on standby is OFF
	// air. Both are correct broadcast usage, and one package cannot hold both
	// under one word — least of all in the DR-23 timeline, where a line reading
	// "STANDBY" would be ambiguous between a card that is next and a station
	// that is silent. The operator-facing word stays STANDBY; the code says
	// which standby it means.
	OffAir

	// numPowers bounds the registry; it is not itself a power.
	numPowers
)

// String names the power for the transition line DR-23 asks for — a schedule
// doing nothing should say why, and "the radio is stopped" is the commonest
// answer there is.
//
// Written the same way Track.String is, and for the same reason: two members do
// not need a registry, and a table function that only one caller reads is a
// table nobody can see is complete.
func (p Power) String() string {
	if p < 0 || p >= numPowers {
		return ""
	}
	switch p {
	case Running:
		return "RUNNING"
	case OffAir:
		return "OFF AIR" // the operator sees "STANDBY (DEAD AIR)"; the log says which standby
	}
	return "STOPPED"
}

// Powered is the listener starting or stopping the radio.
//
// One event rather than a Start and a Stop, so the rule about what changes lives
// in one handler instead of two that must agree.
type Powered struct {
	isEvent
	To Power
}

// Power is what the Director is doing: running, or stopped.
func (d Director) Power() Power {
	if err := invariant.Check(d.power >= 0 && d.power < numPowers, "the director is in a declared power state"); err != nil {
		return Stopped
	}
	return d.power
}

// advances reports whether the Director may move a card along on this track.
//
// THE ALERT RAIL IS NOT SUBJECT TO IT, and that asymmetry is the point. Stopping
// the radio stops the PROGRAMME, not the hazards: today Stop halts the engine,
// which silences the broadcast, while a takeover reads through the narrator's
// own path and is never asked about the deck's mode (app/executors.go asks only
// audible() and muted()). Mute is the control for "do not speak
// to me"; stop is the control for "do not play me a programme".
//
// One predicate with two askers — taking the air, and preparing what follows.
func (d Director) advances(t Track) bool {
	if t < 0 || t >= numTracks {
		return false // a track outside the registry advances nothing
	}
	// FAIL CLOSED FIRST, and it is checked BEFORE the rail's exemption. It used
	// to sit after it, so a power outside the registry advanced the ALERT RAIL —
	// the one track the comment was written to protect. Nothing reachable
	// produces such a value (onPowered validates), so this was latent, and it
	// was found by walking the registry rather than a hand-written list of
	// powers (MVS-D-78).
	//
	// Standby is unbypassable for the same reason: a corrupt power must not be
	// a way around dead air.
	if d.power < 0 || d.power >= numPowers {
		return false
	}
	if d.power == OffAir {
		return false // dead air holds everything, the rail included
	}
	if t == AlertRail {
		return true
	}
	return d.power == Running
}

// onPowered starts or stops the programme.
//
// Stopping takes the programme off the air at once — the listener pressed stop
// and the broadcast goes quiet — and releases the band with it, because a
// callout left standing for a read that has ended is exactly the stale state
// DR-24 exists to prevent. It is NOT a discard of the schedule: the rest of the
// rotation waits, and starting again picks it up.
func (d Director) onPowered(ev Powered) (Director, []Effect) {
	if err := invariant.Check(ev.To >= 0 && ev.To < numPowers, "the radio is set to a declared power state"); err != nil {
		return d, nil
	}
	if err := invariant.Check(d.power >= 0 && d.power < numPowers, "the radio was in a declared power state"); err != nil {
		return d, nil
	}
	if d.power == ev.To {
		return d, nil // a repeated command is not a second event
	}
	d.power = ev.To
	d, fx := d.silenceTheProgramme()
	d, more := d.settle()
	return d, append(fx, more...)
}

// silenceTheProgramme takes a main-track card off the air when the radio stops.
//
// An ALERT is left alone: whether a hazard is on the air is the takeover's to
// say, and it pairs its own release. Nothing admitted is dropped unread (DR-3) —
// the rail keeps draining while the programme is stopped.
func (d Director) silenceTheProgramme() (Director, []Effect) {
	if d.power == Running {
		return d, nil
	}
	card, live := d.lineup.OnAir()
	if !live {
		return d, nil
	}
	track, _, held := d.lineup.find(card.ID)
	if !held || track == AlertRail {
		return d, nil
	}
	d, fx, _ := d.takeOffTheAir(card.ID, Discarded) // onPowered settles once, after this
	return d, fx
}
