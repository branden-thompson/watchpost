package lineup

// air.go — which programme reaches the engine (D-74).
//
// TWO PROGRAMMES SHARE ONE ENGINE, and until now they shared one GATE with it:
// `advances(MainTrack)` decided both whether the station's line-up advanced and
// whether the operator's own listening rotated, so with the station running both
// could produce audio at once. That is what the HUM LEAD heard — "audio in
// Broadcaster is still pulling audio from Observer."
//
// THE WORDS ARE THE STATION'S OWN, and they were chosen because "listener" had
// come to mean two different people. In a dashboard the listener is whoever is
// at the keyboard; in a BROADCAST product the listener is the audience in the
// service area. So:
//
//	MONITOR    what the operator listens to OFF AIR — Observer's own rotation
//	PROGRAMME  what the station puts OUT — the console's line-up
//
// and "listener" goes back to meaning the audience, where it belongs.
//
// THE AIR IS NOT THE POWER. A station can be STOPPED with the console in front
// of the operator: the programme owns the air and is not running, which is
// silence — and hazards still read, because that is ruled separately and the
// rail is exempt from both. Power says WHETHER the station broadcasts; the air
// says WHICH of the two programmes may.

import "github.com/branden-thompson/watchpost/platform/invariant"

// Air is which programme may reach the engine.
type Air int

const (
	// AirMonitor is the operator's own listening — the ZERO VALUE, because a
	// Director that has been told nothing is not a station on the air. Every
	// build before this one behaved exactly this way, and a default that
	// silently claimed the air would be the more dangerous of the two.
	AirMonitor Air = iota

	// AirProgramme is the station's line-up.
	AirProgramme

	// numAirs bounds the registry; it is not itself an air.
	numAirs
)

// String names the air for a diagnostic. An undeclared value names ITSELF as
// such rather than as one of the two — a log that silently reports "monitor"
// for a corrupt value is worse than one that says it does not know.
func (a Air) String() string {
	switch a {
	case AirMonitor:
		return "MONITOR"
	case AirProgramme:
		return "PROGRAMME"
	}
	return "air(?)"
}

// Aired hands the air to one programme or the other.
//
// ONE DECLARER, AND IT IS MASTERCONTROL (FR-5.4's shape, one concept along).
// The power already works this way — "they are the ONLY producers of a power
// change from the console" — and the air is the same kind of fact: something the
// operator DID, not something a tune happened to imply. Three declarers of the
// power is exactly how listening on one surface came to put the other ON AIR.
type Aired struct {
	isEvent
	To Air
}

// onAired moves the air, and moves nothing else.
//
// A REPEATED COMMAND IS NOT A SECOND EVENT — the rule `onPowered` and
// `onCutOver` already state, for the same reason: the operator swapping to a
// surface they are already on must not re-settle the schedule.
func (d Director) onAired(ev Aired) (Director, []Effect) {
	if err := invariant.Check(ev.To >= 0 && ev.To < numAirs, "the air is handed to a declared programme"); err != nil {
		return d, nil
	}
	if d.air == ev.To {
		return d, nil
	}
	d.air = ev.To
	// THE PROGRAMME IS SILENCED WHEN THE AIR LEAVES IT, exactly as a stop
	// silences it: a card on the air when the operator walks to the other
	// surface would go on reading into a programme nobody is carrying.
	d, fx := d.silenceTheProgramme()
	d, more := d.settle()
	return d, append(fx, more...)
}

// advancesMonitor reports whether the OPERATOR'S OWN listening may move on.
//
// IT ASKS THE MONITOR'S OWN POWER, NOT THE STATION'S, and that is the split. The
// station's power says whether the STATION broadcasts; `d.monitor` says whether
// the OPERATOR is listening. They were one field, which is what made Observer's
// tune declare the station ON AIR (D-69's root).
//
// IT DOES ASK `carries`, because that flag means the operator PARKED the
// station on the bed — and a rotation that moved on afterwards would take away
// the relay they just chose.
func (d Director) advancesMonitor() bool {
	if err := invariant.Check(d.air >= 0 && d.air < numAirs, "the air is one of the declared programmes"); err != nil {
		return false // fail closed: an undeclared air carries nothing
	}
	return d.monitor && d.air == AirMonitor && !d.bed.carries
}

// Monitored says whether the operator's own listening is running.
//
// THE SECOND POWER, RULED BY THE HUM LEAD (2026-09-10: "two powers"). They are
// genuinely two facts — is the STATION broadcasting, is the OPERATOR listening —
// and the only reason they were one field is that there was one programme. One
// field meant Observer's tune declared the station ON AIR, which is D-69's root
// and the reason `ctrl+o` was refused after a round trip.
//
// A BOOL, NOT A `Power`. A monitor has two states; `Power` has three, and the
// third — dead air — is a thing a STATION does. A monitor typed as a Power would
// carry a state it can never legitimately be in, and `advancesMonitor` would
// have to decide what OffAir means for something that is not on the air at all.
type Monitored struct {
	isEvent
	Running bool
}

// onMonitored starts or stops the operator's own listening.
//
// IT TOUCHES NOTHING ELSE. Stopping the monitor is not a station event: the
// line-up does not pause, the rail does not hold, and nothing leaves the air —
// the operator simply stopped listening. Every one of those WOULD have happened
// before D-74, because this arrived as `Powered{Stopped}`.
func (d Director) onMonitored(ev Monitored) (Director, []Effect) {
	if d.monitor == ev.Running {
		return d, nil // a repeated command is not a second event
	}
	d.monitor = ev.Running
	return d.settle()
}

// monitorWord names the monitor's state for a trace, so a log reads as a state
// rather than as a boolean.
func monitorWord(running bool) string {
	if running {
		return "RUNNING"
	}
	return "STOPPED"
}
