package lineup

// bed.go — the live broadcast the programme rides on, and the one decision the
// Director makes about it: when a Watchlist relay has held it long enough and
// the next location should take over (T3.2b, DR-3).
//
// THE DWELL WAS A TIMER INSIDE THE RADIO DECK. `armDwell` set a
// `time.AfterFunc` and `advanceQueue` fired on it, which made "when does the bed
// move" a thing you could only observe by waiting five minutes with a real
// clock. Here it is a pure function of the bed's state, the listener's settings
// and `now` — so a test states the schedule instead of watching for it (DR-2,
// NFR-D-1).

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Tuned says the bed moved: the deck is now carrying this location.
//
// LIVE IS THE HALF THAT MATTERS. A relay never ends, so Watchlist gives it a
// fixed turn and moves on; the synthesised broadcast ends on its own and needs
// no dwell at all. The old code asked the same question as `d.mode == "live"`.
type Tuned struct {
	isEvent
	Ref  string
	Live bool
}

// Programme is the listener's rotation: the queue, in their order, and how long
// a live relay holds the bed before the next one takes it.
//
// A ZERO DWELL MEANS NEVER ADVANCE, which is how "Repeat is not Watchlist"
// reaches the Director without the deck's mode enum coming with it. The rule the
// listener set is "move on after five minutes"; that a different repeat mode
// expresses itself as "never" is the caller's translation, made once at the
// call site rather than by a second copy of the enum in here.
type Programme struct {
	isEvent
	Watchlist []string
	Dwell     time.Duration
}

// bed is what the broadcast is riding on now.
type bed struct {
	ref   string    // the location carrying it; "" = nothing is tuned
	live  bool      // a relay, which dwells; the synth broadcast does not
	since time.Time // when it took the bed, for the dwell

	// asked and askedAt are the tune the Director has issued and not yet seen
	// land (FR-9.3). A rotation that is told to move and does not is a station
	// that has gone quiet with nothing coming, and nothing else observes it:
	// the deck reports Tuned when audio actually plays, so silence here is
	// silence everywhere.
	asked   string
	askedAt time.Time
}

// tuneLands is how long a tune has to land before the rotation is reported as
// stalled (FR-9.3).
//
// THIRTY SECONDS, AND THE DERIVATION MATTERS MORE THAN THE NUMBER. A resolve
// and a connect "can take seconds" — the reason the dwell starts at Playing
// rather than at the ask — and a failed relay falls through to a synthesised
// cycle, which re-renders at roughly ten seconds per utterance on Piper. Thirty
// is comfortably past both paths and still inside the time a listener would
// give a station that has stopped talking.
const tuneLands = 30 * time.Second

// onTuned records where the bed went. The dwell starts HERE, at the moment
// audio actually plays, rather than when the tune was asked for — a resolve and
// a connect can take seconds, and charging those to the listener's five minutes
// would cut every turn short by however slow the network was that time.
func (d Director) onTuned(ev Tuned) (Director, []Effect) {
	if err := invariant.Check(ev.Ref != "", "a tune names where the bed went"); err != nil {
		return d, nil
	}
	// A REPEAT REPORT DOES NOT RESTART THE TURN. A live relay sends a status
	// every time its title changes, and each one says Playing — so a countdown
	// restarted on every report would never elapse and the rotation would stop
	// dead on whichever station talks most. The deck's armDwell was idempotent
	// for exactly this reason ("a relay title change re-arms; it must not
	// restart the countdown"), and the rule has to survive the move.
	if d.bed.ref == ev.Ref && d.bed.live == ev.Live && !d.bed.since.IsZero() {
		return d, nil // the same bed, still carrying: its turn is already running
	}
	d.bed = bed{ref: ev.Ref, live: ev.Live, since: d.now}
	return d, nil
}

// tuneAsked records a tune the Director has just issued, so a tick can notice
// that it never landed.
func (d Director) tuneAsked(ref string) Director {
	d.bed.asked, d.bed.askedAt = ref, d.now
	return d
}

// stalledRotation is the report a tune that never landed raises, or nothing.
//
// UNLESS THE OPERATOR CHOSE SILENCE (FR-9.3, reworded by the HUM LEAD): a
// stopped station and a rotation that does not move by itself are both
// deliberate, and I-2 holds that a deliberate non-delivery is not a fault. The
// only case here is a station that was TOLD to move and did not.
//
// ONCE PER ASK. The pending tune is cleared as the report goes out, so a tick
// every second does not raise a window every second — the noise regression
// fault.go exists to avoid.
func (d Director) stalledRotation() (Director, []Effect) {
	if d.bed.asked == "" || d.now.Sub(d.bed.askedAt) < tuneLands {
		return d, nil
	}
	if !d.advances(MainTrack) {
		d.bed.asked = "" // stopped while it was in flight: not a fault, and not pending either
		return d, nil
	}
	ref := d.bed.asked
	d.bed.asked = ""
	return d, []Effect{Escalate{Reason: "the station was asked to move to " + ref + " and did not"}}
}

// onProgramme takes the listener's rotation.
func (d Director) onProgramme(ev Programme) (Director, []Effect) {
	if err := invariant.Check(ev.Dwell >= 0, "a dwell is not negative"); err != nil {
		return d, nil
	}
	for _, ref := range ev.Watchlist { // bounded by the queue (P10-02)
		// A BLANK ENTRY WOULD BE A TUNE TO NOWHERE, and the rotation would stop
		// there rather than reporting anything: the bed would be asked for a
		// location with no name and the deck would resolve nothing.
		if err := invariant.Check(ref != "", "every entry in the rotation names a location"); err != nil {
			return d, nil
		}
	}
	d.settings.Watchlist = append([]string(nil), ev.Watchlist...) // the caller keeps its slice
	d.settings.Dwell = ev.Dwell
	return d, nil
}

// advanceBed is the dwell, asked on every tick: the effect that moves the bed on
// when a live relay has had its turn, or nothing.
//
// IT RESTARTS THE DWELL AT THE MOMENT IT DECIDES, not when the new relay
// arrives. Waiting for the `Tuned` that follows would leave the deadline in the
// past until the tune resolved, and every tick in between would emit another
// advance — a resolve that took two seconds would fire two more tunes behind the
// one already in flight.
func (d Director) advanceBed() (Director, []Effect) {
	if !d.dwellElapsed() {
		return d, nil
	}
	next, ok := d.nextInWatchlist()
	if !ok {
		return d, nil // nothing to move to: the bed keeps what it has
	}
	// A ONE-ENTRY WATCHLIST IS ALREADY WHERE IT IS GOING. The wrap makes the
	// next entry the current one, so advancing would cut the audio and re-tune
	// the same relay every five minutes for no reason — the listener hears their
	// only station restart on a timer. Found by the density gate sending me back
	// through this function.
	if next == d.bed.ref {
		d.bed.since = d.now // its turn starts again; there is nowhere else to go
		return d, nil
	}
	if err := invariant.Check(next != "", "the bed is only ever tuned somewhere"); err != nil {
		return d, nil // a queue carrying a blank entry would tune the bed to nothing
	}
	d.bed.since = d.now
	// THE TURN RESTARTS HERE, and this is the check that it actually did. The
	// Director is a VALUE: a reset written to a copy that is not returned would
	// leave the deadline in the past, and every tick after this would emit
	// another tune behind the one already in flight — the bed landing wherever
	// the last of them won. It costs one comparison and it catches the one
	// mistake this shape invites.
	if err := invariant.Check(!d.dwellElapsed(), "an advance restarts the turn it just spent"); err != nil {
		return d, nil
	}
	return d.tuneAsked(next), []Effect{Tune{Ref: next}}
}

// Ended says the programme on the bed finished on its own — the synthesised
// broadcast played to the end of its cycle. A live relay never ends, which is
// why it has a dwell and this does not apply to it.
//
// IT IS AN EVENT, NOT A TICK. The dwell is a deadline the Director can compute;
// a cycle ending is something only the deck can observe, so it arrives the same
// way an arrival does. `advanceQueue` had both callers and the absorb needs
// both, which the state table made visible before any of this was written.
type Ended struct{ isEvent }

// onEnded moves the bed on when the programme it was carrying finished.
//
// NO DWELL IS CONSULTED. The turn is over because the broadcast is over, not
// because a clock said so — a cycle that ran two minutes moves on at two
// minutes, exactly as it does today.
func (d Director) onEnded(Ended) (Director, []Effect) {
	// SOMETHING HAS TO HAVE BEEN PLAYING for a cycle to have ended. A deck
	// reporting one with no bed tuned is a wiring fault, and advancing on it
	// would move the rotation from nowhere to its first entry — a station
	// starting itself out of silence.
	if err := invariant.Check(d.bed.ref != "", "a cycle only ends on a bed that was carrying one"); err != nil {
		return d, nil
	}
	if !d.advances(MainTrack) {
		return d, nil // stopped: nothing follows a listener's stop
	}
	if d.settings.Dwell <= 0 {
		return d, nil // not Watchlist: the rotation does not move on by itself
	}
	next, ok := d.nextInWatchlist()
	if !ok || next == d.bed.ref {
		return d, nil // nowhere else to go
	}
	d.bed.since = d.now
	return d.tuneAsked(next), []Effect{Tune{Ref: next}}
}

// dwellElapsed reports whether the live relay on the bed has had its turn.
func (d Director) dwellElapsed() bool {
	// A STOPPED PROGRAMME DOES NOT ADVANCE. The listener pressed stop; the bed
	// moving on afterwards would be the station starting itself again five
	// minutes later. `advances` is the one carrier of that question and the
	// alert rail is deliberately exempt from it — hazards still speak over a
	// stopped programme — so this asks it for the MAIN TRACK, which is what the
	// bed carries.
	if !d.advances(MainTrack) {
		return false
	}
	if d.settings.Dwell <= 0 || !d.bed.live || d.bed.ref == "" {
		return false // not Watchlist, not a relay, or nothing tuned
	}
	if d.bed.since.IsZero() {
		return false // never observed taking the air; it has not started its turn
	}
	// THE BED CANNOT HAVE TAKEN THE AIR IN THE FUTURE. If it had — a clock that
	// ran backwards, a Tuned stamped against a later now than this tick — the
	// subtraction below never reaches the dwell and the rotation stalls in
	// silence, with nothing to say why.
	if err := invariant.Check(!d.bed.since.After(d.now), "the bed took the air before now"); err != nil {
		return false
	}
	return !d.now.Before(d.bed.since.Add(d.settings.Dwell))
}

// nextInWatchlist is the location after the one on the bed, wrapping; the first
// when the bed is on something outside the queue.
//
// THE SAME RULE THE DECK'S nextInQueue HAD, moved rather than rewritten — a
// listener whose bed is on a location they since removed from the watchlist
// rejoins at the top rather than stopping.
func (d Director) nextInWatchlist() (string, bool) {
	q := d.settings.Watchlist
	if len(q) == 0 {
		return "", false
	}
	for i, ref := range q { // bounded by the queue (P10-02)
		if ref == d.bed.ref {
			return q[(i+1)%len(q)], true
		}
	}
	return q[0], true
}
